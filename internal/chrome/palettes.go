package chrome

import (
	"image/color"
	"math"

	"charm.land/lipgloss/v2"
	"github.com/theboringhumane/theboringfloor/internal/office"
)

// These palettes use the same semantic slots as imported VS Code themes.
// Layout and interaction never depend on a palette's name.
func additionalThemes() []Theme {
	type palette struct {
		name, bg, surface, fg, dim, accent, red, green, cyan, purple, blue, amber string
		dark                                                                      bool
	}
	palettes := []palette{
		{"tokyo-night", "#1a1b26", "#24283b", "#c0caf5", "#828bb8", "#7aa2f7", "#f7768e", "#9ece6a", "#7dcfff", "#bb9af7", "#7aa2f7", "#e0af68", true},
		{"catppuccin-mocha", "#1e1e2e", "#313244", "#cdd6f4", "#9399b2", "#cba6f7", "#f38ba8", "#a6e3a1", "#94e2d5", "#f5c2e7", "#89b4fa", "#fab387", true},
		{"catppuccin-latte", "#eff1f5", "#dce0e8", "#4c4f69", "#6c6f85", "#8839ef", "#d20f39", "#40a02b", "#179299", "#ea76cb", "#1e66f5", "#df8e1d", false},
		{"nord", "#2e3440", "#3b4252", "#eceff4", "#a3b1c6", "#88c0d0", "#bf616a", "#a3be8c", "#8fbcbb", "#b48ead", "#81a1c1", "#ebcb8b", true},
		{"gruvbox", "#282828", "#3c3836", "#ebdbb2", "#a89984", "#fabd2f", "#fb4934", "#b8bb26", "#8ec07c", "#d3869b", "#83a598", "#fe8019", true},
		{"one-dark", "#282c34", "#21252b", "#abb2bf", "#9098a7", "#61afef", "#e06c75", "#98c379", "#56b6c2", "#c678dd", "#61afef", "#e5c07b", true},
		{"rose-pine", "#191724", "#26233a", "#e0def4", "#908caa", "#c4a7e7", "#eb6f92", "#9ccfd8", "#9ccfd8", "#c4a7e7", "#ebbcba", "#f6c177", true},
		{"github-light", "#ffffff", "#f6f8fa", "#24292f", "#57606a", "#0969da", "#cf222e", "#1a7f37", "#0598bc", "#8250df", "#0969da", "#9a6700", false},
	}
	var out []Theme
	for _, p := range palettes {
		t := basePalette(p.dark)
		t.Name, t.Dark = p.name, p.dark
		t.PanelBg, t.BarBg, t.White, t.Dim = lipgloss.Color(p.bg), lipgloss.Color(p.surface), lipgloss.Color(p.fg), lipgloss.Color(p.dim)
		t.Accent, t.Err, t.OK, t.Info = lipgloss.Color(p.accent), lipgloss.Color(p.red), lipgloss.Color(p.green), lipgloss.Color(p.cyan)
		t.Magenta, t.Blue, t.Warn, t.Question = lipgloss.Color(p.purple), lipgloss.Color(p.blue), lipgloss.Color(p.amber), lipgloss.Color(p.amber)
		t.Border = mix(t.Dim, t.PanelBg, .5)
		finishPalette(&t)
		registerFloorPalette(t)
		out = append(out, t)
	}
	return out
}

func basePalette(dark bool) Theme {
	name := "paper"
	if dark {
		name = "cockpit"
	}
	for _, t := range themeList {
		if t.Name == name {
			return t
		}
	}
	panic("missing base palette")
}

func finishPalette(t *Theme) {
	t.Black = lipgloss.Color("#07111c")
	if luminance(t.Accent) < .35 {
		t.Black = lipgloss.Color("#ffffff")
	}
	t.ToolColor = t.Info
	t.RoleBoss, t.RoleHR, t.RoleDev = t.Accent, t.Err, t.Info
	t.RoleScout, t.RoleReviewer, t.RoleRunner = t.OK, t.Magenta, t.Blue
	t.DiffAddBg, t.DiffDelBg = mix(t.OK, t.PanelBg, .15), mix(t.Err, t.PanelBg, .15)
	t.DiffAddFg, t.DiffDelFg, t.DiffCtxFg, t.DiffGutterFg = t.OK, t.Err, t.Dim, t.Dim
	t.Glamour, t.ChromaStyle = "light", "github"
	if t.Dark {
		t.Glamour, t.ChromaStyle = "dark", "github-dark"
	}
}

func registerFloorPalette(t Theme) {
	colors := map[string]string{
		"black": hexColor(t.Black), "gray": hexColor(t.Dim), "grey": hexColor(t.Dim),
		"white": hexColor(t.White), "yellow": hexColor(t.Accent), "red": hexColor(t.Err),
		"green": hexColor(t.OK), "cyan": hexColor(t.Info), "blue": hexColor(t.Blue), "magenta": hexColor(t.Magenta),
	}
	for _, name := range []string{"white", "yellow", "red", "green", "cyan", "blue", "magenta"} {
		colors[name+"Bright"] = colors[name]
	}
	office.RegisterTheme(t.Name, colors)
}

func mix(fg, bg color.Color, amount float64) color.Color {
	r, g, b, _ := fg.RGBA()
	x, y, z, _ := bg.RGBA()
	blend := func(a, b uint32) uint8 { return uint8((float64(a)*amount + float64(b)*(1-amount)) / 257) }
	return color.RGBA{R: blend(r, x), G: blend(g, y), B: blend(b, z), A: 255}
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	linear := func(v uint32) float64 {
		x := float64(v) / 65535
		if x <= .04045 {
			return x / 12.92
		}
		return math.Pow((x+.055)/1.055, 2.4)
	}
	return .2126*linear(r) + .7152*linear(g) + .0722*linear(b)
}
