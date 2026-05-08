data "kubernetes_secret_v1" "example" {
  metadata {
    name      = "example-secret"
    namespace = "kube-system"
  }
  binary_data = {
    "keystore.p12" = ""
    another_field  = ""
  }
}
