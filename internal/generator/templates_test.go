package generator

import (
	"strings"
	"testing"

	"gocloud-cli/internal/models"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := NewTemplateEngine()

	if engine == nil {
		t.Error("NewTemplateEngine() returned nil")
		return
	}

	if engine.templates == nil {
		t.Error("NewTemplateEngine() templates map should not be nil")
	}

	// Check that all expected templates are loaded
	expectedTemplates := []string{
		"terragrunt.hcl.tpl",
		"metadata.tf.tpl",
		"_secrets.tf.tpl",
		"main.tf.base.tpl",
		"main.tf.foundation.tpl",
		"main.tf.project.tpl",
		"main.tf.workload.tpl",
		"main.tf.organization.tpl",
		"main.tf.security.tpl",
	}

	for _, templateName := range expectedTemplates {
		if _, exists := engine.templates[templateName]; !exists {
			t.Errorf("Template %s should be loaded but was not found", templateName)
		}
	}
}

func TestTemplateEngineRender(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name             string
		templateName     string
		data             *models.TemplateData
		expectError      bool
		expectedContains []string
	}{
		{
			name:         "render metadata template",
			templateName: "metadata.tf.tpl",
			data: &models.TemplateData{
				Client:          "test-client",
				Company:         "gcl",
				Region:          "us-east-1",
				RegionShortCode: "use1",
				Version:         "v1.0.0",
				Environment:     "dev",
				EnvironmentName: "Development",
				EnvKey:          "dev",
				Layer:           "base",
			},
			expectError: false,
			expectedContains: []string{
				"aws_region  = \"us-east-1\"",
				"environment = \"Development\"",
				"company = \"gcl\"",
				"region  = \"use1\"",
				"env     = \"dev\"",
				"layer   = \"base\"",
			},
		},
		{
			name:         "render metadata template with project",
			templateName: "metadata.tf.tpl",
			data: &models.TemplateData{
				Client:          "test-client",
				Company:         "gcl",
				Region:          "us-east-1",
				RegionShortCode: "use1",
				Version:         "v1.0.0",
				Environment:     "dev",
				EnvironmentName: "Development",
				EnvKey:          "dev",
				Project:         "core",
				ProjectKey:      "core",
				ProjectName:     "core",
				Layer:           "project",
			},
			expectError: false,
			expectedContains: []string{
				"project     = \"core\"",
				"project = \"core\"",
			},
		},
		{
			name:         "render metadata template with custom metadata",
			templateName: "metadata.tf.tpl",
			data: &models.TemplateData{
				Client:          "test-client",
				Company:         "gcl",
				Region:          "us-east-1",
				RegionShortCode: "use1",
				Version:         "v1.0.0",
				Environment:     "dev",
				EnvironmentName: "Development",
				EnvKey:          "dev",
				Layer:           "base",
				MetadataLines: []string{
					"    public_domain  = \"example.com\"",
					"    private_domain = \"internal.example.com\"",
				},
			},
			expectError: false,
			expectedContains: []string{
				"public_domain  = \"example.com\"",
				"private_domain = \"internal.example.com\"",
			},
		},
		{
			name:         "render terragrunt template without dependencies",
			templateName: "terragrunt.hcl.tpl",
			data: &models.TemplateData{
				Dependencies: []string{},
			},
			expectError: false,
			expectedContains: []string{
				"include \"root\" {",
				"path = find_in_parent_folders(\"root.hcl\")",
			},
		},
		{
			name:         "render terragrunt template with dependencies",
			templateName: "terragrunt.hcl.tpl",
			data: &models.TemplateData{
				Dependencies: []string{"../base", "../foundation"},
			},
			expectError: false,
			expectedContains: []string{
				"include \"root\" {",
				"dependencies {",
				"\"../base\",",
				"\"../foundation\",",
			},
		},
		{
			name:         "render main.tf.base template",
			templateName: "main.tf.base.tpl",
			data: &models.TemplateData{
				Version: "v1.0.0",
			},
			expectError: false,
			expectedContains: []string{
				"module \"base\" {",
				"source  = \"gocloudLa/standard-platform/aws//modules/base\"",
				"version = \"v1.0.0\"",
				"metadata = local.metadata",
			},
		},
		{
			name:         "render main.tf.foundation template",
			templateName: "main.tf.foundation.tpl",
			data: &models.TemplateData{
				Version: "v1.0.0",
			},
			expectError: false,
			expectedContains: []string{
				"module \"foundation\" {",
				"source  = \"gocloudLa/standard-platform/aws//modules/foundation\"",
				"version = \"v1.0.0\"",
				"providers = {",
				"aws.use1 = aws.use1",
			},
		},
		{
			name:         "render main.tf.project template",
			templateName: "main.tf.project.tpl",
			data: &models.TemplateData{
				Version: "v1.0.0",
			},
			expectError: false,
			expectedContains: []string{
				"module \"project\" {",
				"source  = \"gocloudLa/standard-platform/aws//modules/project\"",
				"version = \"v1.0.0\"",
				"metadata = local.metadata",
			},
		},
		{
			name:         "render main.tf.workload template",
			templateName: "main.tf.workload.tpl",
			data: &models.TemplateData{
				Version: "v1.0.0",
			},
			expectError: false,
			expectedContains: []string{
				"module \"workload\" {",
				"source  = \"gocloudLa/standard-platform/aws//modules/workload\"",
				"version = \"v1.0.0\"",
				"providers = {",
				"aws.use1 = aws.use1",
				"metadata = local.metadata",
			},
		},
		{
			name:         "render main.tf.organization template",
			templateName: "main.tf.organization.tpl",
			data: &models.TemplateData{
				Version: "v1.0.0",
			},
			expectError: false,
			expectedContains: []string{
				"module \"organization\" {",
				"source  = \"gocloudLa/standard-platform/aws//modules/organization\"",
				"version = \"v1.0.0\"",
			},
		},
		{
			name:         "render main.tf.security template",
			templateName: "main.tf.security.tpl",
			data: &models.TemplateData{
				Version: "v1.0.0",
			},
			expectError: false,
			expectedContains: []string{
				"module \"security\" {",
				"source  = \"gocloudLa/standard-platform/aws//modules/security\"",
				"version = \"v1.0.0\"",
				"providers = {",
				"aws.log = aws.log",
				"aws.kms = aws.kms",
			},
		},
		{
			name:         "render _secrets.tf template",
			templateName: "_secrets.tf.tpl",
			data: &models.TemplateData{
				Layer: "base",
			},
			expectError: false,
			expectedContains: []string{
				"data \"aws_ssm_parameter\" \"terraform\" {",
				"name = \"/terraform/${local.common_name}-base\"",
				"locals {",
				"secrets = jsondecode(data.aws_ssm_parameter.terraform.value)",
			},
		},
		{
			name:         "non-existent template",
			templateName: "non-existent.tpl",
			data:         &models.TemplateData{},
			expectError:  true,
		},
		{
			name:         "render metadata template for organization",
			templateName: "metadata.tf.tpl",
			data: &models.TemplateData{
				Client:          "test-client",
				Company:         "gcl",
				Region:          "us-east-1",
				RegionShortCode: "use1",
				Version:         "v1.0.0",
				Environment:     "org",
				EnvironmentName: "Organization",
				EnvKey:          "org",
				Layer:           "organization",
				MetadataLines: []string{
					"    public_domain  = \"example.com\"",
					"    private_domain = \"internal.example.com\"",
				},
			},
			expectError: false,
			expectedContains: []string{
				"aws_region  = \"us-east-1\"",
				"environment = \"Organization\"",
				"public_domain  = \"example.com\"",
				"private_domain = \"internal.example.com\"",
				"env     = \"org\"",
				"layer   = \"organization\"",
			},
		},
		{
			name:         "nil data",
			templateName: "metadata.tf.tpl",
			data:         nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Render(tt.templateName, tt.data)

			if tt.expectError {
				if err == nil {
					t.Errorf("Render() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Render() expected no error but got: %v", err)
				}

				// Check that result contains expected strings
				for _, expected := range tt.expectedContains {
					if !strings.Contains(result, expected) {
						t.Errorf("Render() result does not contain expected string: %s", expected)
					}
				}

				// Check that result is not empty
				if strings.TrimSpace(result) == "" {
					t.Error("Render() result should not be empty")
				}
			}
		})
	}
}

func TestTemplateEngineRenderWithComplexData(t *testing.T) {
	engine := NewTemplateEngine()

	// Test with complex data structure
	data := &models.TemplateData{
		Client:          "test-client",
		Company:         "gcl",
		Region:          "us-east-1",
		RegionShortCode: "use1",
		Version:         "v1.0.0",
		Environment:     "dev",
		EnvironmentName: "Development",
		EnvKey:          "dev",
		Project:         "core",
		ProjectKey:      "core",
		ProjectName:     "core",
		Layer:           "project",
		Metadata: map[string]interface{}{
			"public_domain":  "example.com",
			"private_domain": "internal.example.com",
		},
		MetadataLines: []string{
			"    public_domain  = \"example.com\"",
			"    private_domain = \"internal.example.com\"",
		},
		Dependencies: []string{"../foundation", "../base"},
	}

	result, err := engine.Render("metadata.tf.tpl", data)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	// Check that all expected content is present
	expectedContent := []string{
		"aws_region  = \"us-east-1\"",
		"environment = \"Development\"",
		"project     = \"core\"",
		"public_domain  = \"example.com\"",
		"private_domain = \"internal.example.com\"",
		"company = \"gcl\"",
		"region  = \"use1\"",
		"env     = \"dev\"",
		"project = \"core\"",
		"layer   = \"project\"",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Render() result does not contain expected string: %s", expected)
		}
	}
}

func TestTemplateEngineRenderTerragruntWithDependencies(t *testing.T) {
	engine := NewTemplateEngine()

	data := &models.TemplateData{
		Dependencies: []string{"../base", "../foundation", "../project/core"},
	}

	result, err := engine.Render("terragrunt.hcl.tpl", data)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	// Check that dependencies section is present
	expectedContent := []string{
		"include \"root\" {",
		"path = find_in_parent_folders(\"root.hcl\")",
		"dependencies {",
		"paths = [",
		"\"../base\",",
		"\"../foundation\",",
		"\"../project/core\",",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Render() result does not contain expected string: %s", expected)
		}
	}
}

func TestTemplateEngineRenderTerragruntWithoutDependencies(t *testing.T) {
	engine := NewTemplateEngine()

	data := &models.TemplateData{
		Dependencies: []string{},
	}

	result, err := engine.Render("terragrunt.hcl.tpl", data)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	// Check that dependencies section is NOT present
	if strings.Contains(result, "dependencies {") {
		t.Error("Render() result should not contain dependencies section when Dependencies is empty")
	}

	// Check that basic structure is present
	expectedContent := []string{
		"include \"root\" {",
		"path = find_in_parent_folders(\"root.hcl\")",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(result, expected) {
			t.Errorf("Render() result does not contain expected string: %s", expected)
		}
	}
}

func TestTemplateEngineRenderSecretsTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		layer    string
		expected string
	}{
		{
			name:     "base layer",
			layer:    "base",
			expected: "/terraform/${local.common_name}-base",
		},
		{
			name:     "foundation layer",
			layer:    "foundation",
			expected: "/terraform/${local.common_name}-foundation",
		},
		{
			name:     "project layer",
			layer:    "project",
			expected: "/terraform/${local.common_name}-project",
		},
		{
			name:     "workload layer",
			layer:    "workload",
			expected: "/terraform/${local.common_name}-workload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &models.TemplateData{
				Layer: tt.layer,
			}

			result, err := engine.Render("_secrets.tf.tpl", data)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			if !strings.Contains(result, tt.expected) {
				t.Errorf("Render() result does not contain expected string: %s", tt.expected)
			}
		})
	}
}

func TestTemplateEngineRenderMainTemplates(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name           string
		templateName   string
		expectedModule string
	}{
		{
			name:           "base template",
			templateName:   "main.tf.base.tpl",
			expectedModule: "base",
		},
		{
			name:           "foundation template",
			templateName:   "main.tf.foundation.tpl",
			expectedModule: "foundation",
		},
		{
			name:           "project template",
			templateName:   "main.tf.project.tpl",
			expectedModule: "project",
		},
		{
			name:           "workload template",
			templateName:   "main.tf.workload.tpl",
			expectedModule: "workload",
		},
		{
			name:           "organization template",
			templateName:   "main.tf.organization.tpl",
			expectedModule: "organization",
		},
		{
			name:           "security template",
			templateName:   "main.tf.security.tpl",
			expectedModule: "security",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &models.TemplateData{
				Version: "v1.0.0",
			}

			result, err := engine.Render(tt.templateName, data)
			if err != nil {
				t.Fatalf("Render() failed: %v", err)
			}

			expectedContent := []string{
				"module \"" + tt.expectedModule + "\" {",
				"source  = \"gocloudLa/standard-platform/aws//modules/" + tt.expectedModule + "\"",
				"version = \"v1.0.0\"",
			}
			if tt.expectedModule == "security" {
				expectedContent = append(expectedContent,
					"providers = {",
					"aws.log = aws.log",
					"aws.kms = aws.kms",
				)
			}

			for _, expected := range expectedContent {
				if !strings.Contains(result, expected) {
					t.Errorf("Render() result does not contain expected string: %s", expected)
				}
			}
		})
	}
}

func TestTemplateEngineRenderProvidersWithAssumeRole(t *testing.T) {
	engine := NewTemplateEngine()
	data := &models.TemplateData{
		Providers: []models.ProviderSpec{
			{
				Name:   "aws",
				Region: "local.metadata.aws_region",
				Alias:  "sha",
				AssumeRole: &models.ProviderAssumeRole{
					RoleARN:     "arn:aws:iam::904233109008:role/OrganizationAccountAccessRole",
					SessionName: "TerraformSession",
				},
			},
		},
	}
	result, err := engine.Render("providers.tf.tpl", data)
	if err != nil {
		t.Fatalf("Render() failed: %v", err)
	}
	if want := "region  = local.metadata.aws_region"; !strings.Contains(result, want) {
		t.Errorf("Render() result should contain %q for unquoted local ref; got:\n%s", want, result)
	}
	for _, s := range []string{"assume_role {", "role_arn", "OrganizationAccountAccessRole", "session_name", "TerraformSession"} {
		if !strings.Contains(result, s) {
			t.Errorf("Render() result does not contain expected string: %q", s)
		}
	}
}

// Benchmark tests for template rendering performance
func BenchmarkTemplateEngineRender(b *testing.B) {
	engine := NewTemplateEngine()
	data := &models.TemplateData{
		Client:          "test-client",
		Company:         "gcl",
		Region:          "us-east-1",
		RegionShortCode: "use1",
		Version:         "v1.0.0",
		Environment:     "dev",
		EnvironmentName: "Development",
		EnvKey:          "dev",
		Layer:           "base",
		MetadataLines: []string{
			"    public_domain  = \"example.com\"",
			"    private_domain = \"internal.example.com\"",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Render("metadata.tf.tpl", data)
		if err != nil {
			b.Fatalf("Render() failed: %v", err)
		}
	}
}

func BenchmarkTemplateEngineRenderComplex(b *testing.B) {
	engine := NewTemplateEngine()
	data := &models.TemplateData{
		Client:          "test-client",
		Company:         "gcl",
		Region:          "us-east-1",
		RegionShortCode: "use1",
		Version:         "v1.0.0",
		Environment:     "Development",
		EnvKey:          "dev",
		Project:         "core",
		Layer:           "project",
		Metadata: map[string]interface{}{
			"public_domain":  "example.com",
			"private_domain": "internal.example.com",
		},
		MetadataLines: []string{
			"    public_domain  = \"example.com\"",
			"    private_domain = \"internal.example.com\"",
		},
		Dependencies: []string{"../foundation", "../base"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.Render("terragrunt.hcl.tpl", data)
		if err != nil {
			b.Fatalf("Render() failed: %v", err)
		}
	}
}
