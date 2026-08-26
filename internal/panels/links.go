// links.go — OPEN IN OS BROWSER: the chat transcript's `o` hotkey.
//
// The gesture: the member drags (or just presses) over a transcript
// bubble — the webpage-style selection seam of chat_selection.go — and
// presses `o`. The app (model.go) claims the key ONLY while:
//
//  1. the chat tab is focused (the transcript is a chat surface),
//  2. the CURRENT mark resolves to a bubble carrying ≥1 VERIFIED target
//     (OpenTargets below), and
//  3. NO floating modal is up (permission / question / /model picker —
//     a parked turn outranks browsing; the picker's own keys win).
//
// ONE target fires straight through the verdict seam; MULTIPLE float the
// small centered target card here (the /session picker's shape, minus the
// filter — target counts are tiny). On success the activity tab logs
// "→ opened: <name>"; on failure a dim office row in the transcript reads
// "could not open: <reason>" — a browser hiccup is never fatal.
//
// TARGETS (ExtractTargets / ExtractChatTargets):
//
//   - URLs: http:// and https:// only — the scheme gate is the whole
//     trust decision (no network fetch ever happens client-side).
//   - FILE PATHS in prose: /abs, ~/, ./ and ../ tokens — EACH verified
//     with os.Stat AFTER sanitizing/expanding/absoluting; a token that
//     does not name an existing file on disk is NOT a target (never
//     launch an unverified path — prose like "// comment" or markdown
//     link fragments vanish at this gate, deterministically).
//   - media: a boss bubble's attach carrier (state.ParseMediaMeta,
//     wave 62): a media Filename that itself is a verified path joins
//     the set, Media-flagged (bare names that stat nowhere are skipped).
//
// ORDER is deterministic: appear-order over the text scan (one leftmost
// pass, URLs before path tokens at a shared start byte), media items
// last, deduped by resolved value (absolute path / full URL) with the
// FIRST occurrence winning.
//
// SHELL-OUT (the clipboard.go clipboardCopyText precedent): the runner
// is a package-level seam — systemOpen by default (darwin `open -g`
// found via exec.LookPath, linux/BSD `xdg-open`; NEVER `sh -c`), swapped
// by tests and the uishot harness through SetOpenRunnerForShot.
package panels

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// LinkKind discriminates the two verified target classes.
type LinkKind int

const (
	// LinkFile — an os.Stat-verified on-disk path, ABSOLUTE after
	// resolution (sanitizePath); a last-mile re-stat gates the exec.
	LinkFile LinkKind = iota
	// LinkURL — an http(s) URL (scheme-validated, never fetched).
	LinkURL
)

// LinkTarget — ONE openable target found in a transcript bubble.
type LinkTarget struct {
	Kind  LinkKind // file path vs URL (the open primitive needs no more)
	Value string   // LinkURL: the full URL · LinkFile: the ABSOLUTE cleaned path
	Name  string   // display name (file basename / URL without scheme)
	Media bool     // rode a boss bubble's attach carrier (the 🖼 glyph in the picker)
}

// linkScanRe — the ONE leftmost extraction pass over bubble text: an
// http(s) URL, else a path-shaped token (~/, ./, ../, /). The alternation
// order wins ties at a shared start byte (a URL swallows its own /path).
// Whitespace and the markdown/prose closers (" ' < > [ ] ( ) `) terminate
// a token — `[x](./a.png)` extracts ./a.png, not ./a.png).
var linkScanRe = regexp.MustCompile(`https?://[^\s"'<>\[\]()]{1,}` + `|(?:~|\.\.?|)/[^\s"'<>\[\]()]{1,}`)

// linkTrimRight — prose punctuation that trails a spoken token but is
// never part of the resource ("see https://x.y/z." — the dot ends the
// sentence, not the host).
const linkTrimRight = ".,;:!?"

// ExtractTargets scans ONE text (a bubble body) for openable targets:
// http(s) URLs plus os.Stat-verified file paths, in APPEAR ORDER, deduped
// by resolved value. Unverifiable path tokens (the file moved, the token
// was prose) are skipped — extraction IS the existence gate, so a marker
// or an `o` press only ever sees targets that resolve on disk RIGHT NOW.
func ExtractTargets(text string) []LinkTarget {
	if text == "" {
		return nil
	}
	var out []LinkTarget
	seen := map[string]bool{}
	for _, span := range linkScanRe.FindAllStringIndex(text, -1) {
		tok := strings.TrimRight(text[span[0]:span[1]], linkTrimRight)
		t, ok := classifyToken(tok)
		if !ok || seen[t.Value] {
			continue
		}
		seen[t.Value] = true
		out = append(out, t)
	}
	return out
}

// ExtractChatTargets is a transcript bubble's WHOLE open-target set: the
// body text's scan (URLs/paths in appear order), then the attach carrier's
// media filenames (each verified exactly like a prose path; a barren
// chip-only name that stats nowhere is skipped). Deduped by resolved
// value, first occurrence winning; media entries flag Media for the
// picker's glyph.
func ExtractChatTargets(m state.ChatMsg) []LinkTarget {
	out := ExtractTargets(m.Text)
	seen := make(map[string]bool, len(out))
	for _, t := range out {
		seen[t.Value] = true
	}
	if items, ok := state.ParseMediaMeta(m.Meta); ok {
		for _, it := range items {
			t, ok := resolvePathTarget(strings.TrimSpace(it.Filename))
			if !ok || seen[t.Value] {
				continue
			}
			t.Media = true
			seen[t.Value] = true
			out = append(out, t)
		}
	}
	return out
}

// classifyToken turns one scanned token into a VERIFIED target, dropping
// scheme-less URLs (host empty) and any file token that fails the disk
// gate.
func classifyToken(tok string) (LinkTarget, bool) {
	if tok == "" {
		return LinkTarget{}, false
	}
	if strings.HasPrefix(tok, "http://") || strings.HasPrefix(tok, "https://") {
		name := strings.TrimPrefix(strings.TrimPrefix(tok, "https://"), "http://")
		if name == "" {
			return LinkTarget{}, false
		}
		return LinkTarget{Kind: LinkURL, Value: tok, Name: name}, true
	}
	return resolvePathTarget(tok)
}

// resolvePathTarget verifies a path-shaped token: sanitize (control
// bytes in transcript prose are a hard no), expand ~, absolutize against
// the process cwd, clean — then the os.Stat gate. Only verified FILES
// become targets: directories are rejected on purpose — prose artifacts
// ("// comment" → "/", "."/"..", a bare "~") all STAT HAPPY as dirs, and
// the feature's scope is file/url (a Finder/xdg-open window for the
// filesystem ROOT is never the intent).
func resolvePathTarget(raw string) (LinkTarget, bool) {
	p, err := sanitizePath(raw)
	if err != nil {
		return LinkTarget{}, false
	}
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		return LinkTarget{}, false
	}
	return LinkTarget{Kind: LinkFile, Value: p, Name: filepath.Base(p)}, true
}

// sanitizePath — the path half of "never launch an unverified string":
// no control bytes (a transcript token must never reach an exec argv with
// a newline payload), ~ expansion, absolute against cwd, cleaned.
func sanitizePath(raw string) (string, error) {
	if raw == "" || strings.ContainsAny(raw, "\x00\r\n") {
		return "", fmt.Errorf("unsafe path %q", raw)
	}
	p := raw
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand ~: %w", err)
		}
		if p == "~" {
			p = home
		} else {
			p = filepath.Join(home, p[2:])
		}
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("abs %q: %w", raw, err)
	}
	return filepath.Clean(abs), nil
}

// -------------------------------------------------------------------
// the OS browser shell-out
// -------------------------------------------------------------------

// openRunner — THE shell-out seam (clipboard.go's clipboardCopyText
// precedent): systemOpen by default, swapped by tests and the uishot
// harness, ALWAYS restored (leaks across suites break determinism).
var openRunner = systemOpen

// openLookPath — the tool probe (exec.LookPath's twin): stubbed by tests
// to prove the missing-tool verdict without depending on the host PATH.
var openLookPath = exec.LookPath

// errNoOpenTool — the degraded-host verdict: neither `open` nor
// `xdg-open` found. Renders as the dim "could not open:" transcript row
// (never fatal).
var errNoOpenTool = errors.New("no open/xdg-open tool on this host")

// SetOpenRunnerForShot swaps the shell-out seam for a shot/test harness
// and returns the restore closure. The house's ForShot pattern (the wedge
// watchdog's SetWedgeAfterForShot twin): the REAL exec otherwise.
func SetOpenRunnerForShot(fn func(LinkTarget) error) (restore func()) {
	old := openRunner
	openRunner = fn
	return func() { openRunner = old }
}

// OpenInBrowser is the app package's entry (internal/app cannot reach the
// unexported twin) — the same verified open.
func OpenInBrowser(t LinkTarget) error { return openInBrowser(t) }

// openInBrowser fires ONE verified target at the OS default browser /
// handler. A file target re-stats RIGHT BEFORE the exec (the extraction
// gate ran at render/pick time; files move — never launch an unverified
// path). The exec itself rides the seam, so tests capture and shots stub.
func openInBrowser(t LinkTarget) error {
	if strings.TrimSpace(t.Value) == "" {
		return errors.New("empty target")
	}
	if t.Kind == LinkFile {
		if _, err := os.Stat(t.Value); err != nil {
			return fmt.Errorf("path no longer on disk: %s", t.Value)
		}
	}
	return openRunner(t)
}

// systemOpen — the platform dispatch, both tools exec.LookPath-gated (a
// missing tool is a VERDICT, never a spawn): darwin runs `open -g`
// (background — the browser must not steal the terminal's focus); linux
// and the BSDs run bare `xdg-open` (some tool; windows needs rundll32 —
// out of scope, the verdict row says so). NEVER `sh -c`.
func systemOpen(t LinkTarget) error {
	if runtime.GOOS == "darwin" {
		if _, err := openLookPath("open"); err != nil {
			return errNoOpenTool
		}
		if err := exec.Command("open", "-g", t.Value).Run(); err != nil {
			return fmt.Errorf("open: %w", err)
		}
		return nil
	}
	if _, err := openLookPath("xdg-open"); err != nil {
		return errNoOpenTool
	}
	if err := exec.Command("xdg-open", t.Value).Run(); err != nil {
		return fmt.Errorf("xdg-open: %w", err)
	}
	return nil
}

// -------------------------------------------------------------------
// selection → bubble resolution
// -------------------------------------------------------------------

// blockAtRow maps an ABSOLUTE content row (selLines space) to the block
// that rendered it — mergeBlockHits' exact row arithmetic (blocks stacked,
// ONE blank separator row between). Separator and out-of-range rows
// return nil.
func (c *Chat) blockAtRow(row int) *chatBlock {
	r := 0
	for i, blk := range c.blocks {
		if i > 0 {
			r++ // the ONE separator row between timeline items
		}
		if row >= r && row < r+blk.rows {
			return blk
		}
		r += blk.rows
	}
	return nil
}

// OpenTargets resolves the open-in-browser candidate set under the CURRENT
// transcript mark (armed one-cell presses AND finalized drags both row a
// span): the normalized span walks in reading order and the FIRST msg
// block carrying ≥1 verified target wins (worker-thread group blocks and
// separator rows carry none — groups never pin a target set). nil while
// no mark is live or nothing under it verifies — the app then lets `o`
// fall through and type into the draft, unclaimed.
func (c *Chat) OpenTargets() []LinkTarget {
	if !c.SelectionActive() {
		return nil
	}
	loRow, _, hiRow, _ := c.selNorm()
	for r := loRow; r <= hiRow; r++ {
		blk := c.blockAtRow(r)
		if blk == nil || blk.src.ID == "" {
			continue
		}
		if targets := ExtractChatTargets(blk.src); len(targets) > 0 {
			return targets
		}
	}
	return nil
}

// -------------------------------------------------------------------
// the target picker float (≥2 targets)
// -------------------------------------------------------------------

// linkPickState — the open target card: the bubble's candidate set (≥2 —
// a lone target fires straight past this card) and the cursor.
type linkPickState struct {
	targets []LinkTarget
	sel     int
}

// linkPickHint — the card's dim footer (sessHint's twin).
const linkPickHint = "↑/↓: move · enter: open · esc: cancel"

// linkHigh — the cursor row's reversed-accent run (sessHigh's twin).
var linkHigh = lipgloss.NewStyle().Foreground(chrome.Accent).Reverse(true)

// SetLinkPickHandlers wires the app's pick/cancel ferries (the session
// picker's contract): enter on a row submits its target, esc cancels —
// both land app-side as tea.Msgs, the model value copy stays the single
// writer.
func (c *Chat) SetLinkPickHandlers(pick func(t LinkTarget) tea.Cmd, cancel func() tea.Cmd) {
	c.onLinkPick, c.onLinkCancel = pick, cancel
}

// OpenLinkPicker floats the card over the chat with the resolved set.
// The selection's mark STAYS (esc-lawful layers: the card owns the keys
// until it closes, then the mark is the next esc's); the cursor starts on
// the first target (appear order — text scan before media).
func (c *Chat) OpenLinkPicker(targets []LinkTarget) {
	if len(targets) == 0 {
		return
	}
	c.openPick = &linkPickState{targets: targets}
}

// CloseLinkPicker closes the card (the app's pick/cancel landings both
// run it — the card owns zero side effects of its own). Idempotent.
func (c *Chat) CloseLinkPicker() { c.openPick = nil }

// LinkPickerOpen reports whether the target card is live — the app's `o`
// claim yields while it is (the card owns the key; re-opening must not
// reset a browsed cursor), and the app-level esc-to-clear-selection gate
// yields too (the card's esc wins first).
func (c *Chat) LinkPickerOpen() bool { return c.openPick != nil }

// linkMove wraps the cursor by d rows (the permission popover's exact
// wrap).
func (c *Chat) linkMove(d int) {
	if n := len(c.openPick.targets); n > 0 {
		c.openPick.sel = (c.openPick.sel + d + n) % n
	}
}

// linkPickKey owns EVERY key while the card floats (question-modal
// contract — claimed in Chat.Update AFTER the permission/question
// floats: a parked turn's modal outranks browsing): ↑/↓/tab move, enter
// submits the cursor's target through onLinkPick, esc fires onLinkCancel,
// pgup/pgdn still scroll the transcript (sessKey's rule); every other key
// is swallowed — typing belongs nowhere while the card owns the mark.
func (c *Chat) linkPickKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		if c.onLinkCancel != nil {
			return c.onLinkCancel()
		}
		return nil
	case "up":
		c.linkMove(-1)
		return nil
	case "down", "tab":
		c.linkMove(1)
		return nil
	case "enter":
		p := c.openPick
		if p == nil || p.sel < 0 || p.sel >= len(p.targets) || c.onLinkPick == nil {
			return nil
		}
		return c.onLinkPick(p.targets[p.sel])
	case "pgup", "pgdown":
		var cmd tea.Cmd
		c.vp, cmd = c.vp.Update(msg)
		if msg.String() == "pgdown" && c.vp.AtBottom() {
			c.follow = true
		} else {
			c.follow = false
		}
		return cmd
	}
	return nil
}

// linkCard renders the picker's rows, each EXACTLY cardW display cells
// (rails included) — the float-splice contract: cells are replaced, never
// lines, so the layout never jumps mid-pick.
func (c *Chat) linkCard() (rows []string, cardW int) {
	cardW = floatCardW(c.w)
	inner := cardW - 2 // the content column between the │ rails
	rail := func(s string) string {
		return chrome.AccentText.Render("│") + s + chrome.AccentText.Render("│")
	}
	blank := strings.Repeat(" ", inner)
	p := c.openPick

	rows = append(rows, chrome.AccentText.Render("╭"+strings.Repeat("─", inner)+"╮"))

	// Title row: ACCENT-bold header left, dim count badge right.
	title := "OPEN IN BROWSER"
	badge := "1/" + itoa(len(p.targets))
	gap := inner - 2 - lipgloss.Width(title) - lipgloss.Width(badge)
	if gap < 1 {
		gap = 1
	}
	rows = append(rows, rail(" "+chrome.AccentText.Bold(true).Render(title)+strings.Repeat(" ", gap)+
		chrome.DimText.Render(badge)+" "))

	// The targets, one row each (the count is small — no windowing,
	// sessVisibleRows was for the root session list's lengths).
	for i, t := range p.targets {
		rows = append(rows, rail(linkMenuRow(t, i == p.sel, inner)))
	}
	rows = append(rows, rail(blank))

	// Footer hint — dim italic (sessHint's slot).
	rows = append(rows, rail(chrome.DimText.Italic(true).Render(" "+fitPlain(linkPickHint, inner-2))+" "))
	rows = append(rows, chrome.AccentText.Render("╰"+strings.Repeat("─", inner)+"╯"))
	return rows, cardW
}

// linkMenuRow renders one target row of EXACTLY inner cells: the cursor
// marker + kind glyph + display name left, the dim value right; the
// highlighted row runs reversed-accent across the whole content column.
func linkMenuRow(t LinkTarget, on bool, inner int) string {
	mark := "  "
	if on {
		mark = "› "
	}
	glyph := "🔗" // LinkURL
	switch {
	case t.Media:
		glyph = "🖼"
	case t.Kind == LinkFile:
		glyph = "📄"
	}
	body := mark + glyph + " " + t.Name + " " + t.Value
	if on {
		return linkHigh.Render(fitLabel(body, inner))
	}
	out := mark + glyph + " " + t.Name + " " + chrome.DimText.Render(t.Value)
	return fitLabel(out, inner)
}

// linkCardGeom — where the card sits, in CHAT CONTENT COORDS (row 0 = the
// viewport's first rendered row): centered over the WHOLE panel on both
// axes, fixed regardless of transcript scroll (the sess/perm/quest cards'
// identical contract).
func (c *Chat) linkCardGeom() (top, left, cardW int, rows []string) {
	rows, cardW = c.linkCard()
	top = (c.h - len(rows)) / 2
	if top < 0 {
		top = 0 // a very short panel pins the card to the top edge
	}
	left = (c.w - cardW) / 2
	if left < 0 {
		left = 0
	}
	return top, left, cardW, rows
}

// linkOverlay splices the card's rows over the assembled background
// lines cell-wise (ANSI-aware, permSplice): background pixels outside the
// frame survive and the row count never changes. Runs FIRST of the float
// stack (bottom) — the parked turn's permission/question cards splice
// over it and own the keys; the session picker (a peer browse) layers
// above too.
func (c *Chat) linkOverlay(bg []string) []string {
	top, left, _, rows := c.linkCardGeom()
	for i, row := range rows {
		y := top + i
		if y < 0 || y >= len(bg) {
			continue // a short panel clips the card instead of growing
		}
		bg[y] = permSplice(bg[y], left, row, lipgloss.Width(row), c.w)
	}
	return bg
}
