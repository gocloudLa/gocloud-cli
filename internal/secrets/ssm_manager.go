package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"

	"gocloud-cli/internal/models"
	"gocloud-cli/internal/utils"
)

// Manager handles AWS SSM secrets operations
type Manager struct {
	config      *models.Config
	ssm         *ssm.Client
	clientCache map[string]*ssm.Client
	cacheMutex  sync.RWMutex
}

// Layer represents a layer path with parsed components
type Layer struct {
	LayerType    string `json:"layer_type"`    // base, foundation, project, workload
	Environment  string `json:"environment"`   // shared, production, staging, etc.
	Project      string `json:"project"`       // project name (only for project/workload)
	CommonName   string `json:"common_name"`   // {company}-{env} or {company}-{env}-{project}
	SSMParameter string `json:"ssm_parameter"` // /terraform/{common_name}-{layer}
}

// Secret represents a secret key-value pair
type Secret struct {
	Key         string      `json:"key"`         // Clave dentro del JSON
	Value       interface{} `json:"value"`       // Valor (string, number, bool)
	Type        string      `json:"type"`        // Tipo de dato: string, number, boolean
	Layer       string      `json:"layer"`       // Layer: base, foundation, project, workload
	Environment string      `json:"environment"` // Environment: shared, production, etc.
	Project     string      `json:"project"`     // Project (solo para layers project/workload)
}

// Valid layer types for O(1) lookup
var validLayerTypes = map[string]bool{
	"base":         true,
	"foundation":   true,
	"organization": true,
	"project":      true,
	"workload":     true,
}

// formatCredentialError returns a simple credential error message
func formatCredentialError(_ *Layer) string {
	return "AWS credentials not available or expired"
}

// NewManager creates a new secrets manager
func NewManager(config *models.Config) (*Manager, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.Infrastructure == nil {
		return nil, fmt.Errorf("infrastructure config is required")
	}
	if config.Infrastructure.Client == "" {
		return nil, fmt.Errorf("client is required")
	}

	// Load AWS configuration with shared config
	// Use the same AWS_CONFIG_FILE that SSO commands use
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(config.Infrastructure.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create SSM client
	ssmClient := ssm.NewFromConfig(cfg)

	return &Manager{
		config:      config,
		ssm:         ssmClient,
		clientCache: make(map[string]*ssm.Client),
	}, nil
}

// getSSMClientForLayer creates an SSM client for a specific layer/environment with caching
func (m *Manager) getSSMClientForLayer(layer *Layer) (*ssm.Client, error) {
	// Determine the profile name based on the environment
	profileName := fmt.Sprintf("%s-%s", m.config.Infrastructure.Client, layer.Environment)

	// Check cache first
	m.cacheMutex.RLock()
	if client, exists := m.clientCache[profileName]; exists {
		m.cacheMutex.RUnlock()
		return client, nil
	}
	m.cacheMutex.RUnlock()

	// Create new client if not in cache
	configFile := ".aws/config"

	// Load AWS configuration with the specific profile and config file
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(m.config.Infrastructure.Region),
		awsconfig.WithSharedConfigProfile(profileName),
		awsconfig.WithSharedConfigFiles([]string{configFile}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for profile %s: %w", profileName, err)
	}

	client := ssm.NewFromConfig(cfg)

	// Cache the client
	m.cacheMutex.Lock()
	m.clientCache[profileName] = client
	m.cacheMutex.Unlock()

	return client, nil
}

// ParseLayerPath parses a layer path string into a Layer struct
// ParseLayerPath parses a layer path string into a Layer struct (method version for Manager)
func (m *Manager) ParseLayerPath(layerPath string) (*Layer, error) {
	return ParseLayerPath(layerPath, m.config)
}

// ParseLayerPath parses a layer path string into a Layer struct (static function)
func ParseLayerPath(layerPath string, config *models.Config) (*Layer, error) {
	parts := strings.Split(layerPath, "/")
	if len(parts) < 1 || len(parts) > 3 {
		return nil, fmt.Errorf("invalid layer path format: %s. Expected format: layer/environment or layer/project/environment", layerPath)
	}

	layer := &Layer{
		LayerType: parts[0],
	}

	// Single-segment path: only "organization" is valid
	if len(parts) == 1 {
		if layer.LayerType != "organization" {
			return nil, fmt.Errorf("invalid layer path format: %s. Expected format: layer/environment or layer/project/environment", layerPath)
		}
	}

	// Validate layer type using O(1) map lookup
	if !validLayerTypes[layer.LayerType] {
		validTypes := make([]string, 0, len(validLayerTypes))
		for t := range validLayerTypes {
			validTypes = append(validTypes, t)
		}
		return nil, fmt.Errorf("invalid layer type: %s. Valid types: %v", layer.LayerType, validTypes)
	}

	// Organization layer: single segment "organization" (global layer, env key "org")
	if layer.LayerType == "organization" {
		if len(parts) != 1 {
			return nil, fmt.Errorf("invalid path for organization layer: %s. Expected: organization", layerPath)
		}
		layer.Environment = "org"
		layer.CommonName = fmt.Sprintf("%s-%s", config.Infrastructure.Company, layer.Environment)
		layer.SSMParameter = fmt.Sprintf("/terraform/%s-%s", layer.CommonName, layer.LayerType)
		return layer, nil
	}

	// Parse based on layer type
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid layer path format: %s. Expected format: layer/environment or layer/project/environment", layerPath)
	}
	if layer.LayerType == "base" || layer.LayerType == "foundation" {
		// Format: base/production, foundation/staging
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid path for %s layer: %s. Expected: %s/environment", layer.LayerType, layerPath, layer.LayerType)
		}
		layer.Environment = parts[1]
	} else {
		// Format: project/core/production, workload/webapp/staging
		if len(parts) != 3 {
			switch layer.LayerType {
			case "project":
				return nil, fmt.Errorf("project is required for project layer")
			case "workload":
				return nil, fmt.Errorf("project is required for workload layer")
			default:
				return nil, fmt.Errorf("invalid path for %s layer: %s. Expected: %s/project/environment", layer.LayerType, layerPath, layer.LayerType)
			}
		}
		layer.Project = parts[1]
		layer.Environment = parts[2]
	}

	// Validate environment exists in config
	if _, exists := config.Infrastructure.Environments[layer.Environment]; !exists {
		return nil, fmt.Errorf("environment not found")
	}

	// Build common name
	if layer.Project != "" {
		layer.CommonName = fmt.Sprintf("%s-%s-%s", config.Infrastructure.Company, layer.Environment, layer.Project)
	} else {
		layer.CommonName = fmt.Sprintf("%s-%s", config.Infrastructure.Company, layer.Environment)
	}

	// Build SSM parameter name
	layer.SSMParameter = fmt.Sprintf("/terraform/%s-%s", layer.CommonName, layer.LayerType)

	return layer, nil
}

// updateSSMParameter is a helper function to update SSM parameters
func (m *Manager) updateSSMParameter(layer *Layer, secretsMap map[string]interface{}) error {
	// Convert to JSON
	jsonData, err := json.Marshal(secretsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets to JSON: %w", err)
	}

	// Get SSM client for this layer
	ssmClient, err := m.getSSMClientForLayer(layer)
	if err != nil {
		return err
	}

	// Update SSM parameter
	_, err = ssmClient.PutParameter(context.Background(), &ssm.PutParameterInput{
		Name:      aws.String(layer.SSMParameter),
		Value:     aws.String(string(jsonData)),
		Type:      "SecureString",
		Overwrite: aws.Bool(true),
	})
	if err != nil {
		// Check if it's a credential error
		if utils.IsCredentialError(err) {
			return fmt.Errorf("%s", formatCredentialError(layer))
		}
		return fmt.Errorf("failed to update SSM parameter: %w", err)
	}

	return nil
}

// secretsToMap converts a slice of secrets to a map
func secretsToMap(secrets []Secret) map[string]interface{} {
	secretsMap := make(map[string]interface{}, len(secrets))
	for _, secret := range secrets {
		secretsMap[secret.Key] = secret.Value
	}
	return secretsMap
}

// ListSecrets retrieves all secrets for a layer
func (m *Manager) ListSecrets(layer *Layer) ([]Secret, error) {
	// Get SSM client for this layer
	ssmClient, err := m.getSSMClientForLayer(layer)
	if err != nil {
		return nil, err
	}

	// Get parameter from AWS SSM
	result, err := ssmClient.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           aws.String(layer.SSMParameter),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		// Check if it's a credential error
		if utils.IsCredentialError(err) {
			return nil, fmt.Errorf("%s", formatCredentialError(layer))
		}
		return nil, fmt.Errorf("failed to get SSM parameter %s: %w", layer.SSMParameter, err)
	}

	// Parse JSON value
	var secretsMap map[string]interface{}
	if err := json.Unmarshal([]byte(*result.Parameter.Value), &secretsMap); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from SSM parameter: %w", err)
	}

	// Convert to Secret structs
	secrets := make([]Secret, 0, len(secretsMap))
	for key, value := range secretsMap {
		secret := Secret{
			Key:         key,
			Value:       value,
			Type:        getValueType(value),
			Layer:       layer.LayerType,
			Environment: layer.Environment,
			Project:     layer.Project,
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// GetSecret retrieves a specific secret value
func (m *Manager) GetSecret(layer *Layer, key string) (interface{}, error) {
	secrets, err := m.ListSecrets(layer)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		if secret.Key == key {
			return secret.Value, nil
		}
	}

	return nil, fmt.Errorf("secret key '%s' not found in layer: %s", key, layer.SSMParameter)
}

// SetSecret sets a secret value
func (m *Manager) SetSecret(layer *Layer, key string, value interface{}) error {
	// Get current secrets
	secrets, err := m.ListSecrets(layer)
	if err != nil {
		// If parameter doesn't exist, start with empty map
		secrets = []Secret{}
	}

	// Convert to map for easier manipulation
	secretsMap := secretsToMap(secrets)

	// Add or update the key
	secretsMap[key] = value

	// Update SSM parameter
	return m.updateSSMParameter(layer, secretsMap)
}

// DeleteSecret deletes a secret key
func (m *Manager) DeleteSecret(layer *Layer, key string) error {
	// Get current secrets
	secrets, err := m.ListSecrets(layer)
	if err != nil {
		return fmt.Errorf("failed to get current secrets: %w", err)
	}

	// Convert to map for easier manipulation
	secretsMap := secretsToMap(secrets)

	// Check if key exists
	if _, exists := secretsMap[key]; !exists {
		return fmt.Errorf("secret key '%s' not found in layer: %s", key, layer.SSMParameter)
	}

	// Delete the key
	delete(secretsMap, key)

	// Update SSM parameter
	return m.updateSSMParameter(layer, secretsMap)
}

// getValueType returns the type of a value as a string
func getValueType(value interface{}) string {
	switch v := value.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return reflect.TypeOf(v).String()
	}
}

// CheckSecrets checks if secrets exist for a layer without retrieving them
func (m *Manager) CheckSecrets(layer *Layer) (string, error) {
	// Get SSM client for this layer
	ssmClient, err := m.getSSMClientForLayer(layer)
	if err != nil {
		return "", err
	}

	// Try to get parameter metadata (faster than getting the full parameter)
	_, err = ssmClient.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name: aws.String(layer.SSMParameter),
	})

	if err != nil {
		// Check if it's a credential error
		if utils.IsCredentialError(err) {
			return "", fmt.Errorf("%s", formatCredentialError(layer))
		}

		// Check if it's a ParameterNotFound error
		if strings.Contains(err.Error(), "ParameterNotFound") {
			return "NOT_FOUND", nil
		}

		// Other errors
		return "", fmt.Errorf("failed to check SSM parameter %s: %w", layer.SSMParameter, err)
	}

	return "EXISTS", nil
}

// InitSecrets initializes secrets for a layer with empty JSON
func (m *Manager) InitSecrets(layer *Layer) error {
	// Get SSM client for this layer
	ssmClient, err := m.getSSMClientForLayer(layer)
	if err != nil {
		return err
	}

	// Initialize with empty JSON
	emptyJSON := "{}"

	// Put parameter with empty JSON
	_, err = ssmClient.PutParameter(context.Background(), &ssm.PutParameterInput{
		Name:      aws.String(layer.SSMParameter),
		Value:     aws.String(emptyJSON),
		Type:      types.ParameterTypeSecureString,
		Overwrite: aws.Bool(false), // Don't overwrite if exists
	})

	if err != nil {
		// Check if it's a credential error
		if utils.IsCredentialError(err) {
			return fmt.Errorf("%s", formatCredentialError(layer))
		}

		// Check if parameter already exists
		if strings.Contains(err.Error(), "ParameterAlreadyExists") {
			return fmt.Errorf("parameter already exists")
		}

		// Other errors
		return fmt.Errorf("failed to initialize SSM parameter %s: %w", layer.SSMParameter, err)
	}

	return nil
}
