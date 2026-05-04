package moddeps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// OutdatedModule is one registry module pin that is behind the latest published version.
type OutdatedModule struct {
	Source               string   `json:"source"`
	Current              string   `json:"current"`
	Latest               string   `json:"latest"`
	Paths                []string `json:"paths"`
	UpstreamCommitTitles []string `json:"upstream_commit_titles"`
}

// ModuleLineStatus is one row for plain / JSON human-readable module pins (deduped by source).
type ModuleLineStatus struct {
	Source  string `json:"source"`
	Current string `json:"current"`
	Latest  string `json:"latest,omitempty"`
	Path    string `json:"path"`
	Status  string `json:"status"` // ok, outdated, private (non-registry sources omitted, same as plain output)
}

// ModulesCheckJSON is `gocloud module deps check --json`: same module rows as plain output (no providers).
type ModulesCheckJSON struct {
	Root    string              `json:"root"`
	Modules []ModuleLineStatus  `json:"modules"`
	Summary ModulesCheckSummary `json:"summary"`
}

// ModulesCheckSummary mirrors the plain-text summary line for modules only.
type ModulesCheckSummary struct {
	OutdatedCount int `json:"outdated_count"`
}

// BumpPlanItem is one outdated pin at one file location (--bump-plan emits one item per .tf path).
type BumpPlanItem struct {
	Source               string   `json:"source"`
	Current              string   `json:"current"`
	Latest               string   `json:"latest"`
	Path                 string   `json:"path"`
	Branch               string   `json:"branch"`
	PRTitle              string   `json:"pr_title"`
	PRBody               string   `json:"pr_body"`
	Marker               string   `json:"marker"`
	UpstreamCommitTitles []string `json:"upstream_commit_titles"`
}

// BumpPlan is the JSON envelope for `check --bump-plan`.
type BumpPlan struct {
	Items []BumpPlanItem `json:"items"`
}

// ListOutdatedModules returns outdated registry modules (same logic as Python list_outdated_modules).
func (c *Client) ListOutdatedModules(ctx context.Context, root string) ([]OutdatedModule, error) {
	rawMods, _, err := CollectTFDeps(root)
	if err != nil {
		return nil, err
	}
	grouped := make(map[[2]string][]string)
	for _, d := range rawMods {
		k := [2]string{d.Source, d.Version}
		grouped[k] = append(grouped[k], d.Path)
	}

	var keys [][2]string
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})

	out := make([]OutdatedModule, 0)
	for _, key := range keys {
		source, current := key[0], key[1]
		paths := grouped[key]
		parts := strings.Split(source, "/")
		if len(parts) != 3 {
			continue
		}
		versions := c.GetModuleVersions(ctx, parts[0], parts[1], parts[2])
		if len(versions) == 0 {
			continue
		}
		latest := LatestVersion(versions)
		if latest == "" || CmpVersion(current, latest) >= 0 {
			continue
		}
		titles := c.UpstreamCommitTitles(ctx, source, current, latest)
		set := make(map[string]struct{})
		var uniq []string
		for _, p := range paths {
			if _, ok := set[p]; !ok {
				set[p] = struct{}{}
				uniq = append(uniq, p)
			}
		}
		sort.Strings(uniq)
		out = append(out, OutdatedModule{
			Source:               source,
			Current:              current,
			Latest:               latest,
			Paths:                uniq,
			UpstreamCommitTitles: titles,
		})
	}
	return out, nil
}

// bumpBranch returns a unique branch name per (source, latest, path) so parallel jobs do not collide.
func bumpBranch(source, latest, path string) string {
	slug := strings.ReplaceAll(source, "/", "-")
	sum := sha256.Sum256([]byte(source + "\n" + latest + "\n" + path))
	suf := hex.EncodeToString(sum[:4])
	return fmt.Sprintf("deps/terraform-%s-%s-%s", slug, latest, suf)
}

// BuildBumpPlan emits one item per outdated pin per file (same shape as check --json for `path`).
func (c *Client) BuildBumpPlan(ctx context.Context, root string) (*BumpPlan, error) {
	rows, err := c.ListOutdatedModules(ctx, root)
	if err != nil {
		return nil, err
	}
	items := make([]BumpPlanItem, 0)
	for _, row := range rows {
		titles := append([]string(nil), row.UpstreamCommitTitles...)
		for _, p := range row.Paths {
			meta := c.BuildPRMeta(ctx, row.Source, row.Current, row.Latest, []string{p}, titles)
			items = append(items, BumpPlanItem{
				Source:               row.Source,
				Current:              row.Current,
				Latest:               row.Latest,
				Path:                 p,
				Branch:               bumpBranch(row.Source, row.Latest, p),
				PRTitle:              meta.Title,
				PRBody:               meta.Body,
				Marker:               meta.Marker,
				UpstreamCommitTitles: titles,
			})
		}
	}
	return &BumpPlan{Items: items}, nil
}

// moduleCheckLines builds the same module rows as the plain-text report (Terraform modules section only).
func (c *Client) moduleCheckLines(ctx context.Context, modules []TFDep) (rows []ModuleLineStatus, outdated int) {

	seenMod := make(map[string][2]string) // source -> (version, path)
	for _, d := range modules {
		if prev, ok := seenMod[d.Source]; !ok || prev[0] != d.Version {
			seenMod[d.Source] = [2]string{d.Version, d.Path}
		}
	}

	var sources []string
	for s := range seenMod {
		sources = append(sources, s)
	}
	sort.Strings(sources)

	for _, source := range sources {
		curPath := seenMod[source]
		current, path := curPath[0], curPath[1]
		parts := strings.Split(source, "/")
		if len(parts) != 3 {
			continue
		}
		versions := c.GetModuleVersions(ctx, parts[0], parts[1], parts[2])
		if len(versions) == 0 {
			rows = append(rows, ModuleLineStatus{
				Source: source, Current: current, Path: path, Status: "private",
			})
			continue
		}
		latest := LatestVersion(versions)
		if latest == "" {
			continue
		}
		st := ModuleLineStatus{
			Source: source, Current: current, Latest: latest, Path: path,
		}
		if CmpVersion(current, latest) < 0 {
			st.Status = "outdated"
			outdated++
		} else {
			st.Status = "ok"
		}
		rows = append(rows, st)
	}
	return rows, outdated
}

// WriteHumanReport prints the colored plain report (stdout). Returns outdated module count.
func (c *Client) WriteHumanReport(ctx context.Context, w io.Writer, root string) (outdated int, err error) {
	modules, provs, err := CollectTFDeps(root)
	if err != nil {
		return 0, err
	}

	dim := color.New(color.FgHiBlack).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	modRows, outdated := c.moduleCheckLines(ctx, modules)

	_, _ = fmt.Fprintf(w, "=== Terraform modules ===\n\n")
	for _, row := range modRows {
		switch row.Status {
		case "private":
			_, _ = fmt.Fprintf(w, "%s\n", dim(fmt.Sprintf("  %s @ %s  (in %s) [private/not in public registry]", row.Source, row.Current, row.Path)))
		case "outdated":
			line := fmt.Sprintf("  %s  %s → %s  (in %s)", row.Source, row.Current, row.Latest, row.Path)
			_, _ = fmt.Fprintf(w, "%s %s\n", yellow("OUTDATED"), line)
		case "ok":
			line := fmt.Sprintf("  %s @ %s  (in %s)", row.Source, row.Current, row.Path)
			_, _ = fmt.Fprintf(w, "%s %s\n", green("OK"), line)
		}
	}

	seenProv := make(map[string][2]string)
	for _, d := range provs {
		if _, ok := seenProv[d.Source]; !ok {
			seenProv[d.Source] = [2]string{d.Version, d.Path}
		}
	}

	_, _ = fmt.Fprintf(w, "\n=== Terraform providers (required_providers) ===\n\n")
	var provSources []string
	for s := range seenProv {
		provSources = append(provSources, s)
	}
	sort.Strings(provSources)
	for _, source := range provSources {
		cp := seenProv[source]
		constraint, path := cp[0], cp[1]
		parts := strings.Split(source, "/")
		if len(parts) != 2 {
			continue
		}
		latest := c.GetProviderLatest(ctx, parts[0], parts[1])
		if latest == "" {
			_, _ = fmt.Fprintf(w, "  %s\n", dim(fmt.Sprintf("%s  constraint: %s  (in %s) [not found]", source, constraint, path)))
			continue
		}
		_, _ = fmt.Fprintf(w, "  %s  constraint: %s  |  latest in registry: %s  (in %s)\n", source, constraint, green(latest), path)
	}

	if outdated > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", color.New(color.Bold).Sprintf("Summary: %d module(s) have a newer version available.", outdated))
	} else {
		_, _ = fmt.Fprintf(w, "\n%s\n", color.New(color.Bold).Sprint("Summary: No outdated modules (or only private modules)."))
	}
	return outdated, nil
}

// BuildModulesCheckJSON builds `--json` output: same module lines as plain text (modules section only), no providers.
func (c *Client) BuildModulesCheckJSON(ctx context.Context, root string) (*ModulesCheckJSON, error) {
	modules, _, err := CollectTFDeps(root)
	if err != nil {
		return nil, err
	}
	rows, outdated := c.moduleCheckLines(ctx, modules)
	return &ModulesCheckJSON{
		Root:    root,
		Modules: rows,
		Summary: ModulesCheckSummary{OutdatedCount: outdated},
	}, nil
}

// EncodeJSON writes JSON with indentation (matches Python json.dump indent=2).
func EncodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
