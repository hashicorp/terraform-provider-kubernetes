# terraform-operator-module

A Terraform module for managing the HashiCorp [terraform-k8s operator](https://github.com/hashicorp/terraform-k8s) with the Kubernetes and Kubernetes-alpha providers. 

## Usage
```
provider "kubernetes" {}
provider "kubernetes-alpha" {}
resource "kubernetes_namespace" "example" {
  metadata {
    name = kubernetes_manifest.namespace.object.metadata.name
  }
}

module "terraform-operator" {
  source = "github.com/hashicorp/terraform-provider-kubernetes-alpha/tree/master/examples/terraform-operator"

namespace       = kubernetes_manifest.namespace.object.metadata.name
tfc_credentials = file(var.tfc_credentials)
access_key_id     = var.access_key_id
secret_acess_key = var.secret_acess_key
}
```
