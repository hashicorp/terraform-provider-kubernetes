---
subcategory: "manifest"
page_title: "Kubernetes: kubernetes_resource"
description: |-
  This is a generic data source for Kubernetes API resources
---

# <no value>

This data source is a generic way to retrieve resources from the Kubernetes API.

<no value>

### Example: Get data from a ConfigMap

```terraform
data "kubernetes_resource" "example" {
  api_version = "v1"
  kind        = "ConfigMap"

  metadata {
    name      = "example"
    namespace = "default"
  }
}

output "test" {
  value = data.kubernetes_resource.example.object.data.TEST
}
```

