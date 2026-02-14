package utils

import "strings"

// IsCredentialError checks if an error is related to AWS credentials/authentication
func IsCredentialError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Check for common credential-related error patterns
	credentialPatterns := []string{
		"failed to refresh cached credentials",
		"refresh cached SSO token failed",
		"unable to refresh SSO token",
		"InvalidGrantException",
		"ExpiredTokenException",
		"UnauthorizedOperation",
		"AccessDenied",
		"InvalidClientTokenId",
		"SignatureDoesNotMatch",
		"InvalidAccessKeyId",
		"ExpiredToken",
		"TokenRefreshRequired",
		"AWS credentials not available or expired",
	}

	for _, pattern := range credentialPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
