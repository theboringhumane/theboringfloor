package chatcontext

import (
	"strings"
	"testing"
)

func TestExtractStrictGrammarAndClamp(t *testing.T) {
	for _, tc := range []struct {
		name        string
		in, cleaned string
		count       int
		ok          bool
	}{
		{"default", "before\n⟦recent-messages⟧\nafter", "before\nafter", 20, true},
		{"count", "⟦recent-messages: 3⟧", "", 3, true},
		{"low clamp", "⟦recent-messages: 0⟧", "", 1, true},
		{"high clamp", "⟦recent-messages: 99⟧", "", 50, true},
		{"overflow clamps", "⟦recent-messages: 999999999999999999999999⟧", "", 50, true},
		{"embedded stays", "ask ⟦recent-messages⟧ please", "ask ⟦recent-messages⟧ please", 0, false},
		{"negative stays", "⟦recent-messages: -1⟧", "⟦recent-messages: -1⟧", 0, false},
		{"word stays", "⟦recent-messages: lots⟧", "⟦recent-messages: lots⟧", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleaned, count, ok := Extract(tc.in)
			if cleaned != tc.cleaned || count != tc.count || ok != tc.ok {
				t.Fatalf("Extract(%q) = (%q, %d, %v), want (%q, %d, %v)", tc.in, cleaned, count, ok, tc.cleaned, tc.count, tc.ok)
			}
		})
	}
}

func TestExtractCapsOneRequestAndStripsAllValidMarkers(t *testing.T) {
	cleaned, count, ok := Extract("⟦recent-messages: 2⟧\nbody\n⟦recent-messages: 40⟧")
	if !ok || count != 2 || cleaned != "body" {
		t.Fatalf("Extract repeated markers = (%q, %d, %v)", cleaned, count, ok)
	}
}

func TestScrubFallbackAndPreamble(t *testing.T) {
	var emitted []int
	if got := Scrub("⟦recent-messages⟧", func(n int) { emitted = append(emitted, n) }); got != "[theboringfloor] recent messages requested: 20" {
		t.Fatalf("marker-only fallback = %q", got)
	}
	if len(emitted) != 1 || emitted[0] != 20 {
		t.Fatalf("emitted counts = %v, want [20]", emitted)
	}
	for _, want := range []string{"⟦recent-messages⟧", "⟦recent-messages: N⟧", "sparingly", "20", "50", "synthetic follow-up message"} {
		if !strings.Contains(PromptPreamble, want) {
			t.Fatalf("PromptPreamble missing %q:\n%s", want, PromptPreamble)
		}
	}
}

func TestIsControlText(t *testing.T) {
	for _, text := range []string{
		"[theboringfloor] recent messages requested: 20",
		"[theboringfloor] recent chat context (last 20 messages, oldest first)\nuser: hello",
		"[theboringfloor] no recent chat context available",
	} {
		if !IsControlText(text) {
			t.Fatalf("IsControlText(%q) = false, want true", text)
		}
	}
	for _, text := range []string{
		"[theboringfloor] unrelated notice",
		"boss: [theboringfloor] recent messages requested: 20",
		"normal answer",
	} {
		if IsControlText(text) {
			t.Fatalf("IsControlText(%q) = true, want false", text)
		}
	}
}
