---
subcategory: "networking/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_ingress_v1"
description: |-
  Ingress is a collection of rules that allow inbound connections to reach the endpoints defined by a backend. An Ingress can be configured to give services externally-reachable urls, load balance traffic, terminate SSL, offer name based virtual hosting etc.
---

# kubernetes_ingress_v1

Ingress is a collection of rules that allow inbound connections to reach the endpoints defined by a backend. An Ingress can be configured to give services externally-reachable urls, load balance traffic, terminate SSL, offer name based virtual hosting etc.
This data source allows you to pull data about such ingress.

## Example Usage

```hcl
data "kubernetes_ingress_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "aws_route53_record" "example" {
  zone_id = data.aws_route53_zone.k8.zone_id
  name    = "example"
  type    = "CNAME"
  ttl     = "300"
  records = [data.kubernetes_ingress_v1.example.status.0.load_balancer.0.ingress.0.hostname]
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard service's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#metadata)

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the service, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Required) Namespace defines the space within which name of the service must be unique.

#### Attributes

* `annotations` - (Optional) An unstructured key value map stored with the service that may be used to store arbitrary metadata. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/e59e666e3464c7d4851136baa8835a311efdfb8e/contributors/devel/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this service. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

## Attribute Reference

### `spec`

#### Attributes

* `default_backend` - Backend defines the referenced service endpoint to which the traffic will be forwarded. See `backend` block attributes below.
* `rule` - A list of host rules used to configure the Ingress. If unspecified, or no rule matches, all traffic is sent to the default backend. See `rule` block attributes below.
* `tls` - TLS configuration. Currently the Ingress only supports a single TLS port, 443. If multiple members of this list specify different hosts, they will be multiplexed on the same port according to the hostname specified through the SNI TLS extension, if the ingress controller fulfilling the ingress supports SNI. See `tls` block attributes below.
* `ingress_class_name` - The name of the IngressClass cluster resource. The associated IngressClass defines which controller will implement the resource. This replaces the deprecated `kubernetes.io/ingress.class` annotation. For backwards compatibility, when that annotation is set, it must be given precedence over this field.

### `backend`

#### Arguments

* `resource` - Resource is an ObjectRef to another Kubernetes resource in the namespace of the Ingress object. If resource is specified, a `service.name` and `service.port` must not be specified.
* `service` - Service references a Service as a Backend.

### `service`

#### Arguments

* `name` - Specifies the name of the referenced service.
* `port` - Specifies the port of the referenced service.

### `port`

* `name` - Name is the name of the port on the Service. 
* `number` - Number is the numerical port number (e.g. 80) on the Service. 

### `rule`

#### Attributes

* `host` - Host is the fully qualified domain name of a network host, as defined by RFC 3986. Note the following deviations from the \"host\" part of the URI as defined in the RFC: 1. IPs are not allowed. Currently an IngressRuleValue can only apply to the IP in the Spec of the parent Ingress. 2. The : delimiter is not respected because ports are not allowed. Currently the port of an Ingress is implicitly :80 for http and :443 for https. Both these may change in the future. Incoming requests are matched against the host before the IngressRuleValue. If the host is unspecified, the Ingress routes all traffic based on the specified IngressRuleValue.
* `http` - http is a list of http selectors pointing to backends. In the example: http:///? -> backend where parts of the url correspond to RFC 3986, this resource will be used to match against everything after the last '/' and before the first '?' or '#'. See `http` block attributes below.


#### `http`

* `path` - Path array of path regex associated with a backend. Incoming urls matching the path are forwarded to the backend, see below for `path` block structure.

#### `path`

* `path` - (Required)  A string or an extended POSIX regular expression as defined by IEEE Std 1003.1, (i.e this follows the egrep/unix syntax, not the perl syntax) matched against the path of an incoming request. Currently it can contain characters disallowed from the conventional \"path\" part of a URL as defined by RFC 3986. Paths must begin with a '/'. If unspecified, the path defaults to a catch all sending traffic to the backend.
* `path_type` - PathType determines the interpretation of the Path matching. PathType can be one of the following values: `ImplementationSpecific`, `Exact`, or `Prefix`. See the [Kubernetes Ingress documentation](https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types) for details. 
* `backend` - Backend defines the referenced service endpoint to which the traffic will be forwarded to.

### `tls`

#### Attributes

* `hosts` - Hosts are a list of hosts included in the TLS certificate. The values in this list must match the name/s used in the tlsSecret. Defaults to the wildcard host setting for the loadbalancer controller fulfilling this Ingress, if left unspecified.
* `secret_name` - SecretName is the name of the secret used to terminate SSL traffic on 443. Field is left optional to allow SSL routing based on SNI hostname alone. If the SNI host in a listener conflicts with the \"Host\" header field used by an IngressRule, the SNI host is used for termination and value of the Host header is used for routing.

## Attributes

### `status`

* `status` - Status is the current state of the Ingress. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status

#### `load_balancer`

* LoadBalancer contains the current status of the load-balancer, if one is present.

##### `ingress`

* `ingress` - Ingress is a list containing ingress points for the load-balancer. Traffic intended for the service should be sent to these ingress points.

###### Attributes

* `ip` -  IP is set for load-balancer ingress points that are IP based (typically GCE or OpenStack load-balancers).
* `hostname` - Hostname is set for load-balancer ingress points that are DNS based (typically AWS load-balancers).
