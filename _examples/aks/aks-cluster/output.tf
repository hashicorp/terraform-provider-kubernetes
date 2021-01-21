output "client_cert" {
  value = azurerm_kubernetes_cluster.test.kube_config.0.client_certificate
}

output "client_key" {
  value = azurerm_kubernetes_cluster.test.kube_config.0.client_key
}

output "ca_cert" {
  value = azurerm_kubernetes_cluster.test.kube_config.0.cluster_ca_certificate
}

output "endpoint" {
  value = azurerm_kubernetes_cluster.test.kube_config.0.host
}
