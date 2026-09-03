# Keystone Architecture V1

## Product boundary

Keystone is a persistent engineering intelligence/control layer between human intent and existing AI development/operations harnesses. It does not replace the harness. The harness executes work; Keystone owns durable state, context, supervision, verification, policy and continuation.

## Core loop

```text
USER REQUEST
 → REQUEST UNDERSTANDING
 → PROJECT STATE + CONTEXT
 → IMPACT / RISK / POLICY
 → WORK PACKET
 → EXISTING HARNESS
 → OBSERVATIONS / ARTIFACTS
 → SUPERVISOR
 → CONTINUE / CORRECT / REPLAN / ASK / STOP
```

Assist mode exposes the next action to the user. Full-auto continues ordinary policy-allowed work and stops at consequential approval, permission, security or unrecoverable-failure boundaries.

## Durable project model

The repository is the portable project boundary. `.keystone/` stores project-specific state and may contain:

```text
project.json
state.json
requirements/
architecture/
decisions/
assumptions/
work/
prompts/
checkpoints/
evidence/
learning/
policies/
manifests/
```

The core entities are Project, Requirement, Decision, Assumption, WorkOrder, WorkPacket, Prompt, HarnessRun, Observation, Artifact, Evidence, Finding, PolicyDecision, Checkpoint, Learning, Release and Capability.

Keystone distinguishes user requirements, observed facts, decisions, assumptions, agent proposals and derived recommendations. Critical state cannot exist only in conversation history. Every harness prompt is durably archived in `.keystone/prompts/{promptID}.json` with explicit turn reason, strategy, hypothesis, expected information gain, and context manifest provenance.

## Technology neutrality

The core does not assume a programming language, framework, IDE, architecture, model provider or harness. Initialization discovers project capabilities and adapters expose technology-specific behavior.

Typical capabilities include language, build, test, browser, database, CI/CD, infrastructure, Git, security and observability.

## Context compiler

Context is progressive:

1. core project identity, active work, requirements, policy and checkpoint;
2. directly relevant architecture, decisions, files, tests and evidence;
3. historical or large artifacts only when required.

Every context item has provenance. Large artifacts remain outside ordinary model context.

## Harness adapters

The stable boundary is conceptually:

```text
Discover → Capabilities → Start → Send → Observe → Interrupt → Resume → Result
```

Adapters may range from instruction-file/manual integration through provider-specific programmatic control. The core must not depend on a particular harness protocol.

## Evidence and validation

Completion is evidence-based, not claim-based. Deterministic observations include Git state, diffs, tests, builds, browser output, logs and configured tool results. Model reasoning may interpret evidence, but must not be the sole basis for high-risk completion, security, destructive data operations or release decisions.

Validation is risk-aware:

```text
Tier 0 static/config/type/lint
Tier 1 affected tests
Tier 2 targeted browser/system validation
Tier 3 milestone regression
Tier 4 release audit
```

Evidence is scoped and invalidated when relevant inputs change.

## Supervisor

The supervisor evaluates outcome and harness behavior. V1 signals include unsupported completion claims, requirement drift, architecture drift, repeated/no-progress loops, unexpected scope, failed validation and excessive repeated activity.

Intervention is minimal: observe normal behavior, record minor inefficiency, optimize repeated inefficiency, intervene on drift, verify unsupported completion, replan loops, and stop/gate security or destructive risk.

## Multi-turn control loop and prompt dispatch

Each harness turn follows a strict canonical event sequence:
`ContextCompiled` → `PromptGenerated` → `PromptDispatched` → `HarnessTurnStarted` → `HarnessObserved` → `HarnessTurnCompleted` → `SupervisorEvaluation` → `NextAction`.

Harness sessions are preserved across turns via `HARNESS_SESSION_RESUMED`. Keystone never generates synthetic harness switch events when the active harness remains the same.

## Environment blocker recovery ("Let the harness fix it")

When deterministic validation encounters an environmental failure (e.g. PostgreSQL, MySQL, Redis, MongoDB, or local port unavailable), Keystone classifies the failure as an `ENVIRONMENT_BLOCKER`. If recoverable by the harness using project configuration (e.g. docker-compose, service restart, local scripts), Keystone generates a targeted recovery prompt instructing the harness to diagnose and recover the local service, verify connectivity, and rerun validation.

If consecutive attempts repeat the exact same failure without code or environment progress, Keystone's supervisor flags a `Loop` finding and halts with an `ASK` decision rather than burning attempts in an infinite retry cycle.

## Policy and authority

Intelligence/planning authority, execution authority and approval authority are separate. Default policy gates production deployment, destructive migrations/data operations, credential changes and force-pushes. Silence is never approval.

## Checkpoints and switching

A checkpoint is a machine-readable continuation contract containing work state, completed/pending validation, changed files, context manifest, unresolved questions and blockers. Context exhaustion, harness failure and model/harness switching use the same recovery mechanism.

A new harness must resume from durable state without the original conversation.

## Learning

Learning is versioned, observable, evidence-backed and reversible. It may be scoped to a project, harness/model or project×harness combination. Learning candidates can improve context selection, strategy, validation and intervention, but cannot silently rewrite core policy.

## V1 boundary

V1 establishes initialization, state, capability detection, work orders, context compilation, harness adapter contracts, observation/evidence, deterministic validation, supervision, policy, checkpoints/recovery, basic learning and Git safety. A mandatory vector database, agent swarm, custom model, cloud service and production auto-remediation are deferred.

## North star

> Keep the engineering thread intact: understand intent, give the harness what it needs, observe what actually happened, recognize failure, correct course, learn from the outcome, and continue until the project reaches a verified state.
