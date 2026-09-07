// Package projects enumerates persisted theboringfloor projects and discovers
// offices that currently answer their local control API.
package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/brand"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/control"
)

const maxProbeWorkers = 16

var (
	// ErrNotFound reports that no persisted project directory has the requested ID.
	ErrNotFound = errors.New("projects: not found")
	// ErrNotLive reports that a project does not have a reachable live office.
	ErrNotLive = errors.New("projects: office not running")
)

// Project is one theboringfloor project known to this machine.
type Project struct {
	ID        string `json:"id"`
	Dir       string `json:"dir"`
	Name      string `json:"name"`
	Live      bool   `json:"live"`
	Backend   string `json:"backend"`
	PrimaryID string `json:"primaryId"`
	Port      int    `json:"port"`
	Version   string `json:"version"`
	SavedAt   int64  `json:"savedAt"`
	ChatCount int    `json:"chatCount"`
}

// Root returns the projects root directory: <home>/.theboringfloor/projects.
func Root() string {
	home := config.HomeOverride()
	if home == "" {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, brand.DotDir, "projects")
}

// IsSystemDir reports whether dir is an ephemeral system location that should
// normally be excluded from project lists. It recognizes /var, /private/var,
// /tmp, and /private/tmp, including their descendants.
func IsSystemDir(dir string) bool {
	cleaned := path.Clean(filepath.ToSlash(dir))
	if !path.IsAbs(cleaned) {
		return false
	}
	segments := strings.Split(strings.TrimPrefix(cleaned, "/"), "/")
	for _, root := range [][]string{
		{"var"},
		{"private", "var"},
		{"tmp"},
		{"private", "tmp"},
	} {
		if len(segments) < len(root) {
			continue
		}
		matches := true
		for i, segment := range root {
			if segments[i] != segment {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

// List enumerates non-system project directories under Root. Projects with a
// live office are probed concurrently, with each probe bounded by timeout.
// Results are sorted with live projects first, then newest saved snapshots,
// then ID. Use ListAll to include projects rooted in ephemeral system paths.
func List(ctx context.Context, timeout time.Duration) ([]Project, error) {
	return list(ctx, timeout, true)
}

// ListAll enumerates every project directory under Root, including projects
// rooted in ephemeral system paths. Most callers should use List.
func ListAll(ctx context.Context, timeout time.Duration) ([]Project, error) {
	return list(ctx, timeout, false)
}

func list(ctx context.Context, timeout time.Duration, filterSystemDirs bool) ([]Project, error) {
	entries, err := os.ReadDir(Root())
	if errors.Is(err, os.ErrNotExist) {
		return []Project{}, nil
	}
	if err != nil {
		return nil, err
	}

	projects := make([]Project, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "" || entry.Name()[0] == '.' {
			continue
		}
		project := readProject(entry.Name())
		if filterSystemDirs && IsSystemDir(project.Dir) {
			continue
		}
		projects = append(projects, project)
	}

	probeProjects(ctx, projects, timeout)
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Live != projects[j].Live {
			return projects[i].Live
		}
		if projects[i].SavedAt != projects[j].SavedAt {
			return projects[i].SavedAt > projects[j].SavedAt
		}
		return projects[i].ID < projects[j].ID
	})
	return projects, nil
}

// Get returns a single project by its ID. ErrNotFound is returned when the ID
// does not name a project directory below Root.
func Get(ctx context.Context, id string, timeout time.Duration) (Project, error) {
	if !validID(id) {
		return Project{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	info, err := os.Stat(filepath.Join(Root(), id))
	if err != nil || !info.IsDir() {
		return Project{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	project := readProject(id)
	if discovery, ok := discoveryFor(project); ok {
		if health, ok := probe(ctx, discovery, timeout); ok {
			applyHealth(&project, discovery, health)
		}
	}
	return project, nil
}

// Discovery returns the authenticated discovery record for a live project.
// ErrNotLive is returned when its discovery record is absent, unreadable, or
// does not answer a health probe.
func Discovery(id string) (control.Discovery, error) {
	if !validID(id) {
		return control.Discovery{}, fmt.Errorf("%w: %s", ErrNotLive, id)
	}
	info, err := os.Stat(filepath.Join(Root(), id))
	if err != nil || !info.IsDir() {
		return control.Discovery{}, fmt.Errorf("%w: %s", ErrNotLive, id)
	}
	project := readProject(id)
	discovery, ok := discoveryFor(project)
	if !ok {
		return control.Discovery{}, fmt.Errorf("%w: %s", ErrNotLive, id)
	}
	if _, ok := probe(context.Background(), discovery, 3*time.Second); !ok {
		return control.Discovery{}, fmt.Errorf("%w: %s", ErrNotLive, id)
	}
	return discovery, nil
}

func validID(id string) bool {
	return id != "" && id != "." && id != ".." && filepath.Base(id) == id
}

func readProject(id string) Project {
	project := Project{ID: id}
	b, err := os.ReadFile(filepath.Join(Root(), id, "session.json"))
	if err != nil {
		return project
	}
	var session app.SessionFile
	if json.Unmarshal(b, &session) != nil {
		return project
	}
	project.Dir = session.Dir
	project.Backend = session.Backend
	project.PrimaryID = session.PrimaryID
	project.SavedAt = session.SavedAt
	project.ChatCount = len(session.Chat)
	if project.Dir != "" {
		project.Name = filepath.Base(project.Dir)
	}
	return project
}

func discoveryFor(project Project) (control.Discovery, bool) {
	if project.Dir == "" || project.ID != control.DirHash(project.Dir) {
		return control.Discovery{}, false
	}
	return control.ReadDiscovery(project.Dir)
}

func probeProjects(ctx context.Context, projects []Project, timeout time.Duration) {
	jobs := make(chan int)
	workers := min(maxProbeWorkers, len(projects))
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				discovery, ok := discoveryFor(projects[index])
				if !ok {
					continue
				}
				health, ok := probe(ctx, discovery, timeout)
				if ok {
					applyHealth(&projects[index], discovery, health)
				}
			}
		}()
	}
	for index := range projects {
		jobs <- index
	}
	close(jobs)
	wg.Wait()
}

func applyHealth(project *Project, discovery control.Discovery, health control.HealthResponse) {
	project.Live = true
	project.Port = discovery.Port
	project.Version = health.Version
	project.Backend = health.Backend
}

func probe(ctx context.Context, discovery control.Discovery, timeout time.Duration) (control.HealthResponse, bool) {
	if discovery.Port <= 0 || discovery.Token == "" || discovery.Dir == "" {
		return control.HealthResponse{}, false
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{Proxy: nil},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"http://127.0.0.1:"+strconv.Itoa(discovery.Port)+control.RouteHealth, nil)
	if err != nil {
		return control.HealthResponse{}, false
	}
	request.Header.Set("Authorization", "Bearer "+discovery.Token)
	response, err := client.Do(request)
	if err != nil {
		return control.HealthResponse{}, false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return control.HealthResponse{}, false
	}
	var health control.HealthResponse
	if json.NewDecoder(response.Body).Decode(&health) != nil || !health.OK || health.Dir != discovery.Dir {
		return control.HealthResponse{}, false
	}
	// Do not use Discovery.Stale here: its executable-name check reports a
	// false negative when a live theboringfloor binary is invoked as an alias.
	return health, true
}
