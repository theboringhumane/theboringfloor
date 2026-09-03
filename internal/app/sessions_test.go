package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// scratchHome points THEBORINGOFFICE_HOME at a t.TempDir for the sessions tests.
func scratchHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("THEBORINGOFFICE_HOME", home)
	return home
}

func TestSessionRoundTrip(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	st := state.OfficeState{
		Employees: []state.Employee{
			{ID: "dev-1", Name: "tekton-1", Role: state.RoleDeveloper, Seat: "desk-1", Sprite: state.SpriteAtDesk},
		},
		Tasks: []state.BoardTask{{ID: "t1", Title: "wire the SSE stream", Status: state.TaskDone, At: 1}},
		Mails: []state.MailItem{{ID: "m1", From: "tekton-1", To: "boss", At: 2, Subject: "return", Kind: state.MailReturn}},
		Chat: []state.ChatMsg{
			{ID: "u1", From: "user", Kind: "user", Text: "say pineapple", At: 3},
			{ID: "b1", From: "boss", Kind: "boss", Text: "Pineapple.", At: 4},
		},
	}
	sf := Snapshot(dir, "ses-123", st)
	if err := SaveSession(dir, sf); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession: no session found after save")
	}
	if !got.Fresh() {
		t.Fatal("freshly written session reports stale")
	}
	if got.Dir != dir || got.PrimaryID != "ses-123" {
		t.Fatalf("round trip mismatch: dir=%q primary=%q", got.Dir, got.PrimaryID)
	}
	if len(got.Chat) != 2 || len(got.Tasks) != 1 || len(got.Mails) != 1 || len(got.Agents) != 1 {
		t.Fatalf("round trip surface counts: chat=%d tasks=%d mails=%d agents=%d",
			len(got.Chat), len(got.Tasks), len(got.Mails), len(got.Agents))
	}
}

// TestSessionPathProjectsLayout pins the canonical on-disk shape after the
// projects/<hash> migration:
// <home>/.theboringoffice/projects/<dirhash>/session.json.
func TestSessionPathProjectsLayout(t *testing.T) {
	home := scratchHome(t)
	dir := t.TempDir()
	want := filepath.Join(home, ".theboringfloor", "projects", SessionDirHash(dir), "session.json")
	if got := SessionPath(dir); got != want {
		t.Fatalf("SessionPath must live under .theboringfloor/projects/<hash>: got %q, want %q", got, want)
	}
}

// TestSaveSessionWritesProjectsOnly pins the write side of the migration:
// SaveSession creates projects/<hash>/session.json and NEVER a file under
// the old ~/.theboringoffice/sessions or ~/.grafeio/sessions roots.
func TestSaveSessionWritesProjectsOnly(t *testing.T) {
	home := scratchHome(t)
	dir := t.TempDir()
	if err := SaveSession(dir, Snapshot(dir, "ses-x", state.OfficeState{})); err != nil {
		t.Fatal(err)
	}
	hash := SessionDirHash(dir)
	want := filepath.Join(home, ".theboringfloor", "projects", hash, "session.json")
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("session.json must land at the projects path: %v", err)
	}
	for _, old := range []string{
		filepath.Join(home, ".theboringoffice", "sessions", hash, "session.json"),
		filepath.Join(home, ".grafeio", "sessions", hash, "session.json"),
	} {
		if _, err := os.Stat(old); !os.IsNotExist(err) {
			t.Fatalf("no file may be written at the old path %q (stat err=%v)", old, err)
		}
	}
}

// TestLoadSession_SessionsDirFallback pins the migration-era read contract:
// an office session stored under the pre-migration ~/.theboringoffice/sessions
// root is still found when projects/<hash> has no file (that root is read
// only), and in the fallback chain it outranks the pre-rename ~/.grafeio
// root: projects > .theboringoffice/sessions > .grafeio/sessions.
func TestLoadSession_SessionsDirFallback(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	st := state.OfficeState{
		Chat: []state.ChatMsg{{ID: "b1", From: "boss", Kind: "boss", Text: "old layout office", At: 4}},
	}
	sf := Snapshot(dir, "ses-oldlayout", st)

	plant := func(base, id string) string {
		path := filepath.Join(base, SessionDirHash(dir), "session.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		s := sf
		s.PrimaryID = id
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	// Leg 1: only the pre-migration same-home copy exists → found.
	old := plant(legacySessionsBase(), "ses-oldlayout")
	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession must read the pre-migration ~/.theboringoffice/sessions copy when projects/ has none")
	}
	if got.PrimaryID != "ses-oldlayout" {
		t.Fatalf("old-layout restore mismatch: primary=%q", got.PrimaryID)
	}

	// Leg 2: the grafeio-era copy ALSO exists → the same-home old layout
	// still wins (fallback chain order: projects > sessions > .grafeio).
	grafeio := plant(grafeioSessionsBase(), "ses-grafeio")
	got, ok = LoadSession(dir)
	if !ok || got.PrimaryID != "ses-oldlayout" {
		t.Fatalf("the .theboringoffice/sessions copy must outrank the .grafeio one: %+v ok=%v", got, ok)
	}

	// Leg 3: a save lands ONLY under the projects root, wins the next load,
	// and leaves BOTH old files byte-untouched.
	sf.PrimaryID = "ses-new"
	if err := SaveSession(dir, sf); err != nil {
		t.Fatal(err)
	}
	got, ok = LoadSession(dir)
	if !ok || got.PrimaryID != "ses-new" {
		t.Fatalf("after save the projects root must win: %+v ok=%v", got, ok)
	}
	for path, id := range map[string]string{old: "ses-oldlayout", grafeio: "ses-grafeio"} {
		if b, _ := os.ReadFile(path); !strings.Contains(string(b), id) {
			t.Fatalf("old-layout file %q must stay untouched, got %s", path, b)
		}
	}
}

// TestLoadSession_LegacyFallback pins the rename-era read contract: an
// office session stored under the pre-rename ~/.grafeio/sessions root is
// still found when BOTH newer roots have none (the grafeio root is read
// only), so an upgrade relaunches into the old transcript instead of
// silently starting over.
func TestLoadSession_LegacyFallback(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	st := state.OfficeState{
		Chat: []state.ChatMsg{{ID: "b1", From: "boss", Kind: "boss", Text: "old office", At: 4}},
	}
	sf := Snapshot(dir, "ses-legacy", st)

	// Plant the file at the PRE-RENAME path (SaveSession writes the new one).
	legacy := filepath.Join(grafeioSessionsBase(), SessionDirHash(dir), "session.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(sf)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, b, 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("LoadSession must read the legacy ~/.grafeio copy when both newer roots have none")
	}
	if got.PrimaryID != "ses-legacy" {
		t.Fatalf("legacy restore mismatch: primary=%q", got.PrimaryID)
	}

	// A save still lands ONLY under the new root.
	sf.PrimaryID = "ses-new"
	if err := SaveSession(dir, sf); err != nil {
		t.Fatal(err)
	}
	got, ok = LoadSession(dir)
	if !ok || got.PrimaryID != "ses-new" {
		t.Fatalf("after save the new root must win: %+v ok=%v", got, ok)
	}
	if b, _ := os.ReadFile(legacy); !strings.Contains(string(b), "ses-legacy") {
		t.Fatalf("legacy file must stay untouched, got %s", b)
	}
}

func TestSessionMalformedFallsSilent(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	if err := os.MkdirAll(SessionPath(dir)[:len(SessionPath(dir))-len("/session.json")], 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(SessionPath(dir), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if sf, ok := LoadSession(dir); ok || sf != nil {
		t.Fatalf("malformed session must degrade silently (ok=false), got %+v", sf)
	}
	if _, ok := LoadSession(dir + "-missing"); ok {
		t.Fatal("missing session must report ok=false")
	}
}

func TestSessionFreshWindow(t *testing.T) {
	fresh := &SessionFile{Dir: "/x", SavedAt: time.Now().Add(-3 * 24 * time.Hour).UnixMilli()}
	if !fresh.Fresh() {
		t.Fatal("3-day-old session must be fresh")
	}
	stale := &SessionFile{Dir: "/x", SavedAt: time.Now().Add(-5 * 24 * time.Hour).UnixMilli()}
	if stale.Fresh() {
		t.Fatal("5-day-old session must be stale")
	}
}

func TestSnapshotCaps(t *testing.T) {
	st := state.OfficeState{}
	for i := 0; i < 205; i++ {
		st.Chat = append(st.Chat, state.ChatMsg{ID: "c", From: "user", Text: "x"})
	}
	for i := 0; i < 60; i++ {
		st.Tasks = append(st.Tasks, state.BoardTask{ID: "t"})
		st.Mails = append(st.Mails, state.MailItem{ID: "m"})
	}
	sf := Snapshot("/d", "p", st)
	if len(sf.Chat) != sessionChatCap {
		t.Fatalf("chat not trimmed to %d (got %d)", sessionChatCap, len(sf.Chat))
	}
	if len(sf.Tasks) != sessionListCap || len(sf.Mails) != sessionListCap {
		t.Fatalf("tasks/mails not trimmed to %d (got %d/%d)", sessionListCap, len(sf.Tasks), len(sf.Mails))
	}
}

func TestSessionDirHashStable(t *testing.T) {
	a, b := SessionDirHash("/tmp/foo"), SessionDirHash("/tmp/foo")
	if a != b || len(a) != 40 {
		t.Fatalf("hash unstable or not sha1 hex: %q vs %q", a, b)
	}
	if SessionDirHash("/tmp/foo") == SessionDirHash("/tmp/bar") {
		t.Fatal("distinct dirs share a hash")
	}
}

// restoreRows picks the live chat rows carrying the restore-notice
// prefix — the rows the boot-notice dedupe must hold at exactly one.
func restoreRows(chat []state.ChatMsg) []state.ChatMsg {
	var out []state.ChatMsg
	for _, c := range chat {
		if c.From == "office" && strings.HasPrefix(c.Text, restoreNoticePrefix) {
			out = append(out, c)
		}
	}
	return out
}

// bootMetaRows counts rows marked boot-scoped (Meta bootNoticeMeta) —
// they must NEVER reach session.json.
func bootMetaRows(chat []state.ChatMsg) int {
	n := 0
	for _, c := range chat {
		if c.Meta == bootNoticeMeta {
			n++
		}
	}
	return n
}

// TestSessionsSnapshotStripsBootMeta pins the write side of the dedupe:
// the boot-scoped restore line renders in the live chat but is stripped
// by Snapshot — while red error notices and the deliberately persisted
// "closing — relaunching" row (session_picker.go) ride along as before.
func TestSessionsSnapshotStripsBootMeta(t *testing.T) {
	st := state.OfficeState{Chat: []state.ChatMsg{
		{ID: "n1", From: "office", Text: "restored office session from 10:00 (5 msgs) · /new for a fresh office", Meta: bootNoticeMeta, At: 1},
		{ID: "n2", From: "office", Text: "send failed: boom", Meta: "error", At: 2},
		{ID: "n3", From: "office", Text: "closing — relaunching as `theboringoffice -s ses-y`", At: 3},
		{ID: "u1", From: "user", Kind: "user", Text: "hi", At: 4},
	}}
	sf := Snapshot("/d", "p", st)
	if len(sf.Chat) != 3 {
		t.Fatalf("Snapshot must strip ONLY the boot-scoped row, kept %d of 4: %+v", len(sf.Chat), sf.Chat)
	}
	if bootMetaRows(sf.Chat) != 0 {
		t.Fatalf("no Meta=%q row may persist: %+v", bootNoticeMeta, sf.Chat)
	}
	for i, id := range []string{"n2", "n3", "u1"} {
		if sf.Chat[i].ID != id {
			t.Fatalf("kept rows drifted: position %d = %q, want %q", i, sf.Chat[i].ID, id)
		}
	}
}

// TestSessionsBootNoticeDedupe drives the full loop twice over a
// LEGACY-POLLUTED session.json (restore lines persisted as plain office
// notices by pre-dedupe builds, plus a stuck Pending bubble and the
// deliberate closing row): each boot hydrates the file through the
// self-clean, shows exactly ONE restore line live (boot-scoped), and
// persists ZERO — a fixpoint the second round-trip re-proves.
func TestSessionsBootNoticeDedupe(t *testing.T) {
	scratchHome(t)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	polluted := state.OfficeState{Chat: []state.ChatMsg{
		{ID: "u1", From: "user", Kind: "user", Text: "ship the thing", At: 1},
		{ID: "r1", From: "office", Text: "restored office session from 09:12 (88 msgs) · old boot", At: 2},
		{ID: "r2", From: "office", Text: "restored office session from 09:41 (90 msgs) · older boot", At: 3},
		{ID: "p1", From: "boss", Kind: "boss", Pending: true, At: 4},
		{ID: "c1", From: "office", Text: "closing — relaunching as `theboringoffice -s ses-x`", At: 5},
	}}
	if err := SaveSession(cwd, Snapshot(cwd, "ses-x", polluted)); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	// Snapshot PRESERVES the legacy rows (they carry no Meta marker — the
	// strip targets new boot-scoped rows only), so hydrate's prefix
	// self-clean is the leg proven against on-disk pollution.
	if raw, ok := LoadSession(cwd); !ok || len(restoreRows(raw.Chat)) != 2 {
		t.Fatalf("fixture must plant 2 legacy restore rows, ok=%v chat=%+v", ok, raw.Chat)
	}

	// BOOT 1: hydrate self-cleans the legacy lines + the stuck Pending
	// bubble, keeps the transcript + the closing row, then emits exactly
	// ONE boot-scoped restore line.
	m1 := New(&pinBackend{}, nil)
	if live := restoreRows(m1.st.Chat); len(live) != 1 || live[0].Meta != bootNoticeMeta {
		t.Fatalf("boot 1 must show exactly ONE boot-scoped restore line, got %+v", live)
	}
	if !chatHas(m1, "ship the thing") || !chatHas(m1, "closing — relaunching") {
		t.Fatalf("boot 1 must keep the transcript + closing row: %v", chatTexts(m1))
	}
	for _, c := range m1.st.Chat {
		if c.Pending {
			t.Fatalf("a stuck Pending bubble must never hydrate: %+v", c)
		}
	}

	// PERSIST 1: the quit-path snapshot strips the boot-scoped line —
	// ZERO restore rows reach the file; the closing row still rides.
	sf1 := Snapshot(cwd, "ses-x", m1.st)
	if err := SaveSession(cwd, sf1); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	onDisk, ok := LoadSession(cwd)
	if !ok {
		t.Fatal("session.json missing after persist 1")
	}
	if n := len(restoreRows(onDisk.Chat)); n != 0 {
		t.Fatalf("ZERO restore lines must persist after boot 1, got %d: %+v", n, onDisk.Chat)
	}
	if n := bootMetaRows(onDisk.Chat); n != 0 {
		t.Fatalf("ZERO Meta=%q rows must persist after boot 1, got %d", bootNoticeMeta, n)
	}
	if n := 0; true {
		for _, c := range onDisk.Chat {
			if strings.Contains(c.Text, "closing — relaunching") {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("the deliberate closing row must still persist, got %d", n)
		}
	}

	// BOOT 2: the clean file hydrates with nothing to scrub; the new boot
	// announces itself exactly once again.
	m2 := New(&pinBackend{}, nil)
	if live := restoreRows(m2.st.Chat); len(live) != 1 || live[0].Meta != bootNoticeMeta {
		t.Fatalf("boot 2 must show exactly ONE boot-scoped restore line, got %+v", live)
	}

	// PERSIST 2: fixpoint — still zero restore rows in the snapshot.
	sf2 := Snapshot(cwd, "ses-x", m2.st)
	if n := len(restoreRows(sf2.Chat)) + bootMetaRows(sf2.Chat); n != 0 {
		t.Fatalf("round trip 2 must persist ZERO restore rows, got %d", n)
	}
	if !strings.Contains(chatTextsJoin(sf2.Chat), "ship the thing") {
		t.Fatalf("the real transcript must survive both cycles: %+v", sf2.Chat)
	}
}

// chatTextsJoin flattens chat rows for a substring proof.
func chatTextsJoin(chat []state.ChatMsg) string {
	var b strings.Builder
	for _, c := range chat {
		b.WriteString(c.Text)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestSaveLatestWins(t *testing.T) {
	scratchHome(t)
	dir := t.TempDir()
	if err := SaveSession(dir, Snapshot(dir, "ses-old", state.OfficeState{})); err != nil {
		t.Fatal(err)
	}
	first, _ := LoadSession(dir)
	if err := SaveSession(dir, Snapshot(dir, "ses-new", state.OfficeState{})); err != nil {
		t.Fatal(err)
	}
	got, ok := LoadSession(dir)
	if !ok {
		t.Fatal("session disappeared after overwrite")
	}
	if got.PrimaryID != "ses-new" {
		t.Fatalf("latest write did not win (primary=%q)", got.PrimaryID)
	}
	if want := filepath.Join(".theboringfloor", "projects", SessionDirHash(dir), "session.json"); !strings.HasSuffix(SessionPath(dir), want) {
		t.Fatalf("session path shape drifted: %q, want suffix %q", SessionPath(dir), want)
	}
	_ = first
}
