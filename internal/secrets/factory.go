package secrets

import (
	"fmt"
	"strings"

	"gocloud-cli/internal/models"
)

// SecretsManagerInterface defines the interface that both SSM and SOPS managers must implement
type SecretsManagerInterface interface {
	ListSecrets(layer *Layer) ([]Secret, error)
	GetSecret(layer *Layer, key string) (interface{}, error)
	SetSecret(layer *Layer, key string, value interface{}) error
	DeleteSecret(layer *Layer, key string) error
	CheckSecrets(layer *Layer) (string, error)
	InitSecrets(layer *Layer) error
	EditSecrets(layer *Layer) error
	ParseLayerPath(layerPath string) (*Layer, error)
}

// ResolveSecretsConfig resolves the secrets configuration with hierarchy
// Priority: Workload > Project > Environment > Global > Default ("ssm")
func ResolveSecretsConfig(config *models.Config, layerPath string) (*models.SecretsConfig, error) {
	if config == nil || config.Infrastructure == nil {
		// Default to "ssm"
		return &models.SecretsConfig{Type: "ssm"}, nil
	}

	// Parse layer path
	parts := strings.Split(layerPath, "/")
	if len(parts) < 1 || len(parts) > 3 {
		return nil, fmt.Errorf("invalid layer path format: %s", layerPath)
	}

	layerType := parts[0]
	var projectKey, envKey string

	if len(parts) == 1 {
		// Format: organization (global layer, no environment)
		if layerType != "organization" {
			return nil, fmt.Errorf("invalid layer path format: %s", layerPath)
		}
		envKey = "org"
	} else if len(parts) == 2 {
		// Format: base/production, foundation/staging
		envKey = parts[1]
	} else {
		// Format: project/core/production, workload/webapp/staging
		projectKey = parts[1]
		envKey = parts[2]
	}

	// Use the infrastructure config's resolve method
	return config.Infrastructure.ResolveSecretsConfig(layerType, projectKey, envKey), nil
}

// NewManagerForLayer creates the appropriate secrets manager based on the layer path and configuration
func NewManagerForLayer(config *models.Config, layerPath, workingDir string) (SecretsManagerInterface, error) {
	// Resolve secrets config
	secretsConfig, err := ResolveSecretsConfig(config, layerPath)
	if err != nil {
		return nil, err
	}

	// Create appropriate manager based on type
	switch secretsConfig.Type {
	case "sops":
		return NewSOPSManager(config, workingDir)
	case "ssm", "":
		// Default to SSM
		return NewManager(config)
	default:
		return nil, fmt.Errorf("unknown secrets backend type: %s", secretsConfig.Type)
	}
}
