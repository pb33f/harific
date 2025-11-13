package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view <har-file>",
	Short: "Open a HAR file in the terminal UI viewer",
	Long: `Launch an interactive terminal user interface to browse and explore
the contents of a HAR file. Navigate through requests and responses,
view headers, bodies, and timing information in a user-friendly format.`,
	Args: cobra.ExactArgs(1), // Require exactly one positional argument
	Example: `  braid view recording.har
  braid view recording.har -v`,
	RunE: runView,
}

func init() {
	rootCmd.AddCommand(viewCmd)
}

func runView(cmd *cobra.Command, args []string) error {
	harFile := args[0]

	if err := ValidateHARFile(harFile); err != nil {
		return fmt.Errorf("invalid HAR file: %w", err)
	}

	// tui handles indexing with spinner
	if err := LaunchTUI(harFile); err != nil {
		return fmt.Errorf("failed to launch TUI: %w", err)
	}

	return nil
}