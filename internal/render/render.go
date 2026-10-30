package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
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

func Render(markdown, style string, width int) (string, error) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return "", fmt.Errorf("create renderer: %w", err)
	}
	return renderer.Render(markdown)
}
