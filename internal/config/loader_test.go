package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const minimalValidYAML = `
infrastructure:
  client: test-client
  company: gcl
  region: us-east-1
  version: "0.17.0"
  environments:
    dev:
      name: Development
      dir_name: dev
      aws_account: "123456789012"
`

func TestLoadConfigWithPath_EmptyUsesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gocloud.yaml")
	if err := os.WriteFile(configPath, []byte(minimalValidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Warning: failed to restore cwd: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	cfg, err := LoadConfigWithPath("")
	if err != nil {
		t.Fatalf("LoadConfigWithPath(\"\") unexpected error: %v", err)
	}
	if cfg == nil || cfg.Infrastructure == nil {
		t.Fatal("LoadConfigWithPath(\"\") returned nil config or infrastructure")
	}
	if cfg.Infrastructure.Company != "gcl" {
		t.Errorf("LoadConfigWithPath(\"\") company = %q, want gcl", cfg.Infrastructure.Company)
	}
}

func TestLoadConfigWithPath_RelativePathInCwd(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gocloud.yaml")
	if err := os.WriteFile(configPath, []byte(minimalValidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Logf("Warning: failed to restore cwd: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	cfg, err := LoadConfigWithPath("gocloud.yaml")
	if err != nil {
		t.Fatalf("LoadConfigWithPath(\"gocloud.yaml\") unexpected error: %v", err)
	}
	if cfg == nil || cfg.Infrastructure == nil {
		t.Fatal("LoadConfigWithPath returned nil config or infrastructure")
	}
	if cfg.Infrastructure.Client != "test-client" {
		t.Errorf("LoadConfigWithPath client = %q, want test-client", cfg.Infrastructure.Client)
	}
}

func TestLoadConfigWithPath_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "custom.yaml")
	if err := os.WriteFile(configPath, []byte(minimalValidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := LoadConfigWithPath(configPath)
	if err != nil {
		t.Fatalf("LoadConfigWithPath(absolute) unexpected error: %v", err)
	}
	if cfg == nil || cfg.Infrastructure == nil {
		t.Fatal("LoadConfigWithPath returned nil config or infrastructure")
	}
	if cfg.Infrastructure.Region != "us-east-1" {
		t.Errorf("LoadConfigWithPath region = %q, want us-east-1", cfg.Infrastructure.Region)
	}
}

func TestLoadConfigWithPath_NotFound(t *testing.T) {
	_, err := LoadConfigWithPath("nonexistent-gocloud.yaml")
	if err == nil {
		t.Fatal("LoadConfigWithPath(nonexistent) expected error")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("LoadConfigWithPath error = %q, want to contain 'config file not found'", err.Error())
	}
}

func TestLoadConfigWithPath_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "bad.yaml")
	invalidYAML := "infrastructure:\n  company: [unclosed"
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	_, err := LoadConfigWithPath(configPath)
	if err == nil {
		t.Fatal("LoadConfigWithPath(invalid yaml) expected error")
	}
	// loadConfigFromFile wraps manager error; may be "invalid yaml syntax" or "failed to load configuration"
	if !strings.Contains(err.Error(), "yaml") && !strings.Contains(err.Error(), "failed to load") {
		t.Errorf("LoadConfigWithPath error = %q, want to contain 'yaml' or 'failed to load'", err.Error())
	}
}

func TestLoadConfigWithPathAndAWS_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gocloud.yaml")
	if err := os.WriteFile(configPath, []byte(minimalValidYAML), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := LoadConfigWithPathAndAWS(configPath)
	if err != nil {
		t.Fatalf("LoadConfigWithPathAndAWS unexpected error: %v", err)
	}
	if cfg == nil || cfg.Infrastructure == nil {
		t.Fatal("LoadConfigWithPathAndAWS returned nil config or infrastructure")
	}
	if got := os.Getenv("AWS_CONFIG_FILE"); got != "" {
		// Restore so we don't affect other tests
		_ = os.Unsetenv("AWS_CONFIG_FILE")
		if !strings.HasSuffix(got, ".aws/config") {
			t.Errorf("AWS_CONFIG_FILE = %q, expected path ending with .aws/config", got)
		}
	}
}

func TestLoadConfigWithPathAndAWS_PropagatesError(t *testing.T) {
	_, err := LoadConfigWithPathAndAWS("nonexistent.yaml")
	if err == nil {
		t.Fatal("LoadConfigWithPathAndAWS(nonexistent) expected error")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("LoadConfigWithPathAndAWS error = %q, want to contain 'config file not found'", err.Error())
	}
}
