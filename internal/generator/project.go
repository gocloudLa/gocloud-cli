package generator

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gocloud-cli/internal/logger"
	"gocloud-cli/internal/models"
	"gocloud-cli/internal/utils"
)

// ErrFileSkipped is returned when user chooses to skip a file
var ErrFileSkipped = errors.New("file skipped by user")

// ProjectGenerator generates infrastructure projects
type ProjectGenerator struct {
	config     *models.InfrastructureConfig
	engine     *TemplateEngine
	workingDir string
	force      bool
}

// NewProjectGenerator creates a new project generator
func NewProjectGenerator(config *models.InfrastructureConfig, workingDir string, force bool) *ProjectGenerator {
	return &ProjectGenerator{
		config:     config,
		engine:     NewTemplateEngine(),
		workingDir: workingDir,
		force:      force,
	}
}

// writeFileWithConfirmation writes content to a file with confirmation if content differs
func (pg *ProjectGenerator) writeFileWithConfirmation(path, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := utils.CreateDirectory(dir); err != nil {
		return err
	}

	// Check if file exists and needs confirmation
	if utils.FileExists(path) {
		// Special handling for main.tf files - selective update only
		if filepath.Base(path) == "main.tf" {
			if pg.shouldUpdateMainTf(path, content) {
				// Check if file is empty or has no module content
				existingContent, err := utils.ReadFile(path)
				if err != nil {
					// If we can't read the file, write the complete content
					if err := utils.WriteFile(path, content); err != nil {
						return fmt.Errorf("failed to write main.tf: %w", err)
					}
					logger.Info("File '%s' created with complete content", path)
					return nil
				}

				// If file is empty or has no module content, write complete content
				if strings.TrimSpace(existingContent) == "" || !strings.Contains(existingContent, "module \"") {
					if err := utils.WriteFile(path, content); err != nil {
						return fmt.Errorf("failed to write main.tf: %w", err)
					}
					logger.Info("File '%s' initialized with complete content", path)
					return nil
				}

				// Update only the version lines, preserve all other content
				if err := pg.updateMainTfVersionOnly(path, content); err != nil {
					return fmt.Errorf("failed to update version in main.tf: %w", err)
				}
				logger.Info("File '%s' version updated (content preserved)", path)
				return nil
			} else {
				// Check if legend needs to be added even if version is unchanged
				if err := pg.ensureLegendInMainTfFile(path); err != nil {
					return fmt.Errorf("failed to ensure legend in main.tf: %w", err)
				}
				logger.Info("File '%s' exists and will NOT be modified (version unchanged)", path)
				return ErrFileSkipped
			}
		} else {
			// Read existing content to compare
			existingContent, err := utils.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read existing file %s: %w", path, err)
			}

			// If content is different, ask for confirmation (unless force is enabled)
			if existingContent != content {
				if !pg.force {
					confirm, err := utils.PromptYesNo(fmt.Sprintf("File '%s' exists and will be updated with new content. Continue? (y/N)", path), false)
					if err != nil {
						return fmt.Errorf("failed to get user confirmation: %w", err)
					}
					if !confirm {
						logger.Info("File '%s' skipped by user - keeping existing content", path)
						return ErrFileSkipped
					}
				} else {
					logger.Info("File '%s' exists and will be updated with new content (--force enabled)", path)
				}
			}
		}
	}

	// Write file
	if err := utils.WriteFile(path, content); err != nil {
		return err
	}
	return nil
}

// shouldUpdateMainTf determines if a main.tf file should be updated based on version changes
func (pg *ProjectGenerator) shouldUpdateMainTf(path, newContent string) bool {
	// Read existing content
	existingContent, err := utils.ReadFile(path)
	if err != nil {
		// If we can't read the file, assume it should be updated
		return true
	}

	// If file is empty, it should be updated
	if strings.TrimSpace(existingContent) == "" {
		return true
	}

	// Extract source and version from existing content
	existingSource := pg.extractSourceFromMainTf(existingContent)
	existingVersions := pg.extractAllVersionsFromMainTf(existingContent)

	// Extract source and version from new content
	newSource := pg.extractSourceFromMainTf(newContent)
	newVersion := pg.extractVersionFromMainTf(newContent)

	// LÓGICA SIMPLE: Si existe regex ^  version = ya estamos en modo registry
	hasExistingVersion := len(existingVersions) > 0
	hasNewVersion := newVersion != ""

	// Si ya estamos en modo registry (tiene versiones), solo comparar versiones
	if hasExistingVersion && hasNewVersion {
		// Check if any existing version is different from the target version
		for _, existingVersion := range existingVersions {
			if existingVersion != newVersion {
				logger.Info("Version change detected: %s -> %s", existingVersion, newVersion)
				return true
			}
		}
		return false // Same versions, no update needed
	}

	// Si estamos en modo Git (no tiene versiones) y queremos cambiar a registry
	if !hasExistingVersion && hasNewVersion {
		logger.Info("Source change detected: Git -> registry")
		return true
	}

	// Si estamos en modo registry y queremos cambiar a Git
	if hasExistingVersion && !hasNewVersion {
		logger.Info("Source change detected: registry -> Git")
		return true
	}

	// Si ambos son Git, comparar source
	if !hasExistingVersion && !hasNewVersion {
		if existingSource != newSource {
			logger.Info("Git source change detected: %s -> %s", existingSource, newSource)
			return true
		}
		return false // Same Git source, no update needed
	}

	// Default: no update needed
	return false
}

// extractVersionFromMainTf extracts the version from a main.tf file content
func (pg *ProjectGenerator) extractVersionFromMainTf(content string) string {
	// Look for the version pattern in the module block
	// Pattern: version = "VERSION"
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "version =") {
			// Extract version from version = "VERSION"
			if idx := strings.Index(line, "version ="); idx != -1 {
				versionPart := line[idx+9:] // Skip "version ="
				versionPart = strings.TrimSpace(versionPart)
				// Remove quotes
				if strings.HasPrefix(versionPart, `"`) && strings.HasSuffix(versionPart, `"`) {
					versionPart = versionPart[1 : len(versionPart)-1]
				}
				return strings.TrimSpace(versionPart)
			}
		}
	}
	return ""
}

// extractAllVersionsFromMainTf extracts all versions from a main.tf file content
func (pg *ProjectGenerator) extractAllVersionsFromMainTf(content string) []string {
	var versions []string
	lines := strings.Split(content, "\n")
	versionRegex := regexp.MustCompile(`^  version = ".*"`)
	for _, line := range lines {
		// Don't trim spaces - we need the exact indentation for the regex
		if versionRegex.MatchString(line) {
			// Extract version from version = "VERSION"
			if idx := strings.Index(line, "version ="); idx != -1 {
				versionPart := line[idx+9:] // Skip "version ="
				versionPart = strings.TrimSpace(versionPart)
				// Remove quotes
				if strings.HasPrefix(versionPart, `"`) && strings.HasSuffix(versionPart, `"`) {
					versionPart = versionPart[1 : len(versionPart)-1]
				}
				versions = append(versions, strings.TrimSpace(versionPart))
			}
		}
	}
	return versions
}

// updateMainTfVersionOnly updates only the version line in a main.tf file, preserving all other content
func (pg *ProjectGenerator) updateMainTfVersionOnly(path, newContent string) error {
	// Read existing content
	existingContent, err := utils.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	// Extract new version from the new content
	newVersion := pg.extractVersionFromMainTf(newContent)

	// Check if we're changing from registry (with version) to Git (without version)
	existingVersions := pg.extractAllVersionsFromMainTf(existingContent)
	hasExistingVersion := len(existingVersions) > 0
	hasNewVersion := newVersion != ""

	// If changing from registry to Git (version -> no version), update source lines only
	if hasExistingVersion && !hasNewVersion {
		// Extract source info from new content
		newSource := pg.extractSourceFromMainTf(newContent)
		if newSource == "" {
			return fmt.Errorf("could not extract source from new content")
		}

		// Update only source and version lines, preserve everything else
		updatedContent := pg.updateSourceInContent(existingContent, newSource, "")

		// Add legend if it doesn't exist
		updatedContent = pg.ensureLegendInMainTf(updatedContent)

		// Write the updated content back to file
		if err := utils.WriteFile(path, updatedContent); err != nil {
			return fmt.Errorf("failed to write updated file: %w", err)
		}
		return nil
	}

	// If changing from Git to registry (no version -> version), update source lines only
	if !hasExistingVersion && hasNewVersion {
		// For Git to Registry transition, we need to change source format and add version
		// Pass empty newSource to indicate Registry format, and newVersion for the version line
		updatedContent := pg.updateSourceInContent(existingContent, "", newVersion)

		// Add legend if it doesn't exist
		updatedContent = pg.ensureLegendInMainTf(updatedContent)

		// Write the updated content back to file
		if err := utils.WriteFile(path, updatedContent); err != nil {
			return fmt.Errorf("failed to write updated file: %w", err)
		}
		return nil
	}

	// If both have versions, update only the version line
	if hasNewVersion {
		// Update only the version line in existing content
		updatedContent := pg.updateVersionInContent(existingContent, newVersion)

		// Add legend if it doesn't exist
		updatedContent = pg.ensureLegendInMainTf(updatedContent)

		// Write the updated content back to file
		if err := utils.WriteFile(path, updatedContent); err != nil {
			return fmt.Errorf("failed to write updated file: %w", err)
		}
		return nil
	}

	// If both are Git sources, check if the source changed (branch/tag/commit)
	existingSource := pg.extractSourceFromMainTf(existingContent)
	newSource := pg.extractSourceFromMainTf(newContent)
	if existingSource != newSource {
		// Update only the source line, preserve everything else
		updatedContent := pg.updateSourceInContent(existingContent, newSource, "")

		// Add legend if it doesn't exist
		updatedContent = pg.ensureLegendInMainTf(updatedContent)

		// Write the updated content back to file
		if err := utils.WriteFile(path, updatedContent); err != nil {
			return fmt.Errorf("failed to write updated file: %w", err)
		}
		return nil
	}

	// If neither has version (both Git) and source is the same, no update needed
	return nil
}

// updateVersionInContent updates only module version lines (with 2-space indentation) while preserving everything else
func (pg *ProjectGenerator) updateVersionInContent(content, newVersion string) string {
	lines := strings.Split(content, "\n")
	versionRegex := regexp.MustCompile(`^  version = ".*"`)

	for i, line := range lines {
		// Use regex to match exactly: "  version = "(.*)""
		// This ensures we only match lines that start with exactly 2 spaces, contain "version =", and have quoted values
		if versionRegex.MatchString(line) {
			// Replace the entire line with the new version, preserving the exact format
			lines[i] = `  version = "` + newVersion + `"`
		}
	}

	return strings.Join(lines, "\n")
}

// updateSourceInContent updates source and version lines when changing between registry and Git
func (pg *ProjectGenerator) updateSourceInContent(content string, newSource, newVersion string) string {
	if newSource != "" {
		// MODO GIT: source + eliminar versiones
		// Cambiar source a Git
		content = pg.replaceAllRegex(content, `  source\s*=.*`, `  source = "`+newSource+`"`)
		// Eliminar TODAS las versiones (eliminar la línea completa)
		content = pg.replaceAllRegex(content, `  version\s*=.*\n?`, "")
	} else {
		// MODO REGISTRY: eliminar versiones + source+version
		// 1. PRIMERO: Eliminar TODAS las versiones (eliminar la línea completa)
		content = pg.replaceAllRegex(content, `  version\s*=.*\n?`, "")

		// 2. SEGUNDO: Reemplazar source SIEMPRE por bloque multilinea
		// Necesitamos determinar el layer name para construir el source correcto
		layerName := pg.extractLayerNameFromContent(content)
		content = pg.replaceAllRegex(content, `  source\s*=.*`, `  source  = "gocloudLa/standard-platform/aws//modules/`+layerName+`"`+"\n"+`  version = "`+newVersion+`"`)
	}

	return content
}

// replaceAllRegex replaces all matches of a regex pattern with a replacement string
func (pg *ProjectGenerator) replaceAllRegex(content, pattern, replacement string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(content, replacement)
}

// extractLayerNameFromContent extracts the layer name from the content
func (pg *ProjectGenerator) extractLayerNameFromContent(content string) string {
	// Try to extract from existing source lines
	lines := strings.Split(content, "\n")
	sourceRegex := regexp.MustCompile(`  source\s*=.*`)
	for _, line := range lines {
		if sourceRegex.MatchString(line) {
			// Extract from patterns like:
			// source = "git@github.com:repo.git//modules/base?ref=main"
			// source  = "gocloudLa/standard-platform/aws//modules/base"
			if strings.Contains(line, "//modules/") {
				parts := strings.Split(line, "//modules/")
				if len(parts) > 1 {
					modulePart := strings.Split(parts[1], "?")[0]  // Remove ?ref=main
					modulePart = strings.Split(modulePart, `"`)[0] // Remove trailing quote
					return strings.TrimSpace(modulePart)
				}
			}
		}
	}

	// Fallback: try to determine from file path or default to "base"
	return "base"
}

// extractSourceFromMainTf extracts the source from a main.tf file content
func (pg *ProjectGenerator) extractSourceFromMainTf(content string) string {
	lines := strings.Split(content, "\n")
	sourceRegex := regexp.MustCompile(`^  source\s*=`)
	for _, line := range lines {
		// Match source lines with 1 or 2-space indentation
		if sourceRegex.MatchString(line) {
			// Extract source from source = "SOURCE" or source  = "SOURCE"
			if idx := strings.Index(line, "="); idx != -1 {
				sourcePart := line[idx+1:] // Skip "="
				sourcePart = strings.TrimSpace(sourcePart)
				// Remove quotes
				if strings.HasPrefix(sourcePart, `"`) && strings.HasSuffix(sourcePart, `"`) {
					sourcePart = sourcePart[1 : len(sourcePart)-1]
				}
				return strings.TrimSpace(sourcePart)
			}
		}
	}
	return ""
}

// ensureLegendInMainTf ensures that the GoCloud CLI legend is present at the top of the file
func (pg *ProjectGenerator) ensureLegendInMainTf(content string) string {
	// Check if the legend already exists
	if strings.Contains(content, "This file is generated and maintained by GoCloud CLI") {
		return content
	}

	// Define the new legend
	legend := `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# You CAN edit this file manually to add your custom configuration
# GoCloud CLI will only update the module version when needed
# =============================================================================

`

	// Add the legend at the beginning of the file
	return legend + content
}

// ensureLegendInMainTfFile ensures that the GoCloud CLI legend is present in the file
func (pg *ProjectGenerator) ensureLegendInMainTfFile(path string) error {
	// Read existing content
	existingContent, err := utils.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read existing file: %w", err)
	}

	// Check if legend already exists
	if strings.Contains(existingContent, "This file is generated and maintained by GoCloud CLI") {
		return nil // Legend already exists, no need to modify
	}

	// Add legend to content
	updatedContent := pg.ensureLegendInMainTf(existingContent)

	// Write the updated content back to file
	if err := utils.WriteFile(path, updatedContent); err != nil {
		return fmt.Errorf("failed to write updated file: %w", err)
	}

	logger.Info("File '%s' legend added", path)
	return nil
}

// mapRegionToShortCode maps AWS regions to short codes used in metadata (see models.AWSRegionShortCodes).
func mapRegionToShortCode(region string) string {
	if shortCode, exists := models.AWSRegionShortCodes[region]; exists {
		return shortCode
	}
	return generateRegionCode(region)
}

// generateRegionCode generates a short code from region name as fallback
func generateRegionCode(region string) string {
	// Pattern: us-east-1 -> use1, eu-west-1 -> euw1
	parts := strings.Split(region, "-")
	if len(parts) >= 3 {
		// Take first 2 chars of first part, first char of second part, first char of third part
		first := parts[0]
		second := parts[1]
		third := parts[2]

		if len(first) >= 2 && len(second) >= 1 && len(third) >= 1 {
			return strings.ToLower(first[:2] + second[:1] + third[:1])
		}
	}

	// Ultimate fallback
	return "use1"
}

// CreateProjectStructure creates the complete directory structure
func (pg *ProjectGenerator) CreateProjectStructure() error {
	logger.Info("Creating project structure for client: %s", pg.config.Client)

	// Determine root directory based on workingDir
	var rootDir string
	if pg.workingDir == "." {
		// Use current directory, no subdirectory needed
		rootDir = "."
	} else {
		// Use specified working directory
		rootDir = pg.workingDir
		if err := utils.CreateDirectory(rootDir); err != nil {
			return err
		}
	}

	// Create base and foundation layers for each environment (only if enabled)
	for envName := range pg.config.Environments {
		dirName := pg.getDirectoryName(envName)

		// Create base layer only if enabled
		if pg.shouldGenerateLayer("base", envName) {
			basePath := filepath.Join(rootDir, "base", dirName)
			if err := utils.CreateDirectory(basePath); err != nil {
				return err
			}
		}

		// Create foundation layer only if enabled
		if pg.shouldGenerateLayer("foundation", envName) {
			foundationPath := filepath.Join(rootDir, "foundation", dirName)
			if err := utils.CreateDirectory(foundationPath); err != nil {
				return err
			}
		}
	}

	// Create project directories for each environment
	for envName, env := range pg.config.Environments {
		dirName := pg.getDirectoryName(envName)
		for _, project := range env.Projects {
			projectDirName := models.GetProjectDirectoryName(project)
			projectPath := filepath.Join(rootDir, "project", projectDirName, dirName)
			if err := utils.CreateDirectory(projectPath); err != nil {
				return err
			}
		}
	}

	// Create workload directories for each environment
	for envName, env := range pg.config.Environments {
		dirName := pg.getDirectoryName(envName)
		for _, workload := range env.Workloads {
			workloadDirName := models.GetWorkloadDirectoryName(workload)
			workloadPath := filepath.Join(rootDir, "workload", workloadDirName, dirName)
			if err := utils.CreateDirectory(workloadPath); err != nil {
				return err
			}
		}
	}

	// Create organization directory only if enabled (requires infrastructure.organization.aws_account)
	if pg.isOrganizationLayerEnabled() {
		orgPath := filepath.Join(rootDir, "organization")
		if err := utils.CreateDirectory(orgPath); err != nil {
			return err
		}
	}

	// Create security directory only if enabled (requires infrastructure.security.aws_account)
	if pg.isSecurityLayerEnabled() {
		secPath := filepath.Join(rootDir, "security")
		if err := utils.CreateDirectory(secPath); err != nil {
			return err
		}
	}

	logger.Info("Project structure created successfully")
	return nil
}

// GenerateConfigFiles generates all configuration files
func (pg *ProjectGenerator) GenerateConfigFiles() error {
	logger.Info("Generating configuration files")

	// Generate root configuration files
	if err := pg.generateRootConfigs(); err != nil {
		return err
	}

	// Generate layer-specific files
	if err := pg.generateLayerConfigs(); err != nil {
		return err
	}

	// Generate organization files
	if err := pg.generateOrganizationConfigs(); err != nil {
		return err
	}

	// Generate security files
	if err := pg.generateSecurityConfigs(); err != nil {
		return err
	}

	logger.Info("Configuration files generated successfully")
	return nil
}

// SetupAWSSSO sets up AWS SSO configuration
func (pg *ProjectGenerator) SetupAWSSSO() error {
	logger.Info("Setting up AWS SSO configuration")

	// Create AWS SSO configuration
	ssoConfig := &models.AWSConfig{
		Profiles: make(map[string]string),
		SSOConfig: models.SSOConfig{
			StartURL: fmt.Sprintf("https://%s.awsapps.com/start#/", pg.config.Client),
			Region:   pg.config.Region,
			RoleName: "Admin",
		},
		BackendConfig: models.BackendConfig{
			Bucket:        fmt.Sprintf("%s-shared-s3-backend", pg.config.Company),
			Region:        pg.config.Region,
			DynamoDBTable: fmt.Sprintf("%s-shared-s3-backend", pg.config.Company),
			Encrypt:       true,
		},
	}

	// Generate AWS profiles for each environment (only if SSO is enabled)
	for envName, env := range pg.config.Environments {
		if models.ShouldEnableSSO(env) {
			profileName := fmt.Sprintf("%s-%s", pg.config.Client, envName)
			ssoConfig.Profiles[envName] = profileName
		}
	}

	// TODO: Generate AWS SSO configuration files
	logger.Info("AWS SSO configuration setup completed")
	return nil
}

// CreateSecretsStructure creates the secrets structure
func (pg *ProjectGenerator) CreateSecretsStructure() error {
	logger.Info("Creating secrets structure")

	// TODO: Create SSM Parameter structure
	// This would involve creating the parameter names and initial values
	// for each environment and layer

	logger.Info("Secrets structure created successfully")
	return nil
}

// InitializeGit initializes a git repository
func (pg *ProjectGenerator) InitializeGit() error {
	if pg.workingDir == "" {
		return fmt.Errorf("working directory is required")
	}

	logger.Info("Initializing git repository")

	// Check if .git directory already exists
	gitDir := filepath.Join(pg.workingDir, ".git")
	if utils.DirectoryExists(gitDir) {
		logger.Info("Git repository already exists")
		return nil
	}

	// Create .git directory
	if err := utils.CreateDirectory(gitDir); err != nil {
		return fmt.Errorf("failed to create .git directory: %w", err)
	}

	// Create basic git structure
	objectsDir := filepath.Join(gitDir, "objects")
	if err := utils.CreateDirectory(objectsDir); err != nil {
		return fmt.Errorf("failed to create objects directory: %w", err)
	}

	refsDir := filepath.Join(gitDir, "refs")
	if err := utils.CreateDirectory(refsDir); err != nil {
		return fmt.Errorf("failed to create refs directory: %w", err)
	}

	// Create HEAD file
	headContent := "ref: refs/heads/main\n"
	headPath := filepath.Join(gitDir, "HEAD")
	if err := utils.WriteFile(headPath, headContent); err != nil {
		return fmt.Errorf("failed to create HEAD file: %w", err)
	}

	// Create config file
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
`
	configPath := filepath.Join(gitDir, "config")
	if err := utils.WriteFile(configPath, configContent); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	logger.Info("Git repository initialized successfully")
	return nil
}

// GenerateDocumentation generates project documentation
func (pg *ProjectGenerator) GenerateDocumentation() error {
	if pg.config == nil {
		return fmt.Errorf("config is required")
	}

	logger.Info("Generating README.md")

	// Generate README.md
	readmeContent := pg.generateREADME()
	readmePath := filepath.Join(pg.workingDir, "README.md")
	if err := pg.writeFileWithConfirmation(readmePath, readmeContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("README.md skipped by user")
		} else {
			return err
		}
	}

	logger.Info("README.md generated successfully")
	return nil
}

// generateRootConfigs generates root-level configuration files
func (pg *ProjectGenerator) generateRootConfigs() error {
	// Generate root.hcl (empty)
	rootContent := `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# DO NOT EDIT MANUALLY - Changes will be overwritten on next generation
# =============================================================================
`
	rootPath := filepath.Join(pg.workingDir, "root.hcl")
	if err := pg.writeFileWithConfirmation(rootPath, rootContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("root.hcl skipped by user")
		} else {
			return err
		}
	}

	if IsGitignoreGenerationEnabledForConfig(pg.config) {
		gitignorePath := filepath.Join(pg.workingDir, ".gitignore")
		if err := pg.writeFileWithConfirmation(gitignorePath, infrastructureGitignoreContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info(".gitignore skipped by user")
			} else {
				return err
			}
		}
	} else {
		logger.Info(".gitignore skipped - infrastructure.enable_gitignore is false")
	}

	// Note: terragrunt.hcl in root is no longer needed with the new structure

	return nil
}

// generateLayerConfigs generates layer-specific configuration files
func (pg *ProjectGenerator) generateLayerConfigs() error {
	// Generate files for base and foundation layers
	for envName := range pg.config.Environments {
		// Generate base layer files only if enabled
		if pg.shouldGenerateLayer("base", envName) {
			if err := pg.generateLayerFiles("base", envName); err != nil {
				return err
			}
		} else {
			logger.Info("base/%s skipped - layer disabled", pg.getDirectoryName(envName))
		}

		// Generate foundation layer files only if enabled
		if pg.shouldGenerateLayer("foundation", envName) {
			if err := pg.generateLayerFiles("foundation", envName); err != nil {
				return err
			}
		} else {
			logger.Info("foundation/%s skipped - layer disabled", pg.getDirectoryName(envName))
		}
	}

	// Generate files for project and workload layers
	for envName, env := range pg.config.Environments {
		// Generate project files
		for _, project := range env.Projects {
			if err := pg.generateProjectFiles("project", project, envName); err != nil {
				return err
			}
		}

		// Generate workload files
		for _, workload := range env.Workloads {
			if err := pg.generateProjectFiles("workload", workload, envName); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateLayerFiles generates files for a specific layer and environment
func (pg *ProjectGenerator) generateLayerFiles(layer string, env string) error {
	dirName := pg.getDirectoryName(env)
	layerPath := filepath.Join(pg.workingDir, layer, dirName)

	// Generate metadata.tf
	metadataData := pg.buildTemplateData(layer, env)
	metadataContent, err := pg.engine.Render("metadata.tf.tpl", metadataData)
	if err != nil {
		return err
	}
	metadataPath := filepath.Join(layerPath, "metadata.tf")
	if err := pg.writeFileWithConfirmation(metadataPath, metadataContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("%s/%s/metadata.tf skipped by user", layer, dirName)
		} else {
			return err
		}
	}

	// Generate terragrunt.hcl only if terragrunt is enabled
	if pg.shouldGenerateTerragrunt(layer, "", env) {
		terragruntData := pg.buildTemplateData(layer, env)
		terragruntContent, err := pg.engine.Render("terragrunt.hcl.tpl", terragruntData)
		if err != nil {
			return err
		}
		terragruntPath := filepath.Join(layerPath, "terragrunt.hcl")
		if err := pg.writeFileWithConfirmation(terragruntPath, terragruntContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("%s/%s/terragrunt.hcl skipped by user", layer, dirName)
			} else {
				return err
			}
		}
	} else {
		// If terragrunt is disabled, remove existing terragrunt.hcl file if it exists
		terragruntPath := filepath.Join(layerPath, "terragrunt.hcl")
		if utils.FileExists(terragruntPath) {
			if err := utils.DeleteFile(terragruntPath); err != nil {
				logger.Error("Failed to delete %s: %v", terragruntPath, err)
			} else {
				logger.Info("%s/%s/terragrunt.hcl removed - terragrunt disabled", layer, dirName)
			}
		} else {
			logger.Info("%s/%s/terragrunt.hcl skipped - terragrunt disabled", layer, dirName)
		}
	}

	// Generate _secrets.tf only if secrets are enabled
	if pg.shouldGenerateSecrets(layer, "", env) {
		secretsData := pg.buildTemplateData(layer, env)
		secretsContent, err := pg.engine.Render("_secrets.tf.tpl", secretsData)
		if err != nil {
			return err
		}
		secretsPath := filepath.Join(layerPath, "_secrets.tf")
		if err := pg.writeFileWithConfirmation(secretsPath, secretsContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("%s/%s/_secrets.tf skipped by user", layer, dirName)
			} else {
				return err
			}
		}
	} else {
		logger.Info("%s/%s/_secrets.tf skipped - secrets disabled", layer, dirName)
	}

	// Generate providers.tf if providers are enabled
	if pg.shouldGenerateProviders(layer, "", env) {
		providersData := pg.buildProviderTemplateData(layer, "", env)
		providersContent, err := pg.engine.Render("providers.tf.tpl", &models.TemplateData{
			Providers: providersData.Providers,
		})
		if err != nil {
			return err
		}
		providersPath := filepath.Join(layerPath, "providers.tf")
		if err := pg.writeFileWithConfirmation(providersPath, providersContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("%s/%s/providers.tf skipped by user", layer, dirName)
			} else {
				return err
			}
		}
	} else {
		logger.Info("%s/%s/providers.tf skipped - providers disabled", layer, dirName)
	}

	// Generate backend.tf if backend is enabled
	if pg.shouldGenerateBackend(layer, "", env) {
		backendData := pg.buildBackendTemplateData(layer, "", env)
		backendContent, err := pg.engine.Render("backend.tf.tpl", &models.TemplateData{
			BackendType:          backendData.Type,
			BackendBucket:        backendData.Bucket,
			BackendKey:           backendData.Key,
			BackendRegion:        backendData.Region,
			BackendDynamoDBTable: backendData.DynamoDBTable,
			BackendEncrypt:       backendData.Encrypt,
			BackendProfile:       backendData.Profile,
			BackendAssumeRole:    backendData.AssumeRole,
		})
		if err != nil {
			return err
		}
		backendPath := filepath.Join(layerPath, "backend.tf")
		if err := pg.writeFileWithConfirmation(backendPath, backendContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("%s/%s/backend.tf skipped by user", layer, dirName)
			} else {
				return err
			}
		}
	} else {
		logger.Info("%s/%s/backend.tf skipped - backend disabled", layer, dirName)
	}

	// Generate main.tf using layer-specific template
	mainData := pg.buildTemplateData(layer, env)
	templateName := fmt.Sprintf("main.tf.%s.tpl", layer)
	mainContent, err := pg.engine.Render(templateName, mainData)
	if err != nil {
		return err
	}
	mainPath := filepath.Join(layerPath, "main.tf")
	if err := pg.writeFileWithConfirmation(mainPath, mainContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("%s/%s/main.tf skipped by user", layer, dirName)
		} else {
			return err
		}
	}

	return nil
}

// generateOrganizationConfigs generates organization-level configuration files
func (pg *ProjectGenerator) generateOrganizationConfigs() error {
	// Organization layer requires infrastructure.organization with aws_account (backend, secrets, SSO)
	if !pg.isOrganizationLayerEnabled() {
		logger.Info("organization layer skipped - disabled or infrastructure.organization.aws_account not set")
		return nil
	}

	// Generate organization files (single files, no environment-specific files)
	orgData := pg.buildTemplateData("organization", "")

	// Generate metadata.tf using the standard metadata template
	metadataContent, err := pg.engine.Render("metadata.tf.tpl", orgData)
	if err != nil {
		return err
	}
	metadataPath := filepath.Join(pg.workingDir, "organization", "metadata.tf")
	if err := pg.writeFileWithConfirmation(metadataPath, metadataContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("organization/metadata.tf skipped by user")
		} else {
			return err
		}
	}

	// Generate main.tf using organization template
	mainContent, err := pg.engine.Render("main.tf.organization.tpl", orgData)
	if err != nil {
		return err
	}
	mainPath := filepath.Join(pg.workingDir, "organization", "main.tf")
	if err := pg.writeFileWithConfirmation(mainPath, mainContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("organization/main.tf skipped by user")
		} else {
			return err
		}
	}

	// Generate organization/terragrunt.hcl (like base: no dependencies) or remove if disabled
	if pg.shouldGenerateTerragrunt("organization", "", "org") {
		terragruntContent, err := pg.engine.Render("terragrunt.hcl.tpl", orgData)
		if err != nil {
			return err
		}
		terragruntPath := filepath.Join(pg.workingDir, "organization", "terragrunt.hcl")
		if err := pg.writeFileWithConfirmation(terragruntPath, terragruntContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("organization/terragrunt.hcl skipped by user")
			} else {
				return err
			}
		}
	} else {
		terragruntPath := filepath.Join(pg.workingDir, "organization", "terragrunt.hcl")
		if utils.FileExists(terragruntPath) {
			if err := utils.DeleteFile(terragruntPath); err != nil {
				logger.Error("Failed to delete %s: %v", terragruntPath, err)
			} else {
				logger.Info("organization/terragrunt.hcl removed - terragrunt disabled")
			}
		} else {
			logger.Info("organization/terragrunt.hcl skipped - terragrunt disabled")
		}
	}

	// Generate organization/_secrets.tf when secrets are enabled (respects infrastructure.organization.secrets)
	if pg.shouldGenerateSecrets("organization", "", "org") {
		secretsContent, err := pg.engine.Render("_secrets.tf.tpl", orgData)
		if err != nil {
			return err
		}
		secretsPath := filepath.Join(pg.workingDir, "organization", "_secrets.tf")
		if err := pg.writeFileWithConfirmation(secretsPath, secretsContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("organization/_secrets.tf skipped by user")
			} else {
				return err
			}
		}
	} else {
		logger.Info("organization/_secrets.tf skipped - secrets disabled")
	}

	// Generate organization/providers.tf when providers are enabled
	if pg.shouldGenerateProviders("organization", "", "org") {
		providersData := pg.buildProviderTemplateData("organization", "", "org")
		providersContent, err := pg.engine.Render("providers.tf.tpl", &models.TemplateData{
			Providers: providersData.Providers,
		})
		if err != nil {
			return err
		}
		providersPath := filepath.Join(pg.workingDir, "organization", "providers.tf")
		if err := pg.writeFileWithConfirmation(providersPath, providersContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("organization/providers.tf skipped by user")
			} else {
				return err
			}
		}
	} else {
		logger.Info("organization/providers.tf skipped - providers disabled")
	}

	// Generate organization/backend.tf when backend is enabled
	if pg.shouldGenerateBackend("organization", "", "org") {
		backendData := pg.buildBackendTemplateData("organization", "", "org")
		backendContent, err := pg.engine.Render("backend.tf.tpl", &models.TemplateData{
			BackendType:          backendData.Type,
			BackendBucket:        backendData.Bucket,
			BackendKey:           backendData.Key,
			BackendRegion:        backendData.Region,
			BackendDynamoDBTable: backendData.DynamoDBTable,
			BackendEncrypt:       backendData.Encrypt,
			BackendProfile:       backendData.Profile,
			BackendAssumeRole:    backendData.AssumeRole,
		})
		if err != nil {
			return err
		}
		backendPath := filepath.Join(pg.workingDir, "organization", "backend.tf")
		if err := pg.writeFileWithConfirmation(backendPath, backendContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("organization/backend.tf skipped by user")
			} else {
				return err
			}
		}
	} else {
		logger.Info("organization/backend.tf skipped - backend disabled")
	}

	return nil
}

// generateSecurityConfigs generates security-level configuration files (global layer, same pattern as organization).
func (pg *ProjectGenerator) generateSecurityConfigs() error {
	if !pg.isSecurityLayerEnabled() {
		logger.Info("security layer skipped - disabled or infrastructure.security.aws_account not set")
		return nil
	}

	secData := pg.buildTemplateData("security", "")

	metadataContent, err := pg.engine.Render("metadata.tf.tpl", secData)
	if err != nil {
		return err
	}
	metadataPath := filepath.Join(pg.workingDir, "security", "metadata.tf")
	if err := pg.writeFileWithConfirmation(metadataPath, metadataContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("security/metadata.tf skipped by user")
		} else {
			return err
		}
	}

	mainContent, err := pg.engine.Render("main.tf.security.tpl", secData)
	if err != nil {
		return err
	}
	mainPath := filepath.Join(pg.workingDir, "security", "main.tf")
	if err := pg.writeFileWithConfirmation(mainPath, mainContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("security/main.tf skipped by user")
		} else {
			return err
		}
	}

	if pg.shouldGenerateTerragrunt("security", "", "sec") {
		terragruntContent, err := pg.engine.Render("terragrunt.hcl.tpl", secData)
		if err != nil {
			return err
		}
		terragruntPath := filepath.Join(pg.workingDir, "security", "terragrunt.hcl")
		if err := pg.writeFileWithConfirmation(terragruntPath, terragruntContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("security/terragrunt.hcl skipped by user")
			} else {
				return err
			}
		}
	} else {
		terragruntPath := filepath.Join(pg.workingDir, "security", "terragrunt.hcl")
		if utils.FileExists(terragruntPath) {
			if err := utils.DeleteFile(terragruntPath); err != nil {
				logger.Error("Failed to delete %s: %v", terragruntPath, err)
			} else {
				logger.Info("security/terragrunt.hcl removed - terragrunt disabled")
			}
		} else {
			logger.Info("security/terragrunt.hcl skipped - terragrunt disabled")
		}
	}

	if pg.shouldGenerateSecrets("security", "", "sec") {
		secretsContent, err := pg.engine.Render("_secrets.tf.tpl", secData)
		if err != nil {
			return err
		}
		secretsPath := filepath.Join(pg.workingDir, "security", "_secrets.tf")
		if err := pg.writeFileWithConfirmation(secretsPath, secretsContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("security/_secrets.tf skipped by user")
			} else {
				return err
			}
		}
	} else {
		logger.Info("security/_secrets.tf skipped - secrets disabled")
	}

	if pg.shouldGenerateProviders("security", "", "sec") {
		providersData := pg.buildProviderTemplateData("security", "", "sec")
		providersContent, err := pg.engine.Render("providers.tf.tpl", &models.TemplateData{
			Providers: providersData.Providers,
		})
		if err != nil {
			return err
		}
		providersPath := filepath.Join(pg.workingDir, "security", "providers.tf")
		if err := pg.writeFileWithConfirmation(providersPath, providersContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("security/providers.tf skipped by user")
			} else {
				return err
			}
		}
	} else {
		logger.Info("security/providers.tf skipped - providers disabled")
	}

	if pg.shouldGenerateBackend("security", "", "sec") {
		backendData := pg.buildBackendTemplateData("security", "", "sec")
		backendContent, err := pg.engine.Render("backend.tf.tpl", &models.TemplateData{
			BackendType:          backendData.Type,
			BackendBucket:        backendData.Bucket,
			BackendKey:           backendData.Key,
			BackendRegion:        backendData.Region,
			BackendDynamoDBTable: backendData.DynamoDBTable,
			BackendEncrypt:       backendData.Encrypt,
			BackendProfile:       backendData.Profile,
			BackendAssumeRole:    backendData.AssumeRole,
		})
		if err != nil {
			return err
		}
		backendPath := filepath.Join(pg.workingDir, "security", "backend.tf")
		if err := pg.writeFileWithConfirmation(backendPath, backendContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("security/backend.tf skipped by user")
			} else {
				return err
			}
		}
	} else {
		logger.Info("security/backend.tf skipped - backend disabled")
	}

	return nil
}

// getDirectoryName determines the directory name for an environment using the fallback logic
func (pg *ProjectGenerator) getDirectoryName(envKey string) string {
	envConfig, exists := pg.config.Environments[envKey]
	if !exists {
		return envKey
	}

	// Option 1: Use dir_name if specified
	if envConfig.DirName != "" {
		return envConfig.DirName
	}

	// Option 2: Use name as directory (lowercase, spaces to _)
	if envConfig.Name != "" {
		return models.NormalizeDisplayName(envConfig.Name)
	}

	// Option 3: Use environment key (fallback)
	return envKey
}

// buildTemplateData builds template data for rendering
func (pg *ProjectGenerator) buildTemplateData(layer, env string) *models.TemplateData {
	// Build environments map
	environments := make(map[string]models.Environment)
	for envName, envConfig := range pg.config.Environments {
		environments[envName] = envConfig
	}

	// Get environment name and key
	var envName string
	var envKey string
	var envVersion string
	if env != "" {
		envConfig, exists := pg.config.Environments[env]
		if exists {
			envName = envConfig.Name
			envKey = env
			envVersion = models.ResolveVersion(envConfig, pg.config.Version)
		} else {
			envName = env
			envKey = env
			envVersion = pg.config.Version
		}
	} else {
		switch layer {
		case "organization":
			envName = "Organization"
			envKey = "org"
		case "security":
			envName = "Security"
			envKey = "sec"
		default:
			envName = ""
			envKey = ""
		}
		envVersion = pg.config.Version
	}

	// Build common names
	commonNamePrefix := fmt.Sprintf("%s-%s", pg.config.Company, envKey)
	commonName := commonNamePrefix

	// Get region for this environment
	region := pg.getRegionForEnvironment(envKey)

	// Build metadata
	metadata := map[string]interface{}{
		"aws_region":  region,
		"environment": envName,
		"key": map[string]interface{}{
			"company": pg.config.Company,
			"region":  mapRegionToShortCode(region),
			"env":     envKey,
			"layer":   layer,
		},
	}

	// Set project field for project and workload layers
	var project string
	if layer == "project" || layer == "workload" {
		project = "core"
	}

	// Calculate dependencies using the new logic
	dependencies := models.CalculateDependencies(layer, project, env, pg.config)

	// Resolve backend configuration
	backendConfig := models.ResolveBackendConfig(pg.config)

	// Resolve and align custom metadata according to layer/environment hierarchy.
	resolvedMetadata := pg.config.ResolveMetadata(layer, envKey)
	metadataLines := pg.buildAlignedMetadataLines(resolvedMetadata)

	// Get region for this environment
	region = pg.getRegionForEnvironment(envKey)

	// Resolve source configuration
	var sourceConfig models.SourceConfig
	if env != "" {
		envConfig, exists := pg.config.Environments[env]
		if exists {
			sourceConfig = envConfig.GetSource(pg.config)
		} else {
			// Create a dummy environment for global source resolution
			dummyEnv := models.Environment{}
			sourceConfig = dummyEnv.GetSource(pg.config)
		}
	} else {
		// Create a dummy environment for global source resolution
		dummyEnv := models.Environment{}
		sourceConfig = dummyEnv.GetSource(pg.config)
	}

	// Resolve secrets config
	secretsConfig := pg.config.ResolveSecretsConfig(layer, "", envKey)
	secretsBackendType := secretsConfig.Type

	return &models.TemplateData{
		Client:                pg.config.Client,
		Company:               pg.config.Company,
		Region:                region,
		RegionShortCode:       mapRegionToShortCode(region),
		Version:               envVersion,
		Source:                sourceConfig.Source,
		SourceRef:             sourceConfig.SourceRef,
		IsGitSource:           sourceConfig.IsGit,
		BackendPattern:        backendConfig.Pattern,
		BackendRegion:         backendConfig.Region,
		BackendAccount:        backendConfig.Account,
		BackendEncrypt:        backendConfig.Encrypt,
		BackendBucketName:     backendConfig.BucketName,
		BackendDynamoDBTable:  backendConfig.DynamoDBTableName,
		AWSSSO:                pg.config.AWSSSO,
		Environments:          environments,
		ProcessedEnvironments: models.ProcessEnvironments(pg.config),
		Layer:                 layer,
		Project:               "", // No project for base/foundation layers
		ProjectKey:            "", // No project for base/foundation layers
		ProjectName:           "", // No project for base/foundation layers
		Environment:           envKey,
		EnvironmentName:       envName,
		EnvKey:                envKey,
		CommonName:            commonName,
		CommonNamePrefix:      commonNamePrefix,
		Metadata:              metadata,
		MetadataLines:         metadataLines,
		Dependencies:          dependencies,
		SecretsBackendType:    secretsBackendType,
	}
}

// generateREADME generates the README.md content
func (pg *ProjectGenerator) generateREADME() string {
	if pg.config == nil {
		return "# Error: Configuration is required to generate README"
	}

	// Generate dynamic content using helper functions
	environmentTable := pg.generateEnvironmentTable()
	commandExamples := pg.generateCommandExamples()

	return fmt.Sprintf(`<!-- =============================================================================
This file is generated and maintained by GoCloud CLI
DO NOT EDIT MANUALLY - Changes will be overwritten on next generation
============================================================================= -->

# %s Infrastructure Project

This repository contains the infrastructure configuration for **%s** using Terraform.

## 📋 Project Overview

- **Client**: %s
- **Company**: %s
- **Region**: %s
- **Terraform Version**: %s

## 🌍 Environments Configuration

%s

## 🚀 Quick Start

### 1. **Initial AWS SSO Configuration**

#### ⚡ Automatic Option (Recommended)
`+"```"+`bash
# Setup AWS SSO profiles
gocloud sso setup

# Login to all AWS profiles
gocloud sso login --all

# Verify SSO status
gocloud sso verify
`+"```"+`

#### 🔧 Manual Option
`+"```"+`bash
# 1. Generate AWS configuration
terragrunt init

# 2. Set environment variable
export AWS_CONFIG_FILE=$(pwd)/.aws/config

# 3. Login to all profiles
# The automatic script will detect and login to all available profiles
# Or manually for specific profiles:
AWS_CONFIG_FILE=$(pwd)/.aws/config aws sso login --profile <profile-name>
`+"```"+`

### 2. **Initialize the Entire Project**
`+"```"+`bash
# Initialize and update all modules
terragrunt init --all
`+"```"+`

### 3. **Verify Configuration**
`+"```"+`bash
# Verify everything is configured correctly
gocloud sso verify

# Plan all environments (just to verify)
terragrunt plan -concise --all
`+"```"+`

## 🏗️ Daily Work Commands

%s

## 📝 Important Notes

- **Local Configuration**: The `+"`"+`.aws/config`+"`"+` file is generated automatically and doesn't pollute your global configuration
- **Credentials**: You only need to do `+"`"+`aws sso login`+"`"+` when credentials expire
- **Version Control**: Generated files are excluded from git
- **Parallelization**: Scripts execute operations in parallel for greater speed
- **Authentication Modes**: Supports both auto-generated SSO profiles and user-provided authentication via `+"`"+`TF_AWS_NO_PROFILE=true`+"`"+`

## 🆘 Troubleshooting

### **Error: "failed to get shared config profile"**
`+"```"+`bash
# Run the configuration script
gocloud sso setup
`+"```"+`

### **Expired Credentials**
`+"```"+`bash
# Check profile status
gocloud sso verify

# Re-login to specific profiles
AWS_CONFIG_FILE=$(pwd)/.aws/config aws sso login --profile <profile-name>
`+"```"+`

### **Custom Authentication**
`+"```"+`bash
# Force to not use auto-generated profiles and trust user-provided authentication
TF_AWS_NO_PROFILE=true terragrunt plan --all
`+"```"+`

### **Initialization Issues**
`+"```"+`bash
# Clean and reinitialize
terragrunt init -upgrade -reconfigure --all
`+"```"+`

---

## 🤝 Support

- **Email**: info@gocloud.la
- **Website**: [www.gocloud.la](https://www.gocloud.la)
- **AWS Partner**: Advanced Partner (Terraform, DevOps, GenAI)
`,
		pg.config.Client,
		pg.config.Client,
		pg.config.Client,
		pg.config.Company,
		pg.config.Region,
		pg.config.Version,
		environmentTable,
		commandExamples,
	)
}

// generateProjectFiles generates files for a specific project and environment
func (pg *ProjectGenerator) generateProjectFiles(layerType string, item interface{}, env string) error {
	dirName := pg.getDirectoryName(env)

	// Get project/workload information
	var projectKey, projectDirName string
	switch layerType {
	case "project":
		projectKey = models.GetProjectKey(item)
		projectDirName = models.GetProjectDirectoryName(item)
	case "workload":
		projectKey = models.GetWorkloadKey(item)
		projectDirName = models.GetWorkloadDirectoryName(item)
	default:
		return fmt.Errorf("unsupported layer type: %s", layerType)
	}

	layerPath := filepath.Join(pg.workingDir, layerType, projectDirName, dirName)

	// Generate metadata.tf
	metadataData := pg.buildProjectTemplateData(layerType, item, env)
	metadataContent, err := pg.engine.Render("metadata.tf.tpl", metadataData)
	if err != nil {
		return err
	}
	metadataPath := filepath.Join(layerPath, "metadata.tf")
	if err := pg.writeFileWithConfirmation(metadataPath, metadataContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("%s/%s/%s/metadata.tf skipped by user", layerType, projectDirName, dirName)
		} else {
			return err
		}
	}

	// Generate terragrunt.hcl only if terragrunt is enabled
	if pg.shouldGenerateTerragrunt(layerType, projectKey, env) {
		terragruntData := pg.buildProjectTemplateData(layerType, item, env)
		terragruntContent, err := pg.engine.Render("terragrunt.hcl.tpl", terragruntData)
		if err != nil {
			return err
		}
		terragruntPath := filepath.Join(layerPath, "terragrunt.hcl")
		if err := pg.writeFileWithConfirmation(terragruntPath, terragruntContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("%s/%s/%s/terragrunt.hcl skipped by user", layerType, projectDirName, dirName)
			} else {
				return err
			}
		}
	} else {
		// If terragrunt is disabled, remove existing terragrunt.hcl file if it exists
		terragruntPath := filepath.Join(layerPath, "terragrunt.hcl")
		if utils.FileExists(terragruntPath) {
			if err := utils.DeleteFile(terragruntPath); err != nil {
				logger.Error("Failed to delete %s: %v", terragruntPath, err)
			} else {
				logger.Info("%s/%s/%s/terragrunt.hcl removed - terragrunt disabled", layerType, projectDirName, dirName)
			}
		} else {
			logger.Info("%s/%s/%s/terragrunt.hcl skipped - terragrunt disabled", layerType, projectDirName, dirName)
		}
	}

	// Generate _secrets.tf only if secrets are enabled
	if pg.shouldGenerateSecrets(layerType, projectKey, env) {
		secretsData := pg.buildProjectTemplateData(layerType, item, env)
		secretsContent, err := pg.engine.Render("_secrets.tf.tpl", secretsData)
		if err != nil {
			return err
		}
		secretsPath := filepath.Join(layerPath, "_secrets.tf")
		if err := pg.writeFileWithConfirmation(secretsPath, secretsContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("%s/%s/%s/_secrets.tf skipped by user", layerType, projectDirName, dirName)
			} else {
				return err
			}
		}
	} else {
		logger.Info("%s/%s/%s/_secrets.tf skipped - secrets disabled", layerType, projectDirName, dirName)
	}

	// Generate providers.tf if providers are enabled
	if pg.shouldGenerateProviders(layerType, projectKey, env) {
		providersData := pg.buildProviderTemplateData(layerType, projectKey, env)
		providersContent, err := pg.engine.Render("providers.tf.tpl", &models.TemplateData{
			Providers: providersData.Providers,
		})
		if err != nil {
			return err
		}
		providersPath := filepath.Join(layerPath, "providers.tf")
		if err := pg.writeFileWithConfirmation(providersPath, providersContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("%s/%s/%s/providers.tf skipped by user", layerType, projectDirName, dirName)
			} else {
				return err
			}
		}
	} else {
		logger.Info("%s/%s/%s/providers.tf skipped - providers disabled", layerType, projectDirName, dirName)
	}

	// Generate backend.tf if backend is enabled
	if pg.shouldGenerateBackend(layerType, projectKey, env) {
		backendData := pg.buildBackendTemplateData(layerType, projectKey, env)
		backendContent, err := pg.engine.Render("backend.tf.tpl", &models.TemplateData{
			BackendType:          backendData.Type,
			BackendBucket:        backendData.Bucket,
			BackendKey:           backendData.Key,
			BackendRegion:        backendData.Region,
			BackendDynamoDBTable: backendData.DynamoDBTable,
			BackendEncrypt:       backendData.Encrypt,
			BackendProfile:       backendData.Profile,
			BackendAssumeRole:    backendData.AssumeRole,
		})
		if err != nil {
			return err
		}
		backendPath := filepath.Join(layerPath, "backend.tf")
		if err := pg.writeFileWithConfirmation(backendPath, backendContent); err != nil {
			if errors.Is(err, ErrFileSkipped) {
				logger.Info("%s/%s/%s/backend.tf skipped by user", layerType, projectDirName, dirName)
			} else {
				return err
			}
		}
	} else {
		logger.Info("%s/%s/%s/backend.tf skipped - backend disabled", layerType, projectDirName, dirName)
	}

	// Generate main.tf using layer-specific template
	mainData := pg.buildProjectTemplateData(layerType, item, env)
	templateName := fmt.Sprintf("main.tf.%s.tpl", layerType)
	mainContent, err := pg.engine.Render(templateName, mainData)
	if err != nil {
		return err
	}
	mainPath := filepath.Join(layerPath, "main.tf")
	if err := pg.writeFileWithConfirmation(mainPath, mainContent); err != nil {
		if errors.Is(err, ErrFileSkipped) {
			logger.Info("%s/%s/%s/main.tf skipped by user", layerType, projectDirName, dirName)
		} else {
			return err
		}
	}

	return nil
}

// buildProjectTemplateData builds template data for project-specific files
func (pg *ProjectGenerator) buildProjectTemplateData(layerType string, item interface{}, env string) *models.TemplateData {
	// Build environments map
	environments := make(map[string]models.Environment)
	for envName, envConfig := range pg.config.Environments {
		environments[envName] = envConfig
	}

	// Get environment name and key
	var envName string
	var envKey string
	var envVersion string
	envConfig, exists := pg.config.Environments[env]
	if exists {
		envName = envConfig.Name
		envKey = env
		envVersion = models.ResolveVersion(envConfig, pg.config.Version)
	} else {
		envName = env
		envKey = env
		envVersion = pg.config.Version
	}

	// Get region for this environment
	region := pg.getRegionForEnvironment(envKey)

	// Get project/workload information
	var projectKey, projectDisplayName string
	switch layerType {
	case "project":
		projectKey = models.GetProjectKey(item)
		projectDisplayName = models.GetProjectDisplayName(item)
	case "workload":
		projectKey = models.GetWorkloadKey(item)
		projectDisplayName = models.GetWorkloadDisplayName(item)
	default:
		projectKey = ""
		projectDisplayName = ""
	}

	// Build metadata
	metadata := map[string]interface{}{
		"aws_region":  region,
		"environment": envName,
		"project":     projectDisplayName,
		"key": map[string]interface{}{
			"company": pg.config.Company,
			"region":  mapRegionToShortCode(region),
			"env":     envKey,
			"project": projectKey,
			"layer":   layerType,
		},
	}

	// Build common name
	commonNamePrefix := fmt.Sprintf("%s-%s", pg.config.Company, envKey)
	commonName := fmt.Sprintf("%s-%s", commonNamePrefix, projectKey)

	// Calculate dependencies using the new logic
	dependencies := models.CalculateDependencies(layerType, projectKey, env, pg.config)

	// Resolve backend configuration
	backendConfig := models.ResolveBackendConfig(pg.config)

	// Resolve and align custom metadata according to layer/environment hierarchy.
	resolvedMetadata := pg.config.ResolveMetadata(layerType, envKey)
	metadataLines := pg.buildAlignedMetadataLines(resolvedMetadata)

	// Get region for this environment
	region = pg.getRegionForEnvironment(envKey)

	// Resolve source configuration
	var sourceConfig models.SourceConfig
	if exists {
		sourceConfig = envConfig.GetSource(pg.config)
	} else {
		// Create a dummy environment for global source resolution
		dummyEnv := models.Environment{}
		sourceConfig = dummyEnv.GetSource(pg.config)
	}

	// Resolve secrets config
	secretsConfig := pg.config.ResolveSecretsConfig(layerType, projectKey, envKey)
	secretsBackendType := secretsConfig.Type

	return &models.TemplateData{
		Client:             pg.config.Client,
		Company:            pg.config.Company,
		Region:             region,
		RegionShortCode:    mapRegionToShortCode(region),
		Version:            envVersion,
		Source:             sourceConfig.Source,
		SourceRef:          sourceConfig.SourceRef,
		IsGitSource:        sourceConfig.IsGit,
		BackendPattern:     backendConfig.Pattern,
		BackendRegion:      backendConfig.Region,
		BackendAccount:     backendConfig.Account,
		BackendEncrypt:     backendConfig.Encrypt,
		Environments:       environments,
		Layer:              layerType,
		Project:            projectKey,
		ProjectKey:         projectKey,
		ProjectName:        projectDisplayName,
		Environment:        envKey,
		EnvironmentName:    envName,
		EnvKey:             envKey,
		CommonName:         commonName,
		CommonNamePrefix:   commonNamePrefix,
		Metadata:           metadata,
		MetadataLines:      metadataLines,
		Dependencies:       dependencies,
		SecretsBackendType: secretsBackendType,
	}
}

// buildProviderTemplateData builds template data for rendering providers.tf
func (pg *ProjectGenerator) buildProviderTemplateData(layerType, project, env string) *models.ProviderTemplateData {
	// Resolve provider configuration with hierarchy
	providerConfig := pg.config.ResolveProviderConfig(layerType, project, env)

	// Default providers configuration
	defaultProviders := []models.ProviderSpec{
		{
			Name:   "aws",
			Region: "local.metadata.aws_region",
		},
		{
			Name:   "aws",
			Region: "us-east-1",
			Alias:  "use1",
		},
	}

	// Use custom providers if configured
	if providerConfig != nil && len(providerConfig.DefaultProviders) > 0 {
		defaultProviders = providerConfig.DefaultProviders
	}

	// Add profiles if enabled (default to true)
	useProfiles := true
	if providerConfig != nil && providerConfig.UseProfiles != nil {
		useProfiles = *providerConfig.UseProfiles
	}

	if useProfiles {
		// Get environment configuration for profile (only applied to AWS providers)
		envConfig, exists := pg.config.Environments[env]
		var profile string
		if layerType == "organization" && env == "org" && pg.config.Organization != nil && pg.config.Organization.AWSAccount != "" {
			profile = fmt.Sprintf("%s-org", pg.config.Client)
		} else if layerType == "security" && env == "sec" && pg.config.Security != nil && pg.config.Security.AWSAccount != "" {
			profile = fmt.Sprintf("%s-sec", pg.config.Client)
		} else if exists {
			hasSSO := pg.config.AWSSSO != nil || (envConfig.AWSSSO != nil)
			if hasSSO {
				profile = fmt.Sprintf("%s-%s", pg.config.Client, env)
			}
		}
		// Only fill profile when not set in YAML; explicit per-provider profile wins.
		if profile != "" {
			for i := range defaultProviders {
				if defaultProviders[i].Name == "aws" && defaultProviders[i].Profile == "" {
					defaultProviders[i].Profile = profile
				}
			}
		}
	}

	return &models.ProviderTemplateData{
		Providers: defaultProviders,
	}
}

// buildBackendTemplateData builds template data for rendering backend.tf
func (pg *ProjectGenerator) buildBackendTemplateData(layerType, project, env string) *models.BackendTemplateData {
	// Resolve backend configuration with hierarchy
	backendConfig := pg.config.ResolveBackendConfig(layerType, project, env)

	envConfig, exists := pg.config.Environments[env]
	// Organization layer: no Environments["org"]; use synthetic env from organization.aws_account when set
	if layerType == "organization" && env == "org" && !exists && pg.config.Organization != nil && pg.config.Organization.AWSAccount != "" {
		envConfig = models.Environment{Name: "Organization", AWSAccount: pg.config.Organization.AWSAccount}
		exists = true
	}
	if layerType == "security" && env == "sec" && !exists && pg.config.Security != nil && pg.config.Security.AWSAccount != "" {
		envConfig = models.Environment{Name: "Security", AWSAccount: pg.config.Security.AWSAccount}
		exists = true
	}
	if !exists || backendConfig == nil {
		// Return minimal default if no environment or backend config
		return &models.BackendTemplateData{
			Type:    "s3",
			Bucket:  fmt.Sprintf("%s-s3-backend", pg.config.Company),
			Region:  "us-east-1",
			Encrypt: true,
		}
	}

	// Build assume role ARN
	// Get the backend account ID (where the state is stored)
	backendAccountID := envConfig.AWSAccount // Default to current environment
	if backendEnv, exists := pg.config.Environments[backendConfig.Account]; exists {
		backendAccountID = backendEnv.AWSAccount
	}

	// Build assume role ARN
	assumeRole := &models.AssumeRoleConfig{}

	// Check if there's a custom role template in backend configuration
	if backendConfig.RoleTemplate != "" {
		// Use custom role template
		roleName := pg.processRoleTemplate(backendConfig.RoleTemplate, layerType, project, env, envConfig, backendAccountID)
		assumeRole.RoleARN = fmt.Sprintf("arn:aws:iam::%s:role/%s",
			backendAccountID, // Account where state is stored
			roleName)
	} else {
		// Use default pattern for general cases (restored original pattern)
		assumeRole.RoleARN = fmt.Sprintf("arn:aws:iam::%s:role/%s-%s-%s-%s",
			backendAccountID, // Account where state is stored (e.g., "inf")
			pg.config.Company,
			backendConfig.Account,
			backendConfig.Pattern,
			envConfig.AWSAccount) // Account of current environment
	}

	// Add profile if enabled
	profile := ""
	// Check if AWS SSO is configured (global or environment level)
	hasSSO := pg.config.AWSSSO != nil || (envConfig.AWSSSO != nil)
	if hasSSO {
		profile = fmt.Sprintf("%s-%s", pg.config.Client, env)
	}

	// Use configured bucket name or build default
	bucketName := backendConfig.BucketName
	if bucketName == "" {
		bucketName = fmt.Sprintf("%s-%s-%s", pg.config.Company, backendConfig.Account, backendConfig.Pattern)
	}

	// Use configured DynamoDB table name or build default
	dynamoDBTable := backendConfig.DynamoDBTableName
	if dynamoDBTable == "" {
		dynamoDBTable = fmt.Sprintf("%s-%s-%s", pg.config.Company, backendConfig.Account, backendConfig.Pattern)
	}

	// Use configured backend type or default to "s3"
	backendType := backendConfig.Type
	if backendType == "" {
		backendType = "s3"
	}

	// Use configured key template or build default
	keyTemplate := backendConfig.KeyTemplate
	if keyTemplate == "" {
		switch layerType {
		case "organization":
			keyTemplate = fmt.Sprintf("%s/organization/terraform.tfstate", envConfig.AWSAccount)
		case "security":
			keyTemplate = fmt.Sprintf("%s/security/terraform.tfstate", envConfig.AWSAccount)
		default:
			envSeg := models.EnvironmentNameForBackendKey(env, envConfig)
			keyTemplate = fmt.Sprintf("%s/%s-%s/terraform.tfstate", envConfig.AWSAccount, layerType, envSeg)
			if project != "" {
				keyTemplate = fmt.Sprintf("%s/%s-%s-%s/terraform.tfstate", envConfig.AWSAccount, layerType, project, envSeg)
			}
		}
	} else {
		// Process template variables
		keyTemplate = pg.processKeyTemplate(keyTemplate, layerType, project, env, envConfig)
	}

	// Use configured profile control or default to true
	useProfile := true
	if backendConfig.UseProfile != nil {
		useProfile = *backendConfig.UseProfile
	}

	// Only add profile if useProfile is true
	if !useProfile {
		profile = ""
	}

	// Apply hierarchy overrides if configured
	if backendConfig.Type != "" {
		backendType = backendConfig.Type
	}
	if backendConfig.KeyTemplate != "" {
		keyTemplate = pg.processKeyTemplate(backendConfig.KeyTemplate, layerType, project, env, envConfig)
	}
	if backendConfig.UseProfile != nil {
		useProfile = *backendConfig.UseProfile
		if !useProfile {
			profile = ""
		}
	}

	// Use default values when no global backend config exists
	region := "us-east-1"
	encrypt := true
	if pg.config.Backend != nil {
		if pg.config.Backend.Region != "" {
			region = pg.config.Backend.Region
		}
		encrypt = pg.config.Backend.Encrypt
	}

	return &models.BackendTemplateData{
		Type:          backendType,
		Bucket:        bucketName,
		Key:           keyTemplate,
		Region:        region,
		DynamoDBTable: dynamoDBTable,
		Encrypt:       encrypt,
		Profile:       profile,
		AssumeRole:    assumeRole,
	}
}

// processKeyTemplate processes template variables in key template
func (pg *ProjectGenerator) processKeyTemplate(template, layerType, project, env string, envConfig models.Environment) string {
	// If project is empty, remove the dash before {{.Project}} to avoid trailing dashes
	result := template
	if project == "" {
		// Remove dash before {{.Project}} when project is empty
		result = strings.ReplaceAll(result, "-{{.Project}}", "")
	}

	// Replace template variables
	result = strings.ReplaceAll(result, "{{.AccountID}}", envConfig.AWSAccount)
	result = strings.ReplaceAll(result, "{{.Layer}}", layerType)
	result = strings.ReplaceAll(result, "{{.Project}}", project)
	result = strings.ReplaceAll(result, "{{.Environment}}", env)
	result = strings.ReplaceAll(result, "{{.EnvironmentName}}", models.EnvironmentNameForBackendKey(env, envConfig))
	result = strings.ReplaceAll(result, "{{.Company}}", pg.config.Company)
	result = strings.ReplaceAll(result, "{{.Region}}", pg.config.Region)
	result = strings.ReplaceAll(result, "{{.Client}}", pg.config.Client)
	return result
}

// processRoleTemplate processes template variables in role template
func (pg *ProjectGenerator) processRoleTemplate(template, layerType, project, env string, envConfig models.Environment, backendAccountID string) string {
	// If project is empty, remove the dash before {{.Project}} to avoid trailing dashes
	result := template
	if project == "" {
		// Remove dash before {{.Project}} when project is empty
		result = strings.ReplaceAll(result, "-{{.Project}}", "")
	}

	// Replace template variables
	result = strings.ReplaceAll(result, "{{.AccountID}}", envConfig.AWSAccount)
	result = strings.ReplaceAll(result, "{{.BackendAccountID}}", backendAccountID)
	result = strings.ReplaceAll(result, "{{.Layer}}", layerType)
	result = strings.ReplaceAll(result, "{{.Project}}", project)
	result = strings.ReplaceAll(result, "{{.Environment}}", env)
	result = strings.ReplaceAll(result, "{{.EnvironmentName}}", models.EnvironmentNameForBackendKey(env, envConfig))
	result = strings.ReplaceAll(result, "{{.Company}}", pg.config.Company)
	result = strings.ReplaceAll(result, "{{.BackendAccount}}", pg.config.Backend.Account)
	result = strings.ReplaceAll(result, "{{.BackendPattern}}", pg.config.Backend.Pattern)
	result = strings.ReplaceAll(result, "{{.Region}}", pg.config.Region)
	result = strings.ReplaceAll(result, "{{.Client}}", pg.config.Client)
	return result
}

// GenerateREADME generates the README file (public method for testing)
func (pg *ProjectGenerator) GenerateREADME() error {
	if pg.config == nil {
		return fmt.Errorf("config is required to generate README")
	}
	if pg.workingDir == "" {
		return fmt.Errorf("working directory is required")
	}
	readmeContent := pg.generateREADME()
	readmePath := filepath.Join(pg.workingDir, "README.md")
	return pg.writeFileWithConfirmation(readmePath, readmeContent)
}

// GenerateRootConfigs generates root configuration files (public method for testing)
func (pg *ProjectGenerator) GenerateRootConfigs() error {
	if pg.config == nil {
		return fmt.Errorf("config is required to generate root configs")
	}
	return pg.generateRootConfigs()
}

// GenerateLayerConfigs generates layer configuration files (public method for testing)
func (pg *ProjectGenerator) GenerateLayerConfigs() error {
	if pg.config == nil {
		return fmt.Errorf("config is required to generate layer configs")
	}
	return pg.generateLayerConfigs()
}

// GenerateProjectFiles generates project files (public method for testing)
func (pg *ProjectGenerator) GenerateProjectFiles() error {
	if pg.config == nil {
		return fmt.Errorf("config is required to generate project files")
	}
	// Generate files for all projects in all environments
	for envKey, env := range pg.config.Environments {
		for _, project := range env.Projects {
			if err := pg.generateProjectFiles("project", project, envKey); err != nil {
				return err
			}
		}
	}
	return nil
}

// GenerateWorkloadFiles generates workload files (public method for testing)
func (pg *ProjectGenerator) GenerateWorkloadFiles() error {
	if pg.config == nil {
		return fmt.Errorf("config is required to generate workload files")
	}
	// Generate files for all workloads in all environments
	for envKey, env := range pg.config.Environments {
		for _, workload := range env.Workloads {
			if err := pg.generateProjectFiles("workload", workload, envKey); err != nil {
				return err
			}
		}
	}
	return nil
}

// GenerateOrganizationFiles generates organization files (public method for testing)
func (pg *ProjectGenerator) GenerateOrganizationFiles() error {
	if pg.config == nil {
		return fmt.Errorf("config is required to generate organization files")
	}
	return pg.generateOrganizationConfigs()
}

// GenerateSecurityFiles generates security layer files (public method for testing)
func (pg *ProjectGenerator) GenerateSecurityFiles() error {
	if pg.config == nil {
		return fmt.Errorf("config is required to generate security files")
	}
	return pg.generateSecurityConfigs()
}

// Generate runs the complete generation process (public method for testing)
func (pg *ProjectGenerator) Generate() error {
	if pg.config == nil {
		return fmt.Errorf("config is required to generate project")
	}
	// Run all generation steps
	if err := pg.CreateProjectStructure(); err != nil {
		return err
	}
	if err := pg.GenerateConfigFiles(); err != nil {
		return err
	}
	if err := pg.GenerateRootConfigs(); err != nil {
		return err
	}
	if err := pg.GenerateLayerConfigs(); err != nil {
		return err
	}
	if err := pg.GenerateProjectFiles(); err != nil {
		return err
	}
	if err := pg.GenerateWorkloadFiles(); err != nil {
		return err
	}
	if err := pg.GenerateOrganizationFiles(); err != nil {
		return err
	}
	if err := pg.GenerateSecurityFiles(); err != nil {
		return err
	}
	if err := pg.GenerateREADME(); err != nil {
		return err
	}
	return nil
}

// shouldGenerateSecrets determines if secrets should be generated for a specific layer/project
// following the hierarchy: workload/project -> environment -> infrastructure (same as shouldGenerateTerragrunt).
func (pg *ProjectGenerator) shouldGenerateSecrets(layerType, project, env string) bool {
	// Organization layer is global: allow infrastructure.organization.enable_secrets override.
	if layerType == "organization" {
		// Organization layer only exists when configured with aws_account (handled by Generate flow),
		// but secrets enablement is decided here.
		if pg.config.Organization != nil && pg.config.Organization.EnableSecrets != nil {
			return *pg.config.Organization.EnableSecrets
		}
		if pg.config.EnableSecrets != nil {
			return *pg.config.EnableSecrets
		}
		return true
	}
	if layerType == "security" {
		if pg.config.Security != nil && pg.config.Security.EnableSecrets != nil {
			return *pg.config.Security.EnableSecrets
		}
		if pg.config.EnableSecrets != nil {
			return *pg.config.EnableSecrets
		}
		return true
	}

	// Get environment configuration
	envConfig, exists := pg.config.Environments[env]
	if !exists {
		// If environment doesn't exist, use infrastructure default
		if pg.config.EnableSecrets != nil {
			return *pg.config.EnableSecrets
		}
		// Default to true if not specified
		return true
	}

	// Check if it's a project or workload (has project parameter)
	if project != "" {
		// Check workloads first (same order as shouldGenerateTerragrunt)
		if layerType == "workload" {
			for _, workloadInterface := range envConfig.Workloads {
				var workloadName string
				var workloadConfig *models.WorkloadItem

				// Normalize map[interface{}]interface{} (YAML unmarshal) to map[string]interface{}
				var w map[string]interface{}
				switch item := workloadInterface.(type) {
				case string:
					workloadName = item
				case map[interface{}]interface{}:
					w = models.ToMapStringInterface(item)
				case map[string]interface{}:
					w = item
				}
				if w != nil {
					if len(w) == 1 {
						for key, value := range w {
							workloadName = key
							if valueMap, ok := value.(map[string]interface{}); ok {
								if enableSecrets, ok := valueMap["enable_secrets"].(bool); ok {
									workloadConfig = &models.WorkloadItem{
										Name:          key,
										EnableSecrets: &enableSecrets,
									}
								}
							}
						}
					} else {
						if name, ok := w["name"].(string); ok {
							workloadName = name
							if enableSecrets, ok := w["enable_secrets"].(bool); ok {
								workloadConfig = &models.WorkloadItem{
									Name:          name,
									EnableSecrets: &enableSecrets,
								}
							}
						}
					}
				}

				if workloadName == project {
					if workloadConfig != nil && workloadConfig.EnableSecrets != nil {
						return *workloadConfig.EnableSecrets
					}
					break
				}
			}
		}

		// Check projects (same as shouldGenerateTerragrunt)
		if layerType == "project" {
			for _, projectInterface := range envConfig.Projects {
				var projectName string
				var enableSecrets *bool

				// Normalize map[interface{}]interface{} (YAML unmarshal) to map[string]interface{}
				var p map[string]interface{}
				switch item := projectInterface.(type) {
				case string:
					projectName = item
				case map[interface{}]interface{}:
					p = models.ToMapStringInterface(item)
				case map[string]interface{}:
					p = item
				}
				if p != nil {
					if len(p) == 1 {
						for key, value := range p {
							projectName = key
							if valueMap, ok := value.(map[string]interface{}); ok {
								if es, ok := valueMap["enable_secrets"].(bool); ok {
									enableSecrets = &es
								}
							}
						}
					} else {
						if name, ok := p["name"].(string); ok {
							projectName = name
							if es, ok := p["enable_secrets"].(bool); ok {
								enableSecrets = &es
							}
						}
					}
				}

				if projectName == project {
					if enableSecrets != nil {
						return *enableSecrets
					}
					break
				}
			}
		}
	}

	// Check environment level
	if envConfig.EnableSecrets != nil {
		return *envConfig.EnableSecrets
	}

	// Fall back to infrastructure level
	if pg.config.EnableSecrets != nil {
		return *pg.config.EnableSecrets
	}
	// Default to true if not specified
	return true
}

// shouldGenerateTerragrunt determines if terragrunt.hcl should be generated for a specific layer/project
// following the hierarchy: workload/project -> environment -> infrastructure
func (pg *ProjectGenerator) shouldGenerateTerragrunt(layerType, project, env string) bool {
	// Get environment configuration
	envConfig, exists := pg.config.Environments[env]
	if !exists {
		// If environment doesn't exist, use infrastructure default
		if pg.config.EnableTerragrunt != nil {
			return *pg.config.EnableTerragrunt
		}
		// Default to true if not specified
		return true
	}

	// Check if it's a project or workload (has project parameter)
	if project != "" {
		// Check workloads first
		if layerType == "workload" {
			for _, workloadInterface := range envConfig.Workloads {
				var workloadName string
				var workloadConfig *models.WorkloadItem

				switch w := workloadInterface.(type) {
				case string:
					workloadName = w
				case map[string]interface{}:
					// Handle case where workload is defined as: - workload-name: {enable_terragrunt: false}
					if len(w) == 1 {
						for key, value := range w {
							workloadName = key
							if valueMap, ok := value.(map[string]interface{}); ok {
								if enableTerragrunt, ok := valueMap["enable_terragrunt"].(bool); ok {
									workloadConfig = &models.WorkloadItem{
										Name:             key,
										EnableTerragrunt: &enableTerragrunt,
									}
								}
							}
						}
					} else {
						// Handle case where workload is defined as: - {name: workload-name, enable_terragrunt: false}
						if name, ok := w["name"].(string); ok {
							workloadName = name
							if enableTerragrunt, ok := w["enable_terragrunt"].(bool); ok {
								workloadConfig = &models.WorkloadItem{
									Name:             name,
									EnableTerragrunt: &enableTerragrunt,
								}
							}
						}
					}
				}

				if workloadName == project {
					// If workload has explicit enable_terragrunt setting, use it
					if workloadConfig != nil && workloadConfig.EnableTerragrunt != nil {
						return *workloadConfig.EnableTerragrunt
					}
					// Otherwise, fall through to environment level
					break
				}
			}
		}

		// Check projects
		if layerType == "project" {
			for _, projectInterface := range envConfig.Projects {
				var projectName string
				var projectConfig *models.ProjectItem

				switch p := projectInterface.(type) {
				case string:
					projectName = p
				case map[string]interface{}:
					// Handle case where project is defined as: - project-name: {enable_terragrunt: false}
					if len(p) == 1 {
						for key, value := range p {
							projectName = key
							if valueMap, ok := value.(map[string]interface{}); ok {
								if enableTerragrunt, ok := valueMap["enable_terragrunt"].(bool); ok {
									projectConfig = &models.ProjectItem{
										Name:             key,
										EnableTerragrunt: &enableTerragrunt,
									}
								}
							}
						}
					} else {
						// Handle case where project is defined as: - {name: project-name, enable_terragrunt: false}
						if name, ok := p["name"].(string); ok {
							projectName = name
							if enableTerragrunt, ok := p["enable_terragrunt"].(bool); ok {
								projectConfig = &models.ProjectItem{
									Name:             name,
									EnableTerragrunt: &enableTerragrunt,
								}
							}
						}
					}
				}

				if projectName == project {
					// If project has explicit enable_terragrunt setting, use it
					if projectConfig != nil && projectConfig.EnableTerragrunt != nil {
						return *projectConfig.EnableTerragrunt
					}
					// Otherwise, fall through to environment level
					break
				}
			}
		}
	}

	// Check environment level
	if envConfig.EnableTerragrunt != nil {
		return *envConfig.EnableTerragrunt
	}

	// Fall back to infrastructure level
	if pg.config.EnableTerragrunt != nil {
		return *pg.config.EnableTerragrunt
	}
	// Default to true if not specified
	return true
}

// shouldGenerateProviders determines if providers.tf should be generated
// following the hierarchy: workload > project > environment > global
func (pg *ProjectGenerator) shouldGenerateProviders(layerType, project, env string) bool {
	// Check if providers configuration exists at any level
	providerConfig := pg.config.ResolveProviderConfig(layerType, project, env)

	// If no configuration exists, default to true (generate providers)
	if providerConfig == nil {
		return true
	}

	// If configuration exists, generate providers
	return true
}

// shouldGenerateBackend determines if backend.tf should be generated
// following the hierarchy: workload > project > environment > global
func (pg *ProjectGenerator) shouldGenerateBackend(layerType, project, env string) bool {
	// Check if backend configuration exists at any level
	backendConfig := pg.config.ResolveBackendConfig(layerType, project, env)

	// If no configuration exists, default to true (generate backend)
	if backendConfig == nil {
		return true
	}

	// Check if backend is explicitly disabled
	if backendConfig.Type == "disabled" {
		return false
	}

	// If configuration exists, generate backend
	return true
}

// shouldGenerateLayer determines if a specific layer should be generated for a specific environment
// following the hierarchy: environment -> infrastructure -> default (true)
// Note: organization and security layers are global and should not use this function with a real env key
func (pg *ProjectGenerator) shouldGenerateLayer(layerType, env string) bool {
	// Organization is global; enabled only when layers.organization and infrastructure.organization.aws_account are set
	if layerType == "organization" {
		return pg.isOrganizationLayerEnabled()
	}
	if layerType == "security" {
		return pg.isSecurityLayerEnabled()
	}

	// Get environment configuration
	envConfig, exists := pg.config.Environments[env]
	if !exists {
		// If environment doesn't exist, use infrastructure default
		return pg.getLayerDefault(layerType)
	}

	// Check environment level
	if envConfig.Layers != nil {
		switch layerType {
		case "base":
			if envConfig.Layers.Base != nil {
				return *envConfig.Layers.Base
			}
		case "foundation":
			if envConfig.Layers.Foundation != nil {
				return *envConfig.Layers.Foundation
			}
		}
	}

	// Fall back to infrastructure level
	return pg.getLayerDefault(layerType)
}

// getLayerDefault returns the default value for a layer from infrastructure config or true
func (pg *ProjectGenerator) getLayerDefault(layerType string) bool {
	if pg.config.Layers != nil {
		switch layerType {
		case "base":
			if pg.config.Layers.Base != nil {
				return *pg.config.Layers.Base
			}
		case "foundation":
			if pg.config.Layers.Foundation != nil {
				return *pg.config.Layers.Foundation
			}
		case "organization":
			if pg.config.Layers.Organization != nil {
				return *pg.config.Layers.Organization
			}
		case "security":
			if pg.config.Layers.Security != nil {
				return *pg.config.Layers.Security
			}
		}
	}
	// Default to true if not specified
	return true
}

// isOrganizationLayerEnabled returns true only when the organization layer is enabled in layers
// AND infrastructure.organization is defined with aws_account (required for backend, secrets, SSO).
// If infrastructure.organization is not defined or has no aws_account, organization is not generated.
func (pg *ProjectGenerator) isOrganizationLayerEnabled() bool {
	if pg.config.Organization == nil || pg.config.Organization.AWSAccount == "" {
		return false
	}
	return pg.getLayerDefault("organization")
}

func (pg *ProjectGenerator) isSecurityLayerEnabled() bool {
	if pg.config.Security == nil || pg.config.Security.AWSAccount == "" {
		return false
	}
	return pg.getLayerDefault("security")
}

// IsOrganizationLayerEnabledForConfig reports whether the organization layer should be generated
// for the given infrastructure config. Used by cmd (e.g. dry-run) and tests.
func IsOrganizationLayerEnabledForConfig(config *models.InfrastructureConfig) bool {
	return models.IsOrganizationEnabled(config)
}

// IsSecurityLayerEnabledForConfig reports whether the security layer should be generated for the given infrastructure config.
func IsSecurityLayerEnabledForConfig(config *models.InfrastructureConfig) bool {
	return models.IsSecurityEnabled(config)
}

// IsGitignoreGenerationEnabledForConfig reports whether gocloud generate should write root `.gitignore`.
// Default is true when omitted; set infrastructure.enable_gitignore: false to skip (CLI does not manage the file).
func IsGitignoreGenerationEnabledForConfig(config *models.InfrastructureConfig) bool {
	if config == nil {
		return true
	}
	if config.EnableGitignore != nil {
		return *config.EnableGitignore
	}
	return true
}

// GetEnabledLayersFromConfig returns all enabled layer paths from the configuration
// This function can be used by other commands (like secrets) to determine which layers exist
func GetEnabledLayersFromConfig(config *models.Config) []string {
	var layers []string

	// Check if infrastructure config exists
	if config.Infrastructure == nil {
		return layers
	}

	// Create a temporary generator to use the existing logic
	pg := &ProjectGenerator{config: config.Infrastructure}

	// Check each environment
	for envKey := range config.Infrastructure.Environments {
		// Base layer
		if pg.shouldGenerateLayer("base", envKey) {
			layers = append(layers, fmt.Sprintf("base/%s", envKey))
		}

		// Foundation layer
		if pg.shouldGenerateLayer("foundation", envKey) {
			layers = append(layers, fmt.Sprintf("foundation/%s", envKey))
		}

		// Project layers (always enabled if they exist in config)
		env := config.Infrastructure.Environments[envKey]
		for _, project := range env.Projects {
			projectKey := models.GetProjectKey(project)
			layers = append(layers, fmt.Sprintf("project/%s/%s", projectKey, envKey))
		}

		// Workload layers (always enabled if they exist in config)
		for _, workload := range env.Workloads {
			workloadKey := models.GetWorkloadKey(workload)
			layers = append(layers, fmt.Sprintf("workload/%s/%s", workloadKey, envKey))
		}
	}

	// Organization layer: only when infrastructure.organization.aws_account is set and layer enabled
	if pg.isOrganizationLayerEnabled() {
		layers = append(layers, "organization")
	}
	if pg.isSecurityLayerEnabled() {
		layers = append(layers, "security")
	}

	return layers
}

// getRegionForEnvironment returns the region for a specific environment
// following the hierarchy: environment -> infrastructure -> default
func (pg *ProjectGenerator) getRegionForEnvironment(envKey string) string {
	if pg.config == nil {
		return ""
	}
	return pg.config.RegionForEnvironment(envKey)
}

// generateEnvironmentTable generates a markdown table with environment information
func (pg *ProjectGenerator) generateEnvironmentTable() string {
	if pg.config == nil {
		return "No configuration."
	}
	if len(pg.config.Environments) == 0 && !pg.isOrganizationLayerEnabled() && !pg.isSecurityLayerEnabled() {
		return "No environments configured."
	}

	var table strings.Builder
	table.WriteString("| Environment | Region | Projects | Workloads | Terragrunt | Secrets |\n")
	table.WriteString("|-------------|--------|----------|-----------|------------|----------|\n")

	// Use preserved environment order or fallback to sorted keys
	envKeys := pg.config.GetEnvironmentOrder()

	for _, envKey := range envKeys {
		env := pg.config.Environments[envKey]

		// Get environment name (use key if name is empty)
		envName := env.Name
		if envName == "" {
			envName = envKey
		}

		// Get region (use environment specific or default to global)
		region := env.Region
		if region == "" {
			region = pg.config.Region
		}

		// Get projects list - one per line
		var projectNames []string
		for _, project := range env.Projects {
			projectNames = append(projectNames, models.GetProjectDirectoryName(project))
		}
		var projectsStr string
		if len(projectNames) == 0 {
			projectsStr = "-"
		} else {
			projectsStr = strings.Join(projectNames, "<br>")
		}

		// Get workloads list - one per line
		var workloadNames []string
		for _, workload := range env.Workloads {
			workloadNames = append(workloadNames, models.GetWorkloadDirectoryName(workload))
		}
		var workloadsStr string
		if len(workloadNames) == 0 {
			workloadsStr = "-"
		} else {
			workloadsStr = strings.Join(workloadNames, "<br>")
		}

		// Check terragrunt status (simplified - just check environment level)
		terragruntStatus := "✅"
		if env.EnableTerragrunt != nil && !*env.EnableTerragrunt {
			terragruntStatus = "❌"
		} else if pg.config.EnableTerragrunt != nil && !*pg.config.EnableTerragrunt {
			terragruntStatus = "❌"
		}

		// Check secrets status (simplified - just check environment level)
		secretsStatus := "✅"
		if env.EnableSecrets != nil && !*env.EnableSecrets {
			secretsStatus = "❌"
		} else if pg.config.EnableSecrets != nil && !*pg.config.EnableSecrets {
			secretsStatus = "❌"
		}

		table.WriteString(fmt.Sprintf("| %s | `%s` | %s | %s | %s | %s |\n",
			envName, region, projectsStr, workloadsStr, terragruntStatus, secretsStatus))
	}

	// Organization layer row (global, not per-environment)
	if pg.isOrganizationLayerEnabled() {
		region := pg.config.Region
		terragruntStatus := "✅"
		if pg.config.EnableTerragrunt != nil && !*pg.config.EnableTerragrunt {
			terragruntStatus = "❌"
		}
		secretsStatus := "✅"
		if pg.config.EnableSecrets != nil && !*pg.config.EnableSecrets {
			secretsStatus = "❌"
		}
		table.WriteString(fmt.Sprintf("| Organization | `%s` | - | - | %s | %s |\n",
			region, terragruntStatus, secretsStatus))
	}
	if pg.isSecurityLayerEnabled() {
		region := pg.config.Region
		terragruntStatus := "✅"
		if pg.config.EnableTerragrunt != nil && !*pg.config.EnableTerragrunt {
			terragruntStatus = "❌"
		}
		secretsStatus := "✅"
		if pg.config.EnableSecrets != nil && !*pg.config.EnableSecrets {
			secretsStatus = "❌"
		}
		table.WriteString(fmt.Sprintf("| Security | `%s` | - | - | %s | %s |\n",
			region, terragruntStatus, secretsStatus))
	}

	return table.String()
}

// generateCommandExamples generates updated command examples
func (pg *ProjectGenerator) generateCommandExamples() string {
	var examples strings.Builder

	examples.WriteString("### **Global Commands**\n\n")
	examples.WriteString("```bash\n")
	examples.WriteString("# Initialize and update all modules\n")
	examples.WriteString("terragrunt init --all\n\n")
	examples.WriteString("# Plan all infrastructure\n")
	examples.WriteString("terragrunt plan -concise --all\n")
	examples.WriteString("```\n\n")

	examples.WriteString("### **Environment Commands**\n\n")

	// Use preserved environment order or fallback to sorted keys
	envKeys := pg.config.GetEnvironmentOrder()

	// Generate environment-specific examples in the same order as defined
	for _, envKey := range envKeys {
		// Skip synthetic global layer keys (not real environment dirs)
		if envKey == "org" || envKey == "sec" {
			continue
		}

		env := pg.config.Environments[envKey]
		envName := env.Name
		if envName == "" {
			envName = envKey
		}

		// Get the correct directory name using getDirectoryName
		dirName := pg.getDirectoryName(envKey)

		examples.WriteString(fmt.Sprintf("#### **Example: %s Environment**\n\n", envName))
		examples.WriteString("```bash\n")
		examples.WriteString("# Initialize\n")
		examples.WriteString(fmt.Sprintf("terragrunt init --all --queue-include-dir \"*/%s\" --queue-include-dir \"*/*/%s\"\n\n", dirName, dirName))
		examples.WriteString("# Plan\n")
		examples.WriteString(fmt.Sprintf("terragrunt plan -concise --all --queue-include-dir \"*/%s\" --queue-include-dir \"*/*/%s\"\n\n", dirName, dirName))
		examples.WriteString("# Apply\n")
		examples.WriteString(fmt.Sprintf("terragrunt apply --all --queue-include-dir \"*/%s\" --queue-include-dir \"*/*/%s\"\n", dirName, dirName))
		examples.WriteString("```\n\n")
	}

	examples.WriteString("### **Specific Directory Commands**\n\n")
	examples.WriteString("You can also work with specific directories using `--working-dir`:\n\n")
	examples.WriteString("```bash\n")
	examples.WriteString("# Initialize a specific directory\n")
	examples.WriteString("terragrunt init --working-dir=./base/shared/\n\n")
	examples.WriteString("# Plan a specific directory\n")
	examples.WriteString("terragrunt plan --working-dir=./base/shared/\n\n")
	examples.WriteString("# Apply a specific directory\n")
	examples.WriteString("terragrunt apply --working-dir=./base/shared/\n")
	examples.WriteString("```\n\n")

	// Organization layer commands (when enabled)
	if pg.isOrganizationLayerEnabled() {
		examples.WriteString("### **Organization Layer**\n\n")
		examples.WriteString("```bash\n")
		examples.WriteString("# Initialize organization layer\n")
		examples.WriteString("terragrunt init --working-dir=./organization/\n\n")
		examples.WriteString("# Plan organization layer\n")
		examples.WriteString("terragrunt plan --working-dir=./organization/\n\n")
		examples.WriteString("# Apply organization layer\n")
		examples.WriteString("terragrunt apply --working-dir=./organization/\n")
		examples.WriteString("```\n\n")
	}
	if pg.isSecurityLayerEnabled() {
		examples.WriteString("### **Security Layer**\n\n")
		examples.WriteString("```bash\n")
		examples.WriteString("# Initialize security layer\n")
		examples.WriteString("terragrunt init --working-dir=./security/\n\n")
		examples.WriteString("# Plan security layer\n")
		examples.WriteString("terragrunt plan --working-dir=./security/\n\n")
		examples.WriteString("# Apply security layer\n")
		examples.WriteString("terragrunt apply --working-dir=./security/\n")
		examples.WriteString("```\n\n")
	}

	return examples.String()
}

// buildAlignedMetadataLines renders deterministic, aligned metadata lines for metadata.tf.
func (pg *ProjectGenerator) buildAlignedMetadataLines(customMetadata map[string]interface{}) []string {
	if len(customMetadata) == 0 {
		return nil
	}

	var keys []string
	for key := range customMetadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	maxKeyLength := 0
	for _, key := range keys {
		if len(key) > maxKeyLength {
			maxKeyLength = len(key)
		}
	}

	metadataLines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := customMetadata[key]
		alignedLine := fmt.Sprintf("    %-*s = \"%s\"", maxKeyLength, key, value)
		metadataLines = append(metadataLines, alignedLine)
	}

	return metadataLines
}
