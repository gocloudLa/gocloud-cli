# GoCloud CLI — reference excerpts

Stable links:

- Repo: [github.com/gocloudLa/gocloud-cli](https://github.com/gocloudLa/gocloud-cli)
- Example YAML: [example/gocloud.yaml](https://github.com/gocloudLa/gocloud-cli/blob/main/example/gocloud.yaml)

## `gocloud generate` behavior (documentation summary)

From the CLI README (paraphrased):

- Validates YAML; errors abort, unknown fields warn.
- Writes directory tree per organization, security (if configured), base, foundation, project, workload.
- Writes `main.tf`, `metadata.tf`, `terragrunt.hcl`, `backend.tf`, `providers.tf`, `_secrets.tf` (when secrets enabled), conditional extras, `.gitignore` (unless `enable_gitignore: false`), project `README.md`.
- `main.tf` is never overwritten as a whole: only `version = "…"` updates when configured platform version changes. Other `main.tf` edits remain manual.

Use `gocloud generate --dry-run` to preview. `--force` overwrites many generated assets (still not full `main.tf` replace) — risky if files were tailored by hand.

## Header on generated `main.tf`

Stacks include a banner like:

```text
# This file is generated and maintained by GoCloud CLI
# You CAN edit this file manually to add your custom configuration
# GoCloud CLI will only update the module version when needed
```

So Standard Platform inputs (`*_parameters`) live here by team choice; the CLI preserves them across runs.

## Hierarchy notes (modern example YAML)

The [full example](https://raw.githubusercontent.com/gocloudLa/gocloud-cli/main/example/gocloud.yaml) shows:

- `infrastructure.layers`: toggles generation of organization, security, base, foundation globally.
- Per-environment `layers` can disable base/foundation for an account.
- Projects/workloads: simple list entries (e.g. `- myproject`) or richer maps (`name`, `dir_name`, `backend`, `providers`, `enable_terragrunt`, …).
- Organization activates when `organization.aws_account` is set (`layers.organization` not `false`).

Individual workspace repositories often ship a smaller `gocloud.yaml` than the full example; the shape and fields still follow the same model.
