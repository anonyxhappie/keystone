# Keystone

> Persistent engineering intelligence between human intent and AI development harnesses.

Keystone is an open-source, language-, framework-, architecture-, and harness-agnostic engineering intelligence/control layer. It sits between a user's request and existing AI development and operations harnesses.

Keystone does not replace the harness. It maintains durable project state, compiles relevant context, generates harness-specific work packets, observes results, evaluates correctness and harness behavior, detects unsupported completion claims, drift, loops and waste, and continues work within explicit policy.

## V1 quick start

```bash
go install github.com/anonyxhappie/keystone/cmd/keystone@v1.0.0
cd your-project
keystone init
keystone status
keystone ask "add the requested feature"
```

`keystone init` is safe to run against an existing project and creates a portable `.keystone/` state boundary. `keystone ask` creates a durable WorkOrder and emits a harness-neutral WorkPacket that can be handed to an existing AI harness.

## V1 commands

- `keystone init` — inspect and initialize project state.
- `keystone status` — inspect durable project state.
- `keystone ask "..."` — create a work order and generate the next harness packet.
- `keystone doctor` — basic local installation check.

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

See [`docs/KEYSTONE_ARCHITECTURE_V1.md`](docs/KEYSTONE_ARCHITECTURE_V1.md) and [`docs/V1_IMPLEMENTATION_PLAN.md`](docs/V1_IMPLEMENTATION_PLAN.md) for the V1 boundary.
