provider "kubernetes" {
  eks_cluster_region = "us-west-2"
  eks_cluster_name = "stack-eks-cluster-dev"
}

resource "kubernetes_namespace" "foo" {
  metadata {
    name = "arghhhh"
  }
}


resource "kubernetes_pod" "bar" {
  metadata {
    name = "terraform-example"
  }

  spec {
    container {
      image = "nginx:1.15.10"
      name  = "example"

      env {
        name  = "environment"
        value = "test"
      }
    }
  }
}
