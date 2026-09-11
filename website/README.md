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
| [`/docs/workspaces`](https://boringfloor.com/docs/workspaces) | floors, teams, tickets, files, history |
| [`/docs/backends`](https://boringfloor.com/docs/backends) | transports |
| [`/docs/mcp-server`](https://boringfloor.com/docs/mcp-server) | MCP server |
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

## Workspace UI

The homepage workspace tour uses actual app renders with illustrative fixtures from `cmd/workspaceshot`. Optimized lossless assets live in `public/shots/workspaces/`. `/docs/workspaces` covers floors, teams, tickets, files, backends, and storage; `/docs/plan-mode` documents automatic routing and the backend-specific planning limits.

The tour uses GitHub Light with a matching light window frame. The palette gallery on the homepage and theme guide shows every built-in theme, with lazy-loaded thumbnails, full-size previews, keyboard selection, and copyable `/theme` commands. Palette colors come directly from `cmd/uishot --theme-catalog`, rather than a separate website color registry.

To refresh the screenshot assets, install [freeze v0.2.2](https://github.com/charmbracelet/freeze) and the website dependencies, then run:

```bash
node scripts/product-shots.mjs           # themes, workspaces, and documentation
node scripts/product-shots.mjs --gallery # themes and GitHub Light workspaces
node scripts/product-shots.mjs --docs    # documentation only
```

The script compiles the Go fixture tools once, uses temporary app/config homes, renders ANSI through Freeze to SVG, and rasterizes optimized images with Sharp. It records intrinsic dimensions in `lib/shot-sizes.json`; `ProductScreenshot` uses them to reserve space before loading. Theme previews are rendered on a consistent canvas. `scripts/docs-shots.sh` remains a compatibility entry point. All captures are simulated UI examples; they do not make model calls. The documentation renderer uses the current command message flow independently of old integration proofs that assert previous layout geometry.

Validate with `node_modules/.bin/tsc --noEmit` and `bun run build`. The production build also checks TypeScript. Preview the exported `out/` directory with a local static server.

## Cobalt design system

The marketing site uses an editorial grid, warm off-white surfaces, and cobalt blue. The shared palette is in `app/globals.css`; responsive layouts and component styles are in `app/blueprint.css`. `PageCover`, `SiteHeader`, and `SiteFooter` keep the content routes consistent.

All geometric illustrations are original SVGs in `components/home/blueprint-art.tsx`. The homepage office is an original Three.js scene loaded separately from the initial page, with an SVG fallback when WebGL is unavailable. Stationary geometry is merged by material. Rendering is capped at 30 fps, stops offscreen or in background tabs, and respects reduced motion. The user can pause or rotate the scene. GSAP handles entrance and scroll motion; content remains readable without animation.

The hero and `/sounds` offer an optional office soundscape: quiet ventilation, typing, printer movement, and a coffee machine. `lib/office-audio.ts` synthesizes it locally with Web Audio. Audio starts only after a click, has a low default level and a volume control, fades out when the scene leaves view or the tab is hidden, and disposes its timers and audio graph on navigation. The seven original downloadable notification chimes remain on `/sounds`, now linked in the main navigation.

The product explorer uses real application captures with illustrative project data. Every view links to its corresponding guide and full-size screenshot. The journal supports search and category filtering. Install commands support macOS, Linux, and Windows, with clipboard feedback.

```bash
bun run check         # TypeScript
bun run build         # all static routes and share images
bun run check:links   # validate exported local links, anchors, and assets
```

Cloudflare Pages serves the existing installer redirects in `public/_redirects`. Local static preview does not interpret those redirects; the install commands point to the production URLs. Google Analytics remains configured for production. The Vercel-only analytics injection was removed because this site is hosted on Cloudflare Pages.
