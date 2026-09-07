package app

import (
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestControlTranscriptToolState(t *testing.T) {
	tests := []struct {
		name string
		meta string
		want string
	}{
		{name: "done", meta: "done\x1f4187", want: "done"},
		{name: "running", meta: "running\x1f4188", want: "running"},
		{name: "error", meta: "error\x1f4189", want: "error"},
		{name: "aborted", meta: "aborted\x1f4190", want: "aborted"},
		{name: "empty", meta: "", want: ""},
		{name: "attachment", meta: state.AttachMeta([]string{"image.png"}), want: ""},
		{name: "media attachment", meta: state.MediaMeta([]state.MediaItem{{Filename: "image.png", Mime: "image/png"}}), want: ""},
		{name: "no separator", meta: "done", want: ""},
		{name: "non-numeric tick", meta: "done\x1fnow", want: ""},
		{name: "extra field", meta: "done\x1f4191\x1fignored", want: ""},
		{name: "unknown state", meta: "waiting\x1f4191", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := controlTranscriptToolState(tt.meta); got != tt.want {
				t.Fatalf("controlTranscriptToolState(%q) = %q, want %q", tt.meta, got, tt.want)
			}
		})
	}
}

func TestControlTranscriptActivityProjection(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Employees = []state.Employee{
		{Name: "tekton-10", Role: state.RoleDeveloper, Task: "Run Flutter tests"},
		{Name: "skopos-4", Role: state.RoleScout, Task: "Map activity state"},
	}
	m.st.Chat = []state.ChatMsg{
		{ID: "tool", From: "tekton-10", Kind: "wtool", Text: "bash · flutter test", At: 1, Meta: "done\x1f4187"},
		{ID: "think", From: "skopos-4", Kind: "wthink", Text: "reading", At: 2, Meta: "running\x1f4188"},
		{ID: "unknown", From: "recycled-desk", Kind: "wtool", Text: "bash · pwd", At: 3, Meta: "error\x1f4189"},
		{ID: "attachment", From: "user", Kind: "user", Text: "review image", At: 4, Meta: state.AttachMeta([]string{"image.png"})},
		{ID: "user", From: "user", Kind: "user", Text: "plain", At: 5},
	}

	response, _ := m.controlTranscript(50, "", false)
	if got := response.Messages[0].Activity; got == nil || got.Role != "developer" || got.Task != "Run Flutter tests" || got.State != "done" {
		t.Fatalf("developer tool activity = %#v", got)
	}
	if got := response.Messages[1].Activity; got == nil || got.Role != "scout" || got.Task != "Map activity state" || got.State != "running" {
		t.Fatalf("scout think activity = %#v", got)
	}
	if got := response.Messages[2].Activity; got == nil || got.Role != "" || got.Task != "" || got.State != "error" {
		t.Fatalf("unknown worker activity = %#v", got)
	}
	if got := response.Messages[3]; got.Activity != nil || len(got.Attachments) != 1 || got.Attachments[0].Name != "image.png" {
		t.Fatalf("attachment message = %#v", got)
	}
	if got := response.Messages[4].Activity; got != nil {
		t.Fatalf("plain user activity = %#v, want nil", got)
	}
}

func TestControlTranscriptActivityDoesNotLeakMeta(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Employees = []state.Employee{{Name: "tekton-10", Role: state.RoleDeveloper, Task: "Inspect activity"}}
	m.st.Chat = []state.ChatMsg{{
		ID: "tool", From: "tekton-10", Kind: "wtool", Text: "bash · pwd", At: 1,
		Meta: "done\x1f4187\x1f/absolute/project",
	}}

	response, _ := m.controlTranscript(50, "", false)
	payload := string(marshalControlResponse(response))
	for _, secret := range []string{"done\x1f4187", "4187", "/absolute/project"} {
		if strings.Contains(payload, secret) {
			t.Fatalf("activity transcript leaked %q: %s", secret, payload)
		}
	}
	if !strings.Contains(payload, `"role":"developer"`) || !strings.Contains(payload, `"task":"Inspect activity"`) {
		t.Fatalf("activity transcript missing role or task: %s", payload)
	}
}

func TestControlTranscriptActivityPlainUserJSONUnchanged(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Chat = []state.ChatMsg{{ID: "user", From: "user", Kind: "user", Text: "plain", At: 1}}

	response, _ := m.controlTranscript(50, "", false)
	payload := string(marshalControlResponse(response))
	want := `{"messages":[{"id":"user","from":"user","kind":"user","text":"plain","at":1}],"truncated":false,"working":false}`
	if payload != want {
		t.Fatalf("plain user transcript = %s, want %s", payload, want)
	}
	if strings.Contains(payload, `"activity"`) {
		t.Fatalf("plain user transcript added activity: %s", payload)
	}
}

func TestControlTranscriptActivityRosterIndexServesLargePage(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	m.st.Employees = []state.Employee{{Name: "tekton-10", Role: state.RoleDeveloper, Task: "Large page"}}
	for i := 0; i < 200; i++ {
		m.st.Chat = append(m.st.Chat, state.ChatMsg{ID: string(rune(i + 1)), From: "tekton-10", Kind: "wtool", Meta: "done\x1f4187"})
	}

	response, _ := m.controlTranscript(500, "", false)
	if len(response.Messages) != 200 {
		t.Fatalf("message count = %d, want 200", len(response.Messages))
	}
	for _, message := range response.Messages {
		if message.Activity == nil || message.Activity.Role != "developer" {
			t.Fatalf("indexed worker activity = %#v", message.Activity)
		}
	}
}
