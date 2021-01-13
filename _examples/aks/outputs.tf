output "kubeconfig_path" {
  value = abspath("${path.root}/kubeconfig")
}

output "cluster_name" {
  value = local.cluster_name
}
