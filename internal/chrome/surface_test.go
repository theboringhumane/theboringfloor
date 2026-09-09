package chrome

import (
	"charm.land/lipgloss/v2"
	"image/color"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// Use the actual screen draw path so untouched cells retain nil colors.
func surfaceCells(frame string) uv.Lines {
	screen := uv.NewScreenBuffer(lipgloss.Width(frame), lipgloss.Height(frame))
	uv.NewStyledString(frame).Draw(&screen, screen.Bounds())
	return screen.Buffer.Lines
}

func TestSolidFrameResolvesDefaultsAndPreservesStyles(t *testing.T) {
	defer restoreTheme()
	// Include reset-looking color components, both extended-color syntaxes,
	// links, wide glyphs, and attributes that must survive a color reset.
	inputs := []string{
		"plain  text",
		"\x1b[1mstrong\x1b[m normal",
		"\x1b[31;44mcolor\x1b[0m default",
		"\x1b[31;44mcolor\x1b[49m background\x1b[39m foreground",
		"\x1b[0;1;49;39mcombined",
		"\x1b[48;2;0;49;39mRGB\x1b[38;2;49;0;39m ink",
		"\x1b[48:2::0:49:39mcolon\x1b[38:2::49:0:39m ink",
		"\x1b[48;5;49mindexed\x1b[38;5;0m ink",
		"\x1b[7;4mreverse underline\x1b[49m default\x1b[27m normal",
		"\x1b]8;;https://example.com\x1b\\linked 界 👩‍💻\x1b]8;;\x1b\\",
		"first\n\x1b[1msecond\x1b[m\nlast",
	}
	for _, theme := range themeList {
		SetTheme(theme.Name)
		for _, input := range inputs {
			original := surfaceCells(input)
			painted := SolidFrame(input, 0, 0)
			if ansi.Strip(painted) != ansi.Strip(input) {
				t.Fatalf("%s: painting changed text: %q", theme.Name, painted)
			}
			actual := surfaceCells(painted)
			for y, row := range original {
				for x, cell := range row {
					if cell.Width == 0 || (cell.Content == " " && cell.Style.IsZero()) { // unstyled blanks are checked separately
						continue
					}
					want := cell.Style
					if want.Fg == nil {
						want.Fg = theme.White
					}
					if want.Bg == nil {
						want.Bg = theme.PanelBg
					}
					got := actual[y][x]
					if !got.Style.Equal(&want) || got.Content != cell.Content || got.Link != cell.Link {
						t.Fatalf("%s at %d,%d in %q: got %+v; want style %+v and content/link %+v", theme.Name, x, y, input, got, want, cell)
					}
				}
			}
		}
	}
}

func TestSolidFramePaintsBlankCells(t *testing.T) {
	defer restoreTheme()
	SetTheme("paper")
	for _, input := range []string{"", "x\n\ny", "\x1b[31mx\x1b[m"} {
		lines := surfaceCells(SolidFrame(input, 8, 5))
		if len(lines) != 5 {
			t.Fatalf("got %d rows, want 5", len(lines))
		}
		for y, row := range lines {
			if len(row) != 8 {
				t.Fatalf("row %d: got %d cells, want 8", y, len(row))
			}
			for x, cell := range row {
				if cell.Style.Bg == nil || cell.Style.Fg == nil {
					t.Fatalf("cell %d,%d still uses terminal defaults", x, y)
				}
			}
		}
	}
}

func TestThemeSurfacesUseFixedRGB(t *testing.T) {
	for _, theme := range themeList {
		for name, c := range map[string]color.Color{
			"canvas": theme.PanelBg, "bar": theme.BarBg,
			"selection": theme.Accent, "ink": theme.White,
		} {
			switch c.(type) {
			case color.RGBA, color.NRGBA:
			default:
				t.Errorf("%s %s must use fixed RGB, got %T", theme.Name, name, c)
			}
		}
	}
}

func BenchmarkSolidFrame(b *testing.B) {
	row := strings.Repeat("\x1b[1mtext\x1b[m ", 30)
	frame := strings.Repeat(row+"\n", 41) + row
	b.ReportAllocs()
	for b.Loop() {
		SolidFrame(frame, 150, 42)
	}
}
