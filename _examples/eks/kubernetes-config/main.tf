provider "kubernetes" {
  host                   = var.cluster_endpoint
  cluster_ca_certificate = base64decode(var.cluster_ca_cert)
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
    command     = "aws"
  }
}

resource "kubernetes_config_map" "name" {
  depends_on  = [var.cluster_name]
  metadata {
    name      = "aws-auth"
    namespace = "kube-system"
  }

  data = {
    mapRoles = join(
      "\n",
      formatlist(local.mapped_role_format, var.k8s_node_role_arn),
    )
  }
}

# This allows the kubeconfig file to be refreshed during every Terraform apply.
# Optional: this kubeconfig file is only used for manual CLI access to the cluster.
resource "null_resource" "generate-kubeconfig" {
  provisioner "local-exec" {
    command = "aws eks update-kubeconfig --name ${var.cluster_name} --kubeconfig ${path.root}/kubeconfig"
  }
  triggers = {
    always_run = timestamp()
  }
}

resource "kubernetes_namespace" "test" {
  depends_on  = [var.cluster_name]
  metadata {
    name = "test"
  }
}

resource "kubernetes_deployment" "test" {
  depends_on  = [var.cluster_name]
  metadata {
    name = "test"
    namespace= kubernetes_namespace.test.metadata.0.name
  }
  spec {
    replicas = 2
    selector {
      match_labels = {
        app  = "test"
      }
    }
    template {
      metadata {
        labels = {
          app  = "test"
        }
      }
      spec {
        container {
          image = "nginx:1.19.4"
          name  = "nginx"

          resources {
            limits = {
              memory = "512M"
              cpu = "1"
            }
            requests = {
              memory = "256M"
              cpu = "50m"
            }
          }
        }
      }
    }
  }
}

provider "helm" {
  kubernetes {
    host                   = var.cluster_endpoint
    cluster_ca_certificate = base64decode(var.cluster_ca_cert)
    exec {
      api_version = "client.authentication.k8s.io/v1alpha1"
      args        = ["eks", "get-token", "--cluster-name", var.cluster_name]
      command     = "aws"
    }
  }
}

resource helm_release nginx_ingress {
  depends_on  = [var.cluster_name]
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set {
    name  = "service.type"
    value = "ClusterIP"
  }
}
