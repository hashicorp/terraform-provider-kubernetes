data "kubernetes_secret" "example" {
  metadata {
    name = "basic-auth"
  }
}
