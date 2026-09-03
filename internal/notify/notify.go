// Package notify — OS desktop notifications for the office: a look-away
// nudge when a permission ask blocks the floor or the boss's turn lands
// while the terminal is unfocused.
//
// Zero NEW dependencies, same shape as internal/sound/player.go: a resolved
// platform binary, fire-and-forget exec, and total fail-silence (no binary,
// a non-zero exit, or a wedged process all degrade to nothing — the office
// must never break because a notification daemon did).
//
//   - darwin: osascript -e 'display notification "<body>" with title "<title>" sound name "Glass"'
//   - linux:  notify-send <title> <body>  ("permission" kind rides -u critical)
//   - other / not found: resolve to "" — every Send is a graceful no-op.
//
// CLICK-TO-FOCUS IS NOT AVAILABLE without new deps: AppleScript's `display
// notification` carries NO click action (Notification Center click actions
// need a bundled .app + UNUserNotificationCenter), and notify-send action
// hints are desktop-environment flavored. The banner is a pure heads-up —
// the member's way back stays the terminal bell + the in-app [blocked]
// surfacing (the sound bus, untouched).
package notify

import (
	"fmt"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/brand"
)

// minGap — the same kind re-pinged within this window is suppressed, so a
// burst of office events can't stack banners into mush (the sound bus's
// throttleGap twin).
const minGap = 400 * time.Millisecond

// capWindow / capMax — per-kind rolling budget: at most capMax banners of
// ONE kind per capWindow. OS banners live ~5s apiece, so a wedged emitter
// looping inside the office would churn Notification Center; the hooks
// above this bus already fire once-per-cohort/once-per-turn, so this is
// the backstop, not the policy.
const (
	capWindow = 30 * time.Second
	capMax    = 8
)

// watchdog — how long an outlived notifier process may run before we kill
// it (the sound player's twin — a wedged osascript must not leak past this).
const watchdog = 2 * time.Second

// Bus routes look-away nudges to the OS notification daemon. Modes:
//
//	"on"  — fire through the resolved platform notifier
//	"off" — silence
//
// Env override: THEBORINGOFFICE_NO_NOTIFY=1 (pre-rename: GRAFEIO_NO_NOTIFY=1)
// forces "off" above whatever config asked for.
type Bus struct {
	mode     string
	notifier string

	mu   sync.Mutex
	last map[string]time.Time // kind → last fired (minGap clamp)
	caps map[string]kindCap   // kind → rolling capWindow budget
	now  func() time.Time     // test seam; defaults to time.Now

	run func(bin string, args ...string) // exec seam; defaults to spawn
}

// kindCap — one kind's rolling-window budget counter.
type kindCap struct {
	start time.Time // window opened here (rolls after capWindow)
	n     int       // banners fired inside it
}

// ResolveNotifier finds the platform notifier, or "" when none exists.
//   - darwin: osascript
//   - linux:  notify-send
//   - other:  none
func ResolveNotifier() string {
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{"osascript"}
	case "linux":
		candidates = []string{"notify-send"}
	default:
		return ""
	}
	for _, c := range candidates {
		if p, err := osexec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}

// NewBus builds a Bus for the config mode ("on"|"off"|""). "" defaults to
// "on"; anything unrecognized turns OFF — a brain.json typo must never
// spam the desk. The notifier lookup happens once here; an empty
// resolution keeps every Send a graceful no-op.
func NewBus(cfgNotifications string) *Bus {
	mode := cfgNotifications
	if mode == "" {
		mode = "on"
	}
	switch mode {
	case "on", "off":
	default:
		mode = "off"
	}
	if brand.Get("NO_NOTIFY") == "1" {
		mode = "off"
	}
	b := &Bus{
		mode:     mode,
		notifier: ResolveNotifier(),
		last:     make(map[string]time.Time),
		caps:     make(map[string]kindCap),
		now:      time.Now,
	}
	b.run = b.spawn
	return b
}

// Mode returns the effective mode after env overrides and any live SetMode.
func (b *Bus) Mode() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.mode
}

// NotifierPath returns the resolved notifier command path, or "" if none.
func (b *Bus) NotifierPath() string { return b.notifier }

// SetMode flips the mode live (/notify's SetMode seam — /power's live
// governor twin). Unknown values are refused so a busted toggle can never
// silently mute or unmute the desk.
func (b *Bus) SetMode(mode string) {
	switch mode {
	case "on", "off":
	default:
		return
	}
	b.mu.Lock()
	b.mode = mode
	b.mu.Unlock()
}

// Send fires one banner without blocking the caller. It never errors and
// never panics: mode off, no notifier resolved, a throttled kind, or a
// spawn failure all degrade silently (the office is the product, the
// banner is the garnish).
func (b *Bus) Send(kind, title, body string) {
	if b.Mode() == "off" || b.notifier == "" {
		return
	}
	if b.throttled(kind) {
		return
	}
	switch filepath.Base(b.notifier) {
	case "notify-send":
		var args []string
		if kind == "permission" {
			// a blocked floor is THE ping worth the urgency bump: dbus
			// critical keeps it on glass until the member dismisses it.
			args = append(args, "-u", "critical")
		}
		b.run(b.notifier, append(args, title, body)...)
	default:
		// darwin (osascript) — the default branch so any resolved-but-
		// unexpected notifier still lands the AppleScript shape.
		script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`,
			escapeApplescript(body), escapeApplescript(title))
		b.run(b.notifier, "-e", script)
	}
}

// escapeApplescript quotes \ and " for embedding inside a double-quoted
// AppleScript string literal.
func escapeApplescript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

// throttled reports whether kind is over budget right now and, when not,
// stamps it. Two clamps: the 400ms burst gap and the rolling capWindow cap.
// Callers hold the mutex-free contract via b.mu.
func (b *Bus) throttled(kind string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := b.now()
	if t, ok := b.last[kind]; ok && now.Sub(t) < minGap {
		return true
	}
	c := b.caps[kind]
	if now.Sub(c.start) >= capWindow {
		c = kindCap{start: now} // the window rolled: fresh budget
	}
	if c.n >= capMax {
		b.caps[kind] = c
		return true
	}
	c.n++
	b.caps[kind] = c
	b.last[kind] = now
	return false
}

// spawn launches the notifier without waiting and arms a watchdog so a
// hung one never leaks past `watchdog`. Callers hold no locks.
func (b *Bus) spawn(bin string, args ...string) {
	cmd := osexec.Command(bin, args...)
	if err := cmd.Start(); err != nil {
		return
	}
	go func() {
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(watchdog):
			_ = cmd.Process.Kill()
			<-done
		}
	}()
}
