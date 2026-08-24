package notify

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// captRun returns a Bus wired to record every (bin, args…) invocation
// instead of exec'ing — the argv-capture seam the whole suite rides.
func captBus(mode string) (*Bus, *[][]string) {
	got := [][]string{}
	b := NewBus(mode)
	b.run = func(bin string, args ...string) {
		got = append(got, append([]string{bin}, args...))
	}
	return b, &got
}

func TestBusModes(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"on", "on"},
		{"off", "off"},
		{"", "on"},
		{"garbage", "off"}, // a brain.json typo must never spam the desk
	} {
		b := NewBus(tc.in)
		if b.Mode() != tc.want {
			t.Errorf("NewBus(%q).Mode() = %q, want %q", tc.in, b.Mode(), tc.want)
		}
	}
}

func TestBusNoNotifyEnvOverridesConfig(t *testing.T) {
	t.Setenv("THEBORINGOFFICE_NO_NOTIFY", "1")
	if b := NewBus("on"); b.Mode() != "off" {
		t.Fatalf("THEBORINGOFFICE_NO_NOTIFY=1 must force off, got %q", b.Mode())
	}

	b, got := captBus("on")
	b.notifier = "/usr/bin/osascript"
	b.Send("done", "t", "b")
	if len(*got) != 0 {
		t.Fatalf("env-forced-off must never exec, got %v", *got)
	}
}

func TestBusLegacyNoNotifyEnv(t *testing.T) {
	t.Setenv("GRAFEIO_NO_NOTIFY", "1") // pre-rename name keeps working
	if b := NewBus("on"); b.Mode() != "off" {
		t.Fatalf("GRAFEIO_NO_NOTIFY=1 must force off, got %q", b.Mode())
	}
}

func TestResolveNotifier(t *testing.T) {
	p := ResolveNotifier()
	switch runtime.GOOS {
	case "darwin":
		if p == "" || filepath.Base(p) != "osascript" {
			t.Fatalf("darwin must resolve osascript, got %q", p)
		}
	case "linux":
		if p != "" && filepath.Base(p) != "notify-send" {
			t.Fatalf("linux must resolve notify-send or \"\", got %q", p)
		}
	default:
		if p != "" {
			t.Fatalf("unsupported platform must resolve \"\", got %q", p)
		}
	}
}

func TestSendDarwinArgvAndQuoting(t *testing.T) {
	b, got := captBus("on")
	b.notifier = "/usr/bin/osascript"
	b.Send("done", "theboringoffice", `the boss is done — he said "hi" \o/`)
	if len(*got) != 1 {
		t.Fatalf("expected exactly 1 exec, got %d", len(*got))
	}
	argv := (*got)[0]
	wantScript := `display notification "the boss is done — he said \"hi\" \\o/" with title "theboringoffice" sound name "Glass"`
	if argv[0] != "/usr/bin/osascript" || argv[1] != "-e" || argv[2] != wantScript {
		t.Fatalf("darwin argv mismatch:\ngot  %q\nwant [-e + script] %q", argv, wantScript)
	}
}

func TestSendLinuxArgv(t *testing.T) {
	b, got := captBus("on")
	b.notifier = "/usr/bin/notify-send"

	b.Send("done", "theboringoffice", "the boss is done — shipped")
	b.Send("permission", "theboringoffice", "permission needed — boss needs write")

	if len(*got) != 2 {
		t.Fatalf("expected 2 execs, got %d (%v)", len(*got), *got)
	}
	done := (*got)[0]
	if strings.Join(done, " ") != "/usr/bin/notify-send theboringoffice the boss is done — shipped" {
		t.Fatalf("done argv mismatch: %q", done)
	}
	perm := (*got)[1]
	// "permission" kind rides -u critical (blocked floors stay on glass)
	if strings.Join(perm, " ") != "/usr/bin/notify-send -u critical theboringoffice permission needed — boss needs write" {
		t.Fatalf("permission argv mismatch: %q", perm)
	}
}

func TestSendRespectsModeOff(t *testing.T) {
	b, got := captBus("off")
	b.notifier = "/usr/bin/osascript"
	b.Send("done", "t", "b")
	if len(*got) != 0 {
		t.Fatalf("off mode must never exec, got %v", *got)
	}
}

func TestSendWithoutNotifierDegradesSilent(t *testing.T) {
	b, got := captBus("on")
	b.notifier = "" // headless box: graceful no-op
	b.Send("done", "t", "b")
	if len(*got) != 0 {
		t.Fatalf("no notifier resolved must no-op, got %v", *got)
	}
}

func TestSendMinGapPerKind(t *testing.T) {
	b, got := captBus("on")
	b.notifier = "/usr/bin/osascript"
	var now time.Time
	b.now = func() time.Time { return now }

	b.Send("done", "t", "one")
	now = now.Add(minGap - time.Millisecond)
	b.Send("done", "t", "two") // inside the burst gap → suppressed
	if len(*got) != 1 {
		t.Fatalf("re-fire inside %s must throttle, got %d execs", minGap, len(*got))
	}
	now = now.Add(minGap + time.Millisecond)
	b.Send("done", "t", "three") // past the gap → fires
	if len(*got) != 2 {
		t.Fatalf("re-fire past %s must land, got %d execs", minGap, len(*got))
	}
	// a different kind rides its own gap clock
	b.Send("permission", "t", "four")
	if len(*got) != 3 {
		t.Fatalf("different kind must not be throttled by done's clock, got %d execs", len(*got))
	}
}

func TestSendRollingCapPerKind(t *testing.T) {
	b, got := captBus("on")
	b.notifier = "/usr/bin/osascript"
	var now time.Time
	b.now = func() time.Time { return now }

	// fire capMax sends 500ms apart (past the min-gap, inside ONE window)
	for i := 0; i < capMax; i++ {
		now = now.Add(minGap + 100*time.Millisecond)
		b.Send("done", "t", "burst")
	}
	if len(*got) != capMax {
		t.Fatalf("first %d within a window must all land, got %d", capMax, len(*got))
	}
	now = now.Add(minGap + 100*time.Millisecond)
	b.Send("done", "t", "over-cap") // 9th in the same 30s window → capped
	if len(*got) != capMax {
		t.Fatalf("kind past %d per %s must cap, got %d execs", capMax, capWindow, len(*got))
	}
	// roll the window: fresh budget
	now = now.Add(capWindow + time.Second)
	b.Send("done", "t", "next-window")
	if len(*got) != capMax+1 {
		t.Fatalf("a rolled window must re-open the budget, got %d execs", len(*got))
	}
}

func TestSetModeLive(t *testing.T) {
	b, got := captBus("on")
	b.notifier = "/usr/bin/osascript"

	b.SetMode("off")
	b.Send("done", "t", "b")
	if len(*got) != 0 {
		t.Fatalf("SetMode(off) must silence, got %v", *got)
	}
	b.SetMode("bogus") // refused: a busted toggle never mutes/unmutes
	if b.Mode() != "off" {
		t.Fatalf("invalid SetMode must be refused, got %q", b.Mode())
	}
	b.SetMode("on")
	b.Send("done", "t", "b")
	if len(*got) != 1 {
		t.Fatalf("SetMode(on) must re-arm, got %v", *got)
	}
}
