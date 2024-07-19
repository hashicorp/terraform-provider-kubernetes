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
