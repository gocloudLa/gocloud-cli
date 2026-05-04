package moddeps

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// --- root / working directory ---

// ResolveRoot returns the directory to scan for *.tf files.
// Uses dirFlag when non-empty; otherwise the current working directory.
func ResolveRoot(dirFlag string) (string, error) {
	if strings.TrimSpace(dirFlag) != "" {
		p := filepath.Clean(dirFlag)
		return filepath.Abs(p)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return wd, nil
}

// --- semver (registry version ordering) ---

var preReleaseLetter = regexp.MustCompile(`-[a-zA-Z]`)

// ParseSemver parses x.y.z into a comparable triple; strips pre-release/build suffix after - or +.
func ParseSemver(v string) [3]int {
	v = strings.TrimSpace(v)
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	var out [3]int
	for i := 0; i < 3; i++ {
		if i >= len(parts) {
			break
		}
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			n = 0
		}
		out[i] = n
	}
	return out
}

// CmpVersion returns -1 if current < latest, 0 if equal, 1 if current > latest.
func CmpVersion(current, latest string) int {
	c, l := ParseSemver(current), ParseSemver(latest)
	for i := 0; i < 3; i++ {
		if c[i] < l[i] {
			return -1
		}
		if c[i] > l[i] {
			return 1
		}
	}
	return 0
}

// LatestVersion returns latest semver from list (preferring stable over pre-releases).
func LatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	stable := make([]string, 0, len(versions))
	for _, v := range versions {
		if !preReleaseLetter.MatchString(v) {
			stable = append(stable, v)
		}
	}
	use := stable
	if len(use) == 0 {
		use = append([]string(nil), versions...)
	}
	sort.Slice(use, func(i, j int) bool {
		return CmpVersion(use[i], use[j]) < 0
	})
	return use[len(use)-1]
}

// SemverBumpLevel returns major, minor, patch, or same comparing X.Y.Z (pre-release stripped).
func SemverBumpLevel(current, latest string) string {
	c, l := ParseSemver(current), ParseSemver(latest)
	if CmpVersion(current, latest) >= 0 {
		return "same"
	}
	if l[0] != c[0] {
		return "major"
	}
	if l[1] != c[1] {
		return "minor"
	}
	return "patch"
}

// --- scan *.tf ---

// osReadFile allows tests to stub filesystem reads.
var osReadFile = os.ReadFile

// TFDep is a Terraform dependency pin found in a .tf file (registry module or provider source).
type TFDep struct {
	Source  string
	Version string
	Path    string // relative to scan root
}

var (
	moduleBlockRe           = regexp.MustCompile(`(?s)module\s+"[^"]+"\s*\{([^}]+)\}`)
	moduleSourceVerRe       = regexp.MustCompile(`source\s*=\s*["']([^"']+)["']`)
	moduleVersionRe         = regexp.MustCompile(`version\s*=\s*["']([^"']+)["']`)
	registryModuleSrcRe     = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_.-]+/[a-zA-Z0-9]+$`)
	requiredProvidersHeadRe = regexp.MustCompile(`required_providers\s*\{`)
	providerInnerBlockRe    = regexp.MustCompile(`(\w+)\s*=\s*\{([^}]*)\}`)
)

// findMatchingBrace returns the index of the closing brace that matches the open brace at openIdx,
// skipping braces inside double-quoted strings (handles \" escapes).
func findMatchingBrace(s string, openIdx int) int {
	if openIdx < 0 || openIdx >= len(s) || s[openIdx] != '{' {
		return -1
	}
	depth := 0
	inStr := false
	esc := false
	for i := openIdx; i < len(s); i++ {
		c := s[i]
		if esc {
			esc = false
			continue
		}
		if inStr {
			switch c {
			case '\\':
				esc = true
			case '"':
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// CollectTFDeps scans root recursively for *.tf files (skipping .terraform dirs).
func CollectTFDeps(root string) (modules []TFDep, providers []TFDep, err error) {
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".tf") {
			return nil
		}
		if strings.Contains(path, string(filepath.Separator)+".terraform"+string(filepath.Separator)) {
			return nil
		}
		b, readErr := osReadFile(filepath.Clean(path))
		if readErr != nil {
			return readErr
		}
		text := string(b)
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		for _, m := range moduleBlockRe.FindAllStringSubmatchIndex(text, -1) {
			if len(m) < 4 {
				continue
			}
			block := text[m[2]:m[3]]
			src := moduleSourceVerRe.FindStringSubmatch(block)
			ver := moduleVersionRe.FindStringSubmatch(block)
			if len(src) < 2 || len(ver) < 2 {
				continue
			}
			source := strings.TrimSpace(src[1])
			version := strings.TrimSpace(ver[1])
			if registryModuleSrcRe.MatchString(source) {
				modules = append(modules, TFDep{Source: source, Version: version, Path: rel})
			}
		}

		for _, loc := range requiredProvidersHeadRe.FindAllStringIndex(text, -1) {
			openBrace := loc[1] - 1
			closeBrace := findMatchingBrace(text, openBrace)
			if closeBrace < 0 || closeBrace <= openBrace {
				continue
			}
			block := text[openBrace+1 : closeBrace]
			for _, pb := range providerInnerBlockRe.FindAllStringSubmatch(block, -1) {
				if len(pb) < 3 {
					continue
				}
				inner := pb[2]
				src := moduleSourceVerRe.FindStringSubmatch(inner)
				ver := moduleVersionRe.FindStringSubmatch(inner)
				if len(src) < 2 || len(ver) < 2 {
					continue
				}
				source := strings.TrimSpace(src[1])
				version := strings.TrimSpace(ver[1])
				if strings.Contains(source, "/") {
					providers = append(providers, TFDep{Source: source, Version: version, Path: rel})
				}
			}
		}
		return nil
	})
	return modules, providers, err
}

// --- rewrite version pin in one file ---

// ApplyModuleVersionBump rewrites a version pin for source==source and version==oldVer (double-quoted),
// matching the Python --write-bump regex. Returns number of replacements (0 if none).
func ApplyModuleVersionBump(root, source, oldVer, newVer, fileArg string) (int, error) {
	path := fileArg
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, fileArg)
	}
	path = filepath.Clean(path)
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	pat := regexp.MustCompile(
		`(source\s*=\s*"` + regexp.QuoteMeta(source) + `"\s*\r?\n\s*version\s*=\s*)"` + regexp.QuoteMeta(oldVer) + `"`,
	)
	oldStr := string(b)
	n := len(pat.FindAllString(oldStr, -1))
	newText := pat.ReplaceAllString(oldStr, "$1\""+newVer+`"`)
	if n == 0 {
		return 0, nil
	}
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := os.WriteFile(path, []byte(newText), mode); err != nil {
		return 0, fmt.Errorf("write %s: %w", path, err)
	}
	return n, nil
}
