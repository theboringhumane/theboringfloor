// boardsync_test.go — the board-sync hook (reconcileBoardDone) contract:
//
//	(a) owner-name oldest-first flip with two doings of the same name;
//	(b) worker-colliding flips NEVER happen (named completion against a
//	    stranger's row; unnamed completion with matches across owners);
//	(c) agentmemory-mirrored rows are never flipped by the sweep;
//	(d) a boss completion whose title prefixes a boss-owned doing row
//	    flips it;
//	(e) dedupe: the same event re-applied flips nothing and never
//	    regresses a done row;
//	(f) the sync note fires exactly once per flipped batch (end-to-end
//	    through maybeChildReturned), and a replayed idle stays silent;
//	(g) non-completion event kinds and non-DOING rows are no-ops.
package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// plantBoardDoing seeds ctx.tasks with one DOING row (keyed by its session).
func plantBoardDoing(ctx *normCtx, session, taskID, title, owner string, at int64) {
	ctx.tasks[session] = state.BoardTask{
		ID: taskID, Title: title, Status: state.TaskInProgress, Owner: owner, At: at,
	}
}

func flipTitles(flipped []state.BoardTask) []string {
	out := make([]string, len(flipped))
	for i, t := range flipped {
		out[i] = t.ID + ":" + t.Title
	}
	return out
}

// (a) two doings of the same name — the return's exact row already closed
// by the existing path; the OLDEST remaining row of that worker flips.
func TestBoardSyncReconcileOwnerNameOldestFirst(t *testing.T) {
	ctx := newNormCtx(nil)
	// tekton-1 stranded pile: the newer row just returned (closed by the
	// exact path), the older one must be the reconcile flip.
	plantBoardDoing(ctx, "ses-a", "task-ses-a", "Audit lints", "tekton-1", 100)
	ctx.tasks["ses-b"] = state.BoardTask{ID: "task-ses-b", Title: "Audit morning side",
		Status: state.TaskDone, Owner: "tekton-1", At: 200} // the exact-path row, already closed
	plantBoardDoing(ctx, "ses-c", "task-ses-c", "Scan the repo", "skopos-1", 50)

	flipped := reconcileBoardDone(ctx, state.Event{
		Kind:         state.EvReturned,
		EmployeeID:   "ses-b",
		EmployeeName: "tekton-1",
		Task:         ctx.tasks["ses-b"],
		Mail:         state.MailItem{From: "tekton-1", Subject: "return: Audit morning side"},
	})
	if len(flipped) != 1 || flipped[0].ID != "task-ses-a" {
		t.Fatalf("the OLDEST same-owner doing must flip, got %v", flipTitles(flipped))
	}
	if got := ctx.tasks["ses-a"].Status; got != state.TaskDone {
		t.Fatalf("ctx row must read done after the sweep, got %s", got)
	}
	if got := ctx.tasks["ses-c"].Status; got != state.TaskInProgress {
		t.Fatalf("the distinct-owner row must stay doing (worker collision), got %s", got)
	}

	// Reverse the stamps: with the ordering flipped the OTHER row is the
	// oldest — the pick rides (At, ID), never the map walk.
	ctx2 := newNormCtx(nil)
	plantBoardDoing(ctx2, "ses-a", "task-ses-a", "Audit lints", "tekton-1", 900)
	plantBoardDoing(ctx2, "ses-c", "task-ses-c", "Audit morning side", "tekton-1", 100)
	flipped2 := reconcileBoardDone(ctx2, state.Event{
		Kind:         state.EvReturned,
		EmployeeID:   "ses-gone",
		EmployeeName: "tekton-1",
		Task:         state.BoardTask{ID: "task-ses-gone", Title: "unrelated", Status: state.TaskDone},
	})
	if len(flipped2) != 1 || flipped2[0].ID != "task-ses-c" {
		t.Fatalf("oldest-first must follow the stamp, got %v", flipTitles(flipped2))
	}
}

// (b) worker collisions never flip: a NAMED completion only touches its own
// worker's rows; an UNNAMED one facing matches across owners is ambiguity —
// nothing flips.
func TestBoardSyncReconcileWorkerCollisionNoFlip(t *testing.T) {
	// Named: tekton-2's completion titles onto tekton-1's row — owner
	// present, so only tekton-2 rows may flip; there are none.
	ctx := newNormCtx(nil)
	plantBoardDoing(ctx, "ses-a", "task-ses-a", "Wire R17 Razorpay mandate", "tekton-1", 100)
	flipped := reconcileBoardDone(ctx, state.Event{
		Kind:         state.EvReturned,
		EmployeeID:   "ses-b",
		EmployeeName: "tekton-2",
		Task:         state.BoardTask{ID: "task-ses-b", Title: "Wire R17 Razorpay mandate", Status: state.TaskDone},
	})
	if len(flipped) != 0 || ctx.tasks["ses-a"].Status != state.TaskInProgress {
		t.Fatalf("a named completion must never flip another worker's row, flipped=%v", flipTitles(flipped))
	}

	// Unnamed: matches across >1 distinct owner = doubt = no flip.
	ctx2 := newNormCtx(nil)
	plantBoardDoing(ctx2, "ses-a", "task-ses-a", "Registry rot cleanup", "tekton-1", 100)
	plantBoardDoing(ctx2, "ses-b", "task-ses-b", "Registry rot cleanup sweep", "skopos-1", 200)
	flipped2 := reconcileBoardDone(ctx2, state.Event{
		Kind: state.EvReturned,
		Task: state.BoardTask{ID: "task-elsewhere", Title: "Registry rot cleanup — pass 2", Status: state.TaskDone, Owner: "manager"},
		Mail: state.MailItem{Subject: "return: Registry rot cleanup — pass 2"},
	})
	if len(flipped2) != 0 {
		t.Fatalf("cross-worker ambiguity must flip NOTHING, got %v", flipTitles(flipped2))
	}
	if ctx2.tasks["ses-a"].Status != state.TaskInProgress || ctx2.tasks["ses-b"].Status != state.TaskInProgress {
		t.Fatalf("ambiguous sweep must leave both rows doing: %+v / %+v", ctx2.tasks["ses-a"], ctx2.tasks["ses-b"])
	}
}

// (c) agentmemory-mirrored rows (syncBoard's ids, e.g. "act-*") are the
// mirror's own state — the sweep never flips them, even for its own flow.
func TestBoardSyncReconcileAgentmemoryRowsSafe(t *testing.T) {
	ctx := newNormCtx(nil)
	plantBoardDoing(ctx, "amx-1", "act-42", "QUE-1: answer the quiz", "agentmemory", 100)
	plantBoardDoing(ctx, "ses-a", "task-ses-a", "Audit lints", "tekton-1", 200)

	// A queue completion carrying the exact mirror title+owner still leaves
	// the mirror row alone (owner "queue" matches nothing office-owned).
	if flipped := reconcileBoardDone(ctx, state.Event{
		Kind:         state.EvReturned,
		EmployeeName: "queue",
		Task:         state.BoardTask{Title: "QUE-1: answer the quiz"},
	}); len(flipped) != 0 {
		t.Fatalf("the queue completion must not flip office/mirror rows, got %v", flipTitles(flipped))
	}
	if got := ctx.tasks["amx-1"].Status; got != state.TaskInProgress {
		t.Fatalf("the agentmemory row must stay exactly as syncBoard left it, got %s", got)
	}

	// And a return from the row's OWN (mirror) owner: the id allowlist bars
	// it outright — only the office row flips.
	flipped := reconcileBoardDone(ctx, state.Event{
		Kind:         state.EvReturned,
		EmployeeID:   "ses-x",
		EmployeeName: "tekton-1",
		Task:         state.BoardTask{ID: "task-ses-x", Title: "Audit lints — done", Status: state.TaskDone},
	})
	if len(flipped) != 1 || flipped[0].ID != "task-ses-a" {
		t.Fatalf("only the office-owned row may flip, got %v", flipTitles(flipped))
	}
	if got := ctx.tasks["amx-1"].Status; got != state.TaskInProgress {
		t.Fatalf("the mirror row must survive the sweep untouched, got %s", got)
	}
}

// (d) a boss completion whose task title prefixes (normalized, >= 8 runes)
// a boss-owned DOING row flips it; a shared short prefix never does.
func TestBoardSyncReconcileBossPrefixFlips(t *testing.T) {
	ctx := newNormCtx(nil)
	plantBoardDoing(ctx, "ses-a", "task-ses-a", "Audit morning side", "boss", 100)
	plantBoardDoing(ctx, "ses-b", "task-ses-b", "Run targeted DELTA-1 test", "tekton-1", 200)

	// Normalization covers case + whitespace transport across the two
	// spellings of the same brief.
	flipped := reconcileBoardDone(ctx, state.Event{
		Kind:         state.EvReturned,
		EmployeeID:   "ses-boss",
		EmployeeName: "boss",
		Task:         state.BoardTask{ID: "task-ses-boss", Title: "audit   morning\nside — findings posted", Status: state.TaskDone, Owner: "boss"},
	})
	if len(flipped) != 1 || flipped[0].ID != "task-ses-a" {
		t.Fatalf("the boss-owned prefix-matched doing must flip, got %v", flipTitles(flipped))
	}
	if got := ctx.tasks["ses-b"].Status; got != state.TaskInProgress {
		t.Fatalf("the non-matching row must stay doing, got %s", got)
	}

	// Short prefix (< 8 normalized runes): "Audit" alone claims nothing.
	// UNNAMED completion here — a named one flips the same-owner row via
	// the owner-name rule by design; this leg pins the prefix threshold.
	ctx2 := newNormCtx(nil)
	plantBoardDoing(ctx2, "ses-a", "task-ses-a", "Audit lints", "boss", 100)
	if flipped := reconcileBoardDone(ctx2, state.Event{
		Kind: state.EvReturned,
		Task: state.BoardTask{ID: "task-x", Title: "Audit", Status: state.TaskDone},
	}); len(flipped) != 0 {
		t.Fatalf("a %d-rune prefix must never flip (< %d), got %v", 5, 8, flipTitles(flipped))
	}

	// Unnamed completion, ONE owner across the matches: flips (no doubt).
	ctx3 := newNormCtx(nil)
	plantBoardDoing(ctx3, "ses-a", "task-ses-a", "Build DN-105 console", "tekton-1", 100)
	plantBoardDoing(ctx3, "ses-b", "task-ses-b", "Build DN-105 console modal", "tekton-1", 200)
	flipped3 := reconcileBoardDone(ctx3, state.Event{
		Kind: state.EvReturned,
		Task: state.BoardTask{ID: "task-x", Title: "Build DN-105 console — shipped", Status: state.TaskDone},
	})
	if len(flipped3) != 1 || flipped3[0].ID != "task-ses-a" {
		t.Fatalf("a single-owner unnamed prefix sweep flips its oldest, got %v", flipTitles(flipped3))
	}
}

// (e) dedupe: re-applying the same completion flips nothing twice and never
// regresses a done row back to doing.
func TestBoardSyncReconcileDedupe(t *testing.T) {
	ctx := newNormCtx(nil)
	plantBoardDoing(ctx, "ses-a", "task-ses-a", "Audit lints", "tekton-1", 100)
	ev := state.Event{
		Kind:         state.EvReturned,
		EmployeeID:   "ses-b",
		EmployeeName: "tekton-1",
		Task:         state.BoardTask{ID: "task-ses-b", Title: "Audit lints pass", Status: state.TaskDone},
	}
	first := reconcileBoardDone(ctx, ev)
	if len(first) != 1 || first[0].ID != "task-ses-a" {
		t.Fatalf("first sweep must flip once, got %v", flipTitles(first))
	}
	second := reconcileBoardDone(ctx, ev)
	if len(second) != 0 {
		t.Fatalf("re-applying the same event must flip NOTHING, got %v", flipTitles(second))
	}
	if got := ctx.tasks["ses-a"].Status; got != state.TaskDone {
		t.Fatalf("a reconciled row must NEVER regress to doing, got %s", got)
	}
}

// (g) guards: non-completion kinds never sweep; pending rows and the
// exact-path row are out of the sweep's reach.
func TestBoardSyncReconcileGuards(t *testing.T) {
	ctx := newNormCtx(nil)
	plantBoardDoing(ctx, "ses-a", "task-ses-a", "Audit lints", "tekton-1", 100)
	// A mail-shaped event (not a completion) must no-op even fully titled.
	for _, kind := range []state.EventKind{state.EvMail, state.EvTask, state.EvStatus, state.EvChatBoss, state.EvDispatch} {
		if flipped := reconcileBoardDone(ctx, state.Event{
			Kind: kind, EmployeeName: "tekton-1",
			Task: state.BoardTask{Title: "Audit lints"},
		}); len(flipped) != 0 {
			t.Fatalf("kind %s must never sweep, got %v", kind, flipTitles(flipped))
		}
	}
	// Pending (not DOING) rows are out of scope; the exact-path row the
	// caller closed itself is never re-flipped/re-noted by the sweep.
	ctx.tasks["ses-p"] = state.BoardTask{ID: "task-ses-p", Title: "Audit lints queued",
		Status: state.TaskPending, Owner: "tekton-1", At: 50}
	ctx.tasks["ses-x"] = state.BoardTask{ID: "task-ses-x", Title: "Audit lints exact",
		Status: state.TaskInProgress, Owner: "tekton-1", At: 10}
	flipped := reconcileBoardDone(ctx, state.Event{
		Kind:         state.EvReturned,
		EmployeeID:   "ses-x",
		EmployeeName: "tekton-1",
	})
	if len(flipped) != 1 || flipped[0].ID != "task-ses-a" {
		t.Fatalf("only the DOING non-exact row flips, got %v", flipTitles(flipped))
	}
	if got := ctx.tasks["ses-p"].Status; got != state.TaskPending {
		t.Fatalf("pending rows stay pending, got %s", got)
	}
	if got := ctx.tasks["ses-x"].Status; got != state.TaskInProgress {
		t.Fatalf("the exact-path row is the caller's business — untouched, got %s", got)
	}
}

// boardSyncStubServe — maybeChildReturned's I/O for TWO known children:
// each session's message listing carries one assistant text.
func boardSyncStubServe(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sessionID, text string
		switch r.URL.Path {
		case "/session/ses-child-1/message":
			sessionID, text = "ses-child-1", "lints wired; the morning side needs the next pass"
		case "/session/ses-child-2/message":
			sessionID, text = "ses-child-2", "unrelated brief done"
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		enc, _ := json.Marshal(text)
		w.Write([]byte(`[{"info":{"id":"m-1","sessionID":"` + sessionID + `","role":"assistant","finish":"stop","time":{"created":1,"completed":2}},"parts":[{"id":"p1","type":"text","text":` + string(enc) + `}]}]`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// (f) end-to-end through maybeChildReturned: the exact row closes via the
// EXISTING path, the stranded same-worker row flips via the sweep, and the
// "[office] board sync: flipped N rows to done" note fires EXACTLY once —
// a replayed idle frame doesn't re-flip, re-note, or regress anything.
func TestBoardSyncChildReturnFlipsAndNotesOnce(t *testing.T) {
	srv := boardSyncStubServe(t)
	dir := t.TempDir()
	b := newLiveBackend(srv.URL, dir, config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	t.Cleanup(func() { _ = b.Stop() })
	b.mu.Lock()
	b.baseURL = srv.URL
	b.primaryID = "ses-primary"
	b.am = probeAgentmemory("http://127.0.0.1:1")
	b.ctx.employees["ses-child-1"] = state.Employee{
		ID: "ses-child-1", Name: "tekton-1", Role: state.RoleDeveloper, Seat: "dev-1",
		Sprite: state.SpriteAtDesk, Task: "Audit lints",
	}
	// The returning child's own brief (the exact-path row) —
	plantBoardDoing(b.ctx, "ses-child-1", "task-ses-child-1", "Audit lints", "tekton-1", 200)
	// — and a STRANDED row of the same worker: its return event never came.
	plantBoardDoing(b.ctx, "ses-ghost", "task-ses-ghost", "Audit morning side", "tekton-1", 100)
	// A different worker's doing, same title family — must survive.
	plantBoardDoing(b.ctx, "ses-sco", "task-ses-sco", "Audit lints recon", "skopos-1", 300)
	// A queued-but-never-started row — out of the DOING lane's sweep.
	b.ctx.tasks["ses-pend"] = state.BoardTask{ID: "task-ses-pend", Title: "Audit lints later",
		Status: state.TaskPending, Owner: "tekton-1", At: 400}
	b.mu.Unlock()

	b.maybeChildReturned("ses-child-1")

	// Exactly ONE reconcile flip: the stranded tekton-1 row (owner-name).
	doneTasks := eventsMatching(log, func(e state.Event) bool {
		return e.Kind == state.EvTask && e.Task.Status == state.TaskDone
	})
	if len(doneTasks) != 2 {
		t.Fatalf("expected exact-path + reconcile flip = 2 done upserts, got %d: %v",
			len(doneTasks), flipTitles(func() []state.BoardTask {
				var r []state.BoardTask
				for _, e := range doneTasks {
					r = append(r, e.Task)
				}
				return r
			}()))
	}
	if doneTasks[1].Task.ID != "task-ses-ghost" {
		t.Fatalf("the reconcile flip must be the stranded row, got %q", doneTasks[1].Task.ID)
	}
	// The note: exactly once, exactly the member-visible copy.
	notes := eventsMatching(log, func(e state.Event) bool {
		return e.Kind == state.EvStatus && strings.Contains(e.Text, "board sync: flipped")
	})
	if len(notes) != 1 || notes[0].Text != "[office] board sync: flipped 1 rows to done" {
		t.Fatalf("the sync note fires exactly once with the batch count, got %+v", func() []string {
			var s []string
			for _, e := range notes {
				s = append(s, e.Text)
			}
			return s
		}())
	}
	// The untouched lanes.
	if got := b.ctx.tasks["ses-sco"].Status; got != state.TaskInProgress {
		t.Fatalf("the distinct-owner row must stay doing, got %s", got)
	}
	if got := b.ctx.tasks["ses-pend"].Status; got != state.TaskPending {
		t.Fatalf("pending rows are out of the sweep, got %s", got)
	}

	// Replay the idle: everything silent — no new flips, no second note,
	// no regressions.
	b.maybeChildReturned("ses-child-1")
	if n := eventsMatching(log, func(e state.Event) bool {
		return e.Kind == state.EvTask && e.Task.Status == state.TaskDone
	}); len(n) != 2 {
		t.Fatalf("replayed idle must emit no further done upserts, got %d", len(n))
	}
	if n := log.textCount("board sync: flipped"); n != 1 {
		t.Fatalf("still exactly one note after the replay, got %d", n)
	}
	if got := b.ctx.tasks["ses-ghost"].Status; got != state.TaskDone {
		t.Fatalf("the flipped row must stay done, got %s", got)
	}

	// A second worker's return with no stranded same-worker rows: the sweep
	// runs, flips NOTHING, and stays silent (no zero-flip note).
	b.mu.Lock()
	b.ctx.employees["ses-child-2"] = state.Employee{
		ID: "ses-child-2", Name: "dikastes-1", Role: state.RoleReviewer, Seat: "rev-1",
		Sprite: state.SpriteAtDesk, Task: "Review the map",
	}
	plantBoardDoing(b.ctx, "ses-child-2", "task-ses-child-2", "Review the map", "dikastes-1", 500)
	b.mu.Unlock()
	b.maybeChildReturned("ses-child-2")
	if n := log.textCount("board sync: flipped"); n != 1 {
		t.Fatalf("a flipless sweep must not emit the note, got %d", n)
	}
	if got := b.ctx.tasks["ses-sco"].Status; got != state.TaskInProgress {
		t.Fatalf("skopos-1's row survives every sweep, got %s", got)
	}
}

// The queue hook: QueueItemDone fires the sweep with owner "queue" — quiet
// for office rows (the conservative floor) while office rows DOING around
// it stay put; the note stays silent.
func TestBoardSyncQueueDoneSweepQuiet(t *testing.T) {
	dir := t.TempDir()
	b := newLiveBackend("http://127.0.0.1:1", dir, config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)
	t.Cleanup(func() { _ = b.Stop() })
	b.mu.Lock()
	b.am = probeAgentmemory("http://127.0.0.1:1")
	b.primaryID = "ses-primary"
	plantBoardDoing(b.ctx, "ses-a", "task-ses-a", "answer the quiz line", "tekton-1", 100)
	b.mu.Unlock()

	id := b.QueueItemStart(2, "answer the quiz line")
	b.QueueItemDone(id)
	b.QueueItemDone(id) // twice-safe: latch first, sweep second — no double sweep

	waitLedgerFile(t, dir, func(s string) bool {
		return strings.Contains(s, "answer the quiz line")
	}, "the queue completion entry")
	if n := log.textCount("board sync: flipped"); n != 0 {
		t.Fatalf("owner \"queue\" owns no office rows — the note must stay silent, got %d", n)
	}
	if got := b.ctx.tasks["ses-a"].Status; got != state.TaskInProgress {
		t.Fatalf("queue completions never flip a live worker's row, got %s", got)
	}
}
