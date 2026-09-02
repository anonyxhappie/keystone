package validation

import (
	"context"
	"testing"
)

func TestRunUsesArgvAndWorkingDirectory(t *testing.T) {
	r := Run(context.Background(), Command{Name: "echo", Args: []string{"sh", "-c", "printf pass"}, Dir: t.TempDir()})
	if !r.Passed || r.Stdout != "pass" {
		t.Fatalf("unexpected result: %+v", r)
	}
}

func TestRunReportsFailure(t *testing.T) {
	r := Run(context.Background(), Command{Name: "fail", Args: []string{"sh", "-c", "exit 7"}})
	if r.Passed || r.ExitCode != 7 {
		t.Fatalf("unexpected failure result: %+v", r)
	}
}
