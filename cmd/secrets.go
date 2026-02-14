package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gocloud-cli/internal/config"
	"gocloud-cli/internal/generator"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/secrets"
	"gocloud-cli/internal/utils"
)

var errSecretsDisabled = errors.New("secrets disabled")

var (
	secretsConfig   string
	secretsCheckAll bool
	secretsCheckEnv string
	secretsInitAll  bool
	secretsInitEnv  string
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets (SSM or SOPS) for GoCloud infrastructure layers",
	Long: `Manage secrets for GoCloud infrastructure layers (SSM Parameter Store or SOPS).

Layer path format:
• base/production - Base layer in production
• foundation/staging - Foundation layer in staging
• project/core/production - Core project in production
• workload/webapp/staging - Webapp workload in staging

`,
	ValidArgsFunction: completeSecretsSubcommands,
}

var secretsListCmd = &cobra.Command{
	Use:               "list [layer-path]",
	Short:             "List secrets for a specific layer",
	Long:              `List all secrets for a specific layer path.`,
	Args:              cobra.ExactArgs(1),
	RunE:              runSecretsList,
	ValidArgsFunction: completeLayerPaths,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

var secretsGetCmd = &cobra.Command{
	Use:               "get [layer-path] [key]",
	Short:             "Get a specific secret value",
	Long:              `Get the value of a specific secret key.`,
	Args:              cobra.ExactArgs(2),
	RunE:              runSecretsGet,
	ValidArgsFunction: completeSecretsGetArgs,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

var secretsSetCmd = &cobra.Command{
	Use:               "set [layer-path] [key] [value]",
	Short:             "Set a secret value",
	Long:              `Set the value of a secret key.`,
	Args:              cobra.ExactArgs(3),
	RunE:              runSecretsSet,
	ValidArgsFunction: completeSecretsSetArgs,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

var secretsDeleteCmd = &cobra.Command{
	Use:               "delete [layer-path] [key]",
	Short:             "Delete a secret key",
	Long:              `Delete a specific secret key from a layer.`,
	Args:              cobra.ExactArgs(2),
	RunE:              runSecretsDelete,
	ValidArgsFunction: completeSecretsDeleteArgs,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

var secretsEditCmd = &cobra.Command{
	Use:               "edit [layer-path]",
	Short:             "Edit secrets using external text editor",
	Long:              `Edit secrets for a specific layer using an external text editor.`,
	Args:              cobra.ExactArgs(1),
	RunE:              runSecretsEdit,
	ValidArgsFunction: completeLayerPaths,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

var secretsCheckCmd = &cobra.Command{
	Use:               "check [layer-path]",
	Short:             "Check if secrets exist for a specific layer or all layers",
	Long:              `Check if secrets exist for a specific layer or all layers.`,
	Args:              cobra.MaximumNArgs(1),
	RunE:              runSecretsCheck,
	ValidArgsFunction: completeLayerPaths,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

var secretsInitCmd = &cobra.Command{
	Use:               "init [layer-path]",
	Short:             "Initialize secrets for a specific layer or all layers",
	Long:              `Initialize secrets for a specific layer or all layers with empty JSON.`,
	Args:              cobra.MaximumNArgs(1),
	RunE:              runSecretsInit,
	ValidArgsFunction: completeLayerPaths,
	SilenceUsage:      true,
	SilenceErrors:     true,
}

func init() {
	// Command is registered in root.go

	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsDeleteCmd)
	secretsCmd.AddCommand(secretsEditCmd)
	secretsCmd.AddCommand(secretsCheckCmd)
	secretsCmd.AddCommand(secretsInitCmd)

	// Global flags
	secretsCmd.PersistentFlags().StringVarP(&secretsConfig, "config", "c", "gocloud.yaml", "Configuration file path")

	// Check command flags
	secretsCheckCmd.Flags().BoolVar(&secretsCheckAll, "all", false, "Check all layers in the project")
	secretsCheckCmd.Flags().StringVar(&secretsCheckEnv, "environment", "", "Check all layers for a specific environment")

	// Init command flags
	secretsInitCmd.Flags().BoolVar(&secretsInitAll, "all", false, "Initialize secrets for all layers in the project")
	secretsInitCmd.Flags().StringVar(&secretsInitEnv, "environment", "", "Initialize secrets for all layers in a specific environment")
}

// isContentError checks if an error is related to content not found (should not show help)
func isContentError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	contentErrorPatterns := []string{
		"not found",
		"does not exist",
		"not found in layer",
		"parameter not found",
		"ParameterNotFound",
	}

	for _, pattern := range contentErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// handleCredentialError prints a simple error message and exits
func handleCredentialError(err error) {
	if utils.IsCredentialError(err) {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

// handleContentError prints a simple error message and exits without showing help
func handleContentError(err error) {
	if isContentError(err) {
		// Check if it's a ParameterNotFound error and show a cleaner message
		if strings.Contains(err.Error(), "ParameterNotFound") {
			fmt.Fprintf(os.Stderr, "Error: SSM parameter does not exist\n")
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		}
		os.Exit(1)
	}
}

func runSecretsList(cmd *cobra.Command, args []string) error {
	layerPath := args[0]
	_, manager, layer, err := getSecretsManagerAndLayer(layerPath)
	if err != nil {
		if errors.Is(err, errSecretsDisabled) {
			return nil
		}
		return err
	}

	// List secrets
	secretsList, err := manager.ListSecrets(layer)
	if err != nil {
		handleCredentialError(err)
		handleContentError(err)
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	// Display results
	if len(secretsList) == 0 {
		utils.PrintWarning("No secrets found for layer: %s", layerPath)
		return nil
	}

	utils.PrintSuccess("Secrets for layer: %s", layerPath)
	if layer.SSMParameter != "" {
		utils.PrintSuccess("SSM Parameter: %s", layer.SSMParameter)
	}
	fmt.Println()

	for _, secret := range secretsList {
		utils.PrintText("  %s\n", secret.Key)
	}

	return nil
}

func runSecretsGet(cmd *cobra.Command, args []string) error {
	layerPath := args[0]
	key := args[1]
	_, manager, layer, err := getSecretsManagerAndLayer(layerPath)
	if err != nil {
		if errors.Is(err, errSecretsDisabled) {
			return nil
		}
		return err
	}

	value, err := manager.GetSecret(layer, key)
	if err != nil {
		handleCredentialError(err)
		handleContentError(err)
		return fmt.Errorf("failed to get secret: %w", err)
	}

	utils.PrintSuccess("Secret: %s", key)
	utils.PrintSuccess("Layer: %s", layerPath)
	if layer.SSMParameter != "" {
		utils.PrintSuccess("SSM Parameter: %s", layer.SSMParameter)
	}
	fmt.Println()
	utils.PrintText("Value: %v", value)

	return nil
}

func runSecretsSet(cmd *cobra.Command, args []string) error {
	layerPath := args[0]
	key := args[1]
	value := args[2]
	_, manager, layer, err := getSecretsManagerAndLayer(layerPath)
	if err != nil {
		if errors.Is(err, errSecretsDisabled) {
			return nil
		}
		return err
	}

	err = manager.SetSecret(layer, key, value)
	if err != nil {
		handleCredentialError(err)
		handleContentError(err)
		return fmt.Errorf("failed to set secret: %w", err)
	}

	utils.PrintSuccess("✅ Secret '%s' set successfully for layer: %s", key, layerPath)
	return nil
}

func runSecretsDelete(cmd *cobra.Command, args []string) error {
	layerPath := args[0]
	key := args[1]
	_, manager, layer, err := getSecretsManagerAndLayer(layerPath)
	if err != nil {
		if errors.Is(err, errSecretsDisabled) {
			return nil
		}
		return err
	}

	err = manager.DeleteSecret(layer, key)
	if err != nil {
		handleCredentialError(err)
		handleContentError(err)
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	utils.PrintSuccess("✅ Secret '%s' deleted successfully from layer: %s", key, layerPath)
	return nil
}

func runSecretsEdit(cmd *cobra.Command, args []string) error {
	layerPath := args[0]
	_, manager, layer, err := getSecretsManagerAndLayer(layerPath)
	if err != nil {
		if errors.Is(err, errSecretsDisabled) {
			return nil
		}
		return err
	}

	err = manager.EditSecrets(layer)
	if err != nil {
		handleCredentialError(err)
		handleContentError(err)
		return err
	}
	return nil
}

func runSecretsCheck(cmd *cobra.Command, args []string) error {
	config, err := loadSecretsConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	var layersToCheck []string
	if secretsCheckAll {
		layersToCheck = getAllLayersFromConfig(config)
	} else if secretsCheckEnv != "" {
		layersToCheck = getLayersForEnvironment(config, secretsCheckEnv)
	} else if len(args) > 0 {
		layersToCheck = []string{args[0]}
	} else {
		fmt.Println("Must specify a layer path, --environment, or --all")
		return nil
	}

	var enabledLayers []string
	for _, layerPath := range layersToCheck {
		if err := checkSecretsEnabled(config, layerPath); err != nil {
			continue
		}
		enabledLayers = append(enabledLayers, layerPath)
	}

	if len(enabledLayers) == 0 {
		fmt.Println("No layers with secrets enabled found")
		return nil
	}
	layersToCheck = enabledLayers

	utils.PrintInfo("🔍 Checking secrets status...")
	existsCount := 0
	notFoundCount := 0
	noAccessCount := 0
	workingDir := getSecretsWorkingDir()

	for _, layerPath := range layersToCheck {
		layer, err := secrets.ParseLayerPath(layerPath, config)
		if err != nil {
			continue
		}
		manager, err := secrets.NewManagerForLayer(config, layerPath, workingDir)
		if err != nil {
			continue
		}

		status, err := manager.CheckSecrets(layer)
		if err != nil {
			if utils.IsCredentialError(err) {
				utils.PrintWarning("⚠️  %s: NO_ACCESS (profile not logged in)", layerPath)
				noAccessCount++
			}
			continue
		}

		// Display result
		switch status {
		case "EXISTS":
			utils.PrintSuccess("✅ %s: EXISTS", layerPath)
			existsCount++
		case "NOT_FOUND":
			utils.PrintError("❌ %s: NOT_FOUND", layerPath)
			notFoundCount++
		}
	}

	// Show summary
	fmt.Println()
	total := existsCount + notFoundCount + noAccessCount
	if total > 0 {
		utils.PrintInfo("Summary: %d/%d layers have secrets (%d missing, %d skipped)",
			existsCount, total, notFoundCount, noAccessCount)
	}

	return nil
}

func runSecretsInit(cmd *cobra.Command, args []string) error {
	config, err := loadSecretsConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	var layersToInit []string
	if secretsInitAll {
		layersToInit = getAllLayersFromConfig(config)
	} else if secretsInitEnv != "" {
		layersToInit = getLayersForEnvironment(config, secretsInitEnv)
	} else if len(args) > 0 {
		layersToInit = []string{args[0]}
	} else {
		fmt.Println("Must specify a layer path, --environment, or --all")
		return nil
	}

	var enabledLayers []string
	for _, layerPath := range layersToInit {
		if err := checkSecretsEnabled(config, layerPath); err != nil {
			continue
		}
		enabledLayers = append(enabledLayers, layerPath)
	}
	if len(enabledLayers) == 0 {
		fmt.Println("No layers with secrets enabled found")
		return nil
	}
	layersToInit = enabledLayers

	// SOPS: verify/create KMS keys for all environments that use SOPS before initializing any layer
	envKeysForSOPS := make(map[string]bool)
	for _, layerPath := range layersToInit {
		secretsConfig, err := secrets.ResolveSecretsConfig(config, layerPath)
		if err != nil {
			continue
		}
		if secretsConfig.Type != "sops" {
			continue
		}
		_, _, env, ok := parseLayerPathComponents(layerPath)
		if !ok {
			continue
		}
		envKeysForSOPS[env] = true
	}
	if len(envKeysForSOPS) > 0 {
		utils.PrintInfo("🚀 Initializing KMSs...")
		envKeys := make([]string, 0, len(envKeysForSOPS))
		for k := range envKeysForSOPS {
			envKeys = append(envKeys, k)
		}
		sopsManager, err := secrets.NewSOPSManager(config, getSecretsWorkingDir())
		if err != nil {
			return fmt.Errorf("failed to create SOPS manager for KMS check: %w", err)
		}
		if err := sopsManager.EnsureAllKMSKeys(envKeys, true); err != nil {
			return fmt.Errorf("KMS verification/creation failed: %w", err)
		}
	}

	utils.PrintInfo("🚀 Initializing secrets...")
	initCount := 0
	skipCount := 0
	errorCount := 0
	workingDir := getSecretsWorkingDir()

	for _, layerPath := range layersToInit {
		layer, err := secrets.ParseLayerPath(layerPath, config)
		if err != nil {
			utils.PrintError("❌ %s: Invalid layer path", layerPath)
			errorCount++
			continue
		}
		manager, err := secrets.NewManagerForLayer(config, layerPath, workingDir)
		if err != nil {
			utils.PrintError("❌ %s: Failed to create manager - %v", layerPath, err)
			errorCount++
			continue
		}

		status, err := manager.CheckSecrets(layer)
		if err != nil {
			if utils.IsCredentialError(err) {
				utils.PrintWarning("⚠️  %s: SKIPPED (profile not logged in)", layerPath)
				skipCount++
			} else {
				utils.PrintError("❌ %s: Error checking secrets - %v", layerPath, err)
				errorCount++
			}
			continue
		}

		// Skip if already exists
		if status == "EXISTS" {
			utils.PrintWarning("⚠️  %s: SKIPPED (already exists)", layerPath)
			skipCount++
			continue
		}

		// Initialize secrets
		err = manager.InitSecrets(layer)
		if err != nil {
			if strings.Contains(err.Error(), "parameter already exists") {
				utils.PrintWarning("⚠️  %s: SKIPPED (already exists)", layerPath)
				skipCount++
			} else if utils.IsCredentialError(err) {
				utils.PrintWarning("⚠️  %s: SKIPPED (profile not logged in)", layerPath)
				skipCount++
			} else {
				utils.PrintError("❌ %s: Failed to initialize - %v", layerPath, err)
				errorCount++
			}
			continue
		}

		utils.PrintSuccess("✅ %s: INITIALIZED", layerPath)
		initCount++
	}

	// Show summary
	fmt.Println()
	total := initCount + skipCount + errorCount
	if total > 0 {
		utils.PrintInfo("Summary: %d initialized, %d skipped, %d errors",
			initCount, skipCount, errorCount)
	}

	return nil
}

func loadSecretsConfig() (*models.Config, error) {
	return config.LoadConfigWithPathAndAWS(secretsConfig)
}

// getSecretsWorkingDir returns the working directory (config file dir or current dir)
func getSecretsWorkingDir() string {
	abs, err := filepath.Abs(secretsConfig)
	if err != nil {
		return "."
	}
	return filepath.Dir(abs)
}

// getSecretsManagerAndLayer loads config, checks secrets enabled, and returns manager + layer for the given layerPath.
// Returns errSecretsDisabled when secrets are disabled for the layer (caller should exit with success).
func getSecretsManagerAndLayer(layerPath string) (*models.Config, secrets.SecretsManagerInterface, *secrets.Layer, error) {
	cfg, err := loadSecretsConfig()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	if !checkSecretsEnabledSilent(cfg, layerPath) {
		return nil, nil, nil, errSecretsDisabled
	}
	manager, err := secrets.NewManagerForLayer(cfg, layerPath, getSecretsWorkingDir())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize secrets manager: %w", err)
	}
	layer, err := secrets.ParseLayerPath(layerPath, cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid layer path %q: %w", layerPath, err)
	}
	return cfg, manager, layer, nil
}

// parseLayerPathComponents parses layerPath into layerType, project, env. Returns ok=false if invalid.
// Accepts: "organization" (1 part), "layer/env" (2 parts), "layer/project/env" (3 parts).
func parseLayerPathComponents(layerPath string) (layerType, project, env string, ok bool) {
	parts := strings.Split(layerPath, "/")
	switch len(parts) {
	case 1:
		if parts[0] != "organization" {
			return "", "", "", false
		}
		return "organization", "", "org", true
	case 2:
		return parts[0], "", parts[1], true
	case 3:
		return parts[0], parts[1], parts[2], true
	default:
		return "", "", "", false
	}
}

// checkSecretsEnabled verifies if secrets are enabled for a specific layer path
func checkSecretsEnabled(config *models.Config, layerPath string) error {
	layerType, project, env, ok := parseLayerPathComponents(layerPath)
	if !ok {
		utils.PrintText("Invalid layer path format: %s\n", layerPath)
		return nil
	}
	if config.Infrastructure == nil || !shouldGenerateSecretsForPath(config.Infrastructure, layerType, project, env) {
		return fmt.Errorf("secrets are disabled for %s - check your gocloud.yaml configuration", layerPath)
	}
	return nil
}

// checkSecretsEnabledSilent verifies if secrets are enabled and prints a message if disabled.
// Returns true if enabled, false if disabled (and prints message).
func checkSecretsEnabledSilent(config *models.Config, layerPath string) bool {
	layerType, project, env, ok := parseLayerPathComponents(layerPath)
	if !ok {
		utils.PrintText("Invalid layer path format: %s\n", layerPath)
		return false
	}
	if config.Infrastructure == nil || !shouldGenerateSecretsForPath(config.Infrastructure, layerType, project, env) {
		utils.PrintText("Secrets are disabled for %s - check your gocloud.yaml configuration\n", layerPath)
		return false
	}
	return true
}

// shouldGenerateSecretsForPath determines if secrets should be generated for a specific path
// This is a simplified version of the generator's shouldGenerateSecrets function
func shouldGenerateSecretsForPath(infra *models.InfrastructureConfig, layerType, project, env string) bool {
	// Organization layer: enabled when organization.aws_account is set and not explicitly disabled by layers.organization
	if layerType == "organization" {
		if infra.Organization == nil || infra.Organization.AWSAccount == "" {
			return false
		}
		if infra.Layers != nil && infra.Layers.Organization != nil && !*infra.Layers.Organization {
			return false
		}
		if infra.EnableSecrets != nil {
			return *infra.EnableSecrets
		}
		return true
	}

	// Get environment configuration
	envConfig, exists := infra.Environments[env]
	if !exists {
		// If environment doesn't exist, use infrastructure default
		if infra.EnableSecrets != nil {
			return *infra.EnableSecrets
		}
		// Default to true if not specified
		return true
	}

	// Check if it's a workload (has project)
	if project != "" {
		// Find the workload in the environment
		for _, workloadInterface := range envConfig.Workloads {
			var workloadName string
			var workloadConfig *models.WorkloadItem

			// Normalize map[interface{}]interface{} (YAML unmarshal) to map[string]interface{}
			var w map[string]interface{}
			switch item := workloadInterface.(type) {
			case string:
				workloadName = item
			case map[interface{}]interface{}:
				w = models.ToMapStringInterface(item)
			case map[string]interface{}:
				w = item
			}
			if w != nil {
				// Handle case where workload is defined as: - workload-name: {enable_secrets: false}
				if len(w) == 1 {
					for key, value := range w {
						workloadName = key
						if valueMap, ok := value.(map[string]interface{}); ok {
							if enableSecrets, ok := valueMap["enable_secrets"].(bool); ok {
								workloadConfig = &models.WorkloadItem{
									Name:          key,
									EnableSecrets: &enableSecrets,
								}
							}
						}
					}
				} else {
					// Handle case where workload is defined as: - {name: workload-name, enable_secrets: false}
					if name, ok := w["name"].(string); ok {
						workloadName = name
						if enableSecrets, ok := w["enable_secrets"].(bool); ok {
							workloadConfig = &models.WorkloadItem{
								Name:          name,
								EnableSecrets: &enableSecrets,
							}
						}
					}
				}
			}

			if workloadName == project {
				// If workload has explicit enable_secrets setting, use it
				if workloadConfig != nil && workloadConfig.EnableSecrets != nil {
					return *workloadConfig.EnableSecrets
				}
				// Otherwise, fall through to environment level
				break
			}
		}
	}

	// Check environment level
	if envConfig.EnableSecrets != nil {
		return *envConfig.EnableSecrets
	}

	// Fall back to infrastructure level
	if infra.EnableSecrets != nil {
		return *infra.EnableSecrets
	}
	// Default to true if not specified
	return true
}

// completeLayerPaths provides bash completion for layer paths
func completeLayerPaths(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Load configuration
	config, err := loadSecretsConfig()
	if err != nil {
		// If config can't be loaded, return empty completion
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Use the same logic as the generator to get enabled layers
	completions := generator.GetEnabledLayersFromConfig(config)

	// Filter completions based on what user has typed
	var filtered []string
	for _, completion := range completions {
		if strings.HasPrefix(completion, toComplete) {
			filtered = append(filtered, completion)
		}
	}

	// If there's exactly one completion, allow space after it
	if len(filtered) == 1 {
		return filtered, cobra.ShellCompDirectiveDefault
	}

	// For multiple completions, don't add space
	return filtered, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

// completeSecretsSubcommands provides bash completion for secrets subcommands
func completeSecretsSubcommands(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	subcommands := []string{
		"list\tList secrets for a specific layer",
		"get\tGet a specific secret value",
		"set\tSet a secret value",
		"delete\tDelete a secret",
		"edit\tEdit secrets interactively",
		"check\tCheck if secrets exist for a specific layer or all layers",
		"init\tInitialize secrets for a specific layer or all layers",
	}

	// Filter based on what user has typed
	var filtered []string
	for _, subcommand := range subcommands {
		if strings.HasPrefix(subcommand, toComplete) {
			filtered = append(filtered, subcommand)
		}
	}

	// Always allow space after completion to enable flag completion
	return filtered, cobra.ShellCompDirectiveDefault
}

// completeSecretsGetArgs provides dynamic completion for secrets get command
func completeSecretsGetArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// If we have 0 arguments, complete layer paths
	if len(args) == 0 {
		return completeLayerPaths(cmd, args, toComplete)
	}

	// If we have 1 or more arguments, complete secret keys for the given layer
	if len(args) >= 1 {
		return completeSecretKeys(args[0], toComplete)
	}

	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

// completeSecretsSetArgs provides dynamic completion for secrets set command
func completeSecretsSetArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// If we have 0 arguments, complete layer paths
	if len(args) == 0 {
		return completeLayerPaths(cmd, args, toComplete)
	}

	// If we have 1 argument, complete secret keys for the given layer
	if len(args) == 1 {
		return completeSecretKeys(args[0], toComplete)
	}

	// If we have 2 or more arguments, no completion needed for value
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

// completeSecretsDeleteArgs provides dynamic completion for secrets delete command
func completeSecretsDeleteArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// If we have 0 arguments, complete layer paths
	if len(args) == 0 {
		return completeLayerPaths(cmd, args, toComplete)
	}

	// If we have 1 or more arguments, complete secret keys for the given layer
	return completeSecretKeys(args[0], toComplete)
}

// completeSecretKeys gets secret keys for a specific layer by calling the secrets list command
func completeSecretKeys(layerPath, toComplete string) ([]string, cobra.ShellCompDirective) {
	config, err := loadSecretsConfig()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	manager, err := secrets.NewManagerForLayer(config, layerPath, getSecretsWorkingDir())
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	layer, err := secrets.ParseLayerPath(layerPath, config)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Get secrets for the layer
	secretsList, err := manager.ListSecrets(layer)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	// Extract keys and filter based on what user has typed
	var filtered []string
	for _, secret := range secretsList {
		if strings.HasPrefix(secret.Key, toComplete) {
			filtered = append(filtered, secret.Key)
		}
	}

	// If there's exactly one completion, allow space after it
	if len(filtered) == 1 {
		return filtered, cobra.ShellCompDirectiveDefault
	}

	// For multiple completions, don't add space
	return filtered, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

// getAllLayersFromConfig returns all enabled layer paths from the configuration
// This function reuses the same logic as the generator to ensure consistency
func getAllLayersFromConfig(config *models.Config) []string {
	return generator.GetEnabledLayersFromConfig(config)
}

// getLayersForEnvironment returns all layer paths for a specific environment
func getLayersForEnvironment(config *models.Config, environment string) []string {
	var layers []string

	// Check if infrastructure config exists
	if config.Infrastructure == nil {
		return layers
	}

	// Check if environment exists
	env, exists := config.Infrastructure.Environments[environment]
	if !exists {
		return layers
	}

	// Base layers for this environment
	layers = append(layers, fmt.Sprintf("base/%s", environment))
	layers = append(layers, fmt.Sprintf("foundation/%s", environment))

	// Project layers for this environment
	for _, project := range env.Projects {
		layers = append(layers, fmt.Sprintf("project/%s/%s", project, environment))
	}

	// Workload layers for this environment
	for _, workload := range env.Workloads {
		layers = append(layers, fmt.Sprintf("workload/%s/%s", workload, environment))
	}

	return layers
}
