// plan_shape_test.go — the plan presentation SHAPE GATE: the left plan
// pane only mirrors boss replies that LOOK like plans (looksLikePlan),
// boss chatter/status narration never presents. Covers the unit matrix
// (structure signal + document heft), the app-level mirror/reject flow,
// the debounced/once-per-session transcript notes, user-edit protection
// under the gate, chatter-arriving-late pane retention, the persistence
// gate, and the starter-template flow's indifference to the gate.
package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// bossChatter fixtures — the field-report shape: work-narration, one or
// two prose lines, no markdown structure. They must NEVER present.
const (
	planChatterOne   = "quick sync — sent the lobby brief to ops; scanning wall options now, the plan proper lands in a beat."
	planChatterTwo   = "still sketching here — comparing matte panels against glass lanes; the structured plan is next."
	planChatterProse = "Quick sync on where we are: the review pipeline rehearsals are done locally.\n" +
		"They surfaced real HEAD defects worth fixing before the next wave lands.\n" +
		"I am loading the implementation skill and sending three read-only scouts out."
)

// fieldReportChatter — the exact REPORTED failure shape (v0.2.12 live
// session): boss work-narration that presented as if it were the plan.
const fieldReportChatter = "Quick 'where are we': the review pipeline rehearsals are done locally, they surfaced real HEAD defects… Loading the CN implementation skill and sending three read-only scouts to map every surface, then the plan lands."

// TestPlanShapeLooksLikePlanMatrix pins the gate's unit contract: ONE
// markdown structure signal (heading / bullet / numbered / fence / bold
// section label) + ≥3 lines + ≥160 non-whitespace chars. Chatter fails,
// fragments fail, heft-less structure fails — every passing exemplar is
// a structured document.
func TestPlanShapeLooksLikePlanMatrix(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"structured plan (heading+bullets)", gatedPlan("Matrix plan", "matrix"), true},
		{"heading only + heft", "# Lobby wall plan\n# Steps for the gallery wall\n# Finish details for the matte panels\n# Kanban lanes and the return shelf\n# Zero clerical chrome anywhere near the entrance\n# Warm uplights washing the panel seams", true},
		{"bullets only + heft", "- matte panels azimuth-washed along the long east wall of the lobby\n- glassmorphic kanban lanes for the return shelf by the tea machine\n- zero clerical chrome within sight of the entrance doors of the lobby\n- warm uplights washing every panel seam from top to bottom", true},
		{"numbered only + heft", "1. measure the east wall and mark the panel seams with chalk line\n2. wash the matte panels with the azimuth finish in two even coats\n3. hang the glassmorphic kanban lanes so the glass sits flush with the rail\n4. aim the warm uplights along every panel seam, top to bottom", true},
		{"fence with content", "```sh\nmeasure --east-wall --mark-seams\nwash --matte --azimuth --coats 2\nhang --kanban-lanes --flush-with-rail --level\npolish --uplights --warm --seams top-to-bottom\ninspect --corner-bench --visitors --sightlines\n```", true},
		{"bold section labels", "**Goal**: a gallery lobby wall that feels calm instead of corporate.\n**Steps**: matte panels azimuth-washed, glassmorphic kanban lanes, zero clerical chrome near the entrance doors.\n**Why now**: the lobby is the first surface every visitor reads.", true},
		// mixed prose WITH structure: a numbered list inside narration is
		// still a document — the gate is shape-only (anonymized), and
		// structure wins.
		{"prose with a numbered list", "Here's what I'll do when the lobby wall lands: gather the panels and the lanes first.\n1. measure the east wall and mark the panel seams with chalk\n2. wash the matte panels with the azimuth finish in two coats\n3. hang the kanban lanes so the glass sits flush with the rail", true},

		{"boss chatter one-liner", planChatterOne, false},
		{"boss chatter two-liner", planChatterOne + "\n" + planChatterTwo, false},
		{"multi-line prose, no markdown", planChatterProse, false},
		{"the reported v0.2.12 narration", fieldReportChatter, false},
		{"only the word Goal", "Goal", false},
		{"tiny bullet", "- a", false},
		{"dash run", "----------", false},
		{"fence only, no heft", "```\nls\n```", false},
		{"bullet only, no heft", "- matte panels\n- kanban lanes\n- zero chrome", false},
		{"bold label only, no heft", "**Goal**: lobby wall", false},
		{"whitespace only", "   \n  \n   ", false},
		{"empty", "", false},
	}
	for _, c := range cases {
		if got := looksLikePlan(c.text); got != c.want {
			t.Errorf("%s: looksLikePlan = %t, want %t", c.name, got, c.want)
		}
	}
}

// TestPlanShapeGateBoundaries pins the heft floor's exact edges: ≥160
// non-whitespace chars passes (with a structure signal), 159 fails, and
// a 2-line document fails regardless of size.
func TestPlanShapeGateBoundaries(t *testing.T) {
	atFloor := "# T\n- " + strings.Repeat("x", 153) + "\ntail" // 2+154+4 = 160 non-ws
	underFloor := "# T\n- " + strings.Repeat("x", 152) + "\ntail"
	if got := looksLikePlan(atFloor); !got {
		t.Error("exactly 160 non-whitespace chars with a bullet signal must pass")
	}
	if got := looksLikePlan(underFloor); got {
		t.Error("159 non-whitespace chars must fail the heft floor")
	}
	twoLine := "# T\n- " + strings.Repeat("x", 200)
	if got := looksLikePlan(twoLine); got {
		t.Error("a 2-line document must fail the line floor regardless of heft")
	}
}

// TestPlanShapeChatterNeverPresents pins the app-level gate: in plan
// mode, completed boss CHATTER replies never mirror into the pane (the
// pane stays empty+hidden, the idle hint keeps riding); the escape-valve
// note teaches ONCE per office session, then stays silent; a later
// plan-shaped reply presents normally.
func TestPlanShapeChatterNeverPresents(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	// chatter #1: nothing presents; the valve teaches once
	m = bossReply(t, m, "bc1", planChatterOne)
	if got := m.plan.Value(); got != "" {
		t.Fatalf("chatter must never mirror into the pane, got %q", got)
	}
	if m.planPaneVisible() {
		t.Fatal("the floor slot must stay with the office while only chatter has arrived")
	}
	if n := countOfficeRows(m, planChatValveNotice); n != 1 {
		t.Fatalf("the escape-valve note must post exactly once, got %d", n)
	}
	if got := m.hintLine(); got != planHintIdle {
		t.Fatalf("the idle hint keeps riding while nothing presented, got %q", got)
	}

	// chatter #2 (different text): valve ONCE per session — no note-spam
	m = bossReply(t, m, "bc2", planChatterTwo)
	if n := countOfficeRows(m, planChatValveNotice); n != 1 {
		t.Fatalf("the valve is once per session, got %d", n)
	}
	if n := countOfficeRows(m, planNotPlanNotice); n != 0 {
		t.Fatalf("the empty pane never earns the kept-its-last-plan note, got %d", n)
	}
	if m.planPaneVisible() {
		t.Fatal("pane still hidden after the second chatter")
	}

	// the field-report narration shape, verbatim: never presents either
	m = bossReply(t, m, "bc3", fieldReportChatter)
	if got := m.plan.Value(); got != "" {
		t.Fatalf("the reported narration must never present, got %q", got)
	}

	// a plan-shaped reply: presents as before
	plan := gatedPlan("Lobby gallery wall", "matte")
	m = bossReply(t, m, "b1", plan)
	if got := m.plan.Value(); got != plan {
		t.Fatalf("the plan-shaped reply mirrors, got %q", got)
	}
	if !m.planPaneVisible() {
		t.Fatal("a presented plan owns the floor slot")
	}
	if got := m.hintLine(); got != planHintPane {
		t.Fatalf("the pane hint swaps on presentation, got %q", got)
	}
}

// TestPlanShapeGateKeepsUserEdit pins user-edit protection under the
// gate: while the pane carries USER edits, chatter earns the dim kept-
// its-last-plan note (once — never a clobber), and a plan-shaped reply
// still hits the classic anti-clobber (your edited plan kept). The gate
// is ANONYMIZED: shape decides, not sender labels — but the From=="boss"
// guard still applies, so a plan-shaped bubble from a non-boss sender
// never presents.
func TestPlanShapeGateKeepsUserEdit(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	v1 := gatedPlan("Plan v1", "matte")
	m = bossReply(t, m, "b1", v1)
	m = runMsg(t, m, paneClick())
	m = runMsg(t, m, pressKey('!'))
	edited := v1 + "!"
	if !m.plan.UserDirty() {
		t.Fatal("setup: the edit must latch userDirty")
	}

	// chatter while user-edited: the edit survives; the kept-last-plan
	// note fires ONCE for that turn
	m = bossReply(t, m, "bc1", planChatterOne)
	if m.plan.Value() != edited {
		t.Fatalf("chatter must never clobber a user-edited pane, got %q", m.plan.Value())
	}
	if n := countOfficeRows(m, planNotPlanNotice); n != 1 {
		t.Fatalf("one kept-its-last-plan note per turn, got %d", n)
	}

	// a plan-shaped reply while user-edited: the anti-clobber note (the
	// existing contract) — the edits still win
	v2 := gatedPlan("Plan v2", "glassmorphic")
	m = bossReply(t, m, "b2", v2)
	if m.plan.Value() != edited {
		t.Fatalf("anti-clobber: the edit survives the plan-shaped reply, got %q", m.plan.Value())
	}
	if n := countOfficeRows(m, planKeptNotice); n != 1 {
		t.Fatalf("the anti-clobber note rides exactly once, got %d", n)
	}

	// the guard is metadata-blind ONLY for shape: a plan-shaped bubble
	// from a NON-boss sender still never presents
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss,
		Msg: state.ChatMsg{ID: "impostor-1", From: "user", Text: gatedPlan("Impostor", "fake")}})
	if m.plan.Value() != edited {
		t.Fatalf("a non-boss sender must never present, got %q", m.plan.Value())
	}
}

// TestPlanShapeChatterAfterPlanKeepsPane pins late chatter: once a real
// plan rides the pane, boss chatter never replaces it — the dim note
// explains (at most once per boss turn; identical chatter text stays
// silent; a new turn with different chatter notes once more).
func TestPlanShapeChatterAfterPlanKeepsPane(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	plan := gatedPlan("Lobby gallery wall", "matte")
	m = bossReply(t, m, "b1", plan)

	// late chatter: the pane keeps its last plan; the note fires once
	m = bossReply(t, m, "bc1", planChatterOne)
	if m.plan.Value() != plan {
		t.Fatalf("late chatter must keep the presented plan, got %q", m.plan.Value())
	}
	if n := countOfficeRows(m, planNotPlanNotice); n != 1 {
		t.Fatalf("the kept-its-last-plan note fires once, got %d", n)
	}

	// the SAME chatter text again (a new bubble, same words): silent
	m = bossReply(t, m, "bc2", planChatterOne)
	if n := countOfficeRows(m, planNotPlanNotice); n != 1 {
		t.Fatalf("identical chatter stays silent, got %d", n)
	}

	// a NEW turn with DIFFERENT chatter: the note fires once more
	m = bossReply(t, m, "bc3", planChatterTwo)
	if n := countOfficeRows(m, planNotPlanNotice); n != 2 {
		t.Fatalf("a new turn's chatter notes once more, got %d", n)
	}
	if m.plan.Value() != plan {
		t.Fatalf("the pane still keeps its last plan, got %q", m.plan.Value())
	}

	// the identical message ID re-delivered (replace-in-place safety):
	// never a second note for the same turn
	m = bossReply(t, m, "bc3", planChatterTwo)
	if n := countOfficeRows(m, planNotPlanNotice); n != 2 {
		t.Fatalf("the note is once per boss turn (message ID), got %d", n)
	}
}

// TestPlanShapePersistenceGate pins: session.json's planText only ever
// holds plan-shaped documents — chatter-shaped scratch, the starter, and
// the empty buffer all project as "".
func TestPlanShapePersistenceGate(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)

	if got := m.planText(); got != "" {
		t.Fatalf("an empty buffer projects nothing, got %q", got)
	}
	m.plan.SetValue(planChatterProse) // 3 prose lines, no markdown structure
	if got := m.planText(); got != "" {
		t.Fatalf("chatter-shaped scratch never persists, got %q", got)
	}
	plan := gatedPlan("Persisted plan", "persist")
	m.plan.SetValue(plan)
	if got := m.planText(); got != plan {
		t.Fatalf("a plan-shaped buffer persists, got %q", got)
	}
}

// TestPlanShapeStarterFlowUnaffected pins the starter template's
// indifference to the gate: a click arms the scaffold as before, chatter
// landing over the armed template stays SILENT (nothing to explain), and
// the next plan-shaped boss reply replaces the template exactly like
// before (the scaffold is not a user edit).
func TestPlanShapeStarterFlowUnaffected(t *testing.T) {
	b := &agentRecBackend{}
	m := New(b, nil)
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	m = runMsg(t, m, ctrlP())

	m = runMsg(t, m, paneClick())
	if !m.plan.IsStarter() {
		t.Fatal("a click into an empty pane must arm the starter scaffold")
	}
	m = runMsg(t, m, escKey()) // blur back to chat, template keeps riding

	// chatter over the armed starter: pane keeps the template, NO notes
	m = bossReply(t, m, "bc1", planChatterOne)
	if !m.plan.IsStarter() {
		t.Fatal("chatter must not touch the armed starter template")
	}
	if n := countOfficeRows(m, planNotPlanNotice); n != 0 {
		t.Fatalf("the armed starter earns silence, got %d", n)
	}
	if n := countOfficeRows(m, planChatValveNotice); n != 0 {
		t.Fatalf("the valve is for the empty pane, got %d", n)
	}

	// a plan-shaped reply replaces the template (never a user edit)
	plan := gatedPlan("Starter replacement", "replaces")
	m = bossReply(t, m, "b1", plan)
	if m.plan.Value() != plan {
		t.Fatalf("the boss's plan still replaces the starter, got %q", m.plan.Value())
	}
	if m.plan.IsStarter() {
		t.Fatal("an adopted boss plan is not the starter")
	}
}
