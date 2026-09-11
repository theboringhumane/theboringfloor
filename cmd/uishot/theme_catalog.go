package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"

	"github.com/theboringhumane/theboringfloor/internal/chrome"
)

// The website uses the same ordered palette registry as the application.
// Run with an isolated config directory to export only the built-in themes.
func writeThemeCatalog(w io.Writer) error {
	type palette struct {
		ID         string   `json:"id"`
		Dark       bool     `json:"dark"`
		Background string   `json:"background"`
		Foreground string   `json:"foreground"`
		Colors     []string `json:"colors"`
	}
	hex := func(c color.Color) string {
		r, g, b, _ := c.RGBA()
		return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
	}
	previous := chrome.CurrentTheme().Name
	defer chrome.SetTheme(previous)
	var palettes []palette
	for _, name := range chrome.ThemeNames() {
		chrome.SetTheme(name)
		t := chrome.CurrentTheme()
		palettes = append(palettes, palette{
			ID: name, Dark: t.Dark, Background: hex(t.PanelBg), Foreground: hex(t.White),
			Colors: []string{hex(t.Accent), hex(t.Info), hex(t.OK), hex(t.Magenta), hex(t.Err)},
		})
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(palettes)
}
