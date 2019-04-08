---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_endpoints"
sidebar_current: "docs-kubernetes-resource-endpoint-x"
description: |-
  An Endpoints resource is an abstraction, linked to a Service, which defines the list of endpoints that actually implement the service.
---

# kubernetes_endpoints

An Endpoints resource is an abstraction, linked to a Service, which defines the list of endpoints that actually implement the service.


## Example Usage

```hcl
resource "kubernetes_endpoints" "example" {
  metadata {
    name = "terraform-example"
  }

  subsets {
    addresses {
      ip = "10.0.0.4"
    }

    ports {
      port     = 80
      protocol = "TCP"
    }
  }
}

resource "kubernetes_service" "example" {
  metadata {
    name = "${kubernetes_endpoints.example.metadata.0.name}"
  }

  spec {
    port {
      port       = 8080
      targetPort = 80
    }
  }
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard endpoints' metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata)
* `subsets` - (Optional) A list of IP address(es) and port(s) that comprise the target service.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the endpoints resource that may be used to store arbitrary metadata. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/annotations)
* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the endpoints resource. May match selectors of replication controllers and services. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/labels)
* `name` - (Optional) Name of the endpoints resource, must be unique. Cannot be updated. This name should correspond with an accompanying Service resource. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#names)
* `namespace` - (Optional) Namespace defines the space within which name of the endpoints resource must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this endpoints resource that can be used by clients to determine when endpoints resource has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `self_link` - A URL representing this endpoints resource.
* `uid` - The unique in time and space value for this endpoints resource. For more info see [Kubernetes reference](http://kubernetes.io/docs/user-guide/identifiers#uids)

### `subsets`

#### Arguments

* `addresses` - (Optional) A list of IP addresses which offer the related ports and are ready to accept traffic. These endpoints should be considered safe for load balancers and clients to utilize.
* `not_ready_addresses` - (Optional) A list of IP addresses which offer the related ports but are not currently marked as ready because they have not yet finished starting, have recently failed a readiness check, or have recently failed a liveness check.
* `ports` - (Optional) A list of port numbers available on the related IP addresses.

### `addresses`

#### Attributes

* `ip` - The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).
* `hostname` - (Optional) The Hostname of this endpoint.
* `node_name` - (Optional) Node hosting this endpoint. This can be used to determine endpoints local to a node.

### `not_ready_addresses`

#### Attributes

* `ip` - The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).
* `hostname` - (Optional) The Hostname of this endpoint.
* `node_name` - (Optional) Node hosting this endpoint. This can be used to determine endpoints local to a node.

### `ports`

#### Arguments

* `name` - (Optional) The name of this port within the endpoint. All ports within the endpoint must have unique names. Optional if only one port is defined on this endpoint.
* `port` - (Required) The port that will be utilized by this endpoint.
* `protocol` - (Optional) The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.

## Import

An Endpoints resource can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_endpoints.example default/terraform-name
```
