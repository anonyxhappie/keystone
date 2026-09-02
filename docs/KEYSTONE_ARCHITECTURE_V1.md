# Keystone Architecture V1

## 1. Product definition

Keystone is a persistent engineering intelligence and control layer that sits between human intent and existing AI development/operations harnesses.

It does **not** replace Codex, Antigravity, Claude Code, Cursor, or another execution harness. The harness remains responsible for performing work. Keystone provides the durable project understanding, context construction, supervision, verification, learning, and policy layer around that work.

The same model applies beyond software coding: any project that has intent, artifacts, work, validation, evidence, and lifecycle state can use Keystone.

## 2. Core interaction model

Every user request enters Keystone first.

```text
USER REQUEST
     ↓
REQUEST UNDERSTANDING
     ↓
PROJECT STATE + CONTEXT
     ↓
IMPACT / RISK / POLICY
     ↓
NEXT-ACTION PLANNER
     ↓
HARNESS PROMPT BUILDER
     ↓
EXISTING HARNESS
     ↓
OBSERVATION / ARTIFACTS / TOOL RESULTS
     ↓
SUPERVISOR
     ↓
┌────┼───────────┬────────────┐
↓    ↓           ↓            ↓
CONTINUE  CORRECT/REPLAN   ASK USER    COMPLETE
```

The loop may run once in assisted mode or repeatedly in full-auto mode.

## 3. Modes

### Assisted mode

Keystone generates the next harness action, interprets the returned result, updates project state, and presents the next recommended action. The user remains in the loop.

### Full-auto mode

Keystone continues the request/result loop without requiring manual confirmation for ordinary actions that are allowed by policy. It stops for consequential decisions, explicit approval gates, unavailable permissions/credentials, security blockers, unrecoverable failures, or completion.

Full-auto means **autonomous within policy**, never unrestricted authority.

## 4. Harness boundary

Keystone is harness-agnostic.

```text
                  KEYSTONE
                      │
               Harness Adapter
                      │
       ┌──────────────┼──────────────┐
       ↓              ↓              ↓
     Codex       Antigravity     Claude Code
```

An adapter translates a Keystone work packet into harness-specific instructions and translates harness output back into normalized observations.

The core must not depend on any single harness protocol.

When a harness cannot be programmatically controlled, Keystone must still support assisted operation through generated context/instructions and imported results.

## 5. Installation and portability

Keystone is an open-source, cross-platform tool. The project state format must be language/framework/architecture agnostic.

The preferred user experience is repository-native initialization:

```bash
forgeos init
```

The executable/distribution name may be finalized independently from the repository/product name. Distribution should support a standalone executable as the canonical runtime, with convenience installation mechanisms such as GitHub Releases, package-manager wrappers, and containers.

A managed repository contains a portable `.keystone/` directory. The user's application does not need to depend on a Keystone runtime library.

## 6. Repository state

```text
.keystone/
├── project.yaml
├── state.yaml
├── requirements/
├── architecture/
├── decisions/
├── assumptions/
├── work/
├── checkpoints/
├── evidence/
├── learning/
├── policies/
└── manifests/
```

The repository remains the primary portable project boundary. No critical engineering knowledge may exist only in a chat transcript.

Global installation state may exist outside the repository, but project-specific facts, decisions, checkpoints, evidence, and learnings must be recoverable from the repository or explicitly configured external stores.

## 7. Universal project model

Keystone must not define a project as "source code."

The abstract model is:

```text
Project
  ↓
Intent / Requirements
  ↓
Artifacts / Architecture
  ↓
Work
  ↓
Validation
  ↓
Evidence
  ↓
Lifecycle state / Release
```

This supports software, infrastructure, data, design, documentation, research, automation, and other structured projects.

Technology-specific integrations are capabilities, not core assumptions.

## 8. Capability detection

On initialization Keystone inspects the repository/environment and builds a capability registry.

Examples:

```yaml
languages: [typescript, python]
test: [vitest, pytest]
build: [pnpm]
browser: [playwright]
ci: [github-actions]
database: [postgres]
```

Detection may identify:

- languages;
- frameworks;
- package/build systems;
- test runners;
- linters/type checkers;
- browser tooling;
- databases/migrations;
- infrastructure;
- CI/CD;
- design artifacts;
- deployment targets;
- observability systems;
- existing agent instructions.

Core workflows operate on capabilities and observed project structure rather than hard-coded framework assumptions.

## 9. Request model

The original user request is immutable source intent.

A normalized work order may add interpretation, but must retain:

```yaml
id: WO-...
source_request: "original user request"
change_type: FEATURE
priority: normal
quality_target: production
autonomy: high
status: PLANNED
```

Keystone must distinguish:

- user requirement;
- observed repository fact;
- architectural decision;
- assumption;
- agent proposal;
- derived recommendation.

These categories must never be silently conflated.

## 10. Project state model

Keystone separates:

### Desired state

Requirements, policies, target architecture, roadmap and release intent.

### Observed state

Repository contents, Git state, test results, browser results, runtime telemetry, deployments and external tool observations.

### Derived state

Impact analysis, risk, recommended tasks, context selection, progress estimates and release readiness.

Derived state must be recomputable from durable inputs where practical.

## 11. Context intelligence

Keystone's context engine determines what the harness needs to know for the current action.

### Core context

Always include the minimum invariant information required to preserve intent and policy:

- project identity;
- current lifecycle state;
- active work order;
- requirements/acceptance criteria;
- relevant policies;
- blockers;
- checkpoint information.

### Relevant context

Include affected requirements, architecture, decisions, assumptions, source paths, tests, security constraints and valid evidence.

### On-demand context

Retrieve large historical documents, adjacent subsystems, old artifacts, detailed logs, traces and other information only when required.

Every context item should carry provenance.

## 12. Prompt construction

Keystone should generate a **work packet**, not rely on a universal mega-prompt.

```json
{
  "workOrder": "WO-184",
  "objective": "Implement nearby discovery",
  "requirements": ["R-41", "R-44"],
  "constraints": ["preserve provider licensing boundaries"],
  "architecture": ["DiscoveryProvider", "LocationService"],
  "files": ["src/discovery/*"],
  "validation": ["integration", "browser"],
  "expectedOutput": ["changes", "tests", "blockers", "claims"]
}
```

The harness adapter renders the work packet into the native style best supported by that harness.

## 13. Observation model

Keystone observes more than natural-language responses.

Where available it should collect:

- harness messages;
- tool calls;
- commands;
- files read;
- files changed;
- diffs;
- tests;
- build output;
- browser results;
- logs/metrics/traces;
- Git state;
- token/usage metrics;
- elapsed time;
- errors;
- external API interactions.

Observation should be normalized into structured records with links to large artifacts.

## 14. Agent supervision

The supervisor evaluates both **outcome** and **behavior**.

It should detect signals such as:

### Unsupported claims / hallucination

The harness claims completion, verification, API usage, test success, or another fact that is not supported by observable evidence.

### Requirement drift

The implementation departs from the original request or explicit constraints without a recorded decision.

### Architecture drift

The harness introduces patterns inconsistent with established architecture without sufficient rationale.

### Repeated failure / looping

The harness repeats substantially the same action after receiving the same failure signal without meaningful new information.

### Low information gain

Large amounts of reading, tool calls, or model output produce little or no measurable progress.

### Excessive context/tool usage

The harness consumes substantially more context, tools, or time than justified by the task and available evidence.

### Cargo-cult / obsolete methodology

The proposed approach ignores capabilities, abstractions, standards, or project conventions already available, or introduces unnecessary complexity without evidence of need.

### Premature completion

The harness declares success while requirements, tests, evidence, or release criteria remain incomplete.

These are signals for intervention, not automatic proof of failure. Keystone must use evidence and confidence thresholds.

## 15. Engineering judgment

For material changes, Keystone may request an independent critique of the implementation.

The critic evaluates:

- correctness;
- requirement compliance;
- architecture alignment;
- security/privacy;
- maintainability;
- test quality;
- operational impact;
- efficiency;
- methodological quality;
- materially better alternatives.

A critique should not automatically rewrite working code. It produces evidence and recommendations that the policy/planner evaluates.

## 16. Progress model

Keystone should maintain an evidence-based progress estimate.

```yaml
progress:
  requirements: 7/8
  implementation: 0.85
  validation: 0.60
  confidence: 0.78
next_best_action: "complete authorization journey"
```

Progress must not be derived solely from harness statements. It should be corroborated by repository and validation evidence.

## 17. Adaptive next-action selection

After every meaningful observation, Keystone decides the next action from current state rather than blindly following the original plan.

Possible actions:

```text
CONTINUE
REFINE_PROMPT
EXPAND_CONTEXT
RUN_VALIDATION
REPAIR
REPLAN
CHANGE_HARNESS
CHANGE_MODEL
REQUEST_REVIEW
ASK_USER
WAIT
STOP
```

The system should prefer the **smallest justified intervention**.

## 18. Risk and policy

Every consequential action is evaluated against project policy and risk.

```text
ACTION
  ↓
RISK + POLICY
  ↓
AUTO / APPROVAL / ASK / BLOCK
```

Typical defaults:

- edit code: auto;
- run tests: auto;
- browser validation: auto;
- create branch: auto;
- commit: policy-dependent;
- open PR: policy-dependent;
- deploy staging: policy-dependent;
- production deployment: explicit policy;
- destructive migration: approval;
- destructive data operation: approval or block;
- credential/secret changes: restricted;
- force push: prohibited by default.

Silence must never be interpreted as approval for an explicitly gated action.

## 19. Validation and evidence

Validation depth is determined by impact and risk.

```text
Tier 0  static/config/type checks
Tier 1  affected unit/integration
Tier 2  targeted browser/system journeys
Tier 3  milestone regression
Tier 4  release audit
```

Evidence records must include scope and validity information so successful checks can be reused when inputs remain unchanged.

Example:

```yaml
id: EV-...
type: browser_test
status: PASS
work_order: WO-...
commit: abc123
inputs_hash: ...
artifacts:
  - browser/EV-93.json
summary:
  assertions: 22
  console_errors: 0
  network_failures: 0
```

## 20. Evidence invalidation

Prior evidence becomes invalid or suspect when relevant inputs change, including:

- affected code;
- dependencies;
- requirements;
- schema/data;
- security policy;
- environment;
- integration configuration;
- explicit evidence expiry.

Keystone should invalidate only affected evidence where dependency information allows it, rather than rerunning the entire project unnecessarily.

## 21. Checkpoints and resumability

A checkpoint is a machine-readable continuation contract.

```yaml
id: CP-...
work_order: WO-...
state: FEATURE_VALIDATION
completed: [implementation, tier0, tier1]
pending: [tier2]
changed_files: [src/...]
last_commit: abc123
context_manifest: ctx-...
unresolved_questions: []
blockers: []
```

Context exhaustion, quota exhaustion, crashes, harness failure and model switching all use the same checkpoint mechanism.

A new harness must be able to resume without the original conversation.

## 22. Harness/model switching

Keystone may switch harnesses or models when:

- the current worker is unavailable;
- context/usage limits are reached;
- repeated failures indicate poor fit;
- another worker has stronger demonstrated performance for the task;
- policy requires an independent review.

Switching must occur from reconstructable state, not from hidden conversational memory.

Example:

```text
Codex
 ↓
checkpoint
 ↓
Antigravity
 ↓
continue
```

## 23. Learning system

Keystone should learn from observed outcomes, but learning must be explicit, evidence-backed, versioned and reversible.

Learning dimensions include:

### Project learning

Patterns specific to the repository, architecture, tools, tests and operational environment.

### Harness/model learning

Observed strengths, weaknesses, failure modes and efficiency characteristics of a harness/model.

### Project × harness learning

Conditional patterns such as a harness performing poorly on a particular project subsystem.

A learning record should contain:

```yaml
id: L-...
scope: project_harness
observation: "Repeated unrelated repository reads during auth work"
evidence_refs: [EV-1, EV-2, EV-7]
confidence: 0.91
proposed_change: "Restrict initial auth context to auth/RBAC dependencies"
status: ACTIVE
version: 1
```

Keystone must not silently self-modify core policy. Learning produces versioned candidates that are activated according to policy and evidence thresholds.

## 24. Self-improvement loop

```text
OBSERVE
  ↓
IDENTIFY FAILURE / INEFFICIENCY
  ↓
COLLECT EVIDENCE
  ↓
FORM LEARNING CANDIDATE
  ↓
CONFIDENCE / POLICY CHECK
  ↓
VERSIONED LEARNING
  ↓
IMPROVED FUTURE CONTEXT / STRATEGY
```

The goal is not to make a model permanently "smarter". The goal is to make Keystone increasingly effective at selecting context, prompts, harnesses, validation and interventions for real projects.

## 25. Token and efficiency intelligence

Efficiency must be measured as useful engineering progress per cost, not tokens alone.

Track:

- input/output tokens where available;
- context size;
- repeated reads;
- tool calls;
- redundant tool calls;
- elapsed time;
- retries;
- model/harness switches;
- files read/changed;
- validation cost;
- successful outcome.

Keystone should detect waste patterns and adapt future execution, but must not reduce required context or validation merely to lower token counts.

## 26. Production operations capability

Production operations are an extension capability, not a special case in the core model.

Optional integrations may provide:

- logs;
- metrics;
- traces;
- alerts;
- deployments;
- incidents;
- health checks.

A production event can become a normal Keystone work order:

```text
ALERT
 ↓
OBSERVE
 ↓
DIAGNOSE
 ↓
HYPOTHESIS
 ↓
HARNESS WORK
 ↓
VALIDATE
 ↓
DEPLOY / APPROVE
 ↓
OBSERVE
```

Production authority remains policy-controlled.

## 27. Capability/plugin architecture

Capabilities should be extensible without modifying the core state model.

Potential capability families:

```text
harnesses
languages/frameworks
Git/GitHub
CI/CD
testing
browser
security
data
cloud/infrastructure
observability
monitoring
alerting
incident management
design
```

Each capability declares what it can observe or execute, required permissions, input/output schemas, and evidence it produces.

## 28. Security model

Keystone separates:

- intelligence/planning authority;
- execution authority;
- approval authority.

Tools receive least-privilege capabilities per work order.

Secrets should not be placed into ordinary model context. Prefer runtime references and controlled execution environments.

All sensitive actions should be auditable.

## 29. Artifact separation

Large artifacts live outside normal model context:

```text
.keystone/evidence/
```

or an explicitly configured artifact store.

Examples:

- raw logs;
- Playwright traces;
- screenshots;
- full test reports;
- security reports;
- build logs;
- telemetry samples.

The model receives compact summaries and artifact references and retrieves details only when necessary.

## 30. Git safety

Keystone must understand Git but must not silently destroy developer work.

It should track:

- current branch;
- base revision;
- dirty state;
- changed files;
- commits;
- tags;
- release references.

By default:

- never discard uncommitted user changes;
- never force-push;
- never reset destructively without explicit policy;
- never create a release tag before release gates pass.

## 31. Release model

A release candidate links:

```text
requirements
→ implementation
→ validation
→ security evidence
→ operational evidence
→ changelog
→ Git reference
→ release decision
```

Release policy defines blocking criteria in advance.

Recommended severity model:

- P0: catastrophic/security/data-loss blocker;
- P1: explicit release requirement blocker;
- P2: important hardening;
- P3: future improvement/debt.

Agents and reviewers must not invent new P1 criteria merely because additional improvements are possible.

## 32. Observability of Keystone itself

Keystone should expose its own run history and metrics:

- work-order success rate;
- intervention rate;
- hallucination/unsupported-claim detections;
- loop detections;
- retries;
- model/harness switches;
- average context size;
- token efficiency;
- validation pass rate;
- false-positive supervisor interventions;
- time to completion;
- learning candidates accepted/rejected.

This is required to evaluate whether Keystone itself is improving.

## 33. V1 implementation boundary

V1 should remain deliberately small. It should establish the stable contracts for:

1. repository initialization;
2. project state;
3. capability detection;
4. request/work-order normalization;
5. context compilation;
6. prompt/work-packet generation;
7. at least one harness adapter;
8. result/observation normalization;
9. deterministic validation integration;
10. evidence storage;
11. checkpoints/recovery;
12. policy/risk decisions;
13. basic supervisor signals;
14. project/harness learning records;
15. Git safety.

Do **not** make production monitoring, a large agent swarm, a mandatory vector database, or a custom model a V1 dependency.

## 34. V1 acceptance scenarios

A reference project must demonstrate:

1. Initialize Keystone in an existing project.
2. Detect the project's capabilities without assuming its stack.
3. Accept a natural-language request.
4. Generate a coherent work order and context packet.
5. Send that packet through a harness adapter.
6. Interpret the harness result and continue the loop.
7. Detect at least one unsupported completion claim.
8. Detect at least one repeated/no-progress loop.
9. Detect a requirement or architecture deviation.
10. Select targeted validation based on impact.
11. Persist evidence.
12. Recover from context/quota interruption.
13. Switch harness/model without losing state.
14. Run in assisted and full-auto modes.
15. Stop safely at a consequential approval boundary.
16. Record a project/harness learning candidate.
17. Preserve developer Git changes.
18. Produce an evidence-backed completion/release decision.

## 35. Non-goals

V1 is not:

- a replacement coding agent;
- a replacement IDE;
- a mandatory cloud service;
- an autonomous unrestricted production operator;
- a model-training platform;
- a universal plugin marketplace;
- a giant multi-agent swarm.

## 36. North star

> **Keystone keeps the engineering thread intact: understand the intent, give the harness what it needs, observe what actually happened, recognize when the work is going wrong, correct course, learn from the outcome, and continue until the project reaches a verified state.**
