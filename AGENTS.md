# Keystone Agent Contract

This repository defines Keystone, a persistent engineering intelligence/control layer between human intent and existing AI development and operations harnesses.

## Read this first

**`docs/KEYSTONE_PRODUCT_MODEL.md` is the product-level source of truth for the current direction.** Read it before making architectural or behavioral changes.

Keystone's purpose is to keep an existing AI harness productively working for long periods with minimal human intervention. The harness is the primary worker; Keystone is the persistent driver, navigator, supervisor, verifier, memory, and continuity layer.

The current repository state is the only development baseline until a stable release is declared. Do not depend on deleted historical release tags.

## Hard architectural boundaries

1. **Do not build another coding-agent/harness runtime.** Keystone drives existing harnesses through adapters.
2. **Harness-first execution.** If the harness can perform actionable engineering work, Keystone should normally instruct the harness to do it rather than duplicate that work itself.
3. **Keystone is a control loop, not a one-shot runner.** A run may contain many real harness turns until verified completion or a genuine human/policy boundary.
4. **Prompt dispatch is real and auditable.** Creating a WorkOrder or starting a provider process is not equivalent to sending work. Every harness instruction must be actually dispatched and durably associated with its provider session.
5. **The real harness conversation is the primary interactive experience.** Stream and persist actual user, harness, tool, command, file, validation, approval and error events. Internal lifecycle telemetry must not replace the conversation.
6. **Session continuity is durable.** Preserve the real provider session ID and use native resume where supported. Do not create synthetic sessions or fake continuity.
7. **Project/session switching changes the entire active context.** Never mix conversation history, state or pending work between projects or sessions.
8. **Harness questions are first-class human-interaction events.** Expose the actual question, pause autonomous execution, and send the user's answer back into the same session where supported. Persist meaningful rationale.
9. **Environment blockers should be delegated to the harness when recoverable.** Do not waste harness turns repeating the same failure. Diagnose, generate a targeted recovery instruction, dispatch it, and supervise the result.
10. **Retry means new information or strategy.** Repeating an unchanged action is not intelligent continuation; detect no-progress loops.
11. **Project state is durable.** Do not put critical engineering knowledge only in conversation history or ephemeral process memory.
12. **Evidence outranks claims.** A harness saying "done" is not proof of completion.
13. **Autonomy is policy-bounded.** Never silently grant destructive, security-sensitive, credential-sensitive, data-loss, or production authority.
14. **Learning is explicit and reversible.** Do not implement opaque self-modifying policy.
15. **Core abstractions remain technology-agnostic.** Language, framework, IDE, architecture, provider, model, and harness details belong behind capabilities/adapters.
16. **Prefer deterministic verification.** Model judgment can prioritize and interpret evidence, but high-risk completion decisions must not depend solely on model opinion.
17. **Preserve developer work.** Never silently discard uncommitted changes or use destructive Git operations outside explicit policy.

## Required reasoning discipline

Before implementing a substantial change:

- inspect the product model and relevant architecture/project state;
- identify affected requirements and invariants;
- distinguish facts, assumptions, proposals, and decisions;
- choose the smallest coherent implementation slice;
- define how the result will be validated;
- update durable state/evidence when the change materially affects project knowledge;
- prove real provider interaction when changing harness control behavior.

## Canonical control loop

```text
USER REQUEST
 -> UNDERSTAND / ASSESS
 -> CONTEXT
 -> GENERATE PROMPT
 -> ACTUALLY DISPATCH TO HARNESS
 -> HARNESS WORKS
 -> STREAM / OBSERVE
 -> VERIFY
 -> SUPERVISE
 -> DECIDE
 -> CONTINUE / CORRECT / RECOVER / REPLAN / SWITCH / ASK / APPROVE / BLOCK / STOP / COMPLETE
 -> if more harness work is needed: generate and dispatch a new prompt
```

A harness session is not complete merely because a process exited. A Keystone autonomous continuation must be grounded in evidence and result in a real next harness interaction when work remains.

## Supervisor behavior

The supervisor should intervene minimally:

- normal behavior → observe;
- minor inefficiency → record;
- repeated inefficiency → optimize context/tooling;
- requirement drift → intervene;
- unsupported completion claim → verify;
- environment blocker → let the harness recover when safely possible;
- loop/no-progress → replan;
- security or destructive risk → stop or require approval;
- genuine user decision → expose the harness/Keystone question and wait.

Do not manufacture blockers merely because additional improvements are possible. Respect explicit release criteria and distinguish hard blockers from optional hardening.

## Context discipline

Use progressive context:

1. project identity and active work;
2. directly relevant requirements, architecture, decisions, files, tests, policies and evidence;
3. historical or large artifacts only when needed.

Do not read the entire repository or every historical document by default. Prefer native harness session continuity so established context does not need to be resent.

## Implementation quality

- Keep interfaces small and testable.
- Prefer explicit schemas over implicit conventions.
- Keep provenance on derived state and evidence.
- Make recovery/checkpoint behavior first-class.
- Treat prompt generation and dispatch as observable state transitions.
- Keep conversation history durable and replayable.
- Avoid premature vector databases, agent swarms, custom model training, or production-operations complexity when a simpler deterministic mechanism is sufficient.
- Add tests for state transitions, policy boundaries, adapter behavior, evidence validity, prompt dispatch, streaming, session continuity, and recovery paths.

## Reference documents

Read these before architectural changes:

- `docs/KEYSTONE_PRODUCT_MODEL.md` — current product model and control-loop definition.
- `docs/KEYSTONE_ARCHITECTURE_V2.md` — system architecture and durable lifecycle.
- `docs/IMPLEMENTATION_STATUS.md` — executable capability audit; do not treat a row as proof when current behavior contradicts it.
