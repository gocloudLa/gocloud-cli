package githubauth

// VerifyResult is the three-way outcome of checking GitHub organization membership.
type VerifyResult int

const (
	// Indeterminate means membership could not be confirmed or denied (status error, unparseable
	// status, missing read:org scope/empty org list, or no expected organization configured).
	// Whether that is a hard failure or a warning is a caller policy decision, not part of this
	// classifier's contract.
	Indeterminate VerifyResult = iota
	// Match means the account is a confirmed member of the expected organization.
	Match
	// Mismatch means membership was positively checked and the account does NOT belong to the
	// expected organization. Callers decide whether that is a hard failure or soft report
	// (gocloud sso verify treats it as soft, matching AWS account-mismatch).
	Mismatch
)

// VerifyOrganization compares the logged-in account's organizations against expectedOrg.
//
// Mismatch requires ALL of: statusErr == nil, status.Parsed == true, expectedOrg != "",
// len(status.Organizations) > 0, and expectedOrg not present in status.Organizations.
// Anything else (errors, unparseable status, no expected org, no org data available) is
// Indeterminate — never treated as a confirmed mismatch.
func VerifyOrganization(status ParsedStatus, statusErr error, expectedOrg string) VerifyResult {
	if statusErr != nil || !status.Parsed || expectedOrg == "" || len(status.Organizations) == 0 {
		return Indeterminate
	}

	for _, org := range status.Organizations {
		if org == expectedOrg {
			return Match
		}
	}

	return Mismatch
}
