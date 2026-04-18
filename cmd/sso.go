package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	cfg "gocloud-cli/internal/config"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/utils"
)

var (
	ssoAllProfiles      bool
	ssoSpecificProfiles string
)

var ssoCmd = &cobra.Command{
	Use:   "sso",
	Short: "Manage AWS SSO configuration and authentication",
	Long:  `Manage AWS SSO configuration and authentication.`,
}

var ssoSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup AWS SSO configuration",
	Long:  `Setup AWS SSO configuration by generating .aws/config file from project configuration.`,
	RunE:  runSSOSetup,
}

var ssoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available AWS profiles",
	Long:  `List all available AWS profiles from the generated .aws/config file.`,
	RunE:  runSSOList,
}

var ssoLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to AWS profiles",
	Long:  `Login to AWS profiles interactively or via parameters.`,
	RunE:  runSSOLogin,
}

var ssoVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify AWS SSO profile status",
	Long:  `Verify the status of AWS SSO profiles by checking credentials and account information.`,
	RunE:  runSSOVerify,
}

func init() {
	// SSO command flags
	ssoLoginCmd.Flags().BoolVar(&ssoAllProfiles, "all", false, "Login to all available profiles")
	ssoLoginCmd.Flags().StringVar(&ssoSpecificProfiles, "profiles", "", "Comma-separated list of profile names to login")

	// Add subcommands
	ssoCmd.AddCommand(ssoSetupCmd)
	ssoCmd.AddCommand(ssoListCmd)
	ssoCmd.AddCommand(ssoLoginCmd)
	ssoCmd.AddCommand(ssoVerifyCmd)

	// Command is registered in root.go
}

// checkAWSCLI checks if AWS CLI is installed
func checkAWSCLI() error {
	cmd := exec.Command("aws", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("AWS CLI is not installed or not in PATH. Please install AWS CLI first: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html")
	}
	return nil
}

func runSSOSetup(cmd *cobra.Command, args []string) error {
	utils.PrintInfo("\n🚀 Setting up AWS SSO Configuration")
	utils.PrintInfo("==================================\n")

	// Load configuration
	utils.PrintWarning("📋 Loading project configuration...")

	// Get config file from root command flags
	configFile, _ := cmd.Flags().GetString("config")

	var config *models.Config
	var err error

	if configFile != "" {
		// Use specified config file
		cfgManager := cfg.NewManager()
		configData, err := cfgManager.LoadConfig(configFile)
		if err != nil {
			// Check if it's a file not found error
			if os.IsNotExist(err) {
				return fmt.Errorf("config file not found: %s", configFile)
			}
			// Check if it's a YAML syntax error
			if strings.Contains(err.Error(), "yaml") {
				return fmt.Errorf("invalid yaml syntax: %w", err)
			}
			return fmt.Errorf("failed to load configuration from %s: %w", configFile, err)
		}
		config = configData
	} else {
		// Try to load from current directory
		config, err = loadConfiguration()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	// Create .aws directory
	utils.PrintWarning("📝 Creating .aws directory...")
	if err := os.MkdirAll(".aws", 0755); err != nil {
		return fmt.Errorf("failed to create .aws directory: %w", err)
	}

	// Create .aws/.gitignore
	gitignoreContent := "*\n"
	if err := os.WriteFile(".aws/.gitignore", []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .aws/.gitignore: %w", err)
	}

	// Generate AWS config file
	utils.PrintWarning("🔧 Generating .aws/config file...")
	awsConfigContent, err := generateAWSConfig(config)
	if err != nil {
		return fmt.Errorf("failed to generate AWS config: %w", err)
	}

	// Write AWS config file
	awsConfigPath := filepath.Join(".aws", "config")
	if err := os.WriteFile(awsConfigPath, []byte(awsConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write .aws/config: %w", err)
	}

	// Set environment variable
	awsConfigFile := filepath.Join(".", ".aws", "config")
	if err := os.Setenv("AWS_CONFIG_FILE", awsConfigFile); err != nil {
		return fmt.Errorf("failed to set AWS_CONFIG_FILE: %w", err)
	}

	utils.PrintSuccess("✅ AWS SSO configuration setup completed!")
	utils.PrintText("   AWS_CONFIG_FILE: %s\n", awsConfigFile)
	// Match generateAWSConfig: only environments with enable_sso (default true) get a profile.
	profileCount := 0
	for _, env := range config.Infrastructure.Environments {
		if models.ShouldEnableSSO(env) {
			profileCount++
		}
	}
	if organizationSSOEnabled(config.Infrastructure) {
		profileCount++
	}
	if securitySSOEnabled(config.Infrastructure) {
		profileCount++
	}
	utils.PrintText("   Generated profiles: %d\n", profileCount)

	return nil
}

func runSSOList(cmd *cobra.Command, args []string) error {
	utils.PrintInfo("\n📋 AWS Profiles List")
	utils.PrintInfo("==================\n")

	// Check if .aws/config exists
	if _, err := os.Stat(".aws/config"); os.IsNotExist(err) {
		return fmt.Errorf("❌ Error: .aws/config file not found. Run 'gocloud sso setup' first")
	}

	// Read and parse profiles
	profiles, err := getAWSProfiles()
	if err != nil {
		return fmt.Errorf("failed to read profiles: %w", err)
	}

	if len(profiles) == 0 {
		utils.PrintWarning("⚠️  No profiles found in .aws/config")
		return nil
	}

	utils.PrintText("Available profiles:\n")
	for _, profile := range profiles {
		utils.PrintText("  *) %s\n", profile)
	}

	return nil
}

func runSSOLogin(cmd *cobra.Command, args []string) error {
	utils.PrintInfo("\n🔐 AWS SSO Login")
	utils.PrintInfo("===============\n")

	// Check if AWS CLI is installed
	if err := checkAWSCLI(); err != nil {
		return err
	}

	// Check if .aws/config exists
	if _, err := os.Stat(".aws/config"); os.IsNotExist(err) {
		return fmt.Errorf("❌ Error: .aws/config file not found. Run 'gocloud sso setup' first")
	}

	// Get available profiles
	profiles, err := getAWSProfiles()
	if err != nil {
		return fmt.Errorf("failed to read profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles found in .aws/config")
		return nil // Exit silently with success code
	}

	// Determine which profiles to login
	var profilesToLogin []string

	if ssoAllProfiles {
		utils.PrintWarning("🎯 Mode: Login to all profiles")
		profilesToLogin = profiles
	} else if ssoSpecificProfiles != "" {
		utils.PrintWarning("🎯 Mode: Login to specific profiles")
		requestedProfiles := strings.Split(ssoSpecificProfiles, ",")

		for _, profile := range requestedProfiles {
			profile = strings.TrimSpace(profile)
			if isValidProfile(profile, profiles) {
				profilesToLogin = append(profilesToLogin, profile)
			} else {
				utils.PrintText("Profile '%s' not found\n", profile)
				return nil // Exit silently with success code
			}
		}
	} else {
		// Interactive mode
		utils.PrintWarning("🎯 Mode: Interactive")
		utils.PrintText("Available profiles:\n")
		for _, profile := range profiles {
			utils.PrintText("  *) %s\n", profile)
		}

		utils.PrintText("\nOptions:")
		utils.PrintText("  1) All profiles")
		utils.PrintText("  2) Specific profiles")
		utils.PrintText("  3) Cancel")

		var choice string
		fmt.Print("\nSelect an option (1-3): ")
		if _, err := fmt.Scanln(&choice); err != nil {
			return fmt.Errorf("failed to read user choice: %w", err)
		}

		switch choice {
		case "1":
			utils.PrintWarning("🎯 Login to all profiles")
			profilesToLogin = profiles
		case "2":
			utils.PrintText("\nEnter profile names or numbers separated by commas:")
			var input string
			if _, err := fmt.Scanln(&input); err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			inputs := strings.Split(input, ",")
			for _, profileInput := range inputs {
				profileInput = strings.TrimSpace(profileInput)

				// Check if it's a number
				if isNumber(profileInput) {
					index := parseInt(profileInput) - 1
					if index >= 0 && index < len(profiles) {
						profilesToLogin = append(profilesToLogin, profiles[index])
					} else {
						return fmt.Errorf("❌ Error: Invalid profile number '%s'", profileInput)
					}
				} else {
					if isValidProfile(profileInput, profiles) {
						profilesToLogin = append(profilesToLogin, profileInput)
					} else {
						utils.PrintText("Profile '%s' not found\n", profileInput)
						return nil // Exit silently with success code
					}
				}
			}
		case "3":
			utils.PrintSuccess("❌ Operation cancelled")
			return nil
		default:
			return fmt.Errorf("❌ Invalid option")
		}
	}

	if len(profilesToLogin) == 0 {
		fmt.Println("No profiles to login")
		return nil // Exit silently with success code
	}

	utils.PrintText("\n📋 Profiles to login:")
	for _, profile := range profilesToLogin {
		utils.PrintText("  - %s\n", profile)
	}

	// Login to profiles in parallel
	utils.PrintWarning("\n🔑 Logging in to selected profiles in parallel...")

	// Set AWS_CONFIG_FILE environment variable
	awsConfigFile := filepath.Join(".", ".aws", "config")
	if err := os.Setenv("AWS_CONFIG_FILE", awsConfigFile); err != nil {
		return fmt.Errorf("failed to set AWS_CONFIG_FILE: %w", err)
	}

	// Start all login processes in parallel
	var wg sync.WaitGroup
	results := make(chan string, len(profilesToLogin))

	// Start a goroutine to display results in real-time
	successCountChan := make(chan int, 1)
	go func() {
		successCount := 0
		for result := range results {
			if strings.HasPrefix(result, "✅") {
				successCount++
			}
			utils.PrintSuccess("%s", result)
		}
		successCountChan <- successCount
	}()

	for _, profile := range profilesToLogin {
		wg.Add(1)
		go func(profileName string) {
			defer wg.Done()

			utils.PrintText("🔐 Logging in to profile: %s\n", profileName)

			loginCmd := exec.Command("aws", "sso", "login", "--profile", profileName)
			// Capture output to suppress verbose messages
			var stdout, stderr bytes.Buffer
			loginCmd.Stdout = &stdout
			loginCmd.Stderr = &stderr

			if err := loginCmd.Run(); err != nil {
				results <- fmt.Sprintf("❌ %s", profileName)
			} else {
				// AWS SSO login succeeded, now validate credentials
				if err := verifyProfileSilent(profileName); err != nil {
					results <- fmt.Sprintf("❌ %s: Expired or invalid credentials", profileName)
				} else {
					results <- fmt.Sprintf("✅ %s", profileName)
				}
			}
		}(profile)
	}

	// Wait for all processes to complete
	utils.PrintWarning("⏳ Waiting for all logins to complete...")
	wg.Wait()
	close(results)

	// Wait a moment for the results goroutine to finish
	time.Sleep(100 * time.Millisecond)

	// Get the final success count
	successCount := <-successCountChan

	// successCount now reflects the actual validation results

	if successCount == len(profilesToLogin) {
		utils.PrintSuccess("\n✅ All profiles authenticated successfully!")
	} else {
		failedCount := len(profilesToLogin) - successCount
		utils.PrintError("\n❌ Authentication failed for %d/%d profiles", failedCount, len(profilesToLogin))
		utils.PrintWarning("💡 Run 'gocloud sso verify' for detailed status")

		// Exit with error code but don't return error to avoid showing usage
		os.Exit(1)
	}

	utils.PrintText("\n📋 Useful commands:")
	utils.PrintText("  terragrunt plan -concise --all")

	// Show export command for AWS_CONFIG_FILE
	utils.PrintInfo("\n🔧 Environment Setup")
	utils.PrintText("To use the newly configured profiles, export the AWS_CONFIG_FILE environment variable:")
	utils.PrintWarning("  export AWS_CONFIG_FILE=$(pwd)/.aws/config")

	return nil
}

// Helper functions

func getAWSProfiles() ([]string, error) {
	content, err := os.ReadFile(".aws/config")
	if err != nil {
		return nil, err
	}

	var profiles []string
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[profile ") {
			profile := strings.TrimPrefix(line, "[profile ")
			profile = strings.TrimSuffix(profile, "]")
			profiles = append(profiles, profile)
		}
	}

	return profiles, nil
}

func isValidProfile(profile string, availableProfiles []string) bool {
	for _, p := range availableProfiles {
		if p == profile {
			return true
		}
	}
	return false
}

func isNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func parseInt(s string) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return 0
	}
	return result
}

// loadConfiguration loads the project configuration
func loadConfiguration() (*models.Config, error) {
	return cfg.LoadConfigWithPath("gocloud.yaml")
}

// organizationSSOEnabled returns whether to add the organization SSO profile ({client}-org).
// We add it when organization.aws_account is set, unless layers.organization is explicitly false.
// So if you have organization.aws_account but no "layers" key, you still get the org profile.
func organizationSSOEnabled(infra *models.InfrastructureConfig) bool {
	return models.IsOrganizationEnabled(infra)
}

func securitySSOEnabled(infra *models.InfrastructureConfig) bool {
	return models.IsSecurityEnabled(infra)
}

// generateAWSConfig generates AWS config content from project configuration
func generateAWSConfig(config *models.Config) (string, error) {
	var content strings.Builder

	// Get global AWS SSO settings
	globalSSO := config.Infrastructure.AWSSSO
	client := config.Infrastructure.Client

	// Map to track which sessions we've already written
	writtenSessions := make(map[string]bool)

	// Generate profiles and their corresponding sessions (only for environments with SSO enabled)
	for envKey, env := range config.Infrastructure.Environments {
		// Skip environments that have SSO disabled
		if !models.ShouldEnableSSO(env) {
			continue
		}

		// Generate profile name: {client}-{environment}
		profileName := fmt.Sprintf("%s-%s", client, envKey)

		// Determine SSO settings for this environment
		ssoStartURL := globalSSO.StartURL
		ssoRoleName := globalSSO.RoleName

		// Use environment-specific SSO settings if available
		if env.AWSSSO != nil {
			if env.AWSSSO.StartURL != "" {
				ssoStartURL = env.AWSSSO.StartURL
			}
			if env.AWSSSO.RoleName != "" {
				ssoRoleName = env.AWSSSO.RoleName
			}
		}

		// Generate unique session name for this environment's SSO configuration
		sessionName := fmt.Sprintf("%s-%s", client, envKey)

		// Write SSO session configuration if we haven't written it yet
		if !writtenSessions[sessionName] {
			content.WriteString(fmt.Sprintf("[sso-session %s]\n", sessionName))
			content.WriteString(fmt.Sprintf("sso_start_url = %s\n", ssoStartURL))
			content.WriteString(fmt.Sprintf("sso_region = %s\n", globalSSO.Region))
			content.WriteString("sso_registration_scopes = sso:account:access\n")
			content.WriteString("\n")
			writtenSessions[sessionName] = true
		}

		// Write profile configuration
		content.WriteString(fmt.Sprintf("[profile %s]\n", profileName))
		content.WriteString(fmt.Sprintf("sso_session = %s\n", sessionName))
		content.WriteString(fmt.Sprintf("sso_account_id = %s\n", env.AWSAccount))
		content.WriteString(fmt.Sprintf("sso_role_name = %s\n", ssoRoleName))
		content.WriteString(fmt.Sprintf("region = %s\n", config.Infrastructure.Region))
		content.WriteString("output = json\n")
		content.WriteString("\n")
	}

	// Organization layer: add profile {client}-org when organization is enabled and aws_account is set
	if organizationSSOEnabled(config.Infrastructure) {
		org := config.Infrastructure.Organization
		profileName := fmt.Sprintf("%s-org", client)
		ssoStartURL := globalSSO.StartURL
		ssoRoleName := globalSSO.RoleName
		if org.AWSSSO != nil {
			if org.AWSSSO.StartURL != "" {
				ssoStartURL = org.AWSSSO.StartURL
			}
			if org.AWSSSO.RoleName != "" {
				ssoRoleName = org.AWSSSO.RoleName
			}
		}
		sessionName := fmt.Sprintf("%s-org", client)
		if !writtenSessions[sessionName] {
			content.WriteString(fmt.Sprintf("[sso-session %s]\n", sessionName))
			content.WriteString(fmt.Sprintf("sso_start_url = %s\n", ssoStartURL))
			content.WriteString(fmt.Sprintf("sso_region = %s\n", globalSSO.Region))
			content.WriteString("sso_registration_scopes = sso:account:access\n")
			content.WriteString("\n")
			writtenSessions[sessionName] = true
		}
		content.WriteString(fmt.Sprintf("[profile %s]\n", profileName))
		content.WriteString(fmt.Sprintf("sso_session = %s\n", sessionName))
		content.WriteString(fmt.Sprintf("sso_account_id = %s\n", org.AWSAccount))
		content.WriteString(fmt.Sprintf("sso_role_name = %s\n", ssoRoleName))
		content.WriteString(fmt.Sprintf("region = %s\n", config.Infrastructure.Region))
		content.WriteString("output = json\n")
		content.WriteString("\n")
	}

	// Security layer: profile {client}-sec when security.aws_account is set (same rules as organization)
	if securitySSOEnabled(config.Infrastructure) {
		sec := config.Infrastructure.Security
		profileName := fmt.Sprintf("%s-sec", client)
		ssoStartURL := globalSSO.StartURL
		ssoRoleName := globalSSO.RoleName
		if sec.AWSSSO != nil {
			if sec.AWSSSO.StartURL != "" {
				ssoStartURL = sec.AWSSSO.StartURL
			}
			if sec.AWSSSO.RoleName != "" {
				ssoRoleName = sec.AWSSSO.RoleName
			}
		}
		sessionName := fmt.Sprintf("%s-sec", client)
		if !writtenSessions[sessionName] {
			content.WriteString(fmt.Sprintf("[sso-session %s]\n", sessionName))
			content.WriteString(fmt.Sprintf("sso_start_url = %s\n", ssoStartURL))
			content.WriteString(fmt.Sprintf("sso_region = %s\n", globalSSO.Region))
			content.WriteString("sso_registration_scopes = sso:account:access\n")
			content.WriteString("\n")
			writtenSessions[sessionName] = true
		}
		content.WriteString(fmt.Sprintf("[profile %s]\n", profileName))
		content.WriteString(fmt.Sprintf("sso_session = %s\n", sessionName))
		content.WriteString(fmt.Sprintf("sso_account_id = %s\n", sec.AWSAccount))
		content.WriteString(fmt.Sprintf("sso_role_name = %s\n", ssoRoleName))
		content.WriteString(fmt.Sprintf("region = %s\n", config.Infrastructure.Region))
		content.WriteString("output = json\n")
		content.WriteString("\n")
	}

	return content.String(), nil
}

// runSSOVerify verifies the status of AWS SSO profiles
func runSSOVerify(cmd *cobra.Command, args []string) error {
	utils.PrintInfo("🔍 Verifying AWS SSO configuration...")

	// Check if .aws/config exists
	awsConfigPath := ".aws/config"
	if _, err := os.Stat(awsConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("❌ Error: .aws/config file not found\n💡 Run: gocloud sso setup")
	}

	utils.PrintSuccess("✅ Configuration file found")

	// Set AWS_CONFIG_FILE environment variable
	if err := os.Setenv("AWS_CONFIG_FILE", filepath.Join(".", awsConfigPath)); err != nil {
		return fmt.Errorf("failed to set AWS_CONFIG_FILE: %w", err)
	}

	// Get profiles from configuration
	profiles, err := getAWSProfiles()
	if err != nil {
		return fmt.Errorf("❌ Error reading profiles: %v", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No profiles found in .aws/config")
		return nil // Exit silently with success code
	}

	utils.PrintInfo("📋 Profiles found:")
	for _, profile := range profiles {
		utils.PrintText("  - %s\n", profile)
	}

	fmt.Println()
	utils.PrintInfo("📋 Verifying all profiles...")

	// Verify each profile
	for _, profile := range profiles {
		if err := verifyProfile(profile); err != nil {
			utils.PrintError("❌ Error verifying %s: %v", profile, err)
		}
	}

	fmt.Println()
	utils.PrintSuccess("🎯 Verification completed!")
	return nil
}

// verifyProfile verifies a single AWS profile
func verifyProfile(profile string) error {

	// Get expected account ID from configuration
	expectedAccount, err := getExpectedAccountID(profile)
	if err != nil {
		return fmt.Errorf("❌ %s: Could not get account_id from configuration file", profile)
	}

	// Check if credentials are valid
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile)
	cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_CONFIG_FILE=%s", filepath.Join(".", ".aws/config")))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		utils.PrintError("❌ %s: Expired or invalid credentials", profile)
		utils.PrintWarning("💡 Run: gocloud sso login --profiles %s", profile)
		return fmt.Errorf("credentials validation failed")
	}

	// Get current account ID
	cmd = exec.Command("aws", "sts", "get-caller-identity", "--profile", profile, "--query", "Account", "--output", "text")
	cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_CONFIG_FILE=%s", filepath.Join(".", ".aws/config")))

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("❌ %s: Failed to get current account ID", profile)
	}

	currentAccount := strings.TrimSpace(string(output))

	if currentAccount == expectedAccount {
		utils.PrintSuccess("✅ %s: OK (Account: %s)", profile, currentAccount)
	} else {
		utils.PrintError("❌ %s: Account mismatch (Expected: %s, Got: %s)", profile, expectedAccount, currentAccount)
	}

	return nil
}

// verifyProfileSilent verifies a single AWS profile without printing messages (for use during login)
func verifyProfileSilent(profile string) error {
	// Get expected account ID from configuration
	expectedAccount, err := getExpectedAccountID(profile)
	if err != nil {
		return fmt.Errorf("could not get account_id from configuration file")
	}

	// Check if credentials are valid
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile)
	cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_CONFIG_FILE=%s", filepath.Join(".", ".aws/config")))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("expired or invalid credentials")
	}

	// Get current account ID
	cmd = exec.Command("aws", "sts", "get-caller-identity", "--profile", profile, "--query", "Account", "--output", "text")
	cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_CONFIG_FILE=%s", filepath.Join(".", ".aws/config")))

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current account ID")
	}

	currentAccount := strings.TrimSpace(string(output))

	if currentAccount != expectedAccount {
		return fmt.Errorf("account mismatch")
	}

	return nil
}

// getExpectedAccountID extracts the expected account ID from the AWS config file
func getExpectedAccountID(profile string) (string, error) {
	content, err := os.ReadFile(".aws/config")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	inProfile := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if we're entering the target profile
		if strings.HasPrefix(line, fmt.Sprintf("[profile %s]", profile)) {
			inProfile = true
			continue
		}

		// Check if we're leaving the profile section
		if inProfile && strings.HasPrefix(line, "[") {
			break
		}

		// Extract account ID if we're in the target profile
		if inProfile && strings.HasPrefix(line, "sso_account_id = ") {
			accountID := strings.TrimPrefix(line, "sso_account_id = ")
			return strings.TrimSpace(accountID), nil
		}
	}

	return "", fmt.Errorf("account ID not found for profile %s", profile)
}
