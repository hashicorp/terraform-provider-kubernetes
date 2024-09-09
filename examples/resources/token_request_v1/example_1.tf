resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "test"
  }
}

resource "kubernetes_token_request_v1" "test" {
  metadata {
    name = kubernetes_service_account_v1.test.metadata.0.name
  }
  spec {
    audiences = [
      "api",
      "vault",
      "factors"
    ]
  }
}

output "tokenValue" {
  value = kubernetes_token_request_v1.test.token
}
