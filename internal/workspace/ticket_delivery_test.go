package workspace

import (
	"errors"
	"strings"
	"testing"
)

func TestTicketHandoffPreservesStateAndRejectsStaleEdits(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	dir := t.TempDir()
	f, err := PutTicket(dir, Ticket{Title: "Team invitations", Status: "backlog", Priority: "P1", Checklist: []Check{{Text: "Expired links rejected"}}})
	if err != nil {
		t.Fatal(err)
	}
	before := f.Tickets[0]
	f, err = LinkTicket(dir, before.ID, "codex", "session-1", before.Updated)
	if err != nil {
		t.Fatal(err)
	}
	linked := f.Tickets[0]
	if linked.Status != before.Status || linked.Checklist[0].Done || linked.Backend != "codex" || linked.Session != "session-1" || linked.Updated <= before.Updated {
		t.Fatal(linked)
	}
	if _, err = PutTicket(dir, before); !errors.Is(err, ErrTicketConflict) {
		t.Fatalf("stale edit: %v", err)
	}
	if _, err = LinkTicket(dir, before.ID, "opencode", "session-2", before.Updated); !errors.Is(err, ErrTicketConflict) {
		t.Fatalf("stale link: %v", err)
	}
	linked.Result, linked.Verification = "Invitations delivered", "go test ./... passed; expires after 24h"
	f, err = PutTicket(dir, linked)
	if err != nil || f.Tickets[0].Result != linked.Result || f.Tickets[0].Verification != linked.Verification || f.Tickets[0].Session != "session-1" {
		t.Fatal(f, err)
	}
	if !strings.Contains(TicketPrompt(linked), "[ ] Expired links rejected") || !strings.Contains(TicketPrompt(linked), "Do not claim checks passed") {
		t.Fatal(TicketPrompt(linked))
	}
}
