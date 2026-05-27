---
subcategory: "storage/v1"
page_title: "Kubernetes: kubernetes_csi_driver_v1"
description: |-
  The Container Storage Interface (CSI) is a standard for exposing arbitrary block and file storage systems to containerized workloads on Container Orchestration Systems (COs) like Kubernetes.
---

# <no value>

<no value>

<no value>

## Example Usage

```terraform
resource "kubernetes_csi_driver_v1" "example" {
  metadata {
    name = "terraform-example"
  }

  spec {
    attach_required        = true
    pod_info_on_mount      = true
    volume_lifecycle_modes = ["Ephemeral"]
  }
}
```

## Import

kubernetes_csi_driver_v1 can be imported using its name, e.g.

```
$ terraform import kubernetes_csi_driver_v1.example terraform-example
```
