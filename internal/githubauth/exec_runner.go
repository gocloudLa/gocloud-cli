package githubauth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// ExecRunner is the real, production Runner: it shells out to the `gh` CLI using exec.Command
// with a plain argv slice — never a shell string — so no shell metacharacter in any argument
// (organization name, profile name, etc.) can ever be interpreted by a shell.
//
// HTTPClient, DeviceCodeURL, TokenURL, OpenBrowser, and Out are overridable so tests can run the
// device-flow login hermetically; the zero value of each falls back to the real GitHub endpoints,
// http.DefaultClient, the OS's browser launcher, and os.Stdout used in production.
//
// ConfigDir, when non-empty, is passed to every `gh` subprocess as GH_CONFIG_DIR — the same
// per-project scoping mechanism AWS_CONFIG_FILE gives the AWS profiles this package sits next to
// (see cmd/sso.go), so a `gh` session started in one project directory never leaks into another.
type ExecRunner struct {
	HTTPClient    *http.Client
	DeviceCodeURL string
	TokenURL      string
	OpenBrowser   func(string) error
	Out           io.Writer
	ConfigDir     string
}

// env returns the environment every `gh` subprocess should run with: the process's own
// environment, plus GH_CONFIG_DIR when ConfigDir is set (appended last so it always wins over
// any GH_CONFIG_DIR already present in the parent environment).
func (r *ExecRunner) env() []string {
	if r.ConfigDir == "" {
		return nil
	}
	return append(os.Environ(), "GH_CONFIG_DIR="+r.ConfigDir)
}

// Available reports whether the `gh` binary can be located on PATH.
func (r *ExecRunner) Available() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

func (r *ExecRunner) httpClient() *http.Client {
	if r.HTTPClient != nil {
		return r.HTTPClient
	}
	return http.DefaultClient
}

func (r *ExecRunner) deviceCodeURL() string {
	if r.DeviceCodeURL != "" {
		return r.DeviceCodeURL
	}
	return defaultDeviceCodeURL
}

func (r *ExecRunner) tokenURL() string {
	if r.TokenURL != "" {
		return r.TokenURL
	}
	return defaultTokenURL
}

func (r *ExecRunner) openBrowserFunc() func(string) error {
	if r.OpenBrowser != nil {
		return r.OpenBrowser
	}
	return openBrowser
}

func (r *ExecRunner) out() io.Writer {
	if r.Out != nil {
		return r.Out
	}
	return os.Stdout
}

// Login drives GitHub's OAuth Device Authorization Grant (RFC 8628) directly: it requests a
// device code, opens the browser immediately (no "press Enter" step), and polls for the token.
// The resulting token is then loaded into gh's own credential store non-interactively via
// `gh auth login --with-token` (no "Authenticate Git with your GitHub credentials?" prompt — that
// prompt only fires in gh's interactive mode) followed by `gh auth setup-git`, which configures
// the git credential helper without a prompt. `gh auth token`/`gh auth status` — and Terraform's
// github provider CLI-auth fallback — see the result exactly as if the user had run
// `gh auth login` themselves.
func (r *ExecRunner) Login(ctx context.Context) error {
	client := r.httpClient()

	// "repo read:org gist" are gh's own login minimum scopes (see cli/cli's
	// internal/authflow.minimumScopes) — gh auth login --with-token hard-rejects a token missing
	// "repo". "admin:org" is added on top because this login exists so Terraform's github
	// provider can manage the organization (members, teams, org settings): "read:org" alone lets
	// terraform-provider-github read that state but 403s on any write (e.g. removing a member).
	dc, err := requestDeviceCode(ctx, client, r.deviceCodeURL(), "repo read:org gist admin:org")
	if err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}

	out := r.out()
	_, _ = fmt.Fprintf(out, "First copy your one-time code: %s\n", dc.UserCode)
	_, _ = fmt.Fprintf(out, "Opening %s in your browser...\n", dc.VerificationURI)
	if err := r.openBrowserFunc()(dc.VerificationURI); err != nil {
		_, _ = fmt.Fprintf(out, "Could not open a browser automatically: open %s and enter the code above.\n", dc.VerificationURI)
	}

	token, err := pollForToken(ctx, client, r.tokenURL(), dc.DeviceCode, dc.Interval, dc.ExpiresIn)
	if err != nil {
		return fmt.Errorf("waiting for authorization: %w", err)
	}

	loginCmd := exec.CommandContext(ctx, "gh", "auth", "login",
		"--hostname", "github.com",
		"--git-protocol", "https",
		"--with-token",
	)
	loginCmd.Stdin = strings.NewReader(token)
	loginCmd.Env = r.env()
	var loginErr bytes.Buffer
	loginCmd.Stderr = &loginErr
	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("gh auth login --with-token failed: %w: %s", err, strings.TrimSpace(loginErr.String()))
	}

	setupCmd := exec.CommandContext(ctx, "gh", "auth", "setup-git", "--hostname", "github.com")
	setupCmd.Env = r.env()
	var setupErr bytes.Buffer
	setupCmd.Stderr = &setupErr
	if err := setupCmd.Run(); err != nil {
		return fmt.Errorf("gh auth setup-git failed: %w: %s", err, strings.TrimSpace(setupErr.String()))
	}

	return nil
}

// Token returns the current `gh` auth token.
func (r *ExecRunner) Token(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	cmd.Env = r.env()
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// Status returns the combined output of `gh auth status`, with an appended organizations block
// (see orgsBeginMarker/orgsEndMarker) populated from `gh api user/orgs` when that call succeeds.
// The organization lookup is best-effort: if it fails (e.g. missing read:org scope), the marker
// block is simply omitted, and ParseStatus/VerifyOrganization treat that as indeterminate rather
// than an error.
func (r *ExecRunner) Status(ctx context.Context) (string, error) {
	statusCmd := exec.CommandContext(ctx, "gh", "auth", "status")
	statusCmd.Env = r.env()
	var combined bytes.Buffer
	statusCmd.Stdout = &combined
	statusCmd.Stderr = &combined
	statusErr := statusCmd.Run()

	orgsCmd := exec.CommandContext(ctx, "gh", "api", "user/orgs", "--jq", ".[].login")
	orgsCmd.Env = r.env()
	orgsOut, orgsErr := orgsCmd.Output()
	if orgsErr == nil {
		combined.WriteString("\n")
		combined.WriteString(orgsBeginMarker)
		combined.WriteString("\n")
		combined.Write(orgsOut)
		combined.WriteString(orgsEndMarker)
		combined.WriteString("\n")
	}

	return combined.String(), statusErr
}
