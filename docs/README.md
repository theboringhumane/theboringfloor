# Docs

Canonical manual is the **website**. This folder is the in-repo map: architecture + screenshots + links.

**Site:** [theboringoffice.pages.dev/docs](https://theboringoffice.pages.dev/docs)

## Manual

### Install & setup

| Page | |
|---|---|
| [Getting started](https://theboringoffice.pages.dev/docs/getting-started) | curl, `--demo`, live office, `--session` |

### Core

| Page | |
|---|---|
| [Backends](https://theboringoffice.pages.dev/docs/backends) | `opencode` serve-attach or `claudecode` stream-json |
| [Chat & work threads](https://theboringoffice.pages.dev/docs/chat-and-threads) | streaming replies, nested worker threads |
| [Plan mode](https://theboringoffice.pages.dev/docs/plan-mode) | draft, edit, approve → build |

### Workflow

| Page | |
|---|---|
| [Permissions & questions](https://theboringoffice.pages.dev/docs/permissions-and-questions) | allow once / always / reject |
| [Queue, board & memory](https://theboringoffice.pages.dev/docs/queue-board-memory) | backlog, board sync, ledger |

### Panels & reference

| Page | |
|---|---|
| [Terminal & git tabs](https://theboringoffice.pages.dev/docs/terminal-and-git-tabs) | PTY shell + live git |
| [Browser tab](https://theboringoffice.pages.dev/docs/browser-tab) | text lane · headless screenshots · opt-in zenbu |
| [Layout, themes & power](https://theboringoffice.pages.dev/docs/layout-themes-power) | compact, themes, battery |
| [Keys & slash commands](https://theboringoffice.pages.dev/docs/keys-and-slash) | every binding, every `/` |

## Also on the site

- [Home](https://theboringoffice.pages.dev)
- [Get started (tour)](https://theboringoffice.pages.dev/get-started)
- [Vision](https://theboringoffice.pages.dev/vision)
- [Sounds](https://theboringoffice.pages.dev/sounds)
- [Blog](https://theboringoffice.pages.dev/blog)

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
