// Package config — one file to run the office: ~/.theboringfloor/configs/brain.json
//
// The file is created with defaults on first run. Precedence:
// CLI flag > brain.json > persisted UI prefs (~/.config/theboringfloor/theme) > defaults.
//
// Rename-era compatibility: grafeio → theboringoffice → theboringfloor.
// Load merges prior dirs into ~/.theboringfloor (and ~/.config/theboringfloor).
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/theboringhumane/theboringfloor/internal/brand"
)

// HomeOverride returns the test/harness scratch-root override:
// THEBORINGFLOOR_HOME, then THEBORINGOFFICE_HOME, then GRAFEIO_HOME.
// "" means "use $HOME".
func HomeOverride() string { return brand.Get("HOME") }

func homeRoot() string {
	home := HomeOverride()
	if home == "" {
		home = os.Getenv("HOME")
	}
	return home
}

// Path returns the brain.json location, honoring HomeOverride() (tests).
func Path() string {
	return filepath.Join(homeRoot(), brand.DotDir, "configs", "brain.json")
}

// officePath is the theboringoffice-era brain.json. Read fallback only.
func officePath() string {
	return filepath.Join(homeRoot(), brand.OfficeDotDir, "configs", "brain.json")
}

// legacyPath is the grafeio-era brain.json. Read fallback only.
func legacyPath() string {
	return filepath.Join(homeRoot(), brand.GrafeioDotDir, "configs", "brain.json")
}

// PowerMode — battery posture of the whole app.
type PowerMode string

const (
	PowerAuto        PowerMode = "auto"        // adaptive tick: busy = fast, idle = slow
	PowerPerformance PowerMode = "performance" // always fast
	PowerSaver       PowerMode = "saver"       // slow tick, coalesced renders, quieter office
)

// ModelRef — a provider/model string opencode understands ("anthropic/claude-sonnet-4-5").
type ModelRef string

type BossConfig struct {
	Name  string   `json:"name"`  // display name: "boss (oikonomos)"
	Model ModelRef `json:"model"` // orch/boss model override (prompt-level); empty = server default
	// Concierge answers instantly when the boss's turn is occupied: its own
	// lightweight session that answers directly or dispatches its own developers.
	Concierge bool `json:"concierge"` // default true
}

type RoleConfig struct {
	Model      ModelRef `json:"model"`      // NOTE: honored when opencode supports per-agent model dispatch; documented in README
	NamePrefix string   `json:"namePrefix"` // roster naming seed (e.g. "tekton" -> tekton-1)
}

type UIConfig struct {
	Theme          string    `json:"theme"`          // empty = LoadPersistedTheme fallback
	Power          PowerMode `json:"power"`          // auto|performance|saver
	TickMs         int       `json:"tickMs"`         // 0 = power-mode default base (180/500)
	AmbientChatter bool      `json:"ambientChatter"` // office banter bubbles
	Sounds         string    `json:"sounds"`         // "on" | "bell" (terminal bell only) | "off"
	Notifications  string    `json:"notifications"`  // "on" | "off" — OS desktop pings while the terminal is unfocused
	Images         string    `json:"images"`         // "auto" (detect lane, ASCII fallback) | "ascii" (half-block raster always) | "off" (chips only)
	SidebarWidth   int       `json:"sidebarWidth"`   // right panel cols, 0 = default 80 (26..100)
	Compact        bool      `json:"compact"`        // compact layout mode
}

// ImagesDefault — the image-preview posture every brain.json resolves to
// when ui.images is absent/empty (pre-schema files keep meaning exactly
// what they meant: previews on).
const ImagesDefault = "auto"

// ValidImagesMode reports whether mode names a real image lane posture.
func ValidImagesMode(mode string) bool {
	return mode == "auto" || mode == "ascii" || mode == "off"
}

// AttributionDefault — the majdoor commit-attribution posture every
// brain.json resolves to when the top-level "attribution" key is
// absent/empty (pre-schema files keep meaning exactly what they meant:
// the majdoor's trailer rides every commit).
const AttributionDefault = "on"

// ValidAttribution reports whether v names a real attribution posture:
// "on" auto-installs the office's commit-msg hook (the MajdoorTrailer on
// every commit), "off" removes our hook and stamps nothing.
func ValidAttribution(v string) bool {
	return v == "on" || v == "off"
}

type BackendConfig struct {
	// Name selects the LLM transport the office boots on: "opencode" (the
	// default — `opencode serve` + SSE) or "claudecode" (the claude CLI in
	// headless stream-json mode). install.sh --backend seeds it, /backend
	// swaps and persists it mid-flight. "" backfills to "opencode" on Load
	// (pre-schema brain.json files keep meaning exactly what they meant).
	Name             string `json:"name,omitempty"`   // opencode|claudecode (default opencode)
	AgentmemoryURL   string `json:"agentmemoryUrl"`   // default localhost:3111
	Server           string `json:"server"`           // pinned opencode serve URL (else spawn)
	AgentmemoryPollS int    `json:"agentmemoryPollS"` // board sync seconds (default 5)
	// BossModel is the "provider/model" override ("anthropic/claude-sonnet-4")
	// the primary (boss) session's prompts carry. "" = server default.
	// Validation is deliberately lite: a value WITHOUT a "/" is kept as-is —
	// the wire layer skips it (opencode needs providerID AND modelID), and
	// a well-formed-but-wrong one errors visibly at the serve (the backend's
	// status line surfaces it and retries bare).
	BossModel string `json:"bossModel,omitempty"`
	// CTOModel is the comparable override for the office CTO
	// ("theboringcto"): the routing rule is title-based
	// (state.IsArchitectureBrief is the ONE matcher; the floor hires those
	// child sessions as the CTO). "" = server default. Same validation-lite
	// rule as BossModel.
	CTOModel string `json:"ctoModel,omitempty"`
}

// BackendNameDefault is the transport every brain.json resolves to when
// backend.name is absent/empty (pre-schema files, hand-trimmed configs).
const BackendNameDefault = "opencode"

// BackendNameClaude is the claude-code transport name (backend.NewClaude).
const BackendNameClaude = "claudecode"

// ValidBackendName reports whether name names a real transport.
func ValidBackendName(name string) bool {
	return name == BackendNameDefault || name == BackendNameClaude
}

// ResolvedName normalizes the selected transport: "" (and any omission)
// means the default. Callers that only DISPLAY the name use this; the
// constructor gate (ValidBackendName) validates the non-empty case.
func (c BackendConfig) ResolvedName() string {
	if c.Name == "" {
		return BackendNameDefault
	}
	return c.Name
}

type Config struct {
	Version int `json:"version"`
	// Attribution is the office-wide majdoor commit-attribution switch:
	// "on" (the default) makes boot install the office's commit-msg hook
	// into the current repo (every commit carries the MajdoorTrailer);
	// "off" removes our own hook and stamps nothing. The boot-time ensure
	// lives in internal/app (EnsureMajdoorHook).
	Attribution string                `json:"attribution"` // "on" | "off"
	Boss        BossConfig            `json:"boss"`
	Roles       map[string]RoleConfig `json:"roles"` // developer|scout|reviewer|runner|hr
	UI          UIConfig              `json:"ui"`
	Backend     BackendConfig         `json:"backend"`
}

// Default returns the stock config (also the file skeleton written on first boot).
func Default() *Config {
	return &Config{
		Version:     1,
		Attribution: AttributionDefault,
		Boss:        BossConfig{Name: "boss (oikonomos)", Model: "", Concierge: true},
		Roles: map[string]RoleConfig{
			"developer": {NamePrefix: "tekton"},
			"scout":     {NamePrefix: "skopos"},
			"reviewer":  {NamePrefix: "dikastes"},
			"runner":    {NamePrefix: "hemerodromos"},
			"hr":        {NamePrefix: "mnemosyne"},
		},
		UI: UIConfig{
			Theme:          "",
			Power:          PowerAuto,
			TickMs:         0,
			AmbientChatter: true,
			Sounds:         "on",
			Notifications:  "on",
			Images:         ImagesDefault,
			SidebarWidth:   0,
			Compact:        false,
		},
		Backend: BackendConfig{
			Name:             BackendNameDefault,
			AgentmemoryURL:   "http://localhost:3111",
			Server:           "",
			AgentmemoryPollS: 5,
		},
	}
}

// Load reads brain.json; creates it (parents + defaults) when absent.
// Prior dirs are merged into ~/.theboringfloor first (see MigrateHome).
// Unknown keys are tolerated; bad JSON returns the error (caller decides).
func Load() (*Config, error) {
	MigrateHome()
	p := Path()
	readFrom := p
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		if ob, oerr := os.ReadFile(officePath()); oerr == nil {
			b, err, readFrom = ob, nil, officePath()
		} else if lb, lerr := os.ReadFile(legacyPath()); lerr == nil {
			b, err, readFrom = lb, nil, legacyPath()
		}
	}
	if errors.Is(err, os.ErrNotExist) {
		cfg := Default()
		if werr := save(p, cfg); werr != nil {
			return cfg, fmt.Errorf("config: could not write default %s: %w", p, werr)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", readFrom, err)
	}
	cfg := Default()
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", readFrom, err)
	}
	if cfg.Boss.Name == "" {
		cfg.Boss.Name = "boss (oikonomos)"
	}
	if cfg.Backend.AgentmemoryURL == "" {
		cfg.Backend.AgentmemoryURL = "http://localhost:3111"
	}
	// Backend-name backfill: pre-schema brain.json files (written before
	// the install selector existed) carry no name key; they mean — and
	// must keep meaning — the opencode transport.
	if cfg.Backend.Name == "" {
		cfg.Backend.Name = BackendNameDefault
	}
	// Images backfill (same house rule as the backend name): a brain.json
	// written before the image preview landed carries no images key — it
	// means, and keeps meaning, previews on in auto mode.
	if cfg.UI.Images == "" {
		cfg.UI.Images = ImagesDefault
	}
	// Attribution backfill (same house rule): a brain.json written before
	// the attribution knob landed carries no key — it means, and keeps
	// meaning, attribution ON. A bogus value is tolerated the same way:
	// normalized to the default, never fatal — a typo must not silently
	// switch a default-on feature off.
	if !ValidAttribution(cfg.Attribution) {
		cfg.Attribution = AttributionDefault
	}
	return cfg, nil
}

// Save writes the current config back (used by in-app mutation commands).
func Save(cfg *Config) error {
	return save(Path(), cfg)
}

func save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
