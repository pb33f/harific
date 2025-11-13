package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/pb33f/braid/tui"
)

func LaunchTUI(harFile string) error {
	model, err := tui.NewHARViewModel(harFile)
	if err != nil {
		return fmt.Errorf("failed to create TUI model: %w", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}