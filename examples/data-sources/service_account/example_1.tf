data "kubernetes_service_account" "example" {
  metadata {
    name = "terraform-example"
  }
}

data "kubernetes_secret" "example" {
  metadata {
    name = "${data.kubernetes_service_account.example.default_secret_name}"
  }
}
