package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/silviohfc/marky/internal/cli"
	"github.com/silviohfc/marky/internal/pager"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], environment()))
}

func environment() cli.Env {
	stdoutFd := int(os.Stdout.Fd())
	return cli.Env{
		Stdin:    os.Stdin,
		StdinTTY: term.IsTerminal(int(os.Stdin.Fd())),
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		IsTTY:    term.IsTerminal(stdoutFd),
		Size: func() (int, int) {
			w, h, err := term.GetSize(stdoutFd)
			if err != nil {
				return 0, 0
			}
			return w, h
		},
		Dark:     lipgloss.HasDarkBackground,
		RunPager: runPager,
	}
}

func runPager(name string, render pager.Renderer) error {
	program := tea.NewProgram(
		pager.New(name, render),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := program.Run()
	return err
}
