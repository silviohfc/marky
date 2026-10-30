package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

const (
	MaxReadingWidth = 100
	MinWidth        = 20
	fallbackWidth   = 80
)

func ContentWidth(override, termWidth int, isTTY bool) int {
	if override > 0 {
		return override
	}
	if !isTTY {
		return fallbackWidth
	}
	return max(MinWidth, min(termWidth, MaxReadingWidth))
}

func Center(text string, termWidth, contentWidth int) string {
	margin := (termWidth - contentWidth) / 2
	if margin <= 0 {
		return text
	}
	padding := strings.Repeat(" ", margin)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = padding + line
	}
	return strings.Join(lines, "\n")
}

func ResolveStyle(style string, darkBackground bool) string {
	if style != "auto" {
		return style
	}
	if darkBackground {
		return "dark"
	}
	return "light"
}

// PlainStyle renders without colors or escape codes, for non-TTY output.
const PlainStyle = "plain"

func Render(markdown, style string, width int) (string, error) {
	return render(markdown, style, width, layout{})
}

func render(markdown, style string, width int, l layout) (string, error) {
	p, ok := palettes[style]
	if !ok {
		return "", fmt.Errorf("unknown style %q", style)
	}
	profile := termenv.TrueColor
	if style == PlainStyle {
		profile = termenv.Ascii
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(theme(p, width, l)),
		glamour.WithColorProfile(profile),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return "", fmt.Errorf("create renderer: %w", err)
	}
	source, blocks := extractBlocks(markdown)
	out, err := renderer.Render(source)
	if err != nil {
		return "", err
	}
	if out, err = insertBlocks(out, blocks, style, width, l); err != nil {
		return "", err
	}
	if style == PlainStyle {
		return plain(out), nil
	}
	return out, nil
}

// plain strips the text attributes glamour still emits under the ASCII
// profile, plus the trailing padding on every line.
func plain(text string) string {
	lines := strings.Split(xansi.Strip(text), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}
