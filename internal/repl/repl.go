package repl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/control"
	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/harness"
	"github.com/anonyxhappie/keystone/internal/session"
	"github.com/anonyxhappie/keystone/internal/state"
	"github.com/anonyxhappie/keystone/internal/ui"
	"github.com/anonyxhappie/keystone/internal/validation"
	"github.com/anonyxhappie/keystone/internal/work"
)

// REPL manages the persistent interactive terminal session.
type REPL struct {
	root           string
	harness        string
	sessionID      string
	term           *ui.Terminal
	editor         *ui.PromptEditor
	cachedSessions []session.Session
	cachedProjects []session.Project
	in             io.Reader
	out            io.Writer
}

// New creates a new interactive REPL session.
func New(root, defaultHarness string, in io.Reader, out io.Writer) *REPL {
	if defaultHarness == "" {
		defaultHarness = "auto"
	}
	term := ui.New(out)
	editor := ui.NewPromptEditor(in, out, defaultHarness, "", "ready")
	r := &REPL{
		root:    root,
		harness: defaultHarness,
		term:    term,
		editor:  editor,
		in:      in,
		out:     out,
	}
	editor.SetSuggestionProvider(r.provideSuggestions)
	return r
}

// Run starts the persistent interactive loop. It only terminates when the user manually exits.
func (r *REPL) Run() error {
	r.printWelcome()

	for {
		r.editor.SetContext(r.harness, r.sessionID, "ready")
		line, err := r.editor.ReadLine()
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(r.out, "Goodbye.")
				break
			}
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for exit
		if line == "/exit" || line == "/quit" || line == "exit" || line == "quit" {
			fmt.Fprintln(r.out, "Exiting Keystone interactive session. Goodbye!")
			break
		}

		// Slash commands
		if strings.HasPrefix(line, "/") {
			r.handleSlashCommand(line)
			continue
		}

		// Text prompt
		r.executePrompt(line)
	}

	return nil
}

func (r *REPL) printWelcome() {
	account := detectAccount()
	modelInfo := detectModel(r.harness)
	ui.PrintBanner(r.out, r.root, r.harness, account, modelInfo)
}

func detectAccount() string {
	if home, err := os.UserHomeDir(); err == nil {
		p := filepath.Join(home, ".gemini", "google_accounts.json")
		if data, err := os.ReadFile(p); err == nil {
			var acc struct {
				Active *string  `json:"active"`
				Old    []string `json:"old"`
			}
			if err := json.Unmarshal(data, &acc); err == nil {
				if acc.Active != nil && *acc.Active != "" {
					return *acc.Active + " (Google AI Pro)"
				}
				if len(acc.Old) > 0 {
					return acc.Old[0] + " (Google AI Pro)"
				}
			}
		}
	}
	return "Keystone Supervisor"
}

func detectModel(harnessName string) string {
	if harnessName == "antigravity" || harnessName == "agy" {
		return "Gemini 3.8 Flash (High)"
	}
	if harnessName == "codex" {
		return "Codex (o3 / High)"
	}
	return "Supervised Autonomy"
}

func (r *REPL) formatPrompt() string {
	sessBadge := ""
	if r.sessionID != "" {
		shortID := r.sessionID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		sessBadge = fmt.Sprintf(":%s", shortID)
	}
	harnessBadge := r.harness
	return fmt.Sprintf("%s%skeystone [%s%s]%s > ", ui.Bold, ui.Cyan, harnessBadge, sessBadge, ui.Reset)
}

func (r *REPL) handleSlashCommand(input string) {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/help":
		r.showHelp()
	case "/harness":
		r.handleHarness(args)
	case "/sessions", "/conversations", "/history":
		r.handleListSessions()
	case "/resume", "/continue":
		r.handleResume(args)
	case "/new":
		r.sessionID = ""
		fmt.Fprintf(r.out, "%s Started fresh conversation context.\n\n", ui.Green+"✔"+ui.Reset)
	case "/projects":
		r.handleListProjects()
	case "/cd", "/project":
		r.handleSwitchProject(args)
	case "/status":
		r.handleStatus()
	case "/verify", "/validate":
		r.handleVerify()
	case "/review":
		r.handleReview()
	case "/clear":
		fmt.Fprint(r.out, "\033[H\033[2J")
	default:
		fmt.Fprintf(r.out, "%s Unknown command %q. Type %s for available commands.\n\n", ui.Yellow+"▲"+ui.Reset, cmd, ui.Bold+"/help"+ui.Reset)
	}
}

func (r *REPL) showHelp() {
	fmt.Fprintf(r.out, "\n%s\n", ui.Bold+"Available Slash Commands:"+ui.Reset)
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/sessions"+ui.Reset, "List conversations from Keystone and installed harnesses")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/resume [id|#]"+ui.Reset, "Resume an existing conversation by ID or index number")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/new"+ui.Reset, "Start a brand-new conversation (resets active session)")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/harness [name]"+ui.Reset, "View or switch active harness (antigravity, codex, auto)")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/projects"+ui.Reset, "List local code projects and workspaces")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/project [path|#]"+ui.Reset, "Switch active workspace to another project")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/status"+ui.Reset, "Inspect current project state and latest checkpoint")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/verify"+ui.Reset, "Run deterministic validation checks (tests/linters) on demand")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/review"+ui.Reset, "Show supervisor findings, drift detection, and advice")
	fmt.Fprintf(r.out, "  %-24s %s\n", ui.Cyan+"/clear"+ui.Reset, "Clear terminal screen")
	fmt.Fprintf(r.out, "  %-24s %s\n\n", ui.Cyan+"/exit"+ui.Reset, "Exit the interactive session")
}

func (r *REPL) handleHarness(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(r.out, "Active harness: %s\nAvailable harnesses: codex, antigravity, auto\n\n", ui.Bold+ui.Cyan+r.harness+ui.Reset)
		return
	}
	target := strings.ToLower(strings.TrimSpace(args[0]))
	if target != "codex" && target != "antigravity" && target != "agy" && target != "auto" {
		fmt.Fprintf(r.out, "%s Invalid harness %q. Valid options: codex, antigravity, auto\n\n", ui.Red+"✘"+ui.Reset, target)
		return
	}
	if target == "agy" {
		target = "antigravity"
	}
	r.harness = target
	fmt.Fprintf(r.out, "%s Switched active harness to %s.\n\n", ui.Green+"✔"+ui.Reset, ui.Bold+ui.Cyan+r.harness+ui.Reset)
}

func (r *REPL) handleListSessions() {
	sessions := session.DiscoverSessions(r.root)
	r.cachedSessions = sessions
	if len(sessions) == 0 {
		fmt.Fprintf(r.out, "No prior sessions found for Keystone or local harnesses.\n\n")
		return
	}

	fmt.Fprintf(r.out, "\n%s\n", ui.Bold+"Recent Conversations & Sessions:"+ui.Reset)
	fmt.Fprintf(r.out, "%-4s %-12s %-38s %s\n", "#", "Harness", "Session ID", "Title / Preview")
	fmt.Fprintln(r.out, strings.Repeat("─", 80))

	limit := len(sessions)
	if limit > 15 {
		limit = 15
	}
	for i := 0; i < limit; i++ {
		s := sessions[i]
		activeMark := " "
		if s.ID == r.sessionID {
			activeMark = "*"
		}
		title := s.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		fmt.Fprintf(r.out, "%s%-3d %-12s %-38s %s\n", activeMark, i+1, ui.Cyan+s.Harness+ui.Reset, s.ID, ui.Dim+title+ui.Reset)
	}
	fmt.Fprintf(r.out, "\nType %s to resume any session.\n\n", ui.Bold+"/resume <#|id>"+ui.Reset)
}

func (r *REPL) handleResume(args []string) {
	if len(args) == 0 {
		r.handleListSessions()
		return
	}

	target := args[0]
	// Check if numeric index
	if idx, err := strconv.Atoi(target); err == nil {
		if len(r.cachedSessions) == 0 {
			r.cachedSessions = session.DiscoverSessions(r.root)
		}
		if idx >= 1 && idx <= len(r.cachedSessions) {
			s := r.cachedSessions[idx-1]
			r.sessionID = s.ID
			r.harness = s.Harness
			fmt.Fprintf(r.out, "%s Resumed session #%d: %s (%s: %s)\n\n", ui.Green+"✔"+ui.Reset, idx, s.ID, s.Harness, s.Title)
			return
		}
		fmt.Fprintf(r.out, "%s Invalid session index %d. Run %s to see valid numbers.\n\n", ui.Red+"✘"+ui.Reset, idx, ui.Bold+"/sessions"+ui.Reset)
		return
	}

	// Target is explicit session ID
	r.sessionID = target
	fmt.Fprintf(r.out, "%s Set active conversation session to %s.\n\n", ui.Green+"✔"+ui.Reset, r.sessionID)
}

func (r *REPL) handleListProjects() {
	projects := session.DiscoverProjects(r.root)
	r.cachedProjects = projects

	fmt.Fprintf(r.out, "\n%s\n", ui.Bold+"Local Code Projects:"+ui.Reset)
	fmt.Fprintf(r.out, "%-4s %-24s %s\n", "#", "Project Name", "Path")
	fmt.Fprintln(r.out, strings.Repeat("─", 70))

	limit := len(projects)
	if limit > 15 {
		limit = 15
	}
	for i := 0; i < limit; i++ {
		p := projects[i]
		activeMark := " "
		if p.Active {
			activeMark = "*"
		}
		fmt.Fprintf(r.out, "%s%-3d %-24s %s\n", activeMark, i+1, ui.Bold+p.Name+ui.Reset, ui.Dim+p.Path+ui.Reset)
	}
	fmt.Fprintf(r.out, "\nType %s to switch workspace.\n\n", ui.Bold+"/project <#|path>"+ui.Reset)
}

func (r *REPL) handleSwitchProject(args []string) {
	if len(args) == 0 {
		r.handleListProjects()
		return
	}

	target := args[0]
	if idx, err := strconv.Atoi(target); err == nil {
		if len(r.cachedProjects) == 0 {
			r.cachedProjects = session.DiscoverProjects(r.root)
		}
		if idx >= 1 && idx <= len(r.cachedProjects) {
			target = r.cachedProjects[idx-1].Path
		} else {
			fmt.Fprintf(r.out, "%s Invalid project index %d.\n\n", ui.Red+"✘"+ui.Reset, idx)
			return
		}
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(r.out, "%s Invalid path %q: %v\n\n", ui.Red+"✘"+ui.Reset, target, err)
		return
	}
	if info, err := os.Stat(absPath); err != nil || !info.IsDir() {
		fmt.Fprintf(r.out, "%s Directory does not exist: %s\n\n", ui.Red+"✘"+ui.Reset, absPath)
		return
	}

	r.root = absPath
	r.sessionID = "" // reset conversation on workspace switch
	fmt.Fprintf(r.out, "%s Switched active workspace to: %s\n\n", ui.Green+"✔"+ui.Reset, ui.Bold+r.root+ui.Reset)
}

func (r *REPL) handleStatus() {
	e, err := control.Open(r.root)
	if err != nil {
		fmt.Fprintf(r.out, "%s Failed to open project: %v\n\n", ui.Red+"✘"+ui.Reset, err)
		return
	}
	snap, err := e.Store.LoadSnapshot()
	if err != nil {
		fmt.Fprintf(r.out, "Project: %s\nNo active run state.\n\n", r.root)
		return
	}
	fmt.Fprintf(r.out, "\n%s\n", ui.Bold+"Keystone Project Status:"+ui.Reset)
	fmt.Fprintf(r.out, "  • Workspace: %s\n", r.root)
	fmt.Fprintf(r.out, "  • Lifecycle: %s\n", ui.Bold+ui.Cyan+snap.Lifecycle+ui.Reset)
	fmt.Fprintf(r.out, "  • Last Run:  %s\n", snap.RunID)
	fmt.Fprintf(r.out, "  • Harness:   %s (Session: %s)\n", snap.HarnessID, snap.HarnessSessionID)
	if snap.LastError != "" {
		fmt.Fprintf(r.out, "  • Note:      %s\n", ui.Yellow+snap.LastError+ui.Reset)
	}
	fmt.Fprintln(r.out)
}

func (r *REPL) handleVerify() {
	fmt.Fprintf(r.out, "%s Running deterministic project validations...\n", ui.Cyan+"▶"+ui.Reset)
	var p domain.Project
	if err := state.New(r.root).Read("project.json", &p); err != nil {
		fmt.Fprintf(r.out, "%s Project not initialized: %v\n\n", ui.Red+"✘"+ui.Reset, err)
		return
	}
	plan := validation.PlanFor(domain.Risk{Level: "low"}, p.Capabilities)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	for _, c := range validation.Commands(plan, r.root, p.Capabilities) {
		res := validation.Run(ctx, c)
		if res.Passed {
			fmt.Fprintf(r.out, "  %s %s (%s)\n", ui.Green+"✔"+ui.Reset, res.Name, time.Duration(res.Duration).Round(time.Millisecond))
		} else {
			fmt.Fprintf(r.out, "  %s %s (failed: exit code %d)\n", ui.Red+"✘"+ui.Reset, ui.Bold+res.Name+ui.Reset, res.ExitCode)
		}
	}
	fmt.Fprintln(r.out)
}

func (r *REPL) handleReview() {
	e, err := control.Open(r.root)
	if err != nil {
		fmt.Fprintf(r.out, "%s Failed to open project: %v\n\n", ui.Red+"✘"+ui.Reset, err)
		return
	}
	snap, err := e.Store.LoadSnapshot()
	if err != nil {
		fmt.Fprintf(r.out, "No snapshot available for review.\n\n")
		return
	}
	fmt.Fprintf(r.out, "\n%s\n", ui.Bold+"Supervisor Review Findings:"+ui.Reset)
	if len(snap.Findings) == 0 {
		fmt.Fprintf(r.out, "  %s No active architectural blockers or anomalies detected.\n\n", ui.Green+"✔"+ui.Reset)
		return
	}
	for _, f := range snap.Findings {
		fmt.Fprintf(r.out, "  • [%s] %s: %s\n", f.Severity, ui.Bold+f.Type+ui.Reset, f.Explanation)
	}
	fmt.Fprintln(r.out)
}

func (r *REPL) executePrompt(prompt string) {
	e, err := control.Open(r.root)
	if err != nil {
		fmt.Fprintf(r.out, "%s Engine init error: %v\n\n", ui.Red+"✘"+ui.Reset, err)
		return
	}

	e.RequestedHarness = r.harness
	e.ResumeSessionID = r.sessionID
	e.AdapterFactory = func() harness.Adapter {
		if r.harness != "" && r.harness != "auto" {
			adapter, _, err := harness.SelectHarness(context.Background(), r.root, r.harness)
			if err == nil && adapter != nil {
				return adapter
			}
		}
		cfg, err := harness.LoadConfig(r.root)
		if err != nil {
			return nil
		}
		return harness.NewAdapter(context.Background(), r.root, cfg)
	}

	r.term.Header(r.root, r.harness, work.IsReadOnlyRequest(prompt), prompt)
	e.OnEvent = r.term.OnEvent
	e.OnObservation = r.term.OnObservation

	report, runErr := e.Run(context.Background(), prompt, nil)
	if runErr != nil {
		fmt.Fprintf(r.out, "\n%s Keystone run error: %v\n", ui.Red+"✘"+ui.Reset, runErr)
	}

	// Update session ID if established
	if report.HarnessSessionID != "" {
		r.sessionID = report.HarnessSessionID
	}

	summaries := []ui.ValidationSummary{}
	for _, v := range report.Validations {
		summary := ""
		if !v.Passed {
			if strings.Contains(v.Stderr, "Can't reach database server") || strings.Contains(v.Stdout, "Can't reach database server") {
				summary = "Environment blocker: PostgreSQL not reachable at localhost:5432"
			} else if strings.Contains(v.Stdout, "new blank line at EOF") {
				summary = "Git style: trailing blank line at EOF"
			} else if v.Stderr != "" {
				summary = strings.Split(strings.TrimSpace(v.Stderr), "\n")[0]
			} else {
				summary = fmt.Sprintf("exit code %d", v.ExitCode)
			}
		}
		summaries = append(summaries, ui.ValidationSummary{
			Name:    v.Name,
			Passed:  v.Passed,
			Summary: summary,
		})
	}

	r.term.Report(
		report.RunID,
		report.WorkOrderID,
		report.HarnessID,
		report.HarnessSessionID,
		string(report.State),
		string(report.NextAction.Type),
		report.NextAction.Reason,
		report.ReadOnly,
		len(report.Mutations),
		report.ContextTokens,
		report.Attempts,
		report.Error,
		summaries,
	)

	// Interactive approval prompt if blocked or requiring approval
	if report.NextAction.Type == "ASK" || report.NextAction.RequiresApproval {
		if r.term.PromptApproval(report.NextAction.Reason) {
			_ = e.Store.Write(fmt.Sprintf("approvals/APP-%d.json", time.Now().UnixNano()), domain.Approval{
				ID:         fmt.Sprintf("APP-%d", time.Now().UnixNano()),
				Action:     "CONTINUE",
				RunID:      report.RunID,
				ApprovedBy: "interactive-user",
				Reason:     "interactive approval in shell",
				At:         time.Now().UTC(),
			})
			fmt.Fprintf(r.out, "%s Approval recorded. Continuing execution...\n", ui.Green+"✔"+ui.Reset)
			_, _ = e.Continue(context.Background(), nil)
		}
	}
}

func (r *REPL) provideSuggestions(input string) []ui.CommandItem {
	if !strings.HasPrefix(input, "/") {
		return nil
	}

	// 1. /project or /cd with space
	if strings.HasPrefix(input, "/project ") || strings.HasPrefix(input, "/cd ") {
		query := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(input, "/project "), "/cd ")))
		if len(r.cachedProjects) == 0 {
			r.cachedProjects = session.DiscoverProjects(r.root)
		}
		var items []ui.CommandItem
		for i, p := range r.cachedProjects {
			label := fmt.Sprintf("%d. %s", i+1, p.Name)
			if p.Active {
				label += " *"
			}
			if query == "" || strings.Contains(strings.ToLower(p.Name), query) || strings.Contains(strings.ToLower(p.Path), query) {
				items = append(items, ui.CommandItem{
					Command:     label,
					Description: p.Path,
					InsertText:  fmt.Sprintf("/project %d", i+1),
					Immediate:   true,
				})
			}
		}
		return items
	}

	// 2. /resume or /continue with space
	if strings.HasPrefix(input, "/resume ") || strings.HasPrefix(input, "/continue ") {
		query := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(input, "/resume "), "/continue ")))
		if len(r.cachedSessions) == 0 {
			r.cachedSessions = session.DiscoverSessions(r.root)
		}
		var items []ui.CommandItem
		limit := len(r.cachedSessions)
		if limit > 20 {
			limit = 20
		}
		for i := 0; i < limit; i++ {
			s := r.cachedSessions[i]
			shortID := s.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			label := fmt.Sprintf("%d. %s (%s)", i+1, s.Harness, shortID)
			if query == "" || strings.Contains(strings.ToLower(s.Title), query) || strings.Contains(strings.ToLower(s.ID), query) || strings.Contains(strings.ToLower(s.Harness), query) {
				items = append(items, ui.CommandItem{
					Command:     label,
					Description: s.Title,
					InsertText:  fmt.Sprintf("/resume %d", i+1),
					Immediate:   true,
				})
			}
		}
		return items
	}

	// 3. /harness with space
	if strings.HasPrefix(input, "/harness ") {
		harnesses := []struct {
			name string
			desc string
		}{
			{"antigravity", "Google Antigravity CLI (Gemini 3.8 Flash)"},
			{"codex", "OpenAI Codex CLI (o3-mini)"},
			{"auto", "Auto-select best available harness"},
		}
		query := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(input, "/harness ")))
		var items []ui.CommandItem
		for _, h := range harnesses {
			if query == "" || strings.Contains(h.name, query) {
				items = append(items, ui.CommandItem{
					Command:     h.name,
					Description: h.desc,
					InsertText:  "/harness " + h.name,
					Immediate:   true,
				})
			}
		}
		return items
	}

	// 4. Default: slash commands matching prefix
	lower := strings.ToLower(input)
	var matched []ui.CommandItem
	for _, c := range ui.DefaultSlashCommands {
		if strings.HasPrefix(strings.ToLower(c.Command), lower) {
			matched = append(matched, c)
		}
	}
	return matched
}
