package office

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

func TestCockpitGridKeepsCellsAndFurniture(t *testing.T) {
	before := ansiColors
	defer func() { ansiColors = before }()
	SetTheme("cockpit")
	for _, size := range [][2]int{{28, 27}, {40, 12}, {60, 19}, {70, 7}, {120, 40}} {
		rows := BuildRows(state.OfficeState{}, size[0], size[1])
		for _, row := range rows {
			plain := Styleless([]Row{row})
			if !utf8.ValidString(plain) || ansi.StringWidth(plain) != size[0] {
				t.Fatalf("%v: instrument glyphs broke cell geometry: %q", size, plain)
			}
		}
		if size[0] >= 40 && !strings.Contains(Styleless(rows), "[=BOSS=]") {
			t.Fatalf("%v: cockpit erased boss desk", size)
		}
	}
}
