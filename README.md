# Keystone

> Persistent engineering intelligence between human intent and AI development harnesses.

Keystone is an open-source, language-, framework-, architecture-, and harness-agnostic engineering intelligence/control layer. It sits between a user's request and existing AI development and operations harnesses.

Keystone does not replace the harness. It maintains durable project state, compiles relevant context, generates harness-specific work packets, observes results, evaluates correctness and harness behavior, detects unsupported completion claims, drift, loops and waste, and continues work within explicit policy.

## V2 quick start

```bash
go install github.com/anonyxhappie/keystone/v2/cmd/keystone@v2.0.1
cd your-project
keystone init
keystone status
keystone run "add the requested feature"
```

`keystone init` creates a portable `.keystone/` state boundary. `keystone run` creates a durable work order, advances the canonical lifecycle, executes a configured harness, persists observations and evidence, validates the result, and stops or completes only through the state machine. When no harness configuration exists, Keystone auto-detects a working `codex` or `agy` CLI; otherwise it records a durable `BLOCKED` checkpoint.

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

Codex JSONL events and Antigravity stream-JSON events are normalized into durable session, turn, tool, command, file, usage, result, and error observations. Provider conversation/session IDs are stored in checkpoints so `keystone continue` can use provider-native resume. Keystone sets the child process working directory directly and never adds provider permission-bypass flags.

The command receives the structured work packet on stdin. Its newline-delimited stdout is normalized into observations. `keystone ask` remains the manual/instruction-file integration.

## Commands

Completion requires at least one discovered deterministic validation check to pass; a harness process exit or completion claim is never sufficient by itself.

- `keystone init` — inspect and initialize project state.
- `keystone status` — inspect durable project state.
- `keystone ask "..."` — create a work order and generate the next harness packet.
- `keystone run "..."` — execute the bounded control loop through the configured harness.
- `keystone continue` — reconstruct the latest durable work order and resume only through permitted recovery transitions.
- `keystone pause`, `keystone approve`, `keystone stop` — record explicit control decisions and approval provenance.
- `keystone validate` — run deterministic project checks without a harness.
- `keystone review` — inspect persisted findings and review recommendation.
- `keystone replay <run-id>` — replay events and reconstruct the canonical machine state.
- `keystone doctor` — inspect Git and discovered harness capabilities.
- `keystone version` — print the executable version.

## Design principles

- **Harness-agnostic:** existing AI harnesses remain the execution layer.
- **Project-owned state:** engineering context survives model, agent, machine, and context changes.
- **Evidence over claims:** completion is verified through observable artifacts and deterministic checks.
- **Autonomy within policy:** full-auto operation is possible without unrestricted destructive authority.
- **Adaptive supervision:** Keystone evaluates outcomes and how effectively the harness worked.
- **Project-specific learning:** repeated successes and mistakes can improve future context, strategy and supervision.
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

See [docs/KEYSTONE_ARCHITECTURE_V2.md](docs/KEYSTONE_ARCHITECTURE_V2.md), [docs/V2_OPERATIONS.md](docs/V2_OPERATIONS.md), and [docs/IMPLEMENTATION_STATUS.md](docs/IMPLEMENTATION_STATUS.md) for the V2 contracts, operations, limitations, and executable audit.
