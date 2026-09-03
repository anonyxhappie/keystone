package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/observation"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgBlue   = "\033[44m"
)

// Terminal renders live streaming progress and human-readable reports.
type Terminal struct {
	out       io.Writer
	noColor   bool
	startTime time.Time
}

// New creates a new Terminal renderer.
func New(out io.Writer) *Terminal {
	noColor := os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
	return &Terminal{
		out:       out,
		noColor:   noColor,
		startTime: time.Now(),
	}
}

func (t *Terminal) color(code, text string) string {
	if t.noColor {
		return text
	}
	return code + text + Reset
}

func (t *Terminal) timestamp() string {
	elapsed := time.Since(t.startTime)
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	return t.color(Dim, fmt.Sprintf("[%02d:%02d]", mins, secs))
}

// Header displays the run banner with project and harness info.
func (t *Terminal) Header(root, requestedHarness string, readOnly bool, request string) {
	harnessLabel := requestedHarness
	if harnessLabel == "" || harnessLabel == "auto" {
		harnessLabel = "auto"
	}
	modeLabel := "Standard"
	if readOnly {
		modeLabel = t.color(Green, "Read-Only (enforced: 0 mutations allowed)")
	}

	reqDisplay := request
	if len(reqDisplay) > 90 {
		reqDisplay = reqDisplay[:87] + "..."
	}

	fmt.Fprintln(t.out)
	fmt.Fprintf(t.out, "%s\n", t.color(Bold+Cyan, "╭── Keystone ─────────────────────────────────────────────────────────────"))
	fmt.Fprintf(t.out, "%s  %s  %s\n", t.color(Cyan, "│"), t.color(Bold, "Workspace:"), root)
	fmt.Fprintf(t.out, "%s  %s    %s\n", t.color(Cyan, "│"), t.color(Bold, "Harness:"), t.color(Cyan, harnessLabel))
	fmt.Fprintf(t.out, "%s  %s     %s\n", t.color(Cyan, "│"), t.color(Bold, "Policy:"), modeLabel)
	fmt.Fprintf(t.out, "%s  %s    %s\n", t.color(Cyan, "│"), t.color(Bold, "Request:"), t.color(Italic, fmt.Sprintf("%q", reqDisplay)))
	fmt.Fprintf(t.out, "%s\n\n", t.color(Bold+Cyan, "╰─────────────────────────────────────────────────────────────────────────"))
}

// OnEvent formats live engine events as they occur.
func (t *Terminal) OnEvent(ev observation.Event) {
	ts := t.timestamp()
	switch ev.Type {
	case "REQUEST_ACCEPTED":
		fmt.Fprintf(t.out, "%s %s Request accepted (Order: %s)\n", ts, t.color(Cyan, "▶"), ev.Payload["workOrderId"])
	case "GIT_BASELINE_DETAILED":
		preRun := ev.Payload["preRunFiles"]
		fmt.Fprintf(t.out, "%s %s Git baseline captured (%v uncommitted/dirty files tracked for safe rollback)\n", ts, t.color(Green, "🔒"), preRun)
	case "CONTEXT_PLAN":
		fmt.Fprintf(t.out, "%s %s Context compiled (%v / 20,000 tokens)\n", ts, t.color(Blue, "📦"), ev.Payload["tokens"])
	case "HARNESS_SELECTED":
		harness := ev.Payload["selectedHarness"]
		mode := ev.Payload["selectionMode"]
		fmt.Fprintf(t.out, "%s %s Authoritative harness selected: %s (%s)\n", ts, t.color(Bold+Cyan, "🚀"), t.color(Bold+White, fmt.Sprint(harness)), mode)
	case "RUN_DISPATCHED":
		session := ev.Payload["sessionId"]
		fmt.Fprintf(t.out, "%s %s Dispatched to harness (session: %v)\n", ts, t.color(Yellow, "⚡"), session)
	case "RUN_STOPPED":
		status := ev.Payload["status"]
		fmt.Fprintf(t.out, "%s %s Harness execution turn finished (status: %v)\n", ts, t.color(Cyan, "🏁"), status)
	case "MUTATIONS_DETECTED":
		count := ev.Payload["count"]
		fmt.Fprintf(t.out, "%s %s Repository mutations detected: %v files modified\n", ts, t.color(Yellow, "⚠️"), count)
	case "MUTATIONS_RESTORED":
		restored := ev.Payload["restored"]
		fmt.Fprintf(t.out, "%s %s Non-destructive rollback: restored %v files to baseline\n", ts, t.color(Green, "🔒"), restored)
	case "POLICY_DECISION":
		action := ev.Payload["action"]
		decision := ev.Payload["decision"]
		if decision == "REQUIRE_APPROVAL" {
			fmt.Fprintf(t.out, "%s %s Policy gate [%s]: %s (%v)\n", ts, t.color(Yellow, "▲"), action, t.color(Yellow, fmt.Sprint(decision)), ev.Payload["reason"])
		} else if decision == "STOP" {
			fmt.Fprintf(t.out, "%s %s Policy enforcement [%s]: STOP (%v)\n", ts, t.color(Red, "⛔"), action, ev.Payload["reason"])
		}
	case "DECISION":
		decision := ev.Payload["decision"]
		reason := ev.Payload["reason"]
		fmt.Fprintf(t.out, "%s %s Lifecycle decision: %s (%v)\n", ts, t.color(Bold, "⚖️"), t.color(Bold+Cyan, fmt.Sprint(decision)), reason)
	case "HARNESS_SWITCHED":
		fmt.Fprintf(t.out, "%s %s Switched harness: %v → %v (Attempt %v)\n", ts, t.color(Yellow, "🔄"), ev.Payload["from"], ev.Payload["to"], ev.Payload["attempt"])
	case "HARNESS_SESSION_RESUMED":
		fmt.Fprintf(t.out, "%s %s Harness session resumed: %v (Attempt %v)\n", ts, t.color(Green, "↳ 🔁"), ev.Payload["sessionId"], ev.Payload["attempt"])
	case "PromptDispatched":
		if reason, ok := ev.Payload["reason"].(string); ok && reason != "" {
			fmt.Fprintf(t.out, "   %s %s Keystone direction: %s\n", ts, t.color(Bold+Magenta, "↳ 🛡️"), reason)
		}
	case "FAILURE_DIAGNOSIS":
		if summary, ok := ev.Payload["summary"].(string); ok && summary != "" {
			fmt.Fprintf(t.out, "   %s %s Supervisor diagnosis: %s\n", ts, t.color(Bold+Red, "↳ 🔍"), summary)
		}
	case "HARNESS_REQUEST_ANSWERED":
		req := ev.Payload["request"]
		ans := ev.Payload["answer"]
		reason := ev.Payload["reason"]
		fmt.Fprintf(t.out, "   %s %s Harness requested: %v\n   %s %s Keystone answered: %v (Reason: %v)\n", ts, t.color(Yellow, "↳ ❓"), req, ts, t.color(Green, "↳ 🛡️"), ans, reason)
	}
}

// OnObservation formats real-time streaming harness tool calls and stdout.
func (t *Terminal) OnObservation(obs domain.Observation) {
	ts := t.timestamp()
	switch obs.Type {
	case "TOOL_STARTED":
		summary := obs.Summary
		if len(summary) > 120 {
			summary = summary[:117] + "..."
		}
		fmt.Fprintf(t.out, "   %s %s %s\n", ts, t.color(Cyan, "↳ ⚡ Tool:"), summary)
	case "TOOL_COMPLETED":
		summary := obs.Summary
		if len(summary) > 120 {
			summary = summary[:117] + "..."
		}
		fmt.Fprintf(t.out, "   %s %s %s\n", ts, t.color(Green, "↳ ✔ Result:"), summary)
	case "COMMAND_STARTED":
		summary := obs.Summary
		if len(summary) > 120 {
			summary = summary[:117] + "..."
		}
		fmt.Fprintf(t.out, "   %s %s %s\n", ts, t.color(Yellow, "↳ 💻"), t.color(Dim, summary))
	case "FILE_TOUCHED", "FILE_CHANGED":
		fmt.Fprintf(t.out, "   %s %s %s\n", ts, t.color(Green, "↳ 📄"), obs.Summary)
	case "HARNESS_REQUEST":
		fmt.Fprintf(t.out, "   %s %s Harness requested: %s\n", ts, t.color(Yellow, "↳ ❓"), obs.Summary)
	case "COMPLETION_CLAIM":
		lines := strings.Split(strings.TrimSpace(obs.Summary), "\n")
		if len(lines) == 1 && len(obs.Summary) <= 120 {
			fmt.Fprintf(t.out, "   %s %s %s\n", ts, t.color(Bold+Cyan, "↳ 💬"), t.color(Italic, obs.Summary))
		} else {
			fmt.Fprintf(t.out, "   %s %s\n", ts, t.color(Bold+Cyan, "↳ 💬 Assistant:"))
			for _, line := range lines {
				lineTrimmed := strings.TrimRight(line, " \t\r")
				if lineTrimmed != "" {
					fmt.Fprintf(t.out, "     %s\n", lineTrimmed)
				} else {
					fmt.Fprintln(t.out)
				}
			}
		}
	case "MESSAGE_RECEIVED", "AGENT_RESPONSE":
		lines := strings.Split(strings.TrimSpace(obs.Summary), "\n")
		for _, line := range lines {
			lineTrimmed := strings.TrimRight(line, " \t\r")
			if lineTrimmed != "" {
				fmt.Fprintf(t.out, "   %s %s %s\n", ts, t.color(Bold+Green, "↳ 🤖"), lineTrimmed)
			}
		}
	case "SESSION_STARTED":
		fmt.Fprintf(t.out, "   %s %s %s\n", ts, t.color(Blue, "↳ 🔗"), obs.Summary)
	case "STDOUT":
		lines := strings.Split(strings.TrimSpace(obs.Summary), "\n")
		for _, line := range lines {
			lineTrimmed := strings.TrimSpace(line)
			if lineTrimmed != "" {
				if len(lineTrimmed) > 100 {
					lineTrimmed = lineTrimmed[:97] + "..."
				}
				fmt.Fprintf(t.out, "   %s %s %s\n", ts, t.color(Dim, "│"), t.color(Dim, lineTrimmed))
			}
		}
	}
}

// ValidationSummary holds clean check results for terminal display.
type ValidationSummary struct {
	Name    string
	Passed  bool
	Summary string
}

// Report renders the final executive summary card.
func (t *Terminal) Report(
	runID, workOrderID, harnessID, sessionID string,
	state string,
	nextActionType, nextActionReason string,
	readOnly bool,
	mutationsCount int,
	contextTokens int,
	attempts int,
	maxAttempts int,
	errorMsg string,
	validations []ValidationSummary,
) {
	fmt.Fprintln(t.out)
	var statusBadge string
	switch state {
	case "COMPLETE":
		statusBadge = t.color(BgGreen+White+Bold, " RUN COMPLETE ")
	case "STOPPED":
		statusBadge = t.color(BgYellow+White+Bold, " RUN STOPPED ")
	case "BLOCKED":
		statusBadge = t.color(BgRed+White+Bold, " RUN BLOCKED ")
	default:
		statusBadge = t.color(BgBlue+White+Bold, fmt.Sprintf(" %s ", state))
	}

	fmt.Fprintf(t.out, "%s\n", t.color(Bold, "═════════════════════════════════════════════════════════════════════════"))
	fmt.Fprintf(t.out, "  %s   Run: %s\n", statusBadge, t.color(Bold+White, runID))
	fmt.Fprintf(t.out, "%s\n", t.color(Bold, "─────────────────────────────────────────────────────────────────────────"))
	fmt.Fprintf(t.out, "  • %-18s %s\n", "Harness:", t.color(Cyan, harnessID))
	if sessionID != "" {
		fmt.Fprintf(t.out, "  • %-18s %s\n", "Harness Session:", sessionID)
	}
	if maxAttempts <= 0 {
		maxAttempts = 6
	}
	fmt.Fprintf(t.out, "  • %-18s %d of %d\n", "Attempts:", attempts, maxAttempts)

	if readOnly {
		if mutationsCount == 0 {
			fmt.Fprintf(t.out, "  • %-18s %s\n", "Repository:", t.color(Green, "✔ 0 mutations (read-only verified, user dirty state intact)"))
		} else {
			fmt.Fprintf(t.out, "  • %-18s %s\n", "Repository:", t.color(Red, fmt.Sprintf("✘ %d forbidden mutations detected and safely reverted", mutationsCount)))
		}
	}
	if contextTokens > 0 {
		fmt.Fprintf(t.out, "  • %-18s %s\n", "Context Tokens:", fmt.Sprintf("%d / 20,000", contextTokens))
	}

	if len(validations) > 0 {
		fmt.Fprintf(t.out, "\n  %s\n", t.color(Bold, "Deterministic Validation Results:"))
		for _, v := range validations {
			if v.Passed {
				fmt.Fprintf(t.out, "    %s %s\n", t.color(Green, "✔"), v.Name)
			} else {
				reason := v.Summary
				if reason == "" {
					reason = "failed"
				}
				fmt.Fprintf(t.out, "    %s %s (%s)\n", t.color(Red, "✘"), t.color(Bold, v.Name), t.color(Dim, reason))
			}
		}
	}

	if errorMsg != "" {
		fmt.Fprintf(t.out, "\n  %s %s\n", t.color(Bold, "Outcome Note:"), t.color(Yellow, errorMsg))
	}

	fmt.Fprintf(t.out, "  %s %s\n", t.color(Bold, "Next Action:"), t.color(Bold+Cyan, nextActionType))
	if nextActionReason != "" {
		fmt.Fprintf(t.out, "  %s %s\n", t.color(Bold, "Reason:"), nextActionReason)
	}

	fmt.Fprintf(t.out, "\n  %s\n", t.color(Bold, "Audit Trail & Artifacts:"))
	fmt.Fprintf(t.out, "    • State:     %s\n", t.color(Dim, ".keystone/state.json"))
	fmt.Fprintf(t.out, "    • Events:    %s\n", t.color(Dim, fmt.Sprintf(".keystone/events.jsonl (view with `keystone replay %s`)", runID)))
	fmt.Fprintf(t.out, "    • Review:    %s\n", t.color(Dim, "Run `keystone review` for structured findings"))
	fmt.Fprintf(t.out, "%s\n\n", t.color(Bold, "═════════════════════════════════════════════════════════════════════════"))
}

// PromptApproval asks the user interactively for confirmation if stdin is available.
func (t *Terminal) PromptApproval(reason string) bool {
	fmt.Fprintf(t.out, "\n%s %s\n", t.color(Yellow+Bold, "? Action requires approval:"), reason)
	fmt.Fprintf(t.out, "%s ", t.color(Bold, "Would you like to approve and proceed? [y/N]:"))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	resp := strings.ToLower(strings.TrimSpace(response))
	return resp == "y" || resp == "yes"
}
