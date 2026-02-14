package secrets

import (
	"gocloud-cli/internal/models"
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &models.Config{
				CLI: models.DefaultCLIConfig(),
				Infrastructure: &models.InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Environments: map[string]models.Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "config is required",
		},
		{
			name: "config without infrastructure",
			config: &models.Config{
				CLI: models.DefaultCLIConfig(),
			},
			expectError: true,
			errorMsg:    "infrastructure config is required",
		},
		{
			name: "config without client",
			config: &models.Config{
				CLI: models.DefaultCLIConfig(),
				Infrastructure: &models.InfrastructureConfig{
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
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
			errorMsg:    "client is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("NewManager() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("NewManager() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewManager() expected no error but got: %v", err)
				}
				if manager == nil {
					t.Errorf("NewManager() returned nil manager")
				}
			}
		})
	}
}

func TestParseLayerPath(t *testing.T) {
	tests := []struct {
		name        string
		layerPath   string
		config      *models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name:      "valid base layer path",
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
			expectError: false,
		},
		{
			name:      "valid foundation layer path",
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
			expectError: false,
		},
		{
			name:      "valid project layer path",
			layerPath: "project/core/dev",
			config: &models.Config{
				Infrastructure: &models.InfrastructureConfig{
					Client: "test-client",
					Environments: map[string]models.Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
							Projects:   []interface{}{"core"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:      "valid workload layer path",
			layerPath: "workload/core/dev",
			config: &models.Config{
				Infrastructure: &models.InfrastructureConfig{
					Client: "test-client",
					Environments: map[string]models.Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
							Projects:   []interface{}{"core"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:      "invalid layer path format",
			layerPath: "invalid",
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
		{
			name:      "invalid layer type",
			layerPath: "invalid/dev",
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
			name:      "non-existent environment",
			layerPath: "base/invalid",
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
			errorMsg:    "environment not found",
		},
		{
			name:      "project layer without project",
			layerPath: "project/dev",
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
			errorMsg:    "project is required for project layer",
		},
		{
			name:      "workload layer without project",
			layerPath: "workload/dev",
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
			errorMsg:    "project is required for workload layer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			layer, err := manager.ParseLayerPath(tt.layerPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseLayerPath() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ParseLayerPath() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ParseLayerPath() expected no error but got: %v", err)
				}
				if layer == nil {
					t.Errorf("ParseLayerPath() returned nil layer")
				}
			}
		})
	}
}

func TestFormatCredentialError(t *testing.T) {
	got := formatCredentialError(nil)
	want := "AWS credentials not available or expired"
	if got != want {
		t.Errorf("formatCredentialError(nil) = %q, want %q", got, want)
	}
	got2 := formatCredentialError(&Layer{})
	if got2 != want {
		t.Errorf("formatCredentialError(&Layer{}) = %q, want %q", got2, want)
	}
}

func TestListSecrets(t *testing.T) {
	t.Skip("Skipping TestListSecrets - requires AWS configuration")
	tests := []struct {
		name        string
		layerPath   string
		config      *models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name:      "list secrets for base layer",
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
			expectError: false,
		},
		{
			name:      "list secrets for foundation layer",
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
			expectError: false,
		},
		{
			name:      "list secrets with invalid layer path",
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
			errorMsg:    "invalid layer path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			layer, err := manager.ParseLayerPath(tt.layerPath)
			if err != nil {
				t.Fatalf("Failed to parse layer path: %v", err)
			}

			secrets, err := manager.ListSecrets(layer)

			if tt.expectError {
				if err == nil {
					t.Errorf("ListSecrets() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ListSecrets() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ListSecrets() expected no error but got: %v", err)
				}
				if secrets == nil {
					t.Errorf("ListSecrets() returned nil secrets")
				}
			}
		})
	}
}

func TestGetSecret(t *testing.T) {
	t.Skip("Skipping TestGetSecret - requires AWS configuration")
	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		config      *models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name:      "get existing secret",
			layerPath: "base/dev",
			secretKey: "api_key",
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
			expectError: false,
		},
		{
			name:      "get non-existent secret",
			layerPath: "base/dev",
			secretKey: "non-existent-key",
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
			errorMsg:    "secret not found",
		},
		{
			name:      "get secret with invalid layer path",
			layerPath: "invalid/layer",
			secretKey: "api_key",
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
			errorMsg:    "invalid layer path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			layer, err := manager.ParseLayerPath(tt.layerPath)
			if err != nil {
				t.Fatalf("Failed to parse layer path: %v", err)
			}

			value, err := manager.GetSecret(layer, tt.secretKey)

			if tt.expectError {
				if err == nil {
					t.Errorf("GetSecret() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetSecret() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetSecret() expected no error but got: %v", err)
				}
				if value == "" {
					t.Errorf("GetSecret() returned empty value")
				}
			}
		})
	}
}

func TestSetSecret(t *testing.T) {
	t.Skip("Skipping TestSetSecret - requires AWS configuration")
	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		secretValue string
		config      *models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "set new secret",
			layerPath:   "base/dev",
			secretKey:   "new_api_key",
			secretValue: "new-secret-value",
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
			expectError: false,
		},
		{
			name:        "update existing secret",
			layerPath:   "base/dev",
			secretKey:   "api_key",
			secretValue: "updated-secret-value",
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
			expectError: false,
		},
		{
			name:        "set secret with empty value",
			layerPath:   "base/dev",
			secretKey:   "api_key",
			secretValue: "",
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
			errorMsg:    "secret value cannot be empty",
		},
		{
			name:        "set secret with invalid layer path",
			layerPath:   "invalid/layer",
			secretKey:   "api_key",
			secretValue: "secret-value",
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
			errorMsg:    "invalid layer path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			layer, err := manager.ParseLayerPath(tt.layerPath)
			if err != nil {
				t.Fatalf("Failed to parse layer path: %v", err)
			}

			err = manager.SetSecret(layer, tt.secretKey, tt.secretValue)

			if tt.expectError {
				if err == nil {
					t.Errorf("SetSecret() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("SetSecret() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SetSecret() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	t.Skip("Skipping TestDeleteSecret - requires AWS configuration")
	tests := []struct {
		name        string
		layerPath   string
		secretKey   string
		config      *models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name:      "delete existing secret",
			layerPath: "base/dev",
			secretKey: "api_key",
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
			expectError: false,
		},
		{
			name:      "delete non-existent secret",
			layerPath: "base/dev",
			secretKey: "non-existent-key",
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
			errorMsg:    "secret not found",
		},
		{
			name:      "delete secret with invalid layer path",
			layerPath: "invalid/layer",
			secretKey: "api_key",
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
			errorMsg:    "invalid layer path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			layer, err := manager.ParseLayerPath(tt.layerPath)
			if err != nil {
				t.Fatalf("Failed to parse layer path: %v", err)
			}

			err = manager.DeleteSecret(layer, tt.secretKey)

			if tt.expectError {
				if err == nil {
					t.Errorf("DeleteSecret() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("DeleteSecret() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteSecret() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestGetSSMClientForLayer(t *testing.T) {
	t.Skip("Skipping TestGetSSMClientForLayer - requires AWS configuration")
	tests := []struct {
		name        string
		layer       *Layer
		config      *models.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid layer",
			layer: &Layer{
				LayerType:   "base",
				Environment: "dev",
				Project:     "",
			},
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
			expectError: false,
		},
		{
			name:        "nil layer",
			layer:       nil,
			config:      &models.Config{},
			expectError: true,
			errorMsg:    "layer is required",
		},
		{
			name: "layer with non-existent environment",
			layer: &Layer{
				LayerType:   "base",
				Environment: "invalid",
				Project:     "",
			},
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
			errorMsg:    "environment not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(tt.config)
			if err != nil {
				t.Fatalf("Failed to create manager: %v", err)
			}

			client, err := manager.getSSMClientForLayer(tt.layer)

			if tt.expectError {
				if err == nil {
					t.Errorf("GetSSMClientForLayer() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetSSMClientForLayer() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetSSMClientForLayer() expected no error but got: %v", err)
				}
				if client == nil {
					t.Errorf("GetSSMClientForLayer() returned nil client")
				}
			}
		})
	}
}
