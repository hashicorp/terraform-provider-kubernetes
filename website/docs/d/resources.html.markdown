---
subcategory: "manifest"
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_resources"
description: |-
  This data source is a generic way to query for a list of resources from the Kubernetes API and filter them. 
---

# kubernetes_resources

This data source is a generic way to query for a list of Kubernetes resources and filter them using a label or field selector.

### Example: Get a list of namespaces excluding "kube-system" using `field_selector`

```hcl
data "kubernetes_resources" "example" {
  api_version    = "v1"
  kind           = "Namespace"
  field_selector = "metadata.name!=kube-system"
}

output "test" {
  value = length(data.kubernetes_resources.example.objects)
}
```

### Example: Get a list of namespaces excluding "kube-system" using `label_selector`

```hcl
data "kubernetes_resources" "example" {
  api_version    = "v1"
  kind           = "Namespace"
  label_selector = "kubernetes.io/metadata.name!=kube-system"
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
* `object` - (Optional) The response returned from the API server.

