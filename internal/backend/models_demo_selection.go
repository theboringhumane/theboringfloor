package backend

import (
	"context"
	"errors"
	"fmt"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func (b *demoBackend) demoModelReady(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.fl.isStopped() {
		return errors.New("demo backend stopped")
	}
	return nil
}

func (b *demoBackend) ListModelAgents(ctx context.Context) ([]state.ModelAgentInfo, error) {
	if err := b.demoModelReady(ctx); err != nil {
		return nil, err
	}
	return []state.ModelAgentInfo{
		{Name: "explore", Description: "Explore the demo project"},
		{Name: "general", Description: "Handle a demo task"},
	}, nil
}

func (b *demoBackend) SetModel(ctx context.Context, target state.ModelTarget, ref string) error {
	if err := b.demoModelReady(ctx); err != nil {
		return err
	}
	if ref != "" {
		found := false
		for _, row := range DemoModels() {
			if row.SelectionRef() == ref {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown demo model %q", ref)
		}
	}
	if target.Agent != "" {
		agents, err := b.ListModelAgents(ctx)
		if err != nil {
			return err
		}
		found := false
		for _, agent := range agents {
			if agent.Name == target.Agent {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown demo agent %q", target.Agent)
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	b.modelSelections[target.Agent] = ref
	return nil
}
