package render

import (
	"strings"

	"github.com/charmbracelet/glamour/ansi"
)

type palette struct {
	text, muted, subtle, surface, base string
	primary, secondary, green, peach   string
	codeTheme                          string
}

// Catppuccin Mocha / Latte inspired palettes.
var palettes = map[string]palette{
	"dark": {
		text: "#CDD6F4", muted: "#7F849C", subtle: "#45475A", surface: "#313244", base: "#1E1E2E",
		primary: "#89B4FA", secondary: "#CBA6F7", green: "#A6E3A1", peach: "#FAB387",
		codeTheme: "catppuccin-mocha",
	},
	"light": {
		text: "#4C4F69", muted: "#8C8FA1", subtle: "#BCC0CC", surface: "#E6E9EF", base: "#EFF1F5",
		primary: "#1E66F5", secondary: "#8839EF", green: "#40A02B", peach: "#FE640B",
		codeTheme: "catppuccin-latte",
	},
	// plain keeps the same layout with no colors, for non-TTY output.
	"plain": {},
}

const documentMargin = 2

// Theme builds a glamour style that shows no Markdown syntax markers
// (no #, *, ` or > tokens) and draws rules that span the given width.
func Theme(name string, width int) (ansi.StyleConfig, bool) {
	p, ok := palettes[name]
	if !ok {
		return ansi.StyleConfig{}, false
	}
	ruleWidth := max(1, width-2*documentMargin)
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{BlockPrefix: "\n", BlockSuffix: "\n", Color: color(p.text)},
			Margin:         uintPtr(documentMargin),
		},
		Paragraph: ansi.StyleBlock{},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: color(p.muted), Italic: boolPtr(true)},
			Indent:         uintPtr(1),
			IndentToken:    strPtr("┃ "),
		},
		List: ansi.StyleList{LevelIndent: 2},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{BlockSuffix: "\n", Bold: boolPtr(true)},
		},
		H1: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Prefix: " ", Suffix: " ", Color: color(p.base), BackgroundColor: color(p.secondary), Bold: boolPtr(true),
		}},
		H2:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "▌ ", Color: color(p.primary)}},
		H3:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: color(p.secondary)}},
		H4:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: color(p.green)}},
		H5:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: color(p.peach)}},
		H6:            ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: color(p.muted), Bold: boolPtr(false), Italic: boolPtr(true)}},
		Text:          ansi.StylePrimitive{},
		Strikethrough: ansi.StylePrimitive{CrossedOut: boolPtr(true), Color: color(p.muted)},
		Emph:          ansi.StylePrimitive{Italic: boolPtr(true)},
		Strong:        ansi.StylePrimitive{Bold: boolPtr(true)},
		HorizontalRule: ansi.StylePrimitive{
			Color:  color(p.subtle),
			Format: "\n" + strings.Repeat("─", ruleWidth) + "\n",
		},
		Item:        ansi.StylePrimitive{BlockPrefix: "• "},
		Enumeration: ansi.StylePrimitive{BlockPrefix: ". "},
		Task:        ansi.StyleTask{Ticked: "✓ ", Unticked: "○ "},
		Link:        ansi.StylePrimitive{Color: color(p.muted), Underline: boolPtr(true)},
		LinkText:    ansi.StylePrimitive{Color: color(p.primary), Bold: boolPtr(true)},
		Image:       ansi.StylePrimitive{Color: color(p.muted), Underline: boolPtr(true)},
		ImageText:   ansi.StylePrimitive{Color: color(p.secondary), Format: "▣ {{.text}}"},
		Code: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
			Prefix: " ", Suffix: " ", Color: color(p.peach), BackgroundColor: color(p.surface),
		}},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: color(p.text)},
				Margin:         uintPtr(documentMargin),
			},
			Theme: p.codeTheme,
		},
		Table: ansi.StyleTable{
			StyleBlock:      ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Color: color(p.text)}},
			CenterSeparator: strPtr("┼"),
			ColumnSeparator: strPtr("│"),
			RowSeparator:    strPtr("─"),
		},
		DefinitionDescription: ansi.StylePrimitive{BlockPrefix: "\n→ "},
	}, true
}

func color(hex string) *string {
	if hex == "" {
		return nil
	}
	return &hex
}

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }
func uintPtr(u uint) *uint    { return &u }
