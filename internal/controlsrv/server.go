// Package controlsrv serves the office control API on a loopback-only socket.
// Every request requires the office's bearer token. Handlers never touch the
// Bubble Tea model; they only emit state.Events through the supplied sink, so
// the UI goroutine remains the sole owner of model state. Admission control
// bounds both accepted connections and concurrent read projections: read
// requests are rejected immediately before entering the pending-request
// registry when all slots are occupied.
package controlsrv

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	maxPlanBodyBytes        = 1 << 20
	defaultMaxInFlightReads = 16
	defaultMaxConnections   = 32
	maxHeaderBytes          = 16 << 10
)

// Options configures a loopback control server. Registry must be shared with
// the UI goroutine, which fulfills read projections after it receives events.
type Options struct {
	Dir          string
	Version      string
	Sink         func(state.Event)
	Token        string
	Registry     *control.Registry
	QueryTimeout time.Duration
	// MaxInFlightReads bounds read projections that wait for the UI goroutine.
	// Zero selects the default of 16.
	MaxInFlightReads int
	// MaxConnections bounds simultaneous accepted TCP connections. Zero selects
	// the default of 32.
	MaxConnections int
}

// Server is the loopback HTTP bridge to the UI event sink.
type Server struct {
	dir            string
	version        string
	sink           func(state.Event)
	token          string
	registry       *control.Registry
	queryTimeout   time.Duration
	readSlots      chan struct{}
	maxConnections int

	mu       sync.Mutex
	listener net.Listener
	http     *http.Server
}

// New constructs a Server. A nil event sink would make all requests unsafe, so
// it is rejected immediately.
func New(opts Options) *Server {
	if opts.Sink == nil {
		panic("controlsrv: nil Sink")
	}
	if opts.Registry == nil {
		opts.Registry = control.NewRegistry()
	}
	if opts.QueryTimeout <= 0 {
		opts.QueryTimeout = control.ReplyDeadline
	}
	if opts.MaxInFlightReads <= 0 {
		opts.MaxInFlightReads = defaultMaxInFlightReads
	}
	if opts.MaxConnections <= 0 {
		opts.MaxConnections = defaultMaxConnections
	}
	return &Server{
		dir: opts.Dir, version: opts.Version, sink: opts.Sink, token: opts.Token,
		registry: opts.Registry, queryTimeout: opts.QueryTimeout,
		readSlots: make(chan struct{}, opts.MaxInFlightReads), maxConnections: opts.MaxConnections,
	}
}

// Start binds an ephemeral loopback port and serves requests in the background.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return errors.New("controlsrv: already started")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.listener = &limitedListener{Listener: listener, slots: make(chan struct{}, s.maxConnections)}
	s.http = &http.Server{
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    maxHeaderBytes,
	}
	go func(server *http.Server, ln net.Listener) {
		_ = server.Serve(ln)
	}(s.http, listener)
	return nil
}

// Port returns the resolved loopback port, or zero before Start.
func (s *Server) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return 0
	}
	if address, ok := s.listener.Addr().(*net.TCPAddr); ok {
		return address.Port
	}
	return 0
}

// Token returns the configured bearer token.
func (s *Server) Token() string { return s.token }

// Close gracefully stops the HTTP server. It is safe to call more than once.
func (s *Server) Close() error {
	s.mu.Lock()
	server := s.http
	s.http = nil
	s.listener = nil
	s.mu.Unlock()
	if server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

// Addr returns the listener address for in-package tests and diagnostics.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !s.authorized(r) {
		s.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	switch r.URL.Path {
	case control.RouteHealth:
		if !s.requireMethod(w, r, http.MethodGet) {
			return
		}
		s.health(w)
	case control.RoutePlan:
		if !s.requireMethod(w, r, http.MethodGet) {
			return
		}
		s.read(w, control.QueryPlan, 0)
	case control.RoutePlanPresent:
		if !s.requireMethod(w, r, http.MethodPost) {
			return
		}
		s.planWrite(w, r, state.EvPlanPresent)
	case control.RoutePlanUpdate:
		if !s.requireMethod(w, r, http.MethodPost) {
			return
		}
		s.planWrite(w, r, state.EvPlanUpdate)
	case control.RouteTranscript:
		if !s.requireMethod(w, r, http.MethodGet) {
			return
		}
		limit, ok := s.transcriptLimit(w, r)
		if !ok {
			return
		}
		s.read(w, control.QueryTranscript, limit)
	case control.RouteStatus:
		if !s.requireMethod(w, r, http.MethodGet) {
			return
		}
		s.read(w, control.QueryStatus, 0)
	default:
		s.writeError(w, http.StatusNotFound, "not found")
	}
}

func (s *Server) authorized(r *http.Request) bool {
	const prefix = "Bearer "
	authorization := r.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, prefix) {
		return false
	}
	provided := authorization[len(prefix):]
	if len(provided) != len(s.token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(s.token)) == 1
}

func (s *Server) requireMethod(w http.ResponseWriter, r *http.Request, want string) bool {
	if r.Method == want {
		return true
	}
	w.Header().Set("Allow", want)
	s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	return false
}

func (s *Server) health(w http.ResponseWriter) {
	payload, result := s.query(control.QueryStatus, 0)
	if result == querySaturated {
		s.writeError(w, http.StatusServiceUnavailable, "office busy")
		return
	}
	if result == queryTimedOut {
		s.writeError(w, http.StatusGatewayTimeout, "office busy")
		return
	}
	var status control.StatusResponse
	if err := json.Unmarshal(payload, &status); err != nil {
		s.writeError(w, http.StatusInternalServerError, "invalid office response")
		return
	}
	s.writeJSON(w, http.StatusOK, control.HealthResponse{
		OK: true, Dir: s.dir, Version: s.version, Backend: status.Backend,
	})
}

func (s *Server) read(w http.ResponseWriter, query string, limit int) {
	payload, result := s.query(query, limit)
	if result == querySaturated {
		s.writeError(w, http.StatusServiceUnavailable, "office busy")
		return
	}
	if result == queryTimedOut {
		s.writeError(w, http.StatusGatewayTimeout, "office busy")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

type queryResult uint8

const (
	queryComplete queryResult = iota
	queryTimedOut
	querySaturated
)

func (s *Server) query(query string, limit int) ([]byte, queryResult) {
	// Read admission is intentionally immediate rather than queued. This keeps
	// a busy UI from creating unbounded handler goroutines or registry entries.
	select {
	case s.readSlots <- struct{}{}:
		defer func() { <-s.readSlots }()
	default:
		return nil, querySaturated
	}

	id, reply := s.registry.NewRequest()
	s.sink(state.Event{Kind: state.EvControlQuery, ControlReqID: id, ControlQuery: query, ControlLimit: limit})
	timer := time.NewTimer(s.queryTimeout)
	defer timer.Stop()
	select {
	case payload := <-reply:
		return payload, queryComplete
	case <-timer.C:
		s.registry.Cancel(id)
		return nil, queryTimedOut
	}
}

func (s *Server) planWrite(w http.ResponseWriter, r *http.Request, kind state.EventKind) {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxPlanBodyBytes)
	defer r.Body.Close()
	var request control.PlanWriteRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := ensureEOF(decoder); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	text := strings.TrimSpace(request.Text)
	if text == "" {
		s.writeError(w, http.StatusBadRequest, "empty plan text")
		return
	}
	s.sink(state.Event{Kind: kind, PlanToolText: text})
	s.writeJSON(w, http.StatusOK, control.OKResponse{OK: true})
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("extra JSON value")
	}
	return err
}

func (s *Server) transcriptLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return 0, true
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, true
	}
	if limit < 0 {
		s.writeError(w, http.StatusBadRequest, "invalid limit")
		return 0, false
	}
	if limit > 500 {
		s.writeError(w, http.StatusBadRequest, "limit exceeds 500")
		return 0, false
	}
	return limit, true
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, control.ErrorResponse{Error: message})
}

// limitedListener rejects excess connections immediately after accept instead
// of allowing net/http to create handlers for them. Each admitted connection
// releases its slot exactly once when the server closes it.
type limitedListener struct {
	net.Listener
	slots chan struct{}
}

func (l *limitedListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		select {
		case l.slots <- struct{}{}:
			return &limitedConn{Conn: conn, release: func() { <-l.slots }}, nil
		default:
			_ = conn.Close()
		}
	}
}

type limitedConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *limitedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}

var _ http.Handler = (*Server)(nil)
