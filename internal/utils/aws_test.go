package utils

import (
	"errors"
	"testing"
)

func TestIsCredentialError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"non-credential error", errors.New("some other error"), false},
		{"failed to refresh cached credentials", errors.New("failed to refresh cached credentials"), true},
		{"refresh cached SSO token failed", errors.New("refresh cached SSO token failed"), true},
		{"unable to refresh SSO token", errors.New("unable to refresh SSO token"), true},
		{"InvalidGrantException", errors.New("InvalidGrantException"), true},
		{"ExpiredTokenException", errors.New("ExpiredTokenException"), true},
		{"UnauthorizedOperation", errors.New("UnauthorizedOperation"), true},
		{"AccessDenied", errors.New("AccessDenied"), true},
		{"InvalidClientTokenId", errors.New("InvalidClientTokenId"), true},
		{"SignatureDoesNotMatch", errors.New("SignatureDoesNotMatch"), true},
		{"InvalidAccessKeyId", errors.New("InvalidAccessKeyId"), true},
		{"ExpiredToken", errors.New("ExpiredToken"), true},
		{"TokenRefreshRequired", errors.New("TokenRefreshRequired"), true},
		{"AWS credentials not available or expired", errors.New("AWS credentials not available or expired"), true},
		{"error containing pattern", errors.New("something failed to refresh cached credentials here"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCredentialError(tt.err)
			if got != tt.expected {
				t.Errorf("IsCredentialError(%v) = %v, expected %v", tt.err, got, tt.expected)
			}
		})
	}
}
