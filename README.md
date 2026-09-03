# Keystone

> Persistent engineering intelligence and control layer for AI development harnesses.

Keystone is an open-source, technology-agnostic control and supervision plane that sits between human intent and existing AI coding harnesses (such as **Antigravity** and **Codex**).

Keystone does **not** replace your external harness. The harness remains the primary worker executing code and tools; Keystone acts as the persistent driver, context compiler, verifier, and continuity supervisor. It maintains durable project knowledge across turns, diagnoses environment blockers, prevents no-progress loops, enforces safety policies, and verifies completion through deterministic evidence.

---

## Quick Start

### Installation

Install the latest binary directly via Go:

```bash
GOPROXY=direct go install github.com/anonyxhappie/keystone/cmd/keystone@latest
```

Or build from source:

```bash
git clone https://github.com/anonyxhappie/keystone.git
cd keystone
go install ./cmd/keystone
```

Ensure `$GOPATH/bin` (typically `~/go/bin`) is in your `$PATH`. Verify the installation:

```bash
keystone doctor
```

### Launching the Interactive Terminal Shell

Navigate to any local codebase and start the interactive shell:

```bash
cd ~/Desktop/code/your-project
keystone
```

You can also specify a harness explicitly on launch:

```bash
keystone --harness antigravity
# or
keystone --harness codex
```

---

## Interactive Shell Experience

Keystone features a persistent terminal shell (similar to Claude Code and Antigravity) designed for long-running, autonomous engineering workflows:

- **Auto-Restored Context & Preferences**: Keystone remembers your last selected harness, project workspace, and conversation session across launches in `~/.keystone/preferences.json`.
- **Live Conversation Streaming**: Streams real-time assistant responses, tool executions, and results directly to the terminal:
  ```text
  [16:30] ↳ ⚡ Tool: run_command(docker compose up -d postgres)
  [16:31] ↳ ✔ Result: container postgres started
  [16:32] ↳ 🤖 Antigravity: PostgreSQL is now reachable on localhost:5432. Running tests...
  ```
- **Supervisor Transparency**: Whenever Keystone provides guidance or recovery instructions, it states the action and reason clearly in the conversation stream:
  ```text
  [16:33] ↳ 🛡️ Keystone direction: Diagnose and recover PostgreSQL container
          Reason: Deterministic validation vitest failed (port 5432 connection refused)
  ```
- **Conversation History**: When resuming an existing session or switching workspaces, Keystone automatically renders recent conversation turns.

### Interactive Slash Commands

Type `/` in the prompt for interactive dropdown suggestions, or run commands directly:

| Command | Description |
| :--- | :--- |
| `/sessions` | Interactive picker for recent conversations across Keystone and installed harnesses |
| `/resume [id\|#]` | Resume an existing conversation by index number or conversation ID |
| `/new` | Start a brand-new conversation session (resets session ID) |
| `/harness [name]` | View or switch the active harness (`antigravity`, `codex`, `auto`) |
| `/projects` | Interactive local project and workspace switcher |
| `/project [path\|#]` | Switch active workspace directory |
| `/status` | Inspect current project state, active run, and checkpoint |
| `/verify` | Run deterministic project checks (tests, linters) on demand |
| `/review` | Inspect supervisor findings, drift detection, and advice |
| `/replay [run-id]` | Replay execution events from a previous run |
| `/doctor` | Check harness health, environment paths, and Git status |
| `/clear` | Clear the terminal screen |
| `/exit` | Exit the interactive session |

### Natural Language Shell Aliases

You do not have to use slash prefixes for common terminal queries. Keystone recognizes natural-language shell queries and executes them immediately without spawning an unnecessary WorkOrder:

- `list sessions`, `show sessions`, `sessions`, `history` → `/sessions`
- `list projects`, `show projects`, `projects`, `workspaces` → `/projects`
- `doctor`, `check health`, `health`, `system check` → `/doctor`
- `verify`, `run tests`, `check tests`, `validate` → `/verify`
- `status`, `show status`, `current status` → `/status`
- `review`, `findings` → `/review`
- `clear`, `cls` → `/clear`
- `help`, `commands`, `what can you do` → `/help`

---

## How Keystone Works

### 1. The Autonomous Multi-Turn Control Loop
Each turn follows a strict, durable lifecycle:
```text
ContextCompiled → PromptGenerated → PromptDispatched → HarnessTurnStarted
  → HarnessObserved → HarnessTurnCompleted → SupervisorEvaluation → NextAction
```
Every prompt is durably saved in `.keystone/prompts/{promptID}.json` with explicit reason, strategy, hypothesis, and expected info gain.

### 2. Environment Blocker Recovery ("Let the Harness Fix It")
When deterministic validation fails due to local infrastructure (e.g., PostgreSQL `5432`, Redis `6379`, MySQL `3306`, MongoDB `27017`, Docker daemon), Keystone:
1. Classifies the failure as an `ENVIRONMENT_BLOCKER`;
2. Maintains harness session continuity (`HARNESS_SESSION_RESUMED`);
3. Generates targeted recovery guidance directing the harness to inspect project configuration (e.g., `docker compose`), start the service, verify connectivity, and rerun validation.

### 3. Loop & Low-Information Retry Detection
If consecutive attempts repeat identical failures without repository mutations or progress:
- In `auto` mode: switches to an alternative available harness.
- In `explicit` mode: flags a `supervisor.Loop` finding and halts with an `ASK` approval request to prevent burning token budgets.

### 4. Non-Destructive Git Safety & Read-Only Enforcement
- Captures an independent Git baseline snapshot before harness execution.
- Any pre-existing dirty files in your working directory are preserved.
- When executing read-only requests, Keystone detects unexpected file modifications, safely reverts them to baseline, and denies completion claims.

### 5. Evidence Outranks Claims
Completion is never granted simply because an external harness claims "done". Completion requires deterministic verification (tests pass, builds succeed, git diffs match requirements).

---

## Supported AI Harnesses

Keystone auto-detects and connects to existing AI coding harnesses:

### Google Antigravity (`agy`)
- Automatically discovers installed `agy` CLI (`~/.local/bin/agy` or `PATH`).
- Bridges session trajectories between Antigravity IDE (`~/.gemini/antigravity/conversations/`) and Antigravity CLI (`~/.gemini/antigravity-cli/conversations/`) so IDE sessions can be resumed in Keystone without trajectory errors.
- Runs non-interactive print mode with `--output-format stream-json` and `--dangerously-skip-permissions` for headless supervisory control.

### OpenAI Codex (`codex`)
- Automatically discovers installed `codex` CLI (`~/.local/bin/codex` or `PATH`).
- Normalizes Codex JSONL turn, tool, and exec events into canonical Keystone observations.
- Uses `codex exec resume <thread-id>` for multi-turn session continuity.

### Custom Local Process Harness
Configure any custom CLI harness in `.keystone/harness.json`:
```json
{
  "name": "custom-harness",
  "command": "sh",
  "args": ["-c", "your-agent-command"],
  "timeoutSeconds": 300
}
```

---

## CLI Command Reference

| Command | Usage | Description |
| :--- | :--- | :--- |
| `keystone` | `keystone` | Start the persistent interactive terminal shell |
| `keystone --harness <name>` | `keystone --harness antigravity` | Start shell with explicit harness |
| `keystone run <request>` | `keystone run "fix failing vitest tests"` | Run a single supervised request non-interactively |
| `keystone init` | `keystone init` | Initialize portable `.keystone/` state boundary |
| `keystone status` | `keystone status` | Inspect durable project state and active checkpoint |
| `keystone validate` | `keystone validate` | Run discovered project validation checks directly |
| `keystone review` | `keystone review` | Inspect supervisor findings and recommendations |
| `keystone replay <run-id>` | `keystone replay RUN-12345` | Replay events of a previous run |
| `keystone doctor` | `keystone doctor` | Check environment, Git state, and discovered harnesses |
| `keystone version` | `keystone version` | Print version |

---

## Durable Project Model (`.keystone/`)

All Keystone intelligence and state is stored locally within the project:

```text
.keystone/
├── project.json         # Project identity & discovered capabilities
├── state.json           # Materialized runtime state snapshot
├── events.jsonl         # Append-only canonical event journal
├── prompts/             # Durably archived harness prompts per turn
├── checkpoints/         # Machine-readable continuation checkpoints
├── harness-sessions/    # External harness session identities
├── harness-runs/        # Execution run telemetry
├── evidence/            # Cryptographic diff digests & tool evidence
├── findings/            # Supervisor review findings
└── work/                # WorkOrders and WorkPackets
```

---

## Architecture & Contributing

For full architectural invariants, boundaries, and acceptance criteria, see:
- [docs/KEYSTONE_ARCHITECTURE_V1.md](docs/KEYSTONE_ARCHITECTURE_V1.md) — Reference architecture for Keystone V1.
- [AGENTS.md](AGENTS.md) — Keystone Agent Contract & design invariants.

All tests are deterministic and runnable offline:
```bash
go test ./...
```
