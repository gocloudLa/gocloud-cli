package moddeps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultRegistryBase = "https://registry.terraform.io"

// Client calls the Terraform Registry and optionally the GitHub API (for compare / PR hints).
type Client struct {
	RegistryBase string
	HTTPClient   *http.Client
}

func (c *Client) registryBase() string {
	if strings.TrimSpace(c.RegistryBase) != "" {
		return strings.TrimRight(strings.TrimSpace(c.RegistryBase), "/")
	}
	return defaultRegistryBase
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 25 * time.Second}
}

func getJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return err
	}
	return nil
}

// ModuleVersionsResponse mirrors registry /v1/modules/.../versions.
type moduleVersionsResponse struct {
	Modules []struct {
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	} `json:"modules"`
}

// GetModuleVersions fetches module versions from the public registry. Returns nil if 404/private/error (same as Python).
func (c *Client) GetModuleVersions(ctx context.Context, namespace, name, provider string) []string {
	url := fmt.Sprintf("%s/v1/modules/%s/%s/%s/versions", c.registryBase(), namespace, name, provider)
	var data moduleVersionsResponse
	if err := getJSON(ctx, c.httpClient(), url, nil, &data); err != nil {
		return nil
	}
	if len(data.Modules) == 0 {
		return nil
	}
	vers := data.Modules[0].Versions
	out := make([]string, 0, len(vers))
	for _, vv := range vers {
		if vv.Version != "" {
			out = append(out, vv.Version)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type providerResponse struct {
	Version string `json:"version"`
}

// GetProviderLatest returns the latest provider version from the registry, or empty if not found.
func (c *Client) GetProviderLatest(ctx context.Context, namespace, name string) string {
	url := fmt.Sprintf("%s/v1/providers/%s/%s", c.registryBase(), namespace, name)
	var data providerResponse
	if err := getJSON(ctx, c.httpClient(), url, nil, &data); err != nil {
		return ""
	}
	return data.Version
}

// GetModuleVersionDetail returns registry JSON for a specific module version (includes source URL).
func (c *Client) GetModuleVersionDetail(ctx context.Context, namespace, name, provider, version string) map[string]any {
	url := fmt.Sprintf("%s/v1/modules/%s/%s/%s/%s", c.registryBase(), namespace, name, provider, version)
	var raw map[string]any
	if err := getJSON(ctx, c.httpClient(), url, nil, &raw); err != nil {
		return nil
	}
	return raw
}

type githubCompareResponse struct {
	Commits []struct {
		Commit struct {
			Message string `json:"message"`
		} `json:"commit"`
	} `json:"commits"`
}

func githubToken() string {
	if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
		return t
	}
	return strings.TrimSpace(os.Getenv("GH_TOKEN"))
}

// GitHubCompareSubjects returns first-line commit messages between refs (tries v-prefixed tags).
func (c *Client) GitHubCompareSubjects(ctx context.Context, owner, repo, oldVer, newVer string) []string {
	client := c.httpClient()
	headers := map[string]string{
		"Accept":               "application/vnd.github+json",
		"X-GitHub-Api-Version": "2022-11-28",
	}
	if tok := githubToken(); tok != "" {
		headers["Authorization"] = "Bearer " + tok
	}
	var subjects []string
	oldTags := []string{"v" + oldVer, oldVer}
	newTags := []string{"v" + newVer, newVer}
outer:
	for _, old := range oldTags {
		for _, nw := range newTags {
			url := fmt.Sprintf("https://api.github.com/repos/%s/%s/compare/%s...%s", owner, repo, old, nw)
			var data githubCompareResponse
			if err := getJSON(ctx, client, url, headers, &data); err != nil {
				continue
			}
			for _, cm := range data.Commits {
				msg := cm.Commit.Message
				first := strings.TrimSpace(strings.Split(msg, "\n")[0])
				if upstreamSubjectKept(first) {
					subjects = append(subjects, first)
				}
			}
			if len(subjects) > 0 {
				break outer
			}
		}
	}
	return subjects
}
