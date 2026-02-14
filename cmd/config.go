package cmd

import (
	"fmt"
	"path/filepath"

	"gocloud-cli/internal/config"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/utils"
	"gocloud-cli/internal/validation"

	"github.com/spf13/cobra"
)

var (
	configOutputFile       string
	configWorkingDir       string
	configRegion           string
	configCompany          string
	configSkipEnvironments bool
	configSkipAWSSSO       bool
	strictMode             bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage GoCloud configuration files",
	Long:  `Manage GoCloud configuration files.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Create a new GoCloud configuration interactively",
	Long:  `Create a new GoCloud configuration file with interactive prompts.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigInit,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate GoCloud configuration file",
	Long:  `Validate a GoCloud configuration file for errors and warnings.`,
	RunE:  runConfigValidate,
}

func init() {
	// Config init command flags
	configInitCmd.Flags().StringVarP(&configOutputFile, "output", "o", "", "Output file path (default: project-name/gocloud.yaml)")
	configInitCmd.Flags().StringVar(&configWorkingDir, "working-dir", ".", "Working directory for the project")
	configInitCmd.Flags().StringVar(&configRegion, "region", "us-east-1", "AWS region")
	configInitCmd.Flags().StringVar(&configCompany, "company", "", "Company prefix")
	configInitCmd.Flags().BoolVar(&configSkipEnvironments, "skip-environments", false, "Skip environment configuration (CLI only)")
	configInitCmd.Flags().BoolVar(&configSkipAWSSSO, "skip-aws-sso", false, "Skip AWS SSO configuration")

	// Config validate command flags
	configValidateCmd.Flags().StringVarP(&cfgFile, "config", "c", "gocloud.yaml", "Configuration file to validate")
	configValidateCmd.Flags().BoolVar(&strictMode, "strict", false, "Strict validation (treat warnings as errors)")

	// Add subcommands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	utils.PrintInfo("\n🚀 GoCloud Configuration Setup")
	utils.PrintInfo("============================\n")

	// Validate project name
	if err := utils.ValidateProjectName(projectName); err != nil {
		return fmt.Errorf("invalid project name: %w", err)
	}

	// Determine output file path
	outputFile := configOutputFile
	if outputFile == "" {
		var err error
		outputFile, err = config.GetDefaultConfigPath(projectName)
		if err != nil {
			return fmt.Errorf("invalid project name: %w", err)
		}
	}

	// Check if file already exists
	configManager := config.NewManager()
	if configManager.ConfigExists(outputFile) {
		utils.PrintWarning("⚠️  Configuration file already exists: %s", outputFile)
		overwrite, err := utils.PromptYesNo("Do you want to overwrite it? (y/N)", false)
		if err != nil {
			return fmt.Errorf("failed to get user input: %w", err)
		}
		if !overwrite {
			utils.PrintSuccess("Configuration creation cancelled.")
			return nil
		}
	}

	// Create configuration interactively
	config, err := createInteractiveConfig(projectName)
	if err != nil {
		return fmt.Errorf("configuration creation failed: %w", err)
	}

	// Save configuration
	if err := configManager.SaveConfig(config, outputFile); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	utils.PrintSuccess("✅ Configuration saved to: %s", outputFile)

	// Show next steps
	utils.PrintInfo("\n📋 Next Steps:")
	utils.PrintText("  1. Review the configuration file: %s\n", outputFile)
	utils.PrintText("  2. Customize the configuration as needed\n")
	utils.PrintText("  3. Run: cd %s && gocloud generate\n", filepath.Dir(outputFile))

	utils.PrintSuccess("\n🎉 Configuration setup completed successfully!")

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	utils.PrintInfo("\n🔍 GoCloud Configuration Validation")
	utils.PrintInfo("==================================\n")

	// Use default config file if not specified
	if cfgFile == "" {
		cfgFile = "gocloud.yaml"
	}

	// Load configuration file with validation
	configManager := config.NewManager()
	_, validationResult, err := configManager.LoadConfigWithValidation(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration file '%s': %w", cfgFile, err)
	}

	utils.PrintSuccess("✅ Configuration file loaded: %s", cfgFile)

	// Display results
	displayValidationResults(validationResult, strictMode)

	// Return appropriate exit code
	if len(validationResult.Errors) > 0 || (strictMode && len(validationResult.Warnings) > 0) {
		// This is a real error that needs to be fixed, so we keep the error behavior
		return fmt.Errorf("validation failed with %d errors and %d warnings", len(validationResult.Errors), len(validationResult.Warnings))
	}

	utils.PrintSuccess("\n🎉 Configuration validation passed!")
	return nil
}

// createInteractiveConfig creates a new configuration interactively
func createInteractiveConfig(projectName string) (*models.Config, error) {
	config := models.DefaultConfig()

	// CLI Configuration
	utils.PrintInfo("\n⚙️  CLI Configuration")
	utils.PrintText("Configure CLI behavior and settings.\n")

	// Working directory
	workingDir, err := utils.PromptWithDefault("Working directory", configWorkingDir)
	if err != nil {
		return nil, err
	}
	config.CLI.WorkingDir = workingDir

	// Auto backup
	autoBackup, err := utils.PromptYesNo("Enable automatic backups? (Y/n)", true)
	if err != nil {
		return nil, err
	}
	config.CLI.AutoBackup = autoBackup

	if autoBackup {
		backupDir, err := utils.PromptWithDefault("Backup directory", config.CLI.BackupDir)
		if err != nil {
			return nil, err
		}
		config.CLI.BackupDir = backupDir
	}

	// Verbose mode
	verbose, err := utils.PromptYesNo("Enable verbose output? (y/N)", false)
	if err != nil {
		return nil, err
	}
	config.CLI.Verbose = verbose

	// Debug mode
	debug, err := utils.PromptYesNo("Enable debug mode? (y/N)", false)
	if err != nil {
		return nil, err
	}
	config.CLI.Debug = debug

	// Infrastructure Configuration
	utils.PrintInfo("\n🏗️  Infrastructure Configuration")
	utils.PrintText("Configure your infrastructure project settings.\n")

	// Client name
	clientName, err := utils.PromptWithDefault("Client name", projectName)
	if err != nil {
		return nil, err
	}
	config.Infrastructure.Client = clientName

	// Company prefix
	var companyPrefix string
	if configCompany != "" {
		// If provided via flag, validate it
		if err := validation.ValidateCompanyPrefix(configCompany); err != nil {
			return nil, fmt.Errorf("invalid company prefix: %w", err)
		}
		companyPrefix = configCompany
	} else {
		// Otherwise prompt for it with validation
		companyPrefix, err = utils.PromptWithValidationRequired("Company prefix (2-10 lowercase letters)", validation.ValidateCompanyPrefix)
		if err != nil {
			return nil, err
		}
	}
	config.Infrastructure.Company = companyPrefix

	// AWS Region
	region, err := utils.PromptWithDefault("AWS Region", configRegion)
	if err != nil {
		return nil, err
	}
	config.Infrastructure.Region = region

	// Infrastructure Version
	version, err := utils.PromptWithDefault("Infrastructure Version", "0.17.0")
	if err != nil {
		return nil, err
	}
	config.Infrastructure.Version = version

	// Backend Configuration
	utils.PrintInfo("\n🗄️  Backend Configuration")
	utils.PrintText("Configure Terraform backend settings.\n")

	useBackend, err := utils.PromptYesNo("Configure custom backend settings? (y/N)", false)
	if err != nil {
		return nil, err
	}

	if useBackend {
		config.Infrastructure.Backend = &models.BackendInfrastructureConfig{}

		// Backend Pattern
		pattern, err := utils.PromptWithDefault("Backend Pattern", "s3-backend")
		if err != nil {
			return nil, err
		}
		config.Infrastructure.Backend.Pattern = pattern

		// Backend Region
		backendRegion, err := utils.PromptWithDefault("Backend Region", region)
		if err != nil {
			return nil, err
		}
		config.Infrastructure.Backend.Region = backendRegion

		// Backend Account
		backendAccount, err := utils.PromptWithDefault("Backend Account (environment key)", "sha")
		if err != nil {
			return nil, err
		}
		config.Infrastructure.Backend.Account = backendAccount

		// Backend Encryption
		backendEncrypt, err := utils.PromptYesNo("Enable backend encryption? (Y/n)", true)
		if err != nil {
			return nil, err
		}
		config.Infrastructure.Backend.Encrypt = backendEncrypt
	}

	// AWS SSO Configuration (optional)
	if !configSkipAWSSSO {
		utils.PrintInfo("\n🔐 AWS SSO Configuration")
		utils.PrintText("Configure AWS SSO for your project (optional).\n")

		useSSO, err := utils.PromptYesNo("Configure AWS SSO? (y/N)", false)
		if err != nil {
			return nil, err
		}

		if useSSO {
			config.Infrastructure.AWSSSO = &models.SSOConfig{}

			// SSO Region
			ssoRegion, err := utils.PromptWithDefault("SSO Region", region)
			if err != nil {
				return nil, err
			}
			config.Infrastructure.AWSSSO.Region = ssoRegion

			// SSO Start URL
			startURL, err := utils.PromptString("SSO Start URL (e.g., https://client.awsapps.com/start#/)")
			if err != nil {
				return nil, err
			}
			config.Infrastructure.AWSSSO.StartURL = startURL

			// SSO Role Name
			roleName, err := utils.PromptWithDefault("SSO Role Name", "Admin")
			if err != nil {
				return nil, err
			}
			config.Infrastructure.AWSSSO.RoleName = roleName
		}
	}

	// Organization layer (optional)
	utils.PrintInfo("\n🏛️  Organization Layer")
	utils.PrintText("The organization layer is a global layer for org-level resources. It requires an AWS account for backend, secrets, and SSO.\n")

	useOrganization, err := utils.PromptYesNo("Enable organization layer? (y/N)", false)
	if err != nil {
		return nil, err
	}
	if useOrganization {
		if config.Infrastructure.Layers == nil {
			config.Infrastructure.Layers = &models.LayerConfig{}
		}
		orgTrue := true
		config.Infrastructure.Layers.Organization = &orgTrue
		config.Infrastructure.Organization = &models.OrganizationLayerConfig{}

		// AWS Account (required for organization)
		orgAccount, err := utils.PromptWithValidationRequired("Organization AWS Account ID (12 digits)", validation.ValidateAWSAccountID)
		if err != nil {
			return nil, err
		}
		config.Infrastructure.Organization.AWSAccount = orgAccount

		// Optional: secrets backend for organization (e.g. sops instead of global ssm)
		useOrgSecrets, err := utils.PromptYesNo("Use different secrets backend for organization? (e.g. SOPS) (y/N)", false)
		if err != nil {
			return nil, err
		}
		if useOrgSecrets {
			secretsType, err := utils.PromptWithDefault("Secrets type for organization (ssm/sops)", "sops")
			if err != nil {
				return nil, err
			}
			if secretsType != "" {
				config.Infrastructure.Organization.Secrets = &models.SecretsConfig{Type: secretsType}
			}
		}

		// Optional: custom backend for organization (bucket, key, etc.)
		useOrgBackend, err := utils.PromptYesNo("Use custom backend settings for organization? (y/N)", false)
		if err != nil {
			return nil, err
		}
		if useOrgBackend {
			config.Infrastructure.Organization.Backend = &models.BackendInfrastructureConfig{}
			backendPattern, err := utils.PromptWithDefault("Backend pattern", "s3-backend")
			if err != nil {
				return nil, err
			}
			config.Infrastructure.Organization.Backend.Pattern = backendPattern
			backendRegion, err := utils.PromptWithDefault("Backend region", region)
			if err != nil {
				return nil, err
			}
			config.Infrastructure.Organization.Backend.Region = backendRegion
			config.Infrastructure.Organization.Backend.Encrypt = true
		}
	}

	// Environment Configuration
	if !configSkipEnvironments {
		utils.PrintInfo("\n🌍 Environment Configuration")
		utils.PrintText("Configure your environments. You can add as many as you need.\n")

		environments, err := createInteractiveEnvironments()
		if err != nil {
			return nil, err
		}
		config.Infrastructure.Environments = environments
	} else {
		// CLI-only mode: no environments required
		config.Infrastructure.Environments = map[string]models.Environment{}
	}

	return config, nil
}

// createInteractiveEnvironments creates environments interactively
func createInteractiveEnvironments() (map[string]models.Environment, error) {
	environments := make(map[string]models.Environment)

	utils.PrintInfo("\n🌍 Environment Configuration")
	utils.PrintText("Let's configure your environments. You can choose from predefined environments or create custom ones.\n")

	// Show available predefined environments
	utils.ShowPredefinedEnvironmentOptions()

	// Ask for all predefined environments at once
	selectedPredefined, err := utils.PromptForPredefinedEnvironments()
	if err != nil {
		return nil, err
	}

	// Configure selected predefined environments
	for _, predefinedEnv := range selectedPredefined {
		// Check if environment key already exists
		if _, exists := environments[predefinedEnv.Key]; exists {
			utils.PrintError("❌ Environment key '%s' already exists. Skipping.", predefinedEnv.Key)
			continue
		}

		environment, err := utils.ConfigurePredefinedEnvironment(predefinedEnv)
		if err != nil {
			return nil, err
		}
		environments[predefinedEnv.Key] = *environment
	}

	// Ask if user wants to add additional environments
	for {
		addCustom, err := utils.PromptYesNoRequired("Add additional environment? (y/n)")
		if err != nil {
			return nil, err
		}
		if !addCustom {
			break
		}

		envKey, environment, err := utils.ConfigureCustomEnvironment()
		if err != nil {
			return nil, err
		}

		// Check if environment key already exists
		if _, exists := environments[envKey]; exists {
			utils.PrintError("❌ Environment key '%s' already exists. Please choose a different key.", envKey)
			continue
		}

		environments[envKey] = *environment
	}

	return environments, nil
}

// Validation functions moved from cmd/validate.go

func displayValidationResults(result *models.ValidationResult, strict bool) {
	displayValidationResultsText(result, strict)
}

func displayValidationResultsText(result *models.ValidationResult, strict bool) {
	utils.PrintInfo("\n📋 Validation Results:")
	utils.PrintText("===================\n")

	if len(result.Errors) > 0 {
		utils.PrintError("\n❌ Errors (%d):", len(result.Errors))
		for i, err := range result.Errors {
			utils.PrintError("  %d. %s", i+1, err)
		}
	}

	if len(result.Warnings) > 0 {
		if strict {
			utils.PrintError("\n❌ Warnings (treated as errors) (%d):", len(result.Warnings))
		} else {
			utils.PrintWarning("\n⚠️  Warnings (%d):", len(result.Warnings))
		}
		for i, warning := range result.Warnings {
			if strict {
				utils.PrintError("  %d. %s", i+1, warning)
			} else {
				utils.PrintWarning("  %d. %s", i+1, warning)
			}
		}
	}

	if len(result.Errors) == 0 && (len(result.Warnings) == 0 || !strict) {
		utils.PrintSuccess("\n✅ No validation errors found")
		if len(result.Warnings) > 0 {
			utils.PrintWarning("   (with %d warnings)", len(result.Warnings))
		}
	}
}
