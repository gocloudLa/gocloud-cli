---
description: OpenTofu-preferred stacks; Standard Platform wrappers only; secrets via local.secrets; no raw aws_* beyond wrappers.
globs: "**/*.tf"
alwaysApply: false
---

# OpenTofu / `.tf` stack rules

Prefer the OpenTofu CLI (`tofu`) over HashiCorp Terraform for init, plan, apply, validate (syntax/state remain Terraform-compatible `.tf`).

## No redundant `resource "aws_*"`

- Configure only the Standard Platform `module` per layer/stack and the appropriate `*_parameters` / optional `*_defaults`.
- Keep a single layer module block in each stack (`module "organization"`, `module "base"`, `module "foundation"`, `module "project"`, or `module "workload"`) and include `metadata = local.metadata`.
- Do not add standalone `resource "aws_..."` for capabilities already covered by a wrapper (ALB, VPC, RDS, ECS, Lambda, …).
- If something is unsupported, say so and point to upgrading the pin, opening an issue, or using the mapped wrapper’s README — do not invent raw resources silently.

## Secrets, inputs, trivial `locals`

- Do not add `variables.tf`. Stack values come from `local.secrets.<name>` (`_secrets.tf` JSON). Secret keys must use lowercase `snake_case` (for example `local.secrets.app_db_password`). Populate with `gocloud secrets edit <stack-directory>` if a key is missing.
- Do not add `locals.tf` only to alias strings/composites for `main.tf` — put those expressions in `main.tf` unless it is strictly necessary or the stack already does it that way. Same for `conditions.tf`: keep only the zone `locals` template below unless strictly necessary or already extended.

## File layout & style

- `main.tf`, `metadata.tf` (generated), `data_sources.tf` / `conditions.tf` as needed, `_secrets.tf` (defines `local.secrets`).
- Match existing stacks: `/*----------------------------------------------------------------------*/` section separators, titled blocks (`/* VPC Parameters */`).
- In `main.tf` section titles, always use `Parameters` (for example `/* ALB Parameters */`), not `Variables`.
- After the GoCloud `main.tf` banner, no lines before the `module` block; elsewhere avoid long narrative comments—the HCL should stay readable without paragraphs of documentation (short clarifications next to non-obvious bits are fine).
- If a stack needs `local.zone_*` but `conditions.tf` is missing (and GoCloud has not generated it yet), add **only** this pattern when necessary—prefer `gocloud generate` once the CLI emits this file:
  ```hcl
  locals {
    zone_public   = local.metadata.key.env == "prd" ? local.metadata.public_domain : "${local.metadata.key.env}.${local.metadata.public_domain}"
    zone_private  = local.metadata.key.env == "prd" ? local.metadata.private_domain : "${local.metadata.key.env}.${local.metadata.private_domain}"
    zone_internal = local.metadata.key.env == "prd" ? local.metadata.internal_domain : "${local.metadata.key.env}.${local.metadata.internal_domain}"
  }
  ```

## Ask instead of guessing

- Confirm environment / project when ambiguous (for example core vs example, dev vs prd).
- If required module inputs are unknown (for example subnet IDs, certificate ARN, ECS cluster name), ask; do not fabricate.
