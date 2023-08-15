---
subcategory: "networking/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_ingress_class"
description: |-
  Ingresses can be implemented by different controllers, often with different configuration. Each Ingress should specify a class, a reference to an IngressClass resource that contains additional configuration including the name of the controller that should implement the class.
---

# kubernetes_ingress_class

Ingresses can be implemented by different controllers, often with different configuration. Each Ingress should specify a class, a reference to an IngressClass resource that contains additional configuration including the name of the controller that should implement the class.


## Example Usage

```hcl
resource "kubernetes_ingress_class" "example" {
  metadata {
    name = "example"
  }

  spec {
    controller = "example.com/ingress-controller"
    parameters {
      api_group = "k8s.example.com"
      kind      = "IngressParameters"
      name      = "external-lb"
    }
  }
}
```



## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard ingress's metadata. For more info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
* `spec` - (Required) Spec defines the behavior of a ingress. https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
* `wait_for_load_balancer` - (Optional) Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created. Defaults to `false`.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the ingress that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/

* `name` - (Optional) Name of the ingress class, must be unique. Cannot be updated. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
* `uid` - The unique in time and space value for this service. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids

### `spec`

#### Arguments

* `controller` - (Required) the name of the controller that should handle this class.
* `parameters` - (Optional) Parameters is a link to a custom resource containing additional configuration for the controller. See `parameters` block attributes below.

### `parameters`

#### Arguments

* `name` - (Required) The name of resource being referenced.
* `kind` - (Required) The type of resource being referenced.
* `api_group` - (Optional) The group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group.
* `scope` - (Optional) Refers to a cluster or namespace scoped resource. This may be set to "Cluster" (default) or "Namespace". Field can be enabled with IngressClassNamespacedParams feature gate.
* `namespace` - (Optional) The namespace of the resource being referenced. This field is required when scope is set to "Namespace" and must be unset when scope is set to "Cluster".

## Import

Ingress Classes can be imported using its name, e.g:

```
$ terraform import kubernetes_ingress_class.example example
```
