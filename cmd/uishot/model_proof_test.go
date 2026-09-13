package main

import (
	"bytes"
	"strings"
	"testing"
)

// Exercises the full key -> slash -> catalog -> apply -> disk-save chain.
// The proof itself asserts each intermediate frame, target and next-send call.
func TestModelProof(t *testing.T) {
	var out bytes.Buffer
	if err := runModelProof(&out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"opencode / native agent types",
		"claudecode / narrow agent card",
		"codex / agent unsupported",
		`SetModel target="explore" ref="openai/gpt-5.4"`,
		`SetModel target="Explore" ref="haiku"`,
		`Send prompt="model proof next send" selected={"":"gpt-5.4-mini"}`,
		"PASS: two byte-identical runs",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing proof checkpoint %q", want)
		}
	}
}
