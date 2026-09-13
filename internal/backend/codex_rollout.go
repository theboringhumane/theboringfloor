package backend

// Read-only recovery from the Codex CLI's on-disk rollout store.
//
// Why this exists: the office drives `codex` by spawning `codex exec` once
// per turn and reading its stdout synchronously (see codex.go). When the
// office process dies mid-turn — most commonly a floor switch that kills
// the parent before the child's turn finishes — that turn's output is lost
// from the office's point of view, even though the codex CLI itself kept
// writing a durable, incremental record to disk the whole time. This file
// reads that record back so a relaunched office can recover what already
// completed, with no cooperation from the dying process required.
//
// Store shape, verified against real files on this machine (not merely
// assumed from documentation, since none exists — see below):
//
//   - One JSONL file per thread at
//     ~/.codex/sessions/YYYY/MM/DD/rollout-<ts>-<thread_id>[_<other_uuid>].jsonl.
//     The UUID immediately after the timestamp in the FILENAME is the
//     thread id (confirmed equal to the file's own session_meta.id and to
//     item_completed's thread_id field on every real file inspected). A
//     second, trailing "_<uuid>" segment is NOT the thread id — it did not
//     match session_meta.session_id, session_id, or any field found inside
//     the matching file's own content on the one example inspected; treat
//     it as an opaque continuation/fork marker and ignore it for lookup.
//   - A thread can have more than one file (observed on this machine: a
//     bare rollout-*-<id>.jsonl next to a later
//     rollout-*-<id>_<other>.jsonl for the same thread id, from a session
//     that was later continued/forked). This package picks the
//     most-recently-modified match — the newest file is the one that was
//     appended to last and therefore holds the freshest recoverable state;
//     an older sibling can only be a stale snapshot from before a fork.
//   - Every line is one JSON object with at least a top-level "type". The
//     type that matters for recovery is "event_msg" with
//     payload.type == "item_completed": this is the CLI's own record that
//     one piece of work — a message, a reasoning block, a tool call —
//     finished. There is no partial-item representation anywhere in the
//     schema: an item in flight when the process died left no trace and
//     cannot be recovered. Other top-level types seen on this machine
//     (response_item, token_usage_record, token_count, turn_context,
//     world_state, inter_agent_communication_metadata, compacted,
//     session_meta) are ignored here: response_item duplicates
//     item_completed's content (often as encrypted_content, useless for
//     recovery) and the rest are bookkeeping, not recoverable work.
//   - "event_msg" with payload.type == "task_complete" additionally carries
//     last_agent_message: the full final agent message text for that turn,
//     independent of whether an AgentMessage item_completed line for it is
//     present. It is surfaced as its own message-kind item because a
//     schema change could drop the AgentMessage item_completed line while
//     keeping this one, and losing the single most useful string in the
//     file to save a possible duplicate is the wrong trade.
//
// This schema is Codex-internal, undocumented, and can change silently
// across CLI versions. Every decode here is defensive: an unrecognized or
// malformed line is skipped, never fatal, and a file with none of the
// expected shape yields nothing rather than garbage or a panic.

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

// RolloutItem is one completed item recovered from codex's rollout store.
type RolloutItem struct {
	Kind        string // normalized: "message" | "reasoning" | "tool" | "other"
	Text        string // the item's human-readable content, may be empty
	CompletedAt int64  // unix millis, 0 when unknown
}

// codexRolloutRoot resolves the directory that holds the YYYY/MM/DD rollout
// tree. A package var, not a plain function call, so tests substitute a
// fake root under t.TempDir() and never touch the member's real ~/.codex —
// the same seam convention as floorHandoffSpawn (internal/app/floor_handoff.go).
var codexRolloutRoot = defaultCodexRolloutRoot

func defaultCodexRolloutRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".codex", "sessions")
}

// errCodexEmptyThreadID is the one genuinely exceptional input this package
// rejects outright: every other failure to recover (missing store, missing
// thread, unreadable file, unrecognized schema) is normal best-effort
// absence-of-data, not an error a caller has to handle.
var errCodexEmptyThreadID = errors.New("backend: RecoverCodexThread requires a non-empty thread id")

// RecoverCodexThread reads the rollout store for threadID and returns the
// completed items in chronological order, plus whether the thread's last
// turn reached completion. A missing store, an unreadable file, or an
// unrecognized schema yields (nil, false, nil) — recovery is best-effort
// and must never be an error a caller has to handle.
func RecoverCodexThread(threadID string) ([]RolloutItem, bool, error) {
	if threadID == "" {
		return nil, false, errCodexEmptyThreadID
	}

	path := locateCodexRolloutFile(threadID)
	if path == "" {
		return nil, false, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, false, nil
	}
	defer f.Close()

	return parseCodexRollout(f)
}

// locateCodexRolloutFile finds the rollout file for threadID with a
// targeted glob — never a full walk of the store, which on a long-lived
// member's machine can hold thousands of sessions across many months. The
// thread id sits right after the timestamp in the filename
// (rollout-<ts>-<thread_id>.jsonl or rollout-<ts>-<thread_id>_<other>.jsonl),
// so a glob anchored on that id is exact enough: a false-positive
// substring match would require another thread id to contain this whole
// UUID, which does not happen in practice. When more than one file
// matches (a thread that was later forked/continued into a sibling file),
// the most recently modified one wins, since that is the file that kept
// receiving completed items last.
func locateCodexRolloutFile(threadID string) string {
	root := codexRolloutRoot()
	if root == "" {
		return ""
	}

	pattern := filepath.Join(root, "*", "*", "*", "rollout-*"+threadID+"*.jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	if len(matches) == 1 {
		return matches[0]
	}

	// Precompute every mtime up front (zero value when Stat fails) so the
	// comparator below is a strict, transitive ordering over fixed values —
	// sort.Slice requires that, and stat-ing lazily inside the comparator
	// could violate it if a file changes between comparisons.
	mtimes := make(map[string]int64, len(matches))
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil {
			mtimes[m] = info.ModTime().UnixNano()
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if mtimes[matches[i]] != mtimes[matches[j]] {
			return mtimes[matches[i]] > mtimes[matches[j]]
		}
		return matches[i] < matches[j] // deterministic tie-break
	})
	return matches[0]
}

// rolloutLine is the minimal top-level envelope every rollout line shares.
// Everything past "type" is decoded lazily via json.RawMessage so an
// unrecognized or reshaped payload never breaks decoding of the envelope
// itself.
type rolloutLine struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// rolloutEventPayload covers the event_msg payload shapes this package
// understands: item_completed and task_complete. Unknown payload.type
// values decode fine (their fields simply stay zero) and are skipped by
// the caller.
type rolloutEventPayload struct {
	Type string `json:"type"`

	// item_completed
	Item struct {
		Type        string   `json:"type"`
		SummaryText []string `json:"summary_text"`
		Content     []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Command []string `json:"command"`
		Stdout  string   `json:"stdout"`
		Server  string   `json:"server"`
		Tool    string   `json:"tool"`
		Result  struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	} `json:"item"`
	CompletedAtMs int64 `json:"completed_at_ms"`

	// task_started / task_complete
	TurnID          string `json:"turn_id"`
	LastAgentMsg    string `json:"last_agent_message"`
	CompletedAtSecs int64  `json:"completed_at"`
}

// parseCodexRollout streams f line by line — the store can hold
// multi-megabyte files, so nothing here reads the whole file into memory
// at once — decoding each line defensively and collecting completed items
// in chronological order (the file itself is already append-only and
// therefore chronologically ordered; no re-sorting is needed or done).
func parseCodexRollout(f *os.File) ([]RolloutItem, bool, error) {
	var items []RolloutItem

	// lastTurnCompleted tracks only the MOST RECENTLY started turn: a
	// task_started sets pending=true for that turn id; a matching
	// task_complete clears it. Any earlier turn's own completion (or a
	// same-turn item_completed line that arrives after task_complete, as
	// happens for background command output on this machine) never
	// reopens or otherwise affects this — it answers exactly "did the
	// LAST turn this file recorded starting also finish", which is the
	// wording of the contract. A file with no task_started at all reports
	// false: there is no evidence of a completed turn to claim.
	var lastStartedTurnID string
	var lastTurnCompleted bool

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var envelope rolloutLine
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue // malformed line: skip it, keep going
		}
		if envelope.Type != "event_msg" || len(envelope.Payload) == 0 {
			continue
		}

		var payload rolloutEventPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			continue // malformed payload: skip it, keep going
		}

		switch payload.Type {
		case "task_started":
			lastStartedTurnID = payload.TurnID
			lastTurnCompleted = false

		case "task_complete":
			if payload.TurnID != "" && payload.TurnID == lastStartedTurnID {
				lastTurnCompleted = true
			}
			if payload.LastAgentMsg != "" {
				items = append(items, RolloutItem{
					Kind:        "message",
					Text:        payload.LastAgentMsg,
					CompletedAt: payload.CompletedAtSecs * 1000,
				})
			}

		case "item_completed":
			if item, ok := normalizeCodexRolloutItem(payload); ok {
				items = append(items, item)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		// Truncated/unreadable mid-file: return whatever completed items
		// were recovered before the read failed rather than discarding
		// them, per the "never an error a caller has to handle" contract.
		return items, lastTurnCompleted, nil
	}

	return items, lastTurnCompleted, nil
}

// normalizeCodexRolloutItem maps one item_completed payload onto the four
// Kind values the contract exposes. Recognized item.type values, observed
// directly across real rollout files on this machine:
//
//	AgentMessage     -> "message"   (item.content[].text, type "Text")
//	Reasoning        -> "reasoning" (item.summary_text, joined)
//	CommandExecution -> "tool"      (item.stdout, else the command line)
//	McpToolCall      -> "tool"      (item.result.content[].text)
//	FileChange       -> "tool"      (no reliable single text field; empty)
//	SubAgentActivity -> "other"     (no text field; empty)
//	ContextCompaction-> "other"     (no text field; empty)
//	anything else    -> "other"     (best-effort text scrape, never dropped)
//
// ok is false only when the line had no recognizable item at all (empty
// item.type), so the caller can skip it instead of appending a fully
// empty placeholder.
func normalizeCodexRolloutItem(payload rolloutEventPayload) (RolloutItem, bool) {
	item := payload.Item
	if item.Type == "" {
		return RolloutItem{}, false
	}

	out := RolloutItem{CompletedAt: payload.CompletedAtMs}

	switch item.Type {
	case "AgentMessage":
		out.Kind = "message"
		out.Text = joinContentText(item.Content)

	case "Reasoning":
		out.Kind = "reasoning"
		out.Text = joinStrings(item.SummaryText)

	case "CommandExecution":
		out.Kind = "tool"
		if item.Stdout != "" {
			out.Text = item.Stdout
		} else {
			out.Text = joinStrings(item.Command)
		}

	case "McpToolCall":
		out.Kind = "tool"
		out.Text = joinContentText(item.Result.Content)

	case "FileChange":
		out.Kind = "tool"

	case "SubAgentActivity", "ContextCompaction":
		out.Kind = "other"

	default:
		out.Kind = "other"
		// Best-effort scrape for a future/unknown item type: try every
		// text-bearing field this package already knows how to read
		// before giving up on empty text. Never drop the item itself —
		// an empty Text with the right CompletedAt still tells a caller
		// something finished.
		switch {
		case len(item.Content) > 0:
			out.Text = joinContentText(item.Content)
		case len(item.SummaryText) > 0:
			out.Text = joinStrings(item.SummaryText)
		case item.Stdout != "":
			out.Text = item.Stdout
		case len(item.Result.Content) > 0:
			out.Text = joinContentText(item.Result.Content)
		}
	}

	return out, true
}

func joinContentText(content []struct {
	Type string `json:"type"`
	Text string `json:"text"`
}) string {
	var out string
	for _, c := range content {
		if c.Text == "" {
			continue
		}
		if out != "" {
			out += "\n"
		}
		out += c.Text
	}
	return out
}

func joinStrings(parts []string) string {
	var out string
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out != "" {
			out += "\n"
		}
		out += p
	}
	return out
}
