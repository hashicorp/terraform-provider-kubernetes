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

All six share one pod spec implementation. Their resource type names, `namespace/name` import IDs, import identities and Kubernetes API versions do not change, and existing state is upgraded without re-importing anything. A few nested blocks become list-of-object arguments, which changes their configuration syntax.

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

4. **Install the provider.** Run `terraform init -upgrade`.
5. **Review the plan.** Run `terraform validate` and `terraform plan`. A syntax-only change must not replace or update a workload in Kubernetes. A one-time in-place update that changes only Terraform state is expected in some cases (see [Behavior changes](#behavior-changes)). Investigate any replacement before applying.
6. **Apply and confirm.** Apply the reviewed plan, then run `terraform plan` again and confirm that it is empty.

Test the upgrade in a non-production workspace first, and repeat these steps for each workspace.

## Configuration syntax changes

The following arguments change from nested blocks to list-of-object arguments, assigned with `=`:

| Argument | Resources |
| --- | --- |
| `resources` in `container` and `init_container` | All six |
| `image_pull_secrets` and `readiness_gate` in the pod spec | All six |
| `strategy` and its nested `rolling_update` | Deployment, DaemonSet |
| `persistent_volume_claim_retention_policy` | StatefulSet |
| `selector` and its nested `match_expressions` | Job, CronJob |

The following rules apply to all of them:

- `resources`, `strategy`, `rolling_update`, `persistent_volume_claim_retention_policy` and `selector` take exactly one object. `image_pull_secrets` and `readiness_gate` take one or more objects.
- To leave one of these arguments unset, omit it or set it to `null`. An empty list `[]` is rejected during planning, except for `match_expressions`.
- Replace an empty block such as `resources {}` with `resources = [{}]`.
- Values keep their types: maps such as `limits`, `requests` and `match_labels` remain maps.
- State keeps the same shape, so references such as `kubernetes_deployment_v1.example.spec[0].template[0].spec[0].container[0].resources[0].limits["cpu"]` are unchanged.

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

  resources = [{
    limits = {
      cpu    = "500m"
      memory = "512Mi"
    }
    requests = {
      cpu    = "250m"
      memory = "128Mi"
    }
  }]
}
```

Make the same change in every `init_container`.

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

### Dynamic blocks

A `dynamic` block cannot generate one of the changed arguments. Replace it with a list expression, keeping the original order:

```terraform
image_pull_secrets = [
  for name in var.image_pull_secret_names : {
    name = name
  }
]
```

When the source can be absent or empty, use a conditional that yields `null`, since an empty list is rejected:

```terraform
resources = var.container_resources == null ? null : [{
  limits   = var.container_resources.limits
  requests = var.container_resources.requests
}]
```

`dynamic` blocks for unchanged blocks, such as `container` or `volume`, remain valid.

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
strategy = [{
  type = "RollingUpdate"
  rolling_update = [{
    max_surge       = "25%"
    max_unavailable = "25%"
  }]
}]
```

Do not configure `rolling_update` with the `Recreate` strategy.

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
strategy = [{
  type = "RollingUpdate"
  rolling_update = [{
    max_surge       = "0"
    max_unavailable = "1"
  }]
}]
```

Keep the DaemonSet's own values; its defaults (`max_surge = "0"`, `max_unavailable = "1"`) differ from a Deployment's. Do not configure `rolling_update` with the `OnDelete` strategy.

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
persistent_volume_claim_retention_policy = [{
  when_deleted = "Delete"
  when_scaled  = "Retain"
}]
```

`update_strategy` and `volume_claim_template` keep their block syntax.

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
selector = [{
  match_labels = {
    app = "report"
  }
  match_expressions = [{
    key      = "tier"
    operator = "In"
    values   = ["batch"]
  }]
}]
```

### kubernetes_cron_job_v1

Changed arguments: `spec.job_template.spec.selector`, `spec.job_template.spec.selector.match_expressions`, and the pod spec arguments under `spec.job_template.spec.template.spec`. The selector changes as shown for [kubernetes_job_v1](#kubernetes_job_v1).

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

Keep all other values unchanged, and do not keep both resource blocks for the same object. A move changes only the Terraform address; review the plan and investigate any replacement before applying.

- `kubernetes_cron_job` manages `batch/v1beta1` CronJobs, which Kubernetes 1.25 and later no longer serve. Make sure the CronJob is available through `batch/v1` before moving it.
- Provider 1.x stored `false` for an omitted `automount_service_account_token`, whose default has been `true` since 2.0. Kubernetes cannot change it on an existing Pod or Job, so a Pod or Job whose state was last written by 1.x plans a replacement. Set `automount_service_account_token = false` to keep the object. Likewise, provider versions before 2.21.0 sent and stored `0` for an omitted Job `backoff_limit`, whose default is now `6`; set `backoff_limit = 0` to keep such a Job unchanged.

## Behavior changes

- **Empty lists are rejected** for the list-of-object arguments above during planning. Omit the argument or use `null`.
- **One-time state updates.** Earlier versions stored some empty values differently, such as explicitly configured `labels = {}`. The first plan can show an in-place update for them that changes only Terraform state, not the object in Kubernetes. In an update plan, computed metadata such as `metadata[0].resource_version` can show as `(known after apply)`.
- **Numbers keep their spelling.** A number Kubernetes stores, such as `run_as_user = "01000"`, `replicas = "01"` or a file mode, keeps the configured spelling in state instead of planning a change on every plan.
- **Empty strings on API-defaulted fields.** `""` on a pod spec field that Kubernetes defaults, such as `image_pull_policy`, `service_account_name` or `scheduler_name`, keeps the Kubernetes default and never forces replacement. A new object records `""` until the next refresh, which records the value Kubernetes chose. On other fields, such as the CronJob `timezone` or a volume mount `sub_path`, `""` clears the value.
- **Pod resource quantities** keep the configured spelling in state for new Pods, such as `"0.5"` rather than `"500m"`. This can change string outputs, not allocations.
- **Pod replacement.** The provider updates only the labels, annotations and `active_deadline_seconds` of an existing Pod. Changing any other spec field, such as a volume, volume mount, probe or affinity, replaces the Pod; earlier versions reported success without changing it. Setting or lowering `active_deadline_seconds` is an in-place update, but raising or removing it replaces the Pod, since Kubernetes rejects that change.
- **Job replacement.** Any change to a Job's pod template or `pod_failure_policy` replaces the Job, since Kubernetes does not allow updating them. Labels the Job controller adds to the pod template, such as `job-name`, are ignored.
- **Admission-added Job template metadata.** Pod template labels and annotations that an admission webhook adds to a Job are not recorded in state. In state from 3.x or an import, the first apply removes such keys from state with an in-place update that leaves the Job unchanged. Until that apply has run, removing a configured pod template label or annotation from such a Job is not applied, as in 3.x; use `terraform apply -replace=<address>` to apply it.
- **CronJob updates** change only what the configuration changes. Labels, annotations and spec fields that are not recorded in state, such as fields set by admission controllers, keep their live values.
- **StatefulSet volume claim templates.** Kubernetes does not allow changing the claim templates of an existing StatefulSet. As in 3.x, a change to the `requests`, labels or annotations of a `volume_claim_template` is planned in place but not applied, and the next plan shows it again; the plan now warns about it. To apply such a change, replace the StatefulSet, for example with `terraform apply -replace=<address>`. Adding or removing a claim template, or changing its `access_modes` or `limits`, replaces the StatefulSet.
- **Zero-valued pod security context.** A pod-level `security_context` block that sets only empty or `false` values, such as `run_as_non_root = false`, is sent to Kubernetes as an empty security context; earlier versions sent `runAsNonRoot: false`. Kubernetes treats both the same way. Removing such a block from a Job created by an earlier version replaces the Job.
- **`name` with `generate_name`.** Setting both is accepted as in 3.x but now gives a warning, since Kubernetes ignores `generate_name` when `name` is set.

## Performance

The six resources use more CPU per operation than their Plugin SDKv2 versions did; most of the overhead is in the Plugin Framework itself. On a real cluster, API requests usually dominate the run time, so the difference is most noticeable in large workspaces.
