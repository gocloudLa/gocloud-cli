package githubauth

import "context"

// Runner abstracts invoking the `gh` CLI so cmd/sso.go can inject a hermetic FakeRunner in
// tests and use the real ExecRunner in production. Never reads/writes GITHUB_TOKEN or GH_TOKEN
// (those belong to internal/moddeps for an unrelated purpose).
type Runner interface {
	// Available reports whether the `gh` binary can be located (e.g. via exec.LookPath).
	Available() bool
	// Login runs GitHub's OAuth device flow directly (no interactive prompts) and loads the
	// resulting token into gh's own credential store.
	Login(ctx context.Context) error
	// Token returns the current `gh` auth token (e.g. `gh auth token`).
	Token(ctx context.Context) (string, error)
	// Status returns the raw combined output of `gh auth status` (plus an appended
	// organizations block, see orgsBeginMarker/orgsEndMarker) for ParseStatus to parse.
	Status(ctx context.Context) (string, error)
}
