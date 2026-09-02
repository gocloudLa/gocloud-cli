package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gocloud-cli/internal/githubauth"
	"gocloud-cli/internal/testutils"
)

// withFakeGitHubRunner injects fake into newGitHubRunner for the duration of the test and
// restores the original constructor via t.Cleanup, so no test ever spawns a real `gh` process.
func withFakeGitHubRunner(t *testing.T, fake *githubauth.FakeRunner) {
	t.Helper()
	original := newGitHubRunner
	newGitHubRunner = func() githubauth.Runner { return fake }
	t.Cleanup(func() { newGitHubRunner = original })
}

const githubSSOConfigYAML = `
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
  github_sso:
    organization: "gocloud-la"
`

const githubSSOConfigYAMLNoOrgCheck = `
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
`

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "gocloud.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write gocloud.yaml: %v", err)
	}
}

// --- --provider flag validation (before any side effect) ---

func TestSSOProviderFlag_InvalidValueRejectedBeforeSideEffects(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAML)

	err = runRootCmd([]string{"sso", "setup", "--config", "gocloud.yaml", "--provider", "bitbucket"})
	if err == nil {
		t.Fatalf("expected error for invalid --provider value, got nil")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error = %q, want it to mention 'unknown provider'", err.Error())
	}

	if _, statErr := os.Stat(filepath.Join(tempDir, ".aws")); !os.IsNotExist(statErr) {
		t.Errorf(".aws directory was created despite invalid --provider value: validation must run before any side effect")
	}
}

// --- AWS-only backward-compatibility regression ---

func TestSSOSetup_ExplicitProviderAWS_MatchesDefaultBehavior(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)

	const cfgYAML = `
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
  aws_sso:
    start_url: "https://example.awsapps.com/start"
    region: "us-east-1"
    role_name: "AdministratorAccess"
  environments:
    dev:
      name: "Development"
      dir_name: "dev"
      aws_account: "123456789012"
`
	writeConfig(t, tempDir, cfgYAML)

	if err := runRootCmd([]string{"sso", "setup", "--config", "gocloud.yaml", "--provider", "aws"}); err != nil {
		t.Fatalf("SSOSetup() with --provider aws expected no error but got: %v", err)
	}

	awsConfigContent, rerr := os.ReadFile(filepath.Join(tempDir, ".aws", "config"))
	if rerr != nil {
		t.Fatalf("Failed to read generated .aws/config: %v", rerr)
	}
	if !strings.Contains(string(awsConfigContent), "[profile test-client-dev]") {
		t.Errorf(".aws/config = %q, want it to contain profile 'test-client-dev'", awsConfigContent)
	}
}

// --- GitHub-inclusive `sso setup` behavior ---

// TestSSOSetup_GitHub_CreatesScopedConfigDir proves `sso setup --provider github` creates a
// project-local .gh directory with a `.gitignore` (`*`) — the GitHub-side analog of .aws/config
// + .aws/.gitignore — so a later `gocloud sso login --provider github` session stays scoped to
// this project directory instead of writing into the shared ~/.config/gh.
func TestSSOSetup_GitHub_CreatesScopedConfigDir(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAML)

	if err := runRootCmd([]string{"sso", "setup", "--config", "gocloud.yaml", "--provider", "github"}); err != nil {
		t.Fatalf("SSOSetup() with --provider github expected no error but got: %v", err)
	}

	gitignore, rerr := os.ReadFile(filepath.Join(tempDir, ".gh", ".gitignore"))
	if rerr != nil {
		t.Fatalf("Failed to read generated .gh/.gitignore: %v", rerr)
	}
	if strings.TrimSpace(string(gitignore)) != "*" {
		t.Errorf(".gh/.gitignore = %q, want %q", gitignore, "*\n")
	}
}

// --- GitHub-inclusive `sso verify` behavior ---

func TestSSOVerify_GitHub_OrganizationMatch(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAML)

	fake := &githubauth.FakeRunner{
		AvailableResult: true,
		StatusResult: "  Logged in to github.com as octocat (oauth_token)\n" +
			"  Token scopes: 'read:org', 'repo'\n" +
			"GOCLOUD_GH_ORGS_BEGIN\ngocloud-la\nGOCLOUD_GH_ORGS_END\n",
	}
	withFakeGitHubRunner(t, fake)

	if err := runRootCmd([]string{"sso", "verify", "--config", "gocloud.yaml", "--provider", "github"}); err != nil {
		t.Fatalf("expected zero exit (nil error) for a confirmed org match, got: %v", err)
	}
	if !fake.StatusCalled {
		t.Errorf("expected the injected FakeRunner.Status to be called")
	}
}

// TestSSOVerify_GitHub_OrganizationMismatchIsSoft reports mismatch but exits 0 — same soft
// reporting contract as AWS verifyProfile account-mismatch (print ❌, do not fail the command).
func TestSSOVerify_GitHub_OrganizationMismatchIsSoft(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAML)

	fake := &githubauth.FakeRunner{
		AvailableResult: true,
		StatusResult: "  Logged in to github.com as octocat (oauth_token)\n" +
			"  Token scopes: 'read:org', 'repo'\n" +
			"GOCLOUD_GH_ORGS_BEGIN\nsome-other-org\nGOCLOUD_GH_ORGS_END\n",
	}
	withFakeGitHubRunner(t, fake)

	if err := runRootCmd([]string{"sso", "verify", "--config", "gocloud.yaml", "--provider", "github"}); err != nil {
		t.Fatalf("expected soft exit (nil error) for org mismatch, got: %v", err)
	}
}

// TestSSOVerify_GitHub_IndeterminateIsSoft reports indeterminate membership but exits 0 — same
// soft contract as AWS verify (status is printed; the command does not fail).
func TestSSOVerify_GitHub_IndeterminateIsSoft(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAML)

	// No "Token scopes" line and no orgs block => Organizations is empty => Indeterminate, not Mismatch.
	fake := &githubauth.FakeRunner{
		AvailableResult: true,
		StatusResult:    "  Logged in to github.com as octocat (oauth_token)\n",
	}
	withFakeGitHubRunner(t, fake)

	if err := runRootCmd([]string{"sso", "verify", "--config", "gocloud.yaml", "--provider", "github"}); err != nil {
		t.Fatalf("expected soft exit (nil error) when org membership cannot be confirmed, got: %v", err)
	}
}

func TestSSOVerify_GitHub_NotLoggedInIsSoft(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAML)

	fake := &githubauth.FakeRunner{
		AvailableResult: true,
		StatusResult:    "You are not logged into any GitHub hosts\n",
	}
	withFakeGitHubRunner(t, fake)

	if err := runRootCmd([]string{"sso", "verify", "--config", "gocloud.yaml", "--provider", "github"}); err != nil {
		t.Fatalf("expected soft exit (nil error) when not logged in, got: %v", err)
	}
}

func TestSSOVerify_GitHub_OrganizationAbsentSkipsCheckEntirely(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAMLNoOrgCheck)

	fake := &githubauth.FakeRunner{
		AvailableResult: true,
		StatusResult: "  Logged in to github.com as octocat (oauth_token)\n" +
			"GOCLOUD_GH_ORGS_BEGIN\nsome-other-org\nGOCLOUD_GH_ORGS_END\n",
	}
	withFakeGitHubRunner(t, fake)

	// --provider github is requested explicitly but config declares no github_sso block at all,
	// so github is not a declared provider: resolveSSOProviders intersects to empty and this must
	// fail with a clear configuration error rather than silently doing nothing.
	err = runRootCmd([]string{"sso", "verify", "--config", "gocloud.yaml", "--provider", "github"})
	if err == nil {
		t.Fatalf("expected an error: --provider github was requested but infrastructure.github_sso is not configured")
	}
}

// --- GitHub-inclusive `sso login` ---

func TestSSOLogin_GitHubOnly_CallsInjectedRunner(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAML)

	// Mirrors AWS: `sso login` requires `sso setup` to have run first (see
	// TestSSOLogin_GitHub_WithoutSetupFailsWithSetupHint below for the unmet-precondition case).
	if err := os.MkdirAll(filepath.Join(tempDir, ".gh"), 0755); err != nil {
		t.Fatalf("failed to create .gh directory: %v", err)
	}

	fake := &githubauth.FakeRunner{AvailableResult: true}
	withFakeGitHubRunner(t, fake)

	if err := runRootCmd([]string{"sso", "login", "--config", "gocloud.yaml", "--provider", "github"}); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !fake.LoginCalled {
		t.Errorf("expected the injected FakeRunner.Login to be called")
	}
}

// TestSSOLogin_GitHub_WithoutSetupFailsWithSetupHint proves `sso login --provider github` refuses
// to run before `sso setup` has created .gh — mirroring runSSOLoginAWS's ".aws/config file not
// found. Run 'gocloud sso setup' first" precondition. Without this check, login would write gh
// state into githubConfigDir with no .gitignore protecting it (the directory only gets created,
// with its .gitignore, by `sso setup`).
func TestSSOLogin_GitHub_WithoutSetupFailsWithSetupHint(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)
	writeConfig(t, tempDir, githubSSOConfigYAML)

	fake := &githubauth.FakeRunner{AvailableResult: true}
	withFakeGitHubRunner(t, fake)

	err = runRootCmd([]string{"sso", "login", "--config", "gocloud.yaml", "--provider", "github"})
	if err == nil {
		t.Fatalf("expected an error when .gh does not exist, got nil")
	}
	if !strings.Contains(err.Error(), "gocloud sso setup") {
		t.Errorf("error = %q, want it to hint at running 'gocloud sso setup' first", err.Error())
	}
	if fake.LoginCalled {
		t.Errorf("FakeRunner.Login must not be called when the .gh precondition is unmet")
	}
}

// TestSSOLogin_BothProviders_GitHubStillRunsWhenAWSFails proves a combined login does not
// abort the process on AWS failure (no os.Exit): GitHub login still runs, and the command
// returns the AWS error afterward.
func TestSSOLogin_BothProviders_GitHubStillRunsWhenAWSFails(t *testing.T) {
	tempDir, err := testutils.CreateTempDir()
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = testutils.CleanupTempDir(tempDir) }()
	chdirTemp(t, tempDir)

	const cfgYAML = `
infrastructure:
  client: "test-client"
  company: "gcl"
  region: "us-east-1"
  version: "v1.0.0"
  github_sso:
    organization: "gocloud-la"
  aws_sso:
    start_url: "https://example.awsapps.com/start"
    region: "us-east-1"
    role_name: "AdministratorAccess"
  environments:
    dev:
      name: "Development"
      dir_name: "dev"
      aws_account: "123456789012"
`
	writeConfig(t, tempDir, cfgYAML)

	// GitHub setup present; AWS .aws/config intentionally absent so runSSOLoginAWS fails fast.
	if err := os.MkdirAll(filepath.Join(tempDir, ".gh"), 0755); err != nil {
		t.Fatalf("failed to create .gh directory: %v", err)
	}

	fake := &githubauth.FakeRunner{AvailableResult: true}
	withFakeGitHubRunner(t, fake)

	err = runRootCmd([]string{"sso", "login", "--config", "gocloud.yaml"})
	if err == nil {
		t.Fatalf("expected AWS login error (missing AWS CLI or .aws/config), got nil")
	}
	if !fake.LoginCalled {
		t.Errorf("expected GitHub login to run even after AWS login failed (no os.Exit abort); aws err was: %v", err)
	}
}
