---
layout: "kubernetes"
page_title: "Provider: Kubernetes"
description: |-
  The Kubernetes (K8s) provider is used to interact with the resources supported by Kubernetes. The provider needs to be configured with the proper credentials before it can be used.
---

# Kubernetes Provider

The Kubernetes (K8S) provider is used to interact with the resources supported by Kubernetes. The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "my-context"
}

resource "kubernetes_namespace" "example" {
  metadata {
    name = "my-first-namespace"
  }
}
```

## Kubernetes versions

Both backward and forward compatibility with Kubernetes API is mostly defined
by the [official K8S Go library](https://github.com/kubernetes/kubernetes) (prior to `1.1` release)
and [client Go library](https://github.com/kubernetes/client-go) which we ship with Terraform.
Below are versions of the library bundled with given versions of Terraform.

* Terraform `<= 0.9.6` (prior to provider split) - Kubernetes `1.5.4`
* Terraform `0.9.7` (prior to provider split) `< 1.1` (provider version) - Kubernetes `1.6.1`
* `1.1+` - Kubernetes `1.7`

## Stacking with managed Kubernetes cluster resources

Terraform providers for various cloud providers feature resources to spin up managed Kubernetes clusters on services such as EKS, AKS and GKE. Such resources (or data-sources) will have attributes that expose the credentials needed for the Kubernetes provider to connect to these clusters.

To use these credentials with the Kubernetes provider, they can be interpolated into the respective attributes of the Kubernetes provider configuration block.

~> **WARNING** When using interpolation to pass credentials to the Kubernetes provider from other resources, these resources SHOULD NOT be created in the same Terraform module where Kubernetes provider resources are also used. This will lead to intermittent and unpredictable errors which are hard to debug and diagnose. The root issue lies with the order in which Terraform itself evaluates the provider blocks vs. actual resources. Please refer to [this section of Terraform docs](https://www.terraform.io/docs/configuration/providers.html#provider-configuration) for further explanation.

The most reliable way to configure the Kubernetes provider is to ensure that the cluster itself and the Kubernetes provider resources can be managed with separate `apply` operations. Data-sources can be used to convey values between the two stages as needed.

For specific usage examples, see the guides for [AKS](https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/_examples/aks/README.md), [EKS](https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/_examples/eks/README.md), and [GKE](https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/_examples/gke/README.md).


## Authentication

~> NOTE: The provider does not use the `KUBECONFIG` environment variable by default. See the attribute reference below for the environment variables that map to provider block attributes.

The Kubernetes provider can get its configuration in two ways:

1. _Explicitly_ by supplying attributes to the provider block. This includes:
   * [Using a kubeconfig file](#file-config)
   * [Supplying credentials](#credentials-config)
   * [Exec plugins](#exec-plugins)
2. _Implicitly_ through environment variables. This includes:
   * [Using the in-cluster config](#in-cluster-config)

The provider always first tries to load **a config file** from a given location
when `config_path` or `config_paths` (or their equivalent environment variables) are set. 
Depending on whether you have a current context set this _may_ require 
`config_context_auth_info` and/or `config_context_cluster` and/or `config_context`.

For a full list of supported provider authentication arguments and their corresponding environment variables, see the [argument reference](#argument-reference) below.


### File config

The easiest way is to supply a path to your kubeconfig file using the `config_path` attribute or using the `KUBE_CONFIG_PATH` environment variable. A kubeconfig file may have multiple contexts. If `config_context` is not specified, the provider will use the `default` context.

```hcl
provider "kubernetes" {
  config_path = "~/.kube/config"
}
```

The provider also supports multiple paths in the same way that kubectl does using the `config_paths` attribute or `KUBE_CONFIG_PATHS` environment variable.

```hcl
provider "kubernetes" {
  config_paths = [
    "/path/to/config_a.yaml",
    "/path/to/config_b.yaml"
  ]
}
```

### Credentials config

You can also configure the host, basic auth credentials, and client certificate authentication explicitly or through environment variables.

```hcl
provider "kubernetes" {
  host = "https://cluster_endpoint:port"

  client_certificate     = file("~/.kube/client-cert.pem")
  client_key             = file("~/.kube/client-key.pem")
  cluster_ca_certificate = file("~/.kube/cluster-ca-cert.pem")
}
```

### In-cluster Config

The provider uses the `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` environment variables to detect when it is running inside a cluster, so in this case you do not need to specify any attributes in the provider block if you want to connect to the local kubernetes cluster.

If you want to connect to a different cluster than the one terraform is running inside, configure the provider as [above](#credentials-config).

Find more comprehensive `in-cluster` config example [here](https://github.com/hashicorp/terraform-provider-kubernetes/tree/main/_examples/in-cluster).

## Exec plugins

Some cloud providers have short-lived authentication tokens that can expire relatively quickly. To ensure the Kubernetes provider is receiving valid credentials, an exec-based plugin can be used to fetch a new token before initializing the provider. For example, on EKS, the command `eks get-token` can be used:

```hcl
provider "kubernetes" {
  host                   = var.cluster_endpoint
  cluster_ca_certificate = base64decode(var.cluster_ca_cert)
  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
    command     = "aws"
  }
}
```

## Examples 

For further reading, see these examples which demonstrate different approaches to keeping the cluster credentials up to date: [AKS](https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/_examples/aks/README.md), [EKS](https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/_examples/eks/README.md), and [GKE](https://github.com/hashicorp/terraform-provider-kubernetes/blob/main/_examples/gke/README.md).

## Ignore Kubernetes annotations and labels

In certain cases, external systems can add and modify resources annotations and labels for their own purposes. However, Terraform will remove them since they are not presented in the code. It also might be hard to update code accordingly to stay tuned with the changes that come outside. In order to address this `ignore_annotations` and `ignore_labels` attributes were introduced on the provider level. They allow Terraform to ignore certain annotations and labels across all resources. Please bear in mind, that all data sources remain unaffected and the provider always returns all labels and annotations, in spite of the `ignore_annotations` and `ignore_labels` settings. The same is applicable for the pod and job definitions that fall under templates.

Both attributes support RegExp to match metadata objects more effectively.

### Examples

The following example demonstrates how to ignore particular annotation keys:

```hcl
provider "kubernetes" {
  ignore_annotations = [
    "cni\\.projectcalico\\.org\\/podIP",
    "cni\\.projectcalico\\.org\\/podIPs",
  ]
}
```

Next example demonstrates how to ignore AWS load balancer annotations:

```hcl
provider "kubernetes" {
  ignore_annotations = [
    "^service\\.beta\\.kubernetes\\.io\\/aws-load-balancer.*",
  ]
}
```

Since dot `.`, forward slash `/`, and some other symbols have special meaning in RegExp, they should be escaped by adding a double backslash in front of them if you want to use them as they are.

## Argument Reference

The following arguments are supported:

* `host` - (Optional) The hostname (in form of URI) of the Kubernetes API. Can be sourced from `KUBE_HOST`.
* `username` - (Optional) The username to use for HTTP basic authentication when accessing the Kubernetes API. Can be sourced from `KUBE_USER`.
* `password` - (Optional) The password to use for HTTP basic authentication when accessing the Kubernetes API. Can be sourced from `KUBE_PASSWORD`.
* `insecure` - (Optional) Whether the server should be accessed without verifying the TLS certificate. Can be sourced from `KUBE_INSECURE`. Defaults to `false`.
* `tls_server_name` - (Optional) Server name passed to the server for SNI and is used in the client to check server certificates against. Can be sourced from `KUBE_TLS_SERVER_NAME`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. Can be sourced from `KUBE_CLIENT_CERT_DATA`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. Can be sourced from `KUBE_CLIENT_KEY_DATA`.
* `cluster_ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. Can be sourced from `KUBE_CLUSTER_CA_CERT_DATA`.
* `config_path` - (Optional) A path to a kube config file. Can be sourced from `KUBE_CONFIG_PATH`.
* `config_paths` - (Optional) A list of paths to the kube config files. Can be sourced from `KUBE_CONFIG_PATHS`.
* `config_context` - (Optional) Context to choose from the config file. Can be sourced from `KUBE_CTX`.
* `config_context_auth_info` - (Optional) Authentication info context of the kube config (name of the kubeconfig user, `--user` flag in `kubectl`). Can be sourced from `KUBE_CTX_AUTH_INFO`.
* `config_context_cluster` - (Optional) Cluster context of the kube config (name of the kubeconfig cluster, `--cluster` flag in `kubectl`). Can be sourced from `KUBE_CTX_CLUSTER`.
* `token` - (Optional) Token of your service account.  Can be sourced from `KUBE_TOKEN`.
* `proxy_url` - (Optional) URL to the proxy to be used for all API requests. URLs with "http", "https", and "socks5" schemes are supported. Can be sourced from `KUBE_PROXY_URL`.
* `exec` - (Optional) Configuration block to use an [exec-based credential plugin] (https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins), e.g. call an external command to receive user credentials.
    * `api_version` - (Required) API version to use when decoding the ExecCredentials resource, e.g. `client.authentication.k8s.io/v1beta1`.
    * `command` - (Required) Command to execute.
    * `args` - (Optional) List of arguments to pass when executing the plugin.
    * `env` - (Optional) Map of environment variables to set when executing the plugin.
* `ignore_annotations` - (Optional) List of Kubernetes metadata annotations to ignore across all resources handled by this provider for situations where external systems are managing certain resource annotations. Each item is a regular expression.
* `ignore_labels` - (Optional) List of Kubernetes metadata labels to ignore across all resources handled by this provider for situations where external systems are managing certain resource labels. Each item is a regular expression.
