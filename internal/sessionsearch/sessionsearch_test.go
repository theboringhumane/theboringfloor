package sessionsearch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestLoadHappyAndPlansAndInfo(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	fixture := sessionFixture{Dir: dir, PrimaryID: "boss-1", Backend: "opencode", PrimaryIDs: map[string]string{"opencode": "boss-1"}, PlanText: "draft", ApprovedPlanText: "approved", SavedAt: 42, Chat: []state.ChatMsg{{ID: "one", Text: "hello"}}}
	writeSession(t, home, ".theboringfloor/projects", dir, fixture)

	loaded, ok := Load(dir)
	if !ok || loaded.PrimaryID != "boss-1" || loaded.Backend != "opencode" {
		t.Fatalf("Load() = %#v, %v", loaded, ok)
	}
	if got, ok := ApprovedPlan(dir); !ok || got != "approved" {
		t.Fatalf("ApprovedPlan() = %q, %v", got, ok)
	}
	if got, ok := Draft(dir); !ok || got != "draft" {
		t.Fatalf("Draft() = %q, %v", got, ok)
	}
	if got, ok := Info(dir); !ok || got != (Meta{Dir: dir, Backend: "opencode", PrimaryID: "boss-1", ChatCount: 1, SavedAt: 42}) {
		t.Fatalf("Info() = %#v, %v", got, ok)
	}
}

func TestLoadMissingCorruptAndNullCollections(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	if got, ok := Load(dir); got != nil || ok {
		t.Fatalf("missing Load() = %#v, %v", got, ok)
	}
	path := sessionPath(home, ".theboringfloor/projects", dir)
	writeFile(t, path, "{")
	if got, ok := Load(dir); got != nil || ok {
		t.Fatalf("corrupt Load() = %#v, %v", got, ok)
	}
	writeFile(t, path, `{"dir":"`+dir+`","chat":null,"tasks":null,"mails":null,"agents":null}`)
	loaded, ok := Load(dir)
	if !ok || loaded.Chat != nil || loaded.Tasks != nil || loaded.Mails != nil || loaded.Agents != nil {
		t.Fatalf("null collections Load() = %#v, %v", loaded, ok)
	}
}

func TestLoadCanonicalRootOnly(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	writeSession(t, home, ".theboringfloor/projects", dir, sessionFixture{Dir: dir, PrimaryID: "canonical"})
	loaded, ok := Load(dir)
	if !ok || loaded.PrimaryID != "canonical" {
		t.Fatalf("Load() = %#v, %v", loaded, ok)
	}
}

func TestLoadReadRoots(t *testing.T) {
	cases := []struct {
		name      string
		canonical bool
		fallback  bool
		want      string
	}{
		{name: "canonical only", canonical: true, want: "canonical"},
		{name: "fallback only", fallback: true, want: "fallback"},
		{name: "both present", canonical: true, fallback: true, want: "canonical"},
		{name: "neither present"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			home := fakeHome(t)
			dir := filepath.Join(home, "work")
			if tt.canonical {
				writeSession(t, home, ".theboringfloor/projects", dir, sessionFixture{Dir: dir, PrimaryID: "canonical"})
			}
			if tt.fallback {
				writeSession(t, home, ".theboringfloor/sessions", dir, sessionFixture{Dir: dir, PrimaryID: "fallback"})
			}
			loaded, ok := Load(dir)
			if tt.want == "" {
				if ok || loaded != nil {
					t.Fatalf("Load() = %#v, %v; want no session", loaded, ok)
				}
				return
			}
			if !ok || loaded.PrimaryID != tt.want {
				t.Fatalf("Load() = %#v, %v; want primary %q", loaded, ok, tt.want)
			}
		})
	}
}

func TestLoadCorruptCanonicalDoesNotFallBack(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	writeSession(t, home, ".theboringfloor/sessions", dir, sessionFixture{Dir: dir, PrimaryID: "fallback"})
	writeFile(t, sessionPath(home, ".theboringfloor/projects", dir), "{")
	if got, ok := Load(dir); ok || got != nil {
		t.Fatalf("corrupt canonical Load() = %#v, %v; must not use fallback", got, ok)
	}
}

func TestTranscriptTailLimitCapAndPending(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	chat := make([]state.ChatMsg, 0, 510)
	for i := 0; i < 510; i++ {
		chat = append(chat, state.ChatMsg{ID: string(rune(i + 1)), Text: "message", At: int64(i)})
	}
	chat[509].Pending = true
	writeSession(t, home, ".theboringfloor/projects", dir, sessionFixture{Dir: dir, Chat: chat})
	if got, ok := Transcript(dir, 2); !ok || len(got) != 2 || got[0].At != 507 || got[1].At != 508 {
		t.Fatalf("Transcript(2) = %#v, %v", got, ok)
	}
	if got, ok := Transcript(dir, 999); !ok || len(got) != 500 || got[0].At != 9 || got[len(got)-1].At != 508 {
		t.Fatalf("Transcript(cap) first=%v last=%v len=%d ok=%v", got[0].At, got[len(got)-1].At, len(got), ok)
	}
	if got, ok := Transcript(dir, 0); !ok || len(got) != 50 || got[0].At != 459 || got[len(got)-1].At != 508 {
		t.Fatalf("Transcript(default) first=%v last=%v len=%d ok=%v", got[0].At, got[len(got)-1].At, len(got), ok)
	}
}

func TestSearchCaseInsensitiveRankingAndEmptyQuery(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	writeSession(t, home, ".theboringfloor/projects", dir, sessionFixture{Dir: dir, Chat: []state.ChatMsg{
		{ID: "one", From: "boss", Text: "Alpha alpha alpha", At: 1},
		{ID: "two", From: "boss", Text: "alpha alpha", At: 20},
		{ID: "three", From: "ALPHA-worker", Text: "unrelated", At: 30},
		{ID: "pending", From: "boss", Text: "alpha alpha alpha alpha", At: 99, Pending: true},
	}})
	hits, ok := Search(dir, " ALpHa ", 20)
	if !ok || len(hits) != 3 || hits[0].Message.ID != "one" || hits[1].Message.ID != "two" || hits[2].Message.ID != "three" {
		t.Fatalf("Search() ids = %#v, %v", hitIDs(hits), ok)
	}
	if hits[2].Snippet != "ALPHA-worker" {
		t.Fatalf("from-match snippet = %q", hits[2].Snippet)
	}
	if got, ok := Search(dir, " \t\n ", 20); !ok || len(got) != 0 {
		t.Fatalf("empty Search() = %#v, %v", got, ok)
	}
}

func TestSearchCJKSnippetIsRuneSafeAndClipped(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	text := strings.Repeat("界", 300) + " needle " + strings.Repeat("語", 300)
	writeSession(t, home, ".theboringfloor/projects", dir, sessionFixture{Dir: dir, Chat: []state.ChatMsg{{ID: "cjk", Text: text}}})
	hits, ok := Search(dir, "needle", 1)
	if !ok || len(hits) != 1 {
		t.Fatalf("Search() = %#v, %v", hits, ok)
	}
	if !utf8.ValidString(hits[0].Snippet) || utf8.RuneCountInString(hits[0].Snippet) > 240 || !strings.Contains(hits[0].Snippet, "needle") || !strings.HasPrefix(hits[0].Snippet, "…") || !strings.HasSuffix(hits[0].Snippet, "…") {
		t.Fatalf("bad CJK snippet %q", hits[0].Snippet)
	}
}

func TestSearchDefaultAndCapLimits(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	chat := make([]state.ChatMsg, 205)
	for i := range chat {
		chat[i] = state.ChatMsg{ID: string(rune(i + 1)), Text: "match", At: int64(i)}
	}
	writeSession(t, home, ".theboringfloor/projects", dir, sessionFixture{Dir: dir, Chat: chat})
	if got, ok := Search(dir, "match", 0); !ok || len(got) != 20 || got[0].Message.At != 204 || got[len(got)-1].Message.At != 185 {
		t.Fatalf("Search(default) first=%v last=%v len=%d ok=%v", got[0].Message.At, got[len(got)-1].Message.At, len(got), ok)
	}
	if got, ok := Search(dir, "match", 999); !ok || len(got) != 200 || got[0].Message.At != 204 || got[len(got)-1].Message.At != 5 {
		t.Fatalf("Search(cap) first=%v last=%v len=%d ok=%v", got[0].Message.At, got[len(got)-1].Message.At, len(got), ok)
	}
}

func TestLoadNeverReadsTemporarySessionFile(t *testing.T) {
	home := fakeHome(t)
	dir := filepath.Join(home, "work")
	dirPath := filepath.Dir(sessionPath(home, ".theboringfloor/projects", dir))
	writeFile(t, filepath.Join(dirPath, ".session-123.tmp"), `{"dir":"`+dir+`","primaryID":"tmp"}`)
	if got, ok := Load(dir); got != nil || ok {
		t.Fatalf("Load() read temporary file: %#v, %v", got, ok)
	}
}

type sessionFixture struct {
	Dir              string            `json:"dir"`
	PrimaryID        string            `json:"primaryID"`
	Backend          string            `json:"backend,omitempty"`
	PrimaryIDs       map[string]string `json:"primaryIDs,omitempty"`
	Chat             []state.ChatMsg   `json:"chat"`
	PlanText         string            `json:"planText,omitempty"`
	ApprovedPlanText string            `json:"approvedPlanText,omitempty"`
	SavedAt          int64             `json:"savedAt"`
}

func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("THEFLOOR_HOME", home)
	return home
}

func sessionPath(home, root, dir string) string {
	return filepath.Join(home, root, dirHash(dir), "session.json")
}

func writeSession(t *testing.T, home, root, dir string, fixture sessionFixture) {
	t.Helper()
	data, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, sessionPath(home, root, dir), string(data))
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hitIDs(hits []Hit) []string {
	ids := make([]string, len(hits))
	for i, hit := range hits {
		ids[i] = hit.Message.ID
	}
	return ids
}
