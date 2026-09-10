package panels

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/office"
)

func TestImportedThemePickerPreviewAndCancel(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	before := chrome.CurrentTheme().Name
	t.Cleanup(func() { chrome.SetTheme(before); office.SetTheme(before) })
	path := filepath.Join(t.TempDir(), "palette.json")
	if err := os.WriteFile(path, []byte(`{"name":"Custom Picker","colors":{"editor.background":"#102030","focusBorder":"#99ddff"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	name, err := chrome.ImportTheme(path)
	if err != nil {
		t.Fatal(err)
	}
	c := NewChat(nil)
	c.SetSize(80, 30)
	c.slashOpen = true
	c.slashMode = slashModeTheme
	c.refilterSlash()
	idx := slices.Index(c.slashThemes, name)
	if idx < 0 || len(c.slashThemes) < 15 {
		t.Fatalf("picker is missing builtins or import: %v", c.slashThemes)
	}
	c.slashSel = idx
	c.previewTheme()
	if chrome.CurrentTheme().Name != name {
		t.Fatal("import cannot be previewed")
	}
	c.closeSlashPicker(true)
	if chrome.CurrentTheme().Name != before {
		t.Fatal("cancel didn't restore the previous palette")
	}
	if _, _, ok := slashFragmentOf("/theme import /tmp/theme.json"); ok {
		t.Fatal("theme picker intercepted an import path")
	}
}
