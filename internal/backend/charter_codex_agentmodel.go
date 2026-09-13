package backend

import (
	"context"
	"errors"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// Native role config files set model after spawn-model selection. They cannot
// represent defaults which preserve explicit per-dispatch model precedence.
// See https://developers.openai.com/codex/multi-agent/ (custom agents).
func codexAgentModelUnsupported() error {
	return errors.New("Codex per-agent model defaults are unavailable: native role config models override explicit spawn models; configure Codex roles directly or use inherited models")
}

func (*codexBackend) ListModelAgents(ctx context.Context) ([]state.ModelAgentInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, codexAgentModelUnsupported()
}
