package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFloorTeamsAndTicketsPersistIndependently(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	project := t.TempDir()
	f, err := Register(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Teams) != 4 {
		t.Fatal(f.Teams)
	}
	if _, err = AddTeam(project, "Platform"); err != nil {
		t.Fatal(err)
	}
	f, err = PutTicket(project, Ticket{Title: "Ship file explorer", Status: "blocked", Priority: "P1", Team: "frontend", Checklist: []Check{{Text: "supports symlinks"}}})
	if err != nil {
		t.Fatal(err)
	}
	ticket := f.Tickets[0]
	ticket.Status = "review"
	ticket.Checklist[0].Done = true
	if _, err = PutTicket(project, ticket); err != nil {
		t.Fatal(err)
	}
	f, err = Load(project)
	if err != nil || len(f.Teams) != 5 || len(f.Tickets) != 1 || !f.Tickets[0].Checklist[0].Done || f.Tickets[0].Status != "review" {
		t.Fatalf("roundtrip: %+v %v", f, err)
	}
	other := t.TempDir()
	g, err := Register(other)
	if err != nil || len(g.Tickets) != 0 {
		t.Fatalf("leaked tickets: %+v %v", g, err)
	}
	if _, err = PutTicket(project, Ticket{Title: "bad team", Status: "backlog", Priority: "P2", Team: "missing"}); err == nil {
		t.Fatal("accepted unknown team")
	}
	if _, err = AddTeam(project, "platform"); err == nil {
		t.Fatal("accepted duplicate team")
	}
}
func TestConversationMetadataDoesNotReadTranscript(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	dir := t.TempDir()
	base := ConversationDir(dir, "codex", "same/id")
	if err := WriteJSON(filepath.Join(base, "meta.json"), Conversation{ID: "same/id", Backend: "codex", Title: "Preview", Messages: 20}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "session.json"), []byte("not valid json"), 0600); err != nil {
		t.Fatal(err)
	}
	rows, err := Conversations(dir)
	if err != nil || len(rows) != 1 || rows[0].Title != "Preview" {
		t.Fatalf("metadata listing: %+v %v", rows, err)
	}
	if base == ConversationDir(dir, "claudecode", "same/id") {
		t.Fatal("backends share an archive")
	}
}
