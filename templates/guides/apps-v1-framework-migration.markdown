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

The first plan is normally empty. State written by earlier provider versions can contain empty values where the Plugin Framework stores `null`, such as empty metadata maps, empty selector collections, an empty `generate_name`, or an unused `rolling_update`. These can show a one-time in-place update that changes only Terraform state; it does not update or replace the workload. Computed metadata such as `resource_version` and `generation` can show as `(known after apply)` in an update plan.

A pod-level `security_context` block that sets only empty or `false` values, such as `security_context { run_as_non_root = false }`, is sent to Kubernetes as an empty security context; earlier versions sent `runAsNonRoot: false` for it. Kubernetes treats an unset `runAsNonRoot` as false, so the Pods run the same way.

Kubernetes does not allow changing the volume claim templates of an existing StatefulSet. As in earlier versions, a change to the `requests`, labels or annotations of a `volume_claim_template` is planned as an in-place update with a warning and is not sent: the StatefulSet and its PersistentVolumeClaims keep their values, and the next plan shows the change again. To apply it, replace the StatefulSet, for example with `terraform apply -replace`. Other claim template changes, such as `access_modes`, replace the StatefulSet.

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

Identity-based import requires Terraform 1.12 or later and uses `namespace` and `name`.
