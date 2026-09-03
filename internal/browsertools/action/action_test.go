// action_test.go — the MUTATING engine's contracts WITHOUT live chrome:
//
//	validation — unknown ops and empty selectors/expressions fail
//	     BEFORE any browser work;
//	policy — a Decide refusal (file:, plain http with the flag off)
//	     comes back as a headless.PolicyError verbatim and never
//	     launches a process (the gate fires before discovery);
//	CapEvalJSON — undefined/empty reads "null", short JSON passes
//	     through, an over-cap payload cuts rune-safe with the "…" tail;
//	classify — the three failure families (navigation / selector /
//	     eval) phrase deadline vs canceled vs generic errors as their
//	     DISTINCT actionable strings.
//
// The REAL chrome runs live in live_test.go (THEBORINGOFFICE_LIVE_CHROME=1).
package action

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/theboringhumane/theboringfloor/internal/browsertools"
	"github.com/theboringhumane/theboringfloor/internal/headless"
)

func TestNavigateAndActValidation(t *testing.T) {
	cases := []struct {
		name string
		a    Action
		want string
	}{
		{"unknown op", Action{Op: "frobnicate", Sel: "#x"}, `unknown op "frobnicate"`},
		{"click empty selector", Action{Op: OpClick, Sel: "  "}, "click needs a non-empty selector"},
		{"fill empty selector", Action{Op: OpFill, Sel: "", Arg: "v"}, "fill needs a non-empty selector"},
		{"eval empty expression", Action{Op: OpEval, Arg: "  "}, "eval needs a non-empty expression"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := NavigateAndAct(context.Background(), "https://theboring.name", c.a)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("NavigateAndAct(%+v) error = %v, want substring %q", c.a, err, c.want)
			}
		})
	}
}

func TestNavigateAndActPolicyRefusesBeforeLaunch(t *testing.T) {
	// the policy gate fires BEFORE chrome discovery: these run on ANY
	// machine, browser or not, and must come back as PolicyError with
	// the Decide reason verbatim.
	t.Setenv(browsertools.AllowHTTPEnv, "")
	for _, rawurl := range []string{"file:///etc/passwd", "http://theboring.name", "javascript:alert(1)"} {
		_, err := NavigateAndAct(context.Background(), rawurl, Action{Op: OpClick, Sel: "#buy"})
		var pe *headless.PolicyError
		if !errors.As(err, &pe) {
			t.Fatalf("%s: want a PolicyError, got %T: %v", rawurl, err, err)
		}
		want := browsertools.Decide(rawurl, func(string) string { return "" }).Reason
		if pe.Reason != want {
			t.Fatalf("%s: reason = %q, want the Decide verbatim %q", rawurl, pe.Reason, want)
		}
	}
}

func TestCapEvalJSON(t *testing.T) {
	if got := CapEvalJSON(nil); got != "null" {
		t.Fatalf("undefined/empty eval → \"null\", got %q", got)
	}
	if got := CapEvalJSON([]byte("  \n")); got != "null" {
		t.Fatalf("whitespace-only eval → \"null\", got %q", got)
	}
	short := `"hello"`
	if got := CapEvalJSON([]byte(short)); got != short {
		t.Fatalf("short JSON passes through verbatim, got %q", got)
	}
	// over-cap with multi-byte runes astride the cut: never splits one,
	// always carries the truncation tail.
	huge := `"` + strings.Repeat("é", EvalCapBytes) + `"`
	got := CapEvalJSON([]byte(huge))
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("a truncated payload carries the … tail, got tail %q", got[len(got)-8:])
	}
	if !utf8.ValidString(got) {
		t.Fatal("the cut must never split a multi-byte rune")
	}
	if len(got) > EvalCapBytes+len("…") {
		t.Fatalf("the cap is %d bytes (+tail), got %d", EvalCapBytes, len(got))
	}
}

func TestClassifyFamilies(t *testing.T) {
	deadline := fmt.Errorf("wrap: %w", context.DeadlineExceeded)
	canceled := fmt.Errorf("wrap: %w", context.Canceled)
	generic := errors.New("boom")

	// navigation phase
	if err := classifyNav("https://x", deadline); !strings.Contains(err.Error(), "timed out loading after 20s") ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("nav deadline class: %v", err)
	}
	if err := classifyNav("https://x", generic); !strings.Contains(err.Error(), "navigation failed") {
		t.Fatalf("nav generic class: %v", err)
	}
	// selector phase: the deadline NAMES the selector (click waits on
	// visibility, fill on presence) — never confused with a slow page.
	if err := classifySel("https://x", OpClick, "#buy", deadline); !strings.Contains(err.Error(),
		`selector "#buy" did not match a visible node within the 20s budget`) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("click deadline class: %v", err)
	}
	if err := classifySel("https://x", OpFill, "#name", deadline); !strings.Contains(err.Error(),
		`selector "#name" did not match a node within the 20s budget`) {
		t.Fatalf("fill deadline class: %v", err)
	}
	if err := classifySel("https://x", OpClick, "#buy", generic); !strings.Contains(err.Error(),
		`click on selector "#buy" failed: boom`) {
		t.Fatalf("click generic class: %v", err)
	}
	// eval phase: a JS exception is its own class.
	if err := classifyEval("https://x", generic); !strings.Contains(err.Error(), "eval failed: boom") {
		t.Fatalf("eval generic class: %v", err)
	}
	if err := classifyEval("https://x", deadline); !strings.Contains(err.Error(), "eval timed out after 20s") {
		t.Fatalf("eval deadline class: %v", err)
	}
	// canceled stays its own word in every family.
	for _, err := range []error{
		classifyNav("https://x", canceled), classifySel("https://x", OpClick, "#b", canceled), classifyEval("https://x", canceled),
	} {
		if !strings.Contains(err.Error(), "canceled") {
			t.Fatalf("canceled must read canceled: %v", err)
		}
	}
}
