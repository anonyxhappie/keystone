package harness

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/anonyxhappie/keystone/v2/internal/domain"
)

// CLIAdapter bridges a provider's documented headless CLI into Keystone's
// existing Adapter contract. Each turn is a bounded process. Provider-native
// conversation/session resume is used for follow-up turns; Keystone never
// embeds or reimplements the provider's agent runtime.
type CLIAdapter struct {
	Config   Config
	Root     string
	provider string
	command  string

	ctx    context.Context
	cancel context.CancelFunc

	mu             sync.Mutex
	waitOnce       sync.Once
	cmd            *exec.Cmd
	out            *bufio.Reader
	stderr         bytes.Buffer
	runID          string
	sessionID      string
	started        bool
	finished       bool
	result         domain.Status
	providerResult domain.Status
	pending        []domain.Observation
	stderrReported bool
	version        string
}

func NewAdapter(ctx context.Context, root string, cfg Config) Adapter {
	switch normalizeProvider(cfg.Provider) {
	case "codex":
		return NewCodexAdapter(ctx, root, cfg)
	case "antigravity":
		return NewAntigravityAdapter(ctx, root, cfg)
	default:
		return NewProcessAdapter(ctx, cfg)
	}
}

func NewCodexAdapter(ctx context.Context, root string, cfg Config) *CLIAdapter {
	return newCLIAdapter(ctx, root, cfg, "codex")
}

func NewAntigravityAdapter(ctx context.Context, root string, cfg Config) *CLIAdapter {
	return newCLIAdapter(ctx, root, cfg, "antigravity")
}

// NewCodex is kept as a concise constructor for integrations that select a
// provider directly rather than through harness.json.
func NewCodex(ctx context.Context, root string, cfg Config) *CLIAdapter {
	return NewCodexAdapter(ctx, root, cfg)
}

// NewAntigravity is kept as a concise constructor for integrations that select
// a provider directly rather than through harness.json.
func NewAntigravity(ctx context.Context, root string, cfg Config) *CLIAdapter {
	return NewAntigravityAdapter(ctx, root, cfg)
}

func newCLIAdapter(ctx context.Context, root string, cfg Config, provider string) *CLIAdapter {
	if cfg.Command == "" {
		cfg.Command = defaultProviderCommand(provider)
	}
	if cfg.Name == "" {
		cfg.Name = provider
	}
	if cfg.Provider == "" {
		cfg.Provider = provider
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 300
	}
	child, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
	return &CLIAdapter{Config: cfg, Root: root, provider: provider, command: cfg.Command, ctx: child, cancel: cancel, result: domain.StatusUnknown, providerResult: domain.StatusUnknown}
}

func (a *CLIAdapter) Discover() error {
	if a.command == "" {
		return fmt.Errorf("empty %s harness command", a.provider)
	}
	_, version, ok := probeCommand(a.command)
	if !ok {
		return fmt.Errorf("%s harness unavailable: %s", a.provider, version)
	}
	a.mu.Lock()
	a.version = version
	a.mu.Unlock()
	return nil
}

func (a *CLIAdapter) HarnessID() string { return a.provider }

func (a *CLIAdapter) SessionID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessionID
}

func (a *CLIAdapter) Version() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.version != "" {
		return a.version
	}
	return "configured"
}

func (a *CLIAdapter) Capabilities() []string {
	capabilities, _, _, _, _ := providerCapabilities(a.provider)
	return unique(append(capabilities, a.Config.Capabilities...))
}

func (a *CLIAdapter) Metadata() Metadata {
	_, control, observability, evidence, limitations := providerCapabilities(a.provider)
	return Metadata{Provider: a.provider, Version: a.Version(), Control: append([]string(nil), control...), Observability: append([]string(nil), observability...), Evidence: append([]string(nil), evidence...), Limitations: append([]string(nil), limitations...)}
}

func (a *CLIAdapter) Start(packet domain.WorkPacket) (string, error) {
	return a.start(Render(packet), "")
}

func (a *CLIAdapter) Send(prompt string) error {
	a.mu.Lock()
	active := a.started && !a.finished
	sessionID := a.sessionID
	a.mu.Unlock()
	if active {
		return fmt.Errorf("%w: %s CLI turns are one-shot; use a completed session for follow-up", ErrUnsupported, a.provider)
	}
	if sessionID == "" {
		return io.ErrClosedPipe
	}
	_, err := a.start(prompt, sessionID)
	return err
}

func (a *CLIAdapter) Resume(checkpoint domain.Checkpoint) error {
	_, err := a.ResumePacket(checkpoint, domain.WorkPacket{WorkOrderID: checkpoint.WorkOrderID, Objective: "resume durable Keystone checkpoint"})
	return err
}

func (a *CLIAdapter) ResumePacket(checkpoint domain.Checkpoint, packet domain.WorkPacket) (string, error) {
	sessionID := checkpoint.HarnessSessionID
	if sessionID == "" {
		return "", fmt.Errorf("%w: checkpoint has no %s provider session id", ErrUnsupported, a.provider)
	}
	prompt := Render(packet) + "\n\nResume from durable Keystone checkpoint:\n" + marshalCheckpoint(checkpoint)
	return a.start(prompt, sessionID)
}

func (a *CLIAdapter) start(prompt, resumeSession string) (string, error) {
	a.mu.Lock()
	if a.started && !a.finished {
		a.mu.Unlock()
		return "", fmt.Errorf("%s harness already has an active turn", a.provider)
	}
	a.runID = fmt.Sprintf("RUN-%d", time.Now().UnixNano())
	a.sessionID = resumeSession
	a.pending = nil
	a.stderr.Reset()
	a.stderrReported = false
	a.result = domain.StatusUnknown
	a.providerResult = domain.StatusUnknown
	a.finished = false
	a.started = true
	a.waitOnce = sync.Once{}
	a.mu.Unlock()

	args := a.args(prompt, resumeSession)
	cmd := exec.CommandContext(a.ctx, a.command, args...)
	cmd.Dir = a.Root
	cmd.Stdin = bytes.NewReader(nil)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	cmd.Stderr = &a.stderr
	if err := cmd.Start(); err != nil {
		a.mu.Lock()
		a.started = false
		a.finished = true
		a.result = domain.StatusFailed
		a.mu.Unlock()
		return "", fmt.Errorf("start %s: %w", a.provider, err)
	}
	a.mu.Lock()
	a.cmd = cmd
	a.out = bufio.NewReader(stdout)
	runID := a.runID
	a.mu.Unlock()

	// Both supported providers emit a session/conversation event at the start
	// of headless execution. Capture it before returning so the durable
	// HarnessSession record is keyed by the real provider session.
	items, err := a.readUntilReady()
	a.mu.Lock()
	a.pending = append(a.pending, items...)
	sessionID := a.sessionID
	a.mu.Unlock()
	if err != nil {
		a.finish()
		if diagnostic := a.diagnostic(); diagnostic != "" {
			return "", fmt.Errorf("%s did not start a session: %w (%s)", a.provider, err, diagnostic)
		}
		return "", fmt.Errorf("%s did not start a session: %w", a.provider, err)
	}
	if sessionID == "" {
		// A provider may omit the initial id in an informational first event. The
		// adapter remains resumable if a later event supplies it.
		sessionID = runID
	}
	return sessionID, nil
}

func (a *CLIAdapter) readUntilReady() ([]domain.Observation, error) {
	items := []domain.Observation{}
	for i := 0; i < 16; i++ {
		item, err := a.readLine()
		if item != nil {
			items = append(items, *item)
			if a.SessionID() != "" {
				return items, nil
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return items, io.EOF
			}
			return items, err
		}
	}
	return items, nil
}

func (a *CLIAdapter) Observe() ([]domain.Observation, error) {
	a.mu.Lock()
	if len(a.pending) > 0 {
		item := a.pending[0]
		a.pending = a.pending[1:]
		a.mu.Unlock()
		return []domain.Observation{item}, nil
	}
	a.mu.Unlock()
	item, err := a.readLine()
	if item != nil {
		return []domain.Observation{*item}, nil
	}
	if errors.Is(err, io.EOF) {
		a.finish()
		if stderrItem := a.takeStderrObservation(); stderrItem != nil {
			return []domain.Observation{*stderrItem}, nil
		}
	}
	return nil, err
}

func (a *CLIAdapter) readLine() (*domain.Observation, error) {
	a.mu.Lock()
	out := a.out
	if out == nil {
		a.mu.Unlock()
		return nil, io.ErrClosedPipe
	}
	line, err := out.ReadString('\n')
	a.mu.Unlock()
	line = strings.TrimSpace(line)
	if line != "" {
		item := a.observation(line)
		// A complete final line is an observation, not an EOF at the call site.
		return &item, nil
	}
	return nil, err
}

func (a *CLIAdapter) finish() {
	a.waitOnce.Do(func() {
		a.mu.Lock()
		cmd := a.cmd
		a.mu.Unlock()
		if cmd == nil {
			return
		}
		err := cmd.Wait()
		a.mu.Lock()
		defer a.mu.Unlock()
		a.finished = true
		if a.providerResult == domain.StatusFailed || err != nil {
			a.result = domain.StatusFailed
		} else if a.providerResult == domain.StatusCompleted {
			a.result = domain.StatusCompleted
		} else {
			a.result = domain.StatusCompleted
		}
	})
}

func (a *CLIAdapter) Interrupt() error {
	a.mu.Lock()
	cmd := a.cmd
	finished := a.finished
	a.mu.Unlock()
	if finished || cmd == nil || cmd.Process == nil {
		return nil
	}
	err := cmd.Process.Signal(os.Interrupt)
	if err != nil {
		err = cmd.Process.Kill()
	}
	a.mu.Lock()
	a.providerResult = domain.StatusFailed
	a.result = domain.StatusFailed
	a.mu.Unlock()
	return err
}

func (a *CLIAdapter) Stop() error {
	err := a.Interrupt()
	a.cancel()
	return err
}

func (a *CLIAdapter) Result() (domain.Status, error) {
	a.finish()
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.result, nil
}

func (a *CLIAdapter) takeStderrObservation() *domain.Observation {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.stderrReported || strings.TrimSpace(a.stderr.String()) == "" {
		return nil
	}
	a.stderrReported = true
	message := redactDiagnostic(a.stderr.String())
	return &domain.Observation{ID: fmt.Sprintf("OBS-%d", time.Now().UnixNano()), RunID: a.runID, Type: "ERROR", Summary: truncate(message, 4096), Timestamp: time.Now().UTC(), Payload: map[string]any{"provider": a.provider, "stderr": truncate(message, 4096)}}
}

func (a *CLIAdapter) diagnostic() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return providerDiagnostic(a.stderr.String())
}

func (a *CLIAdapter) args(prompt, resumeSession string) []string {
	args := append([]string(nil), a.Config.Args...)
	switch a.provider {
	case "codex":
		if resumeSession != "" {
			args = append(args, "exec", "resume")
			if !hasArg(args, "--json") {
				args = append(args, "--json")
			}
			args = append(args, resumeSession, prompt)
		} else {
			args = append(args, "exec")
			if !hasArg(args, "--json") {
				args = append(args, "--json")
			}
			if !hasArg(args, "--sandbox") {
				args = append(args, "--sandbox", "workspace-write")
			}
			if !hasArg(args, "--cd") && !hasArg(args, "-C") {
				args = append(args, "--cd", a.Root)
			}
			args = append(args, prompt)
		}
	case "antigravity":
		args = append(args, "-p", prompt)
		if !hasArg(args, "--output-format") {
			args = append(args, "--output-format", "stream-json")
		}
		if resumeSession != "" && !hasArg(args, "--conversation") {
			args = append(args, "--conversation", resumeSession)
		}
	default:
		args = append(args, prompt)
	}
	return args
}

func flattenEvent(raw map[string]any) map[string]any {
	out := make(map[string]any, len(raw)+8)
	for k, v := range raw {
		out[k] = v
	}
	for _, nestedKey := range []string{"step_update", "result", "init", "turn", "item"} {
		if sub, ok := raw[nestedKey].(map[string]any); ok {
			for k, v := range sub {
				if _, exists := out[k]; !exists {
					out[k] = v
				}
			}
		}
	}
	return out
}

func (a *CLIAdapter) observation(line string) domain.Observation {
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return domain.Observation{ID: fmt.Sprintf("OBS-%d", time.Now().UnixNano()), RunID: a.currentRunID(), Type: "MESSAGE_RECEIVED", Summary: truncate(redact(line), 4096), Timestamp: time.Now().UTC(), Payload: map[string]any{"provider": a.provider, "raw": truncate(redact(line), 8192)}}
	}
	return a.observationFromEvent(raw)
}

func (a *CLIAdapter) observationFromEvent(raw map[string]any) domain.Observation {
	flat := flattenEvent(raw)
	eventType := firstString(flat, "type", "event", "kind")
	if eventType == "" {
		eventType = "message"
	}
	provider := a.provider
	sessionID := firstString(flat, "thread_id", "conversation_id", "session_id", "sessionId")
	if sessionID != "" {
		a.mu.Lock()
		a.sessionID = sessionID
		a.mu.Unlock()
	}
	obsType, summary := classifyProviderEvent(provider, eventType, flat)
	if status := providerEventStatus(provider, eventType, flat); status != domain.StatusUnknown {
		a.mu.Lock()
		a.providerResult = status
		a.mu.Unlock()
	}
	payload := redactValue(flat)
	if payloadMap, ok := payload.(map[string]any); ok {
		payloadMap["provider"] = provider
		payloadMap["eventType"] = eventType
		if sessionID != "" {
			payloadMap["sessionId"] = sessionID
		}
	}
	id := firstString(flat, "id", "event_id", "eventId")
	if id == "" {
		id = fmt.Sprintf("OBS-%s", a.provider)
	}
	return domain.Observation{ID: id + a.idSuffix(), RunID: a.currentRunID(), Type: obsType, Summary: truncate(redact(summary), 4096), Payload: payload.(map[string]any), Timestamp: time.Now().UTC()}
}

func (a *CLIAdapter) currentRunID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.runID
}

func (a *CLIAdapter) idSuffix() string {
	return fmt.Sprintf("-%d", time.Now().UnixNano())
}

func classifyProviderEvent(provider, eventType string, raw map[string]any) (string, string) {
	lower := strings.ToLower(eventType)
	if provider == "codex" {
		switch lower {
		case "thread.started":
			return "SESSION_STARTED", "Codex thread started: " + firstString(raw, "thread_id")
		case "turn.started":
			return "TURN_STARTED", "Codex turn started"
		case "turn.completed":
			return "TURN_COMPLETED", "Codex turn completed"
		case "turn.failed", "error":
			return "ERROR", providerSummary(raw, "Codex reported an error")
		case "item.started", "item.completed":
			item := object(raw, "item")
			itemType := strings.ToLower(firstString(item, "type", "item_type"))
			summary := providerSummary(item, providerSummary(raw, "Codex item"))
			completed := lower == "item.completed"
			switch {
			case strings.Contains(itemType, "command") || strings.Contains(itemType, "shell"):
				if completed {
					return "COMMAND_COMPLETED", summary
				}
				return "COMMAND_STARTED", summary
			case strings.Contains(itemType, "file") || strings.Contains(itemType, "edit"):
				return "FILE_CHANGED", summary
			case strings.Contains(itemType, "tool") || strings.Contains(itemType, "mcp"):
				if completed {
					return "TOOL_COMPLETED", summary
				}
				return "TOOL_STARTED", summary
			default:
				message := firstString(item, "text", "message", "content")
				if message == "" {
					message = summary
				}
				return classify(message), message
			}
		}
		return classify(providerSummary(raw, eventType)), providerSummary(raw, eventType)
	}

	switch lower {
	case "init", "session.started", "conversation.started":
		return "SESSION_STARTED", "Antigravity conversation started: " + firstString(raw, "conversation_id", "session_id")
	case "step_update", "step.update":
		stepType := strings.ToLower(firstString(raw, "step_type", "stepType", "type_name"))
		if strings.Contains(stepType, "tool") {
			state := strings.ToLower(firstString(raw, "status", "state", "step_status"))
			if state == "started" || state == "running" || state == "in_progress" {
				return "TOOL_STARTED", providerSummary(raw, firstString(raw, "tool_name", "toolName", "name"))
			}
			return "TOOL_COMPLETED", providerSummary(raw, firstString(raw, "tool_name", "toolName", "name"))
		}
		if strings.Contains(stepType, "agent") || strings.Contains(stepType, "response") || firstString(raw, "text_delta", "text", "response") != "" {
			message := firstString(raw, "text_delta", "text", "response", "message")
			return classify(message), message
		}
		return "MESSAGE_RECEIVED", providerSummary(raw, "Antigravity step update")
	case "result", "conversation.completed", "turn.completed":
		message := firstString(raw, "response", "result", "message", "text")
		if message == "" {
			message = "Antigravity conversation completed"
		}
		typ := classify(message)
		if typ == "MESSAGE_RECEIVED" && strings.EqualFold(firstString(raw, "status"), "success") {
			typ = "COMPLETION_CLAIM"
		}
		return typ, message
	case "error", "failed", "turn.failed":
		return "ERROR", providerSummary(raw, "Antigravity reported an error")
	default:
		return classify(providerSummary(raw, eventType)), providerSummary(raw, eventType)
	}
}

func providerEventStatus(provider, eventType string, raw map[string]any) domain.Status {
	status := strings.ToLower(firstString(raw, "status", "state", "result"))
	if provider == "codex" {
		switch strings.ToLower(eventType) {
		case "turn.failed", "error":
			return domain.StatusFailed
		case "turn.completed":
			return domain.StatusCompleted
		}
		return domain.StatusUnknown
	}
	if strings.Contains(status, "fail") || strings.Contains(status, "error") || strings.Contains(status, "cancel") {
		return domain.StatusFailed
	}
	if strings.Contains(status, "success") || strings.Contains(status, "complete") || strings.Contains(status, "done") {
		return domain.StatusCompleted
	}
	return domain.StatusUnknown
}

func providerSummary(raw map[string]any, fallback string) string {
	for _, key := range []string{"command", "tool_name", "toolName", "response", "message", "text", "text_delta", "aggregated_output", "output", "error"} {
		if value := firstString(raw, key); value != "" {
			return value
		}
	}
	return fallback
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return typed
			}
		case fmt.Stringer:
			return typed.String()
		case float64, bool:
			return fmt.Sprint(typed)
		}
	}
	return ""
}

func object(values map[string]any, key string) map[string]any {
	if value, ok := values[key].(map[string]any); ok {
		return value
	}
	return map[string]any{}
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			low := strings.ToLower(key)
			if strings.Contains(low, "token") || strings.Contains(low, "secret") || strings.Contains(low, "password") || strings.Contains(low, "api_key") || strings.Contains(low, "apikey") {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = redactValue(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = redactValue(item)
		}
		return out
	case string:
		return truncate(redact(typed), 8192)
	default:
		return value
	}
}

func marshalCheckpoint(checkpoint domain.Checkpoint) string {
	b, err := json.Marshal(checkpoint)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func hasArg(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
	}
	return false
}

func unique(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

var diagnosticURL = regexp.MustCompile(`https?://[^\s]+`)

func redactDiagnostic(value string) string {
	value = diagnosticURL.ReplaceAllString(value, "[URL_REDACTED]")
	return redact(value)
}

func providerDiagnostic(value string) string {
	low := strings.ToLower(value)
	switch {
	case strings.Contains(low, "authentication required") || strings.Contains(low, "log in") || strings.Contains(low, "oauth"):
		return "authentication required"
	case strings.Contains(low, "permission denied") || strings.Contains(low, "operation not permitted"):
		return "local permission denied"
	case strings.Contains(low, "timed out") || strings.Contains(low, "timeout"):
		return "provider startup timed out"
	case strings.TrimSpace(value) != "":
		return "provider emitted a startup diagnostic"
	default:
		return ""
	}
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}
