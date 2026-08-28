<div align="center">

<img src="website/public/imgs/logo.jpg" alt="theboringoffice" width="88" />

# theboringoffice

**A startup office in your terminal, staffed by real agents.**

Chat with the boss. Watch the floor — employees walk, type, drop mail, hit the tea machine.
Right panel is yours. Left pane is the floor (or the in-TUI browser).

[Website](https://theboringoffice.pages.dev) · [Docs](https://theboringoffice.pages.dev/docs) · [Get started](https://theboringoffice.pages.dev/get-started) · [Discord](https://discord.gg/YPDsHVHTVf)

<br />

[![Go](https://img.shields.io/github/go-mod/go-version/theboringhumane/theboringoffice?style=for-the-badge&logo=go&logoColor=white&label=Go)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/theboringhumane/theboringoffice?style=for-the-badge&logo=github&label=release)](https://github.com/theboringhumane/theboringoffice/releases)
[![Release workflow](https://img.shields.io/github/actions/workflow/status/theboringhumane/theboringoffice/release.yml?style=for-the-badge&logo=githubactions&logoColor=white&label=build)](https://github.com/theboringhumane/theboringoffice/actions/workflows/release.yml)
[![Go Reference](https://img.shields.io/badge/pkg.go.dev-reference-007d9c?style=for-the-badge&logo=go&logoColor=white)](https://pkg.go.dev/github.com/theboringhumane/theboringoffice)
[![License: MIT](https://img.shields.io/badge/license-MIT-111111?style=for-the-badge)](LICENSE)
[![Discord](https://img.shields.io/badge/Discord-join%20community-5865F2?style=for-the-badge&logo=discord&logoColor=white)](https://discord.gg/YPDsHVHTVf)

</div>

![chat tab](docs/shots-go/chat.png)

Under the wallpaper it is real: the manager is **[Oikonomos](https://github.com/theboringhumane/oikonomos)**, employees are **opencode sub-agents**, the board is **agentmemory actions**, mail is **agentmemory signals**.

## Install

One-liner — binary plus [agentmemory](https://github.com/rohitg00/agentmemory) as a reboot-safe service:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh
```

Pick the LLM transport at install (`opencode` default; `claudecode` needs the [claude](https://docs.anthropic.com/en/docs/claude-code) CLI):

```bash
curl -fsSL \
  https://raw.githubusercontent.com/theboringhumane/theboringoffice/main/install.sh | sh -s -- --backend claudecode
```

Then:

```bash
theboringoffice            # live office
theboringoffice --demo     # touring mode
theboringoffice --version  # stamp: version, commit, date
```

Pin a tag, or grab a prebuilt (darwin/linux, amd64/arm64) from [Releases](https://github.com/theboringhumane/theboringoffice/releases):

```bash
go install github.com/theboringhumane/theboringoffice/cmd/theboringoffice@latest
```

Full on-ramp: **[Getting started](https://theboringoffice.pages.dev/docs/getting-started)** · local notes in [`docs/`](docs/README.md).

## Docs

Manual lives on the site. This repo keeps a thin index so GitHub readers land in the right room.

| In-repo | Website |
|---|---|
| [Docs hub](docs/README.md) | [Docs home](https://theboringoffice.pages.dev/docs) |
| [Architecture](docs/architecture.md) | [Vision](https://theboringoffice.pages.dev/vision) |
| [Website](website/README.md) | [Get started](https://theboringoffice.pages.dev/get-started) |
| [Commands (`cmd/`)](cmd/README.md) | [Sounds](https://theboringoffice.pages.dev/sounds) |
| [Scripts](scripts/README.md) | [Blog](https://theboringoffice.pages.dev/blog) |

### Manual (site)

**Install & setup**
- [Getting started](https://theboringoffice.pages.dev/docs/getting-started) — curl, demo, live office, `--session`

**Core**
- [Backends](https://theboringoffice.pages.dev/docs/backends) — opencode or claudecode
- [Chat & work threads](https://theboringoffice.pages.dev/docs/chat-and-threads)
- [Plan mode](https://theboringoffice.pages.dev/docs/plan-mode)

**Workflow**
- [Permissions & questions](https://theboringoffice.pages.dev/docs/permissions-and-questions)
- [Queue, board & memory](https://theboringoffice.pages.dev/docs/queue-board-memory)

**Panels & reference**
- [Terminal & git tabs](https://theboringoffice.pages.dev/docs/terminal-and-git-tabs)
- [Browser tab](https://theboringoffice.pages.dev/docs/browser-tab)
- [Layout, themes & power](https://theboringoffice.pages.dev/docs/layout-themes-power)
- [Keys & slash commands](https://theboringoffice.pages.dev/docs/keys-and-slash)

Config file: `~/.theboringoffice/configs/brain.json` (`theboringoffice --print-default-config`). Details: [backends](https://theboringoffice.pages.dev/docs/backends) + [layout](https://theboringoffice.pages.dev/docs/layout-themes-power).

## Peek

<p>
  <img src="docs/shots-go/agents.png" alt="agents tab" width="49%" />
  <img src="docs/shots-go/chat-stream.png" alt="streaming" width="49%" />
</p>

![diffs](docs/shots-go/chat-diff.png)

## Keys

One line per key — the full table lives at [keys & slash commands](https://theboringoffice.pages.dev/docs/keys-and-slash).

| Key | Does |
|---|---|
| `tab` / `shift+tab` / `1..7` | switch the right panel: chat · terminal · agents · board · mail · activity · git |
| `ctrl+b` | flip the left pane: floor ↔ browser |
| `enter` | send to the boss — free-sends into the backlog while it's busy |
| `shift+enter` / `ctrl+j` | newline in the chat input |
| `@` | attach-file picker — type to filter, enter/tab attach |
| `ctrl+v` | paste text — attaches the image instead when the clipboard holds one |
| big paste | chat pastes >20 lines or >2000 chars collapse to a `[pasted N lines · M chars]` chip — one backspace unit, full text sent on submit |
| `/model` · `/session` · `@` | pickers filter as you type — `N/M` badge, esc clears the filter, then closes |
| `y` `a` `n` `esc` | answer a permission prompt — allow once / always / reject / defer |
| `ctrl+q` | arm quit — works everywhere |

Browser tab (the left pane, behind `ctrl+b`):

| Key | Does |
|---|---|
| `↑`/`↓` or `j`/`k` | move the link cursor |
| `o` | open the focused link |
| `e` | edit the URL inline in the location bar — prefilled, enter opens, esc cancels |
| `O` | open the current page in the OS browser |
| `[` / `]` | back / forward, 100-page history ring |
| `r` | reload in place |
| `pgup` / `pgdn` | scroll the body |
| `q` / `esc` | back to the floor |

On kitty/ghostty with Chrome, pages render as headless screenshots — ` shot ` badge, PNGs under `~/.theboringoffice/shots/` — and the boss can screenshot pages for you, snapshot pages to read for itself, and (with your approve-once permission) click, fill and eval on them. Pastes into the terminal tab reach the shell bracketed-paste-wrapped. Everywhere else the browser is text on purpose.

## Community

<p>
  <a href="https://discord.gg/YPDsHVHTVf">
    <img src="https://cdn.simpleicons.org/discord/5865F2" alt="Discord" width="28" height="28" />
  </a>
  &nbsp;
  <a href="https://github.com/theboringhumane/theboringoffice">
    <img src="https://cdn.simpleicons.org/github/181717" alt="GitHub" width="28" height="28" />
  </a>
</p>

**[Join the Discord](https://discord.gg/YPDsHVHTVf)** — floor talk, backends, bugs, shots.

Office memory rides [agentmemory](https://github.com/rohitg00/agentmemory). Install script wires it, or `npm install -g @agentmemory/agentmemory`.

Commits through the office can stamp `Co-authored-by: TheBoringMajdoor` — [scripts](scripts/README.md).

## License

MIT © [theboringhumane](https://github.com/theboringhumane) / theboredteam
