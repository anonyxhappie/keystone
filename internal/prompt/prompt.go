package prompt

import (
	"fmt"
	"strings"
	"time"

	"github.com/anonyxhappie/keystone/internal/domain"
)

// Generate creates a structured, durable, evidence-backed prompt for a harness turn.
func Generate(
	order domain.WorkOrder,
	packet domain.WorkPacket,
	turn int,
	harnessID string,
	harnessSessionID string,
	contextManifest string,
	diagnosis *domain.FailureDiagnosis,
	retry *domain.RetryStrategy,
) domain.Prompt {
	now := time.Now().UTC()
	runID := order.RunID
	if runID == "" {
		runID = fmt.Sprintf("RUN-%d", now.UnixNano())
	}
	promptID := fmt.Sprintf("PRM-%s-%d", runID, turn)

	reason := "Initial work packet execution"
	strategy := "initial_implementation"
	hypothesis := "Harness will execute initial work packet and fulfill requirements"
	expectedInfoGain := "Implementation changes and deterministic verification evidence"

	if turn > 1 && diagnosis != nil {
		reason = diagnosis.Summary
		strategy = string(diagnosis.Class)
		hypothesis = fmt.Sprintf("Harness will resolve %s via targeted action", diagnosis.Summary)
		expectedInfoGain = "Verified resolution of previous failure"
	}

	if retry != nil {
		if retry.Reason != "" {
			reason = retry.Reason
		}
		if retry.Strategy != "" {
			strategy = retry.Strategy
		}
		if retry.Hypothesis != "" {
			hypothesis = retry.Hypothesis
		}
		if retry.ExpectedInfoGain != "" {
			expectedInfoGain = retry.ExpectedInfoGain
		}
	}

	var sb strings.Builder

	// 1. Role & Identity
	sb.WriteString("You are the primary worker executing engineering tasks under Keystone supervision.\n")
	sb.WriteString("Keystone supervises, verifies, and maintains project continuity; you perform actionable repository work using your tools.\n\n")

	// 2. Objective & Request
	sb.WriteString(fmt.Sprintf("User Request:\n%s\n\n", order.SourceRequest))

	// 3. Critical Policy Constraints
	if order.ReadOnly || packet.ReadOnly {
		sb.WriteString("CRITICAL POLICY CONSTRAINT: This is an explicit READ-ONLY execution.\n")
		sb.WriteString("Do NOT modify, create, or delete any files under any circumstances.\n")
		sb.WriteString("Inspect and report only. Keystone enforces this constraint via independent repository baseline inspection.\n\n")
	}

	if len(order.Constraints) > 0 {
		sb.WriteString("Execution Constraints:\n")
		for _, c := range order.Constraints {
			sb.WriteString(fmt.Sprintf("- %s\n", c))
		}
		sb.WriteString("\n")
	}

	// 4. Turn Direction & Guidance
	if turn == 1 {
		sb.WriteString("Turn #1 Direction:\n")
		sb.WriteString("Execute the requested work. Inspect repository files, understand project structure, make required changes, run tests, and report your findings.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("Turn #%d Direction: %s\n", turn, reason))
		if diagnosis != nil {
			sb.WriteString(fmt.Sprintf("Diagnosis: %s\n", diagnosis.Summary))
			if diagnosis.Details != "" {
				sb.WriteString(fmt.Sprintf("Failure Details:\n%s\n", diagnosis.Details))
			}
			if diagnosis.RecoveryInstruction != "" {
				sb.WriteString(fmt.Sprintf("Required Action:\n%s\n", diagnosis.RecoveryInstruction))
			}
		}
		sb.WriteString("\nImportant Turn Guidance:\n")
		sb.WriteString("- Do not repeat the same failed action without new diagnosis.\n")
		sb.WriteString("- If an environment dependency or local service was unavailable, diagnose and recover it using existing project configuration.\n")
		sb.WriteString("- Verify your changes before concluding.\n\n")
	}

	// 5. Requirements & Acceptance Criteria
	if len(order.Requirements) > 0 || len(packet.Requirements) > 0 {
		sb.WriteString("Requirements & Criteria:\n")
		for _, req := range packet.Requirements {
			sb.WriteString(fmt.Sprintf("- %s\n", req))
		}
		for _, crit := range packet.CompletionCriteria {
			sb.WriteString(fmt.Sprintf("- Criterion: %s\n", crit))
		}
		sb.WriteString("\n")
	}

	// 6. Context Manifest & Key Files
	if len(packet.Context) > 0 {
		sb.WriteString("Relevant Context Items:\n")
		for _, ctxRef := range packet.Context {
			sb.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", ctxRef.Type, ctxRef.Path, ctxRef.Reason))
		}
		sb.WriteString("\n")
	}

	// 7. Validation Criteria
	if len(packet.Validation) > 0 {
		sb.WriteString("Deterministic Validation Checks:\n")
		for _, v := range packet.Validation {
			sb.WriteString(fmt.Sprintf("- %s\n", v))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Report what you did, which tools/commands you executed, and what evidence supports completion. Do not declare completion without corroborating evidence.")

	return domain.Prompt{
		SchemaVersion:    "2",
		ID:               promptID,
		WorkOrderID:      order.ID,
		RunID:            runID,
		Turn:             turn,
		HarnessID:        harnessID,
		HarnessSessionID: harnessSessionID,
		ContextManifest:  contextManifest,
		Reason:           reason,
		Strategy:         strategy,
		Hypothesis:       hypothesis,
		ExpectedInfoGain: expectedInfoGain,
		Content:          sb.String(),
		Directive:        order.Directive,
		Dispatched:       false,
		CreatedAt:        now,
	}
}
