package generator

// infrastructureGitignoreContent is the default .gitignore written at the generation root by `gocloud generate`
// when infrastructure.enable_gitignore is true (default). Same overwrite rules as providers.tf / backend.tf.
const infrastructureGitignoreContent = `# =============================================================================
# This file is generated and maintained by GoCloud CLI
# DO NOT EDIT MANUALLY - Changes will be overwritten on next generation
# =============================================================================

# Local .terraform directories
**/.terraform/*

# .tfstate files
*.tfstate
*.tfstate.*

# Terraform lock files (optional; many repos commit .terraform.lock.hcl instead)
*.terraform.lock.*

# Crash log files
crash.log
crash.*.log

# tfvars / json tfvars (often secrets)
*.tfvars
*.tfvars.json
*.auto.tfvars
!*.tfvars.example
!*.tfvars.json.example

# Transient lock metadata during apply
.terraform.tfstate.lock.info

# Terragrunt (https://github.com/gruntwork-io/terragrunt/blob/master/.gitignore)
.terragrunt-cache
.terragrunt-stack

# Test / coverage outputs (any tooling)
coverage/
coverage.*
*.lcov
htmlcov/
.coverage
*.cover

# Overrides (.tf and OpenTofu .tofu)
override.tf
override.tofu
override.tf.json
override.tofu.json
*_override.tf
*_override.tofu
*_override.tf.json
*_override.tofu.json

# CLI config
.terraformrc
terraform.rc

# .env secrets; keep templates as .env.example / .env.sample
.env
.env.*
!.env.example
!.env.sample

# OS noise (macOS / Windows / Linux)
.DS_Store
.AppleDouble
.LSOverride
._*
Thumbs.db
ehthumbs.db
Desktop.ini
$RECYCLE.BIN/
*~
.directory
.nfs*

# Ignore builds directories
**/builds/*
`
