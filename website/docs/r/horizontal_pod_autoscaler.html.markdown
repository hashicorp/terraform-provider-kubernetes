---
layout: "kubernetes"
subcategory: "autoscaling/v1"
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

## Example Usage, with `behavior`

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

    behavior {
      scale_down {
        stabilization_window_seconds = 300
        select_policy                = "Min"
        policy {
          period_seconds = 120
          type           = "Pods"
          value          = 1
        }

        policy {
          period_seconds = 310
          type           = "Percent"
          value          = 100
        }
      }
      scale_up {
        stabilization_window_seconds = 600
        select_policy                = "Max"
        policy {
          period_seconds = 180
          type           = "Percent"
          value          = 100
        }
        policy {
          period_seconds = 600
          type           = "Pods"
          value          = 5
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
* `spec` - (Required) Behaviour of the autoscaler. For more info see [Kubernetes reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the horizontal pod autoscaler that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the horizontal pod autoscaler. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the horizontal pod autoscaler, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the horizontal pod autoscaler must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this horizontal pod autoscaler that can be used by clients to determine when horizontal pod autoscaler has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this horizontal pod autoscaler. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Arguments

* `max_replicas` - (Required) Upper limit for the number of pods that can be set by the autoscaler.
* `min_replicas` - (Optional) Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.
* `scale_target_ref` - (Required) Reference to scaled resource. e.g. Replication Controller
* `target_cpu_utilization_percentage` - (Optional) Target average CPU utilization (represented as a percentage of requested CPU) over all the pods. If not specified the default autoscaling policy will be used.
* `metric` - (Optional) A metric on which to scale.
* `behavior` - (Optional) Behavior configures the scaling behavior of the target in both Up and Down directions (scale_up and scale_down fields respectively)

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

See [here](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/) for documentation on resource management for pods and containers.

### `described_object`

#### Arguments

* `api_version` - (Optional) API version of the referent. This argument is optional for the `v1` API version referents and mandatory for the rest.
* `kind` - (Required) Kind of the referent. e.g. `ReplicationController`. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#types-kinds)
* `name` - (Required) Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `scale_target_ref`

#### Arguments

* `api_version` - (Optional) API version of the referent. This argument is optional for the `v1` API version referents and mandatory for the rest.
* `kind` - (Required) Kind of the referent. e.g. `ReplicationController`. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#types-kinds)
* `name` - (Required) Name of the referent. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)

### `behavior`

#### Arguments

* `scale_up` - (Optional) Scaling policy for scaling Up
* `scale_down` - (Optional) Scaling policy for scaling Down


### `scale_up`

#### Arguments

* `policy` - (Required) List of potential scaling polices which can be used during scaling. At least one policy must be specified, otherwise the scaling rule will be discarded as invalid.
* `select_policy` - (Optional) Used to specify which policy should be used. If not set, the default value Max is used.
* `stabilization_window_seconds` - (Optional) Number of seconds for which past recommendations should be considered while scaling up or scaling down. This value must be greater than or equal to zero and less than or equal to 3600 (one hour). If not set, use the default values: - For scale up: 0 (i.e. no stabilization is done). - For scale down: 300 (i.e. the stabilization window is 300 seconds long).

### `policy`

#### Arguments

* `period_seconds` - (Required) Period specifies the window of time for which the policy should hold true. PeriodSeconds must be greater than zero and less than or equal to 1800 (30 min).
* `type` - (Required) Type is used to specify the scaling policy: Percent or Pods
* `value` - (Required) Value contains the amount of change which is permitted by the policy. It must be greater than zero.


## Import

Horizontal Pod Autoscaler can be imported using the namespace and name, e.g.

```
$ terraform import kubernetes_horizontal_pod_autoscaler.example default/terraform-example
```
