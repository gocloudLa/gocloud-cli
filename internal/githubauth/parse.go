// Package githubauth wraps the `gh` CLI (GitHub CLI) to support the optional GitHub SSO
// provider for `gocloud sso`. Login uses GitHub's OAuth device flow (stdlib HTTP) and then
// loads the token into gh's credential store; status/org checks shell out to `gh`. It never
// reads/writes GITHUB_TOKEN or GH_TOKEN — those belong to internal/moddeps, an unrelated feature.
package githubauth

import (
	"regexp"
	"strings"
)

// orgsBeginMarker/orgsEndMarker delimit the organization-login block that ExecRunner.Status
// appends after `gh auth status` output (one org login per line), so ParseStatus can extract
// them without depending on gh's auth-status text format for org membership (gh does not
// report org membership in `gh auth status` itself).
const (
	orgsBeginMarker = "GOCLOUD_GH_ORGS_BEGIN"
	orgsEndMarker   = "GOCLOUD_GH_ORGS_END"
)

// ParsedStatus is the structured result of parsing raw `gh` status output.
type ParsedStatus struct {
	// Parsed reports whether raw was recognizable gh auth status output at all.
	// False means "unparseable" — callers must treat this as indeterminate, never as an error.
	Parsed bool
	// LoggedIn reports whether gh reported an active session for github.com.
	LoggedIn bool
	// Account is the logged-in username, empty if not logged in or not found.
	Account string
	// Scopes are the OAuth token scopes gh reported, empty if none/unavailable.
	Scopes []string
	// Organizations are the GitHub organization logins the account belongs to, as reported by
	// `gh api user/orgs` (requires the read:org scope) — empty if unavailable.
	Organizations []string
}

var (
	// Matches both the older "as <account>" wording and gh v2.98.0's "account <account>" wording
	// (real-world evidence: "Logged in to github.com account fmidaglia-gocloud (keyring)").
	loggedInAsRe  = regexp.MustCompile(`(?i)Logged in to [^\s]+ (?:as|account) ([^\s(]+)`)
	tokenScopesRe = regexp.MustCompile(`(?i)Token scopes:\s*(.+)`)
	notLoggedInRe = regexp.MustCompile(`(?i)You are not logged (in|into)`)
)

// ParseStatus is a pure function that never errors: unparseable input simply yields
// Parsed: false. Callers (VerifyOrganization, scope-advisory checks) treat that as indeterminate.
func ParseStatus(raw string) ParsedStatus {
	organizations := parseOrgsBlock(raw)

	if notLoggedInRe.MatchString(raw) {
		return ParsedStatus{Parsed: true, LoggedIn: false}
	}

	m := loggedInAsRe.FindStringSubmatch(raw)
	if m == nil {
		if len(organizations) > 0 {
			// Orgs were reported but no recognizable login line — still unparseable as a whole.
			return ParsedStatus{}
		}
		return ParsedStatus{}
	}

	status := ParsedStatus{
		Parsed:        true,
		LoggedIn:      true,
		Account:       m[1],
		Organizations: organizations,
	}

	if sm := tokenScopesRe.FindStringSubmatch(raw); sm != nil {
		status.Scopes = parseScopes(sm[1])
	}

	return status
}

// parseScopes splits a "Token scopes: 'gist', 'read:org', 'repo'" value into trimmed,
// unquoted scope names.
func parseScopes(raw string) []string {
	parts := strings.Split(raw, ",")
	scopes := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "'\"`")
		if p != "" {
			scopes = append(scopes, p)
		}
	}
	if len(scopes) == 0 {
		return nil
	}
	return scopes
}

// parseOrgsBlock extracts organization logins between orgsBeginMarker and orgsEndMarker.
func parseOrgsBlock(raw string) []string {
	begin := strings.Index(raw, orgsBeginMarker)
	if begin < 0 {
		return nil
	}
	rest := raw[begin+len(orgsBeginMarker):]
	end := strings.Index(rest, orgsEndMarker)
	if end < 0 {
		return nil
	}
	block := rest[:end]

	var orgs []string
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			orgs = append(orgs, line)
		}
	}
	return orgs
}
