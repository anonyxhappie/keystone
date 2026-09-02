# Keystone V2.1.0 Release Audit

Date: 2026-09-03
Target Release: `v2.1.0`
Git Branch: `main`

## Executive Summary

Keystone V2.1.0 is the hardened, verified implementation of Keystone's engineering intelligence and control layer supervising existing development harnesses. This audit documents all implemented capabilities, test verifications, real environment smoke tests, fixture simulations, and architectural boundaries.

---

## 1. What Was Implemented

1. **Canonical 12-State Control Lifecycle**:
   - Strictly enforced transitions: `REQUEST → UNDERSTAND → ASSESS → PLAN → CONTEXT → DISPATCH → EXECUTE → OBSERVE → VERIFY → EVALUATE → SUPERVISE → DECIDE`.
   - `NextAction` populated with canonical target, rationale, risk score, allowed flag, and `PolicyDecision` (`ALLOW` / `REQUIRE_APPROVAL`).

2. **Real Headless Harness Provider Bridges**:
   - **Antigravity Adapter**: Drives `agy -p <prompt> --output-format stream-json`, flattens nested schemas (`step_update`, `result`, `init`, `turn`), normalizes events into observations (`TOOL_STARTED`, `FILE_CHANGED`, `COMPLETION_CLAIM`), extracts token usage statistics, supports session continuation.
   - **Codex Adapter**: Drives `codex exec --json` (and `codex exec resume <session-id> <prompt>`), normalizes JSONL protocol events (`thread.started`, `turn.started`, `turn.finished`), captures tool invocations and file modifications, extracts session identity, and enforces command-line argument isolation.

3. **Deterministic Verification & Supervisor**:
   - Findings system detecting:
     - `UNVERIFIED_COMPLETION` (`F-UNVERIFIED`): Rejects harness claims when deterministic validation has not passed.
     - `PREMATURE_COMPLETION` (`F-PREMATURE`): Detects claims unsupported by requirements.
     - `REQUIREMENT_DRIFT` (`F-DRIFT`): Detects changes drifting from acceptance criteria.
     - `ARCHITECTURE_DRIFT` (`F-ARCH`): Detects changes violating architectural boundaries.
     - `UNEXPECTED_SCOPE` (`F-SCOPE`): Detects modifications to out-of-scope files.
     - `LOOP` (`F-LOOP`): Flags identical repeated actions without state progress.
     - `REPEATED_FAILURE` (`F-REPEAT`): Halts repeated failures (default threshold = 3).
     - `EXCESSIVE_ACTIVITY` (`F-TOOLS`, `F-CONTEXT`): Halts runs exceeding budget caps.
     - `INEFFICIENT_ACTIVITY` (`F-READS`): Detects duplicate file reads with low information gain.
   - All findings retain deterministic provenance and evidence references (`EvidenceIDs`).

4. **Multi-Harness Switching & State Continuity**:
   - Checkpoint-safe switching across different harness adapters.
   - Preserves `WorkOrderID`, `Request`, `ContextManifest`, `Observations`, `EvidenceIDs`, `Findings`, and `PolicyDecisions`.
   - Rejects illegal cross-provider session resumption, logging `RESUME_HARNESS_MISMATCH` and starting a clean session on the newly designated provider.

5. **Durable Snapshot, Journal & Replay**:
   - Append-only event journal (`events.jsonl`) with monotonic sequence numbers, atomic writes, and idempotency key conflict protection.
   - Materialized `Snapshot` cached for fast status, with automatic fallback reconstruction from the event journal when snapshot files are missing or corrupted.
   - Comprehensive `keystone replay <run-id>` reconstructing the full operational history.

6. **Full CLI Audit (All 13 Commands)**:
   - `keystone init`: Initializes project state and detects project capabilities.
   - `keystone status`: Reports live run state, active harness, and next action.
   - `keystone ask`: Compiles work packet without dispatching.
   - `keystone run`: Full automated control loop from request through verification.
   - `keystone continue`: Resumes paused, approved, or checkpointed runs.
   - `keystone pause`: Writes durable pause marker to cleanly interrupt observations.
   - `keystone approve`: Approves high-risk or policy-gated operations.
   - `keystone stop`: Transitions active run to terminal `STOPPED` state safely.
   - `keystone validate`: Runs deterministic validation tiers and captures results.
   - `keystone review`: Evaluates evidence, git diffs, and supervisor findings.
   - `keystone replay`: Reconstructs complete run report from the event journal.
   - `keystone doctor`: Audits host capabilities, Git state, and installed harnesses.
   - `keystone version`: Reports current semantic version (`2.1.0`).

---

## 2. Real Tool Smoke Tests (Host Environment)

1. **Google Antigravity (`agy`)**:
   - Host path: `/Users/akshay/.local/bin/agy` (v1.1.24)
   - Verified live in `TestLiveAntigravitySmoke`:
     - CLI discovered via `exec.LookPath`.
     - Executed live against real Google Antigravity backend with stream-JSON formatting.
     - Normalized streaming events (`init` → `step_update` → `result`).
     - Session ID captured and persisted in snapshot and journal.
     - Token accounting verified (`input_tokens`, `output_tokens`).
     - Deterministic validation executed and passed.
     - Final state transitioned to `COMPLETE`.
     - Full operational history verified via `Replay()`.

2. **OpenAI Codex (`codex`)**:
   - Host path: `/opt/homebrew/bin/codex` (`codex-cli 0.152.1`)
   - Verified live in `TestLiveCodexFailClosedSmoke`:
     - Discovered and version-probed successfully by `keystone doctor`.
     - Launched live via `codex exec --json`.
     - Correctly parsed startup events (`thread.started`).
     - Safely caught OpenAI account quota exhaustion without hanging or crashing.
     - Fails closed: records error observation, transitions to `BLOCKED`, requires explicit approval (`Allowed: false`), preventing uncontrolled execution.

---

## 3. Mock & Fixture Verifications

1. **Scripted Deterministic Scenarios (Scenarios A through I)**:
   - Scenario A: Successful work order execution to verified `COMPLETE`.
   - Scenario B: False completion claim rejected when validation fails (`UNVERIFIED_COMPLETION`).
   - Scenario C: Bounded repeated failures stopped safely after reaching attempt limit.
   - Scenario D: Excessive activity halted when tool call limit exceeded.
   - Scenario E: High-risk actions (`destroy database`) blocked pending human approval.
   - Scenario F: Process crash / abnormal harness termination caught and safely stopped.
   - Scenario G: Inefficient repeated reads flagged with `INEFFICIENT_ACTIVITY` recommendation.
   - Scenario H: Multi-harness switching preserving work order, context, and evidence.
   - Scenario I: Pause, checkpoint, resume, and continue state machine transitions.

2. **Resilience & Boundary Tests**:
   - Invalid / corrupted journal lines properly rejected with descriptive error.
   - Corrupted materialized snapshots rebuilt from event history.
   - Confined state paths prevent directory traversal / symlink escape outside `.keystone/`.
   - Sensitive credentials and tokens redacted from captured command outputs and logs.
   - Cross-provider session resumption mismatch caught and fallback initiated.

---

## 4. Known Boundaries and Gaps

- **Remote Cloud CI / Browser Adapters**: Explicitly outside the local-first engine boundary. Remote cloud execution requires configuring a remote adapter or external runner command.
- **Semantic Model Review**: Keystone provides deterministic review inputs and evidence. Unbounded natural-language model judgment is kept behind external harnesses to preserve deterministic verification.
- **Process Registry / Daemon**: Keystone uses local-first durable filesystem markers (`control/pause.json`, snapshots) rather than background daemon processes or OS PID registries.

---

## 5. Verification Sign-Off

- `go test -v ./...` → **PASS** (100% test pass across all packages)
- `go vet ./...` → **PASS** (zero static analysis diagnostics)
- `go test -race ./...` → **PASS** (data-race free concurrency)
- `go build -o keystone ./cmd/keystone` → **PASS**
- Release Version: `v2.1.0`
