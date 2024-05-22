resource "kubernetes_persistent_volume_claim" "example" {
  metadata {
    name = "exampleclaimname"
  }
  spec {
    access_modes = ["ReadWriteMany"]
    resources {
      requests = {
        storage = "5Gi"
      }
    }
    volume_name = "${kubernetes_persistent_volume.example.metadata.0.name}"
  }
}

resource "kubernetes_persistent_volume" "example" {
  metadata {
    name = "examplevolumename"
  }
  spec {
    capacity = {
      storage = "10Gi"
    }
    access_modes = ["ReadWriteMany"]
    persistent_volume_source {
      gce_persistent_disk {
        pd_name = "test-123"
      }
    }
  }
}
