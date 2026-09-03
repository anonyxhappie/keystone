package diagnosis

import (
	"testing"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/validation"
)

func TestClassifyPostgreSQLEnvironmentBlocker(t *testing.T) {
	validations := []validation.Result{
		{
			Name:     "go",
			Passed:   false,
			ExitCode: 1,
			Stderr:   "Can't reach database server at 'localhost':'5432'\nPlease make sure your database server is running at 'localhost':'5432'.",
		},
	}
	d := Classify(nil, validations, domain.StatusCompleted, nil, nil, false)
	if d.Class != domain.FailureEnvironmentBlocker {
		t.Fatalf("expected ENVIRONMENT_BLOCKER, got %s", d.Class)
	}
	if !d.RecoverableByHarness {
		t.Fatalf("expected RecoverableByHarness=true")
	}
	if d.RequiresHuman {
		t.Fatalf("expected RequiresHuman=false")
	}
	if d.Target != "postgresql://localhost:5432" {
		t.Fatalf("unexpected target: %s", d.Target)
	}
	if d.RecoveryInstruction == "" {
		t.Fatalf("expected non-empty recovery instruction")
	}
}

func TestClassifyReadOnlyPolicyViolation(t *testing.T) {
	mutations := []domain.FileMutation{
		{Path: "foo.go", Action: "created"},
	}
	d := Classify(nil, nil, domain.StatusCompleted, nil, mutations, true)
	if d.Class != domain.FailurePolicyBlock {
		t.Fatalf("expected POLICY_BLOCK, got %s", d.Class)
	}
	if d.RecoverableByHarness {
		t.Fatalf("expected RecoverableByHarness=false for read-only violation")
	}
	if !d.RequiresHuman {
		t.Fatalf("expected RequiresHuman=true for read-only violation")
	}
}

func TestClassifyTestFailure(t *testing.T) {
	validations := []validation.Result{
		{
			Name:     "go",
			Passed:   false,
			ExitCode: 1,
			Stderr:   "--- FAIL: TestUserAuth (0.01s)\n    auth_test.go:42: expected token but got nil",
		},
	}
	d := Classify(nil, validations, domain.StatusCompleted, nil, nil, false)
	if d.Class != domain.FailureTestFailure {
		t.Fatalf("expected TEST_FAILURE, got %s", d.Class)
	}
	if !d.RecoverableByHarness {
		t.Fatalf("expected RecoverableByHarness=true")
	}
}

func TestClassifyCodeError(t *testing.T) {
	validations := []validation.Result{
		{
			Name:     "go-build",
			Passed:   false,
			ExitCode: 1,
			Stderr:   "./main.go:10:2: syntax error: unexpected newline",
		},
	}
	d := Classify(nil, validations, domain.StatusCompleted, nil, nil, false)
	if d.Class != domain.FailureCodeError {
		t.Fatalf("expected CODE_ERROR, got %s", d.Class)
	}
	if !d.RecoverableByHarness {
		t.Fatalf("expected RecoverableByHarness=true")
	}
}

func TestClassifyAuthFailure(t *testing.T) {
	obs := []domain.Observation{
		{
			Type:    "ERROR",
			Summary: "API_KEY missing from environment",
		},
	}
	d := Classify(obs, nil, domain.StatusFailed, nil, nil, false)
	if d.Class != domain.FailureExternalDependency {
		t.Fatalf("expected EXTERNAL_DEPENDENCY, got %s", d.Class)
	}
	if !d.RequiresHuman {
		t.Fatalf("expected RequiresHuman=true")
	}
}
