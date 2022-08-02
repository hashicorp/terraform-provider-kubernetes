output "kubeconfig_path" {
  value = local_file.kubeconfig.filename
}

output "cluster_name" {
  value = azurerm_kubernetes_cluster.tf-k8s-acc.name
}
