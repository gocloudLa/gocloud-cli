package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"

	"gocloud-cli/internal/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// EditSecrets opens a text editor to edit the JSON secrets directly
func (m *Manager) EditSecrets(layer *Layer) error {
	// Get current secrets
	secrets, err := m.ListSecrets(layer)
	if err != nil {
		// Check if it's a credential error - if so, return it directly
		if strings.Contains(err.Error(), "AWS credentials not available or expired") {
			return err
		}
		// If parameter doesn't exist, start with empty JSON
		secrets = []Secret{}
	}

	// Convert secrets to JSON map
	secretsMap := make(map[string]interface{})
	for _, secret := range secrets {
		secretsMap[secret.Key] = secret.Value
	}

	// Convert to pretty JSON
	jsonData, err := json.MarshalIndent(secretsMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets to JSON: %w", err)
	}

	// Create temporary file with safe name
	safeName := strings.ReplaceAll(layer.SSMParameter, "/", "-")
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("gocloud-secrets-%s-*.json", safeName))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			log.Printf("Warning: failed to remove temp file: %v", err)
		}
	}()

	// Write current JSON to temp file
	if _, err := tmpFile.Write(jsonData); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			log.Printf("Warning: failed to close temp file: %v", closeErr)
		}
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Get and validate editor
	editor := getEditor()
	if editor == "" {
		return fmt.Errorf("no suitable text editor found")
	}

	// Show initial info
	utils.PrintWarning("Opening editor for layer: %s", layer.SSMParameter)
	utils.PrintWarning("SSM Parameter: %s", layer.SSMParameter)
	utils.PrintWarning("Editor: %s", editor)
	fmt.Println()

	// Main editing loop
	var newSecretsMap map[string]interface{}
	for {
		// Open editor
		if err := openEditor(editor, tmpFile.Name()); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		// Read and validate modified JSON
		modifiedData, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return fmt.Errorf("failed to read modified file: %w", err)
		}

		// Try to validate JSON
		if err := json.Unmarshal(modifiedData, &newSecretsMap); err != nil {
			// JSON is invalid, show error and ask user to fix
			utils.PrintError("❌ Invalid JSON format detected!")
			utils.PrintError("Error: %v", err)
			fmt.Println()

			utils.PrintWarning("The file contains invalid JSON. Please fix the format and try again.")
			fmt.Println()

			// Show the problematic content with line numbers
			utils.PrintInfo("Content that needs to be fixed:")
			lines := strings.Split(string(modifiedData), "\n")
			for i, line := range lines {
				utils.PrintText("%2d: %s\n", i+1, line)
			}
			fmt.Println()

			// Ask if user wants to retry
			utils.PrintWarning("Press Enter to reopen editor and fix the JSON, or Ctrl+C to cancel...")
			if _, err := fmt.Scanln(); err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			// Loop will continue and validate again
			continue
		}

		// JSON is valid, check if there are actual changes
		if !hasChanges(secretsMap, newSecretsMap) {
			utils.PrintWarning("No changes detected. Exiting.")
			return nil
		}

		// Show preview of changes
		utils.PrintSuccess("✅ JSON format is valid!")
		fmt.Println()
		utils.PrintInfo("Preview of changes:")

		showChanges(secretsMap, newSecretsMap)
		fmt.Println()

		// Confirm changes
		if !confirmChanges() {
			utils.PrintWarning("Changes cancelled.")
			return nil
		}

		// Apply changes
		utils.PrintSuccess("Applying changes...")

		// Convert back to JSON for SSM
		finalJSON, err := json.Marshal(newSecretsMap)
		if err != nil {
			return fmt.Errorf("failed to marshal final JSON: %w", err)
		}

		// Get SSM client for this layer
		ssmClient, err := m.getSSMClientForLayer(layer)
		if err != nil {
			return fmt.Errorf("failed to get SSM client: %w", err)
		}

		// Update SSM parameter
		_, err = ssmClient.PutParameter(context.Background(), &ssm.PutParameterInput{
			Name:      aws.String(layer.SSMParameter),
			Value:     aws.String(string(finalJSON)),
			Type:      "SecureString",
			Overwrite: aws.Bool(true),
		})
		if err != nil {
			// Check if it's a size limit error
			if strings.Contains(err.Error(), "Standard tier parameters support a maximum parameter value of 4096 characters") {
				if err := handleSizeLimitError(err, newSecretsMap, tmpFile.Name(), editor); err != nil {
					return err
				}
				// Continue the main loop to re-edit
				continue
			}
			return fmt.Errorf("failed to update SSM parameter: %w", err)
		}

		utils.PrintSuccess("✅ Changes applied successfully!")
		utils.PrintSuccess("SSM Parameter updated: %s", layer.SSMParameter)

		return nil
	}
}

// openEditor opens the specified editor with the given file
func openEditor(editor, filename string) error {
	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// hasChanges efficiently checks if there are actual changes between two maps
func hasChanges(oldMap, newMap map[string]interface{}) bool {
	// Check if maps have different lengths
	if len(oldMap) != len(newMap) {
		return true
	}

	// Check if any values are different using reflect.DeepEqual for better performance
	for key, newValue := range newMap {
		if oldValue, exists := oldMap[key]; !exists || !reflect.DeepEqual(oldValue, newValue) {
			return true
		}
	}

	return false
}

// showChanges displays a preview of what will be changed
func showChanges(oldMap, newMap map[string]interface{}) {
	// Show added/modified secrets
	for key, value := range newMap {
		if oldValue, exists := oldMap[key]; exists {
			if !reflect.DeepEqual(oldValue, value) {
				utils.PrintWarning("  Modified: %s = %v", key, value)
			}
		} else {
			utils.PrintSuccess("  Added: %s = %v", key, value)
		}
	}

	// Show deleted secrets
	for key := range oldMap {
		if _, exists := newMap[key]; !exists {
			utils.PrintError("  Deleted: %s", key)
		}
	}
}

// confirmChanges prompts the user for confirmation
func confirmChanges() bool {
	for {
		utils.PrintWarning("Do you want to apply these changes? (y/n): ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			utils.PrintError("Failed to read user response: %v", err)
			continue
		}

		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "y":
			return true
		case "n":
			return false
		default:
			utils.PrintError("Please enter 'y' for yes or 'n' for no.")
		}
	}
}

// handleSizeLimitError handles the SSM parameter size limit error
func handleSizeLimitError(err error, newSecretsMap map[string]interface{}, tmpFileName, editor string) error {
	utils.PrintError("❌ Parameter size limit exceeded!")
	utils.PrintError("Error: %v", err)
	fmt.Println()

	utils.PrintWarning("The JSON content is too large for the standard SSM parameter tier.")
	utils.PrintWarning("You need to either:")
	utils.PrintWarning("1. Reduce the size of your secrets")
	utils.PrintWarning("2. Upgrade to advanced-parameter tier")
	fmt.Println()

	// Show the current content size
	finalJSON, _ := json.Marshal(newSecretsMap)
	contentSize := len(string(finalJSON))
	utils.PrintInfo("Current content size: %d characters (limit: 4096)", contentSize)
	fmt.Println()

	// Ask if user wants to go back to editor
	utils.PrintWarning("Press Enter to go back to editor and reduce content size, or Ctrl+C to cancel...")
	if _, err := fmt.Scanln(); err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	// Write the current content back to the temp file with proper formatting
	prettyJSON, err := json.MarshalIndent(newSecretsMap, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pretty JSON: %w", err)
	}

	if err := os.WriteFile(tmpFileName, prettyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write content back to file: %w", err)
	}

	// Reopen editor with the same content
	cmd := exec.Command(editor, tmpFileName)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	// Return nil to continue the main loop
	return nil
}

// getEditor returns the appropriate text editor to use
func getEditor() string {
	// Check environment variables first
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}

	// Default editors based on OS
	switch runtime.GOOS {
	case "windows":
		// Try common Windows editors
		editors := []string{"notepad", "code", "notepad++"}
		for _, editor := range editors {
			if _, err := exec.LookPath(editor); err == nil {
				return editor
			}
		}
		return "notepad"

	case "darwin":
		// macOS - try common editors
		editors := []string{"vim", "nano", "code", "subl"}
		for _, editor := range editors {
			if _, err := exec.LookPath(editor); err == nil {
				return editor
			}
		}
		return "vim"

	default:
		// Linux/Unix - try common editors
		editors := []string{"vim", "nano", "code", "subl", "gedit"}
		for _, editor := range editors {
			if _, err := exec.LookPath(editor); err == nil {
				return editor
			}
		}
		return "vim"
	}
}

// createTempFile creates a temporary file with the given content
func createTempFile(content string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "gocloud-test-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			log.Printf("Warning: failed to close temp file: %v", closeErr)
		}
		if removeErr := os.Remove(tmpFile.Name()); removeErr != nil {
			log.Printf("Warning: failed to remove temp file: %v", removeErr)
		}
		return nil, fmt.Errorf("failed to write content to temporary file: %w", err)
	}

	return tmpFile, nil
}

// parseSecretsFromContent parses JSON content into a secrets map
func parseSecretsFromContent(content string) (map[string]interface{}, error) {
	var secrets map[string]interface{}
	if err := json.Unmarshal([]byte(content), &secrets); err != nil {
		return nil, fmt.Errorf("failed to parse JSON content: %w", err)
	}

	// Validate that all values are strings
	for key, value := range secrets {
		if value == nil {
			return nil, fmt.Errorf("all values must be strings, but key '%s' has null value", key)
		}
		if _, ok := value.(string); !ok {
			return nil, fmt.Errorf("all values must be strings, but key '%s' has type %T", key, value)
		}
	}

	return secrets, nil
}

// validateSecretsJSON validates that the content is valid JSON
func validateSecretsJSON(content string) error {
	var temp map[string]interface{}
	if err := json.Unmarshal([]byte(content), &temp); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Validate that all values are strings and not empty
	for key, value := range temp {
		if value == nil {
			return fmt.Errorf("all values must be strings, but key '%s' has null value", key)
		}

		strValue, ok := value.(string)
		if !ok {
			return fmt.Errorf("all values must be strings, but key '%s' has non-string value", key)
		}

		if strValue == "" {
			return fmt.Errorf("empty values are not allowed, but key '%s' has empty string", key)
		}
	}

	return nil
}
