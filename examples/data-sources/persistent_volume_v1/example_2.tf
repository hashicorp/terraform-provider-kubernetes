data "kubernetes_persistent_volume_v1" "example" {
  metadata {
    name = "terraform-example"
  }
}
data "kubernetes_secret" "example" {
  metadata {
    name = data.kubernetes_persistent_volume_v1.example.spec[0].persistent_volume_source[0].azure_file[0].secret_name
  }
}
output "azure_storageaccount_name" {
  value = data.kubernetes_secret.example.data.azurestorageaccountname
}
output "azure_storageaccount_key" {
  value = data.kubernetes_secret.example.data.azurestorageaccountkey
}
