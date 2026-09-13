// codex_recovery_test.go — hermetic proof for codex_recovery.go's gate,
// idempotence, and rendering. Every test substitutes codexRecoveryParse
// (never touches a real ~/.codex) and points THEFLOOR_HOME at a fresh
// t.TempDir() (never touches the developer's real home), per the brief's
// "tests must never touch it" rule.
package app

import (
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/backend"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// countingBackend is a minimal state.Backend stub that counts every
// backend-facing call so a test can assert recovery never triggers one.
type countingBackend struct {
	mode      state.Mode
	sendCalls int
}

func (b *countingBackend) Mode() state.Mode                      { return b.mode }
func (b *countingBackend) Start(func(state.Event)) error         { return nil }
func (b *countingBackend) Send(string) error                     { b.sendCalls++; return nil }
func (b *countingBackend) AnswerPermission(string, string) error { return nil }
func (b *countingBackend) AnswerQuestion(string, [][]string) error {
	return nil
}
func (b *countingBackend) RejectQuestion(string) error            { return nil }
func (b *countingBackend) MCPServers() ([]state.MCPServer, error) { return nil, nil }
func (b *countingBackend) ReconnectMCP(string) error              { return nil }
func (b *countingBackend) Stop() error                            { return nil }

// newRecoveryModel builds a Model pinned to backendName with sessDir set
// to a throwaway directory — New()'s own session-restore logic never
// fires here (the stub is ModeDemo), so sessDir is set directly, same as
// any other field access from within package app's own tests.
func newRecoveryModel(t *testing.T, backendName string) Model {
	t.Helper()
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	cfg := config.Default()
	cfg.Backend.Name = backendName
	m := New(&countingBackend{mode: state.ModeDemo}, cfg)
	m.sessDir = t.TempDir()
	return m
}

// plantCodexThread writes a minimal session.json pinning threadID as the
// codex primary — the only on-disk state codexRecoveryCmd itself reads
// (the rollout file read is stubbed out via codexRecoveryParse below).
func plantCodexThread(t *testing.T, dir, threadID string) {
	t.Helper()
	sf := SessionFile{Dir: dir, Backend: config.BackendNameCodex, PrimaryIDs: map[string]string{config.BackendNameCodex: threadID}}
	if err := SaveSession(dir, sf); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
}

// stubParse installs a fake codexRecoveryParse for the duration of the
// test and restores the real one on cleanup — the seam requirement 7
// calls for, mirroring floorHandoffSpawn/floorHandoffReady.
func stubParse(t *testing.T, fn func(string) ([]backend.RolloutItem, bool, error)) {
	t.Helper()
	orig := codexRecoveryParse
	codexRecoveryParse = fn
	t.Cleanup(func() { codexRecoveryParse = orig })
}

func chatMetaTexts(m Model, meta string) []string {
	var out []string
	for _, c := range m.st.Chat {
		if c.Meta == meta {
			out = append(out, c.Text)
		}
	}
	return out
}

// TestCodexRecoveryRendersInterruptedTurn is requirement 3+4+7's main
// positive case: codex backend + persisted thread id + lastTurnCompleted
// == false renders the marker notice plus one row per item, in order,
// mapped onto sensible existing row kinds.
func TestCodexRecoveryRendersInterruptedTurn(t *testing.T) {
	m := newRecoveryModel(t, config.BackendNameCodex)
	plantCodexThread(t, m.sessDir, "thread-abc")
	var gotThreadID string
	stubParse(t, func(id string) ([]backend.RolloutItem, bool, error) {
		gotThreadID = id
		return []backend.RolloutItem{
			{Kind: "reasoning", Text: "weighing two approaches", CompletedAt: 1},
			{Kind: "tool", Text: "read · src/main.go", CompletedAt: 2},
			{Kind: "message", Text: "done — patched the parser.", CompletedAt: 3},
		}, false, nil
	})

	cmd := codexRecoveryCmd(m.backendName(), m.sessDir)
	msg := cmd()
	nm, _ := m.Update(msg)
	m = nm.(Model)

	if gotThreadID != "thread-abc" {
		t.Fatalf("parser called with threadID=%q, want thread-abc", gotThreadID)
	}
	if len(m.st.Chat) != 4 { // marker + 3 items
		t.Fatalf("chat len=%d, want 4 (marker + 3 items):\n%+v", len(m.st.Chat), m.st.Chat)
	}
	marker := m.st.Chat[0]
	if marker.From != "office" || marker.Kind != "" {
		t.Fatalf("marker row = %+v, want From=office Kind=\"\"", marker)
	}
	if !strings.Contains(marker.Text, "3 completed step") || !strings.Contains(marker.Text, "interrupted codex turn") ||
		!strings.Contains(marker.Text, "never saved") {
		t.Fatalf("marker text missing required copy: %q", marker.Text)
	}
	think := m.st.Chat[1]
	if think.Kind != "think" || think.From != "boss" || think.Text != "weighing two approaches" {
		t.Fatalf("reasoning row = %+v, want Kind=think From=boss", think)
	}
	tool := m.st.Chat[2]
	if tool.Kind != "tool" || tool.From != "boss" || tool.Meta != "done" || tool.Text != "read · src/main.go" {
		t.Fatalf("tool row = %+v, want Kind=tool From=boss Meta=done", tool)
	}
	msgRow := m.st.Chat[3]
	if msgRow.Kind != "office" || msgRow.From != "office" || !strings.HasPrefix(msgRow.Text, "↩ recovered — ") ||
		!strings.Contains(msgRow.Text, "done — patched the parser.") {
		t.Fatalf("message row = %+v, want Kind=office with recovered prefix", msgRow)
	}
}

// TestCodexRecoveryNoOpWhenLastTurnCompleted proves the decision gate:
// lastTurnCompleted == true renders NOTHING, even with non-empty items —
// that content is already in the transcript.
func TestCodexRecoveryNoOpWhenLastTurnCompleted(t *testing.T) {
	m := newRecoveryModel(t, config.BackendNameCodex)
	plantCodexThread(t, m.sessDir, "thread-done")
	stubParse(t, func(string) ([]backend.RolloutItem, bool, error) {
		return []backend.RolloutItem{{Kind: "message", Text: "already shown"}}, true, nil
	})

	msg := codexRecoveryCmd(m.backendName(), m.sessDir)()
	nm, _ := m.Update(msg)
	m = nm.(Model)

	if len(m.st.Chat) != 0 {
		t.Fatalf("chat should stay empty on a completed last turn, got %+v", m.st.Chat)
	}
}

// TestCodexRecoveryNoOpForNonCodexBackend proves the backend gate: the
// parser must never even be consulted for opencode/claudecode.
func TestCodexRecoveryNoOpForNonCodexBackend(t *testing.T) {
	m := newRecoveryModel(t, config.BackendNameDefault)
	plantCodexThread(t, m.sessDir, "thread-x") // present, but backend isn't codex
	called := false
	stubParse(t, func(string) ([]backend.RolloutItem, bool, error) {
		called = true
		return []backend.RolloutItem{{Kind: "message", Text: "should never be read"}}, false, nil
	})

	msg := codexRecoveryCmd(m.backendName(), m.sessDir)()
	nm, _ := m.Update(msg)
	m = nm.(Model)

	if called {
		t.Fatal("parser must never be called for a non-codex backend")
	}
	if len(m.st.Chat) != 0 {
		t.Fatalf("chat should stay empty for a non-codex backend, got %+v", m.st.Chat)
	}
}

// TestCodexRecoveryNoOpWhenNoThreadIDPersisted proves the persisted-id
// gate: a codex backend with no session.json (or none pinning codex) at
// all skips recovery — never calling the parser with an empty id.
func TestCodexRecoveryNoOpWhenNoThreadIDPersisted(t *testing.T) {
	m := newRecoveryModel(t, config.BackendNameCodex) // no plantCodexThread call
	called := false
	stubParse(t, func(string) ([]backend.RolloutItem, bool, error) {
		called = true
		return nil, false, nil
	})

	msg := codexRecoveryCmd(m.backendName(), m.sessDir)()
	nm, _ := m.Update(msg)
	m = nm.(Model)

	if called {
		t.Fatal("parser must never be called with no persisted thread id")
	}
	if len(m.st.Chat) != 0 {
		t.Fatalf("chat should stay empty with no persisted thread id, got %+v", m.st.Chat)
	}
}

// TestCodexRecoveryNoOpWhenParserReturnsEmpty proves the empty-result
// gate: a matching thread id whose parser call yields nothing (missing
// store, unparseable file — RecoverCodexThread's own documented
// best-effort miss) renders nothing.
func TestCodexRecoveryNoOpWhenParserReturnsEmpty(t *testing.T) {
	m := newRecoveryModel(t, config.BackendNameCodex)
	plantCodexThread(t, m.sessDir, "thread-empty")
	stubParse(t, func(string) ([]backend.RolloutItem, bool, error) {
		return nil, false, nil
	})

	msg := codexRecoveryCmd(m.backendName(), m.sessDir)()
	nm, _ := m.Update(msg)
	m = nm.(Model)

	if len(m.st.Chat) != 0 {
		t.Fatalf("chat should stay empty when the parser returns nothing, got %+v", m.st.Chat)
	}
}

// TestCodexRecoveryNeverSetsBusyOrSendsToBackend is requirement 5: after
// a successful recovery the office reads idle (officeBusy false) and the
// backend recorded zero Send calls.
func TestCodexRecoveryNeverSetsBusyOrSendsToBackend(t *testing.T) {
	m := newRecoveryModel(t, config.BackendNameCodex)
	cb := m.backend.(*countingBackend)
	plantCodexThread(t, m.sessDir, "thread-busy-check")
	stubParse(t, func(string) ([]backend.RolloutItem, bool, error) {
		return []backend.RolloutItem{{Kind: "message", Text: "hello"}}, false, nil
	})

	msg := codexRecoveryCmd(m.backendName(), m.sessDir)()
	nm, _ := m.Update(msg)
	m = nm.(Model)

	if len(m.st.Chat) == 0 {
		t.Fatal("setup broken: expected recovered rows to assert idleness against")
	}
	if officeBusy(m.st, false, false) {
		t.Fatalf("office must read idle after recovery, state=%+v", m.st)
	}
	if len(m.st.Bubbles) != 0 {
		t.Fatalf("recovery must not create a pending bubble, got %+v", m.st.Bubbles)
	}
	if cb.sendCalls != 0 {
		t.Fatalf("recovery must never call backend.Send, got %d calls", cb.sendCalls)
	}
}

// TestCodexRecoveryIsIdempotent is requirement 6: a second delivery of
// the SAME codexRecoveredMsg in one session must not duplicate rows.
func TestCodexRecoveryIsIdempotent(t *testing.T) {
	m := newRecoveryModel(t, config.BackendNameCodex)
	plantCodexThread(t, m.sessDir, "thread-idem")
	stubParse(t, func(string) ([]backend.RolloutItem, bool, error) {
		return []backend.RolloutItem{{Kind: "message", Text: "once only"}}, false, nil
	})

	msg := codexRecoveryCmd(m.backendName(), m.sessDir)()
	nm, _ := m.Update(msg)
	m = nm.(Model)
	firstLen := len(m.st.Chat)
	if firstLen != 2 { // marker + 1 item
		t.Fatalf("chat len=%d after first delivery, want 2", firstLen)
	}

	// Redeliver the exact same message.
	nm2, _ := m.Update(msg)
	m = nm2.(Model)
	if len(m.st.Chat) != firstLen {
		t.Fatalf("chat len=%d after second delivery, want unchanged %d (idempotence broken):\n%+v",
			len(m.st.Chat), firstLen, m.st.Chat)
	}
	recovered := chatMetaTexts(m, codexRecoveryMeta)
	if strings.Count(strings.Join(recovered, "\n"), "once only") != 1 {
		t.Fatalf("item text duplicated across deliveries: %+v", recovered)
	}
}
