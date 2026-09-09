package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/projects"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

func workspaceRequest(g *gateway, method, suffix, body string, auth bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, apiPrefix+"/projects/p"+suffix, strings.NewReader(body))
	if auth {
		r.Header.Set("Authorization", "Bearer gate-token")
	}
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	g.ServeHTTP(w, r)
	return w
}
func TestGatewayFloorDataAndProtectedFiles(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	root := t.TempDir()
	g := newTestGateway(t, nil)
	g.get = func(context.Context, string, time.Duration) (projects.Project, error) {
		return projects.Project{ID: "p", Dir: root}, nil
	}
	os.WriteFile(filepath.Join(root, "README.md"), []byte("# Hello"), 0600)
	for _, suffix := range []string{"/workspace", "/files", "/conversation", "/plan", "/workspace/action", "/teams", "/tickets"} {
		if w := workspaceRequest(g, "GET", suffix, "", false); w.Code != 401 {
			t.Fatalf("unprotected %s", suffix)
		}
	}
	if w := workspaceRequest(g, "POST", "/teams", `{"name":"QA"}`, true); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := workspaceRequest(g, "POST", "/tickets", `{"title":"Mobile explorer","status":"backlog","priority":"P1","team":"frontend"}`, true); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w := workspaceRequest(g, "GET", "/workspace", "", true)
	var body struct {
		Floor workspace.Floor `json:"floor"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != 200 || len(body.Floor.Teams) != 5 || len(body.Floor.Tickets) != 1 {
		t.Fatalf("workspace: %s", w.Body.String())
	}
	if w := workspaceRequest(g, "GET", "/files?path=README.md", "", true); w.Code != 200 || !strings.Contains(w.Body.String(), "# Hello") {
		t.Fatalf("preview: %s", w.Body.String())
	}
	if w := workspaceRequest(g, "GET", "/files?path=../README.md", "", true); w.Code != 400 {
		t.Fatal("allowed traversal")
	}
	if w := workspaceRequest(g, "POST", "/tickets", `{"title":"bad","status":"unknown"}`, true); w.Code != 400 {
		t.Fatal("allowed invalid ticket")
	}
}
func TestGatewayArchivedConversationRetainsMetadata(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	root := t.TempDir()
	g := newTestGateway(t, nil)
	g.get = func(context.Context, string, time.Duration) (projects.Project, error) {
		return projects.Project{ID: "p", Dir: root}, nil
	}
	sf := app.SessionFile{Dir: root, Backend: "codex", PrimaryID: "thread", Chat: []state.ChatMsg{{ID: "user", From: "user", Kind: "user", Text: "Question"}, {ID: "answer", From: "boss", Kind: "boss", Text: "Answer"}, {ID: "partial", From: "boss", Text: "Still typing", Pending: true}}}
	if err := workspace.WriteJSON(filepath.Join(workspace.ConversationDir(root, "codex", "thread"), "session.json"), sf); err != nil {
		t.Fatal(err)
	}
	w := workspaceRequest(g, "GET", "/conversation?backend=codex&session=thread", "", true)
	var body struct {
		Messages []control.TranscriptMessage `json:"messages"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != 200 || len(body.Messages) != 2 {
		t.Fatalf("archive: %s", w.Body.String())
	}
	if w := workspaceRequest(g, "GET", "/conversation?backend=claudecode&session=thread", "", true); w.Code != 404 {
		t.Fatal("cross-backend archive opened")
	}
}
func TestGatewayLiveTicketUsesOfficeOwner(t *testing.T) {
	var received control.WorkspaceAction
	office := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != control.RouteWorkspaceAction || r.Header.Get("Authorization") != "Bearer office-token" {
			t.Errorf("invalid office request")
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(202)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer office.Close()
	g := newTestGateway(t, nil)
	g.get = func(context.Context, string, time.Duration) (projects.Project, error) {
		return projects.Project{ID: "p", Dir: t.TempDir(), Live: true}, nil
	}
	g.discovery = testDiscovery(t, office.URL)
	w := workspaceRequest(g, "POST", "/tickets", `{"title":"Ship it","status":"backlog","priority":"P2"}`, true)
	if w.Code != 202 || received.Action != "ticket-save" {
		t.Fatalf("did not use live owner: %s %+v", w.Body.String(), received)
	}
}
