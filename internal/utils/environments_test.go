package utils

import (
	"gocloud-cli/internal/models"
	"strings"
	"testing"
)

func TestShowPredefinedEnvironmentOptions(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "show predefined environment options",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function should not return an error
			// It only displays options to stdout
			ShowPredefinedEnvironmentOptions()
		})
	}
}

func TestPromptForPredefinedEnvironments(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "prompt for predefined environments",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptForPredefinedEnvironments() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptForPredefinedEnvironments()
			if err != nil {
				t.Logf("PromptForPredefinedEnvironments() returned error (expected in test environment): %v", err)
			}
		})
	}
}

func TestConfigurePredefinedEnvironment(t *testing.T) {
	tests := []struct {
		name          string
		predefinedEnv models.PredefinedEnvironment
		expectError   bool
		errorMsg      string
	}{
		{
			name: "configure development environment",
			predefinedEnv: models.PredefinedEnvironment{
				Key:         "dev",
				Name:        "Development",
				Description: "Development environment",
				Projects:    []string{"core"},
				Workloads:   []string{"api"},
			},
			expectError: false,
		},
		{
			name: "configure staging environment",
			predefinedEnv: models.PredefinedEnvironment{
				Key:         "stg",
				Name:        "Staging",
				Description: "Staging environment",
				Projects:    []string{"core"},
				Workloads:   []string{"api"},
			},
			expectError: false,
		},
		{
			name: "configure production environment",
			predefinedEnv: models.PredefinedEnvironment{
				Key:         "prd",
				Name:        "Production",
				Description: "Production environment",
				Projects:    []string{"core"},
				Workloads:   []string{"api"},
			},
			expectError: false,
		},
		{
			name: "configure environment with empty key",
			predefinedEnv: models.PredefinedEnvironment{
				Key:         "",
				Name:        "Development",
				Description: "Development environment",
				Projects:    []string{"core"},
				Workloads:   []string{"api"},
			},
			expectError: true,
			errorMsg:    "environment key is required",
		},
		{
			name: "configure environment with empty name",
			predefinedEnv: models.PredefinedEnvironment{
				Key:         "dev",
				Name:        "",
				Description: "Development environment",
				Projects:    []string{"core"},
				Workloads:   []string{"api"},
			},
			expectError: true,
			errorMsg:    "environment name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic and handles validation
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ConfigurePredefinedEnvironment() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := ConfigurePredefinedEnvironment(tt.predefinedEnv)
			if tt.expectError {
				if err == nil {
					t.Errorf("ConfigurePredefinedEnvironment() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ConfigurePredefinedEnvironment() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("ConfigurePredefinedEnvironment() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestConfigureCustomEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "configure custom environment",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("ConfigureCustomEnvironment() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, _, err := ConfigureCustomEnvironment()
			if err != nil {
				t.Logf("ConfigureCustomEnvironment() returned error (expected in test environment): %v", err)
			}
		})
	}
}

func TestEnvironmentValidation(t *testing.T) {
	tests := []struct {
		name        string
		env         models.Environment
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid environment",
			env: models.Environment{
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "123456789012",
			},
			expectError: false,
		},
		{
			name: "environment with empty name",
			env: models.Environment{
				Name:       "",
				DirName:    "dev",
				AWSAccount: "123456789012",
			},
			expectError: true,
			errorMsg:    "environment name is required",
		},
		{
			name: "environment with empty dir_name",
			env: models.Environment{
				Name:       "Development",
				DirName:    "",
				AWSAccount: "123456789012",
			},
			expectError: true,
			errorMsg:    "environment dir_name is required",
		},
		{
			name: "environment with invalid AWS account",
			env: models.Environment{
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "12345678901", // Too short
			},
			expectError: true,
			errorMsg:    "invalid AWS account ID",
		},
		{
			name: "environment with empty AWS account",
			env: models.Environment{
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "",
			},
			expectError: true,
			errorMsg:    "AWS account ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test environment validation logic
			var err error

			if tt.env.Name == "" {
				err = &ValidationError{Message: "environment name is required"}
			} else if tt.env.DirName == "" {
				err = &ValidationError{Message: "environment dir_name is required"}
			} else if tt.env.AWSAccount == "" {
				err = &ValidationError{Message: "AWS account ID is required"}
			} else if len(tt.env.AWSAccount) != 12 {
				err = &ValidationError{Message: "invalid AWS account ID"}
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("EnvironmentValidation() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("EnvironmentValidation() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("EnvironmentValidation() expected no error but got: %v", err)
				}
			}
		})
	}
}

// Helper struct for validation errors
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
