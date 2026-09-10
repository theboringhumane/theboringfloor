package office

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// stampCockpit keeps the office's actual seat/walker coordinates. The
// tactical grid is only painted into unoccupied cells; it cannot obscure
// agents, speech bubbles, or furniture.
func stampCockpit(g []Row, w, h int) {
	if w < 4 || h < 3 {
		return
	}
	border := style{fg: "gray"}
	// The legacy ASCII floor's put() is byte-based. Instrument glyphs are
	// single-cell Unicode runes, so write them by rune without changing the
	// established furniture/sprite renderer.
	put := func(g []Row, w, h, x, y int, text string, paint *style) {
		if y < 0 || y >= h {
			return
		}
		for _, ch := range text {
			if x >= 0 && x < w {
				g[y][x] = Cell{Ch: ch, FG: paint.fg, Bold: paint.bold}
			}
			x++
		}
	}
	for y := 1; y < h-1; y++ {
		put(g, w, h, 0, y, "│", &border)
		put(g, w, h, w-1, y, "│", &border)
		if y%3 != 0 {
			continue
		}
		for x := 3; x < w-2; x += 6 {
			if g[y][x].Ch == ' ' && g[y][x-1].Ch == ' ' && g[y][x+1].Ch == ' ' {
				put(g, w, h, x, y, "·", &border)
			}
		}
	}
	put(g, w, h, 0, 0, "┌"+strings.Repeat("─", w-2)+"┐", &border)
	put(g, w, h, 0, h-1, "└"+strings.Repeat("─", w-2)+"┘", &border)
	put(g, w, h, 2, 0, ansi.Truncate(" 01 / TACTICAL FLOOR ", w-4, ""), &style{fg: "yellow", bold: true})
	put(g, w, h, 2, h-1, ansi.Truncate(" OFFICE / AGENT POSITIONS ", w-4, ""), &style{fg: "white"})
}
