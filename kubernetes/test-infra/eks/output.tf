output "kubeconfig_path" {
  value = abspath("${local.kubeconfig_path}/${local.kubeconfig_name}")
}

