package panels

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

type fileEntry struct {
	Path      string
	Dir, Link bool
	Depth     int
}
type filesLoaded struct {
	path    string
	entries []fileEntry
	err     error
}
type filePreview struct {
	path, body string
	err        error
}
type FileAttachMsg struct{ Attachment state.Attachment }
type Files struct {
	root               string
	w, h               int
	entries            []fileEntry
	selected, scroll   int
	expanded           map[string]bool
	preview, body, err string
	loading            bool
	query              string
	searching          bool
}

func NewFiles(root string) *Files           { return &Files{root: root, expanded: map[string]bool{}} }
func (f *Files) Title() string              { return "files" }
func (f *Files) SetSize(w, h int)           { f.w, f.h = w, h }
func (f *Files) SetState(state.OfficeState) {}
func (f *Files) Editing() bool              { return f.searching }
func (f *Files) Refresh() tea.Cmd {
	f.expanded = map[string]bool{}
	f.loading = true
	return f.readDir("")
}

var ignoredDirs = map[string]bool{".git": true, "node_modules": true, "vendor": true, ".dart_tool": true, ".next": true, "build": true, "dist": true, ".venv": true, "__pycache__": true}

func (f *Files) readDir(path string) tea.Cmd {
	root := f.root
	return func() tea.Msg {
		// Resolve every access, including directories: never follow a symlink out
		// of this project or read special devices through a repository entry.
		abs, err := projectFile(root, path)
		if err != nil {
			return filesLoaded{path: path, err: err}
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			return filesLoaded{path: path, err: err}
		}
		var rows []fileEntry
		depth := 0
		if path != "" {
			depth = strings.Count(filepath.ToSlash(path), "/") + 1
		}
		for _, e := range entries {
			if e.IsDir() && ignoredDirs[e.Name()] {
				continue
			}
			rows = append(rows, fileEntry{Path: filepath.Join(path, e.Name()), Dir: e.IsDir(), Link: e.Type()&os.ModeSymlink != 0, Depth: depth})
			if len(rows) >= 10000 {
				break
			}
		}
		sort.Slice(rows, func(i, j int) bool {
			if rows[i].Dir != rows[j].Dir {
				return rows[i].Dir
			}
			return strings.ToLower(rows[i].Path) < strings.ToLower(rows[j].Path)
		})
		return filesLoaded{path: path, entries: rows}
	}
}
func projectFile(root, path string) (string, error) {
	canon, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	canon, err = filepath.Abs(canon)
	if err != nil {
		return "", err
	}
	abs, err := filepath.EvalSymlinks(filepath.Join(canon, path))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(canon, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("file is outside this project")
	}
	return abs, nil
}
func (f *Files) open(path string) tea.Cmd {
	f.preview = path
	f.loading = true
	root := f.root
	return func() tea.Msg {
		abs, err := projectFile(root, path)
		if err != nil {
			return filePreview{path: path, err: err}
		}
		info, err := os.Stat(abs)
		if err != nil {
			return filePreview{path: path, err: err}
		}
		if !info.Mode().IsRegular() {
			return filePreview{path: path, err: fmt.Errorf("preview supports regular text files")}
		}
		file, err := os.Open(abs)
		if err != nil {
			return filePreview{path: path, err: err}
		}
		defer file.Close()
		raw, err := io.ReadAll(io.LimitReader(file, 256*1024+1))
		if err != nil {
			return filePreview{path: path, err: err}
		}
		if bytes.IndexByte(raw, 0) >= 0 || !utf8.Valid(raw) {
			return filePreview{path: path, body: "Binary file · text preview unavailable"}
		}
		truncated := len(raw) > 256*1024
		if truncated {
			raw = raw[:256*1024]
		}
		plain := strings.Map(func(r rune) rune {
			if unicode.IsControl(r) && r != '\n' && r != '\t' {
				return -1
			}
			return r
		}, ansi.Strip(string(raw)))
		plain = strings.ReplaceAll(plain, "\t", "    ")
		var out bytes.Buffer
		if err := quick.Highlight(&out, plain, filepath.Ext(path), "terminal256", "dracula"); err != nil {
			out.WriteString(plain)
		}
		body := out.String()
		if truncated {
			body += "\n… preview limited to 256 KiB"
		}
		return filePreview{path: path, body: body}
	}
}
func (f *Files) visible() []int {
	var ids []int
	for i, e := range f.entries {
		if f.query == "" || strings.Contains(strings.ToLower(e.Path), strings.ToLower(f.query)) {
			ids = append(ids, i)
		}
	}
	return ids
}
func (f *Files) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case filesLoaded:
		f.loading = false
		if m.err != nil {
			f.err = m.err.Error()
			return nil
		}
		f.err = ""
		if m.path == "" {
			f.entries = m.entries
			f.expanded = map[string]bool{}
			f.selected = 0
		} else {
			for i, e := range f.entries {
				if e.Path == m.path && !f.expanded[m.path] {
					rest := append([]fileEntry(nil), f.entries[i+1:]...)
					f.entries = append(f.entries[:i+1], m.entries...)
					f.entries = append(f.entries, rest...)
					f.expanded[m.path] = true
					break
				}
			}
		}
		return nil
	case filePreview:
		if m.path != f.preview {
			return nil
		}
		f.loading = false
		f.scroll = 0
		if m.err != nil {
			f.err = m.err.Error()
			f.body = ""
		} else {
			f.err = ""
			f.body = m.body
		}
		return nil
	}
	if p, ok := msg.(tea.PasteMsg); ok && f.searching {
		f.query += strings.Join(strings.Fields(p.Content), " ")
		return nil
	}
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil
	}
	key := k.String()
	if f.searching {
		switch key {
		case "esc":
			f.query = ""
			f.searching = false
		case "enter":
			f.searching = false
		case "backspace":
			r := []rune(f.query)
			if len(r) > 0 {
				f.query = string(r[:len(r)-1])
			}
		default:
			if k.Text != "" {
				f.query += k.Text
			}
		}
		f.selected = 0
		return nil
	}
	ids := f.visible()
	switch key {
	case "/":
		f.searching = true
	case "esc":
		f.query = ""
		if f.w < 60 {
			f.preview = ""
			f.body = ""
		}
	case "r":
		return f.Refresh()
	case "up", "k":
		f.selected = max(0, f.selected-1)
	case "down", "j":
		f.selected = min(max(0, len(ids)-1), f.selected+1)
	case "pgdown":
		f.scroll = min(max(0, len(strings.Split(f.body, "\n"))-max(1, f.h-6)), f.scroll+max(1, f.h-6))
	case "pgup":
		f.scroll = max(0, f.scroll-max(1, f.h-6))
	case "enter", "right", "l", "left", "h":
		if len(ids) == 0 {
			return nil
		}
		idx := ids[min(f.selected, len(ids)-1)]
		e := f.entries[idx]
		if e.Dir {
			if f.expanded[e.Path] {
				end := idx + 1
				for end < len(f.entries) && f.entries[end].Depth > e.Depth {
					delete(f.expanded, f.entries[end].Path)
					end++
				}
				f.entries = append(f.entries[:idx+1], f.entries[end:]...)
				delete(f.expanded, e.Path)
			} else {
				return f.readDir(e.Path)
			}
		} else {
			return f.open(e.Path)
		}
	case "a":
		if len(ids) == 0 {
			return nil
		}
		e := f.entries[ids[min(f.selected, len(ids)-1)]]
		if e.Dir {
			return nil
		}
		abs, err := projectFile(f.root, e.Path)
		if err != nil {
			f.err = err.Error()
			return nil
		}
		att := state.Attachment{Name: filepath.Base(e.Path), Path: abs, Mime: mime.TypeByExtension(filepath.Ext(e.Path))}
		return func() tea.Msg { return FileAttachMsg{att} }
	}
	return nil
}
func (f *Files) View() string {
	leftW := max(20, min(42, f.w/3))
	rightW := max(10, f.w-leftW-3)
	ids := f.visible()
	left := []string{chrome.PanelHeader.Render("EXPLORER"), chrome.PanelDim.Render(filepath.Base(f.root)), ""}
	if f.query != "" || f.searching {
		left[1] = "/ " + f.query
	}
	start := max(0, f.selected-max(0, f.h-9))
	for pos := start; pos < len(ids); pos++ {
		e := f.entries[ids[pos]]
		icon := "  "
		if e.Dir {
			icon = "▸ "
			if f.expanded[e.Path] {
				icon = "▾ "
			}
		}
		if e.Link {
			icon = "↗ "
		}
		label := strings.Repeat("  ", e.Depth) + icon + safeFileLabel(filepath.Base(e.Path))
		if pos == f.selected {
			label = chrome.TabActive.Render(ansi.Truncate(label, leftW, "…"))
		} else if e.Dir {
			label = chrome.PanelHeader.Render(label)
		}
		left = append(left, label)
	}
	right := []string{chrome.PanelHeader.Render(safeFileLabel(f.preview)), chrome.PanelDim.Render("READ ONLY · PgUp / PgDn scroll preview"), ""}
	if f.err != "" {
		right = append(right, f.err)
	}
	if f.preview == "" {
		right = append(right, "Explore your project", "", "Enter expands a folder or opens a file.", "a attaches the selected file to your conversation.", "", "Dependencies and build output are hidden.")
	}
	if f.loading {
		right = append(right, "Loading…")
	} else {
		lines := strings.Split(f.body, "\n")
		for i := f.scroll; i < len(lines) && len(right) < f.h-3; i++ {
			right = append(right, chrome.PanelDim.Render(fmt.Sprintf("%4d  ", i+1))+lines[i])
		}
	}
	if f.err != "" {
		right = append(right, f.err)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, workBox(strings.Join(left, "\n"), leftW, f.h-2), workRule(f.h-2), workBox(strings.Join(right, "\n"), rightW, f.h-2))
	if f.w < 60 {
		body = workFit(strings.Join(left, "\n"), f.w, f.h-2)
		if f.preview != "" {
			body = workFit(strings.Join(right, "\n"), f.w, f.h-2)
		}
	}
	return workFit(body+"\n"+chrome.PanelDim.Render("↑↓ select   Enter open / collapse   / filter   a attach   r refresh"), f.w, f.h)
}
func safeFileLabel(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
