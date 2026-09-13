// floor_handoff.go — hands the floor currently open in THIS process off to
// a detached background office instead of killing it on a floor switch.
//
// Only the opencode backend is detach-safe today (see launchFloor's gate):
// an `opencode serve` child is proven to survive its spawner's death when
// nothing signals its process group (the whole point of
// isolateHandoffProcessGroup below), and a headless theboringfloor boots
// fine with no TTY, writes its discovery record, and answers its loopback
// control API. claude and codex are NOT proven safe to detach — claude's
// turn rides stdin/stdout pipes that die with this process, and codex's
// send() blocks synchronously inside the dying process — so they keep
// today's kill-on-switch behavior (launchFloor's other branch).
//
// handoffCurrentFloor is the entry point launchFloor calls, synchronously,
// BEFORE ever returning tea.Quit — so an aborted handoff truly leaves this
// office exactly as it was: nothing has been persisted-and-quit yet, the
// TUI just keeps running.
package app

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// handoffReadyTimeout bounds how long handoffCurrentFloor waits for the
// detached office to answer its own /v1/health before giving up and
// aborting the switch (decision: "bounded timeout (5 seconds)"). A var,
// not a const, so tests shrink it — same convention as main.go's
// stopDeadline.
var handoffReadyTimeout = 5 * time.Second

// handoffPollInterval paces the readiness poll inside handoffReadyTimeout.
var handoffPollInterval = 100 * time.Millisecond

// floorHandoffSpawn launches the detached office and returns a kill func to
// reap it if the handoff must be aborted (nil kill + nil error should never
// happen together; a non-nil error means nothing was started). serverURL,
// when non-empty, is the departing floor's own OWNED opencode serve (see
// state.ServerAttachable): the detached child attaches to it with
// --server instead of spawning a fresh one, so the in-flight turn survives
// the switch too, not just the floor. Empty means "no attach seam" (a
// non-opencode-attachable backend, or the serve could not be resolved) —
// the child falls back to today's spawn-its-own-serve boot. A package
// var, not a plain function call, so tests substitute a fake that never
// execs a real binary — the same seam convention as SpawnTerminal
// (model.go) and BackendFactory (model.go).
var floorHandoffSpawn = spawnDetachedOffice

// floorHandoffReady polls for the detached office's readiness. A package
// var for the same test-injection reason as floorHandoffSpawn — production
// tests must never depend on a real spawned process answering real HTTP.
var floorHandoffReady = waitForHandoffReady

// spawnDetachedOffice launches a new theboringfloor process pinned to dir
// and session, detached from THIS process's lifetime: own process group,
// /dev/null stdio, cwd=dir. Mirrors cmd/floorgate/gateway.go's
// newOfficeCommandForBinary (lines 86-99 — os.DevNull on all three stdio
// fds, command.Dir = dir, isolateOfficeProcessGroup) and
// process_unix.go/process_other.go's Setpgid split — reimplemented here
// (as isolateHandoffProcessGroup, floor_handoff_unix.go/
// floor_handoff_other.go) because cmd/floorgate is `package main` and
// cannot be imported.
func spawnDetachedOffice(dir, session, serverURL string) (kill func(), err error) {
	binary, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve own binary: %w", err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	defer null.Close()
	// --backend opencode is explicit, not inferred: handoffCurrentFloor is
	// only ever called while THIS process's backend already IS opencode
	// (launchFloor's gate), so there is no ambiguity to resolve — but a
	// detached child must never silently pick up a stale/different
	// brain.json default.
	args := []string{"--backend", "opencode"}
	if serverURL != "" {
		// Attach to the departing floor's own live serve (see main.go's
		// --server flag / backend.NewLive's resolution order) instead of
		// spawning a fresh one: the in-flight turn survives the switch,
		// not just the floor.
		args = append(args, "--server", serverURL)
	}
	if session != "" {
		args = append(args, "-s", session)
	}
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = null, null, null
	// THEFLOOR_NO_CONTROL is stripped so the detached office's control API
	// (and therefore its discovery record + /v1/health) is always up,
	// regardless of what this process's own environment carries — mirrors
	// gateway.go's withoutEnvironment call, same env var.
	cmd.Env = withoutHandoffControlEnv(os.Environ())
	isolateHandoffProcessGroup(cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait() // reap — never leave a zombie behind
		}
	}, nil
}

func withoutHandoffControlEnv(environment []string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if key != "THEFLOOR_NO_CONTROL" {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// waitForHandoffReady polls dir's discovery record + GET /v1/health until
// the detached office answers OK, or timeout elapses.
func waitForHandoffReady(dir string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if probeOfficeHealth(dir) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(handoffPollInterval)
	}
}

// probeOfficeHealth reads dir's discovery record and, if it names a live
// process, calls its loopback GET /v1/health with the record's bearer
// token (controlsrv.Server.authorized expects "Bearer <token>").
func probeOfficeHealth(dir string) bool {
	d, ok := control.ReadDiscovery(dir)
	if !ok || d.Stale() {
		return false
	}
	url := fmt.Sprintf("http://127.0.0.1:%d%s", d.Port, control.RouteHealth)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+d.Token)
	client := &http.Client{Timeout: 750 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// handoffCurrentFloor hands the CURRENTLY OPEN floor (m.sessDir, its
// current opencode primary session) to a detached background office before
// this process quits/execs into a different floor. It is the opencode-only
// alternative to the busy-refusal in launchFloor: the departing floor
// stays live and resumable instead of dying with this process.
//
// When the current backend implements state.ServerAttachable and its
// ServerURL() is non-empty, the detached child ATTACHES to that same
// running `opencode serve` (--server <url>) instead of spawning a fresh
// one — the in-flight turn survives too, not just the floor. The ordered
// sequence is strictly fail-safe: read ServerURL() -> spawn the detached
// child (with --server when available) -> wait for its health -> ONLY
// once healthy, call ReleaseServe() so this process stops owning the
// serve (its own later Stop() then skips signalling it — see
// liveBackend.Stop). If the child never comes up, ReleaseServe() is never
// called, the switch aborts, and the normal shutdown kills the serve
// exactly as it does today: an orphaned serve with no owner must never
// happen. A backend without the capability (or whose serve cannot be
// handed off — externally attached, already dead) simply falls back to
// today's spawn-a-fresh-serve behavior on the child.
//
// Sequencing around the discovery file matters: this process's OWN live
// discovery record for dir (written at boot, main.go) would make the
// child's own control.WriteDiscovery fail with ErrLiveDiscovery — our PID
// is still alive, so WriteDiscovery's staleness check never lets it steal
// the slot. The record is removed first to free it for the child, saved
// beforehand so a failed handoff can restore it verbatim: a genuinely
// unmodified "leave everything as it was" on abort, not just a notice.
//
// Returns true when the departing floor is confirmed healthy and the
// switch may proceed; false when it aborts (a notice is already shown,
// nothing about this office's state has changed).
func (m *Model) handoffCurrentFloor() bool {
	dir := m.sessDir
	if dir == "" {
		return true // nothing to hand off — proceed exactly as before
	}
	session := m.PrimarySessionID()
	attachable, canAttach := m.backend.(state.ServerAttachable)
	var serverURL string
	if canAttach {
		serverURL = attachable.ServerURL()
	}
	saved, hadDiscovery := control.ReadDiscovery(dir)
	if hadDiscovery {
		_ = control.RemoveDiscovery(dir)
	}
	kill, spawnErr := floorHandoffSpawn(dir, session, serverURL)
	ready := false
	if spawnErr == nil {
		ready = floorHandoffReady(dir, handoffReadyTimeout)
	}
	if ready {
		if serverURL != "" {
			// Only NOW — the detached office is confirmed healthy — may
			// this process stop owning the serve. A release before this
			// point risks the one outcome that must never happen: a
			// serve with no owner at all.
			attachable.ReleaseServe()
		}
		return true
	}
	if hadDiscovery {
		_ = control.WriteDiscovery(dir, saved) // restore — we are still live here
	}
	if kill != nil {
		kill()
	}
	reason := "the detached office never answered its health check"
	if spawnErr != nil {
		reason = spawnErr.Error()
	}
	m.noticeErr("Could not hand this floor to the background (" + reason + ") — staying here.")
	m.tabs.SetActive(0)
	return false
}
