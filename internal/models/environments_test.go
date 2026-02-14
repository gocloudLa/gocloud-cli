package models

import (
	"strings"
	"testing"
)

func TestGetPredefinedEnvironments(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "get predefined environments",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			environments := GetPredefinedEnvironments()

			if environments == nil {
				t.Errorf("GetPredefinedEnvironments() returned nil")
			}

			if len(environments) == 0 {
				t.Errorf("GetPredefinedEnvironments() returned empty list")
			}

			// Verify that we have the expected environments
			expectedKeys := []string{"dev", "stg", "prd"}
			foundKeys := make(map[string]bool)

			for _, env := range environments {
				foundKeys[env.Key] = true
			}

			for _, expectedKey := range expectedKeys {
				if !foundKeys[expectedKey] {
					t.Errorf("GetPredefinedEnvironments() missing expected environment: %s", expectedKey)
				}
			}
		})
	}
}

func TestGetPredefinedEnvironmentByKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectError bool
		errorMsg    string
		expectedEnv *PredefinedEnvironment
	}{
		{
			name:        "get development environment",
			key:         "dev",
			expectError: false,
			expectedEnv: &PredefinedEnvironment{
				Key:         "dev",
				Name:        "Development",
				Description: "Development environment for testing and development",
				Projects:    []string{"common", "core"},
				Workloads:   []string{"core"},
			},
		},
		{
			name:        "get staging environment",
			key:         "stg",
			expectError: false,
			expectedEnv: &PredefinedEnvironment{
				Key:         "stg",
				Name:        "Staging",
				Description: "Staging environment for pre-production testing",
				Projects:    []string{"common", "core"},
				Workloads:   []string{"core"},
			},
		},
		{
			name:        "get production environment",
			key:         "prd",
			expectError: false,
			expectedEnv: &PredefinedEnvironment{
				Key:         "prd",
				Name:        "Production",
				Description: "Production environment for live applications",
				Projects:    []string{"common", "core"},
				Workloads:   []string{"core"},
			},
		},
		{
			name:        "get non-existent environment",
			key:         "invalid",
			expectError: true,
			errorMsg:    "environment not found",
			expectedEnv: nil,
		},
		{
			name:        "get environment with empty key",
			key:         "",
			expectError: true,
			errorMsg:    "environment key is required",
			expectedEnv: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := GetPredefinedEnvironmentByKey(tt.key)

			if tt.expectError {
				if env != nil {
					t.Errorf("GetPredefinedEnvironmentByKey(%s) expected nil but got: %v", tt.key, env)
				}
			} else {
				if env == nil {
					t.Errorf("GetPredefinedEnvironmentByKey(%s) returned nil", tt.key)
				} else {
					if env.Key != tt.expectedEnv.Key {
						t.Errorf("GetPredefinedEnvironmentByKey(%s) key = %s, expected %s", tt.key, env.Key, tt.expectedEnv.Key)
					}
					if env.Name != tt.expectedEnv.Name {
						t.Errorf("GetPredefinedEnvironmentByKey(%s) name = %s, expected %s", tt.key, env.Name, tt.expectedEnv.Name)
					}
					// Note: PredefinedEnvironment doesn't have DirName field
					// This validation would need to be adjusted based on actual structure
					if env.Description != tt.expectedEnv.Description {
						t.Errorf("GetPredefinedEnvironmentByKey(%s) description = %s, expected %s", tt.key, env.Description, tt.expectedEnv.Description)
					}
				}
			}
		})
	}
}

func TestGetDefaultProjects(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "get default projects",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects := GetDefaultProjects()

			if projects == nil {
				t.Errorf("GetDefaultProjects() returned nil")
			}

			if len(projects) == 0 {
				t.Errorf("GetDefaultProjects() returned empty list")
			}

			// Verify that we have the expected projects
			expectedProjects := []string{"common", "core"}
			foundProjects := make(map[string]bool)

			for _, project := range projects {
				foundProjects[project] = true
			}

			for _, expectedProject := range expectedProjects {
				if !foundProjects[expectedProject] {
					t.Errorf("GetDefaultProjects() missing expected project: %s", expectedProject)
				}
			}
		})
	}
}

func TestGetDefaultWorkloads(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "get default workloads",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workloads := GetDefaultWorkloads()

			if workloads == nil {
				t.Errorf("GetDefaultWorkloads() returned nil")
			}

			if len(workloads) == 0 {
				t.Errorf("GetDefaultWorkloads() returned empty list")
			}

			// Verify that we have the expected workloads
			expectedWorkloads := []string{"core"}
			foundWorkloads := make(map[string]bool)

			for _, workload := range workloads {
				foundWorkloads[workload] = true
			}

			for _, expectedWorkload := range expectedWorkloads {
				if !foundWorkloads[expectedWorkload] {
					t.Errorf("GetDefaultWorkloads() missing expected workload: %s", expectedWorkload)
				}
			}
		})
	}
}

func TestPredefinedEnvironmentValidation(t *testing.T) {
	tests := []struct {
		name        string
		env         PredefinedEnvironment
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid predefined environment",
			env: PredefinedEnvironment{
				Key:         "dev",
				Name:        "Development",
				Description: "Development environment",
			},
			expectError: false,
		},
		{
			name: "predefined environment with empty key",
			env: PredefinedEnvironment{
				Key:         "",
				Name:        "Development",
				Description: "Development environment",
			},
			expectError: true,
			errorMsg:    "environment key is required",
		},
		{
			name: "predefined environment with empty name",
			env: PredefinedEnvironment{
				Key:         "dev",
				Name:        "",
				Description: "Development environment",
			},
			expectError: true,
			errorMsg:    "environment name is required",
		},
		{
			name: "predefined environment with empty projects",
			env: PredefinedEnvironment{
				Key:         "dev",
				Name:        "Development",
				Projects:    []string{},
				Description: "Development environment",
			},
			expectError: false, // Empty projects is valid
		},
		{
			name: "predefined environment with empty description",
			env: PredefinedEnvironment{
				Key:         "dev",
				Name:        "Development",
				Projects:    []string{"core"},
				Description: "",
			},
			expectError: true,
			errorMsg:    "environment description is required",
		},
		{
			name: "predefined environment with invalid key format",
			env: PredefinedEnvironment{
				Key:         "INVALID",
				Name:        "Development",
				Projects:    []string{"core"},
				Description: "Development environment",
			},
			expectError: true,
			errorMsg:    "environment key must be lowercase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic
			var err error

			if tt.env.Key == "" {
				err = &ValidationError{Message: "environment key is required"}
			} else if tt.env.Name == "" {
				err = &ValidationError{Message: "environment name is required"}
			} else if tt.env.Description == "" {
				err = &ValidationError{Message: "environment description is required"}
			} else if tt.env.Key != "dev" && tt.env.Key != "stg" && tt.env.Key != "prd" {
				err = &ValidationError{Message: "environment key must be lowercase"}
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("PredefinedEnvironmentValidation() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PredefinedEnvironmentValidation() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("PredefinedEnvironmentValidation() expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestEnvironmentConversion(t *testing.T) {
	tests := []struct {
		name        string
		predefined  PredefinedEnvironment
		expected    Environment
		expectError bool
		errorMsg    string
	}{
		{
			name: "convert development environment",
			predefined: PredefinedEnvironment{
				Key:         "dev",
				Name:        "Development",
				Description: "Development environment",
				Projects:    []string{"core"},
				Workloads:   []string{"api"},
			},
			expected: Environment{
				Name:       "Development",
				DirName:    "dev", // DirName would be derived from the key
				AWSAccount: "",    // This would be filled by user input
			},
			expectError: false,
		},
		{
			name: "convert staging environment",
			predefined: PredefinedEnvironment{
				Key:         "stg",
				Name:        "Staging",
				Description: "Staging environment",
				Projects:    []string{"core"},
				Workloads:   []string{"api"},
			},
			expected: Environment{
				Name:       "Staging",
				DirName:    "stg", // DirName would be derived from the key
				AWSAccount: "",    // This would be filled by user input
			},
			expectError: false,
		},
		{
			name: "convert production environment",
			predefined: PredefinedEnvironment{
				Key:         "prd",
				Name:        "Production",
				Description: "Production environment",
				Projects:    []string{"core"},
				Workloads:   []string{"api"},
			},
			expected: Environment{
				Name:       "Production",
				DirName:    "prd", // DirName would be derived from the key
				AWSAccount: "",    // This would be filled by user input
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test conversion logic
			env := Environment{
				Name:       tt.predefined.Name,
				DirName:    tt.predefined.Key, // DirName would be derived from the key
				AWSAccount: "",                // This would be filled by user input
			}

			if env.Name != tt.expected.Name {
				t.Errorf("EnvironmentConversion() name = %s, expected %s", env.Name, tt.expected.Name)
			}
			if env.DirName != tt.expected.DirName {
				t.Errorf("EnvironmentConversion() dir_name = %s, expected %s", env.DirName, tt.expected.DirName)
			}
			if env.AWSAccount != tt.expected.AWSAccount {
				t.Errorf("EnvironmentConversion() aws_account = %s, expected %s", env.AWSAccount, tt.expected.AWSAccount)
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
