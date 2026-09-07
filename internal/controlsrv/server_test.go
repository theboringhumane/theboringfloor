package controlsrv

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const testToken = "test-control-token"

type routeCoverageCase struct {
	name, method, wrongMethod, path, body string
}

var controlRouteCoverageCases = []routeCoverageCase{
	{"RouteHealth", http.MethodGet, http.MethodPost, control.RouteHealth, ""},
	{"RoutePlan", http.MethodGet, http.MethodPost, control.RoutePlan, ""},
	{"RoutePlanPresent", http.MethodPost, http.MethodGet, control.RoutePlanPresent, `{"text":"plan"}`},
	{"RoutePlanUpdate", http.MethodPost, http.MethodGet, control.RoutePlanUpdate, `{"text":"plan"}`},
	{"RouteTranscript", http.MethodGet, http.MethodPost, control.RouteTranscript, ""},
	{"RouteStatus", http.MethodGet, http.MethodPost, control.RouteStatus, ""},
	{"RouteMessage", http.MethodPost, http.MethodGet, control.RouteMessage, `{"text":"message"}`},
	{"RouteStop", http.MethodPost, http.MethodGet, control.RouteStop, ""},
	{"RouteSessionNew", http.MethodPost, http.MethodGet, control.RouteSessionNew, ""},
	{"RouteBusy", http.MethodGet, http.MethodPost, control.RouteBusy, ""},
}

type fakeSink struct {
	mu       sync.Mutex
	events   []state.Event
	registry *control.Registry
	respond  bool
	release  <-chan struct{}
}

func (f *fakeSink) send(event state.Event) {
	f.mu.Lock()
	f.events = append(f.events, event)
	f.mu.Unlock()
	if event.Kind != state.EvControlQuery || !f.respond {
		return
	}
	payload := cannedPayload(event.ControlQuery)
	go func() {
		if f.release != nil {
			<-f.release
		}
		f.registry.Fulfill(event.ControlReqID, payload)
	}()
}

func (f *fakeSink) snapshot() []state.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]state.Event(nil), f.events...)
}

func cannedPayload(query string) []byte {
	if strings.HasPrefix(query, control.QueryTranscript+"?page=1") {
		if strings.Contains(query, "before=unknown") {
			return []byte(`{"error":"unknown before cursor"}`)
		}
		return []byte(`{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":42}],"truncated":false,"hasMore":true}`)
	}
	switch query {
	case control.QueryPlan:
		return []byte(`{"draft":"draft plan","approved":"approved plan","hasApproved":true}`)
	case control.QueryTranscript:
		return []byte(`{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":42}],"truncated":false}`)
	case control.QueryBusy:
		return []byte(`{"busy":true,"pendingBoss":true,"thinking":true,"delegating":false,"questionParked":false}`)
	default:
		return []byte(`{"dir":"/workspace","backend":"opencode","primaryId":"ses_1","planDraftLen":10,"planApprovedLen":13,"chatCount":1}`)
	}
}

func newTestServer(t *testing.T, respond bool, timeout time.Duration) (*Server, *fakeSink, string) {
	return newTestServerWithOptions(t, respond, timeout, 0, nil)
}

func newTestServerWithOptions(t *testing.T, respond bool, timeout time.Duration, maxReads int, release <-chan struct{}) (*Server, *fakeSink, string) {
	t.Helper()
	registry := control.NewRegistry()
	fake := &fakeSink{registry: registry, respond: respond, release: release}
	server := New(Options{
		Dir: "/workspace", Version: "v1.2.3", Token: testToken, Sink: fake.send,
		Registry: registry, QueryTimeout: timeout, MaxInFlightReads: maxReads,
		MaxConnections: 128,
	})
	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server, fake, "http://127.0.0.1:" + strconv.Itoa(server.Port())
}

func request(t *testing.T, client *http.Client, method, url string, body io.Reader, auth string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	return response.StatusCode, string(payload)
}

func TestRoutesHappyPath(t *testing.T) {
	_, fake, baseURL := newTestServer(t, true, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken

	tests := []struct {
		name, method, path, body, want string
	}{
		{"health", http.MethodGet, control.RouteHealth, "", `{"ok":true,"dir":"/workspace","version":"v1.2.3","backend":"opencode"}`},
		{"plan", http.MethodGet, control.RoutePlan, "", `{"draft":"draft plan","approved":"approved plan","hasApproved":true}`},
		{"plan present", http.MethodPost, control.RoutePlanPresent, `{"text":"  proposed plan  "}`, `{"ok":true}`},
		{"plan update", http.MethodPost, control.RoutePlanUpdate, `{"text":"  revised plan  "}`, `{"ok":true}`},
		{"transcript", http.MethodGet, control.RouteTranscript + "?limit=12", "", `{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":42}],"truncated":false,"hasMore":true}`},
		{"status", http.MethodGet, control.RouteStatus, "", `{"dir":"/workspace","backend":"opencode","primaryId":"ses_1","planDraftLen":10,"planApprovedLen":13,"chatCount":1}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body io.Reader
			if test.body != "" {
				body = bytes.NewBufferString(test.body)
			}
			status, got := request(t, client, test.method, baseURL+test.path, body, auth)
			if status != http.StatusOK {
				t.Fatalf("status = %d, body = %s", status, got)
			}
			assertJSONEqual(t, test.want, got)
		})
	}

	events := fake.snapshot()
	if len(events) != 6 {
		t.Fatalf("event count = %d, want 6", len(events))
	}
	assertQuery(t, events[0], control.QueryStatus, 0)
	assertQuery(t, events[1], control.QueryPlan, 0)
	if events[2].Kind != state.EvPlanPresent || events[2].PlanToolText != "proposed plan" {
		t.Fatalf("present event = %#v", events[2])
	}
	if events[3].Kind != state.EvPlanUpdate || events[3].PlanToolText != "revised plan" {
		t.Fatalf("update event = %#v", events[3])
	}
	if events[4].ControlQuery != control.QueryTranscript+"?page=1" || events[4].ControlLimit != 12 {
		t.Fatalf("transcript query event = %#v", events[4])
	}
	assertQuery(t, events[5], control.QueryStatus, 0)
}

func TestAuthNotFoundAndMethodErrors(t *testing.T) {
	_, _, baseURL := newTestServer(t, true, time.Second)
	client := &http.Client{Timeout: time.Second}
	for _, test := range []struct {
		name, method, path, auth string
		want                     int
	}{
		{"missing auth", http.MethodGet, control.RouteHealth, "", http.StatusUnauthorized},
		{"wrong scheme", http.MethodGet, control.RouteHealth, "Token " + testToken, http.StatusUnauthorized},
		{"bad token", http.MethodGet, control.RouteHealth, "Bearer wrong", http.StatusUnauthorized},
		{"not found", http.MethodGet, "/v1/nope", "Bearer " + testToken, http.StatusNotFound},
		{"wrong method", http.MethodPost, control.RoutePlan, "Bearer " + testToken, http.StatusMethodNotAllowed},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, got := request(t, client, test.method, baseURL+test.path, nil, test.auth)
			if status != test.want {
				t.Fatalf("status = %d, want %d; body = %s", status, test.want, got)
			}
		})
	}
}

func TestEveryDeclaredControlRouteIsServed(t *testing.T) {
	_, _, baseURL := newTestServer(t, true, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken

	declared := declaredControlRoutes(t)
	described := make(map[string]struct{}, len(controlRouteCoverageCases))
	for _, route := range controlRouteCoverageCases {
		if _, duplicate := described[route.name]; duplicate {
			t.Fatalf("route coverage contains duplicate descriptor for control.%s", route.name)
		}
		described[route.name] = struct{}{}
	}
	for name := range declared {
		if _, ok := described[name]; !ok {
			t.Fatalf("route coverage is missing descriptor for control.%s", name)
		}
	}
	for name := range described {
		if !declared[name] {
			t.Fatalf("route coverage descriptor control.%s has no matching declared route constant", name)
		}
	}

	for _, route := range controlRouteCoverageCases {
		t.Run(route.name, func(t *testing.T) {
			var body io.Reader
			if route.body != "" {
				body = bytes.NewBufferString(route.body)
			}
			status, got := request(t, client, route.method, baseURL+route.path, body, auth)
			if status == http.StatusNotFound {
				t.Fatalf("route coverage: control.%s %s %s returned 404; add a handler to controlsrv.Server.ServeHTTP; body = %s", route.name, route.method, route.path, got)
			}
		})
	}
}

func TestEveryDeclaredControlRouteRejectsWrongMethod(t *testing.T) {
	_, _, baseURL := newTestServer(t, true, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken

	for _, route := range controlRouteCoverageCases {
		t.Run(route.name, func(t *testing.T) {
			status, got := request(t, client, route.wrongMethod, baseURL+route.path, nil, auth)
			if status != http.StatusMethodNotAllowed {
				t.Fatalf("control.%s wrong-method status = %d, want 405; body = %s", route.name, status, got)
			}
		})
	}
}

func declaredControlRoutes(t *testing.T) map[string]bool {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("route coverage: unable to locate server_test.go")
	}
	controlFile := filepath.Join(filepath.Dir(testFile), "..", "control", "control.go")
	file, err := parser.ParseFile(token.NewFileSet(), controlFile, nil, 0)
	if err != nil {
		t.Fatalf("route coverage: parse %s: %v", controlFile, err)
	}
	routes := make(map[string]bool)
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, specification := range general.Specs {
			value, ok := specification.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range value.Names {
				if strings.HasPrefix(name.Name, "Route") {
					routes[name.Name] = true
				}
			}
		}
	}
	if len(routes) == 0 {
		t.Fatalf("route coverage: no control.Route* constants found in %s", controlFile)
	}
	return routes
}

func TestPlanWriteValidationAndFireAndForget(t *testing.T) {
	_, fake, baseURL := newTestServer(t, false, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken
	for _, test := range []struct {
		name, body string
		want       int
	}{
		{"empty", `{"text":" \n\t "}`, http.StatusBadRequest},
		{"non JSON", "not json", http.StatusBadRequest},
		{"oversized", `{"text":"` + string(bytes.Repeat([]byte("x"), maxPlanBodyBytes)) + `"}`, http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, _ := request(t, client, http.MethodPost, baseURL+control.RoutePlanPresent, bytes.NewBufferString(test.body), auth)
			if status != test.want {
				t.Fatalf("status = %d, want %d", status, test.want)
			}
		})
	}
	status, got := request(t, client, http.MethodPost, baseURL+control.RoutePlanPresent, bytes.NewBufferString(`{"text":"  now  "}`), auth)
	if status != http.StatusOK {
		t.Fatalf("fire-and-forget status = %d, body = %s", status, got)
	}
	assertJSONEqual(t, `{"ok":true}`, got)
	status, got = request(t, client, http.MethodPost, baseURL+control.RoutePlanUpdate, bytes.NewBufferString(`{"text":"  later  "}`), auth)
	if status != http.StatusOK {
		t.Fatalf("fire-and-forget update status = %d, body = %s", status, got)
	}
	assertJSONEqual(t, `{"ok":true}`, got)
	events := fake.snapshot()
	if len(events) != 2 || events[0].Kind != state.EvPlanPresent || events[0].PlanToolText != "now" ||
		events[1].Kind != state.EvPlanUpdate || events[1].PlanToolText != "later" {
		t.Fatalf("events = %#v, want two trimmed fire-and-forget plan events", events)
	}
}

func TestControlMutationRoutes(t *testing.T) {
	_, fake, baseURL := newTestServer(t, false, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken
	for _, test := range []struct {
		name, path, body string
		want             state.Event
	}{
		{"message", control.RouteMessage, `{"text":"  hello office  "}`, state.Event{Kind: state.EvControlSend, ControlText: "hello office"}},
		{"stop", control.RouteStop, "", state.Event{Kind: state.EvControlStop}},
		{"session new", control.RouteSessionNew, "", state.Event{Kind: state.EvControlNew}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body io.Reader
			if test.body != "" {
				body = bytes.NewBufferString(test.body)
			}
			status, got := request(t, client, http.MethodPost, baseURL+test.path, body, auth)
			if status != http.StatusOK {
				t.Fatalf("status = %d, body = %s", status, got)
			}
			assertJSONEqual(t, `{"ok":true}`, got)
		})
	}
	events := fake.snapshot()
	if len(events) != 3 {
		t.Fatalf("event count = %d, want 3", len(events))
	}
	for index, want := range []state.Event{
		{Kind: state.EvControlSend, ControlText: "hello office"},
		{Kind: state.EvControlStop},
		{Kind: state.EvControlNew},
	} {
		if got := events[index]; got.Kind != want.Kind || got.ControlText != want.ControlText {
			t.Fatalf("event[%d] = %#v, want %#v", index, got, want)
		}
	}
}

func TestBusyRouteHappyPath(t *testing.T) {
	_, fake, baseURL := newTestServer(t, true, time.Second)
	status, got := request(t, &http.Client{Timeout: time.Second}, http.MethodGet, baseURL+control.RouteBusy, nil, "Bearer "+testToken)
	if status != http.StatusOK {
		t.Fatalf("status = %d, body = %s", status, got)
	}
	assertJSONEqual(t, `{"busy":true,"pendingBoss":true,"thinking":true,"delegating":false,"questionParked":false}`, got)
	events := fake.snapshot()
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	assertQuery(t, events[0], control.QueryBusy, 0)
}

func TestMessageValidation(t *testing.T) {
	_, fake, baseURL := newTestServer(t, false, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken
	for _, test := range []struct {
		name, body, want string
	}{
		{"empty", `{"text":" \n\t "}`, `{"error":"empty message text"}`},
		{"invalid JSON", "not json", `{"error":"invalid JSON"}`},
		{"trailing JSON", `{"text":"hello"}{}`, `{"error":"invalid JSON"}`},
		{"oversized", `{"text":"` + string(bytes.Repeat([]byte("x"), maxPlanBodyBytes)) + `"}`, `{"error":"invalid JSON"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, got := request(t, client, http.MethodPost, baseURL+control.RouteMessage, bytes.NewBufferString(test.body), auth)
			if status != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", status, got)
			}
			assertJSONEqual(t, test.want, got)
		})
	}
	if events := fake.snapshot(); len(events) != 0 {
		t.Fatalf("invalid messages emitted events = %#v", events)
	}
}

func TestNewControlRoutesAuthAndMethodErrors(t *testing.T) {
	_, _, baseURL := newTestServer(t, true, time.Second)
	client := &http.Client{Timeout: time.Second}
	for _, test := range []struct {
		name, method, path, auth string
		want                     int
	}{
		{"message auth", http.MethodPost, control.RouteMessage, "", http.StatusUnauthorized},
		{"stop auth", http.MethodPost, control.RouteStop, "", http.StatusUnauthorized},
		{"session new auth", http.MethodPost, control.RouteSessionNew, "", http.StatusUnauthorized},
		{"busy auth", http.MethodGet, control.RouteBusy, "", http.StatusUnauthorized},
		{"message method", http.MethodGet, control.RouteMessage, "Bearer " + testToken, http.StatusMethodNotAllowed},
		{"stop method", http.MethodGet, control.RouteStop, "Bearer " + testToken, http.StatusMethodNotAllowed},
		{"session new method", http.MethodGet, control.RouteSessionNew, "Bearer " + testToken, http.StatusMethodNotAllowed},
		{"busy method", http.MethodPost, control.RouteBusy, "Bearer " + testToken, http.StatusMethodNotAllowed},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, got := request(t, client, test.method, baseURL+test.path, nil, test.auth)
			if status != test.want {
				t.Fatalf("status = %d, want %d; body = %s", status, test.want, got)
			}
		})
	}
}

func TestBusyReadAdmissionRejectsAtCapacity(t *testing.T) {
	release := make(chan struct{})
	_, fake, baseURL := newTestServerWithOptions(t, true, time.Second, 1, release)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken
	response := make(chan int, 1)
	go func() {
		status, _ := request(t, client, http.MethodGet, baseURL+control.RouteStatus, nil, auth)
		response <- status
	}()
	waitForEventCount(t, fake, 1)
	status, got := request(t, client, http.MethodGet, baseURL+control.RouteBusy, nil, auth)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("saturated status = %d, want 503; body = %s", status, got)
	}
	assertJSONEqual(t, `{"error":"office busy"}`, got)
	if events := fake.snapshot(); len(events) != 1 {
		t.Fatalf("saturated busy request created %d events, want 1", len(events))
	}
	close(release)
	if status := <-response; status != http.StatusOK {
		t.Fatalf("drained request status = %d, want 200", status)
	}
}

func TestTranscriptLimitValidation(t *testing.T) {
	_, fake, baseURL := newTestServer(t, true, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken
	for _, path := range []string{control.RouteTranscript + "?limit=-1"} {
		status, _ := request(t, client, http.MethodGet, baseURL+path, nil, auth)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", path, status)
		}
	}
	for _, path := range []string{control.RouteTranscript, control.RouteTranscript + "?limit=not-a-number", control.RouteTranscript + "?limit=501"} {
		status, _ := request(t, client, http.MethodGet, baseURL+path, nil, auth)
		if status != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", path, status)
		}
	}
	events := fake.snapshot()
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3", len(events))
	}
	assertQuery(t, events[0], control.QueryTranscript, 0)
	assertQuery(t, events[1], control.QueryTranscript, 0)
	if events[2].ControlQuery != control.QueryTranscript+"?page=1" || events[2].ControlLimit != 501 {
		t.Fatalf("clamped limit event = %#v", events[2])
	}
}

func TestTranscriptBeforeValidationAndForwarding(t *testing.T) {
	_, fake, baseURL := newTestServer(t, true, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken

	status, got := request(t, client, http.MethodGet, baseURL+control.RouteTranscript+"?limit=12&before=m-12", nil, auth)
	if status != http.StatusOK {
		t.Fatalf("valid cursor status = %d, body = %s", status, got)
	}
	assertJSONEqual(t, `{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":42}],"truncated":false,"hasMore":true}`, got)
	events := fake.snapshot()
	if len(events) != 1 || events[0].ControlQuery != "transcript?page=1&before=m-12" || events[0].ControlLimit != 12 {
		t.Fatalf("valid cursor event = %#v", events)
	}

	for _, path := range []string{
		control.RouteTranscript + "?before=%",
		control.RouteTranscript + "?before=unknown",
	} {
		status, got = request(t, client, http.MethodGet, baseURL+path, nil, auth)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400; body = %s", path, status, got)
		}
		if !strings.Contains(got, `"error":`) {
			t.Fatalf("%s: non-JSON error = %s", path, got)
		}
	}
}

func TestTimeoutCancelsPendingRequest(t *testing.T) {
	_, fake, baseURL := newTestServer(t, false, 20*time.Millisecond)
	status, got := request(t, &http.Client{Timeout: time.Second}, http.MethodGet, baseURL+control.RouteStatus, nil, "Bearer "+testToken)
	if status != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, body = %s", status, got)
	}
	assertJSONEqual(t, `{"error":"office busy"}`, got)
	events := fake.snapshot()
	if len(events) != 1 || fake.registry.Fulfill(events[0].ControlReqID, []byte(`{}`)) {
		t.Fatalf("timed out request was not cancelled: %#v", events)
	}
}

func TestReadAdmissionRejectsAtCapacityAndRestoresAllSlots(t *testing.T) {
	release := make(chan struct{})
	_, fake, baseURL := newTestServerWithOptions(t, true, time.Second, 2, release)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken

	responses := make(chan int, 2)
	for range 2 {
		go func() {
			status, _ := request(t, client, http.MethodGet, baseURL+control.RouteStatus, nil, auth)
			responses <- status
		}()
	}
	waitForEventCount(t, fake, 2)

	status, got := request(t, client, http.MethodGet, baseURL+control.RoutePlan, nil, auth)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("saturated status = %d, want 503; body = %s", status, got)
	}
	assertJSONEqual(t, `{"error":"office busy"}`, got)
	if gotEvents := len(fake.snapshot()); gotEvents != 2 {
		t.Fatalf("saturated request created %d events, want 2", gotEvents)
	}

	close(release)
	for range 2 {
		if status := <-responses; status != http.StatusOK {
			t.Fatalf("drained request status = %d, want 200", status)
		}
	}

	responses = make(chan int, 2)
	for range 2 {
		go func() {
			status, _ := request(t, client, http.MethodGet, baseURL+control.RouteStatus, nil, auth)
			responses <- status
		}()
	}
	for range 2 {
		if status := <-responses; status != http.StatusOK {
			t.Fatalf("restored-capacity request status = %d, want 200", status)
		}
	}
}

func TestTimedOutReadReleasesAdmissionSlot(t *testing.T) {
	_, fake, baseURL := newTestServerWithOptions(t, false, 20*time.Millisecond, 1, nil)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken
	for range 2 {
		status, got := request(t, client, http.MethodGet, baseURL+control.RouteStatus, nil, auth)
		if status != http.StatusGatewayTimeout {
			t.Fatalf("timed-out status = %d, want 504; body = %s", status, got)
		}
	}
	if events := fake.snapshot(); len(events) != 2 {
		t.Fatalf("timed-out reads emitted %d events, want 2 (second slot was not restored)", len(events))
	}
}

func TestReadAdmissionConcurrentHammerDoesNotDeadlockOrLeak(t *testing.T) {
	_, _, baseURL := newTestServerWithOptions(t, true, time.Second, 4, nil)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken
	const callers = 64
	statuses := make(chan int, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, _ := request(t, client, http.MethodGet, baseURL+control.RouteStatus, nil, auth)
			statuses <- status
		}()
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != http.StatusOK && status != http.StatusServiceUnavailable {
			t.Fatalf("hammer status = %d, want 200 or 503", status)
		}
	}

	for range 4 {
		status, got := request(t, client, http.MethodGet, baseURL+control.RouteStatus, nil, auth)
		if status != http.StatusOK {
			t.Fatalf("post-hammer capacity status = %d, want 200; body = %s", status, got)
		}
	}
}

func TestLoopbackAndCloseIdempotent(t *testing.T) {
	server, _, _ := newTestServer(t, true, time.Second)
	address, ok := server.Addr().(*net.TCPAddr)
	if !ok || !address.IP.IsLoopback() {
		t.Fatalf("listener address = %#v, want loopback TCP address", server.Addr())
	}
	if err := server.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestNewPanicsForNilSink(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("New did not panic for nil Sink")
		}
	}()
	New(Options{})
}

func TestNewPanicsForEmptyToken(t *testing.T) {
	for _, token := range []string{"", " \t\n "} {
		t.Run(strconv.Quote(token), func(t *testing.T) {
			defer func() {
				if got := recover(); got != "controlsrv: empty Token" {
					t.Fatalf("panic = %v, want controlsrv: empty Token", got)
				}
			}()
			New(Options{Sink: func(state.Event) {}, Token: token})
		})
	}
}

func TestNewAppliesAdmissionDefaultsAndHeaderLimit(t *testing.T) {
	server := New(Options{Sink: func(state.Event) {}, Token: testToken})
	if got := cap(server.readSlots); got != defaultMaxInFlightReads {
		t.Fatalf("default read slots = %d, want %d", got, defaultMaxInFlightReads)
	}
	if server.maxConnections != defaultMaxConnections {
		t.Fatalf("default max connections = %d, want %d", server.maxConnections, defaultMaxConnections)
	}
	if err := server.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer server.Close()
	if server.http.MaxHeaderBytes != maxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", server.http.MaxHeaderBytes, maxHeaderBytes)
	}
}

func assertQuery(t *testing.T, event state.Event, query string, limit int) {
	t.Helper()
	if event.Kind != state.EvControlQuery || event.ControlQuery != query || event.ControlLimit != limit || event.ControlReqID == "" {
		t.Fatalf("query event = %#v", event)
	}
}

func assertJSONEqual(t *testing.T, want, got string) {
	t.Helper()
	var expected, actual any
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatalf("bad expected JSON %q: %v", want, err)
	}
	if err := json.Unmarshal([]byte(got), &actual); err != nil {
		t.Fatalf("bad actual JSON %q: %v", got, err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("JSON = %s, want %s", got, want)
	}
}

func waitForEventCount(t *testing.T, fake *fakeSink, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(fake.snapshot()) >= want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("event count did not reach %d; got %d", want, len(fake.snapshot()))
}
