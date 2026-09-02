# Keystone V1 → V2 Roadmap

This document defines the single runtime state machine used from V1.0 through V2.0. Every version adds capabilities to the same lifecycle; no version introduces a competing orchestration model.

## Canonical lifecycle

```text
REQUEST
→ UNDERSTAND
→ ASSESS
→ PLAN
→ CONTEXT
→ DISPATCH
→ EXECUTE
→ OBSERVE
→ VERIFY
→ EVALUATE
→ SUPERVISE
→ DECIDE
→ CONTINUE / CORRECT / REPLAN / ASK / APPROVE / STOP / COMPLETE
```

Control states are durable and replayable. A harness is always an execution dependency, never the system of record.

## Release progression

### V1.0 — Foundation

Durable project state, work orders, context compilation, policy, checkpoints, validation planning, evidence and Git safety foundations.

### V1.1 — Observe

Real harness boundary, dispatch/observe loop, normalized run events, CLI continuation and pause controls.

**Success condition:** Keystone can observe an externally executed engineering run and continue from persisted state.

### V1.2 — Verify

Evidence graph, completion-claim verification, evidence invalidation, deterministic validation execution, requirement-to-evidence traceability.

**Success condition:** "done" means verified, not merely claimed by a harness.

### V1.3 — Judge

Behavioral supervision for unsupported claims, requirement/architecture drift, repeated/no-progress behavior, unexpected scope and inefficient tool activity.

**Success condition:** Keystone can recognize a failing or low-quality execution pattern before blindly continuing it.

### V1.4 — Act

Full-auto execution, bounded retries, next-action selection, approval boundaries, safe stop, resume, correction and replan.

**Success condition:** a normal request can progress through multiple harness interactions without manual prompt orchestration.

### V1.5 — Learn

Project, harness/model and project×harness learning records; candidate/evaluate/activate lifecycle; outcome metrics.

**Success condition:** repeated work measurably improves future context, strategy or routing.

### V1.6 — Switch

Multiple harness adapters, capability-aware routing, checkpoint-safe model/harness switching, adapter health/performance metrics.

**Success condition:** work can move between compatible harnesses without losing state.

### V1.7 — Understand

Deeper project model: repository topology, dependency hints, instructions, architecture/decision indexing, richer context ranking and invalidation.

**Success condition:** task context is derived from project relationships rather than broad repository dumping.

### V1.8 — Deliver

Release candidates, CI/CD observations, deployment state, changelog/release evidence, operational capabilities and incident-as-work support.

**Success condition:** the same lifecycle can govern implementation through release and deployment.

### V1.9 — Trust

Event-sourced state reconstruction, idempotent operations, replay, stronger permission boundaries, secret redaction, failure-injection coverage and supervisor quality metrics.

**Success condition:** Keystone can explain and recover from its own failures and cannot silently corrupt project state.

### V2.0 — Control

Unified engineering control plane with stable APIs/contracts across lifecycle, harnesses, validation, policy, evidence, learning and optional production operations.

**Success condition:** a project can be operated through Keystone with a replaceable harness layer and a persistent, evidence-backed engineering control loop.

## Non-negotiable invariants

- One canonical state machine from V1.0 through V2.0.
- User intent is preserved verbatim.
- No silent destructive Git operations.
- No high-risk action without explicit policy/approval.
- Evidence outranks harness claims.
- Derived state is reconstructable from durable inputs.
- Learning is versioned, evidence-backed and reversible.
- Full-auto means autonomous within policy, never unrestricted authority.
- Harnesses/models remain replaceable.
- Token optimization may not reduce semantic sufficiency or explicit release gates.
