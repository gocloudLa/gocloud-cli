package githubauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// githubCLIClientID is the public OAuth client ID GitHub issues to its own official "GitHub CLI"
// application (see cli/cli's internal/authflow/flow.go). Device-flow client IDs are not secret by
// design (RFC 8628 has no client secret), and reusing this exact ID is intentional: the resulting
// token is a genuine "GitHub CLI" credential, so gh's own commands (auth status/token/setup-git)
// store and present it exactly as if the user had run `gh auth login` themselves — there is no
// separate "gocloud" identity being misrepresented to the GitHub consent screen.
const githubCLIClientID = "178c6fc778ccc68e1d6a"

const (
	defaultDeviceCodeURL = "https://github.com/login/device/code"
	defaultTokenURL      = "https://github.com/login/oauth/access_token"
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	Error       string `json:"error"`
	Interval    int    `json:"interval"` // only present on a "slow_down" response
}

// requestDeviceCode starts a GitHub OAuth Device Authorization Grant (RFC 8628) request.
func requestDeviceCode(ctx context.Context, client *http.Client, endpoint, scope string) (*deviceCodeResponse, error) {
	form := url.Values{"client_id": {githubCLIClientID}, "scope": {scope}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting device code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading device code response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var out deviceCodeResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("parsing device code response: %w", err)
	}
	if out.DeviceCode == "" || out.UserCode == "" || out.VerificationURI == "" {
		return nil, fmt.Errorf("device code response missing required fields: %s", strings.TrimSpace(string(body)))
	}
	return &out, nil
}

// pollForToken polls the token endpoint per RFC 8628 until the user authorizes in the browser,
// the device code expires, or the request is denied.
func pollForToken(ctx context.Context, client *http.Client, endpoint, deviceCode string, interval, expiresIn int) (string, error) {
	if interval < 1 {
		interval = 1
	}
	deadline := time.Now().Add(time.Duration(expiresIn) * time.Second)

	for {
		if expiresIn > 0 && time.Now().After(deadline) {
			return "", errors.New("device code expired before authorization completed")
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}

		form := url.Values{
			"client_id":   {githubCLIClientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("polling for token: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return "", fmt.Errorf("reading token response: %w", readErr)
		}

		var tok tokenResponse
		if err := json.Unmarshal(body, &tok); err != nil {
			return "", fmt.Errorf("parsing token response: %s", strings.TrimSpace(string(body)))
		}

		switch tok.Error {
		case "":
			if tok.AccessToken != "" {
				return tok.AccessToken, nil
			}
			return "", fmt.Errorf("token response missing access_token: %s", strings.TrimSpace(string(body)))
		case "authorization_pending":
			continue
		case "slow_down":
			if tok.Interval > 0 {
				interval = tok.Interval
			} else {
				interval += 5
			}
			continue
		case "expired_token":
			return "", errors.New("device code expired before authorization completed")
		case "access_denied":
			return "", errors.New("authorization was denied")
		default:
			return "", fmt.Errorf("device flow error: %s", tok.Error)
		}
	}
}

// openBrowser opens targetURL in the user's default browser. A failure here is never fatal to the
// caller's login flow — it just means the user has to open the printed URL manually.
func openBrowser(targetURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
	default:
		cmd = exec.Command("xdg-open", targetURL)
	}
	return cmd.Start()
}
