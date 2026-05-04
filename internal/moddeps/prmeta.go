package moddeps

import (
	"context"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var (
	upstreamReleaseChoreRe = regexp.MustCompile(`(?i)^chore(\([^)]*\))?\s*:\s*release\b`)
	conventionalRe         = regexp.MustCompile(`^(?i)(?P<type>feat|fix|perf|refactor|docs|style|test|build|ci|chore|revert)(?P<scope>\([^)]+\))?(?P<bang>!)?:`)
	prRefSuffixRe          = regexp.MustCompile(`\s*\(#\d+\)\s*$`)
)

func upstreamSubjectKept(first string) bool {
	if first == "" {
		return false
	}
	s := strings.TrimSpace(first)
	if strings.HasPrefix(strings.ToLower(s), "merge branch") {
		return false
	}
	if upstreamReleaseChoreRe.MatchString(s) {
		return false
	}
	return true
}

func parseGitHubRepo(sourceURL string) (owner, repo string, ok bool) {
	if sourceURL == "" || !strings.Contains(sourceURL, "github.com") {
		return "", "", false
	}
	u := sourceURL
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) >= 2 {
		return parts[0], parts[1], true
	}
	return "", "", false
}

// UpstreamCommitTitles resolves the module GitHub repo from registry metadata and lists compare subjects.
func (c *Client) UpstreamCommitTitles(ctx context.Context, moduleSource, current, latest string) []string {
	parts := strings.Split(moduleSource, "/")
	if len(parts) != 3 {
		return nil
	}
	detail := c.GetModuleVersionDetail(ctx, parts[0], parts[1], parts[2], latest)
	if detail == nil {
		return nil
	}
	srcURL, _ := detail["source"].(string)
	ghOwner, ghRepo, ok := parseGitHubRepo(srcURL)
	if !ok {
		return nil
	}
	return c.GitHubCompareSubjects(ctx, ghOwner, ghRepo, current, latest)
}

var scopeLayerAlias = map[string]string{"base": "foundation"}

func moduleMiddleName(source string) string {
	parts := strings.Split(source, "/")
	if len(parts) == 3 {
		return parts[1]
	}
	return strings.ReplaceAll(source, "/", "-")
}

func prScopeFromPaths(source string, paths []string) string {
	middle := moduleMiddleName(source)
	if len(paths) == 0 {
		return "deps/" + middle
	}
	first := strings.Trim(strings.ReplaceAll(paths[0], "\\", "/"), "/")
	seg := strings.Split(first, "/")
	if len(seg) >= 2 && (seg[0] == "modules" || seg[0] == "examples") {
		layer := seg[1]
		if alt, ok := scopeLayerAlias[seg[1]]; ok {
			layer = alt
		}
		return layer + "/" + middle
	}
	if len(seg) >= 1 {
		return seg[0] + "/" + middle
	}
	return "deps/" + middle
}

var bangConvRe = regexp.MustCompile(`(?i)^(feat|fix|perf|refactor|docs|style|test|build|ci|chore|revert)(\([^)]+\))?!:`)

func upstreamIndicatesBreaking(subjects []string) bool {
	for _, s := range subjects {
		st := strings.TrimSpace(s)
		if bangConvRe.MatchString(st) {
			return true
		}
		if strings.Contains(strings.ToLower(st), "breaking change") {
			return true
		}
	}
	return false
}

func commitBangMatch(subject string) bool {
	return bangConvRe.MatchString(strings.TrimSpace(subject))
}

func commitKindAndBreaking(current, latest string, subjects []string) (kind string, breaking bool) {
	level := SemverBumpLevel(current, latest)
	breaking = level == "major" || upstreamIndicatesBreaking(subjects)
	if breaking {
		return "feat", true
	}
	if level == "minor" {
		return "feat", false
	}
	if level == "patch" {
		return "fix", false
	}
	return "chore", false
}

func cleanHeadlineForTitle(headline string) string {
	s := strings.TrimSpace(headline)
	for {
		t := strings.TrimSpace(prRefSuffixRe.ReplaceAllString(s, ""))
		if t == s {
			break
		}
		s = t
	}
	return s
}

func firstLinePreferring(subjects []string, commitType *string) string {
	for _, s := range subjects {
		st := strings.TrimSpace(s)
		sm := conventionalRe.FindStringSubmatch(st)
		if len(sm) < 2 {
			continue
		}
		typ := strings.ToLower(sm[1])
		if commitType != nil && typ != *commitType {
			continue
		}
		prefix := sm[0]
		rest := strings.TrimSpace(st[len(prefix):])
		if rest != "" {
			return truncateRunes(rest, 200)
		}
		return truncateRunes(st, 200)
	}
	return ""
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func bestTitleHeadline(subjects []string, bumpLevel string) string {
	if len(subjects) == 0 {
		return "update module pin"
	}
	fix := "fix"
	feat := "feat"
	if bumpLevel == "major" {
		if hit := firstLinePreferring(subjects, &feat); hit != "" {
			return hit
		}
	}
	if bumpLevel == "patch" {
		if hit := firstLinePreferring(subjects, &fix); hit != "" {
			return hit
		}
	}
	if bumpLevel == "minor" {
		if hit := firstLinePreferring(subjects, &feat); hit != "" {
			return hit
		}
	}
	if hit := firstLinePreferring(subjects, nil); hit != "" {
		return hit
	}
	return truncateRunes(strings.TrimSpace(strings.Split(subjects[0], "\n")[0]), 200)
}

func buildSquashTitle(kind string, breaking bool, scope, headline, current, latest string, maxLen int) string {
	headline = cleanHeadlineForTitle(headline)
	ver := "(" + current + "→" + latest + ") "
	var prefixPart string
	if breaking && kind == "feat" {
		prefixPart = "feat(" + scope + ")!"
	} else if breaking {
		prefixPart = kind + "(" + scope + ")!"
	} else {
		prefixPart = kind + "(" + scope + ")"
	}
	fixed := prefixPart + ": " + ver
	core := fixed + headline
	if len(core) <= maxLen {
		return core
	}
	room := maxLen - len(fixed)
	if room < 12 {
		if len(fixed) > maxLen {
			return fixed[:maxLen]
		}
		return fixed
	}
	trimmed := headline
	if len(trimmed) > room {
		trimmed = strings.TrimRight(trimmed[:room], ".,; ")
		if len(trimmed) < len(headline) {
			trimmed += "…"
		}
	}
	out := fixed + trimmed
	if len(out) > maxLen {
		return out[:maxLen]
	}
	return out
}

// PRMeta holds suggested squash title, PR body, and marker comment (matches Python build_pr_meta).
type PRMeta struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Marker string `json:"marker"`
}

// BuildPRMeta builds squash title/body/marker for a module bump PR.
func (c *Client) BuildPRMeta(ctx context.Context, moduleSource, current, latest string, paths []string, titles []string) PRMeta {
	parts := strings.Split(moduleSource, "/")
	sortedPaths := append([]string(nil), paths...)
	sort.Strings(sortedPaths)

	marker := "<!-- terraform-deps:" + moduleSource + "|" + current + "|" + latest + " -->"
	var bodyLines []string
	bodyLines = append(bodyLines, marker, "")
	bodyLines = append(bodyLines, "Bump Terraform registry module **`"+moduleSource+"`** from **`"+current+"`** → **`"+latest+"`**.", "")
	bodyLines = append(bodyLines, "### Files", "")
	for _, p := range sortedPaths {
		bodyLines = append(bodyLines, "- `"+p+"`")
	}
	bodyLines = append(bodyLines, "", "### Upstream commits", "")

	if len(parts) == 3 && len(titles) > 0 {
		for i, s := range titles {
			if i >= 30 {
				break
			}
			bodyLines = append(bodyLines, "- "+s)
		}
	} else if len(parts) == 3 {
		detail := c.GetModuleVersionDetail(ctx, parts[0], parts[1], parts[2], latest)
		srcURL := ""
		if detail != nil {
			if s, ok := detail["source"].(string); ok {
				srcURL = s
			}
		}
		if owner, repo, ok := parseGitHubRepo(srcURL); ok {
			bodyLines = append(bodyLines,
				"_No substantive commits to list via [compare](https://github.com/"+owner+"/"+repo+"/compare) "+
					"(tags may differ from registry, or only automated `chore: release` commits in range)._")
		} else {
			bodyLines = append(bodyLines, "_Could not resolve GitHub source from registry._")
		}
	} else {
		bodyLines = append(bodyLines, "_Could not resolve module source._")
	}

	scope := prScopeFromPaths(moduleSource, sortedPaths)
	bumpLevel := SemverBumpLevel(current, latest)
	breakingUpstream := false
	for _, s := range titles {
		if commitBangMatch(s) || strings.Contains(strings.ToLower(s), "breaking change") {
			breakingUpstream = true
			break
		}
	}
	hlLevel := bumpLevel
	if breakingUpstream && bumpLevel == "patch" {
		hlLevel = "minor"
	}
	headline := bestTitleHeadline(titles, hlLevel)
	kind, breaking := commitKindAndBreaking(current, latest, titles)
	title := buildSquashTitle(kind, breaking, scope, headline, current, latest, 250)

	return PRMeta{Title: title, Body: strings.Join(bodyLines, "\n"), Marker: marker}
}
