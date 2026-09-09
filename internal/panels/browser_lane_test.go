// browser_lane_test.go — the premium browser lane's contract suite:
// the resolve matrix (pure, shell-out-free), the per-boot memoization,
// the failure-fallback state machine (fake session seam — no test owns a
// real child), the URL-state persistence across fallback, the suspend/
// resume/close lifecycle, the RegionView chrome, and ONE real-spawn reap
// test against a fixture-PATH fake binary (the --openurl precedent: real
// PTY, real exec, hermetic binary) proving no child leaks past a tab
// switch or office shutdown.
package panels

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/term"
)

// laneEnv builds the injected env read for the pure resolve matrix.
func laneEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func lookFound(string) (string, error)   { return "/fixture/terminal-browser", nil }
func lookMissing(string) (string, error) { return "", errors.New("exec: no command") }

// TestBrowserLaneResolveMatrix — the pure lane table: kitty-found,
// kitty-missing (PATH-resolution-failure), terminal-mismatch, kill-switch
// (both spellings), and the wave-85 OPT-IN gate (premium requires
// THEFLOOR_ZENBU_LANE=1 explicitly — "0" is NOT set). Every miss
// is the universal text lane.

// TestBrowserLaneOptInGate — the wave-85 default-off pivot's REASONED
// contract (the pure reasoned core's new class): premium requires the
// opt-in flag explicitly; qualified-but-unflagged is the opt-in-off class
// carrying the flag's name for the hint renderer (the killSwitchVar-style
// passthrough); the kill-switch keeps precedence over the opt-in gate;
// and "0" is NOT set. The hint copy for the class is pinned here too.

// pinKittyEnv — the hermetic kitty-capable host stub for the live-read
// tests (uishot's stubTermEnv checklist: every detect-layer input owned).
// The wave-85 opt-in flag is SET here: this is the "premium resolves"
// stub — every lane test that spawns/expects the premium embed rides it
// (the freeze/splitter/wrapper suites included); tests that want the
// default-off world clear the flag themselves after pinning.
func pinKittyEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TERM_PROGRAM", "ghostty")
	t.Setenv("TMUX", "")
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM_PROGRAM_VERSION", "")
	t.Setenv("WEZTERM_UNIX_SOCKET", "")
	t.Setenv("VSCODE_PID", "")
	t.Setenv("TERM", "xterm-256color")
	t.Setenv(BrowserLaneOffEnv, "")
	t.Setenv(TerminalBrowserOffEnv, "")
	t.Setenv(BrowserLaneOptInEnv, "1")
	old := zenbuLookPath
	zenbuLookPath = lookFound
	t.Cleanup(func() { zenbuLookPath = old })
}

// -------------------------------------------------------------------
// the fake session (the controller's zenbuSess seam — no real child)
// -------------------------------------------------------------------

type fakeZenbuSess struct {
	grid   *term.Grid
	sb     *term.Scrollback
	url    string
	cols   int
	rows   int
	alive  bool
	exited bool
	code   int
	life   time.Duration
	closed bool
	writes int
	frozen bool // the keep-alive posture (Freeze/Unfreeze flip it)
}

func newFakeZenbuSess(url string, cols, rows int) *fakeZenbuSess {
	return &fakeZenbuSess{
		grid: term.NewGrid(cols, rows), sb: term.NewScrollback(0),
		url: url, cols: cols, rows: rows, alive: true, code: -1,
	}
}

func (f *fakeZenbuSess) Alive() bool                  { return f.alive }
func (f *fakeZenbuSess) Exited() bool                 { return f.exited }
func (f *fakeZenbuSess) ExitCode() int                { return f.code }
func (f *fakeZenbuSess) Lifetime() time.Duration      { return f.life }
func (f *fakeZenbuSess) URL() string                  { return f.url }
func (f *fakeZenbuSess) Grid() *term.Grid             { return f.grid }
func (f *fakeZenbuSess) Scrollback() *term.Scrollback { return f.sb }
func (f *fakeZenbuSess) Size() (int, int)             { return f.cols, f.rows }
func (f *fakeZenbuSess) Close() error                 { f.closed, f.alive = true, false; return nil }
func (f *fakeZenbuSess) Write(p []byte) (int, error)  { f.writes++; return len(p), nil }
func (f *fakeZenbuSess) Freeze() error                { f.frozen = true; return nil }
func (f *fakeZenbuSess) Unfreeze() error              { f.frozen = false; return nil }
func (f *fakeZenbuSess) Frozen() bool                 { return f.frozen }
func (f *fakeZenbuSess) Resize(c, r int) error {
	f.cols, f.rows = c, r
	f.grid.SetSize(c, r)
	return nil
}

// fakeSpawnPins swaps the spawn seam for a factory capturing every fake
// it mints (the spawn count IS the no-flap latch's evidence).
func fakeSpawnPins(t *testing.T) *[]*fakeZenbuSess {
	t.Helper()
	var made []*fakeZenbuSess
	old := spawnZenbuSession
	spawnZenbuSession = func(url string, cols, rows int) (zenbuSess, error) {
		f := newFakeZenbuSess(url, cols, rows)
		made = append(made, f)
		return f, nil
	}
	t.Cleanup(func() { spawnZenbuSession = old })
	return &made
}

// fakeSpawnPinsCount swaps the spawn seam for the REAL factory wrapped in
// a counter (a test needing a REAL child AND the spawn-count evidence —
// the no-respawn latch over the genuine PTY seam).
func fakeSpawnPinsCount(t *testing.T) *int {
	t.Helper()
	count := 0
	old := spawnZenbuSession
	spawnZenbuSession = func(url string, cols, rows int) (zenbuSess, error) {
		count++
		return newZenbuSession(url, cols, rows)
	}
	t.Cleanup(func() { spawnZenbuSession = old })
	return &count
}

// laneWaitGrid — bounded wait for the child's bytes to paint the embedded
// grid (the reader loop is async; the assert itself carries no timing).
func laneWaitGrid(t *testing.T, g *term.Grid, want string) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		for y := 0; y < g.Rows(); y++ {
			if strings.Contains(g.LineText(y), want) {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the child's bytes never painted the embedded grid (want %q)", want)
}

// laneAssertReaped — the bounded-reap verdict: the session observed the
// exit AND the pid is gone (no child ever leaks past the office).
func laneAssertReaped(t *testing.T, s *ZenbuSession, pid int) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); !s.Exited() && time.Now().Before(deadline); {
		time.Sleep(10 * time.Millisecond)
	}
	if !s.Exited() {
		t.Fatal("the bounded reap observed the child's exit")
	}
	if err := syscall.Kill(pid, 0); err == nil || !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("no child leaks past the kill: kill(%d, 0) = %v", pid, err)
	}
}

// laneProcessState — the macOS-safe (no /proc) process state letter(s)
// ("Ts" — a stopped session leader; "Ss" — a live one): the SIGSTOP
// verdict's second rider after the heartbeat stall.
func laneProcessState(t *testing.T, pid int) string {
	t.Helper()
	out, err := exec.Command("ps", "-o", "stat=", "-p", fmt.Sprint(pid)).Output()
	if err != nil {
		t.Fatalf("ps -o stat= -p %d: %v", pid, err)
	}
	return strings.TrimSpace(string(out))
}

// -------------------------------------------------------------------
// the freeze-preserve discipline (the production freeze-leak fix):
// park PRESERVES the pending tail + open chain — the thawed child's
// tail COMPLETES the in-flight chain into a valid frame; resetting at
// the freeze dropped the chain's HEAD and the resumed tail painted the
// grid with raw base64 (the member's ~7 dense rows on re-open)
// -------------------------------------------------------------------

// kittyStreamPayload — the streaming fake's frame content (512B → 684
// base64 chars): BIG enough that a leaked mid-body tail trips the ≥40
// b64Runs signature several times over (kittyTestB64's 76-char fragments
// could hide under the threshold; the real child's ~4KB chunks never
// could). Content is irrelevant — the store never image-decodes.
var kittyStreamPayload = func() []byte {
	p := make([]byte, 0, 512)
	p = append(p, "\x89PNG\r\n\x1a\n"...)
	for len(p) < 512 {
		p = append(p, byte('A'+len(p)%26))
	}
	return p
}()

func kittyStreamB64() string { return base64.StdEncoding.EncodeToString(kittyStreamPayload) }

// browserLaneStreamFake — requirement 4's REAL streaming child (the
// production Electron's fixture-scale twin): ~2fps of THREE-chunk kitty
// frames (a=T m=1 / m=1 / m=0) under the child's every-repaint id i=1,
// EVERY chunk's APC body split across a 60ms sleep (the mid-body freeze
// windows — a freeze landing there catches the splitter holding a
// PARTIAL APC body, the exact leak shape), 300ms between generations.
func browserLaneStreamFake(root, b64 string) error {
	fake := "#!/bin/sh\n" +
		"printf '\\033[2J\\033[H'\n" +
		"printf 'TB-STREAM\\r\\n'\n" +
		"while true; do\n" +
		"printf '\\033_Ga=T,t=d,f=100,i=1,q=2,m=1;" + b64[:114] + "'\n" +
		"sleep 0.06\n" +
		"printf '" + b64[114:228] + "\\033\\\\'\n" +
		"printf '\\033_Gm=1;" + b64[228:342] + "'\n" +
		"sleep 0.06\n" +
		"printf '" + b64[342:456] + "\\033\\\\'\n" +
		"printf '\\033_Gm=0;" + b64[456:570] + "'\n" +
		"sleep 0.06\n" +
		"printf '" + b64[570:] + "\\033\\\\'\n" +
		"sleep 0.3\n" +
		"done\n"
	return os.WriteFile(filepath.Join(root, "terminal-browser"), []byte(fake), 0o755)
}

// zenbuSplitState — the splitter's freeze-point posture (read under the
// split mutex): the pending byte count, the scanner mode flags, whether
// a chunk chain is open, whether the buffer holds a DANGLING opener
// (unscanned ESC_G + partial body — the park scans nothing), and the
// pending head bytes (the freeze-point dump's evidence).
type zenbuSplitState struct {
	pending  int
	inAPC    bool
	discard  bool
	chain    bool
	parked   bool
	dangling bool
	pendHead string
}

func laneSplitState(s *ZenbuSession) zenbuSplitState {
	s.split.mu.Lock()
	defer s.split.mu.Unlock()
	head := s.split.pending
	if len(head) > 48 {
		head = head[:48]
	}
	return zenbuSplitState{
		pending:  len(s.split.pending),
		inAPC:    s.split.inAPC,
		discard:  s.split.discard,
		chain:    s.split.chain != nil,
		parked:   s.split.parked,
		dangling: danglingKittyOpener(s.split.pending),
		pendHead: string(head),
	}
}

// lanePlacementSeq — the live placement's store seq (the streaming
// fake's i=1 latest-wins placement is the commit clock: +1 per landed
// generation).
func lanePlacementSeq(t *testing.T, sess *ZenbuSession) uint64 {
	t.Helper()
	ps := sess.images.placements()
	if len(ps) != 1 {
		t.Fatalf("the streaming lane holds exactly one placement, got %d", len(ps))
	}
	return ps[0].seq
}

// laneWaitPlacementSeq — bounded wait for the store seq to reach want
// (the reader loop is async; the assert itself carries no timing).
func laneWaitPlacementSeq(t *testing.T, sess *ZenbuSession, want uint64, what string) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if lanePlacementSeq(t, sess) >= want {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("%s never landed (placement seq stuck at %d, want >= %d)", what, lanePlacementSeq(t, sess), want)
}

// laneFreezeMidStream — freeze the streaming child at a random point,
// looping until the park catches bytes IN FLIGHT (a non-empty pending
// and/or an open chain — the production shape: the Electron child
// repaints ~1.4MB chains at ~2fps, so it is virtually ALWAYS mid-chain
// when frozen). A between-frames freeze (nothing in flight) is the
// no-op leg: thaw and drift the phase.
func laneFreezeMidStream(t *testing.T, c *BrowserLaneController, sess *ZenbuSession) zenbuSplitState {
	t.Helper()
	for attempt := 0; attempt < 40; attempt++ {
		c.Suspend()
		time.Sleep(60 * time.Millisecond) // the kernel PTY buffer drains into the parked splitter
		st := laneSplitState(sess)
		if st.pending > 0 || st.chain {
			return st
		}
		c.Resume()
		time.Sleep(time.Duration(53+17*(attempt%6)) * time.Millisecond) // the phase drift
	}
	t.Fatal("40 flips never froze the stream mid-chain (the production case)")
	return zenbuSplitState{}
}

// TestBrowserLaneParkEmptyPendingNoop — edge discipline 3(a): a freeze
// with NOTHING in flight is a pure no-op — no drop note, no reset, and
// the lane parses cleanly across the boundary.
func TestBrowserLaneParkEmptyPendingNoop(t *testing.T) {
	ks, store, g, sb := newKittyRig(40, 8)
	ks.park()
	ks.unpark()
	if drops, note := store.dropStats(); drops != 0 || note != "" {
		t.Fatalf("an empty-pending park/unpark logs NOTHING: drops=%d note=%q", drops, note)
	}
	stream, b64 := kittyScriptedStream()
	if _, err := ks.Write([]byte(stream)); err != nil {
		t.Fatalf("splitter write: %v", err)
	}
	store.mu.Lock()
	im := store.images[1]
	store.mu.Unlock()
	if im == nil || !im.placed || im.b64 != b64 {
		t.Fatalf("the frame parses cleanly across the no-op boundary: %+v", im)
	}
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("zero base64 downstream (%d runs)", n)
	}
}

// TestBrowserLaneParkPreservesChain — the fix's DETERMINISTIC core (no
// process): one frame commits, the next generation's chain opens, the
// PARK lands, and the in-flight tail arrives WHILE PARKED (the SIGSTOP's
// effect lag) — the pending tail + open chain are PRESERVED (zero
// store/grid/scrollback mutations), the unpark drains the buffered tail
// (a partial APC holds), and the thawed child's remaining bytes COMPLETE
// the chain into a valid frame — ZERO base64 downstream. (The old
// reset-at-park dropped the chain's HEAD here; the tail leaked.)
func TestBrowserLaneParkPreservesChain(t *testing.T) {
	b64 := kittyTestB64()
	ks, store, g, sb := newKittyRig(40, 8)
	// generation 1 commits (the retained frame), generation 2's chain opens.
	ks.Write([]byte("\x1b[2J\x1b[H" + "TB-TOOLBAR\r\n" + kittyAPC("a=T,t=d,f=100,i=1,q=2", b64)))
	ks.Write([]byte(kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20])))
	seq0 := store.placements()[0].seq

	ks.park()
	gridAtPark := g.ScreenText()
	sbLenAtPark := len(sb.Raw())
	// the in-flight tail lands WHILE PARKED: an unterminated APC body.
	if _, err := ks.Write([]byte("\x1b_Gm=1;" + b64[20:40])); err != nil {
		t.Fatalf("parked write: %v", err)
	}
	// ZERO mutations while parked: the store, the grid, the scrollback
	// are a frozen snapshot; the bytes sit in the pending buffer.
	if got := store.placements()[0].seq; got != seq0 {
		t.Fatalf("ZERO store mutations while parked: seq moved %d → %d", seq0, got)
	}
	if got := g.ScreenText(); got != gridAtPark {
		t.Fatalf("ZERO grid mutations while parked:\n--- at park ---\n%s\n--- now ---\n%s", gridAtPark, got)
	}
	if got := len(sb.Raw()); got != sbLenAtPark {
		t.Fatalf("ZERO scrollback mutations while parked: %d → %d bytes", sbLenAtPark, got)
	}
	ks.mu.Lock()
	held := len(ks.pending) > 0 && ks.chain != nil
	ks.mu.Unlock()
	if !held {
		t.Fatal("the park PRESERVES the pending tail + the open chain")
	}

	ks.unpark() // the buffered tail drains: no terminator yet → it holds
	if got := store.placements()[0].seq; got != seq0 {
		t.Fatalf("the unpark's drain commits nothing without the terminator: seq %d → %d", seq0, got)
	}
	// the thawed child's tail completes chunk 2, then chunk 3 lands.
	ks.Write([]byte("\x1b\\"))
	if got := store.placements()[0].seq; got != seq0 {
		t.Fatalf("chunk 2 (m=1) continues the chain without committing: seq %d → %d", seq0, got)
	}
	ks.Write([]byte(kittyAPC("m=0", b64[40:])))
	// THE chain completed into a valid frame: full payload, placed, +1 seq.
	store.mu.Lock()
	im := store.images[1]
	store.mu.Unlock()
	if im == nil || !im.placed || im.b64 != b64 {
		t.Fatalf("the preserved chain completed into the valid frame: %+v", im)
	}
	if got := store.placements()[0].seq; got != seq0+1 {
		t.Fatalf("exactly ONE new apply across the thaw: seq %d → %d, want %d", seq0, got, seq0+1)
	}
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("ZERO base64 across the freeze/thaw (%d runs):\n%s", n, g.ScreenText())
	}
	if got := g.LineText(0); got != "TB-TOOLBAR" {
		t.Fatalf("the chrome survives: %q", got)
	}
}

// TestBrowserLaneParkedChainCap — edge discipline 3(b): the 8MiB chain
// cap (shrunk here) still bounds a pathological pending chain — the
// WHOLE over-cap chain arrives while PARKED (buffered unscanned), the
// unpark's drain hits the cap mid-join and drops the chain log-once —
// NEVER a flush to the grid or the scrollback — and the lane keeps
// working.
func TestBrowserLaneParkedChainCap(t *testing.T) {
	old := maxKittyChainB64
	maxKittyChainB64 = 32 // shrunk for the test
	t.Cleanup(func() { maxKittyChainB64 = old })
	b64 := kittyTestB64() // 76 chars > the 32-char cap once joined
	ks, store, g, sb := newKittyRig(40, 8)
	ks.park()
	ks.Write([]byte(kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20]) +
		kittyAPC("m=1", b64[20:60]) + // 20+40 = 60 joined > 32 → the cap fires in the drain
		kittyAPC("m=0", b64[60:])))
	if drops, _ := store.dropStats(); drops != 0 {
		t.Fatalf("ZERO store mutations while parked: drops=%d", drops)
	}
	ks.unpark() // the drain scans the buffered chain: the cap drops it
	drops, note := store.dropStats()
	if drops < 2 || !strings.Contains(note, "cap") {
		t.Fatalf("the cap drop + the chain-less remainder log-once: drops=%d note=%q", drops, note)
	}
	store.mu.Lock()
	if len(store.images) != 0 {
		t.Fatalf("the over-cap chain never stores: %d", len(store.images))
	}
	store.mu.Unlock()
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("the over-cap chain never leaks downstream (%d runs)", n)
	}
	ks.Write([]byte("alive"))
	if got := g.LineText(0); got != "alive" {
		t.Fatalf("the lane survives the parked cap drop: %q", got)
	}
}

// TestBrowserLaneParkedBufferCap — the parked path's OWN overflow (the
// APC body cap, shrunk here): an UNSCANNED ESC_G opener + a runaway
// unterminated body arrive while parked and bust the cap — the buffer +
// the open chain drop log-once AND the DISCARD mode engages (the
// dangling opener means the thawed child's next bytes are that APC's
// tail — they must discard to their terminator, NEVER flush to grid).
func TestBrowserLaneParkedBufferCap(t *testing.T) {
	old := maxKittyAPCBody
	maxKittyAPCBody = 64 // shrunk for the test
	t.Cleanup(func() { maxKittyAPCBody = old })
	b64 := kittyTestB64()
	ks, store, g, sb := newKittyRig(40, 8)
	ks.Write([]byte(kittyAPC("a=T,t=d,f=100,i=1,q=2,m=1", b64[:20]))) // the chain opens
	ks.park()
	// the runaway: an UNSCANNED opener + unterminated body over the cap.
	ks.Write([]byte("\x1b_Gm=1;" + strings.Repeat("A", 100))) // 109 bytes > 64
	ks.mu.Lock()
	parked, inAPC, discard, pend, chain := ks.parked, ks.inAPC, ks.discard, len(ks.pending), ks.chain != nil
	ks.mu.Unlock()
	if !parked || inAPC || !discard || pend != 0 || chain {
		t.Fatalf("the parked overflow drops the buffer + chain and engages the discard: parked=%v inAPC=%v discard=%v pending=%d chain=%v",
			parked, inAPC, discard, pend, chain)
	}
	drops, note := store.dropStats()
	if drops < 2 || !strings.Contains(note, "parked buffer over the cap") {
		t.Fatalf("the chain drop + the cap drop log-once: drops=%d note=%q", drops, note)
	}
	// the thaw: the runaway body's REST + its terminator discard; the
	// stream resyncs AFTER it.
	ks.unpark()
	ks.Write([]byte(strings.Repeat("B", 50) + "\x1b\\" + "alive"))
	if got := g.LineText(0); got != "alive" {
		t.Fatalf("the discard eats the runaway tail; the resync paints: %q", got)
	}
	if n := gridB64Runs(g) + b64Runs(string(sb.Raw())); n != 0 {
		t.Fatalf("the runaway tail NEVER flushes downstream (%d runs):\n%s", n, g.ScreenText())
	}
}
