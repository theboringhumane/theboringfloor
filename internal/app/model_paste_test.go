// model_paste_test.go — the app's ONE paste router (model.go routePaste).
// tea.PasteMsg (Terminal.app's cmd+v, every terminal's bracketed paste)
// lands on EXACTLY ONE surface, mirroring the KEY path's ownership:
//
//	plan pane (focused)  — pinned by TestPlanPasteMsgRoutesToFocusedPane
//	captured terminal    — the shell owns the keyboard while captured
//	question float       — a parked turn owns the chat's input
//	/model picker        — the Paste(string) seam, else the notice
//	thread-focus view    — no text surface → the notice
//	chat textarea        — chat tab, and the RELEASED terminal tab
//	everything else      — ONE dim "paste: nothing focused accepts text"
//
// Every leg also asserts the paste landed NOWHERE ELSE (never two
// places, never silently nowhere).
package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// pasteFakeTerm — a recording TerminalTab double: pastes and keys into
// the "shell" counted separately. What is NOT recorded proves the gate.
type pasteFakeTerm struct {
	pastes  []string
	keys    int
	focuses int
	blurs   int
}

func (f *pasteFakeTerm) Title() string                 { return "term" }
func (f *pasteFakeTerm) SetSize(w, h int)              {}
func (f *pasteFakeTerm) SetState(st state.OfficeState) {}
func (f *pasteFakeTerm) View() string                  { return "" }
func (f *pasteFakeTerm) Alive() bool                   { return true }
func (f *pasteFakeTerm) Close() error                  { return nil }
func (f *pasteFakeTerm) Focus()                        { f.focuses++ }
func (f *pasteFakeTerm) Blur()                         { f.blurs++ }
func (f *pasteFakeTerm) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		f.pastes = append(f.pastes, msg.Content)
	case tea.KeyPressMsg:
		f.keys++
	}
	return nil
}

// pasteSetupTerminal — a scratch-home model with the SpawnTerminal seam
// wired to a recording fake (grab_test's shape), sized desktop.
func pasteSetupTerminal(t *testing.T) (Model, *recBackend, *pasteFakeTerm) {
	t.Helper()
	scratchHome(t)
	fake := &pasteFakeTerm{}
	prev := SpawnTerminal
	SpawnTerminal = func(cols, rows int) (TerminalTab, error) { return fake, nil }
	t.Cleanup(func() { SpawnTerminal = prev })
	b := &recBackend{}
	m := New(b, nil)
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = nm.(Model)
	return m, b, fake
}

func pasteMsg(s string) tea.PasteMsg { return tea.PasteMsg{Content: s} }

// TestPasteRoutingChatTextarea — the default: the chat tab's draft takes
// the paste (and NOTHING else does).
func TestPasteRoutingChatTextarea(t *testing.T) {
	m, b, fake := pasteSetupTerminal(t)
	m = runMsg(t, m, pasteMsg("chat-bound paste"))
	if len(fake.pastes) != 0 {
		t.Fatalf("the paste must never touch the terminal, got %q", fake.pastes)
	}
	m = runMsg(t, m, enterKey())
	if len(b.sentTexts) != 1 || b.sentTexts[0] != "chat-bound paste" {
		t.Fatalf("the chat draft must carry the paste into Send, got %+v", b.sentTexts)
	}
}

// TestPasteRoutingQuestionFloat — a parked question owns the chat's
// input: the paste lands in the popover's answer field (batched), the
// main draft stays empty, ctrl+enter ships it through AnswerQuestion.
func TestPasteRoutingQuestionFloat(t *testing.T) {
	m, b, _ := pasteSetupTerminal(t)
	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, QuestionID: "que-rt",
		EmployeeName: "boss", ToolState: "pending",
		Questions: []state.QuestionItem{{Question: "paste the stack trace?"}},
	})
	if m.question == nil {
		t.Fatal("precondition: the question hold must be open")
	}
	paste := "goroutine 1 [running]:\nmain.main()"
	m = runMsg(t, m, pasteMsg(paste))
	if len(b.sentTexts) != 0 {
		t.Fatalf("the disabled main draft must NOT eat the paste, got %+v", b.sentTexts)
	}
	m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModCtrl}))
	if len(b.qAnswers) != 1 || len(b.qAnswers[0].answers) != 1 ||
		len(b.qAnswers[0].answers[0]) != 1 || b.qAnswers[0].answers[0][0] != paste {
		t.Fatalf("the pasted multi-line answer must ship VERBATIM, got %+v", b.qAnswers)
	}
}

// TestPasteRoutingTerminalCaptured — while the shell owns the keyboard,
// the paste forwards to the terminal tab's panel (the PTY write +
// bracket-wrap live in panels/terminal.go, pinned there).
func TestPasteRoutingTerminalCaptured(t *testing.T) {
	m, b, fake := pasteSetupTerminal(t)
	m = runMsg(t, m, grabTab())       // chat → terminal (spawns the fake)
	m = runMsg(t, m, grabCtrlSpace()) // dive in
	if !m.termCapturedNow() {
		t.Fatal("precondition: ctrl+space must capture the shell")
	}
	m = runMsg(t, m, pasteMsg("echo pasted-ok"))
	if len(fake.pastes) != 1 || fake.pastes[0] != "echo pasted-ok" {
		t.Fatalf("the captured shell must receive the paste, got %q", fake.pastes)
	}
	// …and ONLY the shell: the chat draft must stay empty (send nothing).
	m = runMsg(t, m, grabShiftTab()) // back to chat
	m = runMsg(t, m, enterKey())
	if len(b.sentTexts) != 0 {
		t.Fatalf("the chat draft must NOT share a captured paste, got %+v", b.sentTexts)
	}
}

// TestPasteRoutingCapturedTerminalBeatsQuestion — the key path's exact
// precedence: while CAPTURED the shell owns EVERYTHING (a parked float
// included), so the paste goes to the PTY, not the question field.
func TestPasteRoutingCapturedTerminalBeatsQuestion(t *testing.T) {
	m, b, fake := pasteSetupTerminal(t)
	m = runMsg(t, m, state.Event{Kind: state.EvQuestion, QuestionID: "que-cap",
		EmployeeName: "boss", ToolState: "pending",
		Questions: []state.QuestionItem{{Question: "ignored while captured"}},
	})
	m = runMsg(t, m, grabTab())
	m = runMsg(t, m, grabCtrlSpace())
	m = runMsg(t, m, pasteMsg("shell-owned"))
	if len(fake.pastes) != 1 || fake.pastes[0] != "shell-owned" {
		t.Fatalf("captured: the shell owns the paste over the float, got %q", fake.pastes)
	}
	if len(b.qAnswers) != 0 {
		t.Fatalf("the question field must NOT eat a captured paste, got %+v", b.qAnswers)
	}
}

// TestPasteRoutingTerminalReleasedChatFallback — a RELEASED terminal tab
// gives the paste to the chat draft (the office owns the keyboard there;
// the PTY never sees it).
func TestPasteRoutingTerminalReleasedChatFallback(t *testing.T) {
	m, b, fake := pasteSetupTerminal(t)
	m = runMsg(t, m, grabTab()) // chat → terminal, RELEASED by default
	m = runMsg(t, m, pasteMsg("released-paste"))
	if len(fake.pastes) != 0 {
		t.Fatalf("released: the PTY must never see the paste, got %q", fake.pastes)
	}
	m = runMsg(t, m, grabShiftTab()) // back to chat — the draft must hold it
	m = runMsg(t, m, enterKey())
	if len(b.sentTexts) != 1 || b.sentTexts[0] != "released-paste" {
		t.Fatalf("released terminal: the chat draft takes the paste, got %+v", b.sentTexts)
	}
}

// TestPasteRoutingIgnoredNotice — no focused text surface (the agents
// tab): ONE dim notice, never a silent drop.
func TestPasteRoutingIgnoredNotice(t *testing.T) {
	m, b, fake := pasteSetupTerminal(t)
	m = runMsg(t, m, grabTab()) // chat → terminal
	m = runMsg(t, m, grabTab()) // terminal → agents
	if idx := m.tabs.ActiveIndex(); idx != 2 {
		t.Fatalf("precondition: agents tab, got %d", idx)
	}
	m = runMsg(t, m, pasteMsg("nowhere-bound"))
	if got := lastOfficeMsg(m); got != pasteIgnoreNotice {
		t.Fatalf("the ignored paste toasts the dim notice, got %q", got)
	}
	if len(fake.pastes) != 0 || len(b.sentTexts) != 0 {
		t.Fatalf("the ignored paste landed somewhere: pastes=%q sends=%+v", fake.pastes, b.sentTexts)
	}
}

// TestPasteRoutingModelPickerFilter — the /model picker grew its Paste
// seam (the picker-search wave): a paste while it is open routes INTO the
// picker's filter — never the ignore notice, never the disabled textarea.
func TestPasteRoutingModelPickerFilter(t *testing.T) {
	m, b, _ := pasteSetupTerminal(t)
	m.modelPick = panels.NewModelPicker(nil, nil)
	m = runMsg(t, m, pasteMsg("gpt-5"))
	if got := lastOfficeMsg(m); got == pasteIgnoreNotice {
		t.Fatalf("a seamed picker paste must NOT toast the ignore notice")
	}
	m = runMsg(t, m, enterKey())
	if len(b.sentTexts) != 0 {
		t.Fatalf("the disabled draft must NOT eat a picker paste, got %+v", b.sentTexts)
	}
}

// TestPasteRoutingPrecedenceTable — the precedence ladder as ONE table
// (the brief's routing contract): each leg drives a fresh model into a
// surface state and asserts the ONE owner of the paste.
func TestPasteRoutingPrecedenceTable(t *testing.T) {
	questionEv := state.Event{Kind: state.EvQuestion, QuestionID: "que-tbl",
		EmployeeName: "boss", ToolState: "pending",
		Questions: []state.QuestionItem{{Question: "table leg"}},
	}
	type leg struct {
		name      string
		setup     func(t *testing.T, m Model) Model
		wantWhere string // "chat" | "question" | "terminal" | "notice"
	}
	legs := []leg{
		{"chat tab → textarea", func(t *testing.T, m Model) Model { return m }, "chat"},
		{"terminal captured → shell", func(t *testing.T, m Model) Model {
			m = runMsg(t, m, grabTab())
			return runMsg(t, m, grabCtrlSpace())
		}, "terminal"},
		{"terminal released → chat", func(t *testing.T, m Model) Model {
			return runMsg(t, m, grabTab())
		}, "chat"},
		{"question float → answer field", func(t *testing.T, m Model) Model {
			return runMsg(t, m, questionEv)
		}, "question"},
		{"question + captured → shell", func(t *testing.T, m Model) Model {
			m = runMsg(t, m, questionEv)
			m = runMsg(t, m, grabTab())
			return runMsg(t, m, grabCtrlSpace())
		}, "terminal"},
		{"agents tab → notice", func(t *testing.T, m Model) Model {
			m = runMsg(t, m, grabTab())
			return runMsg(t, m, grabTab())
		}, "notice"},
	}
	for _, l := range legs {
		t.Run(l.name, func(t *testing.T) {
			m, b, fake := pasteSetupTerminal(t)
			m = l.setup(t, m)
			m = runMsg(t, m, pasteMsg("TBL-MARKER"))
			switch l.wantWhere {
			case "chat":
				if len(fake.pastes) != 0 || len(b.qAnswers) != 0 {
					t.Fatalf("paste leaked (pastes=%q qAnswers=%+v)", fake.pastes, b.qAnswers)
				}
				if m.tabs.ActiveIndex() != 0 {
					m = runMsg(t, m, grabShiftTab()) // released-terminal leg: hop back to chat
				}
				m = runMsg(t, m, enterKey())
				if len(b.sentTexts) != 1 || !strings.Contains(b.sentTexts[0], "TBL-MARKER") {
					t.Fatalf("chat must own the paste, sends=%+v", b.sentTexts)
				}
			case "question":
				if len(fake.pastes) != 0 || len(b.sentTexts) != 0 {
					t.Fatalf("paste leaked (pastes=%q sends=%+v)", fake.pastes, b.sentTexts)
				}
				m = runMsg(t, m, tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Mod: tea.ModCtrl}))
				if len(b.qAnswers) != 1 || b.qAnswers[0].answers[0][0] != "TBL-MARKER" {
					t.Fatalf("the question field must own the paste, qAnswers=%+v", b.qAnswers)
				}
			case "terminal":
				if len(fake.pastes) != 1 || fake.pastes[0] != "TBL-MARKER" {
					t.Fatalf("the shell must own the paste, pastes=%q", fake.pastes)
				}
				if len(b.qAnswers) != 0 || len(b.sentTexts) != 0 {
					t.Fatalf("paste leaked (qAnswers=%+v sends=%+v)", b.qAnswers, b.sentTexts)
				}
			case "notice":
				if got := lastOfficeMsg(m); got != pasteIgnoreNotice {
					t.Fatalf("the ignore leg must toast the notice, got %q", got)
				}
			}
		})
	}
}
