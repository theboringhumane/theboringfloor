// uishot — deterministic UI shot harness for theboringoffice v2.
//
// Runs the REAL app model against a scripted stub backend (fixed event
// script: hires/dispatches/working/returned+mail/blocked/bubbles, boss
// EvThought + EvTool chains, two chat rounds — one boss reply contains
// markdown — plus the deep-work stream: EvQuestion, EvFileDiff, EvPermission
// for both boss and a child employee, and a rapid triple-send typed into the
// textarea while the boss reply is pending). Fixed 130x32, ~4s, then prints
// the final frame between ===== UI SHOT ===== markers.
//
//	go run ./cmd/uishot [--tab chat|agents|board|mail|activity]
//	                    [--theme noir|paper|mono|dracula|solarized]
//	                    [--slash]   (also simulates typing /theme + /themes)
//	                    [--perm]    (auto-answers the boss permission "once" at 3s)
//	                    [--diffs]   (expands all diff entries via ctrl+d)
//	                    [--debug]   (queue flush proof: resolves the pending boss,
//	                                prints [queue] trace lines, longer window)
//	                    [--think]   (think-stream proof: one CallID streamed in
//	                                accumulated updates, then collapsed after
//	                                Done — prints BOTH frames: mid-stream at
//	                                t=2.0s and collapsed at t=3.2s)
//	                    [--think-stop mid|done] (with --think: print just ONE
//	                                frame — mid = streaming, done = collapsed —
//	                                for the gallery freeze shot)
//	                    [--stream]  (chat-stream proof: one "bossmsg-m1" bubble
//	                                streamed as 5 ACCUMULATED pending updates
//	                                300ms apart, then the pinned final; prints
//	                                frame mid-stream (partial bubble growing in
//	                                the viewport, typing row below the divider)
//	                                and after done (one single settled bubble —
//	                                replace-in-place, no dup).
//	                                A message is typed mid-stream to prove
//	                                FREE-QUEUING: it goes straight to the
//	                                backend while the stream is live — the
//	                                ordering trace prints Send/done + the
//	                                "busy · N queued (server)" compose.)
//	                    [--ask-answer] (question-hold proof: boss EvQuestion q-1
//	                                opens the answer modal at 1.5s (parked turn —
//	                                typing placeholder removed); typing "the
//	                                toggle one" + enter at 2.5s must hit
//	                                AnswerQuestion, the entry gains a dim
//	                                "✓ answered", the resumed boss reply closes
//	                                the turn. Prints BOTH frames + the capture
//	                                log; an employee question stays activity-only.)
//	                    [--ask-esc]   (esc defers the hold with a notice,
//	                                /question re-opens it, answer still via
//	                                AnswerQuestion)
//	                    [--ask-queue] (queue-hold proof: a line typed while the
//	                                hold is outstanding ENQUEUES; flush fires
//	                                only after resolved + completed boss reply —
//	                                ordering trace prints it)
//	                    [--batch]    (intelligent-backlog proof: boss busy ~3s
//	                                while three messages enqueue as backlog
//	                                #1 #2 #3; the turn-complete flush must be
//	                                ONE composed [BATCH DISPATCH] send. Prints
//	                                the frame, the composed batch text, the
//	                                stub Send/QueueItemStart/Done logs and the
//	                                ordering trace)
//	                    [--batch-respawn] (failure-respawn proof: the stub
//	                                rejects the first batch Send — the app
//	                                must ResetPrimary(true) and resend the
//	                                SAME batch once; second send succeeds)
//	                    [--power auto|saver|performance|all]
//	                                power-governor proof: the model runs in a
//	                                manual event loop (every update renders a
//	                                frame — that is what the caches count) for
//	                                a 6s scripted window per mode; prints tick
//	                                counts (performance > auto > saver), the
//	                                floor frame-cache hit %, the TickDelay
//	                                decision table (busy/idle/drift + tickMs
//	                                override), the /power + /model slash demo
//	                                (chat frame + persisted brain.json), and a
//	                                custom boss-name agents frame.
//	                    [--social]  (social-clock proof: scripted window pumping
//	                                EvTicks synchronously (no wall clock — tick-
//	                                seeded = deterministic). THREE frames:
//	                                SOCIAL A = tea request asked (bubble «<B>:
//	                                coffee?»), SOCIAL B = both sprites walking
//	                                to the machine, SOCIAL C = gossip chain
//	                                mid-fire; plus the banter chain trace, the
//	                                question-modal gate assert (nothing fires
//	                                while a modal is open), and a two-run
//	                                determinism check over the frame triplet.)
//	                    [--layout]  (layout-modes proof: THREE frames over the
//	                                same scripted window — NORMAL (sidebar 80),
//	                                compact (sidebar 30, short tab labels, 2-row
//	                                chat input, compressed topbar) and wide 90 —
//	                                each with its computed width asserts)
//	                    [--planshot] (plan-mode screenshot, conversation-first
//	                                + the presentation shape gate: ctrl+p flips
//	                                ONLY the mode (chat keeps focus, pane
//	                                hidden while empty); TWO boss CHATTER
//	                                replies (status narration) never present —
//	                                the pane stays hidden, the escape-valve
//	                                note fires ONCE; the plan-SHAPED reply
//	                                (# Goal / # Steps) then mirrors passively
//	                                into the pane — TWO frames: t=2.0s floor
//	                                kept + [plan] badge + idle hint + valve
//	                                note; t=3.6s PLAN · markdown pane carrying
//	                                the boss's text, floor swapped out, chat
//	                                input still owns focus)
//	                    [--terminal] (terminal-tab proof: the stub TermPanel
//	                                (terminal_panel_stub.go — uishot ONLY)
//	                                wires through app.SpawnTerminal; selecting
//	                                the "terminal" tab lazy-spawns it. Wave-42
//	                                OPT-IN capture flow: RELEASED default
//	                                (typed letters never reach the shell,
//	                                a real TAB key event leaves the tab),
//	                                ctrl+space TOGGLES capture BOTH ways
//	                                (wave-41: letters/tab/shift+tab/digits
//	                                all reach the shell), ctrl+o releases
//	                                IN PLACE as the alias, leaving while
//	                                captured auto-releases and re-entry
//	                                starts RELEASED — frames + asserts, and
//	                                CloseTerminal kills it. Deterministic:
//	                                the synchronous drive runs twice and
//	                                must produce byte-identical frames)
//	                    [--stop]    (/stop proof: typed mid-stream — captures
//	                                AbortSessions, then ONE unwound frame:
//	                                "stopped by user" placeholder, " (stopped)"
//	                                stream appendix, tools ✗ aborted, thread
//	                                (· N tool calls ✗ stopped), queue intact)
//	                    [--bypass]  (/bypass proof (synchronous): arming
//	                                confirm → ⚠ BYPASS topbar segment + ON
//	                                notice → stray EvPermission auto-answered
//	                                allow-once (no modal) → instant OFF;
//	                                two drives byte-identical)
//	                    [--stuck]   (boss-stuck-busy proof (synchronous):
//	                                the boss goes busy at 200ms and NEVER
//	                                completes; the W1 watchdog (harness-seamed
//	                                to a 30ms threshold) fires its ONE red
//	                                wedge row + hint swap; then the /stop leg
//	                                runs against an AbortSessions stubbed to
//	                                FAIL — the office still unwinds (G1))
//	                    [--freesend] (free-queuing proof: boss busy 3s, two
//	                                prompts sent DURING — both Send() calls
//	                                land immediately (trace: both BEFORE the
//	                                turn-completed marker), statusline "busy ·
//	                                2 queued (server)", FIFO drain after)
//	                    [--concierge] (concierge routing proof: busy sends
//	                                route to SendConcierge (captured, notice
//	                                ONCE, office placeholder+bubbles, agents
//	                                row answering|on call); idle boss picks
//	                                Send again — zero duplication)
//	                    [--notifications] (OS desktop notification proof
//	                                (synchronous): a recording NotifyBus at the
//	                                app's seam — focused startup stays silent
//	                                (default true: unsupported terminals never
//	                                false-ping); BLUR opens the window; ONE
//	                                cohort ping for the boss ask (the child ask
//	                                coalesces, generic "needed — <agent> needs
//	                                <tool>" copy, no ToolSummary leak); answering
//	                                the front shrinks the cohort but keeps it
//	                                silent; the boss's completion pings ONCE
//	                                ("the boss is done — <clipped reply>");
//	                                refocus = silence; re-blur on the live cohort
//	                                re-nudges at once; /notify off → zero
//	                                captures + persisted brain.json proof)
//	                    [--boardsync] (completion board-sync proof (synchronous):
//	                                THREE DOING rows staged — tekton-1 twice
//	                                (older/newer), skopos-1 once sharing a
//	                                title with a LATER distinct-owner return —
//	                                then TWO returns drive it the way the live
//	                                backend emits them (boardsync.go): return 1
//	                                closes its exact row + the sweep flips
//	                                tekton-1's OLDEST row, ONE "[office] board
//	                                sync: flipped 1 rows to done" note; return 2
//	                                (tekton-2, distinct owner) closes only its
//	                                own row — skopos-1's lookalike NEVER flips.
//	                                Frame A 3 doing / frame B done counts +
//	                                note; two drives byte-identical)
//	                    [--images]  (inbound boss-turn image preview proof
//	                                (synchronous): the stub pins a completed
//	                                boss turn carrying the 8×8 checker PNG as
//	                                a data-URL file part (Meta carrier +
//	                                Event.Media payload); the lazy rasterize
//	                                lands the 🖼 chip + 4 pinned half-block
//	                                truecolor rows in the frame; the /images
//	                                off leg paints chips only; two drives
//	                                byte-identical. --lane kitty proves the
//	                                wave-86 routing: the media rows are
//	                                APC-free and the wrapper splices DECSC +
//	                                CUP(the pinned absolute cell) + the
//	                                f=100 APC + DECRC after a flush)
//	                    [--images-pty] (the PRODUCTION-path twin of the
//	                                kitty leg: the REAL cursed renderer over
//	                                the ZenbuFrameWriter on os.Stdout
//	                                (main.go's exact wiring), the checker
//	                                pin through p.Send — the PTY harness
//	                                (/tmp/drive_chatimage.py) counts the
//	                                ESC_G f=100 frames on the wire)
//	                    [--links]   (open-in-browser proof (synchronous): a
//	                                boss bubble wears the · o (open) beacon; a
//	                                press marks it, `o` floats the OPEN IN
//	                                BROWSER card over BOTH verified targets
//	                                (URL + on-disk media filename), enter
//	                                fires the URL through the STUBBED runner;
//	                                activity logs "→ opened: opencode.ai/docs";
//	                                the no-mark leg types "o"; byte-identical)
//	                    [--openurl] (terminal-browser candidate-lane proof
//	                                (synchronous, REAL fake binaries): a
//	                                scratch fixture dir carries a logging
//	                                `terminal-browser` (+ `open`/`xdg-open`)
//	                                on a pinned PATH + hermetic ghostty env;
//	                                leg A resolves terminal-browser and a
//	                                press+`o` logs its call (system log
//	                                EMPTY); leg B (fake exits 1) cascades the
//	                                SAME URL to the system opener — one
//	                                attempt each, "→ opened:" intact;
//	                                byte-identical twice)
//	                    [--browser --lane kitty] (browser tab premium-lane
//	                                proof (synchronous, REAL fake binary):
//	                                leg A embeds the fake zenbu child on the
//	                                real PTY seam — its bytes paint the grid,
//	                                the frame wears the " zenbu " badge +
//	                                "▸ zenbu terminal-browser · <url>" strip,
//	                                Close group-kills + reaps; leg B (dead at
//	                                immediately, ~180ms < 300ms) falls back
//	                                to text mode
//	                                with the exact dim note, URL state
//	                                intact; byte-identical twice)
//	                    [--browser --lane keepalive] (the freeze/thaw flip
//	                                cycle (synchronous, REAL fake binary):
//	                                /open → ctrl+b (floor — the child
//	                                FREEZES: PID stable + alive + ps T…,
//	                                ONE a=d through the wrapper's diff) →
//	                                ctrl+b (the SAME pid thaws; the
//	                                RETAINED frame re-emits byte-
//	                                identically, ZERO new child bytes) →
//	                                ctrl+b → ctrl+c (the quit path reaps
//	                                the frozen child, the delete riding
//	                                the direct seam); ONE spawn total;
//	                                byte-identical twice)
//	                    [--browser --lane hint] (the text lane's "why" row
//	                                (synchronous, hermetic): PATH pinned to
//	                                an EMPTY fixture dir — the probe misses
//	                                by construction — so the starter card
//	                                wears the dim "text lane — terminal-
//	                                browser not on PATH · …" hint under the
//	                                location bar and /open keeps it pinned
//	                                over the warm text page; two drives
//	                                byte-identical)
//	                    [--browsertab] (browser TAB text-viewer proof
//	                                (synchronous, REAL pinned-port stub
//	                                server): /open <stub>/fixture.html typed
//	                                through the REAL chat input + slash
//	                                popover renders the fixture as text rows
//	                                — "▸ <url> · Fixture Gazette" bar, bold
//	                                headings, "link alpha [1]"/"link beta
//	                                [2]"/"link gamma [3]" rows, the 🖼 chip,
//	                                the " │ " table — pgdn scrolls the
//	                                tail-marker row into view; two drives
//	                                byte-identical)
package main

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/backend"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/headless"
	"github.com/theboringhumane/theboringfloor/internal/office"
	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	shotCols    = 130
	shotRows    = 32
	shotDur     = 4000 * time.Millisecond
	shotDurLong = 6500 * time.Millisecond // --debug: drain the whole queue chain
	defaultTab  = "agents"
	bossReplyMD = "**Done** — created `hello.html`:\n" +
		"- dark navy bg\n" +
		"- white text\n" +
		"```sh\n" +
		"echo done\n" +
		"```"
)

// bossDiffBody — a markdown-heavy README.md diff close to the reference
// panel: **bold** and `code` spans inside tinted rows (proves chroma paints
// md on top of the tints), realistic @@ old/new numbers, and >30 rows total
// so the expanded "+N more" clip still fires.
const bossDiffBody = "--- a/README.md\n" +
	"+++ b/README.md\n" +
	"@@ -40,6 +40,7 @@ renders **bold**, *italic*, `code` and lists.\n" +
	" \n" +
	" ## What you get\n" +
	"   format and wrap inside the panel instead of bleeding through the UI.\n" +
	" \n" +
	"- **Bug:** Scrolling everywhere (viewport), mouse wheel, multi-line input,\n" +
	"   typing spinner while the `boss` works.\n" +
	"- Native single binary. The **Ink/Node** v0.1 app is preserved under\n" +
	"   [`node-legacy/`](node-legacy/) (tagged `node-v0.1.0`).\n" +
	"+ Native single binary. Themes: `--theme noir|paper|mono|dracula`\n" +
	"+ (also `/theme` in-app, persisted to `~/.config/theboringoffice/theme`).\n" +
	" \n" +
	"   ## Behind the glass\n" +
	"@@ -49,6 +50,10 @@ renders **bold**, *italic*, `code` and lists.\n" +
	"   Boss turns stream as markdown; user turns wrap plain.\n" +
	"- ~90 MB of RAM idle, **instant** startup.\n" +
	"+ ~12 MB of RAM idle, **instant** startup.\n" +
	"- See [docs/](docs/) for the tour.\n" +
	"+ See [docs/](docs/) for the full tour, [AGENTS.md](AGENTS.md) for rules.\n" +
	" \n" +
	"   ### Run it\n" +
	"-```sh\n" +
	"-cd theboringoffice && go build ./...\n" +
	"-```\n" +
	"+```sh\n" +
	"+go run ./cmd/theboringoffice --theme dracula\n" +
	"+theboringoffice --theme paper\n" +
	"+```\n" +
	" \n" +
	" | Key | Action |\n" +
	" |-----|--------|\n" +
	"-| `ctrl+t` | toggle thinking |\n" +
	"+| `ctrl+t` | toggle thinking blocks |\n" +
	"+| `ctrl+d` | toggle expanded diffs |\n" +
	"+| `/diffs off` | collapse all diffs |\n" +
	" \n" +
	" ## Layout\n" +
	" \n" +
	"-- side: chat with the boss\n" +
	"- floor: the office\n" +
	"-```\n" +
	"-┌──────┐\n" +
	"-│ desk │\n" +
	"-└──────┘\n" +
	"-```\n" +
	"+ sidebar: chat with the boss (**all** history)\n" +
	"+ floor: the office grid, mail, board, activity tabs\n" +
	"+```text\n" +
	"+┌────────┬────────┐\n" +
	"+│  chat  │ floor  │\n" +
	"+└────────┴────────┘\n" +
	"+```"

// employeeDiffBody — a brand-new Go file (--- /dev/null) so the expanded
// header reads "← New file …" and all rows are additions with Go syntax.
const employeeDiffBody = "--- /dev/null\n" +
	"+++ b/src/main.go\n" +
	"@@ -0,0 +1,5 @@\n" +
	"+package main\n" +
	"+\n" +
	"+func main() {\n" +
	"+\tprintln(\"hello, theboringoffice\")\n" +
	"+run()\n" +
	"+}"

// stubBackend is the deterministic scripted backend for the shot.
type stubBackend struct {
	emit       func(state.Event)
	done       chan struct{}
	start      time.Time
	flushQueue bool              // --debug: script resolves the round-2 pending boss
	thinkMode  bool              // --think: script streams one think CallID instead
	streamMode bool              // --stream: script streams one "bossmsg-" reply instead
	askMode    string            // --ask-*: "" | "answer" | "esc" | "queue" (question-hold proof)
	permAnswer string            // recorded by AnswerPermission for the final print
	sendSeq    int               // unique reply IDs per Send (replace-by-ID safety)
	answerLog  []string          // recorded by AnswerQuestion/RejectQuestion (the capture proof)
	trace      func(line string) // --stream/--ask-*: ordering-trace sink

	batchMode       bool     // --batch/--batch-respawn: backlog-batch proof script
	respawnMode     bool     // --batch-respawn: reject the first batch Send once
	batchFailedOnce bool     // the one-shot rejection sentinel
	sendLog         []string // every Send call, verbatim (the proof)
	teamLog         []string // QueueItemStart/Done + ResetPrimary calls (the proof)

	freeMode bool // --freesend: boss busy ~3s, two sends land mid-turn

	abortLog []string // AbortSessions capture (the --stop proof)
	abortErr error    // --stuck: AbortSessions returns this error (the G1 leg)

	powerDemo bool // --power: minimal quiet script for the slash/name legs
	planDemo  bool // --planshot: minimal quiet script (no floats) for the plan-mode shot
}

func mail(id, from, to, subject, body string, kind state.MailKind) state.MailItem {
	return state.MailItem{ID: id, From: from, To: to, At: time.Now().UnixMilli(),
		Subject: subject, Body: body, Kind: kind}
}

func chatMsg(id, from, text string, pending bool) state.ChatMsg {
	return state.ChatMsg{ID: id, From: from, Kind: from, Text: text,
		At: time.Now().UnixMilli(), Pending: pending}
}

// script — fixed ABSOLUTE times (ms from start), fixed payloads;
// deterministic given the same clock. Ends ~3.3s into the ~4s window
// (~3.6s with flushQueue, inside the ~6.5s --debug window).
func (b *stubBackend) script() {
	start := time.Now()
	at := func(ms int, ev state.Event) {
		if d := time.Until(start.Add(time.Duration(ms) * time.Millisecond)); d > 0 {
			time.Sleep(d)
		}
		b.emit(ev)
	}
	if b.thinkMode {
		b.scriptThink(at)
		return
	}
	if b.streamMode {
		b.scriptStream(at)
		return
	}
	if b.askMode != "" {
		b.scriptAsk(at)
		return
	}
	if b.batchMode {
		b.scriptBatch(at)
		return
	}
	if b.freeMode {
		b.scriptFree(at)
		return
	}
	if b.powerDemo {
		b.scriptPowerDemo(at)
		return
	}
	if b.planDemo {
		b.scriptPlanDemo(at)
		return
	}

	at(50, state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — stub backend online"})
	at(100, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	at(250, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteAtDesk}})
	at(400, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "rev-1", Name: "dikastes", Role: state.RoleReviewer, Sprite: state.SpriteAtDesk}})

	// round 1: user asks, boss THINKS (visible, collapsed by default), boss
	// answers with markdown — with a boss tool chain merging running → done.
	at(550, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"make hello.html — dark navy, white text", false)})
	at(600, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	at(650, state.Event{Kind: state.EvThought, EmployeeID: "boss", EmployeeName: "boss",
		Text: "single file, no build step.\ndark navy bg, white text — keep it simple.", Done: false})
	at(720, state.Event{Kind: state.EvThought, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		Text: "activity-line-only thought (employee — no chat entry)", Done: false})
	at(780, state.Event{Kind: state.EvTool, EmployeeID: "boss", EmployeeName: "boss",
		ToolName: "write", ToolSummary: "hello.html", ToolState: "running", CallID: "call-1"})
	at(900, state.Event{Kind: state.EvTool, EmployeeID: "boss", EmployeeName: "boss",
		ToolName: "write", ToolSummary: "hello.html", ToolState: "done", CallID: "call-1"})
	at(1000, state.Event{Kind: state.EvThought, EmployeeID: "boss", EmployeeName: "boss",
		Text: "deck the reply with a list and a code fence so markdown shows.", Done: true})
	at(1080, state.Event{Kind: state.EvDispatch, EmployeeID: "dev-1",
		Task: state.BoardTask{ID: "t1", Title: "build hello.html", At: time.Now().UnixMilli()}})
	at(1150, state.Event{Kind: state.EvWorking, EmployeeID: "dev-1", TaskID: "t1"})
	at(1200, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("b1", "boss", bossReplyMD, false)})

	at(1400, state.Event{Kind: state.EvDispatch, EmployeeID: "sco-1",
		Task: state.BoardTask{ID: "t2", Title: "scan the repo", At: time.Now().UnixMilli()}})
	// employee tool chain: running → done for a non-boss employee
	at(1500, state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		ToolName: "read", ToolSummary: "src/main.go", ToolState: "running", CallID: "call-2"})
	at(1600, state.Event{Kind: state.EvBubble, EmployeeID: "dev-1",
		Text: "this diff is a crime scene.", TTL: 40})
	at(1800, state.Event{Kind: state.EvWorking, EmployeeID: "sco-1", TaskID: "t2"})
	at(1900, state.Event{Kind: state.EvTool, EmployeeID: "dev-1", EmployeeName: "tekton-1",
		ToolName: "read", ToolSummary: "src/main.go", ToolState: "done", CallID: "call-2"})

	// deep-work stream: boss question (chat entry, yellow) + an employee
	// question (activity line ONLY — no chat entry), then diffs for both.
	// The boss question opens the answer modal — resolve it quickly after
	// the entry has landed so LATER scripted workloads (--slash typing at
	// ~1950ms, --perm at 3s, the queue typing at ~3060ms) are not
	// swallowed by the modal (the entry keeps the dim "✓ answered").
	at(1920, state.Event{Kind: state.EvQuestion, EmployeeName: "boss", QuestionID: "q-1",
		Text:        "Which DB should the leaderboard use?",
		ToolSummary: "postgres | sqlite | keep it in memory"})
	at(1935, state.Event{Kind: state.EvQuestion, EmployeeName: "boss", QuestionID: "q-1",
		ToolSummary: "answered", ToolState: "resolved"})
	// the resumed server finishes the parked turn with a completed reply
	// (the contract that unblocks the parked queue path again)
	at(1975, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("b2", "boss",
		"sqlite it is — local, zero setup, fits the leaderboard.", false)})
	at(1960, state.Event{Kind: state.EvQuestion, EmployeeName: "tekton-1", QuestionID: "q-2",
		Text: "employee question — activity line only, no chat entry"})
	at(2000, state.Event{Kind: state.EvFileDiff, EmployeeName: "boss",
		DiffPath: "README.md", DiffAdd: 18, DiffDel: 15, DiffBody: bossDiffBody})
	at(2050, state.Event{Kind: state.EvFileDiff, EmployeeName: "tekton-1",
		DiffPath: "src/main.go", DiffAdd: 5, DiffDel: 0, DiffBody: employeeDiffBody})

	// a return: desk walk + done task + return mail
	at(2100, state.Event{Kind: state.EvReturned, EmployeeID: "dev-1", TaskID: "t1",
		Mail: mail("m1", "tekton-1", "boss", "return: build hello.html", "hello.html is up.", state.MailReturn)})
	at(2300, state.Event{Kind: state.EvMail, Mail: mail("m2", "boss", "tekton-1",
		"brief: footer next", "add a footer", state.MailBrief)})

	// permission prompts: boss (modal replaces the textarea) + child
	// (activity line only, no modal). Both stay pending unless --perm
	// answers the boss one.
	at(2400, state.Event{Kind: state.EvPermission, EmployeeName: "boss", PermissionID: "perm-1",
		ToolName: "write", ToolSummary: "main.go", ToolState: ""})
	at(2450, state.Event{Kind: state.EvPermission, EmployeeName: "tekton-1", SessionID: "child-dev-1",
		PermissionID: "perm-2", ToolName: "bash", ToolSummary: "rm -rf /tmp/scratch", ToolState: ""})

	at(2500, state.Event{Kind: state.EvIdleDrift, EmployeeID: "sco-1"})
	at(2700, state.Event{Kind: state.EvBlocked, EmployeeID: "rev-1", Text: "needs the staging key"})
	at(2750, state.Event{Kind: state.EvBubble, EmployeeID: "rev-1", Text: "anyone seen the staging key?", TTL: 40})

	// round 2: boss still typing when the frame freezes (spinner visible) —
	// unless flushQueue, which resolves it so the queue drains.
	at(3000, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u2", "user",
		"and add a footer please", false)})
	at(3050, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-2", "boss", "", true)})
	at(3300, state.Event{Kind: state.EvMail, Mail: mail("m3", "hr", "all",
		"roster synced", "3 agents seated.", state.MailNotice)})
	if b.flushQueue {
		at(3900, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("b3", "boss",
			"Ship it — and keep typing, I'll keep up.", false)})
	}
}

// scriptThink (--think) — the live think-transcript proof, everything in
// the window's earlier half: one old block ALREADY Done (renders collapsed
// in both frames), then one CallID streamed in 4 ACCUMULATED updates 600ms
// apart, then the final Done update. Frame 1 (t=2.0s) catches update 3 of
// 4 mid-stream (12 lines → "… 2 more above"); frame 2 (t=3.2s) shows both
// blocks collapsed. Lines are kept ≤34 cells so chat-width wrapping stays
// 1:1 with the source lines and the counts are deterministic.
func (b *stubBackend) scriptThink(at func(ms int, ev state.Event)) {
	thought := func(callID, text string, done bool) state.Event {
		return state.Event{Kind: state.EvThought, EmployeeID: "boss",
			EmployeeName: "boss", CallID: callID, Text: text, Done: done}
	}
	part1 := "goals first: weekly leaderboard.\n" +
		"top 20, tie-break on streak."
	part2 := part1 + "\n" +
		"store stays local — sqlite only.\n" +
		"boss peeks but never edits rows.\n" +
		"window rolls monday at midnight.\n" +
		"empty week shows last week's ghost."
	part3 := part2 + "\n" +
		"render: dim row tint, gold crown.\n" +
		"rank one gets the mug sprite.\n" +
		"long names clip at 18 cells.\n" +
		"stale rows fade after 3 weeks.\n" +
		"footer keeps the total count.\n" +
		"no pagination — one screen."
	part4 := part3 + "\n" +
		"panic path: retry once, then hold.\n" +
		"the boss asks before writes.\n" +
		"tests pin the window rollover.\n" +
		"ship friday with the mug.\n" +
		"backlog: per-team leaderboards.\n" +
		"keep the ghost rule — nice touch."

	at(50, state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — think-stream stub online"})
	at(150, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"sketch the leaderboard flow", false)})
	at(200, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	// an older, already-complete thought — collapsed in BOTH frames
	at(350, thought("th-old",
		"weekly beats daily — less noise.\nboss only sees the rollout row.", true))
	// the live stream: 4 accumulated updates, ~600ms apart, one CallID
	at(500, thought("th-1", part1, false))
	at(1100, thought("th-1", part2, false))
	at(1700, thought("th-1", part3, false))
	at(2300, thought("th-1", part4, false))
	at(2900, thought("th-1", part4, true)) // Done: final accumulated text
}

// scriptAsk (--ask-*) — the question-hold deadlock proof: the boss parks
// the turn at the question reply API and a plain typed chat message must
// NOT be Send()n — it must go through AnswerQuestion. Timeline: user
// message + typing placeholder at 150/200ms, a REGRESSION employee
// question at 1200ms (activity line only, never a modal), then the boss
// EvQuestion q-1 pending at 1500ms — the modal opens, the "boss-1"
// placeholder is REMOVED (parked, not typing), and the hold waits for the
// harness (see ask*Workload). Answering → the stub emits "resolved" +
// a completed boss reply (scriptAsk's server-resume leg lives in
// AnswerQuestion, like the real opencode round trip).
func (b *stubBackend) scriptAsk(at func(ms int, ev state.Event)) {
	at(50, state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — question-hold stub online"})
	at(150, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"summarize the flagged rows", false)})
	at(200, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	// regression: an employee question stays activity-line only, no chat
	// entry, no modal — even while the boss hold opens a beat later
	at(1200, state.Event{Kind: state.EvQuestion, EmployeeName: "tekton-1", QuestionID: "q-2",
		Text: "employee question — activity line only, no chat entry"})
	at(1500, state.Event{Kind: state.EvQuestion, EmployeeName: "boss", QuestionID: "q-1",
		Text:        "Which toggle do you want me to flip — the feature flag or the dark-mode switch?",
		ToolSummary: "the toggle one | dark mode | both"})
}

func (b *stubBackend) Mode() state.Mode { return state.ModeDemo }

func (b *stubBackend) Start(emit func(state.Event)) error {
	b.emit = emit
	b.start = time.Now()
	go b.script()
	return nil
}

// streamReply — the --stream bubble's full pinned text; streamParts are its
// accumulated prefixes (what the backend's deltas accumulate to). Kept ≤ one
// sidebar wrap per prefix so the mid-stream frame is deterministic.
const streamReply = "Honey never spoils — jars buried 3,000 years ago were " +
	"still **good to eat**. It crystallizes over time, but a warm water " +
	"bath brings it right back."

var streamParts = []string{
	"Honey never",
	"Honey never spoils — jars buried",
	"Honey never spoils — jars buried 3,000 years ago were still **good to",
	"Honey never spoils — jars buried 3,000 years ago were still **good to eat**. It crystallizes over time,",
	streamReply,
}

// scriptStream (--stream) — the live-typing proof, matching the backend's
// streaming contract: Send stages ONE "boss-N" placeholder; the reply
// arrives as 5 ACCUMULATED pending updates on the STABLE ID "bossmsg-m1"
// (300ms apart), then the pinned final (same ID, Pending=false). The mid
// frame (t=1.25s) shows the grown bubble with the typing row still live
// below the divider (the row runs for the whole pending period now —
// there is no caret); the done frame (t=2.8s) shows exactly ONE settled
// bubble — deltas merged in place, never appended. Done also flushes the
// message typed mid-stream.
func (b *stubBackend) scriptStream(at func(ms int, ev state.Event)) {
	trace := func(line string) {
		if b.trace != nil {
			b.trace(line)
		}
	}
	at(50, state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — chat-stream stub online"})
	at(150, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"tell me about honey", false)})
	at(200, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	for i, part := range streamParts {
		at(500+i*300, state.Event{Kind: state.EvChatBoss,
			Msg: chatMsg("bossmsg-m1", "boss", part, true)})
	}
	// the final pinned update — same stable ID, Pending=false
	at(2000, state.Event{Kind: state.EvChatBoss,
		Msg: chatMsg("bossmsg-m1", "boss", streamReply, false)})
	trace("[stream] done: bossmsg-m1 → pending=false")
}

// scriptBatch (--batch / --batch-respawn) — the intelligent-backlog proof
// under FREE-QUEUING: the client backlog now fills only behind a
// ROADBLOCK, so this script parks the turn at a boss question (500ms —
// the modal opens, the typing placeholder is removed). The workload
// defers the modal, types three messages while the hold is outstanding
// (each ENQUEUES as backlog #1 #2 #3), then re-opens and answers the
// question: resolved + the completed boss reply flushes the backlog as
// ONE composed [BATCH DISPATCH] send — the batch-dispatch contract is
// unchanged, only the roadblock fills the queue now.
func (b *stubBackend) scriptBatch(at func(ms int, ev state.Event)) {
	at(50, state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — backlog-batch stub online"})
	at(150, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"start the standup notes", false)})
	at(200, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	// the ROADBLOCK: the boss parks the turn at the question reply API —
	// a plain Send would deadlock the parked loop, so typed prompts keep
	// ENQUEUEING until the hold resolves (the deferred-modal path is what
	// assembles the batch)
	at(500, state.Event{Kind: state.EvQuestion, EmployeeName: "boss", QuestionID: "q-1",
		Text:        "Where should the standup notes live?",
		ToolSummary: "sqlite | postgres | in memory"})
}

// scriptFree (--freesend) — the free-queuing / anti-stuck proof: the boss
// is busy from 200ms to 3000ms; the workload sends two prompts DURING the
// busy window (each goes STRAIGHT to backend.Send — the serve queues
// them natively). The busy turn completes at 3s, then the staged
// placeholders pin in place FIFO as the server drains its queue.
func (b *stubBackend) scriptFree(at func(ms int, ev state.Event)) {
	trace := func(line string) {
		if b.trace != nil {
			b.trace(line)
		}
	}
	at(50, state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — free-queuing stub online"})
	at(150, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"start the standup notes", false)})
	at(200, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	// busy until 3s — both workload sends ride the server queue in this window
	at(3000, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss",
		"first turn done — standup notes are in drafts.", false)})
	trace("[freesend] turn completed: busy turn drained (server queue starts draining)")
	// the server-drained queue: ONE placeholder per send, pinned in place
	// (replace-by-ID keeps the FIFO slots in the transcript)
	at(3150, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-free-1", "boss",
		"footer in — aligned with the header.", false)})
	at(3300, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-free-2", "boss",
		"two tests land in the next pass.", false)})
}

// Send answers any interactive prompt deterministically (600ms ack). Reply
// IDs are UNIQUE per call ("bx-N") — with replace-by-ID in the reducer, a
// recycled ID would collapse consecutive flushed replies into one bubble.
// batchMode adds the backlog seam: the composed batch echoes as chat-user
// (proving the composite bubble) and stages the typing placeholder like the
// real backends — the pending→non-pending transition closes the board
// rows. respawnMode rejects the FIRST [BATCH DISPATCH] send once (a dead
// boss session), so the app must ResetPrimary + resend the same batch.
func (b *stubBackend) Send(text string) error {
	clip := text
	if r := []rune(clip); len(r) > 60 {
		clip = string(r[:59]) + "…"
	}
	b.sendLog = append(b.sendLog, text)
	if b.trace != nil {
		b.trace("[stub] Send(" + clip + ")")
	}
	if b.respawnMode && !b.batchFailedOnce && strings.HasPrefix(text, "[BATCH DISPATCH") {
		b.batchFailedOnce = true
		if b.trace != nil {
			b.trace("[stub] Send REJECTED — stubbed dead boss session (one-shot)")
		}
		return fmt.Errorf("stub: boss session dead")
	}
	if b.emit != nil && b.freeMode {
		// free-queuing leg: every Send echoes immediately and stages ONE
		// placeholder per send — the FIFO bubble mapping of the real serve
		// queue. The script pins each placeholder in place by ID as the
		// server drains it after the busy turn.
		b.sendSeq++
		seq := b.sendSeq
		emit := b.emit
		emit(state.Event{Kind: state.EvChatUser, Msg: chatMsg(
			fmt.Sprintf("ue-%d", seq), "user", text, false)})
		emit(state.Event{Kind: state.EvChatBoss, Msg: chatMsg(
			fmt.Sprintf("boss-free-%d", seq), "boss", "", true)})
		return nil
	}
	if b.emit != nil {
		if b.batchMode {
			b.sendSeq++
			seq := b.sendSeq
			emit := b.emit
			emit(state.Event{Kind: state.EvChatUser, Msg: chatMsg(
				fmt.Sprintf("ue-%d", seq), "user", text, false)})
			emit(state.Event{Kind: state.EvChatBoss, Msg: chatMsg(
				fmt.Sprintf("boss-batch-%d", seq), "boss", "", true)})
			time.AfterFunc(600*time.Millisecond, func() {
				emit(state.Event{Kind: state.EvChatBoss, Msg: chatMsg(
					fmt.Sprintf("bx-%d", seq), "boss",
					"backlog dispatched: 3 items split across the floor — status table on their return.", false)})
			})
			return nil
		}
		b.sendSeq++
		id := fmt.Sprintf("bx-%d", b.sendSeq)
		reply := "Roger that."
		if b.streamMode {
			reply = "flushed follow-up handled: " + text
		}
		time.AfterFunc(600*time.Millisecond, func() {
			b.emit(state.Event{Kind: state.EvChatBoss,
				Msg: chatMsg(id, "boss", reply, false)})
		})
	}
	return nil
}

// --- teamBackend seam (the backlog board) ----------------------------------
// Log-only twins of the live/demo contract: the frame's proof is the
// printed call log, not an emitted board (board tabs are staged separately).

// QueueItemStart mirrors one backlog item: logs the call, returns the
// deterministic "demo-N" board row id.
func (b *stubBackend) QueueItemStart(index int, title string) string {
	id := fmt.Sprintf("demo-%d", index)
	b.teamLog = append(b.teamLog, fmt.Sprintf("QueueItemStart(%d, %q) -> %s", index, title, id))
	if b.trace != nil {
		b.trace(fmt.Sprintf("[team] QueueItemStart(%d, %q) -> %s", index, title, id))
	}
	return id
}

// QueueItemDone closes the board row when the batch's turn completes.
func (b *stubBackend) QueueItemDone(boardID string) {
	b.teamLog = append(b.teamLog, fmt.Sprintf("QueueItemDone(%s)", boardID))
	if b.trace != nil {
		b.trace("[team] QueueItemDone(" + boardID + ")")
	}
}

// ResetPrimary is the failure-respawn hook: logs the respawn of the boss
// session (the retry resends the SAME batch right after).
func (b *stubBackend) ResetPrimary(forceNew bool) error {
	b.teamLog = append(b.teamLog, fmt.Sprintf("ResetPrimary(%v)", forceNew))
	if b.trace != nil {
		b.trace(fmt.Sprintf("[team] ResetPrimary(%v)", forceNew))
	}
	return nil
}

// AnswerPermission records the reply (proof for --perm) and emits the
// matching "resolved" event a beat later, like the real backends.
func (b *stubBackend) AnswerPermission(permissionID, response string) error {
	b.permAnswer = permissionID + ":" + response
	if b.emit != nil {
		time.AfterFunc(200*time.Millisecond, func() {
			b.emit(state.Event{Kind: state.EvPermission, PermissionID: permissionID,
				EmployeeName: "boss", ToolState: "resolved"})
		})
	}
	return nil
}

// AnswerQuestion records the reply (capture proof for --ask-*) and plays
// the resumed server: the matching "resolved" event a beat later, then a
// COMPLETED boss reply — the contract leg that unblocks the parked queue.
func (b *stubBackend) AnswerQuestion(requestID string, answers [][]string) error {
	pages := make([]string, len(answers))
	for i, a := range answers {
		pages[i] = strings.Join(a, ", ")
	}
	line := fmt.Sprintf("AnswerQuestion(%s, [%s])", requestID, strings.Join(pages, "; "))
	b.answerLog = append(b.answerLog, line)
	if b.trace != nil {
		b.trace("[ask] " + line)
	}
	if b.emit != nil {
		emit := b.emit
		time.AfterFunc(200*time.Millisecond, func() {
			if b.trace != nil {
				b.trace("[ask] resolved: " + requestID)
			}
			emit(state.Event{Kind: state.EvQuestion, QuestionID: requestID,
				EmployeeName: "boss", ToolSummary: "answered", ToolState: "resolved"})
		})
		time.AfterFunc(450*time.Millisecond, func() {
			if b.trace != nil {
				b.trace("[ask] server resumed: completed boss reply")
			}
			b.sendSeq++
			emit(state.Event{Kind: state.EvChatBoss, Msg: chatMsg(
				fmt.Sprintf("bq-%d", b.sendSeq),
				"boss", "flipped — thanks, that clears it.", false)})
		})
	}
	return nil
}

// RejectQuestion records the reply; the hold resolves like an answer.
func (b *stubBackend) RejectQuestion(requestID string) error {
	line := fmt.Sprintf("RejectQuestion(%s)", requestID)
	b.answerLog = append(b.answerLog, line)
	if b.trace != nil {
		b.trace("[ask] " + line)
	}
	if b.emit != nil {
		emit := b.emit
		time.AfterFunc(200*time.Millisecond, func() {
			emit(state.Event{Kind: state.EvQuestion, QuestionID: requestID,
				EmployeeName: "boss", ToolSummary: "rejected", ToolState: "resolved"})
		})
	}
	return nil
}

func (b *stubBackend) Stop() error { return nil }

// MCPServers serves a tiny static fixture so the /mcp slash command has
// something to render in shots; ReconnectMCP records the call like the
// other capture proofs.
func (b *stubBackend) MCPServers() ([]state.MCPServer, error) {
	return []state.MCPServer{
		{Name: "local-memory", Status: "connected", Detail: "12 tools"},
		{Name: "postgres", Status: "failed", Detail: "connection refused"},
	}, nil
}

func (b *stubBackend) ReconnectMCP(name string) error {
	b.answerLog = append(b.answerLog, "ReconnectMCP("+name+")")
	return nil
}

// AbortSessions is the /stop seam (the parallel backend contract: abort
// the primary session AND every live child session). The stub only records
// the call — the --stop frame proves the unwind.
// ListModels — the /model picker's listing seam on the shot stub
// (ADDITIVE, the app's modelListBackend type-assert): the SAME fixed
// five-model gallery the demo backend serves (one fixture — the
// --modelshot frame renders exactly what a demo member sees), so the
// picker's fetch-hop resolves in-process within the 4s window.
func (b *stubBackend) ListModels(ctx context.Context) ([]state.ModelInfo, error) {
	return backend.DemoModels(), nil
}

func (b *stubBackend) AbortSessions() error {
	b.abortLog = append(b.abortLog, "AbortSessions()")
	if b.trace != nil {
		b.trace("[stub] AbortSessions()")
	}
	return b.abortErr
}

// --- concierge seam (--concierge) ------------------------------------------

// evChatOffice — the concierge chat event kind (the parallel contract's
// state.EvChatOffice; aliased for terseness).
var evChatOffice = state.EvChatOffice

// conciergeStub — the --concierge backend: every stubBackend behavior PLUS
// the ConciergeCapable seam (SendConcierge). Kept as an embedding wrapper
// so the PLAIN stub stays concierge-incapable: the existing busy-turn
// proofs (--stream/--freesend) assert the free-send boss-queue path, which
// the app must keep using when the seam is absent.
type conciergeStub struct {
	*stubBackend
	conciergeLog []string // every SendConcierge call, verbatim (the proof)
}

// officeMsg builds one concierge chat message per the parallel contract
// (From "office", Kind "office", ID "office-<msgID>", replace-in-place).
func officeMsg(id, text string, pending bool) state.ChatMsg {
	return state.ChatMsg{ID: id, From: "office", Kind: "office", Text: text,
		At: time.Now().UnixMilli(), Pending: pending}
}

// SendConcierge — the routable concierge seam: capture the prompt (the
// proof), then emit ONLY the empty pending placeholder ("office-<seq>");
// the DETERMINISTIC harness drives the completion events itself.
func (s *conciergeStub) SendConcierge(text string) error {
	s.conciergeLog = append(s.conciergeLog, text)
	if s.trace != nil {
		s.trace("[stub] SendConcierge(" + text + ")")
	}
	if s.emit != nil {
		s.sendSeq++
		s.emit(state.Event{Kind: evChatOffice,
			Msg: officeMsg(fmt.Sprintf("office-%d", s.sendSeq), "", true)})
	}
	return nil
}

// slashWorkload simulates the user typing a slash command into the chat
// textarea and hitting Enter — proving slash dispatch never hits the backend
// and the office notice renders. It types /theme dracula (switch + persist),
// then /themes (listing notice) — they land before the 3050ms pending lock.
func slashWorkload(p *tea.Program) {
	typeLine := func(s string) {
		for _, r := range s {
			p.Send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
			time.Sleep(10 * time.Millisecond)
		}
		p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
		time.Sleep(80 * time.Millisecond)
	}
	time.Sleep(1950 * time.Millisecond)
	typeLine("/theme dracula")
	typeLine("/themes")
}

// queueWorkload types three short messages and hits Enter while the round-2
// boss reply is pending (3050ms) — each lands in the model-level queue; the
// placeholder + statusbar badge show the depth. TEXTS AVOID y/a/n: while the
// permission prompt is open those keys answer it (by design).
func queueWorkload(p *tea.Program) {
	typeLine := func(s string) {
		for _, r := range s {
			p.Send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
			time.Sleep(8 * time.Millisecond)
		}
		p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	}
	time.Sleep(3060 * time.Millisecond)
	typeLine("first queued")
	time.Sleep(100 * time.Millisecond)
	typeLine("look it up")
	time.Sleep(100 * time.Millisecond)
	typeLine("ship this")
}

// permWorkload (--perm) answers the boss permission prompt with "once" 3s
// in — after the prompt opened at 2400ms, before the frame at 4s.
func permWorkload(p *tea.Program) {
	time.Sleep(3000 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
}

// diffsWorkload (--diffs) presses ctrl+d once the diff entries exist
// (2000/2050ms), expanding all of them for the final frame.
func diffsWorkload(p *tea.Program) {
	time.Sleep(2200 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: 'd', Mod: tea.ModCtrl}))
}

// modelshotWorkload (--modelshot) opens the /model PICKER for the final
// frame — the "any model, any backend" gallery shot. The two stacked
// permission asks are answered first (y·y — the modal floats over the
// same region and would bury the card). The command is typed AFTER the
// always-on queue typing (~3.6s), then the two-press dance run: the
// first Enter only APPLIES the popover's "/model" row into the draft
// (bare slash commands never auto-send — the same contract the stop/ask
// proofs dance with), the second SENDS it; the stub's fixed five-model
// listing answers in-process, so by the 4s frame the card shows every
// row with the cursor clamped on the first.
func modelshotWorkload(p *tea.Program) {
	time.Sleep(2500 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	time.Sleep(250 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: 'y', Text: "y"}))
	time.Sleep(890 * time.Millisecond) // ~3.64s — the queue typing below has drained
	for _, r := range "/model" {
		p.Send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		time.Sleep(8 * time.Millisecond)
	}
	time.Sleep(80 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // apply the popover row into the draft
	time.Sleep(90 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // send the bare command — the picker opens
}

// askTypeLine types a line into the open boss question modal rune by rune
// and hits Enter — the text must land in the modal's OWN input (the
// textarea is disabled while the hold is open).
func askTypeLine(p *tea.Program, s string) {
	for _, r := range s {
		p.Send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		time.Sleep(8 * time.Millisecond)
	}
	p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	time.Sleep(60 * time.Millisecond)
}

// askEsc presses esc — with an open question modal this DEFERS the hold
// (notice "(question deferred — /question to reopen)").
func askEsc(p *tea.Program) {
	p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	time.Sleep(60 * time.Millisecond)
}

// askAnswerWorkload (--ask-answer): at 2.5s the user types "the toggle
// one" + enter into the open modal → must hit AnswerQuestion, never Send.
func askAnswerWorkload(p *tea.Program) {
	time.Sleep(2500 * time.Millisecond)
	askTypeLine(p, "the toggle one")
}

// askEscWorkload (--ask-esc): esc defers the hold, /question re-opens it,
// the answer still routes through AnswerQuestion.
func askEscWorkload(p *tea.Program) {
	time.Sleep(2300 * time.Millisecond)
	askEsc(p)
	time.Sleep(500 * time.Millisecond)
	askTypeLine(p, "/question")
	askBareSlashSend(p) // the popover's "/question" row APPLIES; Enter again sends
	time.Sleep(300 * time.Millisecond)
	askTypeLine(p, "the toggle one")
}

// askQueueWorkload (--ask-queue): the queue-hold proof — a message typed
// while the hold is DEFERRED-but-outstanding must ENQUEUE (the turn is
// parked, flushing it would re-create the deadlock); answering q-1 then
// resolving + the completed boss reply must flush it, in that order.
func askQueueWorkload(p *tea.Program) {
	time.Sleep(2300 * time.Millisecond)
	askEsc(p)
	time.Sleep(300 * time.Millisecond)
	askTypeLine(p, "fix the badge too") // enqueued — turn is parked
	time.Sleep(600 * time.Millisecond)
	askTypeLine(p, "/question") // re-open the deferred hold
	askBareSlashSend(p)
	time.Sleep(300 * time.Millisecond)
	askTypeLine(p, "the toggle one") // AnswerQuestion → resume → flush
}

// askBareSlashSend presses Enter when a JUST-TYPED bare slash command is
// sitting in the draft as a popover selection: the first Enter (inside
// askTypeLine/typeLine) only APPLIES the popover row into the draft, so a
// second Enter is what actually sends it. (Zero-match commands fall
// through on the first Enter — see the chat panel's slashOpen arm.)
func askBareSlashSend(p *tea.Program) {
	time.Sleep(120 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	time.Sleep(120 * time.Millisecond)
}

// runThinkShot runs one fresh app+program against a think-mode stub for
// `dur`, then returns the final frame. Two calls with different durations
// = the deterministic before/after pair (--think's frames).
func runThinkShot(tab string, dur time.Duration) (string, error) {
	backend := &stubBackend{done: make(chan struct{}), thinkMode: true}
	m := app.New(backend, config.Default())
	if !m.SelectTab(tab) {
		return "", fmt.Errorf("unknown tab %q", tab)
	}
	p := tea.NewProgram(m,
		tea.WithWindowSize(shotCols, shotRows),
		tea.WithoutRenderer(),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
	)
	emit := func(ev state.Event) { p.Send(ev) }
	if err := backend.Start(emit); err != nil {
		return "", err
	}
	go func() {
		time.Sleep(dur)
		p.Quit()
	}()
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	fm, ok := final.(app.Model)
	if !ok {
		return "", fmt.Errorf("unexpected final model type %T", final)
	}
	return fm.Frame(), nil
}

// streamWorkload (--stream) types ONE message and hits Enter mid-stream —
// the deltas run 500–1700ms and the placeholder went pending at 200ms, so
// Enter lands while the boss reply is outstanding: under FREE-QUEUING it
// goes STRAIGHT to backend.Send (the serve queues it natively — no client
// queue), and its reply lands while the stream is still live. The trace
// lines ([stub] Send / done, all timestamped) prove the order.
func streamWorkload(p *tea.Program) {
	typeLine := func(s string) {
		for _, r := range s {
			p.Send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
			time.Sleep(8 * time.Millisecond)
		}
		p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	}
	time.Sleep(900 * time.Millisecond)
	typeLine("how do bees make it")
}

// batchWorkload (--batch / --batch-respawn): the boss question modal opens
// at 500ms — the user DEFERS it (esc, 1.1s), then types three messages
// while the hold stays outstanding: each ENQUEUES as a numbered backlog
// item. Re-opening the hold and answering it (3.4s+) resumes the turn —
// resolved + the completed boss reply flushes the backlog as ONE batch.
func batchWorkload(p *tea.Program) {
	typeLine := func(s string) {
		for _, r := range s {
			p.Send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
			time.Sleep(8 * time.Millisecond)
		}
		p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	}
	time.Sleep(1100 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})) // defer the hold
	time.Sleep(400 * time.Millisecond)
	typeLine("fix the badge")
	time.Sleep(400 * time.Millisecond)
	typeLine("ship v2")
	time.Sleep(400 * time.Millisecond)
	typeLine("write the release notes")
	time.Sleep(500 * time.Millisecond)
	typeLine("/question") // re-open the deferred hold…
	time.Sleep(200 * time.Millisecond)
	// …the slash popover matches its own "/question" row, so the typed
	// Enter only APPLIES it into the draft; a second Enter sends the
	// command itself (same two-press dance as any bare popover command).
	p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	time.Sleep(400 * time.Millisecond)
	typeLine("sqlite") // …and answer it → resolve → turn complete → flush
}

// freeWorkload (--freesend) sends two prompts while the boss turn is busy
// (200–3000ms). Under free-queuing each goes STRAIGHT to backend.Send —
// the ordering trace shows both before the turn completes.
func freeWorkload(p *tea.Program) {
	typeLine := func(s string) {
		for _, r := range s {
			p.Send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
			time.Sleep(8 * time.Millisecond)
		}
		p.Send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	}
	time.Sleep(700 * time.Millisecond)
	typeLine("add the footer")
	time.Sleep(500 * time.Millisecond)
	typeLine("and two tests")
}

// traceLog collects timestamped ordering lines for the --stream proof
// (enqueue / done / flush), written from the script goroutine, the tea
// update loop (via app.QueueDebugf) and the shot runner.
type traceLog struct {
	mu    sync.Mutex
	start time.Time
	lines []string
}

func (t *traceLog) add(line string) {
	t.mu.Lock()
	t.lines = append(t.lines, fmt.Sprintf("%s @+%dms", line, time.Since(t.start).Milliseconds()))
	t.mu.Unlock()
}

// runStreamShot runs one fresh app+program against a stream-mode stub for
// `dur`, then returns the final frame plus the ordering trace. Two calls
// with different durations = the deterministic mid-stream/after-done pair.
func runStreamShot(dur time.Duration) (string, []string, error) {
	tl := &traceLog{start: time.Now()}
	backend := &stubBackend{done: make(chan struct{}), streamMode: true, trace: tl.add}
	m := app.New(backend, config.Default())
	if !m.SelectTab("chat") {
		return "", nil, fmt.Errorf("unknown tab %q", "chat")
	}
	p := tea.NewProgram(m,
		tea.WithWindowSize(shotCols, shotRows),
		tea.WithoutRenderer(),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
	)
	app.QueueDebugf = func(format string, args ...any) {
		tl.add("[queue] " + fmt.Sprintf(format, args...))
	}
	emit := func(ev state.Event) { p.Send(ev) }
	if err := backend.Start(emit); err != nil {
		return "", nil, err
	}
	go streamWorkload(p)
	go func() {
		time.Sleep(dur)
		p.Quit()
	}()
	final, err := p.Run()
	if err != nil {
		return "", nil, err
	}
	fm, ok := final.(app.Model)
	if !ok {
		return "", nil, fmt.Errorf("unexpected final model type %T", final)
	}
	app.QueueDebugf = nil
	tl.mu.Lock()
	lines := append([]string(nil), tl.lines...)
	tl.mu.Unlock()
	return fm.Frame(), lines, nil
}

// runAskShot runs one fresh app+program against a question-hold stub for
// `dur`, driving the ask workload for `mode`, then returns the final
// frame, the ordering trace, and the stub's captured answer calls.
func runAskShot(mode string, dur time.Duration) (string, []string, []string, error) {
	tl := &traceLog{start: time.Now()}
	backend := &stubBackend{done: make(chan struct{}), askMode: mode, trace: tl.add}
	m := app.New(backend, config.Default())
	if !m.SelectTab("chat") {
		return "", nil, nil, fmt.Errorf("unknown tab %q", "chat")
	}
	p := tea.NewProgram(m,
		tea.WithWindowSize(shotCols, shotRows),
		tea.WithoutRenderer(),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
	)
	app.QueueDebugf = func(format string, args ...any) {
		tl.add("[queue] " + fmt.Sprintf(format, args...))
	}
	emit := func(ev state.Event) { p.Send(ev) }
	if err := backend.Start(emit); err != nil {
		return "", nil, nil, err
	}
	switch mode {
	case "answer":
		go askAnswerWorkload(p)
	case "esc":
		go askEscWorkload(p)
	case "queue":
		go askQueueWorkload(p)
	}
	go func() {
		time.Sleep(dur)
		p.Quit()
	}()
	final, err := p.Run()
	if err != nil {
		return "", nil, nil, err
	}
	fm, ok := final.(app.Model)
	if !ok {
		return "", nil, nil, fmt.Errorf("unexpected final model type %T", final)
	}
	app.QueueDebugf = nil
	tl.mu.Lock()
	lines := append([]string(nil), tl.lines...)
	tl.mu.Unlock()
	return fm.Frame(), lines, backend.answerLog, nil
}

// runBatchShot runs one fresh app+program against the backlog stub and
// returns the frame, ordering trace, verbatim Send calls and team-seam
// calls. respawn=true stubs the first batch Send dead (the respawn proof).
func runBatchShot(respawn bool, dur time.Duration) (string, []string, []string, []string, error) {
	tl := &traceLog{start: time.Now()}
	backend := &stubBackend{done: make(chan struct{}),
		batchMode: true, respawnMode: respawn, trace: tl.add}
	m := app.New(backend, config.Default())
	if !m.SelectTab("chat") {
		return "", nil, nil, nil, fmt.Errorf("unknown tab %q", "chat")
	}
	p := tea.NewProgram(m,
		tea.WithWindowSize(shotCols, shotRows),
		tea.WithoutRenderer(),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
	)
	app.QueueDebugf = func(format string, args ...any) {
		tl.add("[queue] " + fmt.Sprintf(format, args...))
	}
	emit := func(ev state.Event) { p.Send(ev) }
	if err := backend.Start(emit); err != nil {
		return "", nil, nil, nil, err
	}
	go batchWorkload(p)
	go func() {
		time.Sleep(dur)
		p.Quit()
	}()
	final, err := p.Run()
	if err != nil {
		return "", nil, nil, nil, err
	}
	fm, ok := final.(app.Model)
	if !ok {
		return "", nil, nil, nil, fmt.Errorf("unexpected final model type %T", final)
	}
	app.QueueDebugf = nil
	tl.mu.Lock()
	lines := append([]string(nil), tl.lines...)
	tl.mu.Unlock()
	return fm.Frame(), lines, backend.sendLog, backend.teamLog, nil
}

// runFreeShot runs one fresh app+program against a free-mode stub for
// `dur`, typing two prompts during the busy window, then returns the
// final frame plus the ordering trace. Two calls with different durations
// = the busy and the drained pair (--freesend's frames).
func runFreeShot(dur time.Duration) (string, []string, error) {
	tl := &traceLog{start: time.Now()}
	backend := &stubBackend{done: make(chan struct{}), freeMode: true, trace: tl.add}
	m := app.New(backend, config.Default())
	if !m.SelectTab("chat") {
		return "", nil, fmt.Errorf("unknown tab %q", "chat")
	}
	p := tea.NewProgram(m,
		tea.WithWindowSize(shotCols, shotRows),
		tea.WithoutRenderer(),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
	)
	app.QueueDebugf = func(format string, args ...any) {
		tl.add("[queue] " + fmt.Sprintf(format, args...))
	}
	emit := func(ev state.Event) { p.Send(ev) }
	if err := backend.Start(emit); err != nil {
		return "", nil, err
	}
	go freeWorkload(p)
	go func() {
		time.Sleep(dur)
		p.Quit()
	}()
	final, err := p.Run()
	if err != nil {
		return "", nil, err
	}
	fm, ok := final.(app.Model)
	if !ok {
		return "", nil, fmt.Errorf("unexpected final model type %T", final)
	}
	app.QueueDebugf = nil
	tl.mu.Lock()
	lines := append([]string(nil), tl.lines...)
	tl.mu.Unlock()
	return fm.Frame(), lines, nil
}

// --- /stop proof (--stop) ---------------------------------------------------
// Synchronous driver (no wall clock): the boss is mid-stream with its own
// tool running, tekton-1 is at work with one tool still running, a second
// prompt has staged its own placeholder, a client backlog item rides a
// DEFERRED boss question hold (today's reachable enqueue roadblock — see
// below), and the boss has gone quiet (delegating). Typing /stop must
// (1) hit stub.AbortSessions, (2) unwind everything in ONE frame.
//
// Roadblock-premise refresh (the wave-13 script went stale): back then the
// permission modal was non-modal to text — a typed prompt + enter ENQUEUED
// behind it. The wave-16 floating permission popover claims enter as
// "confirm the highlighted option" (y/a/n quick-answer): under the old
// script the enter ANSWERED the ask, the stranded draft then rode `/stop`'s
// first enter into a busy-send, and the concierge-incapable stub's fallback
// free-routed the text SERVER-side ("concierge unavailable — boss queued
// it") — /stop never ran, zero AbortSessions. TODAY the only reachable
// client-queue roadblock is an outstanding boss question hold (the chat
// panel's enter gate: "the turn is parked at the question reply API", the
// placeholder reads "boss is waiting for your answer… · N queued"). The
// /stop promise proven here is unchanged: abort + ONE-frame unwind + the
// LOCAL queue intact and unsent.

func runStopProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	stub := &stubBackend{done: make(chan struct{})}
	m := app.New(stub, config.Default())
	d := &focusDriver{m: m}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})

	key := func(code rune) tea.Cmd {
		tm, c := d.m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
		if fm, ok := tm.(app.Model); ok {
			d.m = fm
		}
		return c
	}
	typeIn := func(s string) {
		for _, r := range s {
			tm, _ := d.m.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
			if fm, ok := tm.(app.Model); ok {
				d.m = fm
			}
		}
	}

	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — stop stub online"})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	// the ROADBLOCK, staged FIRST: the boss parks the turn on a question
	// (parkForQuestion purges pending boss-N placeholders at park time —
	// none exist yet). The member ESC-DEFERS the popover: the hold stays
	// outstanding (questionWaiting keeps the chat enter gate armed), the
	// placeholder reads "boss is waiting for your answer…".
	d.send(state.Event{Kind: state.EvQuestion, EmployeeName: "boss", QuestionID: "qq-1",
		Text: "Roll the SSE stream behind a feature flag?", ToolSummary: "flag it | ship it straight"})
	drainCmd(d, key(tea.KeyEscape), 0) // defer the hold — typing keeps working, enter enqueues
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"wire the sse stream", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	d.send(focusTool("boss", "boss", "call-1", "write", "handler.go", "running"))
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "dev-1",
		Task: state.BoardTask{ID: "t1", Title: "Wire the SSE stream", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "dev-1", TaskID: "t1"})
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/room/manager.go", "running"))
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/room/handler.go", "done"))
	// mid-stream: the accumulated bubble grows (boss-1's empty placeholder
	// is replaced by the real stream ID)
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"Wiring the handler — the **SSE stream** fans out to both workers now,", true)})
	// a second prompt staged during the busy turn gets its OWN placeholder
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u2", "user",
		"and the retry backoff", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-2", "boss", "", true)})
	// the roadblock-held client item: a typed prompt + enter while the
	// (esc'd) question hold is outstanding enqueues LOCALLY — the chat
	// panel's enter gate sees questionWaiting and routes to onEnqueue.
	// This send must never reach the concierge/busy free-route: the drift
	// guard below fails fast with the pointed sendLog dump if it does.
	typeIn("keep me plz")
	drainCmd(d, key(tea.KeyEnter), 0)
	if stub.sendLog != nil {
		return fail("stop: the roadblock item free-routed to the server (sendLog %v) — the enqueue gate must hold it locally", stub.sendLog)
	}
	d.pump(8) // boss quiet at its placeholder with a busy worker → delegating
	preFrame := d.m.Frame()
	if !strings.Contains(preFrame, "delegating · 1 busy") {
		return fail("stop: setup frame missing the delegating row (boss quiet, tekton-1 busy)")
	}
	if strings.Contains(preFrame, "✗") {
		return fail("stop: setup frame already shows an aborted marker before /stop")
	}

	// +1s: the user hits /stop — /stop is a real popover row now, so the
	// first Enter APPLIES the selection into the draft and the SECOND
	// Enter sends it (the same two-press dance as every bare popover
	// command — /question in the ask proofs, /queue in the leg below).
	typeIn("/stop")
	drainCmd(d, key(tea.KeyEnter), 0)
	drainCmd(d, key(tea.KeyEnter), 0)

	st := d.m.State()
	fmt.Println("===== UI SHOT · STOP A — after /stop: placeholders collapsed, stream kept + (stopped), tools ✗ aborted, thread ✗ stopped, queue intact =====")
	frameA := d.m.Frame()
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("--- stub capture (AbortSessions) ---")
	for _, ln := range stub.abortLog {
		fmt.Println(ln)
	}
	if len(stub.abortLog) != 1 {
		return fail("expected exactly 1 AbortSessions call, got %d", len(stub.abortLog))
	}
	for _, want := range []string{
		"stopped by user",                     // (a) the staged boss-2 placeholder collapsed
		"SSE stream",                          // (b) the streamed text survived…
		"(stopped)",                           // …with the appendix
		"[tool] write · handler.go ✗ aborted", // (c) boss inline tool swung
		"Developer Task — Wire the SSE stream (· 2 tool calls ✗ stopped)", // (d) thread collapsed stopped
		"stopped current work — queue intact",                             // (e) statusband notice (the "(N items)" tail can clip under the bar's right segments)
		"q1",                                                              // the client queue badge survived
	} {
		if !strings.Contains(frameA, want) {
			return fail("stop A: frame missing %q", want)
		}
	}
	if want := "stopped current work — queue intact (1 items)"; st.StatusLine != want {
		return fail("stop A: StatusLine = %q, want %q", st.StatusLine, want)
	}
	if st.BossThinking || st.BossDelegating {
		return fail("stop A: BossThinking=%v BossDelegating=%v after /stop (both must clear)", st.BossThinking, st.BossDelegating)
	}
	if strings.Contains(frameA, "… running") {
		return fail("stop A: a tool still renders \"… running\" after /stop")
	}
	// the queued item survived verbatim: /queue lists it (the first Enter
	// applies the popover row, the second sends the command)
	typeIn("/queue")
	drainCmd(d, key(tea.KeyEnter), 0)
	drainCmd(d, key(tea.KeyEnter), 0)
	frameQ := d.m.Frame()
	for _, want := range []string{"1 queued", "1. keep me plz"} {
		if !strings.Contains(frameQ, want) {
			return fail("stop B: /queue listing missing %q — the queued item was lost", want)
		}
	}
	if stub.sendLog != nil {
		return fail("stop: the queued item must NOT have been sent, got sends: %v", stub.sendLog)
	}
	fmt.Println("--- /queue leg (after /stop): the roadblock item survived verbatim, NOT sent ---")
	fmt.Println(frameQ)
	fmt.Println("asserts: OK — AbortSessions captured once; boss-2 placeholder → \"stopped by user\"; streamed text kept + \" (stopped)\"; boss + worker tools ✗ aborted; thread (· 2 tool calls ✗ stopped); BossThinking/BossDelegating cleared; queue intact (1 item enqueued behind the esc-deferred question hold, badge q1, NOT sent — flushes next turn)")
	return nil
}

// --- bypass-permissions mode proof (--bypass) ---------------------------------
// Synchronous driver, TWO byte-identical drives. The legs: "/bypass" typed
// opens the office's question popover as the explicit arming confirm
// (enable/cancel — never one keypress); enter on "enable" arms the mode:
// the ⚠ BYPASS segment rides the topbar and the pinned ON notice lands;
// a stray EvPermission (emitted before the toggle's respawn would land)
// is answered allow-once on the stub's wire with NO modal parking + the
// dim "bypass: auto-approved bash" row; "/bypass" again disables INSTANTLY
// (no confirm): OFF notice, segment gone. The demo stub skips the
// transport respawn by design (live-only) — the respawn contract is
// pinned by internal/app's bypass_test.go.

func runBypassProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	type shot struct {
		label string
		frame string
	}
	drive := func() ([]shot, string, error) {
		stub := &stubBackend{done: make(chan struct{})}
		m := app.New(stub, config.Default())
		d := &focusDriver{m: m}
		d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
		key := func(code rune) tea.Cmd {
			tm, c := d.m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
			if fm, ok := tm.(app.Model); ok {
				d.m = fm
			}
			return c
		}
		typeIn := func(s string) {
			for _, r := range s {
				tm, _ := d.m.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
				if fm, ok := tm.(app.Model); ok {
					d.m = fm
				}
			}
		}
		d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — bypass stub online"})
		d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
			ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})

		var shots []shot

		// leg 1 — /bypass typed: the arming confirm opens (enable/cancel).
		typeIn("/bypass")
		drainCmd(d, key(tea.KeyEnter), 0)
		f1 := d.m.Frame()
		for _, want := range []string{"Enable bypass permissions?", "WITHOUT asking", "enable", "cancel"} {
			if !strings.Contains(ansi.Strip(f1), want) {
				return nil, "", fail("bypass A (confirm open): frame missing %q", want)
			}
		}
		if strings.Contains(ansi.Strip(f1), "⚠ BYPASS") {
			return nil, "", fail("bypass A: the segment must NOT show before the confirm answers")
		}
		shots = append(shots, shot{"A — /bypass typed: the arming confirm (enable/cancel), mode still OFF", f1})

		// leg 2 — enter on "enable": the mode arms: ⚠ BYPASS rides the
		// topbar, the pinned ON notice lands in the transcript.
		drainCmd(d, key(tea.KeyEnter), 0)
		f2 := d.m.Frame()
		for _, want := range []string{"⚠ BYPASS", "bypass permissions: ON", "nothing will ask"} {
			if !strings.Contains(ansi.Strip(f2), want) {
				return nil, "", fail("bypass B (armed): frame missing %q", want)
			}
		}
		shots = append(shots, shot{"B — confirm answered \"enable\": ⚠ BYPASS rides the topbar + the ON notice", f2})

		// leg 3 — a stray EvPermission lands while armed: answered
		// allow-once on the wire, NO modal parks, the dim row logs it.
		tm, c := d.m.Update(state.Event{Kind: state.EvPermission, EmployeeName: "boss",
			PermissionID: "perm-b1", ToolName: "bash", ToolSummary: "rm -rf /tmp/x", ToolState: "pending"})
		if fm, ok := tm.(app.Model); ok {
			d.m = fm
		}
		drainCmd(d, c, 0)
		f3 := d.m.Frame()
		if stub.permAnswer != "perm-b1:once" {
			return nil, "", fail("bypass C: the stray ask must auto-answer allow-once, stub captured %q", stub.permAnswer)
		}
		if strings.Contains(ansi.Strip(f3), "PERMISSION REQUIRED") {
			return nil, "", fail("bypass C: NO modal may park while armed")
		}
		if !strings.Contains(ansi.Strip(f3), "bypass: auto-approved bash") {
			return nil, "", fail("bypass C: the dim auto-approval row is missing")
		}
		if !strings.Contains(ansi.Strip(f3), "⚠ BYPASS") {
			return nil, "", fail("bypass C: the segment must still ride the topbar")
		}
		shots = append(shots, shot{"C — stray EvPermission auto-answered (perm-b1:once on the wire, no modal, dim row)", f3})

		// leg 4 — /bypass again: INSTANT disable (no confirm), OFF notice,
		// the segment leaves the topbar.
		typeIn("/bypass")
		drainCmd(d, key(tea.KeyEnter), 0)
		f4 := d.m.Frame()
		if strings.Contains(ansi.Strip(f4), "Enable bypass permissions?") {
			return nil, "", fail("bypass D: disable must NOT ask for a confirm")
		}
		if strings.Contains(ansi.Strip(f4), "⚠ BYPASS") {
			return nil, "", fail("bypass D: the segment must leave the topbar on disable")
		}
		if !strings.Contains(ansi.Strip(f4), "bypass permissions: OFF") {
			return nil, "", fail("bypass D: the OFF notice is missing")
		}
		shots = append(shots, shot{"D — /bypass again: instant OFF (no confirm), segment gone", f4})
		return shots, stub.permAnswer, nil
	}

	shotsA, answerA, err := drive()
	if err != nil {
		return err
	}
	shotsB, answerB, err := drive()
	if err != nil {
		return err
	}
	if answerA != answerB {
		return fail("the two drives captured different wire answers: %q vs %q", answerA, answerB)
	}
	for i := range shotsA {
		fmt.Printf("===== UI SHOT · --bypass %s =====\n", shotsA[i].label)
		fmt.Println(shotsA[i].frame)
		fmt.Println("===== UI SHOT =====")
		if shotsA[i].frame != shotsB[i].frame {
			return fail("bypass: leg %d differs between the two synchronous drives", i+1)
		}
	}
	fmt.Printf("--- stub capture (AnswerPermission): %s ---\n", answerA)
	fmt.Println("deterministic: OK — two synchronous drives produced byte-identical frames")
	fmt.Println("asserts: OK — /bypass opens the arming confirm (enable/cancel, never one keypress); enable arms: ⚠ BYPASS rides the topbar + the pinned ON notice; a stray EvPermission auto-answers allow-once on the wire (perm-b1:once) with NO modal parked + the dim \"bypass: auto-approved bash\" row; disable is INSTANT (no confirm) with the OFF notice + the segment gone; two drives byte-identical")
	return nil
}

// --- boss-stuck-busy proof (--stuck) ----------------------------------------
// Synchronous driver (no wall clock beyond TWO bounded sleeps the watchdog
// threshold makes deterministic). Leg 1 — W1: the boss goes busy at 200ms
// and NEVER completes; the wedge watchdog (harness-seamed through
// SetWedgeAfterForShot — the production 2m floor can't be slept out in a
// shot) fires exactly ONCE: ONE activity-tab "boss turn wedged" line (the
// transcript stays clean — zero chat rows) and the hint-seam swap. Leg 2 —
// G1: /stop against an AbortSessions stubbed to FAIL (stub.abortErr): the
// office must NOT strand — the placeholder collapses to "stopped by user",
// one dim note records the remote failure, the statusline reads stopped,
// the watchdog re-arms.

func runStuckProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	stub := &stubBackend{done: make(chan struct{}),
		abortErr: errors.New("abort: connection refused (serve dead)")}
	m := app.New(stub, config.Default())
	m.SetWedgeAfterForShot(30 * time.Millisecond) // harness seam — not config
	d := &focusDriver{m: m}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})

	key := func(code rune) tea.Cmd {
		tm, c := d.m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
		if fm, ok := tm.(app.Model); ok {
			d.m = fm
		}
		return c
	}
	typeIn := func(s string) {
		for _, r := range s {
			tm, _ := d.m.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
			if fm, ok := tm.(app.Model); ok {
				d.m = fm
			}
		}
	}
	// the wedge note lives ONLY in the activity tab — count it there via
	// the raw-line read seam (byte-deterministic, no ANSI/clipping).
	wedgeLineCount := func() int {
		n := 0
		for _, ln := range d.m.ActivityLines() {
			if strings.Contains(ln, "boss turn wedged") {
				n++
			}
		}
		return n
	}
	// and the transcript-off invariant: zero chat rows, any meta class.
	chatWedgeRows := func() int {
		n := 0
		for _, c := range d.m.State().Chat {
			if strings.Contains(c.Text, "boss turn wedged") {
				n++
			}
		}
		return n
	}

	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — stuck stub online"})
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user", "ship the report", false)})
	// 200ms: the boss goes busy… and never completes (silence from here).
	// The send-side placeholder is client staging — it never arms the W1
	// watchdog's wall clock — so stage ONE real server-side beat as the
	// turn's last proof of life: a boss EvThought. (A "bossmsg-" stream
	// delta would arm the clock too, but the reducer consumes the boss-N
	// placeholder into it — leg B NEEDS that placeholder outstanding to
	// collapse at /stop.)
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	d.send(state.Event{Kind: state.EvThought, EmployeeID: "boss", EmployeeName: "boss",
		CallID: "wk-1", Text: "opening the report sources…", Done: true})
	time.Sleep(60 * time.Millisecond) // past the 30ms harness threshold
	d.pump(1)                         // the watchdog evaluates on the tick cheap loop

	if n := wedgeLineCount(); n != 1 {
		return fail("stuck A: wedge note must land in the activity tab exactly once after the threshold, got %d", n)
	}
	if n := chatWedgeRows(); n != 0 {
		return fail("stuck A: the wedge note must NEVER land in the transcript, got %d chat rows", n)
	}
	frameA := d.m.Frame()
	fmt.Println("===== UI SHOT · STUCK A — watchdog fired: transcript stays CLEAN (no wedge row), red hint swap on the status bar (turn still pending, never auto-killed) =====")
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")
	for _, want := range []string{"boss looks wedged", "is typing…"} {
		if !strings.Contains(frameA, want) {
			return fail("stuck A: frame missing %q", want)
		}
	}
	if strings.Contains(ansi.Strip(frameA), "boss turn wedged") {
		return fail("stuck A: the transcript frame must NOT show the wedge note")
	}
	var wedgeLine string
	for _, ln := range d.m.ActivityLines() {
		if strings.Contains(ln, "boss turn wedged") {
			wedgeLine = ln
		}
	}
	if !strings.Contains(wedgeLine, "/stop unwinds it (queue intact); the turn may still complete on its own") {
		return fail("stuck A: wedge line copy drifted, got %q", wedgeLine)
	}
	if !d.m.SelectTab("activity") {
		return fail("stuck A: activity tab not selectable")
	}
	frameAct := d.m.Frame()
	fmt.Println("===== UI SHOT · STUCK A (activity) — the wedge note's one home: a single dim-timestamped line =====")
	fmt.Println(frameAct)
	fmt.Println("===== UI SHOT =====")
	if !strings.Contains(ansi.Strip(frameAct), "boss turn wedged") {
		return fail("stuck A: the activity tab frame must show the wedge line")
	}
	if !d.m.SelectTab("chat") {
		return fail("stuck A: could not switch back to the chat tab")
	}
	d.pump(3) // one-shot latch: silence past the note
	if n := wedgeLineCount(); n != 1 {
		return fail("stuck A: the wedge note is one-shot per wedge, got %d lines after more ticks", n)
	}
	if n := chatWedgeRows(); n != 0 {
		return fail("stuck A: still zero chat wedge rows after more ticks, got %d", n)
	}

	// Leg 2 — /stop with the abort RPC FAILING remotely (G1).
	typeIn("/stop")
	drainCmd(d, key(tea.KeyEnter), 0)
	drainCmd(d, key(tea.KeyEnter), 0)

	if len(stub.abortLog) != 1 {
		return fail("stuck B: expected exactly 1 AbortSessions call, got %d", len(stub.abortLog))
	}
	st := d.m.State()
	frameB := d.m.Frame()
	fmt.Println("===== UI SHOT · STUCK B — /stop unwound DESPITE the failed abort RPC: placeholder collapsed, dim failure note, watchdog re-armed =====")
	fmt.Println(frameB)
	fmt.Println("===== UI SHOT =====")
	seenStopped, seenAbortNote := false, false
	for _, c := range st.Chat {
		if c.From == "office" && c.Text == "stopped by user" {
			seenStopped = true
		}
		if c.From == "office" && c.Meta == "" && strings.Contains(c.Text, "abort signal failed remotely") {
			seenAbortNote = true
		}
	}
	if !seenStopped {
		return fail("stuck B: the wedged placeholder must still collapse to \"stopped by user\"")
	}
	if !seenAbortNote {
		return fail("stuck B: the failed abort must print exactly one dim note (never the old stranded early-return)")
	}
	if want := "stopped current work — queue intact (0 items)"; st.StatusLine != want {
		return fail("stuck B: StatusLine = %q, want %q", st.StatusLine, want)
	}
	if strings.Contains(frameB, "boss looks wedged") {
		return fail("stuck B: /stop closes the turn — the wedge hint must retire with it")
	}
	d.pump(2) // nothing outstanding → the watchdog stays disarmed
	if n := wedgeLineCount(); n != 1 {
		return fail("stuck B: no new wedge lines after the unwind, got %d", n)
	}
	fmt.Println("--- stub capture (AbortSessions returned the stubbed error) ---")
	for _, ln := range stub.abortLog {
		fmt.Println(ln)
	}
	fmt.Println("asserts: OK — ONE \"boss turn wedged\" line in the activity tab (ZERO transcript rows) + hint swap at the seamed threshold (one-shot); /stop with AbortSessions FAILING still unwinds (placeholder → \"stopped by user\", dim abort-failure note, \"stopped current work\" statusline, watchdog re-armed, queue intact)")
	return nil
}

// --- concierge routing proof (--concierge) ----------------------------------
// Synchronous driver (no wall clock), two phases. Phase A: the boss is
// busy mid-turn (pending placeholder); TWO user sends must BOTH route to
// the stub's SendConcierge (captured), a dim "office routed: boss busy →
// concierge" line appears EXACTLY ONCE, the office placeholders read
// "office is answering…", and the agents roster pins "office (concierge)
// answering". The completions then pin in place (replace-by-ID) and the
// roster word settles to "on call". Phase B: after the boss turn completes
// the next send hits the boss's Send and the concierge is NOT touched
// (zero duplication — the concierge never answers when the boss is idle).

// driveConciergeCmd executes a cmd chain synchronously: each message's
// Update can RETURN the next cmd (busySendReqMsg → the concierge send
// closure → conciergeSentMsg). Timer arms (spinner/tick) are skipped after
// a short wait — the proof only needs produced messages. The concierge
// emit lands synchronous events mid-send (stub.emit → d.send).
func driveConciergeCmd(d *focusDriver, cmd tea.Cmd, depth int) {
	if cmd == nil || depth > 16 {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(150 * time.Millisecond):
		return // a timer arm: not a message the proof needs
	}
	if msg == nil {
		return
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			driveConciergeCmd(d, c, depth+1)
		}
		return
	}
	tm, next := d.m.Update(msg)
	if fm, ok := tm.(app.Model); ok {
		d.m = fm
	}
	driveConciergeCmd(d, next, depth+1)
}

func runConciergeProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	base := &stubBackend{done: make(chan struct{})}
	stub := &conciergeStub{stubBackend: base}
	m := app.New(stub, config.Default())
	d := &focusDriver{m: m}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	var trace []string
	base.trace = func(line string) { trace = append(trace, line) }
	base.emit = func(ev state.Event) { d.send(ev) }
	app.QueueDebugf = func(format string, args ...any) {
		trace = append(trace, "[queue] "+fmt.Sprintf(format, args...))
	}
	defer func() { app.QueueDebugf = nil }()

	key := func(code rune) tea.Cmd {
		tm, c := d.m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
		if fm, ok := tm.(app.Model); ok {
			d.m = fm
		}
		return c
	}
	typeIn := func(s string) {
		for _, r := range s {
			tm, _ := d.m.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
			if fm, ok := tm.(app.Model); ok {
				d.m = fm
			}
		}
	}

	// setup: the boss is BUSY mid-turn (send + pending placeholder staged)
	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — concierge stub online"})
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user", "plan the api", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})

	// PHASE A — two user sends while the boss is busy: BOTH route to the
	// concierge; the routing notice prints ONCE.
	typeIn("what did we decide on storage")
	driveConciergeCmd(d, key(tea.KeyEnter), 0)
	typeIn("and on caching")
	driveConciergeCmd(d, key(tea.KeyEnter), 0)

	fmt.Println("===== UI SHOT · CONCIERGE A — boss busy: both sends ROUTED to the concierge (placeholder ×2, notice ONCE) =====")
	frameA := d.m.Frame()
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")

	if len(stub.conciergeLog) != 2 {
		return fail("concierge A: expected 2 SendConcierge calls, got %d (%v)", len(stub.conciergeLog), stub.conciergeLog)
	}
	if stub.conciergeLog[0] != "what did we decide on storage" || stub.conciergeLog[1] != "and on caching" {
		return fail("concierge A: routed prompts captured wrong: %v", stub.conciergeLog)
	}
	if len(base.sendLog) != 0 {
		return fail("concierge A: the boss's Send fired while busy+routed (%v) — zero-duplication violated", base.sendLog)
	}
	if n := strings.Count(frameA, "office routed: boss busy → concierge"); n != 1 {
		return fail("concierge A: routing notice appears %d times (want exactly 1 per busy turn)", n)
	}
	if !strings.Contains(frameA, "office is answering…") {
		return fail("concierge A: missing the pending office placeholder \"office is answering…\"")
	}
	if !d.m.SelectTab("agents") {
		return fail("concierge A: agents tab not selectable")
	}
	frameAgentsA := d.m.Frame()
	fmt.Println("===== UI SHOT · CONCIERGE A (agents) — pinned \"office (concierge) answering\" under the boss =====")
	fmt.Println(frameAgentsA)
	fmt.Println("===== UI SHOT =====")
	for _, want := range []string{"office (concierge)", "answering"} {
		if !strings.Contains(frameAgentsA, want) {
			return fail("concierge A (agents): frame missing %q", want)
		}
	}

	// the concierge's answers pin in place (replace-by-ID, Pending→false)
	d.send(state.Event{Kind: evChatOffice, Msg: officeMsg("office-1",
		"storage: **sqlite** — the brief says local-first, zero setup.", false)})
	d.send(state.Event{Kind: evChatOffice, Msg: officeMsg("office-2",
		"caching: one in-process map for now — no redis until v3.", false)})
	if !d.m.SelectTab("chat") {
		return fail("concierge A: chat tab not selectable")
	}
	fmt.Println("===== UI SHOT · CONCIERGE A′ — both office answers PINNED in place (one bubble each, INFO \"office ›\" label) =====")
	frameA2 := d.m.Frame()
	fmt.Println(frameA2)
	fmt.Println("===== UI SHOT =====")
	if !strings.Contains(frameA2, "sqlite") {
		return fail("concierge A′: first office answer missing its pinned text")
	}
	if !strings.Contains(frameA2, "no redis until v3") && !ansiContains(frameA2, "redis") {
		return fail("concierge A′: second office answer missing its pinned text")
	}
	if !d.m.SelectTab("agents") {
		return fail("concierge A′: agents tab not selectable")
	}
	frameAgentsA2 := d.m.Frame()
	for _, want := range []string{"office (concierge)", "on call"} {
		if !strings.Contains(frameAgentsA2, want) {
			return fail("concierge A′ (agents): frame missing %q (settled roster)", want)
		}
	}
	if strings.Contains(frameAgentsA2, "answering") {
		return fail("concierge A′ (agents): roster still says \"answering\" after both pins")
	}
	if !d.m.SelectTab("chat") {
		return fail("concierge A′: chat tab not selectable")
	}

	// PHASE B — the boss turn completes; the next send hits the boss's
	// Send and the concierge is NOT touched.
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss",
		"api plan pinned — sqlite plus the in-process cache.", false)})
	base.emit = nil // the Send leg's 600ms echo would race the frame asserts
	typeIn("make it so")
	driveConciergeCmd(d, key(tea.KeyEnter), 0)
	fmt.Println("===== UI SHOT · CONCIERGE B — boss idle again: the send hits the BOSS, the concierge stays silent (zero duplication) =====")
	frameB := d.m.Frame()
	fmt.Println(frameB)
	fmt.Println("===== UI SHOT =====")
	if len(base.sendLog) != 1 || base.sendLog[0] != "make it so" {
		return fail("concierge B: expected exactly 1 boss Send (\"make it so\"), got %v", base.sendLog)
	}
	if len(stub.conciergeLog) != 2 {
		return fail("concierge B: the concierge answered while the boss was idle (%v) — zero-duplication violated", stub.conciergeLog)
	}

	fmt.Println("--- concierge capture (stub SendConcierge calls) ---")
	for i, s := range stub.conciergeLog {
		fmt.Printf("SendConcierge #%d: %q\n", i+1, s)
	}
	fmt.Println("--- ordering trace ---")
	for _, ln := range trace {
		fmt.Println(ln)
	}
	fmt.Println("asserts: OK — boss busy: 2 sends routed to SendConcierge (captured), boss Send never fired, routing notice printed ONCE, office placeholders \"office is answering…\", answers pinned replace-by-ID (INFO \"office ›\" bubbles), agents row \"answering\" → \"on call\"; boss idle: send hit the boss's Send, concierge untouched")
	return nil
}

// ansiContains is Contains on the ansi-stripped frame (glamour splits
// styled text across escape sequences).
func ansiContains(frame, sub string) bool {
	return strings.Contains(ansi.Strip(frame), sub)
}

// printAskCapture prints the stub's captured AnswerQuestion/RejectQuestion
// calls and FAILS the run when the expected capture is missing (the
// deadlock regression: the answer must never fall through to Send).
func printAskCapture(capture []string, want string) {
	fmt.Println("--- stub capture log ---")
	found := false
	for _, line := range capture {
		fmt.Println(line)
		if strings.Contains(line, want) {
			found = true
		}
	}
	if len(capture) == 0 {
		fmt.Println("<empty>")
	}
	if !found {
		fmt.Fprintf(os.Stderr, "uishot: expected capture %q missing\n", want)
		os.Exit(1)
	}
}

// --- power-governor proof ---------------------------------------------------
// The standard shots run under tea.WithoutRenderer (no View calls until the
// final Frame). The power proof needs the caches exercised per rendered
// frame, so it drives the REAL model in a manual event loop: backend emit
// → channel, every Update feeds a Frame() render pass (that is what the
// floor/app caches count), tea.Tick re-arms land on their governor delay.

// runManualLoop drives model+cmd execution by hand for `dur`, then returns
// the final model. Every processed message renders one frame through the
// real Frame() path (cache-exercising), exactly like the bubbletea runtime.
func runManualLoop(cfg *config.Config, b *stubBackend, tab string, dur time.Duration,
	workload func(send func(tea.Msg))) (app.Model, error) {
	m := app.New(b, cfg)
	var zero app.Model
	if tab != "" && !m.SelectTab(tab) {
		return zero, fmt.Errorf("unknown tab %q", tab)
	}
	msgCh := make(chan tea.Msg, 512)
	exec := func(c tea.Cmd) {
		if c == nil {
			return
		}
		go func() {
			if msg := c(); msg != nil {
				msgCh <- msg
			}
		}()
	}
	var tm tea.Model = m
	exec(tm.Init())
	var cmd tea.Cmd
	tm, cmd = tm.Update(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	exec(cmd)
	if err := b.Start(func(ev state.Event) { msgCh <- ev }); err != nil {
		return zero, err
	}
	if workload != nil {
		workload(func(msg tea.Msg) { msgCh <- msg })
	}
	deadline := time.After(dur)
	for {
		select {
		case <-deadline:
			if fm, ok := tm.(app.Model); ok {
				return fm, nil
			}
			return zero, fmt.Errorf("unexpected model type %T", tm)
		case msg := <-msgCh:
			if bmsg, ok := msg.(tea.BatchMsg); ok {
				for _, sc := range bmsg {
					exec(sc)
				}
				continue
			}
			tm, cmd = tm.Update(msg)
			exec(cmd)
			if fm, ok := tm.(app.Model); ok {
				fm.Frame() // render pass — exercises digest + floor caches
			}
		}
	}
}

// scriptPowerDemo (--power slash leg) — minimal quiet script: one user
// message and a boss typing placeholder that NEVER completes inside the
// window, so the busy placeholder ("jorge is typing…") and the slash
// notices share the final frame.
func (b *stubBackend) scriptPowerDemo(at func(ms int, ev state.Event)) {
	at(50, state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — power-governor stub online"})
	at(150, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"ship the power governor", false)})
	at(200, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
}

// printTickTable — the deterministic TickDelay decision table: synthetic
// busy/idle/drift states across every power mode, plus the tickMs override.
func printTickTable() {
	idleSt := state.OfficeState{Employees: []state.Employee{
		{ID: "manager", Name: "boss", Role: state.RoleManager, Sprite: state.SpriteAtDesk},
		{ID: "hr", Name: "hr", Role: state.RoleHR, Sprite: state.SpriteAtDesk},
	}}
	busySt := state.OfficeState{
		Employees: idleSt.Employees,
		Chat:      []state.ChatMsg{{ID: "boss-1", From: "boss", Pending: true}},
	}
	fmt.Println("--- TickDelay decision table (synthetic states) ---")
	rows := []struct {
		name string
		cfg  *config.Config
	}{
		{"auto", config.Default()},
		{"performance", config.Default()},
		{"saver", config.Default()},
		{"auto+tickMs=50", config.Default()},
	}
	rows[1].cfg.UI.Power = config.PowerPerformance
	rows[2].cfg.UI.Power = config.PowerSaver
	rows[3].cfg.UI.TickMs = 50
	want := []struct{ busy, idle, drift time.Duration }{
		{180 * time.Millisecond, 1 * time.Second, 3 * time.Second},
		{150 * time.Millisecond, 150 * time.Millisecond, 150 * time.Millisecond},
		{400 * time.Millisecond, 2 * time.Second, 2 * time.Second},
		{50 * time.Millisecond, 1 * time.Second, 3 * time.Second},
	}
	fail := false
	for idx, r := range rows {
		b := app.TickDelay(busySt, r.cfg, false, false, 0)
		idleDelay := app.TickDelay(idleSt, r.cfg, false, false, 0)
		d := app.TickDelay(idleSt, r.cfg, false, false, 61*time.Second)
		ok := b == want[idx].busy && idleDelay == want[idx].idle && d == want[idx].drift
		mark := "PASS"
		if !ok {
			mark = "FAIL"
			fail = true
		}
		fmt.Printf("  [%s] %-15s busy=%-6s idle=%-6s drift(61s)=%-6s\n", mark, r.name, b, idleDelay, d)
	}
	if fail {
		fmt.Fprintln(os.Stderr, "uishot: TickDelay decision table mismatch")
		os.Exit(1)
	}
	fmt.Println("asserts: OK — auto 180ms/1s/3s-drift, performance 150ms flat, saver 400ms/2s, tickMs overrides busy")
}

// powerWindow — one power mode's scripted-window tallies.
type powerWindow struct {
	ticks                int
	floorHits, floorMiss uint64
	appHits, appMiss     uint64
}

func runPowerProof(mode string) error {
	// brain.json write-through lands in a scratch THEBORINGOFFICE_HOME — the user's
	// real config is never touched by shots.
	home, err := os.MkdirTemp("", "theboringoffice-power")
	if err != nil {
		return err
	}
	defer os.RemoveAll(home)
	if err := os.Setenv("THEBORINGOFFICE_HOME", home); err != nil {
		return err
	}
	fmt.Printf("--- scratch THEBORINGOFFICE_HOME: %s ---\n", home)

	var modes []config.PowerMode
	switch config.PowerMode(mode) {
	case config.PowerAuto, config.PowerSaver, config.PowerPerformance:
		modes = []config.PowerMode{config.PowerMode(mode)}
	case "all":
		modes = []config.PowerMode{config.PowerAuto, config.PowerSaver, config.PowerPerformance}
	default:
		return fmt.Errorf("unknown power mode %q (auto|saver|performance|all)", mode)
	}

	const window = 6 * time.Second
	fmt.Printf("--- power windows (%s scripted quiet-after-burst window, same stub script per mode) ---\n", window)
	byMode := map[config.PowerMode]powerWindow{}
	for _, pm := range modes {
		cfg := config.Default()
		cfg.UI.Power = pm
		b := &stubBackend{done: make(chan struct{}), flushQueue: true}
		office.CacheReset()
		fm, err := runManualLoop(cfg, b, "agents", window, nil)
		if err != nil {
			return err
		}
		fh, fMiss := office.CacheStats()
		ah, aMiss := fm.FrameCacheStats()
		w := powerWindow{ticks: fm.Ticks(), floorHits: fh, floorMiss: fMiss, appHits: ah, appMiss: aMiss}
		byMode[pm] = w
		hitPct := 0.0
		if fh+fMiss > 0 {
			hitPct = 100 * float64(fh) / float64(fh+fMiss)
		}
		avg := window / time.Duration(w.ticks)
		fmt.Printf("  mode=%-11s ticks=%2d avg-delay=%7s  floor-cache: hits=%3d misses=%3d hit=%05.1f%%  app-frame: hits=%3d misses=%3d\n",
			pm, w.ticks, avg.Round(time.Millisecond), fh, fMiss, hitPct, ah, aMiss)
	}
	if len(modes) == 3 {
		a, s, p := byMode[config.PowerAuto], byMode[config.PowerSaver], byMode[config.PowerPerformance]
		if !(p.ticks > a.ticks && a.ticks > s.ticks) {
			return fmt.Errorf("tick ordering violated: want performance > auto > saver, got %d / %d / %d",
				p.ticks, a.ticks, s.ticks)
		}
		fmt.Printf("asserts: OK — performance(%d) > auto(%d) > saver(%d) ticks in the identical window\n",
			p.ticks, a.ticks, s.ticks)
	}
	if config.PowerMode(mode) != "all" {
		return nil
	}

	printTickTable()

	// slash /power + /model leg: busy typing placeholder carries the custom
	// boss short name; slash notices + brain.json write-through in-frame.
	//
	// Driven SYNCHRONOUSLY (the --slashpop idiom: focusDriver + drainCmd) —
	// real KeyPress messages through the REAL app model with zero wall clock.
	// The old version's real-sleep typing goroutines over a fixed 2400ms
	// window flaked because Enter on a filtered fragment is claimed by the
	// slash popover (slashPicked: pick + PREFILL, never auto-send), so which
	// draft the next line's keys landed on was scheduling luck — the misses
	// sent "/power /power saver" ("unknown mode") and no window, widened or
	// not, repairs that. The picker's REAL path stays exercised here, in a
	// fixed order: "/" opens the popover, the fragment filters it live,
	// Enter prefills "/power " / "/model " (the human flow), the argument
	// types next, and the second Enter commits through the plain slash send
	// (onSend → slashMsg → applySlash). Every step lands before the next
	// starts, so the asserts below are deterministic — without any window
	// widening, let alone a 10s wall.
	cfg := config.Default()
	cfg.Boss.Name = "jorge (El Jefe)"
	fm := app.New(&stubBackend{done: make(chan struct{})}, cfg)
	if !fm.SelectTab("chat") {
		return fmt.Errorf("slash leg: chat tab not found")
	}
	d := &focusDriver{m: fm}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	// the powerDemo script's three payloads (status banner, one user line,
	// the never-completing boss typing placeholder), fed without the clock.
	(&stubBackend{}).scriptPowerDemo(func(_ int, ev state.Event) { d.send(ev) })
	typeIn := func(s string) {
		for _, r := range s {
			d.send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		}
	}
	key := func(code rune) tea.Cmd {
		tm, c := d.m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
		if m, ok := tm.(app.Model); ok {
			d.m = m
		}
		return c
	}
	typeIn("/power")
	drainCmd(d, key(tea.KeyEnter), 0) // popover picks /power → draft prefilled "/power "
	typeIn("saver")
	drainCmd(d, key(tea.KeyEnter), 0) // sends "/power saver" → power=saver + persisted
	typeIn("/model")
	drainCmd(d, key(tea.KeyEnter), 0) // popover picks /model → draft prefilled "/model "
	typeIn("anthropic/claude-haiku-4-5")
	drainCmd(d, key(tea.KeyEnter), 0) // sends "/model <ref>" → boss.model set + persisted
	fm = d.m
	fmt.Println("===== UI SHOT · slash /power + /model (boss.name \"jorge (El Jefe)\", boss typing) =====")
	fmt.Println(fm.Frame())
	fmt.Println("===== UI SHOT =====")
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	if got := fm.Config(); got.UI.Power != config.PowerSaver {
		return fail("expected in-memory power=saver after /power saver, got %q", got.UI.Power)
	}
	if got := fm.Config(); got.Boss.Model != "anthropic/claude-haiku-4-5" {
		return fail("expected in-memory boss.model after /model, got %q", got.Boss.Model)
	}
	bts, rerr := os.ReadFile(config.Path())
	if rerr != nil {
		return fail("read persisted brain.json: %v", rerr)
	}
	if !strings.Contains(string(bts), `"power": "saver"`) || !strings.Contains(string(bts), `"model": "anthropic/claude-haiku-4-5"`) {
		return fail("persisted brain.json missing the /power + /model writes:\n%s", bts)
	}
	fmt.Printf("--- persisted brain.json (%s) ---\n%s", config.Path(), bts)
	fmt.Println("asserts: OK — /power saver honored + persisted, /model set + persisted, placeholder personalized")

	// custom-boss-name leg: the agents roster pins cfg.Boss.Name.
	nb := &stubBackend{done: make(chan struct{})}
	nfm, err := runManualLoop(cfg, nb, "agents", 1600*time.Millisecond, nil)
	if err != nil {
		return err
	}
	frame := nfm.Frame()
	if !strings.Contains(frame, "jorge (El Jefe)") {
		return fail("agents frame missing custom boss name")
	}
	fmt.Println("===== UI SHOT · custom boss name on the agents roster =====")
	fmt.Println(frame)
	fmt.Println("===== UI SHOT =====")
	return nil
}

// --- social-clock proof -----------------------------------------------------
// Deterministic: the model is pumped with SYNCHRONOUS EvTick updates (no
// tea.Program, no wall clock) and the SocialClock seeds its PRNG from
// tick+seq, so a repetition replays bit-for-bit — except package-global
// office walker state across reps, neutralized by prefixing employee IDs
// per rep (identical NAMES/seats/glyphs; the frame never shows the IDs).

// socialDriver — minimal synchronous model pump (one EvTick per step, the
// returned tea.Cmd is the tick re-arm timer; not needed here).
type socialDriver struct {
	m app.Model
}

func newSocialDriver(rep int) *socialDriver {
	backend := &stubBackend{done: make(chan struct{})} // Mode() only — no script
	m := app.New(backend, config.Default())
	d := &socialDriver{m: m}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	pref := fmt.Sprintf("soc%d", rep)
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: pref + "-dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: pref + "-sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: pref + "-rev-1", Name: "dikastes", Role: state.RoleReviewer, Sprite: state.SpriteAtDesk}})
	d.m.Frame() // lock the floor plan before any sprite advance
	return d
}

func (d *socialDriver) send(msg tea.Msg) {
	tm, _ := d.m.Update(msg)
	if fm, ok := tm.(app.Model); ok {
		d.m = fm
	}
}

func (d *socialDriver) pump(n int) {
	for i := 0; i < n; i++ {
		d.send(state.Event{Kind: state.EvTick})
	}
}

func (d *socialDriver) pumpUntil(desc string, maxTicks int, cond func(state.OfficeState) bool) error {
	for i := 0; i < maxTicks; i++ {
		if cond(d.m.State()) {
			return nil
		}
		d.pump(1)
	}
	return fmt.Errorf("social: %s did not happen within %d ticks (state: %s, trace missing)",
		desc, maxTicks, d.m.State().StatusLine)
}

func hasBubbleContaining(st state.OfficeState, sub string) bool {
	for _, b := range st.Bubbles {
		if strings.Contains(b.Text, sub) {
			return true
		}
	}
	return false
}

func countWalkingToCoffee(st state.OfficeState) int {
	n := 0
	for _, e := range st.Employees {
		if e.Sprite == state.SpriteToCoffee || e.Sprite == state.SpriteCoffee {
			n++
		}
	}
	return n
}

// socialFramesSim — the three-printed-frames run: hires, then the forced
// tea request (frame A: the ask bubble; frame B: both sprites walking),
// then the forced gossip chain (frame C: all three beats live). Also
// captures the forced gossip/banter chain traces (speaker › line).
func socialFramesSim(rep int) (frames [3]string, banter, gossip []string, err error) {
	var trace []string
	app.SocialTracef = func(format string, args ...any) {
		trace = append(trace, fmt.Sprintf(format, args...))
	}
	defer func() { app.SocialTracef = nil; app.SocialForceRoll = nil }()
	roll := 0
	app.SocialForceRoll = &roll

	d := newSocialDriver(rep)

	// SOCIAL A — tea request (roll 50): the ask bubble by A.
	roll = 50
	if err := d.pumpUntil("tea ask bubble", 1500, func(st state.OfficeState) bool {
		return hasBubbleContaining(st, "coffee?")
	}); err != nil {
		return frames, nil, nil, err
	}
	roll = -1 // force NOTHING while the sequence plays out (frames stay clean)
	frames[0] = d.m.Frame()

	// SOCIAL B — co-walking: A (t+2) then B (t+6) drift to the machine.
	if err := d.pumpUntil("both walkers to the tea machine", 200, func(st state.OfficeState) bool {
		return countWalkingToCoffee(st) >= 2
	}); err != nil {
		return frames, nil, nil, err
	}
	frames[1] = d.m.Frame()

	// SOCIAL C — gossip chain (roll 70): the PLAN emits its 3 trace lines at
	// once (plan time), so wait for the plan, then pump until the third
	// beat's tick (t0, +5, +10) has landed (+1) before freezing the frame.
	roll = 70
	if err := d.pumpUntil("gossip chain plan", 2500, func(st state.OfficeState) bool {
		n := 0
		for _, ln := range trace {
			if strings.HasPrefix(ln, "gossip: ") {
				n++
			}
		}
		return n >= 3
	}); err != nil {
		return frames, nil, nil, err
	}
	for _, ln := range trace {
		if strings.HasPrefix(ln, "gossip: ") {
			gossip = append(gossip, ln)
		}
	}
	roll = -1
	d.pump(12) // beats at +0/+5/+10 all armed and rendered
	frames[2] = d.m.Frame()

	// banter (roll 10): capture the chain for the PROOF section.
	roll = 10
	banterMark := len(trace)
	if err := d.pumpUntil("banter pair dialog", 2500, func(st state.OfficeState) bool {
		n := 0
		for _, ln := range trace {
			if strings.HasPrefix(ln, "banter: ") {
				n++
			}
		}
		return n >= 2
	}); err != nil {
		return frames, nil, nil, err
	}
	for _, ln := range trace[banterMark:] {
		if strings.HasPrefix(ln, "banter: ") {
			banter = append(banter, ln)
		}
	}
	return frames, banter, gossip, nil
}

// socialModalGateSim — the scripted-modal assert: a boss question opens the
// modal; across 400 pumped ticks NO social beat may fire (no trace, no
// bubbles, no walkers). After the modal resolves, the clock must resume
// (forced tea request lands).
func socialModalGateSim() error {
	var trace []string
	app.SocialTracef = func(format string, args ...any) {
		trace = append(trace, fmt.Sprintf(format, args...))
	}
	defer func() { app.SocialTracef = nil; app.SocialForceRoll = nil }()

	d := newSocialDriver(99)
	base := len(d.m.State().Employees)
	d.send(state.Event{Kind: state.EvQuestion, EmployeeName: "boss", QuestionID: "q-soc",
		Text: "ship tonight or tomorrow?", ToolSummary: "tonight | tomorrow"})
	d.pump(400)
	st := d.m.State()
	if len(trace) != 0 {
		return fmt.Errorf("social fired while the question modal was open: %q", trace)
	}
	if len(st.Bubbles) != 0 {
		return fmt.Errorf("social bubble appeared while the question modal was open: %+v", st.Bubbles)
	}
	if n := countWalkingToCoffee(st); n != 0 {
		return fmt.Errorf("sprite walked to coffee while the question modal was open (%d walkers)", n)
	}
	if len(st.Employees) != base {
		return fmt.Errorf("roster changed unexpectedly (%d -> %d)", base, len(st.Employees))
	}
	// gate lifts when the modal closes
	d.send(state.Event{Kind: state.EvQuestion, EmployeeName: "boss", QuestionID: "q-soc",
		ToolSummary: "answered", ToolState: "resolved"})
	roll := 50
	app.SocialForceRoll = &roll
	if err := d.pumpUntil("social resume after modal close", 1500, func(st state.OfficeState) bool {
		return hasBubbleContaining(st, "coffee?")
	}); err != nil {
		return err
	}
	fmt.Println("  modal gate: PASS — 400 ticks, no social beat while the question modal was open; resumed after resolve")
	return nil
}

func runSocialProof() error {
	if err := socialModalGateSim(); err != nil {
		return err
	}
	frames1, banter1, gossip1, err := socialFramesSim(1)
	if err != nil {
		return err
	}
	frames2, banter2, gossip2, err := socialFramesSim(2)
	if err != nil {
		return err
	}
	// determinism: tick-seeded — the two script runs must be byte-identical.
	for i := 0; i < 3; i++ {
		if frames1[i] != frames2[i] {
			return fmt.Errorf("social: frame %d differs between tick-seeded runs", i+1)
		}
	}
	if strings.Join(banter1, "\n") != strings.Join(banter2, "\n") ||
		strings.Join(gossip1, "\n") != strings.Join(gossip2, "\n") {
		return fmt.Errorf("social: banter/gossip chains differ between tick-seeded runs")
	}
	labels := [3]string{
		"SOCIAL A — tea request: A asks at their desk («<B>: coffee?»)",
		"SOCIAL B — co-walking: both sprites heading to the tea machine",
		"SOCIAL C — gossip chain: three bubbles fired over time, absent third named",
	}
	for i := 0; i < 3; i++ {
		fmt.Printf("===== UI SHOT · %s =====\n", labels[i])
		fmt.Println(frames1[i])
		fmt.Println("===== UI SHOT =====")
	}
	fmt.Println("--- gossip chain (3 beats, absent third named) ---")
	for _, ln := range gossip1 {
		fmt.Println("  " + ln)
	}
	fmt.Println("--- banter chain (one pair dialog, role-banked) ---")
	for _, ln := range banter1 {
		fmt.Println("  " + ln)
	}
	fmt.Println("asserts: OK — modal gate held, tea co-walk fired, gossip 3-beat chain fired, tick-seeded runs byte-identical")
	return nil
}

// --- layout-modes proof (--layout) ------------------------------------------
// THREE frames over the identical scripted window + identical config base,
// differing ONLY by the layout knobs. The expectations pin the CURRENT
// geometry contract (internal/app/model.go resize/sidebarBase; the same 80
// default is pinned app-side by mobile_test.go's LayoutInfo check):
//   - NORMAL  → defaultSidebarW = 80 (bcb1635 "ui: default sidebar 80 cols";
//     the proof's stale 44 predates the 44→68 (0337ec1) → 80 (bcb1635)
//     widenings — the proof's job is pinning the layout MODES to truth,
//     not embalming the old numbers)
//   - compact → compactSidebarW = 30 (ui.compact: short tab labels, 2-row
//     chat input, compressed topbar)
//   - wide 90 → an explicit ui.sidebarWidth (/wide) wins outright over the
//     default, clamped to sidebarMin..sidebarMax = 26..100; 90 keeps the
//     leg genuinely WIDE (> 80) and under the ceiling (floor = 40 ≥ 8-min)
//
// At shotCols=130 the narrow-terminal degrade (w/3 min 20 under degradeCols
// 100) and the mobile stack (under mobileMaxCols 100) never fire. Each
// frame prints its computed geometry and passes width/label asserts.

type layoutLeg struct {
	name        string
	mutate      func(cfg *config.Config)
	wantSidebar int
	compact     bool
}

func runLayoutProof() error {
	legs := []layoutLeg{
		// defaultSidebarW = 80 (the bcb1635 default — was 44 when this proof
		// was written, then 68 via 0337ec1, now 80)
		{"NORMAL", func(*config.Config) {}, 80, false},
		// compactSidebarW = 30
		{"compact", func(c *config.Config) { c.UI.Compact = true }, 30, true},
		// explicit /wide: clamped to 26..100, wins over the default; 90 keeps
		// the leg genuinely wider than the 80 default (56 would now NARROW it)
		{"wide 90", func(c *config.Config) { c.UI.SidebarWidth = 90 }, 90, false},
	}
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	for _, leg := range legs {
		cfg := config.Default()
		leg.mutate(cfg)
		b := &stubBackend{done: make(chan struct{})}
		fm, err := runManualLoop(cfg, b, "chat", 2200*time.Millisecond, nil)
		if err != nil {
			return err
		}
		w, h, side, floor := fm.LayoutInfo()
		frame := fm.Frame()
		fmt.Printf("===== UI SHOT · layout %s =====\n", leg.name)
		fmt.Println(frame)
		fmt.Println("===== UI SHOT =====")
		fmt.Printf("computed: cols=%d rows=%d sidebar=%d floorW=%d\n", w, h, side, floor)
		if w != shotCols || h != shotRows {
			return fail("[%s] geometry drift: cols=%d rows=%d (want %dx%d)", leg.name, w, h, shotCols, shotRows)
		}
		if side != leg.wantSidebar {
			return fail("[%s] sidebar=%d, want %d", leg.name, side, leg.wantSidebar)
		}
		if floor != shotCols-leg.wantSidebar {
			return fail("[%s] floorW=%d, want %d (sidebar %d)", leg.name, floor, shotCols-leg.wantSidebar, leg.wantSidebar)
		}
		if leg.compact {
			// the 30-col sidebar drops to single-letter labels — with all
			// SEVEN right-strip tabs the bar fits the BARE tier at 27
			// cells: " c  t  a  b  m  x  g " (b = board; the old eight-tab
			// strip overflowed into the tight letters tier, seven ride
			// padded). Numbers never fit at 30. (Asserted on the STRIPPED
			// frame. The browser is absent BY DESIGN — it rides the LEFT
			// pane's floor|browser switcher, whose full-word labels are
			// legit.)
			plain := ansi.Strip(frame)
			if !strings.Contains(plain, "c   t   a   b   m   x   g") {
				return fail("[compact] tab bar missing the bare letters tier %q", "c   t   a   b   m   x   g")
			}
			if strings.Contains(plain, "chat") || strings.Contains(plain, "terminal") {
				return fail("[compact] tab bar still shows a full tab name")
			}
			if !strings.Contains(plain, "floor") || !strings.Contains(plain, "browser") {
				return fail("[compact] the left slot's floor|browser switcher must ride the frame")
			}
			if strings.Contains(frame, "DEMO") {
				return fail("[compact] topbar still carries the mode segment (should compress to agents + clock)")
			}
			fmt.Println("asserts: OK — sidebar 30, compact tab labels (c   t   a   b   m   x   g bare tier), left floor|browser switcher, compressed topbar, 2-row chat input")
		} else {
			if !strings.Contains(frame, "terminal") || !strings.Contains(frame, "activity") {
				return fail("[%s] tab bar missing full tab labels (want \"terminal\" + \"activity\" visible — six tabs must never clip)", leg.name)
			}
			if !strings.Contains(frame, "DEMO") {
				return fail("[%s] normal topbar lost the mode segment", leg.name)
			}
			fmt.Printf("asserts: OK — sidebar %d, all six full tab labels visible, full topbar (mode segment present)\n", leg.wantSidebar)
		}
	}
	fmt.Println("asserts: OK — 80 (default) / 30 (compact) / 90 (wide 90) sidebars; floor = 130 - sidebar in every leg")
	return nil
}

// --- plan-mode screenshot (--planshot) ---------------------------------------
// TWO frames over a minimal quiet demo script, the conversation-first plan
// flow AND its shape gate: the script delivers status + hires up front and
// stays quiet while the workload presses ctrl+p at 1.5s (the REAL global
// claim site — no terminal focus, no floats); only THEN does the scripted
// chat round run — user ask → typing placeholder → TWO boss CHATTER replies
// (status narration, no markdown structure) → the boss's markdown PLAN
// reply (# Goal / # Steps). Frame 1 (~2.0s) proves the toggle plus BOTH
// chatter completions present NOTHING (chat-first AND gate-first: empty
// pane → office floor keeps the slot, [plan] badge + idle hint up, the
// once-per-session escape-valve note fired exactly once); frame 2 (~3.6s)
// proves the plan-SHAPED reply presented passively into the pane (chat
// keeps focus — the pane footer's "click to edit" is the visible tell).
// The script schedules NO floats (questions/permissions) for the whole
// window: while a float is up ctrl+p is refused (handleKey's claim order),
// so determinism comes from the quiet script.

// scriptPlanDemo (--planshot) — status, three hires, then (AFTER the
// workload's ctrl+p) one chat round: user ask → typing placeholder → TWO
// boss CHATTER replies (work-narration prose that must NEVER present —
// the pane stays hidden and the once-per-session escape-valve note fires
// a single time) followed by the boss's markdown PLAN reply (# Goal /
// # Steps with bullets), then quiet forever. The plan body carries a
// unique marker word ("azimuth") that appears nowhere else in the frame,
// so the contains-asserts pin the pane's adopted content.
func (b *stubBackend) scriptPlanDemo(at func(ms int, ev state.Event)) {
	at(50, state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — plan-mode stub online"})
	at(100, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	at(250, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteAtDesk}})
	at(400, state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "rev-1", Name: "dikastes", Role: state.RoleReviewer, Sprite: state.SpriteAtDesk}})
	at(1600, state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"plan the lobby gallery wall", false)})
	at(1650, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	// (a) CHATTER — status narration, no markdown structure: the pane
	// must stay hidden; the escape valve teaches once.
	at(1700, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bc1", "boss",
		"quick sync — sent to ops; scanning the lobby wall options now, the structured plan lands in a beat.", false)})
	// (a′) MORE chatter — the valve does not refire (anti-spam).
	at(1850, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bc2", "boss",
		"still sketching — matte panels vs glass lanes, comparing sightlines; the plan proper is next.", false)})
	// (b) the plan-shaped reply — presents into the pane.
	at(2400, state.Event{Kind: state.EvChatBoss, Msg: chatMsg("b1", "boss",
		"# Goal\n"+
			"A gallery lobby wall that feels calm, not corporate.\n"+
			"# Steps\n"+
			"- matte panels azimuth-washed along the long east wall\n"+
			"- glassmorphic kanban lanes for the return shelf by the tea machine\n"+
			"- zero clerical chrome anywhere near the entrance doors", false)})
	// quiet: ctrl+p (workload) + this scripted round are the ONLY beats
}

// planShotWorkload — ONE key: ctrl+p at 1.5s, after the hires settled and
// before the scripted chat round begins. The mode flip is ALL the key
// does (chat keeps focus; the pane opens only when the boss's reply
// completes at 2.4s — plan_mode.go's presentation hook).
func planShotWorkload(p *tea.Program) {
	time.Sleep(1500 * time.Millisecond)
	p.Send(tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl})) // the real claim site: no terminal focus, no floats
}

// runPlanShot runs one fresh app+program against the plan-demo stub for
// `dur` while the workload drives ctrl+p + plan typing, then returns the
// final frame (model still IN plan mode — no second toggle).
func runPlanShot(dur time.Duration) (string, error) {
	backend := &stubBackend{done: make(chan struct{}), planDemo: true}
	m := app.New(backend, config.Default())
	if !m.SelectTab("chat") {
		return "", fmt.Errorf("unknown tab %q", "chat")
	}
	p := tea.NewProgram(m,
		tea.WithWindowSize(shotCols, shotRows),
		tea.WithoutRenderer(),
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
	)
	emit := func(ev state.Event) { p.Send(ev) }
	if err := backend.Start(emit); err != nil {
		return "", err
	}
	go planShotWorkload(p)
	go func() {
		time.Sleep(dur)
		p.Quit()
	}()
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	fm, ok := final.(app.Model)
	if !ok {
		return "", fmt.Errorf("unexpected final model type %T", final)
	}
	return fm.Frame(), nil
}

func runPlanProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	assertPresent := func(frame string, wants ...string) error {
		stripped := ansi.Strip(frame)
		for _, want := range wants {
			if !strings.Contains(stripped, want) {
				return fail("planshot: frame missing %q", want)
			}
		}
		return nil
	}
	assertAbsent := func(frame string, rejects ...string) error {
		stripped := ansi.Strip(frame)
		for _, reject := range rejects {
			if strings.Contains(stripped, reject) {
				return fail("planshot: frame must NOT contain %q", reject)
			}
		}
		return nil
	}

	// FRAME 1 — t=2.0s: ctrl+p landed at 1.5s and BOTH boss chatter
	// replies completed (1.7s/1.85s) → plan mode is ACTIVE but chatter
	// never presents (the shape gate), so the pane is EMPTY and the
	// office floor keeps the slot (conversation-first + gate-first).
	frame1, err := runPlanShot(2000 * time.Millisecond)
	if err != nil {
		return err
	}
	fmt.Println("===== UI SHOT · PLAN frame 1/2 — t=2.0s (plan mode ACTIVE, boss CHATTER completed ×2, pane empty+hidden: office floor keeps the slot, [plan] badge + idle plan hint up, escape-valve note ONCE) =====")
	fmt.Println(frame1)
	fmt.Println("===== UI SHOT =====")
	if err := assertPresent(frame1,
		"[=BOSS=]",                    // the office floor still owns the slot
		"[plan]",                      // statusbar agent badge
		"plan · boss plans read-only", // the idle plan hint prefix (statusline)
		"[office] plan mode",          // the toggle's own notice — proof ctrl+p went through the REAL claim site
		"plan the lobby gallery wall", // the scripted chat round still runs in the sidebar
		"quick sync — sent to ops",    // the boss's CHATTER reply, chat-only
		"boss is chatting",            // the escape-valve note (once-per-session teaching)
	); err != nil {
		return err
	}
	if n := strings.Count(ansi.Strip(frame1), "boss is chatting; when it writes a plan it lands on the left"); n != 1 {
		return fail("planshot: the escape-valve note must fire EXACTLY once per session across the two chatter replies, got %d", n)
	}
	if err := assertAbsent(frame1,
		"PLAN · markdown",         // no pane header while the buffer is empty
		"click to edit",           // no pane footer either
		"didn't look like a plan", // the kept-last-plan note never fires over an empty pane
	); err != nil {
		return err
	}

	// FRAME 2 — t=3.6s: the boss's markdown PLAN reply completed at 2.4s
	// and mirrored PASSIVELY into the pane (the gate lets it through):
	// the pane owns the floor slot carrying the boss's text, chat keeps
	// focus (the pane footer's "click to edit" is the visible tell).
	frame2, err := runPlanShot(3600 * time.Millisecond)
	if err != nil {
		return err
	}
	fmt.Println("===== UI SHOT · PLAN frame 2/2 — t=3.6s (plan-SHAPED boss reply PRESENTED: the markdown pane owns the floor slot with the boss's plan text, [plan] badge + click-to-edit hint, chat input still owns focus) =====")
	fmt.Println(frame2)
	fmt.Println("===== UI SHOT =====")
	if err := assertPresent(frame2,
		"PLAN · markdown",      // pane header in the floor slot
		"azimuth",              // the boss's plan text, mirrored into the pane (unique marker)
		"[plan]",               // statusbar agent badge
		"plan · click to edit", // the pane-visible statusline hint prefix
		"click to edit",        // pane footer hint — the pane is UNFOCUSED: focus visibly stays in the chat input
		"Goal",                 // the plan body's adopted heading (glamour renders the '#' away)
		"A gallery lobby wall that feels calm, not", // the plan body itself, adopted into the pane
	); err != nil {
		return err
	}
	// across the whole run the anti-spam contract held: chatter noted the
	// escape valve ONCE, and the kept-last-plan note never fired.
	if n := strings.Count(ansi.Strip(frame2), "boss is chatting"); n != 1 {
		return fail("planshot: the escape-valve note must remain exactly one row after the plan presented, got %d", n)
	}
	if err := assertAbsent(frame2,
		"[=BOSS=]",                // the pane REPLACES the office floor in the slot
		"flowchart",               // the starter template never armed (empty+hidden default; boss text presented instead)
		"didn't look like a plan", // no rejection note ever fired — nothing stale, nothing to explain
	); err != nil {
		return err
	}
	fmt.Println("asserts: OK — ctrl+p went through the real claim site (office notice) and flipped ONLY the mode; TWO boss chatter replies (status narration) NEVER presented (frame 1: floor kept, [plan] badge + idle hint, pane hidden, escape-valve note EXACTLY once, no kept-last-plan note); the plan-SHAPED reply (# Goal / # Steps, glamour-rendered in the pane) presented passively (frame 2: PLAN · markdown + azimuth marker + the adopted Goal heading, floor swapped out, starter template never armed) while chat kept focus (click-to-edit footer, pane-side hint prefixes)")
	return nil
}

// --- terminal-tab proof (--terminal) ----------------------------------------
// The stub TermPanel (uisshot ONLY) wires through app.SpawnTerminal — the
// production wiring point where cmd/theboringoffice plugs panels.NewTerminal.
// Wave-42: the terminal tab's shell capture is OPT-IN — ctrl+space toggles
// it BOTH ways, ctrl+o releases as the alias, RELEASED by default and on
// every (re-)entry — so the proof walks the full toggle flow SYNCHRONOUSLY
// (focusDriver, no wall clock, no real PTY):
//
//	1. RELEASED default — typed letters never reach the "shell" (the stub's
//	   received counter stays put), the released hint rides the status bar,
//	   and a REAL TAB KEY EVENT LEAVES the terminal tab (regression pin:
//	   the retired ctrl+i dive was byte-identical to tab, 0x09, so the
//	   dive key conflicted with tab-to-leave; ctrl+space is 0x00);
//	2. ctrl+space CAPTURED — waved-41: typed chars, tab/shift+tab and
//	   digits ALL reach the shell (the echo: 4 letters → received +4), no
//	   app switch fires;
//	3. ctrl+space RELEASED — the SAME key toggles back out, in place (no
//	   chat hop), letters consumed again; the ctrl+o alias leg releases
//	   the same way;
//	4. auto-release — leaving while captured drops the capture and
//	   re-entering starts RELEASED (explicit opt-in per visit);
//	5. quit hook — CloseTerminal kills the spawned stub shell.
//
// Byte-for-byte determinism: the same synchronous drive runs TWICE and the
// frames must be identical.

// termDrive — ONE full synchronous walk of the toggle flow. Returns the
// four phase frames (released default, captured, post-release, post-lose
// re-entry), the stub, the spawn-call counter and a close hook that drives
// the runtime leak-guard (model.CloseTerminal) while everything is live.
func termDrive() (frames [4]string, stub *terminalPanelStub, calls int, closeFn func(), err error) {
	app.SpawnTerminal = func(cols, rows int) (app.TerminalTab, error) {
		calls++
		stub = newTerminalPanelStub(cols, rows)
		return stub, nil
	}
	backend := &stubBackend{done: make(chan struct{})}
	d := &focusDriver{m: app.New(backend, config.Default())}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	if !d.m.SelectTab("terminal") {
		return frames, nil, 0, nil, fmt.Errorf("unknown tab %q", "terminal")
	}
	closeFn = func() { d.m.CloseTerminal() }
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	key := func(k tea.KeyPressMsg) { d.send(k) }
	press := func(c rune) { key(tea.KeyPressMsg(tea.Key{Code: c, Text: string(c)})) }
	tabK := tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})
	shiftTabK := tea.KeyPressMsg(tea.Key{Code: tea.KeyTab, Mod: tea.ModShift})
	ctrlSpace := tea.KeyPressMsg(tea.Key{Code: tea.KeySpace, Mod: tea.ModCtrl})
	ctrlO := tea.KeyPressMsg(tea.Key{Code: 'o', Mod: tea.ModCtrl})

	// PHASE 1 — RELEASED default: letters consumed, and a REAL TAB key
	// event leaves the tab. This pins the wave-43 regression forever: the
	// old ctrl+i dive was byte-identical to tab (0x09 on non-kitty
	// terminals) — a software CANNOT today make tab toggle the capture.
	press('h') // the first keypress also dismisses the boot splash
	press('i')
	if stub.received != 0 {
		return frames, nil, 0, nil, fail("released default: typed letters reached the shell (%d)", stub.received)
	}
	if got := d.m.ActiveTabIndex(); got != 1 {
		return frames, nil, 0, nil, fail("precondition: must be on the terminal tab, got %d", got)
	}
	frames[0] = d.m.Frame()
	key(tabK) // a real tab event LEAVES the terminal tab (chat·terminal·agents cycle)
	if got := d.m.ActiveTabIndex(); got != 2 {
		return frames, nil, 0, nil, fail("released default: a tab key event must cycle the office (want agents=2) — never a capture toggle, got %d", got)
	}

	// PHASE 2 — back on the terminal (still RELEASED), ctrl+space toggles
	// the dive in: every key rides wave-41 to the shell.
	key(shiftTabK)
	if got := d.m.ActiveTabIndex(); got != 1 {
		return frames, nil, 0, nil, fail("released round-trip: shift+tab must return to the terminal, got %d", got)
	}
	press('x')
	if stub.received != 0 {
		return frames, nil, 0, nil, fail("re-entry must start RELEASED (no memory), got %d shell keys", stub.received)
	}
	key(ctrlSpace)
	if stub.received != 0 {
		return frames, nil, 0, nil, fail("ctrl+space is app-kept — never shell input, got %d", stub.received)
	}
	for _, r := range "echo" {
		press(r)
	}
	if stub.received != 4 {
		return frames, nil, 0, nil, fail("captured: 4 typed letters must echo into the shell, got %d", stub.received)
	}
	frames[1] = d.m.Frame()
	// wave-41 asserted while captured: tab/shift+tab complete, digits type.
	key(tabK)
	stubExpect := 5
	if stub.received != stubExpect || d.m.ActiveTabIndex() != 1 {
		return frames, nil, 0, nil, fail("captured: tab must reach the shell without switching (received=%d tab=%d)",
			stub.received, d.m.ActiveTabIndex())
	}
	key(shiftTabK)
	stubExpect++
	if stub.received != stubExpect || d.m.ActiveTabIndex() != 1 {
		return frames, nil, 0, nil, fail("captured: shift+tab must reach the shell without a Prev (received=%d tab=%d)",
			stub.received, d.m.ActiveTabIndex())
	}
	press('3')
	stubExpect++
	if stub.received != stubExpect || d.m.ActiveTabIndex() != 1 {
		return frames, nil, 0, nil, fail("captured: \"3\" must type into the shell without jumping (received=%d tab=%d)",
			stub.received, d.m.ActiveTabIndex())
	}
	press('7')
	stubExpect++
	if stub.received != stubExpect || d.m.ActiveTabIndex() != 1 {
		return frames, nil, 0, nil, fail("captured: \"7\" must type into the shell without jumping (received=%d tab=%d)",
			stub.received, d.m.ActiveTabIndex())
	}

	// PHASE 3 — the SAME key toggles back OUT in place (no chat hop):
	// ctrl+space releases, letters consumed again.
	key(ctrlSpace)
	if stub.received != stubExpect {
		return frames, nil, 0, nil, fail("the release press of ctrl+space is app-kept — never shell input, got %d", stub.received)
	}
	if got := d.m.ActiveTabIndex(); got != 1 {
		return frames, nil, 0, nil, fail("ctrl+space releases in place — the tab stays the terminal, got %d", got)
	}
	press('z')
	if stub.received != stubExpect {
		return frames, nil, 0, nil, fail("released again: letters must be consumed, got %d", stub.received)
	}
	frames[2] = d.m.Frame()

	// PHASE 4 — dive again via the toggle, release via the ctrl+o ALIAS
	// (release-only, in place), then capture once more, leave captured,
	// auto-release, re-enter released.
	key(ctrlSpace)
	if got := d.m.ActiveTabIndex(); got != 1 {
		return frames, nil, 0, nil, fail("precondition: still the terminal tab, got %d", got)
	}
	key(ctrlO) // the release alias: captured → released, in place
	if stub.received != stubExpect {
		return frames, nil, 0, nil, fail("ctrl+o is app-kept — never shell input, got %d", stub.received)
	}
	if got := d.m.ActiveTabIndex(); got != 1 {
		return frames, nil, 0, nil, fail("ctrl+o releases in place — the tab stays the terminal, got %d", got)
	}
	press('z')
	if stub.received != stubExpect {
		return frames, nil, 0, nil, fail("post-alias released: letters must be consumed, got %d", stub.received)
	}
	key(ctrlSpace) // captured again for the leave-while-captured leg
	if stub.received != stubExpect {
		return frames, nil, 0, nil, fail("ctrl+space is app-kept on re-dive — never shell input, got %d", stub.received)
	}
	if !d.m.SelectTab("agents") { // a click/event-style tab switch while captured
		return frames, nil, 0, nil, fail("agents tab not selectable")
	}
	press('x') // routed while OFF the terminal — must trip the auto-release
	if stub.received != stubExpect {
		return frames, nil, 0, nil, fail("auto-release: a key off-tab must never reach the shell, got %d", stub.received)
	}
	if !d.m.SelectTab("terminal") {
		return frames, nil, 0, nil, fail("terminal tab not selectable")
	}
	press('w')
	if stub.received != stubExpect {
		return frames, nil, 0, nil, fail("re-entry must start RELEASED (no memory of the capture), got %d", stub.received)
	}
	frames[3] = d.m.Frame()
	return frames, stub, calls, closeFn, nil
}

func runTerminalProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	const releasedHint = "office keys · ctrl+space → shell · ctrl+q quit"
	const capturedHint = "typing → shell · ctrl+space release · ctrl+q quit"

	frames1, stub1, calls1, close1, err := termDrive()
	app.SpawnTerminal = nil
	if err != nil {
		return err
	}
	if calls1 != 1 {
		return fail("lazy-spawn violated: SpawnTerminal factory called %d times (want exactly 1, on first visit)", calls1)
	}
	if stub1 == nil {
		return fail("no stub panel was spawned")
	}
	if stub1.received != 8 {
		return fail("key routing drifted: want exactly 8 shell keys (4 letters + tab + shift+tab + 2 digits), got %d", stub1.received)
	}

	labels := [4]string{
		"PHASE 1 — RELEASED default: letters consumed (keys received: 0), released hint rides the bar, a REAL tab event leaves the tab",
		"PHASE 2 — ctrl+space CAPTURED: \"echo\" typed (keys received: 4), captured hint; wave-41 kept (tab/shift+tab/digits → shell)",
		"PHASE 3 — ctrl+space RELEASED in place (the same key toggles OUT, no chat hop): letters consumed again, released hint back",
		"PHASE 4 — ctrl+o alias releases in place; leave-while-captured auto-releases; re-entry starts RELEASED",
	}
	wantHints := [4]string{releasedHint, capturedHint, releasedHint, releasedHint}
	for i := range frames1 {
		fmt.Printf("===== UI SHOT · terminal toggle — %s =====\n", labels[i])
		fmt.Println(frames1[i])
		fmt.Println("===== UI SHOT =====")
		if !strings.Contains(frames1[i], "uisshot STUB shell") {
			return fail("phase %d: frame missing the stub terminal's content marker", i+1)
		}
		if !strings.Contains(frames1[i], "terminal") {
			return fail("phase %d: frame missing the \"terminal\" tab label", i+1)
		}
		if !strings.Contains(frames1[i], wantHints[i]) {
			return fail("phase %d: frame missing the %q hint", i+1, wantHints[i])
		}
	}
	if !strings.Contains(frames1[0], "$ keys received: 0") {
		return fail("phase 1: the stub must report ZERO keys while released")
	}
	if !strings.Contains(frames1[1], "$ keys received: 4") {
		return fail("phase 2: the \"echo\" letters must count 4 on the stub line")
	}
	fmt.Printf("lazy-spawn: OK — SpawnTerminal called exactly once, on the first visit\n")
	fmt.Printf("key routing: %d shell keys across the toggle flow (4 typed + tab + shift+tab + 2 digits; letters blocked while released)\n", stub1.received)

	// determinism: the synchronous drive replays byte-identically.
	frames2, stub2, calls2, _, err := termDrive()
	app.SpawnTerminal = nil
	if err != nil {
		return err
	}
	if calls2 != 1 {
		return fail("second drive: lazy-spawn violated (%d factory calls)", calls2)
	}
	for i := range frames1 {
		if frames1[i] != frames2[i] {
			return fail("phase %d frame differs between the two synchronous drives", i+1)
		}
	}
	if stub2.received != stub1.received {
		return fail("stub received counter differs between drives (%d vs %d)", stub1.received, stub2.received)
	}
	fmt.Printf("deterministic: OK — two synchronous drives produced byte-identical frames\n")

	// quit hook: cmd/theboringoffice calls CloseTerminal after Run returns —
	// the explicit close is the leak guard.
	if stub1.closed {
		return fail("stub already closed before CloseTerminal — quit hook ran early")
	}
	close1()
	if !stub1.closed {
		return fail("CloseTerminal did not close the shell panel")
	}
	fmt.Printf("quit hook: OK — CloseTerminal closed the spawned shell (alive→false)\n")
	fmt.Println("asserts: OK — RELEASED default (letters consumed, a REAL tab key event leaves the tab — the ctrl+i/tab byte-conflict can never regress, released hint); ctrl+space toggles BOTH ways (dive IN: echo → +4 keys, wave-41 tab/shift+tab/digits ride; toggle OUT in place); ctrl+o alias releases IN PLACE; leave-while-captured auto-releases and re-entry starts RELEASED; lazy-spawn ×1; two drives byte-identical; CloseTerminal kills the stub shell")
	return nil
}

// --- paste proof (--paste) ---------------------------------------------------
// The office-wide tea.PasteMsg surface, driven SYNCHRONOUSLY (focusDriver,
// no wall clock) through the REAL app: every leg is one tea.PasteMsg — the
// msg Terminal.app's cmd+v and every bracketed-paste terminal deliver.
//
//	leg A — CHAT small paste: inserts LITERALLY (one batched op, never
//	        the ~530ms/key drain) and sends verbatim;
//	leg B — CHAT large paste: >20 lines collapses to the one-line chip
//	        "[pasted N lines · M chars]" — the paste body never paints;
//	leg C — the chip eats ONE backspace as a unit (draft + record gone);
//	leg D — expand-on-send: the member sees the chip, the agent's Send
//	        receives the FULL original text;
//	leg E — shift+enter AND ctrl+j insert newlines (enter still sends);
//	leg F — RELEASED terminal tab: the paste falls back to the chat
//	        draft (the shell never sees it), then CAPTURED: the paste
//	        routes to the shell (the stub records the content);
//	leg G — QUESTION float: the popover's answer field takes a multi-
//	        line paste verbatim; ctrl+enter ships AnswerQuestion;
//	leg H — IGNORED paste (agents tab): ONE dim office notice.
//
// Byte-for-byte determinism: the whole drive runs TWICE — frames, send
// log, answer log and stub paste counts must be identical.

// pasteDriveOut — everything the two drives compare + the report prints.
type pasteDriveOut struct {
	frameSmall   string // leg A: literal small paste in the textarea
	frameChip    string // leg B: the collapsed chip row
	framePopped  string // leg C: chip deleted, placeholder back
	frameNewline string // leg E: the 3-row draft
	frameTerm    string // leg F: captured terminal, stub shows pastes: 1
	frameQuest   string // leg G: the echo box carries the pasted lines
	frameNotice  string // leg H: the dim ignore notice on the chat tab
	sendLog      []string
	answerLog    []string
	termPastes   []string
}

func runPasteDrive() (pasteDriveOut, error) {
	var out pasteDriveOut
	fail := func(format string, args ...any) (pasteDriveOut, error) {
		return out, fmt.Errorf(format, args...)
	}
	var stub *terminalPanelStub
	app.SpawnTerminal = func(cols, rows int) (app.TerminalTab, error) {
		stub = newTerminalPanelStub(cols, rows)
		return stub, nil
	}
	backend := &stubBackend{done: make(chan struct{})} // Mode() only — no script
	d := &focusDriver{m: app.New(backend, config.Default())}
	// exec — the exact breadth-first drain the --browsertab proof runs:
	// the chat panel's onSend/onQuestionAnswer return the REAL work as
	// tea.Cmds (backend.Send, the questionAnswerMsg hop) — dropping them
	// (focusDriver.send) would fake the proof. Spinner ticks and cursor
	// blinks are self-re-arming heartbeats: dropped, exactly as runMsg.
	exec := func(msg tea.Msg) {
		tm, cmd := d.m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			d.m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
				// heartbeats re-arm forever — dropped
			default:
				tm2, next := d.m.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					d.m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}
	exec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	exec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape})) // boot-splash dismiss (a lone esc is a no-op in the input)
	paste := func(s string) { exec(tea.PasteMsg{Content: s}) }
	press := func(c rune) { exec(tea.KeyPressMsg(tea.Key{Code: c, Text: string(c)})) }
	enter := func() { exec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) }

	// leg A — small paste: literal insert, verbatim send.
	paste("hello paste world")
	out.frameSmall = d.m.Frame()
	enter()

	// leg B — 31 lines: the chip collapses it.
	lorem := make([]string, 31)
	for i := range lorem {
		lorem[i] = fmt.Sprintf("lorem line %02d", i+1)
	}
	big := strings.Join(lorem, "\n")
	paste(big)
	out.frameChip = d.m.Frame()

	// leg C — one backspace pops the whole chip.
	exec(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	out.framePopped = d.m.Frame()

	// leg D — re-paste, send: the agent receives the FULL text.
	paste(big)
	enter()

	// leg E — shift+enter + ctrl+j newlines; enter sends the 3-row draft.
	press('a')
	press('b')
	exec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModShift})) // ghostty/kitty deliver this
	press('c')
	press('d')
	exec(tea.KeyPressMsg(tea.Key{Code: 'j', Mod: tea.ModCtrl})) // the universal fallback
	press('e')
	press('f')
	out.frameNewline = d.m.Frame()
	enter()

	// leg F — RELEASED terminal: paste → the chat draft (never the shell);
	// CAPTURED: paste → the shell.
	if !d.m.SelectTab("terminal") {
		return fail("terminal tab not selectable")
	}
	if stub == nil {
		return fail("the terminal tab never spawned the stub")
	}
	paste("released-body") // released: the chat draft takes it
	if len(stub.pastes) != 0 {
		return fail("released terminal: the shell must NOT see the paste, got %q", stub.pastes)
	}
	if !d.m.SelectTab("chat") {
		return fail("chat tab not selectable")
	}
	enter() // sends "released-body" — the released-terminal paste landed in the draft
	if !d.m.SelectTab("terminal") {
		return fail("terminal tab not re-selectable")
	}
	exec(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace, Mod: tea.ModCtrl})) // dive in
	paste("echo pasted-ok")
	if len(stub.pastes) != 1 || stub.pastes[0] != "echo pasted-ok" {
		return fail("captured terminal: the shell must own the paste, got %q", stub.pastes)
	}
	out.termPastes = append([]string(nil), stub.pastes...)
	out.frameTerm = d.m.Frame()
	exec(tea.KeyPressMsg(tea.Key{Code: tea.KeySpace, Mod: tea.ModCtrl})) // release back out

	// leg G — the question float's answer field: multi-line paste verbatim.
	exec(state.Event{Kind: state.EvQuestion, QuestionID: "que-paste",
		EmployeeName: "boss", ToolState: "pending",
		Questions: []state.QuestionItem{{Question: "paste the stack trace?"}},
	})
	if !d.m.SelectTab("chat") { // the popover splices into the CHAT panel
		return fail("chat tab not selectable for the question leg")
	}
	paste("goroutine 1 [running]:\nmain.main()")
	out.frameQuest = d.m.Frame()
	exec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModCtrl}))

	// leg H — agents tab: nowhere accepts text → the dim notice (read
	// back on the chat tab, where office notices paint).
	if !d.m.SelectTab("agents") {
		return fail("agents tab not selectable")
	}
	paste("nowhere-bound")
	if !d.m.SelectTab("chat") {
		return fail("chat tab not re-selectable")
	}
	out.frameNotice = d.m.Frame()

	out.sendLog = append([]string(nil), backend.sendLog...)
	out.answerLog = append([]string(nil), backend.answerLog...)
	return out, nil
}

func runPasteProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	out1, err := runPasteDrive()
	app.SpawnTerminal = nil
	if err != nil {
		return err
	}
	out2, err := runPasteDrive()
	app.SpawnTerminal = nil
	if err != nil {
		return err
	}

	// --- leg frames + asserts -------------------------------------------------
	type legFrame struct{ label, frame string }
	frames := []legFrame{
		{"A — chat: a small paste inserts LITERALLY (one batched op)", out1.frameSmall},
		{"B — chat: a 31-line paste COLLAPSES to the one-line chip", out1.frameChip},
		{"C — chat: ONE backspace pops the whole chip", out1.framePopped},
		{"E — chat: shift+enter + ctrl+j insert newlines (enter still sends)", out1.frameNewline},
		{"F — terminal CAPTURED: the paste routes to the shell (stub records)", out1.frameTerm},
		{"G — question float: the answer field takes the paste verbatim", out1.frameQuest},
		{"H — agents tab: the ignored paste toasts ONE dim notice", out1.frameNotice},
	}
	for _, lf := range frames {
		fmt.Printf("===== UI SHOT · paste — %s =====\n", lf.label)
		fmt.Println(lf.frame)
		fmt.Println("===== UI SHOT =====")
	}
	// asserts compare CONTENT, not styling: the textarea's virtual cursor
	// reverse-wraps the single cell it sits on (the empty draft's first
	// placeholder rune), so raw frames can split a word mid-way — strip.
	assertIn := func(frame, tag string, wants ...string) error {
		plain := ansi.Strip(frame)
		for _, w := range wants {
			if !strings.Contains(plain, w) {
				return fail("%s: frame missing %q", tag, w)
			}
		}
		return nil
	}
	assertNotIn := func(frame, tag string, rejects ...string) error {
		plain := ansi.Strip(frame)
		for _, w := range rejects {
			if strings.Contains(plain, w) {
				return fail("%s: frame must NOT contain %q", tag, w)
			}
		}
		return nil
	}

	if err := assertIn(out1.frameSmall, "leg A", "hello paste world"); err != nil {
		return err
	}
	if err := assertIn(out1.frameChip, "leg B", "[pasted 31 lines ·"); err != nil {
		return err
	}
	if err := assertNotIn(out1.frameChip, "leg B", "lorem line 15", "lorem line 30"); err != nil {
		return err
	}
	if err := assertNotIn(out1.framePopped, "leg C", "[pasted 31 lines"); err != nil {
		return err
	}
	if err := assertIn(out1.framePopped, "leg C", "talk to the boss…"); err != nil {
		return err
	}
	if err := assertIn(out1.frameNewline, "leg E", "ab", "cd", "ef"); err != nil {
		return err
	}
	if err := assertIn(out1.frameTerm, "leg F", "pastes: 1"); err != nil {
		return err
	}
	if err := assertIn(out1.frameQuest, "leg G", "QUESTION", "paste the stack trace?", "goroutine 1 [running]:", "main.main()"); err != nil {
		return err
	}
	if err := assertIn(out1.frameNotice, "leg H", "paste: nothing focused accepts text"); err != nil {
		return err
	}

	// --- the send/answer captures ----------------------------------------------
	loremFull := make([]string, 31)
	for i := range loremFull {
		loremFull[i] = fmt.Sprintf("lorem line %02d", i+1)
	}
	big := strings.Join(loremFull, "\n")
	wantSends := []string{"hello paste world", big, "ab\ncd\nef", "released-body"}
	if len(out1.sendLog) != len(wantSends) {
		return fail("send capture: want %d sends, got %d (%q)", len(wantSends), len(out1.sendLog), out1.sendLog)
	}
	for i, w := range wantSends {
		if out1.sendLog[i] != w {
			return fail("send %d: got %q, want %q", i, out1.sendLog[i], w)
		}
	}
	fmt.Printf("send capture: OK — small paste verbatim; the chip EXPANDED on send (the agent receives all 31 lines); newlines joined; the released-terminal paste rode the draft\n")
	if len(out1.answerLog) != 1 || !strings.Contains(out1.answerLog[0], "AnswerQuestion(que-paste, [goroutine 1 [running]:\nmain.main()])") {
		return fail("question capture: want the pasted 2-line answer verbatim, got %q", out1.answerLog)
	}
	fmt.Printf("question capture: OK — %s\n", strings.ReplaceAll(out1.answerLog[0], "\n", "⏎"))

	// --- determinism: the two drives byte-match ---------------------------------
	if out1.frameSmall != out2.frameSmall || out1.frameChip != out2.frameChip ||
		out1.framePopped != out2.framePopped || out1.frameNewline != out2.frameNewline ||
		out1.frameTerm != out2.frameTerm || out1.frameQuest != out2.frameQuest ||
		out1.frameNotice != out2.frameNotice {
		return fail("the two synchronous drives produced different frames")
	}
	if strings.Join(out1.sendLog, "␟") != strings.Join(out2.sendLog, "␟") ||
		strings.Join(out1.answerLog, "␟") != strings.Join(out2.answerLog, "␟") ||
		strings.Join(out1.termPastes, "␟") != strings.Join(out2.termPastes, "␟") {
		return fail("the two drives produced different captures")
	}
	fmt.Printf("deterministic: OK — two synchronous drives produced byte-identical frames + captures\n")
	fmt.Println("asserts: OK — small paste literal + verbatim send; 31-line paste → one-line chip (body hidden); chip = ONE backspace unit; expand-on-send (full text to the agent); shift+enter + ctrl+j newlines; released terminal → chat draft / captured → shell (content-pinned); question field multi-line paste → AnswerQuestion; agents-tab paste → dim notice; two drives byte-identical")
	return nil
}

// --- fix-wave proof (--focus) ----------------------------------------------
// Three frames over ONE synchronous driver (no tea.Program, no wall clock):
// every EvTick is pumped by hand, so the panel state is exact. Frame A
// catches the empty typing placeholder: the typing row sits BELOW the
// divider (first row above the input region), no "▌" anywhere. Frame B
// catches a streaming partial bubble: the text grows in the viewport and
// the typing row STAYS below the divider for the whole pending period —
// still no caret. Frame C catches two concurrent agents: employee tool
// calls grouped into per-agent work threads (headers + merged rows), the
// boss's own tool line still inline, and the boss quiet at its
// placeholder with workers busy → BossDelegating ("boss: delegating ·
// 2 busy" + [delegat] nameplate). Every frame also asserts the chat
// panel's rows stay inside the width the divider draws (wrap, never
// overflow, never clip).

// focusDriver — minimal synchronous model pump (same shape as socialDriver).
type focusDriver struct {
	m app.Model
}

func newFocusDriver() *focusDriver {
	backend := &stubBackend{done: make(chan struct{})} // Mode() only — no script
	m := app.New(backend, config.Default())
	d := &focusDriver{m: m}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	return d
}

func (d *focusDriver) send(msg tea.Msg) {
	tm, _ := d.m.Update(msg)
	if fm, ok := tm.(app.Model); ok {
		d.m = fm
	}
}

func (d *focusDriver) pump(n int) {
	for i := 0; i < n; i++ {
		d.send(state.Event{Kind: state.EvTick})
	}
}

func focusTool(ownerID, ownerName, callID, toolName, summary, toolState string) state.Event {
	return state.Event{Kind: state.EvTool, EmployeeID: ownerID, EmployeeName: ownerName,
		ToolName: toolName, ToolSummary: summary, ToolState: toolState, CallID: callID}
}

// chatPanelSegs yields the chat panel's interior segment of every full
// frame line — the slice between the line's LAST two "│" borders (the
// chat sidebar is the rightmost panel), ansi-stripped for width math.
// Rows without panel borders (topbar/statusbar) are skipped, so indexes
// are panel-relative, not screen rows.
func chatPanelSegs(frame string) []string {
	var segs []string
	for _, ln := range strings.Split(frame, "\n") {
		parts := strings.Split(ansi.Strip(ln), "│")
		if len(parts) < 3 {
			continue
		}
		segs = append(segs, parts[len(parts)-2])
	}
	return segs
}

// chatDividerIdx — the segs index of the chat divider row (a segment of
// pure "─"), -1 when absent.
func chatDividerIdx(segs []string) int {
	for i, seg := range segs {
		t := strings.TrimSpace(seg)
		if t != "" && strings.Trim(t, "─") == "" {
			return i
		}
	}
	return -1
}

// chatPanelTail — the chat panel's segments from the divider down (the
// divider row itself first), capped at n rows.
func chatPanelTail(frame string, n int) []string {
	segs := chatPanelSegs(frame)
	if di := chatDividerIdx(segs); di >= 0 {
		tail := segs[di:]
		if len(tail) > n {
			tail = tail[:n]
		}
		return tail
	}
	return nil
}

// assertChatLayout — the panel hygiene sweep shared by every focus frame:
// no "▌" anywhere (the blinking caret is gone in EVERY state), and every
// chat row stays inside the width the divider row itself draws (wrap,
// never overflow).
func assertChatLayout(tag, frame string) error {
	if strings.Contains(frame, "▌") {
		return fmt.Errorf("%s: found \"▌\" — the stream caret must not exist in any chat state", tag)
	}
	segs := chatPanelSegs(frame)
	di := chatDividerIdx(segs)
	if di < 0 {
		return fmt.Errorf("%s: no chat divider row found", tag)
	}
	budget := ansi.StringWidth(segs[di])
	for i, seg := range segs {
		if w := ansi.StringWidth(seg); w > budget {
			return fmt.Errorf("%s: chat row %d overflows the %d-cell panel budget (%d cells): %q", tag, i, budget, w, strings.TrimSpace(seg))
		}
	}
	return nil
}

// assertTypingRowBelowDivider — the row carrying needle (typing spinner /
// delegating line) sits EXACTLY one segs-step under the divider: the
// first row of the input region, above chips/picker/textarea. While the
// boss is busy the TEXTAREA PLACEHOLDER quotes the same busy text
// ("› boss is typing…") — those prompt rows are skipped; the typing row
// is the one renderer-owned line carrying it (exactly one must exist).
func assertTypingRowBelowDivider(tag, frame, needle string) error {
	segs := chatPanelSegs(frame)
	di := chatDividerIdx(segs)
	if di < 0 {
		return fmt.Errorf("%s: no chat divider row found", tag)
	}
	row := -1
	for i, seg := range segs {
		if !strings.Contains(seg, needle) {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(seg), "›") {
			continue // textarea placeholder quoting the busy text — a prompt row, not the typing row
		}
		if row >= 0 {
			return fmt.Errorf("%s: %q appears on MORE than one chat row", tag, needle)
		}
		row = i
	}
	if row < 0 {
		return fmt.Errorf("%s: typing row %q missing from the chat panel", tag, needle)
	}
	if row != di+1 {
		return fmt.Errorf("%s: typing row must be the FIRST row below the divider (divider at seg %d, row at %d)", tag, di, row)
	}
	return nil
}

func runFocusProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	d := newFocusDriver()
	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — focus stub online"})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"wire the sse stream", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
	d.pump(4) // tick 4 — settled placeholder
	fmt.Println("===== UI SHOT · FOCUS A — empty pending bubble: typing row BELOW the divider, NO caret =====")
	frameA := d.m.Frame()
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")

	// (b) the reply streams in as accumulated pending updates; the
	// placeholder boss-1 is replaced by the first real-bubble update.
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"Wiring the handler —", true)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"Wiring the handler — the **SSE stream** fans out to both workers now.", true)})
	d.pump(2) // tick 6 — mid-stream
	fmt.Println("===== UI SHOT · FOCUS B — streaming bubble grows in the viewport; typing row STAYS below the divider, NO caret =====")
	frameB := d.m.Frame()
	fmt.Println(frameB)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("--- chat panel, divider row down to the input (frame B, ansi-stripped segments) ---")
	for _, seg := range chatPanelTail(frameB, 6) {
		fmt.Println(seg)
	}

	// (c) settle the round-1 bubble, dispatch two workers, storm tool
	// calls for BOTH (interleaved — merge per agent+CallID), one boss
	// inline tool, then a fresh user turn whose placeholder goes quiet.
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"Wiring the handler — the **SSE stream** fans out to both workers now. Watch their threads below.", false)})
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "dev-1",
		Task: state.BoardTask{ID: "t1", Title: "Wire the SSE stream", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "dev-1", TaskID: "t1"})
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "sco-1",
		Task: state.BoardTask{ID: "t2", Title: "Scan the repo", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "sco-1", TaskID: "t2"})
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/room/manager.go", "running"))
	d.send(focusTool("sco-1", "skopos-1", "call-s1", "grep", "SSE, 12 hits", "running"))
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/room/manager.go", "done"))
	d.send(focusTool("sco-1", "skopos-1", "call-s1", "grep", "SSE, 12 hits", "done"))
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/room/handler.go", "running"))
	d.send(focusTool("sco-1", "skopos-1", "call-s2", "read", "internal/api/room.go", "done"))
	d.send(focusTool("boss", "boss", "call-b1", "write", "static/sse.html", "done"))
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u2", "user",
		"and the reconnect backoff", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-2", "boss", "", true)})
	d.pump(10) // tick 16 — boss quiet for 10 ticks, both workers busy
	fmt.Println("===== UI SHOT · FOCUS C — concurrent agents grouped into work threads, boss delegating =====")
	frameC := d.m.Frame()
	fmt.Println(frameC)
	fmt.Println("===== UI SHOT =====")

	// asserts — frame A: typing row below the divider, no caret anywhere,
	// no delegation while the boss itself is generating.
	for _, want := range []string{"is typing…"} {
		if !strings.Contains(frameA, want) {
			return fail("focus A: frame missing %q", want)
		}
	}
	if strings.Contains(frameA, "delegating") {
		return fail("focus A: empty pending placeholder shows delegation text")
	}
	if err := assertTypingRowBelowDivider("focus A", frameA, "is typing…"); err != nil {
		return err
	}
	if err := assertChatLayout("focus A", frameA); err != nil {
		return err
	}
	// asserts — frame B: a STREAMING bubble keeps the typing row for the
	// whole pending period — the partial text lives in the viewport, the
	// pulse stays below the divider. No caret in any state.
	for _, want := range []string{"is typing…", "SSE stream"} {
		if !strings.Contains(frameB, want) {
			return fail("focus B: frame missing %q (streaming text in the viewport AND the typing row at once)", want)
		}
	}
	if err := assertTypingRowBelowDivider("focus B", frameB, "is typing…"); err != nil {
		return err
	}
	if err := assertChatLayout("focus B", frameB); err != nil {
		return err
	}
	// asserts — frame C (both threads COLLAPSED, the default: head + ↳
	// sneak; the merged tool rows themselves sit under ctrl+g)
	strippedC := ansi.Strip(frameC)
	for _, want := range []string{
		"Developer Task — Wire the SSE stream", // per-agent thread head (task from dispatch)
		"Explore Task — Scan the repo",         // second agent, newer at the bottom
		"↳ Edit internal/room/handler.go",      // tekton-1 sneak: newest merged call (bare, opencode form)
		"↳ Read internal/api/room.go",          // skopos-1 sneak: newest merged call
		"[tool] write · static/sse.html ✓",     // boss's own tool stays INLINE (no thread)
		"delegating · 2 busy",                  // settled placeholder text (no spinner)
		"[delegat]",                            // floor nameplate
		"reconnect backoff",                    // round-2 user turn survived the storm
	} {
		if !strings.Contains(strippedC, want) {
			return fail("focus C: frame missing %q", want)
		}
	}
	// thread tool rows are 2-space indented under their header and shaped
	// "<Verb> <rest>"; an indented shaped row carrying the BOSS's write
	// target would prove the boss line got captured into a worker thread.
	if strings.Contains(strippedC, "  [tool] Write static/sse.html") {
		return fail("focus C: boss tool line was captured into a worker thread (must stay inline)")
	}
	// the delegating row rides the SAME below-divider slot as the typing
	// row (a settled swap of it)
	if err := assertTypingRowBelowDivider("focus C", frameC, "delegating · 2 busy"); err != nil {
		return err
	}
	if err := assertChatLayout("focus C", frameC); err != nil {
		return err
	}
	// settle leg: tekton-1 returns (sprite leaves the busy set) → its head
	// swaps to the dim ✓ glyph with the "(· N tool calls ✓ done)" rollup;
	// skopos-1, still working, keeps the animated braille glyph. ctrl+g
	// expands ALL threads: the merged tool rows reappear.
	d.send(state.Event{Kind: state.EvReturned, EmployeeID: "dev-1", TaskID: "t1",
		Mail: mail("m1", "tekton-1", "boss", "return: sse stream", "stream is live.", state.MailReturn)})
	d.pump(1)
	strippedCollapse := ansi.Strip(d.m.Frame())
	if !strings.Contains(strippedCollapse, "✓ Developer Task — Wire the SSE stream (· 2 tool calls ✓ done)") {
		return fail("focus settle: missing settled ✓ thread head for tekton-1")
	}
	if !strings.Contains(strippedCollapse, "Explore Task — Scan the repo") {
		return fail("focus settle: skopos-1's thread went missing — only the RETURNED agent settles")
	}
	if strings.Contains(strippedCollapse, "✓ Explore Task — Scan the repo") {
		return fail("focus settle: skopos-1 settled too — only the RETURNED agent's head swaps to the ✓ glyph")
	}
	d.send(tea.KeyPressMsg(tea.Key{Code: 'g', Mod: tea.ModCtrl}))
	strippedExpanded := ansi.Strip(d.m.Frame())
	if !strings.Contains(strippedExpanded, "Developer Task — Wire the SSE stream") ||
		!strings.Contains(strippedExpanded, "  [tool] Read internal/room/manager.go ✓") {
		return fail("focus expand: ctrl+g did not re-expand the completed thread")
	}
	fmt.Println("asserts: OK — no caret in any state; typing row sits below the divider (above the input) for the WHOLE pending period; delegating row swaps into the same slot; every chat row inside the divider's width budget; worker threads grouped (collapsed heads + ↳ sneaks) + CallID-merged, boss tool inline, [delegat] nameplate, EvReturned settles the head to the ✓ glyph, ctrl+g re-expands the merged rows")
	return nil
}

// runPersistDemoSkipProof (--persist) — the office-session DEMO regression:
// restore + persist are LIVE-only by ruling (demo restore = confusing).
// Seeds a FRESH session.json for cwd in a scratch THEBORINGOFFICE_HOME (so the
// "skip" cannot be a missing-file false pass), runs the standard scripted
// demo shot, then asserts: (1) LoadSession DOES find the seeded file
// (the gate is the mode check, not the file lookup), (2) NO "restored
// office session" notice ever surfaces in the office state chat, (3) the
// demo boot never OVERWRITES the seeded file (SavedAt byte-identical).
func runPersistDemoSkipProof() error {
	home, err := os.MkdirTemp("", "theboringoffice-persist-demo-skip")
	if err != nil {
		return err
	}
	defer os.RemoveAll(home)
	if err := os.Setenv("THEBORINGOFFICE_HOME", home); err != nil {
		return err
	}
	fmt.Printf("--- scratch THEBORINGOFFICE_HOME: %s ---\n", home)

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	seed := app.Snapshot(dir, "ses-demo-fake", state.OfficeState{
		Chat: []state.ChatMsg{
			{ID: "u1", From: "user", Kind: "user", Text: "old turn from a previous run"},
			{ID: "b1", From: "boss", Kind: "boss", Text: "the previous reply"},
		},
	})
	if err := app.SaveSession(dir, seed); err != nil {
		return err
	}
	before, err := os.ReadFile(app.SessionPath(dir))
	if err != nil {
		return err
	}
	if _, ok := app.LoadSession(dir); !ok {
		return fmt.Errorf("seeded session.json not loadable — the skip assert would be vacuous")
	}

	// The standard shot: the REAL app model over the scripted stub (demo
	// mode) for the full window.
	backend := &stubBackend{done: make(chan struct{}), flushQueue: true}
	fm, err := runManualLoop(config.Default(), backend, "chat", shotDur, nil)
	if err != nil {
		return err
	}
	frame := fm.Frame()
	fmt.Println("===== UI SHOT · --persist (demo boot with a seeded session.json — restore must NOT fire) =====")
	fmt.Println(frame)
	fmt.Println("===== UI SHOT =====")

	// (2) no restore notice in the office chat state.
	for _, c := range fm.State().Chat {
		if strings.Contains(c.Text, "restored office session from") {
			return fmt.Errorf("demo boot restored the seeded office session (restore is live-only): %q", c.Text)
		}
	}
	// (2b) the demo script ran normally — its scripted turn is present.
	found := false
	for _, c := range fm.State().Chat {
		if strings.Contains(c.Text, "hello.html") {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("the demo script did not run (no hello.html turn) — cannot claim a clean demo boot")
	}
	// (3) the demo boot never persisted over the seeded file.
	after, err := os.ReadFile(app.SessionPath(dir))
	if err != nil {
		return err
	}
	if string(before) != string(after) {
		return fmt.Errorf("demo boot overwrote session.json (persist is live-only)")
	}
	fmt.Println("asserts: OK — seeded session.json found by LoadSession but demo mode skipped restore, no restore notice in chat, file untouched (live-only)")
	fmt.Println("PERSIST-DEMO-SKIP: OK")
	return nil
}

// --- slash popover proof (--slashpop) --------------------------------------
// Synchronous keys through the REAL app model: typing "/" at a word start
// opens the command popover, "/th" prefix-filters it live, Enter on /theme
// pre-fills "/theme " and flips the box into the THEME picker, arrows apply
// a LIVE preview (two states printed), esc cancels back to the original
// theme, Enter commits through the plain /theme slash path (persist +
// office notice).

// drainCmd bounded-drives a returned cmd tree (the slash commit runs
// through onSend → slashMsg): timer arms (tick/blink) are skipped after a
// short wait — the app re-arms them on its own events; the proof only
// needs the produced MESSAGES.
func drainCmd(d *focusDriver, cmd tea.Cmd, depth int) {
	if cmd == nil || depth > 8 {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(150 * time.Millisecond):
		return // a timer arm (tick/cursor blink): not a message the proof needs
	}
	if msg == nil {
		return
	}
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			drainCmd(d, c, depth+1)
		}
		return
	}
	d.send(msg)
}

func runSlashPopProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	d := newFocusDriver()
	names := chrome.ThemeNames()
	orig := chrome.CurrentTheme().Name
	defer func() { // leave the machine as found (theme file included)
		chrome.SetTheme(orig)
		office.SetTheme(orig)
		_ = chrome.PersistTheme()
	}()
	if len(names) < 3 {
		return fail("slashpop: need ≥3 themes registered, got %d", len(names))
	}
	typeIn := func(s string) {
		for _, r := range s {
			d.send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		}
	}
	key := func(code rune) tea.Cmd {
		tm, c := d.m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
		if fm, ok := tm.(app.Model); ok {
			d.m = fm
		}
		return c
	}

	// frame 1 — "/th": filtered command menu (/theme /themes /thinking)
	typeIn("/th")
	fmt.Println("===== UI SHOT · SLASH A — typed \"/th\": the popover prefix-filters to /theme /themes /thinking =====")
	frameA := d.m.Frame()
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")
	for _, want := range []string{"commands", "› /theme", "/themes", "/thinking", "switch theme (persists)", "/theme <name>"} {
		if !strings.Contains(frameA, want) {
			return fail("slashpop A: frame missing %q", want)
		}
	}
	for _, not := range []string{"/tools", "/clear", "/zen"} {
		if strings.Contains(frameA, not) {
			return fail("slashpop A: unfiltered command %q still shows for fragment \"/th\"", not)
		}
	}

	// Enter applies /theme → "/theme " prefill + the THEME picker opens
	key(tea.KeyEnter)
	fmt.Println("===== UI SHOT · SLASH B — Enter on /theme: \"› /theme \" prefill, live theme list =====")
	frameB := d.m.Frame()
	fmt.Println(frameB)
	fmt.Println("===== UI SHOT =====")
	for _, want := range []string{"theme preview", "noir", "paper", "dracula", "commit + persist"} {
		if !strings.Contains(frameB, want) {
			return fail("slashpop B: frame missing %q", want)
		}
	}
	if !strings.Contains(ansi.Strip(frameB), "› /theme ") {
		return fail("slashpop B: textarea prefill %q missing (raw frame is style-split)", "› /theme ")
	}
	if cur := chrome.CurrentTheme().Name; cur != orig {
		return fail("slashpop B: theme switched on Enter-apply (%s) — preview must wait for arrows", cur)
	}

	// THEME PREVIEW state 1 — one ↓: the second theme paints live
	key(tea.KeyDown)
	p1 := chrome.CurrentTheme().Name
	fmt.Printf("===== UI SHOT · SLASH C — preview state 1 (↓ once): active theme is now %q — PAINTED LIVE =====\n", p1)
	frameC := d.m.Frame()
	fmt.Println(frameC)
	fmt.Println("===== UI SHOT =====")
	if p1 != names[1] {
		return fail("slashpop C: expected live preview %q after one ↓, got %q", names[1], p1)
	}

	// THEME PREVIEW state 2 — another ↓: the third theme paints live
	key(tea.KeyDown)
	p2 := chrome.CurrentTheme().Name
	fmt.Printf("===== UI SHOT · SLASH D — preview state 2 (↓ again): active theme is now %q — PAINTED LIVE =====\n", p2)
	frameD := d.m.Frame()
	fmt.Println(frameD)
	fmt.Println("===== UI SHOT =====")
	if p2 != names[2] {
		return fail("slashpop D: expected live preview %q after two ↓, got %q", names[2], p2)
	}

	// esc cancels the preview session back to the original theme
	key(tea.KeyEscape)
	if cur := chrome.CurrentTheme().Name; cur != orig {
		return fail("slashpop esc: preview was not cancelled back — got %q, want %q", cur, orig)
	}

	// retype to re-open in theme mode, ↓ previews the second theme, Enter
	// COMMITS through the plain /theme path (persist + office notice)
	for i := 0; i < 20; i++ {
		key(tea.KeyBackspace)
	}
	typeIn("/theme ")
	key(tea.KeyDown)
	picked := chrome.CurrentTheme().Name
	drainCmd(d, key(tea.KeyEnter), 0)
	fmt.Println("===== UI SHOT · SLASH E — after commit: draft cleared, office notice \"theme → …\" =====")
	frameE := d.m.Frame()
	fmt.Println(frameE)
	fmt.Println("===== UI SHOT =====")
	if picked != names[1] {
		return fail("slashpop E: commit leg previews %q after one ↓, got %q", names[1], picked)
	}
	if !strings.Contains(frameE, "theme → "+picked) {
		return fail("slashpop E: office notice %q missing after commit", "theme → "+picked)
	}
	if cur := chrome.CurrentTheme().Name; cur != picked {
		return fail("slashpop E: committed theme %q not active (%q)", picked, cur)
	}
	fmt.Println("asserts: OK — \"/\" opens the popover, \"/th\" filters live, /theme prefill flips to the theme picker, arrows preview live (two states printed), esc cancels back, enter commits + persists via the plain slash path")
	return nil
}

// --- employee thinking inside worker threads (--threads-think) --------------
// tekton-1's EvThought entries merge per CallID into its OWN work thread:
// a dim-italic "thinking · N lines" row among the tool rows while the
// thread is live, a "· N think" count in the collapsed summary after
// EvReturned, a full body under ctrl+g. The boss's EvThought path stays
// byte-identical (flow "thinking · N lines", no thread).

func runThreadsThinkProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	d := newFocusDriver()
	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — threads-think stub online"})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"wire the sse stream", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"On it — tekton-1 is wiring the handler.", false)})
	// boss think: unchanged flow rendering (proves the boss path is intact)
	d.send(state.Event{Kind: state.EvThought, EmployeeID: "boss", CallID: "bk-1",
		Text: "weighing the fan-out", Done: false})
	d.send(state.Event{Kind: state.EvThought, EmployeeID: "boss", CallID: "bk-1",
		Text: "weighing the fan-out\nsent it out", Done: true})
	// tekton-1: dispatch, tools, AND a streamed employee thought
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "dev-1",
		Task: state.BoardTask{ID: "t1", Title: "Wire the SSE stream", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "dev-1", TaskID: "t1"})
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/room/manager.go", "done"))
	d.send(state.Event{Kind: state.EvThought, EmployeeID: "dev-1", EmployeeName: "tekton-1", CallID: "tk-1",
		Text: "scanning options\nchoosing approach", Done: false})
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/room/handler.go", "done"))
	d.send(state.Event{Kind: state.EvThought, EmployeeID: "dev-1", EmployeeName: "tekton-1", CallID: "tk-1",
		Text: "scanning options\nchoosing approach\nwriting the patch", Done: true})
	d.pump(4)
	fmt.Println("===== UI SHOT · THINK A — live thread COLLAPSED (the default): NO rollup while running; the count surfaces once settled =====")
	frameA := d.m.Frame()
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")
	strippedA := ansi.Strip(frameA)
	for _, want := range []string{
		"Developer Task — Wire the SSE stream", // live thread head (RoleDeveloper → "Developer")
		"↳ Edit internal/room/handler.go",      // sneak: the thread's NEWEST tool, opencode-shaped + bare
	} {
		if !strings.Contains(strippedA, want) {
			return fail("threads-think A: frame missing %q", want)
		}
	}
	if strings.Contains(strippedA, "stream (· 2 tool calls") {
		return fail("threads-think A: the LIVE head trailed the settled rollup — running heads carry NO rollup")
	}
	if !strings.Contains(strippedA, "thinking · 2 lines") {
		return fail("threads-think A: boss thought missing its collapsed flow row (style-split raw text)")
	}
	if strings.Count(strippedA, "thinking ·") != 1 {
		return fail("threads-think A: the BOSS's thought leaked into a worker thread (only the boss flow row may read \"thinking ·\" — the employee's think rolls up into the summary count)")
	}
	if strings.Contains(strippedA, "writing the patch") {
		return fail("threads-think A: collapsed view must show the one-line count, not the body")
	}

	// EvReturned → the head settles to the dim ✓ glyph; the rollup KEEPS
	// the think count
	d.send(state.Event{Kind: state.EvReturned, EmployeeID: "dev-1", TaskID: "t1",
		Mail: mail("m1", "tekton-1", "boss", "return: sse stream", "stream is live.", state.MailReturn)})
	d.pump(1)
	fmt.Println("===== UI SHOT · THINK B — settled thread: \"✓ … (· 2 tool calls · 1 think ✓ done)\" =====")
	frameB := d.m.Frame()
	fmt.Println(frameB)
	fmt.Println("===== UI SHOT =====")
	strippedB := ansi.Strip(frameB)
	if !strings.Contains(strippedB, "✓ Developer Task — Wire the SSE stream (· 2 tool calls · 1 think ✓ done)") {
		return fail("threads-think B: collapsed summary with the think count missing")
	}
	if strings.Contains(strippedB, "thinking · 3 lines") {
		return fail("threads-think B: think row visible under a collapsed thread")
	}

	// ctrl+g: full expand covers tools AND thoughts (body in natural order)
	d.send(tea.KeyPressMsg(tea.Key{Code: 'g', Mod: tea.ModCtrl}))
	fmt.Println("===== UI SHOT · THINK C — ctrl+g: full expand covers tools AND the thinking body =====")
	frameC := d.m.Frame()
	fmt.Println(frameC)
	fmt.Println("===== UI SHOT =====")
	strippedC := ansi.Strip(frameC)
	for _, want := range []string{
		"✓ Developer Task — Wire the SSE stream",   // expanded head: done glyph, no trailing rollup
		"  [tool] Edit internal/room/handler.go ✓", // tool row, 2-cell indent under the head
		"  thinking",                        // employee thought, expanded
		"writing the patch",                 // body renders on full expand
		"  · 2 tool calls · 1 think ✓ done", // bare closing summary line
	} {
		if !strings.Contains(strippedC, want) {
			return fail("threads-think C: frame missing %q", want)
		}
	}
	// natural order: read row < think body < edit row (chat arrival order —
	// the thought merged in place between the two tool calls)
	if strings.Index(strippedC, "[tool] Read") > strings.Index(strippedC, "writing the patch") ||
		strings.Index(strippedC, "writing the patch") > strings.Index(strippedC, "[tool] Edit") {
		return fail("threads-think C: think body must sit in natural chat order (between the read and edit rows)")
	}
	fmt.Println("asserts: OK — employee EvThought merges per CallID into the agent's thread (collapsed rollup keeps the \"· 1 think\" count), ctrl+g expands tools + thoughts in natural order, boss path byte-identical")
	return nil
}

// --- opencode-style thread rendering (--threads) ------------------------------
// ONE chat frame carrying BOTH thread states of the opencode-style worker
// renderer (internal/panels/threads_opencode.go): skopos-1's scout thread
// LIVE (animated braille glyph — the roster sprite is still working and the
// group's freshest wtool meta-tick sits inside the staleness horizon — with
// NO rollup while running, the BARE ↳ sneak at the newest merged tool call,
// and the live-only "ctrl+g · view subagents" hint row trailing the last
// thread block) beside tekton-1's COMPLETED thread (dim "✓" glyph after
// EvReturned fed the same timeline, "(· N tool calls ✓ done)" rollup).
// Every chat message is REDUCER-SHAPED: focusTool events flow through the
// REAL app reducer, which stamps Kind "wtool", Text "<verb> · <summary>"
// ("read · internal/panels/chat.go") and Meta "<state>␟<tick>" exactly like
// production — and the DISPLAY layer (shapeToolText) alone turns that into
// opencode's "<Verb> <rest>" form ("Read internal/panels/chat.go"). The
// fixture is never pre-shaped into the target text.

func runThreadsProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	d := newFocusDriver()
	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — threads stub online"})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"scout the question kinds, then patch the chat panel", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"Delegating — skopos-1 recons first; tekton-1 already wired the patch.", false)})

	// tekton-1's COMPLETED thread: dispatch → two tools (each running →
	// done merged per CallID) → EvReturned settles the roster sprite.
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "dev-1",
		Task: state.BoardTask{ID: "t1", Title: "Patch the chat panel", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "dev-1", TaskID: "t1"})
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/panels/chat.go", "running"))
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/panels/chat.go", "done"))
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/panels/chat.go", "running"))
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/panels/chat.go", "done"))
	d.send(state.Event{Kind: state.EvReturned, EmployeeID: "dev-1", TaskID: "t1",
		Mail: mail("m1", "tekton-1", "boss", "return: chat panel", "the patch is in.", state.MailReturn)})

	// skopos-1's LIVE thread: dispatched + still at the desk — the newest
	// tool call never resolves before the frame, so the braille glyph
	// carries the running state (the sneak itself stays BARE).
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "sco-1",
		Task: state.BoardTask{ID: "t2", Title: "Scout question kinds recon", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "sco-1", TaskID: "t2"})
	d.send(focusTool("sco-1", "skopos-1", "call-s1", "list", "internal/panels", "done"))
	d.send(focusTool("sco-1", "skopos-1", "call-s2", "read", "internal/panels/chat.go", "running"))
	d.pump(4) // tick 4 — inside the staleness horizon: skopos-1 stays LIVE
	fmt.Println("===== UI SHOT · THREADS — one LIVE thread (animated braille glyph, NO rollup while running, bare ↳ sneak) beside one COMPLETED thread (dim ✓ glyph, ✓ done rollup) =====")
	frame := d.m.Frame()
	fmt.Println(frame)
	fmt.Println("===== UI SHOT =====")

	stripped := ansi.Strip(frame)
	for _, want := range []string{
		"Explore Task — Scout question kinds recon",                       // scout head: RoleScout → "Explore"
		"✓ Developer Task — Patch the chat panel (· 2 tool calls ✓ done)", // settled head: dim ✓ + rollup
		"↳ Edit internal/panels/chat.go",                                  // done sneak: newest merged tool, opencode-shaped, bare
		"↳ Read internal/panels/chat.go",                                  // live sneak: newest call, opencode-shaped, bare
		"ctrl+g · view subagents",                                         // hint row trails the last thread block (≥1 live)
	} {
		if !strings.Contains(stripped, want) {
			return fail("threads: frame missing %q", want)
		}
	}
	// the live head carries the running state in its animated braille
	// glyph ALONE: NEVER the done "✓", and NEVER a trailing rollup
	if strings.Contains(stripped, "✓ Explore Task") {
		return fail("threads: the LIVE scout thread drew the done ✓ glyph")
	}
	if strings.Contains(stripped, "recon (· 2 tool calls") {
		return fail("threads: the LIVE scout head trailed the settled rollup — running heads carry NO rollup")
	}
	// no relics from the retired bordered-card renderer may survive
	for _, banned := range []string{"┌ tekton-1", "┌ skopos-1", "│ [tool]"} {
		if strings.Contains(stripped, banned) {
			return fail("threads: retired bordered-card relic %q in the frame", banned)
		}
	}
	if err := assertChatLayout("threads", frame); err != nil {
		return err
	}
	fmt.Println("asserts: OK — ONE frame carries both thread states: LIVE scout (animated braille glyph, \"Explore Task\" title, NO rollup while running, bare ↳ sneak, ctrl+g hint row) beside COMPLETED developer (dim ✓ glyph, \"✓ done\" rollup, bare ↳ sneak) — every row reducer-shaped in data (Kind wtool, \"<verb> · <summary>\" text), opencode \"<Verb> <rest>\" form applied by the display layer, no pre-shaped fixture text, no bordered-card relics, every chat row inside the divider budget")
	return nil
}

// --- per-call inline thread diffs (--wdiff) ----------------------------------
// The opencode wire carries the patch of every completed Edit tool part
// (state.metadata.filediff); the backend lifts it into a CallID-keyed
// EvFileDiff, the app pins it INSIDE the worker thread as Kind "wdiff",
// and the panel shows it as (collapsed) a dim "· +A -D" suffix on the
// thread's ↳ sneak and (expanded) a suffix on the matching "[tool] Edit"
// row + a one-row "↳ diff · path +A -D" sub-row directly beneath that
// row — a CLICK on the sub-row opens/closes the parsed line-numbered
// body (the flat-diff machinery reused verbatim). Boss/fetch-level diffs
// keep the flat ctrl+d world: this fixture is the per-call seam only.

// focusDiff — the CallID-keyed EvFileDiff the backend emits right after
// a completed edit/write EvTool (per-call worker diffs).
func focusDiff(ownerID, ownerName, callID, path, body string, add, del int) state.Event {
	return state.Event{Kind: state.EvFileDiff, EmployeeID: ownerID, EmployeeName: ownerName,
		SessionID: ownerID, CallID: callID,
		DiffPath: path, DiffBody: body, DiffAdd: add, DiffDel: del}
}

func runWdiffProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	d := newFocusDriver()
	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — wdiff stub online"})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"patch the chat panel rendering", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"tekton-1 is on the patch — watch its thread.", false)})
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "dev-1",
		Task: state.BoardTask{ID: "t1", Title: "Patch the chat panel", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "dev-1", TaskID: "t1"})
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/panels/chat.go", "done"))
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/panels/chat.go", "running"))
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/panels/chat.go", "done"))
	// the patch riding the completed edit call (what metadata.filediff
	// normalizes into): the boss message above may interleave AFTER the
	// tool rows in the timeline — adjacency lands it right under the
	// edit row regardless
	d.send(focusDiff("dev-1", "tekton-1", "call-t2", "internal/panels/chat.go",
		"--- a/internal/panels/chat.go\n"+
			"+++ b/internal/panels/chat.go\n"+
			"@@ -1784,7 +1784,9 @@\n"+
			" first = false\n"+
			" m := item.Msg\n"+
			" switch {\n"+
			"-case m.Kind == thinkKind:\n"+
			"+case m.Kind == thinkKind && !c.compactThreads:\n"+
			"+	// compact threads keep thinking folded\n"+
			" 	c.renderThink(&b, m)\n"+
			"-case m.Kind == toolKind:\n"+
			"+case m.Kind == toolKind || m.Kind == wdiffKind:\n"+
			"+	// per-call diffs ride their tool row\n"+
			" 	toolW := c.contentW() - 1\n",
		8, 3))
	// tekton-1 returns — the head settles to the dim ✓ glyph + rollup
	d.send(state.Event{Kind: state.EvReturned, EmployeeID: "dev-1", TaskID: "t1",
		Mail: mail("m1", "tekton-1", "boss", "return: chat panel", "patch is in.", state.MailReturn)})
	d.pump(4)
	fmt.Println("===== UI SHOT · WDIFF A — collapsed thread: the ↳ sneak of the newest tool gains the dim \"· +8 -3\" count suffix =====")
	frameA := d.m.Frame()
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")
	strippedA := ansi.Strip(frameA)
	for _, want := range []string{
		"Developer Task — Patch the chat panel (· 2 tool calls ✓ done)", // diff NOT a 3rd call
		"↳ Edit internal/panels/chat.go · +8 -3",                        // sneak suffix
	} {
		if !strings.Contains(strippedA, want) {
			return fail("wdiff A: frame missing %q", want)
		}
	}
	if strings.Contains(strippedA, "↳ diff ·") || strings.Contains(strippedA, "compactThreads") {
		return fail("wdiff A: the collapsed thread must not show the ↳ diff row or body")
	}

	// ctrl+g — expanded: the tool row suffix + the ↳ diff sub-row; body
	// still closed
	d.send(tea.KeyPressMsg(tea.Key{Code: 'g', Mod: tea.ModCtrl}))
	fmt.Println("===== UI SHOT · WDIFF B — ctrl+g expanded: \"[tool] Edit … ✓ · +8 -3\" + the \"↳ diff · path +8 -3\" sub-row =====")
	frameB := d.m.Frame()
	fmt.Println(frameB)
	fmt.Println("===== UI SHOT =====")
	strippedB := ansi.Strip(frameB)
	for _, want := range []string{
		"  [tool] Read internal/panels/chat.go ✓",         // the suffix-free sibling
		"  [tool] Edit internal/panels/chat.go ✓ · +8 -3", // the edited call's suffix
		"  ↳ diff · internal/panels/chat.go +8 -3",        // the per-call sub-row
	} {
		if !strings.Contains(strippedB, want) {
			return fail("wdiff B: frame missing %q", want)
		}
	}
	if strings.Contains(strippedB, "compactThreads") {
		return fail("wdiff B: the diff body stays closed until the ↳ row is clicked")
	}
	if err := assertChatLayout("wdiff B", frameB); err != nil {
		return err
	}

	// click the ↳ diff sub-row — REAL mouse coords off the rendered frame
	_, _, _, floorW := d.m.LayoutInfo()
	diffY := -1
	for i, ln := range strings.Split(strippedB, "\n") {
		if strings.Contains(ln, "↳ diff · internal/panels/chat.go") {
			diffY = i
			break
		}
	}
	if diffY < 0 {
		return fail("wdiff C setup: the ↳ diff sub-row is not in the frame")
	}
	clickAt(d, floorW+5, diffY)
	fmt.Println("===== UI SHOT · WDIFF C — clicked the ↳ diff row: the parsed line-numbered body opens IN THE THREAD =====")
	frameC := d.m.Frame()
	fmt.Println(frameC)
	fmt.Println("===== UI SHOT =====")
	strippedC := ansi.Strip(frameC)
	for _, want := range []string{
		"compactThreads",                        // an addition row's body text
		"  [tool] Edit internal/panels/chat.go", // the thread itself stayed expanded
		"  ↳ diff · internal/panels/chat.go +8 -3",
	} {
		if !strings.Contains(strippedC, want) {
			return fail("wdiff C: frame missing %q after the ↳ click", want)
		}
	}
	// click again — the body folds back, the ↳ sub-row survives
	strippedC2 := strings.Split(ansi.Strip(d.m.Frame()), "\n")
	diffY2 := -1
	for i, ln := range strippedC2 {
		if strings.Contains(ln, "↳ diff · internal/panels/chat.go") {
			diffY2 = i
			break
		}
	}
	if diffY2 < 0 {
		return fail("wdiff D setup: the ↳ diff row vanished under the open body")
	}
	clickAt(d, floorW+5, diffY2)
	strippedD := ansi.Strip(d.m.Frame())
	if strings.Contains(strippedD, "compactThreads") {
		return fail("wdiff D: the second click did not close the diff body")
	}
	if !strings.Contains(strippedD, "  ↳ diff · internal/panels/chat.go +8 -3") {
		return fail("wdiff D: the ↳ diff sub-row must survive the fold")
	}
	if err := assertChatLayout("wdiff C", frameC); err != nil {
		return err
	}
	fmt.Println("asserts: OK — per-call diffs pin inside the worker thread (sneak/tool-row \"· +A -D\" suffix, \"↳ diff · path\" sub-row directly beneath its tool row, click opens/closes the line-numbered body, rollup counts the diff as neither tool nor think, boss/fetch-level flat diffs untouched)")
	return nil
}

// --- thread focus view (--threadfocus) ---------------------------------------
// The ctrl+f fullscreen thread panel: two threads scripted (skopos FIRST
// so tekton-1's fresher tags win the live chain; skopos then SETTLES so
// tekton-1 alone is live), ctrl+f mounts the pane on tekton-1 — frame
// asserts: the title row (glyph + "Developer Task — Wire the SSE
// stream · 2 tool calls"), the FULL body rows (tools, the ↳ diff
// sub-row — clicked open through the REAL mouse seam), the hint bar's
// "esc · ctrl+f back to office", then esc closes back to the office
// BYTE-FOR-BYTE.

func runThreadFocusProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	d := newFocusDriver()
	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — threadfocus stub online"})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"wire the sse stream", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"tekton-1 is on the stream — the scout is sweeping.", false)})
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "sco-1",
		Task: state.BoardTask{ID: "t2", Title: "Scan the repo", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "sco-1", TaskID: "t2"})
	d.send(focusTool("sco-1", "skopos-1", "call-s1", "grep", "SSE, 3 hits", "done"))
	d.pump(2)
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "dev-1",
		Task: state.BoardTask{ID: "t1", Title: "Wire the SSE stream", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "dev-1", TaskID: "t1"})
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/room/manager.go", "done"))
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/room/handler.go", "done"))
	d.send(focusDiff("dev-1", "tekton-1", "call-t2", "internal/room/handler.go",
		"--- a/internal/room/handler.go\n"+
			"+++ b/internal/room/handler.go\n"+
			"@@ -33,6 +33,9 @@\n"+
			" func (h *Handler) Serve() {\n"+
			"-\tmux := http.NewServeMux()\n"+
			"+\tmux := h.routes()\n"+
			"+\tmux.Handle(\"/events\", h.sse)\n"+
			"+\thub.fanout(h.sseEvents)\n"+
			" \treturn mux.ServeHTTP(w, r)\n"+
			" }",
		4, 1))
	// the scout settles — tekton-1 is the ONLY live thread now
	d.send(state.Event{Kind: state.EvReturned, EmployeeID: "sco-1", TaskID: "t2",
		Mail: mail("m1", "skopos-1", "boss", "return: scan the repo", "12 hits, all mapped.", state.MailReturn)})
	d.pump(3)
	pre := d.m.Frame()

	// ctrl+f — the live chain resolves to tekton-1
	d.send(tea.KeyPressMsg(tea.Key{Code: 'f', Mod: tea.ModCtrl}))
	fmt.Println("===== UI SHOT · FOCUS A — ctrl+f: tekton-1's thread fullscreen (header glyph + title + counters, FULL body, ↳ diff sub-row, the esc-hint bar) =====")
	frameA := d.m.Frame()
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")
	strippedA := ansi.Strip(frameA)
	for _, want := range []string{
		"Developer Task — Wire the SSE stream · 2 tool calls", // header title + counters
		"  [tool] Read internal/room/manager.go ✓",            // merged tool row
		"  [tool] Edit internal/room/handler.go ✓ · +4 -1",    // the edited call's suffix
		"  ↳ diff · internal/room/handler.go +4 -1",           // the per-call sub-row
		"esc · ctrl+f back to office",                         // the hint bar's leave copy
	} {
		if !strings.Contains(strippedA, want) {
			return fail("threadfocus A: frame missing %q", want)
		}
	}
	if strings.Contains(strippedA, "mux.Handle(\"/events\", h.sse)") {
		return fail("threadfocus A: the wdiff body must wait for the ↳ click")
	}
	if strings.Contains(strippedA, "SSE, 3 hits") {
		return fail("threadfocus A: the sibling thread (skopos-1) leaked into the fullscreen pane")
	}
	// the sibling thread is HIDDEN but the office stays alive underneath —
	// the topbar/statusbar chrome rows survive (the focus covers the mid only)
	if !strings.Contains(frameA, "theboringoffice") && !strings.Contains(strippedA, "agents | board") {
		return fail("threadfocus A: topbar/statusbar chrome vanished under the focus")
	}

	// click the ↳ diff sub-row — REAL screen coords (the pane spans the
	// FULL width: x inside it is just the row itself, +1 topbar row above)
	diffY := -1
	for i, ln := range strings.Split(strippedA, "\n") {
		if strings.Contains(ln, "↳ diff · internal/room/handler.go") {
			diffY = i
			break
		}
	}
	if diffY < 0 {
		return fail("threadfocus B setup: the ↳ diff sub-row is not in the frame")
	}
	clickAt(d, 6, diffY)
	fmt.Println("===== UI SHOT · FOCUS B — clicked the ↳ diff row: the parsed line-numbered body opens INSIDE the pane =====")
	frameB := d.m.Frame()
	fmt.Println(frameB)
	fmt.Println("===== UI SHOT =====")
	strippedB := ansi.Strip(frameB)
	for _, want := range []string{
		"mux.Handle(\"/events\", h.sse)", // an addition row's body text
		"  ↳ diff · internal/room/handler.go +4 -1",
	} {
		if !strings.Contains(strippedB, want) {
			return fail("threadfocus B: frame missing %q after the ↳ click", want)
		}
	}

	// esc — back to the office, byte-identical to the covered frame
	d.send(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	fmt.Println("===== UI SHOT · FOCUS C — esc: back to the office, scroll + draft untouched =====")
	frameC := d.m.Frame()
	fmt.Println(frameC)
	fmt.Println("===== UI SHOT =====")
	strippedC := ansi.Strip(frameC)
	if strings.Contains(strippedC, "esc · ctrl+f back to office") {
		return fail("threadfocus C: the focus hint bar survived esc")
	}
	if !strings.Contains(strippedC, "Explore Task — Scan the repo") && !strings.Contains(strippedC, "tekton-1 is on the stream") {
		return fail("threadfocus C: the office frame did not come back (both threads + boss reply hidden?)")
	}
	if frameC != pre {
		return fail("threadfocus C: the office must return BYTE-IDENTICAL across open+close")
	}
	fmt.Println("asserts: OK — ctrl+f resolves the live thread into the fullscreen pane (header glyph + title + \"· 2 tool calls\", FULL body with the \"· +4 -1\" suffix + \"↳ diff\" sub-row), the ↳ click opens the parsed body inside the pane, the hint bar reads \"esc · ctrl+f back to office\", esc returns the office byte-identical")
	return nil
}

// --- completion board sync (--boardsync) -----------------------------------
// The wave's board reconcile, shown the way the live backend emits it
// (internal/backend/boardsync.go — its rules are go-test-proven there; the
// uishot is the RENDER proof). THREE DOING rows staged with fixed stamps
// (byte determinism): tekton-1 owns t1 (older) + t2 (newer), skopos-1 owns
// t3 "Wire R17 Razorpay mandate". TWO returns drive the flip story:
//
//	return 1 — tekton-1 returns t2 (the NEWER brief): the exact-path
//	  EvTask-done, then the sweep's output — the OLDEST same-owner row
//	  (t1) flips + the ONE dim note "[office] board sync: flipped 1 rows
//	  to done" (exactly one note per flipped batch);
//	return 2 — tekton-2 (a DISTINCT owner) returns "Wire R17 Razorpay
//	  mandate" — skopos-1's doing row shares the title verbatim but a
//	  named completion only matches its own worker: t3 NEVER flips, and
//	  no second note fires.
//
// Frame A shows the three DOING rows + "board 0/3/0"; frame B shows the
// settled board (t1/t2/the exact row of return 2 DONE, skopos-1's t3 STILL
// DOING) + "board 0/1/3" + the note. The whole synchronous drive runs twice
// and both frame pairs must be byte-identical.

func boardSyncDrive(rep int) (frameA, frameB string, err error) {
	d := newFocusDriver()
	if !d.m.SelectTab("board") {
		return "", "", fmt.Errorf("unknown tab %q", "board")
	}
	// Package-global office plan + walker state (internal/office/floor*.go)
	// outlives ONE drive inside a process: the FIRST Frame() locks the real
	// floor-sized plan — hiring before it would seat on the 120x28 default
	// (rep 1) and the locked plan (rep 2), diverging. So: lock the plan
	// FIRST (newSocialDriver's step), and rep-prefix employee IDs against
	// the walkers map (identical names/seats/glyphs — the frame never shows
	// the IDs). dev-2 lands the "floor-0" overflow spot on the narrow shot
	// plan — deterministic, and the sprites are ambient to a BOARD proof.
	pref := fmt.Sprintf("bsc%d", rep)
	_ = d.m.Frame() // lock the floor plan before any sprite advance/seat assign
	hire := func(id, name string, role state.EmployeeRole) {
		d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
			ID: pref + "-" + id, Name: name, Role: role, Sprite: state.SpriteAtDesk}})
	}
	task := func(id, title, owner string, status state.TaskStatus, at int64) state.BoardTask {
		return state.BoardTask{ID: id, Title: title, Status: status, Owner: owner, At: at}
	}
	retMail := func(id, from, subject string) state.MailItem {
		return state.MailItem{ID: id, From: from, To: "manager", At: 500,
			Subject: subject, Body: "done.", Kind: state.MailReturn}
	}

	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — boardsync stub online"})
	hire("dev-1", "tekton-1", state.RoleDeveloper)
	hire("dev-2", "tekton-2", state.RoleDeveloper)
	hire("sco-1", "skopos-1", state.RoleScout)
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: pref + "-dev-1",
		Task: task("t1", "Audit morning side", "tekton-1", state.TaskInProgress, 100)})
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: pref + "-dev-1",
		Task: task("t2", "Audit lints", "tekton-1", state.TaskInProgress, 200)})
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: pref + "-sco-1",
		Task: task("t3", "Wire R17 Razorpay mandate", "skopos-1", state.TaskInProgress, 300)})
	d.pump(2)
	frameA = d.m.Frame()

	// return 1 — tekton-1 closes t2 (exact path), the sweep drains his
	// OLDEST stranded row (t1) behind it; ONE note for the flipped batch.
	d.send(state.Event{Kind: state.EvTask,
		Task: task("t2", "Audit lints", "tekton-1", state.TaskDone, 200)})
	d.send(state.Event{Kind: state.EvReturned, EmployeeID: pref + "-dev-1", TaskID: "t2",
		Mail: retMail("m1", "tekton-1", "return: Audit lints")})
	d.send(state.Event{Kind: state.EvTask,
		Task: task("t1", "Audit morning side", "tekton-1", state.TaskDone, 100)})
	d.send(state.Event{Kind: state.EvStatus, Text: "[office] board sync: flipped 1 rows to done"})

	// return 2 — tekton-2 (distinct owner, title twin of skopos-1's row):
	// his own exact row closes — the reconcile emits NOTHING for skopos-1
	// (worker collision = never flip with doubt) and stays silent.
	d.send(state.Event{Kind: state.EvTask,
		Task: task("task-ses-dev-2", "Wire R17 Razorpay mandate", "tekton-2", state.TaskDone, 400)})
	d.send(state.Event{Kind: state.EvReturned, EmployeeID: pref + "-dev-2", TaskID: "task-ses-dev-2",
		Mail: retMail("m2", "tekton-2", "return: Wire R17 Razorpay mandate")})
	d.pump(2)
	frameB = d.m.Frame()
	return frameA, frameB, nil
}

func runBoardSyncProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	frameA, frameB, err := boardSyncDrive(1)
	if err != nil {
		return err
	}

	fmt.Println("===== UI SHOT · BOARDSYNC A — before the returns: THREE doing rows (tekton-1 ×2 oldest-first 100→200, skopos-1), done lane empty, \"board 0/3/0\" =====")
	fmt.Println(frameA)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("===== UI SHOT · BOARDSYNC B — after the two returns: t2 closed by its exact path, the sweep flipped tekton-1's OLDEST (t1) + the ONE note; skopos-1's title-twin row NEVER flipped (tekton-2's own row closed instead); \"board 0/1/3\" =====")
	fmt.Println(frameB)
	fmt.Println("===== UI SHOT =====")

	strippedA := ansi.Strip(frameA)
	strippedB := ansi.Strip(frameB)

	// frame A — the DOING pile, before the sweep. (Owner tags clip off the
	// 25-char twin title's row at this sidebar width — the twin is pinned
	// by lane + ansi state below; "Audit lints" shows its owner fully.)
	for _, want := range []string{
		"Audit morning side", "Audit lints tekton-1", "Wire R17 Razorpay mandate",
		"board 0/3/0", // three DOING, zero DONE (the statusbar's counts)
	} {
		if !strings.Contains(strippedA, want) {
			return fail("boardsync A: frame missing %q", want)
		}
	}
	if strings.Contains(strippedA, "board sync: flipped") {
		return fail("boardsync A: the sync note fired before any completion")
	}

	// frame B — the settled board.
	// the reconcile note lands EXACTLY once (the status line) — one note
	// per flipped batch, and return 2 adds none (a flipless sweep is silent).
	if n := strings.Count(strippedB, "[office] board sync: flipped 1 rows to done"); n != 1 {
		return fail("boardsync B: the sync note must appear exactly once, got %d", n)
	}
	if !strings.Contains(strippedB, "board 0/1/3") {
		return fail("boardsync B: expected the post-flip counts \"board 0/1/3\"")
	}
	// the distinct-owner pair: skopos-1's DOING row and tekton-2's DONE row
	// share the title — both render, each in its lane's paint (plain Info
	// cyan for the DOING twin, faint OK green for the DONE one).
	if n := strings.Count(strippedB, "Wire R17 Razorpay mandate"); n != 2 {
		return fail("boardsync B: the title-twin pair must both render (doing twin + done twin), got %d rows", n)
	}
	if n := strings.Count(frameB, "\x1b[36mWire R17 Razorpay mandate\x1b[m"); n != 1 {
		return fail("boardsync B: skopos-1's twin must STILL be DOING (plain lane paint), got %d", n)
	}
	if n := strings.Count(frameB, "\x1b[2;32mWire R17 Razorpay mandate\x1b[m"); n != 1 {
		return fail("boardsync B: tekton-2's exact-path twin must be DONE (faint paint), got %d", n)
	}
	// the reconciled OLDEST flip (t1) rides the same visual row as the
	// surviving DOING twin — the sweep never touched the other worker.
	mixed := false
	for _, ln := range strings.Split(strippedB, "\n") {
		if strings.Contains(ln, "Wire R17 Razorpay mandate") && strings.Contains(ln, "Audit morning side") {
			mixed = true
			break
		}
	}
	if !mixed {
		return fail("boardsync B: the surviving DOING twin must sit beside tekton-1's reconciled DONE row")
	}
	// tekton-1's two rows both settled (his return + the sweep's flip).
	if !strings.Contains(frameB, "\x1b[2;32mAudit lints\x1b[m") || !strings.Contains(frameB, "\x1b[2;32mAudit morning side\x1b[m") {
		return fail("boardsync B: tekton-1's exact AND oldest-flip rows must both be DONE (faint paint)")
	}

	// determinism: the same synchronous drive, byte-for-byte (rep-prefixed
	// employee ids + the plan locked pre-hire — see boardSyncDrive).
	frameA2, frameB2, err := boardSyncDrive(2)
	if err != nil {
		return err
	}
	if frameA != frameA2 || frameB != frameB2 {
		return fail("boardsync: two synchronous drives must produce byte-identical frames")
	}

	fmt.Println("asserts: OK — 3 DOING rows staged; return 1 closed its exact row and the sweep flipped tekton-1's OLDEST stranded row (owner-name, oldest-first) + ONE \"[office] board sync: flipped 1 rows to done\" note; return 2 (tekton-2, distinct owner, title twin) flipped NOTHING of skopos-1's — worker collision never flips, no second note; counts 0/3/0 → 0/1/3; two drives byte-identical")
	return nil
}

// --- inbound image previews (--images) ---------------------------------------
// The boss-turn image preview, RENDER proof (the unit contracts live in
// internal/panels' chat_raster tests + internal/app's model_image tests):
// the stub pins a completed boss turn EXACTLY the way opencode.go emits
// it — Msg.Meta the small "attach␟…␟hash" carrier, Event.Media the
// data-URL payload — over the shared gold fixture
// (internal/panels/testdata/checker-8x8.png). The drive feeds the REAL
// model synchronously AND executes the returned cmd tree breadth-first
// (the lazy rasterize probe is a tea.Cmd — spinner blinks and cursor
// heartbeats are dropped, everything else re-feeds, which is how the
// probe lands inside a synchronous proof). Asserts: the 🖼 chip reads
// "🖼 paste-diagram.png · 8×8 · image/png", the pinned half-block
// truecolor rows ride the frame verbatim, the /images off leg paints the
// chip alone, and the whole drive repeated is byte-identical.

// stubTermEnv — the HERMETIC terminal-env stub for the image drives:
// EVERY detect-layer input (termstub's checklist: TMUX, KITTY_WINDOW_ID,
// TERM_PROGRAM(+VERSION), WEZTERM_UNIX_SOCKET, VSCODE_PID,
// ITERM_SESSION_ID, TERM, COLORTERM) is owned by the drive, so the host
// terminal's markers (this repo's dev box runs ghostty — unstubbed, the
// "auto" posture would route the kitty lane and the ASCII base proof
// would pick the host's lane) never leak in. pairs override per leg.
// Returns the restore closure (every leg defers it).
func stubTermEnv(pairs ...[2]string) func() {
	keys := []string{
		"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM", "TERM_PROGRAM_VERSION",
		"WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID", "TERM", "COLORTERM",
	}
	type saved struct {
		v  string
		ok bool
	}
	prev := map[string]saved{}
	for _, k := range keys {
		v, ok := os.LookupEnv(k)
		prev[k] = saved{v, ok}
		os.Setenv(k, "")
	}
	for _, p := range pairs {
		os.Setenv(p[0], p[1])
	}
	return func() {
		for k, s := range prev {
			if s.ok {
				os.Setenv(k, s.v)
			} else {
				os.Unsetenv(k)
			}
		}
	}
}

// asciiTermEnv — the plain-xterm stub: "auto" resolves the universal
// ASCII lane, the v1 base proof's paint on ANY host.
func asciiTermEnv() [][2]string {
	return [][2]string{{"TERM_PROGRAM", "Apple_Terminal"}, {"TERM", "xterm-256color"}}
}

// imagesDrive — ONE synchronous image-preview run (off==true drives the
// /images-posture-off leg) under the HERMETIC stub env (nil = neutral
// ascii — the base legs are host-independent). Returns the final frame.
func imagesDrive(off bool, env [][2]string) (string, error) {
	defer stubTermEnv(env...)()
	backend := &stubBackend{done: make(chan struct{})}
	cfg := config.Default()
	if off {
		cfg.UI.Images = "off"
	}
	m := app.New(backend, cfg)
	// runExec mirrors internal/app/attach_queue_test.go's runMsg: feed the
	// msg, then drain the returned cmd tree breadth-first, dropping the
	// self-re-arming heartbeats (spinner tick, cursor blink). Every OTHER
	// landing msg re-feeds — the image probe's landing included.
	runExec := func(msg tea.Msg) {
		tm, cmd := m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			out := c()
			if out == nil {
				continue
			}
			switch out := out.(type) {
			case tea.BatchMsg:
				queue = append(queue, out...)
			case spinner.TickMsg, cursor.BlinkMsg:
				// heartbeats re-arm forever — dropped, exactly as runMsg does
			default:
				tm2, next := m.Update(out)
				if fm2, ok := tm2.(app.Model); ok {
					m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}
	runExec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — image-preview stub online"})
	runExec(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	runExec(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"what does the diagram look like?", false)})
	runExec(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})

	raw, err := os.ReadFile("internal/panels/testdata/checker-8x8.png")
	if err != nil {
		return "", fmt.Errorf("image fixture: %w", err)
	}
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
	it := state.MediaItem{Mime: "image/png", Filename: "paste-diagram.png", W: 8, H: 8,
		Hash: state.DataURLHash(dataURL), URL: dataURL}
	runExec(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m1", From: "boss", Kind: "boss",
		Text: "Red and blue squares, alternating — the classic checker.",
		At:   1, Pending: false,
		Meta: state.MediaMeta([]state.MediaItem{it}),
	}, Media: []state.MediaItem{it}})
	// NOTE: no EvTick pump here — a tick re-arms the model's tickCmd
	// (tea.Tick → EvTick → …) and a breadth-first drain would chase it
	// forever. The raster probe is synchronous INSIDE the pin's own cmd
	// tree (imageRasterMsg lands, SetImageRaster repaints) — the frame is
	// settled the moment runExec returns.
	return m.Frame(), nil
}

// imagesPinnedRow — the checker's EXACT pinned raster row (panels'
// TestRasterCheckerExactPin contract: fg=top pixel, bg=bottom pixel,
// red|blue alternating, reset tail) — all four half-block rows are this
// string (parities repeat every two pixel rows).
func imagesPinnedRow() string {
	var b strings.Builder
	for x := 0; x < 8; x++ {
		if x%2 == 0 {
			b.WriteString("\x1b[38;2;255;0;0m\x1b[48;2;0;0;255m▀")
		} else {
			b.WriteString("\x1b[38;2;0;0;255m\x1b[48;2;255;0;0m▀")
		}
	}
	b.WriteString("\x1b[m")
	return b.String()
}

// imagesLaneEnvs — the --lane legs' hermetic terminal envs (EVERY
// detect-layer input stubbed, per-leg overrides — the host's real
// terminal never participates):
var imagesLaneEnvs = map[string][][2]string{
	"kitty": {{"TERM_PROGRAM", "ghostty"}, {"TERM", "xterm-ghostty"}},
	"iterm": {{"TERM_PROGRAM", "iTerm.app"}, {"ITERM_SESSION_ID", "w0t0p0:uishot-stub"}, {"TERM", "xterm-256color"}},
	"ascii": {{"TERM_PROGRAM", "Apple_Terminal"}, {"TERM", "xterm-256color"}},
}

// runImagesLaneLeg — ONE native-lane leg of the --lane list: the same
// checker drive under the lane's stub env, byte-pinning the lane's
// escape frame (kitty strip / OSC 1337) or — ascii — the v1 pinned
// rows. Every leg drives TWICE (byte-determinism).
func runImagesLaneLeg(fail func(string, ...any) error, lane string) error {
	env, ok := imagesLaneEnvs[lane]
	if !ok {
		return fail("images --lane: unknown lane %q (kitty|iterm|ascii)", lane)
	}
	// the process-singleton registry: each drive's Frame publishes
	// wholesale — clear up front so a PREVIOUS leg's entries never leak
	// into this leg's snapshot (and never into the next leg's).
	panels.ZenbuRegistry().Clear()
	panels.ZenbuRegistry().PublishChatMedia(nil)
	defer func() { panels.ZenbuRegistry().Clear(); panels.ZenbuRegistry().PublishChatMedia(nil) }()
	frame, err := imagesDrive(false, env)
	if err != nil {
		return err
	}
	chat1 := panels.ZenbuRegistry().ChatSnapshotForTest()
	frame2, err := imagesDrive(false, env)
	if err != nil {
		return err
	}
	chat2 := panels.ZenbuRegistry().ChatSnapshotForTest()
	if frame != frame2 {
		return fail("images --lane %s: two drives must produce byte-identical frames", lane)
	}
	if fmt.Sprintf("%+v", chat1) != fmt.Sprintf("%+v", chat2) {
		return fail("images --lane %s: two drives must publish byte-identical chat-media splices", lane)
	}
	raw, err := os.ReadFile("internal/panels/testdata/checker-8x8.png")
	if err != nil {
		return fmt.Errorf("image fixture: %w", err)
	}
	b64 := base64.StdEncoding.EncodeToString(raw)
	stripped := ansi.Strip(frame)
	switch lane {
	case "kitty":
		// the wave-86 splice routing: the media rows are PURE reservation
		// rows — ZERO APC bytes in the View frame (the renderer would drop
		// them anyway; the wave-81 forensics) — and the chat-media REGION
		// of the frame registry carries the preview instead: {office id =
		// sha1(source)[:8], the ABSOLUTE cell one row below the frame's
		// own chip row at the sidebar's bubble indent, the cached verbatim
		// strip}. The wrapper-level leg then proves the splice bytes: a
		// renderer flush through ZenbuFrameWriter lands DECSC + CUP(the
		// pinned absolute cell) + the f=100 APC + DECRC.
		id8 := panels.KittyIDHash8(panels.KittyImageID(raw))
		wantStart := "\x1b_Ga=T,t=d,f=100,i=" + id8 + ",q=2;"
		if strings.Contains(frame, "\x1b_G") {
			return fail("images --lane kitty: the media rows are APC-free (the splice owns the strip)")
		}
		if strings.Contains(frame, "\x1b]1337;") {
			return fail("images --lane kitty: no OSC 1337 bytes ride the kitty drive")
		}
		if strings.Contains(stripped, "▀") {
			return fail("images --lane kitty: the kitty lane paints ZERO half-blocks")
		}
		if strings.Contains(frame, imagesPinnedRow()) {
			return fail("images --lane kitty: no ASCII rows ride the kitty drive")
		}
		chat := chat2
		if len(chat) != 1 {
			return fail("images --lane kitty: the chat-media region holds exactly the one preview, got %d", len(chat))
		}
		if chat[0].OfficeID != panels.KittyImageID(raw) || !strings.HasPrefix(chat[0].Frame, wantStart) {
			return fail("images --lane kitty: the registry carries the exact strip start %q under i=%s", wantStart, id8)
		}
		if !strings.Contains(chat[0].Frame, b64+"\x1b\\") {
			return fail("images --lane kitty: the base64 payload + the ESC\\ terminator ride the registry's strip")
		}
		if strings.Contains(chat[0].Frame, "c=") || strings.Contains(chat[0].Frame, ",r=") {
			return fail("images --lane kitty: the chat APC carries NO c=/r= (the wave-81 emission ruling)")
		}
		// the pinned ABSOLUTE cell, cross-read against the frame's own
		// chip row: the preview paints exactly one row below its chip.
		chipRow := -1
		for i, ln := range strings.Split(frame, "\n") {
			if strings.Contains(ansi.Strip(ln), "🖼 paste-diagram.png") {
				chipRow = i
				break
			}
		}
		if chipRow < 0 || chat[0].OY != chipRow+1 {
			return fail("images --lane kitty: the preview paints one row below the chip (chip row %d, published OY %d)", chipRow, chat[0].OY)
		}
		// the wrapper-level leg: one renderer flush splices DECSC + CUP
		// (the published absolute cell, 1-based) + the strip + DECRC; a
		// second identical flush splices NOTHING (the bandwidth skip).
		var buf strings.Builder
		w := panels.NewZenbuFrameWriter(&buf, panels.ZenbuRegistry())
		_, _ = w.Write([]byte("FLUSH"))
		wantSplice := "FLUSH\x1b7\x1b[" + fmt.Sprintf("%d", chat[0].OY+1) + ";" + fmt.Sprintf("%d", chat[0].OX+1) + "H" + chat[0].Frame + "\x1b8"
		if buf.String() != wantSplice {
			return fail("images --lane kitty: the wrapper splices the strip at the pinned cell:\n got %q\nwant %q", buf.String()[:100], wantSplice[:100])
		}
		buf.Reset()
		_, _ = w.Write([]byte("FLUSH2"))
		if buf.String() != "FLUSH2" {
			return fail("images --lane kitty: the identical second flush splices nothing (the bandwidth skip), got %q", buf.String()[:80])
		}
		fmt.Printf("lane leg kitty: OK — the media rows are APC-free; the registry holds the preview (i=%s, %d b64 octets, ESC\\-terminated, no c=/r=) at the absolute cell (%d,%d) one row below the frame's chip row %d; the wrapper splices DECSC + CUP(%d;%d) + f=100 APC + DECRC (the second flush skips)\n",
			id8, len(b64), chat[0].OX, chat[0].OY, chipRow, chat[0].OY+1, chat[0].OX+1)
	case "iterm":
		// OSC 1337: ESC ] 1337 ;File=inline=1;width=8:height=4;base64,<b64> ^G
		wantPin := "\x1b]1337;File=inline=1;width=8:height=4;base64,"
		if !strings.Contains(frame, wantPin) {
			return fail("images --lane iterm: the frame must carry the OSC 1337 marker (inline=1, width=8:height=4, base64,)")
		}
		if !strings.Contains(frame, b64+"\x07") {
			return fail("images --lane iterm: the inline payload + the BEL (^G) terminator must ride the frame")
		}
		if strings.Contains(stripped, "▀") {
			return fail("images --lane iterm: the OSC 1337 lane paints ZERO half-blocks")
		}
		fmt.Printf("lane leg iterm: OK — iTerm-stub env routed OSC 1337 inline (width=8:height=4, %d b64 octets, BEL-terminated), zero half-blocks\n", len(b64))
	case "ascii":
		if c := strings.Count(frame, imagesPinnedRow()); c != 4 {
			return fail("images --lane ascii: the pinned checker row ×4, got %d", c)
		}
		if strings.Contains(frame, "\x1b_Ga=T") || strings.Contains(frame, "\x1b]1337;") {
			return fail("images --lane ascii: NO native-lane escapes ride the ascii drive")
		}
		fmt.Println("lane leg ascii: OK — neutral env kept the 4 pinned half-block rows, zero native escapes")
	}
	// every lane keeps the 🖼 chip + the chip→body ordering.
	if !strings.Contains(stripped, "🖼 paste-diagram.png · 8×8 · image/png") {
		return fail("images --lane %s: the 🖼 chip with dims + mime must render", lane)
	}
	if !strings.Contains(stripped, "Red and blue squares, alternating") {
		return fail("images --lane %s: the boss body must render below the preview", lane)
	}
	return nil
}

func runImagesProof(lanes []string) error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	frame, err := imagesDrive(false, asciiTermEnv())
	if err != nil {
		return err
	}
	fmt.Println("===== UI SHOT · IMAGES — completed boss turn carrying the 8×8 checker file part: 🖼 chip + inline half-block truecolor preview =====")
	fmt.Println(frame)
	fmt.Println("===== UI SHOT =====")

	stripped := ansi.Strip(frame)
	if !strings.Contains(stripped, "🖼 paste-diagram.png · 8×8 · image/png") {
		return fail("images: the 🖼 chip with dims + mime must render")
	}
	if n := strings.Count(stripped, "▀"); n != 32 {
		return fail("images: an 8-col × 4-row checker paints 32 half-blocks, got %d", n)
	}
	if c := strings.Count(frame, imagesPinnedRow()); c != 4 {
		return fail("images: the pinned checker row must ride the frame verbatim ×4, got %d", c)
	}

	// the /images off leg: chips alone — the probe never fired.
	offFrame, err := imagesDrive(true, asciiTermEnv())
	if err != nil {
		return err
	}
	offStripped := ansi.Strip(offFrame)
	if !strings.Contains(offStripped, "🖼 paste-diagram.png · 8×8 · image/png") {
		return fail("images: the off leg still paints the chip")
	}
	if strings.Contains(offStripped, "▀") {
		return fail("images: the off leg must paint ZERO half-blocks")
	}

	// determinism: the same synchronous drive, byte-for-byte.
	frame2, err := imagesDrive(false, asciiTermEnv())
	if err != nil {
		return err
	}
	if frame != frame2 {
		return fail("images: two synchronous drives must produce byte-identical frames")
	}
	fmt.Println("asserts: OK — the completed boss turn's file part previews inline: \"🖼 paste-diagram.png · 8×8 · image/png\" chip + 4 pinned half-block rows (32 ▀ cells, red-over-blue truecolor SGR); the /images off leg paints the chip alone (the probe never fires); two drives byte-identical")

	// the --lane legs: native-lane proofs under hermetic stub envs.
	for _, lane := range lanes {
		lane = strings.TrimSpace(lane)
		if lane == "" {
			continue
		}
		if err := runImagesLaneLeg(fail, lane); err != nil {
			return err
		}
	}
	return nil
}

// --- chat image splice: the PRODUCTION-path PTY leg (--images-pty) -----
//
// runImagesPTY drives the chat image preview through main.go's EXACT
// wiring (never the snapshot harness): the REAL cursed renderer writes
// every flush through the ZenbuFrameWriter on os.Stdout, the stub pins
// the checker's completed boss turn through p.Send, and the probe's
// landing repaints. The PTY harness (/tmp/drive_chatimage.py, ghostty
// env, 130x32) counts the ESC_G f=100 frames on the wire: pre-fix the
// renderer eats the View-string APC (ZERO — the wave-81 forensics);
// post-fix the wrapper's chat-media splice lands CUP + the cached APC +
// restore after the flush.
func runImagesPTY() error {
	raw, err := os.ReadFile("internal/panels/testdata/checker-8x8.png")
	if err != nil {
		return fmt.Errorf("images-pty: image fixture: %w", err)
	}
	backend := &stubBackend{done: make(chan struct{})}
	m := app.New(backend, config.Default())
	frameOut := panels.NewZenbuFrameWriter(os.Stdout, panels.ZenbuRegistry())
	p := tea.NewProgram(m,
		tea.WithWindowSize(shotCols, shotRows),
		tea.WithInput(nil),
		tea.WithOutput(frameOut),
	)
	send := func(ev state.Event) { p.Send(ev) }
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(raw)
	it := state.MediaItem{Mime: "image/png", Filename: "paste-diagram.png", W: 8, H: 8,
		Hash: state.DataURLHash(dataURL), URL: dataURL}
	go func() {
		// the event sequence mirrors imagesDrive's synchronous drive (the
		// first event also readies the boot splash → the cascade lifts).
		time.Sleep(300 * time.Millisecond)
		send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — image-preview stub online"})
		send(state.Event{Kind: state.EvHire, Employee: state.Employee{
			ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
		send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
			"what does the diagram look like?", false)})
		time.Sleep(1400 * time.Millisecond) // the boot cascade lifts (~1.6s once ready)
		send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("boss-1", "boss", "", true)})
		send(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID: "bossmsg-m1", From: "boss", Kind: "boss",
			Text: "Red and blue squares, alternating — the classic checker.",
			At:   1, Pending: false,
			Meta: state.MediaMeta([]state.MediaItem{it}),
		}, Media: []state.MediaItem{it}})
		time.Sleep(1800 * time.Millisecond) // the probe lands + frames flush
		p.Quit()
	}()
	_, err = p.Run()
	frameOut.Finish()
	return err
}

// --- open-in-browser (--links) ---------------------------------------------
// The transcript's `o` hotkey, IN-APP KEY RECEIPT proof (the unit contracts
// live in internal/panels' links tests + internal/app's model_browser
// tests): a completed boss bubble carries (a) a URL in its body and (b) a
// media item whose filename is the shared checker FIXTURE — a REAL file, so
// the os.Stat gate verifies it (the proof runs from the repo root, like the
// --images fixture read). The member PRESSES on the bubble (a click's press
// half — the armed one-cell mark resolves the bubble; NO release, so the
// clipboard seam never runs in a shot), then presses `o`: TWO verified
// targets float the centered target card, enter picks the first (appear
// order: the body scan ahead of the media row), and the exec rides the
// STUBBED panels runner (SetOpenRunnerForShot — the real `open -g` would
// push a browser at the USER's screen mid-proof). Asserts: the · o (open)
// beacon rides the bubble, the card paints both rows, the runner captured
// EXACTLY the URL, the activity tab logs "→ opened: opencode.ai/docs", the
// no-mark leg types "o" into the draft (claim refusal), and the whole drive
// repeated is byte-identical.

// linksFixtureRel — the media filename the boss bubble claims, RELATIVE to
// the repo root (the proof's cwd): the os.Stat gate verifies it there.
const linksFixtureRel = "internal/panels/testdata/checker-8x8.png"

// linksFixtureAbs — the same fixture ABSOLUTE (what the resolver hands the
// runner): the capture asserts the delivered value, never the raw token.
func linksFixtureAbs() (string, error) {
	abs, err := filepath.Abs(linksFixtureRel)
	if err != nil {
		return "", fmt.Errorf("links fixture abs: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("links fixture must verify on disk: %w", err)
	}
	return abs, nil
}

// linksFrameOut — ONE synchronous drive's observed artifacts.
type linksFrameOut struct {
	frameBeacon  string // frame A: the bubble wearing the beacon, before the press
	frameCard    string // frame B: the target card floated over the transcript
	frameOpened  string // frame C: after enter — the card gone, the verdict logged
	opened       []panels.LinkTarget
	activityOpen string // the "→ opened:" activity row
	frameNoMark  string // frame D: the no-mark leg — `o` typed into the draft
}

func linksDrive() (linksFrameOut, error) {
	var out linksFrameOut
	restore := panels.SetOpenRunnerForShot(func(t panels.LinkTarget) error {
		out.opened = append(out.opened, t)
		return nil
	})
	defer restore()

	backend := &stubBackend{done: make(chan struct{})}
	m := app.New(backend, config.Default())
	// runExec — the exact breadth-first drain the --images proof runs
	// (self-re-arming heartbeats dropped; every other landing re-feeds,
	// which is how the open's browserOpenMsg verdict lands synchronously).
	runExec := func(msg tea.Msg) {
		tm, cmd := m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
				// heartbeats re-arm forever — dropped, exactly as runMsg does
			default:
				tm2, next := m.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}

	runExec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — open-in-browser stub online"})
	runExec(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"what did the doc + diagram say?", false)})
	// The completed boss bubble: URL in the body + the media item named at
	// the REAL checker fixture (chip-only — no data URL, the open half has
	// nothing to rasterize). TWO verified targets: the URL (body scan)
	// ahead of the media filename (the attach carrier).
	if _, err := linksFixtureAbs(); err != nil {
		return out, err
	}
	it := state.MediaItem{Mime: "image/png", Filename: linksFixtureRel, W: 8, H: 8}
	runExec(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m1", From: "boss", Kind: "boss",
		Text: "Spec: https://opencode.ai/docs — the diagram sits in the repo.",
		At:   1, Pending: false,
		Meta: state.MediaMeta([]state.MediaItem{it}),
	}})

	out.frameBeacon = m.Frame()

	// the press: find the bubble's first transcript row in the frame (the
	// 🖼 chip rides above the body) and click INSIDE the chat panel there
	// (the sidebar is the rightmost panel: floorW = shotCols − sidebar 80,
	// +10 cells inside its viewport region).
	chipRow, pressX := -1, (shotCols-80)+10
	for i, ln := range strings.Split(out.frameBeacon, "\n") {
		if strings.Contains(ln, "checker-8x8.png") {
			chipRow = i
			break
		}
	}
	if chipRow < 0 {
		return out, fmt.Errorf("links: the 🖼 chip row must render in the beacon frame")
	}
	runExec(tea.MouseClickMsg(tea.Mouse{X: pressX, Y: chipRow, Button: tea.MouseLeft}))

	// `o` — TWO verified targets float the target card.
	runExec(tea.KeyPressMsg(tea.Key{Code: 'o', Text: "o"}))
	out.frameCard = m.Frame()

	// enter picks the first target (appear order: URL before media).
	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	out.frameOpened = m.Frame()
	for _, ln := range m.ActivityLines() {
		if strings.Contains(ln, "→ opened:") {
			out.activityOpen = ln
		}
	}
	// the no-mark leg: a FRESH model pressing `o` with nothing marked —
	// the claim refuses and the key types into the draft.
	m2 := app.New(&stubBackend{done: make(chan struct{})}, config.Default())
	runExec2 := func(msg tea.Msg) { // the same drain, over m2
		tm, cmd := m2.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m2 = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
			default:
				tm2, next := m2.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					m2 = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}
	runExec2(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec2(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u2", "user", "hi", false)})
	runExec2(tea.KeyPressMsg(tea.Key{Code: 'o', Text: "o"}))
	out.frameNoMark = m2.Frame()
	return out, nil
}

func runLinksProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	fixtureAbs, err := linksFixtureAbs()
	if err != nil {
		return err
	}
	out, err := linksDrive()
	if err != nil {
		return err
	}
	fmt.Println("===== UI SHOT · LINKS A — the boss bubble wearing the · o (open) beacon (🖼 chip above the URL body) =====")
	fmt.Println(out.frameBeacon)
	fmt.Println("===== UI SHOT =====")
	stripped := ansi.Strip(out.frameBeacon)
	if !strings.Contains(stripped, "🖼 internal/panels/testdata/checker-8x8.png · 8×8 · image/png") {
		return fail("links A: the chip row renders with dims + mime")
	}
	if n := strings.Count(stripped, "o (open)"); n != 1 {
		return fail("links A: exactly ONE bubble wears the · o (open) beacon, got %d", n)
	}

	fmt.Println("===== UI SHOT · LINKS B — press on the bubble + `o`: the OPEN IN BROWSER card floats both verified targets =====")
	fmt.Println(out.frameCard)
	fmt.Println("===== UI SHOT =====")
	cardStripped := ansi.Strip(out.frameCard)
	for _, want := range []string{"OPEN IN BROWSER", "opencode.ai/docs", "checker-8x8.png", "enter: open"} {
		if !strings.Contains(cardStripped, want) {
			return fail("links B: the card paints %q:\n%s", want, cardStripped)
		}
	}

	fmt.Println("===== UI SHOT · LINKS C — enter: the runner fired ONCE (the URL), the card is gone, the activity tab logged the open =====")
	fmt.Println(out.frameOpened)
	fmt.Println("===== UI SHOT =====")
	if len(out.opened) != 1 || out.opened[0].Kind != panels.LinkURL || out.opened[0].Value != "https://opencode.ai/docs" {
		return fail("links C: the runner captured EXACTLY the appear-order-first URL target, got %+v", out.opened)
	}
	if !strings.Contains(out.activityOpen, "→ opened: opencode.ai/docs") {
		return fail("links C: the activity tab logs \"→ opened: opencode.ai/docs\", got %q", out.activityOpen)
	}
	if strings.Contains(ansi.Strip(out.frameOpened), "OPEN IN BROWSER") {
		return fail("links C: the card closed on the pick landing")
	}
	// the media target stayed UNOPENED (the member picked one of two);
	// the verified file path is exactly what the resolver WOULD have handed
	// over — proves the fixture is the second target, not a dropped token.
	targets := panels.ExtractTargets("Spec: https://opencode.ai/docs — the diagram sits in the repo.")
	if len(targets) != 1 {
		return fail("links C: the body scan yields the URL alone (media rides the carrier): %+v", targets)
	}
	if !filepath.IsAbs(fixtureAbs) {
		return fail("links C: the fixture resolves absolute")
	}

	// the no-mark leg: `o` with NO selection types into the draft (claim
	// refusal), and the runner captured NOTHING new.
	if len(out.opened) != 1 {
		return fail("links D: the no-mark leg opened nothing new, got %+v", out.opened)
	}
	noMark := ansi.Strip(out.frameNoMark)
	if !strings.Contains(noMark, "› o") {
		return fail("links D: the unclaimed `o` typed into the draft:\n%s", noMark)
	}

	// determinism: the same synchronous drive, byte-for-byte.
	out2, err := linksDrive()
	if err != nil {
		return err
	}
	if out.frameBeacon != out2.frameBeacon || out.frameCard != out2.frameCard ||
		out.frameOpened != out2.frameOpened || out.activityOpen != out2.activityOpen ||
		out.frameNoMark != out2.frameNoMark {
		return fail("links: two synchronous drives must produce byte-identical frames + verdicts")
	}
	fmt.Println("asserts: OK — the boss bubble wore the · o (open) beacon (extraction gate = os.Stat-verified paths + schemed URLs only); a press marked the bubble, `o` floated the OPEN IN BROWSER card with BOTH verified targets (https://opencode.ai/docs + internal/panels/testdata/checker-8x8.png), enter fired the appear-order-first URL through the STUBBED runner; activity logged \"→ opened: opencode.ai/docs\"; the no-mark leg typed \"o\" into the draft (claim refusal); two drives byte-identical")
	return nil
}

// --- terminal-browser candidate lane (--openurl) ----------------------------
// The `o` open's OPTIONAL browser lane, RESOLVER + REAL-BINARY proof (the
// pure matrix + frame contracts live in internal/app's open_url_test.go):
// a scratch fixture dir carries POSIX FAKES — `terminal-browser` (logs
// "terminal-browser pwd=… args=…" to a capture file, exits $FAKE_TB_EXIT)
// and `open`/`xdg-open` (the system-opener capture, exit 0). PATH pins
// "<fixture>:<orig>" (fixture FIRST shadows any real binary on the host)
// and the terminal env is hermetically stubbed (TERM_PROGRAM=ghostty, TMUX
// + KITTY_WINDOW_ID + the kill-switch cleared — the host's real env can
// never leak a lane in). Leg A: the resolver prefers terminal-browser
// ("resolve=terminal-browser prefer-over-system-open" printed), a press +
// `o` on a single-URL bubble fires the fake straight (no card — ONE
// target), tb.log captured exactly "open https://opencode.ai/docs", the
// system log stayed ABSENT, the activity tab logged "→ opened:", and NO
// "could not open" row rendered. Leg B (FAKE_TB_EXIT=1): the SAME URL
// cascades to the system opener — tb.log holds exactly ONE attempt (never
// a retry), system.log captured the URL, the verdict STILL reads "→
// opened:". Every leg's frames + logs byte-identical across two drives.

// openURLEnvKeys — every env the fixture drive touches, snapshotted +
// restored per drive (a drive pair shares the process).
var openURLEnvKeys = []string{
	"PATH", "TERM_PROGRAM", "TMUX", "KITTY_WINDOW_ID",
	panels.TerminalBrowserOffEnv, "FAKE_TB_EXIT", "FAKE_OPEN_LOG_DIR",
}

// openURLFrameOut — ONE drive's observed artifacts.
type openURLFrameOut struct {
	resolved     string   // the resolver's pick BEFORE the press
	frameOpened  string   // after `o`: the open landed
	activityOpen string   // the "→ opened:" activity row
	tbLog        []string // the fake terminal-browser's captured calls
	sysLog       []string // the system-opener fakes' captured calls
}

// openURLDrive — ONE hermetic drive at the fake's exit code.
func openURLDrive(tbExit string) (openURLFrameOut, error) {
	var out openURLFrameOut
	saved, present := map[string]string{}, map[string]bool{}
	for _, k := range openURLEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k], present[k] = v, true
		}
	}
	defer func() { // restore EVERY key (the drive pair shares the process)
		for _, k := range openURLEnvKeys {
			if present[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	root, err := os.MkdirTemp("", "uishot-openurl-")
	if err != nil {
		return out, fmt.Errorf("openurl fixture: %w", err)
	}
	defer os.RemoveAll(root)
	fakes := map[string]string{
		"terminal-browser": "#!/bin/sh\n" +
			"printf 'terminal-browser pwd=%s args=%s\n' \"$(pwd)\" \"$*\" >> \"$FAKE_OPEN_LOG_DIR/tb.log\"\n" +
			"exit \"${FAKE_TB_EXIT:-0}\"\n",
		"open": "#!/bin/sh\n" +
			"printf 'system-open pwd=%s args=%s\n' \"$(pwd)\" \"$*\" >> \"$FAKE_OPEN_LOG_DIR/system.log\"\n" +
			"exit 0\n",
		"xdg-open": "#!/bin/sh\n" +
			"printf 'system-open pwd=%s args=%s\n' \"$(pwd)\" \"$*\" >> \"$FAKE_OPEN_LOG_DIR/system.log\"\n" +
			"exit 0\n",
	}
	for name, body := range fakes {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o755); err != nil {
			return out, fmt.Errorf("openurl fixture %s: %w", name, err)
		}
	}
	os.Setenv("FAKE_OPEN_LOG_DIR", root)
	os.Setenv("FAKE_TB_EXIT", tbExit)
	os.Setenv("TERM_PROGRAM", "ghostty") // the hermetic kitty-capable host stub
	os.Setenv("TMUX", "")                // tmux's default-miss never sneaks in
	os.Setenv("KITTY_WINDOW_ID", "")
	os.Setenv(panels.TerminalBrowserOffEnv, "")
	os.Setenv("PATH", root+string(os.PathListSeparator)+saved["PATH"])

	// the resolver's pick is pinned BEFORE the press (the live env read).
	out.resolved = app.ResolveBrowserTool().String()

	backend := &stubBackend{done: make(chan struct{})}
	m := app.New(backend, config.Default())
	// the exact breadth-first drain runLinksProof runs (heartbeats
	// dropped; every other landing re-feeds — the open's browserOpenMsg
	// verdict lands synchronously inside the `o` press).
	runExec := func(msg tea.Msg) {
		tm, cmd := m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
			default:
				tm2, next := m.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}

	runExec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — terminal-browser lane fixture online"})
	runExec(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user", "where is the spec?", false)})
	// a completed boss bubble carrying exactly ONE URL → the straight-
	// through open (the multi-target card is --links' contract).
	runExec(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m1", From: "boss", Kind: "boss",
		Text: "Spec: https://opencode.ai/docs — read it.", At: 1, Pending: false}})

	// the press: find the body row wearing the URL in the frame and click
	// INSIDE the chat panel there (sidebar 80 on the left; +10 into the
	// panel region — linksDrive's exact geometry).
	frameBeacon := m.Frame()
	row, pressX := -1, (shotCols-80)+10
	for i, ln := range strings.Split(frameBeacon, "\n") {
		if strings.Contains(ln, "opencode.ai/docs") {
			row = i
			break
		}
	}
	if row < 0 {
		return out, fmt.Errorf("openurl: the URL bubble must render pre-press")
	}
	runExec(tea.MouseClickMsg(tea.Mouse{X: pressX, Y: row, Button: tea.MouseLeft}))
	runExec(tea.KeyPressMsg(tea.Key{Code: 'o', Text: "o"}))
	out.frameOpened = m.Frame()
	for _, ln := range m.ActivityLines() {
		if strings.Contains(ln, "→ opened:") {
			out.activityOpen = ln
		}
	}
	readLog := func(name string) []string {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return nil // never written == the leg never ran
		}
		var lines []string
		for _, ln := range strings.Split(string(raw), "\n") {
			if ln != "" {
				lines = append(lines, ln)
			}
		}
		return lines
	}
	out.tbLog, out.sysLog = readLog("tb.log"), readLog("system.log")
	return out, nil
}

// openURLIdentical — the two-drives byte-identity gate.
func openURLIdentical(a, b openURLFrameOut) bool {
	return a.resolved == b.resolved && a.frameOpened == b.frameOpened &&
		a.activityOpen == b.activityOpen &&
		strings.Join(a.tbLog, "\n") == strings.Join(b.tbLog, "\n") &&
		strings.Join(a.sysLog, "\n") == strings.Join(b.sysLog, "\n")
}

// openURLAssertLeg — BOTH legs' shared surface: resolved lane, log pins,
// the activity landing, and the clean transcript.
func openURLAssertLeg(tag string, out openURLFrameOut, wantSysLines int) error {
	if out.resolved != "terminal-browser" {
		return fmt.Errorf("%s: resolve=terminal-browser prefer-over-system-open expected, got %q", tag, out.resolved)
	}
	if len(out.tbLog) != 1 || !strings.HasSuffix(out.tbLog[0], "args=open https://opencode.ai/docs") {
		return fmt.Errorf("%s: EXACTLY ONE terminal-browser attempt logged with the URL (no retry): %v", tag, out.tbLog)
	}
	if len(out.sysLog) != wantSysLines {
		return fmt.Errorf("%s: the system opener captured %d calls, want %d: %v", tag, len(out.sysLog), wantSysLines, out.sysLog)
	}
	for _, ln := range out.sysLog {
		if !strings.HasSuffix(ln, "https://opencode.ai/docs") {
			return fmt.Errorf("%s: the cascaded call carried the SAME URL: %v", tag, out.sysLog)
		}
	}
	if !strings.Contains(out.activityOpen, "→ opened: opencode.ai/docs") {
		return fmt.Errorf("%s: the activity tab logs \"→ opened: opencode.ai/docs\", got %q", tag, out.activityOpen)
	}
	if strings.Contains(ansi.Strip(out.frameOpened), "could not open:") {
		return fmt.Errorf("%s: a cascaded/absorbed failure never paints the dim row", tag)
	}
	return nil
}

func runOpenURLProof() error {
	// leg A — the happy lane: exit-0 candidate, the whole open lands there.
	a1, err := openURLDrive("0")
	if err != nil {
		return err
	}
	a2, err := openURLDrive("0")
	if err != nil {
		return err
	}
	if err := openURLAssertLeg("openurl A", a1, 0); err != nil {
		return err
	}
	if !openURLIdentical(a1, a2) {
		return fmt.Errorf("openurl A: two drives must be byte-identical")
	}
	fmt.Printf("leg A (resolve=terminal-browser prefer-over-system-open): captured %q — system-opener log ABSENT\n", a1.tbLog[0])
	fmt.Println("===== UI SHOT · OPENURL A — press+`o` painted the page IN-terminal via terminal-browser (exit 0) =====")
	fmt.Println(a1.frameOpened)
	fmt.Println("===== UI SHOT =====")

	// leg B — the cascade: an exit-1 candidate hands the SAME URL to the
	// system opener, ONE attempt each, the verdict intact.
	b1, err := openURLDrive("1")
	if err != nil {
		return err
	}
	b2, err := openURLDrive("1")
	if err != nil {
		return err
	}
	if err := openURLAssertLeg("openurl B", b1, 1); err != nil {
		return err
	}
	if !openURLIdentical(b1, b2) {
		return fmt.Errorf("openurl B: two drives must be byte-identical")
	}
	fmt.Printf("leg B (cascade on exit 1): candidate attempt %q → system-opener captured %q\n", b1.tbLog[0], b1.sysLog[0])
	fmt.Println("===== UI SHOT · OPENURL B — the candidate's exit-1 cascaded the SAME URL to the system opener (\"→ opened:\" intact) =====")
	fmt.Println(b1.frameOpened)
	fmt.Println("===== UI SHOT =====")

	fmt.Println("asserts: OK — resolve=terminal-browser prefer-over-system-open on the hermetic ghostty fixture (PATH pinned \"<fixture>:<orig>\", TMUX/KITTY_WINDOW_ID/kill-switch cleared); leg A: press+`o` fired `terminal-browser open https://opencode.ai/docs` EXACTLY once (system log absent, `→ opened: opencode.ai/docs` logged); leg B (exit 1): the SAME URL cascaded to the system opener — ONE attempt per leg, no retry, no \"could not open\" row, the verdict intact; every leg byte-identical across two drives (real app drive, real fake binaries, real exec)")
	return nil
}

// --- browser tab premium lane (--browser --lane kitty) ----------------------
// The browser tab's EMBEDDED zenbu lane, RESOLVE + REAL-CHILD proof (the
// pure matrix + state-machine contracts live in internal/panels'
// browser_lane_test.go): a scratch fixture dir plants a REAL fake
// `terminal-browser` (prints fake kitty-frame bytes — ESC_G APC payload +
// ESC\ — plus a plain marker row, then either stays alive for
// $FAKE_TB_LIFE seconds or, FAKE_TB_LIFE=0, exits 0 IMMEDIATELY) on a
// pinned "<fixture>:<orig>" PATH under the hermetic ghostty
// stub (stubTermEnv's checklist owned; both kill-switch spellings
// cleared). Leg A: the lane resolves zenbu, the child spawns on the REAL
// PTY seam, its bytes paint the embedded grid, the region frame wears the
// " zenbu " badge + the "▸ zenbu terminal-browser · <url>" strip — then
// Close group-kills + reaps (pid gone, no leak). Leg B (FAKE_TB_LIFE=0 —
// the immediate exit measures ≈180ms through the PTY seam, inside the
// 300ms early-exit window): Poll lands the text-mode fallback — the
// EXACT dim note "zenbu exited (0) — falling back to text mode", the
// " text " badge, the text fixture body, the strip GONE, and the URL
// state (current + visited) intact. Leg S (the kitty STREAM
// passthrough — the rendering proof): the fake streams TWO CHUNKED kitty
// frames (m=1/m=1/m=0, cut mid-quad) under the SAME child i=1 (the real
// child's every-repaint id reuse) interleaved with text chrome; the
// lane's splitter extracts every APC (the grid + scrollback keep ONLY
// the chrome; the RegionView frame carries ZERO APC bytes), the frame
// wrapper re-emits BOTH generations to the OUTER terminal after renderer
// flushes as cursor-save + CUP(absolute cell) + ONE cached
// a=T,t=d,q=2,C=1 APC under the STABLE office id carrying the pane's
// body box c=/r= + cursor-restore — with ZERO a=d between the
// generations (kitty's atomic same-id replace) — and Close flushes
// ESC_Ga=d,d=I directly through the (captured) emit seam. Leg K (the
// MID-CHAIN DEATH — the leak proof): the fake streams 3 chunks of an
// m=1 chain (chunk 2 with a racy OSC-7 report interleaved INSIDE the
// base64 payload — the wave-82 capture's exact shape) and DIES with
// chunk 3 UNTERMINATED; the grid + scrollback contain ZERO base64 (the
// aborted tail discards to its terminator; Close resets the pending
// chain), and Poll latches the text fallback. Every leg byte-identical
// across two drives (the pid-varying reap line is printed, never
// compared).

// browserEnvKeys — every env the lane drive touches, snapshotted +
// restored per drive (the leg pairs share the process).
var browserEnvKeys = []string{
	"PATH", "TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM", "TERM_PROGRAM_VERSION",
	"WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID", "TERM", "COLORTERM",
	panels.BrowserLaneOffEnv, panels.TerminalBrowserOffEnv, "FAKE_TB_LIFE",
	"THEBORINGOFFICE_HOME", "THEBORINGOFFICE_CELL_PX", "THEBORINGOFFICE_ZENBU_LANE",
}

// pinShotEngineAbsent — NO live chrome in the text-lane browser drives:
// the shot lane's verdict is chrome-missing (deterministic on every
// host, cheap — the probe gates before any render). The --lane shot
// drives swap their OWN fake engines instead.
func pinShotEngineAbsent() func() {
	return panels.SetHeadlessForShot(func() (string, bool) { return "", false }, nil)
}

// browserLanePage — the drive's fixed page (stable, never fetched).
const browserLanePage = "https://example.test/docs"

// browserLaneTextFixture — the text-mode body the region paints on the
// fallback leg (Dev A's viewer's stand-in behind the RegionView seam;
// rows sized to the proof pane's 64 cols).
var browserLaneTextFixture = []string{
	"# theboringoffice — text fixture",
	"",
	"text-mode body — the SAME url renders",
	"while zenbu is absent, dead, or off.",
	"[1] " + browserLanePage,
}

// browserFrameOut — ONE drive's observed artifacts.
type browserFrameOut struct {
	resolved   string   // the boot-memoized lane resolve
	frame      string   // the RegionView frame at the assert moment
	note       string   // the controller's fallback note (leg B)
	currentURL string   // the URL state that must survive a fallback
	visited    []string // the open-order history
	reapProof  string   // leg A: the group-kill verdict (pid-varying — printed, never compared)
}

// browserDrive — ONE hermetic drive with the fake alive for tbLife
// seconds ("1000000" = leg A, "0.1" = leg B).
func browserDrive(tbLife string) (browserFrameOut, error) {
	var out browserFrameOut
	saved, present := map[string]string{}, map[string]bool{}
	for _, k := range browserEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k], present[k] = v, true
		}
	}
	defer func() { // restore EVERY key (the drive pairs share the process)
		for _, k := range browserEnvKeys {
			if present[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	root, err := os.MkdirTemp("", "uishot-browser-")
	if err != nil {
		return out, fmt.Errorf("browser fixture: %w", err)
	}
	defer os.RemoveAll(root)
	// FAKE_TB_LIFE "0" = the early-death leg: exit 0 IMMEDIATELY after the
	// prints (measured ≈180ms through the PTY seam — safely inside the
	// 300ms early-exit window; a `sleep 0.1` leg measures ≈285–340ms here
	// because PTY session-leader spawn latency lands ~175ms BEFORE the
	// script's first byte, and would race the window). Any other value:
	// stay alive that long (leg A parks the child ~11 days).
	fake := "#!/bin/sh\n" +
		"printf '\\033_Ga=T,t=d,f=100,i=7e57b001,q=2;RkFLRUtJVFRZRlJBTUU=\\033\\\\\\n'\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"if [ \"${FAKE_TB_LIFE:-1000000}\" = \"0\" ]; then exit 0; fi\n" +
		"exec sleep \"${FAKE_TB_LIFE:-1000000}\"\n"
	if err := os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755); err != nil {
		return out, fmt.Errorf("browser fixture terminal-browser: %w", err)
	}
	os.Setenv("FAKE_TB_LIFE", tbLife)
	os.Setenv("TERM_PROGRAM", "ghostty") // the hermetic kitty-capable host stub
	for _, k := range []string{"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM_VERSION", "WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID"} {
		os.Setenv(k, "")
	}
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv(panels.BrowserLaneOffEnv, "")
	os.Setenv(panels.TerminalBrowserOffEnv, "")
	// the wave-85 opt-in: these drives prove the PREMIUM lane (default-off
	// in production — the headless shot lane is the default premium path).
	os.Setenv("THEBORINGOFFICE_ZENBU_LANE", "1")
	os.Setenv("PATH", root+string(os.PathListSeparator)+saved["PATH"])

	c := panels.NewBrowserLaneController(64, 16)
	out.resolved = c.Lane().String()
	if out.resolved != "zenbu" {
		return out, fmt.Errorf("browser: the hermetic ghostty fixture must resolve the zenbu lane, got %q", out.resolved)
	}
	if err := c.OpenURL(browserLanePage); err != nil {
		return out, fmt.Errorf("browser open: %w", err)
	}
	sess, ok := c.Session().(*panels.ZenbuSession)
	if !ok {
		return out, fmt.Errorf("browser: the premium lane must embed a live *ZenbuSession, got %T", c.Session())
	}
	// the child's bytes must paint the embedded grid (bounded wait — the
	// reader loop is async, the frame carries no timing).
	marker := "zenbu-fake open " + browserLanePage
	painted := false
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline) && !painted; {
		for y := 0; y < sess.Grid().Rows(); y++ {
			if strings.Contains(sess.Grid().LineText(y), marker) {
				painted = true
				break
			}
		}
		if !painted {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if !painted {
		return out, fmt.Errorf("browser: the fake's marker row never painted the embedded grid")
	}
	// ride past the 300ms early-exit window before the Poll; on the
	// early-death leg also wait out the reap itself (cmd.Wait trails the
	// exit by ~10–100ms through the PTY seam — the frame carries no
	// timing, only the landed note).
	if rem := 350*time.Millisecond - sess.Lifetime(); rem > 0 {
		time.Sleep(rem)
	}
	if tbLife == "0" {
		for deadline := time.Now().Add(2 * time.Second); !sess.Exited() && time.Now().Before(deadline); {
			time.Sleep(5 * time.Millisecond)
		}
		if !sess.Exited() {
			return out, fmt.Errorf("browser leg B: the early-death fake never reaped")
		}
	}
	c.Poll()

	if tbLife == "1000000" {
		// leg A — a healthy fake KEEPS the premium lane; Close then proves
		// the lifecycle kill (group SIGKILL + bounded reap, no leak).
		if !c.PremiumActive() {
			return out, fmt.Errorf("browser leg A: a healthy fake must keep the premium lane")
		}
		out.frame = c.RegionView(browserLaneTextFixture)
		pid := sess.Pid()
		c.Close()
		gone := syscall.Kill(pid, 0) != nil
		out.reapProof = fmt.Sprintf("the embedded child was group-killed + reaped: exited=%v process-gone=%v", sess.Exited(), gone)
		if !sess.Exited() || !gone {
			return out, fmt.Errorf("browser leg A: the child must be reaped past Close (%s)", out.reapProof)
		}
	} else {
		// leg B — the immediate death: Poll lands the fallback with the EXACT
		// note; the URL state persists.
		if c.PremiumActive() {
			return out, fmt.Errorf("browser leg B: the early death must drop the premium lane")
		}
		out.note = c.Note()
		if want := fmt.Sprintf(panels.ZenbuFallbackNoteFmt, 0); out.note != want {
			return out, fmt.Errorf("browser leg B: note %q, want %q", out.note, want)
		}
		c.Close()
		out.frame = c.RegionView(browserLaneTextFixture)
	}
	out.currentURL = c.CurrentURL()
	out.visited = c.VisitedURLs()
	return out, nil
}

// browserIdentical — the two-drives byte-identity gate (the pid-varying
// reapProof is EXCLUDED — printed as evidence, never compared).
func browserIdentical(a, b browserFrameOut) bool {
	return a.resolved == b.resolved && a.frame == b.frame && a.note == b.note &&
		a.currentURL == b.currentURL && strings.Join(a.visited, "\n") == strings.Join(b.visited, "\n")
}

// browserAssertPremium — leg A's shared surface: zenbu resolved, the
// badge + strip + the child's painted marker in the frame, no note.
func browserAssertPremium(tag string, out browserFrameOut) error {
	if out.resolved != "zenbu" {
		return fmt.Errorf("%s: lane must resolve zenbu, got %q", tag, out.resolved)
	}
	plain := ansi.Strip(out.frame)
	for _, want := range []string{" zenbu ", "▸ zenbu terminal-browser · " + browserLanePage, "zenbu-fake open " + browserLanePage} {
		if !strings.Contains(plain, want) {
			return fmt.Errorf("%s: the premium frame carries %q:\n%s", tag, want, plain)
		}
	}
	if strings.Contains(plain, "falling back to text mode") {
		return fmt.Errorf("%s: a healthy child never wears the fallback note", tag)
	}
	if out.currentURL != browserLanePage || strings.Join(out.visited, ",") != browserLanePage {
		return fmt.Errorf("%s: URL state = %q visited %v", tag, out.currentURL, out.visited)
	}
	return nil
}

// browserAssertFallback — leg B's shared surface: the exact note, the
// " text " badge + fixture body, the strip GONE, the URL state intact.
func browserAssertFallback(tag string, out browserFrameOut) error {
	if want := fmt.Sprintf(panels.ZenbuFallbackNoteFmt, 0); out.note != want {
		return fmt.Errorf("%s: note %q, want %q", tag, out.note, want)
	}
	plain := ansi.Strip(out.frame)
	for _, want := range []string{" text ", out.note, "# theboringoffice — text fixture", "[1] " + browserLanePage} {
		if !strings.Contains(plain, want) {
			return fmt.Errorf("%s: the fallback frame carries %q:\n%s", tag, want, plain)
		}
	}
	if strings.Contains(plain, "zenbu terminal-browser ·") {
		return fmt.Errorf("%s: the fallback frame drops the premium strip", tag)
	}
	if out.currentURL != browserLanePage || strings.Join(out.visited, ",") != browserLanePage {
		return fmt.Errorf("%s: the fallback preserves the URL state = %q visited %v", tag, out.currentURL, out.visited)
	}
	return nil
}

func runBrowserLaneProof() error {
	// leg A — the healthy embed: resolve zenbu, paint the strip, then the
	// office-shutdown kill reaps the child.
	a1, err := browserDrive("1000000")
	if err != nil {
		return err
	}
	a2, err := browserDrive("1000000")
	if err != nil {
		return err
	}
	if err := browserAssertPremium("browser A", a1); err != nil {
		return err
	}
	if !browserIdentical(a1, a2) {
		return fmt.Errorf("browser A: two drives must be byte-identical")
	}
	fmt.Printf("leg A (premium embed healthy): %s\n", a1.reapProof)
	fmt.Println("===== UI SHOT · BROWSER A — zenbu terminal-browser embedded (badge + strip + the child's painted frame) =====")
	fmt.Println(a1.frame)
	fmt.Println("===== UI SHOT =====")

	// leg B — the immediate early death (<300ms): the text-mode fallback with the dim
	// note, the URL state intact.
	b1, err := browserDrive("0")
	if err != nil {
		return err
	}
	b2, err := browserDrive("0")
	if err != nil {
		return err
	}
	if err := browserAssertFallback("browser B", b1); err != nil {
		return err
	}
	if !browserIdentical(b1, b2) {
		return fmt.Errorf("browser B: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER B — zenbu died at ~180ms (<300ms) — text-mode fallback, dim note, URL state intact =====")
	fmt.Println(b1.frame)
	fmt.Println("===== UI SHOT =====")

	fmt.Println("asserts: OK — lane resolved zenbu under the hermetic ghostty stub (PATH pinned \"<fixture>:<orig>\", TMUX/KITTY_WINDOW_ID/both kill-switch spellings cleared); leg A: the REAL fake child painted the embedded grid (\" zenbu \" badge + \"▸ zenbu terminal-browser · " + browserLanePage + "\" strip), Close group-killed + reaped (pid gone, no leak); leg B (dead immediately, ~180ms < 300ms): the text lane latched with the exact dim note \"zenbu exited (0) — falling back to text mode\", the fixture body painted, the strip dropped, current+visited URL state intact; every leg byte-identical across two drives")

	// leg S — the kitty STREAM passthrough: the fake child paints a
	// chunked kitty frame + text chrome; the lane splits the stream (text
	// only in the grid; the View carries ZERO APC bytes), and the FRAME
	// WRAPPER re-emits the image to the OUTER terminal after a renderer
	// flush (cursor-save + CUP + the cached APC + cursor-restore).
	s1, err := browserStreamDrive()
	if err != nil {
		return err
	}
	s2, err := browserStreamDrive()
	if err != nil {
		return err
	}
	if err := browserAssertStream("browser S", s1); err != nil {
		return err
	}
	if !browserStreamIdentical(s1, s2) {
		return fmt.Errorf("browser S: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER S — the kitty stream split: pure-text View, the frame wrapper emits BOTH generations after the flush =====")
	fmt.Println(s1.frame)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("asserts: OK — the fake child streamed TWO CHUNKED kitty frames (m=1/m=1/m=0, the first chunk cut mid-quad) under the SAME child id i=1 (the real child's every-repaint id reuse) interleaved with ANSI text chrome; the lane's splitter extracted every APC byte (the grid + the retained scrollback paint ONLY the chrome — zero base64 glyphs in any text row, ZERO APC bytes in the View — the renderer would eat them), the image store re-ided BOTH generations to the STABLE office id i=" + browserStreamIDHash8 + " (ZenbuOfficeID(child id, placement) — never the payload, never the child's i=1), and the FRAME WRAPPER re-emitted each generation after a renderer flush as cursor-save + CUP(the absolute cell) + ONE cached a=T,t=d,q=2,C=1 APC carrying the pane's body box c=64,r=14 + cursor-restore — with ZERO a=d between the generations (kitty's same-id atomic replace: no flicker, no empty gap); the registry clear flushed exactly one `ESC_Ga=d,d=I,i=" + browserStreamIDHash8 + ",q=2;ESC\\`, and Close flushed it DIRECTLY through the emit seam (captured: " + fmt.Sprintf("%d", len(s1.emitted)) + " delete frame(s)) — the terminal never keeps a stale image; every leg byte-identical across two drives")

	// leg K — the MID-CHAIN DEATH: the fake streams 3 chunks of an m=1
	// chain (chunk 2 carrying the REAL child's OSC-7 interleave mid-
	// payload, the wave-82 capture's exact shape) and DIES with chunk 3
	// UNTERMINATED — the grid + scrollback must contain ZERO base64.
	k1, err := browserKillDrive()
	if err != nil {
		return err
	}
	k2, err := browserKillDrive()
	if err != nil {
		return err
	}
	if err := browserAssertKill("browser K", k1); err != nil {
		return err
	}
	if !browserKillIdentical(k1, k2) {
		return fmt.Errorf("browser K: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER K — the mid-chain death: the split tail + the OSC-7 interleave leak ZERO base64 =====")
	fmt.Println(k1.frame)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("asserts: OK — the fake child died MID-CHUNKED-FRAME (3 chunks of an m=1 chain: chunk 2 with a racy OSC-7 report interleaved INSIDE the base64 payload — the wave-82 capture's exact shape — chunk 3 UNTERMINATED at the PTY's EOF); the splitter aborted the corrupt command, DISCARDED the tail to its terminator, and the session Close reset the pending chain: the grid + the retained scrollback contain ZERO base64 runs (" + fmt.Sprintf("%d", k1.gridLeak) + " grid / " + fmt.Sprintf("%d", k1.sbLeak) + " scrollback), the chrome painted before the chain survived, and Poll latched the text fallback with the exact dim note \"" + k1.note + "\"; every leg byte-identical across two drives")
	return nil
}

// --- browser tab kitty STREAM passthrough (--browser --lane kitty, leg S) -----
// The premium lane's RENDERING proof (the byte-level contracts live in
// internal/panels' browser_lane_kitty_test.go + zenbu_frame_test.go):
// the fake `terminal-browser` streams the REAL child's wire shape
// (ground-truthed against v0.6.0's native engine) at fixture scale —
// cursor home, a toolbar text row, then TWO full-window frames as
// CHUNKED kitty APCs (ESC_G a=T,…,m=1;<b64 chunk> ESC\ × 2 + m=0, the
// first chunk cut at a NON-4-aligned base64 boundary to prove the join)
// under the SAME child i=1 (the real child reuses it for EVERY repaint —
// kitty's atomic same-id replace), a marker row between the generations.
// Asserts (the wave-82 fixes): the frame's text rows (ANSI-stripped)
// carry ONLY the chrome (badge + strip + toolbar + markers — no base64
// glyph anywhere) and the raw frame carries ZERO APC bytes (bubbletea's
// cell renderer eats zero-width sequences — the image NEVER rides the
// View); the controller's registry contribution through the
// tea.WithOutput frame wrapper appends, after a renderer flush,
// cursor-save + CUP(the absolute cell) + ONE cached a=T,t=d,q=2,C=1 APC
// under the STABLE office id (ZenbuOfficeID(child id, placement)) +
// cursor-restore; EVERY emitted frame carries the pane's body box
// c=64,r=14 (FIX B); the gen-A → gen-B splice carries ZERO a=d (FIX A —
// no flicker, no empty gap); a cleared registry flushes exactly one
// ESC_Ga=d,d=I; and Close flushes the same delete DIRECTLY through the
// (captured) emit seam (panels.SetZenbuEmitForShot). Every leg
// byte-identical across two drives.

// browserStreamPayloadA/B — the drive's TWO deterministic fake frames
// (content irrelevant — the office never image-decodes them; the bytes
// only round-trip base64). TWO generations, because FIX A's proof needs
// the child's repaint pattern: EVERY frame reuses child i=1 (kitty's
// atomic same-id replace — the anti-flicker mechanism).
var browserStreamPayloadA = []byte("\x89PNG\r\n\x1a\nFAKEKITTYFRAME0123456789abcdefghijklmnopqrstuvwxyz")
var browserStreamPayloadB = []byte("\x89PNG\r\n\x1a\nFAKEKITTYFRAME-GEN-B-9876543210zyxwvutsrqponmlkjihgfedcba")

// browserStreamB64A/B — the payloads' base64 (the fake script chunks it).
var browserStreamB64A = base64.StdEncoding.EncodeToString(browserStreamPayloadA)
var browserStreamB64B = base64.StdEncoding.EncodeToString(browserStreamPayloadB)

// browserStreamIDHash8 — the STABLE office-side id (ZenbuOfficeID over
// child i=1 + placement 0 — NEVER the payload: the wave-81 content hash
// re-ided every repaint, the flicker bug). BOTH generations share it.
var browserStreamIDHash8 = panels.KittyIDHash8(panels.ZenbuOfficeID(1, 0))

// browserStreamEmit — one generation's office re-emission: the control
// head (a=T,t=d,q=2,C=1 + the stable id + f=100 + THE PANE'S BODY BOX
// c=64,r=14 — FIX B; the controller drives at 64x16, bodyH = 16-2) +
// the payload + ESC\ verbatim.
func browserStreamEmit(b64 string) string {
	return "\x1b_Ga=T,t=d,q=2,C=1,i=" + browserStreamIDHash8 + ",f=100,c=64,r=14;" + b64 + "\x1b\\"
}

// browserStreamDeleteFrame — Close's direct-flush delete.
var browserStreamDeleteFrame = "\x1b_Ga=d,d=I,i=" + browserStreamIDHash8 + ",q=2;\x1b\\"

// browserStreamOut — ONE stream drive's observed artifacts.
type browserStreamOut struct {
	frame    string   // the RegionView frame at the assert moment (gen B live)
	emitted  []string // the direct-emit seam's captured writes (the Close deletes)
	wrap1    string   // the wrapper's flush over the gen-A registry
	wrap2    string   // the wrapper's flush over the gen-B registry (SAME id, ZERO a=d)
	wrap3    string   // the registry-clear flush (exactly one a=d)
	deletesN int      // the store's drained deletes between gen A → gen B (MUST be 0)
}

// browserStreamFake — the scripted child: home, toolbar, ONE chunked
// frame (gen A), a marker, a SECOND chunked frame (gen B — SAME child
// i=1, the real child's every-repaint id reuse), the marker, park.
func browserStreamFake(root string) error {
	chunked := func(b64 string) string {
		return "printf '\\033_Ga=T,t=d,f=100,i=1,q=2,m=1;" + b64[:7] + "\\033\\\\'\n" +
			"printf '\\033_Gm=1;" + b64[7:41] + "\\033\\\\'\n" +
			"printf '\\033_Gm=0;" + b64[41:] + "\\033\\\\'\n"
	}
	fake := "#!/bin/sh\n" +
		"printf '\\033[2J\\033[H'\n" +
		"printf 'TB-TOOLBAR\\r\\n'\n" +
		chunked(browserStreamB64A) +
		"printf 'GEN-A-PAINTED\\r\\n'\n" +
		"sleep 0.5\n" + // pace the generations: the drive observes gen A BEFORE gen B streams
		chunked(browserStreamB64B) +
		"printf '\\033[4;1H'\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"exec sleep 1000000\n"
	return os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755)
}

// browserStreamDrive — ONE hermetic stream drive (browserDrive's env
// discipline, the emit seam captured per drive).
func browserStreamDrive() (browserStreamOut, error) {
	var out browserStreamOut
	saved, present := map[string]string{}, map[string]bool{}
	for _, k := range browserEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k], present[k] = v, true
		}
	}
	defer func() {
		for _, k := range browserEnvKeys {
			if present[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	root, err := os.MkdirTemp("", "uishot-browser-stream-")
	if err != nil {
		return out, fmt.Errorf("browser-stream fixture: %w", err)
	}
	defer os.RemoveAll(root)
	if err := browserStreamFake(root); err != nil {
		return out, fmt.Errorf("browser-stream fixture terminal-browser: %w", err)
	}
	os.Setenv("TERM_PROGRAM", "ghostty") // the hermetic kitty-capable host stub
	for _, k := range []string{"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM_VERSION", "WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID"} {
		os.Setenv(k, "")
	}
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv(panels.BrowserLaneOffEnv, "")
	os.Setenv(panels.TerminalBrowserOffEnv, "")
	// the wave-85 opt-in: these drives prove the PREMIUM lane (default-off
	// in production — the headless shot lane is the default premium path).
	os.Setenv("THEBORINGOFFICE_ZENBU_LANE", "1")
	os.Setenv("PATH", root+string(os.PathListSeparator)+saved["PATH"])

	restoreEmit := panels.SetZenbuEmitForShot(func(s string) { out.emitted = append(out.emitted, s) })
	defer restoreEmit()

	c := panels.NewBrowserLaneController(64, 16)
	if got := c.Lane().String(); got != "zenbu" {
		return out, fmt.Errorf("browser-stream: the hermetic ghostty fixture must resolve the zenbu lane, got %q", got)
	}
	if err := c.OpenURL(browserLanePage); err != nil {
		return out, fmt.Errorf("browser-stream open: %w", err)
	}
	sess, ok := c.Session().(*panels.ZenbuSession)
	if !ok {
		return out, fmt.Errorf("browser-stream: the premium lane must embed a live *ZenbuSession, got %T", c.Session())
	}
	// the marker comes AFTER its frame's APCs in the script: a visible
	// marker guarantees that generation's chunked transmission committed.
	waitMarker := func(marker string) error {
		for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
			for y := 0; y < sess.Grid().Rows(); y++ {
				if strings.Contains(sess.Grid().LineText(y), marker) {
					return nil
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		return fmt.Errorf("browser-stream: the fake's %q never painted the embedded grid", marker)
	}
	if err := waitMarker("GEN-A-PAINTED"); err != nil {
		return out, err
	}
	// GEN A committed at pane-local (0,1) (home → toolbar row 0 → \r\n).
	imgsA, delsA := c.FrameState()
	if err := waitMarker("zenbu-fake open " + browserLanePage); err != nil {
		return out, err
	}
	// GEN B committed at pane-local (0,2) (the GEN-A marker row pushed
	// the cursor down one) — and its same-(child id, placement) replace
	// must have queued ZERO deletes (FIX A: kitty replaces atomically
	// terminal-side under the SAME office id — no flicker, no empty gap).
	imgsB, delsB := c.FrameState()
	out.deletesN = len(delsA) + len(delsB)
	out.frame = c.RegionView(browserLaneTextFixture)
	// the WRAPPER-level proof (zenbu_frame.go — the production emission
	// seam the View string can never be): each generation's registry
	// contribution publishes at the desktop origin (0,3); one renderer
	// flush through the tea.WithOutput wrapper appends cursor-save +
	// CUP(the absolute cell) + the cached verbatim APC (the STABLE id +
	// the pane body box c=64,r=14) + cursor-restore — and the gen-A →
	// gen-B flush carries ZERO a=d (the SAME id re-emits); a cleared
	// registry then flushes exactly one a=d. A FRESH registry keeps the
	// proof isolated from the app drives.
	reg := &panels.ZenbuFrameRegistry{}
	var buf strings.Builder
	w := panels.NewZenbuFrameWriter(&buf, reg)
	reg.Publish(true, 0, 3, imgsA, nil)
	_, _ = w.Write([]byte("FLUSH-A"))
	out.wrap1 = buf.String()
	buf.Reset()
	reg.Publish(true, 0, 3, imgsB, nil)
	_, _ = w.Write([]byte("FLUSH-B"))
	out.wrap2 = buf.String()
	buf.Reset()
	reg.Clear()
	_, _ = w.Write([]byte("F3"))
	out.wrap3 = buf.String()
	c.Close() // the deletes flush through the captured emit seam
	return out, nil
}

// browserStreamIdentical — the two-drives byte-identity gate (frames +
// the wrapper bytes + the delete counts + the captured delete flushes).
func browserStreamIdentical(a, b browserStreamOut) bool {
	return a.frame == b.frame && a.wrap1 == b.wrap1 && a.wrap2 == b.wrap2 && a.wrap3 == b.wrap3 &&
		a.deletesN == b.deletesN &&
		strings.Join(a.emitted, "\x00") == strings.Join(b.emitted, "\x00")
}

// browserAssertStream — the stream leg's shared surface.
func browserAssertStream(tag string, out browserStreamOut) error {
	plain := ansi.Strip(out.frame)
	for _, want := range []string{" zenbu ", "▸ zenbu terminal-browser · " + browserLanePage, "TB-TOOLBAR", "GEN-A-PAINTED", "zenbu-fake open " + browserLanePage} {
		if !strings.Contains(plain, want) {
			return fmt.Errorf("%s: the premium frame carries %q:\n%s", tag, want, plain)
		}
	}
	for _, b64 := range []string{browserStreamB64A, browserStreamB64B} {
		if strings.Contains(plain, b64[:16]) {
			return fmt.Errorf("%s: base64 glyphs must never reach a text row:\n%s", tag, plain)
		}
	}
	// the View path is PURE TEXT (the renderer eats zero-width sequences —
	// the wave-80 in-View splice never painted; it only bloated frames).
	if strings.Contains(out.frame, "\x1b_G") {
		return fmt.Errorf("%s: the RegionView frame must carry ZERO APC bytes (the wrapper emits):\n%q", tag, out.frame)
	}
	// (a) the emitted frame keys carry the pane's body box c=64,r=14 —
	// byte-pinned in the full splices below.
	// (b) TWO generations, ONE id, ZERO interleaved deletes: the gen-A →
	// gen-B replace queued NOTHING (kitty's a=T replaces atomically
	// terminal-side under the SAME office id — the anti-flicker fix).
	if out.deletesN != 0 {
		return fmt.Errorf("%s: same-(child id, placement) replace must queue ZERO deletes, got %d", tag, out.deletesN)
	}
	// gen A: flush → cursor-save + CUP(5;1) (origin (0,3) + the pane-local
	// commit (0,1), 1-based) + the cached verbatim APC (stable id + the
	// pane box) + cursor-restore.
	if want := "FLUSH-A" + "\x1b7\x1b[5;1H" + browserStreamEmit(browserStreamB64A) + "\x1b8"; out.wrap1 != want {
		return fmt.Errorf("%s: the gen-A splice bytes:\n got %q\nwant %q", tag, out.wrap1, want)
	}
	// gen B: SAME id, commit (0,2) → CUP(6;1) — and NEVER an a=d.
	if want := "FLUSH-B" + "\x1b7\x1b[6;1H" + browserStreamEmit(browserStreamB64B) + "\x1b8"; out.wrap2 != want {
		return fmt.Errorf("%s: the gen-B splice bytes:\n got %q\nwant %q", tag, out.wrap2, want)
	}
	for _, wrap := range []string{out.wrap1, out.wrap2} {
		if strings.Contains(wrap, ",i=1,") {
			return fmt.Errorf("%s: the child's id i=1 must never be re-emitted", tag)
		}
		if strings.Contains(wrap, "\x1b_Ga=d,") {
			return fmt.Errorf("%s: NO a=d may interleave two same-id generations:\n%q", tag, wrap)
		}
		if !strings.Contains(wrap, ",c=64,r=14;") {
			return fmt.Errorf("%s: every emitted frame carries the pane's body box c=64,r=14:\n%q", tag, wrap)
		}
	}
	// the registry clear flushes exactly one a=d for the emitted id.
	if want := "F3" + browserStreamDeleteFrame; out.wrap3 != want {
		return fmt.Errorf("%s: the registry clear flushes %q:\n got %q\nwant %q", tag, browserStreamDeleteFrame, out.wrap3, want)
	}
	// Close: the lane-lifecycle delete rides the direct seam.
	if len(out.emitted) == 0 || !strings.Contains(strings.Join(out.emitted, ""), browserStreamDeleteFrame) {
		return fmt.Errorf("%s: Close must flush %q through the emit seam, got %q", tag, browserStreamDeleteFrame, out.emitted)
	}
	return nil
}

// --- browser tab MID-CHAIN DEATH (--browser --lane kitty, leg K) --------------
// FIX C's harness proof (the byte-level suite lives in internal/panels'
// browser_lane_kitty_test.go): a REAL fake `terminal-browser` on the
// pinned PATH streams chrome + 3 chunks of an m=1 kitty chain — chunk 2
// with a racy OSC-7 cwd report INTERLEAVED inside the base64 payload
// (terminal-browser v0.6.0's exact shape: 4 interleaves in the 26MB
// wave-82 capture), chunk 3 UNTERMINATED — and exits 1 (the PTY's EOF
// lands mid-chain, the session-churn shape). The grid text rows AND the
// retained scrollback must contain ZERO base64 runs (the pre-fix pane
// painted ~30 dense rows of it), the chrome painted before the chain
// survives, and Poll latches the text fallback with the exact dim note.
// Byte-identical across two drives.

// browserKillFake — the dying-mid-chain scripted child.
func browserKillFake(root string) error {
	b64 := browserStreamB64A
	fake := "#!/bin/sh\n" +
		"printf '\\033[2J\\033[H'\n" +
		"printf 'TB-TOOLBAR\\r\\n'\n" +
		"printf 'mid-chain death next\\r\\n'\n" +
		"printf '\\033_Ga=T,t=d,f=100,i=1,q=2,m=1;" + b64[:20] + "\\033\\\\'\n" +
		"printf '\\033_Gm=1;" + b64[20:36] + "\\033]7;file://host/tmp/x\\a" + b64[36:52] + "\\033\\\\'\n" +
		"printf '\\033_Gm=1;" + b64[52:] + "'\n" + // UNTERMINATED — the death lands mid-chunk
		"exit 1\n"
	return os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755)
}

// browserKillOut — ONE kill drive's observed artifacts.
type browserKillOut struct {
	frame    string // the fallback frame after Poll
	note     string // the latched fallback note
	gridLeak int    // base64 runs in the grid text rows (MUST be 0)
	sbLeak   int    // base64 runs in the retained scrollback (MUST be 0)
	chrome   string // grid row 0 (the toolbar, painted before the chain)
}

// browserKillB64Runs — the leak signature: base64-ish runs ≥ 40 chars.
func browserKillB64Runs(s string) int {
	n, run := 0, 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '+' || c == '/' || c == '=' {
			run++
			if run == 40 {
				n++
			}
		} else {
			run = 0
		}
	}
	return n
}

// browserKillDrive — ONE hermetic mid-chain-death drive.
func browserKillDrive() (browserKillOut, error) {
	var out browserKillOut
	saved, present := map[string]string{}, map[string]bool{}
	for _, k := range browserEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k], present[k] = v, true
		}
	}
	defer func() {
		for _, k := range browserEnvKeys {
			if present[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	root, err := os.MkdirTemp("", "uishot-browser-kill-")
	if err != nil {
		return out, fmt.Errorf("browser-kill fixture: %w", err)
	}
	defer os.RemoveAll(root)
	if err := browserKillFake(root); err != nil {
		return out, fmt.Errorf("browser-kill fixture terminal-browser: %w", err)
	}
	os.Setenv("TERM_PROGRAM", "ghostty")
	for _, k := range []string{"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM_VERSION", "WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID"} {
		os.Setenv(k, "")
	}
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv(panels.BrowserLaneOffEnv, "")
	os.Setenv(panels.TerminalBrowserOffEnv, "")
	// the wave-85 opt-in: these drives prove the PREMIUM lane (default-off
	// in production — the headless shot lane is the default premium path).
	os.Setenv("THEBORINGOFFICE_ZENBU_LANE", "1")
	os.Setenv("PATH", root+string(os.PathListSeparator)+saved["PATH"])

	restoreEmit := panels.SetZenbuEmitForShot(func(string) {})
	defer restoreEmit()

	c := panels.NewBrowserLaneController(64, 16)
	if err := c.OpenURL(browserLanePage); err != nil {
		return out, fmt.Errorf("browser-kill open: %w", err)
	}
	sess, ok := c.Session().(*panels.ZenbuSession)
	if !ok {
		return out, fmt.Errorf("browser-kill: the premium lane must embed a live *ZenbuSession, got %T", c.Session())
	}
	// the marker is printed BEFORE the chain — a visible marker guarantees
	// the chrome painted; then wait the reap (the fake exits 1).
	marker := "mid-chain death next"
	painted := false
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline) && !painted; {
		for y := 0; y < sess.Grid().Rows(); y++ {
			if strings.Contains(sess.Grid().LineText(y), marker) {
				painted = true
				break
			}
		}
		if !painted {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if !painted {
		return out, fmt.Errorf("browser-kill: the fake's marker row never painted the embedded grid")
	}
	for deadline := time.Now().Add(3 * time.Second); !sess.Exited() && time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
	}
	if !sess.Exited() {
		return out, fmt.Errorf("browser-kill: the dying fake never reaped")
	}
	// THE pin: zero base64 in the grid text rows AND the retained
	// scrollback (the split tail + the OSC-7 interleave leak nothing).
	for y := 0; y < sess.Grid().Rows(); y++ {
		out.gridLeak += browserKillB64Runs(sess.Grid().LineText(y))
	}
	out.sbLeak = browserKillB64Runs(string(sess.Scrollback().Raw()))
	out.chrome = sess.Grid().LineText(0)
	c.Poll()
	out.note = c.Note()
	out.frame = c.RegionView(browserLaneTextFixture)
	c.Close()
	return out, nil
}

// browserKillIdentical — the two-drives gate.
func browserKillIdentical(a, b browserKillOut) bool {
	return a.frame == b.frame && a.note == b.note && a.gridLeak == b.gridLeak && a.sbLeak == b.sbLeak && a.chrome == b.chrome
}

// browserAssertKill — the kill leg's shared surface.
func browserAssertKill(tag string, out browserKillOut) error {
	if out.gridLeak != 0 || out.sbLeak != 0 {
		return fmt.Errorf("%s: base64 must NEVER leak downstream (grid %d runs, scrollback %d runs)", tag, out.gridLeak, out.sbLeak)
	}
	if out.chrome != "TB-TOOLBAR" {
		return fmt.Errorf("%s: the chrome painted before the chain survives, got %q", tag, out.chrome)
	}
	if want := fmt.Sprintf(panels.ZenbuFallbackNoteFmt, 1); out.note != want {
		return fmt.Errorf("%s: the exit-1 fallback note = %q, want %q", tag, out.note, want)
	}
	plain := ansi.Strip(out.frame)
	if !strings.Contains(plain, out.note) || !strings.Contains(plain, " text ") {
		return fmt.Errorf("%s: the fallback frame carries the note + the text badge:\n%s", tag, plain)
	}
	return nil
}

// --- browser tab LIVE premium lane (--browser --lane live) -------------------
// The LIVE WIRING proof (legs A/B proved the controller; THIS proves the
// pane + app glue the member actually rides): a REAL fake
// `terminal-browser` on a pinned "<fixture>:<orig>" PATH under the
// hermetic ghostty stub, and "/open file://<fixture>" typed through the
// REAL chat input + slash popover — the pane's Open consults the lane
// controller, the child spawns on the real PTY seam, and the LEFT slot's
// frame wears the " zenbu " badge + the "▸ zenbu terminal-browser ·
// <url>" strip + the child's painted marker (the RIGHT strip unmoved on
// chat). Leg C then presses esc — the pane's leave rides BrowserLeaveMsg,
// the app's SuspendLane FREEZES the child (keep-alive: SIGSTOPped,
// ALIVE behind the floor, PID unchanged, ONE spawn total), the floor
// returns, and the ctrl+c quit path reaps the frozen child (no leak).
// Leg D (the fake exits 1) lands the text fallback THROUGH THE APP: the
// pane's real viewer (warm — the fetch rode under the embed) + the dim
// "zenbu exited (1) — falling back to text mode" note, and a re-open
// never re-spawns (the no-flap latch, read off the fake's call log).
// Every leg byte-identical across two drives (the paint-convergence
// waits poll the app's harness seams — no state events, the digest
// stays frozen).

// browserLiveFrameOut — ONE live drive's observed artifacts.
type browserLiveFrameOut struct {
	frameLive     string // after /open — the premium frame (badge + strip + marker)
	frameFloor    string // after esc — the floor restored, the lane FROZEN (keep-alive)
	frameFell     string // leg D: the text lane + the dim fallback note
	leftTab       int
	activeTab     int
	spawns        int  // the fake's call-log line count (the no-flap evidence)
	pidFloorAlive bool // leg C: the frozen child stays ALIVE behind the floor
	pidQuitGone   bool // leg C: the ctrl+c quit path reaps the frozen child
}

// browserLiveFake — the live proof's fake binary: logs every invocation
// (the spawn count rides the log), prints its marker, then runs the
// flavor tail ("sleep" parks ~11 days; "die" exits 1 — the deterministic
// fallback trigger: a real PTY death measures ~180–500ms here and would
// race the 300ms early-exit window under load; the code-0 early-exit
// class is pinned synthetically in panels' controller suite).
func browserLiveFake(root, flavor string) error {
	tail := "exec sleep 1000000"
	if flavor == "die" {
		tail = "exit 1"
	}
	fake := "#!/bin/sh\n" +
		"echo \"$@\" >> \"" + filepath.Join(root, "calls.log") + "\"\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		tail + "\n"
	return os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755)
}

// browserLiveDrive — ONE hermetic live drive with flavor "sleep" (leg C)
// or "die" (leg D).
func browserLiveDrive(flavor string) (browserLiveFrameOut, error) {
	var out browserLiveFrameOut
	defer pinShotEngineAbsent()() // no live chrome (the stub URL is localhost-allowed)
	saved, present := map[string]string{}, map[string]bool{}
	for _, k := range browserEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k], present[k] = v, true
		}
	}
	defer func() { // restore EVERY key (the drive pairs share the process)
		for _, k := range browserEnvKeys {
			if present[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	root, err := os.MkdirTemp("", "uishot-browser-live-")
	if err != nil {
		return out, fmt.Errorf("browser-live fixture: %w", err)
	}
	defer os.RemoveAll(root)
	if err := browserLiveFake(root, flavor); err != nil {
		return out, fmt.Errorf("browser-live fixture terminal-browser: %w", err)
	}
	os.Setenv("TERM_PROGRAM", "ghostty") // the hermetic kitty-capable host stub
	for _, k := range []string{"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM_VERSION", "WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID"} {
		os.Setenv(k, "")
	}
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv(panels.BrowserLaneOffEnv, "")
	os.Setenv(panels.TerminalBrowserOffEnv, "")
	// the wave-85 opt-in: these drives prove the PREMIUM lane (default-off
	// in production — the headless shot lane is the default premium path).
	os.Setenv("THEBORINGOFFICE_ZENBU_LANE", "1")
	os.Setenv("PATH", root+string(os.PathListSeparator)+saved["PATH"])

	backend := &stubBackend{done: make(chan struct{})}
	m := app.New(backend, config.Default())
	// runExec — the exact breadth-first drain the --browsertab proof runs.
	runExec := func(msg tea.Msg) {
		tm, cmd := m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
				// heartbeats re-arm forever — dropped, exactly as runMsg does
			default:
				tm2, next := m.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}

	runExec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — live browser lane stub online"})

	// /open through the REAL chat input: the bracketed-paste path (cmd+v's
	// own msg — pasteFilePaths refuses the spaced composite, the slash
	// popover opens only on a TYPED "/", so Enter on the pasted draft
	// sends straight to slashMsg → applySlash → applyOpenSlash → the
	// pane's Open → THE live wiring). Paste keeps the drive out of the
	// per-key blink crawl (the --browsertab popover dance costs ~530ms
	// per key; this proof runs four drives).
	fixtureAbs, err := filepath.Abs(browserTabFixtureRel)
	if err != nil {
		return out, fmt.Errorf("browser-live fixture path: %w", err)
	}
	url := "file://" + fixtureAbs
	openViaChat := func() {
		runExec(tea.PasteMsg{Content: "/open " + url})
		runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // sends → slashMsg → applySlash → lane spawn + fetch
	}
	openViaChat()

	if flavor == "die" {
		// leg D — the early death: the pane's poll ride (the explicit
		// harness seam, never the frame cache) observes it, the text lane
		// returns with the dim note; a re-open never re-spawns.
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
			m.BrowserLanePoll()
			if !m.BrowserPremiumActive() {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if m.BrowserPremiumActive() {
			return out, fmt.Errorf("browser-live D: the dead child never dropped")
		}
		out.leftTab = m.LeftTabIndex()
		out.activeTab = m.ActiveTabIndex()
		runExec(state.Event{Kind: state.EvStatus, Text: "live lane fell back"})
		out.frameFell = m.Frame()
		// the no-flap latch: a re-open of the fell-back url stays text.
		openViaChat()
		if m.BrowserPremiumActive() {
			return out, fmt.Errorf("browser-live D: a fell-back url re-spawned")
		}
	} else {
		// leg C — the healthy embed: wait for the child's paint (the grid
		// read seam — no state events, the digest stays frozen), then one
		// status bump re-renders and the frame carries the strip.
		painted := false
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && !painted; {
			if m.BrowserLaneGridHas("zenbu-fake open file:///") {
				painted = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !painted {
			return out, fmt.Errorf("browser-live C: the fake's marker row never painted the embedded grid")
		}
		if !m.BrowserPremiumActive() {
			return out, fmt.Errorf("browser-live C: the premium embed must be live")
		}
		out.leftTab = m.LeftTabIndex()     // captured LIVE — the esc leg below
		out.activeTab = m.ActiveTabIndex() // returns the slot to the floor
		runExec(state.Event{Kind: state.EvStatus, Text: "live lane painting"})
		out.frameLive = m.Frame()
		// esc leaves to the floor AND FREEZES the lane (the keep-alive
		// suspend: the child stays ALIVE behind the floor, its PID
		// unchanged — the page repaints instantly on return); the ctrl+c
		// quit path then reaps the frozen child (never a leak).
		pid := m.BrowserLanePid()
		runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
		if m.BrowserPremiumActive() {
			return out, fmt.Errorf("browser-live C: esc must hide the premium session")
		}
		if !m.BrowserLaneSuspended() {
			return out, fmt.Errorf("browser-live C: esc must FREEZE the premium session (keep-alive)")
		}
		if m.BrowserLanePid() != pid {
			return out, fmt.Errorf("browser-live C: the freeze never respawns (pid %d → %d)", pid, m.BrowserLanePid())
		}
		out.pidFloorAlive = pid > 0 && syscall.Kill(pid, 0) == nil
		out.frameFloor = m.Frame()
		runExec(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl})) // the quit path reaps the frozen child
		out.pidQuitGone = pid > 0 && errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
	}
	if b, err := os.ReadFile(filepath.Join(root, "calls.log")); err == nil {
		out.spawns = strings.Count(strings.TrimSpace(string(b)), "\n") + 1
	}
	return out, nil
}

// browserLiveIdentical — the two-drives byte-identity gate.
func browserLiveIdentical(a, b browserLiveFrameOut) bool {
	return a.frameLive == b.frameLive && a.frameFloor == b.frameFloor && a.frameFell == b.frameFell &&
		a.leftTab == b.leftTab && a.activeTab == b.activeTab && a.spawns == b.spawns &&
		a.pidFloorAlive == b.pidFloorAlive && a.pidQuitGone == b.pidQuitGone
}

func runBrowserLiveProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }

	// leg C — the healthy embed THROUGH THE APP GLUE: the zenbu strip
	// renders inside the LEFT slot; esc closes + returns to the floor.
	c1, err := browserLiveDrive("sleep")
	if err != nil {
		return err
	}
	c2, err := browserLiveDrive("sleep")
	if err != nil {
		return err
	}
	if c1.leftTab != 1 {
		return fail("browser-live C: /open must flip the LEFT slot to the browser (leftTab 1), got %d", c1.leftTab)
	}
	if c1.activeTab != 0 {
		return fail("browser-live C: the RIGHT strip must stay on chat (index 0), got %d", c1.activeTab)
	}
	live := ansi.Strip(c1.frameLive)
	for _, want := range []string{
		" zenbu ",                             // the premium badge
		"▸ zenbu terminal-browser · file:///", // the strip (ansi-clipped at the slot width)
		"zenbu-fake open file:///",            // the child's painted marker row
		"· ctrl+b",                            // the switcher strip rides on top
	} {
		if !strings.Contains(live, want) {
			return fail("browser-live C frame missing %q:\n%s", want, live)
		}
	}
	// the kitty-lane premium frame paints NO text-lane hint row anywhere
	// (the hint's absence pin — the binary-missing leg lives at --lane hint).
	if strings.Contains(live, "text lane —") {
		return fail("browser-live C: a premium host never wears the text-lane hint row:\n%s", live)
	}
	// the strip paints INSIDE the LEFT slot — left of the sidebar's chat.
	stripSeen := false
	for _, line := range strings.Split(live, "\n") {
		if strings.Contains(line, "zenbu terminal-browser") {
			stripSeen = true
			zi, ci := strings.Index(line, "zenbu terminal-browser"), strings.Index(line, "chat")
			if ci >= 0 && zi > ci {
				return fail("browser-live C: the zenbu strip must paint LEFT of the sidebar's chat tab: %q", line)
			}
		}
	}
	if !stripSeen {
		return fail("browser-live C: the zenbu strip never rendered")
	}
	floor := ansi.Strip(c1.frameFloor)
	if strings.Contains(floor, "zenbu terminal-browser") || strings.Contains(floor, " zenbu ") {
		return fail("browser-live C: esc must drop the premium chrome:\n%s", floor)
	}
	if !strings.Contains(floor, "· ctrl+b") {
		return fail("browser-live C: the floor frame keeps the switcher strip:\n%s", floor)
	}
	if c1.spawns != 1 {
		return fail("browser-live C: exactly ONE spawn (esc never re-spawns), log %d", c1.spawns)
	}
	if !c1.pidFloorAlive {
		return fail("browser-live C: the frozen child stays ALIVE behind the floor (keep-alive)")
	}
	if !c1.pidQuitGone {
		return fail("browser-live C: the ctrl+c quit path reaps the frozen child (no leak)")
	}
	if !browserLiveIdentical(c1, c2) {
		return fail("browser-live C: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER LIVE C — /open through the REAL app: the zenbu lane embedded in the LEFT slot =====")
	fmt.Println(c1.frameLive)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("===== UI SHOT · BROWSER LIVE C — after esc: the floor returns, the lane suspended (one spawn total) =====")
	fmt.Println(c1.frameFloor)
	fmt.Println("===== UI SHOT =====")

	// leg D — the non-zero early death THROUGH THE APP GLUE: the text lane
	// returns (the fetch rode under the embed) with the exact dim note;
	// the no-flap latch keeps the re-open text.
	d1, err := browserLiveDrive("die")
	if err != nil {
		return err
	}
	d2, err := browserLiveDrive("die")
	if err != nil {
		return err
	}
	fell := ansi.Strip(d1.frameFell)
	for _, want := range []string{
		"zenbu exited (1) — falling back to text mode", // the EXACT dim note
		"▸ file:///",          // the text location bar is back
		"The Fixture Gazette", // the page was warm — the fetch rode under the embed
	} {
		if !strings.Contains(fell, want) {
			return fail("browser-live D frame missing %q:\n%s", want, fell)
		}
	}
	if strings.Contains(fell, "zenbu terminal-browser ·") {
		return fail("browser-live D: the fallback drops the premium strip:\n%s", fell)
	}
	if d1.spawns != 1 {
		return fail("browser-live D: the no-flap latch keeps the re-open text (one spawn), log %d", d1.spawns)
	}
	if !browserLiveIdentical(d1, d2) {
		return fail("browser-live D: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER LIVE D — the child exited 1: the text lane returns with the dim note (re-open never re-spawns) =====")
	fmt.Println(d1.frameFell)
	fmt.Println("===== UI SHOT =====")

	fmt.Println("asserts: OK — the LIVE wiring (never the controller direct): \"/open file://<fixture>\" through the REAL chat input (bracketed paste → Enter → slashMsg → the pane's Open) on the hermetic ghostty stub (PATH pinned \"<fixture>:<orig>\", both kill-switch spellings cleared); leg C: the pane's Open spawned the real fake child on the PTY seam — the LEFT slot's frame wears the \" zenbu \" badge + the \"▸ zenbu terminal-browser · <url>\" strip + the child's painted marker (strip LEFT of the sidebar's chat tab, the RIGHT strip unmoved, NO text-lane hint row anywhere), then esc rode BrowserLeaveMsg → SuspendLane (the child FROZEN with the flip — keep-alive, ALIVE behind the floor, PID unchanged, ONE spawn total, the floor restored — and the ctrl+c quit path reaped it, no leak); leg D (the fake exits 1): the pane's poll ride landed the text fallback THROUGH THE APP — the fixture page warm underneath (never re-fetched), the exact dim \"zenbu exited (1) — falling back to text mode\" note, and the re-open never re-spawned (the no-flap latch read off the fake's call log); every leg byte-identical across two drives")
	return nil
}

// --- browser tab KEEP-ALIVE lane (--browser --lane keepalive) ----------------
// THE flip-cycle proof (the member's ruling: the page is "always shown" —
// keep-alive over fresh reloads, one backgrounded Electron's RAM while
// suspended): a REAL fake `terminal-browser` on the pinned PATH under the
// hermetic ghostty stub streams ONE chunked kitty frame and parks at
// `exec sleep`; "/open file://<fixture>" rides the REAL chat input, then
// ctrl+b flips the LEFT slot floor-ward and back THROUGH THE APP GLUE
// while the frame-splice wrapper (the production emission seam over the
// shared registry, its DirectEmit captured) records every byte:
//
//	open:        a=T under the STABLE office id (the retained frame)
//	ctrl+b:      the child FREEZES — PID stable + ALIVE + ps state T…
//	             (SIGSTOPped) — and the wrapper's diff flushes ONE a=d
//	ctrl+b back: the SAME pid thaws; the RETAINED frame re-emits
//	             BYTE-IDENTICALLY (the parked fake emits ZERO new bytes —
//	             the a=T is definitionally the store's cached frame,
//	             BEFORE any child output), ZERO a=d between
//	ctrl+b:      a=d again; ctrl+c: the quit path's Close reaps the
//	             frozen child, the delete riding the direct seam
//
// The spawn log stays at ONE across the whole cycle (a respawn would log
// a second line AND re-print the marker). Every leg byte-identical
// across two drives (the pid VALUES vary — only the stability verdicts
// are compared).

// browserKeepaliveFake — the flip cycle's scripted child: log the spawn,
// home, toolbar, ONE chunked kitty frame (browserStreamB64A, m=1/m=0 —
// the commit lands pane-local (0,1)), the marker, park forever.
func browserKeepaliveFake(root string) error {
	b64 := browserStreamB64A
	fake := "#!/bin/sh\n" +
		"echo \"$@\" >> \"" + filepath.Join(root, "calls.log") + "\"\n" +
		"printf '\\033[2J\\033[H'\n" +
		"printf 'TB-TOOLBAR\\r\\n'\n" +
		"printf '\\033_Ga=T,t=d,f=100,i=1,q=2,m=1;" + b64[:7] + "\\033\\\\'\n" +
		"printf '\\033_Gm=0;" + b64[7:] + "\\033\\\\'\n" +
		"printf '\\033[3;1H'\n" +
		"printf 'zenbu-fake open %s\\n' \"$2\"\n" +
		"exec sleep 1000000\n"
	return os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755)
}

// browserKeepaliveOut — ONE keep-alive drive's observed artifacts (the
// pid VALUES are excluded from the byte-pin — the verdicts ride).
type browserKeepaliveOut struct {
	spliceOpen      string // the open flush: a=T under the stable office id
	floorFlush      string // the floor-ward flip: exactly ONE a=d
	spliceResume    string // the return flush: the RETAINED frame, byte-identical
	floorFlush2     string // the second floor-ward flip: the a=d again
	quitEmit        string // the quit path's direct-seam delete
	floorState      string // the frozen child's ps state ("T…" — SIGSTOPped)
	spawns          int    // the fake's call-log line count (MUST be 1)
	pidStable       bool   // ONE pid across the whole cycle
	pidAliveOnFloor bool   // the frozen child stays alive (keep-alive)
	pidGoneAfter    bool   // the quit path reaps (no leak)
}

// browserKeepaliveDrive — ONE hermetic keep-alive drive.
func browserKeepaliveDrive() (browserKeepaliveOut, error) {
	var out browserKeepaliveOut
	defer pinShotEngineAbsent()() // no live chrome
	saved, present := map[string]string{}, map[string]bool{}
	for _, k := range browserEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k], present[k] = v, true
		}
	}
	defer func() { // restore EVERY key (the drive pairs share the process)
		for _, k := range browserEnvKeys {
			if present[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	root, err := os.MkdirTemp("", "uishot-browser-keepalive-")
	if err != nil {
		return out, fmt.Errorf("browser-keepalive fixture: %w", err)
	}
	defer os.RemoveAll(root)
	if err := browserKeepaliveFake(root); err != nil {
		return out, fmt.Errorf("browser-keepalive fixture terminal-browser: %w", err)
	}
	os.Setenv("TERM_PROGRAM", "ghostty") // the hermetic kitty-capable host stub
	for _, k := range []string{"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM_VERSION", "WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID"} {
		os.Setenv(k, "")
	}
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv(panels.BrowserLaneOffEnv, "")
	os.Setenv(panels.TerminalBrowserOffEnv, "")
	// the wave-85 opt-in: these drives prove the PREMIUM lane (default-off
	// in production — the headless shot lane is the default premium path).
	os.Setenv("THEBORINGOFFICE_ZENBU_LANE", "1")
	os.Setenv("PATH", root+string(os.PathListSeparator)+saved["PATH"])

	// the frame-splice wrapper over the SHARED registry (the production
	// seam's exact shape), the lane's direct deletes captured.
	reg := panels.ZenbuRegistry()
	reg.Clear()
	defer reg.Clear()
	var buf strings.Builder
	w := panels.NewZenbuFrameWriter(&buf, reg)
	restoreEmit := panels.SetZenbuEmitForShot(w.DirectEmit)
	defer restoreEmit()

	backend := &stubBackend{done: make(chan struct{})}
	m := app.New(backend, config.Default())
	runExec := func(msg tea.Msg) {
		tm, cmd := m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
				// heartbeats re-arm forever — dropped, exactly as runMsg does
			default:
				tm2, next := m.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}
	flush := func(tag string) string {
		buf.Reset()
		_, _ = w.Write([]byte(tag))
		return buf.String()
	}

	runExec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — keep-alive lane stub online"})
	fixtureAbs, err := filepath.Abs(browserTabFixtureRel)
	if err != nil {
		return out, fmt.Errorf("browser-keepalive fixture path: %w", err)
	}
	url := "file://" + fixtureAbs
	runExec(tea.PasteMsg{Content: "/open " + url})
	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // sends → slashMsg → applyOpenSlash → the pane's Open

	// the child's paint converges (the harness seam, never the frame cache).
	painted := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && !painted; {
		if m.BrowserLaneGridHas("zenbu-fake open file:///") {
			painted = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !painted {
		return out, fmt.Errorf("browser-keepalive: the fake's marker row never painted the embedded grid")
	}
	if !m.BrowserPremiumActive() {
		return out, fmt.Errorf("browser-keepalive: the premium embed must be live after /open")
	}
	pid := m.BrowserLanePid()
	if pid <= 0 {
		return out, fmt.Errorf("browser-keepalive: the child's pid must read through the harness seam, got %d", pid)
	}

	// OPEN: one rendered frame publishes the registry; one flush splices.
	_ = m.Frame()
	out.spliceOpen = flush("OPEN")

	ctrlB := tea.KeyPressMsg(tea.Key{Code: 'b', Mod: tea.ModCtrl})
	// ctrl+b → FLOOR: the freeze (PID stable, alive, SIGSTOPped) + ONE a=d.
	runExec(ctrlB)
	if m.BrowserPremiumActive() || !m.BrowserLaneSuspended() {
		return out, fmt.Errorf("browser-keepalive: the floor flip must FREEZE (active=%v suspended=%v)", m.BrowserPremiumActive(), m.BrowserLaneSuspended())
	}
	out.pidStable = m.BrowserLanePid() == pid
	out.pidAliveOnFloor = syscall.Kill(pid, 0) == nil
	if st, err := exec.Command("ps", "-o", "stat=", "-p", fmt.Sprint(pid)).Output(); err == nil {
		out.floorState = strings.TrimSpace(string(st))
	}
	_ = m.Frame()
	out.floorFlush = flush("FLOOR")

	// ctrl+b → BROWSER: the SAME pid thaws; the RETAINED frame re-emits.
	runExec(ctrlB)
	if !m.BrowserPremiumActive() || m.BrowserLaneSuspended() {
		return out, fmt.Errorf("browser-keepalive: the return must THAW (active=%v suspended=%v)", m.BrowserPremiumActive(), m.BrowserLaneSuspended())
	}
	out.pidStable = out.pidStable && m.BrowserLanePid() == pid
	_ = m.Frame()
	out.spliceResume = flush("BACK")

	// ctrl+b → FLOOR again (the a=d repeats), then the ctrl+c QUIT PATH
	// reaps the frozen child (the delete rides the direct seam).
	runExec(ctrlB)
	if !m.BrowserLaneSuspended() {
		return out, fmt.Errorf("browser-keepalive: the second floor flip freezes again")
	}
	_ = m.Frame()
	out.floorFlush2 = flush("FLOOR2")
	buf.Reset()
	runExec(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	out.quitEmit = buf.String()
	out.pidGoneAfter = errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)

	if b, err := os.ReadFile(filepath.Join(root, "calls.log")); err == nil {
		out.spawns = strings.Count(strings.TrimSpace(string(b)), "\n") + 1
	}
	return out, nil
}

// browserKeepaliveIdentical — the two-drives byte-identity gate (the pid
// values vary; the verdicts + the wrapper bytes are the pin).
func browserKeepaliveIdentical(a, b browserKeepaliveOut) bool {
	return a.spliceOpen == b.spliceOpen && a.floorFlush == b.floorFlush &&
		a.spliceResume == b.spliceResume && a.floorFlush2 == b.floorFlush2 &&
		a.quitEmit == b.quitEmit && a.floorState == b.floorState &&
		a.spawns == b.spawns && a.pidStable == b.pidStable &&
		a.pidAliveOnFloor == b.pidAliveOnFloor && a.pidGoneAfter == b.pidGoneAfter
}

func runBrowserKeepaliveProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	k1, err := browserKeepaliveDrive()
	if err != nil {
		return err
	}
	k2, err := browserKeepaliveDrive()
	if err != nil {
		return err
	}

	// OPEN: the retained frame's a=T under the STABLE office id.
	if !strings.Contains(k1.spliceOpen, "\x1b_Ga=T,t=d,q=2,C=1,i="+browserStreamIDHash8+",f=100,") {
		return fail("browser-keepalive: the open splice carries the stable-id a=T:\n%q", k1.spliceOpen)
	}
	if !strings.Contains(k1.spliceOpen, browserStreamB64A) {
		return fail("browser-keepalive: the open splice carries the frame payload verbatim")
	}
	// FLOOR: the wrapper's diff flushes EXACTLY ONE a=d.
	if want := "FLOOR" + browserStreamDeleteFrame; k1.floorFlush != want {
		return fail("browser-keepalive: the floor flip flushes exactly one a=d:\n got %q\nwant %q", k1.floorFlush, want)
	}
	// the freeze's process verdicts: PID stable + alive + SIGSTOPped.
	if !k1.pidStable {
		return fail("browser-keepalive: the flip never respawns (the PID is stable)")
	}
	if !k1.pidAliveOnFloor {
		return fail("browser-keepalive: the frozen child stays ALIVE behind the floor")
	}
	if !strings.Contains(k1.floorState, "T") {
		return fail("browser-keepalive: the frozen child is SIGSTOPped (ps state %q, want T…)", k1.floorState)
	}
	// BACK: the RETAINED frame re-emits BYTE-IDENTICALLY — the parked
	// fake emits ZERO new bytes, so the a=T is definitionally the store's
	// cached frame (BEFORE any child output), with NO a=d interleaved.
	if got, want := k1.spliceResume, "BACK"+strings.TrimPrefix(k1.spliceOpen, "OPEN"); got != want {
		return fail("browser-keepalive: the resume re-emits the retained frame byte-identically:\n got %q\nwant %q", got, want)
	}
	if strings.Contains(k1.spliceResume, "\x1b_Ga=d,") {
		return fail("browser-keepalive: no a=d may interleave the thaw: %q", k1.spliceResume)
	}
	// the second floor flip: the a=d again; the quit: the direct delete.
	if want := "FLOOR2" + browserStreamDeleteFrame; k1.floorFlush2 != want {
		return fail("browser-keepalive: the second floor flip flushes the a=d again:\n got %q\nwant %q", k1.floorFlush2, want)
	}
	if !strings.Contains(k1.quitEmit, browserStreamDeleteFrame) {
		return fail("browser-keepalive: the quit path's Close flushes the delete through the direct seam, got %q", k1.quitEmit)
	}
	if !k1.pidGoneAfter {
		return fail("browser-keepalive: the quit path reaps the frozen child (no leak)")
	}
	if k1.spawns != 1 {
		return fail("browser-keepalive: exactly ONE spawn across the whole cycle, log %d", k1.spawns)
	}
	if !browserKeepaliveIdentical(k1, k2) {
		return fail("browser-keepalive: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER KEEP-ALIVE — the flip cycle: a=T … a=d (floor, child FROZEN + ALIVE, ps " + k1.floorState + ") … a=T (SAME id, the retained frame — byte-identical, instant) … a=d (quit) =====")
	fmt.Println("open splice:   " + fmt.Sprintf("%q", k1.spliceOpen))
	fmt.Println("floor flush:   " + fmt.Sprintf("%q", k1.floorFlush))
	fmt.Println("resume splice: " + fmt.Sprintf("%q", k1.spliceResume))
	fmt.Println("floor2 flush:  " + fmt.Sprintf("%q", k1.floorFlush2))
	fmt.Println("quit emit:     " + fmt.Sprintf("%q", k1.quitEmit))
	fmt.Println("===== UI SHOT =====")
	fmt.Println("asserts: OK — the KEEP-ALIVE flip cycle through the LIVE app glue (the fake child parked at `exec sleep` after ONE chunked kitty frame — ZERO new child bytes after the thaw, so the resume's a=T is definitionally the store's retained frame): ONE spawn across open → ctrl+b (floor) → ctrl+b (browser) → ctrl+b (floor) → ctrl+c; the PID stable + ALIVE + SIGSTOPped (ps state \"" + k1.floorState + "\") behind the floor; each floor-ward flip flushed exactly ONE `ESC_Ga=d,d=I,i=" + browserStreamIDHash8 + ",q=2;ESC\\` through the wrapper's emitted-set diff; the return re-emitted the RETAINED frame BYTE-IDENTICALLY (no respawn, no reload, one frame flush) with ZERO a=d interleaved; the ctrl+c quit path reaped the frozen child (pid gone) with the delete riding the direct seam; every leg byte-identical across two drives")
	return nil
}

// --- browser tab text-lane hint (--browser --lane hint) ----------------------
// The text lane's "why" row through the LIVE APP GLUE (the pane's own
// per-class/persistence contracts live in internal/panels'
// browser_hint_test.go): the hermetic ghostty stub (kitty-capable, both
// kill-switch spellings cleared) with PATH pinned to an EMPTY fixture dir
// — the terminal-browser probe misses BY CONSTRUCTION, no host binary can
// ever leak in — so the pane resolves the text lane with the
// binary-missing class AT pane-creation time. Leg E: ctrl+b flips the
// LEFT slot to the browser and the idle starter card wears the dim hint
// row under the location bar. Leg F: "/open file://<fixture>" through the
// REAL chat input paints the warm text page with the SAME hint persisting
// under the bar (the member sees it at the moment of disappointment, not
// just on the empty card). Every leg byte-identical across two drives.

// browserHintFrameOut — ONE hint drive's observed artifacts.
type browserHintFrameOut struct {
	frameIdle string // ctrl+b — the starter card + the hint row
	frameOpen string // /open — the text page + the persistent hint row
	leftTab   int
	activeTab int
}

// browserHintDrive — ONE hermetic drive with the probe guaranteed to miss.
func browserHintDrive() (browserHintFrameOut, error) {
	var out browserHintFrameOut
	defer pinShotEngineAbsent()() // no live chrome
	saved, present := map[string]string{}, map[string]bool{}
	for _, k := range browserEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k], present[k] = v, true
		}
	}
	defer func() { // restore EVERY key (the drive pairs share the process)
		for _, k := range browserEnvKeys {
			if present[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	root, err := os.MkdirTemp("", "uishot-browser-hint-")
	if err != nil {
		return out, fmt.Errorf("browser-hint fixture: %w", err)
	}
	defer os.RemoveAll(root)
	// PATH pins the EMPTY fixture dir ALONE: exec.LookPath("terminal-browser")
	// misses BY CONSTRUCTION (the host's real PATH never leaks a binary in).
	os.Setenv("PATH", root)
	os.Setenv("TERM_PROGRAM", "ghostty") // the hermetic kitty-capable host stub
	for _, k := range []string{"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM_VERSION", "WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID"} {
		os.Setenv(k, "")
	}
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv(panels.BrowserLaneOffEnv, "")
	os.Setenv(panels.TerminalBrowserOffEnv, "")

	backend := &stubBackend{done: make(chan struct{})}
	m := app.New(backend, config.Default())
	// runExec — the exact breadth-first drain the --browsertab proof runs.
	runExec := func(msg tea.Msg) {
		tm, cmd := m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
				// heartbeats re-arm forever — dropped, exactly as runMsg does
			default:
				tm2, next := m.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}

	runExec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — browser hint stub online"})

	// leg E — ctrl+b to the browser slot: the idle starter card wears the
	// dim hint row under the location bar (the resolve pinned binary-missing
	// at pane creation — the empty PATH is the whole trick).
	runExec(tea.KeyPressMsg(tea.Key{Code: 'b', Mod: tea.ModCtrl}))
	out.leftTab = m.LeftTabIndex()
	out.activeTab = m.ActiveTabIndex()
	out.frameIdle = m.Frame()

	// leg F — back on the floor (the browser slot owns its keys, Enter
	// included — the chat draft only rides from the floor), /open through
	// the REAL chat input (the bracketed-paste idiom from the live-lane
	// drive): the slash flips the slot back to the browser and the warm
	// text page paints with the SAME hint persisting under the bar.
	runExec(tea.KeyPressMsg(tea.Key{Code: 'b', Mod: tea.ModCtrl}))
	fixtureAbs, err := filepath.Abs(browserTabFixtureRel)
	if err != nil {
		return out, fmt.Errorf("browser-hint fixture path: %w", err)
	}
	runExec(tea.PasteMsg{Content: "/open file://" + fixtureAbs})
	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // sends → slashMsg → applySlash → fetch
	out.frameOpen = m.Frame()
	return out, nil
}

// browserHintIdentical — the two-drives byte-identity gate.
func browserHintIdentical(a, b browserHintFrameOut) bool {
	return a.frameIdle == b.frameIdle && a.frameOpen == b.frameOpen &&
		a.leftTab == b.leftTab && a.activeTab == b.activeTab
}

func runBrowserHintProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	e1, err := browserHintDrive()
	if err != nil {
		return err
	}
	e2, err := browserHintDrive()
	if err != nil {
		return err
	}
	if e1.leftTab != 1 {
		return fail("browser-hint E: ctrl+b must flip the LEFT slot to the browser (leftTab 1), got %d", e1.leftTab)
	}
	if e1.activeTab != 0 {
		return fail("browser-hint E: the RIGHT strip must stay on chat (index 0), got %d", e1.activeTab)
	}
	// the binary-missing class's copy — ansi-truncated at the slot width;
	// the class-naming prefix rides every width (the FULL verbatim copy is
	// pinned in panels' browser_hint_test.go).
	const hintPrefix = "text lane — terminal-browser not on PATH"
	idle := ansi.Strip(e1.frameIdle)
	for _, want := range []string{hintPrefix, "▸ enter a url · /open <url> · e to edit · o for file"} {
		if !strings.Contains(idle, want) {
			return fail("browser-hint E idle frame missing %q:\n%s", want, idle)
		}
	}
	// the hint rides UNDER the location bar (the row right below it).
	idleLines := strings.Split(idle, "\n")
	hintUnderBar := false
	for i, line := range idleLines {
		if strings.Contains(line, "▸ browser") && i+1 < len(idleLines) && strings.Contains(idleLines[i+1], hintPrefix) {
			hintUnderBar = true
			break
		}
	}
	if !hintUnderBar {
		return fail("browser-hint E: the hint must ride the row UNDER the location bar:\n%s", idle)
	}
	open := ansi.Strip(e1.frameOpen)
	for _, want := range []string{hintPrefix, "▸ file:///", "The Fixture Gazette"} {
		if !strings.Contains(open, want) {
			return fail("browser-hint F open frame missing %q:\n%s", want, open)
		}
	}
	// the text lane never wears premium chrome, either frame.
	for _, never := range []string{" zenbu ", "zenbu terminal-browser ·"} {
		if strings.Contains(idle, never) || strings.Contains(open, never) {
			return fail("browser-hint: the text lane never wears premium chrome %q:\n%s\n%s", never, idle, open)
		}
	}
	if !browserHintIdentical(e1, e2) {
		return fail("browser-hint: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER HINT E — ctrl+b to the idle browser: the starter card wears the dim text-lane hint (terminal-browser not on PATH) =====")
	fmt.Println(e1.frameIdle)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("===== UI SHOT · BROWSER HINT F — /open file://<fixture>: the hint persists under the location bar over the warm text page =====")
	fmt.Println(e1.frameOpen)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("asserts: OK — the hermetic ghostty stub with PATH pinned to an EMPTY fixture dir (the terminal-browser probe misses BY CONSTRUCTION, no host leak) resolved the text lane with the binary-missing class AT pane creation; leg E: ctrl+b flipped the LEFT slot to the browser (right strip unmoved) and the idle starter card wears the dim hint row UNDER the location bar (\"text lane — terminal-browser not on PATH · full rendering: github.com/zenbu-labs/terminal-browser (or re-run the office installer)\", ansi-truncated at the slot width); leg F: \"/open file://<fixture>\" through the REAL chat input painted the warm text page with the SAME hint persisting under the bar; no premium chrome in either frame; every leg byte-identical across two drives")
	return nil
}

// --- browser tab headless SHOT lane (--browser --lane shot) -------------------
// The headless screenshot lane through the LIVE app glue (the pane's own
// flow contracts + the registry/wrapper byte-pins live in internal/panels'
// browser_panel_lane_test.go + internal/app's browser_frame_test.go): a
// FAKE headless engine behind the panels seam (NO live chrome) renders
// the shared checker PNG, the shot clock + shots home pin byte-stable
// paths, PATH pins an empty fixture dir (no terminal-browser — the zenbu
// lane misses by construction), and "/open file://<fixture>" types
// through the REAL chat input. Flavors: "ok" (the kitty SHOT MODE: the
// " shot " badge + "▸ headless chromium · <url>" strip paint, the
// registry publishes the PNG at absolute (0,3)+pane-local (0,0), the
// wrapper's flush byte-pins cursor-save + CUP(4;1) + the f=100 APC with
// NO c=/r= keys + cursor-restore, esc flushes ONE a=d through the diff,
// ctrl+b re-publishes the CACHED bytes with ZERO new engine calls, the
// PNG saves under the convention's <ts>-<hash8>.png name); "chrome" /
// "refused" / "timeout" (the failure classes' exact dim rows, the text
// page warm underneath, the registry untouched); "nonkitty" (the iTerm
// stub: NO shot mode ever — the text lane + the dim "screenshot: <path>"
// row). Every leg byte-identical across two drives.

// browserShotPinTime — the pinned shot clock (byte-stable saved paths).
var browserShotPinTime = time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)

// browserShotOut — ONE shot drive's observed artifacts.
type browserShotOut struct {
	frame       string // after /open — the shot region (ok) / the failure row (others)
	flush       string // the wrapper's "SHOT" flush (ok: CUP(4;1) + APC; failures: passthrough)
	flushFloor  string // after esc — ok: ONE a=d through the emitted-set diff
	flushBack   string // after ctrl+b — ok: the CACHED bytes re-published byte-identically
	savedPath   string // the pane's saved PNG path
	savedOK     bool   // the saved file's bytes == the checker's
	engineCalls int    // the fake engine's render count (the flip never re-renders)
	engineW     int    // the render's recorded viewport dims
	engineH     int
	leftTab     int
	activeTab   int
}

// browserShotDrive — ONE hermetic drive of the given flavor
// (ok|chrome|refused|timeout|nonkitty).
func browserShotDrive(flavor string) (browserShotOut, error) {
	var out browserShotOut
	saved, present := map[string]string{}, map[string]bool{}
	for _, k := range browserEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k], present[k] = v, true
		}
	}
	defer func() { // restore EVERY key (the drive pairs share the process)
		for _, k := range browserEnvKeys {
			if present[k] {
				os.Setenv(k, saved[k])
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	root, err := os.MkdirTemp("", "uishot-browser-shot-")
	if err != nil {
		return out, fmt.Errorf("browser-shot fixture: %w", err)
	}
	defer os.RemoveAll(root)
	// the shots home is a FIXED path (byte-identity across the pair): the
	// frame prints the saved PNG's path, so the directory may never drift.
	home := filepath.Join(os.TempDir(), "uishot-browser-shot-home")
	if err := os.RemoveAll(home); err != nil {
		return out, fmt.Errorf("browser-shot home reset: %w", err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return out, fmt.Errorf("browser-shot home: %w", err)
	}
	defer os.RemoveAll(home)

	png, err := os.ReadFile("internal/panels/testdata/checker-8x8.png")
	if err != nil {
		return out, fmt.Errorf("browser-shot checker fixture: %w", err)
	}

	// the flavor's fake engine (the panels seam — NO live chrome).
	calls, lastW, lastH := 0, 0, 0
	avail := func() (string, bool) { return "/fake/chrome", true }
	var shotFn func(context.Context, string, int, int) (*headless.Result, error)
	shotFn = func(_ context.Context, rawurl string, w, h int) (*headless.Result, error) {
		calls++
		lastW, lastH = w, h
		switch flavor {
		case "refused":
			return nil, &headless.PolicyError{URL: rawurl, Reason: "plain http to example.test refused"}
		case "timeout":
			return nil, context.DeadlineExceeded
		default: // ok + nonkitty
			return &headless.Result{URL: rawurl, Title: "Fixture Gazette", PNG: png}, nil
		}
	}
	if flavor == "chrome" {
		avail = func() (string, bool) { return "", false }
	}
	restoreEngine := panels.SetHeadlessForShot(avail, shotFn)
	defer restoreEngine()
	restoreClock := panels.SetShotNowForShot(func() time.Time { return browserShotPinTime })
	defer restoreClock()

	// the hermetic host stub: ghostty for the kitty legs, iTerm for the
	// non-kitty gate; PATH pins the EMPTY fixture dir (no terminal-browser,
	// no host leak); the shots home + cell metric pin.
	os.Setenv("PATH", root)
	if flavor == "nonkitty" {
		os.Setenv("TERM_PROGRAM", "iTerm.app")
	} else {
		os.Setenv("TERM_PROGRAM", "ghostty")
	}
	for _, k := range []string{"TMUX", "KITTY_WINDOW_ID", "TERM_PROGRAM_VERSION", "WEZTERM_UNIX_SOCKET", "VSCODE_PID", "ITERM_SESSION_ID"} {
		os.Setenv(k, "")
	}
	os.Setenv("TERM", "xterm-256color")
	os.Setenv("COLORTERM", "truecolor")
	os.Setenv(panels.BrowserLaneOffEnv, "")
	os.Setenv(panels.TerminalBrowserOffEnv, "")
	os.Setenv("THEBORINGOFFICE_ZENBU_LANE", "")
	os.Setenv("THEBORINGOFFICE_HOME", home)
	os.Setenv("THEBORINGOFFICE_CELL_PX", "")

	// the frame-splice wrapper over the SHARED registry (the production
	// seam's exact shape), the lane's direct deletes captured.
	reg := panels.ZenbuRegistry()
	reg.Clear()
	defer reg.Clear()
	var buf strings.Builder
	w := panels.NewZenbuFrameWriter(&buf, reg)
	restoreEmit := panels.SetZenbuEmitForShot(w.DirectEmit)
	defer restoreEmit()

	backend := &stubBackend{done: make(chan struct{})}
	m := app.New(backend, config.Default())
	// runExec — the exact breadth-first drain the --browsertab proof runs.
	runExec := func(msg tea.Msg) {
		tm, cmd := m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
				// heartbeats re-arm forever — dropped, exactly as runMsg does
			default:
				tm2, next := m.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}
	flush := func(tag string) string {
		buf.Reset()
		_, _ = w.Write([]byte(tag))
		return buf.String()
	}

	runExec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — headless engine stub online"})
	fixtureAbs, err := filepath.Abs(browserTabFixtureRel)
	if err != nil {
		return out, fmt.Errorf("browser-shot fixture path: %w", err)
	}
	url := "file://" + fixtureAbs
	runExec(tea.PasteMsg{Content: "/open " + url})
	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // sends → slashMsg → applyOpenSlash → fetch + the shot arm

	out.leftTab = m.LeftTabIndex()
	out.activeTab = m.ActiveTabIndex()
	out.frame = m.Frame() // publishes the registry (ok: the shot's PNG)
	out.flush = flush("SHOT")

	// the keep-alive flip: esc hides the slot (the wrapper's diff flushes
	// the delete), ctrl+b back re-publishes the CACHED bytes (no re-render).
	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	_ = m.Frame()
	out.flushFloor = flush("FLOOR")
	runExec(tea.KeyPressMsg(tea.Key{Code: 'b', Mod: tea.ModCtrl}))
	_ = m.Frame()
	out.flushBack = flush("BACK")

	out.engineCalls, out.engineW, out.engineH = calls, lastW, lastH
	out.savedPath = m.BrowserShotPath()
	if out.savedPath != "" {
		if back, err := os.ReadFile(out.savedPath); err == nil && len(back) == len(png) {
			out.savedOK = true
			for i := range png {
				if back[i] != png[i] {
					out.savedOK = false
					break
				}
			}
		}
	}
	return out, nil
}

// browserShotIdentical — the two-drives byte-identity gate.
func browserShotIdentical(a, b browserShotOut) bool {
	return a.frame == b.frame && a.flush == b.flush && a.flushFloor == b.flushFloor &&
		a.flushBack == b.flushBack && a.savedPath == b.savedPath && a.savedOK == b.savedOK &&
		a.engineCalls == b.engineCalls && a.engineW == b.engineW && a.engineH == b.engineH &&
		a.leftTab == b.leftTab && a.activeTab == b.activeTab
}

// browserShotAPC — the EXPECTED emitted APC for the checker PNG (a=T +
// t=d + q=2 + C=1 + i=<content hash8> + f=100, NO c=/r= keys).
func browserShotAPC(png []byte) string {
	return "\x1b_Ga=T,t=d,q=2,C=1,i=" + panels.KittyIDHash8(panels.KittyImageID(png)) +
		",f=100;" + base64.StdEncoding.EncodeToString(png) + "\x1b\\"
}

// browserShotDelete — the office-side a=d for the checker PNG's content id.
func browserShotDelete(png []byte) string {
	return "\x1b_Ga=d,d=I,i=" + panels.KittyIDHash8(panels.KittyImageID(png)) + ",q=2;\x1b\\"
}

func runBrowserShotProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	png, err := os.ReadFile("internal/panels/testdata/checker-8x8.png")
	if err != nil {
		return fmt.Errorf("browser-shot checker fixture: %w", err)
	}
	wantPath := filepath.Join(os.TempDir(), "uishot-browser-shot-home", "shots",
		"1787918400000-"+panels.KittyIDHash8(panels.KittyImageID(png))+".png")

	// leg OK — the kitty SHOT MODE through the LIVE app glue.
	o1, err := browserShotDrive("ok")
	if err != nil {
		return err
	}
	o2, err := browserShotDrive("ok")
	if err != nil {
		return err
	}
	if o1.leftTab != 1 || o1.activeTab != 0 {
		return fail("browser-shot ok: /open flips the LEFT slot to browser (1), the right strip stays on chat (0): got %d/%d", o1.leftTab, o1.activeTab)
	}
	live := ansi.Strip(o1.frame)
	for _, want := range []string{" shot ", "▸ headless chromium · file:///", "screenshot: /", "· ctrl+b"} {
		if !strings.Contains(live, want) {
			return fail("browser-shot ok frame missing %q:\n%s", want, live)
		}
	}
	// the strip paints INSIDE the LEFT slot (left of the sidebar's chat).
	stripSeen := false
	for _, line := range strings.Split(live, "\n") {
		if strings.Contains(line, "headless chromium") {
			stripSeen = true
			hi, ci := strings.Index(line, "headless chromium"), strings.Index(line, "chat")
			if ci >= 0 && hi > ci {
				return fail("browser-shot ok: the shot strip must paint LEFT of the sidebar's chat tab: %q", line)
			}
		}
	}
	if !stripSeen {
		return fail("browser-shot ok: the shot strip never rendered")
	}
	if o1.savedPath != wantPath {
		return fail("browser-shot ok: the saved PNG rides the pinned convention path:\n got %q\nwant %q", o1.savedPath, wantPath)
	}
	if !o1.savedOK {
		return fail("browser-shot ok: the saved PNG's bytes round-trip the checker's")
	}
	// the wrapper's flush, byte-pinned: cursor-save + CUP(4;1) (absolute
	// (0,3) + pane-local (0,0), 1-based) + the f=100 APC + cursor-restore.
	wantFlush := "SHOT" + "\x1b7\x1b[4;1H" + browserShotAPC(png) + "\x1b8"
	if o1.flush != wantFlush {
		return fail("browser-shot ok: the wrapper's emitted bytes:\n got %q\nwant %q", o1.flush, wantFlush)
	}
	if strings.Contains(o1.flush, ",c=") || strings.Contains(o1.flush, ",r=") {
		return fail("browser-shot ok: the wave-81 ruling bans c=/r= keys: %q", o1.flush)
	}
	if want := "FLOOR" + browserShotDelete(png); o1.flushFloor != want {
		return fail("browser-shot ok: esc flushes exactly one a=d through the wrapper's diff:\n got %q\nwant %q", o1.flushFloor, want)
	}
	if want := "BACK" + "\x1b7\x1b[4;1H" + browserShotAPC(png) + "\x1b8"; o1.flushBack != want {
		return fail("browser-shot ok: the return re-publishes the CACHED bytes byte-identically:\n got %q\nwant %q", o1.flushBack, want)
	}
	if o1.engineCalls != 1 {
		return fail("browser-shot ok: the flip cycle NEVER re-renders (calls %d, want 1)", o1.engineCalls)
	}
	if o1.engineW <= 0 || o1.engineH <= 0 {
		return fail("browser-shot ok: the render fired at the pane box's pixel dims (%dx%d)", o1.engineW, o1.engineH)
	}
	if !browserShotIdentical(o1, o2) {
		return fail("browser-shot ok: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER SHOT — /open through the REAL app: the headless SHOT MODE in the LEFT slot =====")
	fmt.Println(o1.frame)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("===== UI SHOT · BROWSER SHOT — the wrapper's emitted bytes (f=100, NO c=/r=) + the keep-alive flip =====")
	fmt.Println("open flush:  " + fmt.Sprintf("%q", o1.flush))
	fmt.Println("floor flush: " + fmt.Sprintf("%q", o1.flushFloor))
	fmt.Println("back flush:  " + fmt.Sprintf("%q", o1.flushBack))
	fmt.Println("===== UI SHOT =====")

	// the failure classes — each lands its exact dim row (ansi-truncated
	// at the slot width — the class-naming PREFIX rides every width; the
	// FULL verbatim copies pin in panels' browser_panel_lane_test.go).
	type failLeg struct {
		flavor, rowPrefix string
	}
	for _, leg := range []failLeg{
		{"chrome", "text lane — headless chrome not found"},
		{"refused", "text lane — headless render refused"},
		{"timeout", "text lane — headless render timed out"},
	} {
		f1, err := browserShotDrive(leg.flavor)
		if err != nil {
			return err
		}
		f2, err := browserShotDrive(leg.flavor)
		if err != nil {
			return err
		}
		frame := ansi.Strip(f1.frame)
		if !strings.Contains(frame, leg.rowPrefix) {
			return fail("browser-shot %s: the failure lands its dim row (prefix %q):\n%s", leg.flavor, leg.rowPrefix, frame)
		}
		if !strings.Contains(frame, "The Fixture Gazette") {
			return fail("browser-shot %s: the text page stays warm under the failure:\n%s", leg.flavor, frame)
		}
		for _, never := range []string{"▸ headless chromium", " shot "} {
			if strings.Contains(frame, never) {
				return fail("browser-shot %s: a failure never wears shot chrome %q:\n%s", leg.flavor, never, frame)
			}
		}
		if f1.flush != "SHOT" || f1.flushFloor != "FLOOR" || f1.flushBack != "BACK" {
			return fail("browser-shot %s: the registry stays EMPTY (the wrapper passes through):\n%q %q %q", leg.flavor, f1.flush, f1.flushFloor, f1.flushBack)
		}
		if !browserShotIdentical(f1, f2) {
			return fail("browser-shot %s: two drives must be byte-identical", leg.flavor)
		}
		fmt.Printf("===== UI SHOT · BROWSER SHOT %s — the failure class's dim row =====\n%s\n===== UI SHOT =====\n", strings.ToUpper(leg.flavor), f1.frame)
	}

	// leg NONKITTY — the iTerm stub: NO shot mode ever; the text lane +
	// the dim "screenshot: <path>" row (the save still lands).
	n1, err := browserShotDrive("nonkitty")
	if err != nil {
		return err
	}
	n2, err := browserShotDrive("nonkitty")
	if err != nil {
		return err
	}
	nk := ansi.Strip(n1.frame)
	for _, want := range []string{"screenshot: /", "The Fixture Gazette", "▸ file:///"} {
		if !strings.Contains(nk, want) {
			return fail("browser-shot nonkitty frame missing %q:\n%s", want, nk)
		}
	}
	for _, never := range []string{"▸ headless chromium", " shot "} {
		if strings.Contains(nk, never) {
			return fail("browser-shot nonkitty: shot mode is kitty-only (%q):\n%s", never, nk)
		}
	}
	if !n1.savedOK || n1.savedPath != wantPath {
		return fail("browser-shot nonkitty: the save still lands (path %q ok %v)", n1.savedPath, n1.savedOK)
	}
	if n1.flush != "SHOT" || n1.flushFloor != "FLOOR" || n1.flushBack != "BACK" {
		return fail("browser-shot nonkitty: the registry stays EMPTY on a non-kitty host")
	}
	if n1.engineCalls != 1 {
		return fail("browser-shot nonkitty: the flip cycle never re-renders (calls %d)", n1.engineCalls)
	}
	if !browserShotIdentical(n1, n2) {
		return fail("browser-shot nonkitty: two drives must be byte-identical")
	}
	fmt.Println("===== UI SHOT · BROWSER SHOT NONKITTY — the iTerm stub: the text lane + the saved-PNG row (never shot mode) =====")
	fmt.Println(n1.frame)
	fmt.Println("===== UI SHOT =====")

	fmt.Println("asserts: OK — the headless SHOT lane through the LIVE app glue (a FAKE engine behind the panels seam — NO live chrome — rendering the shared checker PNG; the shot clock + a FIXED shots home pin byte-stable paths; PATH pinned empty so the zenbu lane misses by construction; \"/open file://<fixture>\" through the REAL chat input): leg ok (hermetic ghostty stub): the fetch's landing armed ONE render at the pane box's exact pixel dims, the \" shot \" badge + \"▸ headless chromium · <url>\" strip painted INSIDE the LEFT slot (left of the sidebar's chat, the right strip unmoved), the registry published the PNG at absolute (0,3)+pane-local (0,0) — the wrapper's flush byte-pinned cursor-save + CUP(4;1) + `ESC_Ga=T,t=d,q=2,C=1,i=" + panels.KittyIDHash8(panels.KittyImageID(png)) + ",f=100;<b64>ESC\\` with NO c=/r= keys (the wave-81 emission ruling) + cursor-restore — esc hid the slot with exactly ONE `ESC_Ga=d,d=I` through the emitted-set diff, ctrl+b re-published the CACHED bytes byte-identically with ZERO new engine calls (ONE render across the flip cycle), and the PNG saved under shots/1787918400000-<hash8>.png with the checker's exact bytes; the failure classes (chrome-missing / navigation-refused / timeout) each landed their exact dim row with the text page warm underneath and the registry untouched; the nonkitty leg (iTerm stub) NEVER entered shot mode — the text lane + the dim \"screenshot: <path>\" row, the save still landing; every leg byte-identical across two drives")
	return nil
}

// --- browser tab text viewer (--browsertab) ---------------------------------
// The UNCONDITIONAL lane: the browser tab itself — no zenbu, no external
// binary, every host — riding the LEFT pane's floor|browser switcher. A
// REAL stub HTTP server on the pinned loopback port serves the shared
// panels fixture (404 for every other route); the member types
// "/open http://127.0.0.1:52731/fixture.html" through the REAL chat input
// ("/" opens the slash popover, the fragment filters, Enter prefills
// "/open ", the URL types, the second Enter sends → slashMsg → applySlash
// → the pane's FetchPage — the drain runs the round-trip synchronously).
// Asserts: the LEFT slot flipped to browser (the RIGHT strip never moved
// off chat), the switcher strip carries "floor"/"browser" + the ctrl+b
// hint, the "▸ <url>" location bar renders in the LEFT pane, the bold
// heading row, the three indexed link rows, the 🖼 chip + the " │ "
// table after pgdn scrolls them into view, the tail-marker crossing the
// fold — and two drives byte-identical.

// browserTabStubAddr — the PINNED loopback port (byte-determinism: the
// location bar prints the URL, so the port may never drift between drives).
const browserTabStubAddr = "127.0.0.1:52731"

// browserTabFixtureRel — the shared panels fixture, repo-root relative
// (the proof runs from the repo root, the --images fixture-read contract).
const browserTabFixtureRel = "internal/panels/testdata/fixture.html"

// browserTabFrameOut — ONE synchronous drive's observed artifacts.
type browserTabFrameOut struct {
	frameTop     string // after /open — bar + top rows, chip + tail-marker below the fold
	frameScroll1 string // after ONE pgdn — the " │ " table + the 🖼 chip scrolled into view
	frameScroll2 string // after TWO pgdn — the tail-marker crossed the fold
	activeTab    int    // the RIGHT strip's position — /open must never move it
	leftTab      int    // the LEFT slot's switcher — /open flips it to browser
}

func browserTabDrive() (browserTabFrameOut, error) {
	var out browserTabFrameOut
	defer pinShotEngineAbsent()() // no live chrome (the stub URL is localhost-allowed)
	// the TEXT-viewer proof is hermetic on every host: on a kitty-capable
	// machine with terminal-browser on PATH the pane's lane resolve would
	// otherwise spawn a real child out of /open (the premium lane's OWN
	// proofs live at --browser [--lane kitty|live]).
	oldLaneOff, laneOffSet := os.LookupEnv(panels.BrowserLaneOffEnv)
	os.Setenv(panels.BrowserLaneOffEnv, "1")
	defer func() {
		if laneOffSet {
			os.Setenv(panels.BrowserLaneOffEnv, oldLaneOff)
		} else {
			os.Unsetenv(panels.BrowserLaneOffEnv)
		}
	}()
	ln, err := net.Listen("tcp", browserTabStubAddr)
	if err != nil {
		return out, fmt.Errorf("browsertab: stub listen %s: %w (a stale proof still running?)", browserTabStubAddr, err)
	}
	defer ln.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/fixture.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, browserTabFixtureRel)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	backend := &stubBackend{done: make(chan struct{})}
	m := app.New(backend, config.Default())
	// runExec — the exact breadth-first drain the --links/--images proofs
	// run (heartbeats dropped; every other landing re-feeds — the slash
	// send, the fetch round-trip and the page verdict all land here).
	runExec := func(msg tea.Msg) {
		tm, cmd := m.Update(msg)
		if fm, ok := tm.(app.Model); ok {
			m = fm
		}
		queue := []tea.Cmd{cmd}
		for len(queue) > 0 {
			c := queue[0]
			queue = queue[1:]
			if c == nil {
				continue
			}
			res := c()
			if res == nil {
				continue
			}
			switch res := res.(type) {
			case tea.BatchMsg:
				queue = append(queue, res...)
			case spinner.TickMsg, cursor.BlinkMsg:
				// heartbeats re-arm forever — dropped, exactly as runMsg does
			default:
				tm2, next := m.Update(res)
				if fm2, ok := tm2.(app.Model); ok {
					m = fm2
				}
				if next != nil {
					queue = append(queue, next)
				}
			}
		}
	}

	runExec(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	runExec(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — browser tab stub online"})

	// /open through the REAL chat input + popover (the --slashpop idiom).
	url := "http://" + browserTabStubAddr + "/fixture.html"
	typeIn := func(s string) {
		for _, r := range s {
			runExec(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		}
	}
	typeIn("/open")
	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // popover picks /open → draft "/open "
	typeIn(url)
	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // sends → slashMsg → applySlash → fetch

	out.activeTab = m.ActiveTabIndex()
	out.leftTab = m.LeftTabIndex()
	out.frameTop = m.Frame()

	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	out.frameScroll1 = m.Frame()
	runExec(tea.KeyPressMsg(tea.Key{Code: tea.KeyPgDown}))
	out.frameScroll2 = m.Frame()
	return out, nil
}

func runBrowserTabProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	a1, err := browserTabDrive()
	if err != nil {
		return err
	}
	a2, err := browserTabDrive()
	if err != nil {
		return err
	}
	if a1.leftTab != 1 {
		return fail("browsertab: /open must flip the LEFT slot to the browser (leftTab 1), got %d", a1.leftTab)
	}
	if a1.activeTab != 0 {
		return fail("browsertab: /open must never move the RIGHT strip off chat (index 0), got %d", a1.activeTab)
	}
	url := "http://" + browserTabStubAddr + "/fixture.html"
	strip := ansi.Strip(a1.frameTop)
	for _, want := range []string{
		"▸ " + url,            // the location bar rides the LEFT pane (the · title tail truncates at the 50-col slot)
		"The Fixture Gazette", // the h1 row
		"News desk",           // the h2 row
		"Open link alpha [1] for the first story.",
		"Then link beta [2] for the follow-up.",
		"Finally link gamma [3] closes the set.",
		"• shelf item one",
		"· ctrl+b", // the left slot's switcher strip (floor | browser)
	} {
		if !strings.Contains(strip, want) {
			return fail("browsertab top frame missing %q:\n%s", want, strip)
		}
	}
	// the layout itself: the switcher row carries the browser label LEFT of
	// the right strip's chat tab (floor slot left, sidebar right).
	for _, line := range strings.Split(strip, "\n") {
		if strings.Contains(line, "ctrl+b") {
			bi, ci := strings.Index(line, "browser"), strings.Index(line, "chat")
			if bi < 0 || ci < 0 || bi > ci {
				return fail("browsertab: the switcher row must place browser LEFT of the chat tab: %q", line)
			}
			break
		}
	}
	if strings.Contains(strip, "tail-marker") || strings.Contains(strip, "🖼 fixture diagram") {
		return fail("browsertab: the chip + tail-marker must start BELOW the fold (pgdn's proof needs them)")
	}
	// ONE pgdn (the 28-row body hops a full page): the " │ " table + the
	// 🖼 chip scroll into view.
	scrolled1 := ansi.Strip(a1.frameScroll1)
	for _, want := range []string{"Data corner", "agent │ role", "tekton-1 │ developer", "skopos-1 │ scout", "🖼 fixture diagram"} {
		if !strings.Contains(scrolled1, want) {
			return fail("browsertab pgdn-1 frame missing %q:\n%s", want, scrolled1)
		}
	}
	// TWO pgdn (clamped at the content floor): the tail-marker crosses.
	scrolled2 := ansi.Strip(a1.frameScroll2)
	for _, want := range []string{"🖼 fixture diagram", "tail-marker"} {
		if !strings.Contains(scrolled2, want) {
			return fail("browsertab pgdn-2 frame missing %q:\n%s", want, scrolled2)
		}
	}
	if a1.frameTop == a1.frameScroll1 || a1.frameScroll1 == a1.frameScroll2 {
		return fail("browsertab: pgdn must move the scrollback (frames identical)")
	}
	if a1.frameTop != a2.frameTop || a1.frameScroll1 != a2.frameScroll1 || a1.frameScroll2 != a2.frameScroll2 {
		return fail("browsertab: two drives must produce byte-identical frames")
	}
	fmt.Println("===== UI SHOT · BROWSERTAB — /open " + url + " (LEFT slot: floor|browser switcher, top of page) =====")
	fmt.Println(a1.frameTop)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("===== UI SHOT · BROWSERTAB — after ONE pgdn (table + chip over the fold) =====")
	fmt.Println(a1.frameScroll1)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("===== UI SHOT · BROWSERTAB — after TWO pgdn (tail-marker crossing) =====")
	fmt.Println(a1.frameScroll2)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("asserts: OK — the browser renders the stub fixture as navigable text rows in the LEFT pane (floor|browser switcher strip + \"· ctrl+b\" hint on top, the RIGHT strip unmoved on chat): \"▸ " + url + "\" bar, bold headings, \"link alpha [1]\"/\"link beta [2]\"/\"link gamma [3]\" indexed rows, bullet run, ONE pgdn bringing the \" │ \" table + the 🖼 chip over the fold and TWO crossing the tail-marker; the left slot flipped to browser via the REAL /open slash path; two drives byte-identical")
	return nil
}

// --- clickable agents (--click) ----------------------------------------------
// Scripted bubbletea v2 mouse clicks through the REAL model: (S) a click on
// tekton-1's floor sprite selects it — activity tab opens, the agents tab
// pins a ▸ marker, an office notice names it; (D) a double-click on the
// same sprite toggles that agent's chat thread (and jumps there); (H) a
// click on a worker thread's "┌" header row in chat toggles it too.

// clickPairGap — the model's double-click window (400ms); the proof sleeps
// past it between the select phase and the double-click phase.
const clickPairGap = 400 * time.Millisecond

// clickAt sends ONE physical click as press + release — exactly what a real
// terminal emits under CellMotion for each click. The transcript selection
// seam (press arms, a MOTIONLESS release replays the click through the
// legacy path) needs the release, or a chat-region click would hang armed
// forever. One clickAt is still ONE click — the proof's behavior contract
// (one click acts, not two) is untouched.
func clickAt(d *focusDriver, x, y int) {
	d.send(tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft}))
	d.send(tea.MouseReleaseMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft}))
}

func runClickProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	d := newFocusDriver()
	d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — click stub online"})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvHire, Employee: state.Employee{
		ID: "sco-1", Name: "skopos-1", Role: state.RoleScout, Sprite: state.SpriteAtDesk}})
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user",
		"wire the sse stream", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-m1", "boss",
		"Both workers are on it.", false)})
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "dev-1",
		Task: state.BoardTask{ID: "t1", Title: "Wire the SSE stream", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "dev-1", TaskID: "t1"})
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "sco-1",
		Task: state.BoardTask{ID: "t2", Title: "Scan the repo", At: time.Now().UnixMilli()}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "sco-1", TaskID: "t2"})
	d.send(focusTool("dev-1", "tekton-1", "call-t1", "read", "internal/room/manager.go", "done"))
	d.send(focusTool("sco-1", "skopos-1", "call-s1", "grep", "SSE, 12 hits", "done"))
	d.send(focusTool("dev-1", "tekton-1", "call-t2", "edit", "internal/room/handler.go", "done"))
	d.send(focusTool("sco-1", "skopos-1", "call-s2", "read", "internal/api/room.go", "done"))
	d.pump(30) // walkers settle at their anchors; the plan is built
	_ = d.m.Frame()

	clickFloor := func(id string) {
		p, ok := office.SpritePosition(id)
		if !ok {
			return
		}
		// screen coords: floor X is absolute, +1 row for the topbar
		clickAt(d, p.X+1, p.Y+1)
	}

	// (S) single click on tekton-1's sprite → selection
	clickFloor("dev-1")
	fmt.Println("===== UI SHOT · CLICK S — floor click on tekton-1: activity tab opened, agent selected =====")
	frameS := d.m.Frame()
	fmt.Println(frameS)
	fmt.Println("===== UI SHOT =====")
	if !strings.Contains(frameS, "ACTIVITY") {
		return fail("click S: the activity tab did not open on a floor click")
	}
	if !strings.Contains(frameS, "tekton-1") {
		return fail("click S: activity log shows no tekton-1 entries")
	}
	// the notice lives in chat history; the marker on the agents tab
	if !d.m.SelectTab("agents") {
		return fail("click S: agents tab not selectable")
	}
	frameSA := d.m.Frame()
	if !strings.Contains(ansi.Strip(frameSA), "▸ tekton-1") {
		return fail("click S: agents tab missing the ▸ selection marker on tekton-1")
	}
	if !d.m.SelectTab("chat") {
		return fail("click S: chat tab not selectable")
	}
	frameSC := d.m.Frame()
	if !strings.Contains(frameSC, "tekton-1 selected") {
		return fail("click S: office notice \"tekton-1 selected\" missing from chat")
	}
	fmt.Println("--- agents tab (▸ marker) + chat notice verified ---")

	// frame chrome: clicks on the topbar/statusbar rows do NOTHING
	clickAt(d, 40, 0)
	clickAt(d, 40, shotRows-1)
	if strings.Contains(d.m.Frame(), "boss selected") || strings.Contains(d.m.Frame(), "▸ boss") {
		return fail("click chrome: a topbar/statusbar click leaked into a selection")
	}

	// tekton-1 returns; its thread auto-collapses
	d.send(state.Event{Kind: state.EvReturned, EmployeeID: "dev-1", TaskID: "t1",
		Mail: mail("m1", "tekton-1", "boss", "return: sse stream", "stream is live.", state.MailReturn)})
	d.pump(1)
	if !strings.Contains(d.m.Frame(), "Developer Task — Wire the SSE stream (· 2 tool calls ✓ done)") {
		return fail("click D setup: tekton-1 thread did not collapse after EvReturned")
	}
	time.Sleep(clickPairGap + 100*time.Millisecond) // out of the double-click window

	// (D) double-click the sprite → thread expansion toggles, chat opens
	clickFloor("dev-1")
	clickFloor("dev-1")
	fmt.Println("===== UI SHOT · CLICK D — double-click tekton-1: its collapsed thread re-expands =====")
	frameD := d.m.Frame()
	fmt.Println(frameD)
	fmt.Println("===== UI SHOT =====")
	if !strings.Contains(frameD, "Developer Task — Wire the SSE stream") {
		return fail("click D: double-click did not re-expand tekton-1's thread (chat should show its header line)")
	}
	if !strings.Contains(frameD, "  [tool] Read internal/room/manager.go ✓") {
		return fail("click D: expanded thread missing its tool rows")
	}
	if d.m.ActiveTabIndex() != 0 {
		return fail("click D: double-click must jump to the chat tab")
	}

	// (H) click the skopos-1 header row in chat → its thread collapses
	// (find the header's actual screen row in the rendered frame)
	_, _, _, floorW := d.m.LayoutInfo()
	headerY := -1
	for i, ln := range strings.Split(frameD, "\n") {
		if strings.Contains(ln, "Explore Task — Scan the repo") {
			headerY = i
			break
		}
	}
	if headerY < 0 {
		return fail("click H setup: skopos-1 header row not found in the frame")
	}
	clickAt(d, floorW+5, headerY)
	fmt.Println("===== UI SHOT · CLICK H — chat click on the skopos-1 header: its thread collapses =====")
	frameH := d.m.Frame()
	fmt.Println(frameH)
	fmt.Println("===== UI SHOT =====")
	// threads collapse by default (live too), so the header click EXPANDS —
	// the toggle round-trips back to collapsed on the next click.
	if !strings.Contains(frameH, "  [tool] Grep SSE, 12 hits ✓") {
		return fail("click H: header click did not expand skopos-1's collapsed thread")
	}
	headerY2 := -1
	for i, ln := range strings.Split(frameH, "\n") {
		if strings.Contains(ln, "Explore Task — Scan the repo") {
			headerY2 = i
			break
		}
	}
	if headerY2 < 0 {
		return fail("click H: expanded header row not found in the frame")
	}
	clickAt(d, floorW+5, headerY2)
	frameH2 := d.m.Frame()
	// skopos-1 is still LIVE here, so its re-collapsed header carries NO
	// rollup (only settled threads trail "(· N tool calls ✓ done)") — the
	// collapse proof is header still present + tool rows gone.
	if !strings.Contains(frameH2, "Explore Task — Scan the repo") {
		return fail("click H: second header click did not re-collapse skopos-1's thread")
	}
	if strings.Contains(frameH2, "  [tool] Grep SSE, 12 hits ✓") {
		return fail("click H: tool rows still expanded after the second header click")
	}
	fmt.Println("asserts: OK — floor click selects (activity tab + ▸ marker + office notice), double-click toggles the thread + jumps to chat, thread-header clicks toggle round-trip, chrome rows ignore clicks")
	return nil
}

// --- tool-output expansion (--tooloutput) ------------------------------------
// Scripted bubbletea v2 mouse clicks through the REAL model: a boss tool
// call's row toggles its captured ToolOutput body (chat_toolrow.go):
// (A) a RUNNING call expands to the pinned "no output as such" empty
// state; (B) the done event's ToolOutput updates the SAME expanded body
// in place (applyEventCore's SetToolOutput feed — no collapse flicker);
// (C) a second, output-LESS call expands independently to its own empty
// state; (D) the first call folds back while the second stays open.
// Two synchronous drives must produce byte-identical frames.

func runToolOutputProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }
	type shot struct {
		label string
		frame string
	}
	drive := func() ([]shot, error) {
		d := newFocusDriver()
		d.send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] demo — tooloutput stub online"})
		d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("u1", "user", "build it, then read x.go", false)})
		d.send(focusTool("", "", "c-1", "bash", "go build ./...", "running"))
		d.send(focusTool("", "", "c-2", "read", "x.go", "done"))
		d.pump(2)
		var shots []shot

		// every click lands on the row's CURRENT screen position (the
		// chat sidebar is the right-hand panel — floorW+5 is inside it)
		_, _, _, floorW := d.m.LayoutInfo()
		clickRow := func(frame, marker string) error {
			y := -1
			for i, ln := range strings.Split(frame, "\n") {
				if strings.Contains(ansi.Strip(ln), marker) {
					y = i
					break
				}
			}
			if y < 0 {
				return fail("tooloutput: row %q not found in the frame", marker)
			}
			clickAt(d, floorW+5, y)
			return nil
		}

		f0 := d.m.Frame()
		for _, want := range []string{"[tool] ▸ bash · go build ./... … running", "[tool] ▸ read · x.go ✓"} {
			if !strings.Contains(ansi.Strip(f0), want) {
				return nil, fail("tooloutput 0 (collapsed): frame missing %q", want)
			}
		}
		shots = append(shots, shot{"0 — two collapsed tool rows (▸ chevrons, running + done)", f0})

		// (A) click the RUNNING row: the pinned empty state opens under it
		if err := clickRow(f0, "[tool] ▸ bash · go build ./..."); err != nil {
			return nil, err
		}
		fA := d.m.Frame()
		for _, want := range []string{"[tool] ▾ bash · go build ./... … running", "no output as such"} {
			if !strings.Contains(ansi.Strip(fA), want) {
				return nil, fail("tooloutput A (running expand): frame missing %q", want)
			}
		}
		shots = append(shots, shot{"A — clicked the running call: ▾ + the pinned \"no output as such\" empty state", fA})

		// (B) the done event lands WITH output: the SAME expanded body
		// updates in place (the row never re-collapses)
		done := focusTool("", "", "c-1", "bash", "go build ./...", "done")
		done.ToolOutput = "compiling internal/app/model.go\nlinking theboringoffice\nbuild ok in 412ms"
		d.send(done)
		fB := d.m.Frame()
		stripB := ansi.Strip(fB)
		for _, want := range []string{"[tool] ▾ bash · go build ./... ✓", "compiling internal/app/model.go", "linking theboringoffice", "build ok in 412ms"} {
			if !strings.Contains(stripB, want) {
				return nil, fail("tooloutput B (done in place): frame missing %q", want)
			}
		}
		if strings.Contains(stripB, "no output as such") {
			return nil, fail("tooloutput B: the empty state must be gone once the output landed")
		}
		shots = append(shots, shot{"B — done event WITH output: the expanded body updates in place (▾ kept, empty state gone)", fB})

		// (C) the output-LESS second call expands independently
		if err := clickRow(fB, "[tool] ▸ read · x.go"); err != nil {
			return nil, err
		}
		fC := d.m.Frame()
		stripC := ansi.Strip(fC)
		for _, want := range []string{"[tool] ▾ read · x.go ✓", "no output as such", "build ok in 412ms"} {
			if !strings.Contains(stripC, want) {
				return nil, fail("tooloutput C (independent expand): frame missing %q", want)
			}
		}
		shots = append(shots, shot{"C — the output-less read expands too: ITS empty state beside bash's captured body", fC})

		// (D) the first row folds back; the second stays expanded
		if err := clickRow(fC, "[tool] ▾ bash · go build ./..."); err != nil {
			return nil, err
		}
		fD := d.m.Frame()
		stripD := ansi.Strip(fD)
		if !strings.Contains(stripD, "[tool] ▸ bash · go build ./... ✓") {
			return nil, fail("tooloutput D: the second click must fold bash back to ▸")
		}
		if strings.Contains(stripD, "build ok in 412ms") {
			return nil, fail("tooloutput D: bash's body must hide after the fold")
		}
		for _, want := range []string{"[tool] ▾ read · x.go ✓", "no output as such"} {
			if !strings.Contains(stripD, want) {
				return nil, fail("tooloutput D (sibling survives): frame missing %q", want)
			}
		}
		shots = append(shots, shot{"D — bash folds (▸, body gone); read stays expanded (rows independent)", fD})
		return shots, nil
	}

	shotsA, err := drive()
	if err != nil {
		return err
	}
	shotsB, err := drive()
	if err != nil {
		return err
	}
	for i := range shotsA {
		fmt.Printf("===== UI SHOT · --tooloutput %s =====\n", shotsA[i].label)
		fmt.Println(shotsA[i].frame)
		fmt.Println("===== UI SHOT =====")
		if shotsA[i].frame != shotsB[i].frame {
			return fail("tooloutput: leg %d differs between the two synchronous drives", i)
		}
	}
	fmt.Println("deterministic: OK — two synchronous drives produced byte-identical frames")
	fmt.Println("asserts: OK — tool rows wear ▸/▾ chevrons; a click toggles the captured ToolOutput body (wrapped, dim, under the row); a RUNNING call expands to the pinned \"no output as such\" empty state; the done event's ToolOutput updates the SAME expanded body in place (applyEventCore's SetToolOutput feed); an output-less call expands independently; rows fold independently; two drives byte-identical")
	return nil
}

// --- theme/frame matrix (--theme-matrix) ------------------------------------
// Four synchronous, populated frames pin the app-level PanelBg wrapper:
// paper/noir × desktop/mobile. Each frame expands a real tool-output row,
// then raises a real permission modal over the same panel. The checks stay at
// the public frame boundary: terminal geometry and the emitted PanelBg ANSI
// code, rather than inspecting lipgloss internals.
func runThemeMatrixProof() error {
	legs := []struct {
		theme, layout, panelCode string
		width                    int
	}{
		{"paper", "desktop", "48;2;240;241;244", shotCols},
		{"paper", "mobile", "48;2;240;241;244", 70},
		{"noir", "desktop", "48;2;22;22;25", shotCols},
		{"noir", "mobile", "48;2;22;22;25", 70},
	}
	for _, leg := range legs {
		if !chrome.SetTheme(leg.theme) {
			return fmt.Errorf("theme matrix: SetTheme(%q) failed", leg.theme)
		}
		d := &focusDriver{m: app.New(&stubBackend{done: make(chan struct{})}, config.Default())}
		d.send(tea.WindowSizeMsg{Width: leg.width, Height: shotRows})
		d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("theme-user", "user", "show the themed panel", false)})
		d.send(focusTool("", "", "theme-tool", "read", "theme.go", "done"))
		d.send(state.Event{Kind: state.EvTool, EmployeeName: "boss", CallID: "theme-tool", ToolName: "read", ToolSummary: "theme.go", ToolState: "done", ToolOutput: "panel paint verified"})

		collapsed := d.m.Frame()
		needle := "[tool] ▸ read · theme.go ✓"
		y := -1
		for i, line := range strings.Split(collapsed, "\n") {
			if strings.Contains(ansi.Strip(line), needle) {
				y = i
				break
			}
		}
		if y < 0 {
			return fmt.Errorf("theme matrix %s %s: collapsed tool row missing", leg.theme, leg.layout)
		}
		_, _, _, floorW := d.m.LayoutInfo()
		x := 5
		if leg.layout == "desktop" {
			x = floorW + 5
		}
		clickAt(d, x, y)
		// The permission modal is deliberately applied after expansion: the
		// output remains in the populated underlay while the modal proves the
		// wrapper survives an app-level overlay.
		d.send(state.Event{Kind: state.EvPermission, EmployeeName: "boss", PermissionID: "theme-perm", ToolName: "write", ToolSummary: "theme.go"})
		frame := d.m.Frame()
		if err := assertThemeMatrixFrame(leg.theme+" "+leg.layout, frame, leg.width, leg.panelCode); err != nil {
			return err
		}
		if !ansiContains(frame, "PERMISSION") || !ansiContains(frame, "show the themed panel") {
			return fmt.Errorf("theme matrix %s %s: populated panel or permission modal missing", leg.theme, leg.layout)
		}
		fmt.Printf("===== UI SHOT · theme matrix %s %s (expanded tool under permission modal) =====\n", leg.theme, leg.layout)
		fmt.Println(frame)
		fmt.Println("===== UI SHOT =====")
		fmt.Printf("assert: %s %s cols=%d PanelBg=%s tool=expanded modal=permission\n", leg.theme, leg.layout, leg.width, leg.panelCode)
	}
	fmt.Println("asserts: OK — paper/noir × desktop/mobile frames retain exact widths, emitted PanelBg codes, populated chat/tool underlays, and the permission modal")
	return nil
}

func assertThemeMatrixFrame(tag, frame string, width int, panelCode string) error {
	if !strings.Contains(frame, panelCode) {
		return fmt.Errorf("theme matrix %s: PanelBg ANSI code %q missing", tag, panelCode)
	}
	for i, line := range strings.Split(frame, "\n") {
		if got := ansi.StringWidth(line); got != width {
			return fmt.Errorf("theme matrix %s: row %d width=%d want=%d", tag, i, got, width)
		}
	}
	return nil
}

func main() {
	tab := flag.String("tab", defaultTab, "active tab: chat|terminal|agents|board|mail|activity")
	theme := flag.String("theme", "", "force a ui theme: "+strings.Join(chrome.ThemeNames(), "|"))
	themeMatrix := flag.Bool("theme-matrix", false, "theme/frame proof: paper|noir × desktop|mobile with expanded tool output and permission modal; asserts widths + PanelBg ANSI codes")
	slash := flag.Bool("slash", false, "simulate typing /theme dracula + /themes (exercises slash dispatch + theme persist)")
	perm := flag.Bool("perm", false, "auto-answer the boss permission prompt with 'once' at 3s (open → answered)")
	diffs := flag.Bool("diffs", false, "press ctrl+d to expand all diff entries")
	debug := flag.Bool("debug", false, "queue flush proof: resolves the pending boss so the queue drains; prints [queue] trace lines")
	think := flag.Bool("think", false, "think-stream proof: one CallID streamed in accumulated updates, prints BOTH frames (t=2.0s mid-stream expanded, t=3.2s collapsed after Done)")
	thinkStop := flag.String("think-stop", "", "with --think: print ONE frame only (mid = t=2.0s streaming, done = t=3.2s collapsed) for the gallery shot")
	stream := flag.Bool("stream", false, "chat-stream proof: one \"bossmsg-m1\" reply streamed as 5 accumulated pending updates then pinned; prints frame mid-stream (bubble growing in the viewport, typing row live below the divider) and after done (single settled bubble) plus the Send/done ordering trace — the message typed mid-stream FREE-SENDS straight to the backend now (the busy compose \"busy · 1 queued (server)\" paints until the pin)")
	askAnswer := flag.Bool("ask-answer", false, "question-hold proof: boss EvQuestion opens the answer modal (typing placeholder removed, park status line); typing + enter routes through AnswerQuestion — prints BOTH frames (modal open / after answered) + the stub capture log")
	askEsc := flag.Bool("ask-esc", false, "question-hold proof: esc defers the modal (notice), /question re-opens it, the answer still routes through AnswerQuestion")
	askQueue := flag.Bool("ask-queue", false, "queue-hold proof: a message typed while the question hold is outstanding must ENQUEUE; AnswerQuestion → resolved → completed boss reply → flush, ordering trace printed")
	batch := flag.Bool("batch", false, "intelligent-backlog proof: three messages enqueue while the boss is busy; the flush is ONE composed [BATCH DISPATCH] send (frame + batch text + stub logs + trace)")
	batchRespawn := flag.Bool("batch-respawn", false, "failure-respawn proof: the first batch Send is rejected once — the app must ResetPrimary(true) and resend the SAME batch exactly once")
	power := flag.String("power", "", "power-governor proof: 6s scripted window per mode (auto|saver|performance|all) — tick counts, floor frame-cache hit %, TickDelay table, /power + /model slash demo, custom boss-name frame")
	social := flag.Bool("social", false, "social-clock proof: synchronous tick pump — three frames (tea ask / both walking / gossip chain), banter chain trace, question-modal gate assert, tick-seeded determinism check")
	layout := flag.Bool("layout", false, "layout-modes proof: three frames over the same window — NORMAL (sidebar 80, the bcb1635 default), compact (sidebar 30, short tab labels, 2-row chat input, compressed topbar), wide 90 (explicit ui.sidebarWidth, clamped 26..100) — with computed width asserts per frame")
	planshot := flag.Bool("planshot", false, "plan-mode screenshot (conversation-first + shape gate): ctrl+p flips ONLY the mode — TWO frames: t=2.0s plan mode active after TWO boss chatter replies (status narration) — the pane stays empty+hidden (office floor kept, [plan] badge + idle hint, escape-valve note EXACTLY once); the plan-SHAPED reply (# Goal / # Steps with bullets) then mirrors passively — t=3.6s the markdown pane owns the floor slot with the boss's plan text (unique azimuth marker), floor swapped out, starter template never armed, chat input still owns focus (click-to-edit pane footer)")
	terminal := flag.Bool("terminal", false, "terminal-tab proof: the stub TermPanel wires through app.SpawnTerminal — lazy-spawn on first visit, OPT-IN capture toggle flow (released default with a real tab event leaving the tab, ctrl+space toggles BOTH ways, ctrl+o releases as the alias, auto-release on leave, re-entry released), hints + frames + asserts, byte-identical twice")
	paste := flag.Bool("paste", false, "office-wide paste proof (synchronous, tea.PasteMsg driven): chat small-paste literal + batched insert, >20-line/>2000-char paste collapses to the one-line chip (ONE backspace unit, expand-on-send — the agent gets the FULL text), shift+enter + ctrl+j newlines, RELEASED terminal → chat fallback / CAPTURED terminal → shell (stub records the content), question float's answer field takes a multi-line paste verbatim → AnswerQuestion, agents-tab paste → the dim 'paste: nothing focused accepts text' notice; two drives byte-identical")
	focus := flag.Bool("focus", false, "fix-wave proof, THREE synchronous-tick frames: (a) empty pending bubble — typing row below the divider (above the input), NO caret anywhere; (b) streaming partial bubble — text grows in the viewport while the typing row STAYS below the divider for the whole pending period (still no caret); (c) two concurrent agents — per-agent work threads grouped (headers + merged rows), boss tool line still inline, boss idle at the placeholder in delegating state (dim row in the same below-divider slot, [delegat] nameplate). Every frame: no \"▌\", every chat row inside the divider's width budget")
	persist := flag.Bool("persist", false, "office-session DEMO regression: seed a fresh session.json for cwd in a scratch THEBORINGOFFICE_HOME, run the standard demo shot, assert NO restore notice surfaces and the file is untouched (restore is live-only) — prints PERSIST-DEMO-SKIP: OK|FAIL")
	slashpop := flag.Bool("slashpop", false, "slash-popover proof: type \"/th\" → filtered menu (/theme /themes /thinking), Enter pre-fills \"/theme \" → theme picker, arrows preview LIVE (two states printed), esc cancels back, Enter commits + persists via the plain slash path")
	threadsThink := flag.Bool("threads-think", false, "employee-thinking-in-threads proof: tekton-1 EvThought merges per CallID into its work thread (collapsed rollup keeps the \"· 1 think\" count), ctrl+g expands tools + thoughts in natural order — boss path byte-identical")
	threads := flag.Bool("threads", false, "thread-render fixture (opencode renderer): ONE chat frame with BOTH thread states — a LIVE collapsed thread (animated braille glyph, NO rollup while running, bare ↳ sneak, live-only ctrl+g hint row) beside a COMPLETED collapsed thread (dim ✓ glyph, \"✓ done\" rollup) — every message reducer-shaped (Kind wtool, \"<verb> · <summary>\" text, \"<state>␟<tick>\" meta stamped by the REAL app reducer; the display layer shapes it to \"<Verb> <rest>\")")
	threadfocus := flag.Bool("threadfocus", false, "thread-focus view proof (synchronous): two threads, ctrl+f mounts tekton-1's thread fullscreen — header glyph + title + counters, FULL body rows, the ↳ diff sub-row clicked open through the REAL mouse seam, the \"esc · ctrl+f back to office\" hint bar; esc closes back to the office byte-for-byte")
	boardsync := flag.Bool("boardsync", false, "completion board-sync proof (synchronous): 3 DOING rows staged (tekton-1 ×2, skopos-1 title twin); return 1 flips tekton-1's OLDEST stranded row + ONE \"[office] board sync\" note, return 2 (tekton-2, distinct owner) flips NONE of skopos-1's — frames before/after + counts, two drives byte-identical")
	wdiff := flag.Bool("wdiff", false, "per-call thread-diff proof: a completed worker Edit's CallID-keyed EvFileDiff pins INSIDE the thread — collapsed sneak gains the dim \"· +A -D\" suffix, ctrl+g shows the tool-row suffix + the clickable \"↳ diff · path +A -D\" sub-row, a click opens/closes the parsed line-numbered body")
	click := flag.Bool("click", false, "mouse proof: scripted clicks — floor sprite click selects the agent (activity tab + ▸ marker + office notice), double-click toggles its thread + jumps to chat, chat thread-header/summary clicks toggle round-trip, chrome rows ignore clicks")
	tooloutput := flag.Bool("tooloutput", false, "tool-output expansion proof (synchronous, REAL mouse clicks): a boss tool row's ▸/▾ chevron toggles its captured ToolOutput body — a RUNNING call expands to the pinned \"no output as such\" empty state, the done event's ToolOutput updates the SAME expanded body in place, an output-less call expands independently, rows fold independently; two drives byte-identical")
	stop := flag.Bool("stop", false, "/stop proof (synchronous): boss mid-stream with tools running + a staged second placeholder + a roadblock-queued item + delegating state; typing /stop must hit stub.AbortSessions and unwind in ONE frame — \"stopped by user\" placeholders, \" (stopped)\" stream appendix, tools ✗ aborted, thread ✗ stopped, BossThinking/Delegating cleared, queue intact; a /queue leg proves the item survived unsent")
	bypass := flag.Bool("bypass", false, "/bypass session-scoped bypass-permissions proof (synchronous): /bypass opens the arming confirm (enable/cancel); enable arms the mode — the ⚠ BYPASS segment rides the topbar + the pinned ON notice; a stray EvPermission auto-answers allow-once on the stub's wire with NO modal parked + the dim auto-approved row; /bypass again disables INSTANTLY (no confirm) — OFF notice + segment gone; two drives byte-identical")
	stuck := flag.Bool("stuck", false, "boss-stuck-busy proof (synchronous): boss busy at 200ms, never completes — the W1 wedge watchdog (SetWedgeAfterForShot-seamed 30ms threshold) notes ONE \"boss turn wedged\" line in the ACTIVITY tab (zero transcript rows) + hint swap; the /stop leg then runs with AbortSessions stubbed to FAIL and the office still unwinds (placeholder collapsed, dim failure note, watchdog re-armed)")
	freesend := flag.Bool("freesend", false, "free-queuing proof: boss busy 200–3000ms; two prompts sent DURING the window must hit backend.Send IMMEDIATELY (both ([stub] Send lines precede the turn-completed marker in the ordering trace) — frame 1 (t=2.2s) shows \"busy · 2 queued (server)\" + the \"turn 2 · your message rides next\" placeholder; frame 2 (t=3.6s) shows the drained FIFO pins + restored status line")
	concierge := flag.Bool("concierge", false, "concierge routing proof (synchronous, two phases): A) boss busy mid-turn — two sends BOTH route to stub.SendConcierge (capture printed), the \"office routed: boss busy → concierge\" notice prints ONCE, office placeholders read \"office is answering…\", answers pin in place (INFO \"office ›\" bubbles), the agents roster pins \"office (concierge) answering\" → \"on call\"; B) after the boss turn completes, the next send hits the boss's Send and the concierge is NOT called (zero duplication)")
	modelshot := flag.Bool("modelshot", false, "any-model gallery shot: answers the two stacked permission asks (y·y at ~2.5s) so the modal clears, then types \"/model\" AFTER the queue typing and runs the two-press dance (first Enter applies the popover row, second SENDS) so the final frame shows the /model picker OPEN over the frame with the stub's fixed five-model listing, cursor on row 1")
	at := flag.Int("at", 0, "capture the standard-script frame at ms-from-start instead of the usual 4s — e.g. 2920 catches the permission queue modal OPEN (the always-on queue typing's enter keys answer it from ~3.06s on, so the 4s frame has already advanced past it)")
	notifications := flag.Bool("notifications", false, "OS desktop notification proof (synchronous): recording NotifyBus at the app seam — focused startup silent; blur opens the window; ONE boss-ask cohort ping (child coalesces, generic agent+tool copy); front-answer keeps the cohort silent; ONE completion ping (clipped reply); refocus silent; re-blur re-nudges; /notify off → zero captures + persisted brain.json")

	claudeMode := flag.Bool("claude", false, "claude-backend shot: the REAL live claude backend against the compiled cmd/claudestub binary (stream-json). With --planshot (or alone): the plan-mode + permission/dialog control round-trip proof — chatter never presents, plan-shaped reply presents into the pane, req-owl-1/req-q-1 round-trips byte-pinned, subagent Task run returns, two drives byte-identical")
	images := flag.Bool("images", false, "inbound boss-turn image preview proof (synchronous): the stub pins a completed boss turn carrying the 8×8 checker PNG as a data-URL file part (Meta carrier + Event.Media payload — opencode.go's pin shape); the lazy rasterize must land with the 🖼 chip + the pinned half-block truecolor rows in the frame; /images off leg proves chips-only; two drives byte-identical. The base legs run under a hermetic neutral-terminal env stub so the host's real lane never leaks in")
	laneList := flag.String("lane", "", "with --images: comma-separated native-lane legs (kitty,iterm,ascii) — each leg drives the same checker pin under a hermetic stub terminal env (TERM_PROGRAM/ITERM_SESSION_ID/KITTY_WINDOW_ID/TERM… injected, the host's ghostty/iterm markers never leak) and byte-pins the lane's output: kitty → the ESC_G a=T,t=d,f=100,i=<sha1[:8]>,q=2; placeholder strip + b64 payload + ESC\\; iterm → OSC 1337 File=inline=1;width=<cols>:height=<rows>;base64,<b64> BEL; ascii → the v1 pinned half-block rows; every leg byte-identical twice")
	imagesPTY := flag.Bool("images-pty", false, "chat image splice PRODUCTION-path leg: the REAL program (cursed renderer over the ZenbuFrameWriter on os.Stdout, main.go's exact wiring) with the checker's completed boss turn pinned through p.Send — the PTY harness (/tmp/drive_chatimage.py, ghostty env) counts ESC_G f=100 frames on the wire (pre-fix 0: the renderer eats the View-string APC; post-fix the wrapper's chat-media splice lands CUP+APC+restore after the flush)")
	links := flag.Bool("links", false, "open-in-browser proof (synchronous): a boss bubble carries a URL + a media filename pointing at the REAL checker fixture (the os.Stat gate's verified path); a press marks the bubble, `o` floats the OPEN IN BROWSER card over BOTH targets, enter fires the URL through the STUBBED panels runner; the activity tab logs \"→ opened: opencode.ai/docs\"; the no-mark leg types \"o\" into the draft; two drives byte-identical")
	openurl := flag.Bool("openurl", false, "terminal-browser candidate-lane proof (synchronous, REAL fake binaries): a scratch fixture dir plants a logging `terminal-browser` (+ `open`/`xdg-open`) on a pinned \"<fixture>:<orig>\" PATH with a hermetic ghostty env; leg A resolves terminal-browser (\"resolve=terminal-browser prefer-over-system-open\") and a press+`o` on a single-URL bubble logs exactly ONE fake call (system log absent); leg B (FAKE_TB_EXIT=1) cascades the SAME URL to the system opener — ONE attempt per leg, \"→ opened:\" intact, no \"could not open\" row; every leg byte-identical twice")
	browser := flag.Bool("browser", false, "browser tab premium-lane proofs (synchronous, REAL fake binary on a pinned PATH + hermetic ghostty env). --lane kitty (default): the CONTROLLER legs — leg A resolves the zenbu lane and EMBEDS the fake child on the real PTY seam (its bytes paint the grid; the region frame wears the \" zenbu \" badge + \"▸ zenbu terminal-browser · <url>\" strip), then Close group-kills + reaps (no leak); leg B (fake exits immediately, ~180ms < 300ms) lands the text-mode fallback — exact dim note, \" text \" badge, fixture body, strip gone, URL state intact; leg S (the kitty STREAM passthrough): the fake streams TWO CHUNKED kitty frames under the SAME child i=1 + text chrome — the lane splits the stream (text rows carry ZERO base64; the View carries ZERO APC bytes), the frame wrapper re-emits BOTH generations to the OUTER terminal after renderer flushes as cursor-save + CUP(the absolute cell) + ONE cached a=T,t=d,q=2,C=1 APC under the STABLE office id (ZenbuOfficeID(child id, placement)) carrying the pane's body box c=/r= + cursor-restore — ZERO a=d between the generations (kitty's atomic same-id replace) — and Close flushes ESC_Ga=d,d=I directly (captured through the emit seam); leg K (the MID-CHAIN DEATH): the fake dies mid-chunked-frame (chunk 2 OSC-7-interleaved, chunk 3 UNTERMINATED — the wave-82 capture's shape) — grid + scrollback carry ZERO base64, Poll latches the text fallback. --lane live: the LIVE APP-GLUE legs — \"/open file://<fixture>\" typed through the REAL chat input spawns the embed through the pane's own Open (the strip renders INSIDE the left slot, right strip unmoved, NO text-lane hint row), esc FREEZES the session (keep-alive: alive behind the floor, PID unchanged) + returns to the floor, the ctrl+c quit path reaps it; the die leg lands the text fallback through the app (the exact dim note, the warm page, the no-flap latch). --lane keepalive: the freeze/thaw flip cycle — /open → ctrl+b (floor: the child FREEZES, PID stable + alive + ps T…, ONE a=d through the wrapper's diff) → ctrl+b (the SAME pid thaws; the RETAINED frame re-emits byte-identically — the parked fake emits zero new bytes — with ZERO a=d interleaved) → ctrl+b → ctrl+c (the quit path reaps the frozen child, the delete riding the direct seam); ONE spawn total. --lane hint: the text lane's \"why\" row through the LIVE app — PATH pinned to an EMPTY fixture dir (the probe misses by construction) under the hermetic ghostty stub, so ctrl+b shows the idle starter card wearing the dim \"text lane — terminal-browser not on PATH · …\" hint under the location bar and /open keeps it pinned over the warm text page. Every leg byte-identical twice")
	browsertab := flag.Bool("browsertab", false, "browser TAB text-viewer proof on the LEFT pane's floor|browser slot (synchronous, REAL pinned-port stub server on 127.0.0.1:52731): \"/open http://…/fixture.html\" typed through the REAL chat input + slash popover flips the left slot to the browser (right strip unmoved) and renders the shared fixture as text rows — the \"▸ <url>\" bar, bold headings, the indexed link rows (\"link alpha [1]\", \"link beta [2]\", \"link gamma [3]\"), the 🖼 chip, the \" │ \" table rows — then pgdn scrolls the tail-marker row into view; two drives byte-identical")
	flag.Parse()
	if *themeMatrix {
		if err := runThemeMatrixProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *persist {
		if err := runPersistDemoSkipProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *focus {
		if err := runFocusProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *threadfocus {
		if err := runThreadFocusProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *boardsync {
		if err := runBoardSyncProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *stop {
		if err := runStopProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *bypass {
		if err := runBypassProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *stuck {
		if err := runStuckProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *concierge {
		if err := runConciergeProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *freesend {
		fail := func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "uishot: "+format+"\n", args...)
			os.Exit(1)
		}
		stops := []struct {
			at    time.Duration
			label string
		}{
			{2200 * time.Millisecond, "frame 1 — t=2.2s (BUSY: two prompts sent straight to the server mid-turn — status \"busy · 2 queued (server)\", placeholder \"turn 2 · your message rides next\")"},
			{3600 * time.Millisecond, "frame 2 — t=3.6s (DRAINED: busy turn completed, both server-queued sends pinned in place FIFO, busy compose restored)"},
		}
		var lastTrace []string
		for i, s := range stops {
			frame, trace, err := runFreeShot(s.at)
			if err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("===== UI SHOT · --freesend %s =====\n", s.label)
			fmt.Println(frame)
			fmt.Println("===== UI SHOT =====")
			lastTrace = trace
			if i == 0 {
				// the busy-frame contract: the compose + the turn text
				for _, want := range []string{"busy · 2 queued (server)", "turn 2 · your message rides next", "is typing…"} {
					if !strings.Contains(frame, want) {
						fail("freesend frame 1 missing %q", want)
					}
				}
			}
			if i == 1 {
				// the drained frame: all three replies pinned (glamour
				// wraps to the sidebar — assert the stable first-line
				// prefixes), compose gone
				for _, want := range []string{
					"first turn done — standup notes are in", "drafts.",
					"footer in — aligned with the", "header.",
					"two tests land in the next", "pass.",
					"free-queuing stub online",
				} {
					if !strings.Contains(frame, want) {
						fail("freesend frame 2 missing %q", want)
					}
				}
				if strings.Contains(frame, "busy · 2 queued (server)") {
					fail("freesend frame 2 still shows the busy compose after the drain")
				}
			}
		}
		fmt.Println("--- ordering trace (frame-2 run, covers the drain) ---")
		for _, ln := range lastTrace {
			fmt.Println(ln)
		}
		// the anti-stuck contract: BOTH Send() calls land immediately —
		// before the busy turn completes (nothing hides in a client queue).
		sendIdx, doneIdx, enqIdx := 0, -1, -1
		for i, ln := range lastTrace {
			if strings.Contains(ln, "[stub] Send(") {
				sendIdx++
			}
			if strings.Contains(ln, "turn completed") && doneIdx < 0 {
				doneIdx = i
			}
			if strings.Contains(ln, "enqueued") && enqIdx < 0 {
				enqIdx = i
			}
		}
		if sendIdx != 2 {
			fail("expected exactly 2 immediate Send calls during the busy window, got %d", sendIdx)
		}
		if enqIdx >= 0 {
			fail("a prompt ENQUEUED during the busy window (line %d) — free-queuing must send direct", enqIdx)
		}
		if doneIdx < 0 {
			fail("trace missing the turn-completed marker")
		}
		fmt.Println("asserts: OK — both Send() calls immediate (before \"turn completed\"), zero client-queue enqueues, busy compose + \"turn 2 · your message rides next\" live mid-turn, FIFO pins drained in place, status line restored")
		return
	}

	if *slashpop {
		if err := runSlashPopProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *threadsThink {
		if err := runThreadsThinkProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *threads {
		if err := runThreadsProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *wdiff {
		if err := runWdiffProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *click {
		if err := runClickProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *tooloutput {
		if err := runToolOutputProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *layout {
		if err := runLayoutProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *claudeMode {
		if err := runClaudePlanProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *images {
		if err := runImagesProof(strings.Split(*laneList, ",")); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *imagesPTY {
		if err := runImagesPTY(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *links {
		if err := runLinksProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *openurl {
		if err := runOpenURLProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *browser {
		switch lane := strings.TrimSpace(*laneList); lane {
		case "", "kitty":
			if err := runBrowserLaneProof(); err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
		case "live":
			if err := runBrowserLiveProof(); err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
		case "keepalive":
			if err := runBrowserKeepaliveProof(); err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
		case "hint":
			if err := runBrowserHintProof(); err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
		case "shot":
			if err := runBrowserShotProof(); err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "uishot: --browser supports --lane kitty (the controller), --lane live (the LIVE app-glue wiring), --lane keepalive (the freeze/thaw flip cycle), --lane hint (the text-lane why row), or --lane shot (the headless screenshot lane), got %q\n", lane)
			os.Exit(1)
		}
		return
	}

	if *browsertab {
		if err := runBrowserTabProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *planshot {
		if err := runPlanProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *terminal {
		if err := runTerminalProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *paste {
		if err := runPasteProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *social {
		if err := runSocialProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *power != "" {
		if err := runPowerProof(*power); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *notifications {
		if err := runNotificationsProof(); err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// keystroke workloads only reach the textarea / modal on the chat tab
	if (*slash || *perm || *diffs || *debug || *think || *askAnswer || *askEsc || *askQueue || *modelshot) && *tab == defaultTab {
		*tab = "chat"
	}

	if *theme != "" {
		if !chrome.SetTheme(*theme) {
			fmt.Fprintf(os.Stderr, "uishot: unknown theme %q (%s)\n", *theme,
				strings.Join(chrome.ThemeNames(), ", "))
			os.Exit(2)
		}
	}

	if *batch || *batchRespawn {
		frame, trace, sends, team, err := runBatchShot(*batchRespawn, 5600*time.Millisecond)
		if err != nil {
			fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
			os.Exit(1)
		}
		label := "--batch (flush = ONE composed batch send)"
		if *batchRespawn {
			label = "--batch-respawn (first Send rejected → ResetPrimary + resend once)"
		}
		fmt.Printf("===== UI SHOT · %s =====\n", label)
		fmt.Println(frame)
		fmt.Println("===== UI SHOT =====")
		fmt.Println("--- ordering trace ---")
		for _, ln := range trace {
			fmt.Println(ln)
		}
		fmt.Println("--- stub Send calls ---")
		for i, s := range sends {
			fmt.Printf("Send call %d:\n%s\n", i+1, s)
		}
		fmt.Println("--- team seam log ---")
		for _, ln := range team {
			fmt.Println(ln)
		}
		// asserts — the intelligent-backlog contract
		fail := func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "uishot: "+format+"\n", args...)
			os.Exit(1)
		}
		wantSends := 1
		if *batchRespawn {
			wantSends = 2
		}
		if len(sends) != wantSends {
			fail("expected exactly %d Send call(s), got %d", wantSends, len(sends))
		}
		if !strings.HasPrefix(sends[0], "[BATCH DISPATCH — 3 requests arrived") {
			fail("expected ONE batch-composed send of 3 items, got: %q", sends[0])
		}
		for _, item := range []string{"1. fix the badge", "2. ship v2", "3. write the release notes"} {
			if !strings.Contains(sends[0], item) {
				fail("composed batch missing numbered item %q", item)
			}
		}
		if *batchRespawn {
			if sends[1] != sends[0] {
				fail("respawned batch differs from the original")
			}
			hasReset := false
			for _, ln := range team {
				if ln == "ResetPrimary(true)" {
					hasReset = true
				}
			}
			if !hasReset {
				fail("expected ResetPrimary(true) after the rejected batch send")
			}
		}
		for _, id := range []string{"demo-1", "demo-2", "demo-3"} {
			if !strings.Contains(" "+strings.Join(team, " ")+" ", "QueueItemDone("+id+")") {
				fail("expected QueueItemDone(%s) after the batch turn completed", id)
			}
		}
		fmt.Println("asserts: OK — ONE composed batch send, board rows started/done, no second send until flush")
		return
	}

	if *askAnswer || *askEsc || *askQueue {
		mode, stop2, label2 := "answer", 3600*time.Millisecond,
			"frame 2 — t=3.6s (AFTER ANSWER: modal closed, dim ✓ answered on the entry, resumed boss reply)"
		switch {
		case *askEsc:
			mode, stop2, label2 = "esc", 4300*time.Millisecond,
				"frame 2 — t=4.3s (deferred notice → /question reopen → ✓ answered)"
		case *askQueue:
			mode, stop2, label2 = "queue", 4700*time.Millisecond,
				"frame 2 — t=4.7s (queued line held through the hold, flushed after the resumed reply)"
		}
		stops := []struct {
			at    time.Duration
			label string
		}{
			{2200 * time.Millisecond, "frame 1 — t=2.2s (MODAL OPEN: boss asks, typing placeholder REMOVED — parked, not typing)"},
			{stop2, label2},
		}
		var trace, capture []string
		for i, s := range stops {
			frame, t, c, err := runAskShot(mode, s.at)
			if err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("===== UI SHOT · %s =====\n", s.label)
			fmt.Println(frame)
			fmt.Println("===== UI SHOT =====")
			if i == len(stops)-1 {
				trace, capture = t, c
			}
		}
		if len(trace) > 0 {
			fmt.Println("--- ordering trace ---")
			for _, ln := range trace {
				fmt.Println(ln)
			}
		}
		printAskCapture(capture, "AnswerQuestion(q-1, [the toggle one])")
		return
	}

	if *think {
		type stop struct {
			at    time.Duration
			label string
		}
		stops := []stop{
			{2000 * time.Millisecond, "frame 1 — t=2.0s (mid-stream, EXPANDED)"},
			{3200 * time.Millisecond, "frame 2 — t=3.2s (collapsed after Done)"},
		}
		switch *thinkStop {
		case "mid":
			stops = stops[:1]
		case "done":
			stops = stops[1:]
		case "":
		default:
			fmt.Fprintf(os.Stderr, "uishot: --think-stop must be mid|done, got %q\n", *thinkStop)
			os.Exit(2)
		}
		for _, s := range stops {
			frame, err := runThinkShot(*tab, s.at)
			if err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
			if *thinkStop == "" {
				fmt.Printf("===== UI SHOT · %s =====\n", s.label)
			} else {
				fmt.Println("===== UI SHOT =====")
			}
			fmt.Println(frame)
			fmt.Println("===== UI SHOT =====")
		}
		return
	}

	if *stream {
		stops := []struct {
			at    time.Duration
			label string
		}{
			{1250 * time.Millisecond, "frame 1 — t=1.25s (MID-STREAM: grown bubble, typing row below the divider, one message free-sent straight to the server — \"busy · 1 queued (server)\")"},
			{2800 * time.Millisecond, "frame 2 — t=2.8s (AFTER DONE: one settled bubble — replace-in-place, no dup; the free-sent reply landed as its own bubble; busy compose restored)"},
		}
		for _, s := range stops {
			frame, trace, err := runStreamShot(s.at)
			if err != nil {
				fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("===== UI SHOT · %s =====\n", s.label)
			fmt.Println(frame)
			fmt.Println("===== UI SHOT =====")
			fmt.Println("--- ordering trace ---")
			for _, ln := range trace {
				fmt.Println(ln)
			}
		}
		return
	}

	if *debug {
		app.QueueDebugf = func(format string, args ...any) {
			fmt.Printf("[queue] "+format+"\n", args...)
		}
	}

	backend := &stubBackend{done: make(chan struct{}), flushQueue: *debug}
	m := app.New(backend, config.Default())
	if !m.SelectTab(*tab) {
		fmt.Fprintf(os.Stderr, "uishot: unknown tab %q\n", *tab)
		os.Exit(2)
	}

	p := tea.NewProgram(m,
		tea.WithWindowSize(shotCols, shotRows),
		tea.WithoutRenderer(), // no redraw loop; we print the final frame ourselves
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
	)
	// backend events flow in as tea.Msgs through Program.Send (state.Event
	// satisfies the empty tea.Msg/uv.Event interface).
	emit := func(ev state.Event) { p.Send(ev) }
	_ = backend.Start(emit)
	if *slash {
		go slashWorkload(p)
	}
	if *perm {
		go permWorkload(p)
	}
	if *diffs {
		go diffsWorkload(p)
	}
	if *modelshot {
		go modelshotWorkload(p)
	}
	// queue typing always runs — on the agents tab the keys are absorbed by
	// the (non-text) panel; on chat they FREE-SEND (boss busy at 3050ms,
	// no roadblock) — the busy compose paints on the status line.
	go queueWorkload(p)
	go func() {
		if *at > 0 {
			// early capture: the rest of the script keeps firing into a
			// dying program (sends on a dead input channel are dropped)
			time.Sleep(time.Duration(*at) * time.Millisecond)
		} else if *debug {
			time.Sleep(shotDurLong)
		} else {
			time.Sleep(shotDur)
		}
		p.Quit()
	}()

	final, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "uishot: %v\n", err)
		os.Exit(1)
	}
	fm, ok := final.(app.Model)
	if !ok {
		fmt.Fprintf(os.Stderr, "uishot: unexpected final model type %T\n", final)
		os.Exit(1)
	}

	fmt.Println("===== UI SHOT =====")
	fmt.Println(fm.Frame())
	fmt.Println("===== UI SHOT =====")

	if *slash {
		// persist proof: the /theme slash run must have written the file
		path := chrome.ThemeConfigPath()
		content, rerr := os.ReadFile(path)
		fmt.Printf("theme file: %s\n", path)
		if rerr != nil {
			fmt.Printf("theme file content: <error: %v>\n", rerr)
			os.Exit(1)
		}
		fmt.Printf("theme file content: %q\n", strings.TrimSpace(string(content)))
	}
}

// --- OS desktop notification proof (--notifications) ------------------------
// Synchronous driver (no wall clock): a recording NotifyBus sits at the
// app's seam and every leg counts its taps.
//  1. focused startup: a full send + ask + completion cycle mints ZERO taps
//     (the focus latch defaults true — unsupported terminals never
//     false-ping) and the un-blurred completion CONSUMES the turn's arm.
//  2. a fresh send ARMS the done debounce (popover closed — below the open
//     popover Enter confirms its answer instead of sending).
//  3. tea.BlurMsg on an EMPTY cohort: silent.
//  4. the boss's EvPermission pings the cohort opener ONCE (generic
//     "permission needed — <agent> needs <tool>" copy — no ToolSummary leak).
//  5. the child ask behind it coalesces into the SAME cohort (still one tap).
//  6. the click-burst: y answers the front, the wire "resolved" follows, a
//     THIRD ask rides the same cohort — still ONE cohort ping total.
//  7. the armed turn's boss completion (still blurred) pings ONCE with the
//     clipped reply ("the boss is done — …").
//  8. FocusMsg → BlurMsg: a RE-blur on the still-live cohort re-nudges at
//     once, quoting the CURRENT front (the child ask).
//  9. refocused silence: typing through the popover's Enters answers both
//     stacked asks (the cohort empties), the send arms, and the refocused
//     completion adds NOTHING.
// 10. the /notify legs: bare /notify (two-press popover dance) reports the
//     mode; "/notify off" flips config + live bus mode and persists.
// 11. the full hook sweep after off (fresh 0→1 cohort + blur + armed
//     completion) dies at the config gate — zero further captures — and
//     brain.json carries the persisted write.

// notifyTaps — the --notifications proof's recording NotifyBus.
type notifyTaps struct {
	taps  []string
	modes []string
}

func (b *notifyTaps) Notify(kind, title, body string) {
	b.taps = append(b.taps, kind+" | "+title+" | "+body)
}

// SetMode — the /notify live-toggle seam (the app's type-assert finds it).
func (b *notifyTaps) SetMode(mode string) { b.modes = append(b.modes, mode) }

func runNotificationsProof() error {
	fail := func(format string, args ...any) error { return fmt.Errorf(format, args...) }

	// brain.json write-through lands in a scratch THEBORINGOFFICE_HOME —
	// the user's real config is never touched by shots (runPowerProof's
	// shape).
	home, err := os.MkdirTemp("", "theboringoffice-notify")
	if err != nil {
		return err
	}
	defer os.RemoveAll(home)
	if err := os.Setenv("THEBORINGOFFICE_HOME", home); err != nil {
		return err
	}
	fmt.Printf("--- scratch THEBORINGOFFICE_HOME: %s ---\n", home)

	stub := &stubBackend{done: make(chan struct{})}
	m := app.New(stub, config.Default())
	bus := &notifyTaps{}
	m.SetNotifyBus(bus)
	d := &focusDriver{m: m}
	d.send(tea.WindowSizeMsg{Width: shotCols, Height: shotRows})
	typeIn := func(s string) {
		for _, r := range s {
			d.send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		}
	}
	key := func(code rune) tea.Cmd {
		tm, c := d.m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
		if fm, ok := tm.(app.Model); ok {
			d.m = fm
		}
		return c
	}
	wantTaps := func(n int, leg string) error {
		if len(bus.taps) != n {
			return fail("%s: expected %d capture(s), got %d (%v)", leg, n, len(bus.taps), bus.taps)
		}
		return nil
	}

	// leg 1 — t=0: FOCUSED startup (default true). Send → ask → completion
	// stays silent; the un-blurred completion CONSUMES the turn's done arm
	// (the ping exists for completed turns you couldn't watch).
	typeIn("wire the notifier")
	drainCmd(d, key(tea.KeyEnter), 0)
	d.send(state.Event{Kind: state.EvPermission, EmployeeName: "boss", PermissionID: "perm-0",
		ToolName: "read", ToolSummary: "a.go", ToolState: "pending"})
	d.send(state.Event{Kind: state.EvPermission, PermissionID: "perm-0", ToolState: "resolved"})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-0", "boss", "silent startup turn", false)})
	if err := wantTaps(0, "leg 1 (focused startup)"); err != nil {
		return err
	}

	// leg 2 — t≈1800: a fresh send ARMS the done debounce (the permission
	// popover is CLOSED here, so the Enter really sends — below the open
	// popover Enter would confirm its answer instead).
	typeIn("ship it") // no y/a/n letters — the popover owns them below
	drainCmd(d, key(tea.KeyEnter), 0)

	// leg 3 — t≈2300: the member looks away. Blur on an EMPTY cohort is
	// silent (no post-hoc noise for long-resolved asks).
	d.send(tea.BlurMsg{})
	if err := wantTaps(0, "leg 3 (blur on an empty cohort)"); err != nil {
		return err
	}

	// leg 4 — t≈2400: the boss's ask flips the cohort 0→1: ONE ping, generic
	// copy (agent + tool NAME — the ToolSummary path never leaves the glass).
	d.send(state.Event{Kind: state.EvPermission, EmployeeName: "boss", PermissionID: "perm-1",
		ToolName: "write", ToolSummary: "main.go", ToolState: "pending"})
	wantTap0 := "permission | theboringoffice | permission needed — boss needs write"
	if err := wantTaps(1, "leg 4 (boss ask opens the cohort)"); err != nil {
		return err
	}
	if bus.taps[0] != wantTap0 {
		return fail("leg 4: cohort ping copy\ngot  %q\nwant %q", bus.taps[0], wantTap0)
	}

	// leg 5 — t≈2450: the child's ask stacks behind → the cohort coalesces.
	d.send(state.Event{Kind: state.EvPermission, EmployeeName: "tekton-1", PermissionID: "perm-2",
		ToolName: "read", ToolSummary: "srv/x.go", ToolState: "pending"})
	if err := wantTaps(1, "leg 5 (child ask coalesces)"); err != nil {
		return err
	}

	// leg 6 — t≈2500: the click-burst. y (Allow once) releases the front;
	// the wire "resolved" follows; a THIRD ask stacks onto the same cohort.
	// The cohort shrinks but NEVER empties — still ONE cohort ping total.
	drainCmd(d, key('y'), 0)
	d.send(state.Event{Kind: state.EvPermission, PermissionID: "perm-1", ToolState: "resolved"})
	d.send(state.Event{Kind: state.EvPermission, EmployeeName: "skopos-1", PermissionID: "perm-3",
		ToolName: "grep", ToolSummary: "SSE", ToolState: "pending"})
	if err := wantTaps(1, "leg 6 (front answer + resolved + later ask)"); err != nil {
		return err
	}

	// leg 7 — t≈3000: the leg-2-armed turn completes (still blurred) → ONE
	// done ping, the reply clipped to one line.
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-1", "boss",
		"banners landed next to the bell — notifier wired", false)})
	wantTap1 := "done | theboringoffice | the boss is done — banners landed next to the bell — notifier wired"
	if err := wantTaps(2, "leg 7 (boss completion)"); err != nil {
		return err
	}
	if bus.taps[1] != wantTap1 {
		return fail("leg 7: done ping copy\ngot  %q\nwant %q", bus.taps[1], wantTap1)
	}

	// leg 8 — t≈3300: refocus, then RE-BLUR on the still-live cohort (perm-2
	// + perm-3 hang on) → the cohort's own ping fires at once, quoting the
	// CURRENT front (the child ask that advanced to the top).
	d.send(tea.FocusMsg{})
	d.send(tea.BlurMsg{})
	wantTap2 := "permission | theboringoffice | permission needed — tekton-1 needs read"
	if err := wantTaps(3, "leg 8 (re-blur re-nudges the live cohort front)"); err != nil {
		return err
	}
	if bus.taps[2] != wantTap2 {
		return fail("leg 8: re-blur ping copy\ngot  %q\nwant %q", bus.taps[2], wantTap2)
	}

	// leg 9 — t≈3600: the refocused-silence leg. The typed text carries no
	// y/a/n letters; below the open popover each Enter pops the front ask
	// ("Allow once" apiece — the cohort empties, re-arming the next), so the
	// third Enter really sends. The refocused completion then adds NOTHING.
	d.send(tea.FocusMsg{})
	typeIn("focus ok")
	drainCmd(d, key(tea.KeyEnter), 0) // answers perm-2 (popover front)
	drainCmd(d, key(tea.KeyEnter), 0) // answers perm-3 (advanced front)
	drainCmd(d, key(tea.KeyEnter), 0) // popover closed: the send lands
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-2", "boss", "focused turn done", false)})
	if err := wantTaps(3, "leg 9 (refocused completion)"); err != nil {
		return err
	}

	// leg 10 — t≈3900: the /notify legs (the cohort drained in leg 9 — the
	// popover is closed). Bare /notify rides the two-press dance (first
	// Enter applies the popover row into the draft, the second sends the
	// bare command) and reports the mode in-frame; "/notify off" types
	// THROUGH the popover-closing space, one Enter.
	typeIn("/notify")
	drainCmd(d, key(tea.KeyEnter), 0) // apply "› /notify "
	drainCmd(d, key(tea.KeyEnter), 0) // send the bare command → the status notice
	statusFrame := d.m.Frame()
	if !strings.Contains(statusFrame, "notifications on") {
		return fail("leg 10: bare /notify frame missing the status line \"notifications on\":\n%s", statusFrame)
	}
	typeIn("/notify off")
	drainCmd(d, key(tea.KeyEnter), 0)
	if got := d.m.Config().UI.Notifications; got != "off" {
		return fail("leg 10: /notify off must flip cfg.UI.Notifications, got %q", got)
	}
	if len(bus.modes) != 1 || bus.modes[0] != "off" {
		return fail("leg 10: /notify off must live-set the bus mode, got %v", bus.modes)
	}

	// leg 11 — the full hook sweep AFTER off: a FRESH 0→1 cohort (leg 9
	// emptied the last one), a blur (would re-nudge), an armed completion —
	// everything dies at the config gate. Zero further captures.
	d.send(state.Event{Kind: state.EvPermission, EmployeeName: "boss", PermissionID: "perm-4",
		ToolName: "bash", ToolSummary: "rm -rf", ToolState: "pending"})
	d.send(tea.FocusMsg{})
	d.send(tea.BlurMsg{})
	typeIn("quiet boss")              // no y/a/n letters — the perm-4 popover owns them
	drainCmd(d, key(tea.KeyEnter), 0) // answers perm-4 (popover front)
	drainCmd(d, key(tea.KeyEnter), 0) // popover closed: the send lands
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("bossmsg-3", "boss", "silenced turn", false)})
	if err := wantTaps(3, "leg 11 (/notify off: the full hook sweep stays silent)"); err != nil {
		return err
	}

	// persistence proof: brain.json wrote the /notify off.
	bts, rerr := os.ReadFile(config.Path())
	if rerr != nil {
		return fail("read persisted brain.json: %v", rerr)
	}
	if !strings.Contains(string(bts), `"notifications": "off"`) {
		return fail("persisted brain.json missing the /notify write:\n%s", bts)
	}

	fmt.Println("===== UI SHOT · NOTIFICATIONS — cohort + done + toggled-off, capture ledger below =====")
	finalFrame := d.m.Frame()
	fmt.Println(finalFrame)
	fmt.Println("===== UI SHOT =====")
	fmt.Println("--- notify capture (kind | title | body) ---")
	for i, ln := range bus.taps {
		fmt.Printf("#%d %s\n", i+1, ln)
	}
	fmt.Printf("--- live SetMode taps: %v ---\n", bus.modes)
	fmt.Printf("--- persisted brain.json (%s) ---\n%s", config.Path(), bts)
	if !strings.Contains(finalFrame, "notifications → off") {
		return fail("final frame missing the \"notifications → off\" notice:\n%s", finalFrame)
	}
	fmt.Println("asserts: OK — focused startup silent (default true; the un-blurred completion consumed the turn's arm); blur on an empty cohort silent; ONE boss-ask cohort ping (child + third asks coalesce, generic agent+tool copy, no ToolSummary leak); click-burst front answer + resolved silent; ONE done ping off the armed send (clipped reply); re-blur re-nudges the live cohort front; refocused completion silent (typing through the popover answered its stacked asks); /notify bare reports the mode; /notify off → full sweep silent + live SetMode flip + persisted brain.json")
	return nil
}
