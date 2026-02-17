package secrets

import (
	"strings"
	"testing"

	"gocloud-cli/internal/models"
)

func TestResolveSecretsConfig_NilConfig(t *testing.T) {
	got, err := ResolveSecretsConfig(nil, "base/prd")
	if err != nil {
		t.Fatalf("ResolveSecretsConfig(nil, ...) unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("ResolveSecretsConfig(nil, ...) returned nil config")
	}
	if got.Type != "ssm" {
		t.Errorf("ResolveSecretsConfig(nil, ...).Type = %q, want ssm", got.Type)
	}
}

func TestResolveSecretsConfig_NilInfrastructure(t *testing.T) {
	config := &models.Config{Infrastructure: nil}
	got, err := ResolveSecretsConfig(config, "base/prd")
	if err != nil {
		t.Fatalf("ResolveSecretsConfig(nil infra, ...) unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("ResolveSecretsConfig(nil infra, ...) returned nil config")
	}
	if got.Type != "ssm" {
		t.Errorf("ResolveSecretsConfig(nil infra, ...).Type = %q, want ssm", got.Type)
	}
}

func TestResolveSecretsConfig_InvalidLayerPath(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Company: "gcl",
			Secrets: &models.SecretsConfig{Type: "ssm"},
		},
	}

	tests := []struct {
		layerPath string
	}{
		{"base"},    // 1 part
		{"a/b/c/d"}, // 4 parts
		{"base"},    // 1 part
		{"onlyone"},
		{"one/two/three/four"},
	}

	for _, tt := range tests {
		t.Run(tt.layerPath, func(t *testing.T) {
			_, err := ResolveSecretsConfig(config, tt.layerPath)
			if err == nil {
				t.Errorf("ResolveSecretsConfig(%q) expected error", tt.layerPath)
			}
			if !strings.Contains(err.Error(), "invalid layer path") {
				t.Errorf("ResolveSecretsConfig(%q) error = %q, want to contain 'invalid layer path'", tt.layerPath, err.Error())
			}
		})
	}
}

func TestResolveSecretsConfig_ValidPaths(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Company: "gcl",
			Secrets: &models.SecretsConfig{Type: "ssm"},
			Environments: map[string]models.Environment{
				"prd": {
					Name:       "Production",
					DirName:    "prd",
					AWSAccount: "123456789012",
					Secrets:    &models.SecretsConfig{Type: "sops"},
				},
			},
		},
	}

	tests := []struct {
		layerPath   string
		wantType    string
		description string
	}{
		{"base/prd", "sops", "base layer uses env secrets"},
		{"foundation/prd", "sops", "foundation layer uses env secrets"},
		{"base/dev", "ssm", "base/dev falls back to global ssm when env missing"},
		{"organization", "ssm", "organization layer uses global when no override"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			got, err := ResolveSecretsConfig(config, tt.layerPath)
			if err != nil {
				t.Fatalf("ResolveSecretsConfig(%q) error: %v", tt.layerPath, err)
			}
			if got == nil {
				t.Fatal("ResolveSecretsConfig returned nil")
			}
			if got.Type != tt.wantType {
				t.Errorf("ResolveSecretsConfig(%q).Type = %q, want %q", tt.layerPath, got.Type, tt.wantType)
			}
		})
	}
}

func TestResolveSecretsConfig_ProjectWorkloadPath(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Company: "gcl",
			Secrets: &models.SecretsConfig{Type: "ssm"},
			Environments: map[string]models.Environment{
				"prd": {
					Name:       "Production",
					DirName:    "prd",
					AWSAccount: "123456789012",
					Projects: []interface{}{
						models.ProjectItem{
							Key:     "core",
							Name:    "Core",
							Secrets: &models.SecretsConfig{Type: "sops"},
						},
					},
				},
			},
		},
	}

	got, err := ResolveSecretsConfig(config, "project/core/prd")
	if err != nil {
		t.Fatalf("ResolveSecretsConfig(project/core/prd) error: %v", err)
	}
	if got == nil {
		t.Fatal("ResolveSecretsConfig returned nil")
	}
	if got.Type != "sops" {
		t.Errorf("ResolveSecretsConfig(project/core/prd).Type = %q, want sops", got.Type)
	}
}

func TestResolveSecretsConfig_OrganizationOverride(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Company: "gcl",
			Secrets: &models.SecretsConfig{Type: "ssm"},
			Organization: &models.OrganizationLayerConfig{
				Secrets: &models.SecretsConfig{Type: "sops"},
			},
		},
	}
	got, err := ResolveSecretsConfig(config, "organization")
	if err != nil {
		t.Fatalf("ResolveSecretsConfig(organization) error: %v", err)
	}
	if got == nil {
		t.Fatal("ResolveSecretsConfig(organization) returned nil")
	}
	if got.Type != "sops" {
		t.Errorf("ResolveSecretsConfig(organization with override).Type = %q, want sops", got.Type)
	}
}

func TestNewManagerForLayer_SSM(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Client:  "test-client",
			Company: "gcl",
			Secrets: &models.SecretsConfig{Type: "ssm"},
		},
	}

	mgr, err := NewManagerForLayer(config, "base/prd", "")
	if err != nil {
		t.Fatalf("NewManagerForLayer(ssm) error: %v", err)
	}
	if mgr == nil {
		t.Fatal("NewManagerForLayer(ssm) returned nil")
	}
	// Should be SSM manager (we don't call AWS in test)
	if _, ok := mgr.(*Manager); !ok {
		t.Errorf("NewManagerForLayer(ssm) did not return *Manager, got %T", mgr)
	}
}

func TestNewManagerForLayer_SOPS(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Client:  "test-client",
			Company: "gcl",
			Secrets: &models.SecretsConfig{Type: "sops"},
			Environments: map[string]models.Environment{
				"prd": {
					Name:       "Production",
					DirName:    "prd",
					AWSAccount: "123456789012",
				},
			},
		},
	}

	tmpDir := t.TempDir()
	mgr, err := NewManagerForLayer(config, "base/prd", tmpDir)
	if err != nil {
		t.Fatalf("NewManagerForLayer(sops) error: %v", err)
	}
	if mgr == nil {
		t.Fatal("NewManagerForLayer(sops) returned nil")
	}
	if _, ok := mgr.(*SOPSManager); !ok {
		t.Errorf("NewManagerForLayer(sops) did not return *SOPSManager, got %T", mgr)
	}
}

func TestNewManagerForLayer_InvalidLayerPath(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Company: "gcl",
			Secrets: &models.SecretsConfig{Type: "ssm"},
		},
	}

	_, err := NewManagerForLayer(config, "invalid", "")
	if err == nil {
		t.Fatal("NewManagerForLayer(invalid path) expected error")
	}
	if !strings.Contains(err.Error(), "invalid layer path") {
		t.Errorf("NewManagerForLayer error = %q, want to contain 'invalid layer path'", err.Error())
	}
}

func TestNewManagerForLayer_UnknownBackendType(t *testing.T) {
	config := &models.Config{
		Infrastructure: &models.InfrastructureConfig{
			Company: "gcl",
			Secrets: &models.SecretsConfig{Type: "unknown"},
			Environments: map[string]models.Environment{
				"prd": {Name: "Production", DirName: "prd", AWSAccount: "123456789012"},
			},
		},
	}

	_, err := NewManagerForLayer(config, "base/prd", "")
	if err == nil {
		t.Fatal("NewManagerForLayer(unknown type) expected error")
	}
	if !strings.Contains(err.Error(), "unknown secrets backend type") {
		t.Errorf("NewManagerForLayer error = %q, want to contain 'unknown secrets backend type'", err.Error())
	}
}
