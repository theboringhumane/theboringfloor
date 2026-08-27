// browser_open_test.go — the APP leg of the agent browser tool: an
// EvBrowserOpen reaches applyBrowserOpen (the handler the manager's
// one-line applyEvent hookup batches in), an ALLOWED verdict drives the
// browser pane's existing open path (a real localhost fetch through the
// pane's own guard) + posts the dim confirmation notice, a REFUSED
// verdict opens nothing and posts the red reason row, and every other
// event kind no-ops (the batch-leg contract).
package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// drainBrowserCmd — runMsg's BFS for a handler-returned cmd tree (the
// fetch rides a tea.Cmd whose BrowserPageMsg must flow back through
// Update; heartbeats dropped exactly like runMsg).
func drainBrowserCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
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
		case spinner.TickMsg:
		case cursor.BlinkMsg:
		default:
			nm, next := m.Update(out)
			m = nm.(Model)
			if next != nil {
				queue = append(queue, next)
			}
		}
	}
	return m
}

// officeRows — every chat row the office posted (the notice feed).
func officeRows(m Model) []state.ChatMsg {
	var out []state.ChatMsg
	for _, c := range m.st.Chat {
		if c.From == "office" {
			out = append(out, c)
		}
	}
	return out
}

func TestBrowserOpenRequestReachesHandler(t *testing.T) {
	pinBrowserTextLane(t) // hermetic: the lane resolve must never spawn a real child here
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>The Boring Gazette</title></head><body><h1>agent opened me</h1></body></html>`))
	}))
	t.Cleanup(srv.Close)

	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})

	// the allowed request (http://127.0.0.1:* passes BOTH the bridge
	// policy's loopback rule and the pane's own fetch whitelist).
	cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserOpen, Text: srv.URL, BrowserOpenAllowed: true,
	})
	if cmd == nil {
		t.Fatal("an allowed request must return the pane's open cmd")
	}
	m = drainBrowserCmd(t, m, cmd)

	// the pane actually loaded the page (read the pane's own render — no
	// tab-switch assumption, the pane's home is mid-move).
	frame := ansi.Strip(m.browser.View())
	for _, want := range []string{"The Boring Gazette", "agent opened me", srv.URL} {
		if !strings.Contains(frame, want) {
			t.Fatalf("the browser pane must render the fetched page (%q missing):\n%s", want, frame)
		}
	}
	// the dim confirmation notice posted.
	if !lastOfficeNoticeHas(m, "browser: opening "+srv.URL+" (asked by the boss)") {
		t.Fatalf("the confirmation notice must post, office rows: %+v", officeRows(m))
	}
	// and NO red error row landed.
	for _, c := range officeRows(m) {
		if c.Meta == "error" {
			t.Fatalf("an allowed open must never post a red row, got %q", c.Text)
		}
	}
}

func TestBrowserOpenRefusedPostsReason(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	before := len(m.st.Chat)

	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	cmd := m.applyBrowserOpen(state.Event{
		Kind: state.EvBrowserOpen, Text: "http://theboring.name",
		BrowserOpenAllowed: false, BrowserOpenReason: reason,
	})
	if cmd != nil {
		t.Fatal("a refused request must never kick an open cmd")
	}
	if len(m.st.Chat) != before+1 {
		t.Fatalf("exactly ONE notice row lands, got %d new", len(m.st.Chat)-before)
	}
	row := m.st.Chat[len(m.st.Chat)-1]
	if row.From != "office" || row.Meta != "error" {
		t.Fatalf("the refusal is a RED office row, got %+v", row)
	}
	if row.Text != "browser: http://theboring.name — "+reason {
		t.Fatalf("the refusal row carries the bridge's exact reason, got %q", row.Text)
	}
	// the pane stayed idle (the starter card, never the refused URL).
	frame := ansi.Strip(m.browser.View())
	if strings.Contains(frame, "theboring.name") {
		t.Fatalf("a refused URL must never reach the pane, got:\n%s", frame)
	}
}

func TestBrowserOpenHandlerNoopsOnOtherKinds(t *testing.T) {
	m := New(&recBackend{}, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 30})
	before := len(m.st.Chat)

	for _, ev := range []state.Event{
		{Kind: state.EvStatus, Text: "noise"},
		{Kind: state.EvChatBoss, Msg: state.ChatMsg{ID: "bossmsg-x", Text: "hi"}},
		{Kind: state.EvBrowserOpen, Text: "", BrowserOpenAllowed: true}, // shapeless: no URL, silent
	} {
		if cmd := m.applyBrowserOpen(ev); cmd != nil {
			t.Fatalf("kind %q must no-op, got a cmd", ev.Kind)
		}
	}
	if len(m.st.Chat) != before {
		t.Fatalf("no-op kinds must never post notices, got %d new rows", len(m.st.Chat)-before)
	}
}
