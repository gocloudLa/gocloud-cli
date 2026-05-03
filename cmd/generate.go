package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gocloud-cli/internal/config"
	"gocloud-cli/internal/generator"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/utils"

	"github.com/spf13/cobra"
)

// ErrGenerationCancelled is returned when user cancels the generation
var ErrGenerationCancelled = errors.New("generation cancelled by user")

var (
	generateConfigFile string
	generateWorkingDir string
	generateDryRun     bool
	generateForce      bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate infrastructure from validated YAML configuration",
	Long:  `Generate infrastructure project from a validated YAML configuration file.`,
	RunE:  runGenerate,
}

func init() {
	// Generate command flags
	generateCmd.Flags().StringVarP(&generateConfigFile, "config", "c", "gocloud.yaml", "Configuration file to use (default: gocloud.yaml)")
	generateCmd.Flags().StringVar(&generateWorkingDir, "working-dir", ".", "Working directory for infrastructure generation")
	generateCmd.Flags().BoolVar(&generateDryRun, "dry-run", false, "Show what would be generated without creating files")
	generateCmd.Flags().BoolVar(&generateForce, "force", false, "Overwrite existing files without confirmation")

	// Command is registered in cmd/root.go
}

func runGenerate(cmd *cobra.Command, args []string) error {
	utils.PrintInfo("\n🚀 GoCloud Infrastructure Generation")
	utils.PrintInfo("==================================\n")

	// Use default config file if not specified
	if generateConfigFile == "" {
		generateConfigFile = "gocloud.yaml"
	}

	// Load and validate configuration
	utils.PrintWarning("📋 Loading and validating configuration...")
	configManager := config.NewManager()
	config, validationResult, err := configManager.LoadConfigWithValidation(generateConfigFile)
	if err != nil {
		// Check if it's a file not found error
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found: %s", generateConfigFile)
		}
		// Check if it's a YAML syntax error
		if strings.Contains(err.Error(), "yaml") {
			return fmt.Errorf("invalid yaml syntax: %w", err)
		}
		return fmt.Errorf("failed to load configuration file '%s': %w", generateConfigFile, err)
	}

	// Check for validation errors (warnings are OK)
	if len(validationResult.Errors) > 0 {
		utils.PrintError("❌ Configuration validation failed:")
		for _, err := range validationResult.Errors {
			utils.PrintError("   - %s", err)
		}
		// This is a real error that needs to be fixed, so we keep the error behavior
		return fmt.Errorf("configuration validation failed with %d errors", len(validationResult.Errors))
	}

	// Show warnings if any
	if len(validationResult.Warnings) > 0 {
		utils.PrintWarning("⚠️  Configuration warnings (continuing anyway):")
		for _, warning := range validationResult.Warnings {
			utils.PrintWarning("   - %s", warning)
		}
	}

	utils.PrintSuccess("✅ Configuration validated successfully")

	// Determine working directory
	workingDir := generateWorkingDir
	if workingDir == "." {
		workingDir = "."
	}

	// Initialize project generator
	gen := generator.NewProjectGenerator(config.Infrastructure, workingDir, generateForce)

	if generateDryRun {
		return runDryRun(gen, config)
	}

	// Generate project structure
	utils.PrintWarning("📁 Creating project structure...")
	if err := gen.CreateProjectStructure(); err != nil {
		return fmt.Errorf("failed to create project structure: %w", err)
	}

	// Generate configuration files (layers, org, security, root.hcl)
	utils.PrintWarning("📝 Generating configuration files...")
	if err := gen.GenerateConfigFiles(); err != nil {
		return fmt.Errorf("failed to generate config files: %w", err)
	}

	utils.PrintWarning("📄 Generating root .gitignore...")
	if _, err := gen.GenerateGitignore(); err != nil {
		return fmt.Errorf("failed to write root .gitignore: %w", err)
	}

	utils.PrintWarning("🧩 Generating airules bundle (.cursor / .kiro)...")
	if _, err := gen.GenerateAirules(); err != nil {
		return fmt.Errorf("failed to write airules bundle: %w", err)
	}

	// Setup AWS SSO
	utils.PrintWarning("🔐 Setting up AWS SSO configuration...")
	if err := gen.SetupAWSSSO(); err != nil {
		return fmt.Errorf("failed to setup AWS SSO: %w", err)
	}

	// Create secrets structure
	utils.PrintWarning("🔑 Creating secrets structure...")
	if err := gen.CreateSecretsStructure(); err != nil {
		return fmt.Errorf("failed to create secrets structure: %w", err)
	}

	// Generate README.md
	utils.PrintWarning("📚 Generating README.md...")
	if err := gen.GenerateDocumentation(); err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}

	// Show summary
	showGenerationSummary(config, workingDir)

	return nil
}

func runDryRun(_ *generator.ProjectGenerator, config *models.Config) error {
	utils.PrintInfo("\n🔍 DRY RUN - Preview of what would be generated")
	utils.PrintInfo("==============================================\n")

	// Show configuration summary
	utils.PrintText("📋 Configuration Summary:\n")
	utils.PrintText("   Client: %s\n", config.Infrastructure.Client)
	utils.PrintText("   Company: %s\n", config.Infrastructure.Company)
	utils.PrintText("   Region: %s\n", config.Infrastructure.Region)
	utils.PrintText("   Working Directory: %s\n", generateWorkingDir)

	// Show environments
	utils.PrintText("\n🌍 Environments:\n")
	for envKey, env := range config.Infrastructure.Environments {
		utils.PrintText("   %s (%s):\n", envKey, env.Name)
		utils.PrintText("     AWS Account: %s\n", env.AWSAccount)
		if len(env.Projects) > 0 {
			utils.PrintText("     Projects: %v\n", env.Projects)
		}
		if len(env.Workloads) > 0 {
			utils.PrintText("     Workloads: %v\n", env.Workloads)
		}
	}

	// Show directory structure that would be created
	utils.PrintText("\n📁 Directory Structure to be created:\n")
	showDirectoryStructure(config.Infrastructure)

	// Show files that would be generated
	utils.PrintText("\n📝 Files to be generated:\n")
	showFilesToGenerate(config.Infrastructure)

	utils.PrintSuccess("\n✅ Dry run completed - no files were created")
	utils.PrintWarning("💡 Use 'gocloud generate --config %s' to actually generate the infrastructure", generateConfigFile)

	return nil
}

func showDirectoryStructure(config *models.InfrastructureConfig) {
	baseDir := generateWorkingDir
	if baseDir == "." {
		baseDir = "."
	}

	// Root level directories
	utils.PrintText("   %s/\n", baseDir)
	utils.PrintText("   ├── base/\n")
	utils.PrintText("   ├── foundation/\n")
	utils.PrintText("   ├── project/\n")
	utils.PrintText("   ├── workload/\n")
	utils.PrintText("   ├── organization/\n")
	utils.PrintText("   ├── security/\n")
	if generator.IsGitignoreGenerationEnabledForConfig(config) {
		utils.PrintText("   ├── .gitignore\n")
	}
	utils.PrintText("   ├── root.hcl\n")
	utils.PrintText("   └── README.md\n")

	// Environment-specific directories
	for envKey, env := range config.Environments {
		dirName := getDirectoryName(envKey, env)
		utils.PrintText("   ├── base/%s/\n", dirName)
		utils.PrintText("   ├── foundation/%s/\n", dirName)

		// Project directories
		for _, project := range env.Projects {
			utils.PrintText("   ├── project/%s/%s/\n", project, dirName)
		}

		// Workload directories
		for _, workload := range env.Workloads {
			utils.PrintText("   ├── workload/%s/%s/\n", workload, dirName)
		}
	}
}

// shouldGenerateTerragruntForPreview determines if terragrunt.hcl should be generated for preview
// This is a simplified version of the logic in ProjectGenerator for dry-run purposes
func shouldGenerateTerragruntForPreview(config *models.InfrastructureConfig, layerType, project, env string) bool {
	// Get environment configuration
	envConfig, exists := config.Environments[env]
	if !exists {
		// If environment doesn't exist, use infrastructure default
		if config.EnableTerragrunt != nil {
			return *config.EnableTerragrunt
		}
		// Default to true if not specified
		return true
	}

	// Check if it's a project or workload (has project parameter)
	if project != "" {
		// Check workloads first
		if layerType == "workload" {
			for _, workloadInterface := range envConfig.Workloads {
				var workloadName string
				var enableTerragrunt *bool

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
					if len(w) == 1 {
						for key, value := range w {
							workloadName = key
							if valueMap, ok := value.(map[string]interface{}); ok {
								if et, ok := valueMap["enable_terragrunt"].(bool); ok {
									enableTerragrunt = &et
								}
							}
						}
					} else {
						if name, ok := w["name"].(string); ok {
							workloadName = name
							if et, ok := w["enable_terragrunt"].(bool); ok {
								enableTerragrunt = &et
							}
						}
					}
				}

				if workloadName == project {
					// If workload has explicit enable_terragrunt setting, use it
					if enableTerragrunt != nil {
						return *enableTerragrunt
					}
					// Otherwise, fall through to environment level
					break
				}
			}
		}

		// Check projects
		if layerType == "project" {
			for _, projectInterface := range envConfig.Projects {
				var projectName string
				var enableTerragrunt *bool

				// Normalize map[interface{}]interface{} (YAML unmarshal) to map[string]interface{}
				var p map[string]interface{}
				switch item := projectInterface.(type) {
				case string:
					projectName = item
				case map[interface{}]interface{}:
					p = models.ToMapStringInterface(item)
				case map[string]interface{}:
					p = item
				}
				if p != nil {
					if len(p) == 1 {
						for key, value := range p {
							projectName = key
							if valueMap, ok := value.(map[string]interface{}); ok {
								if et, ok := valueMap["enable_terragrunt"].(bool); ok {
									enableTerragrunt = &et
								}
							}
						}
					} else {
						if name, ok := p["name"].(string); ok {
							projectName = name
							if et, ok := p["enable_terragrunt"].(bool); ok {
								enableTerragrunt = &et
							}
						}
					}
				}

				if projectName == project {
					// If project has explicit enable_terragrunt setting, use it
					if enableTerragrunt != nil {
						return *enableTerragrunt
					}
					// Otherwise, fall through to environment level
					break
				}
			}
		}
	}

	// Check environment level
	if envConfig.EnableTerragrunt != nil {
		return *envConfig.EnableTerragrunt
	}

	// Fall back to infrastructure level
	if config.EnableTerragrunt != nil {
		return *config.EnableTerragrunt
	}
	// Default to true if not specified
	return true
}

// shouldGenerateOrganizationSecretsForPreview mirrors generator: organization has no env in config, use infrastructure default
func shouldGenerateOrganizationSecretsForPreview(config *models.InfrastructureConfig) bool {
	if config.EnableSecrets != nil {
		return *config.EnableSecrets
	}
	return true
}

// shouldGenerateOrganizationTerragruntForPreview mirrors generator: organization has no env in config, use infrastructure default
func shouldGenerateOrganizationTerragruntForPreview(config *models.InfrastructureConfig) bool {
	if config.EnableTerragrunt != nil {
		return *config.EnableTerragrunt
	}
	return true
}

func shouldGenerateSecuritySecretsForPreview(config *models.InfrastructureConfig) bool {
	if config.Security != nil && config.Security.EnableSecrets != nil {
		return *config.Security.EnableSecrets
	}
	if config.EnableSecrets != nil {
		return *config.EnableSecrets
	}
	return true
}

func shouldGenerateSecurityTerragruntForPreview(config *models.InfrastructureConfig) bool {
	if config.EnableTerragrunt != nil {
		return *config.EnableTerragrunt
	}
	return true
}

func showFilesToGenerate(config *models.InfrastructureConfig) {
	baseDir := generateWorkingDir
	if baseDir == "." {
		baseDir = "."
	}

	// Root level files
	if generator.IsGitignoreGenerationEnabledForConfig(config) {
		utils.PrintText("   %s/.gitignore\n", baseDir)
	}
	if generator.IsAirulesGenerationEnabledForConfig(config) {
		utils.PrintText("   %s/.cursor/**  (airules bundle maintained by GoCloud CLI)\n", baseDir)
		utils.PrintText("   %s/.kiro/**   (airules bundle maintained by GoCloud CLI)\n", baseDir)
	}
	utils.PrintText("   %s/root.hcl\n", baseDir)
	utils.PrintText("   %s/README.md\n", baseDir)

	// Layer-specific files for each environment
	for envKey, env := range config.Environments {
		dirName := getDirectoryName(envKey, env)

		// Base layer files
		utils.PrintText("   %s/base/%s/_secrets.tf\n", baseDir, dirName)
		utils.PrintText("   %s/base/%s/main.tf\n", baseDir, dirName)
		utils.PrintText("   %s/base/%s/metadata.tf\n", baseDir, dirName)
		utils.PrintText("   %s/base/%s/providers.tf\n", baseDir, dirName)
		utils.PrintText("   %s/base/%s/backend.tf\n", baseDir, dirName)
		if shouldGenerateTerragruntForPreview(config, "base", "", envKey) {
			utils.PrintText("   %s/base/%s/terragrunt.hcl\n", baseDir, dirName)
		}

		// Foundation layer files
		utils.PrintText("   %s/foundation/%s/_secrets.tf\n", baseDir, dirName)
		utils.PrintText("   %s/foundation/%s/main.tf\n", baseDir, dirName)
		utils.PrintText("   %s/foundation/%s/metadata.tf\n", baseDir, dirName)
		utils.PrintText("   %s/foundation/%s/providers.tf\n", baseDir, dirName)
		utils.PrintText("   %s/foundation/%s/backend.tf\n", baseDir, dirName)
		if shouldGenerateTerragruntForPreview(config, "foundation", "", envKey) {
			utils.PrintText("   %s/foundation/%s/terragrunt.hcl\n", baseDir, dirName)
		}

		// Project layer files
		for _, project := range env.Projects {
			projectName := models.GetProjectDirectoryName(project)
			utils.PrintText("   %s/project/%s/%s/_secrets.tf\n", baseDir, projectName, dirName)
			utils.PrintText("   %s/project/%s/%s/main.tf\n", baseDir, projectName, dirName)
			utils.PrintText("   %s/project/%s/%s/metadata.tf\n", baseDir, projectName, dirName)
			utils.PrintText("   %s/project/%s/%s/providers.tf\n", baseDir, projectName, dirName)
			utils.PrintText("   %s/project/%s/%s/backend.tf\n", baseDir, projectName, dirName)
			if shouldGenerateTerragruntForPreview(config, "project", models.GetProjectKey(project), envKey) {
				utils.PrintText("   %s/project/%s/%s/terragrunt.hcl\n", baseDir, projectName, dirName)
			}
		}

		// Workload layer files
		for _, workload := range env.Workloads {
			workloadName := models.GetWorkloadDirectoryName(workload)
			utils.PrintText("   %s/workload/%s/%s/_secrets.tf\n", baseDir, workloadName, dirName)
			utils.PrintText("   %s/workload/%s/%s/main.tf\n", baseDir, workloadName, dirName)
			utils.PrintText("   %s/workload/%s/%s/metadata.tf\n", baseDir, workloadName, dirName)
			utils.PrintText("   %s/workload/%s/%s/providers.tf\n", baseDir, workloadName, dirName)
			utils.PrintText("   %s/workload/%s/%s/backend.tf\n", baseDir, workloadName, dirName)
			if shouldGenerateTerragruntForPreview(config, "workload", models.GetWorkloadKey(workload), envKey) {
				utils.PrintText("   %s/workload/%s/%s/terragrunt.hcl\n", baseDir, workloadName, dirName)
			}
		}
	}

	// Organization layer files (only when infrastructure.organization.aws_account is set; match generator logic)
	if generator.IsOrganizationLayerEnabledForConfig(config) {
		utils.PrintText("   %s/organization/main.tf\n", baseDir)
		utils.PrintText("   %s/organization/metadata.tf\n", baseDir)
		if shouldGenerateOrganizationSecretsForPreview(config) {
			utils.PrintText("   %s/organization/_secrets.tf\n", baseDir)
		}
		if shouldGenerateOrganizationTerragruntForPreview(config) {
			utils.PrintText("   %s/organization/terragrunt.hcl\n", baseDir)
		}
		utils.PrintText("   %s/organization/providers.tf\n", baseDir)
		utils.PrintText("   %s/organization/backend.tf\n", baseDir)
	}

	if generator.IsSecurityLayerEnabledForConfig(config) {
		utils.PrintText("   %s/security/main.tf\n", baseDir)
		utils.PrintText("   %s/security/metadata.tf\n", baseDir)
		if shouldGenerateSecuritySecretsForPreview(config) {
			utils.PrintText("   %s/security/_secrets.tf\n", baseDir)
		}
		if shouldGenerateSecurityTerragruntForPreview(config) {
			utils.PrintText("   %s/security/terragrunt.hcl\n", baseDir)
		}
		utils.PrintText("   %s/security/providers.tf\n", baseDir)
		utils.PrintText("   %s/security/backend.tf\n", baseDir)
	}
}

func showGenerationSummary(config *models.Config, workingDir string) {
	utils.PrintSuccess("\n✅ Infrastructure generation completed successfully!")

	utils.PrintInfo("\n📋 Generated Summary:")
	utils.PrintText("   Client: %s\n", config.Infrastructure.Client)
	utils.PrintText("   Company: %s\n", config.Infrastructure.Company)
	utils.PrintText("   Region: %s\n", config.Infrastructure.Region)
	utils.PrintText("   Working Directory: %s\n", workingDir)
	utils.PrintText("   Environments: %d\n", len(config.Infrastructure.Environments))

	utils.PrintInfo("\n📁 Generated Structure:")
	for envKey, env := range config.Infrastructure.Environments {
		dirName := getDirectoryName(envKey, env)
		utils.PrintText("   %s (%s):\n", envKey, env.Name)

		// Only show layers that are actually generated
		if shouldGenerateLayerForSummary(config, "base", envKey) {
			utils.PrintText("     - base/%s/\n", dirName)
		}
		if shouldGenerateLayerForSummary(config, "foundation", envKey) {
			utils.PrintText("     - foundation/%s/\n", dirName)
		}

		if len(env.Projects) > 0 {
			for _, project := range env.Projects {
				projectDirName := models.GetProjectDirectoryName(project)
				utils.PrintText("     - project/%s/%s/\n", projectDirName, dirName)
			}
		}

		if len(env.Workloads) > 0 {
			for _, workload := range env.Workloads {
				workloadDirName := models.GetWorkloadDirectoryName(workload)
				utils.PrintText("     - workload/%s/%s/\n", workloadDirName, dirName)
			}
		}
	}

	// Show organization layer if enabled
	if shouldGenerateLayerForSummary(config, "organization", "") {
		utils.PrintText("   organization/\n")
	}
	if shouldGenerateLayerForSummary(config, "security", "") {
		utils.PrintText("   security/\n")
	}

	utils.PrintInfo("\n📋 Next Steps:")
	utils.PrintText("  1. Review the generated configuration files\n")
	utils.PrintText("  2. Configure your AWS SSO profiles\n")
	utils.PrintText("  3. Set up your secrets in AWS SSM\n")
	utils.PrintText("  4. Use Terragrunt to deploy your infrastructure\n")
	utils.PrintText("  5. Run 'terragrunt run-all plan' to review changes\n")
}

func getDirectoryName(envKey string, env models.Environment) string {
	if env.DirName != "" {
		return env.DirName
	}
	if env.Name != "" {
		return strings.ToLower(env.Name)
	}
	return envKey
}

// shouldGenerateLayerForSummary determines if a specific layer should be shown in summary
// following the same hierarchy as the generator: environment -> infrastructure -> default (true)
func shouldGenerateLayerForSummary(config *models.Config, layerType, envKey string) bool {
	// Organization requires infrastructure.organization.aws_account (same as generator)
	if layerType == "organization" {
		return config.Infrastructure != nil && generator.IsOrganizationLayerEnabledForConfig(config.Infrastructure)
	}
	if layerType == "security" {
		return config.Infrastructure != nil && generator.IsSecurityLayerEnabledForConfig(config.Infrastructure)
	}

	// Get environment configuration
	env, exists := config.Infrastructure.Environments[envKey]
	if !exists {
		// If environment doesn't exist, use infrastructure default
		return getLayerDefaultForSummary(config, layerType)
	}

	// Check environment level
	if env.Layers != nil {
		switch layerType {
		case "base":
			if env.Layers.Base != nil {
				return *env.Layers.Base
			}
		case "foundation":
			if env.Layers.Foundation != nil {
				return *env.Layers.Foundation
			}
		}
	}

	// Fall back to infrastructure level
	return getLayerDefaultForSummary(config, layerType)
}

// getLayerDefaultForSummary returns the default value for a layer from infrastructure config or true
func getLayerDefaultForSummary(config *models.Config, layerType string) bool {
	if config.Infrastructure.Layers != nil {
		switch layerType {
		case "base":
			if config.Infrastructure.Layers.Base != nil {
				return *config.Infrastructure.Layers.Base
			}
		case "foundation":
			if config.Infrastructure.Layers.Foundation != nil {
				return *config.Infrastructure.Layers.Foundation
			}
		case "organization":
			if config.Infrastructure.Layers.Organization != nil {
				return *config.Infrastructure.Layers.Organization
			}
		case "security":
			if config.Infrastructure.Layers.Security != nil {
				return *config.Infrastructure.Layers.Security
			}
		}
	}
	// Default to true if not specified
	return true
}
