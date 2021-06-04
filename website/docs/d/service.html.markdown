---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service"
description: |-
  A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.
---

# kubernetes_service

A Service is an abstraction which defines a logical set of pods and a policy by
which to access them - sometimes called a micro-service.

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

* `metadata` - (Required) Standard service's metadata. For more information see
[this reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
.

This argument is a block with the following arguments:

* `name` - Name of the service, must be unique. Cannot be updated. For more
information see [this reference](http://kubernetes.io/docs/user-guide/identifiers#names).
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

## Attributes

Besides the arguments provided, you can fetch the following list of attributes.

### spec

`spec` defines the behavior of a service. For more information see
[this reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status).

`spec` itself is a list of maps:

```hcl
spec = [
  {
    cluster_ip                  = "172.20.99.5"
    external_ips                = []
    external_name               = ""
    external_traffic_policy     = ""
    health_check_node_port      = 0
    load_balancer_ip            = ""
    load_balancer_source_ranges = []
    port = [
      {
        name        = "web"
        node_port   = 0
        port        = 9090
        protocol    = "TCP"
        target_port = "9090"
      },
    ]
    publish_not_ready_addresses = false
    selector = {
      "app"        = "prometheus"
      "prometheus" = "kube-prometheus-stack-prometheus"
    }
    session_affinity = "None"
    type             = "ClusterIP"
  }
]
```

Each map has the following structure:

* `cluster_ip` - The IP address of the service. It is usually assigned randomly
by the master. If an address is specified manually and is not in use by others,
it will be allocated to the service; otherwise, creation of the service will
fail. `None` can be specified for headless services when proxying is not
required. Ignored if type is `ExternalName`. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies).
* `external_ips` - A list of IP addresses for which nodes in the cluster will
also accept traffic for this service. These IPs are not managed by Kubernetes.
The user is responsible for ensuring that traffic arrives at a node with this
IP.  A common example is external load-balancers that are not part of the
Kubernetes system.
* `external_name` - The external reference that `kubedns` or equivalent will
return as a CNAME record for this service. No proxying will be involved. Must be
a valid DNS name and requires `type` to be `ExternalName`.
* `external_traffic_policy` - (Optional) Denotes if this Service desires to
route external traffic to node-local or cluster-wide endpoints. `Local`
preserves the client source IP and avoids a second hop for `LoadBalancer` and
`Nodeport` type services, but risks potentially imbalanced traffic spreading.
`Cluster` obscures the client source IP and may cause a second hop to another
node, but should have good overall load-spreading. For more information see
[this reference](https://kubernetes.io/docs/tutorials/services/source-ip/).
* `load_balancer_ip` - Only applies to `type = LoadBalancer`. `LoadBalancer`
will get created with the IP specified in this field. This feature depends on
whether the underlying cloud provider supports specifying this field when a load
balancer is created. This field will be ignored if the cloud-provider does not
support the feature.
* `load_balancer_source_ranges` - If specified and supported by the platform,
this will restrict traffic through the cloud-provider load-balancer will be
restricted to the specified client IPs. This field will be ignored if the
cloud provider does not support the feature. For more information see
[this reference](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/).
* `port` - The list of ports that are exposed by this service. For more
information see [this reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies)
* `selector` - Route service traffic to pods with label keys and values matching
this selector. Only applies to types `ClusterIP`, `NodePort`, and
`LoadBalancer`. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#overview).
* `session_affinity` - Used to maintain session affinity. Supports `ClientIP`
and `None`. Defaults to `None`. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies).
* `type` - Determines how the service is exposed. Defaults to `ClusterIP`. Valid
options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`.
`ExternalName` maps to the specified `external_name`. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#overview).

Since `port` is a data structure itself, it is detailed in the next section as well.

#### port

* `name` - The name of this port within the service. All ports within the
service must have unique names. Optional if only one `ServicePort` is defined on
this service.
* `node_port` - The port on each node on which this service is exposed when
`type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If
specified, it will be allocated to the service if unused or else creation of the
service will fail. Default is to auto-allocate a port if the `type` of this
service requires one. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#type--nodeport).
* `port` - The port that will be exposed by this service.
* `protocol` - The IP protocol for this port. Supports `TCP` and `UDP`. Default
is `TCP`.
* `target_port` - Number or name of the port to access on the pods targeted by
the service. Number must be in the range 1 to 65535. This field is ignored for
services with `cluster_ip = "None"`. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#defining-a-service).

### `metadata`

`metadata` is metadata that all persisted resources must have, which includes
all objects users must create.. For more information, see
[this reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata).

`metadata` itself is a list of maps:

```hcl
metadata = [
  {
    "annotations" = tomap({
      "meta.helm.sh/release-name"      = "kube-prometheus-stack"
      "meta.helm.sh/release-namespace" = "observability"
    })
    "generation" = 0
    "labels" = tomap({
      "app"          = "kube-prometheus-stack-prometheus"
      "chart"        = "kube-prometheus-stack-16.0.1"
      "heritage"     = "Helm"
      "release"      = "kube-prometheus-stack"
      "self-monitor" = "true"
    })
    "name"             = "kube-prometheus-stack-prometheus"
    "namespace"        = "observability"
    "resource_version" = "5285082"
    "self_link"        = ""
    "uid"              = "f3e28348-7f14-4f01-b32b-5e558ed6a132"
  },
]
```

Each map has the following structure:

* `annotations` - (Optional) An unstructured key value map stored with the
service that may be used to store arbitrary metadata. For more information see
[this reference](http://kubernetes.io/docs/user-guide/annotations).
* `labels` - (Optional) Map of string keys and values that can be used to
organize and categorize (scope and select) the service. May match selectors of
replication controllers and services. For more information see
[this reference](http://kubernetes.io/docs/user-guide/labels).
* `generation` - A sequence number representing a specific generation of the
desired state.
* `resource_version` - An opaque value that represents the internal version of
this service that can be used by clients to determine when service has changed.
For more information see
[this reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency).
* `uid` - The unique in time and space value for this service. For more
information see
[this reference](http://kubernetes.io/docs/user-guide/identifiers#uids).

### `status`

`status` is a list containing the most recently observed status of
the service. Populated by the system. Read-only. More information see
[this reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status).

`status` itself is a list of maps:

```hcl
status = [
  {
    "load_balancer" = [
      {
        "ingress" = []
      }
    ]
  }
]
```

Each map has the following structure:

* `load_balancer` - a list (of maps) containing the current status of the
load-balancer, if one is present.

Then each map might have the following structure:

* `ingress` - a list (of maps) containing ingress points for the load-balancer.
Traffic intended for the service should be sent to these ingress points.

Then each map might have the following structure:

* `ip` -  IP is set for load-balancer ingress points that are IP based
(typically GCE or OpenStack load-balancers).
* `hostname` - Hostname is set for load-balancer ingress points that are DNS
based (typically AWS load-balancers).
