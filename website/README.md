# Website

Marketing site + **docs** for [theboringoffice](https://theboringoffice.pages.dev).

Next.js (bun) → Cloudflare Pages. Live: **https://theboringoffice.pages.dev**

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
| [`/`](https://theboringoffice.pages.dev) | home |
| [`/get-started`](https://theboringoffice.pages.dev/get-started) | install tour |
| [`/docs`](https://theboringoffice.pages.dev/docs) | manual index |
| [`/vision`](https://theboringoffice.pages.dev/vision) | why a floor |
| [`/sounds`](https://theboringoffice.pages.dev/sounds) | office chimes |
| [`/blog`](https://theboringoffice.pages.dev/blog) | posts in `content/blog/` |

### Docs pages

| Path | |
|---|---|
| [`/docs/getting-started`](https://theboringoffice.pages.dev/docs/getting-started) | install |
| [`/docs/backends`](https://theboringoffice.pages.dev/docs/backends) | transports |
| [`/docs/chat-and-threads`](https://theboringoffice.pages.dev/docs/chat-and-threads) | chat |
| [`/docs/plan-mode`](https://theboringoffice.pages.dev/docs/plan-mode) | plan |
| [`/docs/permissions-and-questions`](https://theboringoffice.pages.dev/docs/permissions-and-questions) | gates |
| [`/docs/queue-board-memory`](https://theboringoffice.pages.dev/docs/queue-board-memory) | queue / board / ledger |
| [`/docs/terminal-and-git-tabs`](https://theboringoffice.pages.dev/docs/terminal-and-git-tabs) | terminal + git |
| [`/docs/browser-tab`](https://theboringoffice.pages.dev/docs/browser-tab) | browser |
| [`/docs/layout-themes-power`](https://theboringoffice.pages.dev/docs/layout-themes-power) | chrome |
| [`/docs/keys-and-slash`](https://theboringoffice.pages.dev/docs/keys-and-slash) | keys + `/` |

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
