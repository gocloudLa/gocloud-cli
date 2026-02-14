package cmd

import (
	"fmt"
	"os"
	"runtime"

	versionpkg "gocloud-cli/internal/version"

	"github.com/spf13/cobra"
)

var (
	versionCheck  bool
	versionUpdate bool
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and optionally check for updates",
	Long: `Show the current GoCloud CLI version.

With --check, compares against the latest release on GitHub and reports if an update
is available. If outdated, you can use --update to attempt an automatic update
(replaces the current binary; on Windows manual update is recommended), or run the
printed manual commands.`,
	RunE: runVersion,
}

func init() {
	versionCmd.Flags().BoolVar(&versionCheck, "check", false, "check for latest version on GitHub")
	versionCmd.Flags().BoolVar(&versionUpdate, "update", false, "if outdated, attempt to update the binary automatically (Unix only)")
}

func runVersion(cmd *cobra.Command, _ []string) error {
	// Always show current version
	fmt.Fprintln(os.Stdout, cmd.Root().Version)

	if !versionCheck {
		return nil
	}

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
		fmt.Fprintln(os.Stderr, "No pre-built binary for your platform in this release. Build from source: go install github.com/gocloudLa/gocloud-cli@latest")
		return nil
	}

	// Manual update commands (all platforms)
	printManualUpdateCommands(result)

	if versionUpdate {
		if runtime.GOOS == "windows" {
			fmt.Fprintln(os.Stderr, "Automatic update is not supported on Windows. Use the commands above to update.")
			return nil
		}
		fmt.Fprintln(os.Stdout, "Updating...")
		if err := versionpkg.DownloadAndReplace(result.DownloadURL); err != nil {
			return fmt.Errorf("auto-update failed: %w", err)
		}
		fmt.Fprintln(os.Stdout, "Update complete. Run 'gocloud version' to verify.")
	}

	return nil
}

func printManualUpdateCommands(result *versionpkg.CheckResult) {
	const repo = "https://github.com/gocloudLa/gocloud-cli/releases"
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "To update manually, run:")
	fmt.Fprintln(os.Stdout, "")

	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stdout, "  curl -sL -o gocloud.exe %s\n", result.DownloadURL)
		fmt.Fprintln(os.Stdout, "  # Then move gocloud.exe to your PATH (e.g. replace existing gocloud.exe)")
	} else {
		fmt.Fprintf(os.Stdout, "  curl -sL -o gocloud \"%s\"\n", result.DownloadURL)
		fmt.Fprintln(os.Stdout, "  chmod +x gocloud")
		fmt.Fprintln(os.Stdout, "  sudo mv gocloud /usr/local/bin/   # or another directory in your PATH)")
	}
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintf(os.Stdout, "Or see all releases: %s\n", repo)
}
