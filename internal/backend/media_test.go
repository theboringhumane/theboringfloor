// media_test.go — fixture proof for the boss-turn image preview's wire
// seam: ocPart's FilePart fields parse off raw SSE JSON, the message.
// part.updated mapping emits EvChatMedia exactly once per part (payload +
// dims + hash), and sessionMessageRow retains file parts verbatim for
// the history splice.
package backend

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// tinyPNG builds a REAL 2×2 PNG (dims must decode — the media gate
// header-sniffs them).
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(0, 1, color.NRGBA{G: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 255})
	img.SetNRGBA(1, 1, color.NRGBA{A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func mediaFrame(t *testing.T, part map[string]any) ocSSEEvent {
	t.Helper()
	props, err := json.Marshal(map[string]any{"part": part})
	if err != nil {
		t.Fatal(err)
	}
	return ocSSEEvent{Type: "message.part.updated", Properties: props}
}

// TestMediaPartSSEMapping — a file part on the PRIMARY session maps to
// ONE EvChatMedia carrying the bubble identity, the payload URL, the
// header-sniffed dims, and the deterministic hash; a repeated frame for
// the same part is deduped; a child's file part never surfaces.
func TestMediaPartSSEMapping(t *testing.T) {
	pngBytes := tinyPNG(t)
	url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	ctx := newNormCtx(nil)
	frame := mediaFrame(t, map[string]any{
		"id": "part-1", "sessionID": "ses-primary", "messageID": "msg-1",
		"type": "file", "mime": "image/png", "filename": "paste-diagram.png", "url": url,
	})
	evs := mapOCEvent(frame, ctx, "ses-primary", 100)
	if len(evs) != 1 || evs[0].Kind != state.EvChatMedia {
		t.Fatalf("a primary file part must emit ONE EvChatMedia, got %+v", evs)
	}
	ev := evs[0]
	if ev.Msg.ID != "bossmsg-msg-1" || ev.SessionID != "ses-primary" || ev.CallID != "part-1" {
		t.Fatalf("media event must pin msg/part/session ids: %+v", ev)
	}
	if len(ev.Media) != 1 {
		t.Fatalf("want ONE MediaItem, got %+v", ev.Media)
	}
	it := ev.Media[0]
	if it.Mime != "image/png" || it.Filename != "paste-diagram.png" || it.W != 2 || it.H != 2 {
		t.Fatalf("mime/filename/dims must ride: %+v", it)
	}
	if it.URL != url || it.Hash != state.DataURLHash(url) || len(it.Hash) != 12 {
		t.Fatalf("payload + 12-char hash must ride: %+v", it)
	}
	// AND the Meta carrier the app+panel read is buildable from them.
	meta := state.MediaMeta(ev.Media)
	if meta == "" {
		t.Fatal("the carrier builds from the event's MediaItems")
	}
	parsed, ok := state.ParseMediaMeta(meta)
	if !ok || len(parsed) != 1 || parsed[0].Filename != "paste-diagram.png" || parsed[0].Hash != it.Hash {
		t.Fatalf("carrier round-trip mismatch: %q → %+v", meta, parsed)
	}
	// repeat frame: deduped (the part already sighted).
	if evs := mapOCEvent(frame, ctx, "ses-primary", 200); len(evs) != 0 {
		t.Fatalf("a repeated part frame must dedupe, got %+v", evs)
	}
	// a CHILD session's file part stays on the pulse (never EvChatMedia).
	child := mediaFrame(t, map[string]any{
		"id": "part-2", "sessionID": "ses-child", "messageID": "msg-2",
		"type": "file", "mime": "image/png", "filename": "kid.png", "url": url,
	})
	for _, ev := range mapOCEvent(child, ctx, "ses-primary", 300) {
		if ev.Kind == state.EvChatMedia {
			t.Fatalf("a child's file part must never surface as EvChatMedia: %+v", ev)
		}
	}
}

// TestMediaFromPartsGates — the gate matrix over mediaFromParts: non-image
// mimes skip, remote URLs degrade to chip-only (no URL, no hash — the
// client never fetches), undecodable bytes degrade the same, >4 payload
// images cap the runnable set (extras chip rows), and non-file parts
// never even register.
func TestMediaFromPartsGates(t *testing.T) {
	pngBytes := tinyPNG(t)
	good := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	mk := func(id string, p ocPart) ocPart { p.ID, p.SessionID, p.MessageID = id, "s", "m"; return p }
	var parts []ocPart
	// 5 runnable images (cap is 4) + remote + broken + non-image + text part.
	for _, name := range []string{"a.png", "b.png", "c.png", "d.png", "e.png"} {
		parts = append(parts, mk(name, ocPart{Type: "file", Mime: "image/png", Filename: name, URL: good}))
	}
	parts = append(parts,
		mk("remote", ocPart{Type: "file", Mime: "image/png", Filename: "remote.png", URL: "https://example.com/x.png"}),
		mk("broken", ocPart{Type: "file", Mime: "image/png", Filename: "broken.png",
			URL: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("garbage"))}),
		mk("html", ocPart{Type: "file", Mime: "text/html", Filename: "note.html", URL: good}),
		mk("txt", ocPart{Type: "text", Text: "hello"}),
	)
	items := mediaFromParts(parts)
	if len(items) != 7 { // 5 images + remote + broken — html/text never register
		t.Fatalf("want 7 media items, got %d: %+v", len(items), items)
	}
	runnable := 0
	for _, it := range items {
		if it.Hash != "" && it.URL != "" {
			runnable++
		}
		if it.Mime == "" {
			t.Fatalf("every surfaced item keeps its mime: %+v", it)
		}
	}
	if runnable != 4 {
		t.Fatalf("the turn caps at 4 runnable previews, got %d", runnable)
	}
	// e.png (the 5th) is the chip-row extra: dims ride, payload does not.
	if items[4].Filename != "e.png" || items[4].Hash != "" || items[4].URL != "" || items[4].W != 2 || items[4].H != 2 {
		t.Fatalf("the 5th image degrades to a chip row (dims only): %+v", items[4])
	}
	// remote: chip-only, NO URL (a fetch can never be triggered client-side).
	if items[5].Hash != "" || items[5].URL != "" || items[5].W != 0 {
		t.Fatalf("remote URLs degrade chip-only (nothing fetchable rides): %+v", items[5])
	}
	// broken payload: same degrade.
	if items[6].Hash != "" || items[6].URL != "" {
		t.Fatalf("undecodable payloads degrade chip-only: %+v", items[6])
	}
}

// TestSessionMessageRowFileRetention — history keeps the file part's
// URL/mime/filename verbatim alongside text parts (the splice's READING
// story; previews stay a live-lane thing).
func TestSessionMessageRowFileRetention(t *testing.T) {
	var info ocMessage
	info.ID = "msg-9"
	info.Role = "assistant"
	row := sessionMessageRow(info, []ocPart{
		{Type: "text", Text: "see the diagram"},
		{Type: "file", URL: "data:image/png;base64,AAAA", Mime: "image/png", Filename: "diagram.png"},
		{Type: "tool"},
	})
	if len(row.Parts) != 3 {
		t.Fatalf("want 3 retained parts, got %+v", row.Parts)
	}
	f := row.Parts[1]
	if f.Type != "file" || f.URL != "data:image/png;base64,AAAA" || f.Mime != "image/png" || f.Filename != "diagram.png" {
		t.Fatalf("file part must retain url/mime/filename: %+v", f)
	}
	if row.Parts[0].Text != "see the diagram" || row.Parts[2].Type != "tool" {
		t.Fatalf("the old text/tool shapes stay byte-identical: %+v", row.Parts)
	}
}
