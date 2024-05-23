data "kubernetes_persistent_volume_claim" "example" {
  metadata {
    name = "terraform-example"
  }
}
