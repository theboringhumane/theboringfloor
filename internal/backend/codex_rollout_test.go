package backend

// Fixtures below reproduce the exact line shapes observed while inspecting
// real ~/.codex/sessions/**/rollout-*.jsonl files on this machine (never
// written to or read from here — every test uses t.TempDir() and the
// codexRolloutRoot seam). Field names, nesting, and the underscored
// "item_completed"/"task_complete" payload.type values all match what a
// real Codex CLI (v0.153.4 observed) wrote to disk.

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withCodexRolloutRoot points codexRolloutRoot at a fresh temp dir for the
// duration of the test and restores the real seam on cleanup — the same
// convention floor_handoff_test.go uses for floorHandoffSpawn.
func withCodexRolloutRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	orig := codexRolloutRoot
	codexRolloutRoot = func() string { return root }
	t.Cleanup(func() { codexRolloutRoot = orig })
	return root
}

// writeRollout drops content at root/2026/09/13/rollout-<ts>-<threadID>.jsonl,
// matching the real YYYY/MM/DD nesting, and returns the file path.
func writeRollout(t *testing.T, root, threadID, content string) string {
	t.Helper()
	dir := filepath.Join(root, "2026", "09", "13")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, "rollout-2026-09-13T12-00-00-"+threadID+".jsonl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

const fixtureThreadID = "01a09273-3bad-7080-a53e-72e0a0c608fd"

// realisticMultiItemRollout mirrors a real two-turn thread: task_started,
// a Reasoning item, an AgentMessage item, task_complete (first turn, with
// last_agent_message), then a second turn's task_started, a
// CommandExecution item, and its task_complete.
func realisticMultiItemRollout(threadID string) string {
	return `{"timestamp":"2026-09-13T12:00:00.000Z","ordinal":0,"type":"session_meta","payload":{"session_id":"parent-uuid-not-thread-id","id":"` + threadID + `","cwd":"/work"}}
{"timestamp":"2026-09-13T12:00:01.000Z","ordinal":1,"type":"event_msg","payload":{"type":"task_started","turn_id":"turn-1","started_at":1789163296}}
{"timestamp":"2026-09-13T12:00:02.000Z","ordinal":2,"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"turn-1","item":{"type":"Reasoning","id":"rs_1","summary_text":["Thinking about the plan"],"raw_content":[]},"started_at_ms":1000,"completed_at_ms":2000}}
{"timestamp":"2026-09-13T12:00:03.000Z","ordinal":3,"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"turn-1","item":{"type":"AgentMessage","id":"msg_1","content":[{"type":"Text","text":"First reply"}],"phase":"final_answer"},"started_at_ms":2000,"completed_at_ms":3000}}
{"timestamp":"2026-09-13T12:00:04.000Z","ordinal":4,"type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-1","last_agent_message":"First reply","started_at":1789163296,"completed_at":1789163300,"duration_ms":4000}}
{"timestamp":"2026-09-13T12:00:05.000Z","ordinal":5,"type":"event_msg","payload":{"type":"task_started","turn_id":"turn-2","started_at":1789163301}}
{"timestamp":"2026-09-13T12:00:06.000Z","ordinal":6,"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"turn-2","item":{"type":"CommandExecution","id":"exec_1","command":["/bin/zsh","-lc","echo hi"],"stdout":"hi\n","status":"completed"},"started_at_ms":4000,"completed_at_ms":5000}}
{"timestamp":"2026-09-13T12:00:07.000Z","ordinal":7,"type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-2","last_agent_message":"Done.","started_at":1789163301,"completed_at":1789163305,"duration_ms":4000}}
`
}

func TestRecoverCodexThreadRealisticMultiItemInOrder(t *testing.T) {
	root := withCodexRolloutRoot(t)
	writeRollout(t, root, fixtureThreadID, realisticMultiItemRollout(fixtureThreadID))

	items, lastTurnCompleted, err := RecoverCodexThread(fixtureThreadID)
	if err != nil {
		t.Fatalf("RecoverCodexThread: unexpected error %v", err)
	}
	if !lastTurnCompleted {
		t.Fatalf("lastTurnCompleted = false, want true (file ends with a matching task_complete for turn-2)")
	}

	// Chronological order: reasoning, agent message, first task_complete's
	// last_agent_message, command output, second task_complete's
	// last_agent_message.
	wantKinds := []string{"reasoning", "message", "message", "tool", "message"}
	if len(items) != len(wantKinds) {
		t.Fatalf("got %d items, want %d: %+v", len(items), len(wantKinds), items)
	}
	for i, want := range wantKinds {
		if items[i].Kind != want {
			t.Errorf("items[%d].Kind = %q, want %q (item=%+v)", i, items[i].Kind, want, items[i])
		}
	}
	if items[0].Text != "Thinking about the plan" {
		t.Errorf("items[0].Text = %q, want reasoning summary", items[0].Text)
	}
	if items[1].Text != "First reply" {
		t.Errorf("items[1].Text = %q, want AgentMessage text", items[1].Text)
	}
	if items[2].Text != "First reply" {
		t.Errorf("items[2].Text = %q, want first task_complete's last_agent_message", items[2].Text)
	}
	if items[3].Text != "hi\n" {
		t.Errorf("items[3].Text = %q, want command stdout", items[3].Text)
	}
	if items[4].Text != "Done." {
		t.Errorf("items[4].Text = %q, want second task_complete's last_agent_message", items[4].Text)
	}
	// CompletedAt propagates from completed_at_ms for item_completed lines.
	if items[0].CompletedAt != 2000 {
		t.Errorf("items[0].CompletedAt = %d, want 2000", items[0].CompletedAt)
	}
	// task_complete's completed_at is UNIX SECONDS in the real schema;
	// RolloutItem.CompletedAt is documented as millis, so it must be
	// scaled up by 1000, not passed through raw.
	if items[2].CompletedAt != 1789163300*1000 {
		t.Errorf("items[2].CompletedAt = %d, want %d (completed_at seconds * 1000)", items[2].CompletedAt, 1789163300*1000)
	}
}

func TestRecoverCodexThreadMidTurnReportsIncomplete(t *testing.T) {
	root := withCodexRolloutRoot(t)
	threadID := "01a09b56-04a0-73b2-aabf-19dd47a7b74c"
	// Mirrors a real crashed-mid-turn file observed on this machine: a
	// task_started with no matching task_complete anywhere after it.
	content := `{"timestamp":"2026-09-13T20-45-00.000Z","ordinal":0,"type":"event_msg","payload":{"type":"task_started","turn_id":"turn-open","started_at":1}}
{"timestamp":"2026-09-13T20-45-01.000Z","ordinal":1,"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"turn-open","item":{"type":"AgentMessage","id":"msg_1","content":[{"type":"Text","text":"partial progress"}]},"completed_at_ms":10}}
`
	writeRollout(t, root, threadID, content)

	items, lastTurnCompleted, err := RecoverCodexThread(threadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lastTurnCompleted {
		t.Fatalf("lastTurnCompleted = true, want false (no task_complete for turn-open)")
	}
	if len(items) != 1 || items[0].Text != "partial progress" {
		t.Fatalf("expected the one completed item to still be recovered, got %+v", items)
	}
}

func TestRecoverCodexThreadCompletedTurnReportsTrue(t *testing.T) {
	root := withCodexRolloutRoot(t)
	threadID := "01a09842-396c-7643-8b4d-c72486149020"
	content := `{"timestamp":"2026-09-13T06-24-32.000Z","ordinal":0,"type":"event_msg","payload":{"type":"task_started","turn_id":"turn-a","started_at":1}}
{"timestamp":"2026-09-13T06-24-33.000Z","ordinal":1,"type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-a","last_agent_message":"all done","started_at":1,"completed_at":2}}
`
	writeRollout(t, root, threadID, content)

	items, lastTurnCompleted, err := RecoverCodexThread(threadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !lastTurnCompleted {
		t.Fatalf("lastTurnCompleted = false, want true")
	}
	if len(items) != 1 || items[0].Text != "all done" {
		t.Fatalf("expected last_agent_message item, got %+v", items)
	}
}

func TestRecoverCodexThreadSkipsMalformedLineMidFile(t *testing.T) {
	root := withCodexRolloutRoot(t)
	threadID := "01a08e8e-5bde-7d13-bbdc-2f895f84e738"
	content := `{"timestamp":"2026-09-13T00-00-00.000Z","ordinal":0,"type":"event_msg","payload":{"type":"task_started","turn_id":"turn-1","started_at":1}}
{"timestamp":"2026-09-13T00-00-01.000Z","ordinal":1,"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"turn-1","item":{"type":"Reasoning","id":"rs_1","summary_text":["before the break"]},"completed_at_ms":1}}
{this is not valid json at all
{"timestamp":"2026-09-13T00-00-03.000Z","ordinal":3,"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"turn-1","item":{"type":"AgentMessage","id":"msg_1","content":[{"type":"Text","text":"after the break"}]},"completed_at_ms":3}}
{"timestamp":"2026-09-13T00-00-04.000Z","ordinal":4,"type":"event_msg","payload":{"type":"task_complete","turn_id":"turn-1","last_agent_message":"after the break","started_at":1,"completed_at":4}}
`
	writeRollout(t, root, threadID, content)

	items, lastTurnCompleted, err := RecoverCodexThread(threadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !lastTurnCompleted {
		t.Fatalf("lastTurnCompleted = false, want true (the malformed line should not affect turn tracking)")
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3 (malformed line skipped, both sides of it still parse): %+v", len(items), items)
	}
	if items[0].Text != "before the break" || items[1].Text != "after the break" || items[2].Text != "after the break" {
		t.Fatalf("unexpected item texts: %+v", items)
	}
}

func TestRecoverCodexThreadAlienSchemaYieldsNothing(t *testing.T) {
	root := withCodexRolloutRoot(t)
	threadID := "01a00000-0000-0000-0000-000000000000"
	// Valid JSONL, but none of it matches the event_msg/item_completed or
	// task_complete shape this package understands.
	content := `{"kind":"customer_record","id":1,"name":"not a codex line"}
{"type":"world_state","payload":{"full":true,"state":{}}}
{"type":"turn_context","payload":{"turn_id":"x","cwd":"/tmp"}}
`
	writeRollout(t, root, threadID, content)

	items, lastTurnCompleted, err := RecoverCodexThread(threadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items != nil {
		t.Fatalf("items = %+v, want nil", items)
	}
	if lastTurnCompleted {
		t.Fatalf("lastTurnCompleted = true, want false")
	}
}

func TestRecoverCodexThreadMissingThreadYieldsNothing(t *testing.T) {
	withCodexRolloutRoot(t) // empty temp dir, no rollout files at all

	items, lastTurnCompleted, err := RecoverCodexThread("no-such-thread-ever-written")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if items != nil {
		t.Fatalf("items = %+v, want nil", items)
	}
	if lastTurnCompleted {
		t.Fatalf("lastTurnCompleted = true, want false")
	}
}

func TestRecoverCodexThreadEmptyIDReturnsError(t *testing.T) {
	withCodexRolloutRoot(t)

	items, lastTurnCompleted, err := RecoverCodexThread("")
	if err == nil {
		t.Fatalf("expected an error for an empty thread id, got nil (items=%+v completed=%v)", items, lastTurnCompleted)
	}
	if items != nil || lastTurnCompleted {
		t.Fatalf("expected zero-value results alongside the error, got items=%+v completed=%v", items, lastTurnCompleted)
	}
}

func TestRecoverCodexThreadSurfacesLastAgentMessage(t *testing.T) {
	root := withCodexRolloutRoot(t)
	threadID := "01a0865a-995b-7191-be8e-96a355d741fc"
	content := `{"timestamp":"2026-09-13T00-00-00.000Z","ordinal":0,"type":"event_msg","payload":{"type":"task_started","turn_id":"t","started_at":1}}
{"timestamp":"2026-09-13T00-00-01.000Z","ordinal":1,"type":"event_msg","payload":{"type":"task_complete","turn_id":"t","last_agent_message":"the final word, recovered","started_at":1,"completed_at":9,"duration_ms":8000,"time_to_first_token_ms":500}}
`
	writeRollout(t, root, threadID, content)

	items, _, err := RecoverCodexThread(threadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, it := range items {
		if it.Kind == "message" && it.Text == "the final word, recovered" {
			found = true
		}
	}
	if !found {
		t.Fatalf("last_agent_message not surfaced, got %+v", items)
	}
}

// TestRecoverCodexThreadNormalizesObservedItemKinds checks the Kind mapping
// for every item.type actually seen in real rollout files on this machine,
// plus one deliberately unrecognized type to prove the "other" fallback
// never drops the item or panics.
func TestRecoverCodexThreadNormalizesObservedItemKinds(t *testing.T) {
	root := withCodexRolloutRoot(t)
	threadID := "01a09273-a6d0-7c52-90ab-5b09eb7a8a98"
	content := `{"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"t","item":{"type":"AgentMessage","id":"m1","content":[{"type":"Text","text":"hello"}]},"completed_at_ms":1}}
{"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"t","item":{"type":"Reasoning","id":"r1","summary_text":["thinking"]},"completed_at_ms":2}}
{"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"t","item":{"type":"CommandExecution","id":"c1","command":["ls"],"stdout":"file.txt\n"},"completed_at_ms":3}}
{"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"t","item":{"type":"McpToolCall","id":"mc1","server":"cua_repl","tool":"js","result":{"content":[{"type":"text","text":"tool result text"}]}},"completed_at_ms":4}}
{"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"t","item":{"type":"FileChange","id":"fc1","changes":{"a.go":{"type":"add"}}},"completed_at_ms":5}}
{"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"t","item":{"type":"SubAgentActivity","id":"sa1","kind":"interacted","agent_path":"/root"},"completed_at_ms":6}}
{"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"t","item":{"type":"ContextCompaction","id":"cc1"},"completed_at_ms":7}}
{"type":"event_msg","payload":{"type":"item_completed","thread_id":"` + threadID + `","turn_id":"t","item":{"type":"SomeBrandNewItemKindFromAFutureCLI","id":"new1","content":[{"type":"Text","text":"future text"}]},"completed_at_ms":8}}
`
	writeRollout(t, root, threadID, content)

	items, _, err := RecoverCodexThread(threadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantKinds := []string{"message", "reasoning", "tool", "tool", "tool", "other", "other", "other"}
	if len(items) != len(wantKinds) {
		t.Fatalf("got %d items, want %d: %+v", len(items), len(wantKinds), items)
	}
	for i, want := range wantKinds {
		if items[i].Kind != want {
			t.Errorf("items[%d] (completed_at=%d) Kind = %q, want %q", i, items[i].CompletedAt, items[i].Kind, want)
		}
	}
	if items[2].Text != "file.txt\n" {
		t.Errorf("CommandExecution text = %q, want stdout", items[2].Text)
	}
	if items[3].Text != "tool result text" {
		t.Errorf("McpToolCall text = %q, want result content text", items[3].Text)
	}
	// The unrecognized future item type still recovers whatever text it
	// can find via the best-effort scrape, and is never dropped.
	if items[7].Text != "future text" {
		t.Errorf("unknown item type text = %q, want best-effort scrape of content[].text", items[7].Text)
	}
}

// TestRecoverCodexThreadPicksMostRecentlyModifiedFile exercises the
// documented multi-file tie-break: a thread with more than one on-disk
// rollout file (observed for real on this machine, e.g. a session later
// forked/continued) resolves to the most recently modified one.
func TestRecoverCodexThreadPicksMostRecentlyModifiedFile(t *testing.T) {
	root := withCodexRolloutRoot(t)
	threadID := "01a08cae-8710-79c1-bfd5-e52704f7bba5"
	dir := filepath.Join(root, "2026", "09", "13")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	older := filepath.Join(dir, "rollout-2026-09-13T00-27-23-"+threadID+".jsonl")
	newer := filepath.Join(dir, "rollout-2026-09-13T08-59-45-"+threadID+"_01a08e83-9cda-75c1-b327-aada4c92c382.jsonl")

	oldContent := `{"type":"event_msg","payload":{"type":"task_started","turn_id":"old","started_at":1}}
{"type":"event_msg","payload":{"type":"task_complete","turn_id":"old","last_agent_message":"stale answer","started_at":1,"completed_at":2}}
`
	newContent := `{"type":"event_msg","payload":{"type":"task_started","turn_id":"new","started_at":10}}
{"type":"event_msg","payload":{"type":"task_complete","turn_id":"new","last_agent_message":"fresh answer","started_at":10,"completed_at":11}}
`
	if err := os.WriteFile(older, []byte(oldContent), 0o644); err != nil {
		t.Fatalf("WriteFile older: %v", err)
	}
	// Ensure a real, detectable mtime gap regardless of filesystem
	// timestamp resolution.
	oldTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(older, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes older: %v", err)
	}
	if err := os.WriteFile(newer, []byte(newContent), 0o644); err != nil {
		t.Fatalf("WriteFile newer: %v", err)
	}

	items, _, err := RecoverCodexThread(threadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, it := range items {
		if it.Text == "stale answer" {
			t.Fatalf("recovered from the OLDER file; want the most recently modified sibling, got %+v", items)
		}
		if it.Text == "fresh answer" {
			found = true
		}
	}
	if !found {
		t.Fatalf("did not recover the newer file's content, got %+v", items)
	}
}
