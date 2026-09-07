package projects

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestRootHonorsHomeOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("THEFLOOR_HOME", home)
	if got, want := Root(), filepath.Join(home, ".theboringfloor", "projects"); got != want {
		t.Fatalf("Root() = %q, want %q", got, want)
	}
}

func TestIsSystemDir(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{name: "var root", dir: "/var", want: true},
		{name: "var descendant", dir: "/var/folders/temporary", want: true},
		{name: "private var root", dir: "/private/var", want: true},
		{name: "private var descendant", dir: "/private/var/folders/temporary", want: true},
		{name: "tmp root", dir: "/tmp", want: true},
		{name: "tmp descendant", dir: "/tmp/theboringfloor", want: true},
		{name: "private tmp root", dir: "/private/tmp", want: true},
		{name: "private tmp descendant", dir: "/private/tmp/theboringfloor", want: true},
		{name: "ordinary user path", dir: "/Users/x/Projects/foo", want: false},
		{name: "user var directory", dir: "/Users/x/var", want: false},
		{name: "varsity directory", dir: "/Users/x/varsity", want: false},
		{name: "varnish directory", dir: "/home/varnish", want: false},
		{name: "relative path", dir: "var/folders/temporary", want: false},
		{name: "empty path", dir: "", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsSystemDir(test.dir); got != test.want {
				t.Errorf("IsSystemDir(%q) = %t, want %t", test.dir, got, test.want)
			}
		})
	}
}

func TestListFiltersSystemProjects(t *testing.T) {
	home := setupHome(t)
	normalDir := "/Users/x/Projects/normal"
	varDirs := []string{
		"/var/folders/a/T/one",
		"/var/folders/a/T/two",
		"/var/folders/a/T/three",
	}
	writeSession(t, home, control.DirHash(normalDir), app.SessionFile{Dir: normalDir})
	for _, dir := range varDirs {
		writeSession(t, home, control.DirHash(dir), app.SessionFile{Dir: dir})
	}

	filtered, err := List(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if ids := projectIDs(filtered); !slices.Equal(ids, []string{control.DirHash(normalDir)}) {
		t.Fatalf("List() IDs = %v, want only %q", ids, control.DirHash(normalDir))
	}

	all, err := ListAll(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("ListAll() returned %d projects, want 4", len(all))
	}
}

func TestListIgnoresDotfilesAndNonDirectories(t *testing.T) {
	home := setupHome(t)
	writeSession(t, home, "alpha", app.SessionFile{Dir: "/work/alpha", SavedAt: 10})
	projectDir := filepath.Join(Root(), "alpha")
	writeFile(t, filepath.Join(projectDir, ".session-123-456.tmp"), []byte("not json at all"))
	writeFile(t, filepath.Join(Root(), "not-a-project"), []byte("not json at all"))

	got, err := List(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "alpha" {
		t.Fatalf("List() = %#v, want only alpha", got)
	}
}

func TestListToleratesMissingSession(t *testing.T) {
	setupHome(t)
	if err := os.MkdirAll(filepath.Join(Root(), "missing"), 0o755); err != nil {
		t.Fatal(err)
	}
	assertUnknownProject(t, "missing")
}

func TestListToleratesEmptySession(t *testing.T) {
	setupHome(t)
	writeFile(t, filepath.Join(Root(), "empty", "session.json"), nil)
	assertUnknownProject(t, "empty")
}

func TestListToleratesCorruptSession(t *testing.T) {
	setupHome(t)
	writeFile(t, filepath.Join(Root(), "corrupt", "session.json"), []byte(`{"dir":`))
	assertUnknownProject(t, "corrupt")
}

func TestListProbesHealthAndSorts(t *testing.T) {
	home := setupHome(t)
	server := healthServer(t, "/work/live", "live-token", 0, true)
	liveID := control.DirHash("/work/live")
	writeSession(t, home, liveID, app.SessionFile{Dir: "/work/live", Backend: "snapshot", PrimaryID: "p-live", SavedAt: 1, Chat: []state.ChatMsg{{}, {}}})
	writeDiscovery(t, "/work/live", control.Discovery{PID: 1, Port: serverPort(t, server), Token: "live-token", Dir: "/work/live"})
	newerID := control.DirHash("/work/newer")
	olderID := control.DirHash("/work/older")
	writeSession(t, home, newerID, app.SessionFile{Dir: "/work/newer", Backend: "opencode", PrimaryID: "p-new", SavedAt: 30})
	writeSession(t, home, olderID, app.SessionFile{Dir: "/work/older", Backend: "claudecode", PrimaryID: "p-old", SavedAt: 20})

	got, err := List(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if ids := projectIDs(got); !slices.Equal(ids, []string{liveID, newerID, olderID}) {
		t.Fatalf("sorted IDs = %v", ids)
	}
	if got[0] != (Project{ID: liveID, Dir: "/work/live", Name: "live", Live: true, Backend: "live-backend", PrimaryID: "p-live", Port: serverPort(t, server), Version: "v-live", SavedAt: 1, ChatCount: 2}) {
		t.Fatalf("live project = %#v", got[0])
	}
}

func TestListHealthRequiresAuthenticationAndMatchingDirectory(t *testing.T) {
	home := setupHome(t)
	server := healthServer(t, "/wrong", "token", 0, true)
	id := control.DirHash("/work/wrong")
	writeSession(t, home, id, app.SessionFile{Dir: "/work/wrong", Backend: "snapshot"})
	writeDiscovery(t, "/work/wrong", control.Discovery{PID: 1, Port: serverPort(t, server), Token: "token", Dir: "/work/wrong"})

	got, err := List(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Live || got[0].Port != 0 || got[0].Backend != "snapshot" {
		t.Fatalf("mismatched health marked live: %#v", got[0])
	}
}

func TestListProbesConcurrently(t *testing.T) {
	home := setupHome(t)
	for i := range 20 {
		dir := "/work/slow-" + strconv.Itoa(i)
		server := healthServer(t, dir, "token", 100*time.Millisecond, true)
		writeSession(t, home, control.DirHash(dir), app.SessionFile{Dir: dir})
		writeDiscovery(t, dir, control.Discovery{PID: 1, Port: serverPort(t, server), Token: "token", Dir: dir})
	}
	start := time.Now()
	got, err := List(context.Background(), time.Second)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 20 || elapsed >= 500*time.Millisecond {
		t.Fatalf("List() returned %d projects in %s, want 20 in under 500ms", len(got), elapsed)
	}
}

func TestGetAndDiscovery(t *testing.T) {
	home := setupHome(t)
	server := healthServer(t, "/work/live", "secret", 0, true)
	discovery := control.Discovery{PID: 1, Port: serverPort(t, server), Token: "secret", Dir: "/work/live", Version: "record-version"}
	id := control.DirHash("/work/live")
	writeSession(t, home, id, app.SessionFile{Dir: "/work/live"})
	writeDiscovery(t, "/work/live", discovery)

	project, err := Get(context.Background(), id, time.Second)
	if err != nil || !project.Live {
		t.Fatalf("Get() = %#v, %v", project, err)
	}
	gotDiscovery, err := Discovery(id)
	if err != nil || gotDiscovery.PID != discovery.PID || gotDiscovery.Port != discovery.Port || gotDiscovery.Token != discovery.Token || gotDiscovery.Dir != discovery.Dir || gotDiscovery.Version != discovery.Version || gotDiscovery.BootID == "" {
		t.Fatalf("Discovery() = %#v, %v", gotDiscovery, err)
	}
	if _, err := Get(context.Background(), "none", time.Second); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get absent error = %v, want ErrNotFound", err)
	}
}

func TestDiscoveryReturnsErrNotLive(t *testing.T) {
	home := setupHome(t)
	missingID := control.DirHash("/work/missing")
	writeSession(t, home, missingID, app.SessionFile{Dir: "/work/missing"})
	if _, err := Discovery(missingID); !errors.Is(err, ErrNotLive) {
		t.Fatalf("missing discovery error = %v, want ErrNotLive", err)
	}
	unreadableID := control.DirHash("/work/unreadable")
	writeSession(t, home, unreadableID, app.SessionFile{Dir: "/work/unreadable"})
	if err := os.MkdirAll(control.ControlPath("/work/unreadable"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Discovery(unreadableID); !errors.Is(err, ErrNotLive) {
		t.Fatalf("unreadable discovery error = %v, want ErrNotLive", err)
	}
	deadID := control.DirHash("/work/dead")
	writeSession(t, home, deadID, app.SessionFile{Dir: "/work/dead"})
	writeDiscovery(t, "/work/dead", control.Discovery{PID: 1, Port: 1, Token: "token", Dir: "/work/dead"})
	if _, err := Discovery(deadID); !errors.Is(err, ErrNotLive) {
		t.Fatalf("failed health error = %v, want ErrNotLive", err)
	}
}

func assertUnknownProject(t *testing.T, id string) {
	t.Helper()
	got, err := List(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	want := Project{ID: id}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("List() = %#v, want %#v", got, want)
	}
}

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("THEFLOOR_HOME", home)
	return home
}

func writeSession(t *testing.T, home, id string, session app.SessionFile) {
	t.Helper()
	b, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(home, ".theboringfloor", "projects", id, "session.json"), b)
}

func writeDiscovery(t *testing.T, dir string, discovery control.Discovery) {
	t.Helper()
	if err := control.WriteDiscovery(dir, discovery); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatal(err)
	}
}

func healthServer(t *testing.T, dir, token string, delay time.Duration, ok bool) *http.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != control.RouteHealth || r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		_ = json.NewEncoder(w).Encode(control.HealthResponse{OK: ok, Dir: dir, Version: "v-live", Backend: "live-backend"})
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	server.Addr = listener.Addr().String()
	return server
}

func serverPort(t *testing.T, server *http.Server) int {
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

func projectIDs(projects []Project) []string {
	ids := make([]string, len(projects))
	for i, project := range projects {
		ids[i] = project.ID
	}
	return ids
}
