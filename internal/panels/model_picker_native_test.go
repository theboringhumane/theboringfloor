package panels

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestModelPickerNativeReferences(t *testing.T) {
	for _, tc := range []struct {
		row ModelPickRow
		ref string
	}{
		{ModelPickRow{Ref: "vendor/model[1m]", Provider: "ignored", ID: "other"}, "vendor/model[1m]"},
		{ModelPickRow{ID: "sonnet[1m]"}, "sonnet[1m]"},
		{ModelPickRow{ID: "vendor/native/token"}, "vendor/native/token"},
		{ModelPickRow{Provider: "openai", ID: "gpt-5"}, "openai/gpt-5"},
		{ModelPickRow{Ref: "code-reviewer"}, "code-reviewer"},
	} {
		t.Run(tc.ref, func(t *testing.T) {
			p, picks, _ := modelHarness()
			p.SetRows([]ModelPickRow{tc.row})
			p.Paste(tc.ref)
			if len(p.filtered) != 1 {
				t.Fatalf("exact native reference did not match: %q", tc.ref)
			}
			if line := ansi.Strip(modelMenuRow(tc.row, true, 58)); !strings.HasPrefix(line, "› "+tc.ref) {
				t.Fatalf("native reference changed in rendering: %q", line)
			}
			p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
			if len(*picks) != 1 || (*picks)[0] != tc.ref {
				t.Fatalf("want exact callback %q, got %v", tc.ref, *picks)
			}
		})
	}
}

func TestModelPickerNativeAgentDescriptionSearch(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetTitle("SUB-AGENT TYPE · codex")
	p.SetRows([]ModelPickRow{
		{Ref: "code-reviewer", Description: "Reviews complex changes"},
		{Ref: "explorer", Description: "Finds relevant code"},
	})
	p.Paste("REVIEWS\r\ncomplex")
	if len(p.filtered) != 1 || p.filtered[0].Ref != "code-reviewer" {
		t.Fatalf("description paste did not narrow agent types: %+v", p.filtered)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "code-reviewer" {
		t.Fatalf("agent callback changed its native name: %v", *picks)
	}
}

func TestModelPickerDisabledEnter(t *testing.T) {
	p, picks, _ := modelHarness()
	p.SetRows([]ModelPickRow{{Ref: "retired/model", Disabled: true}, {Ref: "available"}})
	if cmd := p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter})); cmd != nil || len(*picks) != 0 {
		t.Fatalf("disabled row accepted: %v", *picks)
	}
	if line := ansi.Strip(modelMenuRow(p.rows[0], true, 22)); !strings.HasPrefix(line, "› ") || !strings.Contains(line, "unavailable") {
		t.Fatalf("disabled cursor status missing on narrow card: %q", line)
	}
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 || (*picks)[0] != "available" {
		t.Fatalf("navigation past disabled entry did not select: %v", *picks)
	}
}

func TestModelPickerNativeMetadataAndCardWidths(t *testing.T) {
	p, _, _ := modelHarness()
	p.SetTitle("SUB-AGENT MODEL · reviewer · codex")
	p.SetRows([]ModelPickRow{
		{Ref: "gpt-5.4", Description: "Balanced", IsDefault: true},
		{Ref: "sonnet[1m]", Description: "Long context", Current: true},
		{Ref: "retired/model", Disabled: true},
	})
	rows, _ := p.modelCard(80)
	proof := ansi.Strip(strings.Join(rows, "\n"))
	t.Logf("native card:\n%s", proof)
	if !strings.Contains(proof, "SUB-AGENT MODEL · reviewer · codex") {
		t.Fatalf("custom target title missing:\n%s", proof)
	}
	for _, tc := range []struct {
		row          int
		want, absent string
	}{
		{0, "backend default", "current"},
		{1, "Long context · current", "backend default"},
	} {
		line := ansi.Strip(modelMenuRow(p.rows[tc.row], true, 58))
		if !strings.Contains(line, tc.want) || strings.Contains(line, tc.absent) || !strings.HasPrefix(line, "› ") {
			t.Fatalf("default/current metadata conflated or cursor missing: %q", line)
		}
	}
	p.SetTitle("SUB-AGENT MODEL · " + strings.Repeat("审查-agent/", 20) + " · codex")
	p.SetEmptyHint(strings.Repeat("No native agent types reported. ", 20))
	catalog := p.rows
	for _, width := range []int{28, 36, 48, 80, 160} {
		for _, empty := range []bool{false, true} {
			p.SetRows(catalog)
			if empty {
				p.SetRows(nil)
			}
			rows, cardW := p.modelCard(width)
			for i, line := range rows {
				if got := lipgloss.Width(line); got != cardW || got > width {
					t.Fatalf("frame=%d row=%d width=%d want=%d: %q", width, i, got, cardW, ansi.Strip(line))
				}
			}
		}
	}
	p.SetTitle("")
	p.SetEmptyHint("")
	if frame := modelFrame(t, p, 80, 24); !strings.Contains(frame, "BOSS MODEL") || !strings.Contains(frame, "no models reported") {
		t.Fatalf("empty setters must restore legacy defaults:\n%s", frame)
	}
}

func TestModelPickerPendingFreezesInput(t *testing.T) {
	p, picks, cancels := modelHarness()
	p.SetRows([]ModelPickRow{{Ref: "first"}, {Ref: "second"}})
	p.SetPending(true)
	for _, code := range []rune{tea.KeyDown, tea.KeyEnter, tea.KeyEscape} {
		p.Key(tea.KeyPressMsg(tea.Key{Code: code}))
	}
	modelType(p, "second")
	p.Paste("second")
	if p.filter != "" || p.Sel() != 0 || len(*picks) != 0 || *cancels != 0 {
		t.Fatalf("pending picker consumed an action: filter=%q sel=%d picks=%v cancels=%d", p.filter, p.Sel(), *picks, *cancels)
	}
	frame := modelFrame(t, p, 80, 24)
	if !strings.Contains(frame, "applying selection…") || strings.Contains(frame, "enter: switch") || strings.Contains(frame, "esc: cancel") {
		t.Fatalf("pending footer promises unavailable actions:\n%s", frame)
	}
	p.SetPending(false)
	p.Key(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(*picks) != 1 {
		t.Fatalf("clearing pending must restore selection: %v", *picks)
	}
}
