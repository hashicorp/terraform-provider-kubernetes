---
subcategory: ""
page_title: "Kubernetes: Pod Plugin Framework Migration"
description: |-
  Update kubernetes_pod_v1 configuration when upgrading to its Plugin Framework implementation.
---

# Migrating kubernetes_pod_v1 to Plugin Framework

`kubernetes_pod_v1` uses Terraform Plugin Framework. The resource keeps its name,
existing state, and `namespace/name` import ID, so an existing `kubernetes_pod_v1`
does not need a rename, `moved` block, or re-import. Only the configuration syntax
of a few nested arguments changes.

## Change container resources from blocks to list assignments

The `resources` argument in both regular and init containers uses a list of
objects, so Kubernetes can populate resource defaults when the argument or either
of its `limits` and `requests` maps is omitted.

Before:

```hcl
container {
  name  = "app"
  image = "nginx:1.21.6"

  resources {
    limits   = { cpu = "100m", memory = "64Mi" }
    requests = { cpu = "100m", memory = "64Mi" }
  }
}
```

After:

```hcl
container {
  name  = "app"
  image = "nginx:1.21.6"

  resources = [{
    limits   = { cpu = "100m", memory = "64Mi" }
    requests = { cpu = "100m", memory = "64Mi" }
  }]
}
```

Make the same edit inside every `init_container`. The outer `container`,
`init_container`, `spec`, and `metadata` blocks retain block syntax. Do not apply
this edit to other resource types merely because they also contain Pod templates.

An omitted `resources` argument can remain omitted. Replace an explicit empty
`resources {}` block with `resources = [{}]`. Preserve any explicitly supplied
maps, including empty maps, rather than substituting API values.
Use `null`, not `[]`, to omit the argument: a configured empty list cannot contain
the resource defaults returned by Kubernetes.

### Dynamic resources blocks

Replace a dynamic block with an expression that produces `null` for omission or
a one-element list. For example, given a nullable `var.container_resources` object:

Before:

```hcl
dynamic "resources" {
  for_each = var.container_resources == null ? [] : [var.container_resources]
  content {
    limits   = resources.value.limits
    requests = resources.value.requests
  }
}
```

After:

```hcl
resources = var.container_resources == null ? null : [{
  limits   = var.container_resources.limits
  requests = var.container_resources.requests
}]
```

The stored value remains a list of objects. Existing expressions such as
`kubernetes_pod_v1.app.spec[0].container[0].resources[0].requests` retain their
indexing.

## Change image-pull secret and readiness-gate references

`spec.image_pull_secrets` and `spec.readiness_gate` also use list assignment:

```hcl
spec {
  image_pull_secrets = [{ name = "registry-credentials" }]
  readiness_gate     = [{ condition_type = "example.com/ready" }]
}
```

These replace `image_pull_secrets { name = "registry-credentials" }` and
`readiness_gate { condition_type = "example.com/ready" }` blocks, respectively.
Keep all list entries and their order. An omitted field can remain omitted;
convert a conditional dynamic block to `null` when no references are configured.
An explicitly configured empty list is rejected; use omission or `null` instead.
The stored collection types and existing indexed expressions are unchanged.

## Resource quantity representation

Newly created Pods keep configured quantity strings in state. For example, a CPU
limit configured as `"0.5"` stays `"0.5"` even though Kubernetes returns `"500m"`.
Existing state values and equivalent quantity edits do not cause a diff, and an
import uses the API's spelling. This can change string outputs, not allocations.

## Empty strings on API-defaulted fields

An empty string on a field that Kubernetes defaults, such as `image_pull_policy`,
`service_account_name`, `scheduler_name` or `node_name`, leaves the value to
Kubernetes. A new Pod records `""` for it until the next refresh, which records
the value Kubernetes chose. The empty string never forces replacement. Before
that first refresh, for example with `terraform plan -refresh=false`, changing
the configuration from `""` to the value Kubernetes chose replaces the Pod. Run a
normal plan or `terraform apply -refresh-only` first.

## Changes that replace the Pod

Kubernetes can update only a few fields of a running Pod. The provider changes
metadata labels and annotations and `spec.active_deadline_seconds` in place.
Any other spec change replaces the Pod, including fields that earlier versions
treated as updatable, such as volumes, volume mounts, probes and affinity.
Earlier versions reported success for those changes without changing the Pod
and showed the same change again on the next plan.

## Review the upgrade

1. Back up state securely and retain the previous provider version and configuration.
2. Update the provider version and make the syntax edits above without changing
   names, images, resource quantities, or other Pod settings.
3. Run `terraform init -upgrade` and `terraform plan`.
4. Check all resources, not just the Pod. The syntax edit must not cause Pod
   deletion, replacement, or a change to its Kubernetes settings. Stop and
   investigate any unexpected operation before applying.
5. Apply the reviewed plan and run `terraform plan` again to check convergence.

Earlier provider versions recorded some empty values differently: an empty
`metadata.generate_name`, configured empty maps such as `labels = {}` or
`node_selector = {}`, and blocks that hold only empty values. A `projected`
volume that groups several projections in one `sources` block, which earlier
versions reported as a change on every plan, is also brought in line once.
These show a one-time in-place update that changes only Terraform state, not
the Pod; the following plan should be empty.

## Move from the deprecated resource

Terraform 1.8 or later supports the cross-type move from `kubernetes_pod` to
`kubernetes_pod_v1`. Both refer to the same Kubernetes `core/v1` Pod.

Change the resource type in configuration, make the syntax edits above,
and add:

```hcl
moved {
  from = kubernetes_pod.app
  to   = kubernetes_pod_v1.app
}
```

Preserve the namespace, name, and all other settings. A `moved` block changes the
Terraform address, not the remote Pod, and does not override replacement rules
for a real configuration change.

## Import

The existing command-line import format is unchanged:

```shell
terraform import kubernetes_pod_v1.app default/app
```

With Terraform 1.12 or later, identity import is also supported:

```hcl
import {
  to = kubernetes_pod_v1.app
  identity = {
    api_version = "v1"
    kind        = "Pod"
    namespace   = "default"
    name        = "app"
  }
}
```

If identity import omits `namespace`, it defaults to `default`; an explicitly empty
namespace is invalid. `api_version` and `kind` must be `v1` and `Pod`.

Import does not recreate the Pod. Provide configuration matching the imported
object and review the next plan before applying.
