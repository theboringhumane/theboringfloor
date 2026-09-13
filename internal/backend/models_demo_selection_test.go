package backend

import (
	"context"
	"errors"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"testing"
)

func TestDemoModelSelection(t *testing.T) {
	cfg := config.Default()
	cfg.SetModelPreference("opencode", "", "saved/model")
	b := newDemoBackend(cfg)
	rows, err := b.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	agents, err := b.ListModelAgents(context.Background())
	if err != nil || len(agents) == 0 {
		t.Fatalf("agents=%v err=%v", agents, err)
	}
	for _, target := range []state.ModelTarget{{}, {Agent: agents[0].Name}} {
		if err := b.SetModel(context.Background(), target, rows[0].SelectionRef()); err != nil {
			t.Fatal(err)
		}
		if b.modelSelections[target.Agent] != rows[0].SelectionRef() {
			t.Fatal("selection not recorded")
		}
		if err := b.SetModel(context.Background(), target, ""); err != nil {
			t.Fatal(err)
		}
	}
	if cfg.EffectiveModel("opencode", "") != "saved/model" {
		t.Fatal("demo changed config")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := b.SetModel(ctx, state.ModelTarget{}, rows[0].SelectionRef()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := b.SetModel(context.Background(), state.ModelTarget{}, "unknown/model"); err == nil {
		t.Fatal("unknown model accepted")
	}
	b.fl.stop()
	if _, err := b.ListModels(context.Background()); err == nil {
		t.Fatal("stopped listing accepted")
	}
}
