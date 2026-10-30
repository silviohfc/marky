package pager

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func numberedLines(width int) (string, error) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d w%d", i, width)
	}
	return strings.Join(lines, "\n"), nil
}

func ready(t *testing.T) Model {
	t.Helper()
	m, _ := New("doc.md", numberedLines).Update(tea.WindowSizeMsg{Width: 80, Height: 11})
	return m.(Model)
}

func press(m Model, msg tea.Msg) Model {
	next, _ := m.Update(msg)
	return next.(Model)
}

func runeKey(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestQuitKeysReturnQuitCommand(t *testing.T) {
	keys := map[string]tea.KeyMsg{
		"q":      runeKey('q'),
		"esc":    {Type: tea.KeyEsc},
		"ctrl+c": {Type: tea.KeyCtrlC},
	}
	for name, k := range keys {
		_, cmd := ready(t).Update(k)
		if !isQuit(cmd) {
			t.Errorf("%s did not quit", name)
		}
	}
}

func TestViewShowsNothingUntilSized(t *testing.T) {
	if got := New("doc.md", numberedLines).View(); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestJScrollsOneLineDown(t *testing.T) {
	m := press(ready(t), runeKey('j'))
	if m.viewport.YOffset != 1 {
		t.Fatalf("offset %d", m.viewport.YOffset)
	}
}

func TestDownArrowScrollsOneLineDown(t *testing.T) {
	m := press(ready(t), tea.KeyMsg{Type: tea.KeyDown})
	if m.viewport.YOffset != 1 {
		t.Fatalf("offset %d", m.viewport.YOffset)
	}
}

func TestKScrollsOneLineUp(t *testing.T) {
	m := press(press(ready(t), runeKey('j')), runeKey('k'))
	if m.viewport.YOffset != 0 {
		t.Fatalf("offset %d", m.viewport.YOffset)
	}
}

func TestSpaceScrollsOnePageDown(t *testing.T) {
	m := press(ready(t), tea.KeyMsg{Type: tea.KeySpace})
	if m.viewport.YOffset != 10 {
		t.Fatalf("offset %d", m.viewport.YOffset)
	}
}

func TestPageDownAndFScrollOnePage(t *testing.T) {
	for _, k := range []tea.KeyMsg{{Type: tea.KeyPgDown}, runeKey('f')} {
		if m := press(ready(t), k); m.viewport.YOffset != 10 {
			t.Errorf("%v: offset %d", k, m.viewport.YOffset)
		}
	}
}

func TestBAndPageUpScrollOnePageUp(t *testing.T) {
	for _, k := range []tea.KeyMsg{{Type: tea.KeyPgUp}, runeKey('b')} {
		m := press(press(ready(t), tea.KeyMsg{Type: tea.KeyPgDown}), k)
		if m.viewport.YOffset != 0 {
			t.Errorf("%v: offset %d", k, m.viewport.YOffset)
		}
	}
}

func TestDScrollsHalfPageDown(t *testing.T) {
	if m := press(ready(t), runeKey('d')); m.viewport.YOffset != 5 {
		t.Fatalf("offset %d", m.viewport.YOffset)
	}
}

func TestUScrollsHalfPageUp(t *testing.T) {
	m := press(press(ready(t), tea.KeyMsg{Type: tea.KeyPgDown}), runeKey('u'))
	if m.viewport.YOffset != 5 {
		t.Fatalf("offset %d", m.viewport.YOffset)
	}
}

func TestBigGJumpsToBottom(t *testing.T) {
	if m := press(ready(t), runeKey('G')); !m.viewport.AtBottom() {
		t.Fatal("not at bottom")
	}
}

func TestEndJumpsToBottom(t *testing.T) {
	if m := press(ready(t), tea.KeyMsg{Type: tea.KeyEnd}); !m.viewport.AtBottom() {
		t.Fatal("not at bottom")
	}
}

func TestGAndHomeJumpToTop(t *testing.T) {
	for _, k := range []tea.KeyMsg{runeKey('g'), {Type: tea.KeyHome}} {
		m := press(press(ready(t), runeKey('G')), k)
		if !m.viewport.AtTop() {
			t.Errorf("%v: not at top", k)
		}
	}
}

func TestMouseWheelScrolls(t *testing.T) {
	m := press(ready(t), tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress, Type: tea.MouseWheelDown})
	if m.viewport.YOffset == 0 {
		t.Fatal("wheel did not scroll")
	}
}

func TestFooterShowsNamePercentAndHints(t *testing.T) {
	m := press(ready(t), runeKey('G'))
	view := m.View()
	for _, want := range []string{"doc.md", "100%", "q quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("footer missing %q in %q", want, view)
		}
	}
}

func TestFooterShowsZeroPercentAtTop(t *testing.T) {
	if !strings.Contains(ready(t).View(), "0%") {
		t.Fatal("missing 0%")
	}
}

func TestViewFillsTerminalHeight(t *testing.T) {
	if got := strings.Count(ready(t).View(), "\n") + 1; got != 11 {
		t.Fatalf("view has %d lines", got)
	}
}

func TestResizeRerendersWithNewWidth(t *testing.T) {
	m := press(ready(t), tea.WindowSizeMsg{Width: 60, Height: 11})
	if !strings.Contains(m.View(), "w60") {
		t.Fatalf("not re-rendered: %q", m.View())
	}
}

func TestResizeKeepsScrollPositionProportional(t *testing.T) {
	m := press(ready(t), runeKey('G'))
	m = press(m, tea.WindowSizeMsg{Width: 60, Height: 11})
	if !m.viewport.AtBottom() {
		t.Fatal("expected to stay at bottom")
	}
	m = press(press(ready(t), tea.KeyMsg{Type: tea.KeyPgDown}), tea.WindowSizeMsg{Width: 60, Height: 21})
	got := m.viewport.ScrollPercent()
	if got < 0.08 || got > 0.14 {
		t.Fatalf("percent %f", got)
	}
}

func TestRenderFailureIsShownInsteadOfContent(t *testing.T) {
	failing := func(int) (string, error) { return "", errors.New("boom") }
	m, _ := New("doc.md", failing).Update(tea.WindowSizeMsg{Width: 80, Height: 11})
	if !strings.Contains(m.View(), "boom") {
		t.Fatalf("got %q", m.View())
	}
}
