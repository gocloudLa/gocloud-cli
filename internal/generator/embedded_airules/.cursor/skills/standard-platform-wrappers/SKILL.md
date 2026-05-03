---
name: standard-platform-wrappers
description: Maps Standard Platform *_parameters blocks to gocloudLa wrapper modules (Terraform Registry `.tf`; use tofu locally). Registry/GitHub docs. For base/foundation/project/workload edits, pins, ALB/VPC/RDS wrappers, standard-platform.
---

# Standard Platform → GoCloud wrappers (docs lookup)

## How to resolve a parameter block to documentation

1. Identify layer (`base` / `foundation` / `project` / `workload` / `organization`) — see `standard-platform-layers.md`.

2. Strip suffix: `alb_parameters` → resource hint `alb`. The backing module is `gocloudLa/wrapper-<name>/aws` where `<name>` uses kebab-case when the HCL identifier has underscores (see table in [reference.md](reference.md)).

3. Pinned versions: In [terraform-aws-standard-platform](https://github.com/gocloudLa/terraform-aws-standard-platform), open `modules/<layer>/main.tf` at the Git tag that matches that workspace’s Standard Platform pin (`gocloud.yaml` → `infrastructure.version`, mirrored in each stack `main.tf` top-level `module` `version`). Read each `module "wrapper_*"` block’s `source` and `version` → [Registry](https://registry.terraform.io/) module paths (`tofu` consumes the same `source` pins).

Updating the standard-platform pin for the whole workspace belongs in `gocloud.yaml` + `gocloud generate` (see `@` `gocloud-cli`), which refreshes `version` lines across stack `main.tf` without overwriting your `*_parameters`.

## Where to read “official module documentation”

- [Terraform Registry](https://registry.terraform.io/) (primary `.tf` source): `/modules/gocloudLa/<wrapper-name>/aws/<version>` — pin version to match the stack’s `main.tf` (example URL shape: `/modules/gocloudLa/wrapper-alb/aws/<version>`).
- Repository: `https://github.com/gocloudLa/terraform-aws-wrapper-<suffix>` — README documents inputs/outputs/examples; submodule layout may expose nested HCL modules.
- `terraform-mcp` (same Registry API): `workspace-mcp.md`. `aws-documentation` = AWS docs only — not Registry modules.

## Upstream branches and pins

Wrapper wiring lives in Standard Platform’s `modules/<layer>/main.tf`. Feature branches (for example `feature/vpc-upgrade`) may add modules or switch some wrappers to Git `source` — compare with your workspace only when exploring upstream; what matters locally is still each stack `main.tf` and `gocloud.yaml` `infrastructure.version`.

## Full mappings

Snapshot tables (wrapper names and example registry versions): [reference.md](reference.md). Treat stack `main.tf` / `gocloud.yaml` as authoritative — refresh `reference.md` after bumping the platform pin so it does not drift.
