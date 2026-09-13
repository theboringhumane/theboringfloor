package backend

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestCodexAgentDefaultsDoNotPretendSpawnPrecedence(t *testing.T) {
	b := NewCodex("unused", t.TempDir(), nil).(*codexBackend)
	for _, role := range []string{"explorer", "worker", "office-alias"} {
		err := b.SetModel(context.Background(), state.ModelTarget{Agent: role}, "native-model")
		if err == nil || !strings.Contains(err.Error(), "override explicit spawn models") {
			t.Fatalf("role %q: %v", role, err)
		}
	}
	if len(b.agentModels) != 0 {
		t.Fatal("unsupported role assignment changed runtime preferences")
	}
	if agents, err := b.ListModelAgents(context.Background()); err == nil || len(agents) != 0 {
		t.Fatalf("unsupported role defaults advertised: %v %v", agents, err)
	}
}

// A stale per-agent entry can only ever reach ModelPreferences some way
// other than the app's own SetModel (which always refuses target.Agent != ""
// before SetModelPreference is called — see
// TestCodexAgentDefaultsDoNotPretendSpawnPrecedence above). Simulate that
// (a hand-edited brain.json, or a future regression) directly via
// cfg.SetModelPreference and prove the self-heal: construction drops the
// stale entry from in-memory state, Start notifies the member exactly once,
// and every subsequent send proceeds normally on the boss model — a send
// must never fail solely because of a stale per-agent entry on disk.
func TestCodexStaleAgentModelSelfHealsNotifiesOnceAndSendsSucceed(t *testing.T) {
	fake, log := codexMetadataFixture(t, `cat >/dev/null
printf '%s\n' '{"type":"thread.started","thread_id":"healed-thread"}' '{"type":"turn.completed"}'
`)
	cfg := &config.Config{}
	cfg.SetModelPreference("codex", "explorer", "native-role-model")
	b := NewCodex(fake.bin, fake.dir, cfg).(*codexBackend)
	if len(b.agentModels) != 0 {
		t.Fatalf("stale per-agent entry was not dropped at construction: %v", b.agentModels)
	}
	var evs []state.Event
	if err := b.Start(func(e state.Event) { evs = append(evs, e) }); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	notices := func() (n int) {
		for _, e := range evs {
			if e.Kind == state.EvStatus && strings.Contains(e.Text, "per-agent model preference") {
				n++
			}
		}
		return
	}
	if n := notices(); n != 1 {
		t.Fatalf("expected exactly one stale-agent-model notice from Start, got %d: %v", n, evs)
	}
	if err := b.Send("must not fail solely because of the stale role entry"); err != nil {
		t.Fatalf("send must succeed on the boss model despite the stale entry: %v", err)
	}
	if _, err := os.Stat(log + ".args"); err != nil {
		t.Fatalf("send should have started inference: %v", err)
	}
	if err := b.Send("second turn must also succeed"); err != nil {
		t.Fatalf("second send must also succeed: %v", err)
	}
	if n := notices(); n != 1 {
		t.Fatalf("notice must not repeat per send, got %d", n)
	}
}
