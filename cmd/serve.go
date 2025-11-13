package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	port int
)

var serveCmd = &cobra.Command{
	Use:   "serve <har-file>",
	Short: "Start a mock server using a HAR file",
	Long: `Start a mock HTTP server that replays responses from a HAR file.
The server will match incoming requests against the captured requests
in the HAR file and return the corresponding responses.`,
	Args: cobra.ExactArgs(1), // Require exactly one positional argument
	Example: `  braid serve recording.har
  braid serve recording.har --port 8080
  braid serve recording.har -p 3000 -v`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&port, "port", "p", 9876, "Port to listen on")
}

func runServe(cmd *cobra.Command, args []string) error {
	harFile := args[0]
	logger := GetLogger()

	// Validate the HAR file exists and is accessible
	if err := ValidateHARFile(harFile); err != nil {
		return err
	}

	// Validate port range
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	logger.Info("starting mock server",
		"har_file", harFile,
		"port", port)

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// Initialize the HAR streamer
	streamer, err := InitializeStreamer(ctx, harFile, logger)
	if err != nil {
		return err
	}
	defer streamer.Close()

	// TODO: Initialize the mock HTTP server here
	// This will be implemented once we have the server package
	logger.Info("mock server starting", "address", fmt.Sprintf("http://localhost:%d", port))

	// capture interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// For now, just stub out the server loop
	logger.Warn("server implementation pending - waiting for interrupt signal")

	// Wait for interrupt signal
	<-sigChan

	// Cancel context to stop any ongoing operations
	cancel()
	logger.Info("shutting down mock server...")

	// TODO: Graceful server shutdown will go here

	logger.Info("mock server stopped")
	return nil
}