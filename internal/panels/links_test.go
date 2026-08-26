// links_test.go — the open-in-OS-browser suite (links.go):
//
//	(a) URL extraction: scheme gate (http/https only), markdown `](…)`
//	    closers, trailing prose punctuation, host-empty drops;
//	(b) path extraction: absolute, ./ ../ relative (cwd), ~/ expansion —
//	    EACH os.Stat-verified; prose tokens ("// comment", dead paths)
//	    never become targets (never launch an unverified path);
//	(c) deterministic order + dedupe by resolved value, first occurrence
//	    winning (./x and its absolute twin dedupe to ONE);
//	(d) the media carrier (state.ParseMediaMeta): a filename that stats
//	    joins as a Media-flagged file target; a barren chip name skips;
//	    a text+media collision dedupes with TEXT first;
//	(e) the exec seam: openInBrowser routes LinkURL/LinkFile through
//	    openRunner (captured, never spawned — the REAL `open -g` would
//	    spawn the user's browser mid-suite, so the exec itself is ONLY
//	    asserted through the seam here), rejects empty values, and
//	    re-stats files at the last mile (a moved file is a verdict);
//	(f) systemOpen's tool probe: a failed exec.LookPath resolves to
//	    errNoOpenTool (darwin branch on this host; the xdg-open branch is
//	    the same shape, one switch arm over);
//	(g) Chat.OpenTargets: no mark → nil; a mark over a link bubble → its
//	    set; a mark over a plain bubble → nil (the `o` claim's whole rule);
//	(h) the · o (open) beacon renders on verified-target bubbles only
//	    (user + boss cases), dangling URLs/dead paths get NO beacon;
//	(i) the target card: enter picks, esc cancels, ↑/↓ wrap, clicks in
//	    its frame are swallowed by cardClaims, and `o`-induced re-opens
//	    never reset a browsed cursor.
//
// No clocks, no fetches, no spawned browsers: every fixture is a literal
// ChatMsg + t.TempDir files, and the exec seam is captured via
// SetOpenRunnerForShot with t.Cleanup restores (parallel-suite hygiene).
package panels

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

// touchFile writes a tiny real file and returns its path (the os.Stat
// gate's positive leg).
func touchFile(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestLinkExtractURLs — scheme gate, markdown closers, prose punctuation,
// and host-empty drops.
func TestLinkExtractURLs(t *testing.T) {
	got := ExtractTargets(
		"see https://opencode.ai/docs, then a markdown [spec](https://spec.dev/page#frag) " +
			"and (http://example.com/inner) plus ftp://nope.plus " +
			"and bare www.skipped.com plus https://.")
	var urls []string
	for _, tgt := range got {
		if tgt.Kind != LinkURL {
			t.Fatalf("text scan must not invent file targets here: %+v", tgt)
		}
		urls = append(urls, tgt.Value)
	}
	want := []string{
		"https://opencode.ai/docs",
		"https://spec.dev/page#frag",
		"http://example.com/inner",
	}
	if strings.Join(urls, "|") != strings.Join(want, "|") {
		t.Fatalf("URL extraction must keep appear order and cut prose closers:\n got %q\nwant %q", urls, want)
	}
	// Names strip the scheme (the card + the "→ opened:" line read terse).
	if got[0].Name != "opencode.ai/docs" {
		t.Fatalf("URL Name strips the scheme, got %q", got[0].Name)
	}
}

// TestLinkExtractPaths — verified-path extraction across all four shapes,
// prose rejection, and the appear-order rule holding ACROSS kinds.
func TestLinkExtractPaths(t *testing.T) {
	root := t.TempDir()
	abs := touchFile(t, filepath.Join(root, "diagram.png"))
	rel := touchFile(t, filepath.Join(root, "notes.md"))
	home := t.TempDir()
	homeFile := touchFile(t, filepath.Join(home, "shot.png"))

	t.Chdir(root)          // ./ resolves from the project cwd's process root
	t.Setenv("HOME", home) // os.UserHomeDir reads $HOME on darwin+linux

	text := abs + " then ./notes.md: see ~/shot.png — " +
		"missing " + filepath.Join(root, "gone.txt") + " and // comment and nope.txt"
	got := ExtractTargets(text)
	var names, values []string
	for _, tgt := range got {
		if tgt.Kind != LinkFile {
			t.Fatalf("file targets only here: %+v", tgt)
		}
		names = append(names, tgt.Name)
		values = append(values, tgt.Value)
	}
	if len(got) != 3 {
		t.Fatalf("exactly the three ON-DISK files verify (missing + prose skip): %v", values)
	}
	if values[0] != abs || values[1] != rel || values[2] != homeFile {
		t.Fatalf("appear order + exact resolution (abs, ./→cwd, ~→$HOME): %v", values)
	}
	if names[0] != "diagram.png" || names[1] != "notes.md" || names[2] != "shot.png" {
		t.Fatalf("Names are basenames: %v", names)
	}
	// every extracted path is ABSOLUTE (the exec must never inherit a cwd)
	for _, v := range values {
		if !filepath.IsAbs(v) {
			t.Fatalf("targets carry absolute paths, got %q", v)
		}
	}
}

// TestLinkExtractOrderDedupe — one leftmost pass: a URL keeps its /path
// (the alternation's tie-break), identical URLs dedupe exact, and the
// ./x-vs-absolutex collision dedupes by RESOLVED value, first winning.
func TestLinkExtractOrderDedupe(t *testing.T) {
	root := t.TempDir()
	f := touchFile(t, filepath.Join(root, "a.txt"))
	t.Chdir(root)
	text := "https://x.dev/p " + "./a.txt https://x.dev/p " + f
	got := ExtractTargets(text)
	if len(got) != 2 {
		t.Fatalf("dedupe by resolved value — URL once, file once: %+v", got)
	}
	if got[0].Kind != LinkURL || got[1].Kind != LinkFile {
		t.Fatalf("appear order: the URL precedes the path token: %+v", got)
	}
	if got[1].Value != f { // ./a.txt resolved to the absolute twin
		t.Fatalf("./a.txt resolves absolute: %q", got[1].Value)
	}
}

// TestLinkMediaTargets — the boss bubble's attach carrier joins the set
// (Media-flagged) when the filename verifies; bare chip names and
// text-collisions skip (text keeps its appear-order head start).
func TestLinkMediaTargets(t *testing.T) {
	root := t.TempDir()
	img := touchFile(t, filepath.Join(root, "checker.png"))
	meta := state.MediaMeta([]state.MediaItem{{Mime: "image/png", Filename: img, W: 8, H: 8}})

	got := ExtractChatTargets(state.ChatMsg{ID: "b1", From: "boss", Text: "the diagram", Meta: meta})
	if len(got) != 1 || !got[0].Media || got[0].Value != img || got[0].Kind != LinkFile {
		t.Fatalf("a verified media filename is ONE Media-flagged file target: %+v", got)
	}

	bogus := state.MediaMeta([]state.MediaItem{{Mime: "image/png", Filename: "paste-diagram.png"}})
	if got := ExtractChatTargets(state.ChatMsg{Text: "x", Meta: bogus}); len(got) != 0 {
		t.Fatalf("a chip-only name that stats nowhere is NOT a target: %+v", got)
	}

	both := state.MediaMeta([]state.MediaItem{{Mime: "image/png", Filename: img}})
	got = ExtractChatTargets(state.ChatMsg{Text: "open " + img, Meta: both})
	if len(got) != 1 || got[0].Media {
		t.Fatalf("text+media collide on ONE resolved value, text's entry wins: %+v", got)
	}

	// a NON-attach carrier (the user-bubble att␟ one) is not media:
	userMeta := state.AttachMeta([]string{img})
	if got := ExtractChatTargets(state.ChatMsg{Text: "", Meta: userMeta}); len(got) != 0 {
		t.Fatalf("the user attach carrier is not a media source: %+v", got)
	}
}

// TestBrowserOpenSeam — the exec rides openRunner (captured, never
// spawned: the real `open -g` would push a browser at the USER's screen
// mid-suite — capture is the whole point), empty values refuse, and the
// last-mile os.Stat re-verifies moved files.
func TestBrowserOpenSeam(t *testing.T) {
	var opened []LinkTarget
	restore := SetOpenRunnerForShot(func(t LinkTarget) error {
		opened = append(opened, t)
		return nil
	})
	t.Cleanup(restore)

	url := LinkTarget{Kind: LinkURL, Value: "https://opencode.ai/docs", Name: "opencode.ai/docs"}
	if err := openInBrowser(url); err != nil {
		t.Fatalf("a verified URL opens clean: %v", err)
	}
	f := touchFile(t, filepath.Join(t.TempDir(), "real.png"))
	file := LinkTarget{Kind: LinkFile, Value: f, Name: "real.png"}
	if err := openInBrowser(file); err != nil {
		t.Fatalf("a verified file opens clean: %v", err)
	}
	if len(opened) != 2 || opened[0] != url || opened[1] != file {
		t.Fatalf("the seam captured both opens in order: %+v", opened)
	}

	// the seam's verdict surfaces verbatim (the transcript's dim row).
	boom := errors.New("boom")
	restore()
	restore = SetOpenRunnerForShot(func(LinkTarget) error { return boom })
	t.Cleanup(restore)
	if err := openInBrowser(url); !errors.Is(err, boom) {
		t.Fatalf("runner errors propagate: %v", err)
	}
	restore()

	// empty values never reach the seam.
	if err := openInBrowser(LinkTarget{Kind: LinkURL, Value: "  "}); err == nil {
		t.Fatal("an empty target refuses without touching the seam")
	}

	// the last-mile re-stat: a file that verified at extraction but moved
	// before the press is a verdict, not a launch.
	gone := filepath.Join(t.TempDir(), "gone.png")
	moved := LinkTarget{Kind: LinkFile, Value: gone, Name: "gone.png"}
	if err := openInBrowser(moved); err == nil || !strings.Contains(err.Error(), "no longer on disk") {
		t.Fatalf("a moved file re-stats to a verdict: %v", err)
	}
}

// TestBrowserSystemOpenToolGate — exec.LookPath's miss resolves to
// errNoOpenTool: the degraded-host verdict (darwin's `open` branch on this
// host; the xdg-open arm is the same shape one switch case over).
func TestBrowserSystemOpenToolGate(t *testing.T) {
	old := openLookPath
	openLookPath = func(string) (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { openLookPath = old })
	err := systemOpen(LinkTarget{Kind: LinkURL, Value: "https://x.dev"})
	if !errors.Is(err, errNoOpenTool) {
		t.Fatalf("a missing platform tool resolves to errNoOpenTool, got %v", err)
	}
}

// linkChat builds ONE boss bubble carryings targets + plain text, sized so
// the bubble's rows sit verbatim at the transcript's head (the selection
// fixture's dimensions: content rows == posted padded lines here).
func linkChat(t *testing.T, msgs ...state.ChatMsg) *Chat {
	t.Helper()
	c := NewChat(nil)
	c.SetSize(44, 24)
	c.SetState(state.OfficeState{Tick: 1, Chat: msgs})
	if len(c.selLines) == 0 {
		t.Fatal("the fixture must post transcript rows")
	}
	return c
}

// TestLinkChatOpenTargets — the `o` claim's resolver: the mark pins the
// bubble, the FIRST span-covered msg block with ≥1 verified target wins.
func TestLinkChatOpenTargets(t *testing.T) {
	root := t.TempDir()
	img := touchFile(t, filepath.Join(root, "shot.png"))
	meta := state.MediaMeta([]state.MediaItem{{Mime: "image/png", Filename: img}})

	// no mark at all: the claim is empty and `o` falls through to typing.
	c := linkChat(t, state.ChatMsg{ID: "b1", From: "boss", Kind: "boss",
		Text: "spec https://opencode.ai/docs done", At: 10})
	if got := c.OpenTargets(); got != nil {
		t.Fatalf("no selection ⇒ no targets: %+v", got)
	}

	// a one-cell armed press over the bubble's first row resolves its URL.
	row := 0
	for i, ln := range c.selLines {
		if strings.Contains(ansi.Strip(ln), "spec https://") {
			row = i
			break
		}
	}
	if !c.SelectionBegin(4, row) {
		t.Fatal("a press on the bubble row arms the selection")
	}
	targets := c.OpenTargets()
	if len(targets) != 1 || targets[0].Kind != LinkURL || targets[0].Value != "https://opencode.ai/docs" {
		t.Fatalf("the pressed bubble resolves its URL: %+v", targets)
	}

	// a mark over a PLAIN bubble finds nothing (the key types "o").
	c2 := linkChat(t, state.ChatMsg{ID: "u1", From: "user", Kind: "user", Text: "plain words", At: 10})
	if !c2.SelectionBegin(4, 0) {
		t.Fatal("press arms")
	}
	if got := c2.OpenTargets(); got != nil {
		t.Fatalf("a plain bubble's mark holds no targets: %+v", got)
	}

	// the media leg: the chip row's bubble resolves its verified filename.
	c3 := linkChat(t, state.ChatMsg{ID: "b2", From: "boss", Kind: "boss",
		Text: "the image", At: 10, Meta: meta})
	if !c3.SelectionBegin(4, 0) { // row 0 is the 🖼 chip row (media rows slot first)
		t.Fatal("press arms on the chip row")
	}
	targets = c3.OpenTargets()
	if len(targets) != 1 || targets[0].Media != true || targets[0].Value != img {
		t.Fatalf("the media bubble resolves its verified filename: %+v", targets)
	}
}

// TestLinkBeaconRender — verified-target bubbles wear " · o (open)" on
// their first row (user + boss cases); dead URLs / dead paths get none.
func TestLinkBeaconRender(t *testing.T) {
	root := t.TempDir()
	img := touchFile(t, filepath.Join(root, "b.png"))
	meta := state.MediaMeta([]state.MediaItem{{Mime: "image/png", Filename: img}})
	c := linkChat(t,
		state.ChatMsg{ID: "u1", From: "user", Kind: "user",
			Text: "look at https://opencode.ai/docs please", At: 10},
		state.ChatMsg{ID: "b1", From: "boss", Kind: "boss",
			Text: "checked.", At: 11, Meta: meta},
		state.ChatMsg{ID: "b2", From: "boss", Kind: "boss",
			Text: "nothing openable at /definitely/missing-" + root[len(root)-8:] + ".png", At: 12},
	)
	text := ansi.Strip(c.renderConversation())
	if n := strings.Count(text, "o (open)"); n != 2 {
		t.Fatalf("exactly the two verified-target bubbles wear the beacon, got %d:\n%s", n, text)
	}
	// the beacon hangs off the user bubble's HEAD row and flows with the
	// house's ANSI-aware fold (the 📎 suffix's exact contract: a full row
	// wraps the beacon to its continuation row rather than clipping).
	lines := strings.Split(text, "\n")
	head := lines[0] + lines[1]
	if !strings.Contains(head, "you ›") || !strings.Contains(head, "o (open)") {
		t.Fatalf("the user bubble's head carries the beacon (wrap-flow):\n%s", head)
	}
	// the dead-path bubble (b2) must NOT wear it anywhere: "definitely"
	// and "missing" surround the only prose region, and no beacon row
	// trails it.
	idx := strings.Index(text, "nothing openable")
	if idx < 0 || strings.Contains(text[idx:], "o (open)") {
		t.Fatalf("the dead-path bubble gets NO beacon:\n%s", text[idx:])
	}
}

// TestLinkPickerKeys — the target card: cursor walk + enter pick + esc
// cancel, click-swallow geometry, and re-open cursor reset.
func TestLinkPickerKeys(t *testing.T) {
	root := t.TempDir()
	img := touchFile(t, filepath.Join(root, "p.png"))
	targets := []LinkTarget{
		{Kind: LinkURL, Value: "https://opencode.ai/docs", Name: "opencode.ai/docs"},
		{Kind: LinkFile, Value: img, Name: "p.png", Media: true},
	}
	c := linkChat(t, state.ChatMsg{ID: "b1", From: "boss", Kind: "boss",
		Text: "spec https://opencode.ai/docs", At: 10})
	var picked []LinkTarget
	cancelled := 0
	c.SetLinkPickHandlers(
		func(t LinkTarget) tea.Cmd { picked = append(picked, t); return nil },
		func() tea.Cmd { cancelled++; return nil },
	)

	c.OpenLinkPicker(targets)
	if !c.LinkPickerOpen() {
		t.Fatal("the card is live")
	}
	// the card RENDERS in View with both rows + the hint, spliced
	// (ansi.Strip for the assert — the rails are styled).
	view := ansi.Strip(c.View())
	if !strings.Contains(view, "OPEN IN BROWSER") ||
		!strings.Contains(view, "opencode.ai/docs") ||
		!strings.Contains(view, "p.png") ||
		!strings.Contains(view, "enter: open") { // the hint's tail may clip on a narrow card
		t.Fatalf("the card paints title + both targets + the hint:\n%s", view)
	}
	// click-swallow: a point INSIDE the card frame is card-claimed.
	top, left, cardW, rows := c.linkCardGeom()
	if !c.cardClaims(left+1, top+1) {
		t.Fatal("the card swallows clicks inside its frame")
	}
	if c.cardClaims(left+cardW+1, top+len(rows)+1) {
		t.Fatal("outside the frame is NOT card territory")
	}
	// cursor walk wraps, enter picks the CURSOR's target.
	c.linkPickKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	c.linkPickKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if c.openPick.sel != 0 {
		t.Fatalf("the cursor wraps around the two rows, sel=%d", c.openPick.sel)
	}
	c.linkPickKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp}))
	if c.openPick.sel != 1 {
		t.Fatalf("up walks back, sel=%d", c.openPick.sel)
	}
	c.linkPickKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(picked) != 1 || picked[0].Value != img {
		t.Fatalf("enter submits the cursor's target: %+v", picked)
	}
	// esc cancels with zero side effects.
	c.linkPickKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	if cancelled != 1 {
		t.Fatalf("esc fired the cancel ferry exactly once: %d", cancelled)
	}
	// swallowed letter: while open, "o" types nothing and re-opens nothing.
	c.linkPickKey(tea.KeyPressMsg(tea.Key{Code: 'o', Text: "o"}))
	if len(picked) != 1 || cancelled != 1 {
		t.Fatal("stray keys are swallowed while the card is up")
	}
	c.CloseLinkPicker()
	if c.LinkPickerOpen() {
		t.Fatal("CloseLinkPicker closes the card")
	}
}

// TestLinkPickerThroughUpdate — the panel-Update claim rank: the card
// owns keys while open, and a live permission popover outranks it (a
// parked turn's modal wins — the whole claim rule).
func TestLinkPickerThroughUpdate(t *testing.T) {
	targets := []LinkTarget{
		{Kind: LinkURL, Value: "https://a.dev", Name: "a.dev"},
		{Kind: LinkURL, Value: "https://b.dev", Name: "b.dev"},
	}
	c := linkChat(t, state.ChatMsg{ID: "b1", From: "boss", Kind: "boss",
		Text: "spec https://a.dev", At: 10})
	var picked []LinkTarget
	c.SetLinkPickHandlers(
		func(t LinkTarget) tea.Cmd { picked = append(picked, t); return nil },
		func() tea.Cmd { return nil },
	)
	c.OpenLinkPicker(targets)
	// enter routed through Update fires the pick ferry.
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if len(picked) != 1 || picked[0].Value != "https://a.dev" {
		t.Fatalf("Update routes the pick: %+v", picked)
	}
	// with a permission popover up the card YIELDS its keys (the modal
	// outranks browsing) — up/down land on the modal, never the picker.
	c.SetPermission(&PermissionView{ID: "perm-1", ToolName: "bash", Summary: "ls", Agent: "boss", Index: 1, Total: 1})
	c.linkPickKey(tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})) // direct: move the cursor to row 1
	c.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))    // routed: the MODAL eats it, not the card
	if c.openPick.sel != 1 {
		t.Fatalf("the permission modal outranked the card: sel must stay 1, got %d", c.openPick.sel)
	}
}
