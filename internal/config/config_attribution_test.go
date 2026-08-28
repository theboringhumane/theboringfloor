package config

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// --- top-level "attribution" knob (majdoor commit-attribution) ----------

// TestAttributionDefaultShape pins Default()'s posture AND that the key
// actually lands in the marshaled skeleton — the file a first boot writes
// and --print-default-config prints.
func TestAttributionDefaultShape(t *testing.T) {
	cfg := Default()
	if cfg.Attribution != AttributionDefault {
		t.Errorf("Default().Attribution = %q, want %q", cfg.Attribution, AttributionDefault)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"attribution": "on"`) {
		t.Errorf("default skeleton lacks the top-level attribution key:\n%s", b)
	}
	var back Config
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Attribution != AttributionDefault {
		t.Errorf("round-trip Attribution = %q, want %q", back.Attribution, AttributionDefault)
	}
}

// TestAttributionDefaultWrittenOnFirstBoot: the brain.json Load() creates
// on a missing file carries the key verbatim.
func TestAttributionDefaultWrittenOnFirstBoot(t *testing.T) {
	useHome(t)
	if _, err := Load(); err != nil {
		t.Fatalf("Load() on missing file: %v", err)
	}
	b, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("read created brain.json: %v", err)
	}
	if !strings.Contains(string(b), `"attribution": "on"`) {
		t.Errorf("first-boot brain.json lacks the attribution key:\n%s", b)
	}
}

// TestValidAttribution pins the two-value whitelist; anything else
// (including a case variant) is not a real posture.
func TestValidAttribution(t *testing.T) {
	for _, tc := range []struct {
		v     string
		valid bool
	}{
		{"on", true},
		{"off", true},
		{"", false},
		{"ON", false},
		{"yes", false},
		{"1", false},
	} {
		if got := ValidAttribution(tc.v); got != tc.valid {
			t.Errorf("ValidAttribution(%q) = %v, want %v", tc.v, got, tc.valid)
		}
	}
}

// TestLoad_AttributionBackfill covers a brain.json written BEFORE the
// knob existed: no "attribution" key must decode and backfill to "on" —
// the file keeps meaning exactly what it always meant. (Neighbor fields
// must survive untouched.)
func TestLoad_AttributionBackfill(t *testing.T) {
	home := useHome(t)
	writeBrain(t, home, `{
  "version": 1,
  "boss": {"name": "chief"},
  "ui": {"theme": "noir"}
}`)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.Attribution != AttributionDefault {
		t.Errorf("attribution backfill = %q, want %q", cfg.Attribution, AttributionDefault)
	}
	if cfg.Boss.Name != "chief" || cfg.UI.Theme != "noir" {
		t.Errorf("neighbor fields must survive the backfill untouched, got boss=%+v ui=%+v", cfg.Boss, cfg.UI)
	}
}

// TestLoad_AttributionPreserved: an explicit "off" (the opt-out) and an
// explicit "on" both come back verbatim — Load never normalizes a valid
// set value.
func TestLoad_AttributionPreserved(t *testing.T) {
	for _, want := range []string{"off", "on"} {
		home := useHome(t)
		writeBrain(t, home, `{"version": 1, "attribution": "`+want+`"}`)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load(%q): %v", want, err)
		}
		if cfg.Attribution != want {
			t.Errorf("Attribution = %q, want the explicit %q preserved", cfg.Attribution, want)
		}
	}
}

// TestLoad_AttributionBogusTreatedAsOn pins the chosen tolerance policy:
// a bogus value is NOT fatal and does NOT silently disable a default-on
// feature — Load normalizes it to the default "on" (ValidAttribution is
// the gate). An explicit empty string resolves the same way.
func TestLoad_AttributionBogusTreatedAsOn(t *testing.T) {
	for _, bogus := range []string{"maybe", ""} {
		home := useHome(t)
		writeBrain(t, home, `{"version": 1, "attribution": "`+bogus+`"}`)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load(bogus %q): %v", bogus, err)
		}
		if cfg.Attribution != AttributionDefault {
			t.Errorf("Attribution for bogus %q = %q, want normalized %q", bogus, cfg.Attribution, AttributionDefault)
		}
	}
}

// TestSave_AttributionRoundTrip: the knob survives a Save→Load cycle in
// both postures, and a saved file carries the key explicitly.
func TestSave_AttributionRoundTrip(t *testing.T) {
	for _, want := range []string{"on", "off"} {
		useHome(t)
		cfg := Default()
		cfg.Attribution = want
		if err := Save(cfg); err != nil {
			t.Fatalf("Save(%q): %v", want, err)
		}
		b, err := os.ReadFile(Path())
		if err != nil {
			t.Fatalf("read saved file: %v", err)
		}
		if !strings.Contains(string(b), `"attribution": "`+want+`"`) {
			t.Errorf("saved file lacks attribution %q:\n%s", want, b)
		}
		got, err := Load()
		if err != nil {
			t.Fatalf("Load() after Save(%q): %v", want, err)
		}
		if got.Attribution != want {
			t.Errorf("round trip Attribution = %q, want %q", got.Attribution, want)
		}
	}
}
