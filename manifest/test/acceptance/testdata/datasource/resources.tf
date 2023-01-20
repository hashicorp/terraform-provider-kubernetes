data "kubernetes_resources" "example"{
    kind = "Namespace"
    api_version = "v1"
    namespace = "test"
    label_selector = var.label_selector
    limit = var.limit
}