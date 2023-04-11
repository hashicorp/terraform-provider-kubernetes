---
subcategory: "manifest"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_labels"
description: |-
  This resource allows Terraform to manage the labels for a resource that already exists
---

# kubernetes_labels

This resource allows Terraform to manage the labels for a resource that already exists. This resource uses [field management](https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management) and [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) to manage only the labels that are defined in the Terraform configuration. Existing labels not specified in the configuration will be ignored. If a label specified in the config and is already managed by another client it will cause a conflict which can be overridden by setting `force` to true. 


## Example Usage

```hcl
resource "kubernetes_labels" "example" {
  api_version = "v1"
  kind        = "ConfigMap"
  metadata {
    name = "my-config"
  }
  labels = {
    "owner" = "myteam"
  }
}
```

## Argument Reference

The following arguments are supported:

* `api_version` - (Required) The apiVersion of the resource to be labelled.
* `kind` - (Required) The kind of the resource to be labelled.
* `metadata` - (Required) Standard metadata of the resource to be labelled. 
* `labels` - (Required) A map of labels to apply to the resource.
* `force` - (Optional) Force management of labels if there is a conflict.
* `field_manager` - (Optional) The name of the [field manager](https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management). Defaults to `Terraform`.

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the resource to be labelled.
* `namespace` - (Optional) Namespace of the resource to be labelled.

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it. 


