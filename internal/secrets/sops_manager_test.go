package secrets

import (
	"errors"
	"testing"

	"gocloud-cli/internal/models"
	"gocloud-cli/internal/testutils"
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

// TestCheckKMSKey_WithMockClient_KeyExists exercises the real checkKMSKey AWS-call path
// (previously unreachable in tests since testCheckKMSKeyFunc always bypassed it) using an
// injected MockKMSClient via NewSOPSManagerWithClient.
func TestCheckKMSKey_WithMockClient_KeyExists(t *testing.T) {
	config := minimalSOPSConfig("lab")
	mockClient := &testutils.MockKMSClient{
		Keys: map[string]string{"alias/gcl-lab-secrets": "key-999"},
	}
	m, err := NewSOPSManagerWithClient(config, ".", mockClient)
	if err != nil {
		t.Fatalf("NewSOPSManagerWithClient: %v", err)
	}

	exists, keyId, err := m.checkKMSKey("lab")
	if err != nil {
		t.Fatalf("checkKMSKey: %v", err)
	}
	if !exists {
		t.Error("checkKMSKey() exists = false, want true")
	}
	if keyId != "key-999" {
		t.Errorf("checkKMSKey() keyId = %q, want %q", keyId, "key-999")
	}
}

// TestCheckKMSKey_WithMockClient_NotFound verifies checkKMSKey returns (false, "", nil) when
// the KMS alias does not exist (mock DescribeKey returns NotFoundException).
func TestCheckKMSKey_WithMockClient_NotFound(t *testing.T) {
	config := minimalSOPSConfig("lab")
	mockClient := &testutils.MockKMSClient{}
	m, err := NewSOPSManagerWithClient(config, ".", mockClient)
	if err != nil {
		t.Fatalf("NewSOPSManagerWithClient: %v", err)
	}

	exists, keyId, err := m.checkKMSKey("lab")
	if err != nil {
		t.Fatalf("checkKMSKey: %v", err)
	}
	if exists {
		t.Error("checkKMSKey() exists = true, want false")
	}
	if keyId != "" {
		t.Errorf("checkKMSKey() keyId = %q, want empty", keyId)
	}
}

// TestCheckKMSKey_WithMockClient_OtherError verifies checkKMSKey propagates unexpected errors
// (i.e. not a NotFound-style error) from DescribeKey.
func TestCheckKMSKey_WithMockClient_OtherError(t *testing.T) {
	config := minimalSOPSConfig("lab")
	mockClient := &testutils.MockKMSClient{
		DescribeKeyError: errors.New("AccessDenied: not authorized"),
	}
	m, err := NewSOPSManagerWithClient(config, ".", mockClient)
	if err != nil {
		t.Fatalf("NewSOPSManagerWithClient: %v", err)
	}

	_, _, err = m.checkKMSKey("lab")
	if err == nil {
		t.Fatal("checkKMSKey() expected error but got nil")
	}
}

// TestEnsureKMSKey_WithMockClient_CreatesKeyAndAlias exercises the real key-creation path
// (CreateKey + CreateAlias) via an injected MockKMSClient, previously unreachable in tests.
func TestEnsureKMSKey_WithMockClient_CreatesKeyAndAlias(t *testing.T) {
	config := minimalSOPSConfig("lab")
	mockClient := &testutils.MockKMSClient{NextKeyID: "new-key-id"}
	m, err := NewSOPSManagerWithClient(config, ".", mockClient)
	if err != nil {
		t.Fatalf("NewSOPSManagerWithClient: %v", err)
	}

	if err := m.ensureKMSKey("lab"); err != nil {
		t.Fatalf("ensureKMSKey: %v", err)
	}

	gotKeyId, exists := mockClient.Keys["alias/gcl-lab-secrets"]
	if !exists {
		t.Fatal("ensureKMSKey() did not create alias 'alias/gcl-lab-secrets'")
	}
	if gotKeyId != "new-key-id" {
		t.Errorf("created alias key id = %q, want %q", gotKeyId, "new-key-id")
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
