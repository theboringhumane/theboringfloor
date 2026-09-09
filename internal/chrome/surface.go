package chrome

import (
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// SolidFrame gives the composed UI its own canvas. An outer lipgloss
// background alone cannot do this: nested styles emit SGR resets that return
// to the terminal default (which may be transparent). Resolve those defaults
// to the theme here, after all panels and overlays have been composed.
// Explicit colors, attributes, hyperlinks, and graphemes pass through intact.
// Call before caching the frame, not on every renderer cache hit.
func SolidFrame(frame string, width, height int) string {
	fg, bg := CurrentTheme().White, PanelBgColor
	foreground := ansi.Style{}.ForegroundColor(fg).String()
	background := ansi.Style{}.BackgroundColor(bg).String()
	base := foreground + background
	pen := uv.Style{Fg: fg, Bg: bg}
	p := ansi.GetParser()
	defer ansi.PutParser(p)
	var out strings.Builder
	out.Grow(len(frame) + len(frame)/4)
	out.WriteString(base)
	var state byte
	col, rows := 0, 1
	pad := func() {
		if col < width {
			out.WriteString(strings.Repeat(" ", width-col))
		}
	}
	for len(frame) > 0 {
		seq, cells, n, next := ansi.DecodeSequence(frame, state, p)
		frame, state = frame[n:], next
		if seq == "\n" {
			pad()
			col = 0
			rows++
		}
		out.WriteString(seq)
		col += cells
		if ansi.HasCsiPrefix(seq) && p.Command() == 'm' {
			// ReadStyle understands extended/colon colors: an RGB component
			// of 0 or 49 is a color value, not a reset instruction.
			uv.ReadStyle(p.Params(), &pen)
			if pen.Fg == nil {
				out.WriteString(foreground)
				pen.Fg = fg
			}
			if pen.Bg == nil {
				out.WriteString(background)
				pen.Bg = bg
			}
		}
	}
	pad()
	if rows < height {
		out.WriteString(ansi.ResetStyle + base)
		for ; rows < height; rows++ {
			out.WriteByte('\n')
			out.WriteString(strings.Repeat(" ", max(0, width)))
		}
	}
	out.WriteString(ansi.ResetStyle)
	return out.String()
}
