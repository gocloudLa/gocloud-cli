package cmd

import (
	"fmt"
	"os"
	"runtime"

	versionpkg "gocloud-cli/internal/version"

	"github.com/spf13/cobra"
)

const cliReleasesURL = "https://github.com/gocloudLa/gocloud-cli/releases"

// versionCmd shows the current CLI version; use subcommands check and update for GitHub release logic.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version",
	Long: `Show the GoCloud CLI version (build time and git commit).

Use "gocloud version check" to compare with the latest release on GitHub.
Use "gocloud version update" to replace this binary from the latest release (macOS/Linux when a matching asset exists).`,
	RunE: runVersionShow,
}

var versionCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for a newer release on GitHub",
	RunE:  runVersionCheck,
}

var versionUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update this binary from the latest GitHub release (macOS/Linux)",
	Long: `Compares with the latest release on GitHub and, if you are outdated and a matching
asset exists, replaces this binary. On Windows, install or upgrade using the official
documentation and releases page.`,
	RunE: runVersionUpdate,
}

func init() {
	versionCmd.AddCommand(versionCheckCmd)
	versionCmd.AddCommand(versionUpdateCmd)
}

func runVersionShow(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stdout, cmd.Root().Version)
	return nil
}

func runVersionCheck(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stdout, cmd.Root().Version)

	result, err := versionpkg.Check(version)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	if result.IsUpToDate {
		fmt.Fprintln(os.Stdout, "You are on the latest release.")
		return nil
	}

	fmt.Fprintf(os.Stdout, "A new version is available: %s (you have %s).\n", result.LatestVersion, result.CurrentVersion)

	if result.DownloadURL == "" {
		fmt.Fprintf(os.Stderr, "No pre-built binary for this platform in that release. See %s\n", cliReleasesURL)
		return nil
	}

	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stdout, "Automatic update from the CLI is not available on Windows. See %s for installation instructions.\n", cliReleasesURL)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Run '%s version update' to update.\n", cmd.Root().Name())
	return nil
}

func runVersionUpdate(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stdout, cmd.Root().Version)

	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "Automatic update is not supported on Windows. See %s for installation instructions.\n", cliReleasesURL)
		return nil
	}

	result, err := versionpkg.Check(version)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	if result.IsUpToDate {
		fmt.Fprintln(os.Stdout, "You are already on the latest release.")
		return nil
	}

	if result.DownloadURL == "" {
		fmt.Fprintf(os.Stderr, "No pre-built binary for this platform in that release. See %s\n", cliReleasesURL)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Updating to %s...\n", result.LatestVersion)
	if err := versionpkg.DownloadAndReplace(result.DownloadURL); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed. See %s for installation instructions.\n", cliReleasesURL)
		return fmt.Errorf("update: %w", err)
	}
	fmt.Fprintln(os.Stdout, "Update complete. Run 'gocloud version' to verify.")
	return nil
}
