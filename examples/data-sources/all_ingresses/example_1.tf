data "kubernetes_all_ingresses" "all" {}

output "all-ingresses" {
  value = data.kubernetes_all_ingresses.all.ingresses
}

# Filter ingresses by label
data "kubernetes_all_ingresses" "frontend" {
  label_selector = "app=frontend"
}

output "frontend-ingresses" {
  value = data.kubernetes_all_ingresses.frontend.ingresses
}
