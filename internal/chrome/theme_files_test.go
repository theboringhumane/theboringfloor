package chrome

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	chstyles "github.com/alecthomas/chroma/v2/styles"
)

func themeLibrary(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	before, wasPinned := current, pinned
	t.Cleanup(func() {
		current, pinned = before, wasPinned
		applyTheme(before)
		loadedThemeDir = ""
		userThemes, userThemeDocs = map[string]Theme{}, map[string]vscodeTheme{}
	})
	LoadUserThemes()
	return t.TempDir()
}

func writeTheme(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestImportThemeJSONCIncludesAndSyntax(t *testing.T) {
	dir := themeLibrary(t)
	writeTheme(t, filepath.Join(dir, "base.json"), `{
		"type": "light",
		"colors": {"editor.background":"#fff", "editor.foreground":"#123456", "terminal.ansiGreen":"#123"},
		"tokenColors":[{"scope":"comment", "settings":{"foreground":"#345678"}}]
	}`)
	path := filepath.Join(dir, "My VS Code theme.jsonc")
	writeTheme(t, path, string(rune(0xfeff))+`{
		// Comments and trailing commas are allowed.
		"name": "My // cockpit /* theme */",
		"include": "./base.json",
		"colors": {
			"focusBorder": "#789",
			"terminal.ansiRed": "#ff000080", /* composited */
		},
		"tokenColors": [
			{"scope": ["keyword", "storage.type"], "settings":{"foreground":"#ab1234", "fontStyle":"bold italic"}},
			{"scope": "comment", "settings":{"foreground":"#456789"}},
		],
	}`)
	before := ThemeKey()
	name, err := ImportTheme(`"` + path + `"`)
	if err != nil {
		t.Fatal(err)
	}
	if name != "custom-my-cockpit-theme" || ThemeKey() != before {
		t.Fatalf("import name=%q changed active theme=%t", name, ThemeKey() != before)
	}
	if !SetTheme(name) || current.Dark {
		t.Fatal("imported light palette cannot be selected")
	}
	for label, got := range map[string]string{
		"background": hexColor(current.PanelBg), "foreground": hexColor(current.White),
		"accent": hexColor(current.Accent), "red": hexColor(current.Err),
	} {
		want := map[string]string{"background": "#ffffff", "foreground": "#123456", "accent": "#778899", "red": "#ff7f7f"}[label]
		if got != want {
			t.Errorf("%s=%s, want %s", label, got, want)
		}
	}
	style := chstyles.Get(current.ChromaStyle)
	if got := style.Get(chroma.Keyword); got.Colour.String() != "#ab1234" || got.Bold != chroma.Yes || got.Italic != chroma.Yes {
		t.Fatalf("keyword style lost: %#v", got)
	}
	if got := style.Get(chroma.Comment).Colour.String(); got != "#456789" {
		t.Fatalf("child token color didn't override include: %s", got)
	}
	saved := filepath.Join(UserThemeDir(), name+".json")
	data, err := os.ReadFile(saved)
	if err != nil {
		t.Fatal(err)
	}
	var doc vscodeTheme
	if err := json.Unmarshal(data, &doc); err != nil || doc.Include != "" {
		t.Fatalf("saved theme must be standalone JSON: %v, include=%q", err, doc.Include)
	}
	if err := PersistTheme(); err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(dir, "base.json"))
	os.Remove(path)
	if errors := LoadUserThemes(); len(errors) != 0 || !SetTheme(LoadPersistedTheme()) {
		t.Fatalf("standalone persisted import did not reload: %v", errors)
	}
}

func TestThemeReimportExportAndBuiltinProtection(t *testing.T) {
	dir := themeLibrary(t)
	path := filepath.Join(dir, "source.json")
	writeTheme(t, path, `{"name":"cockpit","colors":{"editor.background":"#112233","focusBorder":"#88aabb"}}`)
	name, err := ImportTheme(path)
	if err != nil || name != "custom-cockpit" {
		t.Fatalf("builtin collision: %q %v", name, err)
	}
	builtin := themes["cockpit"]
	SetTheme(name)
	firstKey := ThemeKey()
	writeTheme(t, path, `{"name":"cockpit","colors":{"editor.background":"#112233","focusBorder":"#ccddff"}}`)
	if updated, err := ImportTheme(path); err != nil || updated != name {
		t.Fatalf("reimport: %s %v", updated, err)
	}
	SetTheme(name)
	if ThemeKey() == firstKey || hexColor(current.Accent) != "#ccddff" {
		t.Fatal("same-name import did not change palette/cache identity")
	}
	if hexColor(themes["cockpit"].Accent) != hexColor(builtin.Accent) {
		t.Fatal("import overwrote builtin")
	}
	exported := filepath.Join(dir, "editable theme.json")
	if err := ExportTheme("'" + exported + "'"); err != nil {
		t.Fatal(err)
	}
	if err := ExportTheme(exported); err == nil {
		t.Fatal("export overwrote existing file")
	}
	data, _ := os.ReadFile(exported)
	writeTheme(t, exported, strings.ReplaceAll(string(data), "#ccddff", "#aaeeff"))
	edited, err := ImportTheme(exported)
	if err != nil || !SetTheme(edited) || hexColor(current.Accent) != "#aaeeff" {
		t.Fatalf("export/edit/import: %q %v", edited, err)
	}
	if !slices.Contains(ThemeNames(), name) || !slices.Contains(ThemeNames(), edited) {
		t.Fatal("imports missing from theme picker names")
	}
	if errors := LoadUserThemes(); len(errors) != 0 || !SetTheme(name) || hexColor(current.Accent) != "#ccddff" {
		t.Fatalf("updated palette was not persisted: %v", errors)
	}
}

func TestFailedThemeImportPreservesPaletteAndSavedFile(t *testing.T) {
	dir := themeLibrary(t)
	path := filepath.Join(dir, "theme.json")
	writeTheme(t, path, `{"name":"steady","colors":{"editor.background":"#123456"}}`)
	name, err := ImportTheme(path)
	if err != nil {
		t.Fatal(err)
	}
	SetTheme(name)
	key := ThemeKey()
	saved := filepath.Join(UserThemeDir(), name+".json")
	original, _ := os.ReadFile(saved)
	for _, body := range []string{
		`{"name":"steady","colors":{"editor.background":"oops"}}`,
		`{"name":"steady","colors":{"focusBorder":"#xyz"}}`,
		`{"name":"steady","colors":{}}`,
		`{"name":"steady", "include":"theme.json"}`,
		`{"name":"steady", "include":"missing.json"}`,
		`{"name":"steady", "include":"https://example.com/theme.json"}`,
		`{"name":"steady", "tokenColors":"theme.tmTheme"}`,
		`{"name":"steady", "tokenColors":[{"scope":42,"settings":{"foreground":"#fff"}}]}`,
		`{"name":"steady", "tokenColors":[{"scope":"string","settings":{"foreground":"no"}}]}`,
		`{"colors":{"editor.background":"#fff"}, /* unfinished`,
		`{"colors":{"editor.background":"#fff"`,
		strings.Repeat(" ", maxThemeBytes+1),
	} {
		writeTheme(t, path, body)
		if _, err := ImportTheme(path); err == nil {
			t.Fatalf("invalid theme accepted: %.100s", body)
		}
		data, _ := os.ReadFile(saved)
		if string(data) != string(original) || ThemeKey() != key || !SetTheme(name) || ThemeKey() != key {
			t.Fatal("failed import changed active/saved theme")
		}
	}
	// A failed filesystem write must not register a new palette.
	blocked := filepath.Join(dir, "blocked")
	writeTheme(t, blocked, "file, not directory")
	t.Setenv("XDG_CONFIG_HOME", blocked)
	writeTheme(t, path, `{"name":"unsaved","colors":{"editor.background":"#123456"}}`)
	if _, err := ImportTheme(path); err == nil || SetTheme("custom-unsaved") {
		t.Fatal("failed save registered a palette")
	}
}

func TestThemeLibraryReadLimitsAndPartialReload(t *testing.T) {
	dir := themeLibrary(t)
	for i := 0; i < 9; i++ {
		body := `{"colors":{"editor.background":"#123456"}}`
		if i < 8 {
			body = fmt.Sprintf(`{"include":"level-%d.json"}`, i+1)
		}
		writeTheme(t, filepath.Join(dir, fmt.Sprintf("level-%d.json", i)), body)
	}
	if _, err := ImportTheme(filepath.Join(dir, "level-0.json")); err == nil || !strings.Contains(err.Error(), "8 levels") {
		t.Fatalf("include depth not limited: %v", err)
	}
	if _, err := ImportTheme(dir); err == nil {
		t.Fatal("directory accepted as a theme")
	}
	if _, err := ImportTheme("https://example.com/theme.json"); err == nil {
		t.Fatal("remote file accepted")
	}
	if _, err := ImportTheme(filepath.Join(dir, "level-8.json")); err != nil {
		t.Fatal(err)
	}
	writeTheme(t, filepath.Join(UserThemeDir(), "custom-broken.json"), "broken")
	if errors := LoadUserThemes(); len(errors) != 1 || !SetTheme("custom-level-8") {
		t.Fatalf("one broken file hid valid imports: %v", errors)
	}
}

func TestJSONCStringsAndColorFormats(t *testing.T) {
	clean, err := stripJSONC([]byte(`{"text":"escaped \" // /* ,}", "url":"https://example.com", "list":[1,2,],}`))
	var doc map[string]any
	if err != nil || json.Unmarshal(clean, &doc) != nil || doc["text"] != `escaped " // /* ,}` || doc["url"] != "https://example.com" {
		t.Fatalf("JSONC damaged strings: %s (%v)", clean, err)
	}
	bg := themes["paper"].PanelBg
	for value, want := range map[string]string{"#abc": "#aabbcc", "#abc0": hexColor(bg), "#112233": "#112233", "#11223300": hexColor(bg)} {
		c, err := parseThemeColor(value, bg)
		if err != nil {
			t.Fatal(err)
		}
		if hexColor(c) != want {
			t.Errorf("%s = %s, want %s", value, hexColor(c), want)
		}
	}
}
