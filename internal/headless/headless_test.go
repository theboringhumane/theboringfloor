// headless_test.go — the chrome-free contract: discovery table (fake
// stat/PATH seams), policy refusal BEFORE any launch (the reason comes
// back verbatim), contract shape (the exact public API two other
// packages compile against), and the pure helpers (classify, capRunes,
// normalizeLinks). Nothing here spawns a browser.
package headless

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/theboringhumane/theboringoffice/internal/browsertools"
)

// Compile-time signature pins — the contract, verbatim.
var (
	_ func() (string, bool)                                    = Available
	_ func(context.Context, string, int, int) (*Result, error) = Screenshot
	_ func(context.Context, string, int) (*SnapResult, error)  = Snapshot
)

// TestContractShape pins the struct fields (names, order, types).
func TestContractShape(t *testing.T) {
	check := func(name string, v any, fields ...string) {
		t.Helper()
		rt := reflect.TypeOf(v)
		if rt.NumField() != len(fields) {
			t.Fatalf("%s: %d fields, want %d (%v)", name, rt.NumField(), len(fields), fields)
		}
		for i, want := range fields {
			f := rt.Field(i)
			parts := strings.SplitN(want, ":", 2)
			if f.Name != parts[0] || f.Type.String() != parts[1] {
				t.Errorf("%s field %d = %s %s, want %s", name, i, f.Name, f.Type, want)
			}
		}
	}
	check("Result", Result{}, "URL:string", "Title:string", "PNG:[]uint8")
	check("SnapResult", SnapResult{}, "URL:string", "Title:string", "Text:string", "Links:[]headless.Link")
	check("Link", Link{}, "Text:string", "URL:string")
}

// --- discovery table -------------------------------------------------

func statSet(existing ...string) func(string) bool {
	set := map[string]bool{}
	for _, p := range existing {
		set[p] = true
	}
	return func(p string) bool { return set[p] }
}

func lookPathMap(paths map[string]string) func(string) (string, error) {
	return func(name string) (string, error) {
		if p, ok := paths[name]; ok {
			return p, nil
		}
		return "", exec.ErrNotFound
	}
}

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDiscover(t *testing.T) {
	const (
		chromeApp   = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
		chromiumApp = "/Applications/Chromium.app/Contents/MacOS/Chromium"
		edgeApp     = "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
	)
	noPath := lookPathMap(nil)
	cases := []struct {
		name     string
		goos     string
		env      map[string]string
		stat     func(string) bool
		lookPath func(string) (string, error)
		wantPath string
		wantOK   bool
	}{
		{"env override wins over everything", "darwin",
			map[string]string{chromeEnv: "/custom/chrome"},
			statSet("/custom/chrome", chromeApp), nil,
			"/custom/chrome", true},
		{"env override missing on disk falls through", "darwin",
			map[string]string{chromeEnv: "/gone/chrome"},
			statSet(chromeApp), nil,
			chromeApp, true},
		{"env override blank is ignored", "darwin",
			map[string]string{chromeEnv: "   "},
			statSet(chromiumApp), nil,
			chromiumApp, true},
		{"darwin prefers chrome", "darwin", nil,
			statSet(chromeApp, chromiumApp, edgeApp), nil,
			chromeApp, true},
		{"darwin chromium when chrome absent", "darwin", nil,
			statSet(chromiumApp, edgeApp), nil,
			chromiumApp, true},
		{"darwin edge last resort", "darwin", nil,
			statSet(edgeApp), nil,
			edgeApp, true},
		{"darwin none found", "darwin", nil,
			statSet(), nil,
			"", false},
		{"darwin never consults PATH", "darwin", nil,
			statSet(), lookPathMap(map[string]string{"google-chrome": "/usr/bin/google-chrome"}),
			"", false},
		{"linux google-chrome first", "linux", nil,
			nil, lookPathMap(map[string]string{"google-chrome": "/usr/bin/google-chrome", "chromium": "/usr/bin/chromium"}),
			"/usr/bin/google-chrome", true},
		{"linux chromium second", "linux", nil,
			nil, lookPathMap(map[string]string{"chromium": "/usr/bin/chromium"}),
			"/usr/bin/chromium", true},
		{"linux chromium-browser third", "linux", nil,
			nil, lookPathMap(map[string]string{"chromium-browser": "/usr/bin/chromium-browser"}),
			"/usr/bin/chromium-browser", true},
		{"linux edge last", "linux", nil,
			nil, lookPathMap(map[string]string{"microsoft-edge": "/usr/bin/microsoft-edge"}),
			"/usr/bin/microsoft-edge", true},
		{"linux none found", "linux", nil,
			nil, noPath,
			"", false},
		{"linux env override wins", "linux",
			map[string]string{chromeEnv: "/opt/chrome/chrome"},
			statSet("/opt/chrome/chrome"),
			lookPathMap(map[string]string{"google-chrome": "/usr/bin/google-chrome"}),
			"/opt/chrome/chrome", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotPath, gotOK := discover(c.goos, envMap(c.env), c.stat, c.lookPath)
			if gotPath != c.wantPath || gotOK != c.wantOK {
				t.Errorf("discover(%q) = (%q, %v), want (%q, %v)", c.name, gotPath, gotOK, c.wantPath, c.wantOK)
			}
		})
	}
}

// TestAvailableMemoized — the probe runs once; both calls agree.
func TestAvailableMemoized(t *testing.T) {
	p1, ok1 := Available()
	p2, ok2 := Available()
	if p1 != p2 || ok1 != ok2 {
		t.Fatalf("Available not memoized: (%q,%v) then (%q,%v)", p1, ok1, p2, ok2)
	}
	if ok1 && p1 == "" {
		t.Fatal("Available ok with empty path")
	}
}

// --- policy refusal precedes launch ----------------------------------

// TestPolicyRefusalVerbatim — refused URLs return the Decide reason as
// the error verbatim, as a *PolicyError, and (this machine HAS Chrome)
// the refusal provably precedes any browser launch: a post-launch
// failure could never carry the policy text.
func TestPolicyRefusalVerbatim(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "")
	cases := []struct {
		name string
		call func() error
	}{
		{"screenshot plain http non-localhost", func() error {
			_, err := Screenshot(context.Background(), "http://example.com/", 100, 100)
			return err
		}},
		{"snapshot plain http non-localhost", func() error {
			_, err := Snapshot(context.Background(), "http://example.com/", 100)
			return err
		}},
		{"screenshot file scheme", func() error {
			_, err := Screenshot(context.Background(), "file:///etc/passwd", 100, 100)
			return err
		}},
		{"snapshot empty url", func() error {
			_, err := Snapshot(context.Background(), "   ", 100)
			return err
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.call()
			if err == nil {
				t.Fatal("expected refusal, got nil error")
			}
			var pe *PolicyError
			if !errors.As(err, &pe) {
				t.Fatalf("error is %T, want *PolicyError (%v)", err, err)
			}
			// The reason verbatim — recomputed through the REAL policy.
			raw := map[string]string{
				"screenshot plain http non-localhost": "http://example.com/",
				"snapshot plain http non-localhost":   "http://example.com/",
				"screenshot file scheme":              "file:///etc/passwd",
				"snapshot empty url":                  "   ",
			}[c.name]
			want := browsertools.Decide(raw, os.Getenv).Reason
			if err.Error() != want {
				t.Errorf("error %q, want verbatim reason %q", err.Error(), want)
			}
		})
	}
}

// TestPolicyHTTPAllowedWithFlag — the same plain-http URL passes the
// gate once the member exports the flag (it then fails LATER, at
// discovery/launch at the earliest — never with a PolicyError).
func TestPolicyHTTPAllowedWithFlag(t *testing.T) {
	t.Setenv(browsertools.AllowHTTPEnv, "1")
	_, err := Screenshot(context.Background(), "http://127.0.0.1:1/", 50, 50)
	var pe *PolicyError
	if errors.As(err, &pe) {
		t.Fatalf("policy must not refuse with %s=1: %v", browsertools.AllowHTTPEnv, pe)
	}
	// Whether chrome exists here or not, the error must NOT be a policy
	// refusal (loopback is always allowed anyway — belt and suspenders).
}

// --- pure helpers ------------------------------------------------------

func TestClassify(t *testing.T) {
	timeout := classify("https://x", context.DeadlineExceeded)
	if !errors.Is(timeout, context.DeadlineExceeded) || !strings.Contains(timeout.Error(), "timed out") {
		t.Errorf("timeout misclassified: %v", timeout)
	}
	canceled := classify("https://x", context.Canceled)
	if !errors.Is(canceled, context.Canceled) || !strings.Contains(canceled.Error(), "canceled") {
		t.Errorf("cancel misclassified: %v", canceled)
	}
	nav := classify("https://x", errors.New("page load error net::ERR_NAME_NOT_RESOLVED"))
	if !strings.Contains(nav.Error(), "navigation failed") {
		t.Errorf("nav failure misclassified: %v", nav)
	}
}

func TestCapRunes(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"  hello  ", 100, "hello"},
		{"hello world", 5, "hello"},
		{"héllo wörld", 6, "héllo "}, // rune-safe: multi-byte never splits
		{"日本語テスト", 3, "日本語"},
		{"abc", 0, ""},
		{"abc", -1, ""},
		{"", 10, ""},
	}
	for _, c := range cases {
		if got := capRunes(c.in, c.max); got != c.want {
			t.Errorf("capRunes(%q, %d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestNormalizeLinks(t *testing.T) {
	got := normalizeLinks([]Link{
		{Text: " A ", URL: "https://a.com/"},
		{Text: "dup", URL: "https://a.com/"},
		{Text: "js", URL: "javascript:void(0)"},
		{Text: "JS upper", URL: "JavaScript:alert(1)"},
		{Text: "mail", URL: "mailto:x@y.z"},
		{Text: "blank", URL: "   "},
		{Text: " B ", URL: " https://b.com/page "},
	})
	want := []Link{
		{Text: "A", URL: "https://a.com/"},
		{Text: "B", URL: "https://b.com/page"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeLinks = %+v, want %+v", got, want)
	}

	// The 50-cap: 60 distinct URLs in, exactly 50 out, page order kept.
	var many []Link
	for i := 0; i < 60; i++ {
		many = append(many, Link{Text: "t", URL: "https://x.com/" + strings.Repeat("p", i+1)})
	}
	capped := normalizeLinks(many)
	if len(capped) != maxLinks || capped[0] != (Link{Text: "t", URL: "https://x.com/p"}) {
		t.Fatalf("cap: got %d links, first %+v", len(capped), capped[0])
	}
}
