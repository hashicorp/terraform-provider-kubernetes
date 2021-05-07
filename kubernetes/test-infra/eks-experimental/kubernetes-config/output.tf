output "kubeconfig" {
  value = abspath("${path.root}/${local_file.kubeconfig.filename}")
}
