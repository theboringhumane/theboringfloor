package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/office"
)

func TestThemeImportCommandsAndCacheRefresh(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := cockpitModel(t)
	m.resize(168, 46)
	dir := t.TempDir()
	path := filepath.Join(dir, "My theme.jsonc")
	body := `{"name":"Mission","colors":{"editor.background":"#102030","editor.foreground":"#ddecff","focusBorder":"#91deff","terminal.ansiGreen":"#72ffaa"}}`
	write := func(path, text string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(path, body)
	m = runMsg(t, m, slashMsg{text: "/theme import \"" + path + "\""})
	if chrome.CurrentTheme().Name != "custom-mission" || chrome.LoadPersistedTheme() != "custom-mission" {
		t.Fatal("slash import did not select and persist the palette")
	}
	m.chat.StageDraft("Keep this draft")
	before := m.Frame()
	for _, text := range []string{"COMMAND DECK", "TACTICAL FLOOR", "OPERATIONS", "Keep this draft"} {
		if !strings.Contains(ansi.Strip(before), text) {
			t.Fatalf("imported theme lost cockpit content %q", text)
		}
	}
	key := chrome.ThemeKey()
	_, floorMisses := office.CacheStats()
	// Change only an ANSI color: neither theme name nor base/background ink
	// changes, and no app message/nonce is available to invalidate the frame.
	write(path, strings.ReplaceAll(body, "#72ffaa", "#66ddbb"))
	if _, err := chrome.ImportTheme(path); err != nil {
		t.Fatal(err)
	}
	chrome.SetTheme("custom-mission")
	office.SetTheme("custom-mission")
	_, misses := m.FrameCacheStats()
	m.Frame()
	_, nextMisses := m.FrameCacheStats()
	if chrome.ThemeKey() == key || nextMisses != misses+1 {
		t.Fatal("same-name reimport reused stale frame")
	}
	if _, next := office.CacheStats(); next <= floorMisses {
		t.Fatal("same-name reimport reused stale floor colors")
	}
	exported := filepath.Join(dir, "exported palette.json")
	m = runMsg(t, m, slashMsg{text: "/theme export '" + exported + "'"})
	if _, err := os.Stat(exported); err != nil {
		t.Fatal(err)
	}
	saved := filepath.Join(chrome.UserThemeDir(), "custom-mission.json")
	write(saved, strings.ReplaceAll(body, "#102030", "#162738"))
	m = runMsg(t, m, slashMsg{text: "/theme reload"})
	frame := m.Frame()
	if !strings.Contains(frame, "48;2;22;39;56") || !strings.Contains(ansi.Strip(frame), "Keep this draft") {
		t.Fatal("reload didn't repaint while preserving the draft")
	}
	key = chrome.ThemeKey()
	write(path, "bad theme")
	m = runMsg(t, m, slashMsg{text: "/theme import " + path})
	if chrome.ThemeKey() != key || chrome.LoadPersistedTheme() != "custom-mission" {
		t.Fatal("failed slash import changed the selected theme")
	}
}
