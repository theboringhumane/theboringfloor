package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestHydrateSessionStallsInterruptedTasks(t *testing.T) {
	scratchHome(t)
	m := New(&pinBackend{}, nil)
	saved := &SessionFile{Tasks: []state.BoardTask{
		{ID: "in-progress-1", Status: state.TaskInProgress},
		{ID: "pending", Status: state.TaskPending},
		{ID: "done", Status: state.TaskDone},
		{ID: "in-progress-2", Status: state.TaskInProgress},
	}}

	if got := m.hydrateSession(saved); got != 2 {
		t.Fatalf("hydrateSession stalled %d tasks, want 2", got)
	}
	for _, tt := range []struct {
		id     string
		status state.TaskStatus
	}{
		{id: "in-progress-1", status: state.TaskStalled},
		{id: "pending", status: state.TaskPending},
		{id: "done", status: state.TaskDone},
		{id: "in-progress-2", status: state.TaskStalled},
	} {
		for _, task := range m.st.Tasks {
			if task.ID == tt.id && task.Status == tt.status {
				goto found
			}
		}
		t.Fatalf("task %q status was not %q: %+v", tt.id, tt.status, m.st.Tasks)
	found:
	}
}

func TestRestoreNoticeWithoutStalledTasksIsUnchanged(t *testing.T) {
	saved := &SessionFile{
		SavedAt: time.Date(2026, time.September, 7, 14, 5, 0, 0, time.Local).UnixMilli(),
		Chat:    []state.ChatMsg{{ID: "one"}, {ID: "two"}},
	}
	want := fmt.Sprintf("restored office session from %s (%d msgs) · /new for a fresh office · /session prints the id (-s|--session pins one at boot)",
		time.UnixMilli(saved.SavedAt).Local().Format("15:04"), len(saved.Chat))
	if got := RestoreNotice(saved, 0); got != want {
		t.Fatalf("zero-stalled RestoreNotice changed:\n got %q\nwant %q", got, want)
	}
	if got, want := RestoreNotice(saved, 3), "restored office session from 14:05 (2 msgs) · 3 tasks left unfinished · /new for a fresh office · /session prints the id (-s|--session pins one at boot)"; got != want {
		t.Fatalf("stalled RestoreNotice = %q, want %q", got, want)
	}
}

func TestHydrateSessionStalledTasksRoundTrip(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	m1 := New(&pinBackend{}, nil)
	saved := &SessionFile{Dir: dir, Tasks: []state.BoardTask{
		{ID: "interrupted", Status: state.TaskInProgress},
		{ID: "complete", Status: state.TaskDone},
	}}
	if got := m1.hydrateSession(saved); got != 1 {
		t.Fatalf("first hydrate stalled %d tasks, want 1", got)
	}
	if err := SaveSession(dir, Snapshot(dir, "primary", m1.st)); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	persisted, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: no session found after save")
	}
	if persisted.Tasks[0].Status != state.TaskStalled {
		t.Fatalf("persisted interrupted task status = %q, want %q", persisted.Tasks[0].Status, state.TaskStalled)
	}

	m2 := New(&pinBackend{}, nil)
	if got := m2.hydrateSession(persisted); got != 0 {
		t.Fatalf("second hydrate stalled %d tasks, want 0", got)
	}
	if m2.st.Tasks[0].Status != state.TaskStalled {
		t.Fatalf("second hydrate changed stalled task to %q, want %q", m2.st.Tasks[0].Status, state.TaskStalled)
	}
}
