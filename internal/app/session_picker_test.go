// session_picker_test.go — the interactive /session picker contract from
// the model level (fakes only, never a real server):
//
//	(b) buildSessionRows: ROOT sessions only, sorted by Updated desc,
//	    the current session marked, blank titles fallen back to the
//	    short id; relAge / shortSessionID tables;
//	(c) ACCEPT: opening /session lists the rows, enter on another
//	    session re-anchors the office LIVE (resume seam called with the
//	    id), the surfaces clear NewOffice-style, the dim explicit-pin
//	    notice lands, and the pin is persisted into session.json
//	    IMMEDIATELY (next boot auto-restores);
//	(d) ACCEPT-CURRENT: a no-op — "already on session <id>", no wipe,
//	    no seam call;
//	(e) BUSY BLOCK: with a boss turn in flight, accepting a DIFFERENT
//	    session is refused ("boss is busy — /stop or wait, then /session
//	    again"), the picker closes, nothing else changes;
//	(f) FALLBACK: a backend without the listing seam (pinBackend) or a
//	    failing listing closes/never opens the picker and prints the
//	    static summary + a dim picker-unavailable note;
//	(g) ESC: cancels with zero side effects — no notice, no seam call,
//	    picker closed.
package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// sessBackend — a LIVE-mode recording backend with the primary seam PLUS
// the picker's two additive seams: ListSessions (scripted rows/err,
// calls counted) and ResumeOffice (records every re-anchor and moves the
// scripted primary, resumeErr forces the degrade path).
type sessBackend struct {
	recBackend
	primary   string
	rows      []state.SessionRow
	listErr   error
	listCalls int
	resumed   []string
	resumeErr error
}

func (b *sessBackend) Mode() state.Mode          { return state.ModeLive }
func (b *sessBackend) PrimaryOverride(id string) {}
func (b *sessBackend) PrimaryID() string         { return b.primary }
func (b *sessBackend) ListSessions(ctx context.Context) ([]state.SessionRow, error) {
	b.listCalls++
	return b.rows, b.listErr
}
func (b *sessBackend) ResumeOffice(id string) error {
	if b.resumeErr != nil {
		return b.resumeErr
	}
	b.primary = id
	b.resumed = append(b.resumed, id)
	return nil
}

// sessRowsFixture — two roots: the CURRENT one is the newest row (sel 0),
// the older beta row is the row a "down + enter" picks.
func sessRowsFixture() []state.SessionRow {
	return []state.SessionRow{
		{ID: "ses-alpha-new", Title: "alpha brief (current)", Updated: 3000, Created: 100, Messages: 2},
		{ID: "ses-beta-older", Title: "beta office", Updated: 2000, Created: 50, Messages: 41},
		{ID: "ses-child", ParentID: "ses-beta-older", Title: "child", Updated: 2500, Created: 60, Messages: 3},
	}
}

// (b) rows: roots only, Updated desc, current marked, title fallback.
func TestBuildSessionRows(t *testing.T) {
	now := time.UnixMilli(10_000)
	rows := []state.SessionRow{
		{ID: "ses-b", Title: "  beta\n office ", Updated: 200, Created: 100, Messages: 5},
		{ID: "ses-child", ParentID: "ses-b", Updated: 999, Created: 999, Messages: 7},
		{ID: "ses-a", Title: "", Updated: 300, Created: 50, Messages: -1},
		{ID: "ses-c", Title: "gamma notes", Updated: 100, Created: 80, Messages: 0},
	}
	out := buildSessionRows(rows, "ses-c", now)
	if len(out) != 3 {
		t.Fatalf("children must drop out — want 3 roots, got %d: %+v", len(out), out)
	}
	if out[0].ID != "ses-a" || out[1].ID != "ses-b" || out[2].ID != "ses-c" {
		t.Fatalf("Updated desc order broken: %v", []string{out[0].ID, out[1].ID, out[2].ID})
	}
	if out[0].Title != "ses-a" {
		t.Fatalf("a blank title must fall back to the short id, got %q", out[0].Title)
	}
	if out[1].Title != "beta office" {
		t.Fatalf("whitespace/newlines must flatten, got %q", out[1].Title)
	}
	if !out[2].Current || out[0].Current || out[1].Current {
		t.Fatalf("only the attached session is marked current: %+v", out)
	}
	if out[2].Messages != 0 || out[0].Messages != -1 {
		t.Fatalf("message counts (0 and the -1 unknown) must ride through: %+v", out)
	}
}

// (b-2) sort stability: Updated ties break on Created desc, then id asc.
func TestBuildSessionRowsTies(t *testing.T) {
	now := time.UnixMilli(10_000)
	rows := []state.SessionRow{
		{ID: "ses-z", Updated: 100, Created: 10},
		{ID: "ses-a", Updated: 100, Created: 30},
		{ID: "ses-b", Updated: 100, Created: 30},
	}
	out := buildSessionRows(rows, "", now)
	if out[0].ID != "ses-a" || out[1].ID != "ses-b" || out[2].ID != "ses-z" {
		t.Fatalf("tie-break must be Created desc then id asc: %v",
			[]string{out[0].ID, out[1].ID, out[2].ID})
	}
	// no current marking when the backend reports no primary at all.
	for _, r := range out {
		if r.Current {
			t.Fatalf("an empty current id must mark NOTHING: %+v", r)
		}
	}
}

// (b-3) machine ages + short ids.
func TestRelAgeAndShortID(t *testing.T) {
	now := time.UnixMilli(1_000_000_000_000) // a roomy fixed instant
	for _, tc := range []struct {
		stamp int64
		want  string
	}{
		{now.UnixMilli(), "now"}, // same instant
		{now.UnixMilli() - 5_000, "now"},
		{now.UnixMilli() - 45_000, "45s"},
		{now.UnixMilli() - 180_000, "3m"},
		{now.UnixMilli() - 7_200_000, "2h"},
		{now.UnixMilli() - 432_000_000, "5d"},
		{now.UnixMilli() - 1_987_200_000, "3w"},
		{0, "now"},                           // no stamp
		{now.UnixMilli() + 1_000_000, "now"}, // future (clock skew) clamps
	} {
		if got := relAge(now, tc.stamp); got != tc.want {
			t.Fatalf("relAge(%d) = %q, want %q", tc.stamp, got, tc.want)
		}
	}
	if got := shortSessionID("ses-a"); got != "ses-a" {
		t.Fatalf("a short id stays intact, got %q", got)
	}
	if got := shortSessionID("ses_alpha1234567890"); got != "ses_alph" {
		t.Fatalf("a long id clips to leading 8 runes, got %q", got)
	}
}

// (c) ACCEPT: resume seam hit, surfaces cleared, notice, immediate persist.
func TestSessionPickerAcceptResumesLive(t *testing.T) {
	scratchHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	b := &sessBackend{primary: "ses-alpha-new", rows: sessRowsFixture()}
	m := New(b, nil)

	// /session opens the picker and rides the async listing hop.
	m = runMsg(t, m, slashMsg{text: "/session"})
	if b.listCalls != 1 {
		t.Fatalf("one ListSessions hop per /session, got %d", b.listCalls)
	}
	if m.chat == nil || !m.chat.SessionPickerOpen() {
		t.Fatalf("the picker must be open after the listing lands")
	}

	// down + enter: accept the OLDER session (the current one sits at sel 0).
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if len(b.resumed) != 1 || b.resumed[0] != "ses-beta-older" {
		t.Fatalf("the accept must re-anchor via the resume seam, got %v", b.resumed)
	}
	if b.primary != "ses-beta-older" {
		t.Fatalf("the primary must swap live, got %q", b.primary)
	}
	if m.chat.SessionPickerOpen() {
		t.Fatalf("every accept path closes the picker")
	}
	// surfaces cleared + exactly ONE dim explicit-pin notice.
	if len(m.st.Chat) != 1 {
		t.Fatalf("the surfaces must clear to exactly the notice row, got %d entries", len(m.st.Chat))
	}
	last := lastChat(t, m)
	if last.From != "office" || last.Meta == "error" {
		t.Fatalf("the accept notice must be a clean dim office notice: from=%q meta=%q", last.From, last.Meta)
	}
	if last.Text != "resumed session ses-beta-older (explicit pin) · /new for a fresh office" {
		t.Fatalf("the notice text is frozen, got %q", last.Text)
	}
	// the pin must be persisted NOW (the throttled loop would lag a quit).
	sf, ok := LoadSession(cwd)
	if !ok {
		t.Fatalf("the accept must persist session.json immediately")
	}
	if sf.PrimaryID != "ses-beta-older" {
		t.Fatalf("session.json must carry the new pin (next boot auto-restores), got %q", sf.PrimaryID)
	}
}

// (d) ACCEPT-CURRENT: a no-op with the dim notice; no seam call, no wipe.
func TestSessionPickerAcceptCurrentNoop(t *testing.T) {
	scratchHome(t)
	b := &sessBackend{primary: "ses-alpha-new", rows: sessRowsFixture()}
	m := New(b, nil)
	// a pre-existing entry proves "no wipe".
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser,
		Msg: state.ChatMsg{ID: "u1", From: "user", Kind: "user", Text: "keep me", At: 1}})

	m = runMsg(t, m, slashMsg{text: "/session"})
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})) // sel 0 IS the current session

	if len(b.resumed) != 0 {
		t.Fatalf("accepting the current session must NEVER call the resume seam: %v", b.resumed)
	}
	if !chatHas(m, "keep me") {
		t.Fatalf("a no-op accept must not wipe the transcript")
	}
	if !chatHas(m, "already on session ses-alpha-new") {
		t.Fatalf("the already-on notice must land: %v", chatTexts(m))
	}
	if m.chat.SessionPickerOpen() {
		t.Fatalf("the picker closes even on a no-op accept")
	}
}

// (e) BUSY BLOCK: a boss turn in flight refuses a DIFFERENT accept.
func TestSessionPickerBusyBlock(t *testing.T) {
	scratchHome(t)
	b := &sessBackend{primary: "ses-alpha-new", rows: sessRowsFixture()}
	m := New(b, nil)
	// the boss has work in flight (submitted-but-unanswered placeholder).
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "boss-1", From: "boss", Pending: true}})

	m = runMsg(t, m, slashMsg{text: "/session"}) // the picker may OPEN while busy
	if !m.chat.SessionPickerOpen() {
		t.Fatalf("opening the picker while busy is allowed — only accepting is refused")
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	if len(b.resumed) != 0 {
		t.Fatalf("a busy boss must refuse the swap: %v", b.resumed)
	}
	if b.primary != "ses-alpha-new" {
		t.Fatalf("nothing must re-anchor on a refusal, primary=%q", b.primary)
	}
	if m.chat.SessionPickerOpen() {
		t.Fatalf("the refused picker closes")
	}
	if !chatHas(m, "boss is busy — /stop or wait, then /session again") {
		t.Fatalf("the frozen busy-block notice must land: %v", chatTexts(m))
	}
	if !hasPendingBoss(m.st) {
		t.Fatalf("the in-flight turn must survive untouched")
	}
}

// (f) FALLBACK #1: a backend WITHOUT the listing seam gets the static
// summary + the dim unavailable note, and the picker never opens.
func TestSessionSlashFallbackNoSeam(t *testing.T) {
	scratchHome(t)
	b := &pinBackend{primary: "ses-live-9"} // primary seam only — no ListSessions
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/session"})

	if m.chat.SessionPickerOpen() {
		t.Fatalf("a seam-less backend must never open the picker")
	}
	last := lastChat(t, m)
	for _, want := range []string{
		"session: ses-live-9 (primary)",
		"resume on the next boot: theboringoffice -s ses-live-9",
		"(session picker unavailable on this backend",
	} {
		if !strings.Contains(last.Text, want) {
			t.Fatalf("the fallback must contain %q:\n%s", want, last.Text)
		}
	}
	if last.Meta == "error" {
		t.Fatalf("the fallback is a dim note, never an error: meta=%q", last.Meta)
	}
}

// (f-2) FALLBACK #2: a listing FAILURE closes the opened picker and
// prints the static summary carrying the real error.
func TestSessionSlashFallbackListingError(t *testing.T) {
	scratchHome(t)
	b := &sessBackend{primary: "ses-live-9", listErr: errors.New("opencode serve unreachable")}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/session"})
	if b.listCalls != 1 {
		t.Fatalf("the listing hop must run (and fail) exactly once, got %d", b.listCalls)
	}
	if m.chat.SessionPickerOpen() {
		t.Fatalf("a failed listing must close the picker")
	}
	last := lastChat(t, m)
	for _, want := range []string{
		"session: ses-live-9 (primary)",
		"session picker unavailable: opencode serve unreachable",
	} {
		if !strings.Contains(last.Text, want) {
			t.Fatalf("the error fallback must contain %q:\n%s", want, last.Text)
		}
	}
}

// (f-3) a resume-seam failure surfaces the error and wipes NOTHING.
func TestSessionPickerResumeError(t *testing.T) {
	scratchHome(t)
	b := &sessBackend{
		primary: "ses-alpha-new", rows: sessRowsFixture(),
		resumeErr: errors.New("session ses-beta-older not found server-side"),
	}
	m := New(b, nil)
	m = runMsg(t, m, state.Event{Kind: state.EvChatUser,
		Msg: state.ChatMsg{ID: "u1", From: "user", Kind: "user", Text: "keep me", At: 1}})
	m = runMsg(t, m, slashMsg{text: "/session"})
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))

	last := lastChat(t, m)
	if last.Meta != "error" || !strings.Contains(last.Text, "/session: could not resume ses-beta-older") {
		t.Fatalf("a seam failure must be the red error line: meta=%q text=%q", last.Meta, last.Text)
	}
	if !chatHas(m, "keep me") {
		t.Fatalf("a failed accept must not wipe the transcript")
	}
	if len(b.resumed) != 0 {
		t.Fatalf("the failing call returns the error before the primary moves: %v", b.resumed)
	}
}

// (g) ESC: zero side effects — picker closed, no notice, no seam call.
func TestSessionPickerEscZeroEffects(t *testing.T) {
	scratchHome(t)
	b := &sessBackend{primary: "ses-alpha-new", rows: sessRowsFixture()}
	m := New(b, nil)
	m = runMsg(t, m, slashMsg{text: "/session"})
	if !m.chat.SessionPickerOpen() {
		t.Fatalf("precondition: the picker is open")
	}
	before := len(m.st.Chat)
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if m.chat.SessionPickerOpen() {
		t.Fatalf("esc must close the picker")
	}
	if len(m.st.Chat) != before {
		t.Fatalf("esc appends NOTHING (zero side effects), chat %d -> %d", before, len(m.st.Chat))
	}
	if len(b.resumed) != 0 {
		t.Fatalf("esc must never re-anchor: %v", b.resumed)
	}
}
