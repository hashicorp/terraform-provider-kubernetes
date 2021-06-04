---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service"
description: |-
  A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.
---

# kubernetes_service

A Service is an abstraction which defines a logical set of pods and a policy by
which to access them - sometimes called a micro-service.

## Example Usage

```hcl
resource "kubernetes_service" "example" {
  metadata {
    name = "terraform-example"
  }
  spec {
    selector = {
      app = kubernetes_pod.example.metadata.0.labels.app
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

## Example using AWS load balancer

```hcl
variable "cluster_name" {
  type = string
}

data "aws_eks_cluster" "example" {
  name = var.cluster_name
}

data "aws_eks_cluster_auth" "example" {
  name = var.cluster_name
}

provider "aws" {
  region = "us-west-1"
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.example.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.example.certificate_authority[0].data)
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
    command     = "aws"
  }
}

resource "kubernetes_service" "example" {
  metadata {
    name = "example"
  }
  spec {
    port {
      port        = 8080
      target_port = 80
    }
    type = "LoadBalancer"
  }
}

# Create a local variable for the load balancer name.
locals {
  lb_name = split("-", split(".", kubernetes_service.example.status.0.load_balancer.0.ingress.0.hostname).0).0
}

# Read information about the load balancer using the AWS provider.
data "aws_elb" "example" {
  name = local.lb_name
}

output "load_balancer_name" {
  value = local.lb_name
}

output "load_balancer_hostname" {
  value = kubernetes_service.example.status.0.load_balancer.0.ingress.0.hostname
}

output "load_balancer_info" {
  value = data.aws_elb.example
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard service's metadata. For more information see
[this reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata).
* `spec` - (Required) Spec defines the behavior of a service. For more
information see [this reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status).
* `wait_for_load_balancer` - (Optional) Terraform will wait for the load
balancer to have at least 1 endpoint before considering the resource created.
Defaults to `true`.
* `timeouts` - provides the
[Timeouts](/docs/language/resources/syntax.html#operation-timeouts)
configuration options.

### `metadata` arguments

* `annotations` - (Optional) An unstructured key value map stored with the
service that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names are in the
[Well-Known Labels, Annotations and Taints](https://kubernetes.io/docs/reference/labels-annotations-taints).
This is necessary because such annotations can be mutated by server-side
components and consequently cause a perpetual diff in the Terraform `plan`
output. If you explicitly specify any such annotations in the configuration
template then Terraform will consider these as normal resource attributes and
manage them as expected (while still avoiding the perpetual diff problem). For
more information info see
[this reference](http://kubernetes.io/docs/user-guide/annotations).

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique
name ONLY IF the `name` field has not been provided. This value will also be
combined with a unique suffix. For more information see
[this reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency).
* `labels` - (Optional) Map of string keys and values that can be used to
organize and categorize (scope and select) the service. May match selectors of
replication controllers and services.

~> By default, the provider ignores any labels whose key names are in the
[Well-Known Labels, Annotations and Taints](https://kubernetes.io/docs/reference/labels-annotations-taints).
This is necessary because such labels can be mutated by server-side components
and consequently cause a perpetual diff in the Terraform `plan` output. If you
explicitly specify any such labels in the configuration template then Terraform
will consider these as normal resource attributes and manage them as expected
(while still avoiding the perpetual diff problem). For more information see
[this reference](http://kubernetes.io/docs/user-guide/labels).

* `name` - (Optional) Name of the service, must be unique. Cannot be updated.
For more info see
[this reference](http://kubernetes.io/docs/user-guide/identifiers#names).
* `namespace` - (Optional) Defines the space within which name of the
service must be unique.

### `spec` arguments

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
* `publish_not_ready_addresses` - (Optional) When set to true, indicates that
DNS implementations must publish the `notReadyAddresses` of subsets for the
Endpoints associated with the Service. The default value is `false`. The primary
use case for setting this field is to use a StatefulSet's Headless Service to
propagate `SRV` records for its Pods without respect to their readiness for
purpose of peer discovery.
* `selector` - (Optional) Route service traffic to pods with label keys and
values matching this selector. Only applies to types `ClusterIP`, `NodePort`,
and `LoadBalancer`. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#overview).
* `session_affinity` - (Optional) Used to maintain session affinity. Supports
`ClientIP` and `None`. Defaults to `None`. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies).
* `type` - (Optional) Determines how the service is exposed. Defaults to
`ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and
`LoadBalancer`. `ExternalName` maps to the specified `external_name`. For more
information see
[this reference](http://kubernetes.io/docs/user-guide/services#overview).
* `health_check_node_port` - (Optional) Specifies the Healthcheck NodePort for
the service. Only effects when type is set to `LoadBalancer` and
`external_traffic_policy` is set to `Local`.
* `port` - The list of ports that are exposed by this service. For more
information see
[this reference](http://kubernetes.io/docs/user-guide/services#virtual-ips-and-service-proxies).

Since `port` is block itself, each one of the available properties are detailed
below:

* `name` - (Optional) The name of this port within the service. All ports within
the service must have unique names. Optional if only one `ServicePort` is
defined on this service.
* `node_port` - (Optional) The port on each node on which this service is
exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the
system. If specified, it will be allocated to the service if unused or else
creation of the service will fail. Default is to auto-allocate a port if the
`type` of this service requires one. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#type--nodeport).
* `port` - (Required) The port that will be exposed by this service.
* `protocol` - (Optional) The IP protocol for this port. Supports `TCP` and
`UDP`. Default is `TCP`.
* `target_port` - (Optional) Number or name of the port to access on the pods
targeted by the service. Number must be in the range 1 to 65535. This field is
ignored for services with `cluster_ip = "None"`. For more information see
[this reference](http://kubernetes.io/docs/user-guide/services#defining-a-service).

## Attributes

Besides the arguments provided, you can fetch the following list of attributes.

### `metadata`

* `generation`: a sequence number representing a specific generation of the
desired state.
* `resource_version`: an opaque value that represents the internal version of
this service that can be used by clients to determine when service has changed.
For more information see
[this reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency).
* `uid`: the unique in time and space value for this service. For more
information see
[this reference](http://kubernetes.io/docs/user-guide/identifiers#uids).
* `self_link`: a URL representing this object. Populated by the system.
Read-only.
* `status`: a list containing the most recently observed status of
the service. Populated by the system. Read-only. For more information see
[this reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status).

And since `status` itself is a list of maps, each one will have the following
structure:

* `load_balancer`: a list (of maps) containing the current status of the
load-balancer, if one is present.

Then each map might have the following structure:

* `ingress`: a list (of maps) containing ingress points for the load-balancer.
Traffic intended for the service should be sent to these ingress points.

Then each map might have the following structure:

* `ip`: IP is set for load-balancer ingress points that are IP based
(typically GCE or OpenStack load-balancers).
* `hostname`: Hostname is set for load-balancer ingress points that are DNS
based (typically AWS load-balancers).

As a reference, here is a complete example of a `kubernetes_service`
information:

```hcl
example = {
  "id" = "observability/example"
  "metadata" = [
    {
      "annotations"      = tomap(null) /* of string */
      "generate_name"    = ""
      "generation"       = 0
      "labels"           = tomap(null) /* of string */
      "name"             = "example"
      "namespace"        = "observability"
      "resource_version" = "9944172"
      "self_link"        = ""
      "uid"              = "1456d1b1-ea66-4837-827a-5756acef60d7"
    },
  ]
  "spec" = [
    {
      "cluster_ip"                  = "172.20.156.16"
      "external_ips"                = []
      "external_name"               = ""
      "external_traffic_policy"     = "Cluster"
      "health_check_node_port"      = 0
      "load_balancer_ip"            = ""
      "load_balancer_source_ranges" = []
      "port" = [
        {
          "name"        = ""
          "node_port"   = 30466
          "port"        = 8080
          "protocol"    = "TCP"
          "target_port" = "80"
        },
      ]
      "publish_not_ready_addresses" = false
      "selector"                    = {}
      "session_affinity"            = "None"
      "type"                        = "LoadBalancer"
    },
  ]
  "status" = [
    {
      "load_balancer" = [
        {
          "ingress" = [
            {
              "hostname" = "a1456d1b1ea664837827a5756acef60d-1786071345.us-east-2.elb.amazonaws.com"
              "ip"       = ""
            },
          ]
        },
      ]
    },
  ]
  "timeouts"               = {}
  "wait_for_load_balancer" = true
}
```

## Import

Service can be imported using it's `namespace` and `name`, e.g.

```
$ terraform import kubernetes_service.example default/terraform-name
```
