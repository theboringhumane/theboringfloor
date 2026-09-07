package app

import (
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestControlMutationSendPassesAttachments(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &recBackend{}
	m := New(b, nil)
	atts := []state.Attachment{
		{Name: "one.png", Mime: "image/png", Path: "/project/uploads/one.png"},
		{Name: "two.jpg", Mime: "image/jpeg", Path: "/project/uploads/two.jpg"},
	}

	cmd := m.applyControlMutations(state.Event{
		Kind: state.EvControlSend, ControlText: "review these", ControlAttachments: atts,
	})
	if cmd == nil {
		t.Fatal("remote attachment send must return a command")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("remote attachment send command must produce a result")
	}
	if len(b.sentAtts) != 1 {
		t.Fatalf("attachment send calls = %d, want 1", len(b.sentAtts))
	}
	if got := b.sentAtts[0]; !sameAttachments(got, atts) {
		t.Fatalf("sent attachments = %#v, want %#v", got, atts)
	}
}

func TestControlTranscriptAttachmentProjectionDoesNotLeakPaths(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	m := New(&agentRecBackend{}, nil)
	path := "/absolute/project/uploads/one.png"
	m.st.Chat = []state.ChatMsg{
		{
			ID: "outbound", From: "user", Kind: "user", Text: "review", At: 1,
			Meta: state.AttachMeta([]string{"one.png", "two.jpg"}),
		},
		{
			ID: "inbound", From: "boss", Kind: "boss", Text: "here", At: 2,
			Meta: state.MediaMeta([]state.MediaItem{
				{Filename: "one.png", Mime: "image/png", Hash: "a"},
				{Filename: "two.jpg", Mime: "image/jpeg", Hash: "b"},
			}),
		},
		{ID: "plain", From: "user", Kind: "user", Text: "plain", At: 3},
	}

	response, _ := m.controlTranscript(50, "", false)
	if got, want := response.Messages[0].Attachments, []string{"one.png", "two.jpg"}; !attachmentNamesEqual(got, want) {
		t.Fatalf("outbound attachments = %#v, want names %q", got, want)
	}
	if got := response.Messages[1].Attachments; len(got) != 2 || got[0].Name != "one.png" || got[0].Mime != "image/png" || got[1].Name != "two.jpg" || got[1].Mime != "image/jpeg" {
		t.Fatalf("inbound attachments = %#v", got)
	}
	payload := string(marshalControlResponse(response))
	if !strings.Contains(payload, `"name":"one.png"`) || !strings.Contains(payload, `"mime":"image/png"`) {
		t.Fatalf("attachment transcript payload missing metadata: %s", payload)
	}
	if strings.Contains(payload, path) {
		t.Fatalf("attachment transcript payload leaked path: %s", payload)
	}
	if strings.Contains(payload, `"id":"plain","from":"user","kind":"user","text":"plain","at":3,"attachments"`) {
		t.Fatalf("plain transcript message must omit attachments: %s", payload)
	}
}

func attachmentNamesEqual(got []control.TranscriptAttachment, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i].Name != want[i] || got[i].Mime != "" {
			return false
		}
	}
	return true
}

func TestControlMutationSendWithoutAttachmentsRemainsNil(t *testing.T) {
	t.Setenv("THEFLOOR_HOME", t.TempDir())
	b := &recBackend{}
	m := New(b, nil)

	cmd := m.applyControlMutations(state.Event{Kind: state.EvControlSend, ControlText: "plain"})
	if cmd == nil || cmd() == nil {
		t.Fatal("plain remote send must produce a result")
	}
	if len(b.sentAtts) != 1 || b.sentAtts[0] != nil {
		t.Fatalf("plain remote send attachments = %#v, want [nil]", b.sentAtts)
	}
}

func sameAttachments(got, want []state.Attachment) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
