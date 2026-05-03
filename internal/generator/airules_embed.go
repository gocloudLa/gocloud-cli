package generator

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gocloud-cli/internal/logger"
)

// Canonical editor bundles live under FILES_IN ROOT in the repo; run `make sync-airules`
// to refresh internal/generator/embedded_airules before release.

//go:embed all:embedded_airules
var airulesEmbeddedFS embed.FS

const airulesEmbedRoot = "embedded_airules"

func (pg *ProjectGenerator) generateAirulesFromEmbed() (AirulesBundleSummary, error) {
	var sum AirulesBundleSummary
	logger.Info("Generating airules bundle (.cursor / .kiro)")
	if !IsAirulesGenerationEnabledForConfig(pg.config) {
		sum.Disabled = true
		logger.Info("Airules bundle skipped (infrastructure.enable_airules is false)")
		return sum, nil
	}

	err := fs.WalkDir(airulesEmbeddedFS, airulesEmbedRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(airulesEmbedRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := airulesEmbeddedFS.ReadFile(path)
		if err != nil {
			return err
		}
		outPath := filepath.Join(pg.workingDir, filepath.FromSlash(rel))
		r, werr := pg.writeManagedNonMainTextFile(outPath, string(data), true)
		if werr != nil {
			return werr
		}
		switch r {
		case ManagedResultCreated:
			sum.Created++
		case ManagedResultUpdated:
			sum.Updated++
		case ManagedResultUnchanged:
			sum.Unchanged++
		case ManagedResultSkippedUser:
			sum.Skipped++
		default:
			// ManagedResultDisabled should not occur for normal bundle paths
		}
		return nil
	})
	if err != nil {
		return sum, err
	}
	logger.Info("%s", formatAirulesBundleCompletionLog(sum))
	return sum, nil
}

func formatAirulesBundleCompletionLog(s AirulesBundleSummary) string {
	n := s.Created + s.Updated + s.Unchanged + s.Skipped
	if n == 0 {
		return "Airules bundle completed (no files)"
	}
	var parts []string
	if s.Created > 0 {
		parts = append(parts, fmt.Sprintf("%d created", s.Created))
	}
	if s.Updated > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", s.Updated))
	}
	if s.Unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", s.Unchanged))
	}
	if s.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", s.Skipped))
	}
	return fmt.Sprintf("Airules bundle completed (%d files: %s)", n, strings.Join(parts, ", "))
}
