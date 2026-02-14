package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
)

// ValidateProjectName validates project name format
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}

	if len(name) < 2 || len(name) > 20 {
		return fmt.Errorf("project name must be between 2 and 20 characters")
	}

	// Check for lowercase letters, numbers, and hyphens only
	matched, _ := regexp.MatchString(`^[a-z0-9-]+$`, name)
	if !matched {
		return fmt.Errorf("project name must contain only lowercase letters, numbers, and hyphens")
	}

	// Check that it doesn't start or end with hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("project name cannot start or end with hyphen")
	}

	return nil
}

// PromptWithDefault prompts user with a default value
func PromptWithDefault(label, defaultValue string) (string, error) {
	if label == "" {
		return "", fmt.Errorf("label is required")
	}

	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultValue,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
	}
	result, err := prompt.Run()
	if err != nil {
		if err.Error() == "interrupt" || err.Error() == "EOF" || strings.Contains(err.Error(), "^D") {
			return defaultValue, nil
		}
		return "", err
	}

	if strings.TrimSpace(result) == "" {
		return defaultValue, nil
	}

	return result, nil
}

// PromptString prompts user for a string input
func PromptString(label string) (string, error) {
	if label == "" {
		return "", fmt.Errorf("label is required")
	}

	prompt := promptui.Prompt{
		Label: label,
	}
	return prompt.Run()
}

// PromptStringRequired prompts user for a required string input (non-empty)
func PromptStringRequired(label string) (string, error) {
	if label == "" {
		return "", fmt.Errorf("label is required")
	}

	for {
		prompt := promptui.Prompt{
			Label: label,
		}
		result, err := prompt.Run()
		if err != nil {
			return "", err
		}

		result = strings.TrimSpace(result)
		if result != "" {
			return result, nil
		}

		PrintError("❌ This field is required. Please enter a value.")
	}
}

// PromptWithValidation prompts user with validation
func PromptWithValidation(label, defaultValue string, validator func(string) error) (string, error) {
	if validator == nil {
		return "", fmt.Errorf("validator is required")
	}

	value, err := PromptWithDefault(label, defaultValue)
	if err != nil {
		return "", err
	}

	if err := validator(value); err != nil {
		PrintError("❌ %v", err)
		return "", err
	}

	return value, nil
}

// PromptWithValidationRequired prompts user with validation and repeats until valid
func PromptWithValidationRequired(label string, validator func(string) error) (string, error) {
	if label == "" {
		return "", fmt.Errorf("label is required")
	}
	if validator == nil {
		return "", fmt.Errorf("validator is required")
	}

	for {
		prompt := promptui.Prompt{
			Label: label,
		}
		result, err := prompt.Run()
		if err != nil {
			return "", err
		}

		result = strings.TrimSpace(result)
		if result == "" {
			PrintError("❌ This field is required. Please enter a value.")
			continue
		}

		if err := validator(result); err != nil {
			PrintError("❌ %v", err)
			continue
		}

		return result, nil
	}
}

// PromptList prompts user for a comma-separated list
func PromptList(label, defaultValue string) ([]string, error) {
	if label == "" {
		return nil, fmt.Errorf("label is required")
	}

	value, err := PromptWithDefault(label, defaultValue)
	if err != nil {
		return nil, err
	}

	if value == "" {
		return []string{}, nil
	}

	items := strings.Split(value, ",")
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}

	return items, nil
}

// PromptYesNo prompts user for yes/no input
func PromptYesNo(label string, defaultValue bool) (bool, error) {
	if label == "" {
		return false, fmt.Errorf("label is required")
	}

	prompt := promptui.Prompt{
		Label: label,
	}

	if defaultValue {
		prompt.Default = "y"
	} else {
		prompt.Default = "n"
	}

	result, err := prompt.Run()
	if err != nil {
		if err.Error() == "interrupt" || err.Error() == "EOF" || strings.Contains(err.Error(), "^D") {
			return defaultValue, nil
		}
		return false, err
	}

	if strings.TrimSpace(result) == "" {
		return defaultValue, nil
	}

	result = strings.ToLower(strings.TrimSpace(result))
	finalResult := result == "y" || result == "yes" || result == "true" || result == "1"
	return finalResult, nil
}

// PromptYesNoRequired prompts user for yes/no input without default
func PromptYesNoRequired(label string) (bool, error) {
	if label == "" {
		return false, fmt.Errorf("label is required")
	}

	for {
		prompt := promptui.Prompt{
			Label: label,
		}

		result, err := prompt.Run()
		if err != nil {
			return false, err
		}

		result = strings.ToLower(strings.TrimSpace(result))
		if result == "y" || result == "yes" || result == "true" || result == "1" {
			return true, nil
		}
		if result == "n" || result == "no" || result == "false" || result == "0" {
			return false, nil
		}

		PrintError("❌ Please enter 'y' for yes or 'n' for no.")
	}
}
