package cmd

import (
	"fmt"
	"os"

	"gocloud-cli/internal/generator"
	"gocloud-cli/internal/utils"

	"github.com/spf13/cobra"
)

var (
	readmeYamlFile     string
	readmeOutputFile   string
	readmeTemplateURL  string
	readmeTemplateFile string
	readmeTerraformDir string

	// Example command variables
	exampleYamlFile     string
	exampleOutputFile   string
	exampleTemplateURL  string
	exampleTemplateFile string
)

// moduleCmd represents the module command
var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Manage Terraform modules",
	Long:  `Manage and generate documentation for Terraform modules`,
}

// readmeCmd represents the readme command
var readmeCmd = &cobra.Command{
	Use:   "readme",
	Short: "Manage README files for modules",
	Long:  `Generate and manage README files for Terraform modules`,
}

// readmeGenerateCmd represents the readme generate command
var readmeGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate README.md from template and YAML configuration",
	Long: `Generate a README.md file using a Jinja2 template and YAML configuration.
	
This command replicates the functionality of the GitHub Action readme generator,
allowing you to generate READMEs locally using the GoCloud CLI.

Examples:
  gocloud module readme generate
  gocloud module readme generate --yaml custom.yml --output CUSTOM.md
  gocloud module readme generate --template-file ./local-template.md.gotmpl`,
	RunE: runReadmeGenerate,
}

// readmeGenerateExampleCmd represents the readme generate-example command
var readmeGenerateExampleCmd = &cobra.Command{
	Use:   "generate-example",
	Short: "Generate README.md for examples from template and YAML configuration",
	Long: `Generate a README.md file for examples using a Go template and YAML configuration.
	
This command generates README files for example directories, similar to the Python
generate_examples.py script but using Go templates.

Examples:
  gocloud module readme generate-example
  gocloud module readme generate-example --yaml examples/complete/README.yml --output examples/complete/README-new.md
  gocloud module readme generate-example --template-file ./local-template.md.gotmpl`,
	RunE: runReadmeGenerateExample,
}

func init() {
	// Add readme command to module (moduleCmd is registered in root.go)
	moduleCmd.AddCommand(readmeCmd)

	// Add readme generate command to readme
	readmeCmd.AddCommand(readmeGenerateCmd)
	readmeCmd.AddCommand(readmeGenerateExampleCmd)

	// Flags for readme generate
	readmeGenerateCmd.Flags().StringVarP(&readmeYamlFile, "yaml", "y", "README.yml", "Input YAML file")
	readmeGenerateCmd.Flags().StringVarP(&readmeOutputFile, "output", "o", "README.md", "Output README file")
	readmeGenerateCmd.Flags().StringVar(&readmeTemplateURL, "template-url", "", "Template URL")
	readmeGenerateCmd.Flags().Lookup("template-url").Usage = "Template URL (default: GoCloud template) (Example: \"https://github.com/gocloudLa/.github/blob/main/.github/readme-generator/README.md.gotmpl\")"
	readmeGenerateCmd.Flags().StringVarP(&readmeTemplateFile, "template-file", "t", "", "Local template file (overrides template-url)")
	readmeGenerateCmd.Flags().StringVar(&readmeTerraformDir, "terraform-dir", "examples/complete", "Terraform directory for module detection")

	// Flags for readme generate-example
	readmeGenerateExampleCmd.Flags().StringVarP(&exampleYamlFile, "yaml", "y", "README.yml", "Input YAML file")
	readmeGenerateExampleCmd.Flags().StringVarP(&exampleOutputFile, "output", "o", "README.md", "Output README file")
	readmeGenerateExampleCmd.Flags().StringVar(&exampleTemplateURL, "template-url", "", "Template URL")
	readmeGenerateExampleCmd.Flags().Lookup("template-url").Usage = "Template URL (default: GoCloud example template) (Example \"https://github.com/gocloudLa/.github/blob/main/.github/readme-generator/README_example.md.gotmpl\")"
	readmeGenerateExampleCmd.Flags().StringVarP(&exampleTemplateFile, "template-file", "t", "", "Local template file (overrides template-url)")
}

func runReadmeGenerate(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Generating README from template...")

	// Validate input file exists
	if _, err := os.Stat(readmeYamlFile); os.IsNotExist(err) {
		utils.PrintText("Input YAML file not found: %s\n", readmeYamlFile)
		return nil // Exit silently with success code
	}

	// Determine template source
	var templateSource string
	var useEmbedded bool

	if readmeTemplateFile != "" {
		// Use local template file
		if _, err := os.Stat(readmeTemplateFile); os.IsNotExist(err) {
			utils.PrintText("Local template file not found: %s\n", readmeTemplateFile)
			return nil // Exit silently with success code
		}
		templateSource = readmeTemplateFile
		utils.PrintText("📁 Using local template: %s\n", readmeTemplateFile)
	} else if cmd.Flags().Changed("template-url") {
		// Use remote template URL (user explicitly specified --template-url)
		templateSource = readmeTemplateURL
		utils.PrintText("🌐 Using remote template: %s\n", readmeTemplateURL)
	} else {
		// Use embedded template (default behavior when no template specified)
		useEmbedded = true
		utils.PrintText("📦 Using embedded template\n")
	}

	// Create generator instance
	gen := &generator.ReadmeGenerator{
		YamlFile:     readmeYamlFile,
		OutputFile:   readmeOutputFile,
		TemplateURL:  templateSource,
		IsLocalFile:  readmeTemplateFile != "",
		TerraformDir: readmeTerraformDir,
		IsExample:    false,
	}

	// If using embedded template, clear the TemplateURL
	if useEmbedded {
		gen.TemplateURL = ""
	}

	// Generate README
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}

	utils.PrintText("✅ README generated successfully: %s\n", readmeOutputFile)
	return nil
}

func runReadmeGenerateExample(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Generating example README from template...")

	// Validate input file exists
	if _, err := os.Stat(exampleYamlFile); os.IsNotExist(err) {
		utils.PrintText("Input YAML file not found: %s\n", exampleYamlFile)
		return nil // Exit silently with success code
	}

	// Determine template source
	var templateSource string
	var useEmbedded bool

	if exampleTemplateFile != "" {
		// Use local template file
		if _, err := os.Stat(exampleTemplateFile); os.IsNotExist(err) {
			utils.PrintText("Local template file not found: %s\n", exampleTemplateFile)
			return nil // Exit silently with success code
		}
		templateSource = exampleTemplateFile
		utils.PrintText("📁 Using local template: %s\n", exampleTemplateFile)
	} else if cmd.Flags().Changed("template-url") {
		// Use remote template URL (user explicitly specified --template-url)
		templateSource = exampleTemplateURL
		utils.PrintText("🌐 Using remote template: %s\n", exampleTemplateURL)
	} else {
		// Use embedded template (default behavior when no template specified)
		useEmbedded = true
		utils.PrintText("📦 Using embedded template\n")
	}

	// Create generator instance
	gen := &generator.ReadmeGenerator{
		YamlFile:    exampleYamlFile,
		OutputFile:  exampleOutputFile,
		TemplateURL: templateSource,
		IsLocalFile: exampleTemplateFile != "",
		IsExample:   true,
	}

	// If using embedded template, clear the TemplateURL
	if useEmbedded {
		gen.TemplateURL = ""
	}

	// Generate example README
	if err := gen.GenerateExample(); err != nil {
		return fmt.Errorf("failed to generate example README: %w", err)
	}

	utils.PrintText("✅ Example README generated successfully: %s\n", exampleOutputFile)
	return nil
}
