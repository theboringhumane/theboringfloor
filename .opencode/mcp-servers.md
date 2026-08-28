# Available MCP servers

The opencode serve backing this session has the MCP servers
below configured. Their tools are callable as `<server>_<tool>`
(the serve namespaces every MCP tool with its server name).

- `agentmemory` (local) — configured in global opencode config
- `arize-phoenix` (local) — configured in global opencode config
- `composio` (remote) — configured in global opencode config
- `mcp-server-firecrawl` (local) — configured in global opencode config
- `synehq-kole` (local) — configured in global opencode config
- `watch-skill` (local) — configured in global opencode config
- `workspace-mcp` (local) — configured in global opencode config

Only names, types, and provenance are listed — never commands,
URLs, or environment: this file lands in a prompt, and prompts are
no place for API keys. When a dispatch needs one of these servers,
name it in the dispatch brief's CONTEXT so the developer knows the
tools exist; the developer discovers the exact tool names live.

Usage discipline:
- Prefer a purpose-built MCP tool over a shell workaround when one
  covers the job (memory recall, web fetch, tracing) — but never
  guess a tool name: list or probe the server's tools first.
- A failing or absent MCP tool is an ISSUE to report, not a reason
  to fake its output.
- Servers listed here may still be down at call time; treat the
  first call as the health check.
