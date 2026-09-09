package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

func (g *gateway) workspaceRoute(w http.ResponseWriter, r *http.Request, id, suffix string) bool {
	switch suffix {
	case "/workspace", "/teams", "/tickets", "/files", "/conversation", "/plan", "/workspace/action":
	default:
		return false
	}
	if suffix == "/plan" && r.Method == http.MethodGet {
		g.proxy(w, r, id, http.MethodGet, control.RoutePlan, nil)
		return true
	}
	if suffix == "/workspace/action" && r.Method == http.MethodPost {
		var body control.WorkspaceAction
		if !workspaceBody(w, r, &body) {
			return true
		}
		payload, _ := json.Marshal(body)
		g.proxy(w, r, id, http.MethodPost, control.RouteWorkspaceAction, payload)
		return true
	}
	p, err := g.get(r.Context(), id, 10*time.Second)
	if err != nil {
		g.projectError(w, err)
		return true
	}
	if p.Dir == "" {
		writeError(w, 404, "project directory unavailable")
		return true
	}
	switch {
	case suffix == "/workspace" && r.Method == http.MethodGet:
		floor, err := workspace.Load(p.Dir)
		if err != nil {
			writeError(w, 500, "could not read floor")
			return true
		}
		conversations, err := workspace.Conversations(p.Dir)
		if err != nil {
			writeError(w, 500, "could not read conversations")
			return true
		}
		if conversations == nil {
			conversations = []workspace.Conversation{}
		}
		writeJSON(w, 200, map[string]any{"floor": floor, "conversations": conversations})
	case suffix == "/teams" && r.Method == http.MethodPost:
		var b struct {
			Name string `json:"name"`
		}
		if !workspaceBody(w, r, &b) {
			return true
		}
		if p.Live {
			payload, _ := json.Marshal(control.WorkspaceAction{Action: "team-add", Text: b.Name})
			g.proxy(w, r, id, http.MethodPost, control.RouteWorkspaceAction, payload)
			return true
		}
		floor, err := workspace.AddTeam(p.Dir, b.Name)
		if err != nil {
			writeError(w, 400, err.Error())
		} else {
			writeJSON(w, 200, floor)
		}
	case suffix == "/tickets" && r.Method == http.MethodPost:
		var b workspace.Ticket
		if !workspaceBody(w, r, &b) {
			return true
		}
		if p.Live {
			ticket, _ := json.Marshal(b)
			payload, _ := json.Marshal(control.WorkspaceAction{Action: "ticket-save", Ticket: ticket})
			g.proxy(w, r, id, http.MethodPost, control.RouteWorkspaceAction, payload)
			return true
		}
		floor, err := workspace.PutTicket(p.Dir, b)
		if err != nil {
			writeError(w, 400, err.Error())
		} else {
			writeJSON(w, 200, floor)
		}
	case suffix == "/files" && r.Method == http.MethodGet:
		page, err := workspace.ReadFiles(p.Dir, r.URL.Query().Get("path"))
		if err != nil {
			writeError(w, 400, "file is unavailable or outside this project")
		} else {
			writeJSON(w, 200, page)
		}
	case suffix == "/conversation" && r.Method == http.MethodGet:
		backend, session := r.URL.Query().Get("backend"), r.URL.Query().Get("session")
		if backend == "" || session == "" {
			writeError(w, 400, "backend and session are required")
			return true
		}
		file, err := os.Open(filepath.Join(workspace.ConversationDir(p.Dir, backend, session), "session.json"))
		if err != nil {
			writeError(w, 404, "conversation not found")
			return true
		}
		defer file.Close()
		var sf app.SessionFile
		if json.NewDecoder(io.LimitReader(file, 32<<20)).Decode(&sf) != nil || sf.Dir != p.Dir || sf.Backend != backend || sf.PrimaryID != session {
			writeError(w, 400, "conversation is unavailable")
			return true
		}
		// History stays unfiltered: the Android presentation owns its reading view.
		writeJSON(w, 200, map[string]any{"messages": app.ArchiveTranscript(sf), "hasMore": false, "plan": sf.PlanText, "approvedPlan": sf.ApprovedPlanText})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
	return true
}
func workspaceBody(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(value) != nil || rejectTrailingJSON(d) != nil {
		writeError(w, 400, "invalid workspace request")
		return false
	}
	return true
}
