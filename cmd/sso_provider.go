package cmd

import (
	"fmt"
	"sort"
	"strings"

	"gocloud-cli/internal/models"
)

// knownSSOProviders are the individual provider names --provider recognizes.
var knownSSOProviders = map[string]bool{"aws": true, "github": true}

// allSSOProviders is what "all" expands to.
var allSSOProviders = []string{"aws", "github"}

// parseProviderFlag validates and normalizes the --provider flag value.
//
// Domain: "aws", "github", "aws,github" (order-independent), or "all" (alias for every known
// provider). An empty value returns (nil, nil), meaning "unrestricted": defer entirely to
// whatever providers are declared in config. Any other value is a validation error, and this
// function performs no side effects, so it is safe to call from PersistentPreRunE before any
// command logic runs.
func parseProviderFlag(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	if value == "all" {
		out := make([]string, len(allSSOProviders))
		copy(out, allSSOProviders)
		sort.Strings(out)
		return out, nil
	}

	parts := strings.Split(value, ",")
	seen := make(map[string]bool, len(parts))
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("invalid --provider value %q: empty provider name", value)
		}
		if !knownSSOProviders[p] {
			return nil, fmt.Errorf("invalid --provider value %q: unknown provider %q (valid: aws, github, aws,github, all)", value, p)
		}
		if seen[p] {
			return nil, fmt.Errorf("invalid --provider value %q: duplicate provider %q", value, p)
		}
		seen[p] = true
		out = append(out, p)
	}

	sort.Strings(out)
	return out, nil
}

// configDeclaredSSOProviders returns the providers this configuration declares support for.
// "aws" is always declared (it is the pre-existing, always-available provider). "github" is
// declared only when infrastructure.github_sso is present.
func configDeclaredSSOProviders(infra *models.InfrastructureConfig) []string {
	declared := []string{"aws"}
	if infra != nil && infra.GitHubSSO != nil {
		declared = append(declared, "github")
	}
	return declared
}

// resolveSSOProviders computes the effective provider set for a `gocloud sso` invocation:
// config-declared providers intersected with the --provider flag, or the full declared set when
// the flag is empty (unrestricted). flagValue is assumed already validated by parseProviderFlag.
func resolveSSOProviders(declared []string, flagValue string) ([]string, error) {
	flagProviders, err := parseProviderFlag(flagValue)
	if err != nil {
		return nil, err
	}

	if flagProviders == nil {
		out := make([]string, len(declared))
		copy(out, declared)
		sort.Strings(out)
		return out, nil
	}

	declaredSet := make(map[string]bool, len(declared))
	for _, d := range declared {
		declaredSet[d] = true
	}

	var resolved []string
	for _, p := range flagProviders {
		if declaredSet[p] {
			resolved = append(resolved, p)
		}
	}

	sort.Strings(resolved)
	return resolved, nil
}

// isAWSOnly reports whether the resolved provider set is exactly {"aws"} — the critical
// backward-compatibility short-circuit into the pre-existing, byte-for-byte-unchanged AWS code path.
func isAWSOnly(resolved []string) bool {
	return len(resolved) == 1 && resolved[0] == "aws"
}
