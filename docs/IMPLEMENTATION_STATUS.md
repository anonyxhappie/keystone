# Keystone V2 implementation status

This matrix is an audit of executable behavior on `release/v2.0.0`. A capability is not marked implemented solely because a type or interface exists.

| Capability | Implementation | Executable verification | Status |
|---|---|---|---|
| Canonical lifecycle and NextAction | `internal/runtime`, `internal/control` | state-machine and engine tests | IMPLEMENTED |
| Durable snapshot and confined atomic writes | `internal/state` | state persistence, corruption, confinement tests | IMPLEMENTED |
| Append-only journal, idempotency, replay | `internal/observation`, `internal/control/replay.go` | journal and engine replay tests; `keystone replay` | IMPLEMENTED |
| Local process harness bridge | `internal/harness/local.go` | local adapter and engine integration tests | IMPLEMENTED |
| Manual harness packet integration | `internal/harness` | `keystone ask` packet generation | IMPLEMENTED |
| Deterministic Git and validation evidence | `internal/git`, `internal/validation`, `internal/evidence` | evidence and engine tests | IMPLEMENTED |
| Completion claim supervision | `internal/supervisor` | unsupported claim and loop tests | IMPLEMENTED |
| Bounded retries and safe stop | `internal/control/engine.go` | engine limit and failure paths | IMPLEMENTED |
| Checkpointed continuation and approval gate | `internal/checkpoint`, `internal/control` | continuation approval test | IMPLEMENTED |
| Learning candidate lifecycle | `internal/learning` and engine findings | learning lifecycle test | IMPLEMENTED |
| Harness switching | adapter factory, durable checkpoint, and switch event | control integration test with two real local processes | IMPLEMENTED |
| Impact-aware project context | `internal/project`, `internal/context` | context ranking test | IMPLEMENTED |
| Release/deployment/incident modeling | `internal/delivery` | production deployment and incident tests | IMPLEMENTED |
| CI/browser/operational adapters | local command validation and configurable process boundary; delivery policy state | validation and delivery tests; remote adapters remain external | PARTIAL |
| Semantic architecture/requirement review | deterministic inputs exist; no model adapter | review input only | PARTIAL |
| Cross-process pause of a live harness | durable control marker and polling observer | live local-process pause integration test; no daemon/PID registry | PARTIAL |

The PARTIAL entries are explicit local-first boundaries, not successful-result claims. They are non-blocking only when the relevant remote or semantic adapter is not in scope; Keystone fails closed when such evidence is required.

The PARTIAL entries are explicit non-blocking local-first boundaries. The local V2 core is release-ready; Keystone remains conservative whenever a remote/vendor-specific adapter or semantic review provider is unavailable.
