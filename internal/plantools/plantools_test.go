package plantools

import (
	"math/rand"
	"strings"
	"testing"
)

func TestExtractStrictMultilineGrammar(t *testing.T) {
	for _, tc := range []struct {
		name        string
		in, cleaned string
		kind        Kind
		text        string
		ok          bool
	}{
		{"present", "before\n⟦plan-present⟧\n# Plan\n\n- first\n⟦/plan-present⟧\nafter", "before\nafter", Present, "# Plan\n\n- first", true},
		{"update", "⟦plan-update⟧\nrevised\n⟦/plan-update⟧", "", Update, "revised", true},
		{"get", "before\n ⟦plan-get-approved⟧ \nafter", "before\nafter", GetApproved, "", true},
		{"embedded get stays", "ask ⟦plan-get-approved⟧ now", "ask ⟦plan-get-approved⟧ now", "", "", false},
		{"empty block stays", "⟦plan-present⟧\n \n⟦/plan-present⟧", "⟦plan-present⟧\n \n⟦/plan-present⟧", "", "", false},
		{"unclosed block stays", "⟦plan-update⟧\nmissing close", "⟦plan-update⟧\nmissing close", "", "", false},
		{"wrong close stays", "⟦plan-present⟧\nbody\n⟦/plan-update⟧", "⟦plan-present⟧\nbody\n⟦/plan-update⟧", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleaned, got, ok := Extract(tc.in)
			if cleaned != tc.cleaned || got.Kind != tc.kind || got.Text != tc.text || ok != tc.ok {
				t.Fatalf("Extract(%q) = (%q, %+v, %v), want (%q, {%q %q}, %v)", tc.in, cleaned, got, ok, tc.cleaned, tc.kind, tc.text, tc.ok)
			}
		})
	}
}

func TestExtractTreatsValidBlockBodiesAsOpaque(t *testing.T) {
	for _, tc := range []struct {
		name        string
		in, cleaned string
		kind        Kind
		text        string
	}{
		{
			name:    "get inside present",
			in:      "before\n⟦plan-present⟧\n# Plan\n⟦plan-get-approved⟧\n⟦/plan-present⟧\nafter",
			cleaned: "before\nafter",
			kind:    Present,
			text:    "# Plan\n⟦plan-get-approved⟧",
		},
		{
			name:    "get inside update",
			in:      "⟦plan-update⟧\nrevised\n⟦plan-get-approved⟧\n⟦/plan-update⟧",
			cleaned: "",
			kind:    Update,
			text:    "revised\n⟦plan-get-approved⟧",
		},
		{
			name:    "present plus outside get",
			in:      "⟦plan-present⟧\nfirst\n⟦/plan-present⟧\n⟦plan-get-approved⟧",
			cleaned: "",
			kind:    Present,
			text:    "first",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleaned, got, ok := Extract(tc.in)
			if !ok || cleaned != tc.cleaned || got.Kind != tc.kind || got.Text != tc.text {
				t.Fatalf("Extract(%q) = (%q, %+v, %v), want (%q, {%q %q}, true)", tc.in, cleaned, got, ok, tc.cleaned, tc.kind, tc.text)
			}
		})
	}
}

func TestExtractOverlappingAndMalformedMarkersNeverPanic(t *testing.T) {
	for _, tc := range []struct {
		name        string
		in, cleaned string
		kind        Kind
		text        string
		ok          bool
	}{
		{
			name:    "nested block keeps outer directive",
			in:      "⟦plan-present⟧\nouter\n⟦plan-update⟧\ninner\n⟦/plan-update⟧\n⟦/plan-present⟧",
			cleaned: "",
			kind:    Present,
			text:    "outer\n⟦plan-update⟧\ninner\n⟦/plan-update⟧",
			ok:      true,
		},
		{
			name:    "wrong present closer remains while get is extracted",
			in:      "⟦plan-present⟧\nbody\n⟦/plan-update⟧\n⟦plan-get-approved⟧",
			cleaned: "⟦plan-present⟧\nbody\n⟦/plan-update⟧",
			kind:    GetApproved,
			ok:      true,
		},
		{
			name:    "wrong update closer remains while get is extracted",
			in:      "⟦plan-update⟧\nbody\n⟦/plan-present⟧\n⟦plan-get-approved⟧",
			cleaned: "⟦plan-update⟧\nbody\n⟦/plan-present⟧",
			kind:    GetApproved,
			ok:      true,
		},
		{
			name:    "orphan closers remain around extracted get",
			in:      "⟦/plan-present⟧\n⟦plan-get-approved⟧\n⟦/plan-update⟧",
			cleaned: "⟦/plan-present⟧\n⟦/plan-update⟧",
			kind:    GetApproved,
			ok:      true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleaned, got, ok := Extract(tc.in)
			if cleaned != tc.cleaned || got.Kind != tc.kind || got.Text != tc.text || ok != tc.ok {
				t.Fatalf("Extract(%q) = (%q, %+v, %v), want (%q, {%q %q}, %v)", tc.in, cleaned, got, ok, tc.cleaned, tc.kind, tc.text, tc.ok)
			}
		})
	}
}

func TestExtractRandomMarkerStringsNeverPanic(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	parts := []string{"plain", "\n", "⟦plan-present⟧", "⟦/plan-present⟧", "⟦plan-update⟧", "⟦/plan-update⟧", "⟦plan-get-approved⟧", " ", "\r\n"}
	for i := 0; i < 1000; i++ {
		var in strings.Builder
		for j := 0; j < 1+r.Intn(20); j++ {
			in.WriteString(parts[r.Intn(len(parts))])
		}
		Extract(in.String())
	}
}

func FuzzExtractNoPanic(f *testing.F) {
	for _, seed := range []string{
		"⟦plan-present⟧\nbody\n⟦plan-get-approved⟧\n⟦/plan-present⟧",
		"⟦plan-update⟧\nbody\n⟦/plan-present⟧",
		"⟦plan-get-approved⟧",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		Extract(text)
	})
}

func TestExtractCapsHeadAndEmitsOnlyFirstDirective(t *testing.T) {
	long := strings.Repeat("α", MaxRunes+100)
	in := "⟦plan-update⟧\n" + long + "\n⟦/plan-update⟧\nanswer\n⟦plan-get-approved⟧"
	cleaned, got, ok := Extract(in)
	if !ok || got.Kind != Update || cleaned != "answer" {
		t.Fatalf("Extract long/multiple = (%q, %+v, %v)", cleaned, got, ok)
	}
	if len([]rune(got.Text)) != MaxRunes || !strings.HasPrefix(got.Text, strings.Repeat("α", 10)) || !strings.HasSuffix(got.Text, TruncationMarker) {
		t.Fatalf("capped plan = %d runes, suffix %q", len([]rune(got.Text)), got.Text[len(got.Text)-len(TruncationMarker):])
	}
}

func TestScrubFallbackAndPreamble(t *testing.T) {
	var emitted []Directive
	if got := Scrub("⟦plan-get-approved⟧", func(d Directive) { emitted = append(emitted, d) }); got != PlanApprovalStatusRequested {
		t.Fatalf("get-only fallback = %q", got)
	}
	if len(emitted) != 1 || emitted[0].Kind != GetApproved {
		t.Fatalf("emitted directives = %+v", emitted)
	}
	emitted = nil
	if got := Scrub("⟦plan-present⟧\nfirst\n⟦/plan-present⟧\n⟦plan-get-approved⟧", func(d Directive) { emitted = append(emitted, d) }); got != "" {
		t.Fatalf("multiple directive scrub = %q", got)
	}
	if len(emitted) != 1 || emitted[0].Kind != Present || emitted[0].Text != "first" {
		t.Fatalf("multiple directives must emit only the first, got %+v", emitted)
	}
	for _, want := range []string{"[theboringfloor harness — plan tools]", "ctrl+x twice", "⟦plan-present⟧", "⟦plan-update⟧", "⟦plan-get-approved⟧", "never requests approval"} {
		if !strings.Contains(PromptPreamble, want) {
			t.Fatalf("PromptPreamble missing %q:\n%s", want, PromptPreamble)
		}
	}
}
