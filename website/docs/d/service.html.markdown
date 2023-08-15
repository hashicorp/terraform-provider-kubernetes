---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service"
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
  zone_id = "data.aws_route53_zone.k8.zone_id"
  name    = "example"
  type    = "CNAME"
  ttl     = "300"
  records = [data.kubernetes_service.example.status.0.load_balancer.0.ingress.0.hostname]
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard service's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)

## Attributes

* `spec` - Spec defines the behavior of a service. [Kubernetes reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)

## Nested Blocks

### `metadata`

#### Arguments

* `name` - Name of the service, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

#### Attributes

* `annotations` - (Optional) An unstructured key value map stored with the service that may be used to store arbitrary metadata. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this service. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `port`

#### Attributes

* `app_protocol` - (Optional) The application protocol for this port. This field follows standard Kubernetes label syntax. Un-prefixed names are reserved for IANA standard service names (as per [RFC-6335](https://datatracker.ietf.org/doc/html/rfc6335) and [IANA standard service names](http://www.iana.org/assignments/service-names)). Non-standard protocols should use prefixed names such as `mycompany.com/my-custom-protocol`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#application-protocol)
* `name` - The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service.
* `node_port` - The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport)
* `port` - The port that will be exposed by this service.
* `protocol` - The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.
* `target_port` - Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = "None"`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service)

### `spec`

#### Attributes

* `allocate_load_balancer_node_ports` - (Optional) Defines if `NodePorts` will be automatically allocated for services with type `LoadBalancer`. It may be set to `false` if the cluster load-balancer does not rely on `NodePorts`.  If the caller requests specific `NodePorts` (by specifying a value), those requests will be respected, regardless of this field. This field may only be set for services with type `LoadBalancer`. Default is `true`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-nodeport-allocation)
* `cluster_ip` - The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies)
* `cluster_ips` - (Optional) List of IP addresses assigned to this service, and are usually assigned randomly. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise creation of the service will fail. If this field is not specified, it will be initialized from the `clusterIP` field. If this field is specified, clients must ensure that `clusterIPs[0]` and `clusterIP` have the same value. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies)
* `external_ips` - A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP.  A common example is external load-balancers that are not part of the Kubernetes system.
* `external_name` - The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`.
* `external_traffic_policy` - (Optional) Denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints. `Local` preserves the client source IP and avoids a second hop for LoadBalancer and Nodeport type services, but risks potentially imbalanced traffic spreading. `Cluster` obscures the client source IP and may cause a second hop to another node, but should have good overall load-spreading. For more info see [Kubernetes reference](https://kubernetes.io/docs/tutorials/services/source-ip/)
* `ip_families` - (Optional) A list of IP families (e.g. IPv4, IPv6) assigned to this service. This field is usually assigned automatically based on cluster configuration and the `ip_family_policy` field. If this field is specified manually, the requested family is available in the cluster, and `ip_family_policy` allows it, it will be used; otherwise creation of the service will fail. This field is conditionally mutable: it allows for adding or removing a secondary IP family, but it does not allow changing the primary IP family of the Service. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/dual-stack/)
* `ip_family_policy` - (Optional) Represents the dual-stack-ness requested or required by this Service. If there is no value provided, then this field will be set to `SingleStack`. Services can be `SingleStack`(a single IP family), `PreferDualStack`(two IP families on dual-stack configured clusters or a single IP family on single-stack clusters), or `RequireDualStack`(two IP families on dual-stack configured clusters, otherwise fail). The `ip_families` and `cluster_ip` fields depend on the value of this field.
* `internal_traffic_policy` - (Optional) Specifies if the cluster internal traffic should be routed to all endpoints or node-local endpoints only. `Cluster` routes internal traffic to a Service to all endpoints. `Local` routes traffic to node-local endpoints only, traffic is dropped if no node-local endpoints are ready. The default value is `Cluster`.
* `load_balancer_class` - (Optional) The class of the load balancer implementation this Service belongs to. If specified, the value of this field must be a label-style identifier, with an optional prefix. This field can only be set when the Service type is `LoadBalancer`. If not set, the default load balancer implementation is used. This field can only be set when creating or updating a Service to type `LoadBalancer`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-class)
* `load_balancer_ip` - Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.
* `load_balancer_source_ranges` - If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. For more info see [Kubernetes reference](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/).
* `port` - The list of ports that are exposed by this service. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies)
* `selector` - Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/)
* `session_affinity` - Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies)
* `session_affinity_config` - (Optional) Contains the configurations of session affinity. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#proxy-mode-ipvs)
* `type` - Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types)


## Attributes

* `status` - Status is a list containing the most recently observed status of the service. Populated by the system. Read-only. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

### `status`
#### Attributes

* `load_balancer` - a list containing the current status of the load-balancer, if one is present.

### `load_balancer`
#### Attributes

* `ingress` - a list containing ingress points for the load-balancer. Traffic intended for the service should be sent to these ingress points.

### `ingress`
#### Attributes

* `ip` -  IP is set for load-balancer ingress points that are IP based (typically GCE or OpenStack load-balancers).
* `hostname` - Hostname is set for load-balancer ingress points that are DNS based (typically AWS load-balancers).


