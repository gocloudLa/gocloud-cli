package secrets

import (
	"testing"

	"gocloud-cli/internal/models"
)

// minimalSOPSConfig returns a config valid for NewSOPSManager with one environment.
func minimalSOPSConfig(envKey string) *models.Config {
	return &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Client:  "test",
			Company: "gcl",
			Region:  "us-east-1",
			Environments: map[string]models.Environment{
				envKey: {
					Name:       "Lab",
					AWSAccount: "123456789012",
				},
			},
		},
	}
}

// TestEnsureAllKMSKeys_WhenKeyExists_DoesNotCallCreate verifies that when the KMS alias
// already exists (simulated via testCheckKMSKeyFunc), EnsureAllKMSKeys does not attempt
// to create the key/alias. This documents the fix for AlreadyExistsException when
// running secrets init --all on a project where the KMS already exists.
func TestEnsureAllKMSKeys_WhenKeyExists_DoesNotCallCreate(t *testing.T) {
	config := minimalSOPSConfig("lab")
	m, err := NewSOPSManager(config, ".")
	if err != nil {
		t.Fatalf("NewSOPSManager: %v", err)
	}
	// Simulate "alias already exists in AWS" so no CreateKey/CreateAlias is called.
	m.testCheckKMSKeyFunc = func(envKey string) (bool, string, error) {
		return true, "key-123", nil
	}

	err = m.EnsureAllKMSKeys([]string{"lab"}, true)
	if err != nil {
		t.Errorf("EnsureAllKMSKeys when key exists: %v", err)
	}
}

// TestEnsureKMSKey_WhenKeyExists_ReturnsNilWithoutCallingAWS verifies that ensureKMSKey
// (used by InitSecrets) returns nil and does not call CreateKey/CreateAlias when the
// alias already exists. This is the fix for the bug where ensureKMSKey tried to create
// the alias without checking first, causing AlreadyExistsException.
func TestEnsureKMSKey_WhenKeyExists_ReturnsNilWithoutCallingAWS(t *testing.T) {
	config := minimalSOPSConfig("lab")
	m, err := NewSOPSManager(config, ".")
	if err != nil {
		t.Fatalf("NewSOPSManager: %v", err)
	}
	m.testCheckKMSKeyFunc = func(envKey string) (bool, string, error) {
		return true, "key-123", nil
	}

	err = m.ensureKMSKey("lab")
	if err != nil {
		t.Errorf("ensureKMSKey when key exists should return nil: %v", err)
	}
}

// TestEnsureAllKMSKeys_WhenKeyDoesNotExist_AndCreateFalse_CompletesWithoutError verifies that
// when the key does not exist and createIfMissing is false, EnsureAllKMSKeys completes
// without calling AWS create (only prints NOT_FOUND).
func TestEnsureAllKMSKeys_WhenKeyDoesNotExist_AndCreateFalse_CompletesWithoutError(t *testing.T) {
	config := minimalSOPSConfig("lab")
	m, err := NewSOPSManager(config, ".")
	if err != nil {
		t.Fatalf("NewSOPSManager: %v", err)
	}
	m.testCheckKMSKeyFunc = func(envKey string) (bool, string, error) {
		return false, "", nil
	}

	err = m.EnsureAllKMSKeys([]string{"lab"}, false)
	if err != nil {
		t.Errorf("EnsureAllKMSKeys when key does not exist and createIfMissing=false: %v", err)
	}
}

func TestSOPSManagerParseLayerPath(t *testing.T) {
	config := minimalSOPSConfig("prd")
	config.Infrastructure.Environments["dev"] = models.Environment{
		Name:       "Development",
		AWSAccount: "123456789012",
	}
	prd := config.Infrastructure.Environments["prd"]
	prd.Projects = []interface{}{"core"}
	prd.Workloads = []interface{}{"web"}
	config.Infrastructure.Environments["prd"] = prd
	m, err := NewSOPSManager(config, ".")
	if err != nil {
		t.Fatalf("NewSOPSManager: %v", err)
	}

	tests := []struct {
		layerPath   string
		expectError bool
	}{
		{"base/prd", false},
		{"foundation/dev", false},
		{"project/core/prd", false},
		{"workload/web/prd", false},
		{"base", true},
		{"invalid/layer", true},
	}
	for _, tt := range tests {
		t.Run(tt.layerPath, func(t *testing.T) {
			layer, err := m.ParseLayerPath(tt.layerPath)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseLayerPath(%q) expected error", tt.layerPath)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseLayerPath(%q): %v", tt.layerPath, err)
			}
			if layer == nil {
				t.Errorf("ParseLayerPath(%q) returned nil layer", tt.layerPath)
			}
		})
	}
}
