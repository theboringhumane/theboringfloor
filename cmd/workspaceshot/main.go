// workspaceshot renders the real application against isolated local fixtures.
// It never starts an LLM or modifies the user's project history.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/office"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

type fixtureBackend struct{ id string }

func (b *fixtureBackend) Mode() state.Mode                        { return state.ModeLive }
func (b *fixtureBackend) Start(func(state.Event)) error           { return nil }
func (b *fixtureBackend) Send(string) error                       { return nil }
func (b *fixtureBackend) Stop() error                             { return nil }
func (b *fixtureBackend) AnswerPermission(string, string) error   { return nil }
func (b *fixtureBackend) AnswerQuestion(string, [][]string) error { return nil }
func (b *fixtureBackend) RejectQuestion(string) error             { return nil }
func (b *fixtureBackend) MCPServers() ([]state.MCPServer, error)  { return nil, nil }
func (b *fixtureBackend) ReconnectMCP(string) error               { return nil }
func (b *fixtureBackend) PrimaryID() string                       { return b.id }
func (b *fixtureBackend) PrimaryOverride(id string)               { b.id = id }

func main() {
	out := flag.String("out", "/tmp/theboringfloor-workspace-shots", "output directory")
	width := flag.Int("width", 150, "columns")
	height := flag.Int("height", 42, "rows")
	theme := flag.String("theme", "noir", "UI theme")
	flag.Parse()
	scratch, err := os.MkdirTemp("", "floor-ui-proof-")
	must(err)
	defer os.RemoveAll(scratch)
	must(os.Setenv("THEFLOOR_HOME", scratch))
	must(os.MkdirAll(*out, 0755))
	root := filepath.Join(scratch, "developer-platform")
	must(os.MkdirAll(filepath.Join(root, "src"), 0755))
	must(os.WriteFile(filepath.Join(root, "src", "router.go"), []byte("package platform\n\nimport \"net/http\"\n\n// Router scopes project resources to their floor.\nfunc Router() *http.ServeMux {\n    mux := http.NewServeMux()\n    mux.HandleFunc(\"GET /projects\", listProjects)\n    mux.HandleFunc(\"POST /tickets\", createTicket)\n    return mux\n}\n"), 0644))
	must(os.WriteFile(filepath.Join(root, "README.md"), []byte("# Developer platform\n\nA workspace for every project.\n"), 0644))
	must(os.Chdir(root))
	_, err = workspace.Register(root)
	must(err)
	other := filepath.Join(scratch, "design-system")
	must(os.MkdirAll(other, 0755))
	_, err = workspace.Register(other)
	must(err)
	names := []string{"Keyboard command palette", "Build project navigation", "Resolve API credentials", "Review session isolation", "Ship accessible composer"}
	teams := []string{"ui", "frontend", "backend", "coding", "frontend"}
	for i, status := range workspace.Statuses {
		_, err = workspace.PutTicket(root, workspace.Ticket{ID: fmt.Sprintf("TKT-%03d", i+21), Title: names[i], Description: "Keep the developer in context while moving between project work.", Status: status, Priority: []string{"P2", "P1", "P0", "P1", "P2"}[i], Team: teams[i], Owner: []string{"maya", "frontend-1", "api-2", "reviewer", "ui-1"}[i], Checklist: []workspace.Check{{Text: "Keyboard navigation works", Done: true}, {Text: "Existing history survives restart"}, {Text: "Check narrow terminal layout"}}})
		must(err)
	}
	now := time.Now().UnixMilli()
	chat := []state.ChatMsg{
		{ID: "u1", From: "user", At: now - 20000, Text: "Add a project file explorer. Keep the preview next to the tree, and let me attach files to a conversation."},
		{ID: "a1", From: "boss", At: now - 18000, Text: "I’ll add the explorer to this floor and connect it to the existing attachment flow. I’m checking the project boundary and transcript first."},
		{ID: "tool1", From: "boss", Kind: "tool", At: now - 16000, Text: "Read · internal/panels/chat_attach.go", Meta: "done"},
		{ID: "tool2", From: "boss", Kind: "tool", At: now - 14000, Text: "Edit · internal/panels/files.go", Meta: "done"},
		{ID: "a2", From: "boss", At: now - 10000, Text: "The file explorer is ready.\n\n- Expand folders with **Enter**.\n- Preview source with line numbers.\n- Press **a** to attach a file to the composer.\n\nSymlinks outside this project are rejected, and large previews are capped so navigation stays responsive."},
		{ID: "u2", From: "user", At: now - 5000, Text: "Great. Let’s verify it on a narrow terminal too."},
		{ID: "a3", From: "boss", At: now - 2000, Text: "Verified the narrow layout. The explorer keeps the file list readable and opens the selected preview in the available space."},
	}
	sf := app.Snapshot(root, "thread-files", state.OfficeState{Chat: chat})
	sf.Backend = "codex"
	sf.Title = "Project file explorer"
	sf.Team = "frontend"
	must(app.SaveSession(root, sf))
	for i, title := range []string{"API authentication", "Design tokens and focus states"} {
		c := workspace.Conversation{ID: fmt.Sprintf("session-%d", i), Title: title, Backend: []string{"claudecode", "opencode"}[i], Team: teams[i], Messages: 12 + i*8, Updated: now - int64(i)*1800000}
		must(workspace.WriteJSON(filepath.Join(workspace.ConversationDir(root, c.Backend, c.ID), "meta.json"), c))
	}
	if !chrome.SetTheme(*theme) {
		panic("unknown theme: " + *theme)
	}
	office.SetTheme(*theme)
	cfg := config.Default()
	cfg.Backend.Name = "codex"
	cfg.UI.Sounds = "off"
	cfg.UI.Notifications = "off"
	m := app.New(&fixtureBackend{id: "thread-files"}, cfg)
	apply := func(msg tea.Msg) { next, _ := m.Update(msg); m = next.(app.Model) }
	apply(tea.WindowSizeMsg{Width: *width, Height: *height})
	if init := m.Init(); init != nil {
		if cmds, ok := init().(tea.BatchMsg); ok {
			for i, cmd := range cmds {
				if i >= 2 && cmd != nil {
					apply(cmd())
				}
			}
		}
	}
	render := func(name string) { must(os.WriteFile(filepath.Join(*out, name+".ansi"), []byte(m.Frame()), 0644)) }
	apply(state.Event{Kind: state.EvStatus, Text: "Ready · project floors"})
	m.SelectTab("floors")
	render("floors")
	m.SelectTab("board")
	render("board")
	apply(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	render("ticket")
	apply(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	m.SelectTab("files")
	next, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = next.(app.Model)
	if cmd != nil {
		apply(cmd())
	}
	apply(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	next, cmd = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	m = next.(app.Model)
	if cmd != nil {
		apply(cmd())
	}
	render("files")
	m.SelectTab("chat")
	render("transcript")
	apply(tea.KeyPressMsg(tea.Key{Code: 'w', Mod: tea.ModCtrl}))
	render("expanded-transcript")
	apply(tea.KeyPressMsg(tea.Key{Code: 'w', Mod: tea.ModCtrl}))
	apply(state.Event{Kind: state.EvPlanPresent, PlanToolText: "# Project file explorer\n\n## Scope\nGive every floor a project tree and a readable source preview.\n\n## Implementation\n1. Load folders on demand within the project boundary.\n2. Preview source with syntax highlighting and line numbers.\n3. Attach selected files to the active conversation.\n\n## Verification\n- Check keyboard navigation and narrow terminals.\n- Reject symlinks outside the project.\n- Keep existing conversations intact.\n\n## Risk\nCap preview size to keep large files responsive."})
	render("plan")
	apply(tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}))
	apply(tea.KeyPressMsg(tea.Key{Code: 'n', Mod: tea.ModCtrl}))
	render("new-conversation")
	fmt.Println("Rendered floors, board, ticket, files, transcript, and new conversation to", *out)
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
