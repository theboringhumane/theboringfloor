package controlsrv

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const testToken = "test-control-token"

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
	switch query {
	case control.QueryPlan:
		return []byte(`{"draft":"draft plan","approved":"approved plan","hasApproved":true}`)
	case control.QueryTranscript:
		return []byte(`{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":42}],"truncated":false}`)
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
		{"transcript", http.MethodGet, control.RouteTranscript + "?limit=12", "", `{"messages":[{"id":"m1","from":"boss","kind":"chat","text":"hello","at":42}],"truncated":false}`},
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
	assertQuery(t, events[4], control.QueryTranscript, 12)
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

func TestTranscriptLimitValidation(t *testing.T) {
	_, fake, baseURL := newTestServer(t, true, time.Second)
	client := &http.Client{Timeout: time.Second}
	auth := "Bearer " + testToken
	for _, path := range []string{control.RouteTranscript + "?limit=-1", control.RouteTranscript + "?limit=501"} {
		status, _ := request(t, client, http.MethodGet, baseURL+path, nil, auth)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400", path, status)
		}
	}
	for _, path := range []string{control.RouteTranscript, control.RouteTranscript + "?limit=not-a-number"} {
		status, _ := request(t, client, http.MethodGet, baseURL+path, nil, auth)
		if status != http.StatusOK {
			t.Fatalf("%s: status = %d, want 200", path, status)
		}
	}
	events := fake.snapshot()
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	for _, event := range events {
		assertQuery(t, event, control.QueryTranscript, 0)
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

func TestNewAppliesAdmissionDefaultsAndHeaderLimit(t *testing.T) {
	server := New(Options{Sink: func(state.Event) {}})
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
