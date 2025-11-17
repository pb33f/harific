package cmd

import (
    "context"
    "fmt"
    "log/slog"
    "os"

    "github.com/pb33f/braid/motor"
    "github.com/spf13/cobra"
)

var (
    verbose bool
    port    int
    Logger  *slog.Logger

    rootCmd = &cobra.Command{
        Use:   "harific <har-file>",
        Short: "A high-performance HAR file viewer and mock server",
        Long: `Harific is a terminal user interface and server application that allows
reading of large HAR files. It can visualize requests and responses in a
meaningful way, as well as operate as a mock server to replay captured
responses based on incoming requests.`,
        Args: cobra.ExactArgs(1),
        Example: `  harific recording.har
  harific recording.har --port 8080
  harific recording.har -p 3000 -v`,
        PersistentPreRun: func(cmd *cobra.Command, args []string) {
            setupLogger()
        },
        RunE: runHarific,
    }
)

// Execute runs the root command
func Execute() error {
    return rootCmd.Execute()
}

func init() {
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
    rootCmd.Flags().IntVarP(&port, "port", "p", 9876, "Port for mock server (future functionality)")

    // will be reconfigured in PersistentPreRun based on flags
    setupLogger()
}

func runHarific(cmd *cobra.Command, args []string) error {
    harFile := args[0]

    if err := ValidateHARFile(harFile); err != nil {
        return fmt.Errorf("invalid HAR file: %w", err)
    }

    // TODO: When server functionality is implemented, it will start here
    // For now, just launch the TUI
    if err := LaunchTUI(harFile); err != nil {
        return fmt.Errorf("failed to launch TUI: %w", err)
    }

    return nil
}

// setupLogger configures the global slog logger based on the verbose flag
func setupLogger() {
    var opts *slog.HandlerOptions

    if verbose {
        opts = &slog.HandlerOptions{
            Level:     slog.LevelDebug,
            AddSource: true,
        }
    } else {
        opts = &slog.HandlerOptions{
            Level: slog.LevelInfo,
        }
    }

    handler := slog.NewTextHandler(os.Stderr, opts)
    Logger = slog.New(handler)
    slog.SetDefault(Logger)

    if verbose {
        Logger.Debug("verbose logging enabled",
            "level", slog.LevelDebug.String(),
            "pid", os.Getpid())
    }
}

// GetLogger returns the global logger instance
func GetLogger() *slog.Logger {
    if Logger == nil {
        setupLogger()
    }
    return Logger
}

// ValidateHARFile checks if the provided HAR file exists and is accessible
// check if the file exists, and it is not a directory.
func ValidateHARFile(harFile string) error {
    if harFile == "" {
        return fmt.Errorf("HAR file path is required")
    }

    info, err := os.Stat(harFile)
    if err != nil {
        if os.IsNotExist(err) {
            return fmt.Errorf("HAR file does not exist: %s", harFile)
        }
        return fmt.Errorf("error accessing HAR file: %w", err)
    }

    if info.IsDir() {
        return fmt.Errorf("provided path is a directory, not a file: %s", harFile)
    }

    return nil
}

// InitializeStreamer creates and initializes a HAR streamer with standard logging
func InitializeStreamer(ctx context.Context, harFile string, logger *slog.Logger) (motor.HARStreamer, error) {
    opts := motor.DefaultStreamerOptions()
    streamer, err := motor.NewHARStreamer(harFile, opts)
    if err != nil {
        return nil, fmt.Errorf("failed to create HAR streamer: %w", err)
    }

    logger.Debug("building HAR file index...")
    if err := streamer.Initialize(ctx); err != nil {
        if closeErr := streamer.Close(); closeErr != nil {
            logger.Debug("error closing streamer after initialization failure", "error", closeErr)
        }
        return nil, fmt.Errorf("failed to initialize HAR streamer: %w", err)
    }

    index := streamer.GetIndex()
    logger.Info("HAR file loaded",
        "entries", index.TotalEntries,
        "file_size_mb", index.FileSize/(1024*1024),
        "unique_urls", index.UniqueURLs,
        "build_time", index.BuildTime,
        "time_range", fmt.Sprintf("%s to %s",
            index.TimeRange.Start.Format("2006-01-02 15:04:05"),
            index.TimeRange.End.Format("2006-01-02 15:04:05")))

    if index.Creator != nil {
        logger.Debug("HAR creator", "name", index.Creator.Name, "version", index.Creator.Version)
    }
    if index.Browser != nil {
        logger.Debug("HAR browser", "name", index.Browser.Name, "version", index.Browser.Version)
    }

    return streamer, nil
}
