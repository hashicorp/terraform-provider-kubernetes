---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_ingress_v1"
description: |-
  Ingress is a collection of rules that allow inbound connections to reach the endpoints defined by a backend. An Ingress can be configured to give services externally-reachable urls, load balance traffic, terminate SSL, offer name based virtual hosting etc.
---

# kubernetes_ingress_v1

Ingress is a collection of rules that allow inbound connections to reach the endpoints defined by a backend. An Ingress can be configured to give services externally-reachable urls, load balance traffic, terminate SSL, offer name based virtual hosting etc.


## Example Usage

```hcl
resource "kubernetes_ingress_v1" "example_ingress" {
  metadata {
    name = "example-ingress"
  }

  spec {
    default_backend {
      service {
        name = "MyApp1"
        port {
          number = 8080
        }
      }
    }

    rule {
      http {
        path {
          backend {
            service_name = "MyApp1"
            service_port = 8080
          }

          path = "/app1/*"
        }

        path {
          backend {
            service_name = "MyApp2"
            service_port = 8080
          }

          path = "/app2/*"
        }
      }
    }

    tls {
      secret_name = "tls-secret"
    }
  }
}

resource "kubernetes_pod" "example" {
  metadata {
    name = "terraform-example"
    labels = {
      app = "MyApp1"
    }
  }

  spec {
    container {
      image = "nginx:1.7.9"
      name  = "example"

      port {
        container_port = 8080
      }
    }
  }
}

resource "kubernetes_pod" "example2" {
  metadata {
    name = "terraform-example2"
    labels = {
      app = "MyApp2"
    }
  }

  spec {
    container {
      image = "nginx:1.7.9"
      name  = "example"

      port {
        container_port = 8080
      }
    }
  }
}
```

## Example using Nginx ingress controller

```
resource "kubernetes_service" "example" {
  metadata {
    name = "ingress-service"
  }
  spec {
    port {
      port = 80
      target_port = 80
      protocol = "TCP"
    }
    type = "NodePort"
  }
}

resource "kubernetes_ingress_v1" "example" {
  wait_for_load_balancer = true
  metadata {
    name = "example"
    annotations = {
      "kubernetes.io/ingress.class" = "nginx"
    }
  }
  spec {
    rule {
      http {
        path {
          path = "/*"
          backend {
            service_name = kubernetes_service.example.metadata.0.name
            service_port = 80
          }
        }
      }
    }
  }
}

# Display load balancer hostname (typically present in AWS)
output "load_balancer_hostname" {
  value = kubernetes_ingress_v1.example.status.0.load_balancer.0.ingress.0.hostname
}

# Display load balancer IP (typically present in GCP, or using Nginx ingress controller)
output "load_balancer_ip" {
  value = kubernetes_ingress_v1.example.status.0.load_balancer.0.ingress.0.ip
}
```



## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard ingress's metadata. For more info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
* `spec` - (Required) Spec defines the behavior of a ingress. https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
* `wait_for_load_balancer` - (Optional) Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created. Defaults to `false`.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the ingress that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info: http://kubernetes.io/docs/user-guide/annotations

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services.

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info: http://kubernetes.io/docs/user-guide/labels

* `name` - (Optional) Name of the service, must be unique. Cannot be updated. For more info: http://kubernetes.io/docs/user-guide/identifiers#names
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
* `uid` - The unique in time and space value for this service. For more info: http://kubernetes.io/docs/user-guide/identifiers#uids

### `spec`

#### Arguments

* `default_backend` - (Optional) DefaultBackend is the backend that should handle requests that don't match any rule. If Rules are not specified, DefaultBackend must be specified. If DefaultBackend is not set, the handling of requests that do not match any of the rules will be up to the Ingress controller. See `backend` block attributes below.
* `rule` - (Optional) A list of host rules used to configure the Ingress. If unspecified, or no rule matches, all traffic is sent to the default backend. See `rule` block attributes below.
* `tls` - (Optional) TLS configuration. Currently the Ingress only supports a single TLS port, 443. If multiple members of this list specify different hosts, they will be multiplexed on the same port according to the hostname specified through the SNI TLS extension, if the ingress controller fulfilling the ingress supports SNI. See `tls` block attributes below.
* `ingress_class_name` - (Optional) The ingress class name references an IngressClass resource that contains additional configuration including the name of the controller that should implement the class.

### `backend`

#### Arguments


* `resource` - (Optional) Resource is an ObjectRef to another Kubernetes resource in the namespace of the Ingress object. If resource is specified, a `service.name` and `service.port` must not be specified.
* `service` - (Optional) Service references a Service as a Backend.

### `service`

#### Arguments

* `name` - (Optional) Specifies the name of the referenced service.
* `port` - (Optional) Specifies the port of the referenced service.

### `port`

* `name` - (Optional) Name is the name of the port on the Service. 
* `number` - (Optional) Number is the numerical port number (e.g. 80) on the Service. 


#### Arguments

### `rule`

#### Arguments

* `host` - (Optional) Host is the fully qualified domain name of a network host, as defined by RFC 3986. Note the following deviations from the \"host\" part of the URI as defined in the RFC: 1. IPs are not allowed. Currently an IngressRuleValue can only apply to the IP in the Spec of the parent Ingress. 2. The : delimiter is not respected because ports are not allowed. Currently the port of an Ingress is implicitly :80 for http and :443 for https. Both these may change in the future. Incoming requests are matched against the host before the IngressRuleValue. If the host is unspecified, the Ingress routes all traffic based on the specified IngressRuleValue.
* `http` - (Required) http is a list of http selectors pointing to backends. In the example: http:///? -> backend where parts of the url correspond to RFC 3986, this resource will be used to match against everything after the last '/' and before the first '?' or '#'. See `http` block attributes below.


#### `http`

* `path` - (Required) Path array of path regex associated with a backend. Incoming urls matching the path are forwarded to the backend, see below for `path` block structure.

#### `path`

* `path` - (Required)  A string or an extended POSIX regular expression as defined by IEEE Std 1003.1, (i.e this follows the egrep/unix syntax, not the perl syntax) matched against the path of an incoming request. Currently it can contain characters disallowed from the conventional \"path\" part of a URL as defined by RFC 3986. Paths must begin with a '/'. If unspecified, the path defaults to a catch all sending traffic to the backend.
* `path_type` - (Optional) PathType determines the interpretation of the Path matching. PathType can be one of the following values: * Exact: Matches the URL path exactly. * Prefix: Matches based on a URL path prefix split by '/'. Matching is done on a path element by element basis. A path element refers is the list of labels in the path split by the '/' separator. A request is a match for path p if every p is an element-wise prefix of p of the request path. Note that if the last element of the path is a substring of the last element in request path, it is not a match (e.g. /foo/bar matches /foo/bar/baz, but does not match /foo/barbaz). * ImplementationSpecific: Interpretation of the Path matching is up to the IngressClass. Implementations can treat this as a separate PathType or treat it identically to Prefix or Exact path types. Implementations are required to support all path types.
* `backend` - (Required) Backend defines the referenced service endpoint to which the traffic will be forwarded to.

### `tls`

#### Arguments

* `hosts` - (Optional) Hosts are a list of hosts included in the TLS certificate. The values in this list must match the name/s used in the tlsSecret. Defaults to the wildcard host setting for the loadbalancer controller fulfilling this Ingress, if left unspecified.
* `secret_name` - (Optional) SecretName is the name of the secret used to terminate SSL traffic on 443. Field is left optional to allow SSL routing based on SNI hostname alone. If the SNI host in a listener conflicts with the \"Host\" header field used by an IngressRule, the SNI host is used for termination and value of the Host header is used for routing.

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


## Import

Ingress can be imported using its namespace and name:

```
terraform import kubernetes_ingress_v1.<TERRAFORM_RESOURCE_NAME> <KUBE_NAMESPACE>/<KUBE_INGRESS_NAME>
```

e.g.

```
$ terraform import kubernetes_ingress_v1.example default/terraform-name
```
