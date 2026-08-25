// sessions.go — per-directory office-session persistence: quit and relaunch
// in the same folder and the previous transcript + roster + board come back;
// /new starts a fresh office whenever the member wants one.
//
// Layout: <THEBORINGOFFICE_HOME|HOME>/.theboringoffice/sessions/<dirhash>/session.json,
// dirhash = sha1 of the canonical working directory (symlinks resolved
// best-effort). The file carries the primary ("boss") session id plus the
// office surfaces (transcript, roster, board, mail) trimmed to the last 200
// chat / 50 task / 50 mail entries. Writes are ATOMIC — a unique tmp file
// per writer + rename: two theboringoffice instances in the same directory can race,
// can never corrupt; LAST WRITER WINS by design (comment per the workload
// ruling — same-dir concurrent instances are the user's choice).
//
// Restore + persist are LIVE-only. Demo mode is a scripted tour — restoring
// a real transcript into it (or persisting the scripted one) would confuse
// the tour (demo restore = confusing, per the requirement ruling).
//
// Backend seams are ADDITIVE (the app type-asserts them; harness stubs —
// uishot/headless — simply never implement them):
//
//	PrimaryOverride(id) — pre-Start resume pin (server-side 404 falls back
//	                      to find-or-create; boot never hard-fails). Fed by
//	                      session.json's stored id AND by the -s/--session
//	                      boot flag (an explicit pin that skips the freshness
//	                      gate)
//	PrimaryID()         — current primary, feeds the snapshot
//	NewOffice()         — /new: mint a fresh "theboringoffice office" primary NOW
//	ListSessions(ctx)   — /session picker: the server's ROOT sessions with
//	                      message counts (session_picker.go; missing → the
//	                      static summary + a dim unavailable note)
//	ResumeOffice(id)    — /session picker accept's CAPABILITY GATE (a
//	                      with-it backend can be re-pinned by a boot). Kept
//	                      as backend API; the picker itself swaps by QUIT +
//	                      EXEC-REPLACE (session_picker.go).
package app

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

const (
	// sessionChatCap — transcript entries kept on disk (the renderer's
	// in-memory caps stay as they were — chatCap/thinkCap/toolCap).
	sessionChatCap = 200
	// sessionListCap — board + mail rows kept on disk.
	sessionListCap = 50
	// sessionFreshWindow — a session.json older than this is stale: the
	// boot runs the normal find-or-create (it is NOT deleted either — the
	// next cheap-write simply overwrites it).
	sessionFreshWindow = 4 * 24 * time.Hour
	// sessionWriteMinGap — the cheap-write loop cadence (every EvTick
	// checks; at most one write per window).
	sessionWriteMinGap = 5 * time.Second
	// bootNoticeMeta — Meta marker for boot-scoped notices (the restore
	// line): rendered in the live chat but NEVER persisted — Snapshot
	// strips it, so each boot shows exactly ONE restore line on screen
	// and the file carries ZERO on subsequent cycles.
	bootNoticeMeta = "boot"
	// bootWarnNoticeMeta — the wedge watchdog's boot-scoped sibling: a
	// red boot-ONLY warning row ("boss turn wedged …") that must never
	// survive into session.json — a stale one reads as broken hours
	// later, and every boot already re-watches the turn.
	bootWarnNoticeMeta = "boot-warn"
	// restoreNoticePrefix — the legacy self-clean marker: session.json
	// files written before the boot-notice dedupe carry restore lines as
	// PLAIN office notices (no Meta marker); hydrateSession drops them by
	// prefix so an already-polluted file cleans itself on the next boot.
	restoreNoticePrefix = "restored office session from "
)

// SessionFile — the on-disk office session for ONE working directory.
type SessionFile struct {
	Dir       string            `json:"dir"`
	PrimaryID string            `json:"primaryID"`
	Agents    []state.Employee  `json:"agents"`
	Tasks     []state.BoardTask `json:"tasks"`
	Mails     []state.MailItem  `json:"mails"`
	Chat      []state.ChatMsg   `json:"chat"`
	// PlanText — the plan editor's drafted-but-unapproved buffer
	// (live-only; see plan_mode.go). A drafted plan survives quit and
	// relaunch; an empty or never-edited starter template writes nothing
	// (omitempty), and /new + successful approvals clear the field by
	// resetting the buffer itself.
	PlanText string `json:"planText,omitempty"`
	SavedAt  int64  `json:"savedAt"` // unix millis
}

// sessionsBase — <THEBORINGOFFICE_HOME|HOME>/.theboringoffice/sessions (the
// home override is the test/harness scratch root, consistent with
// config.Path(); it also honors the pre-rename GRAFEIO_HOME).
func sessionsBase() string {
	home := config.HomeOverride()
	if home == "" {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".theboringoffice", "sessions")
}

// legacySessionsBase — the pre-rename ("grafeio") sessions root. Read
// fallback only: LoadSession consults it when the new file is absent, so an
// upgrade restores the old office transcript instead of silently starting
// over. Writes always go to sessionsBase().
func legacySessionsBase() string {
	home := config.HomeOverride()
	if home == "" {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".grafeio", "sessions")
}

// SessionDirHash — sha1 of the canonical directory path (EvalSymlinks
// best-effort, Abs fallback; a collision at sha1 is irrelevant).
func SessionDirHash(dir string) string {
	canon, err := filepath.Abs(dir)
	if err != nil {
		canon = dir
	}
	if eval, err := filepath.EvalSymlinks(canon); err == nil {
		canon = eval
	}
	sum := sha1.Sum([]byte(canon))
	return hex.EncodeToString(sum[:])
}

// SessionPath — the session.json location for this working directory.
func SessionPath(dir string) string {
	return filepath.Join(sessionsBase(), SessionDirHash(dir), "session.json")
}

// LoadSession reads the office session for dir. ok=false covers BOTH "no
// file" and "malformed/unreadable file" — a corrupt session degrades
// silently to the normal fresh boot (never an error dialog, never a crash).
func LoadSession(dir string) (*SessionFile, bool) {
	b, err := os.ReadFile(SessionPath(dir))
	if err != nil {
		// rename-era read fallback: the session may live under the old
		// ~/.grafeio root (see legacySessionsBase — never written to).
		b, err = os.ReadFile(filepath.Join(legacySessionsBase(), SessionDirHash(dir), "session.json"))
		if err != nil {
			return nil, false
		}
	}
	var sf SessionFile
	if err := json.Unmarshal(b, &sf); err != nil {
		return nil, false
	}
	if sf.Dir == "" {
		return nil, false
	}
	return &sf, true
}

// Fresh — young enough to offer on boot (4 days).
func (sf *SessionFile) Fresh() bool {
	return sf != nil && sf.SavedAt > 0 &&
		time.Since(time.UnixMilli(sf.SavedAt)) < sessionFreshWindow
}

// Snapshot builds the file body from the live office state, trimmed to the
// on-disk caps (chat last 200 / tasks + mails last 50 — machine trims, not
// NL). Pending placeholder bubbles ride along when present; the RESTORE
// side drops them (a restored "typing…" would pin the busy affordance
// forever). Boot-scoped notices (Meta bootNoticeMeta — the restore line)
// NEVER persist: each boot announces its own restore exactly once instead
// of accumulating one stale line per past boot.
func Snapshot(dir, primaryID string, st state.OfficeState) SessionFile {
	chat := make([]state.ChatMsg, 0, len(st.Chat))
	for _, c := range st.Chat {
		if c.Meta == bootNoticeMeta || c.Meta == bootWarnNoticeMeta {
			continue
		}
		chat = append(chat, c)
	}
	sf := SessionFile{
		Dir:       dir,
		PrimaryID: primaryID,
		Agents:    append([]state.Employee(nil), st.Employees...),
		Tasks:     append([]state.BoardTask(nil), st.Tasks...),
		Mails:     append([]state.MailItem(nil), st.Mails...),
		Chat:      chat,
	}
	if len(sf.Chat) > sessionChatCap {
		sf.Chat = sf.Chat[len(sf.Chat)-sessionChatCap:]
	}
	if len(sf.Tasks) > sessionListCap {
		sf.Tasks = sf.Tasks[len(sf.Tasks)-sessionListCap:]
	}
	if len(sf.Mails) > sessionListCap {
		sf.Mails = sf.Mails[len(sf.Mails)-sessionListCap:]
	}
	return sf
}

// SaveSession writes the snapshot atomically: a UNIQUE tmp file per writer
// (pid+nsec), then rename — concurrent theboringoffice instances in the same
// directory cannot tear the file; last rename wins.
func SaveSession(dir string, sf SessionFile) error {
	path := SessionPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	sf.SavedAt = time.Now().UnixMilli() // always-latest-wins stamp
	b, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(path),
		fmt.Sprintf(".session-%d-%d.tmp", os.Getpid(), time.Now().UnixNano()))
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// RestoreNotice — the dim office notice on a restored boot. Trails the
// resume discoverability hint: /session prints this office's primary id and
// -s|--session pins one explicitly on the next boot (bypassing the
// freshness gate — deliberate resume semantics).
func RestoreNotice(sf *SessionFile) string {
	return fmt.Sprintf("restored office session from %s (%d msgs) · /new for a fresh office · /session prints the id (-s|--session pins one at boot)",
		time.UnixMilli(sf.SavedAt).Local().Format("15:04"), len(sf.Chat))
}

// sessionInfo — the /session slash body: the current primary ("boss")
// session id, where this directory's session.json lives, and the exact
// command that resumes THIS session on a future boot. The id reads "" while
// the backend has not resolved a primary yet (or a stub backend without
// the seam — harnesses, demo) — reported honestly, never invented.
func (m *Model) sessionInfo() string {
	id := ""
	if ps, ok := m.backend.(primarySeamBackend); ok {
		id = ps.PrimaryID()
	}
	if id == "" {
		return "session: no primary resolved yet (try again once the backend is up) · -s|--session <id> pins one at boot"
	}
	path := "n/a (no session dir)"
	if m.sessDir != "" {
		path = SessionPath(m.sessDir)
	}
	return fmt.Sprintf("session: %s (primary)\n  session.json: %s\n  resume on the next boot: theboringoffice -s %s", id, path, id)
}

// NewOfficeNotice — the office notice for /new (the file is KEPT: history
// is available on the next boot until the new session's own persist
// overwrites it — always-latest-wins).
const NewOfficeNotice = "new office spawned · previous transcript archived (kept on disk)"

// --- backend seams (additive; type-asserted, never added to state.Backend)

// primarySeamBackend — office-session boss routing on the live backend.
// PrimaryOverride is called BEFORE Start so the saved primary wins; a
// server-side 404/fetch failure falls back to find-or-create (never a hard
// boot failure). PrimaryID feeds the snapshot. Harness stubs (uishot,
// headless demo) never implement this seam.
type primarySeamBackend interface {
	PrimaryOverride(id string)
	PrimaryID() string
}

// officeSpawnBackend — the /new seam: mint a fresh "theboringoffice office" primary
// NOW (not lazily on the next send). The old session is un-seated, never
// deleted server-side.
type officeSpawnBackend interface {
	NewOffice() (string, error)
}

// --- model wiring ------------------------------------------------------------

// hydrateSession — the restore leg of app.New (live mode only): transcript
// back into the chat, the roster back as SILENT hires — no dispatch events,
// no task assignment: the freshly spawned server does not know these child
// sessions, so they return as seated, idle desks — board + mail back, then
// the dim restore notice.
func (m *Model) hydrateSession(sf *SessionFile) {
	// Transcript: drop Pending:true entries — they are bubbles of a turn
	// the previous process died inside; restoring one would show a stuck
	// "typing…" placeholder that nothing will ever complete. Drop legacy
	// restore notices too (From office + the restore prefix): files
	// written before the boot-notice dedupe carry one plain row per past
	// boot — hydrating them would stack a stale "restored …" pile above
	// this boot's own single line, so the file self-cleans here (the
	// fixed Snapshot never writes them back).
	chat := make([]state.ChatMsg, 0, len(sf.Chat))
	for _, c := range sf.Chat {
		if c.Pending {
			continue
		}
		if c.From == "office" && strings.HasPrefix(c.Text, restoreNoticePrefix) {
			continue
		}
		// legacy wedge self-clean: prefixes committed before the wedge row
		// became boot-scoped (boot-warn) would otherwise print forever.
		if c.From == "office" && strings.HasPrefix(c.Text, "[theboringoffice] boss turn wedged") {
			continue
		}
		chat = append(chat, c)
	}
	m.st.Chat = chat
	// Roster: the fixed seats (manager, hr) are re-seated by the backend's
	// own Start hires / the initial state — and so is the CTO's exec
	// suite: live Start seats the idle pseudo-CTO there deterministically
	// every boot, while a restored CTO row would pin seat "cto" to a
	// child session the server deleted after its return (a dead desk
	// that blocks every future swap). Every other agent returns as a
	// SILENT hire — seated at their old desk, no dispatch, no task.
	for _, e := range sf.Agents {
		if e.Role == state.RoleManager || e.Role == state.RoleHR || e.Role == state.RoleCTO {
			continue
		}
		if findEmployee(m.st, e.ID) != nil {
			continue
		}
		e.Task = ""
		e.Sprite = state.SpriteAtDesk
		m.st.Employees = append(m.st.Employees, e)
	}
	m.st.Tasks = append([]state.BoardTask(nil), sf.Tasks...)
	m.st.Mails = append([]state.MailItem(nil), sf.Mails...)
	// The plan editor's drafted-but-unapproved buffer comes back too —
	// the MODE does not (a boot always lands in build; ctrl+p re-enters).
	// F2: a restored buffer latches m.restoredPlan — it is NOT approvable
	// untouched (the member must open it, edit, then ctrl+x twice).
	if m.plan != nil && sf.PlanText != "" {
		m.plan.SetValue(sf.PlanText)
		m.restoredPlan = true
	}
	m.tabs.SetState(m.st)
	// the restore line is BOOT-SCOPED (Meta bootNoticeMeta): it renders
	// now but Snapshot strips it on every persist — exactly one restore
	// line per boot on screen, zero in the file on subsequent cycles.
	m.appendNotice(RestoreNotice(sf), bootNoticeMeta)
}

// persistOfficeSession snapshots the office (LIVE mode ONLY — the demo
// floor is a scripted tour; persisting it would fake a real transcript on
// the next boot) and writes ~/.theboringoffice/sessions/<dirhash>/session.json.
//
//   - force=false — the cheap-write loop (one EvTick check per render
//     cycle, throttled to sessionWriteMinGap): the disk write itself runs
//     on a goroutine — the UI NEVER blocks on save.
//   - force=true  — the quit path: SYNCHRONOUS, because an async write
//     would lose the race with process exit. Bounded by the on-disk caps
//     (~200 chat + ~50 + ~50 entries), so it is small and fast.
func (m *Model) persistOfficeSession(force bool) {
	if m.st.Mode != state.ModeLive || m.sessDir == "" {
		return
	}
	if !force && time.Since(m.sessLast) < sessionWriteMinGap {
		return
	}
	m.sessLast = time.Now()
	primaryID := ""
	if ps, ok := m.backend.(primarySeamBackend); ok {
		primaryID = ps.PrimaryID()
	}
	dir := m.sessDir
	sf := Snapshot(dir, primaryID, m.st)
	sf.PlanText = m.planText() // plan editor buffer, "" when pristine
	if force {
		_ = SaveSession(dir, sf) // quit path — best effort, bounded + sync
		return
	}
	go func() { _ = SaveSession(dir, sf) }() // async — UI never blocks on disk
}

// PersistSession — the exported FINAL-write hook for the runtime shutdown
// path (cmd/theboringoffice calls it after p.Run() alongside b.Stop; harnesses may
// call it directly). No-op in demo mode.
func (m *Model) PersistSession() { m.persistOfficeSession(true) }

// persistOfficePin — the /session accept's final write (the exec-replace
// contract): persistOfficeSession's quit-path twin but stamped with the
// ACCEPTED id as the primary. The plain call would stamp the backend's
// STILL-current id (nothing moves server-side here — the swap is the
// relaunched `-s <id>` boot's job); stored id == boot pin is exactly
// what lets the new process hydrate the transcript (the closing notice
// row included) straight through the swap. Same guarantees: live-only,
// sessDir-guarded, synchronous + bounded.
func (m *Model) persistOfficePin(primaryID string) {
	if m.st.Mode != state.ModeLive || m.sessDir == "" {
		return
	}
	m.sessLast = time.Now()
	sf := Snapshot(m.sessDir, primaryID, m.st)
	sf.PlanText = m.planText()     // plan editor buffer, "" when pristine
	_ = SaveSession(m.sessDir, sf) // quit path — best effort, bounded + sync
}

// PrimarySessionID — the office's current primary ("boss") session id,
// read off the primarySeamBackend seam exactly like sessionInfo does.
// "" in demo mode, before the backend resolves one, or on seam-less
// harness stubs — callers that PRINT a resume line must say nothing
// then (an id is never invented). cmd/theboringoffice's clean-exit path
// reads it.
func (m *Model) PrimarySessionID() string {
	if ps, ok := m.backend.(primarySeamBackend); ok {
		return ps.PrimaryID()
	}
	return ""
}

// newOffice — the /new slash handler: clear the local surfaces, reset the
// primary hold (ResetPrimary(true) semantics), then mint a BRAND-NEW
// "theboringoffice office" session NOW via the additive seam (the old server-side
// session is only un-seated — never deleted; the on-disk transcript is NOT
// deleted either, the new session's next cheap-write overwrites it:
// always-latest-wins). In demo mode only the local clear happens (no
// server-side session exists).
func (m *Model) newOffice() {
	m.st.Chat = nil
	m.st.Tasks = nil
	m.st.Mails = nil
	m.st.Bubbles = nil
	m.st.BossThinking = false
	m.st.BossDelegating = false
	// the older-history walk dies with the office: a minted-fresh primary
	// gets a fresh pager (the next event re-seeds it — pagerKick), and an
	// in-flight hop from the OLD session dies on its sid guard.
	m.resetPager()
	if m.chat != nil {
		m.chat.ClearAttachments() // staged chips die with the office
	}
	if m.plan != nil {
		m.plan.SetValue(m.planTemplate) // the new office drafts from a fresh canvas
	}
	m.restoredPlan = false // /new's canvas is fresh — nothing "restored"
	m.planSendPending = 0  // and no prior turn's completion follows us in
	if tb, ok := m.team(); ok {
		// Hold is cleared first: NewOffice resolves the fresh session
		// eagerly, so the respawn latch cannot survive into a later Send.
		_ = tb.ResetPrimary(true)
	}
	if ob, ok := m.backend.(officeSpawnBackend); ok && m.st.Mode == state.ModeLive {
		if _, err := ob.NewOffice(); err != nil {
			m.noticeErr("/new: fresh office session failed: " + err.Error())
			return
		}
	}
	m.tabs.SetState(m.st)
	m.notice(NewOfficeNotice)
}
