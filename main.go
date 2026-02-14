package main

import (
	"fmt"
	"os"

	"gocloud-cli/cmd"
	"gocloud-cli/internal/logger"
)

var (
	Version   = "1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Initialize logger
	logger.Init()

	// Set version information
	cmd.SetVersionInfo(Version, BuildTime, GitCommit)

	// Execute root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
