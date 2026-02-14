package cmd

import (
	"testing"

	"gocloud-cli/internal/models"
)

func TestSecretsInitHelperFunctions(t *testing.T) {
	// Test with empty configuration
	config := &models.Config{}

	// Test getAllLayersFromConfig with empty config
	layers := getAllLayersFromConfig(config)
	if len(layers) != 0 {
		t.Errorf("Expected 0 layers for empty config, got %d", len(layers))
	}

	// Test getLayersForEnvironment with empty config
	layers = getLayersForEnvironment(config, "dev")
	if len(layers) != 0 {
		t.Errorf("Expected 0 layers for empty config, got %d", len(layers))
	}
}

func TestSecretsInitWithSampleConfig(t *testing.T) {
	// Create a sample configuration
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Environments: map[string]models.Environment{
				"dev": {
					Projects:  []interface{}{"core", "api"},
					Workloads: []interface{}{"webapp", "worker"},
				},
				"staging": {
					Projects:  []interface{}{"core"},
					Workloads: []interface{}{"webapp"},
				},
			},
		},
	}

	// Test getAllLayersFromConfig
	layers := getAllLayersFromConfig(config)
	expectedLayers := []string{
		"base/dev", "base/staging",
		"foundation/dev", "foundation/staging",
		"project/core/dev", "project/api/dev", "project/core/staging",
		"workload/webapp/dev", "workload/worker/dev", "workload/webapp/staging",
		// organization only when infrastructure.organization.aws_account is set
	}

	if len(layers) != len(expectedLayers) {
		t.Errorf("Expected %d layers, got %d", len(expectedLayers), len(layers))
	}

	// Test getLayersForEnvironment
	devLayers := getLayersForEnvironment(config, "dev")
	expectedDevLayers := []string{
		"base/dev", "foundation/dev",
		"project/core/dev", "project/api/dev",
		"workload/webapp/dev", "workload/worker/dev",
	}

	if len(devLayers) != len(expectedDevLayers) {
		t.Errorf("Expected %d dev layers, got %d", len(expectedDevLayers), len(devLayers))
	}

	// Test getLayersForEnvironment with non-existent environment
	nonExistentLayers := getLayersForEnvironment(config, "production")
	if len(nonExistentLayers) != 0 {
		t.Errorf("Expected 0 layers for non-existent environment, got %d", len(nonExistentLayers))
	}
}
