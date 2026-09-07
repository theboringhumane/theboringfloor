package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	response := authorizedRequest(t, server.URL+apiPrefix+"/projects/"+projectID+"/transcript?limit=12", http.MethodGet, nil, "gate-token")
	body := responseBody(t, response)
	if response.StatusCode != http.StatusOK || body != `{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":1}],"truncated":false}` {
		t.Fatalf("transcript status/body = %d %q", response.StatusCode, body)
	}
	if gotAuthorization != "Bearer "+officeToken || gotPath != "/v1/transcript?limit=12" {
		t.Fatalf("upstream auth/path = %q %q", gotAuthorization, gotPath)
	}

	response = authorizedRequest(t, server.URL+apiPrefix+"/projects/"+projectID+"/message", http.MethodPost, strings.NewReader(`{"text":" hello "}`), "gate-token")
	body = responseBody(t, response)
	if response.StatusCode != http.StatusOK || body == "" || gotMessage != `{"text":"hello"}` {
		t.Fatalf("message status/body/forward = %d %q %q", response.StatusCode, body, gotMessage)
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
		{"large limit", "/projects/p/transcript?limit=501", "", "", "invalid transcript limit"},
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
