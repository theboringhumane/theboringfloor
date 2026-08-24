# theboringoffice

**A startup office in your terminal, staffed by real agents.** (v2 — Go + Bubble Tea)

Chat with the boss. Watch the floor — employees get up, walk to the manager,
take the task, type it out, drop the mail, hit the tea machine. The right panel
is yours: chat, agents, board, mail, activity, git — switch with `tab`.

Underneath the wallpaper, it's all real: the manager is
**[Oikonomos](https://github.com/theboringhumane/oikonomos)**, the employees
are **opencode sub-agents**, the task board is **agentmemory actions**, the
mail room is **agentmemory signals**.

![chat tab](docs/shots-go/chat.png)

## Install

One-liner — binary plus [agentmemory](https://github.com/rohitg00/agentmemory)
wired as a reboot-safe service:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh
```

Then `theboringoffice --demo` for touring mode.

Pin the version you want — every tagged release gets stamped into the binary.

```bash
go install github.com/theboringhumane/theboringoffice/cmd/theboringoffice@v0.1.0   # exact tag
go install github.com/theboringhumane/theboringoffice/cmd/theboringoffice@latest   # newest release
```

Or pull a prebuilt archive (darwin/linux, amd64/arm64, `checksums.txt`
alongside) from [GitHub Releases](https://github.com/theboringhumane/theboringoffice/releases):

```bash
curl -LO https://github.com/theboringhumane/theboringoffice/releases/download/v0.1.0/theboringoffice_0.1.0_darwin_arm64.tar.gz
tar -xzf theboringoffice_0.1.0_darwin_arm64.tar.gz   # -> theboringoffice binary; move it onto your PATH
```

Check the stamp — version, commit, and build date ride every binary:

```bash
$ theboringoffice --version
theboringoffice v0.1.0 (abc1234, 2026-08-22)
```

Then run it:

```bash
theboringoffice            # live: spawns `opencode serve`, real boss (oikonomos)
theboringoffice --demo     # touring mode: simulated events, labeled DEMO
theboringoffice --server http://127.0.0.1:4096   # attach to an existing server
```

## Resuming a session

The office normally re-opens your last chat on launch. To step back into a
*specific* past session instead, hand it the ID:

```bash
theboringoffice --session <your-session-id>   # short form: -s <your-session-id>
```

A bad or forgotten ID never blocks the door — the app warns about the unknown
session, then boots normally with a fresh find-or-create one. No flag, no
change: the automatic last-chat restore still happens.

In-app, `/session` opens a picker of the past sessions the server keeps for
this directory — each row shows the title (or short ID), age, message count
and short ID, with the session you're on marked. Same rules as every other
picker: type to narrow, `↑`/`↓` move, `enter` accepts, `esc` cancels with no
side effects. Accepting switches the office onto it live — nothing from the
old transcript bleeds in — and a dim notice confirms: `resumed session <id>
(explicit pin) · /new for a fresh office`. The choice sticks, so the next
boot re-opens it. Picking the session you're already on is a harmless no-op;
if the boss is mid-work the switch is refused with a notice telling you to
`/stop` or wait — the picker never interrupts active work. And when there's
no server list to show (demo mode, server down), `/session` falls back to
printing the current session ID and where it lives on disk — note it down
for later. Reach for the flag when the last-chat restore landed you in the
wrong room, or when you're deliberately hopping between sessions.

![agents tab](docs/shots-go/agents.png)

Working detail stays visible: opencode-style diffs (line numbers, full-row
red/green tints, inline syntax), thinking blocks, tool calls, permission
popovers (clickable allow-once/always/reject menu, queued "1 of N" when asks
stack up), boss question wizard pages, sub-agent work threads, and a message
queue that never locks you out while the boss types.

![diffs](docs/shots-go/chat-diff.png)

Everything live-streams: the boss's replies type out character-by-character on
one bubble, and thinking transcripts unfold in real time before auto-collapsing.

![streaming](docs/shots-go/chat-stream.png)

And the queue is not a tunnel — it's a **backlog the office manages**: items get
board rows, flush goes out as one `[BATCH DISPATCH]` the boss decomposes into
parallel sub-agents (`/route` forces it early), and a dead boss respawns a fresh
session and resends the batch.

## Popovers and polish

- **Concierge** — send while the boss is mid-task and the office concierge
  answers instantly, as a real conversation turn (noted in chat as "office
  routed: boss busy → concierge"); if the concierge is unavailable a notice
  says so and the prompt rides the backlog instead.
- **Question wizard** — boss questions open as popover pages, classified
  automatically: **text** (free answer), **radio** (pick one), **checkbox**
  (pick several), **confirm** (yes/no). Deferred one? `/question` re-opens it.
- **Permission queue** — permission asks stack: the front of the queue shows
  "1 of N" with a clickable allow-once / always / reject menu
  (`y` `a` `n` `esc`). `/perm` re-opens an esc'd prompt.
- **Work threads** — sub-agent work renders as opencode-style threads right
  in the chat:

  ```text
  ⠿ Explore Task — Scout question kinds recon (· 2 tool calls ✓ done)
    ↳ Read internal/panels/chat.go … running
  ```

  a `  ↳` sneak row previews the latest action; click the header (or
  `ctrl+g`) to expand the full thread.
- **`/stop` + free-send** — `enter` while the boss works never blocks: the
  prompt free-sends into the backlog and the status line reads "busy · N
  queued". `/stop` aborts current work (boss + workers).
- **Plan/build modes** — `ctrl+p` flips between **build** (default) and
  **plan** (`[plan]` on the statusbar): a mode toggle only — prompts ride
  the read-only plan agent, chat keeps focus, and the plan pane stays
  hidden until it has content. Status line: `plan · boss plans read-only ·
  ctrl+p exits · ctrl+x approves a presented plan`. A completed boss reply
  mirrors into the floor slot passively (you keep typing; the hint swaps to
  `plan · click to edit · ctrl+x approve → build · ctrl+p exits`), or click
  the pane region to scratch one from the starter template (`mermaid` block
  inside — captioned `╭─ mermaid diagram ─╮` in the read-only glamour
  render, `esc` back to chat). Edits latch the pane as yours — a fresh boss
  reply leaves it untouched (dim note: `boss replied — your edited plan
  kept`). `ctrl+x` approves → `Approved plan — implement it exactly as
  specified:` plus the plan body goes to the build agent and the mode flips
  back to build; an empty buffer or the untouched starter is refused with a
  notice. A non-empty, non-starter plan persists across boots.
- **Ambient floor** — coffee steam off the tea machine, blinking server-rack
  LEDs, and an uplink ripple along the server-room wall — all tick-driven
  (no timers), so an idle office stays cheap.
- **Transcript copy** — drag-select over the chat transcript with the
  mouse highlights the span; on release it copies through the terminal's
  OSC52 clipboard, and a transient "Copied N chars" note rides the status
  bar. `esc` or a plain click clears the highlight.
- **Boot splash** — animated startup splash, hard-capped at ~4s; any key
  skips it.

## Configure the brain

One file runs the office: **`~/.theboringoffice/configs/brain.json`** (created with
defaults on first run; inspect anytime with `theboringoffice --print-default-config`).
Upgrading from grafeio? Your old `~/.grafeio` config, theme and sessions are
still READ (writes land on the new paths only), and `GRAFEIO_*` env vars keep
working as fallbacks for the new `THEBORINGOFFICE_*` ones.

```jsonc
{
  "boss":    { "name": "boss (oikonomos)", "model": "anthropic/claude-sonnet-4-5" },
  "roles":   { "developer": { "namePrefix": "tekton" }, "scout": { "namePrefix": "skopos" },
               "reviewer": { "namePrefix": "dikastes" }, "runner": { "namePrefix": "hemerodromos" } },
  "ui":      { "theme": "noir", "power": "auto", "tickMs": 0, "ambientChatter": true,
               "sounds": "on", "sidebarWidth": 0, "compact": false },
  "backend": { "agentmemoryUrl": "http://localhost:3111", "server": "", "agentmemoryPollS": 5 }
}
```

Boss model rides every prompt as `{"model":{"providerID","modelID"}}` (serve
1.18.19 `/doc`-verified). Role models are noted as best-effort (sub-agent model
dispatch is opencode's call). `/power`, `/model`, `/theme` all write back.

### Battery (`ui.power`)

| mode | busy | idle | drift (1 min quiet) |
|---|---|---|---|
| `auto` (default) | 180ms ticks | 1s | 3s |
| `performance` | 150ms flat | — | — |
| `saver` | 400ms | 2s | — |

Idle-detection covers streaming, pending replies, walkers, open modals, ambient
bubbles. Renders are memoized (frame digest on the app, `(size, planGen, tick,
renderRev)` on the floor), the agentmemory board poll backs off 2× after five
quiet syncs (cap 4×, reset on change). The office goes cheap when nothing moves.

## The sidebar is a cockpit

Seven tabs with a real terminal in the middle:

- **terminal** — an OS shell (`$SHELL`) on a real PTY, by `creack/pty`:
  lazily spawned on first visit, resizes with the panel, mouse scrolls the
  scrollback, `r` respawns when dead. While the terminal tab is active it
  grabs the keyboard: `tab` goes to the shell (completion), `shift+tab`
  sends `\x1b[Z`, and number keys plus every other office shortcut are NOT
  intercepted — the only app-kept keys are `ctrl+o` (release → back to the
  chat tab) and `ctrl+q` (arms quit, works everywhere). The status bar reads
  `typing → shell · ctrl+o release · ctrl+q quit`.
- **chat** — the boss conversation; **agents** / **board** / **mail** /
  **activity** — office telemetry.
- **git** — live repo status: a header summary (modified / added /
  untracked / deleted counts, plus a +/- lines line) and a scrollable file
  list with status glyphs (`M` modified, `A` added, `??` untracked,
  `D` deleted, `R` renamed; a `*` suffix means staged). `enter` or a mouse
  click on a file opens a colored unified diff (`+` green, `−` red, `@@`
  hunk headers); `b` or `esc` returns to the list, `r` refreshes. A clean
  tree shows "working tree clean".

Layout lives in the config *and* in the app:

```text
/compact on|off     narrow sidebar (30) + short tab letters + 2-row input
/mode normal|compact   same, persisted to brain.json
/wide <n>           sidebar width 26..100 (0 = default 80), persisted
/zen                fullscreen floor, minimal chrome — any key exits
```

Sounds: `ui.sounds = on | bell | off` (or `THEBORINGOFFICE_MUTE=1`).

## Keys

| key | does |
|---|---|
| `tab` / `shift+tab` / `1..7` | switch panel: chat · **terminal** · agents · board · mail · activity · git — while **terminal** is active these are NOT intercepted (`tab` completes in the shell, `shift+tab` sends `\x1b[Z`, digits type) |
| `enter` / click a file (git tab) | open its colored unified diff — `b` / `esc` back to the list, `r` refresh |
| `↑` `↓` `pgup` `pgdn` / wheel | scroll the active panel |
| `enter` | send to the boss (chat) |
| `shift+enter` | newline |
| `@` at a word start (chat) | open the attach-file picker — `↑`/`↓` choose · `enter`/`tab` attach · `esc` close |
| `ctrl+v` (chat) | paste text — attaches the image instead when the clipboard holds one (macOS: `pngpaste` or osascript) |
| `backspace` on an empty input | drop the newest attachment chip |
| `ctrl+t` | expand/collapse completed thinking blocks |
| `ctrl+d` | expand/collapse diff blocks |
| `ctrl+p` | toggle plan/build mode — chat keeps focus, the pane opens only once a plan has content (not with terminal focus or an open float) |
| `ctrl+x` | plan mode: approve the presented/edited plan → sent to the build agent, mode flips back to build |
| `ctrl+o` | release the embedded terminal's keyboard grab → back to the chat tab |
| `ctrl+q` | arm quit (works everywhere, embedded terminal included) |
| `y` `a` `n` `esc` | answer a permission prompt |
| `q` / `ctrl+c` | quit |

Attachments stage as dim `📎` chips above the input (cap 5 — the oldest
drops past it), ride the message queue like text, and go out as prompt file
parts; the echoed user bubble shows a `· 📎 N` suffix. `/clear` or a send
clears the chips.

## Slash commands

`/` at a word start opens the picker: type to narrow (`/th` → `/theme
/themes /thinking`), `↑`/`↓` move, `enter`/`tab` accept. Arrowing through
`/theme` matches applies a live preview.

| command | does |
|---|---|
| `/help` | this list |
| `/theme <name>` | switch theme (persists) |
| `/themes` | list themes |
| `/thinking on\|off` | show/hide thinking blocks |
| `/tools on\|off` | show/hide tool one-liners |
| `/status` | office status |
| `/mcp [reconnect <name>]` | MCP server status; reconnect one server |
| `/clear` | empty the chat |
| `/queue [clear]` | show the backlog (`clear` drops it) |
| `/route` | force-dispatch the backlog now |
| `/perm` | re-open an esc'd permission prompt |
| `/diffs on\|off` | expand/collapse file diffs |
| `/question` | re-open a deferred boss question |
| `/power auto\|performance\|saver` | power governor |
| `/model provider/model` | boss model |
| `/compact on\|off` | compact layout this session |
| `/mode normal\|compact` | layout mode (persists) |
| `/wide 26..100` | sidebar width (0 = default) |
| `/zen` | fullscreen floor, any key exits |
| `/focus floor` | alias of `/zen` |
| `/stop` | abort current work (boss + workers) |
| `/new` | fresh office (transcript archived) |
| `/session` | past-sessions picker — accept to switch the office live (sticks to the next boot); no server list → prints the current id + where it lives |
| `/quit` | exit theboringoffice |

## What v2 (Go) changed

- Chat moved from a fixed bottom bar into a **tabbed right panel** with
  **glamour-rendered markdown** — the boss's `**bold**`, lists and code fences
  format and wrap inside the panel instead of bleeding through the UI.
- Scrolling everywhere (viewport), mouse wheel, multi-line input, typing
  spinner while the boss works.
- New **activity** tab: rolling event log (dispatches, returns, blocks).
- Native single binary. Themes: `--theme noir|paper|mono|dracula|solarized`
  (also `/theme` in-app, persisted to `~/.config/theboringoffice/theme`).
- Slash commands in chat — the full picker-driven table is under
  [Slash commands](#slash-commands).
- The Ink/Node v0.1 app lived on as git tag `node-v0.1.0`.

## Behind the glass

- The boss is a real `opencode serve` primary session — oikonomos manages.
- Employees are real `task` sub-agent sessions (SSE → hire/dispatch/working/returned).
- `docs/shots-go/*` are produced by the app itself + [freeze](https://github.com/charmbracelet/freeze)
  (deterministic scripted backends: `go run ./cmd/uishot`, `go run ./cmd/floorshot`).

Architecture and the floor plan: [docs/architecture.md](docs/architecture.md).

## agentmemory

The office runs on memory. [agentmemory](https://github.com/rohitg00/agentmemory)
is a persistent memory server for AI coding agents — sessions, recall, lessons,
semantic search across sessions. The board already rides its actions and the mail
room its signals; run it alongside and the office remembers across reboots —
decisions, lessons, yesterday's batch: they survive `ctrl+q`. The install script
sets it up as a reboot-safe service, or roll your own with
`npm install -g @agentmemory/agentmemory` ([npm](https://www.npmjs.com/package/@agentmemory/agentmemory)).

— theboringoffice, by [theboringhumane](https://github.com/theboringhumane) at theboredteam.

MIT © theboredteam
