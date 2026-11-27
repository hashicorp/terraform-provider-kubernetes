---
subcategory: "manifest"
page_title: "Kubernetes: kubernetes_manifest_v2"
description: |-
  Manages one Kubernetes object of any kind from its manifest with server-side apply, the way kubectl apply --server-side does, without needing the kind's schema. The successor of kubernetes_manifest.
---

# kubernetes_manifest_v2

~> **Beta:** This resource is new in the 4.0 beta releases. Its arguments and behavior may still change before version 4.0.0 is released.

Manages one Kubernetes object of any kind, built-in or custom, from its manifest. The manifest is sent to the cluster with [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/), as `kubectl apply --server-side` does.

`kubernetes_manifest_v2` is the successor of [`kubernetes_manifest`](manifest.md). It takes the same arguments except `wait_for`, and a `moved` block moves existing resources to it. Unlike `kubernetes_manifest`, it does not read the kind's schema from the cluster:

- Planning a new object makes no request to the cluster. A CustomResourceDefinition and its custom resources, or a cluster and the objects in it, can be created in one apply.
- It does not predict what the cluster returns, which is what makes `kubernetes_manifest` fail with "Provider produced inconsistent result after apply".
- Once Terraform has applied a manifest, values the API server defaults or rewrites, such as `cpu: 1` stored as `"1"`, are not a difference and need no `computed_fields`.

## How it differs from `kubernetes_manifest`

| | `kubernetes_manifest_v2` | `kubernetes_manifest` |
|---|---|---|
| Planning a new object | Makes no request to the cluster. The kind does not have to exist yet. | Reads the kind's OpenAPI schema from the cluster. The cluster must be reachable and the CustomResourceDefinition installed. |
| Validation of the manifest | By the API server when the manifest is applied. Unknown fields are rejected. A number or bool where the kind has a string, such as `data = { PORT = 8080 }` in a ConfigMap, is applied as a string. | At plan time, against the schema. |
| `object` | The object as the cluster returns it, including `status`. Read at creation and on every refresh; an update keeps the last value. Unknown in the plan that creates the object. Fields that are not set are absent. | The object typed by the schema, without `status`. Planned from the manifest on every change. Fields that are not set are null. |
| Changes made outside Terraform | Reported for the fields set in `manifest`, decided by the field ownership the cluster records. | Reported by comparing values, except for `computed_fields`. |
| `computed_fields` | Fields that another controller changes: the change is not reported and an apply leaves it in place. Not needed for server defaults. Empty by default. | Fields the API server may alter. Defaults to `metadata.annotations` and `metadata.labels`. |
| What an apply sends | The fields of `manifest`. | The planned `object`, which includes values read from the cluster. Terraform then also owns fields it did not configure. |
| `wait` | `ready`, `rollout`, `fields` and `condition` can be combined. | Only one of `rollout`, `fields` and `condition`. |
| A namespaced object without `metadata.namespace` | Goes to `default`. | Is an error. |
| `wait_for` | Not available. Use `wait { fields = { ... } }`. | Deprecated. |

## Example Usage

### A manifest written in HCL

```terraform
resource "kubernetes_manifest_v2" "config" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name      = "app-config"
      namespace = "default"
    }
    data = {
      LOG_LEVEL = "info"
    }
  }
}
```

### Inline YAML

```terraform
resource "kubernetes_manifest_v2" "service" {
  yaml_body = <<-YAML
    apiVersion: v1
    kind: Service
    metadata:
      name: web
      namespace: default
      labels:
        app.kubernetes.io/name: web
    spec:
      type: ClusterIP
      selector:
        app.kubernetes.io/name: web
      ports:
      - name: grpc
        port: 80
        targetPort: 4245
    YAML
}
```

`yaml_body` takes the text of one object, as YAML or JSON. A key that appears twice in one mapping is an error, and so is a key that overrides one merged with `<<`. A plan shows a change to the text line by line. When another client changes a field, the refresh lists the field in `drift` and adds a comment that names it at the end of `yaml_body`. The plan shows the comment removed, which restores the field, and does not show the value the other client set unless `show_drift_values = true`. Like every change of `yaml_body`, it shows the configured text, unless that is marked with `sensitive()`. `lifecycle { ignore_changes = [yaml_body] }` leaves such changes in place: an update for another reason then leaves the fields other clients changed to them. Set `manifest` instead to see the changed values in the plan, to mark single values as sensitive, or to use values that are not known until apply without making the whole manifest unknown.

### YAML files

```terraform
terraform {
  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
  }
  # Provider functions require Terraform 1.8 and later.
  required_version = ">= 1.8.0"
}

# One object from a YAML file.
resource "kubernetes_manifest_v2" "issuer" {
  manifest = provider::kubernetes::manifest_decode(file("${path.module}/issuer.yaml"))
}

# Every object of a multi-document YAML file, one resource instance each.
# Namespaces and CustomResourceDefinitions come first, in their own resource:
# the other objects need them, and Terraform applies the instances of one
# resource in no particular order. A local value decodes the file once; in
# for_each, Terraform calls the function again for every object.
locals {
  operator = provider::kubernetes::manifest_decode_map(file("${path.module}/operator.yaml"))
  first    = { for key, object in local.operator : key => object if contains(["Namespace", "CustomResourceDefinition"], object.kind) }
}

resource "kubernetes_manifest_v2" "operator_first" {
  for_each = local.first
  manifest = each.value
}

resource "kubernetes_manifest_v2" "operator" {
  for_each   = { for key, object in local.operator : key => object if !contains(keys(local.first), key) }
  manifest   = each.value
  depends_on = [kubernetes_manifest_v2.operator_first]
}

# A templated manifest.
resource "kubernetes_manifest_v2" "web" {
  manifest = provider::kubernetes::manifest_decode(templatefile("${path.module}/deployment.yaml.tftpl", {
    image = var.image
  }))
}
```

[`manifest_decode_map`](../functions/manifest_decode_map.md) keys each object by `Kind[.group]/[namespace/]name`, for example `Deployment.apps/default/web`. Reordering the documents, or moving one to another file that is passed to the same call, does not change its resource address. The example applies Namespaces and CustomResourceDefinitions first: see [Objects that depend on each other](#objects-that-depend-on-each-other).

### Waiting, computed fields and reading what the cluster set

```terraform
resource "kubernetes_manifest_v2" "web" {
  manifest = provider::kubernetes::manifest_decode(file("${path.module}/deployment.yaml"))

  # Succeed only when the new pods are running.
  wait {
    rollout = true
    condition {
      type   = "Available"
      status = "True"
    }
  }

  # A HorizontalPodAutoscaler sets the replica count after creation.
  computed_fields = ["spec.replicas"]

  timeouts {
    create = "15m"
    update = "15m"
  }
}

# Values the cluster sets are in object.
output "ready_replicas" {
  value = kubernetes_manifest_v2.web.object.status.readyReplicas
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `computed_fields` (List of String) Fields of `manifest` that another controller may change after Terraform sets them, for example `spec.replicas` under an autoscaler. Such a change is not reported, and Terraform stops sending a field once another controller owns it. Values the API server defaults or rewrites need no entry here.
- `deletion_propagation` (String) How deleting the object treats the objects it owns, such as the pods of a Job: `Background` (default) deletes them after the object, `Foreground` before it, and the destroy waits for them, `Orphan` leaves them. See [cascading deletion](https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/).
- `field_manager` (Block, Optional) How Kubernetes records the fields this resource owns. (see [below for nested schema](#nestedblock--field_manager))
- `manifest` (Dynamic) The object to apply: an object value with `apiVersion`, `kind` and `metadata.name`. Set either `manifest` or `yaml_body`. Write it in HCL or decode YAML with `provider::kubernetes::manifest_decode`. Terraform manages the fields set here and leaves all other fields of the object to the cluster. `status` and the metadata fields the API server sets are ignored. Changing the API group, `kind`, `metadata.name` or `metadata.namespace` replaces the object.
- `show_drift_values` (Boolean) With `yaml_body`: also show the values another client set, in the comment that names the changed fields. The values appear in plan output, so leave this off for objects that hold secrets.
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))
- `wait` (Block, Optional) Conditions to wait for after the object is applied. All of them must hold before the apply succeeds. (see [below for nested schema](#nestedblock--wait))
- `yaml_body` (String) The object to apply, as YAML or JSON text holding one object. Set either `yaml_body` or `manifest`. With `yaml_body`, a plan lists the fields another client changed in `drift` and does not show their values.

### Read-Only

- `api_version` (String) The API version of the object.
- `drift` (List of String) With `yaml_body`: the fields of the manifest that another client changed or removed since the last apply. A plan that restores them shows this list becoming empty. Always empty with `manifest`, where the plan shows the changed values instead.
- `kind` (String) The kind of the object.
- `name` (String) The name of the object.
- `namespace` (String) The namespace of the object. Empty for a cluster-scoped object, and `default` when the manifest of a namespaced object names none.
- `object` (Dynamic) The object as the cluster returns it, including the fields the cluster sets and `status`. It is read when the object is created and on every refresh. An update keeps the value of the last refresh, so references to it stay known in a plan; what the update changed in the cluster shows after the next refresh. For a value the manifest sets, refer to `manifest`.
- `uid` (String) The UID the cluster assigned to the object.

<a id="nestedblock--field_manager"></a>
### Nested Schema for `field_manager`

Optional:

- `force_conflicts` (Boolean) Take over fields that another field manager owns with a different value. By default such a conflict fails the apply.
- `name` (String) The name of the field manager. Defaults to `Terraform`. A resource that applies single fields to the same object, such as `kubernetes_labels`, must use another name.


<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
- `delete` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Setting a timeout for a Delete operation is only applicable if changes are saved into state before the destroy operation occurs.
- `update` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).


<a id="nestedblock--wait"></a>
### Nested Schema for `wait`

Optional:

- `condition` (Block List) An entry of `status.conditions` to wait for. (see [below for nested schema](#nestedblock--wait--condition))
- `fields` (Map of String) Fields to wait for, as a map of field path to a regular expression the value must match, for example `{ "status.phase" = "Running" }`. Use `*` to wait for any value.
- `ready` (Boolean) Wait until the object is ready by the rules of its kind: a rollout that completed, a Job that completed, a bound claim, or, for any other kind, a status for the current generation in which a `Ready` condition, if there is one, is `True`. A status that does not record the generation, or no status at all, counts after 10 seconds. A failed object fails the apply.
- `rollout` (Boolean) Wait until the rollout of a Deployment, StatefulSet or DaemonSet completes, as `kubectl rollout status` does.

<a id="nestedblock--wait--condition"></a>
### Nested Schema for `wait.condition`

Required:

- `type` (String) The type of the condition.

Optional:

- `status` (String) The status of the condition. Defaults to `True`.





Each operation has 10 minutes by default. Field paths in `wait.fields` and `computed_fields` are written as in `kubernetes_manifest`, for example `status.containerStatuses[0].ready` or `metadata.annotations["example.com/owner"]`. A map key that is not a plain name, such as `app.kubernetes.io/name`, needs the quoted form. `computed_fields` does not tell the elements of a list apart: an entry for one element applies to all of them.

## Waiting

Without a `wait` block, the apply ends when the API server has accepted the object. `ready = true` waits until the object is ready by the rules of its kind: the rollout of a Deployment, StatefulSet or DaemonSet completed, a Job completed, a PersistentVolumeClaim bound, a Pod ready, a Service given a cluster IP. A Service of type `LoadBalancer` is ready before its load balancer has an address; to have the address in `object` after the apply, wait for `fields = { "status.loadBalancer.ingress[0].hostname" = "*" }`, or `ingress[0].ip` where the load balancer has an IP address. Other kinds, such as custom resources, are ready when no `Reconciling` or `Stalled` condition is set, a `Ready` condition, if there is one, is `True`, and the status describes the current generation. A status that does not record the generation, or no status at all, counts after 10 seconds, so that a controller can act on the apply first; a ConfigMap is ready at once. An object that reports a failure, such as a Deployment past its progress deadline, fails the apply. A wait that times out ends with the last status it saw, for example `last status: PVC is not Bound. phase: Pending`. `rollout`, `condition` and `fields` wait for something specific instead.

Terraform watches the object until every condition of `wait` holds, and polls it where the credentials do not allow a watch. Errors that waiting may resolve, such as a restarting API server, do not end the wait. An object that deletes itself, such as a Job with `ttlSecondsAfterFinished`, is checked in the state it had when it was deleted. `rollout` on a StatefulSet or DaemonSet with the `OnDelete` update strategy waits for its pods to be ready, since such a rollout only continues as pods are deleted.

A wait that fails after the object was created marks the resource tainted, so the next apply replaces it. After a wait that fails on an update, the next plan shows that `object` will be read again: applying it applies the manifest again and waits again, also when the configuration went back to an earlier version.

## Field ownership and drift

Kubernetes records which client set each field of an object. Terraform owns the fields set in `manifest`, under the field manager name, and nothing else:

- Fields the cluster defaults, and fields other controllers or admission webhooks add, are not shown as changes of `manifest` and are never removed.
- A field removed from `manifest` is removed from the object.
- When another client changes or removes a field that the manifest sets, the next plan restores it. With `manifest`, the plan shows the live value changing back to the configured one. With `yaml_body`, the field is listed in `drift`. An empty list or map that another client leaves out of the object is not a change.
- To leave a field that the manifest must set at creation to another controller, list it in `computed_fields`. Terraform sets it when it creates the object and does not report later changes by the other controller. Once another field manager owns the field, Terraform no longer sends it, so that manager stays its only owner. A field that no other manager owns, such as a key added to a map in `computed_fields`, is still applied. A field that is not needed at creation is better left out of the manifest. When a field is added to `computed_fields` after the other controller has changed it, the next plan still shows the change back to the configured value once; the apply leaves the field to the controller.
- The labels and annotations of the manifest that match the provider's `ignore_labels` and `ignore_annotations` are treated as `computed_fields`.

Applying over a field that another client changed fails with a field ownership conflict that names the client. Terraform also warns about the conflict on every plan until the field is restored. See [Field ownership conflicts](#field-ownership-conflicts) for the ways to resolve it.

The field manager name, `Terraform` by default, identifies this resource to the API server. Two resources that apply to the same object under the same name replace each other's fields: two configurations that manage one object, or this resource and one of `kubernetes_annotations`, `kubernetes_config_map_v1_data`, `kubernetes_env`, `kubernetes_labels`, `kubernetes_node_taint` and `kubernetes_secret_v1_data`, which also default to `Terraform`. Give each of them its own name. Changing `field_manager.name` moves the fields this resource applied to the new name; the fields of other resources stay with the old one.

`status` and the metadata fields that only the API server sets, such as `uid`, `resourceVersion` and `creationTimestamp`, are ignored when they appear in `manifest`. A `Secret` written with `stringData` is compared with the `data` the cluster stores.

## Field ownership conflicts

A conflict means that another client changed a field the manifest sets and now owns it, for example a `kubectl edit` or a controller. Terraform does not take such a field back unless the configuration says so. The apply fails with "Field ownership conflict" and names the other client's field manager and the fields, for example `conflict with "kubectl-edit" using apps/v1: .spec.replicas`.

To resolve a conflict, choose one of these:

**Terraform is the source of truth** for the fields the manifest sets, and every apply takes them back:

```terraform
resource "kubernetes_manifest_v2" "web" {
  manifest = provider::kubernetes::manifest_decode(file("${path.module}/web.yaml"))
  field_manager {
    force_conflicts = true
  }
}
```

**Take a field back once**, with `terraform apply -var force_conflicts=true`, and keep the default afterwards:

```terraform
variable "force_conflicts" {
  type    = bool
  default = false
}

resource "kubernetes_manifest_v2" "web" {
  manifest = provider::kubernetes::manifest_decode(file("${path.module}/web.yaml"))
  field_manager {
    force_conflicts = var.force_conflicts
  }
}
```

**Hand the field over** to the controller that changes it, such as an autoscaler. Terraform sets it when it creates the object, then no longer reports or sends it:

```terraform
resource "kubernetes_manifest_v2" "web" {
  manifest        = provider::kubernetes::manifest_decode(file("${path.module}/web.yaml"))
  computed_fields = ["spec.replicas"]
}
```

**Carry on as the other field manager**, for an object that `kubectl apply --server-side` or another tool applied before Terraform managed it. Terraform then writes the fields as that tool did, and the tool and Terraform no longer conflict with each other:

```terraform
resource "kubernetes_manifest_v2" "web" {
  manifest = provider::kubernetes::manifest_decode(file("${path.module}/web.yaml"))
  field_manager {
    name = "kubectl"
  }
}
```

Changing `force_conflicts` plans an update, so setting it to `true` takes the fields back in that apply.

## What `object` holds

`object` is the object as the cluster returned it when it was created, or at the last refresh. In the plan that creates the object, `object` is not known yet, so `for_each` and `count` cannot use it in that plan. In every later plan, references to it are known, so it can feed `for_each`, `count` and arguments of other resources.

An update does not change `object`: Terraform cannot know what the cluster will return, and a guess is what makes `kubernetes_manifest` fail with "inconsistent result". The next plan refreshes it. So:

- For a value the manifest sets, refer to `manifest`, for example `kubernetes_manifest_v2.example.manifest.metadata.labels`. It is current in the same apply.
- For a value the cluster sets and never changes, such as a Service's `spec.clusterIP`, or for `name`, `namespace` and `uid`, `object` and the attributes of the same name are always right.
- A value the cluster changes because of an update, such as `status`, reaches other resources with the plan after that update.

A field that is not set is absent from `object`, not null: wrap a reference to an optional field in `try()`. `object` is not sensitive. It holds everything the cluster returns, including the `data` of a Secret, and Terraform shows it, for example in a plan that replaces or destroys the resource and in `terraform show`.

## Objects that depend on each other

Terraform applies independent resources, and the instances of one resource, in parallel and in no particular order: the order of the documents in a file makes no difference. When an object is created before the namespace it belongs to, or before its CustomResourceDefinition is established, the provider retries for up to one minute. That lets a small file apply in one run. An object that starts after its namespace or kind has been missing for a minute fails at once instead of waiting, so that the operation that creates them can start. A kind that an API group without a dot in its name does not have, such as `Deploymnet` in `apps/v1`, is an error at once: only groups such as `example.com` can get kinds from a CustomResourceDefinition.

The retry is not an ordering guarantee. Terraform runs a limited number of operations at a time, 10 by default (`-parallelism`). If all of them are waiting for an object whose creation has not started, the retries expire and the apply fails with an error that names what is missing; a second apply completes it. A file with more objects in a new namespace than that, such as the install manifest of an operator, often fails its first apply this way. Create the namespaces and CustomResourceDefinitions in their own resource and make the rest depend on it:

```terraform
locals {
  objects = provider::kubernetes::manifest_decode_map(file("${path.module}/operator.yaml"))
  first   = { for key, object in local.objects : key => object if contains(["Namespace", "CustomResourceDefinition"], object.kind) }
}

resource "kubernetes_manifest_v2" "first" {
  for_each = local.first
  manifest = each.value
}

resource "kubernetes_manifest_v2" "rest" {
  for_each   = { for key, object in local.objects : key => object if !contains(keys(local.first), key) }
  manifest   = each.value
  depends_on = [kubernetes_manifest_v2.first]
}
```

On destroy, an object whose namespace or CustomResourceDefinition is already gone counts as deleted. Deleting an object also deletes its dependents, such as the pods of a Job, after the object, as `kubectl delete` does. `deletion_propagation = "Foreground"` deletes them first, and the destroy waits until they are gone; `"Orphan"` leaves them. Changing `deletion_propagation` or `show_drift_values` only updates the state.

## API versions that are no longer served

When the cluster stops serving the API version an object was applied in, the object still exists in the other versions of its API group. Terraform then reads and deletes it through one of those, and warns that `apiVersion` in the manifest needs to change. Applying the manifest with the old version fails until it does. A kind that moved to another API group, such as `Ingress` from `extensions` to `networking.k8s.io`, is not found this way.

## Objects that already exist

Creating an object that already exists in the cluster is an error, as with `kubernetes_manifest`. Import it instead, so that taking over an object, and later deleting it with `terraform destroy`, is always explicit. An object that was deleted and created again outside Terraform under the same name gets another UID. The refresh warns about it, and the resource manages the new object from then on. Between a plan and its apply, the UID keeps an update or a delete from acting on an object that replaced the planned one.

Creating needs the `create` and `patch` permissions for the kind, reading needs `get`, updating `get` and `patch`, and deleting `delete` and `get`. `wait` also uses `watch` when the credentials allow it.

## Limitations

- `metadata.generateName` is not supported: server-side apply needs `metadata.name`.
- The API group, `kind`, `metadata.name` and `metadata.namespace` identify the object, and changing one of them replaces it. A manifest without `metadata.namespace` means the object in `default`, so removing the namespace of an object in another namespace replaces it too. Changing only the version in `apiVersion` updates the object in place.
- If a part of the identity in `manifest` is not known while planning, for example a name that comes from a resource replaced in the same apply, Terraform replaces the object, also when the value turns out to be the same. When `yaml_body`, or the whole `manifest`, is not known while planning, Terraform assumes the same object: if it names another one, that apply fails with "Provider produced inconsistent final plan" without changing the object, and the next one replaces it.
- An object that deletes itself, such as a Job with `ttlSecondsAfterFinished`, is planned for creation again after it is gone.
- A change to a field that Kubernetes treats as immutable is rejected when applied. Replace the object with `terraform apply -replace=ADDRESS`. With `create_before_destroy`, the replacement fails with "Object already exists", since the new object has the same name. Replacing a CustomResourceDefinition deletes all its custom resources.
- Terraform cannot mark single fields of `manifest` as sensitive by itself. A value from a sensitive variable or marked with `sensitive()` stays hidden where it appears in `manifest`, a `Secret` written with literal values is shown, and `object` shows what the cluster stores in either case. The state always holds the whole manifest and the object.
- State and plan time grow with the size of the object, because `manifest` and `object` both hold it. For a large object, such as a CustomResourceDefinition of several hundred kilobytes, `yaml_body` is several times faster.

## Import

Import an object with an ID of the form `apiVersion=<version>,kind=<kind>,[namespace=<namespace>,]name=<name>`, the same as for `kubernetes_manifest`:

```shell
terraform import kubernetes_manifest_v2.web "apiVersion=apps/v1,kind=Deployment,namespace=default,name=web"
```

With Terraform 1.12 or later, an `import` block can use the resource identity instead:

```terraform
import {
  to = kubernetes_manifest_v2.web
  identity = {
    api_version = "apps/v1"
    kind        = "Deployment"
    namespace   = "default"
    name        = "web"
  }
}
```

The imported manifest describes the whole object, without the data of a Secret that another client wrote, so the first plan lists the fields that are in the cluster but not in the configuration as removed from `manifest`. Applying it does not remove them from the object: Terraform only takes ownership of the fields the configuration sets. Of the fields that `kubectl apply` without `--server-side` or another resource of this provider wrote, those the configuration sets move to Terraform's field manager, so their values can change without a conflict. A value the configuration spells differently from the cluster, such as `cpu = 1` for a stored `"1"`, also shows as a change in that plan. An object that this resource applied before is imported with only the fields it owns, and plans no change when the configuration is the same.

An object that `kubectl apply --server-side` or another client applied with server-side apply keeps its fields with that field manager. Changing one of them then fails with a field ownership conflict: see [Field ownership conflicts](#field-ownership-conflicts).

## Moving from `kubernetes_manifest`

With Terraform 1.8 or later, a `moved` block moves an object to this resource without touching it. Change the resource type and keep the arguments as they are:

```terraform
# The arguments stay as they are, and the first plan shows no changes.
moved {
  from = kubernetes_manifest.config
  to   = kubernetes_manifest_v2.config
}

resource "kubernetes_manifest_v2" "config" {
  manifest = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name      = "app-config"
      namespace = "default"
    }
    data = {
      LOG_LEVEL = "info"
    }
  }
}
```

Review references to `object` and to the blocks, and replace `wait_for` with a `wait` block. See the table above for what behaves differently, and the [guide](../guides/applying-manifests.md) for the details.

## Moving from `kubectl_manifest`

With Terraform 1.8 or later, a `moved` block also moves a `kubectl_manifest` resource of another provider. `yaml_body` is taken over as it is: change the resource type and remove the arguments this resource does not have, and the plan shows no change to the manifest. By default, a `kubectl_manifest` waits for some kinds, such as the rollout of a Deployment; add a `wait` block to keep that, for example `wait { ready = true }`. The object keeps its fields until the first apply, which moves the fields `yaml_body` sets to this resource's field manager, `Terraform` by default: from the field manager the `kubectl_manifest` applied as with server-side apply (its `field_manager`, or `kubectl`), and from client-side apply. If the `kubectl_manifest` put the object in another namespace than `yaml_body` names, for example with `override_namespace`, set that namespace in `yaml_body`: otherwise the plan replaces the object.

```terraform
moved {
  from = kubectl_manifest.web
  to   = kubernetes_manifest_v2.web
}

resource "kubernetes_manifest_v2" "web" {
  yaml_body = file("${path.module}/web.yaml")
  wait {
    ready = true
  }
}
```
