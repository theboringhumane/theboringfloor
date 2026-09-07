package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExecSuccess(t *testing.T) {
	response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{"command":"echo hello"}`, "Bearer exec-token")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body execResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Stdout != "hello\n" || body.Stderr != "" || body.ExitCode != 0 || body.Truncated {
		t.Fatalf("response = %#v", body)
	}
}

func TestExecNonZeroAndStderr(t *testing.T) {
	response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{"command":"printf problem >&2; exit 3"}`, "Bearer exec-token")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body execResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.ExitCode != 3 || body.Stderr != "problem" {
		t.Fatalf("response = %#v", body)
	}
}

func TestExecTimeoutKillsProcessGroup(t *testing.T) {
	started := time.Now()
	response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{"command":"sleep 5","timeoutMs":200}`, "Bearer exec-token")
	if elapsed := time.Since(started); elapsed >= 2*time.Second {
		t.Fatalf("timeout request took %s", elapsed)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body execResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.ExitCode != -1 || !strings.Contains(body.Stderr, "command timed out") {
		t.Fatalf("response = %#v", body)
	}
}

func TestExecTruncatesOutput(t *testing.T) {
	response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{"command":"yes x | head -c 300000"}`, "Bearer exec-token")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var body execResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.Truncated || len(body.Stdout) != maxExecOutput {
		t.Fatalf("truncated/stdout length = %t/%d, want true/%d", body.Truncated, len(body.Stdout), maxExecOutput)
	}
}

func TestExecRejectsInvalidRequests(t *testing.T) {
	t.Run("bad cwd", func(t *testing.T) {
		response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{"command":"pwd","cwd":"/definitely/not/a/directory"}`, "Bearer exec-token")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})
	t.Run("empty command", func(t *testing.T) {
		response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{"command":" \t "}`, "Bearer exec-token")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})
	t.Run("malformed JSON", func(t *testing.T) {
		response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{`, "Bearer exec-token")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})
	t.Run("body too large", func(t *testing.T) {
		response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{"command":"echo hello","padding":"`+strings.Repeat("x", maxExecBodyBytes)+`"}`, "Bearer exec-token")
		if response.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})
}

func TestExecDisabledAuthAndMethod(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		response := execRequestForTest(t, newGateway("exec-token"), http.MethodPost, `{"command":"echo hello"}`, "Bearer exec-token")
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "FLOORGATE_EXEC=1") {
			t.Fatalf("status/body = %d/%s", response.Code, response.Body.String())
		}
	})
	for _, token := range []string{"", "Bearer wrong-token"} {
		t.Run("unauthorized "+token, func(t *testing.T) {
			response := execRequestForTest(t, enabledExecGateway(), http.MethodPost, `{"command":"echo hello"}`, token)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
	t.Run("wrong method", func(t *testing.T) {
		response := execRequestForTest(t, enabledExecGateway(), http.MethodGet, "", "Bearer exec-token")
		if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodPost {
			t.Fatalf("status/allow = %d/%q", response.Code, response.Header().Get("Allow"))
		}
	})
}

func enabledExecGateway() *gateway {
	gateway := newGateway("exec-token")
	gateway.exec = true
	return gateway
}

func execRequestForTest(t *testing.T, gateway *gateway, method, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, apiPrefix+"/exec", bytes.NewBufferString(body))
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	gateway.ServeHTTP(response, request)
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	return response
}
