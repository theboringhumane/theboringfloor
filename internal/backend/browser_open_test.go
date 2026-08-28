// browser_open_test.go — the backend legs of the agent browser tool
// (internal/browsertools — the package owns the policy/protocol/bridge
// units; THIS file pins the two backends' wiring):
//
//	opencode: the preamble rides the FIRST prompt to each primary
//	     session (raw echo stays clean, a fresh primary re-briefs, a
//	     FAILED first post re-briefs next Send — never spent on a 500);
//	     a marker in the completing message scrubs off the pinned bubble
//	     and lands as ONE policy-decided EvBrowserOpen;
//	claude: the preamble rides the FIRST stdin user line (the second
//	     ships raw); a marker on the completion pin scrubs + fires the
//	     event BEFORE the pin lands.
package backend

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/browsertools"
	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/state"
)

// browserOpens filters an opencode-side event log down to the tool events.
func browserOpens(log *eventLog) []state.Event {
	return eventsMatching(log, func(e state.Event) bool { return e.Kind == state.EvBrowserOpen })
}

// ---------------------------------------------------------------- opencode

// TestBrowserPreambleRidesFirstBossPrompt — the FIRST Send to a primary
// session carries browsertools.PromptPreamble ahead of the member text
// (the conciergePreamble house pattern); the SECOND ships raw; the
// member's chat-user echo NEVER carries the preamble; and a NEW primary
// id (respawn/fresh) re-briefs exactly once.
func TestBrowserPreambleRidesFirstBossPrompt(t *testing.T) {
	stub := &modelStub{}
	srv := stub.serve(t)
	b := liveStubBackend(stub, srv, config.Default())
	log := &eventLog{}
	b.fl.setEmit(log.emit)

	if err := b.Send("open the docs please"); err != nil {
		t.Fatalf("Send 1: %v", err)
	}
	if err := b.Send("thanks"); err != nil {
		t.Fatalf("Send 2: %v", err)
	}
	posts := stub.promptPosts()
	if len(posts) != 2 {
		t.Fatalf("want 2 prompt POSTs, got %d", len(posts))
	}
	if !strings.Contains(posts[0], browsertools.MarkerOpen) ||
		!strings.Contains(posts[0], "open the docs please") {
		t.Fatalf("the FIRST prompt must carry the preamble + the member text, got %s", posts[0])
	}
	if strings.Contains(posts[1], browsertools.MarkerOpen) {
		t.Fatalf("the SECOND prompt ships raw (no re-brief), got %s", posts[1])
	}
	if !strings.Contains(posts[1], "thanks") {
		t.Fatalf("the SECOND prompt still carries the member text, got %s", posts[1])
	}
	for _, e := range eventsMatching(log, func(e state.Event) bool { return e.Kind == state.EvChatUser }) {
		if strings.Contains(e.Msg.Text, browsertools.MarkerOpen) {
			t.Fatalf("the member echo must stay preamble-free, got %q", e.Msg.Text)
		}
	}

	// a fresh primary id (respawn /new) has no memory of the contract.
	b.mu.Lock()
	b.primaryID = "ses-boss-2"
	b.mu.Unlock()
	if err := b.Send("fresh session"); err != nil {
		t.Fatalf("Send 3 (new primary): %v", err)
	}
	posts = stub.promptPosts()
	if len(posts) != 3 || !strings.Contains(posts[2], browsertools.MarkerOpen) {
		t.Fatalf("a NEW primary must re-brief on its first prompt, got %v", posts)
	}
}

// TestBrowserPreambleRebriefsAfterFailure — the latch sets ONLY on a
// clean post: a 500 on the first prompt means the second Send carries
// the preamble again (never spent on a failure).
func TestBrowserPreambleRebriefsAfterFailure(t *testing.T) {
	var failNext bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_, _ = io.ReadAll(r.Body)
			if strings.HasSuffix(r.URL.Path, "/prompt_async") && failNext {
				failNext = false
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if r.URL.Path == "/session" {
				w.Write([]byte(`{"id":"ses-made","title":""}`))
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)

	b := newLiveBackend("", t.TempDir(), config.Default())
	b.mu.Lock()
	b.baseURL = srv.URL
	b.primaryID = "ses-boss"
	b.mu.Unlock()
	log := &eventLog{}
	b.fl.setEmit(log.emit)

	failNext = true
	if err := b.Send("first attempt"); err != nil {
		t.Fatalf("Send 1 (the failed one) still returns nil: %v", err)
	}
	b.mu.Lock()
	latched := b.browserBriefedFor
	b.mu.Unlock()
	if latched != "" {
		t.Fatalf("a failed post must NOT spend the brief, latched %q", latched)
	}
	if err := b.Send("second attempt"); err != nil {
		t.Fatalf("Send 2: %v", err)
	}
	b.mu.Lock()
	latched = b.browserBriefedFor
	b.mu.Unlock()
	if latched != "ses-boss" {
		t.Fatalf("the clean retry must latch the brief, got %q", latched)
	}
}

// TestBrowserMarkerScrubbedAtPin — opencode lane: the completing
// message's own text carries one whole-line marker; the pinned bubble
// shows ONLY the prose, and ONE allowed EvBrowserOpen carries the URL.
func TestBrowserMarkerScrubbedAtPin(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	rows := `[
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]},
	  {"info":{"id":"m-1","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]}
	]`
	// the fixture embeds the text raw into JSON — \n must arrive ESCAPED.
	b, log := reconcileFixture(t, rows, map[string]string{
		"m-1": "Sure — opening the docs now.\\n⟦open-browser: https://theboring.name⟧",
	})
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	b.sseRecovered()

	pins := bossPins(log)
	if len(pins) != 1 {
		t.Fatalf("want ONE boss pin, got %d", len(pins))
	}
	if pins[0].Msg.Text != "Sure — opening the docs now." {
		t.Fatalf("the pinned text must be scrubbed, got %q", pins[0].Msg.Text)
	}
	opens := browserOpens(log)
	if len(opens) != 1 {
		t.Fatalf("want ONE EvBrowserOpen, got %d: %+v", len(opens), opens)
	}
	if opens[0].Text != "https://theboring.name" || !opens[0].BrowserOpenAllowed ||
		opens[0].BrowserOpenReason != "" {
		t.Fatalf("the open event must carry the allowed URL verdict, got %+v", opens[0])
	}
}

// TestBrowserMarkerBlockedNotice — a plain-http non-localhost marker
// with the flag OFF: the marker still scrubs (the transcript never
// shows directives), the marker-only bubble degrades to the refusal
// note, and the EvBrowserOpen carries the exact reason for the app's
// red notice.
func TestBrowserMarkerBlockedNotice(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	rows := `[
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]},
	  {"info":{"id":"m-1","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]}
	]`
	b, log := reconcileFixture(t, rows, map[string]string{
		"m-1": "⟦open-browser: http://theboring.name⟧",
	})
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	b.sseRecovered()

	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	pins := bossPins(log)
	if len(pins) != 1 {
		t.Fatalf("want ONE boss pin, got %d", len(pins))
	}
	if pins[0].Msg.Text != "[theboringoffice] open-browser refused: "+reason {
		t.Fatalf("the marker-only bubble must degrade to the refusal note, got %q", pins[0].Msg.Text)
	}
	opens := browserOpens(log)
	if len(opens) != 1 || opens[0].BrowserOpenAllowed || opens[0].BrowserOpenReason != reason {
		t.Fatalf("the refused event must carry the exact reason, got %+v", opens)
	}
}

// TestBrowserMarkerFlagOnAllows — the SAME plain-http request with the
// member's flag exported opens cleanly (the env is read at use time).
func TestBrowserMarkerFlagOnAllows(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "1")
	rows := `[
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]},
	  {"info":{"id":"m-1","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]}
	]`
	b, log := reconcileFixture(t, rows, map[string]string{
		"m-1": "⟦open-browser: http://theboring.name⟧",
	})
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	b.sseRecovered()

	opens := browserOpens(log)
	if len(opens) != 1 || !opens[0].BrowserOpenAllowed || opens[0].Text != "http://theboring.name" {
		t.Fatalf("the member's flag must allow plain http, got %+v", opens)
	}
	pins := bossPins(log)
	if len(pins) != 1 || pins[0].Msg.Text != "[theboringoffice] open-browser: http://theboring.name" {
		t.Fatalf("the marker-only bubble degrades to the open note, got %+v", pins)
	}
}

// TestBrowserShotSnapMarkersScrubAtPin — opencode lane: the new
// read-only markers scrub off the pinned bubble exactly like the open
// marker, and each lands as its OWN event kind carrying the allowed
// verdict (screenshot) / the exact refusal reason (snapshot).
func TestBrowserShotSnapMarkersScrubAtPin(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	rows := `[
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]},
	  {"info":{"id":"m-1","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]}
	]`
	// the fixture embeds the text raw into JSON — \n must arrive ESCAPED.
	b, log := reconcileFixture(t, rows, map[string]string{
		"m-1": "Rendering and reading it now.\\n⟦browser-screenshot: https://theboring.name/docs⟧\\n⟦browser-snapshot: http://theboring.name⟧",
	})
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	b.sseRecovered()

	pins := bossPins(log)
	if len(pins) != 1 {
		t.Fatalf("want ONE boss pin, got %d", len(pins))
	}
	if pins[0].Msg.Text != "Rendering and reading it now." {
		t.Fatalf("the pinned text must be scrubbed of BOTH new markers, got %q", pins[0].Msg.Text)
	}
	tool := eventsMatching(log, func(e state.Event) bool {
		return e.Kind == state.EvBrowserScreenshot || e.Kind == state.EvBrowserSnapshot
	})
	if len(tool) != 2 {
		t.Fatalf("want ONE EvBrowserScreenshot + ONE EvBrowserSnapshot, got %+v", tool)
	}
	if tool[0].Kind != state.EvBrowserScreenshot || tool[0].Text != "https://theboring.name/docs" ||
		!tool[0].BrowserOpenAllowed || tool[0].BrowserOpenReason != "" {
		t.Fatalf("the screenshot event must carry the allowed verdict on its own kind, got %+v", tool[0])
	}
	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	if tool[1].Kind != state.EvBrowserSnapshot || tool[1].Text != "http://theboring.name" ||
		tool[1].BrowserOpenAllowed || tool[1].BrowserOpenReason != reason {
		t.Fatalf("the snapshot refusal must carry the exact reason on its own kind, got %+v", tool[1])
	}
}

// TestBrowserShotMarkerOnlyFallback — opencode lane: a screenshot-ONLY
// reply scrubs to the kind-named office note (the pinned bubble never
// goes blank).
func TestBrowserShotMarkerOnlyFallback(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	rows := `[
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]},
	  {"info":{"id":"m-1","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]}
	]`
	b, log := reconcileFixture(t, rows, map[string]string{
		"m-1": "⟦browser-screenshot: https://theboring.name⟧",
	})
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	b.sseRecovered()

	pins := bossPins(log)
	if len(pins) != 1 || pins[0].Msg.Text != "[theboringoffice] browser-screenshot: https://theboring.name" {
		t.Fatalf("the screenshot-only bubble degrades to the kind-named note, got %+v", pins)
	}
	shots := eventsMatching(log, func(e state.Event) bool { return e.Kind == state.EvBrowserScreenshot })
	if len(shots) != 1 || !shots[0].BrowserOpenAllowed || shots[0].Text != "https://theboring.name" {
		t.Fatalf("the screenshot event carries the allowed verdict, got %+v", shots)
	}
}

// TestBrowserActionMarkerScrubbedAtPin — opencode lane: the MUTATING
// marker scrubs off the pinned bubble exactly like its read-only
// siblings and lands as ONE EvBrowserAction carrying the allowed
// verdict AND the parsed payload (op/sel/arg) — the app parks THAT
// behind the member's permission modal. A refused action marker
// degrades the bubble to the kind-named refusal note and the event
// carries the exact reason (the app posts the red row, NO modal).
func TestBrowserActionMarkerScrubbedAtPin(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	rows := `[
	  {"info":{"id":"u-1","sessionID":"ses-primary","role":"user","time":{"created":100}},"parts":[]},
	  {"info":{"id":"m-1","sessionID":"ses-primary","role":"assistant","finish":"stop","time":{"created":110,"completed":120},"cost":0,"tokens":{"input":1,"output":1,"reasoning":0,"cache":{"read":0,"write":0}}},"parts":[]}
	]`
	// the fixture embeds the text raw into JSON — \n must arrive ESCAPED.
	b, log := reconcileFixture(t, rows, map[string]string{
		"m-1": "Clicking buy now.\\n⟦browser-action: https://theboring.name | click: #buy⟧",
	})
	b.mu.Lock()
	b.pendingBoss = []string{"boss-1"}
	b.mu.Unlock()

	b.sseRecovered()

	pins := bossPins(log)
	if len(pins) != 1 {
		t.Fatalf("want ONE boss pin, got %d", len(pins))
	}
	if pins[0].Msg.Text != "Clicking buy now." {
		t.Fatalf("the pinned text must be scrubbed of the action marker, got %q", pins[0].Msg.Text)
	}
	acts := eventsMatching(log, func(e state.Event) bool { return e.Kind == state.EvBrowserAction })
	if len(acts) != 1 {
		t.Fatalf("want ONE EvBrowserAction, got %+v", acts)
	}
	if acts[0].Text != "https://theboring.name" || !acts[0].BrowserOpenAllowed || acts[0].BrowserOpenReason != "" ||
		acts[0].BrowserActionOp != "click" || acts[0].BrowserActionSel != "#buy" || acts[0].BrowserActionArg != "" {
		t.Fatalf("the action event must carry the allowed verdict + payload, got %+v", acts[0])
	}

	// a refused action marker (plain http, flag off): scrubs to the
	// kind-named refusal note + the reason event on the action kind.
	b2, log2 := reconcileFixture(t, rows, map[string]string{
		"m-1": "⟦browser-action: http://theboring.name | fill: #q = a = b⟧",
	})
	b2.mu.Lock()
	b2.pendingBoss = []string{"boss-1"}
	b2.mu.Unlock()
	b2.sseRecovered()

	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	pins2 := bossPins(log2)
	if len(pins2) != 1 || pins2[0].Msg.Text != "[theboringoffice] browser-action refused: "+reason {
		t.Fatalf("the refused action-only bubble degrades to the kind-named note, got %+v", pins2)
	}
	acts2 := eventsMatching(log2, func(e state.Event) bool { return e.Kind == state.EvBrowserAction })
	if len(acts2) != 1 || acts2[0].BrowserOpenAllowed || acts2[0].BrowserOpenReason != reason ||
		acts2[0].BrowserActionOp != "fill" || acts2[0].BrowserActionSel != "#q" || acts2[0].BrowserActionArg != "a = b" {
		t.Fatalf("the refused action event carries the exact reason + payload, got %+v", acts2)
	}

	// a MALFORMED action marker never extracts: the pinned text keeps it
	// VISIBLE, untouched, and no event fires.
	b3, log3 := reconcileFixture(t, rows, map[string]string{
		"m-1": "⟦browser-action: https://theboring.name | click:⟧",
	})
	b3.mu.Lock()
	b3.pendingBoss = []string{"boss-1"}
	b3.mu.Unlock()
	b3.sseRecovered()
	pins3 := bossPins(log3)
	if len(pins3) != 1 || pins3[0].Msg.Text != "⟦browser-action: https://theboring.name | click:⟧" {
		t.Fatalf("a malformed action marker stays visible verbatim, got %+v", pins3)
	}
	if acts3 := eventsMatching(log3, func(e state.Event) bool { return e.Kind == state.EvBrowserAction }); len(acts3) != 0 {
		t.Fatalf("a malformed action marker must never emit, got %+v", acts3)
	}
}

// ---------------------------------------------------------------- claude

// TestClaudeBrowserPreambleRidesFirstLine — the FIRST stdin user line
// carries the preamble ahead of the member text (exact placement via
// the production encoder + the literal marker intro); the second line
// ships raw; the member's chat-user echo stays preamble-free.
func TestClaudeBrowserPreambleRidesFirstLine(t *testing.T) {
	capture := filepath.Join(t.TempDir(), "capture.log")
	stubBody := claudeStubPreambleSh() + `while IFS= read -r line; do
  printf '%s\n' "$line" >> "` + capture + `"
done
`
	stub := claudeStubScript(t, stubBody)
	log := &claudeEventLog{}
	b := newClaudeBackend(stub, t.TempDir(), nil)
	if err := b.Start(log.emit); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = b.Stop() }()

	if err := b.Send("open the docs please"); err != nil {
		t.Fatalf("Send 1: %v", err)
	}
	// 8s budgets (not the usual 2s): the sh stub is a real spawned child
	// and full-suite runs co-schedule the whole repo on a multi-tenant
	// machine — the fast path is unchanged (claudeWait polls), the wide
	// window only absorbs spawn stalls.
	claudeWait(t, "the first user line in the capture", 8*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 1
	})
	if err := b.Send("thanks"); err != nil {
		t.Fatalf("Send 2: %v", err)
	}
	claudeWait(t, "the second user line in the capture", 8*time.Second, func() bool {
		return len(claudeCapture(t, capture)) == 2
	})
	lines := claudeCapture(t, capture)
	if len(lines) != 2 {
		t.Fatalf("want 2 stdin lines, got %d", len(lines))
	}
	wantFirst := string(claudeUserLineFor(browsertools.PromptPreamble + "\n\nopen the docs please"))
	if lines[0] != wantFirst {
		t.Fatalf("the first line must be preamble + prompt:\n got %q\nwant %q", lines[0], wantFirst)
	}
	if !strings.Contains(lines[0], "⟦open-browser:") {
		t.Fatalf("the literal marker intro must ride the first line, got %q", lines[0])
	}
	wantSecond := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"thanks"}]},"parent_tool_use_id":null}`
	if lines[1] != wantSecond {
		t.Fatalf("the second line ships raw:\n got %q\nwant %q", lines[1], wantSecond)
	}
	for _, e := range log.snapshot() {
		if e.Kind == state.EvChatUser && strings.Contains(e.Msg.Text, browsertools.MarkerOpen) {
			t.Fatalf("the member echo must stay preamble-free, got %q", e.Msg.Text)
		}
	}
}

// TestClaudeBrowserMarkerScrubbedAtPin — the completion pin's text
// scrubs markers and fires the open request BEFORE the pin lands
// (deltas are never scrubbed — the pin supersedes them).
func TestClaudeBrowserMarkerScrubbedAtPin(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	b := newClaudeBackend("true", t.TempDir(), nil) // bin never spawned (no Start)
	log := &claudeEventLog{}
	b.fl.setEmit(log.emit)

	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m1", From: "boss", Kind: "boss",
		Text: "Sure — opening the docs now.\n⟦open-browser: https://theboring.name⟧", Pending: false,
	}})

	evs := log.snapshot()
	if len(evs) != 2 {
		t.Fatalf("want EvBrowserOpen THEN the pin (2 events), got %d: %+v", len(evs), evs)
	}
	if evs[0].Kind != state.EvBrowserOpen || evs[0].Text != "https://theboring.name" ||
		!evs[0].BrowserOpenAllowed {
		t.Fatalf("the open request lands FIRST with the allowed verdict, got %+v", evs[0])
	}
	if evs[1].Kind != state.EvChatBoss || evs[1].Msg.Text != "Sure — opening the docs now." {
		t.Fatalf("the pin lands SECOND, scrubbed, got %+v", evs[1])
	}

	// a refused marker-only pin degrades to the refusal note + reason event.
	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m2", From: "boss", Kind: "boss",
		Text: "⟦open-browser: http://theboring.name⟧", Pending: false,
	}})
	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	evs = log.snapshot()
	if len(evs) != 4 {
		t.Fatalf("the refused pin adds 2 events (4 total), got %d", len(evs))
	}
	if evs[2].Kind != state.EvBrowserOpen || evs[2].BrowserOpenAllowed ||
		evs[2].BrowserOpenReason != reason {
		t.Fatalf("the refused event carries the exact reason, got %+v", evs[2])
	}
	if evs[3].Msg.Text != "[theboringoffice] open-browser refused: "+reason {
		t.Fatalf("the refused marker-only pin degrades to the note, got %q", evs[3].Msg.Text)
	}

	// a marker-less pin passes through untouched and emits nothing extra.
	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m3", From: "boss", Kind: "boss", Text: "plain reply", Pending: false,
	}})
	if evs := log.snapshot(); len(evs) != 5 || evs[4].Msg.Text != "plain reply" {
		t.Fatalf("a clean pin is identity + silence, got %+v", evs)
	}
}

// TestClaudeBrowserShotSnapScrubbedAtPin — the new read-only markers
// scrub off the claude completion pin exactly like the open marker:
// each fires its OWN event kind BEFORE the pin lands, in order of
// appearance, and the pinned text keeps only the prose.
func TestClaudeBrowserShotSnapScrubbedAtPin(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	b := newClaudeBackend("true", t.TempDir(), nil) // bin never spawned (no Start)
	log := &claudeEventLog{}
	b.fl.setEmit(log.emit)

	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m1", From: "boss", Kind: "boss",
		Text:    "Rendering and reading it now.\n⟦browser-screenshot: https://theboring.name/docs⟧\n⟦browser-snapshot: https://theboring.name/api⟧",
		Pending: false,
	}})

	evs := log.snapshot()
	if len(evs) != 3 {
		t.Fatalf("want screenshot + snapshot events THEN the pin (3 events), got %d: %+v", len(evs), evs)
	}
	if evs[0].Kind != state.EvBrowserScreenshot || evs[0].Text != "https://theboring.name/docs" ||
		!evs[0].BrowserOpenAllowed {
		t.Fatalf("the screenshot request lands FIRST with the allowed verdict, got %+v", evs[0])
	}
	if evs[1].Kind != state.EvBrowserSnapshot || evs[1].Text != "https://theboring.name/api" ||
		!evs[1].BrowserOpenAllowed {
		t.Fatalf("the snapshot request lands SECOND with the allowed verdict, got %+v", evs[1])
	}
	if evs[2].Kind != state.EvChatBoss || evs[2].Msg.Text != "Rendering and reading it now." {
		t.Fatalf("the pin lands LAST, scrubbed of both new markers, got %+v", evs[2])
	}

	// a refused screenshot-only pin degrades to the kind-named refusal
	// note + the reason event on the screenshot kind.
	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m2", From: "boss", Kind: "boss",
		Text: "⟦browser-screenshot: http://theboring.name⟧", Pending: false,
	}})
	const reason = "plain http to theboring.name refused — export THEBORINGOFFICE_BROWSER_ALLOW_HTTP=1 to allow outbound http pages"
	evs = log.snapshot()
	if len(evs) != 5 {
		t.Fatalf("the refused pin adds 2 events (5 total), got %d", len(evs))
	}
	if evs[3].Kind != state.EvBrowserScreenshot || evs[3].BrowserOpenAllowed ||
		evs[3].BrowserOpenReason != reason {
		t.Fatalf("the refused screenshot event carries the exact reason, got %+v", evs[3])
	}
	if evs[4].Msg.Text != "[theboringoffice] browser-screenshot refused: "+reason {
		t.Fatalf("the refused screenshot-only pin degrades to the kind-named note, got %q", evs[4].Msg.Text)
	}
}

// TestClaudeBrowserActionScrubbedAtPin — the MUTATING marker scrubs off
// the claude completion pin exactly like its read-only siblings: ONE
// EvBrowserAction (allowed verdict + parsed payload) fires BEFORE the
// pin lands, and the pinned text keeps only the prose.
func TestClaudeBrowserActionScrubbedAtPin(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	b := newClaudeBackend("true", t.TempDir(), nil) // bin never spawned (no Start)
	log := &claudeEventLog{}
	b.fl.setEmit(log.emit)

	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m1", From: "boss", Kind: "boss",
		Text:    "Reading the price first.\n⟦browser-action: https://theboring.name | eval: document.querySelector('#price').textContent⟧",
		Pending: false,
	}})

	evs := log.snapshot()
	if len(evs) != 2 {
		t.Fatalf("want the action event THEN the pin (2 events), got %d: %+v", len(evs), evs)
	}
	if evs[0].Kind != state.EvBrowserAction || evs[0].Text != "https://theboring.name" ||
		!evs[0].BrowserOpenAllowed || evs[0].BrowserActionOp != "eval" ||
		evs[0].BrowserActionArg != "document.querySelector('#price').textContent" {
		t.Fatalf("the action request lands FIRST with the allowed verdict + payload, got %+v", evs[0])
	}
	if evs[1].Kind != state.EvChatBoss || evs[1].Msg.Text != "Reading the price first." {
		t.Fatalf("the pin lands SECOND, scrubbed of the action marker, got %+v", evs[1])
	}

	// a MALFORMED action marker stays VISIBLE on the pin and emits nothing.
	b.emitMapped(state.Event{Kind: state.EvChatBoss, Msg: state.ChatMsg{
		ID: "bossmsg-m2", From: "boss", Kind: "boss",
		Text: "⟦browser-action: https://theboring.name | frobnicate: #a⟧", Pending: false,
	}})
	evs = log.snapshot()
	if len(evs) != 3 {
		t.Fatalf("a malformed action marker adds ONLY the pin (3 events total), got %d: %+v", len(evs), evs)
	}
	if evs[2].Msg.Text != "⟦browser-action: https://theboring.name | frobnicate: #a⟧" {
		t.Fatalf("the malformed marker stays visible verbatim, got %q", evs[2].Msg.Text)
	}
}
