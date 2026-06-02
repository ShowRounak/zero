# Zero — Product Requirements Document

| | |
|---|---|
| **Status** | DRAFT v0.2 (expanded) |
| **Last updated** | 2026-06-02 |
| **Owners** | Zero maintainers |
| **Reviewers** | _tbd_ |
| **Supersedes** | openclaude |


---

## Table of Contents
1. Overview & Vision
2. Problem Statement
3. Goals, Non-Goals & Success Metrics
4. Personas & User Journeys
5. Product Principles
6. System Architecture
7. Functional Requirements (F1–F14)
8. Data Models & Contracts
9. Permission & Autonomy Model
10. Non-Functional Requirements
11. Milestones & Exit Criteria
12. Risks & Mitigations
13. Open Decisions
14. Out of Scope
15. Appendix A — Drac capability map
16. Appendix B — Glossary

---

## 1. Overview & Vision

**Zero is a terminal-first, multi-provider AI coding agent you fully own.** One TypeScript codebase powers an interactive TUI, a scriptable headless mode, and a future editor extension, driving an agentic tool loop against whatever model the user chooses.

**Vision statement:** *The most trustworthy place to run a coding agent — portable across providers, honest about cost and permissions, and equally usable by a human at a prompt or a program over a pipe.*

Zero competes with vendor-locked agents (Claude Code, Factory Droid, Codex CLI) on one axis they structurally can't: **provider neutrality + local ownership**. Where those tools optimize for their own model and billing, Zero treats the model as a swappable, per-task choice.

## 2. Problem Statement

Today's terminal coding agents force three compromises:
1. **Vendor lock-in.** The best agents are welded to one provider's models and billing. Switching costs are high; portability is an afterthought.
2. **Opaque autonomy.** Agents run shell commands and edit files with permission models that are either all-or-nothing or hidden.
3. **Throwaway context.** Sessions are ephemeral or trapped in a vendor cloud; you can't easily resume, fork, search, or own your history.

Zero's predecessor *openclaude* proved the multi-provider CLI + extension shape but was JS-fragmented and lacked a coherent core. Zero is the from-scratch TypeScript successor that fixes the architecture and closes these three gaps.

## 3. Goals, Non-Goals & Success Metrics

### 3.1 Goals
- **G1 — Provider portability.** Any OpenAI-compatible endpoint works with only `{baseURL, apiKey, model}`; first-class Anthropic + Gemini; model selectable per task and per session. **[PA: `src/model_registry.js`]**
- **G2 — One TypeScript stack** across core / tools / headless / extension.
- **G3 — Terminal-native UX.** Native copy-paste and scrollback preserved; smooth streaming.
- **G4 — Safe by default.** No system mutation without an explicit, tunable permission decision.
- **G5 — Scriptable parity.** Everything the TUI can do is reachable headlessly over a stable I/O contract. **[PA: `src/cli.js` `exec --input-format stream-json`]**
- **G6 — Durable, ownable sessions.** Persist, resume, fork, search — as plain local files.

### 3.2 Non-Goals (v1) — see §14 for the full list
Cloud sync, relay/remote execution, a plugin **marketplace**, multi-agent "missions" as a headline surface, vendor telemetry pipelines, auto-generated repo wikis.

### 3.3 Success Metrics
| Metric | Target (by v1.0) |
|---|---|
| Providers supported (families) | ≥ 3 first-class (OpenAI-compatible, Anthropic, Gemini) |
| Time-to-first-token rendered | < 500 ms after provider's first byte |
| Switch model mid-session | ≤ 1 command, no restart |
| Mutating action without a permission decision | **0** (hard invariant) |
| Session resume success rate | 100% of well-formed sessions reopen |
| Headless `exec` usable from a script with stable JSON | Yes, documented schema |
| Cold-start to interactive prompt | < 300 ms on a warm cache |
| Crash-free TUI under resize / wide-char / non-TTY | No panics; clean fallback |

## 4. Personas & User Journeys

**P1 — Sasha, solo dev (primary).** Lives in the terminal. Wants "review this file," "fix the failing test," "scaffold a route." Values speed, copy-paste, and not babysitting the agent.
> *Journey:* `zero` → types a request → watches tool rows stream → approves a `bash` test run → reads a unified diff → accepts → continues. Closes the laptop; tomorrow `zero --resume` picks up the thread.

**P2 — CI pipeline (automation).** Runs `zero exec` non-interactively, parses JSON output, fails the build on a non-zero result. Never sees a TUI.

**P3 — Priya, editor user.** Uses the VS Code extension, which drives the same core over the stream protocol. Sees the same tool/diff/permission flow inside the editor.

**P4 — Marco, power user.** Brings his own keys for three providers, runs a cheap model for grunt work and a strong "spec" model for planning, tunes autonomy to `high` in a sandbox and `low` on his main repo.

## 5. Product Principles
1. **Append, don't repaint.** The transcript is scrollback-native; we never seize the whole screen if we can avoid it.
2. **Portable core, swappable edges.** `Provider`, `Tool`, `Renderer`, `SessionStore`, `PermissionPolicy` are interfaces, never assumptions.
3. **Least privilege, surfaced.** Reads are free; mutations are gated and the gate is visible.
4. **Same brain, many mouths.** TUI / headless / extension are thin shells over one agent core.
5. **Boring, inspectable state.** Sessions are plain files you can read, diff, and delete.
6. **Model is data, not code.** Capabilities, cost, and provider routing live in a registry, never hard-coded — because the model/billing landscape is volatile.

## 6. System Architecture

```
                       ┌──────────────────── surfaces ────────────────────┐
   TUI (Ink, default) ─┐   headless `exec` ─┐   VS Code ext ─┐   (later) daemon ─┐
                       │                    │                │                  │
                       └───────────────► Agent Core ◄────────┴──────────────────┘
                                              │  runAgent(prompt, session, policy, callbacks)
        ┌──────────────┬──────────────┬──────┴───────┬───────────────┬───────────────┐
   Provider +      Tool Registry   Session Store   Permission /     MCP Client      Config /
   Model Registry  (+ schemas)     (persist/fork)  Autonomy gate                    Settings
        │
   ┌────┴─────────────────────────┐
   model → {apiProviders[]} with   ← failover/rotation  [PA: src/llm/anthropic/OYA.js]
   rotation; capabilities + cost
```

**Layering rules**
- The **Agent Core** never imports a surface, a concrete provider, or a renderer — only interfaces.
- **Surfaces** (TUI/exec/extension) own I/O and human interaction; they translate Agent Core events into pixels or JSON.
- A single binary dispatches modes. **[PA: `src/cli.js` — `drac` (tui default) / `exec` / `daemon` / `mcp` / `plugin` / `update`]**

**Reference startup pipeline (for parity sequencing).** [PA: `src/agent.js`]: telemetry → resolve append-system-prompt (flag→file→env) → settings overlay → certs → auth/org → host identity → feature flags → settings + autonomy defaults + built-in subagents → diagnostics → sandbox → tool/subagent registration → run. Zero adopts a trimmed version: config load → provider/model resolve → session open → tool registry init → permission policy init → run.

---

## 7. Functional Requirements

Priority: **P0** = v1 must-have · **P1** = v1.x · **P2** = later. Each requirement is testable; P0 features include acceptance criteria.

### F1 — Agent Core & Tool Loop  (P0)
**Description.** The provider-agnostic loop: stream assistant text + tool calls, execute tools (parallel where safe), feed results back, bound turns, emit lifecycle events.

- **REQ-CORE-1 (P0):** Stream text and tool-call deltas; execute tool calls; loop until the model returns no tool calls or `maxTurns` is hit. *(Exists.)*
- **REQ-CORE-2 (P0):** Pluggable system-prompt assembly with an **append-system-prompt** hook (inline text or file path). **[PA: `src/agent.js` `appendSystemPrompt`/`appendSystemPromptFile`]**
- **REQ-CORE-3 (P0):** **Clean interrupt** — `Esc`/`Ctrl-C` aborts in-flight provider stream and tool execution, leaving session state consistent (no half-written tool result, no dangling assistant message).
- **REQ-CORE-4 (P0):** Emit a typed event stream (`text`, `tool_call`, `tool_result`, `usage`, `plan_update`, `error`, `turn_end`) consumed identically by all surfaces.
- **REQ-CORE-5 (P1):** Reasoning-effort passthrough per request (`off|low|medium|high`), gated by model capability. **[PA: `src/model_registry.js` `reasoningEffort.{supported,default}`]**
- **REQ-CORE-6 (P1):** Context-window management — warn/compact when approaching a model's `contextLimits`.

**Acceptance:** Given a scripted provider, a tool call is executed and its result appears in the next request; `Esc` during a long `bash` cancels it and the next user turn starts cleanly; events are emitted in order and a headless consumer can reconstruct the transcript from events alone.

### F2 — Tools  (P0 core, P1 expansion)
**Description.** The agent's hands. Define a tool surface matching a production agent.

Prior-art tool surface **[PA: grep across `src/` — `bash`, `apply_patch`, `task`, `grep`, `glob`, `edit_file`, `create_file`, `view_file`, `web_search`, `fetch_url`, `todo_write`]**:

| Tool | Purpose | Priority | Notes / Prior art |
|---|---|---|---|
| `read_file` | Read file | P0 | exists |
| `list_directory` | Explore tree | P0 | exists |
| `grep` | Content search (ripgrep) | P0 | exists |
| `glob` | Filename/path search | P0 | **[PA: `glob`]** — split from grep |
| `bash` | Run shell command | P0 | exists; **must be permission-gated** (F9) |
| `edit_file` | Exact-string edit | P0 | exists — **add uniqueness guard** (currently replaces first match silently) |
| `write_file` | Create/overwrite file | P0 | **[PA: `create_file`]** |
| `update_plan` | Plan/todo list | P0 | exists; **[PA: `todo_write`]** |
| `apply_patch` | Multi-hunk patch application | P1 | **[PA: `apply_patch` — Drac's primary mutator, 21 refs]** — robust multi-edit |
| `multi_edit` | Batched edits to one file | P1 | derived |
| `web_fetch` / `web_search` | Web access | P1 | **[PA: `fetch_url`/`web_search`]** — provider/permission gated |
| `task` | Spawn a scoped sub-agent | P2 | **[PA: `task` — 19 refs]** — see F11 |

- **REQ-TOOL-1 (P0):** Tools declare a Zod schema; the loop converts to JSON Schema per provider (exists via `z.toJSONSchema`).
- **REQ-TOOL-2 (P0):** Tool results are **structured**: `{ status: ok|error, output: string, truncated?: bool, meta?: {...} }`, so renderers can collapse/expand and headless can serialize. **[PA: Drac indexes `tool_use`/`tool_result` as first-class search kinds — `src/cli.js`]**
- **REQ-TOOL-3 (P0):** `edit_file` rejects the edit if `old_string` is non-unique (require enough context), returning an actionable error.
- **REQ-TOOL-4 (P1):** Per-launch tool enable/disable (`--enabled-tools`, `--disabled-tools`) and a `--list-tools` introspection command. **[PA: `src/cli.js` exec options]**
- **REQ-TOOL-5 (P1):** Every mutating tool routes through the Permission gate (F9) before side effects.

**Acceptance:** All P0 tools registered and schema-valid; a non-unique `edit_file` is refused with guidance; tool results round-trip through the headless JSON contract.

### F3 — Providers & Model Registry  (P0)
**Description.** Decouple *model* from *provider connection*, support per-task model selection, and route/fail over across the API providers that can serve a model.

- **REQ-PROV-1 (P0):** Keep the `Provider` interface; ship OpenAI-compatible provider (exists). Add **Anthropic** and **Gemini** providers behind the same interface.
- **REQ-PROV-2 (P0):** A **Model Registry** — a data map keyed by model id:
  ```ts
  interface ModelEntry {
    id: string;
    name: string; shortName?: string;
    provider: "anthropic" | "openai" | "google" | "xai" | string;
    apiProviders: string[];          // endpoints that can serve this model
    contextLimits: { input: number; output: number };
    reasoningEffort: { supported: ("off"|"low"|"medium"|"high")[]; default: string };
    capabilities: { tools: boolean; thinking?: boolean; pdf?: boolean; vision?: boolean };
    cost: { inputPerMTok?: number; outputPerMTok?: number; tokenMultiplier?: number };
    tier?: "standard" | "premium";
    matchPatterns?: RegExp[];         // fuzzy id resolution
  }
  ```
  **[PA: `src/model_registry.js` — exact field set incl. `apiProviders`, `reasoningEffort`, `contextLimits`, `cost.tokenMultiplier`, `matchPatterns`]**
- **REQ-PROV-3 (P0):** Model selectable per task (`-m/--model`) and per session; the registry gates capabilities (don't offer thinking on a model lacking it). The active model is shown in the TUI footer.
- **REQ-PROV-4 (P1):** **Provider rotation/failover** — when a model lists multiple `apiProviders`, pick an available one and rotate on failure, with an optional locked provider. **[PA: `src/llm/anthropic/OYA.js` `R_$({model,currentProvider,lockedProvider,onRotate,rotateIfValid})`]**
- **REQ-PROV-5 (P1):** Cost/usage accounting — combine `onUsage` token events with registry cost to show running cost in the footer and session summary.
- **REQ-PROV-6 (P1):** "Spec model" — a distinct (often stronger) model for planning vs. execution. **[PA: `src/cli.js` `--spec-model`/`--use-spec`]**

**Acceptance:** A session can switch from an OpenAI-compatible model to an Anthropic model without restart; an unknown model id resolves via `matchPatterns` or fails with a clear message; capability gating prevents invalid requests.

### F4 — Terminal UI  (P0)
**Description.** The default surface. Append-style, scrollback-native, smooth streaming.

Layout (top→bottom): **header** (cwd · git branch · model) → **collapsed tool-call rows** with status dots → **unified diff** for edits → **assistant response** (markdown + syntax-highlighted code) → **bordered input** (`@` file, `/` command, `!` bash, `Esc` interrupt) → **status footer** (model · tokens · cost · mode).

- **REQ-TUI-1 (P0):** Static/dynamic render split — committed transcript via Ink `<Static>`; only the live tail re-renders, so streaming doesn't repaint history.
- **REQ-TUI-2 (P0):** Markdown rendering with syntax-highlighted fenced code (restore the Shiki path; do not regress to flat text).
- **REQ-TUI-3 (P0):** Tool-call rows are collapsible with smart one-line summaries (e.g. `read src/app.tsx`, `bash: cargo test`) and expand to args + full result.
- **REQ-TUI-4 (P0):** Input affordances: `/` command menu (filtered, keyboard-navigable), `@` file completion, `!` to run bash, `Esc` to interrupt.
- **REQ-TUI-5 (P0):** Correct under terminal **resize**, **wide-char/emoji width**, **ANSI-aware wrapping**; **non-TTY fallback** to line-based output.
- **REQ-TUI-6 (P1):** Splash / empty-session screen with the figlet ZERO wordmark (keep the real wordmark; no braille/ersatz glyphs).
- **DECISION GATE:** append/inline vs. alt-screen panes, and Ink vs. OpenTUI — see §13. This PRD assumes **append/inline on Ink + `<Static>`**.

**Acceptance:** Streaming a long answer never repaints earlier messages; resizing mid-stream doesn't corrupt layout; piping to a file yields clean line output; a code block renders highlighted.

### F5 — Sessions  (P0 — currently absent)
**Description.** First-class, durable, ownable conversation history.

- **REQ-SESS-1 (P0):** Persist each session as plain files under `~/.local/share/zero/sessions/<id>/` (or platform equivalent): `meta.json` + an append-only `messages.jsonl`. Survives restart/crash. **[PA: `src/core/session/_Nf.js` — `sessionId`, `messages[]` persisted]**
- **REQ-SESS-2 (P0):** `--resume [id]` (default: most recently modified) and `--fork <id>` (copy into a new id, preserving lineage). **[PA: `src/cli.js` `-r/--resume`, `--fork`]**
- **REQ-SESS-3 (P0):** Session metadata: `{ id, parentId?, title, cwd, model, createdAt, updatedAt, tokens, cost }`.
- **REQ-SESS-4 (P1):** `zero search <query>` across local sessions over message text, tool_use, and tool_result; typo-tolerant; `--kind` filter; `--json`. **[PA: `src/cli.js` `search`/`find` with `--kind message_text|tool_use|tool_result`]**
- **REQ-SESS-5 (P0 seam):** Reserve `parentSessionId` + `callingToolUseId` fields for future subagents (F11) so no migration is needed later. **[PA: `--calling-session-id`/`--calling-tool-use-id`]**

**Acceptance:** Kill the process mid-session; `zero --resume` reopens it with full history; `--fork` creates an independent copy that records its parent.

### F6 — Config & Settings  (P0 core, P1 overlay)
- **REQ-CFG-1 (P0):** Layered resolution — built-in defaults → user (`~/.config/zero/config.json`) → project (`./.zero/…`) → env → CLI flags (highest). Generalize today's provider-only config.
- **REQ-CFG-2 (P1):** A **runtime settings overlay** mergeable per-process (`--settings <path>`) for reproducible CI runs. **[PA: `src/agent.js` `FACTORY_RUNTIME_SETTINGS_PATH`]**
- **REQ-CFG-3 (P2):** Feature flags for staged rollout of risky features. **[PA: `src/agent.js` feature-flags warm fetch]**

### F7 — MCP (Model Context Protocol)  (P1)
- **REQ-MCP-1 (P1):** Connect to MCP servers of type `stdio | http | sse`; surface their tools into the registry. **[PA: `src/cli.js` `mcp add --type stdio|http|sse --env --header`]**
- **REQ-MCP-2 (P1):** `zero mcp add/remove/list`.
- **REQ-MCP-3 (P1):** **Per-server and per-tool permissions**, persistent, with `list/revoke/clear`. **[PA: `src/cli.js` `mcp permissions {list,revoke,clear}`]**

### F8 — Headless / Programmatic I/O  (P0 — enables P2 & P3)
- **REQ-HL-1 (P0):** `zero exec [prompt]` with `-f/--file`, `-o/--output-format <text|json>`, `--cwd`, `-m/--model`, `--session-id`, exit code reflecting success/failure. **[PA: `src/cli.js` `exec`]**
- **REQ-HL-2 (P1):** Streaming multi-turn over stdio: `--input-format stream-json` emitting the F1 event stream as line-delimited JSON; this is the contract the VS Code extension (F14) drives. **[PA: `src/cli.js` `--input-format stream-json|stream-jsonrpc`]**
- **REQ-HL-3 (P0):** In headless mode, **stdout carries only program output** (text or JSON); logs/diagnostics go to stderr. (Clean channel = scriptable.)

**Acceptance:** `zero exec -o json "list files" | jq .` parses; a non-zero agent outcome yields a non-zero exit; no stray log lines pollute stdout.

### F9 — Permissions & Autonomy  (P0 — currently absent; the safety gap) — see §9 for the full model
- **REQ-PERM-1 (P0):** A **PermissionPolicy** gate evaluated before every mutating side effect (`bash`, `write_file`, `edit_file`, `apply_patch`, network tools).
- **REQ-PERM-2 (P0):** **Autonomy levels** `low|medium|high` + `--skip-permissions-unsafe` escape hatch (clearly labeled). **[PA: `src/cli.js` `--auto`, `--skip-permissions-unsafe`; `src/agent.js` `initializeAutonomyFromGlobalDefaults`]**
- **REQ-PERM-3 (P0):** In TUI, a denied/needs-approval action prompts inline; in headless, default-deny unless granted by level or flag.
- **REQ-PERM-4 (P1):** Persistent grants ("always allow `git status`"), with list/revoke/clear. **[PA: MCP permission model]**

### F10 — Observability  (P1)
- **REQ-OBS-1 (P1):** Structured logging with stdout/stderr split; opt-in **local** metrics only — **no vendor telemetry by default** (deliberate divergence from Drac). **[PA: `src/agent.js` `cli_startup_*` counters]**

### F11 — Subagents / Task Orchestration  (P2; seam in P0)
- **REQ-SUB-1 (P0 seam):** Data model carries parent linkage + recursion depth (see F5). **[PA: `--depth`, `--calling-*`]**
- **REQ-SUB-2 (P2):** A `task` tool spawns a scoped sub-agent with its own budget/tools. **[PA: `task` tool]**
- **REQ-SUB-3 (P2):** Built-in specialized agents/commands (e.g. **code-review**, **security-review**), each a named prompt + tool scope. **[PA: deobfuscated sub-agent prompts — "senior staff software engineer and expert code reviewer", "senior security engineer … deep security audit"]**

### F12 — Plugins  (P2)
- **REQ-PLG-1 (P2):** Load user/project-scoped plugins (commands, tools, prompts). **Marketplace deferred.** **[PA: `src/cli.js` `plugin install --scope user|project`]**

### F13 — Distribution & Updates  (P0 build, P1 update)
- **REQ-DIST-1 (P0):** Ship a **single self-contained binary** via `bun build --compile`. *(PR #3's OpenTUI branch regressed this to a `dist/` bundle — preserve the standalone binary as a hard requirement.)*
- **REQ-DIST-2 (P1):** `zero update [--check]` self-update. **[PA: `src/cli.js` `update`]**

### F14 — VS Code Extension  (P1)
- **REQ-EXT-1 (P1):** An extension that drives the Agent Core via F8's stream protocol — **not** a reimplementation. Shows the same tool/diff/permission flow in-editor. Satisfies G2 (one stack) and carries openclaude's extension goal forward.

---

## 8. Data Models & Contracts

### 8.1 Headless event stream (F1/F8)
Line-delimited JSON; one object per line:
```jsonc
{ "type": "text",        "delta": "…" }
{ "type": "tool_call",   "id": "t1", "name": "bash", "args": { "command": "cargo test" } }
{ "type": "permission",  "id": "t1", "decision": "needs_approval", "reason": "mutating" }
{ "type": "tool_result", "id": "t1", "status": "ok", "output": "…", "truncated": false }
{ "type": "usage",       "promptTokens": 1234, "completionTokens": 567 }
{ "type": "plan_update", "plan": [ { "id":"1","content":"…","status":"in_progress" } ] }
{ "type": "turn_end",    "stopReason": "end_turn" }
{ "type": "error",       "message": "…", "hint": "…" }
```

### 8.2 Session on disk (F5)
```
~/.local/share/zero/sessions/<id>/
  meta.json        # { id, parentId?, title, cwd, model, createdAt, updatedAt, tokens, cost }
  messages.jsonl   # append-only: one message/tool event per line
```

### 8.3 Tool contract (F2)
```ts
interface Tool {
  name: string;
  description: string;
  parameters: ZodObject;          // → JSON Schema per provider
  mutating: boolean;              // routes through PermissionPolicy if true
  execute(args, ctx): Promise<{ status: "ok"|"error"; output: string; truncated?: boolean; meta?: object }>;
}
```

### 8.4 Config layering (F6)
`defaults → ~/.config/zero/config.json → ./.zero/config.json → env → CLI flags` (last wins).

---

## 9. Permission & Autonomy Model (F9 detail)

Tools are classed by side-effect. Autonomy level selects auto-approval per class. `--skip-permissions-unsafe` forces `allow` everywhere (loud warning).

| Tool class | `low` | `medium` | `high` |
|---|---|---|---|
| Read (`read_file`, `grep`, `glob`, `list_directory`) | allow | allow | allow |
| Plan (`update_plan`) | allow | allow | allow |
| Workspace write (`edit_file`, `write_file`, `apply_patch`) | **prompt** | allow (within cwd) | allow |
| Shell (`bash`) | **prompt** | **prompt** | allow |
| Network (`web_fetch`, `web_search`) | **prompt** | allow | allow |
| Out-of-workspace / destructive (rm, writes outside cwd) | **deny** | **prompt** | **prompt** |

- **TUI:** `prompt` → inline approve/deny/always-allow. **Headless:** `prompt` → deny unless a persistent grant or `--skip-permissions-unsafe` applies.
- Default level: `low`. **[PA: Drac `--auto low|medium|high` + `--skip-permissions-unsafe`]**

---

## 10. Non-Functional Requirements
- **Performance:** first token < 500 ms after provider's first byte; no full-transcript repaint per token; cold start < 300 ms warm.
- **TUI robustness:** correct under resize, wide-char/emoji, ANSI-bearing wrapped text; non-TTY fallback; clean interrupt.
- **Security:** no mutation without a permission decision (hard invariant); API keys never logged; headless stdout free of stray writes; respect a workspace boundary by default.
- **Portability:** Linux / macOS / Windows; any OpenAI-compatible endpoint via `{baseURL, apiKey, model}`.
- **Reliability:** a malformed provider chunk or tool error degrades to a clean message, never a crash.
- **Testability:** agent core unit-testable with a mock provider (pattern exists); tools tested against temp dirs; permission matrix covered by tests.

## 11. Milestones & Exit Criteria

| Milestone | Theme | Includes | Exit criteria |
|---|---|---|---|
| **M0** | Hardening | §13 TUI decision; `edit_file` uniqueness guard; clean interrupt; config layering (F1,F4,F6,F8 P0) | TUI decision locked; interrupt + guard shipped with tests |
| **M1** | Own your agent | Sessions persist/resume/fork (F5); permission gate + autonomy (F9); model registry + per-task model + Anthropic provider (F3) | A session survives restart; no mutation without a decision; model switch mid-session works |
| **M2** | Scriptable | Headless `exec` formats + stream protocol (F8); session search (F5); cost/usage accounting (F3) | `exec -o json` is documented + parseable; search returns hits; footer shows live cost |
| **M3** | Extensible / v1.0 | MCP client + permissions (F7); single-binary dist + self-update (F13); VS Code extension MVP (F14) | MCP tools usable + permissioned; `zero` ships as one binary; extension drives core over stream protocol |
| **Beyond** | — | Subagents/missions (F11), plugins (F12), feature flags (F6 P2) | — |

**Dependency note:** F9 (permissions) and F5 (sessions) are prerequisites for M2 and for the extension; do not defer them.

## 12. Risks & Mitigations
| # | Risk | Mitigation |
|---|---|---|
| R1 | TUI direction churn (Ink vs OpenTUI, append vs panes) stalls everything | Resolve §13 in M0 before downstream work |
| R2 | Permission model retrofitted late breaks flows | Land it in M1, not later; design tools `mutating`-aware from the start |
| R3 | Provider/billing volatility (subscription credits, model churn) | Keep model/provider as registry data; never hard-code; failover (REQ-PROV-4) |
| R4 | Scope creep toward Drac's full surface (relay, missions, wikis) | Enforce §14 Non-Goals; keep seams, defer surfaces |
| R5 | Single-binary distribution regresses (cf. PR #3) | REQ-DIST-1 is a hard requirement; verify in CI |
| R6 | Legal/IP contamination from reference code | Clean-room only; PRD cites prior art, never code; no Drac source in repo |

## 13. Open Decisions (resolve to proceed)
1. **Append/inline vs. alt-screen panes** — PRD assumes append/inline. *(Active; PR #3 pushes panes.)*
2. **Render stack** — Ink + `<Static>` vs. OpenTUI/Solid. PRD assumes Ink.
3. **Direct model APIs vs. wrapping vendor CLIs** — PRD assumes direct APIs (portability > subscription economics, given billing volatility).
4. **Session storage** — JSON/JSONL files (inspectable) vs. `bun:sqlite` (better search). Leaning files; revisit if F5 search needs an index.
5. **Patch strategy** — adopt `apply_patch` as the primary mutator (Drac/Codex style) vs. keep string-edit primary. Leaning: add `apply_patch` in P1, keep `edit_file` for small changes.

## 14. Out of Scope (v1)
Cloud sync & hosted backend · relay "computers" / remote SSH execution **[PA: `src/computer.js`]** · multi-agent "mission" mode as a surface **[PA: `--mission`]** · plugin marketplace **[PA: `plugin marketplace`]** · auto-generated repo wikis **[PA: `wiki-*`]** · git-authorship notes **[PA: `push-git-ai-notes`]** · vendor telemetry pipelines · a daemon server (kept as a future surface, not v1) **[PA: `daemon`]**.

## 15. Appendix A — Drac Capability Map (prior-art reference only)
- **Surfaces** (`src/cli.js`): `drac` (TUI default) · `exec` (headless; rich options incl. `--output-format`, `--input-format stream-json|stream-jsonrpc`, `--auto`, `--model`, `--spec-model`, `--enabled/disabled-tools`, `--worktree`, `--session-id`, `--fork`, `--mission`) · `daemon` (ws/ipc/unix; spawns child agents) · `mcp` (+permissions) · `plugin` (+marketplace) · `computer` (relay/SSH) · `search` · `update` · `wiki-*` · git-ai hooks.
- **Startup** (`src/agent.js`): telemetry → append-prompt → settings overlay → certs → auth/org → host identity → feature flags → settings+autonomy+built-in subagents → diagnostics → sandbox → tool/subagent registration → run.
- **Providers** (`src/model_registry.js`): anthropic, openai, google, xai (+ Bedrock via bundled AWS SDK); per-model `apiProviders[]` with rotation (`src/llm/anthropic/OYA.js`); reasoning-effort, context limits, cost multipliers, fuzzy `matchPatterns`.
- **Tools** (grep): `bash`, `apply_patch`, `task`, `grep`, `glob`, `edit_file`, `create_file`, `view_file`, `web_search`, `fetch_url`, `todo_write`.
- **Sessions** (`src/core/session/*`): `sessionId`, parent linkage, `messages[]`, persisted; resume/fork/search.
- **Built-in agents** (deobfuscated prompts): code-review, security-review, deep security-audit, statusline-setup.

## 16. Appendix B — Glossary
- **Agent Core** — surface-agnostic loop driving provider + tools.
- **Provider** — an API endpoint family (OpenAI-compatible, Anthropic, Gemini…).
- **Model Registry** — data describing each model's provider routing, capabilities, and cost.
- **Surface** — a way to use the core: TUI, headless `exec`, editor extension, (future) daemon.
- **Autonomy level** — `low|medium|high`, controlling auto-approval of tool classes.
- **Session** — a persisted, resumable, forkable conversation stored as local files.
- **PA** — *prior art*; an inline citation into the Drac reference tree (reference only, never copied).

