# Keystone Architecture V2.0

## Purpose

Keystone is a persistent engineering intelligence and control plane between human intent and replaceable execution harnesses. V2.0 does not replace those harnesses. It preserves the engineering thread while work moves through planning, execution, observation, verification, supervision, policy and recovery.

## Canonical state machine

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
→ CONTINUE / CORRECT / REPLAN / ASK / APPROVE / BLOCK / STOP / COMPLETE
```

The lifecycle is one state machine from V1.0 through V2.0. Features add capabilities to these states rather than creating independent agent runtimes.

## State authority

The durable system of record is an append-only event history plus deterministic state reduction. Materialized state is a cache/reconstruction of that history. Consequential operations use operation IDs and idempotency keys where possible.

The harness is not authoritative. A harness completion claim is an observation requiring verification.

## NextAction

Every non-terminal state resolves toward a machine-readable next action. An action contains its reason, inputs, risk, policy result, approval requirement and target. This makes assist mode and full-auto mode the same runtime with different authority policies.

## Evidence-first completion

Completion requires corroborating evidence appropriate to the risk and requirements. Evidence is scoped to relevant inputs and becomes stale when those inputs change. Typical evidence includes Git state, diffs, tests, builds, browser/system checks, CI results and operational observations.

## Supervisor

Supervisor findings cover correctness, requirement compliance, architecture fit, security, maintainability, test quality, UX where applicable, efficiency, tool discipline and methodology freshness. No single score is authoritative. High-risk decisions require deterministic evidence and/or explicit human approval according to policy.

## Harness portability

Adapters implement a stable lifecycle boundary. Provider-specific protocols stay outside the core. Checkpoints allow a run to move between compatible harnesses without replaying the original conversation.

## Learning

Learning is scoped, versioned, evidence-backed and reversible. Candidates are evaluated against measurable outcomes before activation. Learning may optimize context, validation, strategy or harness routing but cannot silently weaken policy.

## Lifecycle coverage

V2.0 covers software implementation, QA, data work, infrastructure, delivery, release and controlled operations wherever an adapter can expose the required observations and policy boundaries.

## Security and authority

Planning, execution and approval authority remain separate. Secrets are redacted before durable observation persistence where detectable. Workspace confinement, command allowlists, destructive-operation gates, production restrictions and explicit approvals are policy concerns, not model suggestions.

Silence is never approval.

## Failure recovery

Failures are modeled explicitly: harness failure, model failure, tool failure, validation failure, context exhaustion, policy block, permission/credential block and user-required decisions all produce durable checkpoints. Recovery reconstructs the next action from state and evidence rather than relying on conversation history.

## V2.0 boundary

Keystone remains local-first and technology-neutral. A vector database, custom foundation model, mandatory cloud service, agent swarm, or proprietary harness is not required for correctness of the core architecture.

## North star

> Keep the engineering thread intact from intent to verified project state, regardless of which AI harness performs the work.
