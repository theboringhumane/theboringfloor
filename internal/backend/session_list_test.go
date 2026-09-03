// session_list_test.go — the /session picker's backend seams against
// httptest doubles (never a real server):
//
//	(a) ListSessions seam HIT: only ROOT sessions (parentID == "") come
//	    back with their wire times/title and the per-session message
//	    count; a count fetch that fails degrades to Messages=-1 and the
//	    row still renders;
//	(a-2) ListSessions seam MISS: the list call's own failure is the
//	    error (the app's static-fallback trigger);
//	(b) ResumeOffice HIT: the pinned session is verified server-side,
//	    the primary + override latch swap, respawn latches are consumed,
//	    the old boss row fires and the new one hires, and the generalized
//	    "resume <id> (pinned)" note lands;
//	(c) ResumeOffice MISS: a 404/fetch failure keeps the current primary
//	    seated, returns the error and notes the honest degrade;
//	(d) the demo backend NEVER gains either seam (the type-assert guard).
package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// (a) seam hit: roots only, wire fields mapped, counts filled; one
// root's count 404s and lands as -1 without losing the row.
func TestListSessionsHit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session":
			w.Write([]byte(`[
				{"id":"ses-root-a","title":"first office","time":{"created":10,"updated":30}},
				{"id":"ses-child","parentID":"ses-root-a","title":"a child","time":{"created":20,"updated":40}},
				{"id":"ses-root-b","title":"","time":{"created":15,"updated":35}}
			]`))
		case r.Method == http.MethodGet && r.URL.Path == "/session/ses-root-a/message":
			w.Write([]byte(`[{"id":"m1"},{"id":"m2"},{"id":"m3"}]`))
		default: // ses-root-b's count 404s → the row still renders at -1
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	b, _ := resolveFixture(t, srv)

	rows, err := b.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("children must be filtered — want 2 roots, got %d: %+v", len(rows), rows)
	}
	got := map[string]state.SessionRow{}
	for _, r := range rows {
		got[r.ID] = r
		if r.ParentID != "" {
			t.Fatalf("a child leaked into the roots listing: %+v", r)
		}
	}
	a, ok := got["ses-root-a"]
	if !ok {
		t.Fatalf("ses-root-a missing: %+v", rows)
	}
	if a.Title != "first office" || a.Created != 10 || a.Updated != 30 {
		t.Fatalf("wire fields must map verbatim: %+v", a)
	}
	if a.Messages != 3 {
		t.Fatalf("message count must come from /session/{id}/message, got %d", a.Messages)
	}
	bb, ok := got["ses-root-b"]
	if !ok {
		t.Fatalf("ses-root-b missing: %+v", rows)
	}
	if bb.Messages != -1 {
		t.Fatalf("a failed count must degrade to -1 (the row still renders), got %d", bb.Messages)
	}
}

// (a-2) seam miss: the list call itself failing IS the error (the app
// prints the static /session fallback on it).
func TestListSessionsMiss(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	b, _ := resolveFixture(t, srv)

	rows, err := b.ListSessions(context.Background())
	if err == nil {
		t.Fatalf("a failed listing must return the error, got rows %+v", rows)
	}
	if rows != nil {
		t.Fatalf("no rows on a listing failure, got %+v", rows)
	}
}

// (b) ResumeOffice hit: verified server-side, the primary + override
// swap, respawn latches consumed, fire-old/hire-new events, generalized
// pinned note.
func TestResumeOfficeHit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/session/ses-new":
			w.Write([]byte(`{"id":"ses-new","title":"other office","time":{"created":5,"updated":9}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	b, log := resolveFixture(t, srv)
	b.mu.Lock()
	b.primaryID = "ses-old"
	b.respawnFresh = true // a pending respawn must not survive a live re-anchor
	b.respawnOldID = "ses-old"
	b.mu.Unlock()

	if err := b.ResumeOffice("ses-new"); err != nil {
		t.Fatalf("ResumeOffice: %v", err)
	}
	if got := b.PrimaryID(); got != "ses-new" {
		t.Fatalf("the pin must swap the primary live, got %q", got)
	}
	b.mu.Lock()
	override, respawnFresh, respawnOldID := b.primaryOverride, b.respawnFresh, b.respawnOldID
	b.mu.Unlock()
	if override != "ses-new" {
		t.Fatalf("the override latch must follow the pin (persist + next Start), got %q", override)
	}
	if respawnFresh || respawnOldID != "" {
		t.Fatalf("a live re-anchor must consume the respawn latches, got fresh=%v old=%q", respawnFresh, respawnOldID)
	}
	fires, hires := 0, 0
	log.mu.Lock()
	for _, e := range log.evs {
		switch e.Kind {
		case state.EvFire:
			if e.EmployeeID == "ses-old" {
				fires++
			}
		case state.EvHire:
			if e.Employee.ID == "ses-new" && e.Employee.Role == state.RoleManager {
				hires++
			}
		}
	}
	log.mu.Unlock()
	if fires != 1 || hires != 1 {
		t.Fatalf("want ONE fire of the old boss + ONE hire of the new, got %d/%d", fires, hires)
	}
	if n := log.textCount("primary session: resume ses-new (pinned)"); n != 1 {
		t.Fatalf("the generalized pinned note must fire once, got %d", n)
	}
}

// (c) ResumeOffice miss: the server has no such session — the current
// primary stays seated, the error returns for the chat notice, and the
// note reads as an honest degrade.
func TestResumeOfficeMiss(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	b, log := resolveFixture(t, srv)
	b.mu.Lock()
	b.primaryID = "ses-staying"
	b.mu.Unlock()

	if err := b.ResumeOffice("ses-gone"); err == nil {
		t.Fatalf("a server-side miss must return the error")
	}
	if got := b.PrimaryID(); got != "ses-staying" {
		t.Fatalf("the current primary must stay seated on a miss, got %q", got)
	}
	if n := log.textCount("pinned session ses-gone not found server-side — staying on the current office session"); n != 1 {
		t.Fatalf("the degrade note must fire once, got %d", n)
	}
}

// (d) the demo backend NEVER gains either seam — the app's type-assert
// fallback depends on it (guarded like the state seams' stub checks).
func TestDemoLacksSessionPickerSeams(t *testing.T) {
	demo := newDemoBackend(config.Default())
	if _, ok := any(demo).(interface {
		ListSessions(context.Context) ([]state.SessionRow, error)
	}); ok {
		t.Fatalf("the demo backend must NOT implement ListSessions (the /session picker falls back to the static summary)")
	}
	if _, ok := any(demo).(interface{ ResumeOffice(string) error }); ok {
		t.Fatalf("the demo backend must NOT implement ResumeOffice (a scripted tour never re-anchors)")
	}
}
