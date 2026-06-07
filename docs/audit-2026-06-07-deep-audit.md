# Deep audit — `zero-zenline` (PR #101 branch `zenline-runtime-work`)

**Date:** 2026-06-07
**Method:** 12 parallel finders (10 subsystems + cross-cutting *wiring* and *concurrency*) read ~35K LOC of non-test Go in full; every candidate finding was then handed to an independent **adversarial verifier** that re-read the actual code to confirm or refute it. **49 raised → 43 confirmed, 6 refuted.** The two most surprising HIGH findings (H7/H8) were additionally spot-checked against source + git history by hand.

**Confirmed by severity:** 🔴 9 high · 🟠 ~11 medium · 🟡 ~19 low (43 incl. 2 cross-finder duplicates).

> Recurring root theme: **the argument-tolerance / alias layer outran its consumers.** Several tools now accept aliased keys (`file`/`file_path`, `diff`, `cmd`/`script`/`shell`, …), but downstream consumers (checkpoint capture, the destructive-command gate) still read only the canonical key — see H1 and H4.

---

## 🔴 HIGH (9)

### H1 · Checkpoints silently skipped for aliased writes
`internal/tools/mutation_targets.go:10-28`
`MutationTargets` reads only the canonical `path`/`patch` keys via `stringArg`, but `write_file`/`edit_file` resolve `path` via `aliasedStringArg([]string{"path","file","file_path","filename"})` and `apply_patch` via `aliasedStringArg([]string{"patch","diff"})`. A model that writes via `{"file": …}` gets **no checkpoint captured → `/rewind` cannot undo that write.**
**Fix:** resolve path/patch in `MutationTargets` via the same alias key lists the tools use.

### H2 · `rewind` restores wrong bytes when the closest checkpoint is "Skipped"
`internal/sessions/rewind.go:58-82`
Checkpoints apply newest→oldest with no per-path short-circuit. If the closest-to-target state for a path is `Skipped` (oversize/unreadable) while a newer checkpoint carries a blob, the **older blob is written and the run is reported as restored** — the file is left in the wrong state.
**Fix:** handle only the closest-to-target (oldest) entry per path (`if restored[f.Path] { continue }`).

### H3 · `rewind` path guard is lexical-only — in-workspace symlink escapes
`internal/sessions/rewind.go:89-100`
`resolveWithinWorkspace` joins+cleans and rejects `..`, but never resolves symlinks (unlike `tools.resolveWorkspaceTargetPath`, which calls `filepath.EvalSymlinks`). A checkpoint path through an in-workspace symlink pointing outside the workspace passes the guard and **writes outside the workspace** on restore.
**Fix:** `EvalSymlinks` the deepest existing ancestor, re-join missing segments, verify it stays under `EvalSymlinks(root)`.

### H4 · Destructive-command gate ignores `cmd`/`script`/`shell` aliases
`internal/sandbox/risk.go:65`
`Classify` extracts the shell command solely via `argString(request.Args, "command")`, but the `bash` tool reads the command from any of `command`/`cmd`/`script`/`shell` via `aliasedStringArg` (`bash.go:55`). **A model invoking bash via an alias bypasses the `rm -rf` / curl-pipe destructive deny gate entirely.**
**Fix:** in `Classify`, resolve the command across the same alias set (`firstNonEmpty` over `command`/`cmd`/`script`/`shell`).

### H5 · Cancelling a run discards its session events + orphans checkpoint blobs
`internal/tui/model.go:816-832, 384-387, 1016-1032`
The agent goroutine accumulates all session events (incl. `EventSessionCheckpoint` payloads captured before each mutating tool) into a goroutine-local slice. On cancel, `activeRunID` is zeroed and those events are dropped, so checkpoint **blobs are written to disk but never referenced → `/rewind` broken for cancelled runs, plus orphaned blobs.**
**Fix:** flush in-flight `sessionEvents` before clearing `activeRunID` (keep handling the recently-cancelled runID, or append checkpoint/tool events as they occur).

### H6 · MCP schema conversion drops nested array/object shape
`internal/mcp/schema.go:56-94`
`propertyToMCP`/`propertyFromMCP` copy only `Type/Description/Enum/Default/Minimum/Maximum` — they drop `Items` (array element shape), `Properties`, and `Required` in both directions. **Tools with array/object params (`ask_user`, `update_plan`) lose their nested schema over MCP**, so peers see malformed parameters.
**Fix:** recursively map `Items`/`Properties`/`Required` to/from the MCP `items`/`properties`/`required` keys.

### H7 · Interactive permission-mode cycling is advertised but never wired
`internal/tui/model.go` (no `KeyShiftTab` case); labels at `view.go:72,124`
The status bar says "shift+tab to cycle" and renders Auto/Ask/Unsafe labels, but **no key handler ever mutates `m.permissionMode`** — it is set once at construction and frozen.
**Fix:** add a `case tea.KeyShiftTab:` that cycles Auto→Ask→Unsafe.

### H8 · TUI is hardcoded to Auto, which hides all write/shell tools
`internal/cli/app.go:306` + `internal/agent/loop.go:888-896` (`ToolAdvertised`) → `toolDefinitions`/`ToolVisible`
`runTUI` hardcodes `PermissionModeAuto`. `ToolAdvertised` in Auto returns true **only for `PermissionAllow` tools**, so `write_file`/`edit_file`/`bash`/`apply_patch` (all `promptSafety` → `PermissionPrompt`) are **never advertised to the model**. With H7 there is no way to switch out of Auto. Statically, the interactive TUI cannot write files or run shell. Git shows Auto has been set since PR #54.

> ⚠️ **VERIFY LIVE — contradiction.** This conflicts with earlier observed behavior where `write_file` *was* called in the live TUI. I confirmed the code path and git history but could not reconcile it without a TTY (no tmux in the audit env). **Deciding test:** restart the TUI and ask it to write a file — if `write_file` is never attempted, H7/H8 are real and serious; if it writes, this reading is too narrow.
**Fix (if real):** wire shift+tab (H7); in Ask mode prompt tools become advertised and the permission flow fires.

### H9 · MCP stdio client blocks forever on a hung server
`internal/mcp/client.go:181-224` (+ `protocol.go:42-78`)
`request` checks `ctx` once, then takes `client.mu` and does a blocking `bufio.ReadString`/`io.ReadFull` on the server's stdout with **no context awareness**. A stdio MCP server that accepts a request but never replies hangs the agent indefinitely, and the held `client.mu` serializes all other MCP calls behind it.
**Fix:** run `reader.read()` in a goroutine feeding a buffered channel; `select` on `ctx.Done()` (the pattern the OpenAI SSE watchdog already uses).

---

## 🟠 MEDIUM (~11)

- **M1 · ask_user/task results bypass secret-scrubbing** — `loop.go:526-531, 453-459`. Both are intercepted *before* the registry boundary (the documented single scrub point), so a secret in a user answer or sub-agent `FinalAnswer` enters the transcript unredacted with `Redacted=false`. Same trust boundary as the rest of the conversation (hence medium, not high), but it breaks a documented invariant. *Fix:* run both outputs through `redaction.RedactString` and set `Redacted`.
- **M2 · Reactive compaction re-streams text to the user** — `loop.go:115-142`. On a mid-stream context-limit error the retry reuses `OnText`, so partial text already shown is streamed a second time (garbled output). The summary path already omits `OnText` for this reason; the retry path doesn't. *Fix:* drop `OnText`/`OnUsage` on the retry collect.
- **M3 · Data race on `updatePlanTool.currentPlan`** — `update_plan.go:60-75`. Agent goroutine writes it in `Run` while `/plan` reads it via `CurrentPlan()` on the UI goroutine. *Fix:* guard with a mutex.
- **M4 · Anthropic & Gemini have no idle-timeout watchdog** — `anthropic/provider.go:130`, `gemini/provider.go:116`. Only OpenAI got the stall guard; a stalled-but-open Anthropic/Gemini upstream hangs the agent forever. *Fix:* apply the same reader-goroutine + idle-timer watchdog.
- **M5 · No cross-process session lock** — `sessions/store.go:425-437`. The lock is in-memory per `Store`; a CLI `zero sessions rewind` racing the TUI on the same session is not serialized → possible metadata corruption. *Fix:* per-session flock.
- **M6 · `rm -rf` with quoted/braced `$HOME` bypasses the destructive pattern** — `sandbox/risk.go:13`. `rm -rf "$HOME"` and `${HOME}` evade the regex.
- **M7 · Interactive-command detector skip-list incomplete** — `sandbox/safe_command.go:186`. `nice vim`, `timeout … vim`, `sh -c '…'`, `sudo <flag> …` defeat detection. *Fix:* extend the wrapper skip-list, skip leading option tokens, recurse into `sh -c`.
- **M8 · Glamour renderer cache `mdCache` is unbounded** — `zenline/markdown.go:33-39`. The *output* cache is bounded but the *renderer* cache (the expensive goldmark+chroma objects) is not — a resize drag grows it per width. *Fix:* bound+reset `mdCache` too.
- **M9 · Markdown lines not clipped to frame width** — `zenline/render.go:645-663`. Glamour doesn't hard-break unbreakable tokens, so a long URL/code line exceeds the wrap width and breaks the fixed-height layout. *Fix:* clip each line to the frame budget.
- **M10 · `PermLayout` hit-test geometry diverges from the rendered modal** — `zenline/render.go:348-369`. No width/height clamp or overlay-height subtraction, unlike `RenderChat` — mouse clicks can miss the buttons after resize.
- **M11 · Zenline skin shows a misleading "working…" spinner during ask_user** — `tui/zenline_view.go:88-107`. `ChatData` has no ask_user field, so the zenline skin shows no question/options while blocked (the default skin does). *Fix:* add an `AskUser` field to `ChatData` and render it; suppress Working/Thinking when a prompt is pending.

---

## 🟡 LOW (~19) — grouped

**Sandbox heuristics:** piped-installer match requires a literal `"| sh"` space — `|sh`, `|zsh`, `curl|bash` slip by (`risk.go:74`); `chmod -Rf 777`, octal `0777`, `chmod 777 -R`, `rm -rf -- /` evade `matchesDestructive` (`risk.go:13`).

**Streaming / runtime:** collected tool calls can emit out of model order when one ends mid-stream and another flushes at termination (`helpers.go:63-66`); duplicate/empty `ToolCallID`s collapse into one call, silently dropping calls (`helpers.go:36`); `DroppedToolCalls` ignored when a turn also has a valid call (`loop.go:159`); Gemini `emitDone` value receiver makes `state.done=true` a dead store (`gemini/provider.go:200`); Anthropic drops a nameless `tool_use` block without emitting `StreamEventToolCallDropped` (`anthropic/provider.go:165`).

**Sessions:** `readBlob` doesn't re-verify sha256 on read → silent corruption restores bad bytes (`checkpoint.go:147`); `RestoreToSequence` double-counts `FilesRestored/Deleted` for multi-checkpoint paths (`rewind.go:63`); `writeMetadata` uses a fixed `.tmp` path → collides across processes (`store.go:468`).

**Providers:** OpenAI duplicates `providerio` error/redact helpers and has drifted — 503/529 not classified as rate-limit (`openai/provider.go:305`). *Fix:* delete the inline copies, call `providerio.ClassifiedError`/`Redact`.

**CLI wiring:** `--mode`-supplied tool filters skip `validateExecToolFilters` and `--list-tools` because `applyExecMode` runs too late (`exec.go:111-139`); a mode-selected model's deprecation notice is discarded (`exec.go:455`); `zenline` command is dispatched but omitted from `zero --help` (`app.go:371`).

**Plugins / skills:** plugin manifest `tools/hooks/prompts/skills` are fully parsed but **never wired into the runtime** — discovery-only (`plugins.go:326`); plugin-declared skill directories are not merged into the `skill` tool's lookup path (`skills.go:56`).

**Dead code / edges:** `executeTask` no-provider fallback is unreachable dead code (`loop.go:393`); unused `stringSliceArg` (`ask_user.go:184`); optional path args reject an explicit `""` instead of falling back to default (`grep.go:55`); reactive-compaction sets `reactiveAttempted` even on a no-op, disabling later recovery (`compaction.go:281`); MCP `Serve`/connect-`initialize` ignore ctx during blocking reads (`mcp/server.go:41`, `client.go:91`).

---

## ✅ Refuted (6) — checked, not real
1. Repeated-failure stop "orphans `tool_calls`" — mechanics accurate but the impact is unreachable in this codebase.
2. task-scrubbing as a separate high — same issue as M1, same trust boundary (medium).
3. `pruneOrphanBlobs` unlocked delete — the only window is the TUI batched checkpoint, which is guarded against concurrent rewind.
4. `resultSummary` bodyMax "ignored" for diffs — intended design, not a bug.
5. `--profile` "dead flag" — actually used and test-backed (`exec_parse.go`).
6. `connectStdio` pipe leak on Start failure — the stdlib closes descriptors on `Start` failure.

---

## Recommended fix order
1. **H4** (destructive-gate alias bypass) + **H1** (rewind data loss) — shared one-line root cause (alias-aware consumers), highest risk/effort ratio.
2. **H7/H8** — but **confirm live first** (screenshot contradiction); if real, it's a headline functional gap (TUI can't write).
3. **H3** (symlink escape), **H9/H6** (MCP hang + nested schema), **H2/H5** (rewind correctness + cancelled-run blobs).
4. Medium batch: M1 (scrubbing), M2 (double-stream), M3 (plan race), M4 (provider watchdogs).

---
*Generated by an automated multi-agent audit (12 finders → adversarial verifiers). Findings are against the `zenline-runtime-work` branch (PR #101). Each item cites file:line; verify before fixing.*
