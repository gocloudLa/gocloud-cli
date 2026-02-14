package models

// PredefinedEnvironment represents a predefined environment configuration
type PredefinedEnvironment struct {
	Key         string   `json:"key" yaml:"key"`
	Name        string   `json:"name" yaml:"name"`
	Description string   `json:"description" yaml:"description"`
	Projects    []string `json:"projects" yaml:"projects"`
	Workloads   []string `json:"workloads" yaml:"workloads"`
}

// GetPredefinedEnvironments returns the list of predefined environments
func GetPredefinedEnvironments() []PredefinedEnvironment {
	return []PredefinedEnvironment{
		{
			Key:         "sha",
			Name:        "Shared",
			Description: "Shared infrastructure and common services",
			Projects:    []string{"common"},
			Workloads:   []string{},
		},
		{
			Key:         "dev",
			Name:        "Development",
			Description: "Development environment for testing and development",
			Projects:    []string{"common", "core"},
			Workloads:   []string{"core"},
		},
		{
			Key:         "stg",
			Name:        "Staging",
			Description: "Staging environment for pre-production testing",
			Projects:    []string{"common", "core"},
			Workloads:   []string{"core"},
		},
		{
			Key:         "prd",
			Name:        "Production",
			Description: "Production environment for live applications",
			Projects:    []string{"common", "core"},
			Workloads:   []string{"core"},
		},
	}
}

// GetPredefinedEnvironmentByKey returns a predefined environment by its key
func GetPredefinedEnvironmentByKey(key string) *PredefinedEnvironment {
	environments := GetPredefinedEnvironments()
	for _, env := range environments {
		if env.Key == key {
			return &env
		}
	}
	return nil
}

// GetDefaultProjects returns the default projects for custom environments
func GetDefaultProjects() []string {
	return []string{"common", "core"}
}

// GetDefaultWorkloads returns the default workloads for custom environments
func GetDefaultWorkloads() []string {
	return []string{"core"}
}
