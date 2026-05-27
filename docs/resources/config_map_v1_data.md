---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_config_map_v1_data"
description: |-
  This resource allows Terraform to manage the data for a ConfigMap that already exists
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_config_map_v1_data" "example" {
  metadata {
    name = "my-config"
  }
  data = {
    "owner" = "myteam"
  }
}
```

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it.
