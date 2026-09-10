package chrome

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chstyles "github.com/alecthomas/chroma/v2/styles"
)

func defaultTokenSettings(t Theme) []tokenSetting {
	var settings []tokenSetting
	for _, pair := range [][2]string{{"comment", hexColor(t.Dim)}, {"keyword", hexColor(t.Magenta)}, {"string", hexColor(t.OK)}, {"constant.numeric", hexColor(t.Warn)}, {"entity.name.function", hexColor(t.Blue)}, {"entity.name.type", hexColor(t.Accent)}} {
		var setting tokenSetting
		setting.Scope, _ = json.Marshal(pair[0])
		setting.Settings.Foreground = pair[1]
		settings = append(settings, setting)
	}
	return settings
}

// TextMate's common scopes map onto the existing diff syntax renderer.
// Language-specific selectors and semantic-token rules retain fallbacks.
func applyTokenSettings(t *Theme, settings []tokenSetting) error {
	entries := chroma.StyleEntries{chroma.Text: hexColor(t.White), chroma.Comment: hexColor(t.Dim), chroma.Keyword: hexColor(t.Magenta), chroma.LiteralString: hexColor(t.OK), chroma.LiteralNumber: hexColor(t.Warn), chroma.NameFunction: hexColor(t.Blue), chroma.NameClass: hexColor(t.Accent)}
	for _, setting := range settings {
		style := ""
		if setting.Settings.Foreground != "" {
			c, err := parseThemeColor(setting.Settings.Foreground, t.PanelBg)
			if err != nil {
				return fmt.Errorf("token color: %w", err)
			}
			style = hexColor(c)
		}
		for _, attr := range strings.Fields(setting.Settings.FontStyle) {
			if attr == "bold" || attr == "italic" || attr == "underline" {
				style += " " + attr
			}
		}
		var scopes []string
		var scope string
		if len(setting.Scope) == 0 {
			scopes = []string{""}
		} else if json.Unmarshal(setting.Scope, &scope) == nil {
			scopes = strings.Split(scope, ",")
		} else if err := json.Unmarshal(setting.Scope, &scopes); err != nil {
			return fmt.Errorf("token scope must be a string or a string array")
		}
		for _, scope := range scopes {
			if token, ok := scopeToken(strings.TrimSpace(scope)); ok && strings.TrimSpace(style) != "" {
				entries[token] = style
			}
		}
	}
	name := "thefloor-" + t.Name + "-" + t.Revision
	style, err := chroma.NewStyle(name, entries)
	if err != nil {
		return err
	}
	chstyles.Register(style)
	t.ChromaStyle = name
	return nil
}

func scopeToken(scope string) (chroma.TokenType, bool) {
	if scope == "" {
		return chroma.Text, true
	}
	for _, mapping := range []struct {
		scope string
		token chroma.TokenType
	}{
		{"comment", chroma.Comment}, {"string", chroma.LiteralString},
		{"constant.numeric", chroma.LiteralNumber}, {"constant", chroma.NameConstant},
		{"keyword.operator", chroma.Operator}, {"keyword", chroma.Keyword}, {"storage", chroma.Keyword},
		{"entity.name.function", chroma.NameFunction}, {"support.function", chroma.NameFunction},
		{"entity.name.type", chroma.NameClass}, {"entity.name.class", chroma.NameClass}, {"support.type", chroma.NameClass},
		{"entity.name.tag", chroma.NameTag}, {"entity.other.attribute-name", chroma.NameAttribute},
		{"variable", chroma.NameVariable}, {"punctuation", chroma.Punctuation},
	} {
		if scope == mapping.scope || strings.HasPrefix(scope, mapping.scope+".") {
			return mapping.token, true
		}
	}
	return 0, false
}
