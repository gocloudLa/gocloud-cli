package validation

import (
	"strings"
	"testing"
)

func TestValidateEnvironmentKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectError bool
		errorMsg    string
	}{
		{"valid key", "sha", false, ""},
		{"valid key", "stg", false, ""},
		{"valid key", "prd", false, ""},
		{"valid key", "a", false, ""},
		{"valid key", "abc", false, ""},
		{"valid key - with numbers", "st1", false, ""},
		{"valid key - numbers only", "123", false, ""},
		{"valid key - single number", "1", false, ""},
		{"valid key - letter and number", "a1", false, ""},
		{"invalid key - too long", "abcd", true, "must be 1-3 lowercase letters (a-z) and numbers (0-9)"},
		{"invalid key - uppercase", "Sha", true, "must be 1-3 lowercase letters (a-z) and numbers (0-9)"},
		{"invalid key - special chars", "st-g", true, "must be 1-3 lowercase letters (a-z) and numbers (0-9)"},
		{"invalid key - empty", "", true, "must be 1-3 lowercase letters (a-z) and numbers (0-9)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEnvironmentKey(tt.key)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateEnvironmentKey(%s) expected error but got nil", tt.key)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateEnvironmentKey(%s) error message '%s' does not contain '%s'", tt.key, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateEnvironmentKey(%s) expected no error but got: %v", tt.key, err)
				}
			}
		})
	}
}

func TestValidateAWSAccountID(t *testing.T) {
	tests := []struct {
		name        string
		accountID   string
		expectError bool
		errorMsg    string
	}{
		{"valid account ID", "123456789012", false, ""},
		{"valid account ID", "000000000000", false, ""},
		{"invalid account ID - too short", "12345678901", true, "must be exactly 12 digits"},
		{"invalid account ID - too long", "1234567890123", true, "must be exactly 12 digits"},
		{"invalid account ID - contains letters", "12345678901a", true, "must be exactly 12 digits"},
		{"invalid account ID - contains special chars", "12345678901@", true, "must be exactly 12 digits"},
		{"invalid account ID - empty", "", true, "must be exactly 12 digits"},
		{"invalid account ID - garbage", "invalid", true, "must be exactly 12 digits"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAWSAccountID(tt.accountID)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateAWSAccountID(%s) expected error but got nil", tt.accountID)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateAWSAccountID(%s) error message '%s' does not contain '%s'", tt.accountID, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAWSAccountID(%s) expected no error but got: %v", tt.accountID, err)
				}
			}
		})
	}
}

func TestValidateCompanyPrefix(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		expectError bool
		errorMsg    string
	}{
		{"valid prefix - letters only", "gcl", false, ""},
		{"valid prefix - with numbers", "gcl1", false, ""},
		{"valid prefix - with hyphens", "gcl-ai", false, ""},
		{"valid prefix - numbers and hyphens", "company1", false, ""},
		{"valid prefix - minimum length", "ab", false, ""},
		{"valid prefix - maximum length", "abcdefghij", false, ""},
		{"invalid prefix - too short", "a", true, "must be between 2 and 10 characters"},
		{"invalid prefix - too long", "abcdefghijk", true, "must be between 2 and 10 characters"},
		{"invalid prefix - uppercase", "GCL", true, "must contain only lowercase letters, numbers, and hyphens"},
		{"invalid prefix - special chars", "gcl@ai", true, "must contain only lowercase letters, numbers, and hyphens"},
		{"invalid prefix - spaces", "gcl ai", true, "must contain only lowercase letters, numbers, and hyphens"},
		{"invalid prefix - empty", "", true, "company prefix is required"},
		{"valid prefix - starts with hyphen", "-gcl", false, ""},
		{"valid prefix - ends with hyphen", "gcl-", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCompanyPrefix(tt.prefix)
			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateCompanyPrefix(%s) expected error but got nil", tt.prefix)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateCompanyPrefix(%s) error message '%s' does not contain '%s'", tt.prefix, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCompanyPrefix(%s) expected no error but got: %v", tt.prefix, err)
				}
			}
		})
	}
}

func TestIsValidEnvironmentKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"sha", true},
		{"prd", true},
		{"a", true},
		{"abc", true},
		{"1", true},
		{"abcd", false},
		{"", false},
		{"Ab", false},
		{"a-b", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := IsValidEnvironmentKey(tt.key)
			if got != tt.expected {
				t.Errorf("IsValidEnvironmentKey(%q) = %v, expected %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestIsValidAWSAccountID(t *testing.T) {
	tests := []struct {
		accountID string
		expected  bool
	}{
		{"123456789012", true},
		{"000000000000", true},
		{"12345678901", false},
		{"1234567890123", false},
		{"", false},
		{"invalid", false},
		{"12345678901a", false},
	}

	for _, tt := range tests {
		t.Run(tt.accountID, func(t *testing.T) {
			got := IsValidAWSAccountID(tt.accountID)
			if got != tt.expected {
				t.Errorf("IsValidAWSAccountID(%q) = %v, expected %v", tt.accountID, got, tt.expected)
			}
		})
	}
}
