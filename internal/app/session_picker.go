// session_picker.go — the interactive /session picker (the app's half;
// the panel lives in internal/panels/session_picker.go):
//
//   - /session opens the picker over the chat (loading state) while the
//     backend's ListSessions hop rides a tea.Cmd — the input never
//     stalls. Rows = the server's ROOT sessions (parentID == ""), sorted
//     by Updated desc, each showing title (fallback: short id), relative
//     age, message count and short id, the attached one marked.
//   - ACCEPT = QUIT + EXEC-REPLACE: the swap is NOT re-anchored in-app.
//     The accept quits exactly like ctrl+q's own path (pin persisted
//     NOW — stamped with the ACCEPTED id — terminal reaped, transcript
//     row "closing — relaunching as `theboringoffice -s <id>`" recorded)
//     and returns a quit cmd, so cmd's post-Run path can syscall.Exec
//     the same binary as `theboringoffice -s <id>`. The RELAUNCHED
//     boot's resolvePrimary verifies the id server-side (degrades open
//     on a miss) and, stored id == boot pin, hydrates the transcript
//     straight through the swap.
//   - accepting the CURRENT session is a no-op ("already on session …");
//     accepting a DIFFERENT one while the boss has ANY work in flight
//     (pending placeholder/stream, delegation quiet state, parked
//     question turn — routeBusySend's busy triple) is refused with
//     "boss is busy — /stop or wait, then /session again"; esc cancels
//     with zero side effects.
//   - GRACEFUL DEGRADATION: no ListSessions seam (demo, harness stubs)
//     or a failed listing falls back to the static /session summary
//     (current id + session.json path) plus a dim note that the picker
//     is unavailable. Never a crash, never a hung input.
//
// Both backend seams are ADDITIVE (the primarySeamBackend pattern: the
// app type-asserts them, demo + harness stubs never implement them).
package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// --- backend seams (additive; type-asserted, never added to state.Backend)

// sessionListBackend — the picker's listing seam: the server's ROOT
// sessions (parentID == "") with per-session message counts. The live
// backend implements it with a bounded concurrent count fan-out under
// the caller's context; harness stubs (uishot, headless) and the demo
// backend never do — those backends get the static /session summary.
type sessionListBackend interface {
	ListSessions(ctx context.Context) ([]state.SessionRow, error)
}

// officeResumeBackend — the picker's accept CAPABILITY GATE: a backend
// WITH it is a real live backend that the exec-replaced boot can pin a
// session on; a backend WITHOUT it (demo, harness stubs) gets the static
// summary fallback. The resume call itself is kept as backend API (its
// own backend tests intact) but the picker no longer drives it — the
// accept swaps by QUIT + EXEC-REPLACE, and the RELAUNCHED boot's
// resolvePrimary does the server-side verify instead.
type officeResumeBackend interface {
	ResumeOffice(id string) error
}

// --- tea messages ------------------------------------------------------------

// sessionListMsg — the ListSessions hop's landing (a failed listing
// carries err; the picker falls back to the static summary then).
type sessionListMsg struct {
	rows []state.SessionRow
	err  error
}

// sessionPickMsg — the picker accepted a row (its full session id).
type sessionPickMsg struct{ id string }

// sessionPickCancelMsg — esc cancelled the picker (zero side effects).
type sessionPickCancelMsg struct{}

// sessionListTimeout bounds the listing + count fan-out — the picker
// must never hang the input while the server (or network) drags.
const sessionListTimeout = 10 * time.Second

// openSessionPicker — the /session slash handler: open the card in its
// loading state and kick the async ListSessions hop. A backend without
// the listing seam (demo, harness stubs — or any non-live mode) gets the
// static summary plus a dim picker-unavailable note instead (graceful
// degradation, never a dead card).
func (m *Model) openSessionPicker() tea.Cmd {
	lb, ok := m.backend.(sessionListBackend)
	if !ok || m.st.Mode != state.ModeLive || m.chat == nil {
		m.notice(sessionUnavailableNote(m.sessionInfo(), ""))
		return nil
	}
	m.chat.OpenSessionPicker()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), sessionListTimeout)
		defer cancel()
		rows, err := lb.ListSessions(ctx)
		return sessionListMsg{rows: rows, err: err}
	}
}

// sessionUnavailableNote — the static /session body plus the dim note
// that the picker is unavailable (listErr "" covers the no-seam/demo
// backends; non-empty carries the listing failure).
func sessionUnavailableNote(static, listErr string) string {
	if listErr != "" {
		return static + "\n  (session picker unavailable: " + listErr + " — the summary above is all there is)"
	}
	return static + "\n  (session picker unavailable on this backend — the summary above is all there is)"
}

// handleSessionList — the listing hop landed: a failure closes the card
// and prints the static fallback (+ the dim unavailable note); rows fill
// the still-open picker (an esc-cancel while the hop flew wins — the late
// landing is dropped). The attached session's id is re-read HERE so the
// current marker can never go stale between open and fill.
func (m *Model) handleSessionList(msg sessionListMsg) {
	if msg.err != nil {
		if m.chat != nil {
			m.chat.CloseSessionPicker()
		}
		m.notice(sessionUnavailableNote(m.sessionInfo(), msg.err.Error()))
		return
	}
	if m.chat == nil || !m.chat.SessionPickerOpen() {
		return // esc'd (or fallback-closed) while the hop was in flight
	}
	current := ""
	if ps, ok := m.backend.(primarySeamBackend); ok {
		current = ps.PrimaryID()
	}
	m.chat.SetSessionPickerRows(buildSessionRows(msg.rows, current, time.Now()))
}

// acceptSessionPick — the picker accepted a session id (the picker is
// closed HERE — every accept path ends it, per the frozen contract).
//
// ACCEPT = QUIT + EXEC-REPLACE: the office swaps sessions by QUITTING,
// not by re-anchoring in-app — the accept records the relaunch intent on
// the model (execSession; cmd's post-Run path syscall.Exec's the binary
// as `theboringoffice -s <id>`), persists the pin into session.json NOW
// (sync, stamped with the ACCEPTED id), reaps the terminal and returns
// the quit cmd. The resume seam is only the capability gate (a backend
// without it can't be re-pinned by a boot — static summary fallback);
// the accepted id is verified server-side by the RELAUNCHED boot's
// resolvePrimary, which degrades open on a miss. The no-op / refusal /
// fallback paths return nil — nothing to quit for.
func (m *Model) acceptSessionPick(id string) tea.Cmd {
	if m.chat != nil {
		m.chat.CloseSessionPicker()
	}
	current := ""
	if ps, ok := m.backend.(primarySeamBackend); ok {
		current = ps.PrimaryID()
	}
	// accepting the CURRENT session is a harmless no-op (no quit).
	if id == current {
		m.notice("already on session " + id)
		return nil
	}
	// BUSY BLOCK: a boss turn in flight — pending placeholder/stream, the
	// delegation quiet state, or a parked question turn — must never lose
	// its session mid-turn (routeBusySend's exact busy triple). Refused,
	// picker already closed, nothing else changes.
	if hasPendingBoss(m.st) || m.st.BossDelegating || m.questionParked {
		m.notice("boss is busy — /stop or wait, then /session again")
		return nil
	}
	// a backend that can list but not re-anchor: the static fallback —
	// same graceful-degradation rule as the missing list seam.
	if _, ok := m.backend.(officeResumeBackend); !ok {
		m.notice(sessionUnavailableNote(m.sessionInfo(), "this backend cannot re-anchor live"))
		return nil
	}
	// record the exec intent for main's post-Run relaunch (id + nothing
	// else — binary lookup + flag carry-forward are cmd's business).
	m.execSession = id
	// G6 — RE-ANCHOR CLEAR: strip every Pending row from the transcript
	// BEFORE the pin below persists it (mirror of hydrateSession's
	// restore-leg loop in sessions.go: bubbles of a turn the quit kills
	// are ghosts — a persisted one would re-hydrate as a stuck "typing…"
	// placeholder nothing will ever complete; the busy block above
	// already refuses boss-pending anchors, this is the belt-and-braces
	// clear for anything else, e.g. an outstanding concierge placeholder).
	kept := make([]state.ChatMsg, 0, len(m.st.Chat))
	for _, c := range m.st.Chat {
		if c.Pending {
			continue
		}
		kept = append(kept, c)
	}
	m.st.Chat = kept
	// the relaunch rides the transcript as REAL history — persisted with
	// the pin below, the new boot hydrates right through the swap.
	m.notice(fmt.Sprintf("closing — relaunching as `theboringoffice -s %s`", id))
	// the pin must land in session.json NOW (next boot auto-restores):
	// the 5s cheap-write loop would lag the exec, so the forced sync
	// write goes explicitly (small + bounded) with the ACCEPTED id as the
	// primary — persistOfficeSession would stamp the still-current one.
	m.persistOfficePin(id)
	m.closeTerminal()
	return tea.Quit
}

// --- row building (pure — fake-driven unit tests pin it) ---------------------

// buildSessionRows turns the backend's listing into the picker's rows:
// ROOT sessions only (parentID == "" — the seam already filters, this is
// belt-and-braces), sorted by Updated desc (Created breaks ties, the id
// orders those), the attached session marked, blank titles fallen back
// to the short id. now drives the relative age (deterministic tests).
func buildSessionRows(rows []state.SessionRow, currentID string, now time.Time) []panels.SessionPickRow {
	roots := make([]state.SessionRow, 0, len(rows))
	for _, r := range rows {
		if r.ParentID == "" {
			roots = append(roots, r)
		}
	}
	sort.SliceStable(roots, func(i, j int) bool {
		if roots[i].Updated != roots[j].Updated {
			return roots[i].Updated > roots[j].Updated
		}
		if roots[i].Created != roots[j].Created {
			return roots[i].Created > roots[j].Created
		}
		return roots[i].ID < roots[j].ID
	})
	out := make([]panels.SessionPickRow, 0, len(roots))
	for _, r := range roots {
		short := shortSessionID(r.ID)
		title := strings.Join(strings.Fields(r.Title), " ") // flatten newlines/gaps
		if title == "" {
			title = short
		}
		out = append(out, panels.SessionPickRow{
			ID:       r.ID,
			Title:    title,
			Age:      relAge(now, r.Updated),
			Messages: r.Messages,
			ShortID:  short,
			Current:  r.ID == currentID && currentID != "",
		})
	}
	return out
}

// shortSessionID is the row's compact id form: intact when short, else
// the leading 8 runes (git-style short form; the full id still rides the
// accept payload + the resume notice).
func shortSessionID(id string) string {
	r := []rune(id)
	if len(r) <= 8 {
		return id
	}
	return string(r[:8])
}

// relAge is the row's machine age: coarse units, no "ago" (narrow meta
// column). updatedMs<=0 or a future stamp (clock skew) reads "now".
func relAge(now time.Time, updatedMs int64) string {
	if updatedMs <= 0 {
		return "now"
	}
	d := now.Sub(time.UnixMilli(updatedMs))
	switch {
	case d < 10*time.Second:
		return "now"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 14*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	}
}
