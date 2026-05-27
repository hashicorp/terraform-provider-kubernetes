---
subcategory: "core/v1"
page_title: "Kubernetes: kubernetes_persistent_volume_claim"
description: |-
  A PersistentVolumeClaim (PVC) is a request for storage by a user. This data source retrieves information about the specified PVC.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
data "kubernetes_persistent_volume_claim" "example" {
  metadata {
    name = "terraform-example"
  }
}
```

## Import

Persistent Volume Claim can be imported using its namespace and name, e.g.

```
$ terraform import kubernetes_persistent_volume_claim.example default/example-name
```
