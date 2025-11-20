package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view <har-file>",
	Short: "View and explore HAR files in the terminal UI",
	Long: `Open a HAR file in an interactive terminal user interface.
Navigate through requests and responses, view headers, bodies, and metadata.

The TUI provides:
  • Table view of all HTTP transactions
  • Split view for request/response details
  • Search functionality with live filtering
  • File type filtering
  • Syntax highlighting for JSON/YAML content`,
	Args: cobra.ExactArgs(1),
	Example: `  harific view recording.har
  harific view large-capture.har -v`,
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

	if err := LaunchTUI(harFile); err != nil {
		return fmt.Errorf("failed to launch TUI: %w", err)
	}

	return nil
}