---
subcategory: "manifest"
page_title: "Kubernetes: kubernetes_resources"
description: |-
  This data source is a generic way to query for a list of resources from the Kubernetes API and filter them. 
---

# <no value>

This data source is a generic way to query for a list of Kubernetes resources and filter them using a label or field selector.

<no value> 

### Example: Get a list of namespaces excluding "kube-system" using `field_selector`

```terraform
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

```terraform
data "kubernetes_resources" "example" {
  api_version    = "v1"
  kind           = "Namespace"
  label_selector = "kubernetes.io/metadata.name!=kube-system"
}

output "test" {
  value = length(data.kubernetes_resources.example.objects)
}
```

