---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_manifest"
description: |-
  The resource provides a way to create and manage custom resources 
---

# kubernetes_manifest

Represents one Kubernetes resource by supplying a `manifest` attribute. The manifest value is the HCL representation of a Kubernetes YAML manifest. To convert an existing manifest from YAML to HCL, you can use the Terrafrom built-in function [`yamldecode()`](https://www.terraform.io/docs/configuration/functions/yamldecode.html) or [tfk8s](https://github.com/jrhouston/tfk8s).

Once applied, the `object` attribute contains the state of the resource as returned by the Kubernetes API, including all default values.

~> **NOTE:** This resource uses [Server-side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) to carry out plan and apply operations. This means the cluster has to be accessible at plan time. We recommend only using this resource for custom resources or resources not yet fully supported by the provider. 


### Example: Create a Kubernetes ConfigMap

```hcl
resource "kubernetes_manifest" "test-configmap" {
  manifest = {
    "apiVersion" = "v1"
    "kind"       = "ConfigMap"
    "metadata" = {
      "name"      = "test-config"
      "namespace" = "default"
    }
    "data" = {
      "foo" = "bar"
    }
  }
}
```

### Example: Create a Kubernetes Custom Resource Definition

```hcl
resource "kubernetes_manifest" "test-crd" {
  manifest = {
    apiVersion = "apiextensions.k8s.io/v1"
    kind       = "CustomResourceDefinition"

    metadata = {
      name = "testcrds.hashicorp.com"
    }

    spec = {
      group = "hashicorp.com"

      names = {
        kind   = "TestCrd"
        plural = "testcrds"
      }

      scope = "Namespaced"

      versions = [{
        name    = "v1"
        served  = true
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
    }
  }
}

```

## Argument Reference

The following arguments are supported:

- `manifest` (Required) An object Kubernetes manifest describing the desired state of the resource in HCL format.
- `object` (Optional) The resulting resource state, as returned by the API server after applying the desired state from `manifest`.
- `wait_for` (Optional) An object which allows you configure the provider to wait for certain conditions to be met. See below for schema. 

### `wait_for`

#### Arguments

- **fields** (Required) A map of fields and a corresponding regular expression with a pattern to wait for. The provider will wait until the field matches the regular expression. Use `*` for any value. 