---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_horizontal_pod_autoscaler"
description: |-
  Horizontal Pod Autoscaler automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization.
---

# kubernetes_horizontal_pod_autoscaler

Horizontal Pod Autoscaler automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization.


## Example Usage

```hcl
resource "kubernetes_horizontal_pod_autoscaler" "example" {
  metadata {
    name = "terraform-example"
  }

  spec {
    max_replicas = 10
    min_replicas = 8

    scale_target_ref {
      kind = "Deployment"
      name = "MyApp"
    }
  }
}
```

## Example Usage, with `metric`

```hcl
resource "kubernetes_horizontal_pod_autoscaler" "example" {
  metadata {
    name = "test"
  }

  spec {
    min_replicas = 50
    max_replicas = 100

    scale_target_ref {
      kind = "Deployment"
      name = "MyApp"
    }

    metric {
      type = "External"
      external {
        metric {
          name = "latency"
          selector {
            match_labels = {
              lb_name = "test"
            }
          }
        }
        target {
          type  = "Value"
          value = "100"
        }
      }
    }
  }
}
```

## Support for multiple and custom metrics 

The provider currently supports two version of the HorizontalPodAutoscaler API resource.

If you wish to use `autoscaling/v1` use the `target_cpu_utilization_percentage` field.

If you wish to use `autoscaling/v2beta2` then set one or more `metric` fields.

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard horizontal pod autoscaler's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `spec` - (Required) Behaviour of the autoscaler. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the horizontal pod autoscaler that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the horizontal pod autoscaler. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)

* `name` - (Optional) Name of the horizontal pod autoscaler, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the horizontal pod autoscaler must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this horizontal pod autoscaler that can be used by clients to determine when horizontal pod autoscaler has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this horizontal pod autoscaler. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `spec`

#### Arguments

* `max_replicas` - (Required) Upper limit for the number of pods that can be set by the autoscaler.
* `min_replicas` - (Optional) Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.
* `scale_target_ref` - (Required) Reference to scaled resource. e.g. Replication Controller
* `target_cpu_utilization_percentage` - (Optional) Target average CPU utilization (represented as a percentage of requested CPU) over all the pods. If not specified the default autoscaling policy will be used.
* `metric` - (Optional) A metric on which to scale.

### `metric`

#### Arguments

* `type` - (Required) The type of metric. It can be one of "Object", "Pods", "Resource", or "External".
* `object` - (Optional) A metric describing a single kubernetes object (for example, hits-per-second on an Ingress object).
* `pods` - (Optional) A metric describing each pod in the current scale target (for example, transactions-processed-per-second). The values will be averaged together before being compared to the target value.
* `resource` - (Optional) A resource metric (such as those specified in requests and limits) known to Kubernetes describing each pod in the current scale target (e.g. CPU or memory). Such metrics are built in to Kubernetes, and have special scaling options on top of those available to normal per-pod metrics using the "pods" source.
* `external` - (Optional) A global metric that is not associated with any Kubernetes object. It allows autoscaling based on information coming from components running outside of cluster (for example length of queue in cloud messaging service, or QPS from loadbalancer running outside of cluster).

### Metric Type: `external`

#### Arguments

* `metric` - (Required) Identifies the target by name and selector.
* `target` - (Required) The target for the given metric.

### Metric Type: `object`

#### Arguments

* `described_object` - (Required) Reference to the object.
* `metric` - (Required) Identifies the target by name and selector.
* `target` - (Required) The target for the given metric.

### Metric Type: `pods`

#### Arguments

* `metric` - (Required) Identifies the target by name and selector.
* `target` - (Required) The target for the given metric.

### Metric Type: `resource`

#### Arguments

* `name` - (Required) Name of the resource in question.
* `target` - (Required) The target for the given metric.

### `metric` 

#### Arguments

* `name` - (Required) The name of the given metric
* `selector` - (Optional) The label selector for the given metric 

### `target`

#### Arguments

* `type` - (Required) Represents whether the metric type is Utilization, Value, or AverageValue.
* `average_value` - (Optional) The target value of the average of the metric across all relevant pods (as a quantity).
* `average_utilization` - (Optional) The target value of the average of the resource metric across all relevant pods, represented as a percentage of the requested value of the resource for the pods. Currently only valid for Resource metric source type.
* `value` - (Optional) value is the target value of the metric (as a quantity).

#### Quantities

See [here](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#quantity-resource-core) for documentation on quantities.

### `described_object`

#### Arguments

* `api_version` - (Optional) API version of the referent
* `kind` - (Required) Kind of the referent. e.g. `ReplicationController`. For more info see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#types-kinds
* `name` - (Required) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

### `scale_target_ref`

#### Arguments

* `api_version` - (Optional) API version of the referent
* `kind` - (Required) Kind of the referent. e.g. `ReplicationController`. For more info see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#types-kinds
* `name` - (Required) Name of the referent. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)

## Import

Horizontal Pod Autoscaler can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_horizontal_pod_autoscaler.example default/terraform-example
```
