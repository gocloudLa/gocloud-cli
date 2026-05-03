package models

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"gocloud-cli/internal/validation"
)

// Config represents the complete GoCloud configuration file structure
type Config struct {
	CLI            *CLIConfig            `json:"cli" yaml:"cli"`
	Infrastructure *InfrastructureConfig `json:"infrastructure" yaml:"infrastructure"`
}

// CLIConfig represents the CLI configuration section
type CLIConfig struct {
	WorkingDir string `json:"working_dir" yaml:"working_dir"`
	AutoBackup bool   `json:"auto_backup" yaml:"auto_backup"`
	BackupDir  string `json:"backup_dir" yaml:"backup_dir"`
	Verbose    bool   `json:"verbose" yaml:"verbose"`
	Debug      bool   `json:"debug" yaml:"debug"`
}

// DefaultCLIConfig returns a default CLI configuration
func DefaultCLIConfig() *CLIConfig {
	return &CLIConfig{
		WorkingDir: ".",
		AutoBackup: true,
		BackupDir:  ".gocloud-backups",
		Verbose:    false,
		Debug:      false,
	}
}

// DefaultConfig returns a default configuration with CLI and empty infrastructure
func DefaultConfig() *Config {
	return &Config{
		CLI:            DefaultCLIConfig(),
		Infrastructure: &InfrastructureConfig{},
	}
}

// ValidateConfig validates the complete configuration
func ValidateConfig(config *Config) error {
	var validationErrors []string

	// Validate CLI section
	if config.CLI == nil {
		config.CLI = DefaultCLIConfig()
	}

	// Validate Infrastructure section
	if config.Infrastructure == nil {
		validationErrors = append(validationErrors, "infrastructure section is required")
	} else {
		if config.Infrastructure.Client == "" {
			validationErrors = append(validationErrors, "infrastructure.client is required")
		}

		if config.Infrastructure.Company == "" {
			validationErrors = append(validationErrors, "infrastructure.company is required")
		} else {
			if err := validation.ValidateCompanyPrefix(config.Infrastructure.Company); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("infrastructure.company: %v", err))
			}
		}

		if config.Infrastructure.Region == "" {
			validationErrors = append(validationErrors, "infrastructure.region is required")
		}

		// Validate secrets configuration if present
		if config.Infrastructure.Secrets != nil {
			if err := ValidateSecretsConfig(config.Infrastructure.Secrets); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("infrastructure.secrets: %v", err))
			}
		}
		// Validate organization layer secrets if present
		if config.Infrastructure.Organization != nil && config.Infrastructure.Organization.Secrets != nil {
			if err := ValidateSecretsConfig(config.Infrastructure.Organization.Secrets); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("infrastructure.organization.secrets: %v", err))
			}
		}
		if config.Infrastructure.Security != nil && config.Infrastructure.Security.Secrets != nil {
			if err := ValidateSecretsConfig(config.Infrastructure.Security.Secrets); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("infrastructure.security.secrets: %v", err))
			}
		}
		// Validate organization: when layers.organization is true, infrastructure.organization with aws_account is required
		if config.Infrastructure.Layers != nil && config.Infrastructure.Layers.Organization != nil && *config.Infrastructure.Layers.Organization {
			if config.Infrastructure.Organization == nil || config.Infrastructure.Organization.AWSAccount == "" {
				validationErrors = append(validationErrors, "organization layer is enabled (layers.organization: true) but infrastructure.organization.aws_account is required for backend, secrets, and SSO")
			}
		}
		// Validate security: when layers.security is true, infrastructure.security with aws_account is required
		if config.Infrastructure.Layers != nil && config.Infrastructure.Layers.Security != nil && *config.Infrastructure.Layers.Security {
			if config.Infrastructure.Security == nil || config.Infrastructure.Security.AWSAccount == "" {
				validationErrors = append(validationErrors, "security layer is enabled (layers.security: true) but infrastructure.security.aws_account is required for backend, secrets, and SSO")
			}
		}
		// Validate organization aws_account if present (used for SSO profile client-org)
		if config.Infrastructure.Organization != nil && config.Infrastructure.Organization.AWSAccount != "" {
			if err := validation.ValidateAWSAccountID(config.Infrastructure.Organization.AWSAccount); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("infrastructure.organization.aws_account: %v", err))
			}
		}
		if config.Infrastructure.Security != nil && config.Infrastructure.Security.AWSAccount != "" {
			if err := validation.ValidateAWSAccountID(config.Infrastructure.Security.AWSAccount); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("infrastructure.security.aws_account: %v", err))
			}
		}

		// Validate environments (optional - can be empty for CLI-only configs)
		for envKey, env := range config.Infrastructure.Environments {
			if err := validation.ValidateEnvironmentKey(envKey); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("environment key '%s': %v", envKey, err))
			}

			if err := ValidateEnvironment(env); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("environment '%s': %v", envKey, err))
			}

			// Validate environment secrets configuration if present
			if env.Secrets != nil {
				if err := ValidateSecretsConfig(env.Secrets); err != nil {
					validationErrors = append(validationErrors, fmt.Sprintf("environment '%s'.secrets: %v", envKey, err))
				}
			}

			// Validate project-level secrets (supports both ProjectItem and map format from YAML)
			for _, project := range env.Projects {
				if sec := getSecretsConfigFromItem(project); sec != nil {
					if err := ValidateSecretsConfig(sec); err != nil {
						key := GetProjectKey(project)
						validationErrors = append(validationErrors, fmt.Sprintf("environment '%s' project '%s'.secrets: %v", envKey, key, err))
					}
				}
			}
			// Validate workload-level secrets (supports both WorkloadItem and map format from YAML)
			for _, workload := range env.Workloads {
				if sec := getSecretsConfigFromItem(workload); sec != nil {
					if err := ValidateSecretsConfig(sec); err != nil {
						key := GetWorkloadKey(workload)
						validationErrors = append(validationErrors, fmt.Sprintf("environment '%s' workload '%s'.secrets: %v", envKey, key, err))
					}
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "; "))
	}

	return nil
}

// ValidateConfigWithUnknownFields validates configuration and detects unknown fields
func ValidateConfigWithUnknownFields(yamlData []byte) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// First, parse as generic map to detect unknown fields
	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &rawConfig); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse YAML: %v", err))
		result.Valid = false
		return result, nil
	}

	// Check for unknown top-level fields
	knownTopLevel := map[string]bool{
		"cli":            true,
		"infrastructure": true,
	}

	for field := range rawConfig {
		if !knownTopLevel[field] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown top-level field: '%s'", field))
		}
	}

	// Check CLI section for unknown fields
	if cliSection, exists := rawConfig["cli"]; exists {
		if cliMap, ok := cliSection.(map[string]interface{}); ok {
			knownCLIFields := map[string]bool{
				"working_dir": true,
				"auto_backup": true,
				"backup_dir":  true,
				"verbose":     true,
				"debug":       true,
			}

			for field := range cliMap {
				if !knownCLIFields[field] {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown CLI field: 'cli.%s'", field))
				}
			}
		}
	}

	// Check Infrastructure section for unknown fields
	if infraSection, exists := rawConfig["infrastructure"]; exists {
		if infraMap, ok := infraSection.(map[string]interface{}); ok {
			knownInfraFields := map[string]bool{
				"client":            true,
				"company":           true,
				"region":            true,
				"version":           true,
				"source":            true,
				"source_ref":        true,
				"enable_secrets":    true,
				"enable_terragrunt": true,
				"enable_gitignore":  true,
				"enable_airules":    true,
				"layers":            true,
				"backend":           true,
				"providers":         true,
				"aws_sso":           true,
				"metadata":          true,
				"environments":      true,
				"environment_order": true,
				"secrets":           true,
				"organization":      true,
				"security":          true,
			}

			for field := range infraMap {
				if !knownInfraFields[field] {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown infrastructure field: 'infrastructure.%s'", field))
				}
			}

			// Check environments for unknown fields
			if envsSection, exists := infraMap["environments"]; exists {
				if envsMap, ok := envsSection.(map[string]interface{}); ok {
					knownEnvFields := map[string]bool{
						"name":              true,
						"dir_name":          true,
						"aws_account":       true,
						"region":            true,
						"version":           true,
						"source":            true,
						"source_ref":        true,
						"enable_secrets":    true,
						"enable_terragrunt": true,
						"enable_sso":        true,
						"layers":            true,
						"backend":           true,
						"providers":         true,
						"aws_sso":           true,
						"projects":          true,
						"workloads":         true,
						"secrets":           true,
					}

					for envKey, envData := range envsMap {
						if envMap, ok := envData.(map[string]interface{}); ok {
							for field := range envMap {
								if !knownEnvFields[field] {
									result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown environment field: 'infrastructure.environments.%s.%s'", envKey, field))
								}
							}
						}
					}
				}
			}
		}
	}

	// Now parse as proper Config struct for standard validation
	var config Config
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse configuration: %v", err))
		result.Valid = false
		return result, nil
	}

	// Perform standard validation
	if err := ValidateConfig(&config); err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.Valid = false
	}

	return result, nil
}

// ValidateEnvironment validates a single environment
func ValidateEnvironment(env Environment) error {
	if env.AWSAccount == "" {
		return fmt.Errorf("aws_account is required")
	}

	if err := validation.ValidateAWSAccountID(env.AWSAccount); err != nil {
		return fmt.Errorf("aws_account: %w", err)
	}

	return nil
}

// ValidateSecretsConfig validates a secrets configuration
func ValidateSecretsConfig(secrets *SecretsConfig) error {
	if secrets == nil {
		return nil
	}

	if secrets.Type != "" && secrets.Type != "ssm" && secrets.Type != "sops" {
		return fmt.Errorf("type must be 'ssm' or 'sops', got '%s'", secrets.Type)
	}

	return nil
}
