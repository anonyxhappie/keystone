package policy

import "github.com/anonyxhappie/keystone/internal/domain"

func Evaluate(action string) domain.PolicyDecision {
	gated := map[string]string{
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
