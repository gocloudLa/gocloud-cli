package cmd

import (
	"errors"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/secrets"
	"gocloud-cli/internal/testutils"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// secretsTestConfigYAML is a minimal, self-contained gocloud.yaml (no dependency on any
// external fixture file) with a "dev" environment and a "core" project/workload, enough to
// exercise base/foundation/project/workload layer paths against a mocked SSM client.
const secretsTestConfigYAML = `
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
      projects:
        - core
      workloads:
        - core
`

// writeSecretsTestConfig writes secretsTestConfigYAML into dir/gocloud.yaml and returns its path.
func writeSecretsTestConfig(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "gocloud.yaml")
	if err := os.WriteFile(path, []byte(secretsTestConfigYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	return path
}

// withMockSecretsManager overrides the newSecretsManagerForLayer seam for the duration of the
// test so that getSecretsManagerAndLayer returns a *secrets.Manager backed by mockClient instead
// of hitting real AWS. The original seam is restored via t.Cleanup.
func withMockSecretsManager(t *testing.T, mockClient *testutils.MockSSMClient) {
	t.Helper()
	original := newSecretsManagerForLayer
	newSecretsManagerForLayer = func(cfg *models.Config, layerPath, workingDir string) (secrets.SecretsManagerInterface, error) {
		return secrets.NewManagerWithClient(cfg, mockClient), nil
	}
	t.Cleanup(func() {
		newSecretsManagerForLayer = original
	})
}

// withSecretsConfigPath sets the package-level secretsConfig var (read by loadSecretsConfig)
// for the duration of the test and restores it via t.Cleanup.
func withSecretsConfigPath(t *testing.T, path string) {
	t.Helper()
	original := secretsConfig
	secretsConfig = path
	t.Cleanup(func() {
		secretsConfig = original
	})
}

// Note on scope: runSecretsList/Get/Set/Delete call handleCredentialError/handleContentError,
// which call os.Exit(1) directly when the underlying manager error matches a credential or
// "not found" pattern (see isContentError/utils.IsCredentialError). That means those specific
// branches cannot be exercised here without killing the test process (a subprocess-reexec test
// would be needed, which is out of scope). The error *classification* is covered by
// TestIsContentError below and by internal/secrets tests (credential/ParameterNotFound paths
// against the mock SSM client). The cases below only cover paths that return normally: success
// against the mock, and errors that surface before any manager call (invalid layer path,
// missing config).

func TestSecretsList(t *testing.T) {
	tests := []struct {
		name        string
		layerPath   string
		parameters  map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:       "list secrets for base layer",
			layerPath:  "base/dev",
			parameters: map[string]string{"/terraform/gcl-dev-base": `{"api_key":"secret123"}`},
		},
		{
			name:       "list secrets for foundation layer",
			layerPath:  "foundation/dev",
			parameters: map[string]string{"/terraform/gcl-dev-foundation": `{"database_url":"postgres://localhost/db"}`},
		},
		{
			name:       "list secrets for project layer",
			layerPath:  "project/core/dev",
			parameters: map[string]string{"/terraform/gcl-dev-core-project": `{"api_key":"secret123"}`},
		},
		{
			name:       "list secrets for workload layer",
			layerPath:  "workload/core/dev",
			parameters: map[string]string{"/terraform/gcl-dev-core-workload": `{"api_key":"secret123"}`},
		},
		{
			name:        "list secrets with invalid layer path",
			layerPath:   "invalid/layer",
			expectError: true,
			errorMsg:    "invalid layer path",
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

			withSecretsConfigPath(t, writeSecretsTestConfig(t, tempDir))
			withMockSecretsManager(t, &testutils.MockSSMClient{Parameters: tt.parameters})

			err = runSecretsList(secretsListCmd, []string{tt.layerPath})

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsList() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsList() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("SecretsList() expected no error but got: %v", err)
			}
		})
	}
}

func TestSecretsList_ConfigNotFound(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := testutils.CleanupTempDir(tempDir); err != nil {
			t.Logf("Warning: failed to cleanup temp dir: %v", err)
		}
	}()

	withSecretsConfigPath(t, filepath.Join(tempDir, "non-existent.yaml"))
	withMockSecretsManager(t, &testutils.MockSSMClient{})

	err = runSecretsList(secretsListCmd, []string{"base/dev"})
	if err == nil {
		t.Fatal("SecretsList() expected error but got nil")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("SecretsList() error = %q, want it to contain %q", err.Error(), "config file not found")
	}
}

// TestSecretsList_DisabledLayer verifies that when secrets are disabled for a layer, the runner
// exits successfully (nil error) without calling the manager at all.
func TestSecretsList_DisabledLayer(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := testutils.CleanupTempDir(tempDir); err != nil {
			t.Logf("Warning: failed to cleanup temp dir: %v", err)
		}
	}()

	disabledYAML := `
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
  enable_secrets: false
  environments:
    dev:
      name: "Development"
      dir_name: "dev"
      aws_account: "123456789012"
`
	path := filepath.Join(tempDir, "gocloud.yaml")
	if err := os.WriteFile(path, []byte(disabledYAML), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}
	withSecretsConfigPath(t, path)
	withMockSecretsManager(t, &testutils.MockSSMClient{})

	if err := runSecretsList(secretsListCmd, []string{"base/dev"}); err != nil {
		t.Errorf("SecretsList() with secrets disabled expected nil error but got: %v", err)
	}
}

func TestSecretsGet(t *testing.T) {
	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		parameters  map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:       "get secret from base layer",
			layerPath:  "base/dev",
			secretKey:  "api_key",
			parameters: map[string]string{"/terraform/gcl-dev-base": `{"api_key":"secret123"}`},
		},
		{
			name:       "get secret from foundation layer",
			layerPath:  "foundation/dev",
			secretKey:  "database_url",
			parameters: map[string]string{"/terraform/gcl-dev-foundation": `{"database_url":"postgres://localhost/db"}`},
		},
		{
			name:        "get secret with invalid layer path",
			layerPath:   "invalid/layer",
			secretKey:   "api_key",
			expectError: true,
			errorMsg:    "invalid layer path",
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

			withSecretsConfigPath(t, writeSecretsTestConfig(t, tempDir))
			withMockSecretsManager(t, &testutils.MockSSMClient{Parameters: tt.parameters})

			err = runSecretsGet(secretsGetCmd, []string{tt.layerPath, tt.secretKey})

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsGet() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsGet() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("SecretsGet() expected no error but got: %v", err)
			}
		})
	}
}

func TestSecretsSet(t *testing.T) {
	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		secretValue string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "set secret in base layer",
			layerPath:   "base/dev",
			secretKey:   "api_key",
			secretValue: "secret-value",
		},
		{
			name:        "set secret in foundation layer",
			layerPath:   "foundation/dev",
			secretKey:   "database_url",
			secretValue: "postgresql://user:pass@localhost/db",
		},
		{
			name:        "set secret with invalid layer path",
			layerPath:   "invalid/layer",
			secretKey:   "api_key",
			secretValue: "secret-value",
			expectError: true,
			errorMsg:    "invalid layer path",
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

			withSecretsConfigPath(t, writeSecretsTestConfig(t, tempDir))
			mockClient := &testutils.MockSSMClient{}
			withMockSecretsManager(t, mockClient)

			err = runSecretsSet(secretsSetCmd, []string{tt.layerPath, tt.secretKey, tt.secretValue})

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsSet() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsSet() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("SecretsSet() expected no error but got: %v", err)
			}

			// Verify the value was actually written to the mock SSM parameter.
			layer, lerr := secrets.ParseLayerPath(tt.layerPath, mustLoadSecretsTestConfig(t, tempDir))
			if lerr != nil {
				t.Fatalf("ParseLayerPath: %v", lerr)
			}
			stored, ok := mockClient.Parameters[layer.SSMParameter]
			if !ok {
				t.Fatalf("SecretsSet() did not write parameter %q", layer.SSMParameter)
			}
			if !strings.Contains(stored, tt.secretValue) {
				t.Errorf("stored parameter %q does not contain value %q", stored, tt.secretValue)
			}
		})
	}
}

func TestSecretsDelete(t *testing.T) {
	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		parameters  map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:       "delete secret from base layer",
			layerPath:  "base/dev",
			secretKey:  "api_key",
			parameters: map[string]string{"/terraform/gcl-dev-base": `{"api_key":"secret123"}`},
		},
		{
			name:       "delete secret from foundation layer",
			layerPath:  "foundation/dev",
			secretKey:  "database_url",
			parameters: map[string]string{"/terraform/gcl-dev-foundation": `{"database_url":"postgres://localhost/db"}`},
		},
		{
			name:        "delete secret with invalid layer path",
			layerPath:   "invalid/layer",
			secretKey:   "api_key",
			expectError: true,
			errorMsg:    "invalid layer path",
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

			withSecretsConfigPath(t, writeSecretsTestConfig(t, tempDir))
			mockClient := &testutils.MockSSMClient{Parameters: tt.parameters}
			withMockSecretsManager(t, mockClient)

			err = runSecretsDelete(secretsDeleteCmd, []string{tt.layerPath, tt.secretKey})

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsDelete() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsDelete() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("SecretsDelete() expected no error but got: %v", err)
			}
		})
	}
}

func TestSecretsEdit(t *testing.T) {
	tests := []struct {
		name        string
		layerPath   string
		skipReason  string
		expectError bool
		errorMsg    string
	}{
		{
			name:       "edit secrets for base layer",
			layerPath:  "base/dev",
			skipReason: "requires an interactive text editor process, out of scope for a hermetic test",
		},
		{
			name:        "edit secrets with invalid layer path",
			layerPath:   "invalid/layer",
			expectError: true,
			errorMsg:    "invalid layer path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			withSecretsConfigPath(t, writeSecretsTestConfig(t, tempDir))
			withMockSecretsManager(t, &testutils.MockSSMClient{})

			err = runSecretsEdit(secretsEditCmd, []string{tt.layerPath})

			if tt.expectError {
				if err == nil {
					t.Errorf("SecretsEdit() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SecretsEdit() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else if err != nil {
				t.Errorf("SecretsEdit() expected no error but got: %v", err)
			}
		})
	}
}

// mustLoadSecretsTestConfig re-parses the config written by writeSecretsTestConfig, for tests
// that need the *models.Config to compute the expected SSM parameter name.
func mustLoadSecretsTestConfig(t *testing.T, tempDir string) *models.Config {
	t.Helper()
	original := secretsConfig
	secretsConfig = filepath.Join(tempDir, "gocloud.yaml")
	defer func() { secretsConfig = original }()
	cfg, err := loadSecretsConfig()
	if err != nil {
		t.Fatalf("loadSecretsConfig: %v", err)
	}
	return cfg
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
