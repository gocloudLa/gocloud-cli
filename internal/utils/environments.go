package utils

import (
	"fmt"
	"strings"

	"gocloud-cli/internal/models"
	"gocloud-cli/internal/validation"
)

// ShowPredefinedEnvironmentOptions displays the available predefined environments
func ShowPredefinedEnvironmentOptions() {
	PrintInfo("\n📋 Available Predefined Environments:")
	PrintText("Choose from our recommended environments or create custom ones.\n")

	environments := models.GetPredefinedEnvironments()
	for _, env := range environments {
		PrintWarning("  %s (%s): %s", env.Key, env.Name, env.Description)
		if len(env.Projects) > 0 {
			PrintText("    Projects: %s\n", strings.Join(env.Projects, ", "))
		}
		if len(env.Workloads) > 0 {
			PrintText("    Workloads: %s\n", strings.Join(env.Workloads, ", "))
		}
	}
}

// PromptForPredefinedEnvironments prompts user to select predefined environments
func PromptForPredefinedEnvironments() ([]models.PredefinedEnvironment, error) {
	var selectedEnvironments []models.PredefinedEnvironment
	environments := models.GetPredefinedEnvironments()

	PrintInfo("\n🌍 Select Predefined Environments")
	PrintText("Choose which predefined environments you want to configure:\n")

	for _, env := range environments {
		question := fmt.Sprintf("Add %s environment? (y/N)", env.Name)
		selected, err := PromptYesNo(question, false)
		if err != nil {
			return nil, err
		}

		if selected {
			selectedEnvironments = append(selectedEnvironments, env)
		}
	}

	return selectedEnvironments, nil
}

// ConfigurePredefinedEnvironment configures a predefined environment with user input
func ConfigurePredefinedEnvironment(predefinedEnv models.PredefinedEnvironment) (*models.Environment, error) {
	// Validate input parameters first
	if predefinedEnv.Key == "" {
		return nil, fmt.Errorf("environment key is required")
	}
	if predefinedEnv.Name == "" {
		return nil, fmt.Errorf("environment name is required")
	}

	PrintInfo("\n--- Environment: %s (%s) ---", predefinedEnv.Name, predefinedEnv.Key)
	PrintText("Description: %s\n", predefinedEnv.Description)

	// Environment name (with default from predefined)
	name, err := PromptWithDefault("Environment name", predefinedEnv.Name)
	if err != nil {
		return nil, err
	}

	// Directory name (optional)
	dirName, err := PromptWithDefault("Directory name (optional, press Enter for default)", "")
	if err != nil {
		return nil, err
	}

	// AWS Account ID (required)
	accountID, err := PromptWithValidationRequired("AWS Account ID (12 digits)", validation.ValidateAWSAccountID)
	if err != nil {
		return nil, err
	}

	// Projects (with defaults from predefined)
	projectsDefault := strings.Join(predefinedEnv.Projects, ",")
	projectList, err := PromptList("Projects (comma-separated)", projectsDefault)
	if err != nil {
		return nil, err
	}

	// Convert project list to interface{} array
	var projects []interface{}
	for _, project := range projectList {
		projects = append(projects, project)
	}

	// Workloads (with defaults from predefined)
	workloadsDefault := strings.Join(predefinedEnv.Workloads, ",")
	workloadList, err := PromptList("Workloads (comma-separated)", workloadsDefault)
	if err != nil {
		return nil, err
	}

	// Convert workload list to interface{} array
	var workloads []interface{}
	for _, workload := range workloadList {
		workloads = append(workloads, workload)
	}

	// AWS SSO for this environment (optional)
	var envSSO *models.SSOConfig
	useEnvSSO, err := PromptYesNo("Configure environment-specific AWS SSO? (y/N)", false)
	if err != nil {
		return nil, err
	}

	if useEnvSSO {
		envSSO = &models.SSOConfig{}

		startURL, err := PromptString("Environment SSO Start URL")
		if err != nil {
			return nil, err
		}
		envSSO.StartURL = startURL

		roleName, err := PromptWithDefault("Environment SSO Role Name", fmt.Sprintf("%sAdmin", predefinedEnv.Name))
		if err != nil {
			return nil, err
		}
		envSSO.RoleName = roleName
	}

	return &models.Environment{
		Name:       name,
		DirName:    dirName,
		AWSAccount: accountID,
		AWSSSO:     envSSO,
		Projects:   projects,
		Workloads:  workloads,
	}, nil
}

// ConfigureCustomEnvironment configures an additional environment
func ConfigureCustomEnvironment() (string, *models.Environment, error) {
	PrintInfo("\n--- Additional Environment ---")

	// Environment key
	envKey, err := PromptWithValidationRequired("Environment key (1-3 characters: lowercase letters a-z and numbers 0-9)", validation.ValidateEnvironmentKey)
	if err != nil {
		return "", nil, err
	}

	// Environment name
	name, err := PromptStringRequired("Environment name")
	if err != nil {
		return "", nil, err
	}

	// Directory name (optional)
	dirName, err := PromptString("Directory name (optional)")
	if err != nil {
		return "", nil, err
	}

	// AWS Account ID (required)
	accountID, err := PromptWithValidationRequired("AWS Account ID (12 digits)", validation.ValidateAWSAccountID)
	if err != nil {
		return "", nil, err
	}

	// Projects (with default recommendations)
	defaultProjects := strings.Join(models.GetDefaultProjects(), ",")
	projectList, err := PromptList("Projects (comma-separated)", defaultProjects)
	if err != nil {
		return "", nil, err
	}

	// Convert project list to interface{} array
	var projects []interface{}
	for _, project := range projectList {
		projects = append(projects, project)
	}

	// Workloads (with default recommendations)
	defaultWorkloads := strings.Join(models.GetDefaultWorkloads(), ",")
	workloadList, err := PromptList("Workloads (comma-separated)", defaultWorkloads)
	if err != nil {
		return "", nil, err
	}

	// Convert workload list to interface{} array
	var workloads []interface{}
	for _, workload := range workloadList {
		workloads = append(workloads, workload)
	}

	// AWS SSO for this environment (optional)
	var envSSO *models.SSOConfig
	useEnvSSO, err := PromptYesNo("Configure environment-specific AWS SSO? (y/N)", false)
	if err != nil {
		return "", nil, err
	}

	if useEnvSSO {
		envSSO = &models.SSOConfig{}

		startURL, err := PromptString("Environment SSO Start URL")
		if err != nil {
			return "", nil, err
		}
		envSSO.StartURL = startURL

		roleName, err := PromptString("Environment SSO Role Name")
		if err != nil {
			return "", nil, err
		}
		envSSO.RoleName = roleName
	}

	environment := &models.Environment{
		Name:       name,
		DirName:    dirName,
		AWSAccount: accountID,
		AWSSSO:     envSSO,
		Projects:   projects,
		Workloads:  workloads,
	}

	return envKey, environment, nil
}
