package render

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
	"testing"
)

func TestContentWidthUsesOverrideWhenPositive(t *testing.T) {
	if got := ContentWidth(60, 200, true); got != 60 {
		t.Fatalf("got %d, want 60", got)
	}
}

func TestContentWidthFallsBackTo80WithoutTTY(t *testing.T) {
	if got := ContentWidth(0, 200, false); got != 80 {
		t.Fatalf("got %d, want 80", got)
	}
}

func TestContentWidthCapsAtReadingWidth(t *testing.T) {
	if got := ContentWidth(0, 250, true); got != MaxReadingWidth {
		t.Fatalf("got %d, want %d", got, MaxReadingWidth)
	}
}

func TestContentWidthFollowsNarrowTerminal(t *testing.T) {
	if got := ContentWidth(0, 70, true); got != 70 {
		t.Fatalf("got %d, want 70", got)
	}
}

func TestContentWidthHasMinimum(t *testing.T) {
	if got := ContentWidth(0, 5, true); got != MinWidth {
		t.Fatalf("got %d, want %d", got, MinWidth)
	}
}

func TestCenterPadsEveryLineWithHalfTheFreeSpace(t *testing.T) {
	got := Center("a\nb", 20, 10)
	if got != "     a\n     b" {
		t.Fatalf("got %q", got)
	}
}

func TestCenterLeavesTextAloneWhenNoFreeSpace(t *testing.T) {
	if got := Center("a", 10, 10); got != "a" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveStyleKeepsExplicitChoice(t *testing.T) {
	if got := ResolveStyle("light", true); got != "light" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveStyleAutoPicksDarkOnDarkBackground(t *testing.T) {
	if got := ResolveStyle("auto", true); got != "dark" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveStyleAutoPicksLightOnLightBackground(t *testing.T) {
	if got := ResolveStyle("auto", false); got != "light" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderContainsDocumentText(t *testing.T) {
	out, err := Render("# Hello\n\nworld", "dark", 80)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Hello") || !strings.Contains(out, "world") {
		t.Fatalf("missing text in %q", out)
	}
}

func TestRenderWrapsLongParagraphs(t *testing.T) {
	long := strings.Repeat("word ", 60)
	out, err := Render(long, "dark", 40)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(strings.TrimRight(out, "\n"), "\n") + 1; lines < 5 {
		t.Fatalf("expected wrapping, got %d lines", lines)
	}
}

func TestRenderRejectsUnknownStyle(t *testing.T) {
	if _, err := Render("x", "neon", 80); err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderHidesMarkdownSyntax(t *testing.T) {
	md := "# One\n\n## Two\n\n### Three\n\n**bold** *em* `code` ~~gone~~\n\n> quote\n\n- [x] done\n"
	for _, style := range []string{"dark", "light", PlainStyle} {
		out, err := Render(md, style, 80)
		if err != nil {
			t.Fatal(err)
		}
		text := xansi.Strip(out)
		for _, token := range []string{"#", "**", "*em", "`", "~~", "> ", "[x]"} {
			if strings.Contains(text, token) {
				t.Errorf("%s: found markdown token %q in %q", style, token, text)
			}
		}
	}
}

func TestRenderPlainHasNoEscapesOrTrailingSpaces(t *testing.T) {
	out, err := Render("# Title\n\n**bold** and `code`", PlainStyle, 80)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\x1b") || strings.Contains(out, " \n") {
		t.Fatalf("got %q", out)
	}
}

func TestRenderHorizontalRuleSpansWidth(t *testing.T) {
	out, err := Render("a\n\n---\n\nb", PlainStyle, 40)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, strings.Repeat("─", 40-2*documentMargin)) {
		t.Fatalf("got %q", out)
	}
}
