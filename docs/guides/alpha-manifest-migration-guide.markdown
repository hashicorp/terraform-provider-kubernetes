---
layout: "kubernetes"
page_title: "Migrating `kubernetes_manifest` resources from the kubernetes-alpha provider"
description: |-
  This guide covers adopting `kubernetes_manifest` resources created using the kubernetes-alpha provider.
---

# The kubernetes_manifest resource

Earlier this year we announced a new provider capable of managing any kind of Kubernetes resource, but more specifically Custom Resources via a manifest configuration that could be translated directly from Kubernetes YAML. This was released as the experimental [kubernetes-alpha](https://github.com/hashicorp/terraform-provider-kubernetes-alpha) provider.

The `kubernetes_manifest` resource in now available in the official provider for Kubernetes. This guide walks through the actions needed to adopt existing `kubernetes_manifest` resources into configurations that use the Kubernetes provider.

Follow these steps to migrate your configuration and continue using the `kubernetes_manifest` resource with the Kubernetes provider.

## Step 1: Provider configuration blocks

The provider configuration blocks for the `kubernetes-alpha` provider are no longer supported. To carry over the configuration, simply rename the provider block to "kubernetes".

For example:

```
provider "kubernetes-alpha" {
    config_path = "/my/kube/config"
}
```

becomes

```
provider "kubernetes" {
    config_path = "/my/kube/config"
    experiments {
        manifest_resource = true
    }
}
```

## Step 2: Provider references on resources

The provider references to `kubernetes-alpha` are no longer required. Simply remove the `provider = kubernetes-alpha` text from all `kubernetes_manifest` resources in your configuration.

For example:

```
resource "kubernetes_manifest" "my-resource" {
  provider = kubernetes-alpha
  manifest = {....}
}
```

becomes

```
resource "kubernetes_manifest" "my-resource" {
  manifest = {....}
}
```

## Step 3: Provider version constraints

If your configuration includes a `terraform` block which specifies required provider versions, you should remove any references to provider `kubernetes-alpha` from that block. At the same time, you should add a requirement for provider `kubernetes` version 2.4.0 and above.

For example:

```
terraform {
  required_providers {
    kubernetes-alpha = {
      source  = "hashicorp/kubernetes-alpha"
      version = "0.5.0"
    }
    ...
  }
}
```

becomes:

```
terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.4"
    }
    ...
  }
}
```

If you made any changes to existing entries in the `required_providers` section, make sure to run `terraform init -upgrade` to let Terraform retrieve any required new provider versions.

## Step 4: Replace providers in existing state

If your configuration was already in use with the `kubernetes_alpha` provider, you likely also have Terraform state generated from it.
It is recommended to start fresh and re-apply configurations using the kubernetes provider from a clean slate.
However, in case you find it necessary to preserve state, you can rename the provider associated with any `kubernetes_manifest` resources using the dedicated `replace-provider` command in Terraform.

Run the following command in the directory where the `terraform.tfstate` file is:

```
terraform state replace-provider hashicorp/kubernetes-alpha hashicorp/kubernetes
```

## Mixing 'kubernetes_manifest' with other 'kubernetes_*' resources

In case you plan on adding `kubernetes_manifest` resources to your existing configuration which contains other resources of the Kubernetes provider there are some important aspects to be aware of.

If your present configuration for the Kubernetes provider also creates the Kubernetes cluster using Terraform resources in the same `apply` operation (against best-practice recommendations), this will no longer work when adding `kubernetes_manifest` resources. The reason behind this is that `kubernetes_manifest` require access to the API during planning, at which point the cluster resource would not have yet been created.

As a solution, choose one of the following options:

* separate the cluster creation in a different `apply` operation.
* add a new `apply` operation only for the `kubernetes_manifest` resources.

