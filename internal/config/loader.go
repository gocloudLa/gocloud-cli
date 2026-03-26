package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gocloud-cli/internal/models"
)

// LoadConfigWithPath loads configuration from a specific path with fallback logic
func LoadConfigWithPath(configPath string) (*models.Config, error) {
	manager := NewManager()

	// If no path provided, use default
	if configPath == "" {
		configPath = "gocloud.yaml"
	}

	// If not absolute path, try current directory first
	if !filepath.IsAbs(configPath) {
		currentDir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		// Try current directory
		fullPath := filepath.Join(currentDir, configPath)
		if _, err := os.Stat(fullPath); err == nil {
			return loadConfigFromFile(manager, fullPath)
		}

		// Try parent directory (for secrets command compatibility)
		parentDir := filepath.Dir(currentDir)
		fullPath = filepath.Join(parentDir, configPath)
		if _, err := os.Stat(fullPath); err == nil {
			return loadConfigFromFile(manager, fullPath)
		}
	} else {
		// Absolute path - load directly
		return loadConfigFromFile(manager, configPath)
	}

	return nil, fmt.Errorf("config file not found: %s", configPath)
}

// LoadConfigWithPathAndAWS loads configuration and sets up AWS config file
func LoadConfigWithPathAndAWS(configPath string) (*models.Config, error) {
	config, err := LoadConfigWithPath(configPath)
	if err != nil {
		return nil, err
	}

	// Set AWS_CONFIG_FILE environment variable to use local .aws/config
	// This ensures secrets commands use the same AWS configuration as SSO commands
	awsConfigFile := filepath.Join(filepath.Dir(configPath), ".aws", "config")
	if err := os.Setenv("AWS_CONFIG_FILE", awsConfigFile); err != nil {
		return nil, fmt.Errorf("failed to set AWS_CONFIG_FILE: %w", err)
	}

	return config, nil
}

// loadConfigFromFile loads configuration from a specific file
func loadConfigFromFile(manager *Manager, configPath string) (*models.Config, error) {
	config, err := manager.LoadConfig(configPath)
	if err != nil {
		// Check if it's a file not found error
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		// Check if it's a YAML syntax error
		if strings.Contains(err.Error(), "yaml") {
			return nil, fmt.Errorf("invalid yaml syntax: %w", err)
		}
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return config, nil
}
