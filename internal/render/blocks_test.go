package render

import (
	"strings"
	"testing"

	xansi "github.com/charmbracelet/x/ansi"
)

func visibleLines(out string) []string {
	return strings.Split(xansi.Strip(out), "\n")
}

func TestExtractBlocksReplacesFenceWithPlaceholder(t *testing.T) {
	src, blocks := extractBlocks("before\n```go\nx := 1\n\ty\n```\nafter")
	if len(blocks) != 1 || blocks[0].kind != codeBlock || blocks[0].lang != "go" || blocks[0].text != "x := 1\n\ty" {
		t.Fatalf("blocks = %+v", blocks)
	}
	if !strings.Contains(src, "\n\nMARKYBLOCK0END\n\n") || strings.Contains(src, "```") {
		t.Fatalf("src = %q", src)
	}
}

func TestExtractBlocksKeepsListIndentation(t *testing.T) {
	src, blocks := extractBlocks("- item\n\n  ~~~\n  code\n  ~~~\n")
	if len(blocks) != 1 || blocks[0].text != "code" || !strings.Contains(src, "  MARKYBLOCK0END") {
		t.Fatalf("src = %q blocks = %+v", src, blocks)
	}
}

func TestExtractBlocksReadsUnclosedFenceToTheEnd(t *testing.T) {
	_, blocks := extractBlocks("```\na\nb")
	if len(blocks) != 1 || blocks[0].text != "a\nb" {
		t.Fatalf("blocks = %+v", blocks)
	}
}

func TestExtractBlocksCollectsQuoteWithLazyContinuation(t *testing.T) {
	_, blocks := extractBlocks("> one\nlazy\n>\n> two\n\nout")
	if len(blocks) != 1 || blocks[0].kind != quoteBlock || blocks[0].text != "one\nlazy\n\ntwo" {
		t.Fatalf("blocks = %+v", blocks)
	}
}

func TestExtractBlocksLeavesQuoteMarkersInsideCode(t *testing.T) {
	_, blocks := extractBlocks("```\n> not a quote\n```")
	if len(blocks) != 1 || blocks[0].kind != codeBlock || blocks[0].text != "> not a quote" {
		t.Fatalf("blocks = %+v", blocks)
	}
}

func TestCodePanelHasBackgroundAndLanguageLabel(t *testing.T) {
	out, err := Render("text\n\n```go\nfunc main() {}\n```\n\nmore", "dark", 60)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "48;2;24;24;37") {
		t.Fatalf("missing panel background in %q", out)
	}
	lines := visibleLines(out)
	var panel []string
	for i, l := range lines {
		if strings.Contains(l, "func main") {
			panel = lines[i-1 : i+2]
		}
	}
	if panel == nil || !strings.HasSuffix(strings.TrimRight(panel[0], " "), "go") {
		t.Fatalf("no labelled panel in %q", lines)
	}
	for _, l := range panel {
		if w := xansi.StringWidth(l); w != 60-documentMargin {
			t.Errorf("panel line %q is %d wide, want %d", l, w, 60-documentMargin)
		}
	}
}

func TestCodePanelWrapsLongLines(t *testing.T) {
	out, err := Render("```\n"+strings.Repeat("x", 120)+"\n```", "dark", 40)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range visibleLines(out) {
		if w := xansi.StringWidth(l); w > 40 {
			t.Fatalf("line %q is %d wide", l, w)
		}
	}
}

// Regression: glamour wrapped lists inside quotes twice and pushed words
// outside the bar.
func TestQuoteLinesAllCarryTheBarAndFit(t *testing.T) {
	md := "> [!NOTE]\n" +
		"> - Aliases and functions only affect **interactive** shells. Scripts, `make` and cron still use the real `cat`.\n" +
		"> - The PowerShell and Nushell snippets follow each shell's documented syntax but were not run on the author's machine. Issues welcome.\n"
	for _, width := range []int{50, 70, 80, 90, 100} {
		out, err := Render(md, PlainStyle, width)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range visibleLines(out) {
			if strings.TrimSpace(l) == "" {
				continue
			}
			if !strings.HasPrefix(strings.TrimLeft(l, " "), "┃") {
				t.Errorf("width %d: line outside the quote bar: %q", width, l)
			}
			if w := xansi.StringWidth(l); w > width {
				t.Errorf("width %d: line %q is %d wide", width, l, w)
			}
		}
	}
}

func TestAlertBecomesTitledCallout(t *testing.T) {
	out, err := Render("> [!WARNING]\n> Careful.", PlainStyle, 60)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "┃ ▲ Warning") || !strings.Contains(out, "┃ Careful.") || strings.Contains(out, "[!") {
		t.Fatalf("got %q", out)
	}
}

func TestNestedQuotesDrawTwoBars(t *testing.T) {
	out, err := Render("> outer\n>\n> > inner", PlainStyle, 60)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "┃ ┃ inner") {
		t.Fatalf("got %q", out)
	}
}

func TestCodeInsideListItemAlignsWithItemText(t *testing.T) {
	out, err := Render("- Item:\n\n  ```\n  code\n  ```\n", PlainStyle, 60)
	if err != nil {
		t.Fatal(err)
	}
	lines := visibleLines(out)
	var item, code string
	for _, l := range lines {
		switch {
		case strings.Contains(l, "Item:"):
			item = l
		case strings.Contains(l, "code"):
			code = l
		}
	}
	column := func(line, word string) int { return xansi.StringWidth(line[:strings.Index(line, word)]) }
	if item == "" || column(code, "code") != column(item, "Item")+panelPad {
		t.Fatalf("item %q, code %q", item, code)
	}
}

func TestSgrSkipsMissingColors(t *testing.T) {
	if got := sgr("", "", false, false); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := sgr("#FF0000", "#000001", true, true); got != "\x1b[1;3;38;2;255;0;0;48;2;0;0;1m" {
		t.Fatalf("got %q", got)
	}
}
