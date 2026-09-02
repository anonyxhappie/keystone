package validation

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type Command struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
	Tier int      `json:"tier"`
}
type Result struct {
	Name     string        `json:"name"`
	Tier     int           `json:"tier"`
	Passed   bool          `json:"passed"`
	ExitCode int           `json:"exitCode"`
	Duration time.Duration `json:"duration"`
	Stdout   string        `json:"stdout,omitempty"`
	Stderr   string        `json:"stderr,omitempty"`
}

// Run executes an explicit argv command. It never invokes a shell.
func Run(ctx context.Context, c Command) Result {
	r := Result{Name: c.Name, Tier: c.Tier}
	if len(c.Args) == 0 {
		r.Stderr = "empty command"
		r.ExitCode = -1
		return r
	}
	start := time.Now()
	cmd := exec.CommandContext(ctx, c.Args[0], c.Args[1:]...)
	var out, errout bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	r.Duration = time.Since(start)
	r.Stdout = out.String()
	r.Stderr = errout.String()
	if err == nil {
		r.Passed = true
		r.ExitCode = 0
		return r
	}
	if e, ok := err.(*exec.ExitError); ok {
		r.ExitCode = e.ExitCode()
	} else {
		r.ExitCode = -1
	}
	return r
}
