package app

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestRecentMessagesFormatsLastTwentyAndPostsNotice(t *testing.T) {
	rb := &recBackend{}
	m := New(rb, config.Default())
	for i := 0; i < 21; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{From: "user", Text: "message " + itoa(i), At: int64(i)})
	}
	m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: "tool-c1", Kind: "tool", From: "boss", Text: "bash · go test ./...", Meta: "done"})
	m.recentToolOutputs["tool-c1"] = "all green"

	m = runMsg(t, m, state.Event{Kind: state.EvRecentMessages, RecentMessagesCount: 20})
	if len(rb.sentTexts) != 1 {
		t.Fatalf("one synthetic follow-up = %v", rb.sentTexts)
	}
	want := "[theboringfloor] recent chat context (last 20 messages, oldest first)\n" +
		"user: message 2\n" + strings.Join(recentUserLines(3, 20), "\n") + "\n" +
		"tool bash (done): go test ./...\n  output: all green"
	if rb.sentTexts[0] != want {
		t.Fatalf("recent follow-up:\n got %q\nwant %q", rb.sentTexts[0], want)
	}
	if !lastOfficeNoticeHas(m, "context: sent 20 recent messages to the boss") {
		t.Fatalf("success must leave the dim context notice: %+v", officeRows(m))
	}
}

func recentUserLines(first, last int) []string {
	lines := make([]string, 0, last-first+1)
	for i := first; i <= last; i++ {
		lines = append(lines, "user: message "+itoa(i))
	}
	return lines
}

func TestRecentMessagesDefensivelyClampsAndSkipsPendingAndControls(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m.st.Chat = []state.ChatMsg{
		{From: "user", Text: "ignore me", Pending: true},
		{From: "user", Text: "\x1b[31mred\x1b[0m\x00\nsecond"},
		{From: "boss", Text: "answer\r\x07"},
		{From: "office", Kind: "office", Text: "noise"},
		{From: "user", Text: "[theboringfloor] recent chat context (last 2 messages, oldest first)"},
	}
	got, sent := m.buildRecentMessagesFollowup(999)
	want := "[theboringfloor] recent chat context (last 50 messages, oldest first)\nuser: red\nsecond\nboss: answer"
	if got != want || sent != 2 {
		t.Fatalf("clamped/sanitized context = (%q, %d), want (%q, 2)", got, sent, want)
	}
	if gotCount := clampRecentMessagesCount(0); gotCount != recentMessagesDefault {
		t.Fatalf("zero count must use default %d, got %d", recentMessagesDefault, gotCount)
	}
}

func TestRecentMessagesRecoveryControlRowsNeverReenterTranscript(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	m.st.Chat = []state.ChatMsg{
		{From: "boss", Text: "[theboringfloor] recent messages requested: 20"},
		{From: "user", Text: "please continue"},
		{From: "boss", Text: "normal answer"},
	}

	first, sent := m.buildRecentMessagesFollowup(20)
	want := "[theboringfloor] recent chat context (last 20 messages, oldest first)\n" +
		"user: please continue\n" +
		"boss: normal answer"
	if first != want || sent != 2 {
		t.Fatalf("first recovery = (%q, %d), want (%q, 2)", first, sent, want)
	}
	if strings.Contains(first, "recent messages requested") {
		t.Fatalf("marker-only fallback leaked into recovery: %q", first)
	}

	// A completed synthetic follow-up may be stored like any other boss row.
	// It must not become input to the next recovery and recursively grow.
	m.st.Chat = append(m.st.Chat, state.ChatMsg{From: "boss", Text: first})
	second, sent := m.buildRecentMessagesFollowup(20)
	if second != want || sent != 2 {
		t.Fatalf("repeated recovery = (%q, %d), want (%q, 2)", second, sent, want)
	}
}

func TestRecentMessagesCapsEntriesAndKeepsNewestPayload(t *testing.T) {
	m := New(&recBackend{}, config.Default())
	for i := 0; i < recentMessagesMax; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{From: "user", Text: strings.Repeat("界", recentEntryMaxRunes+100) + " newest-" + itoa(i)})
	}
	got, sent := m.buildRecentMessagesFollowup(recentMessagesMax)
	if sent != recentMessagesMax || len(got) > recentPayloadMaxBytes || !utf8.ValidString(got) {
		t.Fatalf("payload bound: sent=%d bytes=%d valid=%v", sent, len(got), utf8.ValidString(got))
	}
	if !strings.Contains(got, "…\nuser: ") || !strings.HasSuffix(got, "…") {
		t.Fatalf("payload must mark dropped older context and a trimmed tail:\n%q", got)
	}
	entry := clipRecentRunes(strings.Repeat("x", recentEntryMaxRunes+1), recentEntryMaxRunes)
	if utf8.RuneCountInString(entry) != recentEntryMaxRunes || !strings.HasSuffix(entry, "…") {
		t.Fatalf("entry cap must include the truncation marker, got %d runes %q", utf8.RuneCountInString(entry), entry[len(entry)-4:])
	}
	tool := recentTranscriptEntries([]state.ChatMsg{{ID: "tool-c1", Kind: "tool", Text: "bash · check", Meta: "done"}},
		map[string]string{"tool-c1": strings.Repeat("y", recentOutputMaxRunes+1)})
	if len(tool) != 1 || utf8.RuneCountInString(strings.TrimPrefix(tool[0], "tool bash (done): check\n  output: ")) != recentOutputMaxRunes || !strings.HasSuffix(tool[0], "…") {
		t.Fatalf("tool output must cap at %d runes: %q", recentOutputMaxRunes, tool)
	}
}

func TestRecentMessagesNoContextAndCurrentBackendSwap(t *testing.T) {
	old := newCurrentBackendStub("old")
	latest := newCurrentBackendStub("latest")
	m := New(old, config.Default())
	cmd := m.applyRecentMessages(state.Event{Kind: state.EvRecentMessages, RecentMessagesCount: 1})
	m.currentBackend.replace(latest) // replacement happens before tea executes cmd
	m = runMsg(t, m, cmd())
	requireCurrentCalls(t, old)
	requireCurrentCalls(t, latest, "[theboringfloor] no recent chat context available")
	if !lastOfficeNoticeHas(m, "context: sent 0 recent messages to the boss") {
		t.Fatalf("no-context delivery must still be visible: %+v", officeRows(m))
	}
}

type failingRecentBackend struct {
	*recBackend
	err error
}

func (b *failingRecentBackend) SendWith(string, []state.Attachment) error { return b.err }

func TestRecentMessagesSendFailurePostsVisibleError(t *testing.T) {
	fail := &failingRecentBackend{recBackend: &recBackend{}, err: errors.New("wire down")}
	m := New(fail, config.Default())
	m.st.Chat = []state.ChatMsg{{From: "user", Text: "recover this"}}
	m = runMsg(t, m, state.Event{Kind: state.EvRecentMessages, RecentMessagesCount: 1})
	if !lastOfficeNoticeHas(m, "context: could not send recent messages to the boss — wire down") {
		t.Fatalf("failed delivery must post a visible error: %+v", officeRows(m))
	}
}
