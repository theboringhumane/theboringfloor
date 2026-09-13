// codex_recovery.go — recovers a codex office's last, possibly-interrupted
// turn from the CLI's own on-disk rollout store at office startup.
//
// WHY: switching floors hands a busy floor to a detached background
// office. On opencode the in-flight turn survives; on codex it cannot —
// codex exec runs one turn per process and this office reads its stdout,
// so the office dying kills the turn outright. Worse, codex only ever
// emits/records COMPLETED items, so content mid-generation at the moment
// of death is gone forever — no mechanism recovers it. What IS
// recoverable is everything that finished before the kill: codex writes
// each completed item to ~/.codex/sessions/.../rollout-*-<thread_id>.jsonl
// as it completes (internal/backend/codex_rollout.go's RecoverCodexThread,
// a separate developer's parser — this file only calls it).
//
// GATE (decision, exact): recovery runs once, at startup, ONLY when ALL of
//   - the backend is codex
//   - a codex thread id was persisted to this floor's session.json
//   - the parser reports lastTurnCompleted == false (a cleanly completed
//     last turn is already in the transcript — re-showing it would
//     duplicate content)
//
// Any gate miss, parse error, or empty result is SILENT: no transcript
// row, no error toast — a member who never lost a turn must never see
// anything about this feature (decision).
//
// Recovered rows are informational history, not a live turn: they never
// set busy/pending state, never create a pending boss bubble, and are
// never re-sent to the backend — applyCodexRecovery only ever appends
// already-Pending:false chat rows.
package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/backend"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// codexRecoveryParse is the injected parser seam — the real value is
// backend.RecoverCodexThread; tests substitute a fake so recovery NEVER
// touches a real ~/.codex. Mirrors the floorHandoffSpawn/floorHandoffReady
// convention (floor_handoff.go): a package var over a direct call, purely
// for test substitution.
var codexRecoveryParse = backend.RecoverCodexThread

// codexRecoveryMeta tags every ChatMsg this feature appends (except the
// tool row, whose Meta must instead carry renderTool's "done" state) —
// harmless to the renderer (an unrecognized Meta on a "think"/plain notice
// row is simply ignored) and lets tests identify recovered rows without
// depending on exact copy.
const codexRecoveryMeta = "codex-recovery"

// codexRecoveredMsg is the ONE tea.Msg shape codexRecoveryCmd ever
// returns. Every gate miss, parser error, and the genuinely-empty case all
// land here with a nil Items — one shape for Update to reduce, instead of
// a nil-cmd special case threaded through Init's tea.Batch. The reducer
// (applyCodexRecovery) treats an empty Items as a pure no-op, so "return a
// message that renders nothing" and "return no message" are behaviorally
// identical here; always-a-message was chosen because it keeps the gate
// logic in ONE place (this file) with ONE deterministic, directly
// testable output, rather than splitting it across a bool return and a
// cmd return.
type codexRecoveredMsg struct {
	items []backend.RolloutItem
}

// codexRecoveryCmd builds the tea.Cmd Init() batches in. backendName and
// dir are plain strings captured by the CALLER before this returns — the
// closure below never reads m.* fields: it runs in its own goroutine
// (tea.Cmd semantics) after Init returns, so touching the Model from
// inside it would race the UI goroutine's own mutations. Every step here
// is filesystem I/O (session.json, then the rollout store) and MUST stay
// off that goroutine, which is why the whole gate — not just the parser
// call — lives inside this closure rather than being pre-computed in
// Init().
func codexRecoveryCmd(backendName, dir string) tea.Cmd {
	return func() tea.Msg {
		if backendName != config.BackendNameCodex {
			return codexRecoveredMsg{}
		}
		sf, ok := LoadSession(dir)
		if !ok {
			return codexRecoveredMsg{}
		}
		threadID := sf.primaryIDFor(config.BackendNameCodex)
		if threadID == "" {
			return codexRecoveredMsg{}
		}
		items, lastTurnCompleted, err := codexRecoveryParse(threadID)
		if err != nil || lastTurnCompleted || len(items) == 0 {
			return codexRecoveredMsg{}
		}
		return codexRecoveredMsg{items: items}
	}
}

// applyCodexRecovery reduces one codexRecoveredMsg. Idempotent by
// construction: Init() fires codexRecoveryCmd exactly once per
// tea.Program run, but codexRecoveryApplied still guards against a stray
// redelivery duplicating the recovered block — the SAME safety
// floorStatusInFlight gives the status sweep, applied here to a one-shot
// startup command instead of a recurring tick.
//
// This method ONLY appends already-settled (Pending:false) ChatMsg rows —
// it never touches st.Bubbles, BossThinking, BossDelegating, or calls the
// backend, so officeBusy(m.st, ...) reads exactly as it did before this
// ran (see power.go) and the office stays idle.
func (m *Model) applyCodexRecovery(msg codexRecoveredMsg) {
	if m.codexRecoveryApplied {
		return
	}
	m.codexRecoveryApplied = true
	if len(msg.items) == 0 {
		return
	}
	chat := append([]state.ChatMsg(nil), m.st.Chat...)
	chat = appendChat(chat, codexRecoveryMarker(len(msg.items)))
	for _, item := range msg.items {
		chat = appendChat(chat, codexRecoveryRow(item))
	}
	m.st.Chat = capChat(chat)
	m.tabs.SetState(m.st)
}

// codexRecoveryMarker is the single combined statement the decisions
// require: how many completed steps were recovered, that they came from
// an INTERRUPTED turn (the office died mid-generation), and the honest
// limitation stated once — whatever codex was still generating at that
// moment was never written to its rollout store and is not part of what
// follows. Rendered as a plain office notice (From "office", no Kind) —
// the existing dim local-notice treatment (renderNotice, chat.go) that
// already reads as "the office told you something", never as live boss
// output.
func codexRecoveryMarker(n int) state.ChatMsg {
	plural := "s"
	if n == 1 {
		plural = ""
	}
	text := fmt.Sprintf(
		"↩ recovered %d completed step%s from an interrupted codex turn — "+
			"the office was killed mid-turn. Whatever codex was still "+
			"generating at that moment was never saved and is not shown below.",
		n, plural)
	return state.ChatMsg{
		ID:   nextMsgID(),
		From: "office",
		Meta: codexRecoveryMeta,
		Text: text,
		At:   time.Now().UnixMilli(),
	}
}

// codexRecoveryRow maps one backend.RolloutItem onto the transcript row
// kind that already exists for that shape of content — reusing chat.go's
// established renderers instead of inventing new ones (this package may
// not touch internal/panels):
//
//   - "reasoning" -> Kind "think" (the SAME row a live boss thought uses,
//     collapsed dim "thinking · N lines" by default) — never rendered as
//     assistant prose.
//   - "tool"      -> Kind "tool" (the SAME dim "[tool] …" one-liner a live
//     boss tool call uses), Meta "done" (renderTool's completed styling —
//     the recovered step DID finish, it is simply history now) — never
//     rendered as prose either.
//   - "message"   -> Kind "office" (the concierge's INFO-cyan "office ›"
//     bubble — a REAL rendered turn, but visually a peer of the boss's
//     bubble, never the boss's own yellow "boss ›" bubble a live reply
//     uses) — Text carries an inline "↩ recovered — " marker so an
//     individual bubble reads as recovered even scrolled away from the
//     header notice.
//   - "other"/anything unrecognized -> a plain office notice, same
//     inline marker, for content the parser could not classify.
//
// CompletedAt (unix millis) rides through so the row's timeline position
// reflects when codex actually finished it; 0 (unknown) falls back to
// "now" rather than sorting to the epoch.
func codexRecoveryRow(item backend.RolloutItem) state.ChatMsg {
	at := item.CompletedAt
	if at == 0 {
		at = time.Now().UnixMilli()
	}
	base := state.ChatMsg{ID: nextMsgID(), At: at}
	switch item.Kind {
	case "reasoning":
		base.From = "boss"
		base.Kind = "think"
		base.Meta = codexRecoveryMeta
		base.Text = item.Text
	case "tool":
		base.From = "boss"
		base.Kind = "tool"
		base.Meta = "done" // renderTool reads Meta as the completion state
		base.Text = strings.ReplaceAll(item.Text, "\n", " ")
	case "message":
		base.From = "office"
		base.Kind = "office"
		base.Meta = codexRecoveryMeta
		base.Text = "↩ recovered — " + item.Text
	default: // "other" and anything the parser did not recognize
		base.From = "office"
		base.Meta = codexRecoveryMeta
		base.Text = "↩ recovered — " + item.Text
	}
	return base
}
