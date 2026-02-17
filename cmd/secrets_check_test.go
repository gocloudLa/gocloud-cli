package cmd

import (
	"testing"

	"gocloud-cli/internal/models"
)

func ptrBool(b bool) *bool { return &b }

func TestGetAllLayersFromConfig(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Environments: map[string]models.Environment{
				"dev": {
					Name:      "Development",
					Projects:  []interface{}{"core", "common"},
					Workloads: []interface{}{"api", "webapp"},
				},
				"prd": {
					Name:      "Production",
					Projects:  []interface{}{"core"},
					Workloads: []interface{}{"api"},
				},
			},
		},
	}

	layers := getAllLayersFromConfig(config)

	expectedLayers := []string{
		"base/dev", "foundation/dev",
		"base/prd", "foundation/prd",
		"project/core/dev", "project/common/dev",
		"project/core/prd",
		"workload/api/dev", "workload/webapp/dev",
		"workload/api/prd",
		// organization only when infrastructure.organization.aws_account is set
	}

	if len(layers) != len(expectedLayers) {
		t.Errorf("Expected %d layers, got %d", len(expectedLayers), len(layers))
	}

	// Convert to map for easier checking
	layerMap := make(map[string]bool)
	for _, layer := range layers {
		layerMap[layer] = true
	}

	for _, expected := range expectedLayers {
		if !layerMap[expected] {
			t.Errorf("Expected layer %s not found", expected)
		}
	}
}

func TestGetLayersForEnvironment(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Environments: map[string]models.Environment{
				"dev": {
					Name:      "Development",
					Projects:  []interface{}{"core", "common"},
					Workloads: []interface{}{"api", "webapp"},
				},
				"prd": {
					Name:      "Production",
					Projects:  []interface{}{"core"},
					Workloads: []interface{}{"api"},
				},
			},
		},
	}

	// Test existing environment
	layers := getLayersForEnvironment(config, "dev")
	expectedLayers := []string{
		"base/dev", "foundation/dev",
		"project/core/dev", "project/common/dev",
		"workload/api/dev", "workload/webapp/dev",
	}

	if len(layers) != len(expectedLayers) {
		t.Errorf("Expected %d layers for dev, got %d", len(expectedLayers), len(layers))
	}

	// Convert to map for easier checking
	layerMap := make(map[string]bool)
	for _, layer := range layers {
		layerMap[layer] = true
	}

	for _, expected := range expectedLayers {
		if !layerMap[expected] {
			t.Errorf("Expected layer %s not found for dev environment", expected)
		}
	}

	// Test non-existing environment
	layersEmpty := getLayersForEnvironment(config, "nonexistent")
	if len(layersEmpty) != 0 {
		t.Errorf("Expected 0 layers for nonexistent environment, got %d", len(layersEmpty))
	}
}

func TestSecretsCheckHelperFunctions(t *testing.T) {
	// Test that helper functions don't crash with empty config (no envs, organization layer disabled)
	emptyConfig := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Environments: make(map[string]models.Environment),
			Layers: &models.LayerConfig{
				Organization: ptrBool(false), // so we get 0 layers with no envs
			},
		},
	}

	layers := getAllLayersFromConfig(emptyConfig)
	if len(layers) != 0 {
		t.Errorf("Expected 0 layers for empty config, got %d", len(layers))
	}

	envLayers := getLayersForEnvironment(emptyConfig, "any")
	if len(envLayers) != 0 {
		t.Errorf("Expected 0 layers for environment in empty config, got %d", len(envLayers))
	}
}

func TestParseLayerPathComponents_Organization(t *testing.T) {
	layerType, project, env, ok := parseLayerPathComponents("organization")
	if !ok {
		t.Fatal("parseLayerPathComponents(\"organization\") should be valid")
	}
	if layerType != "organization" || project != "" || env != "org" {
		t.Errorf("parseLayerPathComponents(\"organization\") = %q, %q, %q; want organization, \"\", org", layerType, project, env)
	}
}

func TestShouldGenerateSecretsForPath_Organization(t *testing.T) {
	infra := &models.InfrastructureConfig{
		Organization:  &models.OrganizationLayerConfig{AWSAccount: "123456789012"},
		EnableSecrets: ptrBool(true),
	}
	if !shouldGenerateSecretsForPath(infra, "organization", "", "org") {
		t.Error("shouldGenerateSecretsForPath(organization) with aws_account set should be true")
	}
	infra.Layers = &models.LayerConfig{Organization: ptrBool(false)}
	if shouldGenerateSecretsForPath(infra, "organization", "", "org") {
		t.Error("shouldGenerateSecretsForPath(organization) with layers.organization false should be false")
	}
}
