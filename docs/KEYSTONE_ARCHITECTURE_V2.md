# Keystone Architecture V2

## Purpose

Keystone is a persistent engineering intelligence and control layer between human intent and replaceable execution harnesses. It does **not** replace those harnesses. The harness performs the majority of actionable engineering work; Keystone maintains direction, context, durable memory, observation, verification, supervision, policy, and continuity.

The product-level behavioral source of truth is `docs/KEYSTONE_PRODUCT_MODEL.md`. This document defines the corresponding system architecture.

## Core product boundary

```text
HUMAN
  ↓ intent / decisions
KEYSTONE
  ↓ precise, evidence-derived instruction
EXISTING HARNESS
  ↓ work / tools / commands / files / results
KEYSTONE
  ↓ observe / verify / redirect
EXISTING HARNESS
  ↓ ...
VERIFIED PROJECT STATE
```

Keystone must not become a competing coding-agent runtime. Provider-specific execution remains behind adapters.

## Canonical control loop

A Keystone run is a long-lived feedback loop, not a single provider invocation:

```text
REQUEST
→ UNDERSTAND
→ ASSESS
→ PLAN
→ CONTEXT
→ GENERATE PROMPT
→ DISPATCH TO REAL HARNESS SESSION
→ EXECUTE
→ STREAM / OBSERVE
→ VERIFY
→ EVALUATE
→ SUPERVISE
→ DECIDE
→ CONTINUE / CORRECT / RECOVER / REPLAN / SWITCH / ASK / APPROVE / BLOCK / STOP / COMPLETE
```

When the decision requires more harness work, Keystone generates and actually dispatches a new prompt. When the provider supports native resume, that prompt continues the same real provider session.

## Prompt dispatch is a first-class lifecycle state

The canonical harness-turn event sequence is:

```text
ContextCompiled
→ PromptGenerated
→ PromptDispatched
→ HarnessTurnStarted
→ HarnessObserved
→ HarnessTurnCompleted
→ SupervisorEvaluation
→ NextAction
```

`PromptGenerated` is not `PromptDispatched`. Keystone must not claim dispatch unless the adapter has actually delivered the prompt to the provider and recorded the relevant acknowledgement/result when available.

Every prompt is durably associated with:

- WorkOrder;
- run;
- harness;
- real harness session;
- context manifest/version;
- reason;
- strategy/hypothesis when applicable;
- expected information gain when applicable;
- timestamp;
- dispatch result.

## Harness-first execution

The harness remains the primary worker.

Keystone should normally delegate actionable engineering work to the harness when it is capable of performing it. Keystone directly owns control-plane responsibilities such as context compilation, state persistence, policy decisions, checkpointing, evidence processing, deterministic validation, and supervision.

For recoverable environment failures, Keystone should instruct the harness to diagnose and recover the environment rather than repeatedly invoking the same failed validation or performing unnecessary implementation work itself.

Example:

```text
PostgreSQL unavailable
→ classify ENVIRONMENT_BLOCKER
→ determine harness can recover it
→ generate recovery prompt
→ dispatch to existing session
→ harness diagnoses / starts / repairs service
→ harness verifies connectivity
→ harness reruns affected validation
→ Keystone observes and verifies
→ continue
```

## State authority

The durable system of record is an append-only event history plus deterministic state reduction. Materialized state is a cache/reconstruction of that history. Consequential operations use operation IDs and idempotency keys where possible.

The harness is not authoritative. A harness completion claim is an observation requiring verification.

## Durable session model

A real provider session is a first-class durable object.

Persist at minimum:

- provider/harness;
- real provider session ID;
- project;
- human-readable title;
- status and timestamps;
- normalized conversation/events;
- current WorkOrder/run;
- prompt IDs;
- checkpoints;
- pending user input;
- usage/provider results.

Provider-native resume is preferred. A new session must not be created merely because Keystone is issuing another autonomous turn.

Synthetic harness sessions are not part of the model.

## Interactive conversation model

The interactive terminal/TUI is a rendering of the durable event stream.

The primary visible experience is the real harness conversation, including meaningful tool activity:

```text
USER
HARNESS
TOOL / COMMAND / FILE
HARNESS
KEYSTONE GUIDANCE
VALIDATION / EVIDENCE
...
```

Normalize events into at least:

```text
USER
KEYSTONE
HARNESS
TOOL
SYSTEM
VALIDATION
APPROVAL
ERROR
```

Internal lifecycle telemetry remains available in verbose/debug/replay views but must not replace or obscure the actual harness conversation.

## Session and project continuity

Interactive startup should restore the last valid:

- project;
- harness;
- session;
- mode;
- run/checkpoint;
- pending interaction.

`/sessions` must switch the entire active session state, including project, harness, conversation history, pending state, and checkpoint/run. Conversation history from the previous session must never remain mixed with the selected session.

Project switching must provide the same isolation guarantee.

## Human interaction

Full Auto and Assist use the same lifecycle with different authority policies.

Full Auto continues ordinary policy-allowed work across multiple harness turns without requiring the user to type `continue` after each turn.

If the harness asks a question or requests a decision that cannot be safely inferred, Keystone exposes the actual harness request and pauses. The user's answer becomes part of the same conversation and is sent back into the same provider session where supported. Meaningful rationale is persisted with the decision.

Human intervention is reserved for genuine boundaries such as destructive/irreversible actions, missing credentials, unresolved product decisions, security/policy boundaries, unsafe rollback, or unrecoverable external dependencies.

## NextAction

Every non-terminal state resolves toward a machine-readable next action. An action contains its reason, inputs, risk, policy result, approval requirement and target.

Important next actions include:

```text
CONTINUE
CORRECT
RECOVER
REPLAN
SWITCH
ASK
APPROVE
BLOCK
STOP
COMPLETE
```

A validation failure is not itself a reason to retry the same harness action.

## Failure classification

At minimum, execution/validation failures are classified as:

```text
AGENT_ERROR
CODE_ERROR
TEST_FAILURE
REQUIREMENT_FAILURE
ENVIRONMENT_BLOCKER
EXTERNAL_DEPENDENCY
TOOL_FAILURE
USER_DECISION
POLICY_BLOCK
TIMEOUT
NO_PROGRESS
UNKNOWN
```

Failure classification determines the next action. Environment blockers should first be delegated to the harness when the harness can safely recover them. Repeated identical failure without progress becomes `NO_PROGRESS` and triggers re-planning, justified harness switching, or human intervention.

## Retry semantics

A retry must represent a meaningful change in strategy, information, environment, or execution state.

Track evidence such as:

- previous error;
- previous prompt/action;
- commands/actions repeated;
- changed files/environment;
- new hypothesis;
- expected information gain.

Do not burn attempts by replaying an unchanged action with no new diagnosis.

## Evidence-first completion

Completion requires corroborating evidence appropriate to the risk and requirements. Evidence is scoped to relevant inputs and becomes stale when those inputs change.

Typical evidence includes Git state, diffs, tests, builds, browser/system checks, CI results, tool output, command results, file changes, logs, and operational observations.

Always distinguish:

```text
HARNESS CLAIM
OBSERVED EVIDENCE
KEYSTONE DECISION
```

A harness saying "done" is not proof of completion.

## Supervisor

Supervisor findings cover correctness, requirement compliance, architecture fit, security, maintainability, test quality, UX where applicable, efficiency, tool discipline, methodology freshness, unsupported completion claims, drift, loops, repeated failure, and unexpected scope.

Intervention should be minimal and useful:

```text
normal behavior        → observe
minor inefficiency     → record
repeated inefficiency  → optimize
requirement drift      → correct
unsupported claim      → verify
recoverable blocker    → guide harness
no-progress loop       → replan
high-risk action       → gate / approve
```

The supervisor should not manufacture blockers merely because more improvements are possible.

## Harness portability and selection

Adapters implement a stable lifecycle boundary. Provider-specific protocols remain outside the core.

Conceptual adapter boundary:

```text
Discover
→ Capabilities
→ Start
→ Send
→ Observe / Stream
→ Interrupt
→ Resume
→ Result
```

Explicit harness selection is authoritative. Auto mode may select/switch based on evidence.

```text
--harness codex
--harness antigravity
--harness auto
```

An unavailable explicitly selected harness must fail closed or ask; no silent substitution.

A switch is recorded only when the harness actually changes:

```text
Antigravity → Codex
reason: <evidence>
```

Resuming Antigravity is not an Antigravity → Antigravity switch.

## Context

Context is progressive:

1. project identity and active work;
2. directly relevant requirements, architecture, decisions, files, tests, policies and evidence;
3. historical or large artifacts only when required.

Context budgeting optimizes minimum sufficient context, not minimum token count. Native provider session continuity should be preferred to unnecessary repetition of established context.

## Policy and authority

Planning, execution, and approval authority remain separate. Secrets are redacted before durable observation persistence where detectable. Workspace confinement, command allowlists, destructive-operation gates, production restrictions, and explicit approvals are policy concerns, not model suggestions.

Silence is never approval.

## Read-only and workspace safety

For read-only requests, Keystone must enforce the constraint independently of the harness prompt.

Before execution it captures a baseline. After each relevant turn it detects mutations. Pre-existing user changes must be preserved. Current-run mutations may be reverted only when safe and attributable.

Never use blanket destructive rollback such as `git reset --hard` to enforce read-only safety.

## Checkpoints and recovery

A checkpoint is a machine-readable continuation contract containing work state, completed/pending validation, changed files, context manifest, unresolved questions, blockers, active harness/session, last prompt, and next action.

Recovery must work after process restart, harness crash, context exhaustion, or provider/model switching without depending on ephemeral conversation memory.

## Learning

Learning is scoped, versioned, evidence-backed and reversible. It may optimize context, validation, strategy, recovery, or harness routing but cannot silently weaken policy.

## Local-first boundary

Keystone remains local-first and technology-neutral. A vector database, custom foundation model, mandatory cloud service, agent swarm, or proprietary harness is not required for correctness of the core architecture.

## Definition of a working V2 control loop

The control loop is not complete merely because a provider process starts, a WorkOrder is created, a session ID is discovered, validation runs, or a harness reports completion.

The minimum proof is a real multi-turn interaction:

```text
USER REQUEST
→ Keystone generates prompt
→ prompt is actually dispatched
→ real harness receives it
→ harness performs meaningful work
→ real harness conversation is streamed
→ Keystone observes evidence
→ Keystone generates a new evidence-derived prompt
→ prompt is dispatched into the same session where supported
→ harness continues
→ Keystone verifies outcome
→ COMPLETE / ASK / BLOCK / STOP
```

If this cannot be demonstrated with a real provider adapter, the V2 control loop is not complete.

## North star

> Keep the engineering thread intact: understand intent, give the harness what it needs, watch what it actually does, correct direction when necessary, preserve continuity, and continue until the project reaches a verified state.
