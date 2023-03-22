---
subcategory: "manifest"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_resource"
description: |-
  This is a generic data source for Kubernetes API resources
---

# kubernetes_resource

This data source is a generic way to retrieve resources from the Kubernetes API. 

### Example: Get data from a ConfigMap

```hcl
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

## Argument Reference

The following arguments are supported:

* `api_version` - (Required) The API version for the requested resource.
* `kind` - (Required) The kind for the requested resource.
* `metadata` - (Required) The metadata for the requested resource.
* `object` - (Optional) The response returned from the API server.

### `metadata`

#### Arguments

* `name` - (Required) The name of the requested resource.
* `namespace` - (Optional) The namespace of the requested resource.

