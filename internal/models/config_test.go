package models

import (
	"testing"
)

func TestDefaultCLIConfig(t *testing.T) {
	config := DefaultCLIConfig()

	// Test default values
	if config.WorkingDir != "." {
		t.Errorf("DefaultCLIConfig() WorkingDir = %s, expected '.'", config.WorkingDir)
	}
	if config.AutoBackup != true {
		t.Errorf("DefaultCLIConfig() AutoBackup = %v, expected true", config.AutoBackup)
	}
	if config.BackupDir != ".gocloud-backups" {
		t.Errorf("DefaultCLIConfig() BackupDir = %s, expected '.gocloud-backups'", config.BackupDir)
	}
	if config.Verbose != false {
		t.Errorf("DefaultCLIConfig() Verbose = %v, expected false", config.Verbose)
	}
	if config.Debug != false {
		t.Errorf("DefaultCLIConfig() Debug = %v, expected false", config.Debug)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test CLI config
	if config.CLI.WorkingDir != "." {
		t.Errorf("DefaultConfig() CLI.WorkingDir = %s, expected '.'", config.CLI.WorkingDir)
	}

	// Test Infrastructure config
	if config.Infrastructure.Client != "" {
		t.Errorf("DefaultConfig() Infrastructure.Client = %s, expected empty string", config.Infrastructure.Client)
	}
	if config.Infrastructure.Company != "" {
		t.Errorf("DefaultConfig() Infrastructure.Company = %s, expected empty string", config.Infrastructure.Company)
	}
	if config.Infrastructure.Region != "" {
		t.Errorf("DefaultConfig() Infrastructure.Region = %s, expected empty string", config.Infrastructure.Region)
	}
	if config.Infrastructure.Version != "" {
		t.Errorf("DefaultConfig() Infrastructure.Version = %s, expected empty string", config.Infrastructure.Version)
	}

	// Test Environments - DefaultConfig returns empty InfrastructureConfig
	// so Environments can be nil, which is expected behavior
}

func ptrBool(b bool) *bool { return &b }

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Environments: map[string]Environment{
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
			name: "missing client",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Environments: map[string]Environment{
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
		{
			name: "missing company",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Environments: map[string]Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "company is required",
		},
		{
			name: "missing region",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Version: "v1.0.0",
					Environments: map[string]Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "region is required",
		},
		{
			name: "missing version",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
					// Version missing - not required in current implementation
					Environments: map[string]Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: false, // Version is not validated in current implementation
		},
		{
			name: "invalid company prefix",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "GCL", // Uppercase - invalid
					Region:  "us-east-1",
					Version: "v1.0.0",
					Environments: map[string]Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "company prefix must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name: "empty environments",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:       "test-client",
					Company:      "gcl",
					Region:       "us-east-1",
					Version:      "v1.0.0",
					Environments: map[string]Environment{},
				},
			},
			expectError: false, // Empty environments are not validated in current implementation
		},
		{
			name: "invalid environment key",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Environments: map[string]Environment{
						"INVALID": { // Uppercase - invalid
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "must be 1-3 lowercase letters (a-z) and numbers (0-9)",
		},
		{
			name: "invalid AWS account ID",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Environments: map[string]Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "12345678901", // Too short
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "must be exactly 12 digits",
		},
		{
			name: "invalid project-level secrets type",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Environments: map[string]Environment{
						"lab": {
							Name:       "Laboratory",
							AWSAccount: "123456789012",
							Projects: []interface{}{
								"core",
								map[string]interface{}{
									"example": map[string]interface{}{
										"secrets": map[string]interface{}{"type": "random"},
									},
								},
							},
							Workloads: []interface{}{"core"},
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "project 'example'.secrets",
		},
		{
			name: "organization layer enabled but infrastructure.organization.aws_account missing",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Layers: &LayerConfig{
						Organization: ptrBool(true),
					},
					Environments: map[string]Environment{},
				},
			},
			expectError: true,
			errorMsg:    "infrastructure.organization.aws_account is required",
		},
		{
			name: "organization layer enabled with aws_account is valid",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:  "test-client",
					Company: "gcl",
					Region:  "us-east-1",
					Version: "v1.0.0",
					Layers: &LayerConfig{
						Organization: ptrBool(true),
					},
					Organization: &OrganizationLayerConfig{AWSAccount: "123456789012"},
					Environments: map[string]Environment{},
				},
			},
			expectError: false,
		},
		{
			name: "github_sso with empty organization is invalid",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:    "test-client",
					Company:   "gcl",
					Region:    "us-east-1",
					Version:   "v1.0.0",
					GitHubSSO: &GitHubSSOConfig{Organization: ""},
					Environments: map[string]Environment{
						"dev": {
							Name:       "Development",
							DirName:    "dev",
							AWSAccount: "123456789012",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "infrastructure.github_sso.organization is required",
		},
		{
			name: "github_sso with organization is valid",
			config: &Config{
				CLI: DefaultCLIConfig(),
				Infrastructure: &InfrastructureConfig{
					Client:    "test-client",
					Company:   "gcl",
					Region:    "us-east-1",
					Version:   "v1.0.0",
					GitHubSSO: &GitHubSSOConfig{Organization: "gocloud-la"},
					Environments: map[string]Environment{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateConfig() expected error but got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateConfig() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateConfig() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestValidateConfigWithUnknownFields(t *testing.T) {
	t.Run("valid config with organization passes strict validation", func(t *testing.T) {
		// infrastructure.organization is a known field; must not produce "Unknown infrastructure field" warning
		yamlData := []byte(`
cli: {}
infrastructure:
  client: test-client
  company: gcl
  region: us-east-1
  environments:
    dev:
      name: Development
      dir_name: dev
      aws_account: "123456789012"
  organization:
    aws_account: "123456789012"
`)
		result, err := ValidateConfigWithUnknownFields(yamlData)
		if err != nil {
			t.Fatalf("ValidateConfigWithUnknownFields() err = %v", err)
		}
		for _, w := range result.Warnings {
			if w == "Unknown infrastructure field: 'infrastructure.organization'" {
				t.Errorf("infrastructure.organization is a known field; got unwanted warning: %s", w)
			}
		}
		if !result.Valid {
			t.Errorf("expected valid config; got Errors: %v", result.Errors)
		}
	})

	t.Run("unknown infrastructure field produces warning", func(t *testing.T) {
		yamlData := []byte(`
cli: {}
infrastructure:
  client: test-client
  company: gcl
  region: us-east-1
  environments:
    dev:
      name: Development
      dir_name: dev
      aws_account: "123456789012"
  foo_bar_unknown: something
`)
		result, err := ValidateConfigWithUnknownFields(yamlData)
		if err != nil {
			t.Fatalf("ValidateConfigWithUnknownFields() err = %v", err)
		}
		var found bool
		for _, w := range result.Warnings {
			if w == "Unknown infrastructure field: 'infrastructure.foo_bar_unknown'" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected warning for unknown field infrastructure.foo_bar_unknown; got Warnings: %v", result.Warnings)
		}
	})

	t.Run("unknown top-level field produces warning", func(t *testing.T) {
		yamlData := []byte(`
cli: {}
infrastructure:
  client: test-client
  company: gcl
  region: us-east-1
  environments:
    dev:
      name: Development
      dir_name: dev
      aws_account: "123456789012"
unknown_top: value
`)
		result, err := ValidateConfigWithUnknownFields(yamlData)
		if err != nil {
			t.Fatalf("ValidateConfigWithUnknownFields() err = %v", err)
		}
		var found bool
		for _, w := range result.Warnings {
			if w == "Unknown top-level field: 'unknown_top'" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected warning for unknown top-level field; got Warnings: %v", result.Warnings)
		}
	})

	t.Run("github_sso field passes strict validation", func(t *testing.T) {
		// infrastructure.github_sso is a known field; must not produce "Unknown infrastructure field" warning
		yamlData := []byte(`
cli: {}
infrastructure:
  client: test-client
  company: gcl
  region: us-east-1
  environments:
    dev:
      name: Development
      dir_name: dev
      aws_account: "123456789012"
  github_sso:
    organization: gocloud-la
`)
		result, err := ValidateConfigWithUnknownFields(yamlData)
		if err != nil {
			t.Fatalf("ValidateConfigWithUnknownFields() err = %v", err)
		}
		for _, w := range result.Warnings {
			if w == "Unknown infrastructure field: 'infrastructure.github_sso'" {
				t.Errorf("infrastructure.github_sso is a known field; got unwanted warning: %s", w)
			}
		}
		if !result.Valid {
			t.Errorf("expected valid config; got Errors: %v", result.Errors)
		}
	})

	t.Run("valid minimal config has no warnings", func(t *testing.T) {
		yamlData := []byte(`
cli: {}
infrastructure:
  client: test-client
  company: gcl
  region: us-east-1
  environments:
    dev:
      name: Development
      dir_name: dev
      aws_account: "123456789012"
`)
		result, err := ValidateConfigWithUnknownFields(yamlData)
		if err != nil {
			t.Fatalf("ValidateConfigWithUnknownFields() err = %v", err)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("valid minimal config should have no warnings; got: %v", result.Warnings)
		}
		if !result.Valid {
			t.Errorf("expected valid; got Errors: %v", result.Errors)
		}
	})
}

func TestValidateEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		env         Environment
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid environment",
			env: Environment{
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "123456789012",
			},
			expectError: false,
		},
		{
			name: "missing name",
			env: Environment{
				DirName:    "dev",
				AWSAccount: "123456789012",
			},
			expectError: false, // Name is not validated in current implementation
		},
		{
			name: "missing dir_name",
			env: Environment{
				Name:       "Development",
				AWSAccount: "123456789012",
			},
			expectError: false, // DirName is not validated in current implementation
		},
		{
			name: "missing aws_account",
			env: Environment{
				Name:    "Development",
				DirName: "dev",
			},
			expectError: true,
			errorMsg:    "aws_account is required",
		},
		{
			name: "invalid AWS account ID",
			env: Environment{
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "12345678901", // Too short
			},
			expectError: true,
			errorMsg:    "must be exactly 12 digits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvironment(tt.env)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateEnvironment() expected error but got nil")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateEnvironment() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEnvironment() expected no error but got: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidateSecretsConfig(t *testing.T) {
	tests := []struct {
		name        string
		secrets     *SecretsConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid ssm config",
			secrets:     &SecretsConfig{Type: "ssm"},
			expectError: false,
		},
		{
			name:        "valid sops config",
			secrets:     &SecretsConfig{Type: "sops"},
			expectError: false,
		},
		{
			name:        "empty type (valid)",
			secrets:     &SecretsConfig{Type: ""},
			expectError: false,
		},
		{
			name:        "nil config (valid)",
			secrets:     nil,
			expectError: false,
		},
		{
			name:        "invalid type",
			secrets:     &SecretsConfig{Type: "invalid"},
			expectError: true,
			errorMsg:    "type must be 'ssm' or 'sops'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSecretsConfig(tt.secrets)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateSecretsConfig() expected error but got nil")
				} else if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateSecretsConfig() error = %v, expected to contain '%s'", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateSecretsConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestResolveSecretsConfig(t *testing.T) {
	config := &InfrastructureConfig{
		Company: "gcl",
		Environments: map[string]Environment{
			"dev": {
				AWSAccount: "123456789012",
				Projects: []interface{}{
					"core",
					ProjectItem{
						Key:     "example",
						Secrets: &SecretsConfig{Type: "sops"},
					},
				},
				Workloads: []interface{}{
					"webapp",
					WorkloadItem{
						Key:     "api",
						Secrets: &SecretsConfig{Type: "sops"},
					},
				},
			},
			"prd": {
				AWSAccount: "123456789013",
				Secrets:    &SecretsConfig{Type: "sops"},
			},
		},
		Secrets: &SecretsConfig{Type: "ssm"},
	}

	tests := []struct {
		name       string
		layerType  string
		projectKey string
		envKey     string
		expected   string
	}{
		{
			name:       "project with secrets config",
			layerType:  "project",
			projectKey: "example",
			envKey:     "dev",
			expected:   "sops",
		},
		{
			name:       "project without secrets config (inherits from env)",
			layerType:  "project",
			projectKey: "core",
			envKey:     "dev",
			expected:   "ssm", // Default from global
		},
		{
			name:       "workload with secrets config",
			layerType:  "workload",
			projectKey: "api",
			envKey:     "dev",
			expected:   "sops",
		},
		{
			name:       "workload without secrets config (inherits from env)",
			layerType:  "workload",
			projectKey: "webapp",
			envKey:     "dev",
			expected:   "ssm", // Default from global
		},
		{
			name:       "environment with secrets config",
			layerType:  "base",
			projectKey: "",
			envKey:     "prd",
			expected:   "sops",
		},
		{
			name:       "environment without secrets config (inherits from global)",
			layerType:  "base",
			projectKey: "",
			envKey:     "dev",
			expected:   "ssm",
		},
		{
			name:       "no config at any level (defaults to ssm)",
			layerType:  "base",
			projectKey: "",
			envKey:     "stg",
			expected:   "ssm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ResolveSecretsConfig(tt.layerType, tt.projectKey, tt.envKey)
			if result == nil {
				t.Errorf("ResolveSecretsConfig(%s, %s, %s) = nil, expected non-nil",
					tt.layerType, tt.projectKey, tt.envKey)
				return
			}
			if result.Type != tt.expected {
				t.Errorf("ResolveSecretsConfig(%s, %s, %s).Type = %s, expected %s",
					tt.layerType, tt.projectKey, tt.envKey, result.Type, tt.expected)
			}
		})
	}

	// Organization layer with dedicated override
	configOrg := &InfrastructureConfig{
		Company: "gcl",
		Secrets: &SecretsConfig{Type: "ssm"},
		Organization: &OrganizationLayerConfig{
			Secrets: &SecretsConfig{Type: "sops"},
		},
	}
	orgResult := configOrg.ResolveSecretsConfig("organization", "", "org")
	if orgResult == nil {
		t.Fatal("ResolveSecretsConfig(organization) = nil, expected non-nil")
	}
	if orgResult.Type != "sops" {
		t.Errorf("ResolveSecretsConfig(organization with override).Type = %s, expected sops", orgResult.Type)
	}
	// Other layers unchanged
	baseResult := configOrg.ResolveSecretsConfig("base", "", "prd")
	if baseResult == nil || baseResult.Type != "ssm" {
		t.Errorf("ResolveSecretsConfig(base) should still use global ssm, got %v", baseResult)
	}
}

// TestResolveSecretsConfigWithMapFormat tests that project/workload-level secrets
// are resolved when config is loaded from YAML (projects/workloads as map format, not ProjectItem/WorkloadItem).
// This simulates gocloud-example-config.yaml where e.g. "example" project has secrets.type: "sops".
func TestResolveSecretsConfigWithMapFormat(t *testing.T) {
	// Config as it would be after YAML unmarshal: expanded project/workload as map[string]interface{}
	config := &InfrastructureConfig{
		Company: "gcl",
		Secrets: &SecretsConfig{Type: "ssm"}, // global default SSM
		Environments: map[string]Environment{
			"dev": {
				AWSAccount: "123456789012",
				Projects: []interface{}{
					"core",
					"common",
					map[string]interface{}{
						"example": map[string]interface{}{
							"name":    "Example Project",
							"secrets": map[string]interface{}{"type": "sops"},
						},
					},
				},
				Workloads: []interface{}{
					"webapp",
					map[string]interface{}{
						"sops-app": map[string]interface{}{
							"name":    "SOPS App",
							"secrets": map[string]interface{}{"type": "sops"},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		layerType  string
		projectKey string
		envKey     string
		expected   string
	}{
		{
			name:       "project with secrets in map format (YAML-like)",
			layerType:  "project",
			projectKey: "example",
			envKey:     "dev",
			expected:   "sops",
		},
		{
			name:       "workload with secrets in map format (YAML-like)",
			layerType:  "workload",
			projectKey: "sops-app",
			envKey:     "dev",
			expected:   "sops",
		},
		{
			name:       "project without secrets in map format inherits global",
			layerType:  "project",
			projectKey: "core",
			envKey:     "dev",
			expected:   "ssm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ResolveSecretsConfig(tt.layerType, tt.projectKey, tt.envKey)
			if result == nil {
				t.Errorf("ResolveSecretsConfig(%s, %s, %s) = nil, expected non-nil",
					tt.layerType, tt.projectKey, tt.envKey)
				return
			}
			if result.Type != tt.expected {
				t.Errorf("ResolveSecretsConfig(%s, %s, %s).Type = %s, expected %s",
					tt.layerType, tt.projectKey, tt.envKey, result.Type, tt.expected)
			}
		})
	}
}
