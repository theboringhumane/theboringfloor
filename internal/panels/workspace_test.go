package panels

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesRejectOutsideSymlinkAndPreviewBinary(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(outside, []byte("private"), 0600)
	os.Symlink(outside, filepath.Join(root, "escape"))
	if _, err := projectFile(root, "escape"); err == nil {
		t.Fatal("followed symlink outside project")
	}
	os.WriteFile(filepath.Join(root, "binary"), []byte{1, 0, 3}, 0600)
	f := NewFiles(root)
	f.SetSize(100, 25)
	f.Update(f.Refresh()())
	f.Update(f.open("binary")())
	if !strings.Contains(f.body, "Binary file") {
		t.Fatal(f.body)
	}
}
func TestWorkspaceFormsKeepInputAndBoardShowsBlocked(t *testing.T) {
	f := NewFloors(t.TempDir(), "codex", true)
	f.SetSize(100, 28)
	f.NewConversation()
	f.Update(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "A new feature"}))
	if f.form.value(0) != "A new feature" || f.form.value(1) != "codex" {
		t.Fatal("form lost title or backend")
	}
	f.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	f.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	if f.form.value(1) != "opencode" {
		t.Fatal("backend choice did not move")
	}
	b := NewTickets(t.TempDir(), true)
	b.SetSize(120, 26)
	b.SetState(state.OfficeState{Tasks: []state.BoardTask{{ID: "stalled", Title: "Waiting for credentials", Status: state.TaskStalled}}})
	if frame := ansi.Strip(b.View()); !strings.Contains(frame, "BLOCKED 1") || !strings.Contains(frame, "Waiting for") {
		t.Fatal(frame)
	}
	b.Update(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))
	b.Update(tea.KeyPressMsg(tea.Key{Code: 'a', Text: "New ticket"}))
	b.Update(tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	if len(b.floor.Tickets) != 1 {
		t.Fatal("ticket was not saved")
	}
}
func TestTranscriptSearchKeepsDraft(t *testing.T) {
	c := NewChat(nil)
	c.SetSize(80, 18)
	c.SetState(state.OfficeState{Chat: []state.ChatMsg{{ID: "one", From: "user", Text: "find the needle"}, {ID: "two", From: "boss", Text: "Found it"}}})
	c.StageDraft("unsent draft")
	c.Update(tea.KeyPressMsg(tea.Key{Code: 'r', Mod: tea.ModCtrl}))
	c.Update(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "needle"}))
	if len(c.searchMatches) != 1 {
		t.Fatalf("matches: %v", c.searchMatches)
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	c.Update(tea.PasteMsg{Content: "e"})
	c.SetSize(45, 18)
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(c.searchMatches) != 1 {
		t.Fatal("search lost its match after paste and resize")
	}
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if c.ta.Value() != "unsent draft" {
		t.Fatal("search modified draft")
	}
}
func TestRemovedBrowserNeverProbesOrSpawns(t *testing.T) {
	t.Setenv(BrowserLaneOptInEnv, "1")
	t.Setenv("TERM_PROGRAM", "ghostty")
	old := zenbuLookPath
	zenbuLookPath = func(string) (string, error) { t.Fatal("probed removed package"); return "", nil }
	defer func() { zenbuLookPath = old }()
	if got, reason, _ := ResolveBrowserLaneReasonFrom(func(string) string { return "1" }, zenbuLookPath); got != BrowserLaneText || reason != BrowserLaneRemoved {
		t.Fatal("legacy resolver revived browser")
	}
	if ResolveBrowserLane() != BrowserLaneText {
		t.Fatal("removed browser selected")
	}
	lane, reason, _ := ResolveBrowserLaneReason()
	if lane != BrowserLaneText || reason != BrowserLaneRemoved {
		t.Fatal(lane, reason)
	}
	if ResolveOpenTool() != OpenToolSystemOpen {
		t.Fatal("external links must use system browser")
	}
	if _, err := newZenbuSession("https://example.com", 80, 24); err == nil {
		t.Fatal("removed browser spawned")
	}
}
