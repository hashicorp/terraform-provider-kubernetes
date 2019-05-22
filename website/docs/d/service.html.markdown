---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service"
sidebar_current: "docs-kubernetes-data-source-service"
description: |-
  A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.
---

# kubernetes_service

A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.
This data source allows you to pull data about such service.

## Example Usage

```hcl
data "kubernetes_service" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "aws_route53_record" "example" {
  zone_id = "${data.aws_route53_zone.k8.zone_id}"
  name    = "example"
  type    = "CNAME"
  ttl     = "300"
  records = ["${data.kubernetes_service.example.load_balancer_ingress.0.hostname}"]
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard service's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata)

## Attributes

* `spec` - Spec defines the behavior of a service. [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#spec-and-status)
* `load_balancer_ingress` - A list containing ingress points for the load-balancer (only valid if `type = "LoadBalancer"`)

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Optional) Name of the service, must be unique. Cannot be updated. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

#### Attributes

* `annotations` - (Optional) An unstructured key value map stored with the service that may be used to store arbitrary metadata. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this service.
* `uid` - The unique in time and space value for this service. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `port`

#### Attributes

* `name` - The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service.
* `node_port` - The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#type--nodeport)
* `port` - The port that will be exposed by this service.
* `protocol` - The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.
* `target_port` - Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = "None"`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#defining-a-service)

### `spec`

#### Attributes

* `cluster_ip` - The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies)
* `external_ips` - A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP.  A common example is external load-balancers that are not part of the Kubernetes system.
* `external_name` - The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`.
* `external_traffic_policy` - (Optional) Denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints. `Local` preserves the client source IP and avoids a second hop for LoadBalancer and Nodeport type services, but risks potentially imbalanced traffic spreading. `Cluster` obscures the client source IP and may cause a second hop to another node, but should have good overall load-spreading. More info: https://kubernetes.io/docs/tutorials/services/source-ip/
* `load_balancer_ip` - Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.
* `load_balancer_source_ranges` - If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services-firewalls)
* `port` - The list of ports that are exposed by this service. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies)
* `selector` - Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#overview)
* `session_affinity` - Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies)
* `type` - Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/services#overview)

### `load_balancer_ingress`

#### Attributes

* `hostname` - Hostname which is set for load-balancer ingress points that are DNS based (typically AWS load-balancers)
* `ip` - IP which is set for load-balancer ingress points that are IP based (typically GCE or OpenStack load-balancers)
