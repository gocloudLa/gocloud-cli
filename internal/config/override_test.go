package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const overrideSample = `
infrastructure:
  client: acme
  company: acme
  region: us-east-1
  version: "1.0.0"
  aws_sso:
    region: us-east-1
    start_url: https://acme.awsapps.com/start#/
    role_name: Admin
  providers:
    use_profiles: true
  backend:
    use_profile: true
  environments:
    sha:
      name: Shared
      aws_account: "111111111111"
      aws_sso:
        role_name: Admin
    prd:
      name: Production
      aws_account: "222222222222"
      providers:
        use_profiles: true
`

func TestApplyYAMLOverrides_SetAndNull(t *testing.T) {
	got, err := ApplyYAMLOverrides([]byte(overrideSample), []string{
		"infrastructure.providers.use_profiles: false",
		"infrastructure.backend.use_profile: false",
		"infrastructure.aws_sso: null",
		"infrastructure.environments.sha.aws_sso: null",
	})
	if err != nil {
		t.Fatalf("ApplyYAMLOverrides() error = %v", err)
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(got, &doc); err != nil {
		t.Fatalf("unmarshal result: %v\n%s", err, got)
	}
	infra := doc["infrastructure"].(map[string]interface{})
	if _, ok := infra["aws_sso"]; ok {
		t.Fatalf("infrastructure.aws_sso still present: %#v", infra["aws_sso"])
	}
	providers := infra["providers"].(map[string]interface{})
	if providers["use_profiles"] != false {
		t.Fatalf("use_profiles = %#v, want false", providers["use_profiles"])
	}
	backend := infra["backend"].(map[string]interface{})
	if backend["use_profile"] != false {
		t.Fatalf("use_profile = %#v, want false", backend["use_profile"])
	}
	envs := infra["environments"].(map[string]interface{})
	sha := envs["sha"].(map[string]interface{})
	if _, ok := sha["aws_sso"]; ok {
		t.Fatalf("sha.aws_sso still present")
	}
	if sha["aws_account"] != "111111111111" {
		t.Fatalf("sha.aws_account = %#v", sha["aws_account"])
	}
	if infra["client"] != "acme" {
		t.Fatalf("client was dropped: %#v", infra["client"])
	}
}

func TestApplyYAMLOverrides_PartialFieldKeepsSiblings(t *testing.T) {
	got, err := ApplyYAMLOverrides([]byte(overrideSample), []string{
		"infrastructure.aws_sso.role_name: ReadOnly",
	})
	if err != nil {
		t.Fatalf("ApplyYAMLOverrides() error = %v", err)
	}
	if !strings.Contains(string(got), "start_url:") {
		t.Fatalf("start_url dropped:\n%s", got)
	}
	if !strings.Contains(string(got), "role_name: ReadOnly") {
		t.Fatalf("role_name not updated:\n%s", got)
	}
}

func TestApplyYAMLOverrides_LaterWinsAndPreservesEnvOrder(t *testing.T) {
	got, err := ApplyYAMLOverrides([]byte(overrideSample), []string{
		"infrastructure.region: eu-west-1",
		"infrastructure.region: us-west-2",
	})
	if err != nil {
		t.Fatalf("ApplyYAMLOverrides() error = %v", err)
	}
	var node yaml.Node
	if err := yaml.Unmarshal(got, &node); err != nil {
		t.Fatalf("unmarshal node: %v", err)
	}
	envs := environmentKeys(t, &node)
	if len(envs) != 2 || envs[0] != "sha" || envs[1] != "prd" {
		t.Fatalf("environment order = %v, want [sha prd]", envs)
	}
	if !strings.Contains(string(got), "region: us-west-2") {
		t.Fatalf("region not updated:\n%s", got)
	}
}

func TestApplyYAMLOverrides_Errors(t *testing.T) {
	tests := []struct {
		name     string
		override string
		want     string
	}{
		{name: "empty", override: "   ", want: "--override is empty"},
		{name: "missing value", override: "infrastructure.region:", want: "missing YAML value"},
		{name: "bad shape", override: "not a path", want: "expected 'dotted.path:"},
		{name: "not a mapping", override: "infrastructure.client.extra: true", want: "not a mapping"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ApplyYAMLOverrides([]byte(overrideSample), []string{tt.override})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestLoadConfigWithValidationAndOverrides_DisablesSSOProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gocloud.yaml")
	if err := os.WriteFile(path, []byte(overrideSample), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, result, err := NewManager().LoadConfigWithValidationAndOverrides(path, []string{
		"infrastructure.providers.use_profiles: false",
		"infrastructure.backend.use_profile: false",
		"infrastructure.aws_sso: null",
		"infrastructure.environments.sha.aws_sso: null",
	})
	if err != nil {
		t.Fatalf("LoadConfigWithValidationAndOverrides() error = %v", err)
	}
	if result == nil || !result.Valid {
		t.Fatalf("validation = %#v, want valid", result)
	}
	if cfg.Infrastructure.AWSSSO != nil {
		t.Fatalf("AWSSSO = %#v, want nil", cfg.Infrastructure.AWSSSO)
	}
	if cfg.Infrastructure.Providers == nil || cfg.Infrastructure.Providers.UseProfiles == nil || *cfg.Infrastructure.Providers.UseProfiles {
		t.Fatalf("global use_profiles = %#v, want false", cfg.Infrastructure.Providers)
	}
	if cfg.Infrastructure.Backend == nil || cfg.Infrastructure.Backend.UseProfile == nil || *cfg.Infrastructure.Backend.UseProfile {
		t.Fatalf("global use_profile = %#v, want false", cfg.Infrastructure.Backend)
	}
	sha := cfg.Infrastructure.Environments["sha"]
	if sha.AWSSSO != nil {
		t.Fatalf("sha AWSSSO = %#v, want nil", sha.AWSSSO)
	}
	prd := cfg.Infrastructure.Environments["prd"]
	if prd.Providers == nil || prd.Providers.UseProfiles == nil || !*prd.Providers.UseProfiles {
		t.Fatalf("prd use_profiles = %#v, want true (lower scope kept)", prd.Providers)
	}
	order := cfg.Infrastructure.GetEnvironmentOrder()
	if len(order) != 2 || order[0] != "sha" || order[1] != "prd" {
		t.Fatalf("environment order = %v, want [sha prd]", order)
	}
}

func environmentKeys(t *testing.T, doc *yaml.Node) []string {
	t.Helper()
	root := doc.Content[0]
	infra := mappingValue(root, "infrastructure")
	if infra == nil {
		t.Fatal("infrastructure missing")
	}
	envs := mappingValue(infra, "environments")
	if envs == nil || envs.Kind != yaml.MappingNode {
		t.Fatal("environments missing")
	}
	var keys []string
	for i := 0; i < len(envs.Content); i += 2 {
		keys = append(keys, envs.Content[i].Value)
	}
	return keys
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
