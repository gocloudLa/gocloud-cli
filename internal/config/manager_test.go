package config

import (
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/testutils"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "create new manager",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()

			if manager == nil {
				t.Errorf("NewManager() returned nil")
			}
		})
	}
}

func TestGetDefaultConfigPath(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid project name",
			projectName: "test-project",
			expectError: false,
		},
		{
			name:        "empty project name",
			projectName: "",
			expectError: true,
			errorMsg:    "project name is required",
		},
		{
			name:        "project name with spaces",
			projectName: "test project",
			expectError: true,
			errorMsg:    "project name cannot contain spaces",
		},
		{
			name:        "project name with special characters",
			projectName: "test@project",
			expectError: true,
			errorMsg:    "project name cannot contain special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()

			path, err := manager.GetDefaultConfigPath(tt.projectName)

			if tt.expectError {
				if err == nil {
					t.Errorf("GetDefaultConfigPath() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetDefaultConfigPath() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetDefaultConfigPath() expected no error but got: %v", err)
				}
				if path == "" {
					t.Errorf("GetDefaultConfigPath() expected non-empty path but got empty")
				}

				// Verify path format
				expectedPath := filepath.Join(tt.projectName, "gocloud.yaml")
				if path != expectedPath {
					t.Errorf("GetDefaultConfigPath() = %s, expected %s", path, expectedPath)
				}
			}
		})
	}
}

func TestGetCurrentDirConfigPath(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "get current dir config path",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()

			path := manager.GetCurrentDirConfigPath()

			if path == "" {
				t.Errorf("GetCurrentDirConfigPath() returned empty path")
			}

			// Verify path format
			expectedPath := "gocloud.yaml"
			if path != expectedPath {
				t.Errorf("GetCurrentDirConfigPath() = %s, expected %s", path, expectedPath)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "load valid config",
			configFile:  "valid-config.yaml",
			expectError: false,
		},
		{
			name:        "load non-existent config",
			configFile:  "non-existent.yaml",
			expectError: true,
			errorMsg:    "config file not found",
		},
		{
			name:        "load invalid yaml config",
			configFile:  "invalid.yaml",
			expectError: true,
			errorMsg:    "invalid yaml syntax",
		},
		{
			name:        "load empty config",
			configFile:  "empty.yaml",
			expectError: true,
			errorMsg:    "config file is empty",
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
			switch tt.configFile {
			case "valid-config.yaml":
				err = os.WriteFile(filepath.Join(tempDir, "valid-config.yaml"), []byte(`
cli:
  debug: false
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
  environments:
    dev:
      name: "Development"
      dir_name: "dev"
      aws_account: "123456789012"
`), 0644)
			case "invalid.yaml":
				err = os.WriteFile(filepath.Join(tempDir, "invalid.yaml"), []byte("invalid: yaml: content: ["), 0644)
			case "empty.yaml":
				err = os.WriteFile(filepath.Join(tempDir, "empty.yaml"), []byte(""), 0644)
			}

			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			manager := NewManager()

			config, err := manager.LoadConfig(filepath.Join(tempDir, tt.configFile))

			if tt.expectError {
				if err == nil {
					t.Errorf("LoadConfig() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("LoadConfig() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("LoadConfig() expected no error but got: %v", err)
				}
				if config == nil {
					t.Errorf("LoadConfig() returned nil config")
				}
			}
		})
	}
}

// TestLoadConfig_OrganizationSecrets ensures that infrastructure.organization.secrets
// is loaded from YAML and is used when resolving secrets config for the organization layer.
func TestLoadConfig_OrganizationSecrets(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gocloud.yaml")
	yamlContent := `
cli:
  working_dir: "."
infrastructure:
  client: test-client
  company: gcl
  region: us-east-1
  environments: {}
  enable_secrets: true
  layers:
    base: false
    foundation: false
    organization: true
  secrets:
    type: ssm
  organization:
    secrets:
      type: sops
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	manager := NewManager()
	config, err := manager.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if config == nil || config.Infrastructure == nil {
		t.Fatal("LoadConfig() returned nil config or infrastructure")
	}
	if config.Infrastructure.Organization == nil {
		t.Fatal("LoadConfig(): infrastructure.organization should be set when present in YAML")
	}
	if config.Infrastructure.Organization.Secrets == nil {
		t.Fatal("LoadConfig(): infrastructure.organization.secrets should be set when present in YAML")
	}
	if config.Infrastructure.Organization.Secrets.Type != "sops" {
		t.Errorf("LoadConfig(): infrastructure.organization.secrets.type = %q, want sops",
			config.Infrastructure.Organization.Secrets.Type)
	}
	// ResolveSecretsConfig for organization layer must use the override
	resolved := config.Infrastructure.ResolveSecretsConfig("organization", "", "org")
	if resolved == nil {
		t.Fatal("ResolveSecretsConfig(organization) returned nil")
	}
	if resolved.Type != "sops" {
		t.Errorf("ResolveSecretsConfig(organization).Type = %q, want sops", resolved.Type)
	}
}

func TestSaveConfig(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "save valid config",
			configFile:  "test-config.yaml",
			expectError: false,
		},
		{
			name:        "save config to invalid path",
			configFile:  "/invalid/path/config.yaml",
			expectError: true,
			errorMsg:    "failed to save config",
		},
		{
			name:        "save config to read-only directory",
			configFile:  "/root/config.yaml",
			expectError: true,
			errorMsg:    "failed to save config",
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

			manager := NewManager()

			// Create a test config
			config := &models.Config{
				Infrastructure: &models.InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
				},
			}

			var configPath string
			if tt.configFile == "test-config.yaml" {
				configPath = filepath.Join(tempDir, "test-config.yaml")
			} else {
				configPath = tt.configFile
			}

			err = manager.SaveConfig(config, configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("SaveConfig() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SaveConfig() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SaveConfig() expected no error but got: %v", err)
				}

				// Verify file was created
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					t.Errorf("SaveConfig() did not create config file at %s", configPath)
				}
			}
		})
	}
}

func TestValidateConfigFile(t *testing.T) {
	tests := []struct {
		name        string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "validate valid config",
			configFile:  "valid-config.yaml",
			expectError: false,
		},
		{
			name:        "validate config with missing required fields",
			configFile:  "incomplete-config.yaml",
			expectError: true,
			errorMsg:    "infrastructure.company is required",
		},
		{
			name:        "validate config with invalid values",
			configFile:  "invalid-values.yaml",
			expectError: true,
			errorMsg:    "infrastructure.client is required",
		},
		{
			name:        "validate non-existent config",
			configFile:  "non-existent.yaml",
			expectError: true,
			errorMsg:    "failed to read config file",
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
			switch tt.configFile {
			case "valid-config.yaml":
				err = os.WriteFile(filepath.Join(tempDir, "valid-config.yaml"), []byte(`
cli:
  debug: false
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
  environments:
    dev:
      name: "Development"
      dir_name: "dev"
      aws_account: "123456789012"
`), 0644)
			case "incomplete-config.yaml":
				err = os.WriteFile(filepath.Join(tempDir, "incomplete-config.yaml"), []byte(`
cli:
  debug: false
infrastructure:
  client: "test-client"
  # Missing required fields
`), 0644)
			case "invalid-values.yaml":
				err = os.WriteFile(filepath.Join(tempDir, "invalid-values.yaml"), []byte(`
cli:
  debug: false
infrastructure:
  client: ""
  company: "invalid@company"
  region: "invalid-region"
  version: "v1.0.0"
  environments:
    dev:
      name: "Development"
      dir_name: "dev"
      aws_account: "12345678901"
`), 0644)
			}

			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			manager := NewManager()

			err = manager.ValidateConfigFile(filepath.Join(tempDir, tt.configFile))

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateConfigFile() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateConfigFile() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateConfigFile() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "exists.yaml")
	if err := os.WriteFile(existingFile, []byte("infrastructure:\n  company: gcl"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	manager := NewManager()
	if !manager.ConfigExists(existingFile) {
		t.Error("ConfigExists(existing file) expected true")
	}
	if manager.ConfigExists(filepath.Join(tmpDir, "nonexistent.yaml")) {
		t.Error("ConfigExists(nonexistent) expected false")
	}
}

// Test config struct for testing
type TestConfig struct {
	Client  string `yaml:"client"`
	Company string `yaml:"company"`
	Region  string `yaml:"region"`
	Version string `yaml:"version"`
}
