---
description: MCP — public `.tf` module registry (Terraform Registry API; use `tofu` locally) and AWS docs.
alwaysApply: true
---

# MCP (Terraform Registry + AWS docs)

| Server | Use |
|--------|-----|
| `terraform-mcp` | Public module docs (Registry API): `get_latest_module_version` + `get_module_details` — `.tf` modules consumable by `tofu`. |
| `aws-documentation` | [AWS public documentation](https://docs.aws.amazon.com/) only — not your account. |

Registry module lookup (OpenTofu / `.tf`):

1. `get_latest_module_version`: `module_publisher` = `gocloudLa`, `module_provider` = `aws`, `module_name` = Registry name (e.g. `wrapper-ecs-service`, `wrapper-alb`, `wrapper-rds`).
2. `get_module_details`: `module_id` = `gocloudLa/<module_name>/aws/<version>` — use the version from step 1, or the `version` pinned in the target stack’s `main.tf` / `gocloud.yaml` if you must match that stack.

Optional discovery: `search_modules` with `module_query` = short Registry name (`wrapper-ecs-service`), then `get_module_details` with the returned `module_id`.

If the MCP is down, open the module on the [Terraform Registry](https://registry.terraform.io/) (same modules work with `tofu`) or use `@` `standard-platform-wrappers`.
