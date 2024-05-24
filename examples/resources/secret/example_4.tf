resource "kubernetes_secret" "example" {
  metadata {
    annotations = {
      "kubernetes.io/service-account.name" = "my-service-account"
    }

    generate_name = "my-service-account-"
  }

  type                           = "kubernetes.io/service-account-token"
  wait_for_service_account_token = true
}
