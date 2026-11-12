resource "kubernetes_secret_v1_data" "example" {
  metadata {
    name = "my-secret"
  }
  data = {
    "username" = "admin"
    "password" = "s3cr3t"
  }
}
