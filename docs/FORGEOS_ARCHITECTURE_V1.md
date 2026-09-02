# ForgeOS Architecture V1

## 1. Purpose

ForgeOS is a persistent, project-aware engineering control plane that converts natural-language engineering requests into autonomous, evidence-backed software work. It is not a fixed greenfield application generator.

Supported requests include new products, features, bug fixes, UI improvements, refactors, integrations, version work, reviews, QA, security work, release preparation/execution, continuation and recovery.

## 2. North-star principles

> **Minimum unnecessary tokens and tool work while preserving maximum necessary understanding, reasoning, verification and quality.**

Optimization must never weaken a required quality or release gate.

Additional invariants:

1. **The agent is disposable; project state is not.**
2. **The system is adaptive; workflows are selected, not hard-coded per request type.**
3. **Context is progressive, never artificially compressed below semantic sufficiency.**
4. **Tasks are coherent engineering slices, not arbitrary micro-tasks.**
5. **Deterministic work is delegated to deterministic tooling.**
6. **Evidence is durable and reusable when its validity scope remains intact.**
7. **No consequential release decision is based solely on an LLM assertion.**
8. **User changes and requirements changes are never silently discarded or rewritten.**

## 3. System architecture

```text
USER REQUEST
    -> INTENT RESOLVER
    -> PROJECT STATE / KNOWLEDGE GRAPH
    -> IMPACT + RISK ANALYZER
    -> ADAPTIVE PLANNER / WORK SCHEDULER
    -> CONTEXT COMPILER
    -> AGENT WORKERS <-> DETERMINISTIC TOOLS
    -> EVIDENCE ENGINE
    -> QUALITY GATE
       PASS -> CHECKPOINT / NEXT WORK / VERSION / RELEASE
       FAIL -> DIAGNOSE -> REPLAN
       BLOCKED -> ESCALATE / WAIT / RETRY
```

ForgeOS is composed of five logical planes:

### Control plane

Owns lifecycle, state transitions, scheduling, retries, policy, permissions and release decisions.

### Knowledge plane

Owns project facts, requirements, architecture, decisions, assumptions, traceability and relevant repository structure.

### Execution plane

Runs agent workers and deterministic tools in isolated, observable jobs.

### Evidence plane

Stores normalized test/build/browser/security/data/Git evidence and validity metadata.

### Artifact plane

Stores large logs, screenshots, traces, reports, patches and other artifacts outside normal model context.

The first implementation may colocate these planes in one service/repository, but the contracts should keep them logically separate.

## 4. Durable project state

Each managed repository gets a `.forge/` control plane:

```text
.forge/
├── project.yaml
├── requirements.yaml
├── architecture.yaml
├── decisions.yaml
├── assumptions.yaml
├── roadmap.yaml
├── traceability.yaml
├── gates.yaml
├── state.yaml
├── blockers.yaml
├── work_orders/
├── checkpoints/
└── evidence/
```

Human-readable authoritative specifications remain under `docs/`; machine-oriented state belongs under `.forge/`.

ForgeOS should preserve a stable separation between **desired state**, **observed state** and **derived state**:

```text
Desired:
  requirements / roadmap / policies

Observed:
  repository / tests / browser / runtime / Git / data

Derived:
  impact / risk / task graph / release readiness
```

Derived state must be recomputable. It must not become the sole source of truth.

## 5. Project state graph

The state graph links engineering facts rather than merely storing documents.

```text
Requirement
   |
   +--> Decision
   +--> Architecture element
   +--> Code area
   +--> Test set
   +--> Evidence
   +--> Work order
   +--> Version
   +--> Release
```

It should also represent:

- dependencies;
- ownership/actor;
- risk;
- assumptions;
- blockers;
- external integrations;
- evidence validity;
- supersession/history.

A vector database is not a V1 requirement. Start with structured state, repository analysis, AST/dependency information, exact search and document section retrieval. Add semantic indexing only when measured retrieval failures justify it.

## 6. Request normalization / work orders

Every user request becomes a durable work order before substantial execution.

Conceptual shape:

```yaml
id: WO-...
request: "Add feature X"
change_type: FEATURE
priority: normal
quality_target: production
autonomy: high
affected_version: 1.1.0
status: PLANNED
```

The resolver determines:

- request/change class;
- affected product area;
- target version/release;
- dependencies;
- risk;
- user-visible impact;
- likely task size;
- required questions;
- required validation;
- external capabilities/dependencies.

The normalized request must retain the original user wording. ForgeOS must never replace the original intent with an internally generated paraphrase without preserving the source request.

## 7. Consequential clarification policy

ForgeOS should ask the user only when an unresolved decision can materially alter:

- architecture;
- security/privacy;
- legal/licensing;
- significant cost;
- irreversible schema/data behavior;
- externally visible product behavior where no safe default exists.

Otherwise:

```text
choose safe/default decision
        -> record assumption
        -> continue
```

Every assumption should include:

- statement;
- basis;
- confidence;
- reversible/irreversible classification;
- affected work order;
- supersession status.

When an assumption becomes invalid, ForgeOS should perform impact analysis rather than silently continuing with stale assumptions.

## 8. Adaptive execution graph

Execution is a graph selected per work order, not a mandatory linear pipeline.

Possible states:

```text
INTAKE
  -> DISCOVERY
  -> CLARIFICATION
  -> SPECIFICATION
  -> ARCHITECTURE
  -> DATA_FOUNDATION
  -> PLANNING
  -> IMPLEMENTATION
  -> FEATURE_VALIDATION
  -> INTEGRATION
  -> SECURITY
  -> SYSTEM_QA
  -> RELEASE_CANDIDATE
  -> RELEASED
  -> MAINTENANCE
```

Conditional states may be skipped when not relevant. The planner must record why a state was skipped when that state would normally be expected for the change type.

Failure/control states:

```text
FAILED
BLOCKED
RECOVERING
REPLAN_REQUIRED
CANCELLED
```

Terminal success must be based on evidence and gate evaluation, not worker self-report.

## 9. Task sizing

ForgeOS must optimize for **coherent context**, not minimum task size.

Recommended heuristic:

- **Atomic:** isolated, low-risk change.
- **Vertical slice:** one coherent capability crossing relevant layers.
- **Milestone:** several strongly related slices with shared dependencies.

Split work when context, risk or dependencies become unwieldy—not simply because a file boundary exists.

Merge adjacent tasks when splitting them would force repeated context loading and prevent the agent from reasoning about the feature as a whole.

The scheduler should estimate:

- complexity;
- dependency depth;
- affected files/subsystems;
- security/data impact;
- test surface;
- expected duration.

## 10. Context compiler

Context is progressive and elastic.

### Tier A — invariant context

Always include:

- project identity;
- current version;
- current phase/state;
- active work order;
- immutable project principles;
- release policy relevant to the work;
- active blockers;
- mandatory constraints.

### Tier B — task context

Include:

- directly affected requirements;
- relevant architecture elements;
- decisions and assumptions;
- affected source paths;
- associated tests;
- applicable security/data constraints;
- relevant recent changes;
- valid prior evidence.

### Tier C — expansion context

Retrieved on demand:

- adjacent subsystems;
- historical rationale;
- old release documentation;
- detailed artifacts;
- large source sections not initially required.

The compiler should return provenance for each injected context item so an agent can distinguish:

- requirement;
- observed code fact;
- prior decision;
- assumption;
- derived recommendation.

Context must not silently mix factual requirements with speculative suggestions.

## 11. Context budget and context integrity

ForgeOS should monitor context composition and cost, but it must never optimize solely for token minimization.

A context packet should expose:

```yaml
context:
  estimated_tokens: 12000
  required_items: 31
  optional_items: 8
  expandable_items: 14
  requirement_coverage: 1.0
```

The system should reject an aggressively compressed packet when the required semantic coverage would be lost.

Context summaries must be derived from canonical project state. They are caches, not authorities.

## 12. Agent workers

Workers sit behind a provider-independent adapter.

Potential roles:

- planner;
- implementation;
- debugging;
- architecture;
- code review;
- security;
- data;
- test analysis;
- release analysis;
- documentation.

Workers receive a typed work packet containing:

- work order;
- context packet;
- constraints;
- expected outputs;
- allowed tools;
- quality target;
- checkpoint reference.

The worker must return structured results, not only prose.

Example:

```json
{
  "status": "NEEDS_VALIDATION",
  "changes": ["src/claims/api.ts"],
  "decisions": [],
  "questions": [],
  "tests_requested": ["claims-stateful"],
  "blockers": []
}
```

## 13. Model routing

Model choice is an orchestration policy.

The router may consider:

- task type;
- risk;
- repository size;
- context size;
- required tool-use reliability;
- latency/cost limits;
- previous worker performance;
- provider availability.

Model choice must not bypass quality gates.

Model switching is safe only at a persisted checkpoint boundary or after reconstructable execution state has been captured.

## 14. Deterministic tools

Delegate deterministic work to deterministic tooling:

- typecheck;
- lint;
- unit/integration tests;
- schema/config/data validation;
- duplicate detection;
- dependency analysis;
- builds;
- PWA checks;
- browser automation;
- accessibility checks;
- security scanners;
- migration verification;
- Git operations.

Each tool invocation should produce a normalized result:

```yaml
id: TOOL-...
tool: playwright
status: PASS
exit_code: 0
duration_ms: 4312
artifact_refs:
  - browser/run-184.json
summary:
  assertions: 22
  console_errors: 0
  network_failures: 0
```

Raw output belongs in artifacts unless needed for diagnosis.

## 15. Impact analysis

Before changing code, ForgeOS should attempt to determine:

```text
changed/requested behavior
        ↓
requirements
        ↓
architecture
        ↓
source/dependency graph
        ↓
tests
        ↓
security/data/release gates
```

Impact analysis should produce:

- affected requirements;
- affected code;
- affected data/schema;
- affected tests;
- affected security surfaces;
- affected external integrations;
- affected releases.

If confidence is low, expand context or request a review rather than pretending precision.

## 16. Risk engine

Risk dimensions include:

- security;
- data integrity;
- API behavior;
- UI behavior;
- migration;
- external dependency;
- privacy;
- release impact.

Risk produces a **validation plan**, not merely a number.

Example:

```yaml
risk: HIGH
validation:
  tier0: required
  tier1: required
  tier2: required
  tier3: conditional
  security_review: required
```

## 17. Validation tiers

### Tier 0 — static/offline

Cheap deterministic checks:

- schema;
- config;
- typecheck;
- lint;
- route checks;
- data structure validation;
- import determinism.

### Tier 1 — affected tests

Relevant unit/integration tests selected by dependency and traceability analysis.

### Tier 2 — targeted browser

Relevant Playwright journeys for browser-facing changes.

### Tier 3 — milestone regression

Broader subsystem/application regression at milestone boundaries.

### Tier 4 — release audit

Complete release gates defined by project policy.

Risk-based selection can reduce development-time work. It cannot weaken explicit release gates.

## 18. Test selection

Test selection should be graph-based:

```text
changed files / behavior
        ↓
affected requirements
        ↓
affected test nodes
        ↓
minimal sufficient validation set
```

The selected set must include transitive dependencies where required.

The selection result should record why each test was chosen and which tests were intentionally skipped.

## 19. Browser evidence

Browser automation primarily produces:

- assertions;
- route/state checks;
- console error counts;
- network failure counts;
- accessibility checks;
- structured results.

Screenshots/DOM snapshots/traces are captured for:

- visual checkpoints;
- failures;
- explicit review requests.

Normal successful runs should return concise evidence summaries while retaining detailed artifacts for retrieval.

## 20. Evidence model

Evidence is durable project state.

Conceptual record:

```yaml
id: EV-...
type: browser_test
work_order: WO-...
requirements:
  - R-CLAIM-04
commit: abc123
status: PASS
created_at: ...
validity:
  inputs_hash: ...
  environment_hash: ...
  expires_at: null
artifacts:
  - browser/EV-93.json
```

Evidence must have validity semantics. ForgeOS should reuse prior evidence only when the relevant inputs, environment and scope remain valid.

Examples of invalidation triggers:

- affected code changed;
- dependency version changed;
- relevant requirement changed;
- schema changed;
- environment assumption changed;
- security policy changed;
- evidence TTL expired.

## 21. Checkpointing and recovery

A checkpoint is a durable continuation contract.

Conceptual shape:

```yaml
checkpoint:
  id: CP-...
  work_order: WO-...
  state: FEATURE_VALIDATION
  completed:
    - implementation
    - tier0
    - tier1
  pending:
    - tier2
  changed_files:
    - src/...
  commit: abc123
  context_manifest: ctx-...
  unresolved_questions: []
  blockers: []
```

For context/quota exhaustion:

1. persist checkpoint;
2. persist worker output;
3. persist uncommitted-change state;
4. persist tool/evidence references;
5. produce unresolved-reasoning summary;
6. rebuild context from project state + checkpoint;
7. select replacement worker/model;
8. resume from the pending state.

The recovery process must be deterministic enough that a second worker can continue without requiring the original conversation.

## 22. Failure and retry policy

Not every failure should trigger an immediate retry.

Classify failures:

```text
TRANSIENT_TOOL_FAILURE
MODEL_FAILURE
CONTEXT_EXHAUSTION
VALIDATION_FAILURE
LOGIC_FAILURE
REQUIREMENT_AMBIGUITY
EXTERNAL_BLOCKER
SECURITY_BLOCKER
```

Each class has a different policy:

- transient → bounded retry;
- model/context → checkpoint + replacement;
- validation → diagnose + targeted repair;
- ambiguity → ask user or use documented default;
- external blocker → wait/escalate;
- security blocker → stop affected execution path.

Retries must have budgets and must not create uncontrolled agent loops.

## 23. Human-in-the-loop policy

Human interaction is an exception path, not the normal execution mode.

Ask only for consequential uncertainty.

Human approval may also be required by policy for explicitly gated actions such as:

- irreversible production migration;
- production credential changes;
- destructive data operations;
- releasing a major version;
- high-risk security exceptions.

Do not infer approval from silence.

## 24. Requirements and change management

Requirements are first-class entities.

Every material user-requested change should preserve:

```text
original request
  -> requirement/change
  -> decision
  -> impact analysis
  -> work order
  -> code
  -> tests
  -> evidence
  -> version
  -> release
```

Requirement changes should record:

- original statement;
- proposed new statement;
- reason;
- impact;
- affected versions;
- approval when required;
- supersession relationship.

Implementation details must not silently rewrite requirements.

## 25. Release policy and anti-loop mechanism

Every release has predefined blocking criteria.

Findings are classified:

- **P0:** catastrophic/security/data-loss blocker;
- **P1:** explicit release requirement blocker;
- **P2:** important hardening, non-blocking unless policy says otherwise;
- **P3:** future improvement/debt.

The final auditor may identify violations of existing requirements, security policy or explicit release criteria. It must not manufacture new P1 criteria simply because another test or enhancement could exist.

A release can proceed when all required P0/P1 gates pass and documented P2/P3 items have explicit disposition.

## 26. Git integration

ForgeOS tracks:

- repository identity;
- branch;
- base revision;
- working-tree state;
- changed files;
- work-order commits;
- tags;
- release references;
- migration state where applicable.

ForgeOS must never silently discard user changes.

Protected operations require explicit policy:

- no force-push by default;
- no destructive reset of user work;
- no release tag until release gate passes;
- no commit containing secrets or protected artifacts.

## 27. Security boundaries

ForgeOS must distinguish:

**planner authority**
from
**execution authority**
from
**release authority**.

A planning agent should not automatically gain unrestricted production access.

Tool permissions should be capability-scoped per work order.

Example:

```yaml
allowed_tools:
  - read_repo
  - edit_repo
  - test
  - browser
  - git_commit
forbidden_tools:
  - production_shell
  - credential_read
  - force_push
```

High-risk tools/actions require explicit policy and potentially human approval.

Secrets must never be injected into ordinary model context unless strictly required by an authorized operation; prefer opaque environment/runtime references.

## 28. Artifact and context separation

Large outputs should live outside normal model context:

```text
.artifacts/
├── tests/
├── browser/
├── security/
├── data/
├── builds/
├── logs/
└── releases/
```

Model context receives compact summaries and references. Detailed artifacts are fetched only when diagnostic reasoning needs them.

## 29. Observability and cost accounting

Track per work order and per run:

- context tokens;
- agent tokens;
- tool-output tokens;
- model;
- files read;
- files changed;
- browser duration;
- test duration;
- retries;
- model switches;
- wall-clock duration;
- estimated cost.

Optimize for:

> **validated engineering outcome per total cost and elapsed time.**

Do not optimize raw token count at the expense of semantic coverage or validation quality.

## 30. Project handoff / portability

Any agent or machine must be able to resume from the repository and ForgeOS state alone.

A handoff packet should include:

- current state;
- active work order;
- latest checkpoint;
- relevant context manifest;
- blockers;
- last successful evidence;
- pending validations;
- required tools/models;
- working-tree/commit state.

No critical knowledge may exist only in a chat transcript.

## 31. Reference implementation strategy

Losal is the first reference project because it exercises:

- real data;
- CMS workflows;
- security;
- stateful tests;
- browser QA;
- provenance;
- versioning;
- release gates;
- model/context/tool failure recovery.

ForgeOS core contracts must remain independent of Losal-specific technologies or domain assumptions.

Reference validation scenarios should include:

1. Continue interrupted work.
2. Switch models after context exhaustion.
3. Add a feature with cross-layer changes.
4. Fix a bug with targeted validation.
5. Perform a security-sensitive change.
6. Prepare a release candidate.
7. Reject a release because of an actual P1 blocker.
8. Avoid rejecting a release because of a non-blocking P2/P3 finding.

## 32. V1 implementation boundary

Build the smallest reliable control plane containing:

1. project state;
2. work orders;
3. context compilation;
4. checkpoint/recovery;
5. agent adapter;
6. deterministic tool/test runner;
7. Playwright adapter;
8. evidence store;
9. Git integration;
10. adaptive planner;
11. impact/risk analysis;
12. release gates.

The implementation must establish stable contracts before adding a complex UI or multi-agent swarm.

## 33. Initial machine contracts

The first implementation should define typed schemas/interfaces for at least:

```text
ProjectState
Requirement
Decision
Assumption
WorkOrder
WorkPacket
Checkpoint
ToolRun
Evidence
RiskAssessment
ValidationPlan
ReleaseCandidate
ReleaseDecision
```

These contracts should be versioned and backward-compatible where practical.

## 34. Non-goals for V1

Do not begin with:

- a giant autonomous multi-agent swarm;
- mandatory vector search;
- a custom model;
- complex collaboration/social features;
- unrestricted production access;
- automatic irreversible actions without gates;
- a fixed workflow that assumes every request is a greenfield application.

## 35. V1 acceptance criteria

ForgeOS V1 is not complete until a reference project can demonstrate:

- create/ingest project state;
- normalize multiple request types;
- generate a coherent work order;
- compile task-specific context without loading all project documentation;
- expand context on demand;
- select risk-appropriate validation;
- execute a coding worker through a provider adapter;
- normalize deterministic tool output;
- persist evidence;
- resume after context/quota interruption;
- switch models/workers without losing project state;
- select targeted tests from impact analysis;
- run Playwright through a controlled adapter;
- maintain requirement-to-release traceability;
- preserve user changes;
- enforce capability-scoped tool access;
- distinguish P0/P1 blockers from P2/P3 improvements;
- produce a release decision from explicit evidence.

## 36. Engineering north star

> **Give each worker enough context to make the right decision, give the system enough state to survive without that worker, and spend expensive computation only where it increases verified confidence.**
