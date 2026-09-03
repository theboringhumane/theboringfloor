// Package plantools implements the boss-only plan presentation marker shared
// by the OpenCode and Claude backends.
package plantools

import (
	"regexp"
	"sort"
	"strings"
)

const (
	// MaxRunes bounds persisted plan text while preserving its beginning.
	MaxRunes = 20000
	// TruncationMarker is appended when a plan exceeds MaxRunes.
	TruncationMarker = "\n\n… [plan truncated]"
	// PlanApprovalStatusRequested is the settled bubble for a marker-only
	// approval-status request.
	PlanApprovalStatusRequested = "[theboringfloor] plan approval status requested"
)

// Kind identifies one plan directive.
type Kind string

const (
	Present     Kind = "present"
	Update      Kind = "update"
	GetApproved Kind = "get-approved"
)

// Directive is the backend-neutral payload extracted from one boss reply.
// Text is populated for Present and Update only.
type Directive struct {
	Kind Kind
	Text string
}

// PromptPreamble rides the first boss prompt after the browser and
// recent-message harness preambles.
const PromptPreamble = "[theboringoffice harness — plan tools]\n" +
	"The member reviews and approves plan execution in the plan pane with ctrl+x twice. " +
	"Presenting or updating a plan is presentation only, never execution. Emit at most one directive per reply:\n" +
	"⟦plan-present⟧\n<markdown>\n⟦/plan-present⟧ — present a nonempty plan.\n" +
	"⟦plan-update⟧\n<markdown>\n⟦/plan-update⟧ — update a nonempty plan.\n" +
	"⟦plan-get-approved⟧ — on its own line, read whether the current plan is approved; it never requests approval.\n" +
	"A marker inside a valid plan-present or plan-update block is plan content, not a directive.\n" +
	"The office strips valid directives from your visible reply."

var (
	presentBlockRe = regexp.MustCompile(`(?ms)^[ \t]*⟦plan-present⟧[ \t]*\r?\n(.*?)\r?\n[ \t]*⟦/plan-present⟧[ \t]*(?:\r?\n|$)`)
	updateBlockRe  = regexp.MustCompile(`(?ms)^[ \t]*⟦plan-update⟧[ \t]*\r?\n(.*?)\r?\n[ \t]*⟦/plan-update⟧[ \t]*(?:\r?\n|$)`)
	getRe          = regexp.MustCompile(`(?m)^[ \t]*⟦plan-get-approved⟧[ \t]*(?:\r?\n|$)`)
	blankRe        = regexp.MustCompile(`\n{3,}`)
)

type hit struct {
	start, end int
	directive  Directive
}

// Extract removes valid plan directives and returns the first directive in
// reply order. A malformed, unclosed, or empty presentation block remains
// visible and emits nothing. Valid presentation blocks are opaque: directives
// inside one are plan content, not independently extracted markers. Additional
// valid, non-overlapping directives are scrubbed but do not produce additional
// events, preserving the one-directive-per-reply rule.
func Extract(text string) (string, Directive, bool) {
	blocks := make([]hit, 0)
	appendBlocks(&blocks, text, presentBlockRe, Present)
	appendBlocks(&blocks, text, updateBlockRe, Update)
	blocks = normalize(blocks)

	hits := append([]hit(nil), blocks...)
	for _, m := range getRe.FindAllStringIndex(text, -1) {
		if !insideAny(m[0], m[1], blocks) {
			hits = append(hits, hit{start: m[0], end: m[1], directive: Directive{Kind: GetApproved}})
		}
	}
	if len(hits) == 0 {
		return text, Directive{}, false
	}
	hits = normalize(hits)

	var cleaned strings.Builder
	last := 0
	for _, h := range hits {
		cleaned.WriteString(text[last:h.start])
		last = h.end
	}
	cleaned.WriteString(text[last:])
	return strings.TrimSpace(blankRe.ReplaceAllString(cleaned.String(), "\n\n")), hits[0].directive, true
}

// normalize keeps the earliest directive among overlapping spans. This makes
// extraction safe even if future directive grammars produce nested matches.
func normalize(hits []hit) []hit {
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].start == hits[j].start {
			return hits[i].end > hits[j].end
		}
		return hits[i].start < hits[j].start
	})
	normalized := hits[:0]
	for _, h := range hits {
		if len(normalized) > 0 && h.start < normalized[len(normalized)-1].end {
			continue
		}
		normalized = append(normalized, h)
	}
	return normalized
}

func insideAny(start, end int, blocks []hit) bool {
	for _, block := range blocks {
		if start >= block.start && end <= block.end {
			return true
		}
	}
	return false
}

func appendBlocks(hits *[]hit, text string, re *regexp.Regexp, kind Kind) {
	for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
		body := strings.TrimSpace(text[m[2]:m[3]])
		if body != "" {
			*hits = append(*hits, hit{start: m[0], end: m[1], directive: Directive{Kind: kind, Text: capText(body)}})
		}
	}
}

// Scrub extracts and emits at most one plan directive before returning the
// cleaned transcript. A marker-only approval-status query gets a settled,
// compact bubble; empty plan blocks are invalid and remain visible.
func Scrub(text string, emit func(Directive)) string {
	cleaned, directive, ok := Extract(text)
	if !ok {
		return text
	}
	if emit != nil {
		emit(directive)
	}
	if cleaned != "" {
		return cleaned
	}
	if directive.Kind == GetApproved {
		return PlanApprovalStatusRequested
	}
	return cleaned
}

func capText(text string) string {
	runes := []rune(text)
	if len(runes) <= MaxRunes {
		return text
	}
	marker := []rune(TruncationMarker)
	return string(runes[:MaxRunes-len(marker)]) + TruncationMarker
}
