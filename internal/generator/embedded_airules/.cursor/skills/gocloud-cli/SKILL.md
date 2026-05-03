---
name: gocloud-cli
description: Explains GoCloud YAML, generate vs manual .tf/OpenTofu edits, and when to run gocloud (generate, SSO, secrets). Use for gocloud.yaml, stacks, pins, SSO, backends, Terragrunt, secrets, overwrite behavior.
---

# GoCloud CLI ↔ workspace

Upstream: [gocloud-cli](https://github.com/gocloudLa/gocloud-cli) — companion to Standard Platform. Example config shape: [`example/gocloud.yaml`](https://github.com/gocloudLa/gocloud-cli/blob/main/example/gocloud.yaml). The CLI does not run OpenTofu (`tofu`) / Terragrunt; it generates and updates structure and glue files.

## `gocloud.yaml` roles (mental model)

- `cli`: Runtime for the tool (`working_dir`, backups, verbosity).
- `infrastructure`: Entire workspace model — `client`, `company`, `version` (Standard Platform Registry tag wired into generated `main.tf`), `region`, optional `source`/`source_ref` (Git module source instead of Registry); `metadata` (domains → `locals.metadata` in `metadata.tf`); `backend`, `aws_sso`, `providers`, `secrets`/`enable_secrets`, `layers`, `enable_terragrunt`/`enable_gitignore`; `organization` (and optionally `security`) blocks; `environments`: accounts, `projects`/`workloads`, per-env `layers`, SSO/backend/provider overrides.

Override resolution documented in CLI README: broadly global → environment → project → workload (more specific wins), with organization/security as special globals.

When you redesign accounts, dirs, backends, SSO, Terragrunt, or bump platform source/version at generation time, `gocloud.yaml` should be edited first, then `gocloud generate`.

Non-interactive use (CI, scripting, or unattended agents): `gocloud generate --force` — otherwise prompts block.

## What the CLI regenerates vs what humans own

Generated/modified by `gocloud generate` (typically safe to regenerate with intent):

- Directory tree: `base/<environment_dir>/`, `foundation/<environment_dir>/`, `project/<project_name>/<environment_dir>/`, `workload/<project_name>/<environment_dir>/` (see `go-cloud-workspace.md`).
- `metadata.tf`, `backend.tf`, `providers.tf`, `terragrunt.hcl` (when enabled), `_secrets.tf` (when secrets enabled), conditions files as applicable, root `.gitignore`, project `README`.
- `main.tf`: never fully overwritten. Only the `version = "…"` line updates when `infrastructure.version` (or scoped override) changes. Banner text states you may edit parameters manually ([see generated header](reference.md)).

Edited by humans (day-to-day Standard Platform wiring):

- `main.tf`: Inside the single `module` call — all `*_parameters` / defaults, data references, conditional blocks the generator does not synthesize end-to-end.
- Any `locals.tf`, `data_sources.tf`, `conditions.tf`, `_secrets.tf` content semantics aligned with stacks (depending on secrets backend).
- Operational fixups that `gocloud generate --force` would otherwise prompt over — avoid blind `--force` on stacks with hand-tuned glue files.

## When to steer the user to CLI vs IDE-only edits

| Goal | Prefer |
|------|--------|
| Add/rename env, account, `projects`/`workloads` list | `gocloud.yaml` + `gocloud generate` |
| Bump Standard Platform `version` for all stacks | `gocloud.yaml` `infrastructure.version` + `gocloud generate` (updates `main.tf` version lines) |
| Switch Registry → Git `source + source_ref` | `gocloud.yaml` + `gocloud generate` |
| SSO profiles / `.aws/config` | `gocloud sso setup` (per README) |
| Edit secrets payloads | `gocloud secrets edit <layer-path>` |
| Tune ALB/VPC/etc. payloads | `main.tf` (and related locals/data) |

## Conflict avoidance

Running `gocloud generate --force` may overwrite generated files besides `main.tf`. Confirm with the team before `--force` if `backend.tf`, `providers.tf`, or `metadata.tf` were hand-customized.

More field-level detail and excerpts: [reference.md](reference.md).
