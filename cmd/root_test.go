package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "help command",
			args:        []string{"gocloud", "--help"},
			expectError: false,
		},
		{
			name:        "version command",
			args:        []string{"gocloud", "--version"},
			expectError: false,
		},
		{
			name:        "invalid command",
			args:        []string{"gocloud", "invalid-command"},
			expectError: true,
			errorMsg:    "unknown command",
		},
		{
			name:        "no args",
			args:        []string{"gocloud"},
			expectError: false, // Should show help
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			originalArgs := os.Args
			defer func() {
				os.Args = originalArgs
			}()

			// Set test args
			os.Args = tt.args

			// Execute command
			err := Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Execute() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Execute() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Execute() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSetVersionInfo(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		buildTime   string
		gitCommit   string
		expectError bool
	}{
		{
			name:        "valid version info",
			version:     "1.0.0",
			buildTime:   "2023-01-01T00:00:00Z",
			gitCommit:   "abc123",
			expectError: false,
		},
		{
			name:        "empty version info",
			version:     "",
			buildTime:   "",
			gitCommit:   "",
			expectError: false, // Should handle empty values gracefully
		},
		{
			name:        "special characters in version",
			version:     "1.0.0-beta.1",
			buildTime:   "2023-01-01T00:00:00Z",
			gitCommit:   "abc123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic or return error
			SetVersionInfo(tt.version, tt.buildTime, tt.gitCommit)

			// Verify version info was set (if we had access to internal state)
			// For now, just ensure no panic occurs
		})
	}
}

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "root command with help flag",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "root command with version flag",
			args:        []string{"--version"},
			expectError: false,
		},
		{
			name:        "root command with debug flag",
			args:        []string{"--debug"},
			expectError: false,
		},
		{
			name:        "root command with verbose flag",
			args:        []string{"--verbose"},
			expectError: false,
		},
		{
			name:        "root command with invalid flag",
			args:        []string{"--invalid-flag"},
			expectError: true,
			errorMsg:    "unknown flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new root command for each test
			cmd := rootCmd

			// Set args
			cmd.SetArgs(tt.args)

			// Execute command
			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("RootCommand() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("RootCommand() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("RootCommand() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestRootCommandFlags(t *testing.T) {
	tests := []struct {
		name        string
		flagName    string
		flagValue   string
		expectError bool
	}{
		{
			name:        "config flag",
			flagName:    "config",
			flagValue:   "test-config.yaml",
			expectError: false,
		},
		{
			name:        "working-dir flag",
			flagName:    "working-dir",
			flagValue:   "/tmp/test",
			expectError: true,
		},
		{
			name:        "debug flag",
			flagName:    "debug",
			flagValue:   "true",
			expectError: false,
		},
		{
			name:        "verbose flag",
			flagName:    "verbose",
			flagValue:   "true",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := rootCmd

			// Set flag value
			err := cmd.Flags().Set(tt.flagName, tt.flagValue)

			if tt.expectError {
				if err == nil {
					t.Errorf("SetFlag(%s) expected error but got nil", tt.flagName)
				}
			} else {
				if err != nil {
					t.Errorf("SetFlag(%s) expected no error but got: %v", tt.flagName, err)
				}
			}
		})
	}
}

func TestRootCommandSubcommands(t *testing.T) {
	tests := []struct {
		name        string
		subcommand  string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "config subcommand",
			subcommand:  "config",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "generate subcommand",
			subcommand:  "generate",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "secrets subcommand",
			subcommand:  "secrets",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "sso subcommand",
			subcommand:  "sso",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "module subcommand",
			subcommand:  "module",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "completion subcommand",
			subcommand:  "completion",
			args:        []string{"--help"},
			expectError: false,
		},
		{
			name:        "invalid subcommand",
			subcommand:  "invalid",
			args:        []string{"--help"},
			expectError: true,
			errorMsg:    "unknown command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := rootCmd

			// Set args with subcommand
			args := append([]string{tt.subcommand}, tt.args...)
			cmd.SetArgs(args)

			// Execute command
			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Subcommand(%s) expected error but got nil", tt.subcommand)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Subcommand(%s) error message '%s' does not contain '%s'", tt.subcommand, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Subcommand(%s) expected no error but got: %v", tt.subcommand, err)
				}
			}
		})
	}
}
