output "cluster_id" {
  value = azurerm_kubernetes_cluster.test.id
}

output "data_disk_uri" {
  value = azurerm_managed_disk.test.id
}
