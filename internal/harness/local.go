package harness

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
)

type Config struct {
	Name           string   `json:"name"`
	Provider       string   `json:"provider,omitempty"`
	Command        string   `json:"command"`
	Args           []string `json:"args,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty"`
}

func LoadConfig(root string) (Config, error) {
	var cfg Config
	configPath := filepath.Join(root, ".keystone", "harness.json")
	b, err := os.ReadFile(configPath)
	configured := err == nil
	if err == nil {
		if err = json.Unmarshal(b, &cfg); err != nil {
			return cfg, fmt.Errorf("invalid harness.json: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return cfg, fmt.Errorf("read harness.json: %w", err)
	}
	provider := normalizeProvider(cfg.Provider)
	if provider == "" {
		provider = normalizeProvider(os.Getenv("KEYSTONE_HARNESS"))
	}
	if cfg.Command == "" {
		cfg.Command = os.Getenv("KEYSTONE_HARNESS_COMMAND")
	}
	if cfg.Command == "" && provider != "" {
		cfg.Command = defaultProviderCommand(provider)
	}
	if cfg.Command == "" && !configured && os.Getenv("KEYSTONE_HARNESS") == "" && os.Getenv("KEYSTONE_HARNESS_COMMAND") == "" {
		for _, candidate := range []string{"codex", "antigravity"} {
			command := defaultProviderCommand(candidate)
			if _, _, ok := probeCommand(command); ok {
				provider = candidate
				cfg.Command = command
				break
			}
		}
	}
	if provider != "" {
		cfg.Provider = provider
	}
	if cfg.Name == "" && cfg.Command != "" {
		if provider != "" {
			cfg.Name = provider
		} else {
			cfg.Name = "local-process"
		}
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 300
	}
	if cfg.Command == "" {
		return cfg, fmt.Errorf("no harness configured; create .keystone/harness.json, set KEYSTONE_HARNESS_COMMAND, or install/configure codex or agy")
	}
	return cfg, nil
}

type Discovery struct {
	Name          string   `json:"name"`
	Kind          string   `json:"kind"`
	Available     bool     `json:"available"`
	Provider      string   `json:"provider,omitempty"`
	Version       string   `json:"version,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	Control       []string `json:"control,omitempty"`
	Observability []string `json:"observability,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	Limitations   []string `json:"limitations,omitempty"`
	Reason        string   `json:"reason,omitempty"`
}

func Discover(root string) []Discovery {
	out := []Discovery{{Name: "manual", Kind: "instruction-file", Available: true, Capabilities: []string{"prompt-export", "result-import"}}}
	seen := map[string]bool{"manual": true}
	if cfg, err := LoadConfig(root); err == nil {
		discovery := discoveryForConfig(cfg)
		out = append(out, discovery)
		seen[discovery.Name] = true
	} else {
		out = append(out, Discovery{Name: "local-process", Kind: "local-process", Reason: err.Error()})
		seen["local-process"] = true
	}
	for _, provider := range []string{"codex", "antigravity"} {
		if seen[provider] {
			continue
		}
		cfg := Config{Name: provider, Provider: provider, Command: defaultProviderCommand(provider), TimeoutSeconds: 300}
		out = append(out, discoveryForConfig(cfg))
	}
	return out
}

// SelectHarness authoritatively resolves the harness for a run.
// If an explicit harness is requested, it MUST be satisfied; no silent fallback is permitted.
// If unavailable, it produces a REQUIRE_APPROVAL policy decision and an informative error listing alternatives.
// If auto mode is requested, Keystone selects an available harness and records why.
func SelectHarness(ctx context.Context, root string, requested string) (Adapter, domain.HarnessSelection, error) {
	discoveries := Discover(root)
	availableMap := map[string]Discovery{}
	availableList := []string{}
	allDiscoveries := map[string]Discovery{}

	for _, d := range discoveries {
		allDiscoveries[d.Name] = d
		if d.Provider != "" {
			allDiscoveries[d.Provider] = d
		}
		if d.Available {
			availableMap[d.Name] = d
			availableList = append(availableList, d.Name)
		}
	}

	reqTrimmed := strings.TrimSpace(requested)
	selection := domain.HarnessSelection{
		RequestedHarness:   reqTrimmed,
		AvailableHarnesses: availableList,
	}

	isAuto := reqTrimmed == "" || strings.EqualFold(reqTrimmed, "auto")

	if !isAuto {
		// Explicit selection
		selection.SelectionMode = "explicit"
		norm := normalizeProvider(reqTrimmed)
		if norm == "" {
			if d, ok := allDiscoveries[reqTrimmed]; ok {
				norm = d.Name
			} else {
				selection.PolicyDecision = "REQUIRE_APPROVAL"
				selection.SelectionReason = fmt.Sprintf("unknown or unsupported harness %q; supported options are: codex, antigravity, auto", reqTrimmed)
				return nil, selection, fmt.Errorf("invalid harness %q; supported harnesses are: codex, antigravity, auto", reqTrimmed)
			}
		}

		d, exists := allDiscoveries[norm]
		if !exists || !d.Available {
			reason := "not installed or not available on PATH"
			if exists && d.Reason != "" {
				reason = d.Reason
			}
			selection.PolicyDecision = "REQUIRE_APPROVAL"
			selection.SelectionReason = fmt.Sprintf("explicitly requested harness %q is unavailable: %s", reqTrimmed, reason)
			var alternatives []string
			for _, name := range availableList {
				if name != "manual" {
					alternatives = append(alternatives, name)
				}
			}
			if len(alternatives) == 0 {
				alternatives = []string{"none (install codex or agy)"}
			}
			return nil, selection, fmt.Errorf("requested harness %q is unavailable: %s (available alternatives: %s)", reqTrimmed, reason, strings.Join(alternatives, ", "))
		}

		// Explicit harness is available
		selection.SelectedHarness = norm
		selection.SelectionReason = fmt.Sprintf("user explicitly requested %s", norm)
		selection.PolicyDecision = "ALLOW"
		cfg := Config{
			Name:           norm,
			Provider:       norm,
			Command:        defaultProviderCommand(norm),
			TimeoutSeconds: 300,
		}
		if configuredCfg, err := LoadConfig(root); err == nil && (configuredCfg.Name == norm || configuredCfg.Provider == norm) {
			cfg = configuredCfg
		}
		return NewAdapter(ctx, root, cfg), selection, nil
	}

	// Auto selection mode
	selection.SelectionMode = "auto"
	selection.RequestedHarness = "auto"

	// 1. Check if configured harness.json exists and is valid
	if cfg, err := LoadConfig(root); err == nil {
		if d, ok := availableMap[cfg.Name]; ok && d.Available {
			selection.SelectedHarness = cfg.Name
			selection.SelectionReason = fmt.Sprintf("auto-selected configured harness %q from .keystone/harness.json", cfg.Name)
			selection.PolicyDecision = "ALLOW"
			return NewAdapter(ctx, root, cfg), selection, nil
		}
	}

	// 2. In auto mode, pick an available provider (prefer antigravity or codex)
	if d, ok := availableMap["antigravity"]; ok && d.Available {
		selection.SelectedHarness = "antigravity"
		selection.SelectionReason = "auto-selected antigravity (available on PATH and authenticated)"
		selection.PolicyDecision = "ALLOW"
		cfg := Config{Name: "antigravity", Provider: "antigravity", Command: defaultProviderCommand("antigravity"), TimeoutSeconds: 300}
		return NewAdapter(ctx, root, cfg), selection, nil
	}

	if d, ok := availableMap["codex"]; ok && d.Available {
		selection.SelectedHarness = "codex"
		selection.SelectionReason = "auto-selected codex (available on PATH)"
		selection.PolicyDecision = "ALLOW"
		cfg := Config{Name: "codex", Provider: "codex", Command: defaultProviderCommand("codex"), TimeoutSeconds: 300}
		return NewAdapter(ctx, root, cfg), selection, nil
	}

	// No provider available in auto mode
	selection.PolicyDecision = "REQUIRE_APPROVAL"
	selection.SelectionReason = "no executable harness is available on PATH"
	return nil, selection, fmt.Errorf("no executable harness is available; install and configure codex or agy")
}

func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex":
		return "codex"
	case "agy", "antigravity", "antigravity-ide":
		return "antigravity"
	case "local", "local-process", "process":
		return ""
	default:
		return ""
	}
}

func defaultProviderCommand(provider string) string {
	if normalizeProvider(provider) == "antigravity" {
		return "agy"
	}
	return "codex"
}

func probeCommand(command string) (string, string, bool) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Sprintf("command %q is not on PATH: %v", command, err), false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	output, err := cmd.CombinedOutput()
	version := strings.TrimSpace(redact(string(output)))
	if ctx.Err() != nil {
		return path, fmt.Sprintf("version probe timed out after 2s"), false
	}
	if err != nil {
		if version == "" {
			version = err.Error()
		}
		return path, fmt.Sprintf("version probe failed: %s", version), false
	}
	return path, version, true
}

func providerCapabilities(provider string) (capabilities, control, observability, evidence, limitations []string) {
	base := []string{"start", "observe", "interrupt", "resume", "result", "stdout-observation", "session-resume", "usage", "tool-events", "file-events", "command-events"}
	control = []string{"start", "resume", "interrupt"}
	observability = []string{"jsonl-events", "turn-events", "tool-events", "file-events", "command-events", "usage"}
	evidence = []string{"session-events", "tool-events", "file-events", "command-events", "provider-result"}
	limitations = []string{"provider CLI execution is one-shot per turn; in-turn Send is unavailable and follow-up uses provider session resume"}
	switch normalizeProvider(provider) {
	case "codex":
		return base, control, observability, evidence, limitations
	case "antigravity":
		return base, control, observability, evidence, limitations
	default:
		return []string{"start", "send", "observe", "interrupt", "resume", "result", "stdout-observation"}, []string{"start", "send", "resume", "interrupt"}, []string{"stdout"}, []string{"stdout-observation"}, nil
	}
}

func discoveryForConfig(cfg Config) Discovery {
	provider := normalizeProvider(cfg.Provider)
	if provider == "" {
		provider = normalizeProvider(cfg.Name)
	}
	if provider != "" {
		caps, control, observability, evidence, limitations := providerCapabilities(provider)
		path, version, available := probeCommand(cfg.Command)
		_ = path
		item := Discovery{Name: cfg.Name, Kind: provider, Available: available, Provider: provider, Version: version, Capabilities: append(caps, cfg.Capabilities...), Control: control, Observability: observability, Evidence: evidence, Limitations: limitations}
		if !available {
			item.Reason = version
		}
		return item
	}
	path, reason, available := probeCommand(cfg.Command)
	_ = path
	item := Discovery{Name: cfg.Name, Kind: "local-process", Available: available, Capabilities: append([]string{"start", "send", "observe", "interrupt", "resume", "result", "stdout-observation"}, cfg.Capabilities...)}
	if !available {
		item.Reason = reason
	}
	return item
}

type ProcessAdapter struct {
	Config   Config
	ctx      context.Context
	cancel   context.CancelFunc
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	out      *bufio.Reader
	runID    string
	started  bool
	finished bool
	result   domain.Status
	mu       sync.Mutex
}

type Local = ProcessAdapter

func NewProcessAdapter(ctx context.Context, cfg Config) *ProcessAdapter {
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 300
	}
	child, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
	return &ProcessAdapter{Config: cfg, ctx: child, cancel: cancel, result: domain.StatusUnknown}
}
func NewLocal(ctx context.Context, cfg Config) *ProcessAdapter {
	return NewProcessAdapter(ctx, cfg)
}
func (a *ProcessAdapter) Discover() error {
	if a.Config.Command == "" {
		return fmt.Errorf("empty local harness command")
	}
	return nil
}
func (a *ProcessAdapter) HarnessID() string {
	if a.Config.Name != "" {
		return a.Config.Name
	}
	return "local-process"
}
func (a *ProcessAdapter) Capabilities() []string {
	capabilities := append([]string(nil), a.Config.Capabilities...)
	return append([]string{"start", "send", "observe", "interrupt", "resume", "result", "stdout-observation"}, capabilities...)
}
func (a *ProcessAdapter) Start(packet domain.WorkPacket) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.started {
		return "", fmt.Errorf("harness already started")
	}
	a.cmd = exec.CommandContext(a.ctx, a.Config.Command, a.Config.Args...)
	stdin, err := a.cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	stdout, err := a.cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	a.cmd.Stderr = a.cmd.Stdout
	a.stdin = stdin
	a.out = bufio.NewReader(stdout)
	if err := a.cmd.Start(); err != nil {
		return "", err
	}
	a.started = true
	a.runID = fmt.Sprintf("RUN-%d", time.Now().UnixNano())
	if err := a.sendLocked(Render(packet)); err != nil {
		_ = a.cmd.Process.Kill()
		return "", err
	}
	return a.runID, nil
}
func (a *ProcessAdapter) sendLocked(s string) error {
	if a.stdin == nil {
		return io.ErrClosedPipe
	}
	_, err := io.WriteString(a.stdin, s+"\n")
	return err
}
func (a *ProcessAdapter) Send(s string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.started || a.cmd == nil {
		return io.ErrClosedPipe
	}
	return a.sendLocked(s)
}
func (a *ProcessAdapter) Observe() ([]domain.Observation, error) {
	a.mu.Lock()
	out := a.out
	run := a.runID
	a.mu.Unlock()
	if out == nil {
		return nil, io.ErrClosedPipe
	}
	line, err := out.ReadString('\n')
	if len(line) == 0 && err != nil {
		if err == io.EOF {
			a.finish()
		}
		return nil, err
	}
	line = strings.TrimSpace(line)
	typ := classify(line)
	obs := domain.Observation{ID: fmt.Sprintf("OBS-%d", time.Now().UnixNano()), RunID: run, Type: typ, Summary: redact(line), Timestamp: time.Now().UTC(), Payload: map[string]any{"raw": redact(line)}}
	return []domain.Observation{obs}, nil
}
func (a *ProcessAdapter) finish() {
	a.mu.Lock()
	if a.finished {
		a.mu.Unlock()
		return
	}
	a.finished = true
	c := a.cmd
	stdin := a.stdin
	a.mu.Unlock()
	if stdin != nil {
		_ = stdin.Close()
	}
	if c != nil {
		if err := c.Wait(); err == nil {
			a.mu.Lock()
			a.result = domain.StatusCompleted
			a.mu.Unlock()
		} else {
			a.mu.Lock()
			a.result = domain.StatusFailed
			a.mu.Unlock()
		}
	}
	a.cancel()
}
func (a *ProcessAdapter) Interrupt() error {
	a.mu.Lock()
	c := a.cmd
	a.mu.Unlock()
	if c == nil || c.Process == nil {
		return nil
	}
	err := c.Process.Kill()
	a.mu.Lock()
	a.result = domain.StatusFailed
	a.mu.Unlock()
	return err
}
func (a *ProcessAdapter) Resume(checkpoint domain.Checkpoint) error {
	b, _ := json.Marshal(checkpoint)
	return a.Send("Resume from durable checkpoint: " + string(b))
}
func (a *ProcessAdapter) Result() (domain.Status, error) {
	a.mu.Lock()
	finished := a.finished
	result := a.result
	a.mu.Unlock()
	if !finished {
		a.finish()
		a.mu.Lock()
		result = a.result
		a.mu.Unlock()
	}
	return result, nil
}

func classify(s string) string {
	low := strings.ToLower(s)
	switch {
	case strings.HasPrefix(low, "[tool_started]"):
		return "TOOL_STARTED"
	case strings.HasPrefix(low, "[tool_completed]"):
		return "TOOL_COMPLETED"
	case strings.HasPrefix(low, "[command_started]"):
		return "COMMAND_STARTED"
	case strings.HasPrefix(low, "[command_completed]"):
		return "COMMAND_COMPLETED"
	case strings.HasPrefix(low, "[file_read]"):
		return "FILE_READ"
	case strings.HasPrefix(low, "[file_changed]"):
		return "FILE_CHANGED"
	case strings.Contains(low, "error") || strings.Contains(low, "failed"):
		return "ERROR"
	case strings.Contains(low, "test") && (strings.Contains(low, "pass") || strings.Contains(low, "fail")):
		return "TEST_COMPLETED"
	case strings.Contains(low, "done") || strings.Contains(low, "complete") || strings.Contains(low, "implemented"):
		return "COMPLETION_CLAIM"
	default:
		return "MESSAGE_RECEIVED"
	}
}
func redact(s string) string {
	for _, key := range []string{"api_key", "apikey", "token", "password", "secret"} {
		offset := 0
		for offset < len(s) {
			low := strings.ToLower(s)
			rel := strings.Index(low[offset:], key)
			if rel < 0 {
				break
			}
			i := offset + rel
			end := strings.IndexAny(s[i+len(key):], " \t,;\n")
			if end < 0 {
				end = len(s) - i - len(key)
			}
			end += i + len(key)
			s = s[:i] + key + "=[REDACTED]" + s[end:]
			offset = i + len(key) + len("=[REDACTED]")
		}
	}
	return s
}
