package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/sessionsearch"
)

type officeClient struct { dir string; http *http.Client }

func newOffice(dir string) *officeClient {
	return &officeClient{dir:dir, http:&http.Client{Timeout:3*time.Second, CheckRedirect:func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport:&http.Transport{Proxy:nil}}}
}

func (o *officeClient) discovery() (control.Discovery, error) {
	d, ok := control.ReadDiscovery(o.dir)
	if !ok { return control.Discovery{}, fmt.Errorf("the office is not running in %s", o.dir) }
	if d.Stale() {
		control.PruneStale(o.dir)
		return control.Discovery{}, fmt.Errorf("the office is not running in %s", o.dir)
	}
	var health control.HealthResponse
	if err := o.request(d, http.MethodGet, control.RouteHealth, nil, &health); err != nil || !health.OK || health.Dir != o.dir {
		return control.Discovery{}, fmt.Errorf("the office is not running in %s", o.dir)
	}
	return d, nil
}

func (o *officeClient) request(d control.Discovery, method, path string, body interface{}, result interface{}) error {
	u := "http://127.0.0.1:"+strconv.Itoa(d.Port)+path
	var reader io.Reader
	if body != nil { data, err := json.Marshal(body); if err != nil { return err }; reader = bytes.NewReader(data) }
	req, err := http.NewRequest(method, u, reader); if err != nil { return err }
	req.Header.Set("Authorization", "Bearer "+d.Token)
	if body != nil { req.Header.Set("Content-Type", "application/json") }
	resp, err := o.http.Do(req); if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 { return fmt.Errorf("control API returned %s", resp.Status) }
	return json.NewDecoder(resp.Body).Decode(result)
}

func (o *officeClient) livePlan() (control.PlanResponse, error) { d,e:=o.discovery(); if e!=nil{return control.PlanResponse{},e}; var r control.PlanResponse; return r,o.request(d,http.MethodGet,control.RoutePlan,nil,&r) }
func (o *officeClient) liveTranscript(limit int) (control.TranscriptResponse, error) { d,e:=o.discovery(); if e!=nil{return control.TranscriptResponse{},e}; var r control.TranscriptResponse; return r,o.request(d,http.MethodGet,control.RouteTranscript+"?limit="+strconv.Itoa(limit),nil,&r) }
func (o *officeClient) liveStatus() (control.StatusResponse, error) { d,e:=o.discovery(); if e!=nil{return control.StatusResponse{},e}; var r control.StatusResponse; return r,o.request(d,http.MethodGet,control.RouteStatus,nil,&r) }

func (o *officeClient) call(name string, raw json.RawMessage) (string, bool) {
	switch name {
	case "plan_present", "plan_update": return o.writePlan(name, raw)
	case "plan_get_approved": return o.approvedPlan()
	case "transcript_read": return o.transcript(raw)
	case "transcript_search": return o.search(raw)
	case "office_status": return o.status()
	default: return "unknown tool: "+name, true
	}
}

func (o *officeClient) writePlan(name string, raw json.RawMessage) (string, bool) {
	var a struct{ Text string `json:"text"` }; if err:=decodeArgs(raw,&a); err!=nil{return err.Error(),true}; a.Text=strings.TrimSpace(a.Text); if a.Text=="" {return "plan text must not be empty",true}
	d, err := o.discovery(); if err != nil { return err.Error()+" — start theboringfloor there to "+strings.ReplaceAll(name,"_"," "), true }
	path:=control.RoutePlanPresent; if name=="plan_update" {path=control.RoutePlanUpdate}; var r control.OKResponse
	if err:=o.request(d,http.MethodPost,path,control.PlanWriteRequest{Text:a.Text},&r); err!=nil || !r.OK { if err==nil {err=fmt.Errorf("control API did not confirm the update")}; return "could not "+strings.ReplaceAll(name,"_"," ")+": "+err.Error(),true }
	return "Plan sent to the live office in "+o.dir+" for member review.",false
}

func (o *officeClient) approvedPlan() (string, bool) { if r,err:=o.livePlan(); err==nil {if !r.HasApproved || strings.TrimSpace(r.Approved)=="" {return "No approved plan in the live office.",false}; return r.Approved,false}; p,ok:=sessionsearch.ApprovedPlan(o.dir); if !ok {return "On-disk snapshot: no approved plan (no session snapshot found).",false}; if strings.TrimSpace(p)=="" {return "On-disk snapshot: no approved plan.",false}; return "On-disk snapshot (office is not live):\n\n"+p,false }

func bounded(value, def, max int) int { if value<=0{return def}; if value>max{return max}; return value }
func (o *officeClient) transcript(raw json.RawMessage) (string, bool) { var a struct{Limit int `json:"limit"`}; if err:=decodeArgs(raw,&a);err!=nil{return err.Error(),true}; a.Limit=bounded(a.Limit,50,500); if r,err:=o.liveTranscript(a.Limit);err==nil{return renderLiveTranscript(r),false}; messages,ok:=sessionsearch.Transcript(o.dir,a.Limit);if !ok{return "On-disk snapshot (office is not live): no session transcript found.",false}; return "On-disk snapshot (office is not live):\n"+renderMessages(messages),false }
func (o *officeClient) search(raw json.RawMessage) (string, bool) { var a struct{Query string `json:"query"`; Limit int `json:"limit"`};if err:=decodeArgs(raw,&a);err!=nil{return err.Error(),true};a.Query=strings.TrimSpace(a.Query);if a.Query=="" {return "search query must not be empty",true};a.Limit=bounded(a.Limit,20,200);hits,ok:=sessionsearch.Search(o.dir,a.Query,a.Limit);if !ok{return "On-disk snapshot: no session transcript found.",false};if len(hits)==0{return "On-disk snapshot: no transcript matches for "+strconv.Quote(a.Query)+".",false};var b strings.Builder;b.WriteString("On-disk snapshot search results for "+strconv.Quote(a.Query)+":\n");for i,h:=range hits{fmt.Fprintf(&b,"%d. [%s] %s: %s\n",i+1,formatAt(h.Message.At),h.Message.From,h.Snippet)};return b.String(),false }
func (o *officeClient) status() (string, bool) { if r,err:=o.liveStatus();err==nil{return fmt.Sprintf("Live office\ndir: %s\nbackend: %s\nprimary ID: %s\nplan draft length: %d\napproved plan length: %d\nchat count: %d",o.dir,r.Backend,r.PrimaryID,r.PlanDraftLen,r.PlanApprovedLen,r.ChatCount),false};m,ok:=sessionsearch.Info(o.dir);if !ok{return "Office is not live.\ndir: "+o.dir+"\non-disk snapshot: unavailable",false};return fmt.Sprintf("Office is not live; on-disk snapshot\ndir: %s\nbackend: %s\nprimary ID: %s\nchat count: %d",o.dir,m.Backend,m.PrimaryID,m.ChatCount),false }

func formatAt(at int64) string { if at<=0{return "unknown time"}; return time.UnixMilli(at).Local().Format(time.RFC3339) }
func renderLiveTranscript(r control.TranscriptResponse) string { type msg=control.TranscriptMessage; ms:=make([]msg,len(r.Messages));copy(ms,r.Messages);var b strings.Builder;b.WriteString("Live office transcript:\n");for _,m:=range ms{fmt.Fprintf(&b,"[%s] %s: %s\n",formatAt(m.At),m.From,m.Text)};if r.Truncated{b.WriteString("[truncated]\n")};return b.String() }
func renderMessages(messages []sessionsearch.Message) string {var b strings.Builder;for _,m:=range messages{fmt.Fprintf(&b,"[%s] %s: %s\n",formatAt(m.At),m.From,m.Text)};return b.String()}
