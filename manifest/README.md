# Kubernetes provider for Terraform (alpha) 
<a href="https://terraform.io">
    <img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" alt="Terraform logo" align="right" height="50" />
</a>


![Status: Experimental](https://img.shields.io/badge/status-experimental-EAAA32) [![Releases](https://img.shields.io/github/release/hashicorp/terraform-provider-kubernetes-alpha.svg)](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/releases)
[![LICENSE](https://img.shields.io/github/license/hashicorp/terraform-provider-kubernetes-alpha.svg)](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/blob/master/LICENSE)
![unit tests](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/workflows/unit%20tests/badge.svg)
![acceptance tests](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/workflows/acceptance%20tests/badge.svg)

This Kubernetes provider for Terraform (alpha) supports all API resources in a generic fashion.

This provider allows you to describe any Kubernetes resource using HCL. See [Moving from YAML to HCL](#moving-from-yaml-to-hcl) if you have YAML you want to use with the provider.

Please regard this project as experimental. It still requires extensive testing and polishing to mature into production-ready quality. At this time, we are not planning to create a migration path for resources created with the kubernetes-alpha provider when the `manifest` resource is merged into the official kubernetes provider. For this reason, please do not rely on this provider for production use while we strive towards project maturity. Please [file issues](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/issues/new/choose) generously and detail your experience while using the provider. We welcome your feedback.

Our eventual goal is for this generic resource to become a part of our [official Kubernetes provider](https://github.com/hashicorp/terraform-provider-kubernetes) once it is supported by the Terraform Plugin SDK. However, this work is subject to signficant changes as we iterate towards that level of quality.

## Requirements

* [Terraform](https://www.terraform.io/downloads.html) version 0.14.8 +
* [Kubernetes](https://kubernetes.io/docs/reference) version 1.17.x +
* [Go](https://golang.org/doc/install) version 1.14.x

## Getting Started

If this is your first time here, you can get an overview of the provider by reading our [introductory blog post](https://www.hashicorp.com/blog/deploy-any-resource-with-the-new-kubernetes-provider-for-hashicorp-terraform/).

Otherwise, start by installing the latest release from the [Terraform registry](https://registry.terraform.io/providers/hashicorp/kubernetes-alpha/latest).

Once you have the plugin installed, review the [usage document](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/blob/master/docs/usage.md) in the [docs](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/blob/master/docs/) folder to understand which configuration options are available. You can find the following examples and more in [our examples folder](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/blob/master/examples/). Don't forget to run `terraform init` in your Terraform configuration directory to allow Terraform to detect the provider plugin.

### Create a Kubernetes ConfigMap
```hcl
provider "kubernetes-alpha" {
  config_path = "~/.kube/config" // path to kubeconfig
}

resource "kubernetes_manifest" "test-configmap" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind" = "ConfigMap"
    "metadata" = {
      "name" = "test-config"
      "namespace" = "default"
    }
    "data" = {
      "foo" = "bar"
    }
  }
}
```

### Create a Kubernetes Custom Resource Definition

```hcl
provider "kubernetes-alpha" {
  config_path = "~/.kube/config" // path to kubeconfig
}

resource "kubernetes_manifest" "test-crd" {
  provider = kubernetes-alpha

  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind = "CustomResourceDefinition"
    metadata = {
      name = "testcrds.hashicorp.com"
    }
    spec = {
      group = "hashicorp.com"
      names = {
        kind = "TestCrd"
        plural = "testcrds"
      }
      scope = "Namespaced"
      versions = [{
        name = "v1"
        served = true
        storage = true
        schema = {
          openAPIV3Schema = {
            type = "object"
            properties = {
              data = {
                type = "string"
              }
              refs = {
                type = "number"
              }
            }
          }
        }
      }]
    }
  }
}
```

## Using `wait_for` to block create and update calls

The `kubernetes_manifest` resource supports the ability to block create and update calls until a field is set or has a particular value by specifying the `wait_for` attribute. This is useful for when you create resources like Jobs and Services when you want to wait for something to happen after the resource is created by the API server before Terraform should consider the resource created.

`wait_for` currently supports a `fields` attribute which allows you specify a map of fields paths to regular expressions. You can also specify `*` if you just want to wait for a field to have any value.

```hcl
resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    // ...
  }

  wait_for = {
    fields = {
      # Check the phase of a pod
      "status.phase" = "Running"

      # Check a container's status
      "status.containerStatuses[0].ready" = "true",

      # Check an ingress has an IP
      "status.loadBalancer.ingress[0].ip" = "^(\\d+(\\.|$)){4}"

      # Check the replica count of a Deployment
      "status.readyReplicas" = "2"

      # Check for an annotation
      "metadata.annotations[\"test.annotation\"]" = "*"
    }
  }
}

```

## Moving from YAML to HCL

The `manifest` attribute of the `kubernetes_manifest` resource accepts any arbitrary Kubernetes API object, using Terraform's [map](https://www.terraform.io/docs/configuration/expressions.html#map) syntax. If you have YAML you want to use with this provider, we recommend that you convert it to a map as an initial step and then manage that resource in Terraform, rather than using `yamldecode()` inside the resource block. 

You can quickly convert a single YAML file to an HCL map using this one liner:

```
echo 'yamldecode(file("test.yaml"))' | terraform console
```

Alternatively, there is also an experimental command line tool [tfk8s](https://github.com/jrhouston/tfk8s) you could use to convert Kubernetes YAML manifests into complete Terraform configurations.

## Contributing

We welcome your contribution. Please understand that the experimental nature of this repository means that contributing code may be a bit of a moving target. If you have an idea for an enhancement or bug fix, and want to take on the work yourself, please first [create an issue](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/issues/new/choose) so that we can discuss the implementation with you before you proceed with the work.

You can review our [contribution guide](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/blob/master/_about/CONTRIBUTING.md) to begin. You can also check out our [frequently asked questions](https://github.com/hashicorp/terraform-provider-kubernetes/manifest/blob/master/_about/FAQ.md).

## Experimental Status

By using the software in this repository (the "Software"), you acknowledge that: (1) the Software is still in development, may change, and has not been released as a commercial product by HashiCorp and is not currently supported in any way by HashiCorp; (2) the Software is provided on an "as-is" basis, and may include bugs, errors, or other issues;  (3) the Software is NOT INTENDED FOR PRODUCTION USE, use of the Software may result in unexpected results, loss of data, or other unexpected results, and HashiCorp disclaims any and all liability resulting from use of the Software; and (4) HashiCorp reserves all rights to make all decisions about the features, functionality and commercial release (or non-release) of the Software, at any time and without any obligation or liability whatsoever.
