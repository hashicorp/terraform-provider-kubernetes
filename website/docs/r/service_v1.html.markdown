---
subcategory: "core/v1"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_service_v1"
description: |-
  A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.
---

# kubernetes_service_v1

A Service is an abstraction which defines a logical set of pods and a policy by which to access them - sometimes called a micro-service.


## Example Usage

```hcl
resource "kubernetes_service_v1" "example" {
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
      image = "nginx:1.21.6"
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
    api_version = "client.authentication.k8s.io/v1beta1"
    args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
    command     = "aws"
  }
}

resource "kubernetes_service_v1" "example" {
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
  lb_name = split("-", split(".", kubernetes_service_v1.example.status.0.load_balancer.0.ingress.0.hostname).0).0
}

# Read information about the load balancer using the AWS provider.
data "aws_elb" "example" {
  name = local.lb_name
}

output "load_balancer_name" {
  value = local.lb_name
}

output "load_balancer_hostname" {
  value = kubernetes_service_v1.example.status.0.load_balancer.0.ingress.0.hostname
}

output "load_balancer_info" {
  value = data.aws_elb.example
}
```



## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard service's metadata. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata)
* `spec` - (Required) Spec defines the behavior of a service. [Kubernetes reference](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status)
* `wait_for_load_balancer` - (Optional) Terraform will wait for the load balancer to have at least 1 endpoint before considering the resource created. Defaults to `true`.

## Nested Blocks

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the service that may be used to store arbitrary metadata.

~> By default, the provider ignores any annotations whose key names end with *kubernetes.io*. This is necessary because such annotations can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such annotations in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)

* `generate_name` - (Optional) Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency)
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the service. May match selectors of replication controllers and services. 

~> By default, the provider ignores any labels whose key names end with *kubernetes.io*. This is necessary because such labels can be mutated by server-side components and consequently cause a perpetual diff in the Terraform plan output. If you explicitly specify any such labels in the configuration template then Terraform will consider these as normal resource attributes and manage them as expected (while still avoiding the perpetual diff problem). For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)

* `name` - (Optional) Name of the service, must be unique. Cannot be updated. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names)
* `namespace` - (Optional) Namespace defines the space within which name of the service must be unique.

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this service that can be used by clients to determine when service has changed. For more info see [Kubernetes reference](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency)
* `uid` - The unique in time and space value for this service. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids)

### `spec`

#### Arguments

* `allocate_load_balancer_node_ports` - (Optional) Defines if `NodePorts` will be automatically allocated for services with type `LoadBalancer`. It may be set to `false` if the cluster load-balancer does not rely on `NodePorts`.  If the caller requests specific `NodePorts` (by specifying a value), those requests will be respected, regardless of this field. This field may only be set for services with type `LoadBalancer`. Default is `true`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-nodeport-allocation)
* `cluster_ip` - (Optional) The IP address of the service. It is usually assigned randomly by the master. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise, creation of the service will fail. `None` can be specified for headless services when proxying is not required. Ignored if type is `ExternalName`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies)
* `cluster_ips` - (Optional) List of IP addresses assigned to this service, and are usually assigned randomly. If an address is specified manually and is not in use by others, it will be allocated to the service; otherwise creation of the service will fail. If this field is not specified, it will be initialized from the `clusterIP` field. If this field is specified, clients must ensure that `clusterIPs[0]` and `clusterIP` have the same value. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies)
* `external_ips` - (Optional) A list of IP addresses for which nodes in the cluster will also accept traffic for this service. These IPs are not managed by Kubernetes. The user is responsible for ensuring that traffic arrives at a node with this IP.  A common example is external load-balancers that are not part of the Kubernetes system.
* `external_name` - (Optional) The external reference that kubedns or equivalent will return as a CNAME record for this service. No proxying will be involved. Must be a valid DNS name and requires `type` to be `ExternalName`.
* `external_traffic_policy` - (Optional) Denotes if this Service desires to route external traffic to node-local or cluster-wide endpoints. `Local` preserves the client source IP and avoids a second hop for LoadBalancer and Nodeport type services, but risks potentially imbalanced traffic spreading. `Cluster` obscures the client source IP and may cause a second hop to another node, but should have good overall load-spreading. For more info see [Kubernetes reference](https://kubernetes.io/docs/tutorials/services/source-ip/)
* `ip_families` - (Optional) A list of IP families (e.g. IPv4, IPv6) assigned to this service. This field is usually assigned automatically based on cluster configuration and the `ip_family_policy` field. If this field is specified manually, the requested family is available in the cluster, and `ip_family_policy` allows it, it will be used; otherwise creation of the service will fail. This field is conditionally mutable: it allows for adding or removing a secondary IP family, but it does not allow changing the primary IP family of the Service. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/dual-stack/)
* `ip_family_policy` - (Optional) Represents the dual-stack-ness requested or required by this Service. If there is no value provided, then this field will be set to `SingleStack`. Services can be `SingleStack`(a single IP family), `PreferDualStack`(two IP families on dual-stack configured clusters or a single IP family on single-stack clusters), or `RequireDualStack`(two IP families on dual-stack configured clusters, otherwise fail). The `ip_families` and `cluster_ip` fields depend on the value of this field.
* `internal_traffic_policy` - (Optional) Specifies if the cluster internal traffic should be routed to all endpoints or node-local endpoints only. `Cluster` routes internal traffic to a Service to all endpoints. `Local` routes traffic to node-local endpoints only, traffic is dropped if no node-local endpoints are ready. The default value is `Cluster`.
* `load_balancer_class` - (Optional) The class of the load balancer implementation this Service belongs to. If specified, the value of this field must be a label-style identifier, with an optional prefix. This field can only be set when the Service type is `LoadBalancer`. If not set, the default load balancer implementation is used. This field can only be set when creating or updating a Service to type `LoadBalancer`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#load-balancer-class)
* `load_balancer_ip` - (Optional) Only applies to `type = LoadBalancer`. LoadBalancer will get created with the IP specified in this field. This feature depends on whether the underlying cloud-provider supports specifying this field when a load balancer is created. This field will be ignored if the cloud-provider does not support the feature.
* `load_balancer_source_ranges` - (Optional) If specified and supported by the platform, this will restrict traffic through the cloud-provider load-balancer will be restricted to the specified client IPs. This field will be ignored if the cloud-provider does not support the feature. For more info see [Kubernetes reference](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/).
* `port` - (Optional) The list of ports that are exposed by this service. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies)
* `publish_not_ready_addresses` - (Optional) When set to true, indicates that DNS implementations must publish the `notReadyAddresses` of subsets for the Endpoints associated with the Service. The default value is `false`. The primary use case for setting this field is to use a StatefulSet's Headless Service to propagate `SRV` records for its Pods without respect to their readiness for purpose of peer discovery.
* `selector` - (Optional) Route service traffic to pods with label keys and values matching this selector. Only applies to types `ClusterIP`, `NodePort`, and `LoadBalancer`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/)
* `session_affinity` - (Optional) Used to maintain session affinity. Supports `ClientIP` and `None`. Defaults to `None`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies)
* `session_affinity_config` - (Optional) Contains the configurations of session affinity. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#proxy-mode-ipvs)
* `type` - (Optional) Determines how the service is exposed. Defaults to `ClusterIP`. Valid options are `ExternalName`, `ClusterIP`, `NodePort`, and `LoadBalancer`. `ExternalName` maps to the specified `external_name`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types)
* `health_check_node_port` - (Optional) Specifies the Healthcheck NodePort for the service. Only effects when type is set to `LoadBalancer` and external_traffic_policy is set to `Local`.

### `port`

#### Arguments

* `app_protocol` - (Optional) The application protocol for this port. This field follows standard Kubernetes label syntax. Un-prefixed names are reserved for IANA standard service names (as per [RFC-6335](https://datatracker.ietf.org/doc/html/rfc6335) and [IANA standard service names](http://www.iana.org/assignments/service-names)). Non-standard protocols should use prefixed names such as `mycompany.com/my-custom-protocol`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#application-protocol)
* `name` - (Optional) The name of this port within the service. All ports within the service must have unique names. Optional if only one ServicePort is defined on this service.
* `node_port` - (Optional) The port on each node on which this service is exposed when `type` is `NodePort` or `LoadBalancer`. Usually assigned by the system. If specified, it will be allocated to the service if unused or else creation of the service will fail. Default is to auto-allocate a port if the `type` of this service requires one. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#type-nodeport)
* `port` - (Required) The port that will be exposed by this service.
* `protocol` - (Optional) The IP protocol for this port. Supports `TCP` and `UDP`. Default is `TCP`.
* `target_port` - (Optional) Number or name of the port to access on the pods targeted by the service. Number must be in the range 1 to 65535. This field is ignored for services with `cluster_ip = "None"`. For more info see [Kubernetes reference](https://kubernetes.io/docs/concepts/services-networking/service/#defining-a-service)

### `session_affinity_config`

#### Arguments

* `client_ip` - (Optional) Contains the configurations of Client IP based session affinity.

### `client_ip`

#### Arguments

* `timeout_seconds` - (Optional) Specifies the seconds of `ClientIP` type session sticky time. The value must be > 0 and <= 86400(for 1 day) if ServiceAffinity == `ClientIP`.

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

### Timeouts

`kubernetes_service_v1` provides the following
[Timeouts](/docs/configuration/resources.html#timeouts) configuration options:

- `create` - Default `10 minutes`

## Import

Service can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_service_v1.example default/terraform-name
```
