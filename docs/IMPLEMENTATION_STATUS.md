# Keystone V2 implementation status

This matrix is an audit of executable behavior on `main` for release `v2.1.1`. A capability is not marked implemented solely because a type or interface exists.

| Capability | Implementation | Executable verification | Status |
|---|---|---|---|
| Canonical lifecycle and NextAction | `internal/runtime`, `internal/control` | state-machine and engine tests; canonical 12-state transitions; NextAction target and policyDecision populated | IMPLEMENTED |
| Durable snapshot and confined atomic writes | `internal/state` | state persistence, corruption recovery, confinement tests | IMPLEMENTED |
| Append-only journal, idempotency, replay | `internal/observation`, `internal/control/replay.go` | journal and engine replay tests; `keystone replay <run-id>` reconstructs work order, request, state, harness, observations, evidence, findings, and decisions | IMPLEMENTED |
| Local process harness bridge | `internal/harness/local.go` | local adapter, process crash detection, default timeout safety, and engine integration tests | IMPLEMENTED |
| Codex headless provider bridge | `internal/harness/provider.go` | `codex-cli 0.152.1` installed; startup probe, argument isolation (`exec resume`), event normalization, usage tracking, and fail-closed error handling verified | IMPLEMENTED |
| Antigravity headless provider bridge | `internal/harness/provider.go` | `agy` 1.1.24 installed and authenticated; live end-to-end smoke test passes with real streaming JSON events, token accounting, deterministic validation, and replay | IMPLEMENTED |
| Durable provider session identity and resume | `internal/domain`, `internal/state`, `internal/control`, `internal/harness/provider.go` | checkpoint/snapshot persistence, session resumption, and cross-provider mismatch protection | IMPLEMENTED |
| Manual harness packet integration | `internal/harness` | `keystone ask` packet generation | IMPLEMENTED |
| Deterministic Git and validation evidence | `internal/git`, `internal/validation`, `internal/evidence` | evidence lifecycle, multi-tier validation, and engine tests | IMPLEMENTED |
| Completion claim supervision | `internal/supervisor` | unsupported claim, loop, repeated failure, requirement drift, and inefficient reads tests; completion denied without passing validation | IMPLEMENTED |
| Bounded retries and safe stop | `internal/control/engine.go` | engine limit, consecutive failure threshold, and fail-closed paths | IMPLEMENTED |
| Checkpointed continuation and approval gate | `internal/checkpoint`, `internal/control` | continuation approval, high-risk policy gating, and recovery tests | IMPLEMENTED |
| Learning candidate lifecycle | `internal/learning` and engine findings | candidate transition and learning lifecycle test | IMPLEMENTED |
| Harness switching | adapter factory, durable checkpoint, and switch event | multi-harness switching test preserving work order, context manifest, evidence, findings, and decisions | IMPLEMENTED |
| Impact-aware project context | `internal/project`, `internal/context` | context ranking, instruction injection, progressive budget re-planning, deterministic structural outline compression, and decision audit trail tests | IMPLEMENTED |
| Release/deployment/incident modeling | `internal/delivery` | production deployment, policy gating, and incident traceability tests | IMPLEMENTED |
| Complete CLI audit | `cmd/keystone` | all 13 subcommands (`init`, `status`, `ask`, `run`, `continue`, `pause`, `approve`, `stop`, `validate`, `review`, `replay`, `doctor`, `version`) verified in integration suite | IMPLEMENTED |
| CI/browser/operational adapters | local command validation and configurable process boundary; delivery policy state | validation and delivery tests; remote cloud execution remains an explicit adapter boundary | PARTIAL |
| Semantic architecture/requirement review | deterministic inputs exist; no model adapter | review input only; model judgment belongs behind capabilities | PARTIAL |
| Cross-process pause of a live harness | durable control marker and polling observer | live local-process pause integration test; no daemon/PID registry required by local-first architecture | PARTIAL |

The PARTIAL entries are explicit local-first boundaries, not broken capabilities. Keystone fails closed when external or remote authority is required. All core capabilities, real provider bridges (Antigravity and Codex), and release verification criteria are satisfied for release `v2.1.1`.
