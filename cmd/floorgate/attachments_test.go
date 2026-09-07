package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringfloor/internal/control"
)

const validPNGData = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScL9ZwAAAABJRU5ErkJggg=="

func TestGatewayMessageAttachments(t *testing.T) {
	var forwarded []string
	office := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		forwarded = append(forwarded, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer office.Close()

	gateway := newTestGateway(t, nil)
	gateway.discovery = testDiscovery(t, office.URL)
	server := httptest.NewServer(gateway)
	defer server.Close()

	attachment := `{"name":"pixel.png","mimeType":"image/png","data":"` + validPNGData + `"}`
	cases := []struct {
		name        string
		body        string
		wantStatus  int
		wantForward string
	}{
		{
			name:        "text only remains identical upstream",
			body:        `{"text":"hi"}`,
			wantStatus:  http.StatusOK,
			wantForward: `{"text":"hi"}`,
		},
		{
			name:        "one png attachment remains intact upstream",
			body:        `{"text":"hi","attachments":[` + attachment + `]}`,
			wantStatus:  http.StatusOK,
			wantForward: `{"text":"hi","attachments":[` + attachment + `]}`,
		},
		{
			name:       "four attachments accepted",
			body:       `{"text":"hi","attachments":[` + strings.Join([]string{attachment, attachment, attachment, attachment}, ",") + `]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "five attachments rejected",
			body:       `{"text":"hi","attachments":[` + strings.Join([]string{attachment, attachment, attachment, attachment, attachment}, ",") + `]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad base64 rejected",
			body:       `{"text":"hi","attachments":[{"name":"bad.png","mimeType":"image/png","data":"%%%"}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "disallowed mime rejected",
			body:       `{"text":"hi","attachments":[{"name":"text.txt","mimeType":"text/plain","data":"aGk="}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty text with attachment is proxied",
			body:       `{"text":"  ","attachments":[` + attachment + `]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty text without attachment rejected",
			body:       `{"text":"  "}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unknown field rejected",
			body:       `{"text":"hi","unknown":true}`,
			wantStatus: http.StatusBadRequest,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			before := len(forwarded)
			response := authorizedRequest(t, server.URL+apiPrefix+"/projects/p/message", http.MethodPost, strings.NewReader(test.body), "gate-token")
			if body := responseBody(t, response); response.StatusCode != test.wantStatus || response.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("status/content-type/body = %d %q %q", response.StatusCode, response.Header.Get("Content-Type"), body)
			}
			if test.wantForward != "" {
				if len(forwarded) != before+1 || forwarded[before] != test.wantForward {
					t.Fatalf("forwarded = %#v, want %q", forwarded, test.wantForward)
				}
			} else if len(forwarded) != before && test.wantStatus >= http.StatusBadRequest {
				t.Fatalf("rejected request reached office: %#v", forwarded[before:])
			}
		})
	}
}

func TestGatewayMessageRejectsOversizeAttachment(t *testing.T) {
	gateway := newTestGateway(t, nil)
	called := false
	gateway.discovery = func(string) (control.Discovery, error) {
		called = true
		return control.Discovery{}, nil
	}
	server := httptest.NewServer(gateway)
	defer server.Close()

	encoded := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{'x'}, maxAttachmentBytes+1))
	body := `{"text":"hi","attachments":[{"name":"large.png","mimeType":"image/png","data":"` + encoded + `"}]}`
	response := authorizedRequest(t, server.URL+apiPrefix+"/projects/p/message", http.MethodPost, strings.NewReader(body), "gate-token")
	if got := responseBody(t, response); response.StatusCode != http.StatusRequestEntityTooLarge || got != "{\"error\":\"attachment too large\"}\n" {
		t.Fatalf("status/body = %d %q", response.StatusCode, got)
	}
	if called {
		t.Fatal("oversize attachment reached project discovery")
	}
}

func TestGatewayMessageAuthMethodAndOfficeError(t *testing.T) {
	office := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"office rejected image"}`))
	}))
	defer office.Close()
	gateway := newTestGateway(t, nil)
	gateway.discovery = testDiscovery(t, office.URL)
	server := httptest.NewServer(gateway)
	defer server.Close()

	missing, err := http.NewRequest(http.MethodPost, server.URL+apiPrefix+"/projects/p/message", strings.NewReader(`{"text":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	missing.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(missing)
	if err != nil {
		t.Fatal(err)
	}
	if got := responseBody(t, response); response.StatusCode != http.StatusUnauthorized || got != "{\"error\":\"unauthorized\"}\n" {
		t.Fatalf("missing bearer status/body = %d %q", response.StatusCode, got)
	}

	response = authorizedRequest(t, server.URL+apiPrefix+"/projects/p/message", http.MethodGet, nil, "gate-token")
	if got := responseBody(t, response); response.StatusCode != http.StatusMethodNotAllowed || response.Header.Get("Allow") != http.MethodPost || got != "{\"error\":\"method not allowed\"}\n" {
		t.Fatalf("wrong method status/allow/body = %d %q %q", response.StatusCode, response.Header.Get("Allow"), got)
	}

	response = authorizedRequest(t, server.URL+apiPrefix+"/projects/p/message", http.MethodPost, strings.NewReader(`{"text":"hi"}`), "gate-token")
	if got := responseBody(t, response); response.StatusCode != http.StatusUnprocessableEntity || got != `{"error":"office rejected image"}` {
		t.Fatalf("office error status/body = %d %q", response.StatusCode, got)
	}
}
