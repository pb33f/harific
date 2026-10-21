package cmd

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/pb33f/harific/tui"
)

func LaunchTUI(harFile string) error {
	model, err := tui.NewHARViewModel(harFile)
	if err != nil {
		return fmt.Errorf("failed to create TUI model: %w", err)
	}

	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	// cleanup resources
	if m, ok := finalModel.(*tui.HARViewModel); ok {
		if err := m.Cleanup(); err != nil {
			return fmt.Errorf("cleanup error: %w", err)
		}
	}

	return nil
}