// Package workspace stores project floors independently of agent sessions.
// Floor metadata is small; transcripts live in separate conversation archives.
package workspace

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/brand"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/control"
)

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Check struct {
	Text string `json:"text"`
	Done bool   `json:"done"`
}
type Ticket struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	Team        string  `json:"team,omitempty"`
	Owner       string  `json:"owner,omitempty"`
	Session     string  `json:"session,omitempty"`
	Checklist   []Check `json:"checklist,omitempty"`
	Created     int64   `json:"created"`
	Updated     int64   `json:"updated"`
}
type Floor struct {
	Dir     string   `json:"dir"`
	Name    string   `json:"name"`
	Teams   []Team   `json:"teams"`
	Tickets []Ticket `json:"tickets,omitempty"`
	Updated int64    `json:"updated"`
}
type Conversation struct {
	ID       string `json:"id"`
	Backend  string `json:"backend"`
	Title    string `json:"title"`
	Team     string `json:"team,omitempty"`
	Updated  int64  `json:"updated"`
	Messages int    `json:"messages"`
}

var Statuses = []string{"backlog", "in-progress", "blocked", "review", "done"}
var Priorities = []string{"P0", "P1", "P2", "P3"}
var mu sync.Mutex

func Root() string {
	home := config.HomeOverride()
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, brand.DotDir, "projects")
}
func Dir(dir string) string  { return filepath.Join(Root(), control.DirHash(dir)) }
func Path(dir string) string { return filepath.Join(Dir(dir), "floor.json") }
func Default(dir string) Floor {
	return Floor{Dir: dir, Name: filepath.Base(dir), Teams: []Team{{"ui", "UI"}, {"coding", "Coding"}, {"frontend", "Frontend"}, {"backend", "Backend"}}}
}
func ID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b)
}
func Canonical(dir string) (string, error) {
	if dir == "~" {
		dir, _ = os.UserHomeDir()
	} else if strings.HasPrefix(dir, "~/") {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, dir[2:])
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", errors.New("choose a project directory")
	}
	return abs, nil
}
func Load(dir string) (Floor, error) {
	raw, err := os.ReadFile(Path(dir))
	if os.IsNotExist(err) {
		return Default(dir), nil
	}
	if err != nil {
		return Floor{}, err
	}
	var f Floor
	err = json.Unmarshal(raw, &f)
	if err == nil && (f.Dir == "" || control.DirHash(f.Dir) != control.DirHash(dir)) {
		err = errors.New("floor directory does not match")
	}
	return f, err
}
func Register(dir string) (Floor, error) {
	canon, err := Canonical(dir)
	if err != nil {
		return Floor{}, err
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := Load(canon)
	if err != nil {
		return f, err
	}
	if _, err := os.Stat(Path(canon)); err == nil {
		return f, nil
	}
	f.Updated = time.Now().UnixMilli()
	return f, WriteJSON(Path(canon), f)
}
func Update(dir string, edit func(*Floor) error) (Floor, error) {
	mu.Lock()
	defer mu.Unlock()
	f, err := Load(dir)
	if err != nil {
		return f, err
	}
	if err = edit(&f); err != nil {
		return f, err
	}
	f.Updated = time.Now().UnixMilli()
	return f, WriteJSON(Path(dir), f)
}
func List() ([]Floor, error) {
	entries, err := os.ReadDir(Root())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Floor
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		base := filepath.Join(Root(), e.Name())
		raw, err := os.ReadFile(filepath.Join(base, "floor.json"))
		var f Floor
		if err == nil {
			if json.Unmarshal(raw, &f) != nil {
				continue
			}
		} else { // discover pre-floor installations without rewriting their snapshots
			raw, err = os.ReadFile(filepath.Join(base, "session.json"))
			if err != nil {
				continue
			}
			var old struct {
				Dir string `json:"dir"`
			}
			if json.Unmarshal(raw, &old) != nil || old.Dir == "" {
				continue
			}
			f = Default(old.Dir)
		}
		if f.Dir == "" || control.DirHash(f.Dir) != e.Name() {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}
func WriteJSON(path string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".floor-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
func ConversationDir(dir, backend, id string) string {
	return filepath.Join(Dir(dir), "conversations", fmt.Sprintf("%x", sha256.Sum256([]byte(backend+":"+id))))
}
func Conversations(dir string) ([]Conversation, error) {
	entries, err := os.ReadDir(filepath.Join(Dir(dir), "conversations"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Conversation
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(Dir(dir), "conversations", e.Name(), "meta.json"))
		if err != nil {
			continue
		}
		var c Conversation
		if json.Unmarshal(raw, &c) == nil && c.ID != "" {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Updated > out[j].Updated })
	return out, nil
}
func PutTicket(dir string, t Ticket) (Floor, error) {
	t.Title = strings.TrimSpace(t.Title)
	if t.Title == "" {
		return Floor{}, errors.New("a ticket needs a title")
	}
	if !contains(Statuses, t.Status) || !contains(Priorities, t.Priority) {
		return Floor{}, errors.New("invalid status or priority")
	}
	return Update(dir, func(f *Floor) error {
		if t.Team != "" {
			found := false
			for _, team := range f.Teams {
				found = found || team.ID == t.Team
			}
			if !found {
				return errors.New("unknown team")
			}
		}
		t.Updated = time.Now().UnixMilli()
		for i, old := range f.Tickets {
			if old.ID == t.ID {
				t.Created = old.Created
				f.Tickets[i] = t
				return nil
			}
		}
		if t.ID == "" {
			t.ID = ID("TKT")
		}
		t.Created = t.Updated
		f.Tickets = append(f.Tickets, t)
		return nil
	})
}
func AddTeam(dir, name string) (Floor, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Floor{}, errors.New("a team needs a name")
	}
	return Update(dir, func(f *Floor) error {
		for _, t := range f.Teams {
			if strings.EqualFold(t.Name, name) {
				return errors.New("team already exists")
			}
		}
		f.Teams = append(f.Teams, Team{ID: ID("team"), Name: name})
		return nil
	})
}
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
