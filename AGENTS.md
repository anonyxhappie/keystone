# Keystone Agent Contract

This repository defines Keystone, a persistent engineering intelligence/control layer around existing AI development and operations harnesses.

## Hard architectural boundaries

1. **Do not build another coding-agent/harness runtime unless explicitly requested.** Keystone supervises and drives existing harnesses through adapters.
2. **Project state is durable.** Do not put critical engineering knowledge only in conversation history or ephemeral process memory.
3. **Evidence outranks claims.** A harness saying "done" is not proof of completion.
4. **Autonomy is policy-bounded.** Never silently grant destructive, security-sensitive, credential-sensitive, data-loss, or production authority.
5. **Learning is explicit and reversible.** Do not implement opaque self-modifying policy.
6. **Core abstractions must remain technology-agnostic.** Language, framework, IDE, architecture, provider, model, and harness details belong behind capabilities/adapters.
7. **Prefer deterministic verification.** Model judgment can prioritize and interpret evidence, but high-risk completion decisions must not depend solely on model opinion.
8. **Preserve developer work.** Never silently discard uncommitted changes or use destructive Git operations outside explicit policy.

## Required reasoning discipline

Before implementing a substantial change:

- inspect the relevant architecture and project state;
- identify affected requirements and invariants;
- distinguish facts, assumptions, proposals, and decisions;
- choose the smallest coherent implementation slice;
- define how the result will be validated;
- update durable state/evidence when the change materially affects project knowledge.

## Supervisor behavior

The supervisor should intervene minimally:

- normal behavior → observe;
- minor inefficiency → record;
- repeated inefficiency → optimize context/tooling;
- requirement drift → intervene;
- unsupported completion claim → verify;
- loop/no-progress → replan;
- security or destructive risk → stop or require approval.

Do not manufacture blockers merely because additional improvements are possible. Respect explicit P0/P1 release criteria and distinguish P2/P3 hardening from true blockers.

## Context discipline

Use progressive context:

1. project identity and active work;
2. directly relevant requirements, architecture, decisions, files, tests, policies;
3. historical or large artifacts only when needed.

Do not read the entire repository or every historical document by default.

## Implementation quality

- Keep interfaces small and testable.
- Prefer explicit schemas over implicit conventions.
- Keep provenance on derived state and evidence.
- Make recovery/checkpoint behavior first-class.
- Avoid premature vector databases, agent swarms, custom model training, or production-operations complexity when a simpler deterministic mechanism is sufficient.
- Add tests for state transitions, policy boundaries, adapter behavior, evidence validity, and recovery paths.

## Reference architecture

Read `docs/KEYSTONE_ARCHITECTURE_V1.md` before making architectural changes. That document is the current source of truth for the V1 boundary and acceptance scenarios.
