// terminal_panel_stub.go — STUB (uisshot only, NOT production).
//
// Inline fallback for the parallel-built panels.TermPanel (contract:
// NewTerminal(width, height) (*TermPanel, error) — Title "terminal",
// SetSize / SetState / View / Update + Close / Alive). The --terminal shot
// wires this through app.SpawnTerminal so the lazy-spawn → forward-keys →
// Close plumbing is frame-verifiable BEFORE the real PTY panel lands; the
// manager swaps the factory at integration. Every method here is fake: no
// process, no PTY, just a counting shell-shaped render.
package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// terminalPanelStub — fake TermPanel for the --terminal frame.
type terminalPanelStub struct {
	w, h     int
	received int      // keys routed into the "shell" (routing proof)
	pastes   []string // paste CONTENTS routed into the "shell" (--paste proof)
	closed   bool
}

var _ interface {
	Title() string
	SetSize(w, h int)
	SetState(st state.OfficeState)
	View() string
	Update(msg tea.Msg) tea.Cmd
	Close() error
	Alive() bool
} = (*terminalPanelStub)(nil)

func newTerminalPanelStub(w, h int) *terminalPanelStub {
	return &terminalPanelStub{w: w, h: h}
}

func (t *terminalPanelStub) Title() string { return "terminal" }

func (t *terminalPanelStub) SetSize(w, h int) { t.w, t.h = w, h }

func (t *terminalPanelStub) SetState(st state.OfficeState) {}

func (t *terminalPanelStub) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		t.received++
	case tea.PasteMsg:
		t.pastes = append(t.pastes, msg.Content)
	}
	return nil
}

func (t *terminalPanelStub) Close() error {
	t.closed = true
	return nil
}

func (t *terminalPanelStub) Alive() bool { return !t.closed }

// View — a shell-shaped placeholder card, clipped/padded to w×h.
func (t *terminalPanelStub) View() string {
	w := t.w
	if w < 1 {
		w = 1
	}
	pad := func(s string) string {
		n := 0
		for i := range s {
			if n >= w {
				s = s[:i]
				break
			}
			n++
		}
		for n < w {
			s += " "
			n++
		}
		return s
	}
	lines := []string{
		"$ echo theboringfloor",
		"theboringfloor",
		"(uisshot STUB shell — the real panels.TermPanel is wired by cmd/theboringfloor)",
		fmt.Sprintf("$ keys received: %d · pastes: %d █", t.received, len(t.pastes)),
	}
	for i := range lines {
		lines[i] = pad(lines[i])
	}
	h := t.h
	if h < 1 {
		h = 1
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, pad(""))
	}
	return strings.Join(lines, "\n")
}
