package pager

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const footerHeight = 1

const keyHints = "q quit · j/k scroll · space/b page · g/G top/bottom"

var (
	accent  = lipgloss.AdaptiveColor{Light: "#8839EF", Dark: "#CBA6F7"}
	onAccnt = lipgloss.AdaptiveColor{Light: "#EFF1F5", Dark: "#1E1E2E"}
	muted   = lipgloss.AdaptiveColor{Light: "#8C8FA1", Dark: "#7F849C"}
	track   = lipgloss.AdaptiveColor{Light: "#CCD0DA", Dark: "#45475A"}

	nameStyle    = lipgloss.NewStyle().Bold(true).Foreground(onAccnt).Background(accent).Padding(0, 1)
	hintStyle    = lipgloss.NewStyle().Foreground(muted).Padding(0, 1)
	percentStyle = lipgloss.NewStyle().Bold(true).Foreground(accent).PaddingLeft(1)
	fillStyle    = lipgloss.NewStyle().Foreground(accent)
	trackStyle   = lipgloss.NewStyle().Foreground(track)
)

const progressWidth = 12

type Renderer func(termWidth int) (string, error)

type Model struct {
	name     string
	render   Renderer
	viewport viewport.Model
	width    int
	ready    bool
}

func New(name string, render Renderer) Model {
	return Model{name: name, render: render}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg), nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "g", "home":
			m.viewport.GotoTop()
			return m, nil
		case "G", "end":
			m.viewport.GotoBottom()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if !m.ready {
		return ""
	}
	return m.viewport.View() + "\n" + m.footer()
}

func (m Model) resize(msg tea.WindowSizeMsg) Model {
	height := max(1, msg.Height-footerHeight)
	progress := 0.0
	if m.ready {
		progress = m.viewport.ScrollPercent()
	} else {
		m.viewport = viewport.New(msg.Width, height)
		m.ready = true
	}
	m.width = msg.Width
	m.viewport.Width = msg.Width
	m.viewport.Height = height
	m.viewport.SetContent(m.content(msg.Width))
	m.viewport.SetYOffset(offsetFor(progress, m.viewport))
	return m
}

func (m Model) content(width int) string {
	text, err := m.render(width)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return text
}

func offsetFor(progress float64, v viewport.Model) int {
	maxOffset := max(0, v.TotalLineCount()-v.Height)
	return int(math.Round(progress * float64(maxOffset)))
}

// footer draws: [ name ] hints ............ ━━━━──── 42%
func (m Model) footer() string {
	progress := m.viewport.ScrollPercent()
	filled := int(math.Round(progress * progressWidth))
	right := fillStyle.Render(strings.Repeat("━", filled)) +
		trackStyle.Render(strings.Repeat("─", progressWidth-filled)) +
		percentStyle.Render(fmt.Sprintf("%3d%% ", int(progress*100)))
	left := nameStyle.Render(m.name) + hintStyle.Render(keyHints)
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return lipgloss.NewStyle().MaxWidth(m.width).Render(nameStyle.Render(m.name) + " " + right)
	}
	return left + strings.Repeat(" ", gap) + right
}
