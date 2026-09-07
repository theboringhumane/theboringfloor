package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/projects"
	"github.com/theboringhumane/theboringfloor/internal/version"
)

const (
	apiPrefix       = "/api/v1"
	maxMessageBytes = 64 << 10
)

type gateway struct {
	token     string
	client    *http.Client
	list      func(context.Context, time.Duration) ([]projects.Project, error)
	get       func(context.Context, string, time.Duration) (projects.Project, error)
	discovery func(string) (control.Discovery, error)
}

func newGateway(token string) *gateway {
	return &gateway{
		token: token,
		client: &http.Client{
			Timeout:       10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
			Transport:     &http.Transport{Proxy: nil},
		},
		list:      projects.List,
		get:       projects.Get,
		discovery: projects.Discovery,
	}
}

func (g *gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, apiPrefix+"/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !authorized(r, g.token) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if r.Method == http.MethodGet && r.URL.Path == apiPrefix+"/health" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "version": version.String()})
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == apiPrefix+"/projects" {
		g.projects(w, r)
		return
	}

	id, suffix, ok := projectPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	switch {
	case r.Method == http.MethodGet && suffix == "":
		g.project(w, r, id)
	case r.Method == http.MethodGet && suffix == "/status":
		g.proxy(w, r, id, http.MethodGet, control.RouteStatus, nil)
	case r.Method == http.MethodGet && suffix == "/busy":
		g.proxy(w, r, id, http.MethodGet, control.RouteBusy, nil)
	case r.Method == http.MethodGet && suffix == "/transcript":
		g.transcript(w, r, id)
	case r.Method == http.MethodPost && suffix == "/message":
		g.message(w, r, id)
	case r.Method == http.MethodPost && suffix == "/stop":
		g.proxy(w, r, id, http.MethodPost, control.RouteStop, nil)
	case r.Method == http.MethodPost && suffix == "/new":
		g.proxy(w, r, id, http.MethodPost, control.RouteSessionNew, nil)
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (g *gateway) projects(w http.ResponseWriter, r *http.Request) {
	items, err := g.list(r.Context(), 10*time.Second)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not list projects")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": items})
}

func (g *gateway) project(w http.ResponseWriter, r *http.Request, id string) {
	item, err := g.get(r.Context(), id, 10*time.Second)
	if err != nil {
		g.projectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (g *gateway) transcript(w http.ResponseWriter, r *http.Request, id string) {
	limit := r.URL.Query().Get("limit")
	if limit != "" {
		n, err := strconv.Atoi(limit)
		if err != nil || n < 0 || n > 500 {
			writeError(w, http.StatusBadRequest, "invalid transcript limit")
			return
		}
	}
	path := control.RouteTranscript
	if limit != "" {
		path += "?limit=" + strconv.Itoa(mustAtoi(limit))
	}
	g.proxy(w, r, id, http.MethodGet, path, nil)
}

func (g *gateway) message(w http.ResponseWriter, r *http.Request, id string) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusBadRequest, "content type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMessageBytes)
	defer r.Body.Close()
	var body control.MessageRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid message body")
		return
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "invalid message body")
		return
	}
	body.Text = strings.TrimSpace(body.Text)
	if body.Text == "" {
		writeError(w, http.StatusBadRequest, "empty message text")
		return
	}
	payload, err := json.Marshal(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid message body")
		return
	}
	g.proxy(w, r, id, http.MethodPost, control.RouteMessage, payload)
}

func (g *gateway) proxy(w http.ResponseWriter, r *http.Request, id, method, path string, payload []byte) {
	discovery, err := g.discovery(id)
	if err != nil {
		// projects.Discovery deliberately treats an absent project and an absent
		// live office alike. Consult Get only on that ambiguity so the public
		// gateway can retain its distinct 404 and 409 contract.
		if errors.Is(err, projects.ErrNotLive) {
			if _, getErr := g.get(r.Context(), id, 10*time.Second); errors.Is(getErr, projects.ErrNotFound) {
				g.projectError(w, getErr)
				return
			}
		}
		g.projectError(w, err)
		return
	}
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(r.Context(), method, "http://127.0.0.1:"+strconv.Itoa(discovery.Port)+path, body)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not reach office")
		return
	}
	request.Header.Set("Authorization", "Bearer "+discovery.Token)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := g.client.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
			writeError(w, http.StatusGatewayTimeout, "office timed out")
			return
		}
		writeError(w, http.StatusBadGateway, "could not reach office")
		return
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		writeError(w, http.StatusBadGateway, "office returned an error")
		return
	}
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func (g *gateway) projectError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, projects.ErrNotFound):
		writeError(w, http.StatusNotFound, "project not found")
	case errors.Is(err, projects.ErrNotLive):
		writeError(w, http.StatusConflict, "office not running")
	default:
		writeError(w, http.StatusBadGateway, "could not read project")
	}
}

func authorized(r *http.Request, token string) bool {
	const prefix = "Bearer "
	if strings.TrimSpace(token) == "" {
		return false
	}
	value := r.Header.Get("Authorization")
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	provided := value[len(prefix):]
	if strings.TrimSpace(provided) == "" {
		return false
	}
	if len(provided) != len(token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1
}

func projectPath(path string) (string, string, bool) {
	const prefix = apiPrefix + "/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		return "", "", false
	}
	parts := strings.SplitN(rest, "/", 2)
	if parts[0] == "" {
		return "", "", false
	}
	if len(parts) == 1 {
		return parts[0], "", true
	}
	return parts[0], "/" + parts[1], true
}

func rejectTrailingJSON(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("extra JSON value")
		}
		return err
	}
	return nil
}

func mustAtoi(value string) int {
	n, _ := strconv.Atoi(value)
	return n
}

func isTimeout(err error) bool {
	var networkErr net.Error
	return errors.As(err, &networkErr) && networkErr.Timeout()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type gatewayConfig struct {
	Token     string `json:"token"`
	CreatedAt int64  `json:"createdAt"`
}

func gatewayHome() string {
	if home := config.HomeOverride(); home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func gatewayTokenPath(home string) string {
	return filepath.Join(home, ".theboringfloor", "configs", "gateway.json")
}

func loadGatewayToken(home string) (string, error) {
	path := gatewayTokenPath(home)
	contents, err := os.ReadFile(path)
	if err == nil {
		var stored gatewayConfig
		if err := json.Unmarshal(contents, &stored); err != nil || strings.TrimSpace(stored.Token) == "" || stored.CreatedAt <= 0 {
			return "", fmt.Errorf("read gateway token: invalid %s", path)
		}
		return stored.Token, nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read gateway token: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create gateway token directory: %w", err)
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("mint gateway token: %w", err)
	}
	stored := gatewayConfig{Token: hex.EncodeToString(random), CreatedAt: time.Now().UnixMilli()}
	contents, err = json.Marshal(stored)
	if err != nil {
		return "", fmt.Errorf("encode gateway token: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".gateway-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create gateway token file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return "", fmt.Errorf("secure gateway token file: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return "", fmt.Errorf("write gateway token: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("close gateway token: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return "", fmt.Errorf("install gateway token: %w", err)
	}
	return stored.Token, nil
}

func resolvedBind(flagValue, envValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if envValue != "" {
		return envValue
	}
	return "127.0.0.1:8787"
}

func bindWarning(address net.Addr) string {
	tcp, ok := address.(*net.TCPAddr)
	if ok && tcp.IP.IsLoopback() {
		return ""
	}
	return fmt.Sprintf("floorgate warning: listening on %s; the bearer token is the only protection\n", address.String())
}
