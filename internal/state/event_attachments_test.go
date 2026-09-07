package state

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEventControlAttachments(t *testing.T) {
	var zero Event
	if zero.ControlAttachments != nil {
		t.Fatalf("zero Event ControlAttachments = %#v, want nil", zero.ControlAttachments)
	}

	in := Event{
		Kind:        EvControlSend,
		ControlText: "hi",
		ControlAttachments: []Attachment{{
			Name: "a.png",
			Mime: "image/png",
			Path: "/tmp/a.png",
		}},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal Event: %v", err)
	}
	var out Event
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal Event: %v", err)
	}
	if out.Kind != EvControlSend || out.ControlText != "hi" {
		t.Fatalf("round trip event = %+v, want control send with text hi", out)
	}
	if len(out.ControlAttachments) != 1 {
		t.Fatalf("round trip attachments len = %d, want 1", len(out.ControlAttachments))
	}
	if got, want := out.ControlAttachments[0], in.ControlAttachments[0]; got != want {
		t.Errorf("round trip attachment = %+v, want %+v", got, want)
	}

	raw, err = json.Marshal(Event{Kind: EvControlSend, ControlText: "hi"})
	if err != nil {
		t.Fatalf("marshal Event without attachments: %v", err)
	}
	if strings.Contains(string(raw), `"controlAttachments"`) {
		t.Errorf("Event without attachments carries controlAttachments: %s", raw)
	}
}
