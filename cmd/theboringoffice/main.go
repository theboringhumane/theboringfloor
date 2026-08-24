// theboringoffice — the terminal office (Go). Entry point.
//
//	theboringoffice                 live mode: spawn/attach `opencode serve` for <cwd>
//	theboringoffice --demo          touring mode: simulated events (explicitly labeled)
//	theboringoffice --server URL    attach to an existing server, don't spawn
//	theboringoffice -s SESSION      resume a specific past opencode chat session by id
//	                                (the --session long form; beats the saved-session
//	                                restore and its 4-day freshness gate for this boot —
//	                                /session in-app prints the id)
//	theboringoffice --autokill 6s   exit after duration (CI / screenshot runs)
//	theboringoffice --version       print version and exit
//
// In-app, a /session picker accept swaps sessions by QUIT + EXEC-REPLACE:
// the app quits cleanly (terminal reaped, session persisted, serve child
// stopped) and this process syscall.Exec's the same binary as
// `theboringoffice -s <id>` (see the post-Run block below).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringoffice/internal/app"
	"github.com/theboringhumane/theboringoffice/internal/backend"
	"github.com/theboringhumane/theboringoffice/internal/chrome"
	"github.com/theboringhumane/theboringoffice/internal/config"
	"github.com/theboringhumane/theboringoffice/internal/notify"
	"github.com/theboringhumane/theboringoffice/internal/office"
	"github.com/theboringhumane/theboringoffice/internal/panels"
	"github.com/theboringhumane/theboringoffice/internal/sound"
	"github.com/theboringhumane/theboringoffice/internal/state"
	"github.com/theboringhumane/theboringoffice/internal/version"
)

// sndBus adapts *sound.Bus (Play returns error) to the app's Play-void seam.
type sndBus struct{ *sound.Bus }

func (s sndBus) Play(name string) { _ = s.Bus.Play(name) }

// notifyBus adapts *notify.Bus (the engine call is Send) to the app's
// Notify-void seam; SetMode passes straight through for /notify's live
// toggle.
type notifyBus struct{ *notify.Bus }

func (n notifyBus) Notify(kind, title, body string) { n.Bus.Send(kind, title, body) }

// envOr reads the THEBORINGOFFICE_* env var, falling back to the pre-rename
// GRAFEIO_* name (whole-product rename: grafeio -> theboringoffice; old
// dotfiles, shell aliases and CI exports keep working).
func envOr(key, legacyKey string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return os.Getenv(legacyKey)
}

func main() {
	demo := flag.Bool("demo", envOr("THEBORINGOFFICE_DEMO", "GRAFEIO_DEMO") == "1", "run with simulated events")
	server := flag.String("server", "", "opencode serve URL (attach, don't spawn)")
	session := flag.String("session", "", "resume this opencode chat session id (explicit pin; beats the saved-session restore)")
	sessionShort := flag.String("s", "", "shorthand for -session")
	autokill := flag.Duration("autokill", 0, "exit after this duration (shots/CI)")
	theme := flag.String("theme", "", "color theme: noir|paper|mono|dracula|solarized")
	printCfg := flag.Bool("print-default-config", false, "print the default brain.json and exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return
	}

	if *printCfg {
		b, _ := json.MarshalIndent(config.Default(), "", "  ")
		fmt.Println(string(b))
		fmt.Fprintln(os.Stderr, "(written to "+config.Path()+" on first run)")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[theboringoffice] brain.json: %v (using defaults)\n", err)
		cfg = config.Default()
	}
	if v := envOr("THEBORINGOFFICE_SERVER", "GRAFEIO_SERVER"); v != "" && *server == "" {
		*server = v
	}
	// session-pin precedence: --session > -s > THEBORINGOFFICE_SESSION
	// (GRAFEIO_SESSION fallback) — same shape as the server/theme overlays.
	if *session == "" {
		*session = *sessionShort
	}
	if v := envOr("THEBORINGOFFICE_SESSION", "GRAFEIO_SESSION"); v != "" && *session == "" {
		*session = v
	}
	// theme precedence: --theme flag > THEBORINGOFFICE_THEME (GRAFEIO_THEME fallback) > brain.json ui.theme > persisted > default
	if v := envOr("THEBORINGOFFICE_THEME", "GRAFEIO_THEME"); v != "" && *theme == "" {
		*theme = v
	}
	if *theme == "" {
		*theme = cfg.UI.Theme
	}
	if *theme == "" {
		*theme = chrome.LoadPersistedTheme()
	}
	if *theme != "" {
		if chrome.SetTheme(*theme) {
			office.SetTheme(*theme)
		}
	}

	var b state.Backend
	if *demo {
		b = backend.NewDemo(cfg)
	} else {
		b = backend.NewLive(*server, mustGetwd(), cfg)
	}

	app.SpawnTerminal = func(cols, rows int) (app.TerminalTab, error) {
		return panels.NewTerminal(cols, rows)
	}

	model := app.New(b, cfg, app.WithResumeSession(*session))
	if cfg.UI.Sounds != "" && cfg.UI.Sounds != "off" {
		model.SetSoundBus(sndBus{sound.NewBus(cfg.UI.Sounds, "")})
	}
	// UNCOUPLED from the sounds gate above: a muted speaker config must
	// never mute the desktop look-away pings. NewBus itself normalizes the
	// mode ("" → on, unknown → off) and honors THEBORINGOFFICE_NO_NOTIFY —
	// and wiring unconditionally keeps /notify on able to flip live.
	model.SetNotifyBus(notifyBus{notify.NewBus(cfg.UI.Notifications)})

	p := tea.NewProgram(model)

	// theme auto mode: with nothing pinned anywhere, ask the terminal for
	// its background color (OSC 11) — the reply lands in app.Update as
	// tea.BackgroundColorMsg, which chrome.SetThemeAuto answers, and later
	// spontaneous color events (macOS dark↔light flip) re-theme live. Same
	// goroutine+p.Send pattern as the backend bridge below. Pinned path:
	// skip the request entirely.
	if *theme == "" {
		go p.Send(tea.RequestBackgroundColor())
	}

	// bridge backend goroutines -> tea loop
	go func() {
		if err := b.Start(func(e state.Event) { p.Send(e) }); err != nil {
			p.Send(state.Event{Kind: state.EvStatus, Text: "[theboringoffice] backend failed: " + err.Error()})
		}
	}()

	if *autokill > 0 {
		go func() {
			<-time.After(*autokill)
			p.Send(tea.Quit())
		}()
	}

	if _, err := p.Run(); err != nil {
		model.CloseTerminal() // external p.Quit() bypasses Update — reap the PTY
		model.PersistSession()
		_ = b.Stop() // the serve child dies with us — never leaked on fatal
		fmt.Fprintf(os.Stderr, "[theboringoffice] fatal: %v\n", err)
		os.Exit(1)
	}
	// Clean-exit order is BINDING: terminal reaped → session persisted →
	// serve child killed (BEFORE any exec below — an exec'd process can
	// never run a deferred Stop).
	model.CloseTerminal()
	model.PersistSession()
	_ = b.Stop()

	// /session picker accept = quit + exec-replace (the app recorded the
	// intent via ExecRequest): relaunch the same binary pinned to the
	// accepted session — the swap rides the boot's resolvePrimary instead
	// of an in-app re-anchor.
	if id := model.ExecRequest(); id != "" {
		binary, err := os.Executable()
		if err != nil {
			binary, err = exec.LookPath(os.Args[0]) // os.Executable miss → PATH twin
		}
		if err == nil {
			argv := []string{"theboringoffice", "-s", id}
			// Carry ONLY the attach target (+ the RESOLVED theme) forward;
			// --autokill and --demo are NEVER carried (the picker is
			// live-only).
			if *server != "" {
				argv = append(argv, "--server", *server)
			}
			if name := chrome.CurrentTheme().Name; name != "" {
				argv = append(argv, "--theme", name)
			}
			err = syscall.Exec(binary, argv, os.Environ())
		}
		// syscall.Exec only returns on failure (a failed lookup lands here
		// too): leave the member the exact resume command, exit 0 normally.
		if err != nil {
			fmt.Printf("session %s — resume: theboringoffice -s %s\n", id, id)
		}
		return
	}

	// Normal clean exit (every quit way): hand the member the exact resume
	// line for THIS office's primary. Empty (demo, unresolved, attach-mode
	// with none) prints NOTHING — an id is never invented.
	if id := model.PrimarySessionID(); id != "" {
		fmt.Printf("session %s — resume: theboringoffice -s %s\n", id, id)
	}
}

func mustGetwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}
