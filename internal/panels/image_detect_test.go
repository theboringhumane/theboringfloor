// image_detect_test.go — the env-based lane matrix: nine env combos over
// the four real lanes + the none fold, pure injected-env reads (zero
// shell-outs, zero network — the detection layer is a literal mapping
// table the tests pin corner by corner).
package panels

import (
	"testing"
)

// envOf builds the injected env getter for one combo row.
func envOf(pairs ...string) func(string) string {
	return func(k string) string {
		for i := 0; i+1 < len(pairs); i += 2 {
			if pairs[i] == k {
				return pairs[i+1]
			}
		}
		return ""
	}
}

func TestDetectImageSupportMatrix(t *testing.T) {
	cases := []struct {
		name string
		env  func(string) string
		want ImageLane
	}{
		{"kitty by TERM_PROGRAM", envOf("TERM_PROGRAM", "kitty", "TERM", "xterm-kitty"), KittyLane},
		{"kitty by KITTY_WINDOW_ID (beats TERM_PROGRAM)", envOf("KITTY_WINDOW_ID", "3", "TERM_PROGRAM", "iTerm.app"), KittyLane},
		{"tmux folds conservative ASCII even inside kitty", envOf("TMUX", "/tmp/tmux-1000/default,1,0", "KITTY_WINDOW_ID", "3", "TERM_PROGRAM", "ghostty", "TERM", "xterm-256color"), ASCIILane},
		{"wezterm rides the iterm lane", envOf("WEZTERM_UNIX_SOCKET", "/tmp/wez", "TERM", "xterm-256color"), ITermLane},
		{"vscode PID rides the iterm lane", envOf("VSCODE_PID", "4242", "TERM", "xterm-256color"), ITermLane},
		{"sixel by TERM name", envOf("TERM", "xterm-sixel"), SixelLane},
		{"dumb TERM is the none fold (chips only)", envOf("TERM", "dumb"), NoneLane},
		{"unset TERM is the none fold", envOf(), NoneLane},
		{"plain xterm-256color is ASCII", envOf("TERM", "xterm-256color", "TERM_PROGRAM", "Apple_Terminal"), ASCIILane},
	}
	for _, tc := range cases {
		if got := DetectImageSupportFrom(tc.env); got != tc.want {
			t.Errorf("%s: want %s, got %s", tc.name, tc.want, got)
		}
	}
}

// TestImageLaneString — the notice-word contract (:/images report prints it).
func TestImageLaneString(t *testing.T) {
	for l, want := range map[ImageLane]string{
		ASCIILane: "ascii", KittyLane: "kitty", ITermLane: "iterm", SixelLane: "sixel", NoneLane: "none",
	} {
		if l.String() != want {
			t.Errorf("lane %d: want %q, got %q", l, want, l.String())
		}
	}
}
