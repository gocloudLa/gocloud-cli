package githubauth

import (
	"reflect"
	"testing"
)

// TestParseStatus_LoggedInWithScopesAndOrgs covers the happy path: a well-formed `gh auth status`
// block combined with the organizations marker block ExecRunner appends.
func TestParseStatus_LoggedInWithScopesAndOrgs(t *testing.T) {
	raw := "github.com\n" +
		"  ✓ Logged in to github.com as octocat (oauth_token)\n" +
		"  ✓ Git operations for github.com configured to use https protocol.\n" +
		"  ✓ Token: gho_************************************\n" +
		"  ✓ Token scopes: 'gist', 'read:org', 'repo'\n" +
		orgsBeginMarker + "\n" +
		"gocloud-la\n" +
		"another-org\n" +
		orgsEndMarker + "\n"

	got := ParseStatus(raw)

	want := ParsedStatus{
		Parsed:        true,
		LoggedIn:      true,
		Account:       "octocat",
		Scopes:        []string{"gist", "read:org", "repo"},
		Organizations: []string{"gocloud-la", "another-org"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseStatus() = %+v, want %+v", got, want)
	}
}

// TestParseStatus_NotLoggedIn covers gh's "not logged in" output: it IS parseable (Parsed: true)
// but LoggedIn is false and there are no scopes/orgs to report.
func TestParseStatus_NotLoggedIn(t *testing.T) {
	raw := "You are not logged into any GitHub hosts. To log in, run: gh auth login\n"

	got := ParseStatus(raw)

	want := ParsedStatus{
		Parsed:   true,
		LoggedIn: false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseStatus() = %+v, want %+v", got, want)
	}
}

// TestParseStatus_Unparseable covers garbage/empty input: ParseStatus must never error, it just
// reports Parsed: false so callers (VerifyOrganization, scope checks) treat it as indeterminate.
func TestParseStatus_Unparseable(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"empty string", ""},
		{"random garbage", "###not a gh status output###"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStatus(tt.raw)
			if got.Parsed {
				t.Errorf("ParseStatus(%q).Parsed = true, want false", tt.raw)
			}
		})
	}
}

// TestParseStatus_LoggedInWithoutScopesOrOrgs covers a logged-in status missing the scopes line
// and the orgs marker block entirely (e.g. gh version without token scope reporting, or the orgs
// API call failed so ExecRunner never appended the marker block).
func TestParseStatus_LoggedInWithoutScopesOrOrgs(t *testing.T) {
	raw := "github.com\n" +
		"  ✓ Logged in to github.com as gcl-bot (keyring)\n"

	got := ParseStatus(raw)

	want := ParsedStatus{
		Parsed:   true,
		LoggedIn: true,
		Account:  "gcl-bot",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseStatus() = %+v, want %+v", got, want)
	}
}

// TestParseStatus_LoggedInModernAccountFormat covers gh v2.98.0's real output, which reports
// "Logged in to <host> account <account> (...)" instead of the older "as <account>" wording.
func TestParseStatus_LoggedInModernAccountFormat(t *testing.T) {
	raw := "github.com\n" +
		"  ✓ Logged in to github.com account fmidaglia-gocloud (keyring)\n" +
		"  - Active account: true\n" +
		"  - Git operations protocol: https\n" +
		"  - Token: gho_************************************\n" +
		"  - Token scopes: 'gist', 'read:org', 'repo'\n"

	got := ParseStatus(raw)

	want := ParsedStatus{
		Parsed:   true,
		LoggedIn: true,
		Account:  "fmidaglia-gocloud",
		Scopes:   []string{"gist", "read:org", "repo"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseStatus() = %+v, want %+v", got, want)
	}
}
