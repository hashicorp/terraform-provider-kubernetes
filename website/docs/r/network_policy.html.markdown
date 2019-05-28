---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_network_policy"
sidebar_current: "docs-kubernetes-resource-network-policy"
description: |-
  Kubernetes supports network policies to specificy of how groups of pods are allowed to communicate with each other and other network endpoints.
  NetworkPolicy resources use labels to select pods and define rules which specify what traffic is allowed to the selected pods.
---

# kubernetes_network_policy

Kubernetes supports network policies to specificy of how groups of pods are allowed to communicate with each other and other network endpoints.
NetworkPolicy resources use labels to select pods and define rules which specify what traffic is allowed to the selected pods.
Read more about network policies at https://kubernetes.io/docs/concepts/services-networking/network-policies/

## Example Usage

```hcl
resource "kubernetes_network_policy" "example" {
  metadata {
    name      = "terraform-example-network-policy"
    namespace = "default"
  }

  spec {
    pod_selector {
      match_expressions {
        key      = "name"
        operator = "In"
        values   = ["webfront", "api"]
      }
    }

    ingress = [
      {
        ports = [
          {
            port     = "http"
            protocol = "TCP"
          },
          {
            port     = "8125"
            protocol = "UDP"
          },
        ]

        from = [
          {
            namespace_selector {
              match_labels = {
                name = "default"
              }
            }
          },
          {
            ip_block {
              cidr = "10.0.0.0/8"
              except = [
                "10.0.0.0/24",
                "10.0.1.0/24",
              ]
            }
          },
        ]
      },
    ]

    egress = [{}] # single empty rule to allow all egress traffic

    policy_types = ["Ingress", "Egress"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard network policy's [metadata](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata).

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the network policy that may be used to store arbitrary metadata. 
**By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
More info: http://kubernetes.io/docs/user-guide/annotations
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more about [name idempotency](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#idempotency).
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) network policies. May match selectors of replication controllers and services. 
**By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
More info: http://kubernetes.io/docs/user-guide/labels
* `name` - (Optional) Name of the network policy, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names

#### Attributes

* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this network policy that can be used by clients to determine when network policies have changed. Read more about [concurrency control and consistency](https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency).
* `self_link` - A URL representing this network policy.
* `uid` - The unique in time and space value for this network policy. More info: http://kubernetes.io/docs/user-guide/identifiers#uids


### `spec`

#### Arguments

* `egress` - (Optional) List of egress rules to be applied to the selected pods. Outgoing traffic is allowed if there are no NetworkPolicies selecting the pod (and cluster policy otherwise allows the traffic), OR if the traffic matches at least one egress rule across all of the NetworkPolicy objects whose podSelector matches the pod. If this field is empty then this NetworkPolicy limits all outgoing traffic (and serves solely to ensure that the pods it selects are isolated by default). This field is beta-level in 1.8
* `ingress` - (Optional) List of ingress rules to be applied to the selected pods. Traffic is allowed to a pod if there are no NetworkPolicies selecting the pod (and cluster policy otherwise allows the traffic), OR if the traffic source is the pod's local node, OR if the traffic matches at least one ingress rule across all of the NetworkPolicy objects whose podSelector matches the pod. If this field is empty then this NetworkPolicy does not allow any traffic (and serves solely to ensure that the pods it selects are isolated by default).
* `pod_selector` - (Required) Selects the pods to which this NetworkPolicy object applies. The array of ingress rules is applied to any pods selected by this field. Multiple network policies can select the same set of pods. In this case, the ingress rules for each are combined additively. This field is NOT optional and follows standard label selector semantics. An empty podSelector matches all pods in this namespace.
* `policy_types` (Required) List of rule types that the NetworkPolicy relates to. Valid options are `Ingress`, `Egress`, or `Ingress,Egress`. This field is beta-level in 1.8
**Note**: the native Kubernetes API allows not to specify the `policy_types` property with the following description:
  > If this field is not specified, it will default based on the existence of Ingress or Egress rules; policies that contain an Egress section are assumed to affect Egress, and all policies (whether or not they contain an Ingress section) are assumed to affect Ingress. If you want to write an egress-only policy, you must explicitly specify policyTypes [ "Egress" ]. Likewise, if you want to write a policy that specifies that no egress is allowed, you must specify a policyTypes value that include "Egress" (since such a policy would not include an Egress section and would otherwise default to just [ "Ingress" ]).

  Leaving the `policy_types` property optional here would have prevented an `egress` rule added to a Network Policy initially created without any `egress` rule nor `policy_types` from working as expected. Indeed, the PolicyTypes would have stuck to Ingress server side as the default value is only computed server side on resource creation, not on updates.

### `ingress`

#### Arguments

* `from` - (Optional) List of sources which should be able to access the pods selected for this rule. Items in this list are combined using a logical OR operation. If this field is empty or missing, this rule matches all sources (traffic not restricted by source). If this field is present and contains at least on item, this rule allows traffic only if the traffic matches at least one item in the from list.
* `ports` - (Optional) List of ports which should be made accessible on the pods selected for this rule. Each item in this list is combined using a logical OR. If this field is empty or missing, this rule matches all ports (traffic not restricted by port). If this field is present and contains at least one item, then this rule allows traffic only if the traffic matches at least one port in the list.

### `egress`

#### Arguments
* `to` - (Optional) List of destinations for outgoing traffic of pods selected for this rule. Items in this list are combined using a logical OR operation. If this field is empty or missing, this rule matches all destinations (traffic not restricted by destination). If this field is present and contains at least one item, this rule allows traffic only if the traffic matches at least one item in the to list.
* `ports` - (Optional) List of destination ports for outgoing traffic. Each item in this list is combined using a logical OR. If this field is empty or missing, this rule matches all ports (traffic not restricted by port). If this field is present and contains at least one item, then this rule allows traffic only if the traffic matches at least one port in the list.

### `from`

#### Arguments

* `namespace_selector` - (Optional) Selects Namespaces using cluster scoped-labels. This matches all pods in all namespaces selected by this label selector. This field follows standard label selector semantics. If present but empty, this selector selects all namespaces.
* `pod_selector` - (Optional) This is a label selector which selects Pods in this namespace. This field follows standard label selector semantics. If present but empty, this selector selects all pods in this namespace.


### `ports`

#### Arguments

* `port` - (Optional) The port on the given protocol. This can either be a numerical or named port on a pod. If this field is not provided, this matches all port names and numbers.
* `protocol` - (Optional) The protocol (TCP or UDP) which traffic must match. If not specified, this field defaults to TCP.


### `to`

#### Arguments

* `ip_block` - (Optional) IPBlock defines policy on a particular IPBlock
* `namespace_selector` - (Optional) Selects Namespaces using cluster scoped-labels. This matches all pods in all namespaces selected by this label selector. This field follows standard label selector semantics. If present but empty, this selector selects all namespaces.
* `pod_selector` - (Optional) This is a label selector which selects Pods in this namespace. This field follows standard label selector semantics. If present but empty, this selector selects all pods in this namespace.

### `ip_block`

#### Arguments

* `cidr` - (Optional)	CIDR is a string representing the IP Block Valid examples are "192.168.1.1/24"
* `except` - (Optional) Except is a slice of CIDRs that should not be included within an IP Block. Valid examples are "192.168.1.1/24". Except values will be rejected if they are outside the CIDR range.

### `namespace_selector`

#### Arguments

* `match_expressions` - (Optional) A list of label selector requirements. The requirements are ANDed.
* `match_labels` - (Optional) A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.


### `pod_selector`

#### Arguments

* `match_expressions` - (Optional) A list of label selector requirements. The requirements are ANDed.
* `match_labels` - (Optional) A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.


### `match_expressions`

#### Arguments

* `key` - (Optional) The label key that the selector applies to.
* `operator` - (Optional) A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.
* `values` - (Optional) An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.


## Import

Network policies can be imported using their identifier consisting of  `<namespace-name>/<network-policy-name>`, e.g.:

```
$ terraform import kubernetes_network_policy.example default/terraform-example-network-policy
```
