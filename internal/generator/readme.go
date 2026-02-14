package generator

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"gocloud-cli/internal/templates"

	"gopkg.in/yaml.v3"
)

// ReadmeGenerator handles the generation of README files from templates
type ReadmeGenerator struct {
	YamlFile     string
	OutputFile   string
	TemplateURL  string
	IsLocalFile  bool
	TerraformDir string
	IsExample    bool
}

// ReadmeData represents the structure of the YAML configuration file
type ReadmeData struct {
	ModuleName        string                 `yaml:"module_name"`
	ModuleDescription string                 `yaml:"module_description"`
	ModuleBadges      []Badge                `yaml:"module_badges"`
	Features          []Feature              `yaml:"features"`
	QuickStart        string                 `yaml:"quick_start"`
	InputTable        string                 `yaml:"input_table"`
	OutputsTable      string                 `yaml:"outputs_table"`
	Examples          string                 `yaml:"examples"`
	ImportantNotes    string                 `yaml:"important_notes"`
	ExternalModules   []ExternalModule       `yaml:"external_modules"`
	InputsTablePretty string                 `yaml:"inputs_table_pretty"`
	Extra             map[string]interface{} `yaml:",inline"`
}

// Badge represents a module badge
type Badge struct {
	URL   string `yaml:"url"`
	Image string `yaml:"image"`
	Alt   string `yaml:"alt"`
}

// Feature represents a module feature
type Feature struct {
	Icon             string    `yaml:"icon"`
	Title            string    `yaml:"title"`
	ShortDescription string    `yaml:"short_description"`
	LongDescription  string    `yaml:"long_description"`
	Examples         []Example `yaml:"examples"`
}

// Example represents a feature example
type Example struct {
	Title string `yaml:"title"`
	Code  string `yaml:"code"`
}

// ExternalModule represents an external Terraform module
type ExternalModule struct {
	Name    string `yaml:"name"`
	URL     string `yaml:"url"`
	Version string `yaml:"version"`
}

// ExampleData represents the structure of the example YAML configuration file
type ExampleData struct {
	Title        string   `yaml:"title"`
	Description  string   `yaml:"description"`
	MainPurpose  string   `yaml:"main_purpose"`
	KeyFeatures  []string `yaml:"key_features"`
	ServicesUsed []string `yaml:"services_used,omitempty"`
}

// Generate creates the README file from the template and data
func (rg *ReadmeGenerator) Generate() error {
	// Load YAML configuration
	data, err := rg.loadYamlData()
	if err != nil {
		return fmt.Errorf("failed to load YAML data: %w", err)
	}

	// Detect external modules automatically
	externalModules, err := rg.detectExternalModules()
	if err != nil {
		return fmt.Errorf("failed to detect external modules: %w", err)
	}
	data.ExternalModules = externalModules

	// Process inputs table and format it as markdown
	rg.processInputsTable(data)

	// Load template file (local or remote)
	templateContent, err := rg.loadTemplate()
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Render template with data
	readmeContent, err := rg.renderTemplate(templateContent, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write final README file (remove trailing newline if present)
	content := strings.TrimSuffix(readmeContent, "\n")
	return os.WriteFile(rg.OutputFile, []byte(content), 0644)
}

// detectExternalModules detects external Terraform modules using terraform CLI
func (rg *ReadmeGenerator) detectExternalModules() ([]ExternalModule, error) {
	// Default terraform directory if not specified
	terraformDir := rg.TerraformDir
	if terraformDir == "" {
		terraformDir = "examples/complete"
	}

	// Check if terraform directory exists
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		fmt.Printf("⚠️  Terraform directory '%s' not found, skipping module detection\n", terraformDir)
		return []ExternalModule{}, nil
	}

	fmt.Printf("🔍 Detecting external modules in '%s'...\n", terraformDir)

	// Change to terraform directory and initialize
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(terraformDir); err != nil {
		return nil, fmt.Errorf("failed to change to terraform directory: %w", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			log.Printf("Warning: failed to change back to original directory: %v", err)
		}
	}()

	// Run terraform init
	initCmd := exec.Command("terraform", "init", "-input=false", "-backend=false")
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		return nil, fmt.Errorf("terraform init failed: %w", err)
	}

	// Run terraform modules
	modulesCmd := exec.Command("terraform", "modules")
	output, err := modulesCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("terraform modules failed: %w", err)
	}

	// Parse the output
	modules, err := rg.parseTerraformModulesOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("failed to parse terraform modules output: %w", err)
	}

	fmt.Printf("✅ Found %d external modules\n", len(modules))
	return modules, nil
}

// parseTerraformModulesOutput parses the output of 'terraform modules' command
func (rg *ReadmeGenerator) parseTerraformModulesOutput(output string) ([]ExternalModule, error) {
	lines := strings.Split(output, "\n")
	uniqueModules := make(map[string]ExternalModule)

	// Regex to match module lines like: "eventbridge_create_dump"[registry.terraform.io/terraform-aws-modules/eventbridge/aws] 4.1.0
	moduleRegex := regexp.MustCompile(`.*"([^"]+)"\[([^]]+)\][[:space:]]+([0-9]+\.[0-9]+\.[0-9]+).*`)

	for _, line := range lines {
		matches := moduleRegex.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}

		modulePath := strings.TrimSpace(matches[2])
		version := strings.TrimSpace(matches[3])

		// Skip non-registry modules
		if !strings.HasPrefix(modulePath, "registry.terraform.io/") {
			continue
		}

		// Extract namespace, name, and provider
		pathParts := strings.Split(strings.TrimPrefix(modulePath, "registry.terraform.io/"), "/")
		if len(pathParts) < 3 {
			continue
		}

		namespace := pathParts[0]
		name := pathParts[1]
		provider := pathParts[2]

		// Create module info
		module := ExternalModule{
			Name:    fmt.Sprintf("%s/%s/%s", namespace, name, provider),
			URL:     fmt.Sprintf("https://github.com/%s/terraform-%s-%s", namespace, provider, name),
			Version: version,
		}

		// Use URL + version as unique key to avoid duplicates
		uniqueKey := fmt.Sprintf("%s|%s", module.URL, version)
		uniqueModules[uniqueKey] = module
	}

	// Convert map to slice and sort by name
	var modules []ExternalModule
	for _, module := range uniqueModules {
		modules = append(modules, module)
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})

	return modules, nil
}

// GenerateExample creates an example README file from the template and data
func (rg *ReadmeGenerator) GenerateExample() error {
	// Load YAML configuration
	data, err := rg.loadExampleYamlData()
	if err != nil {
		return fmt.Errorf("failed to load example YAML data: %w", err)
	}

	// Load template file (local or remote)
	templateContent, err := rg.loadTemplate()
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Render template with data
	readmeContent, err := rg.renderExampleTemplate(templateContent, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Write final README file (remove trailing newline if present)
	content := strings.TrimSuffix(readmeContent, "\n")
	return os.WriteFile(rg.OutputFile, []byte(content), 0644)
}

// loadYamlData loads and parses the YAML configuration file
func (rg *ReadmeGenerator) loadYamlData() (*ReadmeData, error) {
	data, err := os.ReadFile(rg.YamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	var readmeData ReadmeData
	if err := yaml.Unmarshal(data, &readmeData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &readmeData, nil
}

// processInputsTable parses the input_table from YAML and formats it as a markdown table
func (rg *ReadmeGenerator) processInputsTable(data *ReadmeData) {
	if data.InputTable == "" {
		data.InputsTablePretty = ""
		return
	}

	// Define table structure
	header := []string{"Name", "Description", "Type", "Default", "Required"}
	var rows [][]string

	// Parse each line from input_table
	for _, line := range strings.Split(data.InputTable, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
			parts := strings.Split(line[1:len(line)-1], "|")
			if len(parts) == len(header) {
				// Clean up each cell
				for i, part := range parts {
					parts[i] = strings.TrimSpace(part)
				}
				rows = append(rows, parts)
			}
		}
	}

	// Generate formatted markdown table
	if len(rows) > 0 {
		data.InputsTablePretty = rg.prettifyMarkdownTable(rows, header)
	} else {
		data.InputsTablePretty = ""
	}
}

// prettifyMarkdownTable creates a properly aligned markdown table
func (rg *ReadmeGenerator) prettifyMarkdownTable(rows [][]string, header []string) string {
	allRows := append([][]string{header}, rows...)
	colWidths := make([]int, len(header))

	// Calculate maximum width for each column
	for i := range header {
		for _, row := range allRows {
			if i < len(row) && len(row[i]) > colWidths[i] {
				colWidths[i] = len(row[i])
			}
		}
	}

	// Build table with header, separator, and rows
	var result []string
	result = append(result, rg.formatRow(header, colWidths))
	result = append(result, rg.formatSeparator(colWidths))
	for _, row := range rows {
		result = append(result, rg.formatRow(row, colWidths))
	}

	return strings.Join(result, "\n")
}

// formatRow formats a single table row with proper alignment
func (rg *ReadmeGenerator) formatRow(row []string, colWidths []int) string {
	var cells []string
	for i, cell := range row {
		if i < len(colWidths) {
			cells = append(cells, fmt.Sprintf("%-*s", colWidths[i], cell))
		}
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

// formatSeparator creates the markdown table separator row
func (rg *ReadmeGenerator) formatSeparator(colWidths []int) string {
	var cells []string
	for _, width := range colWidths {
		cells = append(cells, strings.Repeat("-", width))
	}
	return "| " + strings.Join(cells, " | ") + " |"
}

// renderTemplate processes the template with the provided data
func (rg *ReadmeGenerator) renderTemplate(templateContent string, data *ReadmeData) (string, error) {
	// Create template with custom functions and disable HTML escaping
	tmpl := template.New("readme").Funcs(template.FuncMap{
		"anchorize": func(s string) string {
			// Convert title to anchor format (lowercase, replace spaces with hyphens)
			return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
		},
		"raw": func(s string) string {
			// Return raw HTML without escaping
			return s
		},
	})

	// Parse template with HTML escaping disabled
	tmpl, err := tmpl.Option("missingkey=error").Parse(templateContent)
	if err != nil {
		return "", err
	}

	// Execute template with HTML escaping disabled
	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// loadTemplate loads the template content from local file or remote URL
func (rg *ReadmeGenerator) loadTemplate() (string, error) {
	// If no template specified, use embedded template
	if rg.TemplateURL == "" && !rg.IsLocalFile {
		return rg.loadEmbeddedTemplate()
	}

	if rg.IsLocalFile {
		content, err := os.ReadFile(rg.TemplateURL)
		if err != nil {
			return "", fmt.Errorf("failed to read local template: %w", err)
		}
		return string(content), nil
	}

	// Convert GitHub blob URL to raw URL for direct download
	rawURL := strings.Replace(rg.TemplateURL, "github.com", "raw.githubusercontent.com", 1)
	rawURL = strings.Replace(rawURL, "/blob/", "/", 1)

	resp, err := http.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch remote template: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch template: HTTP %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read template content: %w", err)
	}

	return string(content), nil
}

// loadExampleYamlData loads and parses the example YAML configuration file
func (rg *ReadmeGenerator) loadExampleYamlData() (*ExampleData, error) {
	data, err := os.ReadFile(rg.YamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	var exampleData ExampleData
	if err := yaml.Unmarshal(data, &exampleData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &exampleData, nil
}

// renderExampleTemplate renders the example template with the provided data
func (rg *ReadmeGenerator) renderExampleTemplate(templateContent string, data *ExampleData) (string, error) {
	// Create template with custom functions
	tmpl := template.New("example").Funcs(template.FuncMap{
		"anchorize": func(s string) string {
			// Convert title to anchor format (lowercase, replace spaces with hyphens)
			return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
		},
		"raw": func(s string) string {
			// Return raw HTML without escaping
			return s
		},
	})

	// Parse template
	tmpl, err := tmpl.Option("missingkey=error").Parse(templateContent)
	if err != nil {
		return "", err
	}

	// Execute template
	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", err
	}

	return result.String(), nil
}

// loadEmbeddedTemplate loads the appropriate embedded template based on the generator type
func (rg *ReadmeGenerator) loadEmbeddedTemplate() (string, error) {
	if rg.IsExample {
		return templates.ExampleTemplate, nil
	}
	return templates.ReadmeTemplate, nil
}
