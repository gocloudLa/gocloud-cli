package generator

import (
	"fmt"
	"path/filepath"

	"gocloud-cli/internal/logger"
	"gocloud-cli/internal/utils"
)

// ManagedTextFileResult is the outcome of writing a single managed root text file (not main.tf).
type ManagedTextFileResult int

const (
	// ManagedResultDisabled means the feature flag is off; the write path was not invoked.
	ManagedResultDisabled ManagedTextFileResult = iota
	// ManagedResultCreated means the file did not exist and was written.
	ManagedResultCreated
	// ManagedResultUpdated means the file existed, content differed, and was overwritten.
	ManagedResultUpdated
	// ManagedResultUnchanged means the file existed with identical content.
	ManagedResultUnchanged
	// ManagedResultSkippedUser means the user declined to overwrite an existing different file.
	ManagedResultSkippedUser
)

// AirulesBundleSummary aggregates outcomes for embedded .cursor / .kiro files.
type AirulesBundleSummary struct {
	Disabled  bool
	Created   int
	Updated   int
	Unchanged int
	Skipped   int
}

// writeManagedNonMainTextFile writes a non-main.tf file and reports whether it actually changed.
// Skipped-by-user returns (ManagedResultSkippedUser, nil).
// When quiet is true, per-file INFO logs are suppressed (used for multi-file airules bundle writes).
func (pg *ProjectGenerator) writeManagedNonMainTextFile(path, content string, quiet bool) (ManagedTextFileResult, error) {
	dir := filepath.Dir(path)
	if err := utils.CreateDirectory(dir); err != nil {
		return ManagedResultDisabled, err
	}

	if !utils.FileExists(path) {
		if err := utils.WriteFile(path, content); err != nil {
			return ManagedResultDisabled, err
		}
		return ManagedResultCreated, nil
	}

	existingContent, err := utils.ReadFile(path)
	if err != nil {
		return ManagedResultDisabled, fmt.Errorf("failed to read existing file %s: %w", path, err)
	}

	if existingContent == content {
		return ManagedResultUnchanged, nil
	}

	if !pg.force {
		confirm, err := utils.PromptYesNo(fmt.Sprintf("File '%s' exists and will be updated with new content. Continue? (y/N)", path), false)
		if err != nil {
			return ManagedResultDisabled, fmt.Errorf("failed to get user confirmation: %w", err)
		}
		if !confirm {
			if !quiet {
				logger.Info("File '%s' skipped by user - keeping existing content", path)
			}
			return ManagedResultSkippedUser, nil
		}
	} else {
		if !quiet {
			logger.Info("File '%s' exists and will be updated with new content (--force enabled)", path)
		}
	}

	if err := utils.WriteFile(path, content); err != nil {
		return ManagedResultDisabled, err
	}
	return ManagedResultUpdated, nil
}

// GenerateGitignore writes the root .gitignore when enabled and returns the outcome.
func (pg *ProjectGenerator) GenerateGitignore() (ManagedTextFileResult, error) {
	logger.Info("Generating root .gitignore")
	if !IsGitignoreGenerationEnabledForConfig(pg.config) {
		logger.Info(".gitignore skipped (infrastructure.enable_gitignore is false)")
		return ManagedResultDisabled, nil
	}
	path := filepath.Join(pg.workingDir, ".gitignore")
	r, err := pg.writeManagedNonMainTextFile(path, infrastructureGitignoreContent, false)
	if err != nil {
		return r, err
	}
	switch r {
	case ManagedResultCreated:
		logger.Info(".gitignore generated successfully")
	case ManagedResultUpdated:
		logger.Info(".gitignore updated successfully")
	case ManagedResultUnchanged:
		logger.Info(".gitignore is up to date (no changes)")
	case ManagedResultSkippedUser:
		logger.Info(".gitignore unchanged (overwrite declined)")
	}
	return r, nil
}

// GenerateAirules writes embedded .cursor / .kiro trees when enabled.
func (pg *ProjectGenerator) GenerateAirules() (AirulesBundleSummary, error) {
	return pg.generateAirulesFromEmbed()
}
