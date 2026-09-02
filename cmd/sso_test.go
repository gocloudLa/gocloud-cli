package cmd

import (
	"gocloud-cli/internal/testutils"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// chdirTemp changes the working directory to dir for the duration of the test, restoring the
// original directory via t.Cleanup. Needed because runSSOSetup/runSSOList/runSSOVerify operate
// on ".aws/config" and "gocloud.yaml" relative to the current working directory.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%s): %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Logf("Warning: failed to change back to original directory: %v", err)
		}
	})
}

// runRootCmd runs rootCmd (not a bare subcommand) with args, resetting cfgFile first.
//
// cobra's Command.ExecuteC() always redirects execution to c.Root() when the command has a
// parent ("Regardless of what command execute is called on, run on Root only"), and Root()
// uses its OWN .args field - not the subcommand's. So calling e.g. ssoSetupCmd.SetArgs(args)
// followed by ssoSetupCmd.Execute() silently ignores those args and instead parses whatever
// rootCmd.args currently holds. This is almost certainly why these tests were originally
// skipped as "commands not executing correctly in test environment": routing through rootCmd
// with a full argv (e.g. []string{"sso", "setup", "--config", path}) is required.
func runRootCmd(args []string) error {
	cfgFile = ""
	ssoProviderFlag = ""
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func TestSSOSetup(t *testing.T) {
	// ssoConfigYAML is a minimal, self-contained gocloud.yaml. It must include an aws_sso block:
	// generateAWSConfig dereferences config.Infrastructure.AWSSSO unconditionally for every
	// environment with SSO enabled (the default), so a config without it panics - a pre-existing
	// behavior this task does not change.
	const ssoConfigYAML = `
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
  aws_sso:
    start_url: "https://example.awsapps.com/start"
    region: "us-east-1"
    role_name: "AdministratorAccess"
  environments:
    dev:
      name: "Development"
      dir_name: "dev"
      aws_account: "123456789012"
`

	tests := []struct {
		name          string
		writeConfig   bool
		configContent string
		configArg     string // if non-empty, passed as --config <configArg>
		expectError   bool
		errorMsg      string
	}{
		{
			name:          "setup SSO with valid config",
			writeConfig:   true,
			configContent: ssoConfigYAML,
			configArg:     "gocloud.yaml",
			expectError:   false,
		},
		{
			name:        "setup SSO without config",
			writeConfig: false,
			expectError: true,
			errorMsg:    "config file not found",
		},
		{
			name:        "setup SSO with non-existent config",
			configArg:   "non-existent.yaml",
			expectError: true,
			errorMsg:    "config file not found",
		},
		{
			name:          "setup SSO with invalid config",
			writeConfig:   true,
			configContent: "invalid: yaml: content: [",
			configArg:     "invalid.yaml",
			expectError:   true,
			errorMsg:      "invalid yaml syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()
			chdirTemp(t, tempDir)

			if tt.writeConfig {
				if err := os.WriteFile(filepath.Join(tempDir, tt.configArg), []byte(tt.configContent), 0644); err != nil {
					t.Fatalf("Failed to write test config file: %v", err)
				}
			}

			args := []string{"sso", "setup"}
			if tt.configArg != "" {
				args = append(args, "--config", tt.configArg)
			}
			err = runRootCmd(args)

			if tt.expectError {
				if err == nil {
					t.Errorf("SSOSetup() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SSOSetup() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("SSOSetup() expected no error but got: %v", err)
			}

			awsConfigContent, rerr := os.ReadFile(filepath.Join(tempDir, ".aws", "config"))
			if rerr != nil {
				t.Fatalf("Failed to read generated .aws/config: %v", rerr)
			}
			if !strings.Contains(string(awsConfigContent), "[profile test-client-dev]") {
				t.Errorf(".aws/config = %q, want it to contain profile 'test-client-dev'", awsConfigContent)
			}
		})
	}
}

func TestSSOSetup_OrgEnvironmentDoesNotDuplicateOrganizationProfile(t *testing.T) {
	const ssoConfigYAML = `
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
  aws_sso:
    start_url: "https://example.awsapps.com/start"
    region: "us-east-1"
    role_name: "AdministratorAccess"
  organization:
    aws_account: "111111111111"
  security:
    aws_account: "222222222222"
  environments:
    org:
      name: "Organization"
      aws_account: "111111111111"
      layers:
        base: true
        foundation: true
    sec:
      name: "Security"
      aws_account: "222222222222"
      layers:
        base: true
        foundation: true
    dev:
      name: "Development"
      aws_account: "123456789012"
`

	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := testutils.CleanupTempDir(tempDir); err != nil {
			t.Logf("Warning: failed to cleanup temp dir: %v", err)
		}
	}()
	chdirTemp(t, tempDir)

	if err := os.WriteFile(filepath.Join(tempDir, "gocloud.yaml"), []byte(ssoConfigYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	if err := runRootCmd([]string{"sso", "setup", "--config", "gocloud.yaml"}); err != nil {
		t.Fatalf("SSOSetup() expected no error but got: %v", err)
	}

	awsConfigContent, err := os.ReadFile(filepath.Join(tempDir, ".aws", "config"))
	if err != nil {
		t.Fatalf("Failed to read generated .aws/config: %v", err)
	}
	content := string(awsConfigContent)

	for _, profile := range []string{"test-client-org", "test-client-sec", "test-client-dev"} {
		count := strings.Count(content, "[profile "+profile+"]")
		if count != 1 {
			t.Errorf("profile %q appears %d times in .aws/config, want exactly 1", profile, count)
		}
	}
}

func TestSSOList(t *testing.T) {
	tests := []struct {
		name             string
		writeAWSConfig   bool
		awsConfigContent string
		expectError      bool
		errorMsg         string
	}{
		{
			name:             "list profiles with valid .aws/config",
			writeAWSConfig:   true,
			awsConfigContent: "[profile test-client-dev]\nsso_session = test-client-dev\nregion = us-east-1\n",
			expectError:      false,
		},
		{
			name:             "list profiles with empty .aws/config",
			writeAWSConfig:   true,
			awsConfigContent: "",
			expectError:      false,
		},
		{
			name:           "list profiles without .aws/config",
			writeAWSConfig: false,
			expectError:    true,
			errorMsg:       ".aws/config file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()
			chdirTemp(t, tempDir)

			if tt.writeAWSConfig {
				if err := os.MkdirAll(filepath.Join(tempDir, ".aws"), 0755); err != nil {
					t.Fatalf("Failed to create .aws dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(tempDir, ".aws", "config"), []byte(tt.awsConfigContent), 0644); err != nil {
					t.Fatalf("Failed to write .aws/config: %v", err)
				}
			}

			err = runRootCmd([]string{"sso", "list"})

			if tt.expectError {
				if err == nil {
					t.Errorf("SSOList() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SSOList() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("SSOList() expected no error but got: %v", err)
			}
		})
	}
}

func TestSSOLogin(t *testing.T) {
	t.Skip("Skipping TestSSOLogin - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		configFile  string
		profile     string
		allProfiles bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "login to specific profile",
			configFile:  "gocloud-example-config.yaml",
			profile:     "gcl-dev",
			allProfiles: false,
			expectError: false,
		},
		{
			name:        "login to all profiles",
			configFile:  "gocloud-example-config.yaml",
			profile:     "",
			allProfiles: true,
			expectError: false,
		},
		{
			name:        "login without profile or all flag",
			configFile:  "gocloud-example-config.yaml",
			profile:     "",
			allProfiles: false,
			expectError: true,
			errorMsg:    "profile or --all flag is required",
		},
		{
			name:        "login with non-existent profile",
			configFile:  "gocloud-example-config.yaml",
			profile:     "non-existent-profile",
			allProfiles: false,
			expectError: true,
			errorMsg:    "profile not found",
		},
		{
			name:        "login without config",
			configFile:  "",
			profile:     "gcl-dev",
			allProfiles: false,
			expectError: true,
			errorMsg:    "config file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create test config files
			if tt.configFile == "gocloud-example-config.yaml" {
				// Copy example config
				exampleConfig, err := os.ReadFile("gocloud-example-config.yaml")
				if err != nil {
					t.Skipf("Skipping test: example config not found")
				}
				err = os.WriteFile(filepath.Join(tempDir, "gocloud-example-config.yaml"), exampleConfig, 0644)
				if err != nil {
					t.Fatalf("Failed to copy example config: %v", err)
				}
			}

			// Set up command
			cmd := ssoLoginCmd
			args := []string{}
			if tt.configFile != "" {
				args = append(args, "--config", filepath.Join(tempDir, tt.configFile))
			}
			if tt.profile != "" {
				args = append(args, "--profile", tt.profile)
			}
			if tt.allProfiles {
				args = append(args, "--all")
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("SSOLogin() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SSOLogin() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SSOLogin() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSSOVerify(t *testing.T) {
	t.Skip("Skipping TestSSOVerify - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		configFile  string
		profile     string
		allProfiles bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "verify specific profile",
			configFile:  "gocloud-example-config.yaml",
			profile:     "gcl-dev",
			allProfiles: false,
			expectError: false,
		},
		{
			name:        "verify all profiles",
			configFile:  "gocloud-example-config.yaml",
			profile:     "",
			allProfiles: true,
			expectError: false,
		},
		{
			name:        "verify without profile or all flag",
			configFile:  "gocloud-example-config.yaml",
			profile:     "",
			allProfiles: false,
			expectError: true,
			errorMsg:    "profile or --all flag is required",
		},
		{
			name:        "verify with non-existent profile",
			configFile:  "gocloud-example-config.yaml",
			profile:     "non-existent-profile",
			allProfiles: false,
			expectError: true,
			errorMsg:    "profile not found",
		},
		{
			name:        "verify without config",
			configFile:  "",
			profile:     "gcl-dev",
			allProfiles: false,
			expectError: true,
			errorMsg:    "config file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create test config files
			if tt.configFile == "gocloud-example-config.yaml" {
				// Copy example config
				exampleConfig, err := os.ReadFile("gocloud-example-config.yaml")
				if err != nil {
					t.Skipf("Skipping test: example config not found")
				}
				err = os.WriteFile(filepath.Join(tempDir, "gocloud-example-config.yaml"), exampleConfig, 0644)
				if err != nil {
					t.Fatalf("Failed to copy example config: %v", err)
				}
			}

			// Set up command
			cmd := ssoVerifyCmd
			args := []string{}
			if tt.configFile != "" {
				args = append(args, "--config", filepath.Join(tempDir, tt.configFile))
			}
			if tt.profile != "" {
				args = append(args, "--profile", tt.profile)
			}
			if tt.allProfiles {
				args = append(args, "--all")
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("SSOVerify() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SSOVerify() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SSOVerify() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSSOCommands(t *testing.T) {
	t.Skip("Skipping TestSSOCommands - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "sso setup with valid config",
			command:     "setup",
			args:        []string{"--config", "gocloud-example-config.yaml"},
			expectError: false,
		},
		{
			name:        "sso list with valid config",
			command:     "list",
			args:        []string{"--config", "gocloud-example-config.yaml"},
			expectError: false,
		},
		{
			name:        "sso login with valid config",
			command:     "login",
			args:        []string{"--config", "gocloud-example-config.yaml", "--profile", "gcl-dev"},
			expectError: false,
		},
		{
			name:        "sso verify with valid config",
			command:     "verify",
			args:        []string{"--config", "gocloud-example-config.yaml", "--profile", "gcl-dev"},
			expectError: false,
		},
		{
			name:        "sso invalid command",
			command:     "invalid",
			args:        []string{},
			expectError: true,
			errorMsg:    "unknown command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Copy example config if needed
			if strings.Contains(strings.Join(tt.args, " "), "gocloud-example-config.yaml") {
				exampleConfig, err := os.ReadFile("gocloud-example-config.yaml")
				if err != nil {
					t.Skipf("Skipping test: example config not found")
				}
				err = os.WriteFile(filepath.Join(tempDir, "gocloud-example-config.yaml"), exampleConfig, 0644)
				if err != nil {
					t.Fatalf("Failed to copy example config: %v", err)
				}
			}

			// Set up command
			cmd := ssoCmd
			args := append([]string{tt.command}, tt.args...)
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("SSOCommand(%s) expected error but got nil", tt.command)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SSOCommand(%s) error message '%s' does not contain '%s'", tt.command, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SSOCommand(%s) expected no error but got: %v", tt.command, err)
				}
			}
		})
	}
}

func TestSSOFlags(t *testing.T) {
	tests := []struct {
		name        string
		flagName    string
		flagValue   string
		expectError bool
		cmd         *cobra.Command
	}{
		{
			name:        "profiles flag",
			flagName:    "profiles",
			flagValue:   "test-profile",
			expectError: false,
			cmd:         ssoLoginCmd,
		},
		{
			name:        "all flag",
			flagName:    "all",
			flagValue:   "true",
			expectError: false,
			cmd:         ssoLoginCmd,
		},
		{
			name:        "invalid flag",
			flagName:    "invalid-flag",
			flagValue:   "true",
			expectError: true,
			cmd:         ssoCmd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set flag value
			err := tt.cmd.Flags().Set(tt.flagName, tt.flagValue)

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

func TestIsValidProfile(t *testing.T) {
	profiles := []string{"client-prd", "client-dev", "client-stg"}
	tests := []struct {
		profile string
		want    bool
	}{
		{"client-prd", true},
		{"client-dev", true},
		{"client-stg", true},
		{"other", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			got := isValidProfile(tt.profile, profiles)
			if got != tt.want {
				t.Errorf("isValidProfile(%q) = %v, want %v", tt.profile, got, tt.want)
			}
		})
	}
}

func TestIsNumber(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"1", true},
		{"0", true},
		{"123", true},
		{"abc", false},
		{"", true},
		{"1a", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := isNumber(tt.s)
			if got != tt.want {
				t.Errorf("isNumber(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"1", 1},
		{"0", 0},
		{"42", 42},
		{"x", 0},
		{"", 0},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := parseInt(tt.s)
			if got != tt.want {
				t.Errorf("parseInt(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}
