# Wrapper mapping reference

Prefer `tofu` over `terraform` for local plan/apply/validate (`.tf` unchanged).

Authoritative pins for your workspace: each stack `main.tf` (`module "wrapper_*"` blocks) and `gocloud.yaml` `infrastructure.version`. The tables below are a snapshot for quick lookup — update this file when you bump Standard Platform, or rely on `main.tf` when they disagree.

Standard Platform repo: [terraform-aws-standard-platform](https://github.com/gocloudLa/terraform-aws-standard-platform/releases).

## Organization (`modules/organization/main.tf`)

| Parameter | Registry module |
|-----------|----------------|
| `organization_parameters` | `gocloudLa/wrapper-organization/aws` 1.1.0 |
| `identity_center_parameters` | `gocloudLa/wrapper-identity-center/aws` 1.1.0 |
| `s3_backend_parameters` | `gocloudLa/wrapper-s3-backend/aws` 1.0.3 |

## Base (`modules/base/main.tf`)

| Parameter | Registry module |
|-----------|----------------|
| `vpc_parameters` | `gocloudLa/wrapper-vpc/aws` 1.2.1 |
| `route53_parameters` | `gocloudLa/wrapper-route53-zone/aws` 1.0.0 |
| `cloudmap_parameters` | `gocloudLa/wrapper-cloudmap/aws` 1.0.0 |
| `notifications_parameters` | `gocloudLa/wrapper-notifications/aws` 1.1.4 |

### `feature/vpc-upgrade` deltas (examples)

Uses Git sources for vpc/tgw/vpn/route53/cloudmap wrappers (pinned by `ref=` on private GitHub repos) and `gocloudLa/wrapper-tgw` / `gocloudLa/wrapper-vpn` (+ `vpc_parameter`/`tgw_parameter` wiring). Prefer reading that branch’s `modules/base/main.tf` before assuming registry-only pins.

## Foundation (`modules/foundation/main.tf`)

| Parameter | Registry module |
|-----------|----------------|
| `acm_parameters` | `gocloudLa/wrapper-acm/aws` 1.1.1 |
| `gitlab_runner_parameters` | `gocloudLa/wrapper-gitlab-runner/aws` 1.2.0 |
| `iam_parameters` | `gocloudLa/wrapper-iam/aws` 0.1.2 |
| `aws_backup_parameters` | `gocloudLa/wrapper-aws-backup/aws` 1.0.0 |
| `ses_parameters` | `gocloudLa/wrapper-ses/aws` 1.0.0 |
| `pritunl_parameters` | `gocloudLa/wrapper-pritunl/aws` 1.0.2 |
| `route53_parameters` | `gocloudLa/wrapper-route53-record/aws` 1.0.1 |
| `service_scheduler_parameters` | `gocloudLa/wrapper-service-scheduler/aws` 1.1.3 |
| `waf_parameters` | `gocloudLa/wrapper-waf/aws` 1.0.1 |
| `health_events_parameters` | `gocloudLa/wrapper-health-events/aws` 1.0.0 |
| `cost_control_parameters` | `gocloudLa/wrapper-cost-control/aws` 1.0.0 |

Foundation Route53 records ≠ Base hosted zones: `wrapper-route53-record` vs Base `wrapper-route53-zone`.

## Project (`modules/project/main.tf`)

| Parameter | Registry module |
|-----------|----------------|
| `alb_parameters` | `gocloudLa/wrapper-alb/aws` 1.3.1 |
| `batch_parameters` | `gocloudLa/wrapper-batch/aws` 1.0.0 |
| `ecs_parameters` | `gocloudLa/wrapper-ecs/aws` 1.0.3 |
| `ecr_parameters` | `gocloudLa/wrapper-ecr/aws` 0.1.1 |
| `eks_parameters` | `gocloudLa/wrapper-eks/aws` 1.0.1 |
| `elasticache_parameters` | `gocloudLa/wrapper-elasticache/aws` 1.6.2 |
| `documentdb_parameters` | `gocloudLa/wrapper-documentdb/aws` 1.0.2 |
| `rds_parameters` | `gocloudLa/wrapper-rds/aws` 1.1.2 |
| `rds_aurora_parameters` | `gocloudLa/wrapper-rds-aurora/aws` 1.3.0 |
| `sqs_parameters` | `gocloudLa/wrapper-sqs/aws` 1.0.3 |
| `dynamodb_parameters` | `gocloudLa/wrapper-dynamodb/aws` 1.1.1 |
| `bucket_parameters` | `gocloudLa/wrapper-bucket/aws` 1.1.1 |
| `efs_parameters` | `gocloudLa/wrapper-efs/aws` 1.0.1 |
| `memorydb_parameters` | `gocloudLa/wrapper-memorydb/aws` 1.2.2 |
| `kms_parameters` | `gocloudLa/wrapper-kms/aws` 0.1.0 |
| `kinesis_stream_parameters` | `gocloudLa/wrapper-kinesis-stream/aws` 0.1.0 |

### `feature/vpc-upgrade` deltas

Adds `opensearch_parameters` → `gocloudLa/wrapper-opensearch/aws` 0.1.0; bumps some wrapper versions vs older releases (e.g. `wrapper-rds` 1.2.0, `wrapper-rds-aurora` 1.3.1). After upgrading the platform pin, re-read `modules/project/main.tf` on that release tag.

## Workload (`modules/workload/main.tf`)

| Parameter | Registry module |
|-----------|----------------|
| `static_site_parameters` | `gocloudLa/wrapper-static-site/aws` 1.0.4 |
| `ecs_service_parameters` | `gocloudLa/wrapper-ecs-service/aws` 1.4.2 |
| `batch_job_parameters` | `gocloudLa/wrapper-batch-job/aws` 1.0.4 |
| `lambda_parameters` | `gocloudLa/wrapper-lambda/aws` 1.0.1 |
| `ec2_instance_parameters` | `gocloudLa/wrapper-ec2-instance/aws` 0.1.4 |

## Links

- Platform overview: [`terraform-aws-standard-platform` README](https://github.com/gocloudLa/terraform-aws-standard-platform).
- vpc-upgrade exploration: [`feature/vpc-upgrade` tree](https://github.com/gocloudLa/terraform-aws-standard-platform/tree/feature/vpc-upgrade).
- Registry pattern: `https://registry.terraform.io/modules/gocloudLa/wrapper-alb/aws/<version>` (swap module name + version to match the stack).
