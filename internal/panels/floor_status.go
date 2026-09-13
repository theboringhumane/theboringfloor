package panels

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/control"
)

// FloorState is what the floor panels know about ANOTHER floor's live
// office: whether it currently answers a health probe, and if so, whether
// its own control server reports work in flight or is parked waiting on
// the member (a permission prompt or a boss question). A probe failure of
// any kind (no discovery record, unreachable office, wrong token,
// malformed response) collapses to the zero value: it renders identically
// to a floor that was never started. This is a product decision, not an
// oversight — an unreachable floor is not an error worth surfacing on a
// list of projects the member may not even be using right now.
//
// NeedsYou must never be true unless Live is also true — the shared
// render helpers (floorGlyph/floorStatusLabel) enforce this at read time
// regardless of how the field got set, so a stale or malformed status
// entry can never badge an offline floor as blocked on the member.
type FloorState struct {
	Live     bool
	Busy     bool
	NeedsYou bool
}

// FloorStatus maps a floor's canonical directory hash (control.DirHash, NOT
// its display name or raw Dir string) to its last known FloorState. It is
// the single shared type both floors.go and floor_nav.go read, so the two
// renderers can never drift into showing different states for the same
// floor. A nil or empty FloorStatus renders every floor as not-running —
// this is exactly the correct state before the first sweep has returned.
type FloorStatus map[string]FloorState

// FloorStatusMsg carries one completed status sweep back to the Bubble Tea
// update loop. It is exported (unlike floorsLoaded/ticketsLoaded, which
// stay package-private) because internal/app must name this type directly
// in its top-level Update switch to route the message into (*Floors).Update
// regardless of which panel currently has focus — the persistent floor
// navigator rail needs it even when the Floors panel itself is not active.
type FloorStatusMsg struct {
	Status FloorStatus
}

const (
	// FloorStatusInterval is how often the floor-status sweep re-arms.
	// Exported so internal/app's periodic tick and this package's probe
	// command share one source of truth instead of two copies of "5s".
	FloorStatusInterval = 5 * time.Second

	// floorStatusSweepBudget bounds one entire sweep (every floor, not each
	// individual probe) so a hung or slow-DNS office can never stall the
	// refresh loop past this ceiling. It is wired as the context deadline
	// shared by every HTTP request the sweep issues.
	floorStatusSweepBudget = 2 * time.Second

	// floorStatusWorkers bounds sweep concurrency. A member's floor list is
	// small in practice (a handful to a few dozen project directories, not
	// thousands), and each probe is at most two sequential HTTP round trips
	// to 127.0.0.1 (health, then busy) inside the shared 2s budget. Eight
	// workers keeps a sweep of dozens of floors comfortably inside that
	// budget without opening one goroutine per floor — the same shape as
	// internal/projects' own probeProjects, which bounds itself at 16 for a
	// potentially larger, unfiltered project root.
	floorStatusWorkers = 8
)

// probeOneFloor reports the live/busy state of a single floor directory. It
// never returns an error: every failure mode reports FloorState{} (not
// live), matching the product decision that transport errors are invisible
// in this list.
func probeOneFloor(ctx context.Context, client *http.Client, dir string) FloorState {
	discovery, ok := control.ReadDiscovery(dir)
	if !ok || discovery.Port <= 0 || discovery.Token == "" || discovery.Dir == "" {
		return FloorState{}
	}
	base := "http://127.0.0.1:" + strconv.Itoa(discovery.Port)
	var health control.HealthResponse
	if !floorStatusFetch(ctx, client, base+control.RouteHealth, discovery.Token, &health) {
		return FloorState{}
	}
	if !health.OK || health.Dir != discovery.Dir {
		return FloorState{}
	}
	var busy control.BusyResponse
	// A busy fetch failure still means the office is live and reachable —
	// only Busy/NeedsYou stay at their zero value (not working, not
	// parked), never Live.
	floorStatusFetch(ctx, client, base+control.RouteBusy, discovery.Token, &busy)
	return FloorState{Live: true, Busy: busy.Busy, NeedsYou: busy.QuestionParked}
}

// floorStatusFetch performs one authenticated GET and decodes its JSON body
// into out. It reports false on any transport, status, or decode failure;
// callers treat false uniformly as "could not confirm this state".
func floorStatusFetch(ctx context.Context, client *http.Client, url, token string, out any) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return json.NewDecoder(resp.Body).Decode(out) == nil
}

// sweepFloorStatus probes every directory in dirs concurrently, bounded by
// floorStatusWorkers, and returns once every probe has finished or ctx's
// deadline expires (whichever comes first — a probe still running when ctx
// is cancelled reports as not-live for that entry, never blocking the
// sweep past its budget).
func sweepFloorStatus(ctx context.Context, dirs []string) FloorStatus {
	status := make(FloorStatus, len(dirs))
	if len(dirs) == 0 {
		return status
	}
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport:     &http.Transport{Proxy: nil},
	}
	jobs := make(chan string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range min(floorStatusWorkers, len(dirs)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dir := range jobs {
				state := probeOneFloor(ctx, client, dir)
				mu.Lock()
				status[control.DirHash(dir)] = state
				mu.Unlock()
			}
		}()
	}
	for _, dir := range dirs {
		jobs <- dir
	}
	close(jobs)
	wg.Wait()
	return status
}

// StatusSweepCmd returns a tea.Cmd that probes every OTHER known floor
// (never the current one — we already know our own state, and probing
// ourselves would be silly) and delivers a FloorStatusMsg. The probe itself
// runs inside the returned closure, off the Bubble Tea update goroutine, so
// building this Cmd never blocks on network or filesystem work.
//
// Callers are responsible for overlap suppression: do not invoke this again
// while a previous sweep's FloorStatusMsg is still outstanding.
func (f *Floors) StatusSweepCmd() tea.Cmd {
	dirs := make([]string, 0, len(f.rows))
	for _, r := range f.rows {
		if r.Dir == f.dir {
			continue
		}
		dirs = append(dirs, r.Dir)
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), floorStatusSweepBudget)
		defer cancel()
		return FloorStatusMsg{Status: sweepFloorStatus(ctx, dirs)}
	}
}

// StatusOf returns the last known state for the floor at dir. A dir that
// was never probed (first frame, before any sweep has returned) reports the
// zero FloorState, which renders identically to "not running" — callers
// must not try to distinguish "never probed" from "confirmed not running".
func (f *Floors) StatusOf(dir string) FloorState {
	if f.status == nil {
		return FloorState{}
	}
	return f.status[control.DirHash(dir)]
}

// floorGlyph returns the one-column marker glyph for a floor row plus the
// chrome renderer that should colour it, matching the four-state contract:
// the current floor keeps today's plain "●" (glyphColor nil — it is never
// remotely probed and never carries a remote badge); a live floor parked
// waiting on the member (blocked on a permission prompt or a boss
// question) gets a distinct "▲" in the existing warn colour (the same
// amber the topbar's bypass indicator already uses for "needs your
// attention"); a live+busy floor gets a distinct "◆" in the existing
// accent colour (the same ink cockpit.go already uses for "WORKING"
// employees); a live+idle floor gets a quieter "○" in the existing dim
// colour; an unreachable floor gets the original blank " " with no
// colour, unchanged from today.
//
// Precedence is strict and lives here, in the one place both panels share,
// so it can never drift: needs-you outranks working. A floor that is both
// busy and parked on a prompt renders as needs-you, because that is the
// state a member must act on — "working" is merely informational.
//
// glyphColor is nil for the current floor and for "not running" so callers
// can tell "needs no extra styling" apart from "needs this styling" without
// a second flag.
func floorGlyph(current bool, st FloorState) (glyph string, glyphColor func(...string) string) {
	switch {
	case current:
		return "●", nil
	case st.Live && st.NeedsYou:
		return "▲", chrome.PanelWarn.Render
	case st.Live && st.Busy:
		return "◆", chrome.PanelAccent.Render
	case st.Live:
		return "○", chrome.PanelDim.Render
	default:
		return " ", nil
	}
}

// floorStatusLabel returns the short text appended to a floor's existing
// "N teams · M tickets" line. Only needs-you and live+busy get a label per
// product decision — live+idle is conveyed by the quieter marker alone, and
// the current floor never carries a remote badge at all. The precedence
// mirrors floorGlyph exactly (needs-you outranks working) so the marker and
// its label can never say two different things about the same row.
func floorStatusLabel(current bool, st FloorState) string {
	switch {
	case current || !st.Live:
		return ""
	case st.NeedsYou:
		return " · needs you"
	case st.Busy:
		return " · working"
	default:
		return ""
	}
}
