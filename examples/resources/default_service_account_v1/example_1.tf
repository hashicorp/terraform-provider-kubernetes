resource "kubernetes_default_service_account_v1" "example" {
  metadata {
    namespace = "terraform-example"
  }
  secret {
    name = "${kubernetes_secret_v1.example.metadata.0.name}"
  }
}

resource "kubernetes_secret_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}
