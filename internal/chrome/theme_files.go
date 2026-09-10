package chrome

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxThemeBytes = 2 << 20

type tokenSetting struct {
	Scope    json.RawMessage `json:"scope,omitempty"`
	Settings struct {
		Foreground string `json:"foreground,omitempty"`
		FontStyle  string `json:"fontStyle,omitempty"`
	} `json:"settings"`
}

// vscodeTheme deliberately reads data only. Extensions, JavaScript and remote
// includes are never executed or installed by the importer.
type vscodeTheme struct {
	Name        string            `json:"name"`
	Type        string            `json:"type,omitempty"`
	Include     string            `json:"include,omitempty"`
	Colors      map[string]string `json:"colors"`
	TokenColors []tokenSetting    `json:"tokenColors,omitempty"`
}

var userThemes = map[string]Theme{}
var userThemeDocs = map[string]vscodeTheme{}
var loadedThemeDir string

func UserThemeDir() string { return filepath.Join(themeConfigDir(), "theboringfloor", "themes") }

// LoadUserThemes also reports invalid files; one broken palette cannot hide
// the other imports. Selection/listing load once per config directory.
func LoadUserThemes() []error {
	dir := UserThemeDir()
	loadedThemeDir = dir
	userThemes, userThemeDocs = map[string]Theme{}, map[string]vscodeTheme{}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return []error{err}
	}
	var problems []error
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		count++
		if count > 128 {
			problems = append(problems, fmt.Errorf("theme library is limited to 128 files"))
			break
		}
		path := filepath.Join(dir, entry.Name())
		doc, err := readVSCodeTheme(path, map[string]bool{}, 0)
		if err == nil {
			name := strings.TrimSuffix(entry.Name(), ".json")
			var t Theme
			t, err = mapVSCodeTheme(doc, name)
			if err == nil && !strings.HasPrefix(name, "custom-") {
				err = fmt.Errorf("saved theme filenames must start with custom-; use /theme import to add a theme")
			}
			if err == nil {
				registerFloorPalette(t)
				userThemes[name], userThemeDocs[name] = t, doc
			}
		}
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", entry.Name(), err))
		}
	}
	return problems
}

func ensureUserThemes() {
	if loadedThemeDir != UserThemeDir() {
		LoadUserThemes()
	}
}

func customThemeNames() []string {
	ensureUserThemes()
	names := make([]string, 0, len(userThemes))
	for name := range userThemes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ImportTheme saves a flattened, validated copy. Updating the same custom
// name is atomic; bundled palettes cannot be overwritten by an import.
func ImportTheme(path string) (string, error) {
	path, err := themePath(path)
	if err != nil {
		return "", err
	}
	doc, err := readVSCodeTheme(path, map[string]bool{}, 0)
	if err != nil {
		return "", err
	}
	name := doc.Name
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	name = "custom-" + strings.TrimPrefix(themeSlug(name), "custom-")
	doc.Name, doc.Include = name, ""
	t, err := mapVSCodeTheme(doc, name)
	if err != nil {
		return "", err
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	if len(body)+1 > maxThemeBytes {
		return "", fmt.Errorf("combined theme exceeds 2 MiB")
	}
	entries, err := os.ReadDir(UserThemeDir())
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	count, replacing := 0, false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			count++
			replacing = replacing || entry.Name() == name+".json"
		}
	}
	if count >= 128 && !replacing {
		return "", fmt.Errorf("theme library is limited to 128 files; remove an unused theme first")
	}
	if err := os.MkdirAll(UserThemeDir(), 0o755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(UserThemeDir(), ".theme-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(body, '\n')); err != nil {
		f.Close()
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(f.Name(), filepath.Join(UserThemeDir(), name+".json")); err != nil {
		return "", err
	}
	ensureUserThemes()
	userThemes[name], userThemeDocs[name] = t, doc
	registerFloorPalette(t)
	return name, nil
}

// ExportTheme uses VS Code's own schema as the customization format.
// Existing files are rejected so an export never destroys an edited theme.
func ExportTheme(path string) error {
	path, err := themePath(path)
	if err != nil {
		return err
	}
	t := CurrentTheme()
	doc := vscodeTheme{Name: "My " + t.Name, Type: "dark", Colors: map[string]string{
		"editor.background": hexColor(t.PanelBg), "editor.foreground": hexColor(t.White),
		"statusBar.background": hexColor(t.BarBg), "panel.border": hexColor(t.Border),
		"descriptionForeground": hexColor(t.Dim), "focusBorder": hexColor(t.Accent),
		"button.background": hexColor(t.Accent), "button.foreground": hexColor(t.Black),
		"terminal.ansiRed": hexColor(t.Err), "terminal.ansiGreen": hexColor(t.OK),
		"terminal.ansiCyan": hexColor(t.Info), "terminal.ansiMagenta": hexColor(t.Magenta),
		"terminal.ansiBlue": hexColor(t.Blue), "terminal.ansiYellow": hexColor(t.Warn),
		"diffEditor.insertedTextBackground": hexColor(t.DiffAddBg),
		"diffEditor.removedTextBackground":  hexColor(t.DiffDelBg),
	}}
	if !t.Dark {
		doc.Type = "light"
	}
	ensureUserThemes()
	if saved, ok := userThemeDocs[t.Name]; ok {
		doc.TokenColors = saved.TokenColors
	}
	if len(doc.TokenColors) == 0 {
		doc.TokenColors = defaultTokenSettings(t)
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(body, '\n'))
	closeErr := f.Close()
	if err != nil {
		os.Remove(path)
		return err
	}
	return closeErr
}

func themePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if len(path) >= 2 && ((path[0] == '"' && path[len(path)-1] == '"') || (path[0] == '\'' && path[len(path)-1] == '\'')) {
		path = path[1 : len(path)-1]
	}
	if path == "" {
		return "", fmt.Errorf("provide a local theme file path")
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	if strings.Contains(path, "://") {
		return "", fmt.Errorf("use a local VS Code .json or .jsonc theme file")
	}
	return filepath.Abs(path)
}

func readVSCodeTheme(path string, seen map[string]bool, depth int) (vscodeTheme, error) {
	var doc vscodeTheme
	if depth >= 8 {
		return doc, fmt.Errorf("theme includes exceed 8 levels")
	}
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return doc, err
	}
	if seen[canonical] {
		return doc, fmt.Errorf("cyclic theme include: %s", filepath.Base(path))
	}
	seen[canonical] = true
	defer delete(seen, canonical)
	info, err := os.Stat(canonical)
	if err != nil {
		return doc, err
	}
	if !info.Mode().IsRegular() {
		return doc, fmt.Errorf("theme must be a regular file")
	}
	f, err := os.Open(canonical)
	if err != nil {
		return doc, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxThemeBytes+1))
	if err != nil {
		return doc, err
	}
	if len(data) > maxThemeBytes {
		return doc, fmt.Errorf("theme exceeds 2 MiB")
	}
	clean, err := stripJSONC(data)
	if err != nil {
		return doc, err
	}
	if err = json.Unmarshal(clean, &doc); err != nil {
		return doc, fmt.Errorf("invalid VS Code JSON theme: %w", err)
	}
	if doc.Include != "" {
		if filepath.IsAbs(doc.Include) || strings.Contains(doc.Include, "://") {
			return doc, fmt.Errorf("theme includes must use relative local paths")
		}
		base, err := readVSCodeTheme(filepath.Join(filepath.Dir(canonical), doc.Include), seen, depth+1)
		if err != nil {
			return doc, err
		}
		if base.Colors == nil {
			base.Colors = map[string]string{}
		}
		for key, value := range doc.Colors {
			base.Colors[key] = value
		}
		doc.Colors = base.Colors
		doc.TokenColors = append(base.TokenColors, doc.TokenColors...)
		if doc.Type == "" {
			doc.Type = base.Type
		}
	}
	return doc, nil
}

func mapVSCodeTheme(doc vscodeTheme, name string) (Theme, error) {
	if len(doc.Colors) == 0 && len(doc.TokenColors) == 0 {
		return Theme{}, fmt.Errorf("no VS Code colors or tokenColors found")
	}
	dark := !strings.Contains(strings.ToLower(doc.Type), "light")
	base := basePalette(dark)
	if s := doc.Colors["editor.background"]; s != "" {
		bg, err := parseThemeColor(s, base.PanelBg)
		if err != nil {
			return Theme{}, fmt.Errorf("editor.background: %w", err)
		}
		dark = luminance(bg) < .5
		base = basePalette(dark)
		base.PanelBg = bg
	}
	colors := map[string]color.Color{}
	for key, value := range doc.Colors {
		if value == "" {
			continue
		}
		c, err := parseThemeColor(value, base.PanelBg)
		if err != nil {
			return Theme{}, fmt.Errorf("%s: %w", key, err)
		}
		colors[key] = c
	}
	pick := func(fallback color.Color, keys ...string) color.Color {
		for _, key := range keys {
			if c := colors[key]; c != nil {
				return c
			}
		}
		return fallback
	}
	t := base
	t.Name, t.Dark = name, dark
	t.White = pick(t.White, "editor.foreground", "foreground")
	t.BarBg = pick(mix(t.White, t.PanelBg, .08), "statusBar.background", "titleBar.activeBackground", "sideBar.background")
	t.Border = pick(mix(t.White, t.PanelBg, .3), "panel.border", "sideBar.border", "contrastBorder")
	t.Dim = pick(mix(t.White, t.PanelBg, .6), "descriptionForeground", "editorLineNumber.foreground")
	t.Accent = pick(t.Accent, "focusBorder", "button.background", "terminal.ansiBrightCyan", "terminal.ansiCyan")
	t.Err = pick(t.Err, "terminal.ansiRed", "errorForeground", "editorError.foreground")
	t.OK = pick(t.OK, "terminal.ansiGreen", "gitDecoration.addedResourceForeground")
	t.Info = pick(t.Info, "terminal.ansiCyan", "editorInfo.foreground")
	t.Magenta = pick(t.Magenta, "terminal.ansiMagenta")
	t.Blue = pick(t.Blue, "terminal.ansiBlue", "textLink.foreground")
	t.Warn = pick(t.Warn, "terminal.ansiYellow", "editorWarning.foreground")
	t.Question = t.Warn
	finishPalette(&t)
	t.Black = pick(t.Black, "button.foreground")
	t.DiffAddBg = pick(t.DiffAddBg, "diffEditor.insertedTextBackground", "diffEditor.insertedLineBackground")
	t.DiffDelBg = pick(t.DiffDelBg, "diffEditor.removedTextBackground", "diffEditor.removedLineBackground")
	data, _ := json.Marshal(doc)
	sum := sha256.Sum256(data)
	t.Revision = hex.EncodeToString(sum[:8])
	if err := applyTokenSettings(&t, doc.TokenColors); err != nil {
		return Theme{}, err
	}
	return t, nil
}

func themeSlug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
		if b.Len() >= 64 {
			break
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "theme"
	}
	return s
}

func hexColor(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func parseThemeColor(s string, bg color.Color) (color.Color, error) {
	if !strings.HasPrefix(s, "#") {
		return nil, fmt.Errorf("expected a hex color, got %q", s)
	}
	s = s[1:]
	if len(s) == 3 || len(s) == 4 {
		var b strings.Builder
		for _, r := range s {
			b.WriteRune(r)
			b.WriteRune(r)
		}
		s = b.String()
	}
	if len(s) != 6 && len(s) != 8 {
		return nil, fmt.Errorf("invalid hex color length")
	}
	data, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid hex color")
	}
	c := color.RGBA{R: data[0], G: data[1], B: data[2], A: 255}
	if len(data) == 4 {
		return mix(c, bg, float64(data[3])/255), nil
	}
	return c, nil
}

// Strip comments and trailing commas without interpreting comment markers or
// escaped quotes inside strings. JSON remains responsible for validation.
func stripJSONC(data []byte) ([]byte, error) {
	data = []byte(strings.TrimPrefix(string(data), "\ufeff"))
	out := append([]byte(nil), data...)
	inString, escaped := false, false
	for i := 0; i < len(out); i++ {
		c := out[i]
		if inString {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			continue
		}
		if c != '/' || i+1 >= len(out) {
			continue
		}
		if out[i+1] == '/' {
			for i < len(out) && out[i] != '\n' {
				out[i] = ' '
				i++
			}
		} else if out[i+1] == '*' {
			out[i], out[i+1] = ' ', ' '
			i += 2
			for i+1 < len(out) && !(out[i] == '*' && out[i+1] == '/') {
				if out[i] != '\n' {
					out[i] = ' '
				}
				i++
			}
			if i+1 >= len(out) {
				return nil, fmt.Errorf("unterminated JSONC comment")
			}
			out[i], out[i+1] = ' ', ' '
			i++
		}
	}
	inString, escaped = false, false
	for i, c := range out {
		if inString {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			continue
		}
		if c != ',' {
			continue
		}
		j := i + 1
		for j < len(out) && strings.ContainsRune(" \t\r\n", rune(out[j])) {
			j++
		}
		if j < len(out) && (out[j] == '}' || out[j] == ']') {
			out[i] = ' '
		}
	}
	return out, nil
}
