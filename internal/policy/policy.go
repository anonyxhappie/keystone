package policy

import (
	"path/filepath"
	"strings"

	"github.com/anonyxhappie/keystone/internal/domain"
)

func Evaluate(action string) domain.PolicyDecision {
	gated := map[string]string{
		"high_risk_request":     "high-risk execution requires explicit approval",
		"production_deploy":     "production deployment requires explicit policy/approval",
		"destructive_migration": "destructive migration requires approval",
		"destructive_data":      "destructive data operation requires approval",
		"credential_change":     "credential changes are restricted",
		"force_push":            "force push is prohibited by default",
	}
	if reason, ok := gated[action]; ok {
		return domain.PolicyDecision{Decision: "APPROVE", Reason: reason, Policy: "default-v1", Allowed: false, RequiresApproval: true}
	}
	return domain.PolicyDecision{Decision: "CONTINUE", Reason: "action is permitted by default policy", Policy: "default-v1", Allowed: true}
}

// Command evaluates an argv vector without invoking a shell. It is deliberately
// conservative because execution authority is separate from planning authority.
func Command(argv []string, root string) domain.PolicyDecision {
	return command(argv, root, false)
}

// CommandWithApproval evaluates the same command after an explicit approval
// has been recorded for the current run. Approval is an input to policy, not
// an implicit side effect of a blocked command.
func CommandWithApproval(argv []string, root string, approved bool) domain.PolicyDecision {
	return command(argv, root, approved)
}

func command(argv []string, root string, approved bool) domain.PolicyDecision {
	if len(argv) == 0 {
		return domain.PolicyDecision{Decision: "BLOCK", Reason: "empty command", Policy: "default-v1"}
	}
	joined := strings.ToLower(strings.Join(argv, " "))
	for _, token := range []string{" rm ", "rm -rf", "git reset --hard", "git clean -f", "git push --force", "force-push", "drop table", "truncate table"} {
		if strings.Contains(" "+joined+" ", token) {
			if approved {
				return domain.PolicyDecision{Decision: "CONTINUE", Reason: "explicit approval recorded for potentially destructive command", Policy: "default-v1", Allowed: true, RequiresApproval: true}
			}
			return domain.PolicyDecision{Decision: "APPROVE", Reason: "potentially destructive command requires explicit approval", Policy: "default-v1", RequiresApproval: true}
		}
	}
	for _, arg := range argv[1:] {
		if filepath.IsAbs(arg) && root != "" {
			cleanRoot, _ := filepath.Abs(root)
			cleanArg, _ := filepath.Abs(arg)
			if cleanArg != cleanRoot && !strings.HasPrefix(cleanArg, cleanRoot+string(filepath.Separator)) {
				return domain.PolicyDecision{Decision: "BLOCK", Reason: "command path escapes workspace", Policy: "default-v1"}
			}
		}
	}
	return domain.PolicyDecision{Decision: "CONTINUE", Reason: "command is permitted by default policy", Policy: "default-v1", Allowed: true}
}
