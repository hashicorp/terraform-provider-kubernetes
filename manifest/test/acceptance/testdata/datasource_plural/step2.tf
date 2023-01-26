data "kubernetes_resources" "example"{
    kind = "ConfigMap"
    api_version = "v1"
    namespace = var.namespace
    label_selector = var.label_selector
    limit = var.limit
}