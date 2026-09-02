package githubauth

import (
	"errors"
	"testing"
)

func TestVerifyOrganization(t *testing.T) {
	tests := []struct {
		name        string
		status      ParsedStatus
		statusErr   error
		expectedOrg string
		want        VerifyResult
	}{
		{
			name:        "member of expected org is a match",
			status:      ParsedStatus{Parsed: true, Organizations: []string{"gocloud-la", "other-org"}},
			expectedOrg: "gocloud-la",
			want:        Match,
		},
		{
			name:        "confirmed non-member is a mismatch",
			status:      ParsedStatus{Parsed: true, Organizations: []string{"other-org"}},
			expectedOrg: "gocloud-la",
			want:        Mismatch,
		},
		{
			name:        "status error makes it indeterminate even with org data present",
			status:      ParsedStatus{Parsed: true, Organizations: []string{"other-org"}},
			statusErr:   errors.New("gh: network error"),
			expectedOrg: "gocloud-la",
			want:        Indeterminate,
		},
		{
			name:        "unparseable status is indeterminate",
			status:      ParsedStatus{Parsed: false},
			expectedOrg: "gocloud-la",
			want:        Indeterminate,
		},
		{
			name:        "empty organizations list is indeterminate (e.g. missing read:org scope)",
			status:      ParsedStatus{Parsed: true, Organizations: nil},
			expectedOrg: "gocloud-la",
			want:        Indeterminate,
		},
		{
			name:        "empty expected org is indeterminate",
			status:      ParsedStatus{Parsed: true, Organizations: []string{"gocloud-la"}},
			expectedOrg: "",
			want:        Indeterminate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyOrganization(tt.status, tt.statusErr, tt.expectedOrg)
			if got != tt.want {
				t.Errorf("VerifyOrganization() = %v, want %v", got, tt.want)
			}
		})
	}
}
