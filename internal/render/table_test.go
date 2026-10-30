package render

import (
	"strings"
	"testing"

	xansi "github.com/charmbracelet/x/ansi"
)

const wideTable = "| SKU | Free | Fields |\n|---|--:|---|\n" +
	"| Text Search Essentials (IDs Only) | **Unlimited** | only `places.id` |\n" +
	"| Text Search Enterprise / Nearby Enterprise | 1.000 | `websiteUri`, `nationalPhoneNumber`, `rating`, hours |\n" +
	"| Place Details Pro | 5.000 | same |\n"

func TestExtractBlocksReadsTableWithAlignmentAndEscapedPipes(t *testing.T) {
	src, blocks := extractBlocks("| a | b | c |\n|:--|:-:|--:|\n| `x|y` | p \\| q |\n\nafter")
	if len(blocks) != 1 || blocks[0].kind != tableBlock {
		t.Fatalf("blocks = %+v", blocks)
	}
	b := blocks[0]
	if want := []align{alignLeft, alignCenter, alignRight}; len(b.aligns) != 3 || b.aligns[0] != want[0] || b.aligns[1] != want[1] || b.aligns[2] != want[2] {
		t.Fatalf("aligns = %v", b.aligns)
	}
	if len(b.rows) != 2 || b.rows[1][0] != "`x|y`" || b.rows[1][1] != "p | q" || b.rows[1][2] != "" {
		t.Fatalf("rows = %q", b.rows)
	}
	if strings.Contains(src, "|") || !strings.Contains(src, "after") {
		t.Fatalf("src = %q", src)
	}
}

func TestExtractBlocksIgnoresPipesWithoutDelimiterRow(t *testing.T) {
	_, blocks := extractBlocks("a | b\nc | d")
	if len(blocks) != 0 {
		t.Fatalf("blocks = %+v", blocks)
	}
}

func TestTableSeparatesEveryRowAndFits(t *testing.T) {
	for _, width := range []int{40, 60, 80, 100} {
		out, err := Render(wideTable, PlainStyle, width)
		if err != nil {
			t.Fatal(err)
		}
		var heads, rules int
		for _, l := range visibleLines(out) {
			if w := xansi.StringWidth(l); w > width {
				t.Errorf("width %d: line %q is %d wide", width, l, w)
			}
			heads += strings.Count(l, "┿")
			rules += strings.Count(l, "┼")
		}
		if heads != 2 || rules != 4 {
			t.Errorf("width %d: %d header crossings, %d row crossings in\n%s", width, heads, rules, out)
		}
	}
}

func TestTableWrapsHeaderInsteadOfTruncating(t *testing.T) {
	out, err := Render("| Fields that usually trigger the charge | n |\n|---|---|\n| x | 1 |\n", PlainStyle, 20)
	if err != nil {
		t.Fatal(err)
	}
	if text := strings.Join(strings.Fields(out), " "); !strings.Contains(text, "charge") {
		t.Fatalf("header truncated:\n%s", out)
	}
}

func TestTableAlignsCells(t *testing.T) {
	out, err := Render("| left | right |\n|---|---:|\n| a | b |\n", PlainStyle, 60)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range visibleLines(out) {
		if strings.Contains(l, "a │") && !strings.HasSuffix(strings.TrimRight(l, " "), "│     b") {
			t.Fatalf("right column not right-aligned: %q", l)
		}
	}
}

func TestTableHeaderIsBoldAndColored(t *testing.T) {
	out, err := Render("| head |\n|---|\n| body |\n", "dark", 60)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "38;2;137;179;250;1mhead") {
		t.Fatalf("header not styled: %q", out)
	}
}
