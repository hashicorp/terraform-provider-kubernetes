---
subcategory: "manifest"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_manifest"
description: |-
  The resource provides a way to create and manage custom resources 
---

# kubernetes_manifest

Represents one Kubernetes resource by supplying a `manifest` attribute. The manifest value is the HCL representation of a Kubernetes YAML manifest. To convert an existing manifest from YAML to HCL, you can use the Terraform built-in function [`yamldecode()`](https://www.terraform.io/docs/configuration/functions/yamldecode.html) or [tfk8s](https://github.com/jrhouston/tfk8s).

Once applied, the `object` attribute contains the state of the resource as returned by the Kubernetes API, including all default values.

~> A minimum Terraform version of 0.14.8 is required to use this resource.

### Before you use this resource

- This resource requires API access during planning time. This means the cluster has to be accessible at plan time and thus cannot be created in the same apply operation. We recommend only using this resource for custom resources or resources not yet fully supported by the provider.

- This resource uses [Server-side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) to carry out apply operations. A minimum Kubernetes version of 1.16.x is required, but versions 1.17+ are strongly recommended as the SSA implementation in Kubernetes 1.16.x is incomplete and unstable.


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

## Importing existing Kubernetes resources as `kubernetes_manifest`

Objects already present in a Kubernetes cluster can be imported into Terraform to be managed as `kubernetes_manifest` resources. Follow these steps to import a resource:

### Extract the resource from Kubernetes and transform it into Terraform configuration

```
kubectl get secrets sample -o yaml | tfk8s --strip -o sample.tf
```

### Import the resource state from the cluster

```
terraform import kubernetes_manifest.secret_sample "apiVersion=v1,kind=Secret,namespace=default,name=sample"
```

Note the import ID as the last argument to the import command. This ID points Terraform at which Kubernetes object to read when importing.
It should be constructed with the following syntax: `"apiVersion=<string>,kind=<string>,[namespace=<string>,]name=<string>"`. The `namespace=<string>` in the ID string is required only for Kubernetes namespaced objects and should be omitted for cluster-wide objects.

## Using `wait` to block create and update calls

The `kubernetes_manifest` resource supports the ability to block create and update calls until a field is set or has a particular value by specifying the `wait` block. This is useful for when you create resources like Jobs and Services when you want to wait for something to happen after the resource is created by the API server before Terraform should consider the resource created.

`wait` supports supports a `fields` attribute which allows you specify a map of fields paths to regular expressions. You can also specify `*` if you just want to wait for a field to have any value.

```hcl
resource "kubernetes_manifest" "test" {
  manifest = {
    // ...
  }

  wait {
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

  timeouts {
    create = "10m"
    update = "10m"
    delete = "30s"
  }
}
```

The `wait` block also supports a `rollout` attribute which will wait for rollout to complete on Deployment, StatefulSet, and DaemonSet resources. 

```hcl
resource "kubernetes_manifest" "test" {
  manifest = {
    // ...
  }

  wait {
    rollout = true
  }
}
```

You can also wait for specified conditions to be met by specifying a `condition` block.

```hcl
resource "kubernetes_manifest" "test" {
  manifest = {
    // ...
  }

  wait {
    condition {
      type   = "ContainersReady"
      status = "True"
    }
  }
}
```

## Configuring `field_manager`

The `kubernetes_manifest` exposes configuration of the field manager through the optional `field_manager` block.

```hcl
resource "kubernetes_manifest" "test" {
  provider = kubernetes-alpha

  manifest = {
    // ...
  }

  field_manager {
    # set the name of the field manager
    name = "myteam"

    # force field manager conflicts to be overridden
    force_conflicts = true
  }
}
```

## Computed fields

When setting the value of an field in configuration, Terraform will check that the same value is returned after the apply operation. This ensures that the actual configuration requested by the user is successfully applied. In some cases, with the Kubernetes API this is not the desired behavior. Particularly when using mutating admission controllers, there is a chance that the values configured by the user will be modified by the API. This usually manifest as `Error: Provider produced inconsistent result after apply` and `produced an unexpected new value:` messages when applying.

To accommodate this, the `kubernetes_manifest` resources allows defining so-called "computed" fields. When an field is defined as "computed" Terraform will allow the final value stored in state after `apply` as returned by the API to be different than what the user requested. 

The most common example of this is  `metadata.annotations`. In some cases, the API will add extra annotations on top of the ones configured by the user. Unless the field is declared as "computed" Terraform will throw an error signaling that the state returned by the 'apply' operation is inconsistent with the value defined in the 'plan'.

 To declare an field as "computed" add its full field path to the `computed_fields` field under the respective `kubernetes_manifest` resource. For example, to declare the "metadata.labels" field as "computed", add the following:

```
resource "kubernetes_manifest" "test-ns" {
  manifest = {
    ...
  }

  computed_fields = ["metadata.labels"]
 }
```

**IMPORTANT**: By default, `metadata.labels` and `metadata.annotations` are already included in the list. You don't have to set them explicitly in the `computed_fields` list. To turn off these defaults, set the value of `computed_fields` to an empty list or a concrete list of other fields. For example `computed_fields = []`.

The syntax for the field paths is the same as the one used in the `wait` block.

## Argument Reference

The following arguments are supported:

- `computed_fields` - (Optional) List of paths of fields to be handled as "computed". The user-configured value for the field will be overridden by any different value returned by the API after apply.
- `manifest` (Required) An object Kubernetes manifest describing the desired state of the resource in HCL format.
- `object` (Optional) The resulting resource state, as returned by the API server after applying the desired state from `manifest`.
- `wait` (Optional) An object which allows you configure the provider to wait for specific fields to reach a desired value or certain conditions to be met. See below for schema.
- `wait_for` (Optional, Deprecated) An object which allows you configure the provider to wait for certain conditions to be met. See below for schema. **DEPRECATED: use `wait` block**.
- `field_manager` (Optional) Configure field manager options. See below.

### `wait`

#### Arguments

- `rollout` (Optional) When set to `true` will wait for the resource to roll out, equivalent to `kubectl rollout status`. 
- `condition` (Optional) A set of condition to wait for. You can specify multiple `condition` blocks and it will wait for all of them. 
- `fields` (Optional) A map of field paths and a corresponding regular expression with a pattern to wait for. The provider will wait until the field's value matches the regular expression. Use `*` for any value.

A field path is a string that describes the fully qualified address of a field within the resource, including its parent fields all the way up to "object". The syntax of a path string follows the rules below:

- Fields of objects are addressed with `.`
- Keys of a map field are addressed with `["<key-string>"]`
- Elements of a list or tuple field are addressed with `[<index-numeral>]`

  The following example waits for Kubernetes to create a ServiceAccount token in a Secret, where the `data` field of the Secret is a map.

  ```hcl
  wait {
    fields = {
      "data[\"token\"]" = ".+"
    }
  }
  ```

  You can use the [`type()`](https://developer.hashicorp.com/terraform/language/functions/type) Terraform function to determine the type of a field. With the resource created and present in state, run `terraform console` and then the following command:

  ```hcl
  > type(kubernetes_manifest.my-secret.object.data)
    map(string)
  ```

### `wait_for` (deprecated, use `wait`)

#### Arguments

- `fields` (Optional) A map of fields and a corresponding regular expression with a pattern to wait for. The provider will wait until the field matches the regular expression. Use `*` for any value. 


### `field_manager`

#### Arguments

- `name` (Optional) The name of the field manager to use when applying the resource. Defaults to `Terraform`.
- `force_conflicts` (Optional) Forcibly override any field manager conflicts when applying the resource. Defaults to `false`.

### `timeouts`

See [Operation Timeouts](https://www.terraform.io/docs/language/resources/syntax.html#operation-timeouts)
