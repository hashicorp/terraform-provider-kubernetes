---
page_title: "Migrating apps/v1 workloads to Plugin Framework"
subcategory: ""
description: |-
  Configuration and state migration for Deployment, DaemonSet and StatefulSet resources.
---

# Migrating apps/v1 workloads to Plugin Framework

`kubernetes_deployment_v1`, `kubernetes_daemon_set_v1`, and `kubernetes_stateful_set_v1` use Terraform Plugin Framework. Existing state, resource IDs (`namespace/name`), and indexed references such as `spec[0].template[0].spec[0].container[0].name` retain their shapes. Do not delete state or re-import an existing managed workload.

## Configuration changes

The following arguments use list-assignment syntax instead of repeated blocks:

| Resources | Argument |
| --- | --- |
| All three workloads | Regular and init-container `resources` |
| All three workloads | Template `image_pull_secrets` and `readiness_gate` |
| Deployment and DaemonSet | `strategy` and its nested `rolling_update` |
| StatefulSet | `persistent_volume_claim_retention_policy` |

For example, replace:

```terraform
resources {
  limits = {
    cpu = "500m"
  }
}
```

with:

```terraform
resources = [{
  limits = {
    cpu = "500m"
  }
}]
```

Omit an optional argument, or use `null`, to leave its value computed. Use `resources = [{}]` for the previous `resources {}` form. Use `resources = [{ limits = {}, requests = {} }]` when explicitly configuring empty resource maps. A configured resources list must contain one object; `[]` is not a substitute for omission.

A Deployment rolling-update strategy becomes:

```terraform
strategy = [{
  type = "RollingUpdate"
  rolling_update = [{
    max_surge       = "25%"
    max_unavailable = "25%"
  }]
}]
```

For DaemonSet, use its existing values for `max_surge` and `max_unavailable`; do not copy Deployment defaults. Do not configure `rolling_update` with Deployment `Recreate` or DaemonSet `OnDelete`.

StatefulSet retention configuration becomes:

```terraform
persistent_volume_claim_retention_policy = [{
  when_deleted = "Retain"
  when_scaled  = "Retain"
}]
```

StatefulSet `update_strategy`, `rolling_update` within that strategy, and `volume_claim_template` remain blocks. The `resources` block inside a PVC specification also remains a block.

Replace a `dynamic` block for one of the converted arguments with a list expression, preserving the original iteration order:

```terraform
image_pull_secrets = [
  for name in var.image_pull_secret_names : {
    name = name
  }
]
```

For an empty collection, omit this optional argument or use `null`.

## Upgrade an existing versioned resource

1. Confirm that `terraform plan` with the old provider is empty, and back up state.
2. Update the provider version and the configuration syntax above without changing workload values.
3. Run `terraform plan`. Keeping the same resource address requires no `moved` block.
4. Investigate any replacement or workload-spec change before applying. Apply the reviewed plan, then confirm that another plan is empty.

Computed metadata such as `resource_version` and `generation` can become `(known after apply)` when updating a workload. This alone does not mean replacement.

Previously stored empty metadata maps, container arguments or commands, and node selectors may remain empty rather than null when their arguments are omitted. This preserves compatibility with SDKv2 state after refresh without scheduling a workload update. Removing previously configured non-empty values still removes those managed values; this compatibility behavior does not suppress actual drift.

Refresh can normalize unconfigured metadata `generate_name` values from empty strings to null. Previously stored empty pod-template namespaces remain empty, so an explicit `namespace = ""` does not force replacement. An unused `rolling_update` value for Deployment `Recreate` or DaemonSet `OnDelete` can normalize from an empty list to null. These state-only differences do not change the Kubernetes workload.

With `terraform plan -refresh=false`, SDKv2-written empty `generate_name` values and omitted PVC resource `limits` stored as empty maps can instead appear as in-place state normalizations. These differences alone do not replace the StatefulSet or send a Kubernetes update. Changes to a non-empty `generate_name` or non-empty PVC limits still require replacement.

## Move from a deprecated resource type

The deprecated resource types remain available. To adopt the versioned type, replace the old resource declaration, update references and configuration syntax, and add the matching `moved` block. Cross-type moves require Terraform 1.8 or later.

```terraform
moved {
  from = kubernetes_deployment.example
  to   = kubernetes_deployment_v1.example
}

moved {
  from = kubernetes_daemonset.example
  to   = kubernetes_daemon_set_v1.example
}

moved {
  from = kubernetes_stateful_set.example
  to   = kubernetes_stateful_set_v1.example
}
```

Include only the moves that match resources in your state. Do not keep both old and new resource declarations managing the same Kubernetes object. Review the plan before applying; a state move should not recreate the controller, its Pods, or its persistent volume claims.

## Import

The CLI import ID remains `namespace/name`, for example:

```shell
terraform import kubernetes_deployment_v1.example default/example
terraform import kubernetes_daemon_set_v1.example default/example
terraform import kubernetes_stateful_set_v1.example default/example
```

Identity-based import requires Terraform 1.12 or later and uses `namespace` and `name`. Framework migration itself does not require upgrading Terraform to use ordinary managed resources or string-ID imports.
