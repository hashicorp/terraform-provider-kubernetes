output "kubeconfig_path" {
  value = abspath("${local.kubeconfig_path}/${local.kubeconfig_name}")
}

output "cluster_name" {
  value = module.vpc.cluster_name
}