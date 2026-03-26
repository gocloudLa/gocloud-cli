package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile   string
	verbose   bool
	debug     bool
	version   string
	buildTime string
	gitCommit string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gocloud",
	Short: "CLI tool for managing GoCloud infrastructure projects",
	Long: `GoCloud CLI is a tool for managing Terraform/Terragrunt 
infrastructure projects across multiple clients and environments.`,
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersionInfo sets version information
func SetVersionInfo(v, bt, gc string) {
	version = v
	buildTime = bt
	gitCommit = gc
	rootCmd.Version = fmt.Sprintf("%s (build: %s, commit: %s)", version, buildTime, gitCommit)
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (when set; many commands default to gocloud.yaml in current directory)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug mode")

	// Bind flags to viper
	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to bind verbose flag: %v\n", err)
	}
	if err := viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug")); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to bind debug flag: %v\n", err)
	}

	// Add subcommands
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(ssoCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(moduleCmd)
	rootCmd.AddCommand(versionCmd)

	// Add completion command
	rootCmd.AddCommand(completionCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".gocloud" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gocloud")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:
  $ source <(gocloud completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ gocloud completion bash > /etc/bash_completion.d/gocloud
  # macOS:
  $ gocloud completion bash > /usr/local/etc/bash_completion.d/gocloud

Zsh:
  $ source <(gocloud completion zsh)

  # To load completions for each session, execute once:
  $ gocloud completion zsh > "${fpath[1]}/_gocloud"

Fish:
  $ gocloud completion fish | source

  # To load completions for each session, execute once:
  $ gocloud completion fish > ~/.config/fish/completions/gocloud.fish

PowerShell:
  PS> gocloud completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, execute once:
  PS> gocloud completion powershell > gocloud.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		switch args[0] {
		case "bash":
			err = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			err = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating completion: %v\n", err)
			os.Exit(1)
		}
	},
}
