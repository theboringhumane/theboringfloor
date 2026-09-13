package panels

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

func setupFloorStatusHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("THEFLOOR_HOME", home)
	return home
}

func writeFloorDiscovery(t *testing.T, dir string, discovery control.Discovery) {
	t.Helper()
	if err := control.WriteDiscovery(dir, discovery); err != nil {
		t.Fatal(err)
	}
}

// floorStatusServer answers health with dir/OK and busy with the given
// response, both gated on token, mirroring what a real office's control
// server exposes at RouteHealth and RouteBusy.
func floorStatusServer(t *testing.T, dir, token string, delay time.Duration, busy control.BusyResponse) *http.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		switch r.URL.Path {
		case control.RouteHealth:
			_ = json.NewEncoder(w).Encode(control.HealthResponse{OK: true, Dir: dir, Version: "v-test", Backend: "opencode"})
		case control.RouteBusy:
			_ = json.NewEncoder(w).Encode(busy)
		default:
			http.NotFound(w, r)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	server.Addr = listener.Addr().String()
	return server
}

// floorStatusBlockingServer blocks every health request on release, tracking
// active/peak concurrent health requests under mu, so a test can assert the
// sweep never exceeds floorStatusWorkers in-flight probes.
func floorStatusBlockingServer(t *testing.T, dir, token string, mu *sync.Mutex, active, peak *int, release <-chan struct{}) *http.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case control.RouteHealth:
			mu.Lock()
			*active++
			if *active > *peak {
				*peak = *active
			}
			mu.Unlock()
			<-release
			mu.Lock()
			*active--
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(control.HealthResponse{OK: true, Dir: dir, Version: "v-test", Backend: "opencode"})
		case control.RouteBusy:
			_ = json.NewEncoder(w).Encode(control.BusyResponse{})
		default:
			http.NotFound(w, r)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	server.Addr = listener.Addr().String()
	return server
}

func floorStatusServerPort(t *testing.T, server *http.Server) int {
	t.Helper()
	_, port, err := net.SplitHostPort(server.Addr)
	if err != nil {
		t.Fatal(err)
	}
	value, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestSweepFloorStatusKeysByDirHashNotDisplayName(t *testing.T) {
	setupFloorStatusHome(t)
	dirA := "/work/team-a/same-name"
	dirB := "/work/team-b/same-name" // shares a display name (basename) with dirA
	serverA := floorStatusServer(t, dirA, "token-a", 0, control.BusyResponse{Busy: false})
	writeFloorDiscovery(t, dirA, control.Discovery{PID: 1, Port: floorStatusServerPort(t, serverA), Token: "token-a", Dir: dirA})
	// dirB has no discovery record at all -- it must not inherit dirA's
	// liveness merely because both floors would display the same name.
	status := sweepFloorStatus(context.Background(), []string{dirA, dirB})
	if got := status[control.DirHash(dirA)]; !got.Live {
		t.Fatalf("dirA should be live: %#v", got)
	}
	if got := status[control.DirHash(dirB)]; got.Live {
		t.Fatalf("dirB (same display name, no discovery) reported live: %#v", got)
	}
	if control.DirHash(dirA) == control.DirHash(dirB) {
		t.Fatal("test fixture bug: dirA and dirB hash the same")
	}
}

func TestSweepFloorStatusLiveBusyAndLiveIdle(t *testing.T) {
	setupFloorStatusHome(t)
	busyDir, idleDir := "/work/busy", "/work/idle"
	busyServer := floorStatusServer(t, busyDir, "b-token", 0, control.BusyResponse{Busy: true})
	idleServer := floorStatusServer(t, idleDir, "i-token", 0, control.BusyResponse{Busy: false})
	writeFloorDiscovery(t, busyDir, control.Discovery{PID: 1, Port: floorStatusServerPort(t, busyServer), Token: "b-token", Dir: busyDir})
	writeFloorDiscovery(t, idleDir, control.Discovery{PID: 1, Port: floorStatusServerPort(t, idleServer), Token: "i-token", Dir: idleDir})

	status := sweepFloorStatus(context.Background(), []string{busyDir, idleDir})
	if got := status[control.DirHash(busyDir)]; !got.Live || !got.Busy {
		t.Fatalf("busy floor = %#v, want live+busy", got)
	}
	if got := status[control.DirHash(idleDir)]; !got.Live || got.Busy {
		t.Fatalf("idle floor = %#v, want live and not busy", got)
	}
}

func TestSweepFloorStatusUnreachableFloorIsNotRunningNoError(t *testing.T) {
	setupFloorStatusHome(t)
	dir := "/work/never-registered"
	status := sweepFloorStatus(context.Background(), []string{dir})
	got := status[control.DirHash(dir)]
	if got.Live || got.Busy || got.NeedsYou {
		t.Fatalf("unreachable floor reported live/busy/needs-you: %#v (probe failures must render as not-running, never as an error)", got)
	}
}

// TestSweepFloorStatusPopulatesNeedsYouFromQuestionParked proves the probe
// carries control.BusyResponse.QuestionParked through into FloorState.NeedsYou,
// and that a floor which is busy AND parked still reports Busy true at the
// state layer -- the needs-you-outranks-working precedence is a rendering
// decision made once in floorGlyph/floorStatusLabel, not baked into the
// probed state itself.
func TestSweepFloorStatusPopulatesNeedsYouFromQuestionParked(t *testing.T) {
	setupFloorStatusHome(t)
	parkedDir := "/work/parked"
	busyParkedDir := "/work/busy-parked"
	parkedServer := floorStatusServer(t, parkedDir, "p-token", 0, control.BusyResponse{Busy: false, QuestionParked: true})
	busyParkedServer := floorStatusServer(t, busyParkedDir, "bp-token", 0, control.BusyResponse{Busy: true, QuestionParked: true})
	writeFloorDiscovery(t, parkedDir, control.Discovery{PID: 1, Port: floorStatusServerPort(t, parkedServer), Token: "p-token", Dir: parkedDir})
	writeFloorDiscovery(t, busyParkedDir, control.Discovery{PID: 1, Port: floorStatusServerPort(t, busyParkedServer), Token: "bp-token", Dir: busyParkedDir})

	status := sweepFloorStatus(context.Background(), []string{parkedDir, busyParkedDir})
	if got := status[control.DirHash(parkedDir)]; !got.Live || got.Busy || !got.NeedsYou {
		t.Fatalf("parked floor = %#v, want live+needs-you and not busy", got)
	}
	if got := status[control.DirHash(busyParkedDir)]; !got.Live || !got.Busy || !got.NeedsYou {
		t.Fatalf("busy+parked floor = %#v, want live+busy+needs-you (rendering precedence is applied later)", got)
	}
}

func TestSweepFloorStatusRespectsWorkerBound(t *testing.T) {
	setupFloorStatusHome(t)
	const n = 24
	if n <= floorStatusWorkers {
		t.Fatalf("test fixture bug: need more floors (%d) than floorStatusWorkers (%d) to prove the bound", n, floorStatusWorkers)
	}
	var mu sync.Mutex
	active, peak := 0, 0
	release := make(chan struct{})
	var dirs []string
	for i := range n {
		dir := "/work/slow-" + strconv.Itoa(i)
		dirs = append(dirs, dir)
		server := floorStatusBlockingServer(t, dir, "token", &mu, &active, &peak, release)
		writeFloorDiscovery(t, dir, control.Discovery{PID: 1, Port: floorStatusServerPort(t, server), Token: "token", Dir: dir})
	}
	done := make(chan FloorStatus, 1)
	go func() { done <- sweepFloorStatus(context.Background(), dirs) }()

	// Give every worker goroutine a chance to reach the handler and block
	// there before sampling how many are in flight at once.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		inFlight := active
		mu.Unlock()
		if inFlight >= min(floorStatusWorkers, n) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	inFlight := active
	mu.Unlock()
	if inFlight > floorStatusWorkers {
		t.Fatalf("in-flight probes = %d, want <= %d (floorStatusWorkers)", inFlight, floorStatusWorkers)
	}
	if inFlight == 0 {
		t.Fatal("no probes ever started -- test cannot prove the bound")
	}
	close(release)
	status := <-done
	if len(status) != n {
		t.Fatalf("status has %d entries, want %d", len(status), n)
	}
	mu.Lock()
	finalPeak := peak
	mu.Unlock()
	if finalPeak > floorStatusWorkers {
		t.Fatalf("peak concurrent probes = %d, want <= %d (floorStatusWorkers)", finalPeak, floorStatusWorkers)
	}
}

func TestStatusOfReturnsZeroStateForNilOrUnprobedDir(t *testing.T) {
	f := NewFloors(t.TempDir(), "opencode", true)
	// f.status is nil -- the state before the first sweep has ever
	// returned. It must render as "not running", never panic.
	if got := f.StatusOf("/some/dir"); got.Live || got.Busy {
		t.Fatalf("StatusOf with nil status map = %#v, want zero value", got)
	}
	f.status = FloorStatus{control.DirHash("/known"): {Live: true, Busy: true}}
	if got := f.StatusOf("/unprobed"); got.Live || got.Busy {
		t.Fatalf("StatusOf for a dir the sweep never reached = %#v, want zero value", got)
	}
	if got := f.StatusOf("/known"); !got.Live || !got.Busy {
		t.Fatalf("StatusOf for a known dir = %#v, want live+busy", got)
	}
}

func TestStatusSweepCmdExcludesCurrentFloor(t *testing.T) {
	setupFloorStatusHome(t)
	current := "/work/me"
	other := "/work/them"
	currentServer := floorStatusServer(t, current, "self-token", 0, control.BusyResponse{Busy: true})
	writeFloorDiscovery(t, current, control.Discovery{PID: 1, Port: floorStatusServerPort(t, currentServer), Token: "self-token", Dir: current})
	otherServer := floorStatusServer(t, other, "other-token", 0, control.BusyResponse{Busy: false})
	writeFloorDiscovery(t, other, control.Discovery{PID: 1, Port: floorStatusServerPort(t, otherServer), Token: "other-token", Dir: other})

	f := NewFloors(current, "opencode", false)
	f.rows = []workspace.Floor{{Dir: current, Name: "me"}, {Dir: other, Name: "them"}}

	msg := f.StatusSweepCmd()()
	swept, ok := msg.(FloorStatusMsg)
	if !ok {
		t.Fatalf("StatusSweepCmd() returned %#v, want FloorStatusMsg", msg)
	}
	// The current floor never shows a remote badge -- we already know our
	// own state, and probing ourselves would be silly -- so it must never
	// even appear in the swept status, even though its discovery record
	// would have answered a probe.
	if _, present := swept.Status[control.DirHash(current)]; present {
		t.Fatalf("current floor was probed even though it never should be: %#v", swept.Status)
	}
	if got := swept.Status[control.DirHash(other)]; !got.Live {
		t.Fatalf("other floor was not probed: %#v", got)
	}
}
