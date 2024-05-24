data "kubernetes_secret_v1" "example" {
  metadata {
    name = "basic-auth"
  }
}
