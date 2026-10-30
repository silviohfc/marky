package pager

import (
	"fmt"
	"math"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const footerHeight = 1

const keyHints = "q quit · j/k line · space/b page · d/u half · g/G top/bottom"

var footerStyle = lipgloss.NewStyle().Reverse(true).Padding(0, 1)

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

func (m Model) footer() string {
	text := fmt.Sprintf("%s  %d%%  %s", m.name, int(m.viewport.ScrollPercent()*100), keyHints)
	return footerStyle.Width(m.width).MaxWidth(m.width).Render(text)
}
