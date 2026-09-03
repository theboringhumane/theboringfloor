// claude_usage_test.go — the usage-delta arm: result frames carry RUNNING
// totals (`usage` + `modelUsage`), the mapper emits only the growth over
// the last result for the same session (keyed by session id, deduped per
// result uuid), and an all-zero delta ships nothing.
package backend

import (
	"strconv"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func claudeResultLineSess(sess, turnUUID string, in, out, cacheR, cacheW int64, cost float64) string {
	return `{"type":"result","subtype":"success","is_error":false,"session_id":"` + sess + `","uuid":"` + turnUUID + `",` +
		`"total_cost_usd":` + jsonFloat(cost) + `,` +
		`"usage":{"input_tokens":` + itoa64(in) + `,"output_tokens":` + itoa64(out) +
		`,"cache_read_input_tokens":` + itoa64(cacheR) + `,"cache_creation_input_tokens":` + itoa64(cacheW) + `}}`
}

func jsonFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func TestClaudeUsageDeltasRunningTotal(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-u"

	// turn 1: first result frame -> its full counters are the growth
	evs := claudeFeed(t, ctx, claudeResultLineSess("sess-u", "res-1", 1000, 200, 4000, 900, 0.0042), 100)
	if len(evs) != 1 || evs[0].Kind != state.EvUsage {
		t.Fatalf("turn 1 must emit one EvUsage, got %v", claudeKinds(evs))
	}
	u := evs[0]
	if u.TokensIn != 1000 || u.TokensOut != 200 || u.TokensCacheRead != 4000 || u.TokensCacheWrite != 900 || !floatAlmost(u.CostUSD, 0.0042) {
		t.Fatalf("turn 1 delta drifted: %+v", u)
	}

	// turn 2: running totals grew -> the DELTA is only the growth
	evs = claudeFeed(t, ctx, claudeResultLineSess("sess-u", "res-2", 2400, 450, 8500, 1300, 0.0110), 200)
	if len(evs) != 1 {
		t.Fatalf("turn 2 must emit one EvUsage")
	}
	u = evs[0]
	if u.TokensIn != 1400 || u.TokensOut != 250 || u.TokensCacheRead != 4500 || u.TokensCacheWrite != 400 || !floatAlmost(u.CostUSD, 0.0068) {
		t.Fatalf("turn 2 delta must be growth-only: %+v (want 1400/250/4500/400/0.0068)", u)
	}

	// a replayed frame (uuid dedupe) emits nothing
	if evs := claudeFeed(t, ctx, claudeResultLineSess("sess-u", "res-2", 2400, 450, 8500, 1300, 0.0110), 300); len(evs) != 0 {
		t.Fatalf("replay res-2 emitted %v — the uuid dedupe must swallow it", claudeKinds(evs))
	}

	// turn 3 with identical totals: zero-delta suppressed
	if evs := claudeFeed(t, ctx, claudeResultLineSess("sess-u", "res-3", 2400, 450, 8500, 1300, 0.0110), 400); len(evs) != 0 {
		t.Fatalf("zero-delta result emitted %v — must be suppressed", claudeKinds(evs))
	}

	// a DIFFERENT session tracks its own baseline
	evs = claudeFeed(t, ctx, `{"type":"result","subtype":"success","session_id":"sess-other","uuid":"res-x","total_cost_usd":0.1,"usage":{"input_tokens":10,"output_tokens":5}}`, 500)
	if len(evs) != 1 || evs[0].TokensIn != 10 || evs[0].TokensOut != 5 {
		t.Fatalf("a second session keys its own running total: %+v", evs)
	}
}

func TestClaudeUsageModelUsageFallback(t *testing.T) {
	ctx := newClaudeNormCtx(nil)
	ctx.primaryID = "sess-m"
	// usage blob empty/absent-ish: the modelUsage roll-up supplies counters
	evs := claudeFeed(t, ctx, `{"type":"result","subtype":"success","session_id":"sess-m","uuid":"res-m1","total_cost_usd":0.02,`+
		`"modelUsage":{"claude-a":{"inputTokens":1000,"outputTokens":120,"cacheReadInputTokens":500,"cacheCreationInputTokens":30},`+
		`"claude-b":{"inputTokens":400,"outputTokens":60}}}`, 100)
	if len(evs) != 1 {
		t.Fatalf("modelUsage fallback must emit one EvUsage, got %v", claudeKinds(evs))
	}
	u := evs[0]
	if u.TokensIn != 1400 || u.TokensOut != 180 || u.TokensCacheRead != 500 || u.TokensCacheWrite != 30 || !floatAlmost(u.CostUSD, 0.02) {
		t.Fatalf("modelUsage roll-up drifted: %+v", u)
	}
	// growth on the same baseline: next frame's usage blob wins outright
	evs = claudeFeed(t, ctx, claudeResultLineSess("sess-m", "res-m2", 2000, 300, 900, 60, 0.04), 200)
	if len(evs) != 1 || evs[0].TokensIn != 600 || evs[0].TokensOut != 120 || !floatAlmost(evs[0].CostUSD, 0.02) {
		t.Fatalf("mixed-blob delta drifted: %+v", evs)
	}
}

func floatAlmost(a, b float64) bool {
	d := a - b
	return d > -1e-9 && d < 1e-9
}
