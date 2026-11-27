---
subcategory: ""
page_title: "Kubernetes: Applying manifests with kubernetes_manifest_v2"
description: |-
  How to apply Kubernetes YAML with the kubernetes_manifest_v2 resource, and how to move resources from kubernetes_manifest.
---

# Applying manifests with `kubernetes_manifest_v2`

The [`kubernetes_manifest_v2`](../resources/manifest_v2.md) resource applies a Kubernetes manifest the way `kubectl apply --server-side` does. It is the successor of `kubernetes_manifest`. This guide shows how to go from YAML files to Terraform resources, and how to move existing resources from `kubernetes_manifest` without recreating any object.

Provider functions and moving resources between resource types require Terraform 1.8 or later.

## Choosing a resource

| Use | When |
|---|---|
| A resource for the kind, such as `kubernetes_deployment_v1` | The provider has one and you write the object in HCL. Arguments are documented and checked while planning. |
| `kubernetes_manifest_v2` | You have YAML, the kind is a custom resource, or the provider has no resource for the kind. |
| `kubernetes_manifest` | You need the manifest checked against the cluster's schema while planning. |

## From YAML to resources

One object, inline:

```terraform
resource "kubernetes_manifest_v2" "service" {
  yaml_body = <<-YAML
    apiVersion: v1
    kind: Service
    metadata:
      name: web
      namespace: default
    spec:
      selector:
        app.kubernetes.io/name: web
      ports:
      - port: 80
    YAML
}
```

One object from a file, as an object value. Use this form to see changed values in plans and when single values are sensitive or not known until apply:

```terraform
resource "kubernetes_manifest_v2" "issuer" {
  manifest = provider::kubernetes::manifest_decode(file("${path.module}/issuer.yaml"))
}
```

Every object of a file with several documents, or of a directory:

```terraform
locals {
  operator = provider::kubernetes::manifest_decode_map(file("${path.module}/operator.yaml"))
  all = provider::kubernetes::manifest_decode_map([
    for name in fileset("${path.module}/manifests", "**/*.yaml") : file("${path.module}/manifests/${name}")
  ]...)
}

resource "kubernetes_manifest_v2" "operator" {
  for_each = local.operator
  manifest = each.value
}

resource "kubernetes_manifest_v2" "all" {
  for_each = local.all
  manifest = each.value
}
```

Decode in a local value as shown. When the call is in `for_each`, Terraform runs it again for every resource instance, which makes plans of large bundles several times slower.

Terraform applies these instances in parallel, in no particular order. An object that is created before its namespace or its CustomResourceDefinition retries for up to a minute, which is usually enough. When a file holds many objects that depend on a few, create those few in their own resource and make the rest depend on it, as shown under [Objects that depend on each other](../resources/manifest_v2.md#objects-that-depend-on-each-other).

Each object becomes one resource instance, with its own line in the plan. The instance key is `Kind[.group]/[namespace/]name`, for example `Deployment.apps/default/web`, so it stays the same when documents are reordered or moved between the files of one call. Removing a document plans the destruction of that object only.

For templates, pass the result of `templatefile` to either function. A value in the template that is not known while planning, such as an attribute of a resource created in the same apply, makes the whole result unknown until apply. With `manifest_decode` this is fine for a new object and for a change to an existing one; only a change of the object's name, namespace or kind must be known while planning. With `manifest_decode_map`, `for_each` needs the keys while planning, so the plan fails: keep such values out of the template, or decode those objects one by one with `manifest_decode`.

`for_each` does not accept a sensitive value. When the YAML text is sensitive, iterate over the keys and look the manifest up:

```terraform
locals {
  objects = provider::kubernetes::manifest_decode_map(var.sensitive_yaml)
}

resource "kubernetes_manifest_v2" "all" {
  for_each = toset(nonsensitive(keys(local.objects)))
  manifest = local.objects[each.key]
}
```

## Migrating from `kubernetes_manifest`

Change the resource type and add a `moved` block. The arguments stay as they are:

```terraform
moved {
  from = kubernetes_manifest.config
  to   = kubernetes_manifest_v2.config
}

resource "kubernetes_manifest_v2" "config" {
  manifest = {
    # unchanged
  }
}
```

Both resources use server-side apply with the field manager `Terraform`, so the first plan after the move shows no changes.

A resource that was imported as `kubernetes_manifest` and not applied since cannot be moved: Terraform reports "Error Decoding Private State". Apply it once as `kubernetes_manifest` before the move, or remove it from the state and import it as `kubernetes_manifest_v2` with the same import ID.

Review these points:

| | What changes |
|---|---|
| References to `object` | They are known in every plan except the one that creates the object. An update no longer changes `object` in the same apply: it shows what the cluster returned at the last refresh, so a value the update changes reaches other resources one plan later. Refer to `manifest` for values the manifest sets. Fields that are not set are absent from `object` instead of null, so wrap optional ones in `try()`. `object` now includes `status`. |
| References to the blocks | `field_manager`, `wait` and `timeouts` are single blocks: write `field_manager.name`, not `field_manager[0].name`. |
| Error messages | The texts differ, for example "Field ownership conflict" and "Object already exists". Scripts that match on them need an update. |
| `computed_fields` | No longer needed for values the API server defaults or rewrites, and no longer defaults to `metadata.annotations` and `metadata.labels`: a label or annotation set in `manifest` that another client changes is reported. Keep entries only for fields another controller changes, such as `spec.replicas` under an autoscaler. |
| Field ownership | Terraform owns only the fields set in `manifest`. `kubernetes_manifest` also took ownership of values it had read from the cluster, such as annotations set by a controller. After the first apply with this resource those are no longer Terraform's. |
| `wait_for` | Write it as `wait { fields = { ... } }`. |
| Validation | A manifest with a field the kind does not have is rejected when it is applied, not while planning. |
| Values | A number or bool where the kind has a string, such as `data = { PORT = 8080 }`, is still applied as a string, now when it is applied rather than while planning. |
| Field manager | A `kubernetes_config_map_v1_data`, `kubernetes_labels` or `kubernetes_annotations` resource on the same object also defaults to the field manager `Terraform`. Give it, or this resource, another `field_manager` name. |
| Namespace | A namespaced object without `metadata.namespace` goes to `default` instead of being an error. |
