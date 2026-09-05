package control

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func controlTestDir(t *testing.T) string {
	t.Helper()
	controlTestHome(t)
	return filepath.Join(t.TempDir(), "project")
}

func controlTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("THEFLOOR_HOME", t.TempDir())
}

func TestNewToken(t *testing.T) {
	controlTestHome(t)
	first, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 64 || len(second) != 64 || first == second {
		t.Fatalf("tokens = %q, %q", first, second)
	}
}

func TestDirHashMatchesSessionAlgorithm(t *testing.T) {
	controlTestHome(t)
	const dir = "/tmp/theboringfloor-control-known"
	const want = "0d141a85955c463be18af11e0a92ae554e81804b"
	if got := DirHash(dir); got != want {
		t.Fatalf("DirHash(%q) = %q, want %q", dir, got, want)
	}
}

func TestDiscoveryRoundTripAndRemove(t *testing.T) {
	dir := controlTestDir(t)
	want := Discovery{PID: os.Getpid(), Port: 54321, Token: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Dir: dir, StartedAt: 1788619819000, Version: "0.3.27", BootID: "boot-id"}
	if err := WriteDiscovery(dir, want); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(ControlPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	got, ok := ReadDiscovery(dir)
	if !ok || got != want {
		t.Fatalf("ReadDiscovery() = %#v, %t; want %#v, true", got, ok, want)
	}
	if err := RemoveDiscovery(dir); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadDiscovery(dir); ok {
		t.Fatal("ReadDiscovery after removal returned ok")
	}
	if err := RemoveDiscovery(dir); err != nil {
		t.Fatal(err)
	}
}

func TestReadDiscoveryRejectsMissingAndCorruptFiles(t *testing.T) {
	dir := controlTestDir(t)
	if _, ok := ReadDiscovery(dir); ok {
		t.Fatal("missing discovery returned ok")
	}
	if err := os.MkdirAll(filepath.Dir(ControlPath(dir)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ControlPath(dir), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadDiscovery(dir); ok {
		t.Fatal("corrupt discovery returned ok")
	}
}

func TestDiscoveryStaleDeadPID(t *testing.T) {
	controlTestHome(t)
	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if !(Discovery{PID: cmd.ProcessState.Pid(), Port: 1, Token: "token"}.Stale()) {
		t.Fatal("exited child was not stale")
	}
	if _, known := processExecutableName(os.Getpid()); known && !(Discovery{PID: os.Getpid(), Port: 1, Token: "token"}.Stale()) {
		t.Fatal("non-theboringfloor current process was not stale")
	}
}

func TestPruneStaleRemovesDeadRecord(t *testing.T) {
	dir := controlTestDir(t)
	if err := WriteDiscovery(dir, Discovery{PID: -1, Port: 1, Token: "token"}); err != nil {
		t.Fatal(err)
	}
	if !PruneStale(dir) {
		t.Fatal("PruneStale() = false, want true")
	}
	if _, ok := ReadDiscovery(dir); ok {
		t.Fatal("stale discovery remained after pruning")
	}
}

func TestPruneStaleLeavesLiveRecord(t *testing.T) {
	dir := controlTestDir(t)
	cmd := startOfficeNamedProcess(t)
	if err := WriteDiscovery(dir, Discovery{PID: cmd.Process.Pid, Port: 1, Token: "token"}); err != nil {
		t.Fatal(err)
	}
	if PruneStale(dir) {
		t.Fatal("PruneStale() = true for live record")
	}
	if _, ok := ReadDiscovery(dir); !ok {
		t.Fatal("live discovery was removed")
	}
}

func TestReadDiscoveryLegacyBootID(t *testing.T) {
	dir := controlTestDir(t)
	if err := os.MkdirAll(filepath.Dir(ControlPath(dir)), 0o755); err != nil {
		t.Fatal(err)
	}
	const legacy = `{"pid":1234,"port":54321,"token":"token","dir":"/project","startedAt":1,"version":"0.3.27"}`
	if err := os.WriteFile(ControlPath(dir), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	d, ok := ReadDiscovery(dir)
	if !ok || d.BootID != "" {
		t.Fatalf("ReadDiscovery() = %#v, %t; want legacy record with empty BootID", d, ok)
	}
}

func TestRegistryFulfillBeforeAndAfterWait(t *testing.T) {
	controlTestHome(t)
	r := NewRegistry()
	id, reply := r.NewRequest()
	if !r.Fulfill(id, []byte("before")) {
		t.Fatal("Fulfill before wait returned false")
	}
	if got := string(<-reply); got != "before" {
		t.Fatalf("reply = %q", got)
	}

	id, reply = r.NewRequest()
	go func() { r.Fulfill(id, []byte("after")) }()
	select {
	case got := <-reply:
		if string(got) != "after" {
			t.Fatalf("reply = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for reply")
	}
}

func TestRegistryCancel(t *testing.T) {
	controlTestHome(t)
	r := NewRegistry()
	id, _ := r.NewRequest()
	r.Cancel(id)
	if r.Fulfill(id, []byte("nope")) {
		t.Fatal("Fulfill after Cancel returned true")
	}
}

func TestRegistryConcurrentRequests(t *testing.T) {
	controlTestHome(t)
	const workers = 50
	r := NewRegistry()
	ids := make(chan string, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, reply := r.NewRequest()
			ids <- id
			if !r.Fulfill(id, []byte(id)) {
				t.Errorf("Fulfill(%q) returned false", id)
				return
			}
			if got := string(<-reply); got != id {
				t.Errorf("reply = %q, want %q", got, id)
			}
		}()
	}
	wg.Wait()
	close(ids)
	seen := make(map[string]bool, workers)
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate request ID %q", id)
		}
		seen[id] = true
	}
}

func TestDiscoveryJSONShape(t *testing.T) {
	controlTestHome(t)
	b, err := json.Marshal(Discovery{PID: 1234, Port: 54321, Token: "token", Dir: "/project", StartedAt: 1788619819000, Version: "0.3.27", BootID: "boot-id"})
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"pid":1234,"port":54321,"token":"token","dir":"/project","startedAt":1788619819000,"version":"0.3.27","bootId":"boot-id"}`
	if string(b) != want {
		t.Fatalf("JSON = %s, want %s", b, want)
	}
}

func startOfficeNamedProcess(t *testing.T) *exec.Cmd {
	t.Helper()
	if os.Getenv("GO_WANT_CONTROL_HELPER") == "1" {
		select {}
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(t.TempDir(), "theboringfloor")
	b, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(copyPath, b, 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(copyPath, "-test.run=TestControlHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_CONTROL_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	return cmd
}

func TestControlHelperProcess(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	if os.Getenv("GO_WANT_CONTROL_HELPER") == "1" {
		select {}
	}
}
