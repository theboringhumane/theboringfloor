# Docs

Canonical manual is the **website**. This folder is the in-repo map: architecture + screenshots + links.

**Site:** [theboringfloor.pages.dev/docs](https://boringfloor.com/docs)

## Manual

### Install & setup

| Page | |
|---|---|
| [Getting started](https://boringfloor.com/docs/getting-started) | curl, `theboringfloor --demo`, live office, `--session` |

### Core

| Page | |
|---|---|
| [Backends](https://boringfloor.com/docs/backends) | `opencode` serve-attach or `claudecode` stream-json |
| [Chat & work threads](https://boringfloor.com/docs/chat-and-threads) | streaming replies, nested worker threads |
| [Plan mode](https://boringfloor.com/docs/plan-mode) | draft, edit, approve → build |

### Workflow

| Page | |
|---|---|
| [Permissions & questions](https://boringfloor.com/docs/permissions-and-questions) | allow once / always / reject |
| [Queue, board & memory](https://boringfloor.com/docs/queue-board-memory) | backlog, board sync, ledger |

### Panels & reference

| Page | |
|---|---|
| [Terminal & git tabs](https://boringfloor.com/docs/terminal-and-git-tabs) | PTY shell + live git |
| [Browser tab](https://boringfloor.com/docs/browser-tab) | text lane · headless screenshots · opt-in zenbu |
| [Layout, themes & power](https://boringfloor.com/docs/layout-themes-power) | compact, themes, battery |
| [Keys & slash commands](https://boringfloor.com/docs/keys-and-slash) | every binding, every `/` |

## Also on the site

- [Home](https://boringfloor.com)
- [Get started (tour)](https://boringfloor.com/get-started)
- [Vision](https://boringfloor.com/vision)
- [Sounds](https://boringfloor.com/sounds)
- [Blog](https://boringfloor.com/blog)

## In this folder

| Path | |
|---|---|
| [architecture.md](architecture.md) | floor plan, packages, event → sprite |
| `shots-go/` | TUI stills (`go run ./cmd/uishot`, `go run ./cmd/floorshot` + [freeze](https://github.com/charmbracelet/freeze)) |
| `shots/` | older floor SVGs |

## Repo map

- [Root README](../README.md)
- [Website](../website/README.md)
- [cmd/](../cmd/README.md)
- [scripts/](../scripts/README.md)
