# Keystone

> Persistent engineering intelligence between human intent and AI development harnesses.

Keystone is an open-source, language-, framework-, architecture-, and harness-agnostic engineering intelligence/control layer. It sits between a user's request and existing AI development and operations harnesses.

**Keystone is not another coding agent. The harness does the work; Keystone keeps the engineering thread moving in the right direction.** It understands intent, compiles relevant context, generates and actually dispatches harness instructions, observes the real harness conversation and tool activity, verifies results, detects drift/loops/unsupported claims, recovers where the harness can safely recover, and continues until verified completion or a genuine human/policy boundary.

## Current product model

Read [`docs/KEYSTONE_PRODUCT_MODEL.md`](docs/KEYSTONE_PRODUCT_MODEL.md) before implementing or evaluating Keystone behavior. It is the product-level source of truth for the current direction.

The core experience is:

```text
USER
 ↓
KEYSTONE: understand + contextualize + guide
 ↓
REAL HARNESS: work + tools + commands + files
 ↓
KEYSTONE: stream + observe + verify + supervise
 ↓
NEW EVIDENCE-DERIVED HARNESS PROMPT
 ↓
SAME SESSION WHEN POSSIBLE
 ↓
...
 ↓
VERIFIED RESULT
```

The goal is not to make Keystone own more engineering actions. The goal is **verified useful engineering progress per unit of human attention**.

Historical release tags are not the development baseline. Until a stable release is declared, the current repository state is the only baseline.

## V2 quick start

```bash
go install github.com/anonyxhappie/keystone/cmd/keystone@latest
cd your-project
keystone init
keystone status
keystone run "add the requested feature"
```

`keystone init` creates a portable `.keystone/` state boundary. `keystone run` creates a durable work order, advances the canonical lifecycle, executes a configured harness, persists observations and evidence, validates the result, and continues or stops through the state machine. Provider sessions and conversation state are intended to remain durable across turns and restarts.

Configure a local process harness in `.keystone/harness.json`:

```json
{"name":"local-process","command":"your-harness","args":[],"timeoutSeconds":300}
```

Keystone also bridges the documented headless CLIs of existing AI harnesses. Use either provider explicitly when desired:

```json
{"provider":"codex","name":"codex","command":"codex","timeoutSeconds":1800}
```

```json
{"provider":"antigravity","name":"antigravity","command":"agy","timeoutSeconds":1800}
```

Codex JSONL events and Antigravity stream-JSON events are normalized into durable session, turn, tool, command, file, usage, result, and error observations. Provider conversation/session IDs are stored in checkpoints so provider-native resume can preserve continuity where supported. Keystone sets the child process working directory directly and never adds provider permission-bypass flags.

## Commands

Completion requires evidence appropriate to the work and at least one applicable deterministic validation check where the project exposes one; a harness process exit or completion claim is never sufficient by itself.

- `keystone init` — inspect and initialize project state.
- `keystone status` — inspect durable project state.
- `keystone ask "..."` — create a work order and generate the next harness packet.
- `keystone run "..."` — execute the supervised multi-turn control loop through the configured harness.
- `keystone continue` — reconstruct durable work and resume through permitted recovery transitions.
- `keystone pause`, `keystone approve`, `keystone stop` — record explicit control decisions and approval provenance.
- `keystone validate` — run deterministic project checks without a harness.
- `keystone review` — inspect persisted findings and review recommendation.
- `keystone replay <run-id>` — replay events and reconstruct the canonical machine state.
- `keystone doctor` — inspect Git and discovered harness capabilities.
- `keystone version` — print the executable version.

Interactive mode should restore the last valid project, harness, and session and present the actual provider conversation. `/projects`, `/sessions`, `/harness`, `/resume`, `/pause`, `/approve`, `/stop`, `/review`, `/replay`, and `/doctor` are control commands; ordinary natural-language input is a user request to the active harness through Keystone.

## Design principles

- **Harness-first:** existing AI harnesses remain the execution layer and should perform the majority of actionable engineering work.
- **Real prompt dispatch:** generating a WorkOrder or starting a provider is not enough; Keystone must actually deliver the instruction and record proof.
- **Persistent conversation:** real provider sessions and normalized user/harness/tool events are durable and replayable.
- **Project-owned state:** engineering context survives model, agent, machine, and context changes.
- **Evidence over claims:** completion is verified through observable artifacts and deterministic checks.
- **Autonomy within policy:** full-auto operation continues ordinary work without repeated user `continue` commands, but stops at genuine approval, permission, security, or unrecoverable-failure boundaries.
- **Adaptive supervision:** Keystone evaluates outcomes and how effectively the harness worked.
- **Intelligent recovery:** recoverable environment failures should be handed back to the harness with a new evidence-derived instruction rather than blindly retried.
- **No-progress resistance:** retries must add information or change strategy.
- **Project-specific learning:** repeated successes and mistakes can improve future context, strategy, recovery, and supervision.
- **Language/framework agnostic:** technology details belong behind capabilities/adapters.
- **Extensible:** validation, browser, Git, CI/CD, security, observability and operations capabilities can evolve without changing the core state model.

## Repository structure

```text
.keystone/                 # portable project state
internal/domain/           # technology-neutral entities
internal/project/          # capability discovery
internal/context/          # progressive context compilation
internal/work/             # work orders and packets
internal/harness/          # harness boundary
internal/evidence/         # durable evidence
internal/supervisor/       # behavior supervision
internal/policy/           # risk/policy decisions
internal/state/            # state persistence
docs/                      # architecture and implementation specifications
```

See [`docs/KEYSTONE_PRODUCT_MODEL.md`](docs/KEYSTONE_PRODUCT_MODEL.md), [`docs/KEYSTONE_ARCHITECTURE_V2.md`](docs/KEYSTONE_ARCHITECTURE_V2.md), [`docs/V2_OPERATIONS.md`](docs/V2_OPERATIONS.md), and [`docs/IMPLEMENTATION_STATUS.md`](docs/IMPLEMENTATION_STATUS.md).
