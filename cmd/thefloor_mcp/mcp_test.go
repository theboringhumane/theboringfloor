package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/control"
)

func testOffice(t *testing.T, dir string) *officeClient {
	t.Helper()
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	return newOffice(dir)
}

func TestDiscoveryMissingClassifiesOfficeNotRunning(t *testing.T) {
	o := testOffice(t, t.TempDir())
	_, err := o.discovery()
	if err == nil || !strings.Contains(err.Error(), "not running") { t.Fatalf("discovery() error = %v, want office-not-running", err) }
}

func TestLivePathUsesBearerAndRoute(t *testing.T) {
	dir := t.TempDir()
	var gotAuth, gotPath string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		if r.URL.Path == control.RouteHealth {
			_ = json.NewEncoder(w).Encode(control.HealthResponse{OK:true, Dir:dir})
			return
		}
		_ = json.NewEncoder(w).Encode(control.PlanResponse{Approved:"# Approved", HasApproved:true})
	}))
	defer s.Close()
	u, _ := url.Parse(s.URL); port, _ := strconv.Atoi(u.Port())
	o := testOffice(t, dir)
	cmd := startMCPTestOfficeProcess(t)
	if err := control.WriteDiscovery(dir, control.Discovery{PID:cmd.Process.Pid, Port:port, Token:"secret"}); err != nil { t.Fatal(err) }
	text, isErr := o.call("plan_get_approved", nil)
	if isErr || text != "# Approved" { t.Fatalf("call = (%q, %v)", text, isErr) }
	if gotAuth != "Bearer secret" || gotPath != control.RoutePlan { t.Fatalf("request = auth %q path %q", gotAuth, gotPath) }
}

func TestHealthDirMismatchFallsBackWithoutFurtherRequests(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("THEBORINGOFFICE_HOME", home)
	writeSession(t, home, dir, `{"dir":%q,"approvedPlanText":"# Offline plan"}`)
	var requests int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(control.HealthResponse{OK:true, Dir:dir+"-different"})
	}))
	defer s.Close()
	u, _ := url.Parse(s.URL)
	port, _ := strconv.Atoi(u.Port())
	cmd := startMCPTestOfficeProcess(t)
	if err := control.WriteDiscovery(dir, control.Discovery{PID:cmd.Process.Pid, Port:port, Token:"secret"}); err != nil {
		t.Fatal(err)
	}
	o := newOffice(dir)
	text, bad := o.call("plan_get_approved", nil)
	if bad || !strings.Contains(text, "On-disk snapshot") || !strings.Contains(text, "# Offline plan") {
		t.Fatalf("call = (%q, %v)", text, bad)
	}
	if requests != 1 {
		t.Fatalf("authenticated requests = %d, want 1 health request only", requests)
	}
}

func TestDiskFallbackAndSearch(t *testing.T) {
	dir := t.TempDir(); home := t.TempDir(); t.Setenv("THEBORINGOFFICE_HOME", home)
	writeSession(t, home, dir, `{"dir":%q,"backend":"opencode","chat":[{"id":"1","from":"boss","text":"Need a durable plan","at":1000}],"approvedPlanText":"# Offline plan"}`)
	o := newOffice(dir)
	approved, bad := o.call("plan_get_approved", nil)
	if bad || !strings.Contains(approved, "On-disk snapshot") || !strings.Contains(approved, "# Offline plan") { t.Fatalf("approved = (%q, %v)", approved, bad) }
	search, bad := o.call("transcript_search", json.RawMessage(`{"query":"durable"}`))
	if bad || !strings.Contains(search, "On-disk snapshot") || !strings.Contains(search, "durable plan") { t.Fatalf("search = (%q, %v)", search, bad) }
}

func TestWriteRefusesWhenOfficeAbsent(t *testing.T) {
	o := testOffice(t, t.TempDir())
	text, bad := o.call("plan_present", json.RawMessage(`{"text":"# Plan"}`))
	if !bad || !strings.Contains(text, "start theboringfloor") { t.Fatalf("call = (%q, %v)", text, bad) }
}

func TestLimitsAndEmptyInput(t *testing.T) {
	if bounded(0,50,500)!=50 || bounded(999,50,500)!=500 || bounded(5,50,500)!=5 { t.Fatal("limit clamping failed") }
	o := testOffice(t, t.TempDir())
	for _, tc := range []struct{name, args string}{ {"plan_present", `{"text":"  "}`}, {"transcript_search", `{"query":" "}`} } {
		text,bad:=o.call(tc.name,json.RawMessage(tc.args)); if !bad || !strings.Contains(text,"must not be empty") {t.Fatalf("%s = (%q, %v)",tc.name,text,bad)}
	}
}

func TestStdioInitializeAndToolsListOnlyWritesProtocolJSON(t *testing.T) {
	in := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\",\"params\":{}}\n")
	var out bytes.Buffer
	if err:=serve(in,&out,testOffice(t,t.TempDir()));err!=nil{t.Fatal(err)}
	lines:=strings.Split(strings.TrimSpace(out.String()),"\n");if len(lines)!=2{t.Fatalf("stdout protocol frames = %q",out.String())}
	var listed struct{Result struct{Tools []struct{Name string `json:"name"`} `json:"tools"`} `json:"result"`}; if err:=json.Unmarshal([]byte(lines[1]),&listed);err!=nil{t.Fatal(err)}
	want:=[]string{"plan_present","plan_update","plan_get_approved","transcript_read","transcript_search","office_status"};if len(listed.Result.Tools)!=len(want){t.Fatalf("tools = %#v",listed.Result.Tools)};for i,n:=range want{if listed.Result.Tools[i].Name!=n{t.Fatalf("tool %d = %q, want %q",i,listed.Result.Tools[i].Name,n)}}
}

func writeSession(t *testing.T, home, dir, body string) {
	t.Helper(); path:=filepath.Join(home,".theboringfloor","projects",control.DirHash(dir),"session.json"); if err:=os.MkdirAll(filepath.Dir(path),0755);err!=nil{t.Fatal(err)}; if err:=os.WriteFile(path,[]byte(fmtJSON(body,dir)),0600);err!=nil{t.Fatal(err)}
}
func fmtJSON(template, dir string) string { return strings.ReplaceAll(template, "%q", strconv.Quote(dir)) }

func startMCPTestOfficeProcess(t *testing.T) *exec.Cmd {
	t.Helper()
	if os.Getenv("GO_WANT_MCP_HELPER") == "1" {
		select {}
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(t.TempDir(), "theboringfloor")
	b, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(copyPath, b, 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(copyPath, "-test.run=TestMCPHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_MCP_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	return cmd
}

func TestMCPHelperProcess(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_HOME", t.TempDir())
	if os.Getenv("GO_WANT_MCP_HELPER") == "1" {
		select {}
	}
}
