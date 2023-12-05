---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_endpoints_v1"
description: |-
    An Endpoints resource is an abstraction, linked to a Service, which defines the list of endpoints that actually implement the service.
---

# kubernetes_endpoints_v1

An Endpoints resource is an abstraction, linked to a Service, which defines the list of endpoints that actually implement the service.


## Example Usage

```hcl
data "kubernetes_endpoints_v1" "api_endpoints" {
  metadata {
    name      = "kubernetes"
    namespace = "default"
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard endpoints' metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the endpoints resource.
* `namespace` - (Optional) Namespace defines the space within which name of the endpoints resource must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this endpoints resource that can be used by clients to determine when endpoints resource has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this endpoints resource. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

## Attribute Reference

### `subset`

#### Attributes

* `address` - (Optional) An IP address block which offers the related ports and is ready to accept traffic. These endpoints should be considered safe for load balancers and clients to utilize. Can be repeated multiple times.
* `not_ready_address` - (Optional) A IP address block which offers the related ports but is not currently marked as ready because it have not yet finished starting, have recently failed a readiness check, or have recently failed a liveness check. Can be repeated multiple times.
* `port` - (Optional) A port number block available on the related IP addresses. Can be repeated multiple times.

### `address`

#### Attributes

* `ip` - The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).
* `hostname` - (Optional) The Hostname of this endpoint.
* `node_name` - (Optional) Node hosting this endpoint. This can be used to determine endpoints local to a node.

### `not_ready_address`

#### Attributes

* `ip` - The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).
* `hostname` - (Optional) The Hostname of this endpoint.
* `node_name` - (Optional) Node hosting this endpoint. This can be used to determine endpoints local to a node.

### `port`

#### Attributes

* `name` - (Optional) The name of this port within the endpoint. All ports within the endpoint must have unique names. Optional if only one port is defined on this endpoint.
* `port` - (Required) The port that will be utilized by this endpoint.
* `protocol` - (Optional) The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.

