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
	logger := GetLogger()

	// Validate the HAR file exists and is accessible
	if err := ValidateHARFile(harFile); err != nil {
		return err
	}

	logger.Info("launching terminal UI", "har_file", harFile)

	// Initialize the HAR streamer
	streamer, err := InitializeStreamer(cmd.Context(), harFile, logger)
	if err != nil {
		return err
	}
	defer streamer.Close()

	// Get index for additional verbose logging
	index := streamer.GetIndex()

	// In verbose mode, log first few entries as examples
	if verbose && index.TotalEntries > 0 {
		logger.Debug("sample entries in HAR file:")
		count := 3
		if index.TotalEntries < count {
			count = index.TotalEntries
		}
		for i := 0; i < count; i++ {
			meta := index.Entries[i]
			logger.Debug("entry",
				"index", i,
				"method", meta.Method,
				"url", meta.URL,
				"status", meta.StatusCode)
		}
	}

	// TODO: Initialize and launch the TUI here
	// This will be implemented once we have the TUI package
	logger.Warn("TUI implementation pending")

	// For now, just show a summary
	fmt.Println("\n=== HAR File Summary ===")
	fmt.Printf("File: %s\n", harFile)
	fmt.Printf("Total Entries: %d\n", index.TotalEntries)
	fmt.Printf("Unique URLs: %d\n", index.UniqueURLs)
	fmt.Printf("File Size: %.2f MB\n", float64(index.FileSize)/(1024*1024))
	fmt.Printf("Time Range: %s to %s\n",
		index.TimeRange.Start.Format("2006-01-02 15:04:05"),
		index.TimeRange.End.Format("2006-01-02 15:04:05"))

	if index.Creator != nil {
		fmt.Printf("Creator: %s %s\n", index.Creator.Name, index.Creator.Version)
	}
	if index.Browser != nil {
		fmt.Printf("Browser: %s %s\n", index.Browser.Name, index.Browser.Version)
	}

	fmt.Println("\nTUI viewer not yet implemented. Use 'braid serve' to start the mock server.")

	return nil
}