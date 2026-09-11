package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// Website illustrations use the real UI with a simulated mission. They are
// independent of integration proofs that assert historical terminal geometry.
func websiteShots(out string) error {
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}
	write := func(name, frame, marker string) error {
		if !strings.Contains(ansi.Strip(frame), marker) {
			return fmt.Errorf("website screenshot %s: expected %q", name, marker)
		}
		return os.WriteFile(filepath.Join(out, name+".ansi"), []byte(frame), 0644)
	}
	key := func(d *focusDriver, code rune) {
		next, cmd := d.m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
		d.m = next.(app.Model)
		drainWebsiteCommand(d, cmd, 0)
	}
	typeIn := func(d *focusDriver, text string) {
		for _, r := range text {
			d.send(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		}
	}
	command := func(d *focusDriver, text string) {
		typeIn(d, text)
		key(d, tea.KeyEnter)
		key(d, tea.KeyEnter)
	}

	d := cockpitDriver("cockpit")
	if err := write("layout-normal", d.m.Frame(), "COMMAND DECK"); err != nil {
		return err
	}
	command(d, "/compact on")
	if err := write("layout-compact", d.m.Frame(), "COMMAND"); err != nil {
		return err
	}
	command(d, "/compact off")
	command(d, "/wide 100")
	if err := write("layout-wide", d.m.Frame(), "terminal"); err != nil {
		return err
	}

	d = cockpitDriver("cockpit")
	d.send(tea.KeyPressMsg(tea.Key{Code: 'p', Mod: tea.ModCtrl}))
	if err := write("plan-gated", d.m.Frame(), "plan mode"); err != nil {
		return err
	}
	d.send(state.Event{Kind: state.EvPlanPresent, PlanToolText: "# Goal\nMake project navigation fast and accessible.\n\n## Steps\n1. Add keyboard shortcuts to the project switcher.\n2. Keep the active conversation in view.\n3. Verify narrow terminal layouts.\n\n## Verification\nRun focused tests and review every keyboard path."})
	if err := write("plan-presented", d.m.Frame(), "Goal"); err != nil {
		return err
	}

	d = cockpitDriver("cockpit")
	d.send(state.Event{Kind: state.EvDispatch, EmployeeID: "dev", Task: state.BoardTask{ID: "website-stream", Title: "Wire the SSE stream"}})
	d.send(state.Event{Kind: state.EvWorking, EmployeeID: "dev", TaskID: "website-stream"})
	d.send(focusTool("dev", "tekton-1", "website-read", "read", "internal/room/manager.go", "done"))
	d.send(focusTool("dev", "tekton-1", "website-edit", "edit", "internal/room/handler.go", "done"))
	d.send(focusDiff("dev", "tekton-1", "website-edit", "internal/room/handler.go", "--- a/internal/room/handler.go\n+++ b/internal/room/handler.go\n@@ -1,2 +1,3 @@\n func routes() {\n+    mux.Handle(\"/events\", events)\n }", 1, 0))
	d.pump(3)
	d.send(tea.KeyPressMsg(tea.Key{Code: 'f', Mod: tea.ModCtrl}))
	if err := write("thread-focus", d.m.Frame(), "back to office"); err != nil {
		return err
	}

	d = cockpitDriver("cockpit")
	typeIn(d, "/theme ")
	if err := write("slash-popover", d.m.Frame(), "theme preview"); err != nil {
		return err
	}

	d = cockpitDriver("cockpit")
	d.send(state.Event{Kind: state.EvChatUser, Msg: chatMsg("stop-user", "user", "Check the event stream before we ship.", false)})
	d.send(state.Event{Kind: state.EvChatBoss, Msg: chatMsg("stop-boss", "boss", "Checking the handler and the retry path.", true)})
	d.send(focusTool("boss", "boss", "stop-tool", "read", "internal/room/handler.go", "running"))
	command(d, "/stop")
	return write("stop-unwind", d.m.Frame(), "stopped")
}

// Follow local command messages, including Bubble Tea's private sequenceMsg
// (a []tea.Cmd), while bounding cursor/tick timers. No backend is started.
func drainWebsiteCommand(d *focusDriver, cmd tea.Cmd, depth int) {
	if cmd == nil || depth > 8 {
		return
	}
	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-result:
	case <-time.After(100 * time.Millisecond):
		return
	}
	if msg == nil {
		return
	}
	v := reflect.ValueOf(msg)
	batchType := reflect.TypeOf(tea.BatchMsg{})
	if v.Type().ConvertibleTo(batchType) {
		for _, child := range v.Convert(batchType).Interface().(tea.BatchMsg) {
			drainWebsiteCommand(d, child, depth+1)
		}
		return
	}
	next, more := d.m.Update(msg)
	d.m = next.(app.Model)
	drainWebsiteCommand(d, more, depth+1)
}
