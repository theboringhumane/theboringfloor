package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/projects"
)

func TestGatewayTokenMintReuseAndCorruption(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	home := gatewayHome()
	first, err := loadGatewayToken(home)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(gatewayTokenPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("token mode = %o, want 600", got)
	}
	second, err := loadGatewayToken(home)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("token was reminted: first %q second %q", first, second)
	}
	if err := os.WriteFile(gatewayTokenPath(home), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadGatewayToken(home); err == nil {
		t.Fatal("corrupt token file did not fail startup")
	}
}

func TestGatewayTokenWhitespaceIsStartupError(t *testing.T) {
	home := t.TempDir()
	path := gatewayTokenPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"token":"   ","createdAt":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadGatewayToken(home); err == nil || err.Error() != "read gateway token: invalid "+path {
		t.Fatalf("loadGatewayToken() error = %v, want invalid token file error", err)
	}
}

func TestAuthorizedRejectsEmptyConfiguredAndProvidedTokens(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, apiPrefix+"/health", nil)
	request.Header.Set("Authorization", "Bearer ")
	if authorized(request, "") {
		t.Fatal("authorized empty configured and provided tokens")
	}
}

func TestGatewayAuthProtectsEveryRoute(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	gateway := newTestGateway(t, nil)
	server := httptest.NewServer(gateway)
	defer server.Close()
	for _, header := range []string{"", "Basic gate-token", "Bearer wrong-token"} {
		req, err := http.NewRequest(http.MethodGet, server.URL+apiPrefix+"/health", nil)
		if err != nil {
			t.Fatal(err)
		}
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body := responseBody(t, response)
		if response.StatusCode != http.StatusUnauthorized || body != "{\"error\":\"unauthorized\"}\n" {
			t.Fatalf("header %q: status/body = %d %q", header, response.StatusCode, body)
		}
	}
}

func TestGatewayProxiesOfficeResponsesOverRealListeners(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	const officeToken = "office-token"
	projectDir := t.TempDir()
	projectID := control.DirHash(projectDir)
	var gotAuthorization, gotPath, gotMessage string
	office := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		gotPath = r.URL.RequestURI()
		if r.URL.Path == control.RouteHealth {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"dir":"` + projectDir + `","version":"dev","backend":"opencode"}`))
			return
		}
		if r.URL.Path == control.RouteMessage {
			contents, _ := io.ReadAll(r.Body)
			gotMessage = string(contents)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":1}],"truncated":false}`))
	}))
	defer office.Close()
	fixtureDiscovery(t, projectID, projectDir, office.URL, officeToken)
	gateway := newTestGateway(t, nil)
	server := httptest.NewServer(gateway)
	defer server.Close()

	response := authorizedRequest(t, server.URL+apiPrefix+"/projects/"+projectID+"/transcript?limit=12&before=m-12", http.MethodGet, nil, "gate-token")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/json" || body != `{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":1}],"truncated":false}` {
		t.Fatalf("transcript status/content-type/body = %d %q %q", response.StatusCode, response.Header.Get("Content-Type"), body)
	}
	if gotAuthorization != "Bearer "+officeToken || gotPath != "/v1/transcript?before=m-12&limit=12" {
		t.Fatalf("upstream auth/path = %q %q", gotAuthorization, gotPath)
	}

	response = authorizedRequest(t, server.URL+apiPrefix+"/projects/"+projectID+"/message", http.MethodPost, strings.NewReader(`{"text":" hello "}`), "gate-token")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || body == "" || gotMessage != `{"text":"hello"}` {
		t.Fatalf("message status/body/forward = %d %q %q", response.StatusCode, body, gotMessage)
	}
}

func TestGatewayProxiesOfficeErrorResponses(t *testing.T) {
	for _, test := range []struct {
		name            string
		officeStatus    int
		officeBody      string
		officeType      string
		wantStatus      int
		wantBody        string
		wantContentType string
	}{
		{
			name:            "json not found",
			officeStatus:    http.StatusNotFound,
			officeBody:      `{"error":"not found"}`,
			officeType:      "application/json",
			wantStatus:      http.StatusNotFound,
			wantBody:        `{"error":"not found"}`,
			wantContentType: "application/json",
		},
		{
			name:            "json office busy",
			officeStatus:    http.StatusServiceUnavailable,
			officeBody:      `{"error":"office busy"}`,
			officeType:      "application/json",
			wantStatus:      http.StatusServiceUnavailable,
			wantBody:        `{"error":"office busy"}`,
			wantContentType: "application/json",
		},
		{
			name:            "non-json body",
			officeStatus:    http.StatusGatewayTimeout,
			officeBody:      "upstream timed out",
			officeType:      "text/plain",
			wantStatus:      http.StatusGatewayTimeout,
			wantBody:        "{\"error\":\"office returned an error\"}\n",
			wantContentType: "application/json",
		},
		{
			name:            "empty body",
			officeStatus:    http.StatusInternalServerError,
			officeBody:      "",
			wantStatus:      http.StatusInternalServerError,
			wantBody:        "{\"error\":\"office returned an error\"}\n",
			wantContentType: "application/json",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			office := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.officeType != "" {
					w.Header().Set("Content-Type", test.officeType)
				}
				w.WriteHeader(test.officeStatus)
				_, _ = w.Write([]byte(test.officeBody))
			}))
			defer office.Close()

			gateway := newTestGateway(t, nil)
			gateway.discovery = testDiscovery(t, office.URL)
			server := httptest.NewServer(gateway)
			defer server.Close()

			response := authorizedRequest(t, server.URL+apiPrefix+"/projects/p/busy", http.MethodGet, nil, "gate-token")
			if body := responseBody(t, response); response.StatusCode != test.wantStatus || body != test.wantBody || response.Header.Get("Content-Type") != test.wantContentType {
				t.Fatalf("status/body/content-type = %d %q %q, want %d %q %q", response.StatusCode, body, response.Header.Get("Content-Type"), test.wantStatus, test.wantBody, test.wantContentType)
			}
		})
	}
}

func TestGatewayPreservesGatewayOriginatedFailureResponses(t *testing.T) {
	for _, test := range []struct {
		name        string
		discovery   func(string) (control.Discovery, error)
		get         func(context.Context, string, time.Duration) (projects.Project, error)
		client      *http.Client
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "transport error",
			discovery:   func(string) (control.Discovery, error) { return control.Discovery{Port: 1, Token: "office-token"}, nil },
			client:      &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("dial refused") })},
			wantStatus:  http.StatusBadGateway,
			wantMessage: "could not reach office",
		},
		{
			name:        "timeout",
			discovery:   func(string) (control.Discovery, error) { return control.Discovery{Port: 1, Token: "office-token"}, nil },
			client:      &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })},
			wantStatus:  http.StatusGatewayTimeout,
			wantMessage: "office timed out",
		},
		{
			name:      "unknown project",
			discovery: func(string) (control.Discovery, error) { return control.Discovery{}, projects.ErrNotLive },
			get: func(context.Context, string, time.Duration) (projects.Project, error) {
				return projects.Project{}, projects.ErrNotFound
			},
			wantStatus:  http.StatusNotFound,
			wantMessage: "project not found",
		},
		{
			name:        "office not running",
			discovery:   func(string) (control.Discovery, error) { return control.Discovery{}, projects.ErrNotLive },
			wantStatus:  http.StatusConflict,
			wantMessage: "office not running",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gateway := newTestGateway(t, nil)
			gateway.discovery = test.discovery
			if test.get != nil {
				gateway.get = test.get
			}
			if test.client != nil {
				gateway.client = test.client
			}
			server := httptest.NewServer(gateway)
			defer server.Close()

			response := authorizedRequest(t, server.URL+apiPrefix+"/projects/p/status", http.MethodGet, nil, "gate-token")
			if body := responseBody(t, response); response.StatusCode != test.wantStatus || body != `{"error":"`+test.wantMessage+`"}`+"\n" {
				t.Fatalf("status/body = %d %q, want %d error %q", response.StatusCode, body, test.wantStatus, test.wantMessage)
			}
		})
	}
}

func TestGatewayRejectsInvalidMessageAndTranscriptBeforeProxy(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	gateway := newTestGateway(t, nil)
	called := false
	gateway.discovery = func(string) (control.Discovery, error) {
		called = true
		return control.Discovery{}, nil
	}
	server := httptest.NewServer(gateway)
	defer server.Close()
	cases := []struct {
		name, path, contentType, body, want string
	}{
		{"content type", "/projects/p/message", "text/plain", `{"text":"hi"}`, "content type must be application/json"},
		{"trailing", "/projects/p/message", "application/json", `{"text":"hi"}{}`, "invalid message body"},
		{"empty", "/projects/p/message", "application/json", `{"text":"  "}`, "empty message text"},
		{"bad limit", "/projects/p/transcript?limit=no", "", "", "invalid transcript limit"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			method := http.MethodGet
			if strings.HasSuffix(test.path, "/message") {
				method = http.MethodPost
			}
			request, err := http.NewRequest(method, server.URL+apiPrefix+test.path, strings.NewReader(test.body))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Authorization", "Bearer gate-token")
			if test.contentType != "" {
				request.Header.Set("Content-Type", test.contentType)
			}
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusBadRequest || responseBody(t, response) != `{"error":"`+test.want+`"}`+"\n" {
				t.Fatalf("status/body = %d", response.StatusCode)
			}
		})
	}
	if called {
		t.Fatal("invalid requests reached project discovery")
	}
}

func TestGatewayProjectErrorMapping(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	for _, test := range []struct {
		name string
		err  error
		want int
	}{
		{"not found", projects.ErrNotFound, http.StatusNotFound},
		{"not live", projects.ErrNotLive, http.StatusConflict},
	} {
		t.Run(test.name, func(t *testing.T) {
			gateway := newTestGateway(t, nil)
			gateway.discovery = func(string) (control.Discovery, error) { return control.Discovery{}, test.err }
			server := httptest.NewServer(gateway)
			defer server.Close()
			response := authorizedRequest(t, server.URL+apiPrefix+"/projects/p/status", http.MethodGet, nil, "gate-token")
			if response.StatusCode != test.want {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.want)
			}
			_ = responseBody(t, response)
		})
	}
}

func TestGatewayStartsStoppedProject(t *testing.T) {
	project := projects.Project{ID: "known", Dir: "/registry/project", Live: false}
	gateway := newTestGateway(t, nil)
	gateway.get = func(context.Context, string, time.Duration) (projects.Project, error) {
		return project, nil
	}
	launcher := &recordingLauncher{}
	gateway.launcher = launcher
	server := httptest.NewServer(gateway)
	defer server.Close()

	response := authorizedRequest(t, server.URL+apiPrefix+"/projects/known/start", http.MethodPost, nil, "gate-token")
	if body := responseBody(t, response); response.StatusCode != http.StatusAccepted || body != "{\"id\":\"known\",\"startRequested\":true}\n" {
		t.Fatalf("status/body = %d %q", response.StatusCode, body)
	}
	if got := launcher.dirs(); len(got) != 1 || got[0] != project.Dir {
		t.Fatalf("launcher dirs = %q, want registry dir %q", got, project.Dir)
	}
}

func TestNewOfficeCommandConfiguresHeadlessOffice(t *testing.T) {
	t.Setenv("NO_CONTROL", "1")
	binary, err := exec.LookPath("true")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	command, closeFiles, err := newOfficeCommandForBinary(dir, binary)
	if err != nil {
		t.Fatal(err)
	}
	defer closeFiles()

	if command.Dir != dir {
		t.Fatalf("office command directory = %q, want %q", command.Dir, dir)
	}
	for _, stream := range []struct {
		name string
		file any
	}{
		{name: "stdin", file: command.Stdin},
		{name: "stdout", file: command.Stdout},
		{name: "stderr", file: command.Stderr},
	} {
		file, ok := stream.file.(*os.File)
		if !ok {
			t.Fatalf("%s = %T, want open *os.File for %s", stream.name, stream.file, os.DevNull)
		}
		info, err := file.Stat()
		if err != nil {
			t.Fatalf("stat %s: %v", stream.name, err)
		}
		if info.Mode()&os.ModeCharDevice == 0 {
			t.Fatalf("%s is not a character device: %v", stream.name, info.Mode())
		}
	}
	if command.Stdin == os.Stdin || command.Stdout == os.Stdout || command.Stderr == os.Stderr {
		t.Fatal("office command inherited gateway stdio")
	}
	for _, entry := range command.Env {
		if strings.HasPrefix(entry, "NO_CONTROL=") {
			t.Fatalf("office command retained NO_CONTROL: %q", entry)
		}
	}
}

func TestCommandLauncherReportsMissingBinary(t *testing.T) {
	oldLookPath := officeLookPath
	officeLookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { officeLookPath = oldLookPath })

	err := (commandLauncher{}).Launch(t.TempDir(), func() {})
	if !errors.Is(err, ErrOfficeBinaryNotFound) {
		t.Fatalf("Launch() error = %v, want ErrOfficeBinaryNotFound", err)
	}
}

func TestCommandLauncherReapsChildAndCallsExitedOnce(t *testing.T) {
	binary, err := exec.LookPath("true")
	if err != nil {
		t.Fatal(err)
	}
	oldLookPath := officeLookPath
	officeLookPath = func(string) (string, error) { return binary, nil }
	t.Cleanup(func() { officeLookPath = oldLookPath })

	var calls atomic.Int32
	exited := make(chan struct{})
	if err := (commandLauncher{}).Launch(t.TempDir(), func() {
		if calls.Add(1) == 1 {
			close(exited)
		}
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("exited callback was not called")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("exited callback count = %d, want 1", got)
	}
}

func TestGatewayStartFailures(t *testing.T) {
	for _, test := range []struct {
		name       string
		get        func(context.Context, string, time.Duration) (projects.Project, error)
		launcher   officeLauncher
		wantStatus int
		wantBody   string
	}{
		{
			name: "unknown project",
			get: func(context.Context, string, time.Duration) (projects.Project, error) {
				return projects.Project{}, projects.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
			wantBody:   "{\"error\":\"project not found\"}\n",
		},
		{
			name: "live project",
			get: func(context.Context, string, time.Duration) (projects.Project, error) {
				return projects.Project{ID: "live", Dir: "/registry/live", Live: true}, nil
			},
			wantStatus: http.StatusConflict,
			wantBody:   "{\"error\":\"office already running\"}\n",
		},
		{
			name: "launch error",
			get: func(context.Context, string, time.Duration) (projects.Project, error) {
				return projects.Project{ID: "stopped", Dir: "/registry/stopped"}, nil
			},
			launcher:   launcherFunc(func(string, func()) error { return errors.New("missing binary") }),
			wantStatus: http.StatusBadGateway,
			wantBody:   "{\"error\":\"could not start office\"}\n",
		},
		{
			name: "missing binary",
			get: func(context.Context, string, time.Duration) (projects.Project, error) {
				return projects.Project{ID: "stopped", Dir: "/registry/stopped"}, nil
			},
			launcher:   launcherFunc(func(string, func()) error { return ErrOfficeBinaryNotFound }),
			wantStatus: http.StatusBadGateway,
			wantBody:   "{\"error\":\"could not start office\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			gateway := newTestGateway(t, nil)
			gateway.get = test.get
			if test.launcher != nil {
				gateway.launcher = test.launcher
			}
			server := httptest.NewServer(gateway)
			defer server.Close()

			response := authorizedRequest(t, server.URL+apiPrefix+"/projects/p/start", http.MethodPost, nil, "gate-token")
			if body := responseBody(t, response); response.StatusCode != test.wantStatus || body != test.wantBody {
				t.Fatalf("status/body = %d %q, want %d %q", response.StatusCode, body, test.wantStatus, test.wantBody)
			}
		})
	}
}

func TestGatewayStartRejectsRequestBodies(t *testing.T) {
	gateway := newTestGateway(t, nil)
	launcher := &recordingLauncher{}
	gateway.launcher = launcher
	server := httptest.NewServer(gateway)
	defer server.Close()

	for _, body := range []string{
		`{"dir":"/tmp/attacker"}`,
		`{"command":"sh","args":["-c","evil"],"env":{"PATH":"/tmp"}}`,
	} {
		response := authorizedRequest(t, server.URL+apiPrefix+"/projects/project-1/start", http.MethodPost, strings.NewReader(body), "gate-token")
		if got := responseBody(t, response); response.StatusCode != http.StatusBadRequest || got != "{\"error\":\"start request body must be empty\"}\n" {
			t.Fatalf("body %q: status/body = %d %q", body, response.StatusCode, got)
		}
	}
	if got := launcher.dirs(); len(got) != 0 {
		t.Fatalf("launcher invoked for rejected bodies: %q", got)
	}
}

func TestGatewayStartAllowsOnlyOneConcurrentLaunch(t *testing.T) {
	gateway := newTestGateway(t, nil)
	gateway.get = func(context.Context, string, time.Duration) (projects.Project, error) {
		return projects.Project{ID: "stopped", Dir: "/registry/stopped"}, nil
	}
	launcher := &blockingLauncher{started: make(chan struct{}), release: make(chan struct{})}
	gateway.launcher = launcher
	server := httptest.NewServer(gateway)
	defer server.Close()

	type result struct {
		response *http.Response
		err      error
	}
	responses := make(chan result, 2)
	for range 2 {
		go func() {
			request, err := http.NewRequest(http.MethodPost, server.URL+apiPrefix+"/projects/stopped/start", nil)
			if err == nil {
				request.Header.Set("Authorization", "Bearer gate-token")
				request.Header.Set("Content-Type", "application/json")
				response, requestErr := http.DefaultClient.Do(request)
				responses <- result{response: response, err: requestErr}
				return
			}
			responses <- result{err: err}
		}()
	}
	<-launcher.started
	secondResult := <-responses
	if secondResult.err != nil {
		t.Fatal(secondResult.err)
	}
	second := secondResult.response
	if body := responseBody(t, second); second.StatusCode != http.StatusConflict || body != "{\"error\":\"office already running\"}\n" {
		t.Fatalf("second status/body = %d %q", second.StatusCode, body)
	}
	close(launcher.release)
	firstResult := <-responses
	if firstResult.err != nil {
		t.Fatal(firstResult.err)
	}
	response := firstResult.response
	if body := responseBody(t, response); response.StatusCode != http.StatusAccepted || body != "{\"id\":\"stopped\",\"startRequested\":true}\n" {
		t.Fatalf("first status/body = %d %q", response.StatusCode, body)
	}
	if got := launcher.count(); got != 1 {
		t.Fatalf("launcher count = %d, want 1", got)
	}
}

func TestGatewayStartRejectsNonPostMethod(t *testing.T) {
	gateway := newTestGateway(t, nil)
	server := httptest.NewServer(gateway)
	defer server.Close()
	response := authorizedRequest(t, server.URL+apiPrefix+"/projects/project-1/start", http.MethodGet, nil, "gate-token")
	if body := responseBody(t, response); response.StatusCode != http.StatusMethodNotAllowed || response.Header.Get("Allow") != http.MethodPost || body != "{\"error\":\"method not allowed\"}\n" {
		t.Fatalf("status/allow/body = %d %q %q", response.StatusCode, response.Header.Get("Allow"), body)
	}
}

func TestResolvedBindPrecedence(t *testing.T) {
	if got := resolvedBind("", ""); got != "127.0.0.1:8787" {
		t.Fatalf("default = %q", got)
	}
	if got := resolvedBind("", "100.1.2.3:99"); got != "100.1.2.3:99" {
		t.Fatalf("env = %q", got)
	}
	if got := resolvedBind("127.0.0.1:12", "100.1.2.3:99"); got != "127.0.0.1:12" {
		t.Fatalf("flag = %q", got)
	}
}

func TestBindWarning(t *testing.T) {
	if got := bindWarning(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8787}); got != "" {
		t.Fatalf("loopback warning = %q", got)
	}
	if got := bindWarning(&net.TCPAddr{IP: net.ParseIP("100.64.0.7"), Port: 8787}); got != "floorgate warning: listening on 100.64.0.7:8787; the bearer token is the only protection\n" {
		t.Fatalf("warning = %q", got)
	}
}

func newTestGateway(t *testing.T, discovery *control.Discovery) *gateway {
	t.Helper()
	gateway := newGateway("gate-token")
	gateway.list = func(context.Context, time.Duration) ([]projects.Project, error) {
		return []projects.Project{{ID: "project-1", Name: "Project One", Live: true}}, nil
	}
	gateway.get = func(context.Context, string, time.Duration) (projects.Project, error) {
		return projects.Project{ID: "project-1", Name: "Project One", Live: true}, nil
	}
	if discovery != nil {
		gateway.discovery = func(string) (control.Discovery, error) { return *discovery, nil }
	}
	return gateway
}

func testDiscovery(t *testing.T, rawURL string) func(string) (control.Discovery, error) {
	t.Helper()
	address := strings.TrimPrefix(rawURL, "http://")
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	return func(string) (control.Discovery, error) {
		return control.Discovery{Port: n, Token: "office-token"}, nil
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type launcherFunc func(string, func()) error

func (f launcherFunc) Launch(dir string, exited func()) error {
	return f(dir, exited)
}

type recordingLauncher struct {
	mu      sync.Mutex
	started []string
}

func (l *recordingLauncher) Launch(dir string, _ func()) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.started = append(l.started, dir)
	return nil
}

func (l *recordingLauncher) dirs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.started...)
}

type blockingLauncher struct {
	mu      sync.Mutex
	started chan struct{}
	release chan struct{}
	calls   int
}

func (l *blockingLauncher) Launch(_ string, _ func()) error {
	l.mu.Lock()
	l.calls++
	l.mu.Unlock()
	close(l.started)
	<-l.release
	return nil
}

func (l *blockingLauncher) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.calls
}

func fixtureDiscovery(t *testing.T, id, dir, rawURL, token string) {
	t.Helper()
	address := strings.TrimPrefix(rawURL, "http://")
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	projectPath := filepath.Join(projects.Root(), id)
	if err := os.MkdirAll(projectPath, 0o755); err != nil {
		t.Fatal(err)
	}
	session, err := json.Marshal(map[string]string{"dir": dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, "session.json"), session, 0o600); err != nil {
		t.Fatal(err)
	}
	discovery := control.Discovery{Port: n, Token: token, Dir: dir}
	if err := control.WriteDiscovery(discovery.Dir, discovery); err != nil {
		t.Fatal(err)
	}
}

func authorizedRequest(t *testing.T, url, method string, body io.Reader, token string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if method == http.MethodPost {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func responseBody(t *testing.T, response *http.Response) string {
	t.Helper()
	defer response.Body.Close()
	contents, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
