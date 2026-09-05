// theboringfloor — the terminal office (Go). Entry point.
//
//	theboringfloor                 live mode: spawn/attach `opencode serve` for <cwd>
//	theboringfloor --demo          touring mode: simulated events (explicitly labeled)
//	theboringfloor --server URL    attach to an existing server, don't spawn
//	theboringfloor -s SESSION      resume a specific past opencode chat session by id
//	                                (the --session long form; beats the saved-session
//	                                restore and its 4-day freshness gate for this boot —
//	                                /session in-app prints the id)
//	theboringfloor --autokill 6s   exit after duration (CI / screenshot runs)
//	theboringfloor --backend NAME  LLM transport: opencode|claudecode
//	                                (beats brain.json backend.name this boot;
//	                                /backend swaps mid-flight and persists)
//	theboringfloor --version       print version and exit
//
// In-app, a /session picker accept swaps sessions by QUIT + EXEC-REPLACE:
// the app quits cleanly (terminal reaped, session persisted, serve child
// stopped) and this process syscall.Exec's the same binary as
// `theboringfloor -s <id>` (see the post-Run block below).
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

	"github.com/theboringhumane/theboringfloor/internal/app"
	"github.com/theboringhumane/theboringfloor/internal/backend"
	"github.com/theboringhumane/theboringfloor/internal/cellmetrics"
	"github.com/theboringhumane/theboringfloor/internal/chrome"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/controlsrv"
	"github.com/theboringhumane/theboringfloor/internal/mcpinstall"
	"github.com/theboringhumane/theboringfloor/internal/notify"
	"github.com/theboringhumane/theboringfloor/internal/office"
	"github.com/theboringhumane/theboringfloor/internal/panels"
	"github.com/theboringhumane/theboringfloor/internal/sound"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/version"
)

// sndBus adapts *sound.Bus (Play returns error) to the app's Play-void seam.
type sndBus struct{ *sound.Bus }

func (s sndBus) Play(name string) { _ = s.Bus.Play(name) }

// notifyBus adapts *notify.Bus (the engine call is Send) to the app's
// Notify-void seam; SetMode passes straight through for /notify's live
// toggle.
type notifyBus struct{ *notify.Bus }

func (n notifyBus) Notify(kind, title, body string) { n.Bus.Send(kind, title, body) }

func env(suffix string) string { return config.Env(suffix) }

func main() {
	demo := flag.Bool("demo", env("DEMO") == "1", "run with simulated events")
	server := flag.String("server", "", "opencode serve URL (attach, don't spawn)")
	session := flag.String("session", "", "resume this opencode chat session id (explicit pin; beats the saved-session restore)")
	sessionShort := flag.String("s", "", "shorthand for -session")
	autokill := flag.Duration("autokill", 0, "exit after this duration (shots/CI)")
	theme := flag.String("theme", "", "color theme: noir|paper|mono|dracula|solarized")
	backendName := flag.String("backend", "", "LLM transport: opencode|claudecode (brain.json backend.name is the persisted default)")
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
		fmt.Fprintf(os.Stderr, "[theboringfloor] brain.json: %v (using defaults)\n", err)
		cfg = config.Default()
	}
	// Majdoor attribution (brain.json top-level "attribution", default
	// on): install the office's commit-msg hook into the current repo —
	// or remove our own when off. One synchronous call, never fatal: a
	// repo-less cwd, a foreign hook, or a git hiccup must never block
	// boot (EnsureMajdoorHook's contract); the status is one short line.
	hookStatus, hookErr := app.EnsureMajdoorHook(mustGetwd(), cfg.Attribution == config.AttributionDefault)
	if hookErr != nil {
		fmt.Fprintf(os.Stderr, "[theboringfloor] attribution hook: %v\n", hookErr)
	} else {
		fmt.Fprintf(os.Stderr, "[theboringfloor] attribution hook: %s\n", hookStatus)
	}
	if v := env("SERVER"); v != "" && *server == "" {
		*server = v
	}
	// session-pin precedence: --session > -s > THEFLOOR_SESSION.
	if *session == "" {
		*session = *sessionShort
	}
	if v := env("SESSION"); v != "" && *session == "" {
		*session = v
	}
	// theme precedence: --theme flag > THEFLOOR_THEME > brain.json ui.theme > persisted > default
	if v := env("THEME"); v != "" && *theme == "" {
		*theme = v
	}
	// backend precedence (same overlay shape as server/theme above):
	// --backend flag > THEFLOOR_BACKEND >
	// brain.json backend.name (install.sh --backend's seed) > "opencode".
	if v := env("BACKEND"); v != "" && *backendName == "" {
		*backendName = v
	}
	if *backendName == "" {
		*backendName = cfg.Backend.ResolvedName()
	}
	if !config.ValidBackendName(*backendName) {
		fmt.Fprintf(os.Stderr, "[theboringfloor] --backend must be opencode|claudecode (got %q) — using opencode\n", *backendName)
		*backendName = config.BackendNameDefault
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
		// One resolver for boot AND the in-app /backend swap (app.backendFor):
		// the install-selected brain.json name boots the same transport
		// shape /backend re-constructs mid-flight.
		b = app.BackendFor(*backendName, *server, mustGetwd(), cfg)
	}

	app.SpawnTerminal = func(cols, rows int) (app.TerminalTab, error) {
		return panels.NewTerminal(cols, rows)
	}

	ps := config.LoadProjectSettings(mustGetwd())
	model := app.New(b, cfg, app.WithResumeSession(*session), app.WithServerURL(*server), app.WithBypassPermissions(ps.BypassPermissions))
	if cfg.UI.Sounds != "" && cfg.UI.Sounds != "off" {
		model.SetSoundBus(sndBus{sound.NewBus(cfg.UI.Sounds, "")})
	}
	// UNCOUPLED from the sounds gate above: a muted speaker config must
	// never mute the desktop look-away pings. NewBus itself normalizes the
	// mode ("" → on, unknown → off) and honors THEFLOOR_NO_NOTIFY —
	// and wiring unconditionally keeps /notify on able to flip live.
	model.SetNotifyBus(notifyBus{notify.NewBus(cfg.UI.Notifications)})

	// The premium browser lane's frame-splice seam (panels/zenbu_frame.go):
	// the renderer's every flush passes through the wrapper unchanged, and
	// the lane's live kitty images re-emit AFTER each flush at their
	// absolute cells (the Model publishes the registry per Frame). The
	// lane's direct deletes (suspend/close/quit) route through the same
	// wrapper's DirectEmit so they serialize with frame flushes. The
	// wrapper delegates Fd/Read to os.Stdout (bubbletea's ttyOutput +
	// colorprofile.Detect type-assert the output to term.File).
	frameOut := panels.NewZenbuFrameWriter(os.Stdout, panels.ZenbuRegistry())
	panels.SetZenbuEmit(frameOut.DirectEmit)
	// The cell-metrics probe (internal/cellmetrics): ask the terminal its
	// cell's pixel size (CSI 16t → CSI 6;<h>;<w>t) so the browser tab's
	// headless shots size their viewport in TRUE pixels instead of the
	// 9x18 guess. The query rides the frame writer's serialized
	// DirectEmit (never interleaved mid-frame); the answer is snipped out
	// of stdin by the input wrapper BEFORE bubbletea's parser (no new msg
	// type reaches the app). Non-answering terminals (tmux, iTerm) stay
	// on the fallback after one 150ms window; a late answer still lands
	// for the next shot.
	cellmetrics.SetQueryFunc(func() { frameOut.DirectEmit(cellmetrics.QueryCellSize) })
	cellmetrics.Query() // the boot probe — BEFORE p.Run
	p := tea.NewProgram(model,
		tea.WithOutput(frameOut),
		tea.WithInput(cellmetrics.WrapInput(os.Stdin)),
		// Re-arm the probe on every WindowSizeMsg after the boot's first
		// (font zoom — ctrl+= / ctrl+- in ghostty/kitty — changes the
		// cell's px; Requery owns the skip-first). The msg itself passes
		// through untouched — the app's resize routing is unchanged.
		tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if _, ok := msg.(tea.WindowSizeMsg); ok {
				cellmetrics.Requery()
			}
			return msg
		}),
	)

	// theme auto mode: with nothing pinned anywhere, ask the terminal for
	// its background color (OSC 11) — the reply lands in app.Update as
	// tea.BackgroundColorMsg, which chrome.SetThemeAuto answers, and later
	// spontaneous color events (macOS dark↔light flip) re-theme live. Same
	// goroutine+p.Send pattern as the backend bridge below. Pinned path:
	// skip the request entirely.
	if *theme == "" {
		go p.Send(tea.RequestBackgroundColor())
	}

	// bridge backend goroutines -> tea loop. The sink is SHARED with the
	// app's /backend swap: the mid-flight replacement transport Starts on
	// the very same lane, so its events arrive exactly like the boot's.
	sink := func(e state.Event) { p.Send(e) }
	model.SetEventSink(sink)
	// THEFLOOR_NO_CONTROL=1 disables the optional loopback control API:
	// no listener is bound and no discovery file is written. Its normal path
	// stays outside the model so HTTP handlers only enqueue state.Events and
	// the Bubble Tea goroutine remains the sole reader of live UI state.
	var controlServer *controlsrv.Server
	controlDir := mustGetwd()
	if !config.EnvBool("NO_CONTROL") {
		token, tokenErr := control.NewToken()
		if tokenErr != nil {
			go p.Send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] control API disabled: " + tokenErr.Error()})
		} else {
			registry := control.NewRegistry()
			app.SetControlRegistry(registry)
			candidate := controlsrv.New(controlsrv.Options{
				Dir: controlDir, Version: version.String(), Sink: sink, Token: token, Registry: registry,
			})
			if startErr := candidate.Start(); startErr != nil {
				go p.Send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] control API disabled: " + startErr.Error()})
			} else if discoveryErr := control.WriteDiscovery(controlDir, control.Discovery{
				PID: os.Getpid(), Port: candidate.Port(), Token: token, Dir: controlDir,
				StartedAt: time.Now().UnixMilli(), Version: version.String(),
			}); discoveryErr != nil {
				_ = candidate.Close()
				go p.Send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] control API disabled: " + discoveryErr.Error()})
			} else {
				controlServer = candidate
			}
		}
	}
	stopControl := func() {
		if controlServer == nil {
			return
		}
		if err := control.RemoveDiscovery(controlDir); err != nil {
			fmt.Fprintf(os.Stderr, "[theboringfloor] control discovery cleanup: %v\n", err)
		}
		if err := controlServer.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "[theboringfloor] control server cleanup: %v\n", err)
		}
	}
	// Register thefloor_mcp with the host agents once per boot. This begins off
	// the UI path and is never fatal: a missing companion binary, an
	// unparseable host config, or an absent claude CLI all degrade to a status
	// line. But the OpenCode backend's charter pass discovers MCP servers at
	// Start, so Start MUST wait for this write to finish; otherwise a first-boot
	// attachment can permanently miss the server that this same boot adds.
	// THEFLOOR_NO_MCP_INSTALL=1 opts out entirely.
	mcpInstallDone := make(chan struct{})
	go func() {
		defer close(mcpInstallDone)
		binPath, ok := mcpinstall.ResolveBinary()
		if !ok {
			return
		}
		res, err := mcpinstall.Ensure(binPath)
		if err != nil {
			p.Send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] mcp register: " + err.Error()})
			return
		}
		// Only speak up when something actually changed — an already-present
		// registration must stay silent on every subsequent boot.
		if res.OpenCode.Status == "installed" || res.Claude.Status == "installed" {
			p.Send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] thefloor_mcp registered — opencode: " + res.OpenCode.Status + ", claude: " + res.Claude.Status})
		}
	}()
	go func() {
		// Establish the registration -> charter happens-before edge. In
		// particular, liveBackend.Start calls EnsureCharter before spawning
		// OpenCode, and its MCP attachment must discover the just-registered
		// thefloor_mcp rather than a stale pre-registration config snapshot.
		<-mcpInstallDone
		if err := b.Start(sink); err != nil {
			p.Send(state.Event{Kind: state.EvStatus, Text: "[theboringfloor] backend failed: " + err.Error()})
		}
	}()

	if *autokill > 0 {
		go func() {
			<-time.After(*autokill)
			p.Send(tea.Quit())
		}()
	}

	finalModel, err := p.Run()
	// Sweep the premium lane's terminal-side images BEFORE anything else:
	// the clean quit paths already deleted through the lane Close (these
	// are hushed dupes), but a FATAL exit skips Update — the a=d sweep
	// keeps the child's frames from ghosting over the restored main
	// screen. Seals the wrapper (passthrough-only) either way.
	frameOut.Finish()
	// The LIVE model is the one p.Run() hands back: every Update (the
	// /session picker accept's exec intent included) ran on ITS value
	// chain — the pre-Run `model` variable is a stale copy and must NEVER
	// be read for teardown (reading it silently swallowed the exec-replace:
	// the accept quit cleanly, printed the plain resume line and DIED).
	fm := model
	if m, ok := finalModel.(app.Model); ok {
		fm = m
	}
	if err != nil {
		fm.CloseTerminal() // external p.Quit() bypasses Update — reap the PTY
		fm.PersistSession()
		stopControl()
		stopBounded(b) // the serve child dies with us — never leaked on fatal
		fmt.Fprintf(os.Stderr, "[theboringfloor] fatal: %v\n", err)
		os.Exit(1)
	}
	// Clean-exit order is BINDING: terminal reaped → session persisted →
	// serve child killed (BEFORE any exec below — an exec'd process can
	// never run a deferred Stop). Every step is bounded: CloseTerminal is
	// a PTY kill, PersistSession a capped local write, and stopBounded caps
	// b.Stop() — a wedged serve/network must NEVER hold the process here
	// (the alt screen long restored, the member staring at a dead prompt).
	fm.CloseTerminal()
	fm.PersistSession()
	stopControl()
	stopBounded(b)

	// /session picker accept = quit + exec-replace (the app recorded the
	// intent via ExecRequest): relaunch the same binary pinned to the
	// accepted session — the swap rides the boot's resolvePrimary instead
	// of an in-app re-anchor.
	if id := fm.ExecRequest(); id != "" {
		binary, err := os.Executable()
		if err != nil {
			binary, err = exec.LookPath(os.Args[0]) // os.Executable miss → PATH twin
		}
		if err == nil {
			argv := []string{"theboringfloor", "-s", id}
			// Carry ONLY the attach target (+ the RESOLVED theme) forward;
			// --autokill and --demo are NEVER carried (the picker is
			// live-only). An explicit --backend flag rides too (a
			// /backend swap already persisted itself into brain.json, so
			// only the flag-passed case needs the carry).
			if *server != "" {
				argv = append(argv, "--server", *server)
			}
			if *backendName != "" && *backendName != cfg.Backend.ResolvedName() {
				argv = append(argv, "--backend", *backendName)
			}
			if name := chrome.CurrentTheme().Name; name != "" {
				argv = append(argv, "--theme", name)
			}
			err = syscall.Exec(binary, argv, os.Environ())
		}
		// syscall.Exec only returns on failure (a failed lookup lands here
		// too): leave the member the exact resume command, exit 0 normally.
		if err != nil {
			fmt.Printf("session %s — resume: theboringfloor -s %s\n", id, id)
		}
		return
	}

	// Normal clean exit (every quit way): hand the member the exact resume
	// line for THIS office's primary. Empty (demo, unresolved, attach-mode
	// with none) prints NOTHING — an id is never invented.
	if id := fm.PrimarySessionID(); id != "" {
		fmt.Printf("session %s — resume: theboringfloor -s %s\n", id, id)
	}
}

// stopDeadline caps the WHOLE backend Stop on the process-exit path.
// Stop bounds its own steps too (internal/backend's stopDrainTimeout +
// stopKillGrace ≈ 2.5s); this is the outer belt — a pathological Stop
// (wedged transport, stuck lock, a future regression inside Stop) must
// never hold the process between the restored screen and exit/exec.
// A var, not a const: the deadline test shrinks it.
var stopDeadline = 3 * time.Second

// stopBounded runs b.Stop() under stopDeadline and always returns: the
// beat lands either on Stop finishing or the deadline firing (one stderr
// note then — honest never silent). The runaway goroutine dies with the
// process (os.Exit / syscall.Exec / main return) — never a hang, so the
// quit/exec-replace path exits prompt-free even against a wedged serve.
func stopBounded(b state.Backend) {
	done := make(chan struct{})
	go func() { _ = b.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(stopDeadline):
		fmt.Fprintf(os.Stderr, "[theboringfloor] backend stop exceeded %s — exiting anyway\n", stopDeadline)
	}
}

func mustGetwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}
