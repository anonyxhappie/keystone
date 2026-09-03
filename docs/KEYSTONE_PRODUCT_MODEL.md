# Keystone Product Model

## Status of this document

This document is the product-level source of truth for the current Keystone direction. It clarifies the intended behavior of the system before further implementation work.

Historical release tags are not the development baseline. The current repository state is the only baseline until Keystone reaches a stable release.

## What Keystone is

Keystone is a **persistent engineering intelligence and control layer between a human and an existing AI harness**.

The harness is the worker. Keystone is the drive, navigator, supervisor, memory, verifier, and continuity layer that keeps the harness moving in the right direction for a long-running task.

Keystone is **not**:

- another coding-agent runtime;
- a replacement for Codex, Antigravity, Claude Code, Cursor, or similar harnesses;
- a second model that performs the majority of implementation itself;
- a task queue whose job ends after one provider invocation;
- a wrapper that merely starts a harness and validates its exit code.

The product succeeds when a user can give Keystone a meaningful engineering objective and the selected harness can continue doing useful work for a long time with little or no human intervention.

## The fundamental relationship

```text
                         HUMAN
                           |
                    intent / decisions
                           v
                    +-------------+
                    |   KEYSTONE  |
                    |             |
                    | understand  |
                    | contextualize|
                    | guide       |
                    | observe     |
                    | verify      |
                    | supervise   |
                    | remember    |
                    +------+------+ 
                           |
                    precise instruction
                           v
                 +---------------------+
                 | EXISTING AI HARNESS  |
                 | Codex / Antigravity  |
                 | / other adapters     |
                 +----------+----------+
                            |
                code / tools / commands
                files / browser / tests
                            |
                            v
                    real observations
                            |
                            +-------------> KEYSTONE
                                             |
                                      evaluate progress
                                             |
                              +--------------+--------------+
                              |              |              |
                           continue       redirect       ask user
                              |              |              |
                              +-------+------+              |
                                      |
                              next harness prompt
                                      |
                                      v
                              SAME SESSION when possible
```

## Harness-first principle

Keystone should prefer asking the harness to perform actionable work whenever the harness is capable of doing that work.

For example, if a local PostgreSQL service is unavailable and the harness has the tools and permissions to recover it, Keystone should not stop after reporting the failed test. It should diagnose the situation, generate a targeted recovery instruction, send that instruction to the harness, and supervise the recovery.

The intended flow is:

```text
validation failure
    -> classify failure
    -> determine whether harness can recover it
    -> generate recovery instruction
    -> dispatch to harness
    -> harness diagnoses/fixes/verifies
    -> observe
    -> validate
    -> continue
```

Keystone may directly perform control-plane operations such as state persistence, policy checks, evidence processing, context compilation, checkpointing, and deterministic supervision. It should not unnecessarily duplicate the harness's engineering capabilities.

## The real control loop

A Keystone run is not a single harness invocation. It is a potentially long-lived feedback loop:

```text
USER REQUEST
    |
    v
UNDERSTAND INTENT + CONSTRAINTS
    |
    v
ASSESS RISK / POLICY / PROJECT STATE
    |
    v
COMPILE MINIMUM SUFFICIENT CONTEXT
    |
    v
GENERATE HARNESS PROMPT
    |
    v
DISPATCH PROMPT TO REAL HARNESS SESSION
    |
    v
HARNESS WORKS
    |
    +----> stream conversation
    +----> observe tools
    +----> observe commands
    +----> observe files
    +----> observe tests/browser/runtime
    +----> observe usage/results
    |
    v
SUPERVISE + VERIFY
    |
    v
CLASSIFY OUTCOME
    |
    +--> COMPLETE (only when verified)
    +--> CONTINUE
    +--> CORRECT
    +--> RECOVER ENVIRONMENT
    +--> REPLAN
    +--> SWITCH HARNESS
    +--> ASK USER
    +--> APPROVE / BLOCK / STOP
    |
    +---- if more harness work is needed ----+
                                             |
                                             v
                                  GENERATE NEW PROMPT
                                             |
                                             v
                                    DISPATCH TO HARNESS
                                             |
                                             +----> loop
```

Every autonomous continuation must result from new evidence or a deliberate state transition. A retry is not a duplicate invocation of the same failed action.

## Prompt dispatch is a first-class operation

Creating a WorkOrder or starting a provider process does not mean the harness was instructed to do the work.

Every real harness turn must have durable evidence of:

- prompt ID;
- WorkOrder ID;
- run ID;
- harness ID;
- harness session ID;
- context manifest/version;
- reason for the instruction;
- strategy/hypothesis when applicable;
- expected information gain when applicable;
- dispatch timestamp;
- dispatch result/provider acknowledgement when available.

The canonical lifecycle is:

```text
ContextCompiled
-> PromptGenerated
-> PromptDispatched
-> HarnessTurnStarted
-> HarnessObserved
-> HarnessTurnCompleted
-> SupervisorEvaluation
-> NextAction
```

Keystone must never claim `PromptDispatched` unless the adapter actually delivered the instruction to the provider.

## The harness conversation is the primary user experience

Keystone's interactive mode should feel like a transparent control channel around the selected harness, not like a separate Keystone chatbot.

The visible conversation should primarily contain:

- the user's messages;
- actual harness responses;
- meaningful tool/command/file events;
- Keystone guidance when Keystone intervenes;
- validation/evidence summaries;
- approval questions and decisions.

Internal lifecycle telemetry remains available in verbose/debug/replay views but should not drown out the real harness conversation.

### Message model

Normalize conversation events into at least:

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

The normalized event stream is durable and replayable. The terminal/TUI is a rendering of that stream, not its source of truth.

## Session continuity

A real provider session is a first-class durable object.

Persist at minimum:

- provider/harness;
- real provider session ID;
- project;
- human-readable title;
- creation/update timestamps;
- session status;
- conversation/events;
- current WorkOrder/run;
- prompt IDs;
- checkpoints;
- pending user input;
- usage and provider results.

When the provider supports native resume, Keystone should resume the actual session rather than simulate continuity with a new session and a summary.

When the user starts `keystone`, restore the last valid:

- project;
- harness;
- session;
- mode;
- run/checkpoint;
- pending interaction.

The user should not have to re-select these on every startup.

## Switching sessions and projects

`/sessions` selects a real existing session and must switch the entire active conversational state to that session.

Switching from session A to session B must update together:

```text
active project
active harness
active session
conversation history
pending state
current checkpoint/run
```

No messages from the previous session should remain mixed into the active conversation.

Similarly, project switching must restore the selected project's last valid harness/session state and never cross repository boundaries.

## Human interaction

The normal path should not require the user to repeatedly type `continue`.

In Full Auto mode Keystone continues ordinary policy-allowed work across multiple harness turns.

Human interaction is required when the harness or Keystone reaches a genuine decision boundary, such as:

- missing credentials/secrets;
- destructive or irreversible action requiring approval;
- product/architecture ambiguity that cannot be resolved from project evidence;
- policy/security boundary;
- unsafe rollback;
- unrecoverable external dependency;
- a harness request that genuinely requires the user's decision.

### Harness questions

If the harness asks a question, Keystone must expose the actual question and pause the autonomous loop.

The user's answer becomes a message in the same conversation and, where supported, is sent back into the same provider session.

When a decision has meaningful rationale, persist:

```text
question
user answer
reason/rationale
related session
related turn/prompt
timestamp
```

Do not hide the reason for a human intervention inside an internal log.

## Failure classification and recovery

A validation failure does not automatically mean the harness failed.

At minimum classify outcomes as:

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

### Environment blockers

For an environment blocker, Keystone should first determine whether the existing harness can safely recover the environment.

Example:

```text
PostgreSQL unreachable
        |
        v
ENVIRONMENT_BLOCKER
        |
        v
Can harness diagnose/recover?
        |
       YES
        |
        v
send recovery prompt
        |
        v
harness starts/restarts/fixes DB
        |
        v
harness verifies connectivity
        |
        v
harness reruns affected validation
```

If the harness cannot recover it, Keystone should preserve the checkpoint and ask for the smallest necessary human intervention rather than repeatedly burning harness turns.

## Retry and no-progress policy

A retry must represent a meaningful change in strategy, information, environment, or execution state.

Keystone should detect repeated:

- identical errors;
- identical commands;
- identical hypotheses;
- identical prompts;
- unchanged repository state;
- unchanged environment state;
- no information gain.

Repeated failure without progress becomes `NO_PROGRESS` and should cause re-planning, harness switching when justified, or human intervention—not an endless retry loop.

## Full Auto

Full Auto is not "run the harness once without asking".

It means Keystone continuously drives a policy-allowed engineering thread:

```text
prompt -> work -> observe -> verify -> decide -> prompt -> work -> ...
```

The loop ends only when:

- completion is verified;
- human input/approval is genuinely required;
- policy blocks the action;
- the work is unrecoverable;
- the user explicitly pauses/stops it.

The harness should be able to perform many ordinary engineering actions between human interventions.

## Evidence over claims

The harness is not authoritative about whether the work is complete.

Separate:

```text
HARNESS CLAIM
    vs.
OBSERVED EVIDENCE
    vs.
KEYSTONE DECISION
```

For example, if a harness says "all tests pass" but no tests were executed, Keystone records an unsupported completion claim and continues supervision rather than accepting the statement.

## Autonomy and safety

Harness-first does not mean unrestricted authority.

Keystone continues to enforce policy around:

- destructive operations;
- production operations;
- credentials/secrets;
- data-loss operations;
- force pushes;
- irreversible migrations;
- security-sensitive changes;
- unsafe rollback.

The harness may be the worker, but Keystone remains responsible for ensuring that the worker operates within the user's configured authority.

Silence is never approval.

## Context discipline

Keystone should provide the harness with the minimum sufficient context for the current decision.

Use:

1. project identity and active work;
2. directly relevant requirements, architecture, decisions, files, tests, policies and evidence;
3. historical/large artifacts only when required.

Prefer native session continuity so previously established context does not need to be resent unnecessarily.

Context budgeting should optimize useful harness performance, not merely minimize token count.

## Harness selection

Explicit selection is authoritative:

```text
--harness codex
--harness antigravity
--harness auto
```

An explicit unavailable harness must fail closed or ask the user. Keystone must never silently substitute another harness.

Auto mode may select or switch harnesses when evidence supports the decision.

A harness switch must be visible and durable:

```text
Antigravity -> Codex
Reason: <evidence-based reason>
```

Resuming the same harness is not a harness switch.

## Learning

Keystone may learn from project × harness outcomes, including:

- successful recovery strategies;
- ineffective prompts;
- repeated failure patterns;
- context efficiency;
- validation strategy;
- harness suitability.

Learning must be evidence-backed, versioned, reversible, and unable to silently weaken policy.

## Product success metric

The most important metric is not the number of commands Keystone owns.

It is:

> **Verified useful engineering progress per unit of human attention.**

A strong Keystone run looks like:

```text
Human gives objective
        |
        v
Keystone gives harness precise direction
        |
        v
Harness does substantial work
        |
        v
Keystone observes and verifies
        |
        v
Keystone redirects only when needed
        |
        v
Harness continues
        |
        v
Human intervenes only at real decision boundaries
        |
        v
Verified project state
```

## Definition of a working V2 control loop

The core is not considered complete merely because:

- a provider process starts;
- a WorkOrder is created;
- a session ID is discovered;
- a validation command runs;
- a harness reports completion.

The minimum proof is an actual multi-turn interaction:

```text
USER REQUEST
-> Keystone generates prompt
-> prompt is actually dispatched
-> real harness receives it
-> harness performs work
-> real harness conversation is streamed
-> Keystone observes evidence
-> Keystone generates a new evidence-derived prompt
-> prompt is dispatched into the same session where supported
-> harness continues
-> Keystone verifies outcome
-> COMPLETE / ASK / BLOCK / STOP
```

If this cannot be demonstrated with a real provider adapter, the control loop is not yet complete.
