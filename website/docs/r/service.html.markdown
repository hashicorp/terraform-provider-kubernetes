---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service"
sidebar_current: "docs-kubernetes-resource-service-x"
description: |-
  A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.
---

# kubernetes_service

A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.


## Example Usage

```hcl
resource "kubernetes_service" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    selector = {
      app = "${kubernetes_pod.example.metadata.0.labels.app}"
    }
    session_affinity = "ClientIP"
    port {
      port        = 8080
      target_port = 80
    }

    type = "LoadBalancer"
  }
}

resource "kubernetes_pod" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      app = "MyApp"
    }
  }

  spec {
    container {
      image = "nginx:1.7.9"
      name  = "example"
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard service's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata)
* `spec` - (Required) Spec defines the behavior of a service. [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the service that may be used to store arbitrary metadata.
**By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services. 
**By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem).**
For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
* `name` - (Optional) Name of the service, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this service.
* `uid` - The unique in time and space value for this service. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `spec`

#### Arguments

* `cluster_ip` - (Optional) The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies)
* `external_ips` - (Optional) A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP.  A common example is external load-balancers that are not part of the Kubernetes system.
* `external_name` - (Optional) The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`.
* `external_traffic_policy` - (Optional) Denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints. `Local` preserves the client source IP and avoids a second hop for LoadBalancer and Nodeport type services, but risks potentially imbalanced traffic spreading. `Cluster` obscures the client source IP and may cause a second hop to another node, but should have good overall load-spreading. More info: https://kubernetes.io/docs/tutorials/services/source-ip/
* `load_balancer_ip` - (Optional) Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.
* `load_balancer_source_ranges` - (Optional) If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services-firewalls)
* `port` - (Required) The list of ports that are exposed by this service. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies)
* `publish_not_ready_addresses` - (Optional) When set to true, indicates that DNS implementations must publish the `notReadyAddresses` of subsets for the Endpoints associated with the Service. The default value is `false`. The primary use case for setting this field is to use a StatefulSet's Headless Service to propagate `SRV` records for its Pods without respect to their readiness for purpose of peer discovery.
* `selector` - (Optional) Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#overview)
* `session_affinity` - (Optional) Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies)
* `type` - (Optional) Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#overview)

### `port`

#### Arguments

* `name` - (Optional) The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service.
* `node_port` - (Optional) The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#type--nodeport)
* `port` - (Required) The port that will be exposed by this service.
* `protocol` - (Optional) The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.
* `target_port` - (Optional) Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = "None"`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#defining-a-service)

## Attributes

* `load_balancer_ingress` - A list containing ingress points for the load-balancer (only valid if `type = "LoadBalancer"`)

### `load_balancer_ingress`

#### Attributes

* `ip` - IP which is set for load-balancer ingress points that are IP based (typically GCE or OpenStack load-balancers)
* `hostname` - Hostname which is set for load-balancer ingress points that are DNS based (typically AWS load-balancers)

## Import

Service can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_service.example default/terraform-name
```
