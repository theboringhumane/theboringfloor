import type { Metadata } from 'next'
import { DocsArticle, type DocSection } from '@/components/docs-article'

export const metadata: Metadata = {
  "title": "A floor for every project",
  "description": "Projects, teams, tickets, source files, and conversations share one workspace. Floors stay on the far left, the office sits in the middle, and transcript and tools live on the right.",
  "alternates": {
    "canonical": "/docs/workspaces"
  }
}

const sections: DocSection[] = [
  {
    "id": "layout",
    "title": "Three panes, one workspace",
    "shot": "transcript",
    "alt": "Project floors, the office, and transcript arranged from left to right.",
    "body": "At 100 columns and wider, the project navigator stays on the far left. The middle pane shows the office, browser, or presented plan. The right pane has eight tabs: **Chat, Terminal, Agents, Board, Mail, Activity, Git, and Files**.\n\nUse **Ctrl+E** to focus floors; Escape or Tab returns to the tools. **Ctrl+W** expands the tools while keeping the floor navigator. In narrow terminals, floors open as a drawer instead of squeezing the conversation."
  },
  {
    "id": "projects",
    "title": "Projects and teams",
    "shot": "floors",
    "body": "A floor is a project directory. Open it with `theboringfloor --project /path/to/project`, or use the floor navigator to open a project. Each starts with **UI, Coding, Frontend, and Backend** teams. Add custom teams for your workflow.\n\nTeams organize tickets and provide conversation context; they do not create a fixed pool of running agents. Worker creation and reporting depend on the selected backend. Switching projects or conversations waits until the active work is stopped or finished."
  },
  {
    "id": "conversations",
    "title": "Choose a backend per conversation",
    "shot": "new-conversation",
    "body": "Press **Ctrl+N** to start a conversation on the current floor. Set its title, choose a team, and select **OpenCode**, **Claude Code**, or **Codex**. In the floor navigator, `n` starts a conversation on the selected project.\n\nA saved conversation resumes on its original backend; choosing a different backend creates a separate conversation. Install and log into the corresponding CLI first. Codex uses your saved CLI login and model settings.\n\n```sh\ncodex login\ntheboringfloor --project /path/to/project --backend codex --new\n```\n\nSee [Backends](/docs/backends) for transport-specific capabilities."
  },
  {
    "id": "tickets",
    "title": "Tickets that outlive a session",
    "shot": "board",
    "body": "The Board stores project tickets locally. Create a ticket, assign a team and owner, set priority **P0–P3**, and track it through **Backlog → In progress → Blocked → Review → Done**. Each ticket supports a description and checklist.\n\nUse the board controls to create and edit tickets, move them between statuses, and inspect details. Live agent task rows remain visible as read-only activity. Manual tickets and teams live in `floor.json`, independently of whichever conversation is active."
  },
  {
    "id": "files",
    "title": "Explore and attach project files",
    "shot": "files",
    "body": "Open **Files** to browse the project. Expand a folder with Enter, select source to preview it with line numbers, and press `a` to attach a file to the composer. Folders load on demand.\n\nPreviews are limited to 256 KiB to keep navigation responsive. Files and symlinks must resolve inside the project boundary. Attachment limits vary by backend: Codex accepts images and text files up to 1 MiB each; Claude Code receives file paths."
  },
  {
    "id": "history",
    "title": "Fast startup, durable history",
    "body": "**Ctrl+R** searches the loaded transcript without losing your draft. Conversations have separate local archives, indexed by metadata so the floor list does not parse every transcript.\n\nStorage lives under `~/.theboringfloor/projects/<project-hash>/` (or the configured office home), rather than adding session files to the repository:\n\n- `floor.json`: project teams and manual tickets.\n- `session.json`: a fast startup snapshot of up to 200 recent messages.\n- `conversations/<backend-and-session-hash>/session.json`: up to 10,000 archived messages per conversation.\n- `conversations/<backend-and-session-hash>/meta.json`: title, team, backend, and activity metadata.\n\nThe outgoing conversation is archived before a new one starts. Writes are serialized and use atomic replacement; unchanged snapshots skip disk writes. Older remote-history paging depends on backend support. MCP transcript tools currently read the 200-message project snapshot, not every archive."
  },
  {
    "id": "planning",
    "title": "Plan substantial work first",
    "shot": "plan",
    "body": "Clearly substantial implementation requests enter planning automatically before they are sent. The boss also assesses scope and must plan major features, migrations, and changes spanning multiple layers before implementation.\n\n`plan_present` and `plan_update` open the plan view, including when you were in zen, a worker thread, or expanded tools. **Ctrl+X twice** approves the draft and sends it for implementation. **Ctrl+P** switches modes; explicitly returning to build skips automatic planning for the next request. See [Plan mode](/docs/plan-mode) for the exact behavior and backend limits."
  },
  {
    "id": "browser",
    "title": "A simpler browser setup",
    "body": "The external **terminal-browser** package is removed from the installer and runtime. Old opt-in variables cannot enable it. The built-in text viewer and headless screenshot support remain; external links open in your system browser. See [Browser](/docs/browser-tab)."
  }
]

export default function Page() { return <DocsArticle title="A floor for every project" intro="Projects, teams, tickets, source files, and conversations share one workspace. Floors stay on the far left, the office sits in the middle, and transcript and tools live on the right." sections={sections} /> }
