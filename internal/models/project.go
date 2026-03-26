package models

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// LayerConfig represents which layers should be generated
// Note: Organization layer is global only, not environment-specific
type LayerConfig struct {
	Base         *bool `json:"base" yaml:"base,omitempty"`
	Foundation   *bool `json:"foundation" yaml:"foundation,omitempty"`
	Organization *bool `json:"organization" yaml:"organization,omitempty"` // Global only
}

// OrganizationLayerConfig holds layer-specific overrides for the organization layer only.
// Use this when the organization layer needs a different secrets (or other) config than the global default.
// AWSAccount is required for SSO setup/login: when set, gocloud sso setup generates a profile {client}-org.
type OrganizationLayerConfig struct {
	Secrets       *SecretsConfig               `json:"secrets" yaml:"secrets,omitempty"`
	Providers     *ProviderConfig              `json:"providers" yaml:"providers,omitempty"` // default_providers (e.g. with assume_role) for organization/providers.tf
	Backend       *BackendInfrastructureConfig `json:"backend" yaml:"backend,omitempty"`     // optional backend override for organization/backend.tf
	EnableSecrets *bool                        `json:"enable_secrets" yaml:"enable_secrets,omitempty"`
	AWSAccount    string                       `json:"aws_account" yaml:"aws_account,omitempty"` // AWS account ID for organization (SSO profile client-org)
	AWSSSO        *SSOConfig                   `json:"aws_sso" yaml:"aws_sso,omitempty"`         // Optional SSO overrides for organization profile
}

// SecretsConfig represents the secrets backend configuration
type SecretsConfig struct {
	Type string `json:"type" yaml:"type"` // "ssm" or "sops" (default: "ssm")
}

// InfrastructureConfig represents the configuration for a new infrastructure project
type InfrastructureConfig struct {
	Client           string                       `json:"client" yaml:"client"`
	Company          string                       `json:"company" yaml:"company"`
	Region           string                       `json:"region" yaml:"region"`
	Version          string                       `json:"version" yaml:"version"`
	Source           string                       `json:"source" yaml:"source,omitempty"`
	SourceRef        string                       `json:"source_ref" yaml:"source_ref,omitempty"`
	EnableSecrets    *bool                        `json:"enable_secrets" yaml:"enable_secrets,omitempty"`
	EnableTerragrunt *bool                        `json:"enable_terragrunt" yaml:"enable_terragrunt,omitempty"`
	Layers           *LayerConfig                 `json:"layers" yaml:"layers,omitempty"`
	Backend          *BackendInfrastructureConfig `json:"backend" yaml:"backend,omitempty"`
	AWSSSO           *SSOConfig                   `json:"aws_sso" yaml:"aws_sso,omitempty"`
	Metadata         map[string]interface{}       `json:"metadata" yaml:"metadata,omitempty"`
	Environments     map[string]Environment       `json:"environments" yaml:"environments"`
	EnvironmentOrder []string                     `json:"environment_order" yaml:"environment_order,omitempty"`
	Providers        *ProviderConfig              `json:"providers" yaml:"providers,omitempty"`
	// Secrets backend configuration
	Secrets *SecretsConfig `json:"secrets" yaml:"secrets,omitempty"`
	// Organization layer overrides (secrets, etc.); only applies to the organization layer
	Organization *OrganizationLayerConfig `json:"organization" yaml:"organization,omitempty"`
}

// ProjectItem represents a project item that can be either a string or an object
type ProjectItem struct {
	Key              string                       `json:"key" yaml:"key,omitempty"`
	Name             string                       `json:"name" yaml:"name,omitempty"`
	DirName          string                       `json:"dir_name" yaml:"dir_name,omitempty"`
	EnableTerragrunt *bool                        `json:"enable_terragrunt" yaml:"enable_terragrunt,omitempty"`
	DependsOn        []string                     `json:"depends_on" yaml:"depends_on,omitempty"`
	Providers        *ProviderConfig              `json:"providers" yaml:"providers,omitempty"`
	Backend          *BackendInfrastructureConfig `json:"backend" yaml:"backend,omitempty"`
	Secrets          *SecretsConfig               `json:"secrets" yaml:"secrets,omitempty"`
}

// WorkloadItem represents a workload item that can be either a string or an object
type WorkloadItem struct {
	Key              string   `json:"key" yaml:"key,omitempty"`
	Name             string   `json:"name" yaml:"name,omitempty"`
	DirName          string   `json:"dir_name" yaml:"dir_name,omitempty"`
	EnableSecrets    *bool    `json:"enable_secrets" yaml:"enable_secrets,omitempty"`
	EnableTerragrunt *bool    `json:"enable_terragrunt" yaml:"enable_terragrunt,omitempty"`
	DependsOn        []string `json:"depends_on" yaml:"depends_on,omitempty"`
	// New provider and backend configuration
	Providers *ProviderConfig              `json:"providers" yaml:"providers,omitempty"`
	Backend   *BackendInfrastructureConfig `json:"backend" yaml:"backend,omitempty"`
	// Secrets backend configuration
	Secrets *SecretsConfig `json:"secrets" yaml:"secrets,omitempty"`
}

// Environment represents an environment configuration
type Environment struct {
	Name             string        `json:"name" yaml:"name"`
	DirName          string        `json:"dir_name" yaml:"dir_name,omitempty"`
	AWSAccount       string        `json:"aws_account" yaml:"aws_account"`
	Region           string        `json:"region" yaml:"region,omitempty"`
	Version          string        `json:"version" yaml:"version,omitempty"`
	Source           string        `json:"source" yaml:"source,omitempty"`
	SourceRef        string        `json:"source_ref" yaml:"source_ref,omitempty"`
	EnableSecrets    *bool         `json:"enable_secrets" yaml:"enable_secrets,omitempty"`
	EnableTerragrunt *bool         `json:"enable_terragrunt" yaml:"enable_terragrunt,omitempty"`
	EnableSSO        *bool         `json:"enable_sso" yaml:"enable_sso,omitempty"`
	Layers           *LayerConfig  `json:"layers" yaml:"layers,omitempty"`
	AWSSSO           *SSOConfig    `json:"aws_sso" yaml:"aws_sso,omitempty"`
	Projects         []interface{} `json:"projects" yaml:"projects"`
	Workloads        []interface{} `json:"workloads" yaml:"workloads"`
	DependsOn        []string      `json:"depends_on" yaml:"depends_on,omitempty"`
	// New provider and backend configuration
	Providers *ProviderConfig              `json:"providers" yaml:"providers,omitempty"`
	Backend   *BackendInfrastructureConfig `json:"backend" yaml:"backend,omitempty"`
	// Secrets backend configuration
	Secrets *SecretsConfig `json:"secrets" yaml:"secrets,omitempty"`
}

// Layer represents an infrastructure layer
type Layer struct {
	Name         string   `json:"name" yaml:"name"`
	Path         string   `json:"path" yaml:"path"`
	Dependencies []string `json:"dependencies" yaml:"dependencies"`
	HasProject   bool     `json:"has_project" yaml:"has_project"`
}

// ProcessedEnvironment represents an environment with resolved AWS SSO settings
type ProcessedEnvironment struct {
	StartURL   string `json:"start_url"`
	Region     string `json:"region"`
	RoleName   string `json:"role_name"`
	Profile    string `json:"profile"`
	AWSAccount string `json:"aws_account"`
}

// ShouldEnableSSO determines if SSO should be enabled for an environment
func ShouldEnableSSO(env Environment) bool {
	if env.EnableSSO != nil {
		return *env.EnableSSO
	}
	// Default to true if not specified
	return true
}

// ProcessEnvironments processes environments to resolve AWS SSO settings
func ProcessEnvironments(config *InfrastructureConfig) map[string]ProcessedEnvironment {
	processed := make(map[string]ProcessedEnvironment)
	globalSSO := config.AWSSSO

	for envKey, env := range config.Environments {
		// Process ALL environments, regardless of SSO setting
		processed[envKey] = ProcessedEnvironment{
			StartURL:   resolveStartURL(env, globalSSO),
			Region:     resolveRegion(env, globalSSO),
			RoleName:   resolveRoleName(env, globalSSO),
			Profile:    fmt.Sprintf("%s-%s", config.Client, envKey),
			AWSAccount: env.AWSAccount,
		}
	}

	return processed
}

// Helper functions for deterministic map processing

// ToMapStringInterface converts map[interface{}]interface{} (from yaml.v3 unmarshal into interface{})
// to map[string]interface{} so existing logic can use string keys. Returns nil if not a map.
// Exported for use by cmd and other packages.
func ToMapStringInterface(m interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	if out, ok := m.(map[string]interface{}); ok {
		return out
	}
	in, ok := m.(map[interface{}]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		sk, ok := k.(string)
		if !ok {
			continue
		}
		if v == nil {
			out[sk] = nil
			continue
		}
		if vm, ok := v.(map[interface{}]interface{}); ok {
			out[sk] = ToMapStringInterface(vm)
		} else if vm, ok := v.(map[string]interface{}); ok {
			out[sk] = vm
		} else if sl, ok := v.([]interface{}); ok {
			converted := make([]interface{}, len(sl))
			for i, item := range sl {
				if vm, ok := item.(map[interface{}]interface{}); ok {
					converted[i] = ToMapStringInterface(vm)
				} else if vm, ok := item.(map[string]interface{}); ok {
					converted[i] = vm
				} else {
					converted[i] = item
				}
			}
			out[sk] = converted
		} else {
			out[sk] = v
		}
	}
	return out
}

// getMapKey extracts the project/workload key from a map
func getMapKey(m map[string]interface{}) string {
	if len(m) == 0 {
		return ""
	}
	// For YAML parsing issue: find the key that is not "name"
	// This handles the case where YAML parses as: map[dept:<nil> name:Deposits]
	for k, v := range m {
		if k != "name" && v == nil {
			return k
		}
	}
	// Fallback: return the first key
	for k := range m {
		return k
	}
	return ""
}

// getNestedMap extracts the nested map from a single-key map
func getNestedMap(m map[string]interface{}) map[string]interface{} {
	if len(m) != 1 {
		return nil
	}
	for _, value := range m {
		if nested, ok := value.(map[string]interface{}); ok {
			return nested
		}
	}
	return nil
}

// convertToStringSlice converts []interface{} to []string.
// Handles YAML-unmarshaled slices where elements may be string or other types.
func convertToStringSlice(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, item := range slice {
		if str, ok := item.(string); ok {
			result[i] = str
		} else if item != nil {
			result[i] = fmt.Sprint(item)
		}
	}
	return result
}

// hasExplicitDependsOnProject returns true when the project item has depends_on set (including to []).
// Used to distinguish "no key" (use default) from "depends_on: []" (no dependencies).
func hasExplicitDependsOnProject(item interface{}) bool {
	switch v := item.(type) {
	case map[interface{}]interface{}:
		if m := ToMapStringInterface(v); m != nil {
			return hasExplicitDependsOnProject(m)
		}
		return false
	case map[string]interface{}:
		if nested := getNestedMap(v); nested != nil {
			if _, ok := nested["depends_on"]; ok {
				return true
			}
		}
		_, ok := v["depends_on"]
		return ok
	case ProjectItem:
		return v.DependsOn != nil
	}
	return false
}

// hasExplicitDependsOnWorkload returns true when the workload item has depends_on set (including to []).
// Used to distinguish "no key" (use default) from "depends_on: []" (no dependencies).
func hasExplicitDependsOnWorkload(item interface{}) bool {
	switch v := item.(type) {
	case map[interface{}]interface{}:
		if m := ToMapStringInterface(v); m != nil {
			return hasExplicitDependsOnWorkload(m)
		}
		return false
	case map[string]interface{}:
		if nested := getNestedMap(v); nested != nil {
			if _, ok := nested["depends_on"]; ok {
				return true
			}
		}
		_, ok := v["depends_on"]
		return ok
	case WorkloadItem:
		return v.DependsOn != nil
	}
	return false
}

// NameToDirName converts a display name to a directory-safe name: lowercase and spaces to underscores.
// Used when name is used as directory and dir_name is not specified (project, workload, environment).
func NameToDirName(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), "_")
}

// GetProjectDisplayName extracts the display name for a project (name if specified, otherwise key)
func GetProjectDisplayName(item interface{}) string {
	switch v := item.(type) {
	case string:
		return v
	case map[interface{}]interface{}:
		if m := ToMapStringInterface(v); m != nil {
			return GetProjectDisplayName(m)
		}
		return ""
	case map[string]interface{}:
		// Check for nested map first (new format: key: {name: "value"})
		if nested := getNestedMap(v); nested != nil {
			if name, ok := nested["name"].(string); ok {
				return name
			}
		}
		// Check for direct name field (old format: {name: "value"})
		if name, ok := v["name"].(string); ok {
			return name
		}
		// Fallback to key
		return getMapKey(v)
	case ProjectItem:
		if v.Name != "" {
			return v.Name
		}
		return v.Key
	}
	return ""
}

// GetProjectDirectoryName extracts the directory name for a project
// Priority: dir_name > name (lowercase, spaces to _) > key
func GetProjectDirectoryName(item interface{}) string {
	switch v := item.(type) {
	case string:
		return v
	case map[interface{}]interface{}:
		if m := ToMapStringInterface(v); m != nil {
			return GetProjectDirectoryName(m)
		}
		return ""
	case map[string]interface{}:
		// Check for nested map first (new format: key: {dir_name: "value", name: "value"})
		if nested := getNestedMap(v); nested != nil {
			// 1. dir_name (highest priority)
			if dirName, ok := nested["dir_name"].(string); ok {
				return dirName
			}
			// 2. name (lowercase, spaces to _)
			if name, ok := nested["name"].(string); ok {
				return NameToDirName(name)
			}
		}
		// Check for direct fields (old format: {dir_name: "value", name: "value"})
		if dirName, ok := v["dir_name"].(string); ok {
			return dirName
		}
		if name, ok := v["name"].(string); ok {
			return NameToDirName(name)
		}
		// 3. key (fallback)
		return getMapKey(v)
	case ProjectItem:
		if v.DirName != "" {
			return v.DirName
		}
		if v.Name != "" {
			return NameToDirName(v.Name)
		}
		return v.Key
	}
	return ""
}

// GetProjectKey extracts the key for a project (always the identifier)
func GetProjectKey(item interface{}) string {
	switch v := item.(type) {
	case string:
		return v
	case map[string]interface{}:
		return getMapKey(v)
	case map[interface{}]interface{}:
		return getMapKey(ToMapStringInterface(v))
	case ProjectItem:
		return v.Key
	}
	return ""
}

// GetProjectDependencies extracts dependencies from an interface{} that can be either a string or ProjectItem
func GetProjectDependencies(item interface{}) []string {
	switch v := item.(type) {
	case string:
		return []string{}
	case map[interface{}]interface{}:
		if m := ToMapStringInterface(v); m != nil {
			return GetProjectDependencies(m)
		}
		return []string{}
	case map[string]interface{}:
		// Check for nested map first (new format: key: {depends_on: [...]})
		if nested := getNestedMap(v); nested != nil {
			if deps, ok := nested["depends_on"].([]interface{}); ok {
				return convertToStringSlice(deps)
			}
		}
		// Check for direct field (old format: {depends_on: [...]})
		if deps, ok := v["depends_on"].([]interface{}); ok {
			return convertToStringSlice(deps)
		}
	case ProjectItem:
		return v.DependsOn
	}
	return []string{}
}

// GetWorkloadDisplayName extracts the display name for a workload (name if specified, otherwise key)
func GetWorkloadDisplayName(item interface{}) string {
	switch v := item.(type) {
	case string:
		return v
	case map[interface{}]interface{}:
		if m := ToMapStringInterface(v); m != nil {
			return GetWorkloadDisplayName(m)
		}
		return ""
	case map[string]interface{}:
		// Check for nested map first (new format: key: {name: "value"})
		if nested := getNestedMap(v); nested != nil {
			if name, ok := nested["name"].(string); ok {
				return name
			}
		}
		// Check for direct name field (old format: {name: "value"})
		if name, ok := v["name"].(string); ok {
			return name
		}
		// Fallback to key
		return getMapKey(v)
	case WorkloadItem:
		if v.Name != "" {
			return v.Name
		}
		return v.Key
	}
	return ""
}

// GetWorkloadDirectoryName extracts the directory name for a workload
// Priority: dir_name > name (lowercase, spaces to _) > key
func GetWorkloadDirectoryName(item interface{}) string {
	switch v := item.(type) {
	case string:
		return v
	case map[interface{}]interface{}:
		if m := ToMapStringInterface(v); m != nil {
			return GetWorkloadDirectoryName(m)
		}
		return ""
	case map[string]interface{}:
		// Check for nested map first (new format: key: {dir_name: "value", name: "value"})
		if nested := getNestedMap(v); nested != nil {
			// 1. dir_name (highest priority)
			if dirName, ok := nested["dir_name"].(string); ok {
				return dirName
			}
			// 2. name (lowercase, spaces to _)
			if name, ok := nested["name"].(string); ok {
				return NameToDirName(name)
			}
		}
		// Check for direct fields (old format: {dir_name: "value", name: "value"})
		if dirName, ok := v["dir_name"].(string); ok {
			return dirName
		}
		if name, ok := v["name"].(string); ok {
			return NameToDirName(name)
		}
		// 3. key (fallback)
		return getMapKey(v)
	case WorkloadItem:
		if v.DirName != "" {
			return v.DirName
		}
		if v.Name != "" {
			return NameToDirName(v.Name)
		}
		return v.Key
	}
	return ""
}

// GetWorkloadKey extracts the key for a workload (always the identifier)
func GetWorkloadKey(item interface{}) string {
	switch v := item.(type) {
	case string:
		return v
	case map[string]interface{}:
		return getMapKey(v)
	case map[interface{}]interface{}:
		return getMapKey(ToMapStringInterface(v))
	case WorkloadItem:
		return v.Key
	}
	return ""
}

// GetWorkloadDependencies extracts dependencies from an interface{} that can be either a string or WorkloadItem
func GetWorkloadDependencies(item interface{}) []string {
	switch v := item.(type) {
	case string:
		return []string{}
	case map[interface{}]interface{}:
		if m := ToMapStringInterface(v); m != nil {
			return GetWorkloadDependencies(m)
		}
		return []string{}
	case map[string]interface{}:
		// Check for nested map first (new format: key: {depends_on: [...]})
		if nested := getNestedMap(v); nested != nil {
			if deps, ok := nested["depends_on"].([]interface{}); ok {
				return convertToStringSlice(deps)
			}
		}
		// Check for direct field (old format: {depends_on: [...]})
		if deps, ok := v["depends_on"].([]interface{}); ok {
			return convertToStringSlice(deps)
		}
	case WorkloadItem:
		return v.DependsOn
	}
	return []string{}
}

// CalculateDependencies calculates the default dependencies for a layer/project combination
// and applies any overrides from the configuration
func CalculateDependencies(layer, project, envKey string, config *InfrastructureConfig) []string {
	// Check if there's an explicit depends_on configuration for this environment
	envConfig, exists := config.Environments[envKey]
	if exists && len(envConfig.DependsOn) > 0 {
		// If depends_on is empty array, return no dependencies
		if len(envConfig.DependsOn) == 1 && envConfig.DependsOn[0] == "" {
			return []string{}
		}
		// Return the explicit dependencies
		return envConfig.DependsOn
	}

	// Get the directory name for this environment using the fallback logic
	dirName := envKey
	if exists {
		// Option 1: Use dir_name if specified
		if envConfig.DirName != "" {
			dirName = envConfig.DirName
		} else if envConfig.Name != "" {
			// Option 2: Use name as directory (lowercase, spaces to _)
			dirName = NameToDirName(envConfig.Name)
		}
		// Option 3: Use environment key (fallback) - already set above
	}

	// Default dependency logic
	switch layer {
	case "base":
		return []string{} // base doesn't depend on anything
	case "organization":
		return []string{} // organization is global, no dependencies (like base)
	case "foundation":
		return []string{"../../base/" + dirName}
	case "project":
		if exists {
			for _, projectItem := range envConfig.Projects {
				if GetProjectKey(projectItem) != project {
					continue
				}
				if !hasExplicitDependsOnProject(projectItem) {
					break
				}
				deps := GetProjectDependencies(projectItem)
				var out []string
				for _, dep := range deps {
					if strings.HasPrefix(dep, "foundation") {
						out = append(out, "../../../foundation/"+dirName)
					} else if strings.HasPrefix(dep, "base") {
						out = append(out, "../../base/"+dirName)
					} else {
						out = append(out, dep)
					}
				}
				return out
			}
		}
		return []string{"../../../foundation/" + dirName}
	case "workload":
		// Check if there are explicit dependencies for this workload (including depends_on: [] for none)
		if exists {
			for _, workloadItem := range envConfig.Workloads {
				workloadKey := GetWorkloadKey(workloadItem)
				if workloadKey != project {
					continue
				}
				if !hasExplicitDependsOnWorkload(workloadItem) {
					break
				}
				workloadDeps := GetWorkloadDependencies(workloadItem)
				// Explicit depends_on: [] means no dependencies
				var dependencies []string
				for _, dep := range workloadDeps {
					if strings.HasPrefix(dep, "project/") {
						// Convert "project/key" to "../../../project/dirName/envDirName"
						projectKey := strings.TrimPrefix(dep, "project/")
						for _, projItem := range envConfig.Projects {
							if GetProjectKey(projItem) == projectKey {
								projectDirName := GetProjectDirectoryName(projItem)
								dependencies = append(dependencies, "../../../project/"+projectDirName+"/"+dirName)
								break
							}
						}
					} else if strings.HasPrefix(dep, "foundation") {
						dependencies = append(dependencies, "../../../foundation/"+dirName)
					} else if strings.HasPrefix(dep, "base") {
						dependencies = append(dependencies, "../../../base/"+dirName)
					} else {
						dependencies = append(dependencies, dep)
					}
				}
				return dependencies
			}
		}

		// Default logic: workload depends on project with the same key
		// if the project doesn't exist, try to find a matching project by key
		// then fall back to common

		// Check if the project exists in the configuration
		envConfig, exists := config.Environments[envKey]
		if exists {
			// First, check if there's an exact match by key
			for _, projectItem := range envConfig.Projects {
				projectKey := GetProjectKey(projectItem)
				if projectKey == project {
					projectDirName := GetProjectDirectoryName(projectItem)
					return []string{"../../../project/" + projectDirName + "/" + dirName}
				}
			}

			// If no exact match, try to find a project that contains the workload key
			// or a workload that contains the project key
			for _, projectItem := range envConfig.Projects {
				projectKey := GetProjectKey(projectItem)
				if strings.Contains(project, projectKey) || strings.Contains(projectKey, project) {
					projectDirName := GetProjectDirectoryName(projectItem)
					return []string{"../../../project/" + projectDirName + "/" + dirName}
				}
			}

			// If still no match, try common as fallback
			for _, projectItem := range envConfig.Projects {
				projectKey := GetProjectKey(projectItem)
				if projectKey == "common" {
					projectDirName := GetProjectDirectoryName(projectItem)
					return []string{"../../../project/" + projectDirName + "/" + dirName}
				}
			}
		}

		// If no suitable project found, return empty dependencies
		return []string{}
	default:
		return []string{}
	}
}

// Helper functions to resolve AWS SSO settings
func resolveStartURL(env Environment, globalSSO *SSOConfig) string {
	if env.AWSSSO != nil && env.AWSSSO.StartURL != "" {
		return env.AWSSSO.StartURL
	}
	if globalSSO != nil {
		return globalSSO.StartURL
	}
	return ""
}

func resolveRegion(env Environment, globalSSO *SSOConfig) string {
	if env.AWSSSO != nil && env.AWSSSO.Region != "" {
		return env.AWSSSO.Region
	}
	if globalSSO != nil {
		return globalSSO.Region
	}
	return ""
}

func resolveRoleName(env Environment, globalSSO *SSOConfig) string {
	if env.AWSSSO != nil && env.AWSSSO.RoleName != "" {
		return env.AWSSSO.RoleName
	}
	if globalSSO != nil {
		return globalSSO.RoleName
	}
	return ""
}

// ResolveVersion resolves the version with priority: environment > global
func ResolveVersion(env Environment, globalVersion string) string {
	if env.Version != "" {
		return env.Version
	}
	return globalVersion
}

// ResolveBackendConfig resolves backend configuration with defaults
func ResolveBackendConfig(config *InfrastructureConfig) *BackendInfrastructureConfig {
	if config.Backend != nil {
		backend := *config.Backend
		if backend.Region == "" {
			backend.Region = config.Region
		}
		if backend.Account == "" {
			backend.Account = "sha"
		}
		return &backend
	}

	// Default configuration
	return &BackendInfrastructureConfig{
		Pattern: "s3-backend",
		Region:  config.Region,
		Account: "sha",
		Encrypt: true,
	}
}

// TemplateData represents data passed to templates
type TemplateData struct {
	Client                string                          `json:"client"`
	Company               string                          `json:"company"`
	Region                string                          `json:"region"`
	RegionShortCode       string                          `json:"region_short_code"`
	Version               string                          `json:"version"`
	Source                string                          `json:"source"`
	SourceRef             string                          `json:"source_ref"`
	IsGitSource           bool                            `json:"is_git_source"`
	BackendPattern        string                          `json:"backend_pattern"`
	BackendRegion         string                          `json:"backend_region"`
	BackendAccount        string                          `json:"backend_account"`
	BackendEncrypt        bool                            `json:"backend_encrypt"`
	BackendBucketName     string                          `json:"backend_bucket_name"`
	BackendDynamoDBTable  string                          `json:"backend_dynamodb_table"`
	AWSSSO                *SSOConfig                      `json:"aws_sso"`
	Environments          map[string]Environment          `json:"environments"`
	ProcessedEnvironments map[string]ProcessedEnvironment `json:"processed_environments"`
	Layer                 string                          `json:"layer"`
	Project               string                          `json:"project,omitempty"`      // Key del project
	ProjectKey            string                          `json:"project_key,omitempty"`  // Key del project (alias, deprecated)
	ProjectName           string                          `json:"project_name,omitempty"` // Nombre del project
	Environment           string                          `json:"environment"`            // Key del environment
	EnvironmentName       string                          `json:"environment_name"`       // Nombre del environment
	EnvKey                string                          `json:"env_key"`                // Key del environment (alias, deprecated)
	CommonName            string                          `json:"common_name"`
	CommonNamePrefix      string                          `json:"common_name_prefix"`
	Metadata              map[string]interface{}          `json:"metadata"`
	MetadataLines         []string                        `json:"metadata_lines,omitempty"`
	Dependencies          []string                        `json:"dependencies,omitempty"`
	// Provider and backend template data
	Providers         []ProviderSpec    `json:"providers,omitempty"`
	BackendType       string            `json:"backend_type,omitempty"`
	BackendBucket     string            `json:"backend_bucket,omitempty"`
	BackendKey        string            `json:"backend_key,omitempty"`
	BackendProfile    string            `json:"backend_profile,omitempty"`
	BackendAssumeRole *AssumeRoleConfig `json:"backend_assume_role,omitempty"`
	// Secrets backend type
	SecretsBackendType string `json:"secrets_backend_type,omitempty"` // "ssm" or "sops"
}

// ProviderTemplateData represents data for rendering providers.tf template
type ProviderTemplateData struct {
	Providers []ProviderSpec `json:"providers"`
}

// BackendTemplateData represents data for rendering backend.tf template
type BackendTemplateData struct {
	Type          string            `json:"type"`
	Bucket        string            `json:"bucket"`
	Key           string            `json:"key"`
	Region        string            `json:"region"`
	DynamoDBTable string            `json:"dynamodb_table,omitempty"`
	Encrypt       bool              `json:"encrypt"`
	Profile       string            `json:"profile,omitempty"`
	AssumeRole    *AssumeRoleConfig `json:"assume_role,omitempty"`
}

// AWSConfig represents AWS configuration
type AWSConfig struct {
	Region        string            `json:"region" yaml:"region"`
	Profiles      map[string]string `json:"profiles" yaml:"profiles"`
	SSOConfig     SSOConfig         `json:"sso_config" yaml:"sso_config"`
	BackendConfig BackendConfig     `json:"backend_config" yaml:"backend_config"`
}

// SSOConfig represents AWS SSO configuration
type SSOConfig struct {
	StartURL string `json:"start_url" yaml:"start_url"`
	Region   string `json:"region" yaml:"region"`
	RoleName string `json:"role_name" yaml:"role_name"`
}

// BackendConfig represents Terraform backend configuration
type BackendConfig struct {
	Bucket        string `json:"bucket" yaml:"bucket"`
	Region        string `json:"region" yaml:"region"`
	DynamoDBTable string `json:"dynamodb_table" yaml:"dynamodb_table"`
	Encrypt       bool   `json:"encrypt" yaml:"encrypt"`
}

// BackendInfrastructureConfig represents backend configuration for infrastructure
type BackendInfrastructureConfig struct {
	Pattern           string `json:"pattern" yaml:"pattern"`
	Region            string `json:"region" yaml:"region,omitempty"`
	Account           string `json:"account" yaml:"account"`
	Encrypt           bool   `json:"encrypt" yaml:"encrypt"`
	BucketName        string `json:"bucket_name" yaml:"bucket_name,omitempty"`
	DynamoDBTableName string `json:"dynamodb_table_name" yaml:"dynamodb_table_name,omitempty"`
	// New features
	Type         string `json:"type" yaml:"type,omitempty"`                   // "s3" (default)
	KeyTemplate  string `json:"key_template" yaml:"key_template,omitempty"`   // Template for S3 key
	RoleTemplate string `json:"role_template" yaml:"role_template,omitempty"` // Template for role name
	UseProfile   *bool  `json:"use_profile" yaml:"use_profile,omitempty"`     // Control profiles
}

// ProviderAssumeRole represents assume_role block for a provider (e.g. AWS cross-account)
type ProviderAssumeRole struct {
	RoleARN     string `json:"role_arn" yaml:"role_arn"`
	SessionName string `json:"session_name" yaml:"session_name,omitempty"` // Optional, e.g. "TerraformSession"
}

// ProviderSpec represents a single provider configuration.
// Extra (or top-level keys in YAML other than name, region, alias, profile, assume_role) are rendered as-is in providers.tf;
// values in Extra should be valid HCL right-hand side (e.g. "value" for strings, true/false, numbers).
type ProviderSpec struct {
	Name       string              `json:"name" yaml:"name"`             // "aws", "gitlab", etc.
	Region     string              `json:"region" yaml:"region"`         // AWS: "us-east-1" or "local.metadata.aws_region"
	Alias      string              `json:"alias" yaml:"alias,omitempty"` // "use1"
	Profile    string              `json:"profile" yaml:"profile,omitempty"`
	AssumeRole *ProviderAssumeRole `json:"assume_role" yaml:"assume_role,omitempty"`
	Extra      map[string]string   `json:"extra" yaml:"extra,omitempty"` // Arbitrary provider args (e.g. base_url for gitlab)
}

// knownProviderSpecKeys are top-level keys that are not put into Extra when parsing from YAML/map.
var knownProviderSpecKeys = map[string]bool{
	"name": true, "region": true, "alias": true, "profile": true, "assume_role": true, "extra": true,
}

// toHCLValue formats a value as HCL right-hand side (quoted string, true/false, or number).
func toHCLValue(v interface{}) string {
	if v == nil {
		return `""`
	}
	switch val := v.(type) {
	case string:
		return `"` + strings.ReplaceAll(val, `\`, `\\`) + `"`
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return `"` + strings.ReplaceAll(fmt.Sprint(val), `\`, `\\`) + `"`
	}
}

// UnmarshalYAML implements yaml.Unmarshaler so ProviderSpec can capture arbitrary top-level keys (e.g. base_url for gitlab) into Extra.
func (s *ProviderSpec) UnmarshalYAML(value *yaml.Node) error {
	var raw map[string]interface{}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	pm := ToMapStringInterface(raw)
	if pm == nil {
		return nil
	}
	if v, ok := pm["name"].(string); ok {
		s.Name = v
	}
	if v, ok := pm["region"].(string); ok {
		s.Region = v
	}
	if v, ok := pm["alias"].(string); ok {
		s.Alias = v
	}
	if v, ok := pm["profile"].(string); ok {
		s.Profile = v
	}
	if ar, ok := pm["assume_role"].(map[string]interface{}); ok {
		arStr := ToMapStringInterface(ar)
		if arStr != nil {
			assume := &ProviderAssumeRole{}
			if v, ok := arStr["role_arn"].(string); ok {
				assume.RoleARN = v
			}
			if v, ok := arStr["session_name"].(string); ok {
				assume.SessionName = v
			}
			if assume.RoleARN != "" {
				s.AssumeRole = assume
			}
		}
	}
	// Contents of "extra" key
	if ex, ok := pm["extra"]; ok {
		if exMap := ToMapStringInterface(ex); exMap != nil {
			if s.Extra == nil {
				s.Extra = make(map[string]string)
			}
			for k, v := range exMap {
				s.Extra[k] = toHCLValue(v)
			}
		}
	}
	// Any other top-level key (e.g. base_url) -> Extra
	for k, v := range pm {
		if knownProviderSpecKeys[k] {
			continue
		}
		if s.Extra == nil {
			s.Extra = make(map[string]string)
		}
		s.Extra[k] = toHCLValue(v)
	}
	return nil
}

// ProviderConfig represents provider configuration with hierarchy support
type ProviderConfig struct {
	// Global provider configuration
	DefaultProviders []ProviderSpec `json:"default_providers" yaml:"default_providers,omitempty"`

	// Control of profiles
	UseProfiles *bool `json:"use_profiles" yaml:"use_profiles,omitempty"`

	// Environment-specific overrides
	EnvironmentOverrides map[string]ProviderConfig `json:"environment_overrides" yaml:"environment_overrides,omitempty"`

	// Project/workload-specific overrides
	ProjectOverrides map[string]ProviderConfig `json:"project_overrides" yaml:"project_overrides,omitempty"`
}

// AssumeRoleConfig represents assume role configuration for backend
type AssumeRoleConfig struct {
	RoleARN string `json:"role_arn" yaml:"role_arn"`
	Pattern string `json:"pattern" yaml:"pattern,omitempty"` // "{{.Company}}-{{.BackendAccount}}-{{.BackendPattern}}-{{.AccountID}}"
}

// ModuleVersion represents a module version
type ModuleVersion struct {
	Name    string `json:"name" yaml:"name"`
	Current string `json:"current" yaml:"current"`
	Latest  string `json:"latest" yaml:"latest"`
	Source  string `json:"source" yaml:"source"`
}

// ValidationResult represents validation results
type ValidationResult struct {
	Valid    bool     `json:"valid" yaml:"valid"`
	Errors   []string `json:"errors" yaml:"errors"`
	Warnings []string `json:"warnings" yaml:"warnings"`
}

// DeploymentResult represents deployment results
type DeploymentResult struct {
	Success bool     `json:"success" yaml:"success"`
	Errors  []string `json:"errors" yaml:"errors"`
	Output  string   `json:"output" yaml:"output"`
}

// SourceConfig represents resolved source configuration
type SourceConfig struct {
	Source    string
	SourceRef string
	IsGit     bool
}

// GetSource resolves the source configuration for an environment
// Priority: Environment source > Global source > Registry (fallback)
func (env Environment) GetSource(infra *InfrastructureConfig) SourceConfig {
	// 1. Check if environment has source
	if env.Source != "" {
		ref := env.SourceRef
		if ref == "" {
			ref = "main" // default ref
		}
		return SourceConfig{
			Source:    env.Source,
			SourceRef: ref,
			IsGit:     true,
		}
	}

	// 2. Check if global has source
	if infra.Source != "" {
		ref := infra.SourceRef
		if ref == "" {
			ref = "main" // default ref
		}
		return SourceConfig{
			Source:    infra.Source,
			SourceRef: ref,
			IsGit:     true,
		}
	}

	// 3. Fallback to registry
	return SourceConfig{
		Source:    "",
		SourceRef: "",
		IsGit:     false,
	}
}

// GetVersion resolves the version for an environment
// Priority: Environment version > Global version > "latest" (fallback)
func (env Environment) GetVersion(infra *InfrastructureConfig) string {
	if env.Version != "" {
		return env.Version
	}
	if infra.Version != "" {
		return infra.Version
	}
	return "latest"
}

// GetEnvironmentOrder returns the environments in the order they were defined
func (config *InfrastructureConfig) GetEnvironmentOrder() []string {
	// If we have explicit order, use it
	if len(config.EnvironmentOrder) > 0 {
		return config.EnvironmentOrder
	}

	// Fallback: collect all environment keys and sort them
	var keys []string
	for key := range config.Environments {
		keys = append(keys, key)
	}

	// Sort to ensure consistent order
	// Note: This is a fallback - ideally EnvironmentOrder should be set during config loading
	sort.Strings(keys)
	return keys
}

// IsOrganizationEnabled reports whether the special organization layer/profile is enabled.
// Rule: organization.aws_account must be set, unless layers.organization is explicitly false.
func IsOrganizationEnabled(config *InfrastructureConfig) bool {
	if config == nil || config.Organization == nil || config.Organization.AWSAccount == "" {
		return false
	}
	if config.Layers != nil && config.Layers.Organization != nil && !*config.Layers.Organization {
		return false
	}
	return true
}

// ResolveProviderConfig resolves provider configuration with hierarchy
// Priority: Organization override > Workload > Project > Environment > Global
func (config *InfrastructureConfig) ResolveProviderConfig(layerType, projectKey, envKey string) *ProviderConfig {
	// Start with global configuration
	var result *ProviderConfig
	if config.Providers != nil {
		result = &ProviderConfig{}
		*result = *config.Providers
	}

	// Apply environment overrides (not used for organization; organization has no env in config)
	if layerType != "organization" {
		if envConfig, exists := config.Environments[envKey]; exists && envConfig.Providers != nil {
			result = mergeProviderConfigs(result, envConfig.Providers)
		}
	}

	// Apply project/workload overrides
	if projectKey != "" {
		var projectConfig *ProviderConfig

		// Check if it's a project or workload
		switch layerType {
		case "project":
			if projectConfig = getProjectProviderConfig(config, envKey, projectKey); projectConfig != nil {
				result = mergeProviderConfigs(result, projectConfig)
			}
		case "workload":
			if projectConfig = getWorkloadProviderConfig(config, envKey, projectKey); projectConfig != nil {
				result = mergeProviderConfigs(result, projectConfig)
			}
		}
	}

	// Apply organization-layer-specific override (only for the organization layer)
	if layerType == "organization" && config.Organization != nil && config.Organization.Providers != nil {
		result = mergeProviderConfigs(result, config.Organization.Providers)
	}

	return result
}

// ResolveBackendConfig resolves backend configuration with hierarchy
// Priority: Organization override > Workload > Project > Environment > Global
func (config *InfrastructureConfig) ResolveBackendConfig(layerType, projectKey, envKey string) *BackendInfrastructureConfig {
	// Start with global configuration
	result := &BackendInfrastructureConfig{}
	if config.Backend != nil {
		*result = *config.Backend
	}

	// Apply environment overrides (not used for organization; organization has no env in config)
	if layerType != "organization" {
		if envConfig, exists := config.Environments[envKey]; exists && envConfig.Backend != nil {
			result = mergeBackendInfrastructureConfigs(result, envConfig.Backend)
		}
	}

	// Apply project/workload overrides
	if projectKey != "" {
		var projectConfig *BackendInfrastructureConfig

		// Check if it's a project or workload
		switch layerType {
		case "project":
			if projectConfig = getProjectBackendInfrastructureConfig(config, envKey, projectKey); projectConfig != nil {
				result = mergeBackendInfrastructureConfigs(result, projectConfig)
			}
		case "workload":
			if projectConfig = getWorkloadBackendInfrastructureConfig(config, envKey, projectKey); projectConfig != nil {
				result = mergeBackendInfrastructureConfigs(result, projectConfig)
			}
		}
	}

	// Apply organization-layer-specific override (only for the organization layer)
	if layerType == "organization" && config.Organization != nil && config.Organization.Backend != nil {
		result = mergeBackendInfrastructureConfigs(result, config.Organization.Backend)
	}

	return result
}

// getBackendConfigFromItem extracts *BackendInfrastructureConfig from a project or workload item.
// Supports both ProjectItem/WorkloadItem (from code) and map (from YAML unmarshal: map[string]interface{} or map[interface{}]interface{}).
func getBackendConfigFromItem(item interface{}) *BackendInfrastructureConfig {
	switch v := item.(type) {
	case ProjectItem:
		return v.Backend
	case WorkloadItem:
		return v.Backend
	case map[string]interface{}:
		return getBackendFromMap(v)
	case map[interface{}]interface{}:
		return getBackendFromMap(ToMapStringInterface(v))
	default:
		return nil
	}
}

func getBackendFromMap(v map[string]interface{}) *BackendInfrastructureConfig {
	nested := getNestedMap(v)
	if nested == nil {
		return nil
	}
	b := nested["backend"]
	if b == nil {
		return nil
	}
	return mapToBackendConfig(b)
}

// getProviderConfigFromItem extracts *ProviderConfig from a project or workload item.
// Supports both ProjectItem/WorkloadItem (from code) and map (from YAML unmarshal: map[string]interface{} or map[interface{}]interface{}).
func getProviderConfigFromItem(item interface{}) *ProviderConfig {
	switch v := item.(type) {
	case ProjectItem:
		return v.Providers
	case WorkloadItem:
		return v.Providers
	case map[string]interface{}:
		return getProviderFromMap(v)
	case map[interface{}]interface{}:
		return getProviderFromMap(ToMapStringInterface(v))
	default:
		return nil
	}
}

func getProviderFromMap(v map[string]interface{}) *ProviderConfig {
	nested := getNestedMap(v)
	if nested == nil {
		return nil
	}
	p := nested["providers"]
	if p == nil {
		return nil
	}
	return mapToProviderConfig(p)
}

// mapToBackendConfig builds BackendInfrastructureConfig from a map (from YAML unmarshal).
func mapToBackendConfig(m interface{}) *BackendInfrastructureConfig {
	vm := ToMapStringInterface(m)
	if vm == nil {
		return nil
	}
	out := &BackendInfrastructureConfig{}
	if v, ok := vm["pattern"].(string); ok {
		out.Pattern = v
	}
	if v, ok := vm["region"].(string); ok {
		out.Region = v
	}
	if v, ok := vm["account"].(string); ok {
		out.Account = v
	}
	if v, ok := vm["encrypt"].(bool); ok {
		out.Encrypt = v
	}
	if v, ok := vm["bucket_name"].(string); ok {
		out.BucketName = v
	}
	if v, ok := vm["dynamodb_table_name"].(string); ok {
		out.DynamoDBTableName = v
	}
	if v, ok := vm["type"].(string); ok {
		out.Type = v
	}
	if v, ok := vm["key_template"].(string); ok {
		out.KeyTemplate = v
	}
	if v, ok := vm["role_template"].(string); ok {
		out.RoleTemplate = v
	}
	if v, ok := vm["use_profile"].(bool); ok {
		out.UseProfile = &v
	}
	return out
}

// mapToProviderConfig builds ProviderConfig from a map (from YAML unmarshal).
func mapToProviderConfig(m interface{}) *ProviderConfig {
	vm := ToMapStringInterface(m)
	if vm == nil {
		return nil
	}
	out := &ProviderConfig{}
	if v, ok := vm["use_profiles"].(bool); ok {
		out.UseProfiles = &v
	}
	if raw, ok := vm["default_providers"].([]interface{}); ok && len(raw) > 0 {
		out.DefaultProviders = make([]ProviderSpec, 0, len(raw))
		for _, r := range raw {
			pm := ToMapStringInterface(r)
			if pm != nil {
				spec := ProviderSpec{}
				if v, ok := pm["name"].(string); ok {
					spec.Name = v
				}
				if v, ok := pm["region"].(string); ok {
					spec.Region = v
				}
				if v, ok := pm["alias"].(string); ok {
					spec.Alias = v
				}
				if v, ok := pm["profile"].(string); ok {
					spec.Profile = v
				}
				if ar, ok := pm["assume_role"].(map[string]interface{}); ok {
					assume := &ProviderAssumeRole{}
					if v, ok := ar["role_arn"].(string); ok {
						assume.RoleARN = v
					}
					if v, ok := ar["session_name"].(string); ok {
						assume.SessionName = v
					}
					if assume.RoleARN != "" {
						spec.AssumeRole = assume
					}
				}
				// Any other key (e.g. base_url for gitlab) goes into Extra with HCL-formatted value
				for k, v := range pm {
					if knownProviderSpecKeys[k] {
						continue
					}
					if spec.Extra == nil {
						spec.Extra = make(map[string]string)
					}
					spec.Extra[k] = toHCLValue(v)
				}
				out.DefaultProviders = append(out.DefaultProviders, spec)
			}
		}
	}
	return out
}

// Helper functions for getting project/workload configurations (same strategy as secrets: GetKey + getXxxFromItem)
func getProjectProviderConfig(config *InfrastructureConfig, envKey, projectKey string) *ProviderConfig {
	if envConfig, exists := config.Environments[envKey]; exists {
		for _, project := range envConfig.Projects {
			if GetProjectKey(project) != projectKey {
				continue
			}
			if c := getProviderConfigFromItem(project); c != nil {
				return c
			}
		}
	}
	return nil
}

func getWorkloadProviderConfig(config *InfrastructureConfig, envKey, workloadKey string) *ProviderConfig {
	if envConfig, exists := config.Environments[envKey]; exists {
		for _, workload := range envConfig.Workloads {
			if GetWorkloadKey(workload) != workloadKey {
				continue
			}
			if c := getProviderConfigFromItem(workload); c != nil {
				return c
			}
		}
	}
	return nil
}

// Helper functions for getting project/workload backend infrastructure configurations (same strategy as secrets)
func getProjectBackendInfrastructureConfig(config *InfrastructureConfig, envKey, projectKey string) *BackendInfrastructureConfig {
	if envConfig, exists := config.Environments[envKey]; exists {
		for _, project := range envConfig.Projects {
			if GetProjectKey(project) != projectKey {
				continue
			}
			if c := getBackendConfigFromItem(project); c != nil {
				return c
			}
		}
	}
	return nil
}

func getWorkloadBackendInfrastructureConfig(config *InfrastructureConfig, envKey, workloadKey string) *BackendInfrastructureConfig {
	if envConfig, exists := config.Environments[envKey]; exists {
		for _, workload := range envConfig.Workloads {
			if GetWorkloadKey(workload) != workloadKey {
				continue
			}
			if c := getBackendConfigFromItem(workload); c != nil {
				return c
			}
		}
	}
	return nil
}

// Helper function for merging secrets configurations
func mergeSecretsConfigs(base, override *SecretsConfig) *SecretsConfig {
	result := &SecretsConfig{}

	// Copy base configuration
	if base != nil {
		*result = *base
	}

	// Apply overrides
	if override != nil {
		if override.Type != "" {
			result.Type = override.Type
		}
	}

	return result
}

// Helper function for merging backend infrastructure configurations
func mergeBackendInfrastructureConfigs(base, override *BackendInfrastructureConfig) *BackendInfrastructureConfig {
	result := &BackendInfrastructureConfig{}

	// Copy base configuration
	if base != nil {
		*result = *base
	}

	// Apply overrides
	if override != nil {
		if override.Pattern != "" {
			result.Pattern = override.Pattern
		}
		if override.Region != "" {
			result.Region = override.Region
		}
		if override.Account != "" {
			result.Account = override.Account
		}
		if override.Encrypt != result.Encrypt {
			result.Encrypt = override.Encrypt
		}
		if override.BucketName != "" {
			result.BucketName = override.BucketName
		}
		if override.DynamoDBTableName != "" {
			result.DynamoDBTableName = override.DynamoDBTableName
		}
		if override.Type != "" {
			result.Type = override.Type
		}
		if override.KeyTemplate != "" {
			result.KeyTemplate = override.KeyTemplate
		}
		if override.RoleTemplate != "" {
			result.RoleTemplate = override.RoleTemplate
		}
		if override.UseProfile != nil {
			result.UseProfile = override.UseProfile
		}
	}

	return result
}

// Helper functions for merging configurations
func mergeProviderConfigs(base, override *ProviderConfig) *ProviderConfig {
	result := &ProviderConfig{}

	// Copy base configuration
	if base != nil {
		*result = *base
	}

	// Apply overrides
	if override != nil {
		if override.DefaultProviders != nil {
			result.DefaultProviders = override.DefaultProviders
		}
		if override.UseProfiles != nil {
			result.UseProfiles = override.UseProfiles
		}
		if override.EnvironmentOverrides != nil {
			result.EnvironmentOverrides = override.EnvironmentOverrides
		}
		if override.ProjectOverrides != nil {
			result.ProjectOverrides = override.ProjectOverrides
		}
	}

	return result
}

// ResolveSecretsConfig resolves secrets configuration with hierarchy
// Priority: Organization layer override > Workload > Project > Environment > Global > Default ("ssm")
func (config *InfrastructureConfig) ResolveSecretsConfig(layerType, projectKey, envKey string) *SecretsConfig {
	// Start with global configuration
	result := &SecretsConfig{}
	if config.Secrets != nil {
		*result = *config.Secrets
	} else {
		// Default to "ssm" if no global config
		result.Type = "ssm"
	}

	// Apply environment overrides (not used for organization layer; organization has no env in config)
	if envConfig, exists := config.Environments[envKey]; exists && envConfig.Secrets != nil {
		result = mergeSecretsConfigs(result, envConfig.Secrets)
	}

	// Apply project/workload overrides
	if projectKey != "" {
		var projectConfig *SecretsConfig

		// Check if it's a project or workload
		switch layerType {
		case "project":
			if projectConfig = getProjectSecretsConfig(config, envKey, projectKey); projectConfig != nil {
				result = mergeSecretsConfigs(result, projectConfig)
			}
		case "workload":
			if projectConfig = getWorkloadSecretsConfig(config, envKey, projectKey); projectConfig != nil {
				result = mergeSecretsConfigs(result, projectConfig)
			}
		}
	}

	// Apply organization-layer-specific override (only for the organization layer)
	if layerType == "organization" && config.Organization != nil && config.Organization.Secrets != nil {
		result = mergeSecretsConfigs(result, config.Organization.Secrets)
	}

	return result
}

// getSecretsConfigFromItem extracts *SecretsConfig from a project or workload item.
// Supports both ProjectItem/WorkloadItem (from code) and map (from YAML unmarshal: map[string]interface{} or map[interface{}]interface{}).
func getSecretsConfigFromItem(item interface{}) *SecretsConfig {
	switch v := item.(type) {
	case ProjectItem:
		return v.Secrets
	case WorkloadItem:
		return v.Secrets
	case map[string]interface{}:
		return getSecretsFromMap(v)
	case map[interface{}]interface{}:
		return getSecretsFromMap(ToMapStringInterface(v))
	default:
		return nil
	}
}

func getSecretsFromMap(v map[string]interface{}) *SecretsConfig {
	nested := getNestedMap(v)
	if nested == nil {
		return nil
	}
	s := nested["secrets"]
	if s == nil {
		return nil
	}
	sm := ToMapStringInterface(s)
	if sm == nil {
		return nil
	}
	t, ok := sm["type"].(string)
	if !ok || t == "" {
		return nil
	}
	return &SecretsConfig{Type: t}
}

// Helper function for getting project secrets configuration
func getProjectSecretsConfig(config *InfrastructureConfig, envKey, projectKey string) *SecretsConfig {
	if envConfig, exists := config.Environments[envKey]; exists {
		for _, project := range envConfig.Projects {
			if GetProjectKey(project) != projectKey {
				continue
			}
			if c := getSecretsConfigFromItem(project); c != nil {
				return c
			}
		}
	}
	return nil
}

// Helper function for getting workload secrets configuration
func getWorkloadSecretsConfig(config *InfrastructureConfig, envKey, workloadKey string) *SecretsConfig {
	if envConfig, exists := config.Environments[envKey]; exists {
		for _, workload := range envConfig.Workloads {
			if GetWorkloadKey(workload) != workloadKey {
				continue
			}
			if c := getSecretsConfigFromItem(workload); c != nil {
				return c
			}
		}
	}
	return nil
}
