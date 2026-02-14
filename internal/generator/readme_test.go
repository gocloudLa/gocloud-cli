package generator

import (
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/testutils"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateREADME(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.InfrastructureConfig
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "generate README with valid config",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
				},
			},
			workingDir:  "test-output",
			expectError: false,
		},
		{
			name:        "generate README with nil config",
			config:      nil,
			workingDir:  "test-output",
			expectError: true,
			errorMsg:    "config is required",
		},
		{
			name: "generate README with empty working dir",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
				},
			},
			workingDir:  "",
			expectError: true,
			errorMsg:    "working directory is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory if needed
			var workingDir string
			if tt.workingDir != "" {
				workingDir = filepath.Join(tempDir, tt.workingDir)
				err = os.MkdirAll(workingDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create working dir: %v", err)
				}
			}

			pg := NewProjectGenerator(tt.config, workingDir, false)

			err = pg.GenerateREADME()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateREADME() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateREADME() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateREADME() expected no error but got: %v", err)
				}

				// Verify README file was created
				readmePath := filepath.Join(workingDir, "README.md")
				if _, err := os.Stat(readmePath); os.IsNotExist(err) {
					t.Errorf("GenerateREADME() did not create README.md file")
				}
			}
		})
	}
}

func TestGenerateRootConfigs(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.InfrastructureConfig
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "generate root configs with valid config",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
				},
			},
			workingDir:  "test-output",
			expectError: false,
		},
		{
			name:        "generate root configs with nil config",
			config:      nil,
			workingDir:  "test-output",
			expectError: true,
			errorMsg:    "config is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory
			workingDir := filepath.Join(tempDir, tt.workingDir)
			err = os.MkdirAll(workingDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create working dir: %v", err)
			}

			pg := NewProjectGenerator(tt.config, workingDir, false)

			err = pg.GenerateRootConfigs()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateRootConfigs() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateRootConfigs() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateRootConfigs() expected no error but got: %v", err)
				}

				// Verify root.hcl was created
				rootPath := filepath.Join(workingDir, "root.hcl")
				if _, err := os.Stat(rootPath); os.IsNotExist(err) {
					t.Errorf("GenerateRootConfigs() did not create root.hcl file")
				}
			}
		})
	}
}

func TestGenerateLayerConfigs(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.InfrastructureConfig
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "generate layer configs with valid config",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
				},
			},
			workingDir:  "test-output",
			expectError: false,
		},
		{
			name:        "generate layer configs with nil config",
			config:      nil,
			workingDir:  "test-output",
			expectError: true,
			errorMsg:    "config is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory
			workingDir := filepath.Join(tempDir, tt.workingDir)
			err = os.MkdirAll(workingDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create working dir: %v", err)
			}

			pg := NewProjectGenerator(tt.config, workingDir, false)

			err = pg.GenerateLayerConfigs()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateLayerConfigs() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateLayerConfigs() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateLayerConfigs() expected no error but got: %v", err)
				}

				// Verify layer directories and files were created
				layers := []string{"base", "foundation"}
				for _, layer := range layers {
					layerDir := filepath.Join(workingDir, layer)
					if _, err := os.Stat(layerDir); os.IsNotExist(err) {
						t.Errorf("GenerateLayerConfigs() did not create %s directory", layer)
					}

					// Check for environment directories
					envDir := filepath.Join(layerDir, "dev")
					if _, err := os.Stat(envDir); os.IsNotExist(err) {
						t.Errorf("GenerateLayerConfigs() did not create %s/dev directory", layer)
					}
				}
			}
		})
	}
}

func TestGenerateProjectFiles(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.InfrastructureConfig
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "generate project files with valid config",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
						Projects:   []interface{}{"core"},
					},
				},
			},
			workingDir:  "test-output",
			expectError: false,
		},
		{
			name:        "generate project files with nil config",
			config:      nil,
			workingDir:  "test-output",
			expectError: true,
			errorMsg:    "config is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory
			workingDir := filepath.Join(tempDir, tt.workingDir)
			err = os.MkdirAll(workingDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create working dir: %v", err)
			}

			pg := NewProjectGenerator(tt.config, workingDir, false)

			err = pg.GenerateProjectFiles()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateProjectFiles() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateProjectFiles() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateProjectFiles() expected no error but got: %v", err)
				}

				// Verify project directories and files were created
				projectDir := filepath.Join(workingDir, "project", "core")
				if _, err := os.Stat(projectDir); os.IsNotExist(err) {
					t.Errorf("GenerateProjectFiles() did not create project/core directory")
				}

				// Check for environment directories
				envDir := filepath.Join(projectDir, "dev")
				if _, err := os.Stat(envDir); os.IsNotExist(err) {
					t.Errorf("GenerateProjectFiles() did not create project/core/dev directory")
				}
			}
		})
	}
}

func TestGenerateWorkloadFiles(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.InfrastructureConfig
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "generate workload files with valid config",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
						Projects:   []interface{}{"core"},
						Workloads:  []interface{}{"api"},
					},
				},
			},
			workingDir:  "test-output",
			expectError: false,
		},
		{
			name:        "generate workload files with nil config",
			config:      nil,
			workingDir:  "test-output",
			expectError: true,
			errorMsg:    "config is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory
			workingDir := filepath.Join(tempDir, tt.workingDir)
			err = os.MkdirAll(workingDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create working dir: %v", err)
			}

			pg := NewProjectGenerator(tt.config, workingDir, false)

			err = pg.GenerateWorkloadFiles()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateWorkloadFiles() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateWorkloadFiles() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateWorkloadFiles() expected no error but got: %v", err)
				}

				// Verify workload directories and files were created
				workloadDir := filepath.Join(workingDir, "workload", "api")
				if _, err := os.Stat(workloadDir); os.IsNotExist(err) {
					t.Errorf("GenerateWorkloadFiles() did not create workload/api directory")
				}

				// Check for environment directories
				envDir := filepath.Join(workloadDir, "dev")
				if _, err := os.Stat(envDir); os.IsNotExist(err) {
					t.Errorf("GenerateWorkloadFiles() did not create workload/api/dev directory")
				}
			}
		})
	}
}

func TestGenerateOrganizationFiles(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.InfrastructureConfig
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "generate organization files with valid config",
			config: &models.InfrastructureConfig{
				Client:       "test-client",
				Company:      "gcl",
				Region:       "us-east-1",
				Version:      "v1.0.0",
				Organization: &models.OrganizationLayerConfig{AWSAccount: "123456789012"},
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
				},
			},
			workingDir:  "test-output",
			expectError: false,
		},
		{
			name:        "generate organization files with nil config",
			config:      nil,
			workingDir:  "test-output",
			expectError: true,
			errorMsg:    "config is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory
			workingDir := filepath.Join(tempDir, tt.workingDir)
			err = os.MkdirAll(workingDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create working dir: %v", err)
			}

			pg := NewProjectGenerator(tt.config, workingDir, false)

			err = pg.GenerateOrganizationFiles()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateOrganizationFiles() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateOrganizationFiles() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateOrganizationFiles() expected no error but got: %v", err)
				}

				// Verify organization directory and main.tf were created
				orgDir := filepath.Join(workingDir, "organization")
				if _, err := os.Stat(orgDir); os.IsNotExist(err) {
					t.Errorf("GenerateOrganizationFiles() did not create organization directory")
				}

				// Check for main.tf file (single file, no environment subdirectories)
				mainTfPath := filepath.Join(orgDir, "main.tf")
				if _, err := os.Stat(mainTfPath); os.IsNotExist(err) {
					t.Errorf("GenerateOrganizationFiles() did not create organization/main.tf")
				}

				// Check for metadata.tf file
				metadataTfPath := filepath.Join(orgDir, "metadata.tf")
				if _, err := os.Stat(metadataTfPath); os.IsNotExist(err) {
					t.Errorf("GenerateOrganizationFiles() did not create organization/metadata.tf")
				}

				// Verify metadata.tf content for organization
				metadataContent, err := os.ReadFile(metadataTfPath)
				if err != nil {
					t.Errorf("GenerateOrganizationFiles() failed to read organization/metadata.tf: %v", err)
				} else {
					metadataStr := string(metadataContent)
					if !strings.Contains(metadataStr, `env     = "org"`) {
						t.Errorf("GenerateOrganizationFiles() metadata.tf should contain env = \"org\"")
					}
					if !strings.Contains(metadataStr, `environment = "Organization"`) {
						t.Errorf("GenerateOrganizationFiles() metadata.tf should contain environment = \"Organization\"")
					}
					if !strings.Contains(metadataStr, `layer   = "organization"`) {
						t.Errorf("GenerateOrganizationFiles() metadata.tf should contain layer = \"organization\"")
					}
				}
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.InfrastructureConfig
		workingDir  string
		force       bool
		expectError bool
		errorMsg    string
	}{
		{
			name: "generate with valid config",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
				},
			},
			workingDir:  "test-output",
			force:       false,
			expectError: false,
		},
		{
			name: "generate with force flag",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
				},
			},
			workingDir:  "test-output",
			force:       true,
			expectError: false,
		},
		{
			name:        "generate with nil config",
			config:      nil,
			workingDir:  "test-output",
			force:       false,
			expectError: true,
			errorMsg:    "config is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory
			workingDir := filepath.Join(tempDir, tt.workingDir)
			err = os.MkdirAll(workingDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create working dir: %v", err)
			}

			pg := NewProjectGenerator(tt.config, workingDir, tt.force)

			err = pg.Generate()

			if tt.expectError {
				if err == nil {
					t.Errorf("Generate() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Generate() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Generate() expected no error but got: %v", err)
				}

				// Verify that the generation process completed successfully
				// by checking for key files and directories
				readmePath := filepath.Join(workingDir, "README.md")
				if _, err := os.Stat(readmePath); os.IsNotExist(err) {
					t.Errorf("Generate() did not create README.md file")
				}
			}
		})
	}
}

func TestInitializeGit(t *testing.T) {
	tests := []struct {
		name        string
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "initialize git repository",
			workingDir:  "test-output",
			expectError: false,
		},
		{
			name:        "initialize git with empty working dir",
			workingDir:  "",
			expectError: true,
			errorMsg:    "working directory is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory if needed
			var workingDir string
			if tt.workingDir != "" {
				workingDir = filepath.Join(tempDir, tt.workingDir)
				err = os.MkdirAll(workingDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create working dir: %v", err)
				}
			}

			pg := NewProjectGenerator(nil, workingDir, false)

			err = pg.InitializeGit()

			if tt.expectError {
				if err == nil {
					t.Errorf("InitializeGit() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("InitializeGit() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("InitializeGit() expected no error but got: %v", err)
				}

				// Verify .git directory was created
				gitDir := filepath.Join(workingDir, ".git")
				if _, err := os.Stat(gitDir); os.IsNotExist(err) {
					t.Errorf("InitializeGit() did not create .git directory")
				}
			}
		})
	}
}

func TestGenerateDocumentation(t *testing.T) {
	tests := []struct {
		name        string
		config      *models.InfrastructureConfig
		workingDir  string
		expectError bool
		errorMsg    string
	}{
		{
			name: "generate documentation with valid config",
			config: &models.InfrastructureConfig{
				Client:  "test-client",
				Company: "gcl",
				Region:  "us-east-1",
				Version: "v1.0.0",
				Environments: map[string]models.Environment{
					"dev": {
						Name:       "Development",
						DirName:    "dev",
						AWSAccount: "123456789012",
					},
				},
			},
			workingDir:  "test-output",
			expectError: false,
		},
		{
			name:        "generate documentation with nil config",
			config:      nil,
			workingDir:  "test-output",
			expectError: true,
			errorMsg:    "config is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := testutils.CreateTempDir()
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer func() {
				if err := testutils.CleanupTempDir(tempDir); err != nil {
					t.Logf("Warning: failed to cleanup temp dir: %v", err)
				}
			}()

			// Create working directory
			workingDir := filepath.Join(tempDir, tt.workingDir)
			err = os.MkdirAll(workingDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create working dir: %v", err)
			}

			pg := NewProjectGenerator(tt.config, workingDir, false)

			err = pg.GenerateDocumentation()

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateDocumentation() expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GenerateDocumentation() error message '%s' does not contain '%s'", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GenerateDocumentation() expected no error but got: %v", err)
				}

				// Verify README.md was created
				readmePath := filepath.Join(workingDir, "README.md")
				if _, err := os.Stat(readmePath); os.IsNotExist(err) {
					t.Errorf("GenerateDocumentation() did not create README.md")
				}
			}
		})
	}
}

func TestParseTerraformModulesOutput(t *testing.T) {
	rg := &ReadmeGenerator{}
	output := `
"eventbridge_create_dump"[registry.terraform.io/terraform-aws-modules/eventbridge/aws] 4.1.0
"vpc_module"[registry.terraform.io/terraform-aws-modules/vpc/aws] 5.0.0
`
	modules, err := rg.parseTerraformModulesOutput(output)
	if err != nil {
		t.Fatalf("parseTerraformModulesOutput() error: %v", err)
	}
	if len(modules) == 0 {
		t.Error("parseTerraformModulesOutput() expected at least one module")
	}
	for _, m := range modules {
		if m.Version == "" {
			t.Errorf("parseTerraformModulesOutput() module %s has empty Version", m.Name)
		}
	}
}

func TestFormatRow(t *testing.T) {
	rg := &ReadmeGenerator{}
	row := []string{"name", "desc", "string"}
	colWidths := []int{6, 6, 8}
	got := rg.formatRow(row, colWidths)
	if got == "" {
		t.Error("formatRow() returned empty string")
	}
	if !strings.HasPrefix(got, "| ") || !strings.HasSuffix(got, " |") {
		t.Errorf("formatRow() should produce markdown row: got %q", got)
	}
}

func TestFormatSeparator(t *testing.T) {
	rg := &ReadmeGenerator{}
	colWidths := []int{4, 6, 8}
	got := rg.formatSeparator(colWidths)
	if got == "" {
		t.Error("formatSeparator() returned empty string")
	}
	if !strings.Contains(got, "----") {
		t.Errorf("formatSeparator() should contain dashes: got %q", got)
	}
}

func TestPrettifyMarkdownTable(t *testing.T) {
	rg := &ReadmeGenerator{}
	rows := [][]string{
		{"a", "b", "c"},
		{"x", "y", "z"},
	}
	header := []string{"Col1", "Col2", "Col3"}
	got := rg.prettifyMarkdownTable(rows, header)
	if got == "" {
		t.Error("prettifyMarkdownTable() returned empty string")
	}
	if !strings.Contains(got, "Col1") || !strings.Contains(got, "Col2") {
		t.Errorf("prettifyMarkdownTable() should contain header: got %q", got)
	}
}

func TestLoadEmbeddedTemplate(t *testing.T) {
	rgReadme := &ReadmeGenerator{IsExample: false}
	content, err := rgReadme.loadEmbeddedTemplate()
	if err != nil {
		t.Fatalf("loadEmbeddedTemplate(readme) error: %v", err)
	}
	if content == "" {
		t.Error("loadEmbeddedTemplate(readme) returned empty")
	}

	rgExample := &ReadmeGenerator{IsExample: true}
	contentEx, err := rgExample.loadEmbeddedTemplate()
	if err != nil {
		t.Fatalf("loadEmbeddedTemplate(example) error: %v", err)
	}
	if contentEx == "" {
		t.Error("loadEmbeddedTemplate(example) returned empty")
	}
	if content == contentEx {
		t.Error("loadEmbeddedTemplate: readme and example templates should differ")
	}
}
