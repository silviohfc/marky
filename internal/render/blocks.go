package render

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	xansi "github.com/charmbracelet/x/ansi"
)

// Glamour draws fenced code with no container and double-wraps lists inside
// quotes, pushing words outside the quote bar. So marky pulls fenced code,
// quotes and tables (see table.go) out of the Markdown, leaves a placeholder paragraph in their place,
// and draws them itself once glamour has laid out everything else.

type blockKind int

const (
	codeBlock blockKind = iota
	quoteBlock
	tableBlock
)

type block struct {
	kind blockKind
	lang string // code only
	text string // code body, or the quote's Markdown without the > markers

	rows   [][]string // table only: header first, then body rows
	aligns []align    // table only: one per column
}

const (
	tabWidth   = 4
	panelPad   = 2
	quoteBar   = "┃ "
	minContent = 10
	reset      = "\x1b[0m"
)

var (
	fenceOpen   = regexp.MustCompile("^(\\s*)(`{3,}|~{3,})\\s*([^\\s`]*)[^`]*$")
	quoteLine   = regexp.MustCompile(`^(\s*)>[ \t]?(.*)$`)
	notLazy     = regexp.MustCompile("^\\s*([-*+]\\s|\\d+[.)]\\s|#|```|~~~|---|\\*\\*\\*|___|\\|)")
	placeholder = regexp.MustCompile(`MARKYBLOCK(\d+)END`)
	alertMarker = regexp.MustCompile(`^\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*(.*)$`)
)

func token(i int) string { return fmt.Sprintf("MARKYBLOCK%dEND", i) }

// extractBlocks replaces fenced code blocks and blockquotes with placeholder
// paragraphs, keeping their indentation so they stay inside list items.
func extractBlocks(markdown string) (string, []block) {
	lines := strings.Split(markdown, "\n")
	var out []string
	var blocks []block
	for i := 0; i < len(lines); {
		var b block
		var indent string
		var next int
		if m := fenceOpen.FindStringSubmatch(lines[i]); m != nil {
			indent = m[1]
			b, next = readFence(lines, i, m[1], m[2], m[3])
		} else if m := quoteLine.FindStringSubmatch(lines[i]); m != nil {
			indent = m[1]
			b, next = readQuote(lines, i)
		} else if t, end, ok := readTable(lines, i); ok {
			indent = lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " "))]
			b, next = t, end
		} else {
			out = append(out, lines[i])
			i++
			continue
		}
		out = append(out, "", indent+token(len(blocks)), "")
		blocks = append(blocks, b)
		i = next
	}
	return strings.Join(out, "\n"), blocks
}

func readFence(lines []string, start int, indent, fence, lang string) (block, int) {
	var body []string
	i := start + 1
	for ; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if len(trimmed) >= len(fence) && strings.Trim(trimmed, fence[:1]) == "" {
			i++
			break
		}
		body = append(body, trimIndent(lines[i], len(indent)))
	}
	return block{kind: codeBlock, lang: lang, text: strings.Join(body, "\n")}, i
}

func readQuote(lines []string, start int) (block, int) {
	var body []string
	i := start
	for ; i < len(lines); i++ {
		if m := quoteLine.FindStringSubmatch(lines[i]); m != nil {
			body = append(body, m[2])
			continue
		}
		lazy := strings.TrimSpace(lines[i]) != "" && strings.TrimSpace(body[len(body)-1]) != "" &&
			!notLazy.MatchString(lines[i])
		if !lazy {
			break
		}
		body = append(body, strings.TrimSpace(lines[i]))
	}
	return block{kind: quoteBlock, text: strings.Join(body, "\n")}, i
}

func trimIndent(line string, n int) string {
	for n > 0 && line != "" && line[0] == ' ' {
		line, n = line[1:], n-1
	}
	return line
}

// insertBlocks swaps every placeholder line in glamour's output for the
// drawn block. width is the full line width glamour laid out.
func insertBlocks(out string, blocks []block, style string, width int, l layout) (string, error) {
	lines := strings.Split(out, "\n")
	type slot struct {
		line   int // index in lines of the placeholder
		block  int
		prefix string
		drawn  []string
		err    error
	}
	var slots []slot
	for i, line := range lines {
		if !strings.Contains(line, "MARKYBLOCK") {
			continue
		}
		if m := placeholder.FindStringSubmatchIndex(line); m != nil {
			index, _ := strconv.Atoi(line[m[2]:m[3]])
			slots = append(slots, slot{line: i, block: index, prefix: line[:m[0]]})
		}
	}
	if len(slots) == 0 {
		return out, nil
	}

	// Blocks are independent, and each one is a full glamour render, so
	// they are drawn in parallel.
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i := range slots {
		s := &slots[i]
		prefix := s.prefix
		if hasText(prefix) {
			prefix = strings.Repeat(" ", itemIndent(prefix))
		}
		available := max(minContent, width-xansi.StringWidth(prefix)-l.margin())
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			s.drawn, s.err = drawBlock(blocks[s.block], style, available)
		}()
	}
	wg.Wait()

	result := make([]string, 0, len(lines))
	next := 0
	for _, s := range slots {
		if s.err != nil {
			return "", s.err
		}
		result = append(result, lines[next:s.line]...)
		next = s.line + 1
		prefix := s.prefix
		if hasText(prefix) {
			// Glamour joins a list item's paragraphs onto one line, so the
			// block may follow the item text: keep the text on its own line.
			result = append(result, strings.TrimRight(prefix, " "))
			prefix = strings.Repeat(" ", itemIndent(prefix))
		}
		// The first line keeps glamour's prefix (it may hold a list bullet);
		// the rest are aligned under it.
		lead := strings.Repeat(" ", xansi.StringWidth(prefix))
		for i, d := range s.drawn {
			if i == 0 {
				result = append(result, prefix+reset+d)
				continue
			}
			result = append(result, lead+d)
		}
	}
	result = append(result, lines[next:]...)
	return strings.Join(result, "\n"), nil
}

var listMarker = regexp.MustCompile(`^(\s*(?:•|✓|○|\d+\.)\s)`)

// hasText reports whether a placeholder prefix holds more than indentation
// and a list marker.
func hasText(prefix string) bool {
	visible := xansi.Strip(prefix)
	return strings.TrimSpace(listMarker.ReplaceAllString(visible, "")) != ""
}

// itemIndent is the column where a list item's content starts.
func itemIndent(prefix string) int {
	visible := xansi.Strip(prefix)
	if m := listMarker.FindString(visible); m != "" {
		return xansi.StringWidth(m)
	}
	return len(visible) - len(strings.TrimLeft(visible, " "))
}

func drawBlock(b block, style string, width int) ([]string, error) {
	switch b.kind {
	case codeBlock:
		return drawCode(b, style, width)
	case tableBlock:
		return drawTable(b, style, width)
	}
	return drawQuote(b, style, width)
}

// drawCode draws the code on a full-width panel with padding and a
// language label, so it stands apart from the surrounding text.
func drawCode(b block, style string, width int) ([]string, error) {
	p := palettes[style]
	code := strings.ReplaceAll(strings.TrimRight(b.text, "\n"), "\t", strings.Repeat(" ", tabWidth))
	inner := max(1, width-2*panelPad)
	highlighted, err := highlight(code, b.lang, p)
	if err != nil {
		return nil, err
	}
	bg := sgr("", p.panel, false, false)
	pad := strings.Repeat(" ", panelPad)
	row := func(content string) string {
		content = strings.ReplaceAll(content, reset, reset+bg)
		fill := strings.Repeat(" ", max(0, inner-xansi.StringWidth(content)))
		return bg + pad + content + bg + fill + pad + reset
	}

	label := ""
	if b.lang != "" {
		label = strings.Repeat(" ", max(0, inner-xansi.StringWidth(b.lang))) +
			sgr(p.muted, p.panel, false, true) + b.lang + reset
	}
	lines := []string{row(label)}
	for _, line := range strings.Split(highlighted, "\n") {
		for _, part := range strings.Split(xansi.Hardwrap(line, inner, true), "\n") {
			lines = append(lines, row(part))
		}
	}
	return append(lines, row("")), nil
}

func highlight(code, lang string, p palette) (string, error) {
	if p.codeTheme == "" {
		return code, nil
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	theme, err := styles.Get(p.codeTheme).Builder().Add(chroma.Background, "bg:"+p.panel).Build()
	if err != nil {
		return "", err
	}
	tokens, err := chroma.Coalesce(lexer).Tokenise(nil, code)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := formatters.TTY16m.Format(&buf, theme, tokens); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// Icons are limited to glyphs common monospace fonts ship, in single width.
type alert struct{ icon, label string }

var alerts = map[string]alert{
	"NOTE":      {"◉", "Note"},
	"TIP":       {"◆", "Tip"},
	"IMPORTANT": {"‼", "Important"},
	"WARNING":   {"▲", "Warning"},
	"CAUTION":   {"✕", "Caution"},
}

func alertColor(kind string, p palette) string {
	return map[string]string{
		"NOTE": p.primary, "TIP": p.green, "IMPORTANT": p.secondary,
		"WARNING": p.yellow, "CAUTION": p.red,
	}[kind]
}

// drawQuote renders the quote's Markdown as a fragment narrow enough to
// fit next to the bar, then draws the bar on every line. GitHub alerts
// ([!NOTE], [!TIP], ...) become a titled callout in the alert's color.
func drawQuote(b block, style string, width int) ([]string, error) {
	p := palettes[style]
	text, kind := b.text, ""
	if first, rest, _ := strings.Cut(text, "\n"); alertMarker.MatchString(first) {
		m := alertMarker.FindStringSubmatch(first)
		kind, text = m[1], strings.TrimSpace(m[2]+"\n"+rest)
	}
	inner := max(minContent, width-xansi.StringWidth(quoteBar))
	body, err := render(text, style, inner, layout{fragment: true, quote: kind == ""})
	if err != nil {
		return nil, err
	}

	barColor := p.subtle
	var lines []string
	if kind != "" {
		barColor = alertColor(kind, p)
		a := alerts[kind]
		lines = append(lines, sgr(barColor, "", false, false)+quoteBar+reset+
			sgr(barColor, "", true, false)+a.icon+" "+a.label+reset)
	}
	bar := sgr(barColor, "", false, false) + quoteBar + reset
	for _, line := range trimBlankLines(strings.Split(body, "\n")) {
		lines = append(lines, bar+line)
	}
	return lines, nil
}

func trimBlankLines(lines []string) []string {
	blank := func(s string) bool { return strings.TrimSpace(xansi.Strip(s)) == "" }
	for len(lines) > 0 && blank(lines[0]) {
		lines = lines[1:]
	}
	for len(lines) > 0 && blank(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// sgr builds a truecolor escape sequence. Empty colors are skipped, so the
// plain palette produces no color codes.
func sgr(fg, bg string, bold, italic bool) string {
	var codes []string
	if bold {
		codes = append(codes, "1")
	}
	if italic {
		codes = append(codes, "3")
	}
	if c := rgb(fg); c != "" {
		codes = append(codes, "38;2;"+c)
	}
	if c := rgb(bg); c != "" {
		codes = append(codes, "48;2;"+c)
	}
	if len(codes) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

func rgb(hex string) string {
	if len(hex) != 7 || hex[0] != '#' {
		return ""
	}
	v, err := strconv.ParseUint(hex[1:], 16, 32)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d;%d;%d", v>>16&0xff, v>>8&0xff, v&0xff)
}
