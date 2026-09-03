# Changelog

## 2.1.1

Intelligent context budget handling, deterministic progressive re-planning, and timestamp safety:
- Replaced immediate fail-closed context budget blocking with progressive context re-planning (`PlanContext`).
- Deterministic structural compression for repository files (test suites, markdown documentation, and exported source code declarations) with minimal token footprints.
- Redundancy elimination and task-aware relevance ranking preserving project instruction guardrails (`AGENTS.md`, root `README.md`, `CLAUDE.md`, `.cursorrules`), user constraints, and critical evidence.
- Full context decision audit trails recorded in `packet.ContextDecisions` and durable manifests in `.keystone/manifests/` detailing retained, omitted, and compressed items with rationale and token savings.
- Fail-closed bounded safety check when non-negotiable mandatory items alone exceed the configured budget.
- Fixed zero-value timestamps across `WorkOrder.UpdatedAt`, `Incident.CreatedAt`, and status inspection.
- Added `-C` / `--root` and `KEYSTONE_ROOT` directory override support to the CLI for multi-repository workflows.

## 2.1.0

Hardened completion of Keystone V2 engine, supervisor, and real headless harness providers:
- Real headless provider bridges for Google Antigravity (`agy`) and OpenAI Codex (`codex exec`).
- Live smoke-tested Antigravity execution with real streaming-JSON event normalization, token usage tracking, deterministic validation, and durable replay.
- Live smoke-tested Codex failure isolation: startup probe, argument isolation (`exec resume` session argument syntax), and fail-closed handling on account usage limits.
- Full 12-state canonical runtime lifecycle (`REQUEST` through `DECIDE`), with `Target` and `PolicyDecision` populated on all `NextAction` decisions.
- Multi-harness switching with durable state continuity across attempts and pause/resume cycles.
- Supervisor findings expanded to detect requirement drift (`F-DRIFT`), repeated inefficient reads (`F-READS`), unverified completion claims (`F-UNVERIFIED`), premature completion (`F-PREMATURE`), loop detection (`F-LOOP`), repeated failure (`F-REPEAT`), and activity exhaustion (`F-TOOLS`, `F-CONTEXT`).
- Fixed observation collector concurrency bug that caused out-of-order slice reads on child process stdout pipes.
- Reconstructed comprehensive replay reports in `keystone replay <run-id>` with work orders, requests, state transitions, harness identity, observations, evidence, findings, and policy decisions.
- Full CLI audit verifying all 13 subcommands (`init`, `status`, `ask`, `run`, `continue`, `pause`, `approve`, `stop`, `validate`, `review`, `replay`, `doctor`, `version`).

## 2.0.1

Correct the Go module path and package installation path for the V2 major release. The module is now published as `github.com/anonyxhappie/keystone`, with matching internal imports and installation documentation.

## 2.0.0

The V2 control-plane implementation adds the executable canonical lifecycle, durable event journal and snapshot recovery, local-process and manual harness boundaries, normalized observations, evidence-backed verification, deterministic supervision, bounded full-auto execution, checkpoints, pause/approval/stop controls, learning lifecycle, harness switching, project intelligence, delivery state, policy enforcement, artifact redaction, replay, and the complete CLI.

Known limitations are recorded in docs/IMPLEMENTATION_STATUS.md and docs/IMPLEMENTATION_STATUS.json. Remote CI/browser/operations execution and semantic model review remain explicit adapter boundaries.
