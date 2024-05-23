data "kubernetes_persistent_volume_claim_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}
