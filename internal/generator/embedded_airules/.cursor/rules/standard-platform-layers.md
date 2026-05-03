---
description: Which Standard Platform layer owns which *_parameters blocks and quick examples (ALB, Lambda, VPC, ACM).
alwaysApply: true
---

# Standard Platform — layers vs parameters

| Layer | Repo path pattern | Std Platform `module` source path | Typical `*_parameters` |
|--------|-------------------|---------------------------|-------------------------|
| Organization | `organization/` | `//modules/organization` | `organization_parameters`, `identity_center_parameters`, `s3_backend_parameters` |
| Base | `base/<environment_dir>/` | `//modules/base` | `vpc_parameters`, `route53_parameters`, `cloudmap_parameters`, `notifications_parameters` |
| Foundation | `foundation/<environment_dir>/` | `//modules/foundation` | `acm_parameters`, `gitlab_runner_parameters`, `iam_parameters`, `aws_backup_parameters`, `ses_parameters`, `pritunl_parameters`, `route53_parameters`, `service_scheduler_parameters`, `waf_parameters`, `health_events_parameters`, `cost_control_parameters` |
| Project | `project/<project_name>/<environment_dir>/` | `//modules/project` | `alb_parameters`, `ecs_parameters`, `ecr_parameters`, `eks_parameters`, `elasticache_parameters`, `documentdb_parameters`, `rds_parameters`, `rds_aurora_parameters`, `sqs_parameters`, `dynamodb_parameters`, `bucket_parameters`, `efs_parameters`, `memorydb_parameters`, `kms_parameters`, `kinesis_stream_parameters`, … |
| Workload | `workload/<project_name>/<environment_dir>/` | `//modules/workload` | `static_site_parameters`, `ecs_service_parameters`, `batch_job_parameters`, `lambda_parameters`, `ec2_instance_parameters` |

`project_name`: value from `projects` / `workloads` in `gocloud.yaml`. `environment_dir`: directory the CLI emits per environment — it may differ from the env key in YAML (`dir_name` and CLI rules); see the GoCloud CLI docs and [example/gocloud.yaml](https://github.com/gocloudLa/gocloud-cli/blob/main/example/gocloud.yaml).

Path shape: `project/<project_name>/<environment_dir>/`, `workload/<project_name>/<environment_dir>/` (tracks `gocloud.yaml` — do not enumerate projects in docs).

Quick mental model:

- ALB → Project, `alb_parameters` → resolves to `gocloudLa/wrapper-alb/aws` (see wrapper skill).
- Lambda app → Workload, `lambda_parameters` → `gocloudLa/wrapper-lambda/aws`.
- VPC → Base, `vpc_parameters` → `gocloudLa/wrapper-vpc/aws`.
- ACM → Foundation, `acm_parameters` → `gocloudLa/wrapper-acm/aws`.
