package secrets

import (
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/testutils"
	"os"
	"strings"
	"testing"
)

func TestEditSecrets(t *testing.T) {
	tests := []struct {
		name        string
		layerPath   string
		config      *models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name:      "edit secrets for base layer",
			layerPath: "base/dev",
			config: &models.Config{
				Infrastructure: &models.InfrastructureConfig{
					Client: "test-client",
					Environments: map[string]models.Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: false, // Layer path parsing should succeed
		},
		{
			name:      "edit secrets for foundation layer",
			layerPath: "foundation/dev",
			config: &models.Config{
				Infrastructure: &models.InfrastructureConfig{
					Client: "test-client",
					Environments: map[string]models.Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: false, // Layer path parsing should succeed
		},
		{
			name:      "edit secrets with invalid layer path",
			layerPath: "invalid/layer",
			config: &models.Config{
				Infrastructure: &models.InfrastructureConfig{
					Client: "test-client",
					Environments: map[string]models.Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid layer type",
		},
		{
			name:      "edit secrets with empty layer path",
			layerPath: "",
			config: &models.Config{
				Infrastructure: &models.InfrastructureConfig{
					Client: "test-client",
					Environments: map[string]models.Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid layer path format",
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

			manager, err := NewManager(tt.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			_, err = manager.ParseLayerPath(tt.layerPath)
			if err != nil {
				if tt.expectError && tt.errorMsg != "" && strings.Contains(err.Error(), tt.errorMsg) {
					return // Expected error
				}
				t.Fatalf("Failed to parse layer path: %v", err)
			}

			// Skip the actual EditSecrets call that would open an editor
			// Instead, just test that the layer parsing worked correctly
			if tt.expectError {
				// If we expected an error but parsing succeeded, that's unexpected
				t.Errorf("EditSecrets() expected error but layer parsing succeeded")
			}
		})
	}
}

func TestOpenEditor(t *testing.T) {
	// Skip this test as it would open real editors
	t.Skip("Skipping TestOpenEditor - would open real editors in test environment")
}

func TestCreateTempFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "create temp file with content",
			content:     "test content",
			expectError: false,
		},
		{
			name:        "create temp file with empty content",
			content:     "",
			expectError: false,
		},
		{
			name:        "create temp file with JSON content",
			content:     `{"api_key": "test-value", "database_url": "postgresql://localhost/db"}`,
			expectError: false,
		},
		{
			name:        "create temp file with large content",
			content:     strings.Repeat("test content ", 1000),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := createTempFile(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("CreateTempFile() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("CreateTempFile() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("CreateTempFile() expected no error but got: %v", err)
				}
				if file == nil {
					t.Errorf("CreateTempFile() returned nil file")
				}

				// Clean up
				if file != nil {
					if err := file.Close(); err != nil {
						t.Logf("Warning: failed to close file: %v", err)
					}
					if err := os.Remove(file.Name()); err != nil {
						t.Logf("Warning: failed to remove file: %v", err)
					}
				}
			}
		})
	}
}

func TestParseSecretsFromContent(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		expectError   bool
		errorMsg      string
		expectedCount int
	}{
		{
			name:          "parse valid JSON content",
			content:       `{"api_key": "test-value", "database_url": "postgresql://localhost/db"}`,
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:          "parse empty JSON content",
			content:       `{}`,
			expectError:   false,
			expectedCount: 0,
		},
		{
			name:        "parse invalid JSON content",
			content:     `{"api_key": "test-value", "database_url": "postgresql://localhost/db"`,
			expectError: true,
			errorMsg:    "failed to parse JSON content",
		},
		{
			name:        "parse non-JSON content",
			content:     "not json content",
			expectError: true,
			errorMsg:    "failed to parse JSON content",
		},
		{
			name:        "parse JSON with non-string values",
			content:     `{"api_key": "test-value", "port": 5432}`,
			expectError: true,
			errorMsg:    "all values must be strings",
		},
		{
			name:        "parse JSON with null values",
			content:     `{"api_key": "test-value", "database_url": null}`,
			expectError: true,
			errorMsg:    "all values must be strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets, err := parseSecretsFromContent(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseSecretsFromContent() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ParseSecretsFromContent() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ParseSecretsFromContent() expected no error but got: %v", err)
				}
				if len(secrets) != tt.expectedCount {
					t.Errorf("ParseSecretsFromContent() returned %d secrets, expected %d", len(secrets), tt.expectedCount)
				}
			}
		})
	}
}

func TestValidateSecretsJSON(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "validate valid JSON",
			content:     `{"api_key": "test-value", "database_url": "postgresql://localhost/db"}`,
			expectError: false,
		},
		{
			name:        "validate empty JSON",
			content:     `{}`,
			expectError: false,
		},
		{
			name:        "validate invalid JSON",
			content:     `{"api_key": "test-value", "database_url": "postgresql://localhost/db"`,
			expectError: true,
			errorMsg:    "invalid JSON",
		},
		{
			name:        "validate non-JSON content",
			content:     "not json content",
			expectError: true,
			errorMsg:    "invalid JSON",
		},
		{
			name:        "validate JSON with non-string values",
			content:     `{"api_key": "test-value", "port": 5432}`,
			expectError: true,
			errorMsg:    "all values must be strings",
		},
		{
			name:        "validate JSON with null values",
			content:     `{"api_key": "test-value", "database_url": null}`,
			expectError: true,
			errorMsg:    "all values must be strings",
		},
		{
			name:        "validate JSON with empty string values",
			content:     `{"api_key": "test-value", "database_url": ""}`,
			expectError: true,
			errorMsg:    "empty values are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecretsJSON(tt.content)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateSecretsJSON() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateSecretsJSON() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateSecretsJSON() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestEditSecretsIntegration(t *testing.T) {
	// Skip this test as it would open real editors
	t.Skip("Skipping TestEditSecretsIntegration - would open real editors in test environment")
}
