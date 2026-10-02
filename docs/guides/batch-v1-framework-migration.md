---
subcategory: ""
page_title: "Job and CronJob Framework migration"
description: |-
  Configuration changes, state compatibility, and upgrade steps for the Job and CronJob v1 Plugin Framework migration.
---

# Job and CronJob Framework migration

`kubernetes_job_v1` and `kubernetes_cron_job_v1` use Terraform Plugin Framework. The resource type names, `namespace/name` import IDs, and existing state are kept, and both resources still manage `batch/v1` objects. A few nested blocks now use list-of-object assignment syntax.

## Configuration changes

Only the following paths change from block syntax to list-of-object assignment syntax:

| Job path | CronJob path |
| --- | --- |
| `spec.selector` | `spec.job_template.spec.selector` |
| `spec.selector.match_expressions` | `spec.job_template.spec.selector.match_expressions` |
| `spec.template.spec.image_pull_secrets` | `spec.job_template.spec.template.spec.image_pull_secrets` |
| `spec.template.spec.readiness_gate` | `spec.job_template.spec.template.spec.readiness_gate` |
| `spec.template.spec.container.resources` | `spec.job_template.spec.template.spec.container.resources` |
| `spec.template.spec.init_container.resources` | `spec.job_template.spec.template.spec.init_container.resources` |

`selector` and each container's `resources` accept at most one object. Maps such as `match_labels`, `limits`, and `requests` keep their map syntax. All other blocks, including `metadata`, `spec`, `job_template`, `template`, `container`, `init_container`, `pod_failure_policy`, `volume`, `affinity`, and `timeouts`, remain blocks. Similarly named blocks under affinity, topology spread constraints, or volume claim templates do not change.

### Job selector

Before:

```terraform
selector {
  match_labels = {
    workload = "daily-report"
  }
  match_expressions {
    key      = "workload"
    operator = "In"
    values   = ["daily-report"]
  }
}
```

After:

```terraform
selector = [{
  match_labels = {
    workload = "daily-report"
  }
  match_expressions = [{
    key      = "workload"
    operator = "In"
    values   = ["daily-report"]
  }]
}]
```

### Image pull secrets and readiness gates

Before:

```terraform
image_pull_secrets {
  name = "registry-credentials"
}

readiness_gate {
  condition_type = "example.com/ready"
}
```

After:

```terraform
image_pull_secrets = [{
  name = "registry-credentials"
}]

readiness_gate = [{
  condition_type = "example.com/ready"
}]
```

### Container and init container resources

Before:

```terraform
container {
  name    = "report"
  image   = "busybox:1.36"
  command = ["sh", "-c", "echo reporting"]

  resources {
    limits = {
      cpu    = "200m"
      memory = "64Mi"
    }
    requests = {
      cpu    = "100m"
      memory = "32Mi"
    }
  }
}
```

After:

```terraform
container {
  name    = "report"
  image   = "busybox:1.36"
  command = ["sh", "-c", "echo reporting"]

  resources = [{
    limits = {
      cpu    = "200m"
      memory = "64Mi"
    }
    requests = {
      cpu    = "100m"
      memory = "32Mi"
    }
  }]
}
```

Make the same edit inside every `init_container`.

### Replace dynamic blocks with list expressions

A `dynamic` block cannot generate one of the changed arguments. Replace it with a list expression:

```terraform
dynamic "image_pull_secrets" {
  for_each = var.image_pull_secret_names
  content {
    name = image_pull_secrets.value
  }
}
```

becomes:

```terraform
image_pull_secrets = [
  for secret_name in var.image_pull_secret_names : {
    name = secret_name
  }
]
```

Dynamic blocks for unchanged blocks, such as `container` and `volume`, remain valid.

## State compatibility

Stored values remain lists of objects, so indexed references such as `kubernetes_job_v1.demo.spec[0].template[0].spec[0].container[0].resources[0].limits["cpu"]` do not change.

Omitted optional values are stored as `null` rather than zero or empty values such as `0` or `""`. State written by earlier provider versions can therefore show a one-time in-place update on the first plan. This update changes only Terraform state, not the Job or CronJob in Kubernetes, and the following plan is empty. Outputs that read these values can change from an empty value to `null`.

`backoff_limit_per_index` and `max_failed_indexes` keep their provider default of `0` when omitted.

## Upgrade an existing resource

1. Back up state and keep the previous provider lock file.
2. Update the provider version and make the syntax changes above without changing other values.
3. Run `terraform plan`. Keeping the same resource address requires no `moved` block or re-import.
4. Investigate any replacement before applying; a syntax-only change must not replace a Job or CronJob.
5. Apply the reviewed plan, then confirm that another plan is empty.

## Move from a deprecated resource type

With Terraform 1.8 or later, rename the resource declaration, update references and the nested syntax, and add the matching `moved` block:

```terraform
moved {
  from = kubernetes_job.demo
  to   = kubernetes_job_v1.demo
}

moved {
  from = kubernetes_cron_job.demo
  to   = kubernetes_cron_job_v1.demo
}
```

Include only the moves that match resources in your state, and do not keep both declarations managing the same object. The move can include the one-time state update described above.

~> The deprecated `kubernetes_cron_job` resource uses the `batch/v1beta1` API, which newer clusters no longer serve. Confirm the CronJob is available through `batch/v1` before moving it.

## Import

Import IDs remain `namespace/name`:

```shell
terraform import kubernetes_job_v1.demo default/demo
terraform import kubernetes_cron_job_v1.demo default/demo
```

For Jobs, `wait_for_completion` is not stored in Kubernetes and is set to `true` on import. If the configuration sets it to `false`, the next plan shows an in-place update that changes only state.

## Other behavior changes

- Changing a Job's `pod_failure_policy` rules requires replacement, because Kubernetes does not allow updating them. The same change in a CronJob template is an in-place update.
- Job updates apply changes to `ttl_seconds_after_finished` and `max_failed_indexes`.
- CronJob updates keep labels and annotations that are not recorded in state, such as internal `kubernetes.io/` keys and keys matched by the provider `ignore_labels` and `ignore_annotations` settings.
