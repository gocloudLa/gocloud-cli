package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	"gocloud-cli/internal/models"
	"gocloud-cli/internal/utils"

	"gopkg.in/yaml.v3"
)

// kmsAPI is the minimal subset of the KMS client used by SOPSManager. Defining it here allows
// injecting a mock (see internal/testutils.MockKMSClient) instead of a real *kms.Client in tests.
type kmsAPI interface {
	DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
	CreateKey(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error)
	CreateAlias(ctx context.Context, params *kms.CreateAliasInput, optFns ...func(*kms.Options)) (*kms.CreateAliasOutput, error)
	ScheduleKeyDeletion(ctx context.Context, params *kms.ScheduleKeyDeletionInput, optFns ...func(*kms.Options)) (*kms.ScheduleKeyDeletionOutput, error)
}

// SOPSManager handles SOPS secrets operations
type SOPSManager struct {
	config          *models.Config
	workingDir      string
	kmsClientCache  map[string]kmsAPI
	cacheMutex      sync.RWMutex
	kmsCreatedCache map[string]bool // Track which KMS keys have been created/verified
	kmsMutex        sync.RWMutex

	// testCheckKMSKeyFunc, when set (tests only), replaces AWS calls in getKMSKeyExists.
	testCheckKMSKeyFunc func(envKey string) (exists bool, keyId string, err error)

	// testKMSClient, when set (tests only), is returned by getKMSClientForEnvironment for every
	// environment, bypassing per-profile AWS config loading. Set it via NewSOPSManagerWithClient.
	testKMSClient kmsAPI
}

// NewSOPSManager creates a new SOPS secrets manager
func NewSOPSManager(config *models.Config, workingDir string) (*SOPSManager, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.Infrastructure == nil {
		return nil, fmt.Errorf("infrastructure config is required")
	}
	if config.Infrastructure.Client == "" {
		return nil, fmt.Errorf("client is required")
	}

	// Default working directory to current directory if not specified
	if workingDir == "" {
		workingDir = "."
	}

	return &SOPSManager{
		config:          config,
		workingDir:      workingDir,
		kmsClientCache:  make(map[string]kmsAPI),
		kmsCreatedCache: make(map[string]bool),
	}, nil
}

// NewSOPSManagerWithClient creates a SOPSManager backed by a pre-built KMS client (typically a
// mock such as internal/testutils.MockKMSClient). The injected client is used for every
// environment, bypassing AWS config/profile loading. Intended for tests.
func NewSOPSManagerWithClient(config *models.Config, workingDir string, client kmsAPI) (*SOPSManager, error) {
	m, err := NewSOPSManager(config, workingDir)
	if err != nil {
		return nil, err
	}
	m.testKMSClient = client
	return m, nil
}

// getKMSClientForEnvironment creates a KMS client for a specific environment with caching
func (m *SOPSManager) getKMSClientForEnvironment(envKey string) (kmsAPI, error) {
	// Tests can inject a client that is used for every environment.
	if m.testKMSClient != nil {
		return m.testKMSClient, nil
	}

	// Check cache first
	m.cacheMutex.RLock()
	if client, exists := m.kmsClientCache[envKey]; exists {
		m.cacheMutex.RUnlock()
		return client, nil
	}
	m.cacheMutex.RUnlock()

	// Get environment configuration
	// Get region for this environment
	region := m.getRegionForEnvironment(envKey)

	// Determine the profile name based on the environment
	profileName := fmt.Sprintf("%s-%s", m.config.Infrastructure.Client, envKey)
	configFile := ".aws/config"

	// Load AWS configuration with the specific profile and config file
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
		awsconfig.WithSharedConfigProfile(profileName),
		awsconfig.WithSharedConfigFiles([]string{configFile}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for profile %s: %w", profileName, err)
	}

	client := kms.NewFromConfig(cfg)

	// Cache the client
	m.cacheMutex.Lock()
	m.kmsClientCache[envKey] = client
	m.cacheMutex.Unlock()

	return client, nil
}

// getRegionForEnvironment returns the region for a specific environment
func (m *SOPSManager) getRegionForEnvironment(envKey string) string {
	if m.config == nil || m.config.Infrastructure == nil {
		return ""
	}
	return m.config.Infrastructure.RegionForEnvironment(envKey)
}

// getAccountForEnvironment returns AWS account ID and display name for an env key.
// For "org" / "sec" uses infrastructure.organization / infrastructure.security aws_account; otherwise Environments[envKey].
func (m *SOPSManager) getAccountForEnvironment(envKey string) (accountID, displayName string, err error) {
	if envKey == "org" {
		if m.config.Infrastructure.Organization == nil || m.config.Infrastructure.Organization.AWSAccount == "" {
			return "", "", fmt.Errorf("organization layer not configured (missing infrastructure.organization.aws_account)")
		}
		return m.config.Infrastructure.Organization.AWSAccount, "Organization", nil
	}
	if envKey == "sec" {
		if m.config.Infrastructure.Security == nil || m.config.Infrastructure.Security.AWSAccount == "" {
			return "", "", fmt.Errorf("security layer not configured (missing infrastructure.security.aws_account)")
		}
		return m.config.Infrastructure.Security.AWSAccount, "Security", nil
	}
	envConfig, exists := m.config.Infrastructure.Environments[envKey]
	if !exists {
		return "", "", fmt.Errorf("environment '%s' not found", envKey)
	}
	name := envConfig.Name
	if name == "" {
		name = envKey
	}
	return envConfig.AWSAccount, name, nil
}

// getKMSAlias returns the KMS alias name (without "alias/" prefix) for an environment
func (m *SOPSManager) getKMSAlias(envKey string) string {
	return fmt.Sprintf("%s-%s-secrets", m.config.Infrastructure.Company, envKey)
}

// getKMSAliasFull returns the KMS alias with "alias/" prefix (e.g. "alias/company-env-secrets")
func (m *SOPSManager) getKMSAliasFull(envKey string) string {
	return "alias/" + m.getKMSAlias(envKey)
}

// getKMSAliasARN returns the full ARN for the KMS alias
func (m *SOPSManager) getKMSAliasARN(envKey string) (string, error) {
	accountID, _, err := m.getAccountForEnvironment(envKey)
	if err != nil {
		return "", err
	}

	region := m.getRegionForEnvironment(envKey)
	aliasName := m.getKMSAlias(envKey)

	// Format: arn:aws:kms:region:account-id:alias/alias-name
	return fmt.Sprintf("arn:aws:kms:%s:%s:alias/%s", region, accountID, aliasName), nil
}

// getKMSKeyExists returns whether the KMS key/alias exists for the environment.
// When testCheckKMSKeyFunc is set (tests), it is used instead of calling AWS.
func (m *SOPSManager) getKMSKeyExists(envKey string) (bool, string, error) {
	if m.testCheckKMSKeyFunc != nil {
		return m.testCheckKMSKeyFunc(envKey)
	}
	return m.checkKMSKey(envKey)
}

// checkKMSKey checks if the KMS key exists for an environment (without creating it)
// Returns: (exists bool, keyId string, error)
func (m *SOPSManager) checkKMSKey(envKey string) (bool, string, error) {
	// Get KMS client
	kmsClient, err := m.getKMSClientForEnvironment(envKey)
	if err != nil {
		return false, "", err
	}

	alias := m.getKMSAliasFull(envKey)

	// Check if alias exists
	describeOutput, err := kmsClient.DescribeKey(context.Background(), &kms.DescribeKeyInput{
		KeyId: aws.String(alias),
	})
	if err == nil {
		// Alias exists
		return true, *describeOutput.KeyMetadata.KeyId, nil
	}

	// Check if it's a NotFound error
	if strings.Contains(err.Error(), "NotFoundException") || strings.Contains(err.Error(), "not found") {
		return false, "", nil
	}

	// Some other error occurred
	return false, "", fmt.Errorf("failed to check KMS key: %w", err)
}

// EnsureAllKMSKeys ensures that all KMS keys exist for the given environments
// This is called once before processing multiple layers to avoid duplicate checks
func (m *SOPSManager) EnsureAllKMSKeys(envKeys []string, createIfMissing bool) error {
	// Get unique environments (in case of duplicates)
	envMap := make(map[string]bool)
	for _, envKey := range envKeys {
		envMap[envKey] = true
	}

	// Check each unique environment
	for envKey := range envMap {
		alias := m.getKMSAliasFull(envKey)

		// Check if already verified in this session
		m.kmsMutex.RLock()
		if m.kmsCreatedCache[envKey] {
			m.kmsMutex.RUnlock()
			utils.PrintSuccess("✅ KMS key %s: EXISTS", alias)
			continue
		}
		m.kmsMutex.RUnlock()

		// Check if KMS exists
		kmsExists, keyId, err := m.getKMSKeyExists(envKey)
		if err != nil {
			utils.PrintError("❌ KMS key %s: ERROR - %v", alias, err)
			return fmt.Errorf("failed to check KMS key for environment '%s': %w", envKey, err)
		}

		if kmsExists {
			utils.PrintSuccess("✅ KMS key %s: EXISTS (Key ID: %s)", alias, keyId)
			// Mark as verified
			m.kmsMutex.Lock()
			m.kmsCreatedCache[envKey] = true
			m.kmsMutex.Unlock()
		} else {
			if createIfMissing {
				utils.PrintWarning("⚠️  KMS key %s: NOT_FOUND, creating...", alias)
				if err := m.ensureKMSKey(envKey); err != nil {
					utils.PrintError("❌ KMS key %s: FAILED to create - %v", alias, err)
					return fmt.Errorf("failed to create KMS key for environment '%s': %w", envKey, err)
				}
			} else {
				utils.PrintError("❌ KMS key %s: NOT_FOUND", alias)
			}
		}
	}

	return nil
}

// ensureKMSKey ensures that the KMS key exists for an environment.
// It checks if the alias already exists in AWS before creating; if it exists, marks cache and returns.
func (m *SOPSManager) ensureKMSKey(envKey string) error {
	// Check if we've already created/verified this KMS key in this session
	m.kmsMutex.RLock()
	if m.kmsCreatedCache[envKey] {
		m.kmsMutex.RUnlock()
		return nil
	}
	m.kmsMutex.RUnlock()

	// Verify in AWS: alias may already exist (e.g. from a previous run or another layer)
	exists, _, err := m.getKMSKeyExists(envKey)
	if err != nil {
		return fmt.Errorf("failed to check KMS key: %w", err)
	}
	if exists {
		m.kmsMutex.Lock()
		m.kmsCreatedCache[envKey] = true
		m.kmsMutex.Unlock()
		return nil
	}

	accountID, displayName, err := m.getAccountForEnvironment(envKey)
	if err != nil {
		return err
	}

	alias := m.getKMSAliasFull(envKey)

	// Get KMS client
	kmsClient, err := m.getKMSClientForEnvironment(envKey)
	if err != nil {
		return err
	}

	// Create the KMS key
	commonName := fmt.Sprintf("%s-%s", m.config.Infrastructure.Company, envKey)
	description := fmt.Sprintf("KMS Key para SOPS - %s", commonName)

	// Build KMS key policy
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Sid":    "Enable IAM User Permissions",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"AWS": fmt.Sprintf("arn:aws:iam::%s:root", accountID),
				},
				"Action":   "kms:*",
				"Resource": "*",
			},
			{
				"Sid":    "Allow SOPS encryption/decryption",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"AWS": fmt.Sprintf("arn:aws:iam::%s:root", accountID),
				},
				"Action": []string{
					"kms:Encrypt",
					"kms:Decrypt",
					"kms:ReEncrypt*",
					"kms:GenerateDataKey*",
					"kms:DescribeKey",
				},
				"Resource": "*",
			},
		},
	}

	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal KMS policy: %w", err)
	}

	// Create KMS key
	createKeyOutput, err := kmsClient.CreateKey(context.Background(), &kms.CreateKeyInput{
		Description: aws.String(description),
		Policy:      aws.String(string(policyJSON)),
		Tags: []types.Tag{
			{
				TagKey:   aws.String("Name"),
				TagValue: aws.String(fmt.Sprintf("%s-secrets", commonName)),
			},
			{
				TagKey:   aws.String("Environment"),
				TagValue: aws.String(displayName),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create KMS key: %w", err)
	}

	createdKeyId := *createKeyOutput.KeyMetadata.KeyId

	// Create alias
	_, err = kmsClient.CreateAlias(context.Background(), &kms.CreateAliasInput{
		AliasName:   aws.String(alias),
		TargetKeyId: aws.String(createdKeyId),
	})
	if err != nil {
		// If alias creation fails, try to clean up the key
		_, _ = kmsClient.ScheduleKeyDeletion(context.Background(), &kms.ScheduleKeyDeletionInput{
			KeyId:               aws.String(createdKeyId),
			PendingWindowInDays: aws.Int32(7),
		})
		return fmt.Errorf("failed to create KMS alias: %w", err)
	}

	utils.PrintSuccess("✅ KMS key %s: CREATED (Key ID: %s)", alias, createdKeyId)

	// Mark as created
	m.kmsMutex.Lock()
	m.kmsCreatedCache[envKey] = true
	m.kmsMutex.Unlock()

	return nil
}

// getSecretsFilePath returns the path to the _secrets.yaml file for a layer
func (m *SOPSManager) getSecretsFilePath(layer *Layer) (string, error) {
	// Organization layer: single directory organization/_secrets.yaml (no env subdir)
	if layer.LayerType == "organization" {
		return filepath.Join(m.workingDir, "organization", "_secrets.yaml"), nil
	}
	if layer.LayerType == "security" {
		return filepath.Join(m.workingDir, "security", "_secrets.yaml"), nil
	}

	// Get environment directory name
	envConfig, exists := m.config.Infrastructure.Environments[layer.Environment]
	if !exists {
		return "", fmt.Errorf("environment '%s' not found", layer.Environment)
	}

	dirName := layer.Environment
	if envConfig.DirName != "" {
		dirName = envConfig.DirName
	} else if envConfig.Name != "" {
		dirName = strings.ToLower(envConfig.Name)
	}

	// Build path based on layer type
	var layerPath string
	switch layer.LayerType {
	case "base", "foundation":
		layerPath = filepath.Join(m.workingDir, layer.LayerType, dirName)
	case "project", "workload":
		// Get project/workload directory name
		var projectDirName string
		if layer.LayerType == "project" {
			// Find project in environment
			for _, project := range envConfig.Projects {
				projectKey := models.GetProjectKey(project)
				if projectKey == layer.Project {
					projectDirName = models.GetProjectDirectoryName(project)
					break
				}
			}
		} else {
			// Find workload in environment
			for _, workload := range envConfig.Workloads {
				workloadKey := models.GetWorkloadKey(workload)
				if workloadKey == layer.Project {
					projectDirName = models.GetWorkloadDirectoryName(workload)
					break
				}
			}
		}
		if projectDirName == "" {
			return "", fmt.Errorf("project/workload '%s' not found in environment '%s'", layer.Project, layer.Environment)
		}
		layerPath = filepath.Join(m.workingDir, layer.LayerType, projectDirName, dirName)
	default:
		return "", fmt.Errorf("unsupported layer type: %s", layer.LayerType)
	}

	return filepath.Join(layerPath, "_secrets.yaml"), nil
}

// getProfileForEnvironment returns the AWS profile name for an environment
func (m *SOPSManager) getProfileForEnvironment(envKey string) string {
	return fmt.Sprintf("%s-%s", m.config.Infrastructure.Client, envKey)
}

// decryptSecretsFile decrypts the _secrets.yaml file using SOPS
func (m *SOPSManager) decryptSecretsFile(filePath, envKey string) ([]byte, error) {
	profile := m.getProfileForEnvironment(envKey)
	configFile := ".aws/config"

	// Run sops -d to decrypt
	cmd := exec.Command("sops", "-d", filePath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("AWS_PROFILE=%s", profile),
		fmt.Sprintf("AWS_CONFIG_FILE=%s", configFile),
	)

	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("sops decryption failed: %s", string(exitError.Stderr))
		}
		return nil, fmt.Errorf("sops decryption failed: %w", err)
	}

	return output, nil
}

// encryptSecretsFile encrypts content and writes it to _secrets.yaml using SOPS
func (m *SOPSManager) encryptSecretsFile(filePath, envKey string, content []byte) error {
	// Ensure KMS key exists
	if err := m.ensureKMSKey(envKey); err != nil {
		return fmt.Errorf("failed to ensure KMS key: %w", err)
	}

	// Get KMS alias ARN
	kmsARN, err := m.getKMSAliasARN(envKey)
	if err != nil {
		return fmt.Errorf("failed to get KMS alias ARN: %w", err)
	}

	profile := m.getProfileForEnvironment(envKey)
	configFile := ".aws/config"

	// Create temporary file with unencrypted content
	tmpFile, err := os.CreateTemp("", "gocloud-sops-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Run sops -e to encrypt
	cmd := exec.Command("sops", "-e", "--kms", kmsARN, "--aws-profile", profile, tmpFile.Name())
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("AWS_CONFIG_FILE=%s", configFile),
	)

	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("sops encryption failed: %s", string(exitError.Stderr))
		}
		return fmt.Errorf("sops encryption failed: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write encrypted content to target file
	if err := os.WriteFile(filePath, output, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

// ListSecrets retrieves all secrets for a layer
func (m *SOPSManager) ListSecrets(layer *Layer) ([]Secret, error) {
	filePath, err := m.getSecretsFilePath(layer)
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []Secret{}, nil
	}

	// Decrypt file
	decryptedContent, err := m.decryptSecretsFile(filePath, layer.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secrets file: %w", err)
	}

	// Parse YAML
	var secretsMap map[string]interface{}
	if err := yaml.Unmarshal(decryptedContent, &secretsMap); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
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
func (m *SOPSManager) GetSecret(layer *Layer, key string) (interface{}, error) {
	secrets, err := m.ListSecrets(layer)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		if secret.Key == key {
			return secret.Value, nil
		}
	}

	return nil, fmt.Errorf("secret key '%s' not found in layer: %s", key, layer.Environment)
}

// SetSecret sets a secret value
func (m *SOPSManager) SetSecret(layer *Layer, key string, value interface{}) error {
	// Get current secrets
	secrets, err := m.ListSecrets(layer)
	if err != nil {
		// If file doesn't exist, start with empty map
		secrets = []Secret{}
	}

	// Convert to map for easier manipulation
	secretsMap := make(map[string]interface{})
	for _, secret := range secrets {
		secretsMap[secret.Key] = secret.Value
	}

	// Add or update the key
	secretsMap[key] = value

	// Convert to YAML
	yamlContent, err := yaml.Marshal(secretsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets to YAML: %w", err)
	}

	// Encrypt and write file
	filePath, err := m.getSecretsFilePath(layer)
	if err != nil {
		return err
	}

	return m.encryptSecretsFile(filePath, layer.Environment, yamlContent)
}

// DeleteSecret deletes a secret key
func (m *SOPSManager) DeleteSecret(layer *Layer, key string) error {
	// Get current secrets
	secrets, err := m.ListSecrets(layer)
	if err != nil {
		return fmt.Errorf("failed to get current secrets: %w", err)
	}

	// Convert to map for easier manipulation
	secretsMap := make(map[string]interface{})
	for _, secret := range secrets {
		secretsMap[secret.Key] = secret.Value
	}

	// Check if key exists
	if _, exists := secretsMap[key]; !exists {
		return fmt.Errorf("secret key '%s' not found in layer: %s", key, layer.Environment)
	}

	// Delete the key
	delete(secretsMap, key)

	// Convert to YAML
	yamlContent, err := yaml.Marshal(secretsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets to YAML: %w", err)
	}

	// Encrypt and write file
	filePath, err := m.getSecretsFilePath(layer)
	if err != nil {
		return err
	}

	return m.encryptSecretsFile(filePath, layer.Environment, yamlContent)
}

// CheckSecrets checks if secrets exist for a layer without retrieving them
// Note: KMS keys should be verified beforehand using EnsureAllKMSKeys
func (m *SOPSManager) CheckSecrets(layer *Layer) (string, error) {
	filePath, err := m.getSecretsFilePath(layer)
	if err != nil {
		return "", err
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "NOT_FOUND", nil
	}

	// Try to decrypt to verify it's valid
	_, err = m.decryptSecretsFile(filePath, layer.Environment)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secrets file: %w", err)
	}

	return "EXISTS", nil
}

// InitSecrets initializes secrets for a layer with empty YAML
// Note: KMS keys should be verified/created beforehand using EnsureAllKMSKeys
func (m *SOPSManager) InitSecrets(layer *Layer) error {
	filePath, err := m.getSecretsFilePath(layer)
	if err != nil {
		return err
	}

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("secrets file already exists")
	}

	// Ensure KMS key exists (silently, assuming it was already checked)
	if err := m.ensureKMSKey(layer.Environment); err != nil {
		return fmt.Errorf("failed to ensure KMS key: %w", err)
	}

	// Create empty YAML content
	emptyYAML := []byte("{}\n")

	// Encrypt and write file
	return m.encryptSecretsFile(filePath, layer.Environment, emptyYAML)
}

// EditSecrets opens a text editor to edit the YAML secrets directly
func (m *SOPSManager) EditSecrets(layer *Layer) error {
	// Verify AWS credentials before opening the editor (avoids editing and then failing on save)
	_, _, err := m.getKMSKeyExists(layer.Environment)
	if err != nil {
		if utils.IsCredentialError(err) {
			return fmt.Errorf("%s", formatCredentialError(layer))
		}
		return fmt.Errorf("failed to verify AWS access for layer: %w", err)
	}

	// Get current secrets (file may not exist yet; ListSecrets returns [], nil in that case)
	secrets, err := m.ListSecrets(layer)
	if err != nil {
		if utils.IsCredentialError(err) {
			return fmt.Errorf("%s", formatCredentialError(layer))
		}
		return fmt.Errorf("failed to read existing secrets: %w", err)
	}

	// Convert secrets to YAML map
	secretsMap := make(map[string]interface{})
	for _, secret := range secrets {
		secretsMap[secret.Key] = secret.Value
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(secretsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets to YAML: %w", err)
	}

	// Use the same edit logic as SSM manager but with YAML instead of JSON
	return m.editSecretsYAML(layer, secretsMap, yamlData)
}

// editSecretsYAML handles the YAML editing flow (similar to EditSecrets in edit.go but for YAML)
func (m *SOPSManager) editSecretsYAML(layer *Layer, secretsMap map[string]interface{}, initialYAML []byte) error {
	// Create temporary file with safe name
	safeName := strings.ReplaceAll(layer.Environment, "/", "-")
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("gocloud-secrets-%s-*.yaml", safeName))
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			// Log but don't fail
		}
	}()

	// Write current YAML to temp file
	if _, err := tmpFile.Write(initialYAML); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			return fmt.Errorf("failed to close temp file: %w", closeErr)
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
	filePath, _ := m.getSecretsFilePath(layer)
	utils.PrintWarning("Opening editor for layer: %s", layer.Environment)
	utils.PrintWarning("Secrets file: %s", filePath)
	utils.PrintWarning("Editor: %s", editor)
	fmt.Println()

	// Main editing loop
	var newSecretsMap map[string]interface{}
	for {
		// Open editor
		if err := openEditor(editor, tmpFile.Name()); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		// Read and validate modified YAML
		modifiedData, err := os.ReadFile(tmpFile.Name())
		if err != nil {
			return fmt.Errorf("failed to read modified file: %w", err)
		}

		// Try to validate YAML
		if err := yaml.Unmarshal(modifiedData, &newSecretsMap); err != nil {
			// YAML is invalid, show error and ask user to fix
			utils.PrintError("❌ Invalid YAML format detected!")
			utils.PrintError("Error: %v", err)
			fmt.Println()

			utils.PrintWarning("The file contains invalid YAML. Please fix the format and try again.")
			fmt.Println()

			// Show the problematic content with line numbers
			utils.PrintInfo("Content that needs to be fixed:")
			lines := strings.Split(string(modifiedData), "\n")
			for i, line := range lines {
				utils.PrintText("%2d: %s\n", i+1, line)
			}
			fmt.Println()

			// Ask if user wants to retry
			utils.PrintWarning("Press Enter to reopen editor and fix the YAML, or Ctrl+C to cancel...")
			if _, err := fmt.Scanln(); err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			// Loop will continue and validate again
			continue
		}

		// YAML is valid, check if there are actual changes
		if !hasChanges(secretsMap, newSecretsMap) {
			utils.PrintWarning("No changes detected. Exiting.")
			return nil
		}

		// Show preview of changes
		utils.PrintSuccess("✅ YAML format is valid!")
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

		// Convert back to YAML for encryption
		finalYAML, err := yaml.Marshal(newSecretsMap)
		if err != nil {
			return fmt.Errorf("failed to marshal final YAML: %w", err)
		}

		// Encrypt and write file
		filePath, err := m.getSecretsFilePath(layer)
		if err != nil {
			return fmt.Errorf("failed to get secrets file path: %w", err)
		}

		if err := m.encryptSecretsFile(filePath, layer.Environment, finalYAML); err != nil {
			return fmt.Errorf("failed to encrypt and save secrets: %w", err)
		}

		utils.PrintSuccess("✅ Changes applied successfully!")
		utils.PrintSuccess("Secrets file updated: %s", filePath)

		return nil
	}
}

// ParseLayerPath parses a layer path string into a Layer struct
func (m *SOPSManager) ParseLayerPath(layerPath string) (*Layer, error) {
	return ParseLayerPath(layerPath, m.config)
}
