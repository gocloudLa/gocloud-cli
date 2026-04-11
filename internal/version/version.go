// Package version provides version checking and update logic against GitHub releases.
package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	repoOwner       = "gocloudLa"
	repoName        = "gocloud-cli"
	apiURL          = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
	releasesPageURL = "https://github.com/" + repoOwner + "/" + repoName + "/releases"
)

// Release represents a GitHub release (subset of API response).
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckResult holds the result of a version check.
type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	IsUpToDate     bool
	DownloadURL    string
	AssetName      string
}

// LatestRelease fetches the latest release from GitHub.
func LatestRelease() (*Release, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}
	return &release, nil
}

// NormalizeVersion returns a version string with optional "v" prefix for comparison.
// It returns the form "X.Y.Z" (no leading v).
func NormalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(v, "v") {
		v = v[1:]
	}
	return v
}

// Less returns true if a < b (semver-style comparison).
func Less(a, b string) bool {
	a = NormalizeVersion(a)
	b = NormalizeVersion(b)
	pa := parseParts(a)
	pb := parseParts(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var na, nb int
		if i < len(pa) {
			na, _ = strconv.Atoi(pa[i])
		}
		if i < len(pb) {
			nb, _ = strconv.Atoi(pb[i])
		}
		if na < nb {
			return true
		}
		if na > nb {
			return false
		}
	}
	return false
}

func parseParts(v string) []string {
	var parts []string
	for _, s := range strings.Split(v, ".") {
		// take only leading digits
		i := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i > 0 {
			parts = append(parts, s[:i])
		} else {
			parts = append(parts, "0")
		}
	}
	return parts
}

// AssetNameFor returns the expected asset name for the current platform.
// Pattern: gocloud-{tag}-{goos}-{goarch}[.exe]
func AssetNameFor(tag string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	suffix := ""
	if goos == "windows" {
		suffix = ".exe"
	}
	return fmt.Sprintf("gocloud-%s-%s-%s%s", tag, goos, goarch, suffix)
}

// DownloadURLFor finds the browser_download_url for the given tag and current GOOS/GOARCH.
func DownloadURLFor(release *Release) (name, url string, ok bool) {
	want := AssetNameFor(release.TagName)
	for _, a := range release.Assets {
		if a.Name == want {
			return a.Name, a.BrowserDownloadURL, true
		}
	}
	return "", "", false
}

// Check compares current version with latest release and returns CheckResult.
func Check(currentVersion string) (*CheckResult, error) {
	release, err := LatestRelease()
	if err != nil {
		return nil, err
	}

	cur := NormalizeVersion(currentVersion)
	latest := NormalizeVersion(release.TagName)
	assetName, downloadURL, ok := DownloadURLFor(release)

	result := &CheckResult{
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		IsUpToDate:     !Less(cur, latest),
		DownloadURL:    downloadURL,
		AssetName:      assetName,
	}
	if !ok {
		result.DownloadURL = ""
		result.AssetName = ""
	}
	return result, nil
}

// DownloadAndReplace downloads the binary from url and replaces the current executable.
// On success the caller should exit; the new binary will be used on next run.
func DownloadAndReplace(url string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot get executable path: %w", err)
	}
	// Resolve symlinks so we replace the real binary
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("cannot resolve executable: %w", err)
	}

	dir := filepath.Dir(self)
	tmpFile, err := os.CreateTemp(dir, "gocloud-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	removeTmp := true
	defer func() {
		tmpFile.Close()
		if removeTmp {
			os.Remove(tmpPath)
		}
	}()

	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}
	if err := tmpFile.Chmod(0755); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Replace: rename temp over existing binary (works on Unix while binary is running)
	if err := os.Rename(tmpPath, self); err != nil {
		return fmt.Errorf("replace binary failed; see %s: %w", releasesPageURL, err)
	}
	removeTmp = false
	_ = written
	return nil
}
