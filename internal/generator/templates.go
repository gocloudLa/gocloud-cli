package generator

import (
	"bytes"
	"fmt"
	"text/template"

	"gocloud-cli/internal/models"
)

// TemplateEngine handles template rendering
type TemplateEngine struct {
	templates map[string]*template.Template
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() *TemplateEngine {
	engine := &TemplateEngine{
		templates: make(map[string]*template.Template),
	}
	engine.loadTemplates()
	return engine
}

// Render renders a template with the given data
func (te *TemplateEngine) Render(templateName string, data *models.TemplateData) (string, error) {
	tmpl, exists := te.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

// loadTemplates loads all built-in templates
func (te *TemplateEngine) loadTemplates() {
	// GoCloud CLI generates providers.tf and backend.tf directly
	te.templates["terragrunt.hcl.tpl"] = template.Must(template.New("terragrunt.hcl").Parse(terragruntTemplate))
	te.templates["metadata.tf.tpl"] = template.Must(template.New("metadata.tf").Parse(metadataTemplate))
	te.templates["_secrets.tf.tpl"] = template.Must(template.New("_secrets.tf").Parse(secretsTemplate))

	// Layer-specific main.tf templates
	te.templates["main.tf.base.tpl"] = template.Must(template.New("main.tf.base").Parse(mainBaseTemplate))
	te.templates["main.tf.foundation.tpl"] = template.Must(template.New("main.tf.foundation").Parse(mainFoundationTemplate))
	te.templates["main.tf.project.tpl"] = template.Must(template.New("main.tf.project").Parse(mainProjectTemplate))
	te.templates["main.tf.workload.tpl"] = template.Must(template.New("main.tf.workload").Parse(mainWorkloadTemplate))
	te.templates["main.tf.organization.tpl"] = template.Must(template.New("main.tf.organization").Parse(mainOrganizationTemplate))

	// Provider and backend templates
	te.templates["providers.tf.tpl"] = template.Must(template.New("providers.tf").Parse(providersTemplate))
	te.templates["backend.tf.tpl"] = template.Must(template.New("backend.tf").Parse(backendTemplate))
}

// Template definitions

const terragruntTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# DO NOT EDIT MANUALLY - Changes will be overwritten on next generation
# =============================================================================

include "root" {
  path = find_in_parent_folders("root.hcl")
}{{if .Dependencies}}

dependencies {
  paths = [
    {{range .Dependencies}}
    "{{.}}",
    {{end}}
  ]
}
{{end}}`

const metadataTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# DO NOT EDIT MANUALLY - Changes will be overwritten on next generation
# =============================================================================

locals {

  metadata = {
    aws_region  = "{{.Region}}"
    environment = "{{.EnvironmentName}}"{{if .Project}}
    project     = "{{.ProjectName}}"{{end}}{{if .MetadataLines}}
{{range .MetadataLines}}
{{.}}{{end}}{{end}}

    key = {
      company = "{{.Company}}"
      region  = "{{.RegionShortCode}}"
      env     = "{{.Environment}}"{{if .Project}}
      project = "{{.Project}}"{{end}}
      layer   = "{{.Layer}}"
    }
  }

  common_name_prefix = join("-", [
    local.metadata.key.company,
    local.metadata.key.env
  ]){{if .Project}}

  common_name = join("-", [
    local.common_name_prefix,
    local.metadata.key.project
  ]){{else}}

  common_name = local.common_name_prefix{{end}}

}
`

const secretsTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# DO NOT EDIT MANUALLY - Changes will be overwritten on next generation
# =============================================================================
{{if eq .SecretsBackendType "sops"}}
terraform {
  required_providers {
    sops = {
      source  = "carlpett/sops"
      version = "1.3.0"
    }
  }
}

provider "sops" {}

data "sops_file" "secrets" {
  source_file = "${path.module}/_secrets.yaml"
}

locals {
  sops_secrets = nonsensitive(yamldecode(data.sops_file.secrets.raw))
}
{{else}}
data "aws_ssm_parameter" "terraform" {
  name = "/terraform/${local.common_name}-{{.Layer}}"
}

locals {
  secrets = jsondecode(data.aws_ssm_parameter.terraform.value)
}
{{end}}`

// Layer-specific main.tf templates
const mainBaseTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# You CAN edit this file manually to add your custom configuration
# GoCloud CLI will only update the module version when needed
# =============================================================================

module "base" {
  {{if .IsGitSource}}
  source = "{{.Source}}//modules/base?ref={{.SourceRef}}"
  {{else}}
  source  = "gocloudLa/standard-platform/aws//modules/base"
  version = "{{.Version}}"
  {{end}}

  /*----------------------------------------------------------------------*/
  /* General Variables                                                    */
  /*----------------------------------------------------------------------*/

  metadata = local.metadata

}
`

const mainFoundationTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# You CAN edit this file manually to add your custom configuration
# GoCloud CLI will only update the module version when needed
# =============================================================================

module "foundation" {
  {{if .IsGitSource}}
  source = "{{.Source}}//modules/foundation?ref={{.SourceRef}}"
  {{else}}
  source  = "gocloudLa/standard-platform/aws//modules/foundation"
  version = "{{.Version}}"
  {{end}}

  providers = {
    aws.use1 = aws.use1
  }

  /*----------------------------------------------------------------------*/
  /* General Variables                                                    */
  /*----------------------------------------------------------------------*/

  metadata = local.metadata

}
`

const mainProjectTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# You CAN edit this file manually to add your custom configuration
# GoCloud CLI will only update the module version when needed
# =============================================================================

module "project" {
  {{if .IsGitSource}}
  source = "{{.Source}}//modules/project?ref={{.SourceRef}}"
  {{else}}
  source  = "gocloudLa/standard-platform/aws//modules/project"
  version = "{{.Version}}"
  {{end}}

  /*----------------------------------------------------------------------*/
  /* General Variables                                                    */
  /*----------------------------------------------------------------------*/

  metadata = local.metadata

}
`

const mainWorkloadTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# You CAN edit this file manually to add your custom configuration
# GoCloud CLI will only update the module version when needed
# =============================================================================

module "workload" {
  {{if .IsGitSource}}
  source = "{{.Source}}//modules/workload?ref={{.SourceRef}}"
  {{else}}
  source  = "gocloudLa/standard-platform/aws//modules/workload"
  version = "{{.Version}}"
  {{end}}

  providers = {
    aws.use1 = aws.use1
  }

  /*----------------------------------------------------------------------*/
  /* General Variables                                                    */
  /*----------------------------------------------------------------------*/

  metadata = local.metadata

}
`

const mainOrganizationTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# You CAN edit this file manually to add your custom configuration
# GoCloud CLI will only update the module version when needed
# =============================================================================

module "organization" {
  {{if .IsGitSource}}
  source = "{{.Source}}//modules/organization?ref={{.SourceRef}}"
  {{else}}
  source  = "gocloudLa/standard-platform/aws//modules/organization"
  version = "{{.Version}}"
  {{end}}

}
`

// Provider template
const providersTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# DO NOT EDIT MANUALLY - Changes will be overwritten on next generation
# =============================================================================

{{range .Providers}}
provider "{{.Name}}" {
{{if .Region}}  region  = {{.RegionHCL}}
{{end}}{{if .Alias}}  alias   = "{{.Alias}}"
{{end}}{{if .Profile}}  profile = "{{.Profile}}"
{{end}}{{if .AssumeRole}}  assume_role {
    role_arn     = "{{.AssumeRole.RoleARN}}"{{if .AssumeRole.SessionName}}
    session_name = "{{.AssumeRole.SessionName}}"{{end}}
  }
{{end}}{{range $key, $value := .Extra}}  {{$key}} = {{$value}}
{{end}}}
{{end}}
`

// Backend template
const backendTemplate = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# DO NOT EDIT MANUALLY - Changes will be overwritten on next generation
# =============================================================================

terraform {
  backend "{{.BackendType}}" {
    bucket         = "{{.BackendBucket}}"
    key            = "{{.BackendKey}}"
    region         = "{{.BackendRegion}}"{{if .BackendDynamoDBTable}}
    dynamodb_table = "{{.BackendDynamoDBTable}}"{{end}}{{if .BackendEncrypt}}
    encrypt        = {{.BackendEncrypt}}{{end}}{{if .BackendProfile}}
    profile        = "{{.BackendProfile}}"{{end}}{{if .BackendAssumeRole}}
    assume_role = {
      role_arn = "{{.BackendAssumeRole.RoleARN}}"
    }{{end}}
  }
}
`
