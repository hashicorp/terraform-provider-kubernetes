---
subcategory: "manifest"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_resources"
description: |-
  This data source is a generic way to filter resources from the Kubernetes API. 
---

# kubernetes_resource

This data source is a generic way to filter resources from the Kubernetes API. 

### Example: Get data from a ConfigMap

```hcl
data "kubernetes_resources" "example"{
    kind = "Namespace"
    api_version = "v1"
    namespace = "test"
    label_selector = "kubernetes.io/metadata.name!=kube-system"
    limit = "2"
}

output "test" {
  value = length(data.kubernetes_resources.example.objects)
}
```

## Argument Reference

The following arguments are supported:

* `api_version` - (Required) The API version for the requested resource.
* `kind` - (Required) The kind for the requested resource.
* `label_selector` - (Optional) A selector to restrict the list of returned objects by their labels.
* `field_selector` - (Optional) A selector to restrict the list of returned objects by their fields.
* `namespace` - (Optional) The namespace of the requested resource.

