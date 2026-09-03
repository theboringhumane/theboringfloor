# cmd/

Go entrypoints. Product binary is `theboringfloor` (Go package still `cmd/theboringoffice`). The installer also links `tbo` / `theboringoffice` as shims.

| Package | Job |
|---|---|
| [`theboringoffice`](theboringoffice/) | TUI — flags, backend spawn/attach, tea program |
| [`headless`](headless/) | backend checks: demo, live spawn, `--prompt`, `--batch-probe` |
| [`uishot`](uishot/) | deterministic UI frames vs a stub backend |
| [`floorshot`](floorshot/) | freeze-frame of the office floor |
| [`termshot`](termshot/) | terminal-panel shots |
| [`soundtest`](soundtest/) / [`soundexport`](soundexport/) | synthesized chimes |
| [`claudestub`](claudestub/) | fake claudecode child for tests |
| [`kittyprobe`](kittyprobe/) | kitty graphics probe |

```bash
go run ./cmd/theboringoffice --demo
go run ./cmd/uishot
go run ./cmd/floorshot
```

Install: [Getting started](https://boringfloor.com/docs/getting-started). Architecture: [`../docs/architecture.md`](../docs/architecture.md).
