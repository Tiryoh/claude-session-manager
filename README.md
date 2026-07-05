# csm — claude-session-manager

Track every open Claude Code session and reopen it later, like restoring a
browser's tab group. Built for running many `claude` sessions in parallel
across terminals: if the machine loses power or a terminal is closed by
accident, `csm` still knows which sessions were open and where.

## How it works

Claude Code hooks (`SessionStart`, `UserPromptSubmit`, `SessionEnd`) call
`csm hook` on every session event. `csm` keeps a registry of active sessions
in `active.json` under its config directory (Go's `os.UserConfigDir()` +
`claude-session-manager`; override with `CSM_CONFIG_DIR`):

- Linux: `~/.config/claude-session-manager/active.json`
- macOS: `~/Library/Application Support/claude-session-manager/active.json`

If a session ends normally, its entry is removed. If the machine loses power, `SessionEnd`
never fires — the entry just sits there, which is exactly the recovery
target `csm` is for.

## Install

```bash
go install ./cmd/csm
csm install
```

`csm install` adds three hook entries to `~/.claude/settings.json`,
preserving every other setting untouched, and writes a `.bak` backup first.
Run it again any time — it's idempotent. `csm uninstall` removes them.

## Day to day

```bash
csm list                          # see active sessions, saved groups, bookmarks
csm save sprint-23                # snapshot the current active sessions as a group (interactive picker)
csm save --all --force autosave   # snapshot everything without prompting -- good for cron
csm bookmark robot-fw             # snapshot one session by picking it interactively
csm open                          # pick anything saved and resume it
csm open sprint-23                # open (picking a session from) a specific group
csm open robot-fw --fork          # resume a bookmark as a fork, leaving the original session intact
csm clean --dry-run               # see which active entries are no longer restorable
```

### Guarding against power loss

Run `csm save --all --force autosave` periodically, e.g. via cron:

```cron
*/15 * * * * PATH=$PATH:/usr/local/go/bin csm save --all --force autosave
```

If the machine loses power, `csm open autosave` lists every session that was
open at the last autosave, ready to resume one by one.

## Related projects

Several tools tackle the same "don't lose track of open Claude Code
sessions" problem, but each takes a different approach:

- [drewburchfield/claude-session-manager](https://github.com/drewburchfield/claude-session-manager)
  — bash utility with the same name; scans running processes and writes
  timestamped snapshots, then hands back copy-paste `claude --resume`
  commands per terminal tab. No hook integration, no daemon.
- [Supersynergy/claude-session-restore](https://github.com/Supersynergy/claude-session-restore)
  — restores every session after a crash/reboot by parsing the
  `~/.claude/projects/` transcript directory. Zero dependencies, but
  reconstructs state from transcripts rather than tracking it live.
- [hex/claude-sessions](https://github.com/hex/claude-sessions) (`cs`)
  — adds automatic git-backed autosave (shadow ref) and session locking to
  prevent the same session being opened twice, plus documentation/artifact
  tracking commands (`/summary`, `/checkpoint`, `/wrap`).
- [chronologos/cc-sessions](https://github.com/chronologos/cc-sessions)
  — fast search/resume across *all* projects and machines, not just the
  current one; no grouping/bookmarking concept.
- [tradchenko/claude-sessions](https://github.com/tradchenko/claude-sessions)
  — TUI picker across Claude Code and other CLI agents, with AI-generated
  summaries and a shared cross-session memory store.
- [ZENG3LD/claude-session-restore](https://github.com/ZENG3LD/claude-session-restore)
  — reconstructs prior context from transcripts and git history, tuned to
  handle very large (2GB+) session files.
- [xxGodLiuxx/claude-semantic-session-manager](https://github.com/xxGodLiuxx/claude-semantic-session-manager)
  — semantic search and summarization over past sessions, optimized for
  token usage rather than live tracking.

**What `csm` does differently:**

- **Hook-driven live registry, not a scan or transcript replay.**
  `SessionStart`/`UserPromptSubmit`/`SessionEnd` hooks update
  `active.json` in real time, so `csm` always knows exactly which
  sessions are open — it never has to guess from process lists or
  reconstruct state by re-parsing transcripts.
- **Browser-tab-group mental model.** `save` snapshots many sessions as a
  named group; `bookmark` snapshots a single one. Both are first-class,
  reopenable at any time with `csm open`.
- **`--fork` to resume without disturbing the original** — reopen a
  bookmark/group entry as a new session while the original stays intact.
- **`clean` to prune stale entries** that are no longer restorable (e.g.
  their transcript was deleted).
- **No daemon, no TUI, no AI summarization.** A single Go binary that
  `syscall.Exec`s the plain `claude` binary directly — minimal surface,
  nothing running in the background.

## Scope and caveats

Linux/macOS only. No daemon, no tmux integration — `csm open` resumes one
session at a time by `cd`-ing into its directory and exec'ing
`claude --resume <id>` directly (`syscall.Exec`, replacing the current
process). Because that bypasses the shell, any shell function/alias/flag you
normally wrap `claude` in (e.g. a `claude() { command claude --foo "$@"; }`
function) is **not** applied — `csm open` always launches the plain
`claude` binary found on `PATH`.

## License

MIT — see [LICENSE](LICENSE).
