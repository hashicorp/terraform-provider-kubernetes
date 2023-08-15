---
subcategory: "policy/v1beta1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_pod_disruption_budget"
description: |-
    A Pod Disruption Budget limits the number of pods of a replicated application that are down simultaneously from voluntary disruptions. For example, a quorum-based application would like to ensure that the number of replicas running is never brought below the number needed for a quorum. A web front end might want to ensure that the number of replicas serving load never falls below a certain percentage of the total.
---

# kubernetes_pod_disruption_budget

  A Pod Disruption Budget limits the number of pods of a replicated application that are down simultaneously from voluntary disruptions.
  
  For example, a quorum-based application would like to ensure that the number of replicas running is never brought below the number needed for a quorum. A web front end might want to ensure that the number of replicas serving load never falls below a certain percentage of the total.

## Example Usage

```hcl
resource "kubernetes_pod_disruption_budget" "demo" {
  metadata {
    name = "demo"
  }
  spec {
    max_unavailable = "20%"
    selector {
      match_labels = {
        test = "MyExampleApp"
      }
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard resource's metadata. For more info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
* `spec` - (Required) Spec defines the behavior of a Pod Disruption Budget. https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the resource that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/

* `name` - (Optional) Name of the service, must be unique. Cannot be updated. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
* `uid` - The unique in time and space value for this service. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids

### `spec`

#### Arguments

* `max_unavailable` - (Optional) Specifies the number of pods from the selected set that can be unavailable after the eviction. It can be either an absolute number or a percentage. You can specify only one of max_unavailable and min_available in a single Pod Disruption Budget. max_unavailable can only be used to control the eviction of pods that have an associated controller managing them.
* `min_available` - (Optional) Specifies the number of pods from the selected set that must still be available after the eviction, even in the absence of the evicted pod. min_available can be either an absolute number or a percentage. You can specify only one of min_available and max_unavailable in a single Pod Disruption Budget. min_available can only be used to control the eviction of pods that have an associated controller managing them.
* `selector` - (Optional) A label query over controllers (Deployment, ReplicationController, ReplicaSet, or StatefulSet) that the Pod Disruption Budget should be applied to. For more info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
