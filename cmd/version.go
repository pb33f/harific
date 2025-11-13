package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// These variables are typically set during build time using ldflags
// Example: go build -ldflags "-X github.com/pb33f/braid/cmd.Version=1.0.0"
var (
	Version   = "dev"       // Version of the application
	GitCommit = "unknown"   // Git commit hash
	BuildDate = "unknown"   // Build date
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display detailed version information about the Braid application.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("Braid - HAR File Viewer and Mock Server\n")
	fmt.Printf("Version:    %s\n", Version)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Build Date: %s\n", BuildDate)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}