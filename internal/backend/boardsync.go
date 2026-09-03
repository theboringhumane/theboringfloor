// boardsync.go — the board-sync hook: after a REAL completion signal, sweep
// the office-owned board rows (normCtx.tasks) and flip stranded DOING rows
// to DONE so the member's DOING lane stops accumulating rows whose own
// close-out never arrived (work the boss did directly, flows that never
// returned a child lifecycle event, respawned sessions).
//
// Placement IS the guard: reconcileBoardDone runs ONLY at the two
// completion sites (maybeChildReturned's task-done+returned pair,
// QueueItemDone's ledger/queue flow) — synchronously under the owning
// backend's mutex. Batch respawns / ResetPrimary / re-hire storms, /new
// wipes, and non-completion event kinds (mail, text, title updates) never
// reach it; the Kind latch below is the belt-and-braces twin of that.
//
// Matching is conservative-by-design — a row flips only when exactly one
// rule claims it, and ambiguity across workers flips NOTHING (never flip
// with doubt):
//
//	exact ID    — "task-<sessionID>" is the EXISTING path's row (the
//	              caller already closed it); reconcile excludes it from
//	              every sweep so it never double-flips or inflates the note.
//	owner-name  — the completion's worker name owns other DOING rows: the
//	              OLDEST in-progress one flips (one per completion).
//	title-prefix— normalized titles (lowercase, whitespace transported)
//	              sharing a prefix of >= 8 runes. When the completion
//	              carries a worker name only its rows match; with no name
//	              ANY worker matches — but >1 distinct owner among the
//	              matches is ambiguity: no flip at all.
//
// agentmemory-mirrored rows are NEVER touched — structurally (syncBoard
// writes its rows straight to the emit lane; they never land in
// normCtx.tasks) and defensively (the id allowlist: office rows are always
// "task-<sessionID>", so "act-*"/"amx-*"/"agentmemory-*" mirror ids are
// skipped even if planted).
package backend

import (
	"sort"
	"strings"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// boardSyncPrefixMinRunes — the minimum shared normalized title prefix for
// a prefix-rule flip. Short enough to catch "Fix the ledger gate …" twins,
// long enough that "audit"/"build"/"run" collisions can't flip strangers.
const boardSyncPrefixMinRunes = 8

// normalizeSyncTitle — lowercase + whitespace transported (every whitespace
// run collapses to one space, edges trimmed) so the same brief spelled with
// different casing/spacing across flows still prefixes onto itself.
func normalizeSyncTitle(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// commonPrefixRunes — the shared-prefix length of two normalized strings in
// RUNES (titles may be non-ASCII).
func commonPrefixRunes(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	n := 0
	for n < len(ra) && n < len(rb) && ra[n] == rb[n] {
		n++
	}
	return n
}

// reconcileBoardDone sweeps normCtx.tasks for stranded DOING rows this
// completion accounts for, marks each flip in ctx (status done, original
// stamp kept) and returns the flipped rows — deterministic order (stamp,
// then id) so the caller's emit sequence is byte-stable. ev is the
// completion signal: Kind EvReturned; EmployeeName carries the worker's
// desk name ("" = unnamed — prefix matches may then cross workers, guarded
// by the ambiguity rule); EmployeeID keys the exact-path exclusion; the
// completion's own title rides Task.Title (falling back to a "return: "
// mail subject). An already-done row, a non-completion kind, an unnamed +
// untitled signal, or a doubtful cross-worker match all no-op silently.
func reconcileBoardDone(ctx *normCtx, ev state.Event) []state.BoardTask {
	if ev.Kind != state.EvReturned {
		return nil // completion signals only — mail/text/title churn never sweeps
	}

	owner := strings.TrimSpace(ev.EmployeeName)
	if owner == "" && ev.EmployeeID != "" {
		if emp, ok := ctx.employees[ev.EmployeeID]; ok {
			owner = emp.Name
		}
	}
	if owner == "" {
		owner = strings.TrimSpace(ev.Mail.From)
	}
	title := strings.TrimSpace(ev.Task.Title)
	if title == "" {
		title = strings.TrimSpace(strings.TrimPrefix(ev.Mail.Subject, "return: "))
	}
	exactID := strings.TrimSpace(ev.Task.ID)
	if exactID == "" && ev.EmployeeID != "" {
		exactID = "task-" + ev.EmployeeID
	}

	// Candidates: office-owned (the agentmemory id allowlist), still DOING,
	// not the row the caller's exact path just closed.
	type cand struct {
		key string
		t   state.BoardTask
	}
	var pool []cand
	for k, t := range ctx.tasks {
		if !strings.HasPrefix(t.ID, "task-") {
			continue // agentmemory-mirrored rows own their own lifecycle
		}
		if t.Status != state.TaskInProgress {
			continue
		}
		if t.ID == exactID {
			continue
		}
		pool = append(pool, cand{k, t})
	}
	if len(pool) == 0 {
		return nil
	}
	sort.Slice(pool, func(i, j int) bool {
		if pool[i].t.At != pool[j].t.At {
			return pool[i].t.At < pool[j].t.At
		}
		return pool[i].t.ID < pool[j].t.ID
	})

	flippedIDs := map[string]bool{}
	var flipped []state.BoardTask
	flip := func(c cand) {
		if flippedIDs[c.t.ID] {
			return
		}
		flippedIDs[c.t.ID] = true
		done := c.t
		done.Status = state.TaskDone
		ctx.tasks[c.key] = done
		flipped = append(flipped, done)
	}

	// Rule — owner name, oldest first: the returning worker's OLDEST
	// still-open row goes with them (one per completion — the quiet drain).
	if owner != "" {
		for _, c := range pool {
			if c.t.Owner == owner {
				flip(c)
				break
			}
		}
	}

	// Rule — normalized title-prefix (>= 8 runes): the completion's title
	// prefixes a DOING row's title. A named completion only matches its own
	// worker's rows; an UNNAMED one may match any worker, but matches spread
	// across >1 distinct owner is ambiguity — flip NOTHING (never flip with
	// doubt). Unambiguous matches flip their oldest only.
	if nt := normalizeSyncTitle(title); nt != "" {
		var matches []cand
		owners := map[string]bool{}
		for _, c := range pool {
			if flippedIDs[c.t.ID] {
				continue
			}
			if owner != "" && c.t.Owner != owner {
				continue
			}
			if commonPrefixRunes(nt, normalizeSyncTitle(c.t.Title)) >= boardSyncPrefixMinRunes {
				matches = append(matches, c)
				owners[c.t.Owner] = true
			}
		}
		if len(matches) > 0 && len(owners) <= 1 {
			flip(matches[0]) // pool order == oldest first
		}
	}

	if len(flipped) == 0 {
		return nil
	}
	sort.Slice(flipped, func(i, j int) bool {
		if flipped[i].At != flipped[j].At {
			return flipped[i].At < flipped[j].At
		}
		return flipped[i].ID < flipped[j].ID
	})
	return flipped
}
