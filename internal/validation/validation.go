package validation

import "github.com/anonyxhappie/keystone/v2/internal/domain"

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
		case "build", "lint", "typecheck":
			checks = append(checks, c.Name)
		case "browser":
			if tier >= 2 {
				checks = append(checks, c.Name)
			}
		}
	}
	return Plan{Tier: tier, Checks: checks}
}

func Commands(plan Plan, root string, capabilities []domain.Capability) []Command {
	commands := make([]Command, 0, len(plan.Checks))
	for _, check := range plan.Checks {
		var args []string
		switch check {
		case "git-diff":
			if !hasCapability(capabilities, "vcs", "git") {
				continue
			}
			args = []string{"git", "diff", "HEAD", "--check", "--", ".", ":(exclude).keystone"}
		case "go":
			args = []string{"go", "test", "./..."}
		case "go-build":
			args = []string{"go", "build", "./..."}
		case "vitest":
			args = []string{"npm", "test", "--", "--run"}
		case "jest":
			args = []string{"npm", "test", "--", "--runInBand"}
		case "pytest":
			args = []string{"pytest"}
		case "playwright":
			args = []string{"npx", "--no-install", "playwright", "test"}
		case "eslint":
			args = []string{"npx", "--no-install", "eslint", "."}
		case "typescript":
			args = []string{"npx", "--no-install", "tsc", "--noEmit"}
		case "rust":
			args = []string{"cargo", "test"}
		default:
			continue
		}
		commands = append(commands, Command{Name: check, Args: args, Tier: plan.Tier, Dir: root})
	}
	return commands
}

func hasCapability(capabilities []domain.Capability, kind, name string) bool {
	for _, c := range capabilities {
		if c.Kind == kind && c.Name == name {
			return true
		}
	}
	return false
}
