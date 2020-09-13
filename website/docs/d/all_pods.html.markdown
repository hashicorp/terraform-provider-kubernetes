---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_all_pods"
description: |-
  Lists all pods within a namespace.
---

# kubernetes_all_pods

This data source provides a mechanism for listing the names of all available pods in a Kubernetes namespace.
It can be used to check for existence of a specific pod or to apply another resource to all or a subset of existing pods in a namespace.

## Example Usage

```hcl
data "kubernetes_all_pods" "allpods" {
  namespace = "default"
}

output "all-pods" {
  value = data.kubernetes_all_pods.allpods.pods
}

output "pod-present" {
  value = contains(data.kubernetes_all_pods.allpods.pods, "my-pod")
}

```

## Argument Reference

The following arguments are supported:

* `namespace` - (Optional) Namespace to list. Defaults to "default" 

