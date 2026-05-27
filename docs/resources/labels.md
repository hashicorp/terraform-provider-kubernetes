---
subcategory: "manifest"
page_title: "Kubernetes: kubernetes_labels"
description: |-
  This resource allows Terraform to manage the labels for a resource that already exists
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
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

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it.
