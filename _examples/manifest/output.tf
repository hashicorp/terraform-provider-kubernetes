output "lron" {
  value = jsondecode(kubernetes_manifest.lron.resource).details.firstName
}

output "jim" {
  value = jsondecode(kubernetes_manifest.jim.resource).details.firstName
}