# ForgeOS Architecture V1

## 1. Purpose

ForgeOS is a persistent, project-aware engineering control plane that converts natural-language engineering requests into autonomous, evidence-backed software work. It is not a fixed greenfield application generator.

Supported requests include new products, features, bug fixes, UI improvements, refactors, integrations, version work, reviews, QA, security work, release preparation/execution, continuation and recovery.

## 2. North-star principle

> Minimum unnecessary tokens and tool work while preserving maximum necessary understanding, reasoning, verification and quality.

Optimization must never weaken a required quality or release gate.

## 3. Core architecture

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
       PASS -> CHECKPOINT / VERSION / RELEASE
       FAIL -> DIAGNOSE -> REPLAN
```

The agent is a replaceable worker. Durable project state belongs to ForgeOS.

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

The state graph links requirements, architecture decisions, code changes, tests, evidence, versions and releases, plus dependencies, risks, assumptions, blockers and integrations.

A vector database is not a V1 requirement. Start with structured state, filesystem/document retrieval, repository analysis, AST and dependency information. Add semantic indexing only when retrieval evidence justifies it.

## 5. Intent and workflow

Every request becomes a durable work order before substantial execution. The resolver determines change class, affected version/domain, dependencies, risk, autonomy and validation needs.

Conceptual work order:

```yaml
id: WO-...
type: AUTO
request: "Add feature X"
priority: normal
quality: production
autonomy: high
status: PLANNED
```

The planner selects only the states needed for the request. A small bug fix must not traverse a greenfield product workflow; a release must not bypass required gates.

Execution states:

```text
INTAKE -> DISCOVERY -> CLARIFICATION -> SPECIFICATION -> ARCHITECTURE
       -> DATA_FOUNDATION -> PLANNING -> IMPLEMENTATION
       -> FEATURE_VALIDATION -> INTEGRATION -> SECURITY
       -> SYSTEM_QA -> RELEASE_CANDIDATE -> RELEASED -> MAINTENANCE
```

`CLARIFICATION`, `DATA_FOUNDATION`, `SECURITY` and other stages are conditional. Failure may enter `BLOCKED`, `RECOVERING` or `REPLAN` without losing evidence.

## 6. Human escalation

Ask the user only when ambiguity can materially change a consequential outcome: architecture, security/privacy, legal/licensing, significant cost, irreversible data/schema behavior, or externally visible behavior where no safe default exists.

Otherwise select a reasonable default, record it as an assumption, and continue.

## 7. Context compiler

Context is progressive and elastic.

**Core context:** project identity, version, current state, active work order, immutable principles, blockers and acceptance criteria.

**Relevant context:** affected requirements, architecture, decisions, assumptions, source files, tests, security constraints and relevant evidence.

**On-demand context:** large historical documents, unrelated subsystems, old releases and detailed artifacts.

Retrieval follows dependency and traceability relationships rather than a static list of every project document. Large tool output stays outside model context and is summarized into structured evidence.

## 8. Agent workers

Workers sit behind an adapter boundary and may include implementation, architecture, debugging, review, security, data, testing, release and documentation roles.

Model selection is task-aware. Stronger reasoning models are used where complexity or risk warrants them; faster/cheaper workers handle routine execution and summarization. Quality gates remain model-independent.

## 9. Deterministic tools

ForgeOS delegates deterministic work to deterministic tooling: typecheck, lint, unit/integration tests, schema/config/data validation, duplicate detection, dependency analysis, builds, PWA checks, browser automation, security scanners, migrations and Git operations.

Tool output is normalized into compact evidence records rather than dumped into agent context.

## 10. Risk-based validation

Risk considers security, data, API, UI, migration, external dependency and release impact.

Validation tiers:

- **Tier 0:** static/offline checks.
- **Tier 1:** affected unit/integration tests.
- **Tier 2:** targeted browser journeys.
- **Tier 3:** milestone/system regression.
- **Tier 4:** complete release audit.

Risk determines the minimum sufficient validation during development. Explicit release gates remain mandatory.

## 11. Browser evidence

Browser automation should primarily produce assertions, route/state checks, console-error counts, network-failure counts and accessibility checks. Screenshots, DOM snapshots and traces are captured for visual checkpoints and failures rather than indiscriminately on every run.

## 12. Evidence model

Evidence is durable state and should be linked to work orders, requirements, commits and releases where possible.

```json
{
  "test": "feature-flow",
  "status": "PASS",
  "assertions": 22,
  "consoleErrors": 0,
  "networkFailures": 0,
  "durationMs": 4312
}
```

Previously valid evidence should be reused when its scope and inputs remain valid; ForgeOS should not rerun expensive checks without a reason.

## 13. Recovery

Execution outcomes include `SUCCESS`, `FAILED`, `BLOCKED` and `CONTEXT_EXHAUSTED`.

Recovery procedure:

1. Persist checkpoint.
2. Record completed work and unresolved reasoning.
3. Preserve working-tree changes and evidence.
4. Rebuild minimal required context.
5. Select another capable worker/model when necessary.
6. Resume from checkpoint.

The same mechanism handles model switches, quota exhaustion, tool failures, crashes and interrupted sessions.

## 14. Requirements and traceability

User-requested changes are logged separately from implementation details. Consequential work maintains:

```text
request -> requirement/change -> decision -> work order
        -> code -> tests -> evidence -> version -> release
```

This prevents later agents from confusing original requirements with assumptions or implementation choices.

## 15. Versioning and releases

Versions are explicit ForgeOS state as well as Git references. A release candidate records resolved requirements, blocker disposition, implementation state, test/security/browser/build/data evidence, changelog, commit/reference and release decision.

Release gates are predefined by project policy. Agents must not invent unrelated blocking criteria during a final audit. Findings should be classified by severity (for example P0/P1 release blockers versus P2/P3 hardening) to prevent endless audit loops.

Git integration tracks branch, base revision, changed files, work-order commits, tags, working-tree state and relevant migration state. ForgeOS must never silently discard user changes.

## 16. V1 implementation boundary

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
10. adaptive planner and basic quality gates.

Do not begin with a giant multi-agent swarm, mandatory vector DB, custom model, complex UI, social collaboration layer or automatic irreversible production actions without gates.

## 17. Reference validation

Losal is the first real reference project. It validates ForgeOS against a production-style repository with real data, CMS workflows, security controls, browser QA and release gates. ForgeOS core contracts must remain independent of Losal-specific technology or domain assumptions.