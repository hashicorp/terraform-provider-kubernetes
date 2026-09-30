---
subcategory: ""
page_title: "Job and CronJob Framework migration"
description: |-
  Breaking configuration changes, state compatibility, and upgrade steps for the Job and CronJob v1 Framework migration.
---

# Job and CronJob Framework migration

This guide describes the breaking changes when `kubernetes_job_v1` and `kubernetes_cron_job_v1` move from SDKv2 to the Terraform Plugin Framework. The resource type names and Kubernetes `batch/v1` API remain unchanged. The before examples use SDKv2 syntax, including that of provider version `3.2.1`; the after examples require a provider build containing this migration. This guide does not identify a released provider version containing the migration.

## Breaking changes at a glance

- Selected nested blocks now require **list-of-object attribute** syntax, such as `resources = [{ ... }]`. Their values remain lists, so indexed references do not change.
- A `dynamic` block cannot generate one of these attributes. Replace it with a list expression.
- Absent optional, non-computed leaf fields can be represented as `null` instead of SDKv2 zero or empty values. Review outputs and explicit empty values as well as resource actions.
- Editing a Job's immutable `pod_failure_policy` rules now requires replacement instead of an ineffective in-place update. Corresponding CronJob template rule edits remain in-place updates.

Other blocks remain blocks. In particular, keep `metadata`, `spec`, `job_template`, `template`, `container`, `init_container`, `pod_failure_policy`, `volume`, `affinity`, and `timeouts` in block syntax.

## Required configuration changes

Only the following paths change from block syntax to list-of-object assignment syntax:

| Job path | CronJob path |
| --- | --- |
| `spec.selector` | `spec.job_template.spec.selector` |
| `spec.selector.match_expressions` | `spec.job_template.spec.selector.match_expressions` |
| `spec.template.spec.image_pull_secrets` | `spec.job_template.spec.template.spec.image_pull_secrets` |
| `spec.template.spec.readiness_gate` | `spec.job_template.spec.template.spec.readiness_gate` |
| `spec.template.spec.container.resources` | `spec.job_template.spec.template.spec.container.resources` |
| `spec.template.spec.init_container.resources` | `spec.job_template.spec.template.spec.init_container.resources` |

`selector` and each container's `resources` accept at most one object. `match_expressions`, `image_pull_secrets`, and `readiness_gate` accept lists of objects. Maps such as `match_labels`, `limits`, and `requests` keep their existing map syntax.

Scalar types and defaults are not changed by this syntax conversion: `ttl_seconds_after_finished` remains a string, and `manual_selector` still defaults to `false`.

Do not convert similarly named blocks under affinity, topology spread constraints, or volume claim templates. This is a path-specific change, not a global replacement of every `selector`, `match_expressions`, or `resources` block.

The examples below are fragments to replace inside an existing resource. Preserve the surrounding blocks and existing values; do not add a selector, secret, or readiness condition solely for this migration.

### Job selector and match expressions

Inside the Job's `spec` block, or the CronJob's `spec.job_template.spec` block:

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

Keep the existing `manual_selector` setting and matching Pod template labels. Most Jobs should continue to omit a manual selector so Kubernetes can generate one. Changing a Job's selector values, rather than just their HCL syntax, requires replacement of the Job. Review the actual CronJob plan separately; do not infer its update or replacement behavior from Job immutability.

### Image pull secrets and readiness gates

Inside the Pod template's `spec` block:

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

The secret must already exist in the Pod's namespace, and a controller must set any custom readiness condition. These requirements are unchanged.

### Container and init container resources

Both `container` and `init_container` remain blocks. Only their nested `resources` change:

Before:

```terraform
init_container {
  name    = "prepare"
  image   = "busybox:1.36"
  command = ["sh", "-c", "echo preparing"]

  resources {
    requests = {
      cpu    = "100m"
      memory = "32Mi"
    }
  }
}

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
init_container {
  name    = "prepare"
  image   = "busybox:1.36"
  command = ["sh", "-c", "echo preparing"]

  resources = [{
    requests = {
      cpu    = "100m"
      memory = "32Mi"
    }
  }]
}

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

The Job Pod template remains immutable: changing its resource values requires replacement of the Job. CronJob template changes are not uniformly replacement-only; update versus replacement depends on the edited field and the resulting plan. Kubernetes can update CronJob templates for subsequently created Jobs, but that API capability does not guarantee an in-place Terraform update for every field. Review the actual plan and its replacement reasons; do not combine template value changes with the syntax-only migration.

For example, removing a configured container `env` block is a workload change: it replaces the Job but updates the CronJob in place. This difference matches the released provider's behavior; it is not a reason to replace either resource solely for the migration.

For a CronJob with other settings unchanged, editing its container image or container CPU limit updates the existing CronJob in place. Changing `spec.job_template.spec.completions`, however, requires replacement. These field-specific actions also match provider `3.2.1`; do not infer one action for every field in the Job template.

Changing a rule within a Job's `pod_failure_policy` now requires replacement because Kubernetes makes the policy immutable. The SDK implementation could previously plan an in-place rule edit without applying it; requiring replacement corrects that ineffective update rather than preserving its plan action. Equivalent rule edits within a CronJob template remain in-place updates. Adding or removing the policy block still requires replacement for either resource.

### Replace dynamic blocks with list expressions

For an existing `image_pull_secret_names` variable of type `list(string)`, replace this fragment inside the Pod template's `spec` block:

```terraform
dynamic "image_pull_secrets" {
  for_each = var.image_pull_secret_names
  content {
    name = image_pull_secrets.value
  }
}
```

With:

```terraform
image_pull_secrets = [
  for secret_name in var.image_pull_secret_names : {
    name = secret_name
  }
]
```

Apply the same approach to dynamic blocks at the other changed paths. Dynamic blocks for unchanged blocks, such as `container` and `volume`, remain valid.

## State and output compatibility

The state collection types remain lists of objects. Do not remove list indexes from expressions simply because configuration uses attribute syntax. For example, these outputs are unchanged when the referenced containers have `resources` configured:

```terraform
output "job_name" {
  value = kubernetes_job_v1.demo.metadata[0].name
}

output "job_cpu_limit" {
  value = kubernetes_job_v1.demo.spec[0].template[0].spec[0].container[0].resources[0].limits["cpu"]
}

output "cron_job_cpu_limit" {
  value = kubernetes_cron_job_v1.demo.spec[0].job_template[0].spec[0].template[0].spec[0].container[0].resources[0].limits["cpu"]
}
```

The migration retains these internal schema versions:

| Resource | State schema version | Identity schema version |
| --- | --- | --- |
| `kubernetes_job_v1` | `1` | `1` |
| `kubernetes_cron_job_v1` | `0` | `1` |

Job state schema version `0` is upgraded to version `1`, including historical container and init container resource limits/requests. These are internal state versions, not provider release numbers.

### Null and empty values

Framework distinguishes omission (`null`) from explicit empty values such as `""`, `[]`, and `{}`, and from explicit zero or `false`. For absent optional, non-computed leaves, state and outputs may now contain `null` where SDKv2 exposed a zero or empty value. The exact result depends on the field and prior state.

For example, plans from provider `3.2.1` state show the following representation changes for omitted values. Paths below start at the Job specification: `spec` for a Job, or `spec.job_template.spec` for a CronJob.

| Relative path | Legacy state | Framework plan |
| --- | --- | --- |
| `active_deadline_seconds` | `0` | `null` |
| `backoff_limit_per_index` | `0` | `null` |
| `max_failed_indexes` | `0` | `null` |
| `ttl_seconds_after_finished` | `""` | `null` |
| `template.metadata.generate_name` | `""` | `null` |
| `template.spec.active_deadline_seconds` | `0` | `null` |
| `template.spec.container.working_dir` | `""` | `null` |
| `template.spec.priority_class_name` | `""` | `null` |
| `template.spec.runtime_class_name` | `""` | `null` |
| `template.spec.subdomain` | `""` | `null` |

CronJob `spec.job_template.metadata.generate_name` can likewise change from `""` to `null`. These examples describe plan values, not permission to apply any accompanying resource replacement. Do not configure zero or empty values merely to reproduce old state: explicit values can have different API semantics, and `active_deadline_seconds`, for example, requires a positive integer when configured.

An explicit empty value and an omitted value may therefore produce a state-only change or a change to an output even when no Kubernetes object changes. Review expressions that index optional collections, call functions on possibly null values, or rely on empty-string/empty-map outputs. Use explicit null handling where the module's output contract requires it.

Upgrades from provider `3.2.1` can require a **one-time in-place Terraform update to normalize state**, rather than an empty first migration plan. In the verified minimal, generated-name, and populated Job/CronJob migration cases, applying that normalization preserved the Kubernetes UID and complete desired API specification, made no create/update/patch API requests during migration, and was followed by an empty plan. This describes those normalization cases, not a guarantee for every existing configuration.

Do not assume every post-upgrade diff is harmless normalization. Review the plan's resource actions, replacement reasons, and output changes separately. No blanket no-op plan is promised for every existing configuration.

For unchanged configuration, a normalization-only upgrade must not delete or recreate the Job or CronJob, or change its desired workload specification. When verifying an upgrade, check both the Terraform actions and continuity of the Kubernetes object's UID and desired specification. A successful apply or an in-place action alone is not enough to establish that only state changed.

### Quantity formatting and admission-populated values

Equivalent Kubernetes quantity spellings do not introduce a formatting-related breaking change. For both resources, changing CPU quantities from `"1000m"` to `"1"` or memory quantities from `"1024Mi"` to `"1Gi"` in container and init container limits/requests produces no resource action when the quantity is otherwise unchanged. Actual quantity changes are different and must be reviewed in the plan.

The null normalization described above concerns absent non-computed leaves, not every field omitted from configuration. Admission-populated container `resources` and `requests`, Pod `image_pull_secrets`, and `readiness_gate` values can still be recorded in their optional/computed attributes. Keeping the configuration unchanged preserves these admitted values without requiring the user to copy them into configuration.

## Upgrading an existing v1 resource

If the resource already uses `kubernetes_job_v1` or `kubernetes_cron_job_v1`, retain its Terraform address. **No `moved` block or re-import is required for a same-type provider upgrade.**

1. Back up the current state securely and retain the previous provider lock file. State and saved plans may contain sensitive values; do not commit them.
2. Update the provider selection to a build containing this migration and make the path-specific HCL changes above.
3. Run `terraform validate` and `terraform plan`.
4. Do not apply replacement or deletion solely to accommodate this migration. Investigate those actions, and separate syntax-only edits from changes to immutable Job fields.
5. Apply only a reviewed plan, then run another plan and inspect outputs. For normalization-only changes, confirm the same Kubernetes UID and unchanged desired workload specification.

The minimal examples on the [Job resource page](../resources/job_v1.md#example-usage---no-waiting) and [CronJob resource page](../resources/cron_job_v1.md#example-usage) use ordinary blocks and do not need these syntax edits.

## Moving from deprecated resource types

Changing a deprecated resource type to its versioned type is different from a same-type provider upgrade. Rename the resource declaration, retain its Kubernetes name and namespace, update references and the affected nested syntax, and add the matching `moved` block:

```terraform
moved {
  from = kubernetes_job.demo
  to   = kubernetes_job_v1.demo
}
```

```terraform
moved {
  from = kubernetes_cron_job.demo
  to   = kubernetes_cron_job_v1.demo
}
```

Do not retain both resource declarations managing the same Kubernetes object. Use a Terraform CLI that supports provider-assisted moves between resource types. Review the resulting plan rather than treating a successful state conversion as proof of a no-op migration.

Moving from `kubernetes_job` can include the same one-time in-place state-normalization update described above. The verified Job alias move preserves the object's UID and desired specification without remote writes, then produces an empty subsequent plan. A `moved` block does not justify replacement solely to perform the migration.

The provider accepts JSON state from `hashicorp/kubernetes` for these source types:

| Source resource type | Accepted source state schema versions | Target resource type |
| --- | --- | --- |
| `kubernetes_job` | `0`, `1` | `kubernetes_job_v1` |
| `kubernetes_cron_job` | `0`, `1` | `kubernetes_cron_job_v1` |

Historical version `0` container and init container limits/requests are converted to maps. The CronJob conversion also initializes the previously absent `timezone` field without selecting a new time zone. Legacy flatmap state is not supported by these conversions. The stored `namespace/name` ID and metadata must identify the same object.

~> The deprecated `kubernetes_cron_job` resource remains an SDKv2 resource using the `batch/v1beta1` endpoint. A state conversion does not restore that API on servers that no longer serve it. Refreshing or creating the deprecated resource can fail on newer clusters. Verify the existing object is available through `batch/v1` and check cluster/API compatibility before attempting a move; do not assume that an old beta-resource lifecycle can run on a current server.

## Import

For an existing Kubernetes object not already managed at another Terraform address, import IDs remain `namespace/name`:

```shell
terraform import kubernetes_job_v1.demo default/demo
terraform import kubernetes_cron_job_v1.demo default/demo
```

Create the corresponding resource configuration first and review a plan after import. Do not import again to perform a same-type upgrade or use import to manage the same object under two addresses.

For Jobs, `wait_for_completion` defaults to `true` on import, including identity-based imports. This is a provider-only setting that Kubernetes does not store, so the importer cannot recover a previous `false` value. If the configuration sets `wait_for_completion = false`, an in-place state-only update after import is expected to reconcile that setting. Changing only this setting does not change the Job's API specification and must not require replacement; review any other plan differences separately.

## Related behavior corrections

- Job updates now send changes to `ttl_seconds_after_finished` and `max_failed_indexes` to Kubernetes, including clearing those optional fields. Normal Kubernetes validation and feature availability still apply.
- CronJob updates preserve server-managed metadata and unmanaged labels/annotations on the CronJob, Job template, and Pod template instead of replacing them with only Terraform-managed metadata.
- Job `create`, `update`, and `delete` timeouts each default to `1m`. `wait_for_completion` defaults to `true` and controls waiting for completion on create/update only. Deletion still uses its own timeout and waits for the Job to disappear. Replacement creates use the `create` timeout, not the in-place `update` timeout.

These corrections can affect a plan or apply independently of the HCL syntax conversion. Review them alongside the breaking changes rather than assuming the migration only changes implementation internals.
