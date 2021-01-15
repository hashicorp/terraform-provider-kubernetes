output "cluster_ca_cert" {
  value = azurerm_kubernetes_cluster.test.kube_config.0.client_certificate
}

output "cluster_endpoint" {
  value = azurerm_kubernetes_cluster.test.kube_config.0.host
}

output "cluster_name" {
  value = azurerm_kubernetes_cluster.test.id
}

output "data_disk_uri" {
  value = azurerm_managed_disk.test.id
}

