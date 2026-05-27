---
subcategory: "manifest"
page_title: "Kubernetes: kubernetes_annotations"
description: |-
  This resource allows Terraform to manage the annotations for a resource that already exists
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
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

```terraform
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

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it.
