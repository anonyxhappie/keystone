package diagnosis

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/anonyxhappie/keystone/internal/domain"
	"github.com/anonyxhappie/keystone/internal/validation"
)

var (
	postgresRegex = regexp.MustCompile(`(?i)(can't reach database server|connection refused.*5432|connect.*5432.*refused|5432.*connection refused|5432.*connect.*refused|dial tcp.*5432|localhost:5432|postgresql.*not (running|reachable|accepting)|is the server running on that host and accepting TCP/IP connections|psql: error: connection to server on socket)`)
	mysqlRegex    = regexp.MustCompile(`(?i)(can't connect to MySQL server|connect.*3306.*refused|3306.*connection refused|3306.*connect.*refused|dial tcp.*3306|localhost:3306)`)
	redisRegex    = regexp.MustCompile(`(?i)(connection to redis.*refused|connect.*6379.*refused|6379.*connection refused|6379.*connect.*refused|dial tcp.*6379|localhost:6379)`)
	mongoRegex    = regexp.MustCompile(`(?i)(server selection error.*27017|connect.*27017.*refused|27017.*connection refused|27017.*connect.*refused|dial tcp.*27017|localhost:27017)`)
	dockerRegex   = regexp.MustCompile(`(?i)(cannot connect to the Docker daemon|is the docker daemon running|docker.*daemon.*not running)`)
	timeoutRegex  = regexp.MustCompile(`(?i)(timed out after|context deadline exceeded|operation timed out)`)
	authRegex     = regexp.MustCompile(`(?i)(authentication required|api_key missing|unauthorized|access denied|invalid credentials|oauth token expired|401 unauthorized)`)
)

// Classify inspects validation results, observations, git mutations, and status to produce a structured diagnosis.
func Classify(
	observations []domain.Observation,
	validations []validation.Result,
	status domain.Status,
	policyDecisions []domain.PolicyDecision,
	mutations []domain.FileMutation,
	readOnly bool,
) domain.FailureDiagnosis {
	// 1. Check read-only / policy violations first
	if readOnly && len(mutations) > 0 {
		return domain.FailureDiagnosis{
			Class:                domain.FailurePolicyBlock,
			Summary:              fmt.Sprintf("read-only policy violated: %d repository mutations detected", len(mutations)),
			Details:              fmt.Sprintf("%d forbidden mutations occurred during read-only execution", len(mutations)),
			RecoverableByHarness: false,
			RequiresHuman:        true,
			RecoveryInstruction:  "Read-only constraint violated. Mutations have been safely reverted to baseline. Do not modify files.",
		}
	}

	for _, pd := range policyDecisions {
		if !pd.Allowed || pd.RequiresApproval {
			return domain.FailureDiagnosis{
				Class:                domain.FailurePolicyBlock,
				Summary:              "policy boundary reached: " + pd.Reason,
				Details:              pd.Policy + ": " + pd.Reason,
				RecoverableByHarness: false,
				RequiresHuman:        true,
				RecoveryInstruction:  "Operation requires explicit human approval.",
			}
		}
	}

	// 2. Aggregate validation outputs
	var failedValidations []validation.Result
	var combinedText strings.Builder
	for _, v := range validations {
		if !v.Passed {
			failedValidations = append(failedValidations, v)
			combinedText.WriteString(v.Name + "\n" + v.Stdout + "\n" + v.Stderr + "\n")
		}
	}

	for _, o := range observations {
		if o.Type == "ERROR" {
			combinedText.WriteString(o.Summary + "\n")
		}
	}
	allText := combinedText.String()

	// 3. Environment Blockers: Databases, Docker, local ports
	if postgresRegex.MatchString(allText) {
		return domain.FailureDiagnosis{
			Class:                domain.FailureEnvironmentBlocker,
			Summary:              "PostgreSQL is unreachable at localhost:5432",
			Details:              extractMatchedLine(allText, postgresRegex),
			Target:               "postgresql://localhost:5432",
			RecoverableByHarness: true,
			RequiresHuman:        false,
			RecoveryInstruction: "PostgreSQL is unreachable at localhost:5432. Diagnose and recover the local database using the existing project setup/configuration (e.g. docker compose, service manager, or local initialization scripts). Verify connectivity at localhost:5432, then rerun the failed validation. Do not stop merely because the environment was unavailable. Do not repeat the same failed action without new diagnosis.",
		}
	}

	if mysqlRegex.MatchString(allText) {
		return domain.FailureDiagnosis{
			Class:                domain.FailureEnvironmentBlocker,
			Summary:              "MySQL is unreachable at localhost:3306",
			Details:              extractMatchedLine(allText, mysqlRegex),
			Target:               "mysql://localhost:3306",
			RecoverableByHarness: true,
			RequiresHuman:        false,
			RecoveryInstruction:  "MySQL is unreachable at localhost:3306. Diagnose and recover the local database using the existing project setup/configuration. Verify connectivity, then rerun the failed validation.",
		}
	}

	if redisRegex.MatchString(allText) {
		return domain.FailureDiagnosis{
			Class:                domain.FailureEnvironmentBlocker,
			Summary:              "Redis is unreachable at localhost:6379",
			Details:              extractMatchedLine(allText, redisRegex),
			Target:               "redis://localhost:6379",
			RecoverableByHarness: true,
			RequiresHuman:        false,
			RecoveryInstruction:  "Redis is unreachable at localhost:6379. Diagnose and recover the local Redis service using project configuration. Verify connectivity, then rerun validation.",
		}
	}

	if mongoRegex.MatchString(allText) {
		return domain.FailureDiagnosis{
			Class:                domain.FailureEnvironmentBlocker,
			Summary:              "MongoDB is unreachable at localhost:27017",
			Details:              extractMatchedLine(allText, mongoRegex),
			Target:               "mongodb://localhost:27017",
			RecoverableByHarness: true,
			RequiresHuman:        false,
			RecoveryInstruction:  "MongoDB is unreachable at localhost:27017. Diagnose and recover the local MongoDB service using project configuration. Verify connectivity, then rerun validation.",
		}
	}

	if dockerRegex.MatchString(allText) {
		return domain.FailureDiagnosis{
			Class:                domain.FailureEnvironmentBlocker,
			Summary:              "Docker daemon is not running or unreachable",
			Details:              extractMatchedLine(allText, dockerRegex),
			Target:               "docker",
			RecoverableByHarness: false,
			RequiresHuman:        true,
			RecoveryInstruction:  "Docker daemon is not accessible. Start Docker desktop/engine or provide local container runtime access.",
		}
	}

	// 4. External Dependency / Authentication / Secrets
	if authRegex.MatchString(allText) {
		return domain.FailureDiagnosis{
			Class:                domain.FailureExternalDependency,
			Summary:              "Authentication or credentials missing/expired",
			Details:              extractMatchedLine(allText, authRegex),
			RecoverableByHarness: false,
			RequiresHuman:        true,
			RecoveryInstruction:  "Missing or invalid credentials detected. Explicit user credential provision required.",
		}
	}

	// 5. Timeouts
	if timeoutRegex.MatchString(allText) {
		return domain.FailureDiagnosis{
			Class:                domain.FailureTimeout,
			Summary:              "Operation or validation timed out",
			Details:              extractMatchedLine(allText, timeoutRegex),
			RecoverableByHarness: true,
			RequiresHuman:        false,
			RecoveryInstruction:  "A command timed out. Investigate whether a process hung or infinite loop occurred, and optimize or isolate the slow step.",
		}
	}

	// 6. Test Failures
	if len(failedValidations) > 0 {
		firstFail := failedValidations[0]
		if isTestValidation(firstFail.Name) {
			testDetail := firstFail.Stderr
			if testDetail == "" {
				testDetail = firstFail.Stdout
			}
			return domain.FailureDiagnosis{
				Class:                domain.FailureTestFailure,
				Summary:              fmt.Sprintf("Test check %q failed", firstFail.Name),
				Details:              truncateString(testDetail, 500),
				Target:               firstFail.Name,
				RecoverableByHarness: true,
				RequiresHuman:        false,
				RecoveryInstruction:  fmt.Sprintf("Deterministic test failure in %q. Inspect the failure output and test assertions, identify the defect in the code, fix it, and verify that the tests pass.", firstFail.Name),
			}
		}

		// 7. Code Errors (build, compile, lint, typecheck)
		return domain.FailureDiagnosis{
			Class:                domain.FailureCodeError,
			Summary:              fmt.Sprintf("Validation check %q failed (exit code %d)", firstFail.Name, firstFail.ExitCode),
			Details:              truncateString(firstFail.Stderr+"\n"+firstFail.Stdout, 500),
			Target:               firstFail.Name,
			RecoverableByHarness: true,
			RequiresHuman:        false,
			RecoveryInstruction:  fmt.Sprintf("Validation failure in %q. Inspect the build/lint output, diagnose the syntax or configuration issue, apply the fix, and rerun validation.", firstFail.Name),
		}
	}

	// 8. Harness Agent / Tool Errors
	if status == domain.StatusFailed {
		return domain.FailureDiagnosis{
			Class:                domain.FailureAgentError,
			Summary:              "Harness turn failed or exited with error",
			Details:              truncateString(allText, 500),
			RecoverableByHarness: true,
			RequiresHuman:        false,
			RecoveryInstruction:  "The previous turn encountered an execution failure. Review error details, adjust the approach, and proceed.",
		}
	}

	// 9. Requirement Failure / Unsupported Completion Claim
	return domain.FailureDiagnosis{
		Class:                domain.FailureRequirementFailure,
		Summary:              "Validation did not confirm verified completion",
		RecoverableByHarness: true,
		RequiresHuman:        false,
		RecoveryInstruction:  "Work is incomplete or unverified. Execute required validations and satisfy all acceptance criteria.",
	}
}

func isTestValidation(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "test") || lower == "go" || lower == "vitest" || lower == "jest" || lower == "pytest" || lower == "cargo"
}

func extractMatchedLine(text string, re *regexp.Regexp) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if re.MatchString(trimmed) {
			return truncateString(trimmed, 200)
		}
	}
	return ""
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
