---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_replication_controller"
description: |-
  A Replication Controller ensures that a specified number of pod “replicas” are running at any one time. In other words, a Replication Controller makes sure that a pod or homogeneous set of pods are always up and available. If there are too many pods, it will kill some. If there are too few, the Replication Controller will start more.
---

# kubernetes_replication_controller

A Replication Controller ensures that a specified number of pod “replicas” are running at any one time. In other words, a Replication Controller makes sure that a pod or homogeneous set of pods are always up and available. If there are too many pods, it will kill some. If there are too few, the Replication Controller will start more.

~> **WARNING:** In many cases it is recommended to create a Deployment instead of a Replication Controller.

## Example Usage

```hcl
resource "kubernetes_replication_controller" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      test = "MyExampleApp"
    }
  }

  spec {
    selector = {
      test = "MyExampleApp"
    }
    template {
      metadata {
        labels = {
          test = "MyExampleApp"
        }
        annotations = {
          "key1" = "value1"
        }
      }

      spec {
        container {
          image = "nginx:1.21.6"
          name  = "example"

          liveness_probe {
            http_get {
              path = "/"
              port = 80

              http_header {
                name  = "X-Custom-Header"
                value = "Awesome"
              }
            }

            initial_delay_seconds = 3
            period_seconds        = 3
          }

          resources {
            limits = {
              cpu    = "0.5"
              memory = "512Mi"
            }
            requests = {
              cpu    = "250m"
              memory = "50Mi"
            }
          }
        }
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard replication controller's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `spec` - (Required) Spec defines the specification of the desired behavior of the replication controller. For more info see [Kubernetes reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the replication controller that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the replication controller.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the replication controller, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the replication controller must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this replication controller that can be used by clients to determine when replication controller has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this replication controller. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Arguments

* `min_ready_seconds` - (Optional) Minimum number of seconds for which a newly created pod should be ready without any of its container crashing, for it to be considered available. Defaults to 0 (pod will be considered available as soon as it is ready)
* `replicas` - (Optional) The number of desired replicas. Defaults to 1. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#what-is-a-replicationcontroller)
* `selector` - (Required) A label query over pods that should match the Replicas count. Label keys and values that must match in order to be controlled by this replication controller. **Should match labels (`metadata.0.labels`)**. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#label-selector-and-annotation-conventions)
* `template` - (Required) Template is the object that describes the pod that will be created if insufficient replicas are detected. This takes precedence over a TemplateRef. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/workloads/controllers/replicationcontroller#pod-template)

## Nested Blocks

### `spec.template`

#### Arguments

* `metadata` - (Optional) Standard object's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata). While required by the kubernetes API, this field is marked as optional to allow the usage of the deprecated pod spec fields that were mistakenly placed directly under the `template` block.

* `spec` - (Optional) Specification of the desired behavior of the pod. For more info see [Kubernetes reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)

~> **NOTE:** all the fields from the `spec.template.spec` block are also accepted at the `spec.template` level but that usage is deprecated. All existing configurations should be updated to only use the new fields under `spec.template.spec`. Mixing the usage of deprecated fields with new fields is not supported.

## Nested Blocks

### `spec.template.metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the replication controller that may be used to store arbitrary metadata. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the pods managed by this replication controller . **Should match `selector`**. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#label-selector-and-annotation-conventions)
* `name` - (Optional) Name of the replication controller, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the replication controller must be unique.

## Nested Blocks

### `spec.template.spec`

#### Arguments

These arguments are the same as the for the `spec` block of a Pod.

Please see the [Pod resource](pod.html#spec) for reference.

## Timeouts

The following [Timeout](/docs/configuration/resources.html#operation-timeouts) configuration options are available:

- `create` - (Default `10 minutes`) Used for creating new controller
- `update` - (Default `10 minutes`) Used for updating a controller
- `delete` - (Default `10 minutes`) Used for destroying a controller

## Import

Replication Controller can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_replication_controller.example default/terraform-example
```

~> **NOTE:** Imported `kubernetes_replication_controller` resource will only have their fields from the `spec.template.spec` block in the state. Deprecated fields at the `spec.template` level are not updated during import. Configurations using the deprecated fields should be updated to only use the new fields under `spec.template.spec`.
