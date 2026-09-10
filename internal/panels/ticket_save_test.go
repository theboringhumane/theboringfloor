package panels

import (
	tea "charm.land/bubbletea/v2"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
	"testing"
)

func TestTicketEditorKeepsDraftWhenConcurrentEditWins(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	dir := t.TempDir()
	floor, err := workspace.PutTicket(dir, workspace.Ticket{Title: "Original", Status: "backlog", Priority: "P1"})
	if err != nil {
		t.Fatal(err)
	}
	b := NewTickets(dir, false)
	b.floor = floor
	b.edit(floor.Tickets[0])
	other := floor.Tickets[0]
	other.Title = "Changed on phone"
	if _, err := workspace.PutTicket(dir, other); err != nil {
		t.Fatal(err)
	}
	cmd := b.Update(tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	if cmd == nil || b.form == nil {
		t.Fatal("form closed before save completed")
	}
	result := cmd()
	if !IsWorkspaceResult(result) {
		t.Fatal("save completion would be lost after switching panels")
	}
	b.Update(result)
	if b.form == nil || b.form.value(0) != "Original" || b.form.err == "" || b.saving {
		t.Fatal("conflict discarded the editor")
	}
	saved, _ := workspace.Load(dir)
	if saved.Tickets[0].Title != "Changed on phone" {
		t.Fatal("concurrent edit overwritten")
	}
}
