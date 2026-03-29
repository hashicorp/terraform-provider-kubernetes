# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

output "kubeconfig" {
  value     = azurerm_kubernetes_cluster.test.kube_config_raw
  sensitive = true
}

output "cluster_name" {
  value = "test-aks-${random_pet.name.id}"
}
