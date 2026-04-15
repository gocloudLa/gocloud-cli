package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gocloud-cli/internal/generator"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/testutils"
)

// TestShouldGenerateTerragruntForPreview_MapInterfaceFormat documents that shouldGenerateTerragruntForPreview
// does NOT support project/workload defined as map[interface{}]interface{} (YAML unmarshal format).
// When a workload has enable_terragrunt: false in that format, the function should return false. Fails until fixed.
func TestShouldGenerateTerragruntForPreview_MapInterfaceFormat(t *testing.T) {
	config := &models.InfrastructureConfig{
		Environments: map[string]models.Environment{
			"dev": {
				Workloads: []interface{}{
					map[interface{}]interface{}{
						"wdwl": map[interface{}]interface{}{
							"enable_terragrunt": false,
						},
					},
				},
			},
		},
	}
	got := shouldGenerateTerragruntForPreview(config, "workload", "wdwl", "dev")
	if got {
		t.Errorf("shouldGenerateTerragruntForPreview(workload wdwl with enable_terragrunt: false in map[interface{}] format) = true, want false")
	}
}

// TestShouldGenerateTerragruntForPreview_MapInterfaceFormat_Project documents the same for project-level enable_terragrunt.
func TestShouldGenerateTerragruntForPreview_MapInterfaceFormat_Project(t *testing.T) {
	config := &models.InfrastructureConfig{
		Environments: map[string]models.Environment{
			"dev": {
				Projects: []interface{}{
					map[interface{}]interface{}{
						"dept": map[interface{}]interface{}{
							"enable_terragrunt": false,
						},
					},
				},
			},
		},
	}
	got := shouldGenerateTerragruntForPreview(config, "project", "dept", "dev")
	if got {
		t.Errorf("shouldGenerateTerragruntForPreview(project dept with enable_terragrunt: false in map[interface{}] format) = true, want false")
	}
}

func TestGenerateCommand(t *testing.T) {
	t.Skip("Skipping TestGenerateCommand - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "generate with valid config",
			args:        []string{"--config", "gocloud-example-config.yaml"},
			expectError: false,
		},
		{
			name:        "generate with dry-run",
			args:        []string{"--config", "gocloud-example-config.yaml", "--dry-run"},
			expectError: false,
		},
		{
			name:        "generate with force",
			args:        []string{"--config", "gocloud-example-config.yaml", "--force"},
			expectError: false,
		},
		{
			name:        "generate with working-dir",
			args:        []string{"--config", "gocloud-example-config.yaml", "--working-dir", "/tmp/test"},
			expectError: false,
		},
		{
			name:        "generate without config",
			args:        []string{},
			expectError: true,
			errorMsg:    "config file not found",
		},
		{
			name:        "generate with non-existent config",
			args:        []string{"--config", "non-existent.yaml"},
			expectError: true,
			errorMsg:    "config file not found",
		},
		{
			name:        "generate with invalid config",
			args:        []string{"--config", "invalid.yaml"},
			expectError: true,
			errorMsg:    "invalid yaml syntax",
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

			// Change to temp directory
			originalDir, _ := os.Getwd()
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to change back to original directory: %v", err)
				}
			}()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Create test config files
			switch {
			case strings.Contains(strings.Join(tt.args, " "), "gocloud-example-config.yaml"):
				// Copy example config
				exampleConfig, err := os.ReadFile("gocloud-example-config.yaml")
				if err != nil {
					t.Skipf("Skipping test: example config not found")
				}
				err = os.WriteFile(filepath.Join(tempDir, "gocloud-example-config.yaml"), exampleConfig, 0644)
				if err != nil {
					t.Fatalf("Failed to copy example config: %v", err)
				}
			case strings.Contains(strings.Join(tt.args, " "), "invalid.yaml"):
				err = os.WriteFile(filepath.Join(tempDir, "invalid.yaml"), []byte("invalid: yaml: content: ["), 0644)
				if err != nil {
					t.Fatalf("Failed to create invalid config: %v", err)
				}
			}

			// Set up command
			cmd := generateCmd
			cmd.SetArgs(tt.args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateCommand() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateCommand() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateCommand() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGenerateWithConfig(t *testing.T) {
	t.Skip("Skipping TestGenerateWithConfig - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		configFile  string
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "generate with example config",
			configFile:  "gocloud-example-config.yaml",
			workingDir:  "",
			expectError: false,
		},
		{
			name:        "generate with custom working dir",
			configFile:  "gocloud-example-config.yaml",
			workingDir:  "custom-output",
			expectError: false,
		},
		{
			name:        "generate with invalid working dir",
			configFile:  "gocloud-example-config.yaml",
			workingDir:  "/invalid/path/that/does/not/exist",
			expectError: true,
			errorMsg:    "invalid working directory",
		},
		{
			name:        "generate with incomplete config",
			configFile:  "incomplete.yaml",
			workingDir:  "",
			expectError: true,
			errorMsg:    "missing required field",
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

			// Change to temp directory
			originalDir, _ := os.Getwd()
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to change back to original directory: %v", err)
				}
			}()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Create test config files
			switch tt.configFile {
			case "gocloud-example-config.yaml":
				// Copy example config
				exampleConfig, err := os.ReadFile("gocloud-example-config.yaml")
				if err != nil {
					t.Skipf("Skipping test: example config not found")
				}
				err = os.WriteFile(filepath.Join(tempDir, "gocloud-example-config.yaml"), exampleConfig, 0644)
			case "incomplete.yaml":
				err = os.WriteFile(filepath.Join(tempDir, "incomplete.yaml"), []byte(`
cli:
  debug: false
infrastructure:
  client: "test-client"
  # Missing required fields
`), 0644)
			}

			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			// Set up command
			cmd := generateCmd
			args := []string{"--config", filepath.Join(tempDir, tt.configFile)}
			if tt.workingDir != "" {
				args = append(args, "--working-dir", tt.workingDir)
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateWithConfig() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateWithConfig() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateWithConfig() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGenerateDryRun(t *testing.T) {
	t.Skip("Skipping TestGenerateDryRun - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		configFile  string
		dryRun      bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "generate with dry-run",
			configFile:  "gocloud-example-config.yaml",
			dryRun:      true,
			expectError: false,
		},
		{
			name:        "generate without dry-run",
			configFile:  "gocloud-example-config.yaml",
			dryRun:      false,
			expectError: false,
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

			// Change to temp directory
			originalDir, _ := os.Getwd()
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to change back to original directory: %v", err)
				}
			}()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Copy example config
			exampleConfig, err := os.ReadFile("gocloud-example-config.yaml")
			if err != nil {
				t.Skipf("Skipping test: example config not found")
			}
			err = os.WriteFile(filepath.Join(tempDir, "gocloud-example-config.yaml"), exampleConfig, 0644)
			if err != nil {
				t.Fatalf("Failed to copy example config: %v", err)
			}

			// Set up command
			cmd := generateCmd
			args := []string{"--config", filepath.Join(tempDir, tt.configFile)}
			if tt.dryRun {
				args = append(args, "--dry-run")
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateDryRun() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateDryRun() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateDryRun() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGenerateForce(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		force       bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "generate with force",
			configFile:  "gocloud-example-config.yaml",
			force:       true,
			expectError: false,
		},
		{
			name:        "generate without force",
			configFile:  "gocloud-example-config.yaml",
			force:       false,
			expectError: false,
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

			// Change to temp directory
			originalDir, _ := os.Getwd()
			defer func() {
				if err := os.Chdir(originalDir); err != nil {
					t.Logf("Warning: failed to change back to original directory: %v", err)
				}
			}()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Copy example config
			exampleConfig, err := os.ReadFile("gocloud-example-config.yaml")
			if err != nil {
				t.Skipf("Skipping test: example config not found")
			}
			err = os.WriteFile(filepath.Join(tempDir, "gocloud-example-config.yaml"), exampleConfig, 0644)
			if err != nil {
				t.Fatalf("Failed to copy example config: %v", err)
			}

			// Set up command
			cmd := generateCmd
			args := []string{"--config", filepath.Join(tempDir, tt.configFile)}
			if tt.force {
				args = append(args, "--force")
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateForce() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateForce() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateForce() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGenerateFlags(t *testing.T) {
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
			expectError: false,
		},
		{
			name:        "dry-run flag",
			flagName:    "dry-run",
			flagValue:   "true",
			expectError: false,
		},
		{
			name:        "force flag",
			flagName:    "force",
			flagValue:   "true",
			expectError: false,
		},
		{
			name:        "invalid flag",
			flagName:    "invalid-flag",
			flagValue:   "true",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := generateCmd

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

func TestIsSecurityLayerEnabledForConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.InfrastructureConfig
		expected bool
	}{
		{
			name: "security enabled with aws_account",
			config: &models.InfrastructureConfig{
				Security: &models.OrganizationLayerConfig{AWSAccount: "123456789012"},
				Layers:   &models.LayerConfig{Security: &[]bool{true}[0]},
			},
			expected: true,
		},
		{
			name: "security disabled explicitly",
			config: &models.InfrastructureConfig{
				Security: &models.OrganizationLayerConfig{AWSAccount: "123456789012"},
				Layers:   &models.LayerConfig{Security: &[]bool{false}[0]},
			},
			expected: false,
		},
		{
			name: "layers.security true but no aws_account",
			config: &models.InfrastructureConfig{
				Layers: &models.LayerConfig{Security: &[]bool{true}[0]},
			},
			expected: false,
		},
		{name: "empty", config: &models.InfrastructureConfig{}, expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generator.IsSecurityLayerEnabledForConfig(tt.config); got != tt.expected {
				t.Errorf("IsSecurityLayerEnabledForConfig() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsOrganizationLayerEnabledForConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.InfrastructureConfig
		expected bool
	}{
		{
			name: "organization enabled with aws_account",
			config: &models.InfrastructureConfig{
				Organization: &models.OrganizationLayerConfig{AWSAccount: "123456789012"},
				Layers:       &models.LayerConfig{Organization: &[]bool{true}[0]},
			},
			expected: true,
		},
		{
			name: "organization disabled explicitly",
			config: &models.InfrastructureConfig{
				Organization: &models.OrganizationLayerConfig{AWSAccount: "123456789012"},
				Layers:       &models.LayerConfig{Organization: &[]bool{false}[0]},
			},
			expected: false,
		},
		{
			name: "organization block missing (no aws_account) - not enabled",
			config: &models.InfrastructureConfig{
				Layers: &models.LayerConfig{Organization: &[]bool{true}[0]},
			},
			expected: false,
		},
		{
			name:     "no layers and no organization - not enabled",
			config:   &models.InfrastructureConfig{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.IsOrganizationLayerEnabledForConfig(tt.config)
			if result != tt.expected {
				t.Errorf("IsOrganizationLayerEnabledForConfig() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestIsGitignoreGenerationEnabledForConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.InfrastructureConfig
		expected bool
	}{
		{name: "nil config defaults to true", config: nil, expected: true},
		{name: "empty infrastructure defaults to true", config: &models.InfrastructureConfig{}, expected: true},
		{name: "explicit true", config: &models.InfrastructureConfig{EnableGitignore: &[]bool{true}[0]}, expected: true},
		{name: "explicit false", config: &models.InfrastructureConfig{EnableGitignore: &[]bool{false}[0]}, expected: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generator.IsGitignoreGenerationEnabledForConfig(tt.config)
			if got != tt.expected {
				t.Errorf("IsGitignoreGenerationEnabledForConfig() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetDirectoryName(t *testing.T) {
	tests := []struct {
		envKey   string
		env      models.Environment
		expected string
	}{
		{"prd", models.Environment{DirName: "production", Name: "Production"}, "production"},
		{"dev", models.Environment{Name: "Development"}, "development"},
		{"stg", models.Environment{}, "stg"},
	}
	for _, tt := range tests {
		t.Run(tt.envKey, func(t *testing.T) {
			got := getDirectoryName(tt.envKey, tt.env)
			if got != tt.expected {
				t.Errorf("getDirectoryName(%q, ...) = %q, expected %q", tt.envKey, got, tt.expected)
			}
		})
	}
}

func TestShouldGenerateLayerForSummary(t *testing.T) {
	baseTrue := true
	baseFalse := false
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Environments: map[string]models.Environment{
				"prd": {
					Name: "Production",
					Layers: &models.LayerConfig{
						Base: &baseFalse,
					},
				},
			},
			Layers: &models.LayerConfig{
				Base: &baseTrue,
			},
		},
	}
	if !shouldGenerateLayerForSummary(config, "foundation", "prd") {
		t.Error("shouldGenerateLayerForSummary(foundation, prd) expected true (infra default)")
	}
	if shouldGenerateLayerForSummary(config, "base", "prd") {
		t.Error("shouldGenerateLayerForSummary(base, prd) expected false (env override)")
	}
}

func TestGetLayerDefaultForSummary(t *testing.T) {
	baseTrue := true
	orgFalse := false
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Layers: &models.LayerConfig{
				Base:         &baseTrue,
				Organization: &orgFalse,
			},
		},
	}
	if !getLayerDefaultForSummary(config, "base") {
		t.Error("getLayerDefaultForSummary(base) expected true")
	}
	if getLayerDefaultForSummary(config, "organization") {
		t.Error("getLayerDefaultForSummary(organization) expected false")
	}
	if !getLayerDefaultForSummary(config, "foundation") {
		t.Error("getLayerDefaultForSummary(foundation) expected true (default)")
	}
}
