output "kubeconfig_path" {
  value = abspath("${path.root}/kubeconfig")
}

output "cluster_name" {
  value = module.vpc.cluster_name
}

