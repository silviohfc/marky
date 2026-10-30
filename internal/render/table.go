package render

import (
	"regexp"
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
)

// Glamour hands tables to lipgloss, which truncates header cells, splits the
// width evenly whatever the content, and draws no line between rows, so a row
// whose cells wrap blends into the next one. marky draws tables itself: each
// cell is rendered as a Markdown fragment, wrapped to its column, and every
// row is closed by a rule.

type align int

const (
	alignLeft align = iota
	alignCenter
	alignRight
)

const (
	cellGap      = " │ "
	minColumn    = 3
	preferColumn = 12
)

var delimiterRow = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)*\|?\s*$`)

// readTable collects a GFM table that starts at lines[start] when the next
// line is a delimiter row with the same number of cells as the header.
func readTable(lines []string, start int) (block, int, bool) {
	if start+1 >= len(lines) || !strings.Contains(lines[start], "|") || !delimiterRow.MatchString(lines[start+1]) {
		return block{}, start, false
	}
	header := splitRow(lines[start])
	delims := splitRow(lines[start+1])
	if len(header) != len(delims) {
		return block{}, start, false
	}
	b := block{kind: tableBlock, rows: [][]string{header}}
	for _, d := range delims {
		b.aligns = append(b.aligns, alignOf(d))
	}
	i := start + 2
	for ; i < len(lines) && strings.TrimSpace(lines[i]) != "" && strings.Contains(lines[i], "|"); i++ {
		row := splitRow(lines[i])
		// Rows are cut or filled to the header's cell count, as GFM does.
		row = append(row, make([]string, max(0, len(header)-len(row)))...)
		b.rows = append(b.rows, row[:len(header)])
	}
	return b, i, true
}

// splitRow splits a table line on pipes that are neither escaped nor inside
// a code span.
func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	if strings.HasSuffix(line, "|") && !strings.HasSuffix(line, `\|`) {
		line = line[:len(line)-1]
	}
	var cells []string
	var cell strings.Builder
	inCode := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '\\' && i+1 < len(line) && line[i+1] == '|':
			cell.WriteByte('|')
			i++
			continue
		case c == '`':
			inCode = !inCode
		case c == '|' && !inCode:
			cells = append(cells, strings.TrimSpace(cell.String()))
			cell.Reset()
			continue
		}
		cell.WriteByte(c)
	}
	return append(cells, strings.TrimSpace(cell.String()))
}

func alignOf(delim string) align {
	d := strings.TrimSpace(delim)
	left, right := strings.HasPrefix(d, ":"), strings.HasSuffix(d, ":")
	switch {
	case left && right:
		return alignCenter
	case right:
		return alignRight
	}
	return alignLeft
}

// drawTable lays the columns out to fit width, then draws the header, a
// heavy rule under it and a light rule between body rows.
func drawTable(b block, style string, width int) ([]string, error) {
	p := palettes[style]
	cols := len(b.aligns)
	gaps := (cols - 1) * xansi.StringWidth(cellGap)
	widths, err := columnWidths(b.rows, style, max(cols*minColumn, width-gaps))
	if err != nil {
		return nil, err
	}

	rule := func(fill, cross string) string {
		parts := make([]string, cols)
		for c, w := range widths {
			parts[c] = strings.Repeat(fill, w)
		}
		sep := strings.Repeat(fill, 1) + cross + strings.Repeat(fill, 1)
		return sgr(p.subtle, "", false, false) + strings.Join(parts, sep) + reset
	}
	gap := sgr(p.subtle, "", false, false) + cellGap + reset

	var lines []string
	for r, row := range b.rows {
		cells := make([][]string, cols)
		height := 0
		for c, text := range row {
			cell, err := renderCell(text, style, widths[c], r == 0)
			if err != nil {
				return nil, err
			}
			cells[c] = cell
			height = max(height, len(cell))
		}
		for l := range height {
			parts := make([]string, cols)
			for c := range cols {
				line := ""
				if l < len(cells[c]) {
					line = cells[c][l]
				}
				parts[c] = pad(line, widths[c], b.aligns[c])
			}
			lines = append(lines, strings.Join(parts, gap))
		}
		switch {
		case r == 0:
			lines = append(lines, rule("━", "┿"))
		case r < len(b.rows)-1:
			lines = append(lines, rule("─", "┼"))
		}
	}
	return lines, nil
}

// renderCell renders a cell's inline Markdown wrapped to width, with the
// trailing padding glamour adds removed so the cell can be aligned. Words
// longer than the column are broken, since glamour lets them overflow.
func renderCell(text, style string, width int, head bool) ([]string, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	out, err := render(text, style, width, layout{fragment: true, head: head})
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range trimBlankLines(strings.Split(out, "\n")) {
		visible := xansi.StringWidth(strings.TrimRight(xansi.Strip(line), " "))
		line = xansi.Hardwrap(xansi.Truncate(line, visible, ""), width, true)
		for _, part := range strings.Split(line, "\n") {
			lines = append(lines, part+reset)
		}
	}
	return lines, nil
}

func pad(line string, width int, a align) string {
	free := max(0, width-xansi.StringWidth(line))
	left := 0
	switch a {
	case alignCenter:
		left = free / 2
	case alignRight:
		left = free
	}
	return strings.Repeat(" ", left) + line + strings.Repeat(" ", free-left)
}

// columnWidths gives every column its natural width when the table fits.
// Otherwise each column first gets up to its longest word (or a fair share
// when words are long), and the rest of the room goes to the columns that
// still wrap, in proportion to how much more they need.
func columnWidths(rows [][]string, style string, available int) ([]int, error) {
	cols := len(rows[0])
	natural := make([]int, cols)
	word := make([]int, cols)
	for _, row := range rows {
		for c, text := range row {
			plain, err := render(text, PlainStyle, 1<<16, layout{fragment: true})
			if err != nil {
				return nil, err
			}
			for _, line := range strings.Split(plain, "\n") {
				natural[c] = max(natural[c], xansi.StringWidth(strings.TrimSpace(line)))
				for _, w := range strings.Fields(line) {
					word[c] = max(word[c], xansi.StringWidth(w))
				}
			}
		}
	}
	total := sum(natural)
	if total <= available {
		return natural, nil
	}

	widths := make([]int, cols)
	fair := max(minColumn, available/cols)
	for c := range cols {
		widths[c] = max(min(natural[c], minColumn), min(natural[c], word[c], max(preferColumn, fair)))
	}
	for sum(widths) > available {
		widest := 0
		for c := range cols {
			if widths[c] > widths[widest] {
				widest = c
			}
		}
		if widths[widest] <= minColumn {
			break
		}
		widths[widest]--
	}

	room := available - sum(widths)
	need := make([]int, cols)
	for c := range cols {
		need[c] = natural[c] - widths[c]
	}
	totalNeed := sum(need)
	given := 0
	for c := range cols {
		if totalNeed > 0 {
			extra := room * need[c] / totalNeed
			widths[c] += extra
			given += extra
		}
	}
	// Hand out what integer division left over, one column at a time.
	for c := 0; given < room && totalNeed > 0; c = (c + 1) % cols {
		if widths[c] < natural[c] {
			widths[c]++
			given++
		}
	}
	return widths, nil
}

func sum(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}
