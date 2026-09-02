# Keystone V2 operations guide

## State and authority

Keystone has one authoritative lifecycle:

REQUEST -> UNDERSTAND -> ASSESS -> PLAN -> CONTEXT -> DISPATCH -> EXECUTE -> OBSERVE -> VERIFY -> EVALUATE -> SUPERVISE -> DECIDE

DECIDE can select CONTINUE, CORRECT, REPLAN, ASK, APPROVE, BLOCK, STOP, or COMPLETE. The event journal is the durable source of truth. .keystone/state.json is a materialized cache and may be rebuilt with keystone continue after a valid journal-backed corruption.

The planning authority creates a work packet. The configured harness has execution authority only for the process it launches. Approval authority is represented by a durable approval record and is never inferred from silence.

## Installation and initialization

Keystone is a standalone Go executable. The target project needs no Keystone runtime dependency.

    go install github.com/anonyxhappie/keystone/v2/cmd/keystone@v2.0.1
    cd project
    keystone init

Initialization detects repository capabilities and records instructions and topology in .keystone/project.json. State writes are confined to .keystone, use temporary files followed by sync and rename, and preserve the working tree.

## Harness integration

keystone ask creates a durable work order and prints a manual packet. The packet can be passed to an existing harness or human operator; Keystone does not implement an agent runtime.

For bounded local execution, configure .keystone/harness.json:

    {"name":"local-process","command":"your-harness","args":[],"timeoutSeconds":300}

The process receives one rendered work packet on stdin and emits newline-delimited output on stdout. Output is normalized into observations. Recognized prefixes include [tool_started], [tool_completed], [command_started], [command_completed], [file_read], and [file_changed]. Text containing completion language is retained as a COMPLETION_CLAIM, never as proof of completion.

The local adapter supports discovery, identity, start, send, observe, interrupt, resume, result, timeout and process failure. Missing observability is recorded as a durable OBSERVABILITY_LIMITATION event.

Keystone can auto-select a working `codex` or `agy` executable when no harness file is present. Explicit provider configurations are:

    {"provider":"codex","name":"codex","command":"codex","timeoutSeconds":1800}
    {"provider":"antigravity","name":"antigravity","command":"agy","timeoutSeconds":1800}

The Codex adapter launches `codex exec --json` and resumes with `codex exec resume SESSION-ID`. The Antigravity adapter launches `agy -p ... --output-format stream-json` and resumes with `--conversation CONVERSATION-ID`. The child working directory is set by Keystone, which also preserves the provider's default permission policy; dangerous permission-bypass flags are never inserted automatically. A provider turn is a bounded process, so follow-up `Send` starts a resumed provider turn after the current process exits.

`keystone doctor` reports provider availability, version, control surfaces, observation surfaces, evidence surfaces, and known limitations. An installed binary is not treated as authenticated or usable until its headless session emits a provider session event; authentication and provider startup failures remain durable blockers.

## Verification and evidence

Every run records the work order, requirement, harness session/run, normalized observations, Git baseline/state, validation results, artifacts, findings, policy decisions and checkpoint. Evidence stores its work-order scope, relevant commit/input digest, supporting observation IDs and artifact IDs.

Evidence statuses are CLAIMED, PARTIALLY_VERIFIED, VERIFIED, REJECTED, and STALE. A changed commit or input digest invalidates ValidFor; explicit invalidation persists STALE.

Completion requires a completed harness result, all planned deterministic checks passing, corroborating evidence, no high/critical supervisor findings, and an allowed completion policy decision. A harness claim alone cannot complete a run.

If project discovery finds no executable validation capability, Keystone stops short of COMPLETE rather than treating a process exit or claim as project correctness evidence.

## Supervision and learning

The deterministic supervisor checks unsupported completion, failed validation, requirement/architecture/scope drift, repeated actions, excessive tool/context activity, stale assumptions and premature completion. Findings retain severity, confidence, explanation, recommendation and provenance.

Findings may create evidence-backed learning candidates. Learning moves explicitly through OBSERVED, CANDIDATE, EVALUATED, ACTIVE, REJECTED, and SUPERSEDED. Active project learning is added as a provenance-bearing context reference on later runs. Activation increments the version and every record has a rollback/supersession path. Learning cannot weaken policy.

## Pause, approval, recovery and replay

keystone pause writes a control marker and durable pause state. A live local run checks the marker while observing and interrupts the process safely. keystone continue removes the marker, reconstructs active states through permitted recovery transitions, and resumes the same work order/run. A blocked run first returns ASK; keystone approve CONTINUE records approval provenance before dispatch can resume.

keystone replay RUN-ID validates journal ordering and reconstructs the machine, evidence IDs, decisions, completion claims and observation gaps. Duplicate idempotent events are ignored only when their identity and content agree; conflicting reuse and malformed/truncated journal entries fail closed.

## Policy and delivery

Validation executes argv vectors without a shell. Workspace-escaping paths and destructive commands are blocked or require explicit approval. Force push, destructive reset/clean, destructive data operations, credential changes and protected production deployment are not silently authorized.

The delivery package persists release candidates, deployment plans/blocks and incident-derived high-risk work orders. Release evidence must be verified. Production deployment remains blocked until an explicit policy implementation authorizes it.

## CLI

    keystone init
    keystone status
    keystone ask "request"
    keystone run "request"
    keystone continue
    keystone pause
	keystone approve [CONTINUE|APPROVE]
    keystone stop
    keystone validate
    keystone review
    keystone replay RUN-ID
    keystone doctor
    keystone version

## Troubleshooting

- Keystone is not initialized: run keystone init in the project root.
- no harness configured: install/configure `codex`, `agy`, or `.keystone/harness.json`; the run remains durably blocked until resumed.
- completion was not verified: inspect keystone review, validation artifacts, and keystone replay RUN-ID.
- state snapshot corruption: run keystone continue; recovery succeeds only if the complete event journal is valid.
- malformed journal or non-increasing sequence: preserve the original .keystone/events.jsonl, repair it using an audited backup/recovery procedure, and rerun keystone doctor. Keystone will not silently skip damaged history.
- browser, CI, or operational checks unavailable: configure a compatible local command/harness and keep the resulting observation/evidence in the journal. Keystone does not fabricate remote service observations or bypass production policy.

## Limitations

The V2 core is local-first. Remote CI APIs, vendor-specific browser control, and production deployment execution require an explicitly configured adapter/command and credentials outside Keystone. The core records their observations and enforces policy; it does not invent a successful remote result or grant production authority.
