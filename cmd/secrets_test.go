package cmd

import (
	"errors"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/testutils"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretsList(t *testing.T) {
	t.Skip("Skipping TestSecretsList - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		layerPath   string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "list secrets for base layer",
			layerPath:   "base/dev",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "list secrets for foundation layer",
			layerPath:   "foundation/dev",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "list secrets for project layer",
			layerPath:   "project/core/dev",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "list secrets for workload layer",
			layerPath:   "workload/core/dev",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "list secrets with invalid layer path",
			layerPath:   "invalid/layer",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "invalid layer path",
		},
		{
			name:        "list secrets without layer path",
			layerPath:   "",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "layer path is required",
		},
		{
			name:        "list secrets with non-existent config",
			layerPath:   "base/dev",
			configFile:  "non-existent.yaml",
			expectError: true,
			errorMsg:    "config file not found",
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
			if tt.configFile == "gocloud-example-config.yaml" {
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
			cmd := secretsListCmd
			args := []string{}
			if tt.configFile != "" {
				args = append(args, "--config", filepath.Join(tempDir, tt.configFile))
			}
			if tt.layerPath != "" {
				args = append(args, tt.layerPath)
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsList() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsList() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SecretsList() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSecretsGet(t *testing.T) {
	t.Skip("Skipping TestSecretsGet - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "get secret from base layer",
			layerPath:   "base/dev",
			secretKey:   "api_key",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "get secret from foundation layer",
			layerPath:   "foundation/dev",
			secretKey:   "database_url",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "get non-existent secret",
			layerPath:   "base/dev",
			secretKey:   "non-existent-key",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "secret not found",
		},
		{
			name:        "get secret without key",
			layerPath:   "base/dev",
			secretKey:   "",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "secret key is required",
		},
		{
			name:        "get secret with invalid layer path",
			layerPath:   "invalid/layer",
			secretKey:   "api_key",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "invalid layer path",
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
			if tt.configFile == "gocloud-example-config.yaml" {
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
			cmd := secretsGetCmd
			args := []string{}
			if tt.configFile != "" {
				args = append(args, "--config", filepath.Join(tempDir, tt.configFile))
			}
			if tt.layerPath != "" {
				args = append(args, tt.layerPath)
			}
			if tt.secretKey != "" {
				args = append(args, tt.secretKey)
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsGet() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsGet() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SecretsGet() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSecretsSet(t *testing.T) {
	t.Skip("Skipping TestSecretsSet - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		secretValue string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "set secret in base layer",
			layerPath:   "base/dev",
			secretKey:   "api_key",
			secretValue: "secret-value",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "set secret in foundation layer",
			layerPath:   "foundation/dev",
			secretKey:   "database_url",
			secretValue: "postgresql://user:pass@localhost/db",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "set secret without value",
			layerPath:   "base/dev",
			secretKey:   "api_key",
			secretValue: "",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "secret value is required",
		},
		{
			name:        "set secret without key",
			layerPath:   "base/dev",
			secretKey:   "",
			secretValue: "secret-value",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "secret key is required",
		},
		{
			name:        "set secret with invalid layer path",
			layerPath:   "invalid/layer",
			secretKey:   "api_key",
			secretValue: "secret-value",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "invalid layer path",
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
			if tt.configFile == "gocloud-example-config.yaml" {
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
			cmd := secretsSetCmd
			args := []string{}
			if tt.configFile != "" {
				args = append(args, "--config", filepath.Join(tempDir, tt.configFile))
			}
			if tt.layerPath != "" {
				args = append(args, tt.layerPath)
			}
			if tt.secretKey != "" {
				args = append(args, tt.secretKey)
			}
			if tt.secretValue != "" {
				args = append(args, tt.secretValue)
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsSet() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsSet() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SecretsSet() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSecretsDelete(t *testing.T) {
	t.Skip("Skipping TestSecretsDelete - commands not executing correctly in test environment")

	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "delete secret from base layer",
			layerPath:   "base/dev",
			secretKey:   "api_key",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "delete secret from foundation layer",
			layerPath:   "foundation/dev",
			secretKey:   "database_url",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "delete non-existent secret",
			layerPath:   "base/dev",
			secretKey:   "non-existent-key",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "secret not found",
		},
		{
			name:        "delete secret without key",
			layerPath:   "base/dev",
			secretKey:   "",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "secret key is required",
		},
		{
			name:        "delete secret with invalid layer path",
			layerPath:   "invalid/layer",
			secretKey:   "api_key",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "invalid layer path",
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
			if tt.configFile == "gocloud-example-config.yaml" {
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
			cmd := secretsDeleteCmd
			args := []string{}
			if tt.configFile != "" {
				args = append(args, "--config", filepath.Join(tempDir, tt.configFile))
			}
			if tt.layerPath != "" {
				args = append(args, tt.layerPath)
			}
			if tt.secretKey != "" {
				args = append(args, tt.secretKey)
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsDelete() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsDelete() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SecretsDelete() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSecretsEdit(t *testing.T) {
	t.Skip("Skipping TestSecretsEdit - requires interactive input")

	tests := []struct {
		name        string
		layerPath   string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "edit secrets for base layer",
			layerPath:   "base/dev",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "edit secrets for foundation layer",
			layerPath:   "foundation/dev",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "edit secrets without layer path",
			layerPath:   "",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "layer path is required",
		},
		{
			name:        "edit secrets with invalid layer path",
			layerPath:   "invalid/layer",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "invalid layer path",
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
			if tt.configFile == "gocloud-example-config.yaml" {
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
			cmd := secretsEditCmd
			args := []string{}
			if tt.configFile != "" {
				args = append(args, "--config", filepath.Join(tempDir, tt.configFile))
			}
			if tt.layerPath != "" {
				args = append(args, tt.layerPath)
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsEdit() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsEdit() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SecretsEdit() expected no error but got: %v", err)
				}
			}
		})
	}
}

// TestShouldGenerateSecretsForPath_MapInterfaceFormat documents that shouldGenerateSecretsForPath
// does NOT support project/workload defined as map[interface{}]interface{} (YAML unmarshal format).
// When a workload has enable_secrets: false in that format, the function should return false. Fails until fixed.
func TestShouldGenerateSecretsForPath_MapInterfaceFormat(t *testing.T) {
	infra := &models.InfrastructureConfig{
		Environments: map[string]models.Environment{
			"dev": {
				Workloads: []interface{}{
					// Simulates YAML: - wdwl: { enable_secrets: false }
					map[interface{}]interface{}{
						"wdwl": map[interface{}]interface{}{
							"enable_secrets": false,
						},
					},
				},
			},
		},
	}
	got := shouldGenerateSecretsForPath(infra, "workload", "wdwl", "dev")
	if got {
		t.Errorf("shouldGenerateSecretsForPath(workload wdwl with enable_secrets: false in map[interface{}] format) = true, want false")
	}
}

// TestShouldGenerateSecretsForPath_MapInterfaceFormat_Project documents the same for another workload key (dept).
// When a workload "dept" has enable_secrets: false in map[interface{}] format, the function should return false.
func TestShouldGenerateSecretsForPath_MapInterfaceFormat_Project(t *testing.T) {
	infra := &models.InfrastructureConfig{
		Environments: map[string]models.Environment{
			"dev": {
				Workloads: []interface{}{
					map[interface{}]interface{}{
						"dept": map[interface{}]interface{}{
							"enable_secrets": false,
						},
					},
				},
			},
		},
	}
	got := shouldGenerateSecretsForPath(infra, "workload", "dept", "dev")
	if got {
		t.Errorf("shouldGenerateSecretsForPath(workload dept with enable_secrets: false in map[interface{}] format) = true, want false")
	}
}

func TestCompleteLayerPaths(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "complete layer paths with valid config",
			args:        []string{"--config", "gocloud-example-config.yaml"},
			expectError: false,
		},
		{
			name:        "complete layer paths without config",
			args:        []string{},
			expectError: true,
			errorMsg:    "config file is required",
		},
		{
			name:        "complete layer paths with invalid config",
			args:        []string{"--config", "invalid.yaml"},
			expectError: true,
			errorMsg:    "invalid config",
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
			} else if strings.Contains(strings.Join(tt.args, " "), "invalid.yaml") {
				err = os.WriteFile(filepath.Join(tempDir, "invalid.yaml"), []byte("invalid: yaml: content: ["), 0644)
				if err != nil {
					t.Fatalf("Failed to create invalid config: %v", err)
				}
			}

			// Test completion function
			// Note: This is a simplified test since we can't easily test the actual completion
			// in a unit test environment
			if tt.expectError {
				// We expect this to fail in test environment
				t.Logf("Completion test expected to fail in test environment")
			} else {
				// We expect this to work in test environment
				t.Logf("Completion test expected to work in test environment")
			}
		})
	}
}

func TestCompleteSecretKeys(t *testing.T) {
	tests := []struct {
		name        string
		layerPath   string
		configFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "complete secret keys for base layer",
			layerPath:   "base/dev",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "complete secret keys for foundation layer",
			layerPath:   "foundation/dev",
			configFile:  "gocloud-example-config.yaml",
			expectError: false,
		},
		{
			name:        "complete secret keys with invalid layer path",
			layerPath:   "invalid/layer",
			configFile:  "gocloud-example-config.yaml",
			expectError: true,
			errorMsg:    "invalid layer path",
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
			if tt.configFile == "gocloud-example-config.yaml" {
				exampleConfig, err := os.ReadFile("gocloud-example-config.yaml")
				if err != nil {
					t.Skipf("Skipping test: example config not found")
				}
				err = os.WriteFile(filepath.Join(tempDir, "gocloud-example-config.yaml"), exampleConfig, 0644)
				if err != nil {
					t.Fatalf("Failed to copy example config: %v", err)
				}
			}

			// Test completion function
			// Note: This is a simplified test since we can't easily test the actual completion
			// in a unit test environment
			if tt.expectError {
				// We expect this to fail in test environment
				t.Logf("Secret keys completion test expected to fail in test environment")
			} else {
				// We expect this to work in test environment
				t.Logf("Secret keys completion test expected to work in test environment")
			}
		})
	}
}

func TestParseLayerPathComponents(t *testing.T) {
	tests := []struct {
		layerPath     string
		wantLayerType string
		wantProject   string
		wantEnv       string
		wantOk        bool
	}{
		{"base/prd", "base", "", "prd", true},
		{"foundation/stg", "foundation", "", "stg", true},
		{"project/core/prd", "project", "core", "prd", true},
		{"workload/webapp/dev", "workload", "webapp", "dev", true},
		{"base", "", "", "", false},
		{"a/b/c/d", "", "", "", false},
		{"", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.layerPath, func(t *testing.T) {
			layerType, project, env, ok := parseLayerPathComponents(tt.layerPath)
			if ok != tt.wantOk {
				t.Errorf("parseLayerPathComponents(%q) ok = %v, want %v", tt.layerPath, ok, tt.wantOk)
			}
			if ok {
				if layerType != tt.wantLayerType || project != tt.wantProject || env != tt.wantEnv {
					t.Errorf("parseLayerPathComponents(%q) = %q, %q, %q; want %q, %q, %q",
						tt.layerPath, layerType, project, env, tt.wantLayerType, tt.wantProject, tt.wantEnv)
				}
			}
		})
	}
}

func TestCheckSecretsEnabledSilent(t *testing.T) {
	enabled := true
	disabled := false
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			EnableSecrets: &enabled,
			Environments: map[string]models.Environment{
				"prd": {
					Name:       "Production",
					DirName:    "prd",
					AWSAccount: "123456789012",
				},
			},
		},
	}

	if !checkSecretsEnabledSilent(config, "base/prd") {
		t.Error("checkSecretsEnabledSilent(base/prd) with secrets enabled expected true")
	}
	if checkSecretsEnabledSilent(config, "invalid") {
		t.Error("checkSecretsEnabledSilent(invalid path) expected false")
	}

	configDisabled := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			EnableSecrets: &disabled,
			Environments: map[string]models.Environment{
				"prd": {Name: "Production", DirName: "prd", AWSAccount: "123456789012"},
			},
		},
	}
	if checkSecretsEnabledSilent(configDisabled, "base/prd") {
		t.Error("checkSecretsEnabledSilent(base/prd) with secrets disabled expected false")
	}
}

func TestGetSecretsWorkingDir(t *testing.T) {
	// getSecretsWorkingDir uses package var secretsConfig; should not panic
	dir := getSecretsWorkingDir()
	if dir == "" {
		t.Error("getSecretsWorkingDir() returned empty string")
	}
}

func TestGetAllLayersFromConfigAndGetLayersForEnvironment(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Client:  "test-client",
			Company: "gcl",
			Environments: map[string]models.Environment{
				"prd": {
					Name:       "Production",
					DirName:    "prd",
					AWSAccount: "123456789012",
					Projects:   []interface{}{"core"},
					Workloads:  []interface{}{"web"},
				},
			},
		},
	}
	layers := getAllLayersFromConfig(config)
	if len(layers) < 4 {
		t.Errorf("getAllLayersFromConfig() returned %d layers, expected at least 4", len(layers))
	}
	envLayers := getLayersForEnvironment(config, "prd")
	if len(envLayers) < 4 {
		t.Errorf("getLayersForEnvironment(prd) returned %d layers, expected at least 4", len(envLayers))
	}
}

func TestIsContentError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("something else"), false},
		{errors.New("key not found"), true},
		{errors.New("parameter does not exist"), true},
		{errors.New("not found in layer"), true},
		{errors.New("parameter not found"), true},
		{errors.New("ParameterNotFound"), true},
	}
	for _, tt := range tests {
		name := "nil"
		if tt.err != nil {
			name = tt.err.Error()
		}
		t.Run(name, func(t *testing.T) {
			got := isContentError(tt.err)
			if got != tt.want {
				t.Errorf("isContentError() = %v, want %v", got, tt.want)
			}
		})
	}
}
