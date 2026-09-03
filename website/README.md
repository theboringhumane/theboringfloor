# Website

Marketing site + **docs** for [theboringfloor](https://boringfloor.com).

Next.js (bun) → Cloudflare Pages. Live: **https://boringfloor.com**

## Run

```bash
cd website
bun install
bun run dev
```

```bash
bun run build          # production
bun run preview:cf     # wrangler pages preview of `out`
bun run deploy:prod    # Pages, main
```

## Routes

| Path | |
|---|---|
| [`/`](https://boringfloor.com) | home |
| [`/install.sh`](https://boringfloor.com/install.sh) | 302 → latest `install.sh` on GitHub main |
| [`/install.ps1`](https://boringfloor.com/install.ps1) | 302 → latest `install.ps1` on GitHub main |
| [`/get-started`](https://boringfloor.com/get-started) | install tour |
| [`/docs`](https://boringfloor.com/docs) | manual index |
| [`/vision`](https://boringfloor.com/vision) | why a floor |
| [`/sounds`](https://boringfloor.com/sounds) | office chimes |
| [`/blog`](https://boringfloor.com/blog) | posts in `content/blog/` |

### Docs pages

| Path | |
|---|---|
| [`/docs/getting-started`](https://boringfloor.com/docs/getting-started) | install |
| [`/docs/backends`](https://boringfloor.com/docs/backends) | transports |
| [`/docs/chat-and-threads`](https://boringfloor.com/docs/chat-and-threads) | chat |
| [`/docs/plan-mode`](https://boringfloor.com/docs/plan-mode) | plan |
| [`/docs/permissions-and-questions`](https://boringfloor.com/docs/permissions-and-questions) | gates |
| [`/docs/queue-board-memory`](https://boringfloor.com/docs/queue-board-memory) | queue / board / ledger |
| [`/docs/terminal-and-git-tabs`](https://boringfloor.com/docs/terminal-and-git-tabs) | terminal + git |
| [`/docs/browser-tab`](https://boringfloor.com/docs/browser-tab) | browser |
| [`/docs/layout-themes-power`](https://boringfloor.com/docs/layout-themes-power) | chrome |
| [`/docs/keys-and-slash`](https://boringfloor.com/docs/keys-and-slash) | keys + `/` |

App sources: `app/docs/*/page.tsx`. Site URL: `lib/site.ts`.

## Layout

```
app/            routes
components/     header, footer, home, docs chrome
content/blog/   markdown posts
public/shots/   product + docs stills
public/sounds/  WAV preview files
```

In-repo architecture: [`../docs/architecture.md`](../docs/architecture.md). Hub: [`../docs/README.md`](../docs/README.md).
