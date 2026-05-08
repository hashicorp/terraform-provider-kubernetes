resource "kubernetes_service_account" "example" {
  metadata {
    name = "terraform-example"
  }
}

resource "kubernetes_secret" "example" {
  metadata {
    annotations = {
      "kubernetes.io/service-account.name" = kubernetes_service_account.example.metadata.0.name
    }

    generate_name = "terraform-example-"
  }

  type                           = "kubernetes.io/service-account-token"
  wait_for_service_account_token = true
}
