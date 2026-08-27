# theboringoffice

**A startup office in your terminal, staffed by real agents.** (v2 — Go + Bubble Tea)

Chat with the boss. Watch the floor — employees get up, walk to the manager,
take the task, type it out, drop the mail, hit the tea machine. The right panel
is yours: chat, agents, board, mail, activity, git — switch with `tab`. The
left pane flips between the floor and the in-TUI browser with `ctrl+b`.

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

Pick the LLM transport at install time with `--backend opencode|claudecode`
(opencode is the default; claudecode needs the
[claude](https://docs.anthropic.com/en/docs/claude-code) CLI on PATH — a
missing one is a warning, not a failure). The choice is seeded into
`brain.json`'s `backend.name` and the summary box shows it:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh -s -- --backend claudecode
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
session and resends the batch. Every completion sweeps the board behind it too:
a return (or a flushed queue item) flips stranded DOING rows to DONE — the exact
brief row, then the same worker's oldest, then a normalized ≥8-char title-prefix
match, never with cross-worker doubt, never an agentmemory-mirrored row — with
one dim `[office] board sync: flipped N rows to done` note when anything moved.

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

  a `  ↳` sneak row previews the latest action; `ctrl+g` expands the
  full thread inline in the chat. Clicking the thread's frame (the
  header, its collapsed `  ↳` sneak, or an expanded thread's closing
  summary) closes the transcript view and opens the whole thread as a
  **nested focus pane** — the clicked agent's complete transcript (every
  tool call, think body, and per-call `↳ diff` you can click open)
  scrolling at the frame width with live pulses as it works; the
  statusbar reads "esc · ctrl+f back to office" and leaving returns the
  parent transcript underneath byte-identical (scroll, expansions, draft
  untouched). `ctrl+f` opens the pane without the mouse (most recently
  expanded thread wins, else any live thread). The main chat defers
  re-rendering while the pane is open and catches up in exactly one
  pass on return.
- **`/stop` + free-send** — `enter` while the boss works never blocks: the
  prompt free-sends into the backlog and the status line reads "busy · N
  queued". `/stop` aborts current work (boss + workers). If a busy turn
  goes silent for 2m, the wedge watchdog notes it once — it reads on the
  status bar and in the activity log — never in the transcript.
- **Long memory** — reaching the TOP of the transcript walks one page
  further back in older history: one async page-walk per landing, spliced
  byte-stable into the head, riding the same scroll keys (nothing new).
  Drained top latches silently — no row — and failures back off without
  banners; question and permission floats suppress the gesture. Demo walks
  its canned 500-row history; real older history is live-only. (This is
  the boss's *scrollback*. The office's *completed-work* memory is the
  ledger below.)
- **Long memory ledger** — every completed dispatch (a returned worker or
  a flushed queue item) lands in `<dir>/.opencode/office-ledger.md`:
  newest-first, capped at 50, one `` ### date · title — worker (role) ·
  `verdict` `` block per completion carrying the summary line, files,
  verify digest, proof one-liner and a `ledgerId`. The record *also* rides
  to agentmemory as an `office_dispatch_done` observation when that server
  is up — file-only when it's not, and the boot splash's `memory:` line
  (`memory: ledger armed` vs `memory: file-only (agentmemory :3111
  refused)`) + the status line's `memory:` note say which lane got it.
  The charter pass seeds the file and wires it into the boss's
  instructions, so the NEXT session's boss reads what this project already
  finished **before** re-dispatching it — no more "I don't have any
  memory of a prior task".
- **Office memory** — `/memory` shows what the office shipped, newest
  first, straight from the project ledger (`.opencode/office-ledger.md`):
  each completed dispatch lands a record with its title, worker, role,
  verdict, touched files and proof command. The header counts the records
  and names the memory state (`agentmemory OK` when the backend's
  agentmemory probe is hot, `file-only` otherwise), a missing ledger reads
  as an honest empty state, and every completed dispatch also stamps a
  `[memory] recorded: <title> → ledger` line into the activity tab.
  `/memory <substr>` narrows the rows case-folded over title/files.
- **Plan/build modes** — `ctrl+p` flips between **build** (default) and
  **plan** (`[plan]` on the statusbar): a mode toggle only — prompts ride
  the read-only plan agent, chat keeps focus, and the plan pane stays
  hidden until it has content. Status line: `plan · boss plans read-only ·
  ctrl+p exits · ctrl+x approves a presented plan`. A completed boss reply
  mirrors into the floor slot passively (you keep typing; the hint swaps to
  `plan · click to edit · ctrl+x approve → build · ctrl+p exits`) — but only
  when it looks like a plan: plan-shaped replies present; status chatter
  doesn't (the pane keeps its last plan, one dim note explains), or click
  the pane region to scratch one from the starter template (`mermaid` block
  inside — captioned `╭─ mermaid diagram ─╮` in the read-only glamour
  render, `esc` back to chat). Edits latch the pane as yours — a fresh boss
  reply leaves it untouched (dim note: `boss replied — your edited plan
  kept`). `ctrl+x` approves → `Approved plan — implement it exactly as
  specified:` plus the plan body goes to the build agent and the mode flips
  back to build; an empty buffer or the untouched starter is refused with a
  notice. A non-empty, non-starter, plan-shaped buffer persists across boots.
  The focused pane is a real editor: `shift+arrows`/`shift+home`/`shift+end`
  mark a selection, `ctrl+a` marks the whole buffer, and with a live mark
  `ctrl+c` copies / `ctrl+x` cuts it (no mark: `ctrl+x` keeps its
  approve-toggle, `ctrl+c` keeps its quit claim), `ctrl+v`/`super+v`/`cmd+v`
  paste over it, and a mouse drag marks + copies with the same
  `Copied N chars` statusbar note — `esc` clears a live mark first, then
  blurs.
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
Session state lives per working directory at
**`~/.theboringoffice/projects/<dirhash>/session.json`**.
Upgrading? Your old `~/.theboringoffice/sessions/<dirhash>` session files and
the pre-rename `~/.grafeio` config, theme and sessions are still READ (writes
land on the new paths only), and `GRAFEIO_*` env vars keep
working as fallbacks for the new `THEBORINGOFFICE_*` ones.

```jsonc
{
  "boss":    { "name": "boss (oikonomos)", "model": "anthropic/claude-sonnet-4-5" },
  "roles":   { "developer": { "namePrefix": "tekton" }, "scout": { "namePrefix": "skopos" },
               "reviewer": { "namePrefix": "dikastes" }, "runner": { "namePrefix": "hemerodromos" } },
  "ui":      { "theme": "noir", "power": "auto", "tickMs": 0, "ambientChatter": true,
               "sounds": "on", "sidebarWidth": 0, "compact": false },
  "backend": { "name": "opencode", "agentmemoryUrl": "http://localhost:3111", "server": "", "agentmemoryPollS": 5 }
}
```

`backend.name` selects the LLM transport: `opencode` (the default —
`opencode serve` + SSE, what pre-existing brain.json files silently mean) or
`claudecode` (the claude CLI in headless stream-json mode, one process per
turn). `install.sh --backend` seeds it, `--backend` overrides it for one
boot, `/backend` swaps it mid-flight and persists it — the topbar shows the
active name between mode and agents, and session.json pins session ids PER
TRANSPORT (`primaryIDs`), so swapping back later resumes that transport's
own session instead of cross-pinning ids.

- The `claudecode` transport declares its rendered `request_user_dialog`
  kinds up front (an `initialize` control_request carrying
  `supportedDialogKinds`, the first stdin line of every process — the CLI
  only sends a kind some attached client declared) and renders 28 of them
  as boss question-modal pages: the 12 permission gates
  (`permission_ask_user_question`, `permission_prompt`, `permission_bash`,
  `permission_browser`, `permission_enter_plan_mode`,
  `permission_exit_plan_mode_v2`, `permission_file`, `permission_monitor`,
  `permission_powershell`, `permission_skill`, `permission_webfetch`,
  `permission_workflow` — Allow once / Allow always / Reject), the 12 enum
  consent kinds (`cloud_sync_consent`, `fable_overage_consent_prompt`,
  `refusal_fallback_prompt`, `chrome_install_upsell`,
  `chrome_install_setup`, `auto_mode_setup_review`, `resume_return`,
  `managed_settings_security`, `auto_default_nudge`, `cost_threshold`,
  `ide_onboarding`, `it2_setup` — the answer is the kind's enum string),
  and the 4 structured kinds (`goal_proposal`, `auto_mode_flagged_allow`,
  `sandbox_network_access`, `peer_inbound_approval`). Dismissal is always
  the envelope-level `{behavior:"cancelled"}` (the CLI substitutes the
  kind's own default); `computer_use_approval`, `local_jsx` and
  `mcp_url_elicitation` are deliberately never declared — a kind the
  office can't render faithfully stays parked for the CLI's dialog
  deadline instead of getting a fabricated answer.

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

Seven tabs with a real terminal in the middle (the browser lives on the
left pane — see the last bullet):

- **terminal** — an OS shell (`$SHELL`) on a real PTY, by `creack/pty`:
  lazily spawned on first visit, resizes with the panel, mouse scrolls the
  scrollback, `r` respawns when dead. The terminal does NOT grab the
  keyboard by default: the office keys keep working on its tab (`tab`,
  `shift+tab`, `1..7`, `q`). `ctrl+space` is the ONE capture toggle and
  flips it BOTH ways — dive into shell capture (now every key — `tab`
  completion, `shift+tab` sends `\x1b[Z`, digits, `q`, `ctrl+c` SIGINT —
  goes to the shell) and the same key releases back to the office keys in
  place. `ctrl+o` also releases (alias — release only, never a dive), and
  leaving the tab auto-releases, so every visit starts released. (`ctrl+space`
  emits `0x00`, a key of its own — nothing shares `tab`'s byte.) The status
  bar reads `office keys · ctrl+space → shell · ctrl+q quit` (released) /
  `typing → shell · ctrl+space release · ctrl+q quit` (captured).
  **Mouse select** — drag over terminal text (live screen or scrolled-up
  scrollback, focused or released) highlights it in reverse video; on release
  it copies to the system clipboard (pbcopy / wl-copy / xclip / xsel, plus
  OSC52 best-effort) and a dim `· Copied N chars` note rides the badge row.
  `esc` or typing cancels the highlight. Shell-side mouse modes never fight
  it: the grid ignores `?1000/?1006` and no mouse bytes reach the PTY.
- **chat** — the boss conversation; **agents** / **board** / **mail** /
  **activity** — office telemetry.
- **git** — live repo status: a header summary (modified / added /
  untracked / deleted counts, plus a +/- lines line) and a scrollable file
  list with status glyphs (`M` modified, `A` added, `??` untracked,
  `D` deleted, `R` renamed; a `*` suffix means staged). `enter` or a mouse
  click on a file opens a colored unified diff (`+` green, `−` red, `@@`
  hunk headers); `b` or `esc` returns to the list, `r` refreshes. A clean
  tree shows "working tree clean".
- **browser** — a real in-TUI page viewer on the LEFT pane: the floor
  slot carries a two-tab switcher, **floor** (the office, the default) and
  **browser**, flipped with `ctrl+b`. Web pages render as navigable
  text+link rows, no external dependency. `/open <url>` flips the slot to
  the browser for you: `file://` URLs and bare paths read off disk,
  `https://` opens anywhere and localhost always works; plain `http://`
  beyond localhost needs `THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1` (never
  network silently); fetches are 10s-bounded and 4 MiB-capped, and non-HTML
  payloads land a dim `unsupported content type` row. The page paints a
  `▸ <url> · <title>` bar over wrapped paragraphs, bold headings, bullet
  rows, `a │ b` table rows, code rows, `🖼 <alt>` image chips (image
  bytes are never fetched), and links as `text [n]` with the URLs indexed
  in a side map. `↑`/`↓` move the link cursor (dim → bright), `o` opens
  the focused link (a local file goes to the OS browser; http(s)
  navigates in place), `[` / `]` walk the 100-page history ring (scroll
  restored), `r` reloads, `q` / `esc` leaves back to the floor. The right
  strip never moves — the browser is not one of its tabs.

Layout lives in the config *and* in the app:

```text
/compact on|off     narrow sidebar (30) + short tab letters + 2-row input
/mode normal|compact   same, persisted to brain.json
/wide <n>           sidebar width 26..100 (0 = default 80), persisted
/zen                fullscreen floor, minimal chrome — any key exits
```

Sounds: `ui.sounds = on | bell | off` (or `THEBORINGOFFICE_MUTE=1`).

Browser lane for `o`: on kitty-capable terminals (kitty, ghostty, WezTerm
— never inside tmux) an installed
[`terminal-browser`](https://github.com/zenbu-labs/terminal-browser) binary
opens targets in-terminal first, cascading to the system opener on any
failure; `THEBORINGOFFICE_NO_TERMINAL_BROWSER=1` disables the lane.

Browser premium lane: on kitty/ghostty (tmux and the iTerm2 family
stay text) with the
[`terminal-browser`](https://github.com/zenbu-labs/terminal-browser)
binary on PATH, the browser tab embeds the REAL Chromium browser
automatically — every open (`/open`, the agent's browser-open tool, an
in-pane link follow) paints the live page (CSS, images, full rendering)
inside the left-pane slot under the ` zenbu ` badge and the `▸ zenbu
terminal-browser · <url>` top strip; keys reach the embedded browser
while `ctrl+b` / `q` / `esc` still belong to the office (leaving the slot
suspends the session, returning resumes it). The text lane is the
universal default everywhere else — non-kitty terminals, the binary
absent, or either kill-switch (`THEBORINGOFFICE_TERMINAL_BROWSER_OFF=1`
or `THEBORINGOFFICE_NO_TERMINAL_BROWSER=1`) armed — and the automatic
fallback: a non-zero or instant (<300ms) exit drops that URL back to the
text viewer with its history kept and a dim
`zenbu exited (<code>) — falling back to text mode` note. The pane always
explains its lane choice: while the text lane shows, one dim row under the
location bar says why — the `terminal-browser` binary missing (with the
install link), an unsupported terminal, or a kill-switch left armed.

## Keys

| key | does |
|---|---|
| `tab` / `shift+tab` / `1..7` | switch panel: chat · **terminal** · agents · board · mail · activity · git — only inside shell capture (`ctrl+space` toggle) are these NOT intercepted (`tab` completes in the shell, `shift+tab` sends `\x1b[Z`, digits type) |
| `ctrl+b` | flip the LEFT pane's slot: **floor** ⇄ **browser** (the browser is not a right-strip tab; works from every surface except a captured shell) |
| `enter` / click a file (git tab) | open its colored unified diff — `b` / `esc` back to the list, `r` refresh |
| `/open <url>` | open a page in the **browser** (auto-flips the left slot) — `file://`, a bare path, or `http(s)://…` (`https://` anywhere; plain `http://` beyond localhost needs `THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1`); inside the pane: `↑`/`↓` pick a link, `o` opens it (local file → OS browser, http(s) → navigate), `[` / `]` history, `r` reload, `pgup`/`pgdn` scroll, `q` / `esc` back to the floor |
| `↑` `↓` `pgup` `pgdn` / wheel | scroll the active panel |
| `enter` | send to the boss (chat) |
| `shift+enter` | newline |
| `@` at a word start (chat) | open the attach-file picker — `↑`/`↓` choose · `enter`/`tab` attach · `esc` close |
| `ctrl+v` (chat) | paste text — attaches the image instead when the clipboard holds one (macOS: `pngpaste` or osascript) |
| `backspace` on an empty input | drop the newest attachment chip |
| `ctrl+t` | expand/collapse completed thinking blocks |
| `ctrl+d` | expand/collapse diff blocks |
| `ctrl+f` | open the most recent worker thread as a fullscreen focus view (tool calls + thinks + per-call diffs, live-pulsed; most recently expanded thread wins, else any live thread) — `esc` / `ctrl+f` closes back, office state byte-identical |
| `o` | chat: with a text selection (press/drag) live over a bubble wearing the dim `· o (open)` beacon — open its URL / on-disk file / media image in the browser (kitty-capable terminals: opens with zenbu `terminal-browser` when present; otherwise the system default browser — `open -g` on macOS, `xdg-open` on Linux); several targets float a small picker (`↑`/`↓` move, `enter` opens, `esc` cancels); no selection or no verified target and the key just types `o` |
| `ctrl+p` | toggle plan/build mode — chat keeps focus, the pane opens only once a plan has content (not while shell-captured or with an open float) |
| `ctrl+x` (plan mode) | selection live in the pane: cut the marked text; otherwise approve-toggles (double-press) the presented/edited plan → sent to the build agent, mode flips back to build |
| `ctrl+space` | terminal tab: toggle shell keyboard capture BOTH ways (opt-in; released is the default and leaving auto-releases) |
| `ctrl+o` | release shell capture (alias) |
| `ctrl+q` | arm quit (works everywhere, embedded terminal included) |
| `y` `a` `n` `esc` | answer a permission prompt |
| `q` / `ctrl+c` | quit |

Attachments stage as dim `📎` chips above the input (cap 5 — the oldest
drops past it), ride the message queue like text, and go out as prompt file
parts; the echoed user bubble shows a `· 📎 N` suffix. `/clear` or a send
clears the chips. The `@` picker honors the repo's `.gitignore` (plus
built-in filters for `node_modules/`, `.venv/`, `__pycache__/` and other
build output), so caches and dependency trees never crowd the list.

Images work the other way too: when a **boss turn carries an image** (a
file part the serve stored — e.g. a diagram it read back), the reply
transcript shows a `🖼 paste-diagram.png · 8×8 · image/png` chip above the
answer, followed by an inline preview — v1 paints a half-block truecolor
raster (`▀` cells, fg = top pixel / bg = bottom pixel) that renders in any
color terminal; `/images auto|ascii|off` picks the posture (`auto` probes
the terminal's protocol lane — kitty → iterm2 → ASCII fallback, both
native lanes fall back to ASCII in v1; `off` keeps chips only). Previews
are boss-turn only, ≤4 per turn (extras degrade to chip rows), remote
URLs are never fetched, and undecodable payloads degrade to a dim `🖼
<name> · unsupported image · click txt link` row. Any bubble — those image
turns included — that carries an http(s) URL or a file path that actually
exists on disk (a `.gitignore`-style strictness: `/abs`, `~/`, `./` tokens
are `os.Stat`-verified, directories and dead tokens never qualify) wears the
dim `· o (open)` beacon; mark it with a mouse press/drag and press `o` to
open the target in the OS default browser, with the opened target logged to
the **activity** tab as `→ opened: <name>`.

## Commit attribution — TheBoringMajdoor

Every commit authored through the office is stamped with one trailer:

```text
Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>
```

The majdoor is the office's bot profile on GitHub, and it already exists:
https://github.com/themajdoor, with `themajdoor@theboring.name` registered
under its email settings. GitHub renders co-author credit — and the
majdoor's avatar — on each stamped commit out of the box; there is no
setup step. (If the account ever hides behind GitHub's noreply shield,
swap the trailer's address for its `<id>+themajdoor@users.noreply.github.com`
one.)

To stamp the same trailer on commits you write by hand, install the
`commit-msg` hook into any repo (idempotent, backs up a pre-existing hook to
`commit-msg.bak-majdoor`, and resolves the real hooks dir so `core.hooksPath`
and worktrees work):

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/scripts/install-majdoor-hook.sh | sh -s -- /path/to/repo
```

From a checkout it's just `scripts/install-majdoor-hook.sh /path/to/repo`;
either way `--uninstall` peels the hook back off and restores the backup. The
office's own installer can do it in the same breath —
`install.sh --majdoor-hook /path/to/repo` — and office-authored commits stamp
the identical trailer automatically, no hook needed.

### Author vs co-author

The two layers do different jobs:

- **Office auto-commits** (`THEBORINGOFFICE_AUTO_COMMIT=true`) are *authored*
  by the majdoor. The office exports the four identity vars —
  `GIT_AUTHOR_NAME` / `GIT_AUTHOR_EMAIL` / `GIT_COMMITTER_NAME` /
  `GIT_COMMITTER_EMAIL`, all `TheBoringMajdoor
  <themajdoor@theboring.name>` — into every shell it spawns, so those
  commits show the majdoor as both author and committer. Hand-rolled flows
  outside the office get the same four vars by sourcing
  `. scripts/majdoor-env.sh` (it exports nothing unless the flag is exactly
  `true`).
- **Hand-written commits** keep your identity and pick up the
  `Co-authored-by` trailer above via the `commit-msg` hook — credit, not
  authorship.

Why doesn't authorship ship as a `prepare-commit-msg` hook too? Because a git
hook can't set it: git runs hooks as child processes, and environment exported
inside a child never flows back to the parent git process that writes the
commit. That's why this layer ships as env, not as a hook.

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
| `/memory [filter]` | office memory — completed dispatches from the project ledger, newest first (a substring narrows rows over title/files) |
| `/clear` | empty the chat |
| `/queue [clear]` | show the backlog (`clear` drops it) |
| `/route` | force-dispatch the backlog now |
| `/perm` | re-open an esc'd permission prompt |
| `/diffs on\|off` | expand/collapse file diffs |
| `/images [auto\|ascii\|off]` | boss-turn image previews — bare reports the posture + detected lane; with a mode it persists to brain.json (`auto` = detect, ASCII fallback; `off` = chips only) |
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
| `/backend [opencode\|claudecode]` | bare prints the active LLM transport; with a name it swaps mid-flight — but only while the office is IDLE (a boss turn, queued backlog, live workers, or an unanswered question/permission gets a refusal naming the blockers). The swap archives the turn, persists the name to brain.json, and lands one `[theboringoffice] backend: <old> → <new>` status line |
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
