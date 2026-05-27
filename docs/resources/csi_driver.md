---
subcategory: "storage/v1beta1"
page_title: "Kubernetes: kubernetes_csi_driver"
description: |-
  The Container Storage Interface (CSI) is a standard for exposing arbitrary block and file storage systems to containerized workloads on Container Orchestration Systems (COs) like Kubernetes.
---

# <no value>

The [Container Storage Interface](https://kubernetes-csi.github.io/docs/introduction.html) (CSI) is a standard for exposing arbitrary block and file storage systems to containerized workloads on Container Orchestration Systems (COs) like Kubernetes.

<no value>

## Example Usage

```terraform
resource "kubernetes_csi_driver" "example" {
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

kubernetes_csi_driver can be imported using its name, e.g.

```
$ terraform import kubernetes_csi_driver.example terraform-example
```
