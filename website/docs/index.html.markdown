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

For specific usage examples, see the guides for [AKS](https://github.com/hashicorp/terraform-provider-kubernetes/blob/master/_examples/aks/README.md), [EKS](https://github.com/hashicorp/terraform-provider-kubernetes/blob/master/_examples/eks/README.md), and [GKE](https://github.com/hashicorp/terraform-provider-kubernetes/blob/master/_examples/gke/README.md).


## Authentication

There are generally two ways to configure the Kubernetes provider.

### File config

The provider always first tries to load **a config file** from a given location
when `config_path` or `config_paths` (or their equivalent environment variables) are set. 
Depending on whether you have a current context set this _may_ require 
`config_context_auth_info` and/or `config_context_cluster` and/or `config_context`.

#### Setting default config context

Here's an example of how to set default context and avoid all provider configuration:

```
kubectl config set-context default-system \
  --cluster=chosen-cluster \
  --user=chosen-user

kubectl config use-context default-system
```

Read [more about `kubectl` in the official docs](https://kubernetes.io/docs/user-guide/kubectl-overview/).

### In-cluster service account token

If no other configuration is specified, and when it detects it is running in a kubernetes pod,
the provider will try to use the service account token from the `/var/run/secrets/kubernetes.io/serviceaccount/token` path.
Detection of in-cluster execution is based on the sole availability of both of the `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` environment variables,
with non-empty values.

If you have any other static configuration setting specified in a config file or static configuration, in-cluster service account token will not be tried.

### Statically defined credentials

Another way is **statically** define TLS certificate credentials:

```hcl
provider "kubernetes" {
  host = "https://104.196.242.174"

  client_certificate     = "${file("~/.kube/client-cert.pem")}"
  client_key             = "${file("~/.kube/client-key.pem")}"
  cluster_ca_certificate = "${file("~/.kube/cluster-ca-cert.pem")}"
}
```

or username and password (HTTP Basic Authorization):

```hcl
provider "kubernetes" {
  host = "https://104.196.242.174"

  username = "username"
  password = "password"
}
```



~> If you have **both** valid configurations in a config file and static configuration, the static one is used as an override.
i.e. any static field will override its counterpart loaded from the config.

## Exec-based credential plugins

Some cloud providers have short-lived authentication tokens that can expire relatively quickly. To ensure the Kubernetes provider is receiving valid credentials, an exec-based plugin can be used to fetch a new token before initializing the provider. For example, on EKS, the command `eks get-token` can be used:

```hcl
provider "kubernetes" {
  host                   = var.cluster_endpoint
  cluster_ca_certificate = base64decode(var.cluster_ca_cert)
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
    command     = "aws"
  }
}
```

For further reading, see these examples which demonstrate different approaches to keeping the cluster credentials up to date: [AKS](https://github.com/hashicorp/terraform-provider-kubernetes/blob/master/_examples/aks/README.md), [EKS](https://github.com/hashicorp/terraform-provider-kubernetes/blob/master/_examples/eks/README.md), and [GKE](https://github.com/hashicorp/terraform-provider-kubernetes/blob/master/_examples/gke/README.md).


## Argument Reference

The following arguments are supported:

* `host` - (Optional) The hostname (in form of URI) of Kubernetes master. Can be sourced from `KUBE_HOST`.
* `username` - (Optional) The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_USER`.
* `password` - (Optional) The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint. Can be sourced from `KUBE_PASSWORD`.
* `insecure` - (Optional) Whether the server should be accessed without verifying the TLS certificate. Can be sourced from `KUBE_INSECURE`. Defaults to `false`.
* `client_certificate` - (Optional) PEM-encoded client certificate for TLS authentication. Can be sourced from `KUBE_CLIENT_CERT_DATA`.
* `client_key` - (Optional) PEM-encoded client certificate key for TLS authentication. Can be sourced from `KUBE_CLIENT_KEY_DATA`.
* `cluster_ca_certificate` - (Optional) PEM-encoded root certificates bundle for TLS authentication. Can be sourced from `KUBE_CLUSTER_CA_CERT_DATA`.
* `config_path` - (Optional) A path to a kube config file. Can be sourced from `KUBE_CONFIG_PATH`.
* `config_paths` - (Optional) A list of paths to the kube config files. Can be sourced from `KUBE_CONFIG_PATHS`.
* `config_context` - (Optional) Context to choose from the config file. Can be sourced from `KUBE_CTX`.
* `config_context_auth_info` - (Optional) Authentication info context of the kube config (name of the kubeconfig user, `--user` flag in `kubectl`). Can be sourced from `KUBE_CTX_AUTH_INFO`.
* `config_context_cluster` - (Optional) Cluster context of the kube config (name of the kubeconfig cluster, `--cluster` flag in `kubectl`). Can be sourced from `KUBE_CTX_CLUSTER`.
* `token` - (Optional) Token of your service account.  Can be sourced from `KUBE_TOKEN`.
* `exec` - (Optional) Configuration block to use an [exec-based credential plugin] (https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins), e.g. call an external command to receive user credentials.
    * `api_version` - (Required) API version to use when decoding the ExecCredentials resource, e.g. `client.authentication.k8s.io/v1beta1`.
    * `command` - (Required) Command to execute.
    * `args` - (Optional) List of arguments to pass when executing the plugin.
    * `env` - (Optional) Map of environment variables to set when executing the plugin.
