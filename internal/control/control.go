// Package control is the shared contract between the office (server side) and
// the thefloor_mcp binary (client side). It deliberately has no HTTP or app
// dependencies so both sides can use its wire types and coordination primitives
// without coupling to the TUI.
package control

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/brand"
	"github.com/theboringhumane/theboringfloor/internal/config"
)

const (
	RouteHealth      = "/v1/health"
	RoutePlan        = "/v1/plan"
	RoutePlanPresent = "/v1/plan/present"
	RoutePlanUpdate  = "/v1/plan/update"
	RouteTranscript  = "/v1/transcript"
	RouteStatus      = "/v1/status"

	QueryPlan       = "plan"
	QueryTranscript = "transcript"
	QueryStatus     = "status"

	ReplyDeadline = 2 * time.Second
)

// HealthResponse is the health endpoint response.
type HealthResponse struct {
	OK      bool   `json:"ok"`
	Dir     string `json:"dir"`
	Version string `json:"version"`
	Backend string `json:"backend"`
}

// PlanResponse is the plan endpoint response.
type PlanResponse struct {
	Draft       string `json:"draft"`
	Approved    string `json:"approved"`
	HasApproved bool   `json:"hasApproved"`
}

// PlanWriteRequest is the body accepted by plan present and update endpoints.
type PlanWriteRequest struct {
	Text string `json:"text"`
}

// OKResponse is a successful mutation response.
type OKResponse struct {
	OK bool `json:"ok"`
}

// TranscriptResponse is the transcript endpoint response.
type TranscriptResponse struct {
	Messages  []TranscriptMessage `json:"messages"`
	Truncated bool                `json:"truncated"`
}

// TranscriptMessage is one projected chat message.
type TranscriptMessage struct {
	ID   string `json:"id"`
	From string `json:"from"`
	Kind string `json:"kind"`
	Text string `json:"text"`
	At   int64  `json:"at"`
}

// StatusResponse is the status endpoint response.
type StatusResponse struct {
	Dir             string `json:"dir"`
	Backend         string `json:"backend"`
	PrimaryID       string `json:"primaryId"`
	PlanDraftLen    int    `json:"planDraftLen"`
	PlanApprovedLen int    `json:"planApprovedLen"`
	ChatCount       int    `json:"chatCount"`
}

// ErrorResponse is the body used for control API errors.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Discovery identifies a running office for one project directory.
type Discovery struct {
	PID       int    `json:"pid"`
	Port      int    `json:"port"`
	Token     string `json:"token"`
	Dir       string `json:"dir"`
	StartedAt int64  `json:"startedAt"`
	Version   string `json:"version"`
	BootID    string `json:"bootId,omitempty"`
}

var discoveryMu sync.Mutex

// NewToken returns a 32-byte cryptographically random token encoded as hex.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// DirHash returns the sha1 hash of dir's canonical absolute path. Symlinks are
// resolved when possible, matching the persisted office session path.
func DirHash(dir string) string {
	canon, err := filepath.Abs(dir)
	if err != nil {
		canon = dir
	}
	if eval, err := filepath.EvalSymlinks(canon); err == nil {
		canon = eval
	}
	sum := sha1.Sum([]byte(canon))
	return hex.EncodeToString(sum[:])
}

func home() string {
	if value := config.Env("HOME"); value != "" {
		return value
	}
	return os.Getenv("HOME")
}

// ControlPath returns the discovery file path for dir.
func ControlPath(dir string) string {
	return filepath.Join(home(), brand.DotDir, "projects", DirHash(dir), "control.json")
}

// WriteDiscovery atomically writes d to dir's project discovery file.
func WriteDiscovery(dir string, d Discovery) error {
	discoveryMu.Lock()
	defer discoveryMu.Unlock()
	if d.BootID == "" {
		bootID, err := NewToken()
		if err != nil {
			return err
		}
		d.BootID = bootID
	}
	path := ControlPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".control-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// ReadDiscovery reads dir's discovery file. Missing, unreadable, and malformed
// files all return ok=false.
func ReadDiscovery(dir string) (d Discovery, ok bool) {
	b, err := os.ReadFile(ControlPath(dir))
	if err != nil || json.Unmarshal(b, &d) != nil {
		return Discovery{}, false
	}
	return d, true
}

// RemoveDiscovery removes dir's discovery file. A missing file is not an error.
func RemoveDiscovery(dir string) error {
	discoveryMu.Lock()
	defer discoveryMu.Unlock()
	err := os.Remove(ControlPath(dir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// PruneStale removes dir's discovery file only when its current record is
// stale. It is safe to call concurrently: a replacement record is re-read
// before removal so a live replacement is never removed.
func PruneStale(dir string) bool {
	discoveryMu.Lock()
	defer discoveryMu.Unlock()
	path := ControlPath(dir)
	d, ok := ReadDiscovery(dir)
	if !ok || !d.Stale() {
		return false
	}
	if err := os.Remove(path); err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	}
	return false
}

// Stale reports whether d cannot identify a live usable control server.
func (d Discovery) Stale() bool {
	if d.Port <= 0 || d.Token == "" || d.PID <= 0 {
		return true
	}
	p, err := os.FindProcess(d.PID)
	if err != nil {
		return true
	}
	if p.Signal(syscall.Signal(0)) != nil {
		return true
	}
	name, known := processExecutableName(d.PID)
	if !known {
		// Some platforms cannot inspect another process's executable. Preserve
		// the liveness-only behavior there rather than breaking control clients.
		return false
	}
	return name != brand.CLI && name != brand.CLI+".exe"
}

// processExecutableName returns the executable base name for pid where the OS
// exposes it. Unsupported platforms deliberately report known=false.
func processExecutableName(pid int) (name string, known bool) {
	switch runtime.GOOS {
	case "linux":
		path, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
		if err != nil {
			return "", false
		}
		return filepath.Base(path), true
	case "darwin", "freebsd":
		output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
		if err != nil {
			return "", false
		}
		path := strings.TrimSpace(string(output))
		if path == "" {
			return "", false
		}
		return filepath.Base(path), true
	default:
		return "", false
	}
}

// Registry lets an HTTP handler wait for a projection produced by the UI
// goroutine. Its reply channels are buffered so fulfilment cannot block.
type Registry struct {
	mu      sync.Mutex
	pending map[string]chan []byte
}

var requestSequence atomic.Uint64

// NewRegistry returns an empty pending-request registry.
func NewRegistry() *Registry {
	return &Registry{pending: make(map[string]chan []byte)}
}

// NewRequest creates a unique request ID and its reply channel.
func (r *Registry) NewRequest() (id string, reply <-chan []byte) {
	if r == nil {
		panic("control: nil Registry")
	}
	id = fmt.Sprintf("%x-%x", uint64(time.Now().UnixNano()), requestSequence.Add(1))
	ch := make(chan []byte, 1)
	r.mu.Lock()
	r.pending[id] = ch
	r.mu.Unlock()
	return id, ch
}

// Fulfill delivers payload to id once. It returns false when id is unknown or
// was already fulfilled or cancelled.
func (r *Registry) Fulfill(id string, payload []byte) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	ch, ok := r.pending[id]
	if ok {
		delete(r.pending, id)
	}
	r.mu.Unlock()
	if !ok {
		return false
	}
	ch <- payload
	return true
}

// Cancel discards a pending request. It is safe to call after fulfilment.
func (r *Registry) Cancel(id string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	delete(r.pending, id)
	r.mu.Unlock()
}
