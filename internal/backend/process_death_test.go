package backend

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func terminalBossEvent(t *testing.T, evs []state.Event, id, want string) {
	t.Helper()
	for _, e := range evs {
		if e.Kind == state.EvChatBoss && e.Msg.ID == id {
			if e.Msg.Pending || !strings.Contains(e.Msg.Text, want) {
				t.Fatalf("terminal boss event %q = %+v; want Pending:false containing %q", id, e.Msg, want)
			}
			return
		}
	}
	t.Fatalf("missing terminal boss event %q in %#v", id, evs)
}

func eventLogSnapshot(l *eventLog) []state.Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]state.Event(nil), l.evs...)
}

func resolvedDialogEvent(t *testing.T, evs []state.Event, kind state.EventKind, id string) {
	t.Helper()
	for _, e := range evs {
		if e.Kind != kind {
			continue
		}
		got := e.PermissionID
		if kind == state.EvQuestion {
			got = e.QuestionID
		}
		if got == id && e.ToolState == "resolved" {
			return
		}
	}
	t.Fatalf("missing resolved dialog %v/%q in %#v", kind, id, evs)
}

func TestServeDeathSettlesZeroTextTurnAndDialogs(t *testing.T) {
	log := &eventLog{}
	b := newLiveBackend("", t.TempDir(), config.Default())
	b.fl.setEmit(log.emit)
	proc := &exec.Cmd{}
	b.mu.Lock()
	b.proc = proc
	b.pendingBoss = []string{"boss-zero"}
	b.ctx.textAccum["stream-zero"] = ""
	b.ctx.textStart["stream-zero"] = 1
	b.ctx.pendingPerms["perm-1"] = permHold{SessionID: "ses-1", EmployeeID: "boss", Title: "Bash"}
	b.ctx.pendingQuestions["question-1"] = permHold{SessionID: "ses-1", EmployeeID: "boss"}
	b.mu.Unlock()

	b.handleServeExit(proc, errors.New("exit status 1"))

	b.mu.Lock()
	pending := append([]string(nil), b.pendingBoss...)
	perms, questions := len(b.ctx.pendingPerms), len(b.ctx.pendingQuestions)
	b.mu.Unlock()
	if len(pending) != 0 || perms != 0 || questions != 0 {
		t.Fatalf("serve death left pending state: boss=%v perms=%d questions=%d", pending, perms, questions)
	}
	evs := eventLogSnapshot(log)
	terminalBossEvent(t, evs, "boss-zero", "opencode process exited")
	terminalBossEvent(t, evs, "bossmsg-stream-zero", "opencode process exited")
	resolvedDialogEvent(t, evs, state.EvPermission, "perm-1")
	resolvedDialogEvent(t, evs, state.EvQuestion, "question-1")
}

func TestClaudeDeathSettlesZeroTextTurnAndDialogs(t *testing.T) {
	log := &claudeEventLog{}
	b := newClaudeBackend("claude", t.TempDir(), config.Default())
	b.fl.setEmit(log.emit)
	proc := &exec.Cmd{}
	wait := make(chan struct{})
	exitCh := make(chan error, 1)
	b.mu.Lock()
	b.proc = proc
	b.procErr = newCappedErrBuf()
	b.procWait = wait
	b.busyTurns = 1
	b.pendingBoss = []string{"boss-zero"}
	b.ctx.textAccum["stream-zero"] = ""
	b.ctx.textStart["stream-zero"] = 1
	b.ctx.pendingPerms["perm-1"] = permHold{SessionID: "ses-1", EmployeeID: "boss"}
	b.ctx.permMeta["perm-1"] = claudePermMeta{}
	b.ctx.pendingQuestions["question-1"] = permHold{SessionID: "ses-1", EmployeeID: "boss"}
	b.ctx.dialogMeta["question-1"] = claudeDialogMeta{}
	b.questionStash["question-1"] = []state.QuestionItem{{Question: "continue?"}}
	b.mu.Unlock()
	exitCh <- errors.New("exit status 1")

	b.watchProc(proc, exitCh, wait)

	b.mu.Lock()
	busy, pending := b.busyTurns, append([]string(nil), b.pendingBoss...)
	perms, questions := len(b.ctx.pendingPerms), len(b.ctx.pendingQuestions)
	permMeta, dialogMeta, stash := len(b.ctx.permMeta), len(b.ctx.dialogMeta), len(b.questionStash)
	b.mu.Unlock()
	if busy != 0 || len(pending) != 0 || perms != 0 || questions != 0 || permMeta != 0 || dialogMeta != 0 || stash != 0 {
		t.Fatalf("claude death left pending state: busy=%d boss=%v perms=%d questions=%d permMeta=%d dialogMeta=%d stash=%d", busy, pending, perms, questions, permMeta, dialogMeta, stash)
	}
	evs := log.snapshot()
	terminalBossEvent(t, evs, "boss-zero", "claude process exited")
	terminalBossEvent(t, evs, "bossmsg-stream-zero", "claude process exited")
	resolvedDialogEvent(t, evs, state.EvPermission, "perm-1")
	resolvedDialogEvent(t, evs, state.EvQuestion, "question-1")
}
