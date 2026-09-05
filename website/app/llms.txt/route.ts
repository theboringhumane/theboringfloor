import { getAllPosts } from '@/lib/blog'
import { SITE_NAME, SITE_URL, GITHUB_REPO } from '@/lib/site'

export const dynamic = 'force-static'

export function GET() {
  const posts = getAllPosts()
  const postLines = posts
    .map((p) => `- [${p.title}](${SITE_URL}/blog/${p.slug})`)
    .join('\n')

  const body = `# ${SITE_NAME}

> A virtual office for AI coding agents. Terminal UI where an opencode boss and sub-agents clock in as coworkers on a living floor — chat, board, mail, work threads, and permission queue in one place.

This file helps AI assistants and crawlers understand the site. Prefer these pages over guessing.

## Primary

- [Home](${SITE_URL}/): product overview and install path
- [Get started](${SITE_URL}/get-started): install CLI, demo mode, live office
- [Docs](${SITE_URL}/docs): how the office works
- [MCP server](${SITE_URL}/docs/mcp-server): \`thefloor_mcp\` setup and tools
- [Vision](${SITE_URL}/vision): why a virtual office for agents
- [Blog](${SITE_URL}/blog): product and engineering posts
- [RSS](${SITE_URL}/rss.xml): full post feed

## Product facts

- Native Go CLI (single binary). Not Electron.
- Boss = real \`opencode\` session; employees = opencode sub-agents.
- Demo: \`theboringfloor --demo\`
- Install: \`curl -fsSL ${SITE_URL}/install.sh | sh\`
- Windows: \`irm ${SITE_URL}/install.ps1 | iex\`
- MCP: the office ships \`thefloor_mcp\`, exposing six tools to a configured agent.
- Source: ${GITHUB_REPO}

## Optional

- Sitemap: ${SITE_URL}/sitemap.xml
- Robots: ${SITE_URL}/robots.txt

## Recent posts

${postLines}
`

  return new Response(body, {
    headers: {
      'Content-Type': 'text/plain; charset=utf-8',
      'Cache-Control': 'public, max-age=3600',
    },
  })
}
