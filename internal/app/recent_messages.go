package app

import (
	"strings"
	"unicode"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringoffice/internal/chatcontext"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

const (
	recentMessagesDefault = 20
	recentMessagesMax     = 50
	recentEntryMaxRunes   = 2000
	recentOutputMaxRunes  = 1000
	recentPayloadMaxBytes = 12000
)

// applyRecentMessages handles the backend's compacted-context request. The
// transcript is formatted before its synthetic follow-up is sent, but the
// transport lookup stays inside recentMessagesCmd so a backend replacement
// between event reduction and command execution receives the follow-up.
func (m *Model) applyRecentMessages(ev state.Event) tea.Cmd {
	if ev.Kind != state.EvRecentMessages {
		return nil
	}
	count := clampRecentMessagesCount(ev.RecentMessagesCount)
	followup, sent := m.buildRecentMessagesFollowup(count)
	return recentMessagesCmd(m.currentBackend, followup, sent)
}

func recentMessagesCmd(current *currentBackend, followup string, sent int) tea.Cmd {
	return func() tea.Msg {
		if err := current.send(followup, nil, ""); err != nil {
			return recentMessagesResult{err: err}
		}
		return recentMessagesResult{sent: sent}
	}
}

type recentMessagesResult struct {
	sent int
	err  error
}

// buildRecentMessagesFollowup produces the exact prompt sent to the backend.
// It intentionally consumes the state transcript, not the rendered panel, so
// folding and terminal dimensions cannot change recovery context.
func (m *Model) buildRecentMessagesFollowup(count int) (string, int) {
	count = clampRecentMessagesCount(count)
	entries := recentTranscriptEntries(m.st.Chat, m.recentToolOutputs)
	if len(entries) == 0 {
		return chatcontext.NoRecentChatContext, 0
	}
	if len(entries) > count {
		entries = entries[len(entries)-count:]
	}
	header := chatcontext.RecentChatContextPrefix + " (last " + itoa(count) + " messages, oldest first)"
	return boundRecentPayload(header, entries), len(entries)
}

func clampRecentMessagesCount(n int) int {
	if n < 1 {
		return recentMessagesDefault
	}
	if n > recentMessagesMax {
		return recentMessagesMax
	}
	return n
}

func recentTranscriptEntries(chat []state.ChatMsg, outputs map[string]string) []string {
	entries := make([]string, 0, len(chat))
	for _, msg := range chat {
		if msg.Pending {
			continue
		}
		text := cleanRecentText(msg.Text)
		if text == "" || chatcontext.IsControlText(text) {
			continue
		}
		switch msg.Kind {
		case "tool", "wtool":
			name, summary := splitRecentTool(text)
			entry := "tool " + name + " (" + recentToolState(msg.Meta) + "): " + clipRecentRunes(summary, recentEntryMaxRunes)
			if output := cleanRecentText(outputs[msg.ID]); output != "" {
				entry += "\n  output: " + clipRecentRunes(output, recentOutputMaxRunes)
			}
			entries = append(entries, entry)
		case "user":
			entries = append(entries, "user: "+clipRecentRunes(text, recentEntryMaxRunes))
		case "boss":
			entries = append(entries, "boss: "+clipRecentRunes(text, recentEntryMaxRunes))
		default:
			if msg.Kind == "" && msg.From == "user" {
				entries = append(entries, "user: "+clipRecentRunes(text, recentEntryMaxRunes))
			} else if msg.Kind == "" && msg.From == "boss" {
				entries = append(entries, "boss: "+clipRecentRunes(text, recentEntryMaxRunes))
			}
		}
	}
	return entries
}

// pruneRecentToolOutputs mirrors the panel's transcript-lifetime cache: tool
// bodies are useful only while their compact rows remain in the capped chat.
func (m *Model) pruneRecentToolOutputs() {
	if len(m.recentToolOutputs) <= len(m.st.Chat)+64 {
		return
	}
	live := make(map[string]struct{}, len(m.st.Chat))
	for _, msg := range m.st.Chat {
		live[msg.ID] = struct{}{}
	}
	for id := range m.recentToolOutputs {
		if _, ok := live[id]; !ok {
			delete(m.recentToolOutputs, id)
		}
	}
}

func splitRecentTool(text string) (string, string) {
	if name, summary, ok := strings.Cut(text, " · "); ok {
		return name, summary
	}
	return text, ""
}

func recentToolState(meta string) string {
	if state, _, ok := strings.Cut(meta, "\x1f"); ok {
		return state
	}
	if meta != "" {
		return meta
	}
	return "running"
}

func cleanRecentText(s string) string {
	s = ansi.Strip(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' || (!unicode.IsControl(r) && r != unicode.ReplacementChar) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func clipRecentRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	var b strings.Builder
	for _, r := range s {
		if max == 1 {
			break
		}
		b.WriteRune(r)
		max--
	}
	return b.String() + "…"
}

func boundRecentPayload(header string, entries []string) string {
	body := strings.Join(entries, "\n")
	if len(header)+1+len(body) <= recentPayloadMaxBytes {
		return header + "\n" + body
	}
	// Drop the oldest complete entries first. The newest context is more useful
	// after compaction, and the leading marker makes that loss explicit.
	for len(entries) > 1 && len(header)+1+len("…\n")+len(strings.Join(entries, "\n")) > recentPayloadMaxBytes {
		entries = entries[1:]
	}
	body = strings.Join(entries, "\n")
	prefix := "…\n"
	room := recentPayloadMaxBytes - len(header) - 1 - len(prefix)
	if len(body) > room {
		body = clipUTF8Head(body, room-3) + "…"
	}
	return header + "\n" + prefix + body
}

func clipUTF8Head(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

func itoa(n int) string {
	// Count is bounded to two digits; avoiding fmt here keeps the formatter's
	// hot path allocation-light.
	if n >= 10 {
		return string(rune('0'+n/10)) + string(rune('0'+n%10))
	}
	return string(rune('0' + n))
}
