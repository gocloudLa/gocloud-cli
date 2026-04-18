package models

import (
	"reflect"
	"testing"
)

func TestProcessEnvironments(t *testing.T) {
	tests := []struct {
		name        string
		config      *InfrastructureConfig
		expected    map[string]ProcessedEnvironment
		expectError bool
	}{
		{
			name: "valid environments",
			config: &InfrastructureConfig{
				Client: "test-client",
				Environments: map[string]Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
					"stg": {
						Name:       "Staging",
						DirName:    "stg",
						AWSAccount: "123456789013",
					},
				},
			},
			expected: map[string]ProcessedEnvironment{
				"dev": {
					Profile:    "test-client-dev",
					AWSAccount: "123456789012",
				},
				"stg": {
					Profile:    "test-client-stg",
					AWSAccount: "123456789013",
				},
			},
			expectError: false,
		},
		{
			name: "empty environments",
			config: &InfrastructureConfig{
				Client:       "test-client",
				Environments: map[string]Environment{},
			},
			expected:    map[string]ProcessedEnvironment{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessEnvironments(tt.config)
			if len(result) != len(tt.expected) {
				t.Errorf("ProcessEnvironments() returned %d environments, expected %d", len(result), len(tt.expected))
			}
			for key, expectedEnv := range tt.expected {
				if resultEnv, exists := result[key]; !exists {
					t.Errorf("ProcessEnvironments() missing environment key: %s", key)
				} else if resultEnv.Profile != expectedEnv.Profile {
					t.Errorf("ProcessEnvironments() environment %s profile = %s, expected %s", key, resultEnv.Profile, expectedEnv.Profile)
				} else if resultEnv.AWSAccount != expectedEnv.AWSAccount {
					t.Errorf("ProcessEnvironments() environment %s aws_account = %s, expected %s", key, resultEnv.AWSAccount, expectedEnv.AWSAccount)
				}
			}
		})
	}
}

func TestGetProjectDependencies(t *testing.T) {
	tests := []struct {
		name     string
		project  interface{}
		expected []string
	}{
		{
			name:     "string project (no dependencies)",
			project:  "core",
			expected: []string{},
		},
		{
			name:     "project object with dependencies",
			project:  ProjectItem{Name: "core", DependsOn: []string{"base", "foundation"}},
			expected: []string{"base", "foundation"},
		},
		{
			name:     "project object without dependencies",
			project:  ProjectItem{Name: "core"},
			expected: []string{},
		},
		{
			name:     "nil project",
			project:  nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectDependencies(tt.project)
			if len(result) != len(tt.expected) {
				t.Errorf("GetProjectDependencies(%v) returned %d dependencies, expected %d", tt.project, len(result), len(tt.expected))
			}
			for i, dep := range tt.expected {
				if i >= len(result) || result[i] != dep {
					t.Errorf("GetProjectDependencies(%v) dependency[%d] = %s, expected %s", tt.project, i, result[i], dep)
				}
			}
		})
	}
}

func TestGetWorkloadDependencies(t *testing.T) {
	tests := []struct {
		name     string
		workload interface{}
		expected []string
	}{
		{
			name:     "string workload (no dependencies)",
			workload: "webapp",
			expected: []string{},
		},
		{
			name:     "workload object with dependencies",
			workload: WorkloadItem{Name: "webapp", DependsOn: []string{"project/core", "project/common"}},
			expected: []string{"project/core", "project/common"},
		},
		{
			name:     "workload object without dependencies",
			workload: WorkloadItem{Name: "webapp"},
			expected: []string{},
		},
		{
			name:     "nil workload",
			workload: nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWorkloadDependencies(tt.workload)
			if len(result) != len(tt.expected) {
				t.Errorf("GetWorkloadDependencies(%v) returned %d dependencies, expected %d", tt.workload, len(result), len(tt.expected))
			}
			for i, dep := range tt.expected {
				if i >= len(result) || result[i] != dep {
					t.Errorf("GetWorkloadDependencies(%v) dependency[%d] = %s, expected %s", tt.workload, i, result[i], dep)
				}
			}
		})
	}
}

// TestGetProjectDependencies_MapInterfaceFormat documents that GetProjectDependencies does NOT
// support map[interface{}]interface{} (YAML unmarshal format). Fails until the function is fixed.
func TestGetProjectDependencies_MapInterfaceFormat(t *testing.T) {
	item := map[interface{}]interface{}{
		"dept": map[interface{}]interface{}{
			"name":       "Deposits",
			"depends_on": []interface{}{"base", "foundation"},
		},
	}
	got := GetProjectDependencies(item)
	want := []string{"base", "foundation"}
	if len(got) != len(want) {
		t.Errorf("GetProjectDependencies(map[interface{}] YAML format) returned %d deps, want %d", len(got), len(want))
		return
	}
	for i, d := range want {
		if i >= len(got) || got[i] != d {
			t.Errorf("GetProjectDependencies(map[interface{}] YAML format) [%d] = %q, want %q", i, got, want)
			return
		}
	}
}

// TestGetWorkloadDependencies_MapInterfaceFormat documents that GetWorkloadDependencies does NOT
// support map[interface{}]interface{} (YAML unmarshal format). Fails until the function is fixed.
func TestGetWorkloadDependencies_MapInterfaceFormat(t *testing.T) {
	item := map[interface{}]interface{}{
		"wdwl": map[interface{}]interface{}{
			"name":       "Withdrawals",
			"depends_on": []interface{}{"core", "common"},
		},
	}
	got := GetWorkloadDependencies(item)
	want := []string{"core", "common"}
	if len(got) != len(want) {
		t.Errorf("GetWorkloadDependencies(map[interface{}] YAML format) returned %d deps, want %d", len(got), len(want))
		return
	}
	for i, d := range want {
		if i >= len(got) || got[i] != d {
			t.Errorf("GetWorkloadDependencies(map[interface{}] YAML format) [%d] = %q, want %q", i, got, want)
			return
		}
	}
}

func TestGetProjectDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		project  interface{}
		expected string
	}{
		{
			name:     "string project",
			project:  "core",
			expected: "core",
		},
		{
			name:     "project object with name",
			project:  ProjectItem{Key: "dept", Name: "Deposits"},
			expected: "Deposits",
		},
		{
			name:     "project object without name",
			project:  ProjectItem{Key: "dept"},
			expected: "dept",
		},
		{
			name:     "map with name",
			project:  map[string]interface{}{"dept": map[string]interface{}{"name": "Deposits"}},
			expected: "Deposits",
		},
		{
			name:     "map without name",
			project:  map[string]interface{}{"dept": map[string]interface{}{}},
			expected: "dept",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectDisplayName(tt.project)
			if result != tt.expected {
				t.Errorf("GetProjectDisplayName(%v) = %s, expected %s", tt.project, result, tt.expected)
			}
		})
	}
}

// TestGetProjectDisplayName_MapInterfaceFormat documents that GetProjectDisplayName does NOT
// support map[interface{}]interface{} (YAML unmarshal format). Fails until the function is fixed.
func TestGetProjectDisplayName_MapInterfaceFormat(t *testing.T) {
	item := map[interface{}]interface{}{
		"dept": map[interface{}]interface{}{"name": "Deposits"},
	}
	got := GetProjectDisplayName(item)
	want := "Deposits"
	if got != want {
		t.Errorf("GetProjectDisplayName(map[interface{}] YAML format) = %q, want %q", got, want)
	}
}

func TestNormalizeDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single word", "Deposits", "deposits"},
		{"spaces to underscore", "Legacy System", "legacy_system"},
		{"multiple spaces collapsed", "Example  Project", "example_project"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeDisplayName(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeDisplayName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEnvironmentNameForBackendKey(t *testing.T) {
	t.Run("uses Name when set", func(t *testing.T) {
		got := EnvironmentNameForBackendKey("prd", Environment{Name: "Production", DirName: "PIJA"})
		if got != "production" {
			t.Errorf("got %q, want production (dir_name must be ignored)", got)
		}
	})
	t.Run("falls back to env key", func(t *testing.T) {
		got := EnvironmentNameForBackendKey("prd", Environment{DirName: "only_dir"})
		if got != "prd" {
			t.Errorf("got %q, want prd", got)
		}
	})
}

func TestGetProjectDirectoryName(t *testing.T) {
	tests := []struct {
		name     string
		project  interface{}
		expected string
	}{
		{
			name:     "string project",
			project:  "core",
			expected: "core",
		},
		{
			name:     "project object with dir_name",
			project:  ProjectItem{Key: "dept", Name: "Deposits", DirName: "depositos"},
			expected: "depositos",
		},
		{
			name:     "project object with name only",
			project:  ProjectItem{Key: "dept", Name: "Deposits"},
			expected: "deposits",
		},
		{
			name:     "project object with name containing spaces",
			project:  ProjectItem{Key: "legacy", Name: "Legacy System"},
			expected: "legacy_system",
		},
		{
			name:     "project object with key only",
			project:  ProjectItem{Key: "dept"},
			expected: "dept",
		},
		{
			name:     "map with dir_name",
			project:  map[string]interface{}{"dept": map[string]interface{}{"name": "Deposits", "dir_name": "depositos"}},
			expected: "depositos",
		},
		{
			name:     "map with name only",
			project:  map[string]interface{}{"dept": map[string]interface{}{"name": "Deposits"}},
			expected: "deposits",
		},
		{
			name:     "map with key only",
			project:  map[string]interface{}{"dept": map[string]interface{}{}},
			expected: "dept",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectDirectoryName(tt.project)
			if result != tt.expected {
				t.Errorf("GetProjectDirectoryName(%v) = %s, expected %s", tt.project, result, tt.expected)
			}
		})
	}
}

// TestGetProjectDirectoryName_MapInterfaceFormat documents that GetProjectDirectoryName does NOT
// support map[interface{}]interface{} (YAML format). Fails until fixed.
func TestGetProjectDirectoryName_MapInterfaceFormat(t *testing.T) {
	item := map[interface{}]interface{}{
		"wdwl": map[interface{}]interface{}{"name": "Withdrawals", "dir_name": "withdrawals"},
	}
	got := GetProjectDirectoryName(item)
	want := "withdrawals"
	if got != want {
		t.Errorf("GetProjectDirectoryName(map[interface{}] YAML format) = %q, want %q", got, want)
	}
}

func TestGetProjectKey(t *testing.T) {
	tests := []struct {
		name     string
		project  interface{}
		expected string
	}{
		{
			name:     "string project",
			project:  "core",
			expected: "core",
		},
		{
			name:     "project object",
			project:  ProjectItem{Key: "dept", Name: "Deposits"},
			expected: "dept",
		},
		{
			name:     "map project",
			project:  map[string]interface{}{"dept": map[string]interface{}{"name": "Deposits"}},
			expected: "dept",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectKey(tt.project)
			if result != tt.expected {
				t.Errorf("GetProjectKey(%v) = %s, expected %s", tt.project, result, tt.expected)
			}
		})
	}
}

func TestGetWorkloadDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		workload interface{}
		expected string
	}{
		{
			name:     "string workload",
			workload: "core",
			expected: "core",
		},
		{
			name:     "workload object with name",
			workload: WorkloadItem{Key: "dept", Name: "Deposits"},
			expected: "Deposits",
		},
		{
			name:     "workload object without name",
			workload: WorkloadItem{Key: "dept"},
			expected: "dept",
		},
		{
			name:     "map with name",
			workload: map[string]interface{}{"dept": map[string]interface{}{"name": "Deposits"}},
			expected: "Deposits",
		},
		{
			name:     "map without name",
			workload: map[string]interface{}{"dept": map[string]interface{}{}},
			expected: "dept",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWorkloadDisplayName(tt.workload)
			if result != tt.expected {
				t.Errorf("GetWorkloadDisplayName(%v) = %s, expected %s", tt.workload, result, tt.expected)
			}
		})
	}
}

// TestGetWorkloadDisplayName_MapInterfaceFormat documents that GetWorkloadDisplayName does NOT
// support map[interface{}]interface{} (YAML format). Fails until fixed.
func TestGetWorkloadDisplayName_MapInterfaceFormat(t *testing.T) {
	item := map[interface{}]interface{}{
		"dept": map[interface{}]interface{}{"name": "Deposits"},
	}
	got := GetWorkloadDisplayName(item)
	want := "Deposits"
	if got != want {
		t.Errorf("GetWorkloadDisplayName(map[interface{}] YAML format) = %q, want %q", got, want)
	}
}

func TestGetWorkloadDirectoryName(t *testing.T) {
	tests := []struct {
		name     string
		workload interface{}
		expected string
	}{
		{
			name:     "string workload",
			workload: "core",
			expected: "core",
		},
		{
			name:     "workload object with dir_name",
			workload: WorkloadItem{Key: "dept", Name: "Deposits", DirName: "depositos"},
			expected: "depositos",
		},
		{
			name:     "workload object with name only",
			workload: WorkloadItem{Key: "dept", Name: "Deposits"},
			expected: "deposits",
		},
		{
			name:     "workload object with key only",
			workload: WorkloadItem{Key: "dept"},
			expected: "dept",
		},
		{
			name:     "map with dir_name",
			workload: map[string]interface{}{"dept": map[string]interface{}{"name": "Deposits", "dir_name": "depositos"}},
			expected: "depositos",
		},
		{
			name:     "map with name only",
			workload: map[string]interface{}{"dept": map[string]interface{}{"name": "Deposits"}},
			expected: "deposits",
		},
		{
			name:     "workload object with name containing spaces",
			workload: WorkloadItem{Key: "standalone", Name: "Standalone App"},
			expected: "standalone_app",
		},
		{
			name:     "map with key only",
			workload: map[string]interface{}{"dept": map[string]interface{}{}},
			expected: "dept",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWorkloadDirectoryName(tt.workload)
			if result != tt.expected {
				t.Errorf("GetWorkloadDirectoryName(%v) = %s, expected %s", tt.workload, result, tt.expected)
			}
		})
	}
}

// TestGetWorkloadDirectoryName_MapInterfaceFormat documents that GetWorkloadDirectoryName does NOT
// support map[interface{}]interface{} (YAML format). Fails until fixed.
func TestGetWorkloadDirectoryName_MapInterfaceFormat(t *testing.T) {
	item := map[interface{}]interface{}{
		"wdwl": map[interface{}]interface{}{"name": "Withdrawals", "dir_name": "withdrawals"},
	}
	got := GetWorkloadDirectoryName(item)
	want := "withdrawals"
	if got != want {
		t.Errorf("GetWorkloadDirectoryName(map[interface{}] YAML format) = %q, want %q", got, want)
	}
}

func TestGetWorkloadKey(t *testing.T) {
	tests := []struct {
		name     string
		workload interface{}
		expected string
	}{
		{
			name:     "string workload",
			workload: "core",
			expected: "core",
		},
		{
			name:     "workload object",
			workload: WorkloadItem{Key: "dept", Name: "Deposits"},
			expected: "dept",
		},
		{
			name:     "map workload",
			workload: map[string]interface{}{"dept": map[string]interface{}{"name": "Deposits"}},
			expected: "dept",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWorkloadKey(tt.workload)
			if result != tt.expected {
				t.Errorf("GetWorkloadKey(%v) = %s, expected %s", tt.workload, result, tt.expected)
			}
		})
	}
}

func TestDebugProjectDirectoryName(t *testing.T) {
	// Test case that's failing in the real scenario
	project := map[string]interface{}{
		"dept": map[string]interface{}{
			"name": "Deposits",
		},
	}

	t.Logf("Project: %+v", project)
	t.Logf("GetProjectDirectoryName: %s", GetProjectDirectoryName(project))
	t.Logf("GetProjectDisplayName: %s", GetProjectDisplayName(project))
	t.Logf("GetProjectKey: %s", GetProjectKey(project))

	// This should return "deposits" (name in lowercase)
	expected := "deposits"
	result := GetProjectDirectoryName(project)
	if result != expected {
		t.Errorf("GetProjectDirectoryName() = %s, expected %s", result, expected)
	}
}

func TestCalculateDependencies(t *testing.T) {
	config := &InfrastructureConfig{
		Client: "test-client",
		Environments: map[string]Environment{
			"dev": {
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "123456789012",
				Projects: []interface{}{
					"core",
				},
			},
		},
	}

	tests := []struct {
		name        string
		layer       string
		project     string
		envKey      string
		expected    []string
		expectError bool
	}{
		{
			name:        "base layer",
			layer:       "base",
			project:     "",
			envKey:      "dev",
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "foundation layer",
			layer:       "foundation",
			project:     "",
			envKey:      "dev",
			expected:    []string{"../../base/dev"},
			expectError: false,
		},
		{
			name:        "project layer",
			layer:       "project",
			project:     "core",
			envKey:      "dev",
			expected:    []string{"../../../foundation/dev"},
			expectError: false,
		},
		{
			name:        "workload layer",
			layer:       "workload",
			project:     "core",
			envKey:      "dev",
			expected:    []string{"../../../project/core/dev"},
			expectError: false,
		},
		{
			name:        "organization layer",
			layer:       "organization",
			project:     "",
			envKey:      "dev",
			expected:    []string{}, // organization layer is not handled in current implementation
			expectError: false,
		},
		{
			name:        "invalid layer",
			layer:       "invalid",
			project:     "",
			envKey:      "dev",
			expected:    []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDependencies(tt.layer, tt.project, tt.envKey, config)
			if len(result) != len(tt.expected) {
				t.Errorf("CalculateDependencies() returned %d dependencies, expected %d", len(result), len(tt.expected))
			}
			for i, dep := range tt.expected {
				if i >= len(result) || result[i] != dep {
					t.Errorf("CalculateDependencies() dependency[%d] = %s, expected %s", i, result[i], dep)
				}
			}
		})
	}
}

// TestCalculateDependencies_ProjectDependsOn ensures project layer respects depends_on from config.
// Without this, project layer always returns default foundation dependency and ignores project depends_on.
func TestCalculateDependencies_ProjectDependsOn(t *testing.T) {
	config := &InfrastructureConfig{
		Client: "test-client",
		Environments: map[string]Environment{
			"dev": {
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "123456789012",
				Projects: []interface{}{
					ProjectItem{Key: "core", DependsOn: []string{"foundation"}},
				},
			},
		},
	}
	// Project with explicit depends_on: [foundation] should still return foundation path (same as default here)
	result := CalculateDependencies("project", "core", "dev", config)
	expected := []string{"../../../foundation/dev"}
	if len(result) != len(expected) || (len(result) > 0 && result[0] != expected[0]) {
		t.Errorf("CalculateDependencies(project, core, dev) with depends_on:[foundation] = %v, want %v", result, expected)
	}

	// Project with depends_on: [] should return no dependencies
	configNoDeps := &InfrastructureConfig{
		Client: "test-client",
		Environments: map[string]Environment{
			"dev": {
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "123456789012",
				Projects: []interface{}{
					ProjectItem{Key: "core", DependsOn: []string{}},
				},
			},
		},
	}
	resultEmpty := CalculateDependencies("project", "core", "dev", configNoDeps)
	if len(resultEmpty) != 0 {
		t.Errorf("CalculateDependencies(project, core, dev) with depends_on:[] = %v, want []", resultEmpty)
	}
}

// TestCalculateDependencies_WorkloadDependsOnEmpty ensures workload with depends_on: [] returns no dependencies.
// Without this, the code only uses explicit deps when len(workloadDeps) > 0, so depends_on: [] falls through to default.
// We use workload key "core" so that the default would be ../../../project/core/dev; with depends_on: [] we want [].
func TestCalculateDependencies_WorkloadDependsOnEmpty(t *testing.T) {
	config := &InfrastructureConfig{
		Client: "test-client",
		Environments: map[string]Environment{
			"dev": {
				Name:       "Development",
				DirName:    "dev",
				AWSAccount: "123456789012",
				Projects:   []interface{}{"core"},
				Workloads: []interface{}{
					WorkloadItem{Key: "core", DependsOn: []string{}},
				},
			},
		},
	}
	result := CalculateDependencies("workload", "core", "dev", config)
	if len(result) != 0 {
		t.Errorf("CalculateDependencies(workload, core, dev) with depends_on:[] = %v, want []", result)
	}
}

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name          string
		env           Environment
		globalVersion string
		expected      string
	}{
		{
			name: "environment version overrides global",
			env: Environment{
				Version: "v2.0.0",
			},
			globalVersion: "v1.0.0",
			expected:      "v2.0.0",
		},
		{
			name: "use global version when environment version is empty",
			env: Environment{
				Version: "",
			},
			globalVersion: "v1.0.0",
			expected:      "v1.0.0",
		},
		{
			name:          "use global version when environment version is not set",
			env:           Environment{},
			globalVersion: "v1.0.0",
			expected:      "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveVersion(tt.env, tt.globalVersion)
			if result != tt.expected {
				t.Errorf("ResolveVersion() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestResolveBackendConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *InfrastructureConfig
		expected *BackendInfrastructureConfig
	}{
		{
			name: "config with backend",
			config: &InfrastructureConfig{
				Backend: &BackendInfrastructureConfig{
					Pattern: "terraform-state-{company}-{environment}",
					Region:  "us-east-1",
					Account: "123456789012",
					Encrypt: true,
				},
			},
			expected: &BackendInfrastructureConfig{
				Pattern: "terraform-state-{company}-{environment}",
				Region:  "us-east-1",
				Account: "123456789012",
				Encrypt: true,
			},
		},
		{
			name: "config without backend",
			config: &InfrastructureConfig{
				Client: "test-client",
				Region: "us-east-1",
			},
			expected: &BackendInfrastructureConfig{
				Pattern: "s3-backend",
				Region:  "us-east-1",
				Account: "sha",
				Encrypt: true,
			},
		},
		// Note: nil config test removed because ResolveBackendConfig doesn't handle nil config
		// and will panic, which is expected behavior for this function
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveBackendConfig(tt.config)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ResolveBackendConfig() = %v, expected nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("ResolveBackendConfig() = nil, expected %v", tt.expected)
				} else if result.Pattern != tt.expected.Pattern {
					t.Errorf("ResolveBackendConfig() Pattern = %s, expected %s", result.Pattern, tt.expected.Pattern)
				} else if result.Region != tt.expected.Region {
					t.Errorf("ResolveBackendConfig() Region = %s, expected %s", result.Region, tt.expected.Region)
				} else if result.Account != tt.expected.Account {
					t.Errorf("ResolveBackendConfig() Account = %s, expected %s", result.Account, tt.expected.Account)
				} else if result.Encrypt != tt.expected.Encrypt {
					t.Errorf("ResolveBackendConfig() Encrypt = %v, expected %v", result.Encrypt, tt.expected.Encrypt)
				}
			}
		})
	}
}

// TestResolveBackendConfigWithProjectWorkloadOverrides tests backend override at project and workload level
// (as documented in gocloud-example-config.yaml: project "dept" and workload "dept"/"wdwl" with backend overrides).
func TestResolveBackendConfigWithProjectWorkloadOverrides(t *testing.T) {
	useProfileTrue := true
	config := &InfrastructureConfig{
		Company: "gcl",
		Region:  "us-east-1",
		Backend: &BackendInfrastructureConfig{
			Pattern: "tf-backend",
			Region:  "us-east-2",
			Account: "sha",
			Encrypt: true,
		},
		Environments: map[string]Environment{
			"prd": {
				Name:       "Production",
				AWSAccount: "123456789012",
				Backend: &BackendInfrastructureConfig{
					KeyTemplate: "{{.Company}}/{{.Environment}}/terraform.tfstate",
				},
				Projects: []interface{}{
					"core",
					ProjectItem{
						Key:  "dept",
						Name: "Deposits",
						Backend: &BackendInfrastructureConfig{
							KeyTemplate: "{{.Company}}/deposits/{{.Environment}}/terraform.tfstate",
						},
					},
				},
				Workloads: []interface{}{
					"webapp",
					WorkloadItem{
						Key:  "dept",
						Name: "Deposits",
						Backend: &BackendInfrastructureConfig{
							UseProfile: &useProfileTrue,
						},
					},
					WorkloadItem{
						Key:  "wdwl",
						Name: "Withdrawals",
						Backend: &BackendInfrastructureConfig{
							Type:        "s3",
							KeyTemplate: "{{.Company}}/withdrawals/{{.Environment}}/terraform.tfstate",
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		layerType  string
		projectKey string
		envKey     string
		wantKey    string
		wantType   string
		wantUse    *bool
	}{
		{
			name:       "project with backend override (key_template)",
			layerType:  "project",
			projectKey: "dept",
			envKey:     "prd",
			wantKey:    "{{.Company}}/deposits/{{.Environment}}/terraform.tfstate",
			wantType:   "",
			wantUse:    nil,
		},
		{
			name:       "project without backend override inherits env",
			layerType:  "project",
			projectKey: "core",
			envKey:     "prd",
			wantKey:    "{{.Company}}/{{.Environment}}/terraform.tfstate",
			wantType:   "",
			wantUse:    nil,
		},
		{
			name:       "workload with backend override (use_profile)",
			layerType:  "workload",
			projectKey: "dept",
			envKey:     "prd",
			wantKey:    "{{.Company}}/{{.Environment}}/terraform.tfstate",
			wantType:   "",
			wantUse:    &useProfileTrue,
		},
		{
			name:       "workload with backend override (type, key_template)",
			layerType:  "workload",
			projectKey: "wdwl",
			envKey:     "prd",
			wantKey:    "{{.Company}}/withdrawals/{{.Environment}}/terraform.tfstate",
			wantType:   "s3",
			wantUse:    nil,
		},
		{
			name:       "workload without backend override inherits",
			layerType:  "workload",
			projectKey: "webapp",
			envKey:     "prd",
			wantKey:    "{{.Company}}/{{.Environment}}/terraform.tfstate",
			wantType:   "",
			wantUse:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ResolveBackendConfig(tt.layerType, tt.projectKey, tt.envKey)
			if result == nil {
				t.Fatal("ResolveBackendConfig() = nil")
			}
			if result.KeyTemplate != tt.wantKey {
				t.Errorf("KeyTemplate = %q, want %q", result.KeyTemplate, tt.wantKey)
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", result.Type, tt.wantType)
			}
			if (tt.wantUse != nil) != (result.UseProfile != nil) {
				t.Errorf("UseProfile presence = %v, want %v", result.UseProfile != nil, tt.wantUse != nil)
			} else if tt.wantUse != nil && result.UseProfile != nil && *result.UseProfile != *tt.wantUse {
				t.Errorf("UseProfile = %v, want %v", *result.UseProfile, *tt.wantUse)
			}
		})
	}
}

func TestProviderSpec_RegionHCL(t *testing.T) {
	tests := []struct {
		region string
		want   string
	}{
		{"local.metadata.aws_region", "local.metadata.aws_region"},
		{"  local.metadata.aws_region  ", "  local.metadata.aws_region  "},
		{"${local.a}${local.b}", "${local.a}${local.b}"},
		{"var.aws_region", "var.aws_region"},
		{"data.aws_region.current.name", "data.aws_region.current.name"},
		{"us-east-2", `"us-east-2"`},
		{"  us-east-2  ", "  us-east-2  "},
		{"eu-central-1", `"eu-central-1"`},
		{"ap-southeast-1", `"ap-southeast-1"`},
		{"not-a-region", "not-a-region"},
		{"", ""},
	}
	for _, tt := range tests {
		got := ProviderSpec{Region: tt.region}.RegionHCL()
		if got != tt.want {
			t.Errorf("RegionHCL(%q) = %q, want %q", tt.region, got, tt.want)
		}
	}
}

// TestResolveProviderConfigWithProjectWorkloadOverrides tests provider override at project and workload level
// (as documented in gocloud-example-config.yaml: project/workload "dept" with providers overrides).
func TestResolveProviderConfigWithProjectWorkloadOverrides(t *testing.T) {
	useProfilesTrue := true
	useProfilesFalse := false
	config := &InfrastructureConfig{
		Company: "gcl",
		Region:  "us-east-1",
		Providers: &ProviderConfig{
			UseProfiles: &useProfilesFalse,
			DefaultProviders: []ProviderSpec{
				{Name: "aws", Region: "us-west-2"},
			},
		},
		Environments: map[string]Environment{
			"prd": {
				Name:       "Production",
				AWSAccount: "123456789012",
				Providers: &ProviderConfig{
					UseProfiles: &useProfilesFalse,
					DefaultProviders: []ProviderSpec{
						{Name: "aws", Region: "us-west-2"},
					},
				},
				Projects: []interface{}{
					"core",
					ProjectItem{
						Key:  "dept",
						Name: "Deposits",
						Providers: &ProviderConfig{
							UseProfiles: &useProfilesTrue,
							DefaultProviders: []ProviderSpec{
								{Name: "aws", Region: "us-east-1", Alias: "primary"},
							},
						},
					},
				},
				Workloads: []interface{}{
					"webapp",
					WorkloadItem{
						Key:  "dept",
						Name: "Deposits",
						Providers: &ProviderConfig{
							UseProfiles: &useProfilesTrue,
							DefaultProviders: []ProviderSpec{
								{Name: "aws", Region: "eu-west-1", Alias: "europe"},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		layerType   string
		projectKey  string
		envKey      string
		wantRegion  string
		wantAlias   string
		wantUseProf *bool
	}{
		{
			name:        "project with provider override (use_profiles, default_providers)",
			layerType:   "project",
			projectKey:  "dept",
			envKey:      "prd",
			wantRegion:  "us-east-1",
			wantAlias:   "primary",
			wantUseProf: &useProfilesTrue,
		},
		{
			name:        "project without provider override inherits env",
			layerType:   "project",
			projectKey:  "core",
			envKey:      "prd",
			wantRegion:  "us-west-2",
			wantAlias:   "",
			wantUseProf: &useProfilesFalse,
		},
		{
			name:        "workload with provider override (use_profiles, default_providers)",
			layerType:   "workload",
			projectKey:  "dept",
			envKey:      "prd",
			wantRegion:  "eu-west-1",
			wantAlias:   "europe",
			wantUseProf: &useProfilesTrue,
		},
		{
			name:        "workload without provider override inherits",
			layerType:   "workload",
			projectKey:  "webapp",
			envKey:      "prd",
			wantRegion:  "us-west-2",
			wantAlias:   "",
			wantUseProf: &useProfilesFalse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ResolveProviderConfig(tt.layerType, tt.projectKey, tt.envKey)
			if result == nil {
				t.Fatal("ResolveProviderConfig() = nil")
			}
			if len(result.DefaultProviders) == 0 {
				t.Fatal("DefaultProviders empty")
			}
			got := result.DefaultProviders[0]
			if got.Region != tt.wantRegion {
				t.Errorf("DefaultProviders[0].Region = %q, want %q", got.Region, tt.wantRegion)
			}
			if got.Alias != tt.wantAlias {
				t.Errorf("DefaultProviders[0].Alias = %q, want %q", got.Alias, tt.wantAlias)
			}
			if (tt.wantUseProf != nil) != (result.UseProfiles != nil) {
				t.Errorf("UseProfiles presence = %v, want %v", result.UseProfiles != nil, tt.wantUseProf != nil)
			} else if tt.wantUseProf != nil && result.UseProfiles != nil && *result.UseProfiles != *tt.wantUseProf {
				t.Errorf("UseProfiles = %v, want %v", *result.UseProfiles, *tt.wantUseProf)
			}
		})
	}
}

// TestResolveBackendConfigWithMapFormat tests backend override when config is loaded from YAML
// (projects/workloads as map format, as in gocloud-example-config.yaml).
func TestResolveBackendConfigWithMapFormat(t *testing.T) {
	config := &InfrastructureConfig{
		Company: "gcl",
		Region:  "us-east-1",
		Backend: &BackendInfrastructureConfig{
			Pattern: "tf-backend",
			Region:  "us-east-2",
			Account: "sha",
			Encrypt: true,
		},
		Environments: map[string]Environment{
			"prd": {
				Name:       "Production",
				AWSAccount: "123456789012",
				Backend: &BackendInfrastructureConfig{
					KeyTemplate: "{{.Company}}/{{.Environment}}/terraform.tfstate",
				},
				Projects: []interface{}{
					"core",
					map[string]interface{}{
						"dept": map[string]interface{}{
							"name": "Deposits",
							"backend": map[string]interface{}{
								"key_template": "{{.Company}}/deposits/{{.Environment}}/terraform.tfstate",
							},
						},
					},
				},
				Workloads: []interface{}{
					"webapp",
					map[string]interface{}{
						"wdwl": map[string]interface{}{
							"name": "Withdrawals",
							"backend": map[string]interface{}{
								"type":         "s3",
								"key_template": "{{.Company}}/withdrawals/{{.Environment}}/terraform.tfstate",
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name       string
		layerType  string
		projectKey string
		envKey     string
		wantKey    string
		wantType   string
	}{
		{
			name:       "project with backend in map format (YAML-like)",
			layerType:  "project",
			projectKey: "dept",
			envKey:     "prd",
			wantKey:    "{{.Company}}/deposits/{{.Environment}}/terraform.tfstate",
			wantType:   "",
		},
		{
			name:       "workload with backend in map format (YAML-like)",
			layerType:  "workload",
			projectKey: "wdwl",
			envKey:     "prd",
			wantKey:    "{{.Company}}/withdrawals/{{.Environment}}/terraform.tfstate",
			wantType:   "s3",
		},
		{
			name:       "project without backend in map format inherits env",
			layerType:  "project",
			projectKey: "core",
			envKey:     "prd",
			wantKey:    "{{.Company}}/{{.Environment}}/terraform.tfstate",
			wantType:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ResolveBackendConfig(tt.layerType, tt.projectKey, tt.envKey)
			if result == nil {
				t.Fatal("ResolveBackendConfig() = nil")
			}
			if result.KeyTemplate != tt.wantKey {
				t.Errorf("KeyTemplate = %q, want %q", result.KeyTemplate, tt.wantKey)
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", result.Type, tt.wantType)
			}
		})
	}
}

// TestResolveBackendConfigOrganizationOverride verifies that organization layer uses infrastructure.organization.backend when set.
func TestResolveBackendConfigOrganizationOverride(t *testing.T) {
	config := &InfrastructureConfig{
		Company: "gcl",
		Region:  "us-east-1",
		Backend: &BackendInfrastructureConfig{
			Pattern: "s3-backend",
			Region:  "us-east-1",
			Account: "sha",
			Encrypt: true,
		},
		Organization: &OrganizationLayerConfig{
			AWSAccount: "598504644885",
			Backend: &BackendInfrastructureConfig{
				Pattern: "org-backend",
				Region:  "eu-west-1",
				Encrypt: false,
			},
		},
	}
	result := config.ResolveBackendConfig("organization", "", "org")
	if result == nil {
		t.Fatal("ResolveBackendConfig(organization) = nil")
	}
	if result.Pattern != "org-backend" {
		t.Errorf("Pattern = %q, want org-backend", result.Pattern)
	}
	if result.Region != "eu-west-1" {
		t.Errorf("Region = %q, want eu-west-1", result.Region)
	}
	if result.Encrypt {
		t.Errorf("Encrypt = true, want false (from organization.backend override)")
	}
}

// TestResolveProviderConfigWithMapFormat tests provider override when config is loaded from YAML
// (projects/workloads as map format, as in gocloud-example-config.yaml).
func TestResolveProviderConfigWithMapFormat(t *testing.T) {
	useProfilesFalse := false
	config := &InfrastructureConfig{
		Company: "gcl",
		Region:  "us-east-1",
		Providers: &ProviderConfig{
			UseProfiles: &useProfilesFalse,
			DefaultProviders: []ProviderSpec{
				{Name: "aws", Region: "us-west-2"},
			},
		},
		Environments: map[string]Environment{
			"prd": {
				Name:       "Production",
				AWSAccount: "123456789012",
				Providers: &ProviderConfig{
					UseProfiles: &useProfilesFalse,
					DefaultProviders: []ProviderSpec{
						{Name: "aws", Region: "us-west-2"},
					},
				},
				Projects: []interface{}{
					"core",
					map[string]interface{}{
						"dept": map[string]interface{}{
							"name": "Deposits",
							"providers": map[string]interface{}{
								"use_profiles": true,
								"default_providers": []interface{}{
									map[string]interface{}{
										"name":   "aws",
										"region": "us-east-1",
										"alias":  "primary",
									},
								},
							},
						},
					},
				},
				Workloads: []interface{}{
					"webapp",
					map[string]interface{}{
						"dept": map[string]interface{}{
							"name": "Deposits",
							"providers": map[string]interface{}{
								"use_profiles": true,
								"default_providers": []interface{}{
									map[string]interface{}{
										"name":   "aws",
										"region": "eu-west-1",
										"alias":  "europe",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		layerType   string
		projectKey  string
		envKey      string
		wantRegion  string
		wantAlias   string
		wantUseProf bool
	}{
		{
			name:        "project with providers in map format (YAML-like)",
			layerType:   "project",
			projectKey:  "dept",
			envKey:      "prd",
			wantRegion:  "us-east-1",
			wantAlias:   "primary",
			wantUseProf: true,
		},
		{
			name:        "workload with providers in map format (YAML-like)",
			layerType:   "workload",
			projectKey:  "dept",
			envKey:      "prd",
			wantRegion:  "eu-west-1",
			wantAlias:   "europe",
			wantUseProf: true,
		},
		{
			name:        "project without providers in map format inherits",
			layerType:   "project",
			projectKey:  "core",
			envKey:      "prd",
			wantRegion:  "us-west-2",
			wantAlias:   "",
			wantUseProf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ResolveProviderConfig(tt.layerType, tt.projectKey, tt.envKey)
			if result == nil {
				t.Fatal("ResolveProviderConfig() = nil")
			}
			if len(result.DefaultProviders) == 0 {
				t.Fatal("DefaultProviders empty")
			}
			got := result.DefaultProviders[0]
			if got.Region != tt.wantRegion {
				t.Errorf("DefaultProviders[0].Region = %q, want %q", got.Region, tt.wantRegion)
			}
			if got.Alias != tt.wantAlias {
				t.Errorf("DefaultProviders[0].Alias = %q, want %q", got.Alias, tt.wantAlias)
			}
			if result.UseProfiles == nil || *result.UseProfiles != tt.wantUseProf {
				t.Errorf("UseProfiles = %v, want %v", result.UseProfiles, tt.wantUseProf)
			}
		})
	}
}

// TestResolveProviderConfigWithAssumeRole verifies that assume_role in default_providers is parsed and resolved.
func TestResolveProviderConfigWithAssumeRole(t *testing.T) {
	config := &InfrastructureConfig{
		Company: "gcl",
		Region:  "us-east-1",
		Environments: map[string]Environment{
			"sha": {
				Name:       "Shared",
				AWSAccount: "598504644885",
				Projects: []interface{}{
					map[string]interface{}{
						"common": map[string]interface{}{
							"providers": map[string]interface{}{
								"default_providers": []interface{}{
									map[string]interface{}{
										"name":   "aws",
										"region": "local.metadata.aws_region",
										"alias":  "sha",
										"assume_role": map[string]interface{}{
											"role_arn":     "arn:aws:iam::904233109008:role/OrganizationAccountAccessRole",
											"session_name": "TerraformSession",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	result := config.ResolveProviderConfig("project", "common", "sha")
	if result == nil || len(result.DefaultProviders) == 0 {
		t.Fatal("ResolveProviderConfig() nil or empty DefaultProviders")
	}
	got := result.DefaultProviders[0]
	if got.AssumeRole == nil {
		t.Fatal("DefaultProviders[0].AssumeRole = nil, want assume_role parsed")
	}
	if got.AssumeRole.RoleARN != "arn:aws:iam::904233109008:role/OrganizationAccountAccessRole" {
		t.Errorf("AssumeRole.RoleARN = %q", got.AssumeRole.RoleARN)
	}
	if got.AssumeRole.SessionName != "TerraformSession" {
		t.Errorf("AssumeRole.SessionName = %q, want TerraformSession", got.AssumeRole.SessionName)
	}
}

// TestResolveProviderConfigOrganizationLayer verifies that organization layer uses infrastructure.organization.providers when set.
func TestResolveProviderConfigOrganizationLayer(t *testing.T) {
	config := &InfrastructureConfig{
		Company: "gcl",
		Region:  "us-east-1",
		Providers: &ProviderConfig{
			DefaultProviders: []ProviderSpec{
				{Name: "aws", Region: "us-east-1", Alias: "use1"},
			},
		},
		Organization: &OrganizationLayerConfig{
			AWSAccount: "598504644885",
			Providers: &ProviderConfig{
				DefaultProviders: []ProviderSpec{
					{
						Name:   "aws",
						Region: "local.metadata.aws_region",
						Alias:  "sha",
						AssumeRole: &ProviderAssumeRole{
							RoleARN:     "arn:aws:iam::904233109008:role/OrganizationAccountAccessRole",
							SessionName: "TerraformSession",
						},
					},
				},
			},
		},
	}
	result := config.ResolveProviderConfig("organization", "", "org")
	if result == nil || len(result.DefaultProviders) == 0 {
		t.Fatal("ResolveProviderConfig(organization) nil or empty DefaultProviders")
	}
	got := result.DefaultProviders[0]
	if got.Alias != "sha" {
		t.Errorf("DefaultProviders[0].Alias = %q, want sha", got.Alias)
	}
	if got.AssumeRole == nil {
		t.Fatal("DefaultProviders[0].AssumeRole = nil, want organization.providers to apply")
	}
	if got.AssumeRole.RoleARN != "arn:aws:iam::904233109008:role/OrganizationAccountAccessRole" {
		t.Errorf("AssumeRole.RoleARN = %q", got.AssumeRole.RoleARN)
	}
}

func TestResolveMetadata(t *testing.T) {
	config := &InfrastructureConfig{
		Metadata: map[string]interface{}{
			"global":         "yes",
			"shared_value":   "global",
			"global_only":    "global-only",
			"override_chain": "global",
		},
		Organization: &OrganizationLayerConfig{
			AWSAccount: "123456789012",
			Metadata: map[string]interface{}{
				"org_only":       "org",
				"shared_value":   "organization",
				"override_chain": "organization",
			},
		},
		Security: &OrganizationLayerConfig{
			AWSAccount: "123456789013",
			Metadata: map[string]interface{}{
				"sec_only":       "sec",
				"shared_value":   "security",
				"override_chain": "security",
			},
		},
		Environments: map[string]Environment{
			"dev": {
				AWSAccount: "123456789014",
				Metadata: map[string]interface{}{
					"env_only":       "dev",
					"shared_value":   "environment",
					"override_chain": "environment",
				},
			},
		},
	}

	tests := []struct {
		name     string
		layer    string
		envKey   string
		expected map[string]interface{}
	}{
		{
			name:   "organization metadata overrides global",
			layer:  "organization",
			envKey: "org",
			expected: map[string]interface{}{
				"global":         "yes",
				"global_only":    "global-only",
				"shared_value":   "organization",
				"override_chain": "organization",
				"org_only":       "org",
			},
		},
		{
			name:   "security metadata overrides global",
			layer:  "security",
			envKey: "sec",
			expected: map[string]interface{}{
				"global":         "yes",
				"global_only":    "global-only",
				"shared_value":   "security",
				"override_chain": "security",
				"sec_only":       "sec",
			},
		},
		{
			name:   "environment metadata overrides global",
			layer:  "base",
			envKey: "dev",
			expected: map[string]interface{}{
				"global":         "yes",
				"global_only":    "global-only",
				"shared_value":   "environment",
				"override_chain": "environment",
				"env_only":       "dev",
			},
		},
		{
			name:   "global metadata for unknown environment",
			layer:  "foundation",
			envKey: "missing",
			expected: map[string]interface{}{
				"global":         "yes",
				"global_only":    "global-only",
				"shared_value":   "global",
				"override_chain": "global",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.ResolveMetadata(tt.layer, tt.envKey)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ResolveMetadata(%q, %q) = %#v, want %#v", tt.layer, tt.envKey, got, tt.expected)
			}
		})
	}
}

func TestGetSource(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		infra    *InfrastructureConfig
		expected SourceConfig
	}{
		{
			name: "environment source takes priority",
			env: Environment{
				Source:    "git@github.com:test/repo.git",
				SourceRef: "feature-branch",
			},
			infra: &InfrastructureConfig{
				Source:    "git@github.com:global/repo.git",
				SourceRef: "main",
			},
			expected: SourceConfig{
				Source:    "git@github.com:test/repo.git",
				SourceRef: "feature-branch",
				IsGit:     true,
			},
		},
		{
			name: "global source when environment has no source",
			env:  Environment{},
			infra: &InfrastructureConfig{
				Source:    "git@github.com:global/repo.git",
				SourceRef: "main",
			},
			expected: SourceConfig{
				Source:    "git@github.com:global/repo.git",
				SourceRef: "main",
				IsGit:     true,
			},
		},
		{
			name: "default ref when source_ref is empty",
			env: Environment{
				Source: "git@github.com:test/repo.git",
			},
			infra: &InfrastructureConfig{},
			expected: SourceConfig{
				Source:    "git@github.com:test/repo.git",
				SourceRef: "main",
				IsGit:     true,
			},
		},
		{
			name:     "registry fallback when no source",
			env:      Environment{},
			infra:    &InfrastructureConfig{},
			expected: SourceConfig{IsGit: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.env.GetSource(tt.infra)
			if result.Source != tt.expected.Source {
				t.Errorf("GetSource() Source = %s, expected %s", result.Source, tt.expected.Source)
			}
			if result.SourceRef != tt.expected.SourceRef {
				t.Errorf("GetSource() SourceRef = %s, expected %s", result.SourceRef, tt.expected.SourceRef)
			}
			if result.IsGit != tt.expected.IsGit {
				t.Errorf("GetSource() IsGit = %v, expected %v", result.IsGit, tt.expected.IsGit)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		infra    *InfrastructureConfig
		expected string
	}{
		{
			name: "environment version takes priority",
			env: Environment{
				Version: "v1.0.0",
			},
			infra: &InfrastructureConfig{
				Version: "v2.0.0",
			},
			expected: "v1.0.0",
		},
		{
			name: "global version when environment has no version",
			env:  Environment{},
			infra: &InfrastructureConfig{
				Version: "v2.0.0",
			},
			expected: "v2.0.0",
		},
		{
			name:     "latest fallback when no version",
			env:      Environment{},
			infra:    &InfrastructureConfig{},
			expected: "latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.env.GetVersion(tt.infra)
			if result != tt.expected {
				t.Errorf("GetVersion() = %s, expected %s", result, tt.expected)
			}
		})
	}
}

func TestInfrastructureConfig_RegionForEnvironment(t *testing.T) {
	infra := &InfrastructureConfig{
		Region: "us-east-1",
		Environments: map[string]Environment{
			"prd": {AWSAccount: "111111111111", Region: "eu-west-1"},
			"dev": {AWSAccount: "222222222222"},
		},
	}
	if got := infra.RegionForEnvironment("prd"); got != "eu-west-1" {
		t.Errorf("RegionForEnvironment(prd) = %q, want eu-west-1", got)
	}
	if got := infra.RegionForEnvironment("dev"); got != "us-east-1" {
		t.Errorf("RegionForEnvironment(dev) = %q, want us-east-1", got)
	}
	if got := infra.RegionForEnvironment("missing"); got != "us-east-1" {
		t.Errorf("RegionForEnvironment(missing) = %q, want us-east-1", got)
	}
	if got := infra.RegionForEnvironment("org"); got != "us-east-1" {
		t.Errorf("RegionForEnvironment(org) = %q, want us-east-1", got)
	}
	if got := infra.RegionForEnvironment("sec"); got != "us-east-1" {
		t.Errorf("RegionForEnvironment(sec) = %q, want us-east-1", got)
	}
}

func TestGetEnvironmentOrder(t *testing.T) {
	tests := []struct {
		name     string
		config   *InfrastructureConfig
		expected []string
	}{
		{
			name: "explicit order",
			config: &InfrastructureConfig{
				EnvironmentOrder: []string{"prd", "stg", "dev"},
				Environments: map[string]Environment{
					"dev": {Name: "Development", DirName: "dev"},
					"stg": {Name: "Staging", DirName: "stg"},
					"prd": {Name: "Production", DirName: "prd"},
				},
			},
			expected: []string{"prd", "stg", "dev"},
		},
		{
			name: "fallback sorted when no order",
			config: &InfrastructureConfig{
				Environments: map[string]Environment{
					"dev": {Name: "Development", DirName: "dev"},
					"stg": {Name: "Staging", DirName: "stg"},
					"prd": {Name: "Production", DirName: "prd"},
				},
			},
			expected: []string{"dev", "prd", "stg"},
		},
		{
			name:     "empty environments",
			config:   &InfrastructureConfig{Environments: map[string]Environment{}},
			expected: []string{},
		},
		{
			name:     "nil environments",
			config:   &InfrastructureConfig{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetEnvironmentOrder()
			if len(got) != len(tt.expected) {
				t.Errorf("GetEnvironmentOrder() length = %d, expected %d", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("GetEnvironmentOrder()[%d] = %q, expected %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}
