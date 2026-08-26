// config_backend_test.go — the app-side backend-selection wiring:
//
//   - backendFor: the ONE resolver (boot + swap + headless display) maps
//     "claudecode" to the claude transport and everything else to the
//     opencode transport ("" is opencode — the config backfill contract).
//   - backendNameFromStatus: the EvStatus marker grammar
//     ("[theboringoffice] backend: <name>" boot hint; "… <old> → <new>
//     (turn #N archived)" swap line) with a strict two-name whitelist —
//     a refusal copy is NOT a latch.
//   - the reducer latch: those lines ride applyEvent into
//     OfficeState.BackendName exactly once per event.
//   - the boot restore pin: each transport's hydrate override comes from
//     ITS session.json entry, never cross-pinned.
package app

import (
	"fmt"
	"os"
	"testing"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

func TestBackendForResolver(t *testing.T) {
	dir := t.TempDir()
	opencode := fmt.Sprintf("%T", backendFor("opencode", "", dir, nil))
	claudecode := fmt.Sprintf("%T", backendFor("claudecode", "", dir, nil))
	// The contract that matters is selection, not concrete type names
	// (those stay the backend package's own naming business): opencode
	// and claudecode resolve to DIFFERENT live transports, and every
	// non-claudecode name ("" included — the config backfill) resolves
	// to the opencode one.
	if opencode == claudecode {
		t.Fatalf("opencode and claudecode must resolve to different transports, both %s", opencode)
	}
	for _, name := range []string{"", "zephyr"} {
		if got := fmt.Sprintf("%T", backendFor(name, "", dir, nil)); got != opencode {
			t.Errorf("backendFor(%q) = %s, want the opencode transport (%s)", name, got, opencode)
		}
	}
	for _, b := range []state.Backend{
		backendFor("opencode", "", dir, nil),
		backendFor("claudecode", "", dir, nil),
	} {
		if b == nil || b.Mode() != state.ModeLive {
			t.Errorf("every resolved transport must be a LIVE backend, got %v", b)
		}
	}
	// The exported cmd-facing twin resolves identically to the resolver.
	if got := fmt.Sprintf("%T", BackendFor("claudecode", "", dir, nil)); got != claudecode {
		t.Errorf("BackendFor(claudecode) = %s, want %s", got, claudecode)
	}
	// And the production factory IS the resolver (tests re-point it).
	if got := fmt.Sprintf("%T", BackendFactory("opencode", "", dir, nil)); got != opencode {
		t.Errorf("BackendFactory(opencode) = %s, want %s", got, opencode)
	}
}

func TestBackendNameFromStatus(t *testing.T) {
	for _, tc := range []struct {
		line, want string
		ok         bool
	}{
		{"[theboringoffice] backend: opencode", "opencode", true},
		{"[theboringoffice] backend: claudecode", "claudecode", true},
		// the swap grammar: the arrow's RIGHT side latches.
		{"[theboringoffice] backend: opencode → claudecode (turn #3 archived)", "claudecode", true},
		{"[theboringoffice] backend: claudecode → opencode (turn #0 archived)", "opencode", true},
		// not a latch: other status lines, refusals, junk names, the
		// booting placeholder.
		{"backend swap opencode → claudecode refused — office busy: boss turn in flight", "", false},
		{"[theboringoffice] live - http://127.0.0.1:1 | board: in-memory", "", false},
		{"[theboringoffice] backend: zephyr", "", false},
		{"[theboringoffice] live - booting...", "", false},
		{"", "", false},
	} {
		name, ok := backendNameFromStatus(tc.line)
		if name != tc.want || ok != tc.ok {
			t.Errorf("backendNameFromStatus(%q) = %q,%v — want %q,%v", tc.line, name, ok, tc.want, tc.ok)
		}
	}
}

// The reducer latch: boot hints + swap lines update st.BackendName; every
// other EvStatus leaves it alone; office miss copy never latches.
func TestBackendNameLatchReducer(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	if m.st.BackendName != "" {
		t.Fatalf("a fresh office has NO latched backend name yet, got %q", m.st.BackendName)
	}
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "unrelated noise"})
	if m.st.BackendName != "" {
		t.Fatalf("unrelated statuses must not latch, got %q", m.st.BackendName)
	}
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[theboringoffice] backend: opencode"})
	if m.st.BackendName != "opencode" {
		t.Fatalf("the boot hint must latch opencode, got %q", m.st.BackendName)
	}
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[theboringoffice] memory: agentmemory OK"})
	if m.st.BackendName != "opencode" {
		t.Fatalf("an unrelated EvStatus must never UN-latch, got %q", m.st.BackendName)
	}
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[theboringoffice] backend: opencode → claudecode (turn #5 archived)"})
	if m.st.BackendName != "claudecode" {
		t.Fatalf("the swap line must re-latch to the NEW name, got %q", m.st.BackendName)
	}
}

// The topbar-rendering name falls back to brain.json until the boot hint
// latches (pre-hint frames still name the right transport).
func TestBackendNameFallback(t *testing.T) {
	cfg := config.Default()
	cfg.Backend.Name = config.BackendNameClaude
	m := New(&recBackend{}, cfg)
	if got := m.backendName(); got != "claudecode" {
		t.Fatalf("pre-hint the name must resolve from brain.json, got %q", got)
	}
	m.st.BackendName = "" // latch stays empty: the hint always wins once it lands
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[theboringoffice] backend: claudecode"})
	if got := m.backendName(); got != "claudecode" {
		t.Fatalf("post-hint the latch wins, got %q", got)
	}
	cfg2 := config.Default()
	m2 := New(&recBackend{}, cfg2)
	if got := m2.backendName(); got != "opencode" {
		t.Fatalf("default brain.json resolves opencode, got %q", got)
	}
}

// The boot RESTORE pin is per-transport too: a session.json with both
// entries feeds the CLAUdecode pin on a claudecode boot, the legacy slot
// on an opencode boot (coverage of model.go's hydrate leg, sibling to the
// swap-side pin tests in backend_switch_test.go).
func TestBootHydratePinPerBackend(t *testing.T) {
	scratchHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	// Leg 1: an opencode-ONLY (pre-schema-shaped) file — claudecode boots
	// must never cross-pin the serve's id; opencode boots pin it as today.
	if err := SaveSession(cwd, Snapshot(cwd, "ses-old", storedTranscript())); err != nil {
		t.Fatal(err)
	}
	stubClaude := newSwapStub("")
	cfgClaude := config.Default()
	cfgClaude.Backend.Name = "claudecode"
	_ = New(stubClaude, cfgClaude)
	if got := stubClaude.pinnedOverrides(); got != nil {
		t.Fatalf("claudecode boot on an opencode-only session.json must NOT pin the serve id, got %v", got)
	}
	stubOpen := newSwapStub("")
	_ = New(stubOpen, config.Default())
	if got := stubOpen.pinnedOverrides(); len(got) != 1 || got[0] != "ses-old" {
		t.Fatalf("opencode boot must pin the legacy slot, got %v", got)
	}

	// Leg 2: both entries present — each transport gets its own.
	if err := SaveSession(cwd, SessionFile{
		Dir: cwd, PrimaryID: "ses-op", Backend: "claudecode",
		PrimaryIDs: map[string]string{"opencode": "ses-op", "claudecode": "ses-claude-3"},
		Chat:       storedTranscript().Chat,
	}); err != nil {
		t.Fatal(err)
	}
	stubBoth := newSwapStub("")
	cfgBoth := config.Default()
	cfgBoth.Backend.Name = "claudecode"
	_ = New(stubBoth, cfgBoth)
	if got := stubBoth.pinnedOverrides(); len(got) != 1 || got[0] != "ses-claude-3" {
		t.Fatalf("claudecode boot must pin the claudecode entry, got %v", got)
	}
}
