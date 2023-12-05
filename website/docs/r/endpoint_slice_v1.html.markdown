---
subcategory: "discovery/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_endpoint_slice_v1"
description: |-
  An EndpointSlice contains references to a set of network endpoints.
---

# kubernetes_endpoints_slice_v1

An EndpointSlice contains references to a set of network endpoints.

## Example Usage

```hcl
resource "kubernetes_endpoint_slice_v1" "test" {
  metadata {
    name = "test"
  }

  endpoint {
    condition {
      ready = true
    }
    addresses = ["129.144.50.56"]
  }

  port {
    port = "9000"
    name = "first"
  }

  address_type = "IPv4"
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard endpoints' metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `address_type` - (Required) Specifies the type of address carried by this EndpointSlice. All addresses in this slice must be the same type. This field is immutable after creation. The following address types are currently supported: *IPv4: Represents an IPv4 Address.* IPv6: Represents an IPv6 Address. * FQDN: Represents a Fully Qualified Domain Name.
* `endpoint` - (Required) A list of unique endpoints in this slice. Each slice may include a maximum of 1000 endpoints.
* `port` - (Required) Specifies the list of network ports exposed by each endpoint in this slice. Each port must have a unique name. When ports is empty, it indicates that there are no defined ports. When a port is defined with a nil port value, it indicates "all ports". Each slice may include a maximum of 100 ports.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the endpoints resource that may be used to store arbitrary metadata. 

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the endpoints resource. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the endpoints resource, must be unique. Cannot be updated. This name should correspond with an accompanying Service resource. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the endpoints resource must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this endpoints resource that can be used by clients to determine when endpoints resource has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this endpoints resource. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `endpoint`

#### Arguments

* `addresses` - (Required) addresses of this endpoint. The contents of this field are interpreted according to the corresponding EndpointSlice addressType field. Consumers must handle different types of addresses in the context of their own capabilities. This must contain at least one address but no more than 100. 
* `condition` - (Optional) Contains information about the current status of the endpoint.
* `hostname` - (Optional) hostname of this endpoint. This field may be used by consumers of endpoints to distinguish endpoints from each other (e.g. in DNS names). Multiple endpoints which use the same hostname should be considered fungible (e.g. multiple A values in DNS). Must be lowercase and pass DNS Label (RFC 1123) validation.
* `node_name` - (Optional) Represents the name of the Node hosting this endpoint. This can be used to determine endpoints local to a Node.
* `target_ref` - (Optional) targetRef is a reference to a Kubernetes object that represents this endpoint.
* `zone` - (Optional) The name of the Zone this endpoint exists in.

### `condition`

#### Attributes

* `ready` - (Optional) Indicates that this endpoint is prepared to receive traffic, according to whatever system is managing the endpoint.
* `serving` - (Optional) Serving is identical to ready except that it is set regardless of the terminating state of endpoints. 
* `terminating` - (Optional) Indicates that this endpoint is terminating.

### `target_ref`

#### Attributes

* `ip` - The IP of this endpoint. May not be loopback (127.0.0.0/8), link-local (169.254.0.0/16), or link-local multicast ((224.0.0.0/24).
* `hostname` - (Optional) The Hostname of this endpoint.
* `node_name` - (Optional) Node hosting this endpoint. This can be used to determine endpoints local to a node.
* `zone` - (Optional) The name of the zone this endpoint exists in.

### `port`

#### Arguments

* `name` - (Optional) The name of this port within the endpoint. All ports within the endpoint must have unique names. Optional if only one port is defined on this endpoint.
* `port` - (Required) The port that will be utilized by this endpoint.
* `protocol` - (Optional) The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.
* `app_protocol` - (Optional) The application protocol for this port. This is used as a hint for implementations to offer richer behavior for protocols that they understand.
