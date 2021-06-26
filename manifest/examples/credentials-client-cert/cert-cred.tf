# Example demonstrates how to authenticate to a cluster API using client certificates
#
variable "minikube_ip" {
  type = string
}

provider "kubernetes-alpha" {

  host = "https://${var.minikube_ip}:8443"

  cluster_ca_certificate = file("~/.minikube/ca.crt")

  client_certificate = file("~/.minikube/profiles/minikube/client.crt")
  client_key         = file("~/.minikube/profiles/minikube/client.key")
}

resource "kubernetes_manifest" "test-namespace" {
  provider = kubernetes-alpha

  manifest = {
    "apiVersion" = "v1"
    "kind"       = "Namespace"
    "metadata" = {
      "name" = "tf-demo"
    }
  }
}
