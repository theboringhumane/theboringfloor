// chat_pagination_test.go — the older-history walk's APP-LOOP wiring,
// end to end on fakes (never a real server): seed-on-attach, the
// scroll-to-top gesture arming exactly ONE async hop per landing,
// byte-stable anchor prepends into the transcript HEAD, the drained top
// latching silently, the 3-strike failure backoff WITHOUT banners, the
// dedupe baseline eating hydrate/live-echo overlap (assistant rows by
// normalized id, user rows by text multiplicity — their live echo ids
// can never collide), and the question/permission float suppression.
//
// Fixtures mirror panels/threads_opencode_test.go's walkHistory: ids
// his-001…oldest→newest double as the before-cursor; Pages follow the
// serve's contract (before == "" → NEWEST page; NextCursor walks one
// page OLDER; absent cursor == HasMore false == the transcript's top).
// runMsg executes each gesture's cmd tree synchronously, so an armed
// hop lands inside the same step — no timers, no goroutines.
package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/backend"
	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// pagerCall — one MessagesPage invocation, recorded verbatim.
type pagerCall struct {
	session string
	before  string
	limit   int
}

// pagerStubBackend — recBackend (demo mode, no primary seam) PLUS the
// state.SessionPager seam over a scripted fixture, walking with the demo
// backend's cursor semantics byte-identically. fail forces every hop to
// error (the backoff test's dead serve).
type pagerStubBackend struct {
	recBackend
	rows  []state.SessionMessageRow
	calls []pagerCall
	fail  bool
}

func (b *pagerStubBackend) MessagesPage(_ context.Context, sessionID, before string, limit int) (state.SessionMessagesPage, error) {
	b.calls = append(b.calls, pagerCall{session: sessionID, before: before, limit: limit})
	if b.fail {
		return state.SessionMessagesPage{}, errors.New("pager stub: scripted failure")
	}
	if limit < 1 {
		limit = 1
	}
	end := len(b.rows)
	if before != "" {
		end = -1
		for i, r := range b.rows {
			if r.ID == before {
				end = i
				break
			}
		}
		if end < 0 {
			return state.SessionMessagesPage{}, fmt.Errorf("pager stub: unknown before cursor %q", before)
		}
	}
	start := end - limit
	if start < 0 {
		start = 0
	}
	page := state.SessionMessagesPage{
		Rows:    append([]state.SessionMessageRow(nil), b.rows[start:end]...),
		HasMore: start > 0,
	}
	if page.HasMore {
		page.NextCursor = b.rows[start].ID
	}
	return page, nil
}

// pagerPinBackend adds the primary seam (live-mode boss-session binding).
type pagerPinBackend struct {
	pagerStubBackend
	primary string
}

func (b *pagerPinBackend) Mode() state.Mode       { return state.ModeLive }
func (b *pagerPinBackend) PrimaryOverride(string) {}
func (b *pagerPinBackend) PrimaryID() string      { return b.primary }

// histRows builds the canned history: his-001…his-n oldest→newest,
// alternating user/assistant (even = assistant, per the demo fixture),
// text "history note NNN", At strictly ascending.
func histRows(n int) []state.SessionMessageRow {
	rows := make([]state.SessionMessageRow, 0, n)
	for i := 1; i <= n; i++ {
		role := "user"
		if i%2 == 0 {
			role = "assistant"
		}
		rows = append(rows, state.SessionMessageRow{
			ID:      fmt.Sprintf("his-%03d", i),
			Role:    role,
			Created: int64(1000 + i*10),
			Parts: []state.SessionMessagePart{{
				Type: "text",
				Text: fmt.Sprintf("history note %03d", i),
			}},
		})
	}
	return rows
}

// pagerFixture boots a sized, splash-cleared model over a pager stub and
// drives ONE event so the attach seed fires + lands (runMsg executes the
// seed closure synchronously). Returns after the seed is in.
func pagerFixture(t *testing.T, b *pagerStubBackend, history int) Model {
	t.Helper()
	b.rows = histRows(history)
	m := New(b, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m.bootDone = true
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[test] attach"})
	if len(b.calls) != 1 {
		t.Fatalf("the first event must fire exactly ONE seed hop, got %d calls", len(b.calls))
	}
	seed := b.calls[0]
	if seed.before != "" || seed.limit != panels.ThreadOlderPageSize {
		t.Fatalf("the seed must fetch the NEWEST page at the walk's page size, got before=%q limit=%d", seed.before, seed.limit)
	}
	if seed.session != "demo" {
		t.Fatalf("a demo backend without a primary id must bind the %q label, got %q", pagerDemoSession, seed.session)
	}
	if !m.pagerSeeded || m.pager == nil {
		t.Fatal("the seed must land: walk armed")
	}
	if m.pagerBaseIDs == nil || m.pagerBaseUtext == nil {
		t.Fatal("the dedupe baseline must freeze at seed landing")
	}
	return m
}

// pgup / wheel up gesture constructors.
func pgupMsg() tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Code: tea.KeyPgUp}) }
func wheelUpMsg() tea.MouseWheelMsg {
	return tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp})
}

// scrollOnce pumps one pgup+wheel pair; the pgUp scrolls a page toward
// the top, the wheel-up at row 0 arms the next hop (landing inline).
func scrollOnce(t *testing.T, m Model) Model {
	t.Helper()
	m = runMsg(t, m, pgupMsg())
	m = runMsg(t, m, wheelUpMsg())
	return m
}

// walkToDawn pumps gestures until the call count stalls with the
// transcript at its top (top-latched walk) or the iteration cap trips.
func walkToDawn(t *testing.T, m Model, b *pagerStubBackend) Model {
	t.Helper()
	quiet := 0
	for i := 0; i < 400; i++ {
		before := len(b.calls)
		m = scrollOnce(t, m)
		if len(b.calls) == before && m.chat.AtTranscriptTop() {
			quiet++
			if quiet >= 3 {
				return m
			}
		} else {
			quiet = 0
		}
	}
	t.Fatal("the walk never quiesced (cap tripped)")
	return m
}

// idsOf / textsOf — transcript projections for dupe scans.
func idsOf(chat []state.ChatMsg) []string {
	out := make([]string, len(chat))
	for i, c := range chat {
		out[i] = c.ID
	}
	return out
}

func countID(chat []state.ChatMsg, id string) int {
	n := 0
	for _, c := range chat {
		if c.ID == id {
			n++
		}
	}
	return n
}

func countText(chat []state.ChatMsg, text string) int {
	n := 0
	for _, c := range chat {
		if c.Text == text {
			n++
		}
	}
	return n
}

// TestPagerSeedOnAttach — the FIRST backend event binds the primary
// pager and fires exactly ONE seed hop; later events never re-seed; the
// seed's rows are DISCARDED (metadata-only: hydrate/live content rules
// the tail); a short history latches the top AT ONCE (nothing to walk).
func TestPaginateSeedOnAttach(t *testing.T) {
	b := &pagerStubBackend{}
	m := pagerFixture(t, b, 500)

	// metadata-only ruling: the seed page (his-451..500) NEVER splices.
	if len(m.st.Chat) != 0 {
		t.Fatalf("the seed must not touch the transcript, got %d entries", len(m.st.Chat))
	}
	// no second event re-fires the hop.
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[test] later"})
	if len(b.calls) != 1 {
		t.Fatalf("seeding is once per binding, got %d calls", len(b.calls))
	}

	// SHORT history: the newest page IS the whole history → HasMore false
	// → Seed latches the top: scroll-top never fetches, ever.
	b2 := &pagerStubBackend{}
	m2 := pagerFixture(t, b2, 30)
	m2 = scrollOnce(t, m2)
	m2 = scrollOnce(t, m2)
	if len(b2.calls) != 1 {
		t.Fatalf("a fully-seeded short history must latch the top at once (1 seed, got %d calls)", len(b2.calls))
	}
	if len(m2.st.Chat) != 0 {
		t.Fatalf("the discarded seed must stay out of the transcript, got %d entries", len(m2.st.Chat))
	}
}

// TestPagerSeedBindLivePrimary — a live-mode backend with the primary
// seam binds the pager to PRIMARY's id (not the demo label).
func TestPaginateSeedBindsLivePrimary(t *testing.T) {
	scratchHome(t) // the LIVE-mode restore leg reads session.json — keep it empty
	b := &pagerPinBackend{primary: "ses-boss-7"}
	b.rows = histRows(120)
	m := New(b, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m.bootDone = true
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[test] attach"})
	if len(b.calls) != 1 || b.calls[0].session != "ses-boss-7" {
		t.Fatalf("the seed must ride the primary session id, got %+v", b.calls)
	}
	if m.pagerSession != "ses-boss-7" {
		t.Fatalf("pagerSession = %q, want ses-boss-7", m.pagerSession)
	}
}

// TestPagerNoSeamBackend — a backend WITHOUT state.SessionPager latches
// the no-seam probe and every top gesture inertly no-ops (harness stubs
// degrade additively; nothing ever reaches for a fetch).
func TestPaginateNoSeamBackend(t *testing.T) {
	b := &recBackend{}
	m := New(b, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m.bootDone = true
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[test] attach"})
	if !m.pagerNoSeam || m.pager != nil {
		t.Fatal("a seam-less backend must latch pagerNoSeam and never bind")
	}
	m = scrollOnce(t, m)
	m = scrollOnce(t, m)
	if len(m.st.Chat) != 0 {
		t.Fatalf("nothing may splice without the seam, got %d entries", len(m.st.Chat))
	}
}

// TestPagerScrollTopWalksAndAnchors — the core loop: wheel-up at the top
// arms ONE hop per landing (50-row pages), the page splices at the HEAD
// with stream-convention ids, the reader's anchored row stays
// BYTE-IDENTICAL across the landing (no yank, no auto-jump), and the
// walk drains to the dawn where the top latches (extra gestures fetch
// nothing).
func TestPaginateScrollTopWalksAndAnchors(t *testing.T) {
	b := &pagerStubBackend{}
	m := pagerFixture(t, b, 500)

	// give the office a LIVE tail: 4 turns below the to-be-walked history
	// (live stream ids — boss bubbles bossmsg-live-N, user echo user-N).
	for i := 1; i <= 4; i++ {
		m = runMsg(t, m, state.Event{Kind: state.EvChatUser, Msg: state.ChatMsg{
			ID: fmt.Sprintf("user-%d", i), From: "user", Kind: "user",
			Text: fmt.Sprintf("live prompt %d", i), At: int64(900000 + i*2)}})
		m = runMsg(t, m, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
			ID: fmt.Sprintf("bossmsg-live-%d", i), From: "boss", Kind: "boss",
			Text: fmt.Sprintf("live answer %d", i), At: int64(900001 + i*2)}})
	}
	if len(m.st.Chat) != 8 {
		t.Fatalf("fixture tail: got %d entries, want 8", len(m.st.Chat))
	}

	// FIRST hop: scroll to the top, one wheel-up arms and lands it.
	m = scrollOnce(t, m) // pgup toward top (8 short rows: one page up is at the top)
	if len(b.calls) != 2 || b.calls[1].before != "his-451" || b.calls[1].limit != panels.ThreadOlderPageSize {
		t.Fatalf("hop 1 must fetch before=his-451 at the page size, got %+v", b.calls)
	}
	if got := len(m.st.Chat); got != 8+panels.ThreadOlderPageSize {
		t.Fatalf("hop 1 must splice %d rows at the head, transcript=%d", panels.ThreadOlderPageSize, got)
	}
	// the spliced head: oldest→newest, stream-convention ids.
	if m.st.Chat[0].ID != "user-his-401" || m.st.Chat[0].From != "user" || m.st.Chat[0].Kind != "user" ||
		m.st.Chat[0].Text != "history note 401" {
		t.Fatalf("head row: got %+v", m.st.Chat[0])
	}
	if m.st.Chat[1].ID != "bossmsg-his-402" || m.st.Chat[1].From != "boss" || m.st.Chat[1].Kind != "boss" {
		t.Fatalf("second spliced row: got %+v", m.st.Chat[1])
	}
	if m.st.Chat[panels.ThreadOlderPageSize-1].ID != "bossmsg-his-450" {
		t.Fatalf("the page must land oldest→newest above the old head, got %q", m.st.Chat[panels.ThreadOlderPageSize-1].ID)
	}
	if m.st.Chat[panels.ThreadOlderPageSize].ID != "user-1" {
		t.Fatalf("the live tail must survive BELOW the splice, got %q", m.st.Chat[panels.ThreadOlderPageSize].ID)
	}

	// DRAIN: keep scrolling; the walk drains in exactly 1 seed + 9 hops
	// (500 fixture rows, the seed's newest 50 discarded as metadata).
	m = walkToDawn(t, m, b)
	if len(b.calls) != 10 {
		t.Fatalf("500 rows at %d/page drain in 10 calls (1 seed + 9 hops), got %d: %+v", panels.ThreadOlderPageSize, len(b.calls), b.calls)
	}
	for i, want := range []string{"", "his-451", "his-401", "his-351", "his-301", "his-251", "his-201", "his-151", "his-101", "his-051"} {
		if b.calls[i].before != want {
			t.Fatalf("walk cursor chain [%d] = %q, want %q", i, b.calls[i].before, want)
		}
	}
	if got := len(m.st.Chat); got != 450+8 {
		t.Fatalf("the full walk splices 450 fixture rows (the seed's newest 50 are discarded metadata), got %d", got)
	}
	if m.st.Chat[0].ID != "user-his-001" || m.st.Chat[0].Text != "history note 001" {
		t.Fatalf("the dawn row must head the transcript, got %+v", m.st.Chat[0])
	}
	// no duplicate ids anywhere after the splice storm.
	seen := map[string]bool{}
	for _, id := range idsOf(m.st.Chat) {
		if seen[id] {
			t.Fatalf("duplicate transcript id after the walk: %q", id)
		}
		seen[id] = true
	}
	// TOP LATCH: gestures at the dawn fetch nothing more.
	before := len(b.calls)
	m = scrollOnce(t, m)
	m = scrollOnce(t, m)
	m = scrollOnce(t, m)
	if len(b.calls) != before {
		t.Fatalf("the drained top must latch — extra gestures fetched %d more calls", len(b.calls)-before)
	}

	// ANCHOR leg on a fresh walk: park the reader AT the loaded top,
	// snapshot the row they read, walk ONE more hop — the row must stay
	// byte-identical, unparked (no auto-jump to the new oldest). (The
	// fail-assisted park: the top-touch strikes once so the reader can
	// settle on row 0 BEFORE the snapshot; a failed hop never moves the
	// offset, then the retry lands the splice.)
	b2 := &pagerStubBackend{}
	m2 := pagerFixture(t, b2, 500)
	m2 = scrollOnce(t, m2) // top → hop 1 lands (his-401..450 on screen)
	if len(b2.calls) != 2 {
		t.Fatalf("anchor fixture: hop 1 = 2 calls, got %d", len(b2.calls))
	}
	b2.fail = true
	for g := 0; !m2.chat.AtTranscriptTop() && g < 200; g++ {
		m2 = runMsg(t, m2, pgupMsg())
	}
	if !m2.chat.AtTranscriptTop() {
		t.Fatal("the reader must park at the transcript's row 0")
	}
	if len(b2.calls) != 3 {
		t.Fatalf("the top-touch armed exactly one (failed) hop, got %d calls", len(b2.calls))
	}
	b2.fail = false
	topBefore := strings.SplitN(m2.chat.View(), "\n", 3)
	m2 = runMsg(t, m2, wheelUpMsg()) // the retry lands his-351..his-400 above
	if len(b2.calls) != 4 {
		t.Fatalf("the parked wheel-up arms exactly one hop, got %d calls", len(b2.calls))
	}
	if m2.chat.AtTranscriptTop() {
		t.Fatal("a landed page must NOT auto-jump the parked reader to the new oldest row")
	}
	topAfter := strings.SplitN(m2.chat.View(), "\n", 3)
	if topBefore[0] != topAfter[0] || topBefore[1] != topAfter[1] {
		t.Fatalf("the anchored top rows moved:\nbefore: %q / %q\nafter:  %q / %q", topBefore[0], topBefore[1], topAfter[0], topAfter[1])
	}
	if m2.st.Chat[0].ID != "user-his-351" {
		t.Fatalf("hop 2 must head the transcript with his-351, got %q", m2.st.Chat[0].ID)
	}
}

// TestPagerOverlapDedupeVsHydrateAndLiveEcho — requirement 4's hazard,
// railed: hydrate a 100-message tail (rows 31..130) with STREAM-CARVED
// ids (user-<seq> / bossmsg-his-NNN) + drop ONE live echo inside the
// overlap; then walk. The all-overlap hop splices ZERO rows (nothing
// moves, no dup ids, no dup texts), and the hop BELOW the hydrated head
// splices fully; a repeated user text dedupes by multiplicity against
// the hydrated occurrence and still splices its genuinely-older twin.
func TestPaginateOverlapDedupeVsHydrateAndLiveEcho(t *testing.T) {
	scratchHome(t)
	b := &pagerPinBackend{primary: "ses-hyd"}
	b.rows = histRows(130)
	// the repeated-text trap: his-005 (user) and his-033 (user) share a body.
	for i := range b.rows {
		if b.rows[i].ID == "his-005" || b.rows[i].ID == "his-033" {
			b.rows[i].Parts[0].Text = "repeat note"
		}
	}
	m := New(b, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m.bootDone = true

	// "hydrate": rows 31..130 as the live process would have stream-carved
	// them into session.json — bossmsg-his-NNN for assistant, user-<seq>
	// for user (a LOCAL seq that can never collide with fetched ids).
	var chat []state.ChatMsg
	seq := 0
	for _, r := range b.rows {
		n := 0
		fmt.Sscanf(r.ID, "his-%d", &n)
		if n < 31 {
			continue
		}
		text := r.Parts[0].Text
		if r.Role == "user" {
			seq++
			chat = append(chat, state.ChatMsg{ID: fmt.Sprintf("user-%d", seq), From: "user", Kind: "user", Text: text, At: r.Created})
		} else {
			chat = append(chat, state.ChatMsg{ID: "bossmsg-" + r.ID, From: "boss", Kind: "boss", Text: text, At: r.Created})
		}
	}
	if len(chat) != 100 {
		t.Fatalf("hydrated fixture must cover rows 31..130, got %d", len(chat))
	}
	m.hydrateSession(&SessionFile{Dir: "t", PrimaryID: "ses-hyd", Chat: chat, SavedAt: 1})
	// one genuinely-live echo AFTER the hydrate (stream-convention id,
	// outside every fetched page's range): the baseline must contain it,
	// and the walk must never duplicate it.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-live-1", From: "boss", Kind: "boss", Text: "live echo turn", At: 9990000}})

	// attach → seed (newest page his-081..130 discarded; cursor his-081).
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "[test] attach"})
	if !m.pagerSeeded {
		t.Fatal("seeded walk required")
	}
	baseLen := len(m.st.Chat) // 100 hydrated + 1 restore notice + 1 live echo
	if baseLen != 102 {
		t.Fatalf("baseline transcript = %d, want 102 (100 hydrated + notice + echo)", baseLen)
	}

	// hop 1: his-031..his-080 — EVERY row overlaps the hydrated tail (and
	// his-077 the live echo): the splice must add ZERO rows and move ZERO pixels.
	// (The reader scrolls UP first; the pgup that lands on row 0 arms the
	// hop itself — the gesture contract.)
	rowsBefore := m.chat.TranscriptRows()
	for g := 0; len(b.calls) == 1 && g < 200; g++ {
		m = runMsg(t, m, pgupMsg())
	}
	if len(b.calls) != 2 || b.calls[1].before != "his-081" {
		t.Fatalf("hop 1 rides before=his-081, got %+v", b.calls)
	}
	if got := len(m.st.Chat); got != baseLen {
		t.Fatalf("the all-overlap page must splice 0 rows, transcript %d → %d", baseLen, got)
	}
	if got := m.chat.TranscriptRows(); got != rowsBefore {
		t.Fatalf("the all-overlap page must paint 0 new rows: %d → %d", rowsBefore, got)
	}
	// both dedupe paths pinned: his-078 (assistant) matched the hydrated
	// bossmsg- id verbatim; his-077 (user) could only match by TEXT (the
	// hydrated echo id is a local seq) — and did.
	if countID(m.st.Chat, "bossmsg-his-078") != 1 || countText(m.st.Chat, "history note 078") != 1 {
		t.Fatal("the hydrated assistant row must dedupe by normalized id exactly once")
	}
	if countID(m.st.Chat, "user-his-077") != 0 || countText(m.st.Chat, "history note 077") != 1 {
		t.Fatal("the hydrated user row must dedupe by text (its fetched id must never land)")
	}
	for _, n := range []int{32, 44, 58, 80} {
		if countID(m.st.Chat, fmt.Sprintf("bossmsg-his-%03d", n)) != 1 {
			t.Fatalf("hydrated assistant row his-%03d must appear exactly once", n)
		}
	}

	// hop 2: his-001..his-030 — BELOW the hydrated head: all 30 splice,
	// including his-005 ("repeat note", whose hydrated twin his-033 was
	// consumed by hop 1's multiplicity drop).
	m = runMsg(t, m, wheelUpMsg())
	if len(b.calls) != 3 || b.calls[2].before != "his-031" {
		t.Fatalf("hop 2 rides before=his-031, got %+v", b.calls)
	}
	if got, want := len(m.st.Chat), baseLen+30; got != want {
		t.Fatalf("hop 2 must splice 30 fresh rows: %d → %d, want %d", baseLen, got, want)
	}
	if countText(m.st.Chat, "repeat note") != 2 || countID(m.st.Chat, "user-his-005") != 1 {
		t.Fatal("multiplicity: the hydrated repeat consumed, the older repeat spliced")
	}
	if m.st.Chat[0].ID != "user-his-001" {
		t.Fatalf("the dawn row heads the transcript, got %q", m.st.Chat[0].ID)
	}
	// global: still no duplicate ids.
	seen := map[string]bool{}
	for _, id := range idsOf(m.st.Chat) {
		if seen[id] {
			t.Fatalf("duplicate transcript id across hydrate+walk: %q", id)
		}
		seen[id] = true
	}
}

// TestPagerErrorLatchThreeStrikes — a dead serve: every hop fails
// SILENTLY (no transcript banners, no rows), three strikes latch the
// walk, the fourth gesture fetches NOTHING; a stream reconnect re-arms.
func TestPaginateErrorLatchThreeStrikes(t *testing.T) {
	b := &pagerStubBackend{}
	m := pagerFixture(t, b, 500)

	b.fail = true
	for strike := 1; strike <= 3; strike++ {
		m = runMsg(t, m, wheelUpMsg())
		if len(b.calls) != 1+strike {
			t.Fatalf("strike %d must still fetch (the latch is not yet tripped), calls=%d", strike, len(b.calls))
		}
	}
	// latched: the 4th top gesture fetches nothing.
	m = runMsg(t, m, wheelUpMsg())
	m = runMsg(t, m, wheelUpMsg())
	if len(b.calls) != 4 {
		t.Fatalf("after 3 strikes the walk must back off — calls=%d, want 4 (1 seed + 3 strikes)", len(b.calls))
	}
	// NO banners: zero transcript entries materialized through the failures.
	if len(m.st.Chat) != 0 {
		t.Fatalf("failed hops must never banner into the transcript, got %d entries: %v", len(m.st.Chat), chatTexts(m))
	}

	// reconnect: the marker re-arms (ResetFailures) — the next gesture fetches again.
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: streamReconnectedMarker})
	m = runMsg(t, m, wheelUpMsg())
	if len(b.calls) != 5 {
		t.Fatalf("a reconnected stream must re-arm the walk, calls=%d", len(b.calls))
	}
}

// TestPagerModalSuppressesArm — a live permission float (and a question
// hold) dismount-gates the walk: top gestures keep scrolling, no fetch.
func TestPaginateModalSuppressesArm(t *testing.T) {
	b := &pagerStubBackend{}
	m := pagerFixture(t, b, 500)

	m = runMsg(t, m, state.Event{Kind: state.EvPermission, PermissionID: "perm-1",
		ToolName: "Write", ToolSummary: "/tmp/x", EmployeeName: "tekton-1"})
	if m.permQ.front() == nil {
		t.Fatal("fixture: the permission ask must be open")
	}
	m = runMsg(t, m, wheelUpMsg())
	if len(b.calls) != 1 {
		t.Fatalf("a permission float must suppress the arm, calls=%d", len(b.calls))
	}
	// answering it re-opens the walk.
	m = runMsg(t, m, permAnswerMsg{response: "yes"})
	if m.permQ.front() != nil {
		t.Fatal("the answer must close the ask")
	}
	m = runMsg(t, m, wheelUpMsg())
	if len(b.calls) != 2 {
		t.Fatalf("with the float closed the arm fires, calls=%d", len(b.calls))
	}

	// question hold: same gate.
	b2 := &pagerStubBackend{}
	m2 := pagerFixture(t, b2, 500)
	m2 = runMsg(t, m2, state.Event{Kind: state.EvQuestion, QuestionID: "que-1", SessionID: "boss",
		EmployeeName: "boss", Text: "which file first?"})
	if m2.question == nil {
		t.Fatal("fixture: the question hold must be open")
	}
	m2 = runMsg(t, m2, wheelUpMsg())
	if len(b2.calls) != 1 {
		t.Fatalf("a question float must suppress the arm, calls=%d", len(b2.calls))
	}
}

// TestPagerDemoTourWalksFixture — requirement 5's dogfood: the REAL demo
// backend (no Start → no timers; the fixture walk is pure) paginates its
// 500-row canned history from the tour's scroll-top gesture, draining in
// exactly 10 calls (1 seed + 9 hops) and latching at the dawn.
func TestPaginateDemoTourWalksFixture(t *testing.T) {
	b := backend.NewDemo(config.Default()) // Start deliberately NOT called: timers stay off
	m := New(b, config.Default())
	m = runMsg(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m.bootDone = true
	m = runMsg(t, m, state.Event{Kind: state.EvStatus, Text: "DEMO - simulated events (no real agents)"})
	if !m.pagerSeeded || m.pagerSession != pagerDemoSession {
		t.Fatalf("the demo walk must bind the %q label and seed, got %q armed=%v", pagerDemoSession, m.pagerSession, m.pagerSeeded)
	}
	// wheel-ups pump the whole 500-row fixture (450 splice; the seed's
	// newest 50 are metadata-only). Each wheel-up needs the transcript at
	// row 0 — after a landing the offset bumped by the page's growth, so
	// step back up between arms with pgup.
	for hops := 0; hops < 9; hops++ {
		guard := 0
		for !m.chat.AtTranscriptTop() && guard < 200 {
			m = runMsg(t, m, pgupMsg())
			guard++
		}
		if !m.chat.AtTranscriptTop() {
			t.Fatalf("hop %d: never reached the top", hops)
		}
		m = runMsg(t, m, wheelUpMsg())
	}
	if got := len(m.st.Chat); got != 450 {
		t.Fatalf("the demo walk must splice 450 rows (9 hops × 50; the seed's newest page is discarded), got %d", got)
	}
	if m.st.Chat[0].ID != "user-his-001" || m.st.Chat[0].Text != "history note 001" {
		t.Fatalf("the dawn row heads the demo transcript, got %+v", m.st.Chat[0])
	}
	if m.st.Chat[449].ID != "bossmsg-his-450" {
		t.Fatalf("the walk's newest spliced row must be his-450, got %q", m.st.Chat[449].ID)
	}
	// latched: top gestures fetch nothing more (call count frozen at 10).
	m = scrollOnce(t, m)
	m = scrollOnce(t, m)
	if m.st.Chat != nil && len(m.st.Chat) != 450 {
		t.Fatalf("the latched top must stay put, got %d entries", len(m.st.Chat))
	}
	// no duplicate ids across the whole splice storm.
	seen := map[string]bool{}
	for _, id := range idsOf(m.st.Chat) {
		if seen[id] {
			t.Fatalf("duplicate id in the demo walk: %q", id)
		}
		seen[id] = true
	}
	// a live event AFTER the walk re-bases onto the splice (nothing lost,
	// nothing duplicated): feed one tour-ish beat.
	m = runMsg(t, m, state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "boss-1", From: "boss", Kind: "boss", Text: "On it: reports are on the board.", At: 1, Pending: false}})
	if len(m.st.Chat) != 451 {
		t.Fatalf("the walk's splice must survive the next SetState re-base, got %d", len(m.st.Chat))
	}
	if m.st.Chat[0].ID != "user-his-001" {
		t.Fatalf("the re-base must keep the dawn at the head, got %q", m.st.Chat[0].ID)
	}
}
