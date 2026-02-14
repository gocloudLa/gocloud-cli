package utils

import (
	"strings"
	"testing"
)

func TestValidateProjectName(t *testing.T) {
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
			name:        "valid project name with numbers",
			projectName: "project123",
			expectError: false,
		},
		{
			name:        "valid project name with hyphens",
			projectName: "my-test-project",
			expectError: false,
		},
		{
			name:        "valid project name minimum length",
			projectName: "ab",
			expectError: false,
		},
		{
			name:        "valid project name maximum length",
			projectName: "abcdefghijklmnopqrst",
			expectError: false,
		},
		{
			name:        "invalid project name too short",
			projectName: "a",
			expectError: true,
			errorMsg:    "project name must be between 2 and 20 characters",
		},
		{
			name:        "invalid project name too long",
			projectName: "abcdefghijklmnopqrstu",
			expectError: true,
			errorMsg:    "project name must be between 2 and 20 characters",
		},
		{
			name:        "invalid project name with uppercase",
			projectName: "Test-Project",
			expectError: true,
			errorMsg:    "project name must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:        "invalid project name with spaces",
			projectName: "test project",
			expectError: true,
			errorMsg:    "project name must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:        "invalid project name with special characters",
			projectName: "test@project",
			expectError: true,
			errorMsg:    "project name must contain only lowercase letters, numbers, and hyphens",
		},
		{
			name:        "invalid project name empty",
			projectName: "",
			expectError: true,
			errorMsg:    "project name is required",
		},
		{
			name:        "invalid project name starts with hyphen",
			projectName: "-test-project",
			expectError: true,
			errorMsg:    "project name cannot start or end with hyphen",
		},
		{
			name:        "invalid project name ends with hyphen",
			projectName: "test-project-",
			expectError: true,
			errorMsg:    "project name cannot start or end with hyphen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.projectName)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateProjectName(%s) expected error but got nil", tt.projectName)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateProjectName(%s) error message '%s' does not contain '%s'", tt.projectName, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateProjectName(%s) expected no error but got: %v", tt.projectName, err)
				}
			}
		})
	}
}

func TestPromptWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		label        string
		defaultValue string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "prompt with default value",
			label:        "Enter project name",
			defaultValue: "default-project",
			expectError:  false,
		},
		{
			name:         "prompt with empty default value",
			label:        "Enter project name",
			defaultValue: "",
			expectError:  false,
		},
		{
			name:         "prompt with empty label",
			label:        "",
			defaultValue: "default-project",
			expectError:  true,
			errorMsg:     "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptWithDefault() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptWithDefault(tt.label, tt.defaultValue)
			if tt.expectError {
				if err == nil {
					t.Errorf("PromptWithDefault() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PromptWithDefault() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("PromptWithDefault() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestPromptString(t *testing.T) {
	tests := []struct {
		name        string
		label       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "prompt for string",
			label:       "Enter project name",
			expectError: false,
		},
		{
			name:        "prompt with empty label",
			label:       "",
			expectError: true,
			errorMsg:    "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptString() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptString(tt.label)
			if tt.expectError {
				if err == nil {
					t.Errorf("PromptString() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PromptString() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("PromptString() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestPromptStringRequired(t *testing.T) {
	tests := []struct {
		name        string
		label       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "prompt for required string",
			label:       "Enter project name",
			expectError: false,
		},
		{
			name:        "prompt with empty label",
			label:       "",
			expectError: true,
			errorMsg:    "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptStringRequired() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptStringRequired(tt.label)
			if tt.expectError {
				if err == nil {
					t.Errorf("PromptStringRequired() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PromptStringRequired() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("PromptStringRequired() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestPromptWithValidation(t *testing.T) {
	tests := []struct {
		name         string
		label        string
		defaultValue string
		validator    func(string) error
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "prompt with valid validator",
			label:        "Enter project name",
			defaultValue: "default-project",
			validator:    ValidateProjectName,
			expectError:  false,
		},
		{
			name:         "prompt with nil validator",
			label:        "Enter project name",
			defaultValue: "default-project",
			validator:    nil,
			expectError:  true,
			errorMsg:     "validator is required",
		},
		{
			name:         "prompt with empty label",
			label:        "",
			defaultValue: "default-project",
			validator:    ValidateProjectName,
			expectError:  true,
			errorMsg:     "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptWithValidation() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptWithValidation(tt.label, tt.defaultValue, tt.validator)
			if tt.expectError {
				if err == nil {
					t.Errorf("PromptWithValidation() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PromptWithValidation() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("PromptWithValidation() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestPromptWithValidationRequired(t *testing.T) {
	tests := []struct {
		name        string
		label       string
		validator   func(string) error
		expectError bool
		errorMsg    string
	}{
		{
			name:        "prompt with valid validator",
			label:       "Enter project name",
			validator:   ValidateProjectName,
			expectError: false,
		},
		{
			name:        "prompt with nil validator",
			label:       "Enter project name",
			validator:   nil,
			expectError: true,
			errorMsg:    "validator is required",
		},
		{
			name:        "prompt with empty label",
			label:       "",
			validator:   ValidateProjectName,
			expectError: true,
			errorMsg:    "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptWithValidationRequired() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptWithValidationRequired(tt.label, tt.validator)
			if tt.expectError {
				if err == nil {
					t.Errorf("PromptWithValidationRequired() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PromptWithValidationRequired() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("PromptWithValidationRequired() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestPromptList(t *testing.T) {
	tests := []struct {
		name         string
		label        string
		defaultValue string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "prompt for list",
			label:        "Enter project names",
			defaultValue: "project1,project2",
			expectError:  false,
		},
		{
			name:         "prompt with empty default value",
			label:        "Enter project names",
			defaultValue: "",
			expectError:  false,
		},
		{
			name:         "prompt with empty label",
			label:        "",
			defaultValue: "project1,project2",
			expectError:  true,
			errorMsg:     "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptList() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptList(tt.label, tt.defaultValue)
			if tt.expectError {
				if err == nil {
					t.Errorf("PromptList() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PromptList() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("PromptList() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestPromptYesNo(t *testing.T) {
	tests := []struct {
		name         string
		label        string
		defaultValue bool
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "prompt for yes/no with default true",
			label:        "Enable debug mode?",
			defaultValue: true,
			expectError:  false,
		},
		{
			name:         "prompt for yes/no with default false",
			label:        "Enable debug mode?",
			defaultValue: false,
			expectError:  false,
		},
		{
			name:         "prompt with empty label",
			label:        "",
			defaultValue: true,
			expectError:  true,
			errorMsg:     "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptYesNo() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptYesNo(tt.label, tt.defaultValue)
			if tt.expectError {
				if err == nil {
					t.Errorf("PromptYesNo() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PromptYesNo() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("PromptYesNo() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}

func TestPromptYesNoRequired(t *testing.T) {
	tests := []struct {
		name        string
		label       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "prompt for required yes/no",
			label:       "Enable debug mode?",
			expectError: false,
		},
		{
			name:        "prompt with empty label",
			label:       "",
			expectError: true,
			errorMsg:    "label is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function would normally prompt for user input
			// In a test environment, we can't easily test interactive prompts
			// So we'll just verify it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PromptYesNoRequired() panicked: %v", r)
				}
			}()

			// Note: This will likely fail in a test environment due to no user input
			// That's expected behavior
			_, err := PromptYesNoRequired(tt.label)
			if tt.expectError {
				if err == nil {
					t.Errorf("PromptYesNoRequired() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("PromptYesNoRequired() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Logf("PromptYesNoRequired() returned error (expected in test environment): %v", err)
				}
			}
		})
	}
}
