package validation

import "github.com/anonyxhappie/keystone/internal/domain"

type Plan struct {
	Tier   int
	Checks []string
}

func PlanFor(r domain.Risk, capabilities []domain.Capability) Plan {
	tier := 0
	if r.Level == "medium" {
		tier = 1
	}
	if r.Level == "high" {
		tier = 2
	}
	if r.Level == "release" {
		tier = 4
	}
	checks := []string{"git-diff"}
	for _, c := range capabilities {
		switch c.Kind {
		case "test":
			checks = append(checks, c.Name)
		case "browser":
			if tier >= 2 {
				checks = append(checks, c.Name)
			}
		}
	}
	return Plan{Tier: tier, Checks: checks}
}
