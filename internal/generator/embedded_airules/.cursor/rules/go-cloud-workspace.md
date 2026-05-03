---
description: gocloud.yaml vs hand-edited .tf stacks; prefer OpenTofu (tofu); GoCloud CLI + skills; English in-repo docs.
alwaysApply: true
---

# `gocloud.yaml` vs stacks (quick reference)

## Documentation language

Write comments, docstrings, README and other developer-facing documentation in the workspace repository (including committed rules/skills used by coding agents and generated stack banners where you add custom lines) in English. Chat with the user may use their language; committed technical text stays in English.

### Scope of shared guidance

These rules and skills describe shared GoCloud CLI + Standard Platform patterns. Do not treat them as a source of tenant-specific values (company/client names, AWS account IDs, ARNs, profiles, domains, or numeric pins). Those always come from each repository’s `gocloud.yaml`, stack `main.tf`, and secrets — not from generic docs.

Single source of truth for workspace shape backed by [GoCloud CLI](https://github.com/gocloudLa/gocloud-cli). Full annotated example: [`example/gocloud.yaml`](https://github.com/gocloudLa/gocloud-cli/blob/main/example/gocloud.yaml).

Provisioning CLI: prefer OpenTofu (`tofu`) over Terraform for plan/apply/validate (`.tf` syntax is unchanged).

## What YAML drives

| Area | Controlled by YAML (then `gocloud generate`) |
|------|-----------------------------------------------|
| Standard Platform module pin | `infrastructure.version` (writes `version` in each `main.tf`; optional `source` + `source_ref` for Git modules) |
| Accounts / envs | `infrastructure.environments.<key>` (+ `layers`, `projects`, `workloads`, optional `region` override) |
| Domains → metadata | `infrastructure.metadata` → generated `metadata.tf` / `locals.metadata` |
| State / IAM auth glue | `backend`, `providers` (scopes: global → env → project/workload per CLI docs) |
| SSO | `aws_sso` + `gocloud sso setup` (`.aws/config` profiles) |
| Secrets scaffolding | `enable_secrets`, `secrets.type` (`ssm` / `sops`) → `_secrets.tf`; payloads via `gocloud secrets edit …` |
| Terragrunt / .gitignore | `enable_terragrunt`, `enable_gitignore` |
| Org / Security layers | `organization` (+ optional `security` with `security.aws_account` in richer configs) |

## Mandatory: run `gocloud generate --force` after any YAML change

Whenever you edit `gocloud.yaml` (adding projects, workloads, environments, bumping version, changing backends, etc.), you MUST run `gocloud generate --force` **immediately after the YAML edit and before creating or editing any `.tf` files in the affected stacks**. The CLI creates the directory tree and all generated files (`metadata.tf`, `backend.tf`, `providers.tf`, `_secrets.tf`, `terragrunt.hcl`, scaffold `main.tf`). Only after the CLI has run should you add or modify `main.tf` parameters, `data_sources.tf`, `conditions.tf`, or other hand-edited files. Never manually create stack directories or generated files that the CLI owns.

## What you edit manually in-repo

- `main.tf` per stack: `metadata = local.metadata`, `local.secrets.*` as needed, all `*_parameters` — CLI keeps the module body; it only bumps `module` `version` when `infrastructure.version` changes (see `main.tf` banner).
- Other `.tf` in a stack: `tofu-conventions.md`; typical hand edits: `locals.tf`, `data_sources.tf`, `conditions.tf`.
- Prefer `gocloud.yaml` + `gocloud generate` for new stacks, accounts, project/workload lists, bumping platform globally, backend/SSO/provider templates — not for isolated `_*_parameters` tweaks.

## Agent workflow

Deep behavior, overrides, `--force` risks, SSO/secrets/generate commands: `@` skill `gocloud-cli`. `@` `standard-platform-wrappers` for `*_parameters` → Registry/README. `workspace-mcp.md`, `mcp.json`.

## Repo layout reminder

Stacks live under:

- `organization/` (and `security/` if configured) — no `environment_dir` branch (unlike base/foundation/project/workload).
- `base/<environment_dir>/`, `foundation/<environment_dir>/` — aligned with `gocloud.yaml` environments.
- `project/<project_name>/<environment_dir>/`, `workload/<project_name>/<environment_dir>/` — `project_name` comes from `projects` / `workloads`; `environment_dir` is CLI-derived (YAML env key `dir_name` can override).

- Standard Platform module path: each stack’s `main.tf` uses `gocloudLa/standard-platform/aws//modules/<layer>`, where `<layer>` is one of `organization`, `base`, `foundation`, `project`, `workload` (same idea as the folder kind: org stacks vs base stacks, etc.).
- `*_parameters` is a naming pattern, not one block: open `main.tf` and pass configuration through arguments whose names end with `_parameters` — for example `vpc_parameters`, `alb_parameters`, `identity_center_parameters`. The `*` is shorthand for “whatever prefix”; there is no resource or block literally called `_*_parameters`.
- Which blocks exist per layer: use the table in `standard-platform-layers.md` (layers ↔ typical `…_parameters` lists).

Authoritative pin: `gocloud.yaml` `infrastructure.version` → `main.tf` `version` (not ad-hoc branch README examples; Git `source` overrides Registry pin).
