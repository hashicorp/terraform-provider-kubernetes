---
subcategory: ""
page_title: "Kubernetes: Upgrade Guide for Kubernetes Provider v4.0.0"
description: |-
  This guide covers the changes introduced in v4.0.0 of the Kubernetes provider and what you may need to do to upgrade your configuration.
---

# Upgrading to v4.0.0 of the Kubernetes provider

Version 4.0.0 moves six workload resources from the Terraform Plugin SDKv2 to the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework):

- `kubernetes_deployment_v1`
- `kubernetes_daemon_set_v1`
- `kubernetes_stateful_set_v1`
- `kubernetes_pod_v1`
- `kubernetes_job_v1`
- `kubernetes_cron_job_v1`

All six share one pod spec implementation. Their resource type names, `namespace/name` import IDs, import identities and Kubernetes API versions do not change, and existing state is upgraded without re-importing anything. Selected nested blocks become object arguments or lists of objects, which changes their configuration syntax.

All other resources, including the deprecated unversioned types such as `kubernetes_deployment`, are unchanged.

## Who is affected

- Configurations that use any of the six resources above. If they use one of the arguments listed in [Configuration syntax changes](#configuration-syntax-changes), they must be updated before upgrading.
- Workspaces whose state contains any of the six resources: the provider upgrades their state, and 3.x cannot read it afterwards.

When upgrading from 2.x, also follow the [v3 upgrade guide](v3-upgrade-guide.md).

## Upgrade checklist

1. **Start from an empty plan.** With your current 3.x provider, run `terraform plan` and confirm that it shows no changes. Resolve any drift or pending changes first.
2. **Back up state.** Version 4.0.0 increases the state schema version of the six resources. Once it has written state, provider 3.x can no longer read that state, so rolling back requires restoring the backup.
3. **Update the configuration.** Change the version constraint and make the [syntax changes](#configuration-syntax-changes) below, without changing any values:

    ```terraform
    terraform {
      required_providers {
        kubernetes = {
          source  = "hashicorp/kubernetes"
          version = "~> 4.0"
        }
      }
    }
    ```

4. **Install the provider.** Run `terraform init -upgrade`. Recreate saved plans after upgrading the provider and configuration.
5. **Review the plan.** Run `terraform validate` and `terraform plan`. A syntax-only change must not replace or update a workload in Kubernetes. A one-time in-place update that changes only Terraform state is expected in some cases (see [Behavior changes](#behavior-changes)). Investigate any replacement before applying. Run this first plan without `-target`: until a resource's state has been upgraded, a targeted plan warns `Failed to decode resource from state` for the resources it skips.
6. **Apply and confirm.** Apply the reviewed plan, then run `terraform plan` again and confirm that it is empty.

Test the upgrade in a non-production workspace first, and repeat these steps for each workspace.

## Configuration syntax changes

The following arguments change from 3.x nested blocks to attributes assigned with `=`:

| Argument | Resources | v4 value |
| --- | --- | --- |
| `resources` in `container` and `init_container` | All six | One object |
| `strategy` and its nested `rolling_update` | Deployment, DaemonSet | One object each |
| `persistent_volume_claim_retention_policy` | StatefulSet | One object |
| JobSpec `selector` | Job, CronJob | One object |
| `image_pull_secrets` and `readiness_gate` in the pod spec | All six | List of objects |
| `selector.match_expressions` | Job, CronJob | List of objects |

For each object argument, change `name { ... }` to `name = { ... }`. Omit the argument or set it to `null` to leave it unset. An empty object `{}` is valid; an empty list `[]` is the wrong type. Maps such as `limits`, `requests` and `match_labels` stay maps, and expression `values` stays a set of strings.

In `.tf.json` configuration, use the object form `"resources": { ... }`; the array form `"resources": [{ ... }]` is rejected.

The repeated `image_pull_secrets` and `readiness_gate` arguments require a nonempty list when configured. Omit them or use `null` instead of `[]`. `match_expressions = []` is allowed.

The object fields also change shape in state. Update references, `ignore_changes` child paths and module types as described in [References and module interfaces](#references-and-module-interfaces).

All other blocks keep their block syntax, including `metadata`, `spec`, `template`, `container`, `volume`, `affinity`, the StatefulSet `update_strategy` and `volume_claim_template` (and the `resources` and `selector` blocks inside a claim template), the Job `pod_failure_policy`, and selectors inside affinity and topology spread terms.

### Pod spec arguments

These changes apply to the pod spec of every resource: `spec` for a Pod, `spec.template.spec` for a Deployment, DaemonSet, StatefulSet or Job, and `spec.job_template.spec.template.spec` for a CronJob.

Container and init container resources, before:

```terraform
container {
  name  = "app"
  image = "nginx:1.27"

  resources {
    limits = {
      cpu    = "500m"
      memory = "512Mi"
    }
    requests = {
      cpu    = "250m"
      memory = "128Mi"
    }
  }
}
```

After:

```terraform
container {
  name  = "app"
  image = "nginx:1.27"

  resources = {
    limits = {
      cpu    = "500m"
      memory = "512Mi"
    }
    requests = {
      cpu    = "250m"
      memory = "128Mi"
    }
  }
}
```

Make the same change in every `init_container`. Replace an empty `resources {}` block with `resources = {}`. Omitted `limits` and `requests` can keep their prior or API-computed values; use an explicit empty map, such as `limits = {}`, when you intend to clear that map.

Image pull secrets and readiness gates, before:

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

### Dynamic blocks and conditional values

A `dynamic` block cannot generate one of the changed arguments. For an object argument, assign a nullable object directly:

```terraform
resources = var.container_resources
```

To conditionally set the object:

```terraform
resources = var.configure_resources ? var.container_resources : null
```

For a repeated argument, use a list expression and preserve the original order:

```terraform
image_pull_secrets = length(var.image_pull_secret_names) == 0 ? null : [
  for name in var.image_pull_secret_names : {
    name = name
  }
]
```

`dynamic` blocks for unchanged blocks, such as `container` or `volume`, remain valid.

### References and module interfaces

Remove only the indexes belonging to the converted objects. Keep indexes on enclosing blocks and real collections.

Before:

```terraform
output "cpu_limit" {
  value = kubernetes_deployment_v1.example.spec[0].template[0].spec[0].container[0].resources[0].limits["cpu"]
}
```

After:

```terraform
output "cpu_limit" {
  value = kubernetes_deployment_v1.example.spec[0].template[0].spec[0].container[0].resources.limits["cpu"]
}
```

Apply the same change to selected `strategy`, `rolling_update`, retention-policy and JobSpec `selector` references. For example, `spec[0].strategy[0].rolling_update[0].max_surge` becomes `spec[0].strategy.rolling_update.max_surge`. Deployment, DaemonSet and StatefulSet selectors retain their `[0]` because they remain blocks.

Terraform reports stale references a few at a time, so fixing them one error at a time can take several rounds. Instead, search the configuration and its modules for each converted name followed by `[0]` or `.0.`, such as `resources[0]` and `strategy.0.`.

Child paths in `lifecycle.ignore_changes` need the same edit. For a Deployment, this 3.x path:

```terraform
lifecycle {
  ignore_changes = [spec[0].template[0].spec[0].container[0].resources[0].limits]
}
```

becomes:

```terraform
lifecycle {
  ignore_changes = [spec[0].template[0].spec[0].container[0].resources.limits]
}
```

A path that ignores the entire `resources` argument keeps its existing spelling.

Module variables that expose the complete singleton value change from `list(object(...))` to `object(...)`. For example, replace this interface:

```terraform
variable "container_resources" {
  type = list(object({
    limits   = map(string)
    requests = map(string)
  }))
  default = null
}
```

with:

```terraform
variable "container_resources" {
  type = object({
    limits   = optional(map(string))
    requests = optional(map(string))
  })
  default = null
}
```

Optional object attributes require Terraform 1.3 or later. On earlier versions, keep both fields as `map(string)` and pass `null` for an omitted map.

Update callers to pass an object or `null`, and update outputs and downstream consumers that expect a singleton list. Terraform's state upgrade does not rewrite configuration, module interfaces or remote-state consumers.

### kubernetes_deployment_v1

Changed arguments: `spec.strategy`, `spec.strategy.rolling_update`, and the pod spec arguments under `spec.template.spec`.

Before:

```terraform
strategy {
  type = "RollingUpdate"

  rolling_update {
    max_surge       = "25%"
    max_unavailable = "25%"
  }
}
```

After:

```terraform
strategy = {
  type = "RollingUpdate"
  rolling_update = {
    max_surge       = "25%"
    max_unavailable = "25%"
  }
}
```

Do not configure `rolling_update` with the `Recreate` strategy; its read value is `null`. Omitting `strategy` keeps the prior or API-computed strategy. Setting `strategy = {}` selects the default `RollingUpdate`, so it changes an existing `Recreate` strategy.

### kubernetes_daemon_set_v1

Changed arguments: `spec.strategy`, `spec.strategy.rolling_update`, and the pod spec arguments under `spec.template.spec`.

Before:

```terraform
strategy {
  type = "RollingUpdate"

  rolling_update {
    max_surge       = "0"
    max_unavailable = "1"
  }
}
```

After:

```terraform
strategy = {
  type = "RollingUpdate"
  rolling_update = {
    max_surge       = "0"
    max_unavailable = "1"
  }
}
```

Keep the DaemonSet's own values; its defaults (`max_surge = "0"`, `max_unavailable = "1"`) differ from a Deployment's. Do not configure `rolling_update` with the `OnDelete` strategy; its read value is `null`. Setting `strategy = {}` selects `RollingUpdate`, including when the existing strategy is `OnDelete`; omitting it keeps the prior or API-computed strategy.

### kubernetes_stateful_set_v1

Changed arguments: `spec.persistent_volume_claim_retention_policy`, and the pod spec arguments under `spec.template.spec`.

Before:

```terraform
persistent_volume_claim_retention_policy {
  when_deleted = "Delete"
  when_scaled  = "Retain"
}
```

After:

```terraform
persistent_volume_claim_retention_policy = {
  when_deleted = "Delete"
  when_scaled  = "Retain"
}
```

`persistent_volume_claim_retention_policy = {}` selects `Retain` for both children, including when an existing policy uses `Delete`. Preserve the previous values during a syntax-only upgrade. Omitting the object keeps the prior or API-computed policy.

`update_strategy` and `volume_claim_template` keep their block syntax, including the claim template's `resources` block. If old state contains more than one retention policy, migration reports an error instead of choosing one; reduce it to one policy with the previous provider before upgrading.

### kubernetes_pod_v1

Changed arguments: the pod spec arguments `spec.container.resources`, `spec.init_container.resources`, `spec.image_pull_secrets` and `spec.readiness_gate`, as shown in [Pod spec arguments](#pod-spec-arguments).

### kubernetes_job_v1

Changed arguments: `spec.selector`, `spec.selector.match_expressions`, and the pod spec arguments under `spec.template.spec`.

Before:

```terraform
selector {
  match_labels = {
    app = "report"
  }

  match_expressions {
    key      = "tier"
    operator = "In"
    values   = ["batch"]
  }
}
```

After:

```terraform
selector = {
  match_labels = {
    app = "report"
  }
  match_expressions = [{
    key      = "tier"
    operator = "In"
    values   = ["batch"]
  }]
}
```

Leave the selector omitted when Kubernetes should generate it; keep the existing `manual_selector` setting. State records a generated selector as an object whose fields are `null`, where 3.x recorded an empty list, so a reference to `spec[0].selector` is not `null` even when the selector is not configured.

### kubernetes_cron_job_v1

Changed arguments: `spec.job_template.spec.selector`, `spec.job_template.spec.selector.match_expressions`, and the pod spec arguments under `spec.job_template.spec.template.spec`. The selector changes as shown for [kubernetes_job_v1](#kubernetes_job_v1).

## Additional Pod spec features

The six Framework workload resources also support three optional Kubernetes features:

- `host_users = false` runs a Pod in an isolated Linux user namespace. The cluster and nodes must support [user namespaces](https://kubernetes.io/docs/concepts/workloads/pods/user-namespaces/), including the required kernel and container runtime capabilities. Omitting the argument keeps the existing behavior.
- Container and init container `security_context.proc_mount` accepts `Default` or `Unmasked`. `Unmasked` requires `host_users = false` and cluster support for the [ProcMountType feature](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/#managing-access-to-the-proc-filesystem).
- `volume.image` mounts an OCI image or artifact as a read-only volume. It requires [image volume support](https://kubernetes.io/docs/concepts/storage/volumes/#image) in Kubernetes and the container runtime. In Kubernetes 1.33, this feature is beta and the `ImageVolume` feature gate must be enabled explicitly.

For example, on a cluster that supports these features:

```terraform
resource "kubernetes_pod_v1" "artifact" {
  metadata {
    name = "artifact"
  }
  spec {
    host_users = false
    container {
      name  = "app"
      image = "registry.k8s.io/pause:3.10"
      security_context {
        proc_mount = "Unmasked"
      }
      volume_mount {
        name       = "artifact"
        mount_path = "/artifact"
      }
    }
    volume {
      name = "artifact"
      image = {
        reference   = "registry.k8s.io/conformance:v1.33.0"
        pull_policy = "IfNotPresent"
      }
    }
  }
}
```

`image` is an object argument inside the existing `volume` block. Its `reference` is required for a standalone Pod; workload templates may leave it unset for admission to supply. Omitting `pull_policy` lets Kubernetes select its default. Image volumes are Pod volumes and are not supported by PersistentVolume resources.

For isolated user namespaces, `Unmasked` proc mounts and image volumes, the provider checks API admission with a server-side dry run before writing. Admission webhooks must support dry-run requests. This check does not verify node or container runtime support.

Changes to these immutable settings replace standalone Pods and Jobs. Controller template changes may roll out Pods; CronJob changes apply to new Jobs. Existing state without these attributes is upgraded automatically, and leaving them unconfigured does not enable the features.

## Moving from the deprecated resource types

The deprecated unversioned resource types still use the Plugin SDKv2 and the earlier block syntax. To adopt the versioned type, rename the resource, convert its syntax as described above, update references to the new address, and add a `moved` block. Moving between resource types requires Terraform 1.8 or later.

| Deprecated type | Versioned type |
| --- | --- |
| `kubernetes_deployment` | `kubernetes_deployment_v1` |
| `kubernetes_daemonset` | `kubernetes_daemon_set_v1` |
| `kubernetes_stateful_set` | `kubernetes_stateful_set_v1` |
| `kubernetes_pod` | `kubernetes_pod_v1` |
| `kubernetes_job` | `kubernetes_job_v1` |
| `kubernetes_cron_job` | `kubernetes_cron_job_v1` |

```terraform
moved {
  from = kubernetes_deployment.example
  to   = kubernetes_deployment_v1.example
}
```

Keep all other values unchanged, and do not keep both resource blocks for the same object. A move converts the Terraform address and stored value shapes without recreating the Kubernetes object; review the plan and investigate any replacement before applying.

- `kubernetes_cron_job` manages `batch/v1beta1` CronJobs, which Kubernetes 1.25 and later no longer serve. Make sure the CronJob is available through `batch/v1` before moving it. On those Kubernetes versions, a 3.x plan cannot read such a CronJob and plans to create it, so the empty-plan check in the [upgrade checklist](#upgrade-checklist) cannot pass for it; moving it to `kubernetes_cron_job_v1` resolves that plan.
- Provider 1.x stored `false` for an omitted `automount_service_account_token`, whose default has been `true` since 2.0. Kubernetes cannot change it on an existing Pod or Job, so a Pod or Job whose state was last written by 1.x plans a replacement. Set `automount_service_account_token = false` to keep the object. Likewise, provider versions before 2.21.0 sent and stored `0` for an omitted Job `backoff_limit`, whose default is now `6`; set `backoff_limit = 0` to keep such a Job unchanged.

## Behavior changes

- **One-time state updates.** Earlier versions stored some empty values differently, such as explicitly configured `labels = {}`. The first plan can show an in-place update for them that changes only Terraform state, not the object in Kubernetes. In an update plan, computed metadata such as `metadata[0].resource_version` can show as `(known after apply)`.
- **State after apply.** After a create or update, state records the planned values; only values the plan leaves unknown, such as `uid`, come from Kubernetes. Fields that admission webhooks or other clients add, such as an injected sidecar container, are not recorded by the apply; the next refreshed plan shows them as drift, as in 3.x, and `lifecycle { ignore_changes = [...] }` keeps them out of the plan.
- **Numbers and quantities keep their spelling.** A number or quantity Kubernetes stores, such as `run_as_user = "01000"`, `replicas = "01"`, `cpu = "0.5"` or a file mode, keeps the configured spelling in state instead of planning a change on every plan. This can change string outputs without changing resource allocations.
- **Empty strings on API-defaulted fields.** `""` on a pod spec field that Kubernetes defaults, such as `image_pull_policy`, `service_account_name` or `scheduler_name`, keeps the Kubernetes default and never forces replacement. A new object records `""` until the next refresh, which records the value Kubernetes chose. On other fields, such as the CronJob `timezone` or a volume mount `sub_path`, `""` clears the value.
- **Pod replacement.** The provider updates only the labels, annotations and `active_deadline_seconds` of an existing Pod. Changing any other spec field, such as a volume, volume mount, probe or affinity, replaces the Pod; earlier versions reported success without changing it. Setting or lowering `active_deadline_seconds` is an in-place update, but raising or removing it replaces the Pod, since Kubernetes rejects that change.
- **Pod priority class.** A Pod that leaves out `priority_class_name` records the class Kubernetes assigns, such as a `globalDefault` PriorityClass, instead of planning a replacement on every plan. Removing a configured `priority_class_name` keeps the Pod and its class.
- **Pod tolerations of built-in taints.** A configured toleration of a taint Kubernetes manages, such as `node.kubernetes.io/not-ready` with its own `toleration_seconds`, is recorded in state instead of planning a replacement on every plan. A Pod whose state 3.x last wrote is replaced once more. Tolerations Kubernetes adds for these taints on its own are still not recorded.
- **Planning Pod and Job changes reads the object.** A plan that changes a Pod spec or a Job pod template reads the object from Kubernetes to decide whether it must be replaced, even with `-refresh=false`; 3.x made no API call there. If that read fails or takes longer than 30 seconds, planning fails; plans that change neither still work without the API.
- **Job replacement.** Any change to a Job's pod template or `pod_failure_policy` replaces the Job, since Kubernetes does not allow updating them. Labels the Job controller adds to the pod template, such as `job-name`, are ignored.
- **Admission-added Job template metadata.** Pod template labels and annotations that an admission webhook adds to a Job are not recorded in state. In state from 3.x or an import, the first apply removes such keys from state with an in-place update that leaves the Job unchanged. Until that apply has run, removing a configured pod template label or annotation from such a Job is not applied, as in 3.x; use `terraform apply -replace=<address>` to apply it.
- **Spec updates.** Deployment, DaemonSet, StatefulSet and CronJob updates change only what the plan changes, where 3.x replaced the whole spec (Deployment, DaemonSet, CronJob) or the whole pod template (StatefulSet). Spec fields that the configuration does not manage keep their live values. A change made outside Terraform to a managed field after the plan, for example with `-refresh=false` or a saved plan, is kept by that apply when the plan does not change that field; the next refreshed plan shows it as drift and its apply reverts it.
- **StatefulSet volume claim templates.** Kubernetes does not allow changing the claim templates of an existing StatefulSet. As in 3.x, a change to the `requests`, labels or annotations of a `volume_claim_template` is planned in place but not applied, and the next plan shows it again; the plan now warns about it. To apply such a change, replace the StatefulSet, for example with `terraform apply -replace=<address>`. Adding or removing a claim template, or changing its `access_modes` or `limits`, replaces the StatefulSet.
- **Zero-valued pod security context.** A pod-level `security_context` block that sets only empty values, such as `supplemental_groups = []`, no longer plans a change on every run, and is sent as an empty security context as in 3.x. An explicit `run_as_non_root = false` is sent as `runAsNonRoot: false`, as in 3.x, which Pod Security admission rejects in a `restricted` namespace; adding or removing it in a block that sets nothing else plans no change, while removing the whole block from a Pod or Job that holds `runAsNonRoot: false` replaces it.
- **`name` with `generate_name`.** Setting both is accepted as in 3.x but now gives a warning, since Kubernetes ignores `generate_name` when `name` is set.
