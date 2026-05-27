---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_env"
description: |-
  This resource provides a way to manage environment variables in resources that were created outside of Terraform.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_env" "example" {
  container = "nginx"
  metadata {
    name = "nginx-deployment"
  }

  api_version = "apps/v1"
  kind        = "Deployment"

  env {
    name  = "NGINX_HOST"
    value = "google.com"
  }

  env {
    name  = "NGINX_PORT"
    value = "90"
  }
}
```

## Import

This resource does not support the `import` command. As this resource operates on Kubernetes resources that already exist, creating the resource is equivalent to importing it.
