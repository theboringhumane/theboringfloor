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
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	exec      bool
	client    *http.Client
	list      func(context.Context, time.Duration) ([]projects.Project, error)
	get       func(context.Context, string, time.Duration) (projects.Project, error)
	discovery func(string) (control.Discovery, error)
	launcher  officeLauncher
	startMu   sync.Mutex
	starting  map[string]bool
}

// officeLauncher starts an office in a project directory and returns once the
// launch has either been accepted by the operating system or failed to begin.
// It calls exited after an accepted process exits, so the gateway can allow a
// later start request for a stopped project.
type officeLauncher interface {
	Launch(dir string, exited func()) error
}

type commandLauncher struct{}

var ErrOfficeBinaryNotFound = errors.New("theboringfloor binary not found")
var officeLookPath = exec.LookPath

func (commandLauncher) Launch(dir string, exited func()) error {
	command, closeFiles, err := newOfficeCommand(dir)
	if err != nil {
		return err
	}
	defer closeFiles()
	if err := command.Start(); err != nil {
		return err
	}
	go func() {
		_ = command.Wait()
		exited()
	}()
	return nil
}

func newOfficeCommand(dir string) (*exec.Cmd, func(), error) {
	binary, err := officeLookPath("theboringfloor")
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrOfficeBinaryNotFound, err)
	}
	return newOfficeCommandForBinary(dir, binary)
}

func newOfficeCommandForBinary(dir, binary string) (*exec.Cmd, func(), error) {
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	command := exec.Command(binary)
	command.Dir = dir
	command.Stdin = null
	command.Stdout = null
	command.Stderr = null
	command.Env = withoutEnvironment(os.Environ(), "NO_CONTROL")
	isolateOfficeProcessGroup(command)
	return command, func() { _ = null.Close() }, nil
}

func withoutEnvironment(environment []string, name string) []string {
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if key != name {
			filtered = append(filtered, entry)
		}
	}
	return filtered
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
		launcher:  commandLauncher{},
		starting:  make(map[string]bool),
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
	if r.URL.Path == apiPrefix+"/exec" {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		g.execCommand(w, r)
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
	case suffix == "/message":
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	case r.Method == http.MethodPost && suffix == "/stop":
		g.proxy(w, r, id, http.MethodPost, control.RouteStop, nil)
	case r.Method == http.MethodPost && suffix == "/new":
		g.proxy(w, r, id, http.MethodPost, control.RouteSessionNew, nil)
	case r.Method == http.MethodPost && suffix == "/start":
		g.start(w, r, id)
	case suffix == "/start":
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid transcript limit")
			return
		}
	}
	path := control.RouteTranscript
	query := url.Values{}
	if limit != "" {
		query.Set("limit", strconv.Itoa(mustAtoi(limit)))
	}
	if before, present := r.URL.Query()["before"]; present {
		query["before"] = before
	}
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	g.proxy(w, r, id, http.MethodGet, path, nil)
}

func (g *gateway) message(w http.ResponseWriter, r *http.Request, id string) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusBadRequest, "content type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxMessageBodyBytes)
	defer r.Body.Close()
	var body gatewayMessageRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "message body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid message body")
		return
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "message body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid message body")
		return
	}
	body.Text = strings.TrimSpace(body.Text)
	_, err = validateAttachments(body.Attachments)
	if err != nil {
		if strings.Contains(err.Error(), "too large") {
			writeError(w, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Text == "" && len(body.Attachments) == 0 {
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

func (g *gateway) start(w http.ResponseWriter, r *http.Request, id string) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMessageBytes)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) != 0 {
		writeError(w, http.StatusBadRequest, "start request body must be empty")
		return
	}

	g.startMu.Lock()
	if g.starting[id] {
		g.startMu.Unlock()
		writeError(w, http.StatusConflict, "office already running")
		return
	}
	project, err := g.get(r.Context(), id, 10*time.Second)
	if err != nil {
		g.startMu.Unlock()
		if errors.Is(err, projects.ErrNotFound) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		g.projectError(w, err)
		return
	}
	if project.Live {
		g.startMu.Unlock()
		writeError(w, http.StatusConflict, "office already running")
		return
	}
	if project.Dir == "" {
		g.startMu.Unlock()
		writeError(w, http.StatusBadGateway, "could not start office")
		return
	}
	g.starting[id] = true
	g.startMu.Unlock()

	if err := g.launcher.Launch(project.Dir, func() {
		g.startMu.Lock()
		delete(g.starting, id)
		g.startMu.Unlock()
	}); err != nil {
		log.Printf("floorgate: start office for project %q: %v", id, err)
		g.startMu.Lock()
		delete(g.starting, id)
		g.startMu.Unlock()
		writeError(w, http.StatusBadGateway, "could not start office")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"id": project.ID, "startRequested": true})
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
		g.proxyOfficeError(w, response)
		return
	}
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, response.Body)
}

func (g *gateway) proxyOfficeError(w http.ResponseWriter, response *http.Response) {
	body, err := io.ReadAll(response.Body)
	if err != nil || len(body) == 0 || !json.Valid(body) {
		writeError(w, response.StatusCode, "office returned an error")
		return
	}
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(body)
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
