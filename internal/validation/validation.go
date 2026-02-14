package validation

import (
	"fmt"
	"regexp"
)

// ValidateEnvironmentKey validates if a string is a valid environment key (max 3 characters)
func ValidateEnvironmentKey(key string) error {
	// Environment keys should be lowercase letters (a-z) and numbers (0-9), 1-3 characters
	matched, _ := regexp.MatchString(`^[a-z0-9]{1,3}$`, key)
	if !matched {
		return fmt.Errorf("invalid environment key '%s': must be 1-3 lowercase letters (a-z) and numbers (0-9)", key)
	}
	return nil
}

// IsValidEnvironmentKey validates if a string is a valid environment key (max 3 characters) (legacy function)
func IsValidEnvironmentKey(key string) bool {
	return ValidateEnvironmentKey(key) == nil
}

// ValidateAWSAccountID validates AWS account ID format
func ValidateAWSAccountID(accountID string) error {
	// AWS account IDs are 12 digits
	matched, _ := regexp.MatchString(`^\d{12}$`, accountID)
	if !matched {
		return fmt.Errorf("AWS account ID must be exactly 12 digits")
	}
	return nil
}

// ValidateCompanyPrefix validates company prefix format
func ValidateCompanyPrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("company prefix is required")
	}

	if len(prefix) < 2 || len(prefix) > 10 {
		return fmt.Errorf("company prefix must be between 2 and 10 characters")
	}

	// Check if it contains only lowercase letters, numbers, and hyphens
	matched, _ := regexp.MatchString(`^[a-z0-9-]+$`, prefix)
	if !matched {
		return fmt.Errorf("company prefix must contain only lowercase letters, numbers, and hyphens")
	}

	return nil
}

// IsValidAWSAccountID validates if a string is a valid AWS account ID (legacy function)
func IsValidAWSAccountID(accountID string) bool {
	return ValidateAWSAccountID(accountID) == nil
}
