# Keystone V1 Implementation Plan

## Objective

Build the smallest production-shaped Keystone control loop that proves the architecture without becoming a coding-agent runtime.

## Architectural rule

> Existing AI harnesses execute work. Keystone owns durable state, context, supervision, evidence, policy, and continuation.

Anything that violates this boundary requires an explicit architecture decision.

## V1 sequence

### Phase 0 — Repository foundation

- Define package/module boundaries.
- Add `.keystone/` schema and state conventions.
- Add configuration and policy schema.
- Add structured logging.
- Add test harness for core domain objects.
- Add `AGENTS.md` development contract.

### Phase 1 — Domain model

Implement technology-neutral entities:

```text
Project
Requirement
Decision
Assumption
WorkOrder
WorkPacket
HarnessRun
Observation
Artifact
Evidence
Finding
PolicyDecision
Checkpoint
Learning
Release
Capability
```

Requirements:

- serializable;
- schema-versioned;
- provenance-aware;
- deterministic identifiers where appropriate;
- forward-compatible;
- independent of any model provider or programming language.

### Phase 2 — Project initialization

Implement:

```bash
keystone init
keystone status
```

`init` should:

1. inspect the existing project;
2. detect capabilities;
3. detect Git state;
4. discover agent instruction files;
5. infer the project lifecycle state;
6. create `.keystone/` state;
7. record observations and assumptions;
8. ask only questions whose answers materially affect architecture, security, irreversible actions, or cost.

Initialization must be safe to run against a project that is already mid-development.

### Phase 3 — Capability discovery

Create a provider interface for capability detection.

```text
CapabilityDetector
  detect(project_root) -> CapabilitySet
```

Initial detectors can cover:

- Git;
- common languages;
- package/build managers;
- test runners;
- lint/typecheck;
- browser tooling;
- CI configuration;
- database/migration indicators;
- existing AI-agent instruction files.

Do not encode framework-specific business logic in the core.

### Phase 4 — Work order and context compiler

Implement:

```bash
keystone ask "..."
```

Pipeline:

```text
request
 ↓
normalize
 ↓
requirements / impact
 ↓
select context
 ↓
build WorkPacket
 ↓
render harness prompt
```

Context selection must be progressive rather than dumping the entire repository into model context.

Every selected context item should have a reason/provenance record.

### Phase 5 — Harness adapter contract

Define the stable adapter interface before implementing provider-specific behavior.

Conceptual contract:

```text
HarnessAdapter
├── discover()
├── capabilities()
├── start(work_packet)
├── send(instruction)
├── observe()
├── interrupt()
├── resume(checkpoint)
└── result()
```

The actual interface should accommodate adapters with weaker integration surfaces.

Integration levels:

1. instruction-file integration;
2. CLI/manual exchange;
3. provider-specific programmatic adapter;
4. long-running local control connection.

V1 should implement one real adapter end-to-end and keep the others behind the interface.

### Phase 6 — Observation and evidence

Implement a normalized observation stream.

Minimum observation types:

```text
message
command
file_read
file_change
diff
test
build
browser
git
error
metric
```

Build an evidence store with:

- evidence ID;
- source observation;
- scope;
- commit/input identity;
- status;
- artifact references;
- timestamp;
- provenance.

Evidence must be reusable when its relevant inputs have not changed.

### Phase 7 — Deterministic validation

Implement a validation planner that maps changed capabilities and risk to checks.

Initial tiers:

```text
Tier 0: static/config/type/lint
Tier 1: affected tests
Tier 2: targeted browser/system validation
Tier 3: milestone regression
```

The validator should produce structured evidence rather than only text logs.

Do not make every task run the entire test suite.

### Phase 8 — Supervisor V1

Implement evidence-based supervisor findings for:

- unsupported completion claims;
- requirement drift;
- repeated/no-progress loops;
- unexpected file scope;
- failed validation;
- excessive repeated tool activity where metrics permit detection.

Supervisor output:

```text
Observation
 ↓
Finding(s)
 ↓
Policy evaluation
 ↓
Next action
```

The supervisor should initially be conservative. False-positive intervention is itself a tracked metric.

### Phase 9 — Policy and autonomy

Implement explicit policy evaluation.

Core decisions:

```text
CONTINUE
CORRECT
REPLAN
VALIDATE
ASK
APPROVE
BLOCK
STOP
```

Policy must define which operations are automatically allowed.

Never infer approval from the absence of a user response.

### Phase 10 — Checkpoint and recovery

Implement:

```bash
keystone continue
```

Checkpoint whenever:

- a meaningful work step completes;
- a validation boundary is crossed;
- the harness fails;
- context/quota exhaustion occurs;
- user approval is required;
- Keystone switches execution strategy.

A checkpoint must contain everything required to reconstruct the next action without the original conversation.

### Phase 11 — Learning

Implement learning records before implementing sophisticated optimization.

Initial learning sources:

- repeated failures;
- repeated unnecessary context reads;
- repeated requirement drift;
- validation failure patterns;
- harness-specific success/failure rates;
- project-specific conventions.

Learning lifecycle:

```text
OBSERVED
 ↓
CANDIDATE
 ↓
EVALUATED
 ↓
ACTIVE / REJECTED
 ↓
SUPERSEDED
```

Every activated learning rule must be versioned and reversible.

### Phase 12 — End-to-end reference scenario

Use a real existing project as the integration target.

The reference scenario should prove:

```text
keystone init
 ↓
keystone ask "..."
 ↓
harness execution
 ↓
observation
 ↓
supervision
 ↓
validation
 ↓
evidence
 ↓
checkpoint
 ↓
continue
 ↓
verified completion
```

Losal is the intended first dogfooding target, but Keystone's core must remain generic.

## Repository layout

A proposed implementation layout:

```text
keystone/
├── cmd/
│   └── keystone/
├── internal/
│   ├── domain/
│   ├── state/
│   ├── project/
│   ├── capabilities/
│   ├── context/
│   ├── work/
│   ├── harness/
│   ├── observation/
│   ├── evidence/
│   ├── validation/
│   ├── supervisor/
│   ├── policy/
│   ├── checkpoint/
│   ├── learning/
│   └── git/
├── adapters/
│   └── harnesses/
├── schemas/
├── docs/
└── tests/
```

The exact implementation language is intentionally not prescribed by the architecture. The repository should choose based on the operational requirements of the first implementation rather than prematurely coupling the domain model to a language ecosystem.

## Stable interfaces first

Before provider integrations, lock down these contracts:

1. state serialization;
2. work packet schema;
3. observation schema;
4. evidence schema;
5. policy decision schema;
6. checkpoint schema;
7. learning record schema;
8. capability interface;
9. harness adapter interface;
10. validator interface.

Provider adapters can then evolve without destabilizing the core.

## Deterministic vs model-based responsibilities

### Deterministic first

Use deterministic mechanisms for:

- Git state;
- file/diff inspection;
- test/build execution;
- schema checks;
- dependency analysis;
- capability detection;
- evidence identity/invalidation;
- policy enforcement;
- permission boundaries.

### Model-assisted

Use models for:

- request interpretation;
- semantic requirement extraction;
- architecture reasoning;
- context relevance ranking;
- alternative evaluation;
- behavioral critique;
- ambiguous error interpretation.

Model inference must not be the sole basis for high-risk completion, security approval, destructive data operations, or release readiness.

## V1 test strategy

### Unit

Domain transitions, schema validation, policy evaluation, context selection, evidence invalidation, learning lifecycle.

### Integration

Filesystem, Git, capability detectors, harness adapter, validator and checkpoint persistence.

### Scenario

A complete request/result/supervision/validation/recovery cycle against a fixture repository.

### Failure injection

Explicitly test:

- harness timeout;
- malformed harness output;
- unsupported completion claim;
- repeated failed command;
- context exhaustion;
- missing credential;
- policy-blocked action;
- dirty Git worktree;
- validation failure;
- stale evidence.

## Metrics required from the first implementation

Track enough telemetry to answer whether Keystone is helping:

- completion success;
- time to verified completion;
- context size;
- model/harness usage where available;
- tool calls;
- repeated actions;
- files read/changed;
- validation duration;
- retries;
- intervention count;
- false-positive intervention count;
- evidence reuse;
- harness switches;
- learning candidates and outcomes.

Do not optimize for token reduction at the expense of correctness.

## Explicitly deferred

The following are intentionally not V1 dependencies:

- mandatory vector database;
- custom foundation model;
- multi-agent swarm;
- universal browser computer-use runtime;
- production auto-remediation;
- large plugin marketplace;
- opaque self-modifying policy;
- cloud-only architecture;
- provider-specific core abstractions.

## Definition of V1 done

V1 is complete when an existing project can be initialized and a meaningful engineering request can travel through Keystone, an existing harness, evidence-backed validation, supervision, checkpoint/recovery, and a policy-controlled completion decision — without Keystone itself becoming the coding agent.
