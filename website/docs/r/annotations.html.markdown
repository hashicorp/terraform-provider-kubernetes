---
subcategory: "manifest"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_annotations"
description: |-
  This resource allows Terraform to manage the annotations for a resource that already exists
---

# kubernetes_annotations

This resource allows Terraform to manage the annotations for a resource that already exists. This resource uses [field management](https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management) and [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) to manage only the annotations that are defined in the Terraform configuration. Existing annotations not specified in the configuration will be ignored. If an annotation specified in the config and is already managed by another client it will cause a conflict which can be overridden by setting `force` to true. 


## Example Usage

```hcl
resource "kubernetes_annotations" "example" {
  api_version = "v1"
  kind        = "ConfigMap"
  metadata {
    name = "my-config"
  }
  annotations = {
    "owner" = "myteam"
  }
}
```

## Example Usage: Patching resources which contain a pod template, e.g Deployment, Job

```hcl
resource "kubernetes_annotations" "example" {
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name = "my-config"
  }
  # These annotations will be applied to the Deployment resource itself
  annotations = {
    "owner" = "myteam"
  }
  # These annotations will be applied to the Pods created by the Deployment
  template_annotations = {
    "owner" = "myteam"
  }
}
```

## Argument Reference

The following arguments are supported: 

~> NOTE: At least one of `annotations` or `template_annotations` is required. 

* `api_version` - (Required) The apiVersion of the resource to be annotated.
* `kind` - (Required) The kind of the resource to be annotated.
* `metadata` - (Required) Standard metadata of the resource to be annotated. 
* `annotations` - (Optional) A map of annotations to apply to the resource.
* `template_annotations` - (Optional) A map of annotations to apply to the pod template within the resource.
* `force` - (Optional) Force management of annotations if there is a conflict. Defaults to `false`.
* `field_manager` - (Optional) The name of the [field manager](https://kubernetes.io/docs/reference/using-api/server-side-apply/#field-management). Defaults to `Terraform`.

## Nested Blocks

### `metadata`

#### Arguments

* `name` - (Required) Name of the resource to be annotated.
* `namespace` - (Optional) Namespace of the resource to be annotated.

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it. 


