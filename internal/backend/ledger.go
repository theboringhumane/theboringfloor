// ledger.go — the office ledger WRITER/READER: records every completed
// dispatch (a returned worker, a flushed queue item) into the project's
// <dir>/.opencode/office-ledger.md so the NEXT session's boss knows what
// is already done. This is the fix at the heart of the memory amnesia
// defect: the office used to write ONLY agentmemory queue actions, so a
// re-booted boss re-dispatched finished work ("I don't have any session
// history or memory of a prior task for this project").
//
// Territory split (never fork): the charter pass (charter.go) SEEDS the
// file — header, the memory paragraph, the <!-- ledger:entries --> marker
// — and wires it into instructions; entries below that marker are THIS
// file's alone, appended newest-first, capped at 50.
//
// Guarantees, in the charter's own rhythm (same WriteFile idiom, same
// byte-stability):
//   - byte-stable: a no-change render is never written (self-diff gate),
//     and a duplicated ledgerId is a silent no-op (SAFE to call twice);
//   - atomic: the whole file is staged to a sibling temp file and renamed
//     over the old one — a crash mid-append never leaves half a ledger;
//   - lossy-tolerant: member/foreign blocks below the marker survive
//     verbatim (dedupe + cap ride their raw text) even when they don't
//     parse back into a LedgerEntry; a file that lost its seed/marker is
//     recovered with the canonical seed and its salvageable entries re-homed
//     under it;
//   - pure-shaped: entry rendering is fully deterministic (UTC-pinned
//     dates, sanitized one-line fields) — the same entry yields the same
//     bytes on every machine, which is what makes append tests goldens.
package backend

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LedgerEntry — the completed-dispatch record: what the boss next session
// reads to know. FROZEN SCHEMA, shared across the wave's workers: every
// lane (this file, the agentmemory observation in agentmemory.go, any
// reader) shapes the same struct — do not fork it.
type LedgerEntry struct {
	LedgerID      string   `json:"ledgerId"`
	DispatchTitle string   `json:"dispatchTitle"`
	WorkerName    string   `json:"workerName"`
	WorkerRole    string   `json:"workerRole"`
	WorkerSession string   `json:"workerSessionID"`
	Verdict       string   `json:"verdict"` // "done" | "issues"
	Summary       string   `json:"summary"` // the DONE bullets (untruncated)
	Files         []string `json:"files"`
	VerifyDigest  string   `json:"verifyDigest"`  // compact: last-passed command + exit
	ProofOneLiner string   `json:"proofOneLiner"` // first line of PROOF body
	Issues        []string `json:"issues"`
	CompletedAt   int64    `json:"completedAt"`
	PrimaryID     string   `json:"primarySessionID"`
	Project       string   `json:"project"` // dir basename
}

// ledgerCap — how many completed dispatches one project remembers at most:
// the ledger is the boss's working memory, not an archive; the newest 50
// entries stay, older ones age out (dropped from the END — entries are
// newest-first).
const ledgerCap = 50

// ledgerFileName is the append target inside the chartered .opencode dir.
const ledgerFileName = "office-ledger.md"

// LedgerID — the completion's stable identity: "led-<unixms>-<hash>" where
// the hash is FNV-1a-32 over title|worker, 8 lowercase hex. The stamp makes
// ids chronological; the digest makes a repeat call with the same
// (completedAt, title, worker) produce the SAME id — the "deterministic-
// capable" half tests pin directly. The file dedupes on it, which is what
// makes every append path safe to run twice.
func LedgerID(completedAt int64, title, worker string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(title + "\x1f" + worker))
	return fmt.Sprintf("led-%d-%08x", completedAt, h.Sum32())
}

// Ledger is the per-project handle over <root>/.opencode/office-ledger.md.
// root is the served directory (the live backend's b.directory); the
// project name is its basename — the same spelling the ledger entry's
// Project field carries.
type Ledger struct {
	root    string
	path    string
	project string
}

// NewLedger roots a ledger handle at dir (the file itself may not exist
// yet — Append creates it from the canonical seed).
func NewLedger(root string) *Ledger {
	root = filepath.Clean(root)
	return &Ledger{
		root:    root,
		path:    filepath.Join(root, ".opencode", ledgerFileName),
		project: filepath.Base(root),
	}
}

// ledgerWriteMu serializes appends inside this process: two async
// completions racing one file read-merge-write otherwise interleave.
var ledgerWriteMu sync.Mutex

// Append records e newest-first under the ledger's entries marker.
//
// Deduped on e.LedgerID (an already-present id is a silent no-op, so every
// caller is safe to call twice); capped at ledgerCap (oldest drop off the
// tail); byte-stable (the re-rendered file is compared to the on-disk
// bytes and an identical render is never written); atomic (temp + rename,
// never a partially-written ledger). Missing .opencode dir / file are
// created — an un-chartered project still remembers. Errors are returned
// soft for the caller's status line; nothing here panics.
func (l *Ledger) Append(e LedgerEntry) error {
	if e.LedgerID == "" {
		e.LedgerID = LedgerID(e.CompletedAt, e.DispatchTitle, e.WorkerName)
	}
	ledgerWriteMu.Lock()
	defer ledgerWriteMu.Unlock()

	raw, err := os.ReadFile(l.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("office ledger read: %w", err)
	}
	content := string(raw)
	// The dedupe gate: a landed ledgerId means this exact completion was
	// already recorded (a double-emitted idle, a retried flush, a
	// restarted office re-marking a queue item). Line-shape-tolerant: the
	// id is matched as a bullet so a title mentioning it can't misfire.
	if ledgerIDRe.MatchString(content) && strings.Contains(content, "- ledgerId: "+e.LedgerID) {
		return nil
	}

	preamble, blocks, recovered := splitLedger(content)
	if recovered {
		// Marker lost (malformed ledger): re-home ONLY well-formed entries,
		// re-rendered canonically — garbage prose and unparseable blocks
		// don't survive a recovery (the ledger is machine-owned state).
		salvaged := make([]string, 0, len(blocks))
		for _, blk := range blocks {
			if e, ok := parseLedgerBlock(blk); ok {
				salvaged = append(salvaged, renderLedgerBlock(e))
			}
		}
		blocks = salvaged
	}
	blocks = append([]string{renderLedgerBlock(e)}, blocks...)
	if len(blocks) > ledgerCap {
		blocks = blocks[:ledgerCap] // newest survive; the oldest age out
	}
	var out strings.Builder
	out.WriteString(preamble)
	for _, b := range blocks {
		out.WriteString(b)
	}
	result := out.String()
	if err == nil && result == content {
		return nil // byte-stable self-diff: an unchanged render is never written
	}

	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("office ledger mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "office-ledger-*.tmp")
	if err != nil {
		return fmt.Errorf("office ledger temp: %w", err)
	}
	if _, err := tmp.WriteString(result); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("office ledger write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("office ledger close: %w", err)
	}
	if err := os.Rename(tmp.Name(), l.path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("office ledger rename: %w", err)
	}
	return nil
}

// Entries parses the ledger file's well-formed entry blocks, newest-first.
// Foreign/member blocks that don't match the office's canonical shape
// survive Append verbatim but are skipped here — the reader surface never
// guesses at a hand-rolled block.
func (l *Ledger) Entries() []LedgerEntry {
	raw, err := os.ReadFile(l.path)
	if err != nil {
		return nil
	}
	_, blocks, _ := splitLedger(string(raw))
	var out []LedgerEntry
	for _, b := range blocks {
		if e, ok := parseLedgerBlock(b); ok {
			if e.Project == "" {
				e.Project = l.project
			}
			out = append(out, e)
		}
	}
	return out
}

// Latest returns the newest well-formed ledger entry; ok=false when the
// file is missing or holds no parseable entries.
func (l *Ledger) Latest() (LedgerEntry, bool) {
	entries := l.Entries()
	if len(entries) == 0 {
		return LedgerEntry{}, false
	}
	return entries[0], true
}

// Len counts the ledger's well-formed entries (0 on a missing/seed-only
// file) — the headless probe's "ledger N dispatches" figure.
func (l *Ledger) Len() int { return len(l.Entries()) }

// SummaryText renders a one-screen digest of the newest n entries (n<=0:
// all of them), date-first and one line per completion — the shape a boot
// log or a boss-facing recap can quote verbatim.
func (l *Ledger) SummaryText(project string, n int) string {
	entries := l.Entries()
	if len(entries) == 0 {
		return "no dispatches recorded in " + project + " yet"
	}
	shown := entries
	if n > 0 && n < len(entries) {
		shown = entries[:n]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d dispatches recorded in %s", len(entries), project)
	if len(shown) < len(entries) {
		fmt.Fprintf(&b, " (newest %d shown)", len(shown))
	}
	for _, e := range shown {
		date := time.UnixMilli(e.CompletedAt).UTC().Format("2006-01-02")
		fmt.Fprintf(&b, "\n- %s · %s — %s (%s) · %s: %s",
			date, e.DispatchTitle, e.WorkerName, e.WorkerRole, e.Verdict, e.Summary)
	}
	return b.String()
}

// ---------------------------------------------------------------- file shaping

// ledgerIDRe is the dedupe anchor's sanity form (used to short-circuit the
// Contains scan on ledgers that hold no ids at all).
var ledgerIDRe = regexp.MustCompile(`(?m)^- ledgerId: \S`)

// ledgerBlockStartRe opens an entry block: the canonical date-led header.
// Anything not shaped this way belongs to the preamble (or is dropped on
// recovery) — the writer's own blocks always match.
var ledgerBlockStartRe = regexp.MustCompile(`(?m)^### \d{4}-\d{2}-\d{2} · `)

// splitLedger divides content into (preamble, blocks, recovered): the
// byte-stable preamble (seed text through the end of the
// <!-- ledger:entries --> marker line, verbatim) and the raw entry blocks
// below it. A file WITHOUT a marker — the charter pass never ran, a
// member hand-rolled it, an older sketch — reports recovered=true: the
// canonical seed becomes the preamble and the app re-homes only the
// well-formed entries (see Append — garbage prose and unparseable blocks
// don't survive a recovery; the ledger is machine-owned state).
func splitLedger(content string) (string, []string, bool) {
	if idx := strings.Index(content, ledgerEntriesMarker); idx >= 0 {
		end := idx + len(ledgerEntriesMarker)
		if end < len(content) && content[end] == '\n' {
			end++
		}
		return content[:end], ledgerBlocks(content[end:]), false
	}
	return string(renderLedgerSeed()), ledgerBlocks(content), true
}

// ledgerBlocks cuts tail into raw entry blocks: each runs from its own
// "### YYYY-MM-DD · " header through the next one's start (or EOF).
// Verbatim text — capping and dedupe never re-render an existing block.
func ledgerBlocks(tail string) []string {
	idxs := ledgerBlockStartRe.FindAllStringIndex(tail, -1)
	blocks := make([]string, 0, len(idxs))
	for i, m := range idxs {
		end := len(tail)
		if i+1 < len(idxs) {
			end = idxs[i+1][0]
		}
		blocks = append(blocks, tail[m[0]:end])
	}
	return blocks
}

// renderLedgerBlock shapes one entry's on-disk block — the frozen layout,
// one blank line trailing:
//
//	### 2026-08-25 · <DispatchTitle> — <WorkerName> (<WorkerRole>) · `verdict`
//	- summary: <one-line first meaningful line of Summary>
//	- files: a, b, c
//	- verify: <digest>
//	- proof: <one-liner>
//	- ledgerId: <id>
//
// HEADER fields are one-line-sanitized AND separator-squashed (" · " and
// " — " are the parse separators, so free header text can't keep them);
// BULLET values only flatten (their "- key: " prefix parse is separator-
// blind, so a digest's " — ok (exit 0)" survives verbatim) — everything is
// width-capped; the untruncated record rides the agentmemory lane, the
// file is the digest.
func renderLedgerBlock(e LedgerEntry) string {
	var b strings.Builder
	date := time.UnixMilli(e.CompletedAt).UTC().Format("2006-01-02")
	verdict := headerField(e.Verdict, 16)
	if verdict == "" {
		verdict = "done"
	}
	role := headerField(e.WorkerRole, 24)
	if role == "" {
		role = "worker"
	}
	fmt.Fprintf(&b, "### %s · %s — %s (%s) · `%s`\n",
		date,
		headerField(e.DispatchTitle, 80),
		headerField(e.WorkerName, 40),
		role,
		verdict)
	b.WriteString("- summary: " + firstLedgerLine(e.Summary, 160) + "\n")
	files := "(none)"
	if len(e.Files) > 0 {
		clipped := make([]string, 0, len(e.Files))
		for _, f := range e.Files {
			clipped = append(clipped, oneLedgerLine(f, 100))
		}
		files = strings.Join(clipped, ", ")
	}
	b.WriteString("- files: " + files + "\n")
	b.WriteString("- verify: " + lineOrNone(e.VerifyDigest, 140) + "\n")
	b.WriteString("- proof: " + lineOrNone(e.ProofOneLiner, 140) + "\n")
	b.WriteString("- ledgerId: " + e.LedgerID + "\n\n")
	return b.String()
}

// oneLedgerLine flattens s to a single ledger-safe line — trimmed, every
// newline/CR collapsed to a space — then rune-caps it house-style. ""
// stays "" (callers choose their own empty rendering).
func oneLedgerLine(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return sliceMax(s, max)
}

// headerField is oneLedgerLine plus the header-parse safety squash: the
// entry header parses on " · " and " — ", so free HEADER text (title,
// worker, role, verdict) must not carry them (bullet values may — their
// "- key: " prefix parse never confuses them).
func headerField(s string, max int) string {
	s = strings.ReplaceAll(s, " · ", " - ")
	s = strings.ReplaceAll(s, " — ", " – ")
	return oneLedgerLine(s, max)
}

// firstLedgerLine picks the first MEANINGFUL line of a possibly
// multi-line body (a worker's DONE bullets): non-blank, NOT a contract
// section header (a Summary embedding "DONE\n- ..." starts at its first
// bullet, never at the bare header word), marker-stripped,
// ledger-sanitized and capped; an empty body renders "(none)".
func firstLedgerLine(s string, max int) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || matchLedgerSection(line) != "" {
			continue
		}
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		if line != "" {
			return oneLedgerLine(line, max)
		}
	}
	return "(none)"
}

// lineOrNone is oneLedgerLine with the ledger's "(none)" empty rendering.
func lineOrNone(s string, max int) string {
	if out := oneLedgerLine(s, max); out != "" {
		return out
	}
	return "(none)"
}

// parseLedgerBlock lifts a LedgerEntry back out of one raw block. The
// read-back is lossy BY DESIGN (the file is the digest): Summary is the
// one-liner, CompletedAt carries date precision only (UTC midnight), and
// the fields the file doesn't store (WorkerSession, PrimaryID, Issues)
// come back zero. ok=false means the block isn't the office's canonical
// shape — it survives on disk but stays invisible to readers.
func parseLedgerBlock(block string) (LedgerEntry, bool) {
	lines := strings.Split(block, "\n")
	header := strings.TrimPrefix(lines[0], "### ")
	if header == lines[0] {
		return LedgerEntry{}, false
	}
	// Header grammar: <date> · <title> — <worker> (<role>) · `verdict`.
	// " · " is the field separator (free text never contains it — the
	// writer squashes it), title/worker split on the LAST " — ".
	segs := strings.Split(header, " · ")
	if len(segs) < 3 {
		return LedgerEntry{}, false
	}
	date := segs[0]
	verdictRaw := segs[len(segs)-1]
	middle := strings.Join(segs[1:len(segs)-1], " · ")
	at, err := time.Parse("2006-01-02", date)
	if err != nil {
		return LedgerEntry{}, false
	}
	cut := strings.LastIndex(middle, " — ")
	if cut < 0 {
		return LedgerEntry{}, false
	}
	title := strings.TrimSpace(middle[:cut])
	wr := strings.TrimSpace(middle[cut+len(" — "):])
	worker, role := wr, ""
	if strings.HasSuffix(wr, ")") {
		if open := strings.LastIndex(wr, " ("); open > 0 {
			worker = wr[:open]
			role = wr[open+2 : len(wr)-1]
		}
	}
	e := LedgerEntry{
		LedgerID:      ledgerBullet(block, "ledgerId"),
		DispatchTitle: title,
		WorkerName:    worker,
		WorkerRole:    role,
		Verdict:       strings.Trim(verdictRaw, "` "),
		Summary:       noneToEmpty(ledgerBullet(block, "summary")),
		VerifyDigest:  noneToEmpty(ledgerBullet(block, "verify")),
		ProofOneLiner: noneToEmpty(ledgerBullet(block, "proof")),
		CompletedAt:   at.UnixMilli(),
	}
	if files := noneToEmpty(ledgerBullet(block, "files")); files != "" {
		e.Files = strings.Split(files, ", ")
	}
	if e.LedgerID == "" || e.DispatchTitle == "" || e.WorkerName == "" {
		return LedgerEntry{}, false
	}
	return e, true
}

// ledgerBullet reads one "- <key>: <value>" line out of a block.
func ledgerBullet(block, key string) string {
	prefix := "- " + key + ": "
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

// noneToEmpty maps the ledger's "(none)" placeholder back to "".
func noneToEmpty(s string) string {
	if s == "(none)" {
		return ""
	}
	return s
}

// ---------------------------------------------------------------- return-contract shaping

// ledgerSectionKeys are the developer return contract's named sections
// (the oikonomos developer's DONE / FILES / VERIFY / PROOF / ISSUES) the
// ledger mines digests from. Order matters to nothing; membership does.
var ledgerSectionKeys = []string{"DONE", "FILES", "VERIFY", "PROOF", "ISSUES"}

// matchLedgerSection reports the section keyword a line OPENS, "" when it
// opens none. Tolerant of the contract's real spellings — "DONE", "1.
// DONE", "2) FILES — the paths", "### VERIFY", "**PROOF:**" — and blind to
// prose ("the work is DONE", lowercase "files", a bullet about proof).
func matchLedgerSection(line string) string {
	s := strings.TrimSpace(line)
	s = strings.TrimLeft(s, "#> ")
	// numbered-heading prefix: "1. " / "2) "
	if i := 0; i < len(s) && s[i] >= '0' && s[i] <= '9' {
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i < len(s) && (s[i] == '.' || s[i] == ')') {
			i++
			for i < len(s) && s[i] == ' ' {
				i++
			}
			s = s[i:]
		}
	}
	// detect on a star-free copy so "**DONE**" reads as "DONE".
	detect := strings.ReplaceAll(s, "*", "")
	detect = strings.TrimRight(detect, " ")
	for _, key := range ledgerSectionKeys {
		if !strings.HasPrefix(detect, key) {
			continue
		}
		rest := detect[len(key):]
		if rest == "" || rest[0] == ' ' || rest[0] == ':' || rest[0] == '-' ||
			strings.HasPrefix(rest, "—") || strings.HasPrefix(rest, "–") {
			return key
		}
	}
	return ""
}

// parseLedgerSections splits a worker's return text into its contract
// sections (keyword -> body lines). Content before the first known header
// belongs to no section and is dropped — the ledger mines digests, not
// essays.
func parseLedgerSections(text string) map[string][]string {
	sections := map[string][]string{}
	cur := ""
	for _, line := range strings.Split(text, "\n") {
		if key := matchLedgerSection(line); key != "" {
			cur = key
			continue
		}
		if cur != "" {
			sections[cur] = append(sections[cur], line)
		}
	}
	return sections
}

// ledgerBullets strips blank lines and bullet markers ("- ", "* ") from a
// section body, keeping the remaining bullet texts in order.
func ledgerBullets(lines []string) []string {
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if t[0] == '-' || t[0] == '*' {
			t = strings.TrimSpace(t[1:])
		}
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ledgerFilePaths mines the FILES section's bullet list down to bare
// paths: the contract's "- <path> — why" / "- <path> (why)" shapes drop
// their gloss at the first separator. Capped at 12 (the ledger is a
// digest).
func ledgerFilePaths(lines []string) []string {
	var out []string
	for _, b := range ledgerBullets(lines) {
		path := b
		if i := strings.Index(path, " — "); i > 0 {
			path = path[:i]
		} else if i := strings.Index(path, " – "); i > 0 {
			path = path[:i]
		} else if i := strings.Index(path, " ("); i > 0 {
			path = path[:i]
		}
		path = strings.TrimSpace(path)
		if path != "" {
			out = append(out, path)
		}
		if len(out) >= 12 {
			break
		}
	}
	return out
}

// ledgerFirstLine / ledgerLastLine mine a section's first / last
// meaningful (non-blank, marker-stripped) line, capped — VERIFY keeps its
// last-passed command riding last, PROOF its one-liner first.
func ledgerFirstLine(lines []string, max int) string {
	b := ledgerBullets(lines)
	if len(b) == 0 {
		return ""
	}
	return oneLedgerLine(b[0], max)
}

func ledgerLastLine(lines []string, max int) string {
	b := ledgerBullets(lines)
	if len(b) == 0 {
		return ""
	}
	return oneLedgerLine(b[len(b)-1], max)
}

// ledgerIssues mines the ISSUES section's bullet list; the contract's
// "none" spellings collapse to nil — the caller's verdict rule is:
// non-empty issue bullets => "issues", anything else => "done".
func ledgerIssues(lines []string) []string {
	b := ledgerBullets(lines)
	body := strings.ToLower(strings.Trim(strings.Join(b, " "), ". "))
	if body == "" || body == "none" {
		return nil
	}
	return b
}
