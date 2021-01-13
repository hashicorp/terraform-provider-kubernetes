# This fetches a new token, which will expire in 1 hour.
data "azurerm_kubernetes_cluster" "main" {
  name                = var.cluster_name
  resource_group_name = var.cluster_name
}

provider "kubernetes" {
  host                   = "${data.azurerm_kubernetes_cluster.main.kube_config.0.host}"
  client_certificate     = "${base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.client_certificate)}"
  client_key             = "${base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.client_key)}"
  cluster_ca_certificate = "${base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.cluster_ca_certificate)}"
}

resource "kubernetes_namespace" "test" {
depends_on = [var.cluster_name]
  metadata {
    name = "test"
  }
}

resource "kubernetes_persistent_volume" "test" {
depends_on = [var.cluster_name]
  metadata {
    name = "test"
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes = ["ReadWriteOnce"]
    persistent_volume_source {
      azure_disk {
        caching_mode = "None"
        data_disk_uri = var.disk_uri
        disk_name = "managed"
        kind = "Managed"
      }
    }
  }
}

resource "kubernetes_deployment" "test" {
  metadata {
    name = "test"
    namespace= kubernetes_namespace.test.metadata.0.name
  }
  spec {
    replicas = 2
    selector {
      match_labels = {
        TestLabelOne   = "one"
      }
    }
    template {
      metadata {
        labels = {
          TestLabelOne   = "one"
        }
      }
      spec {
        container {
          image = "nginx:1.19.4"
          name  = "tf-acc-test"

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
    host = var.cluster_endpoint
    token = data.google_client_config.default.access_token
    cluster_ca_certificate = base64decode(var.cluster_ca_cert)
  }
}

resource helm_release nginx_ingress {
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set {
    name  = "service.type"
    value = "ClusterIP"
  }
}

data "template_file" "kubeconfig" {
  template = file("${path.module}/kubeconfig-template.yaml")

  vars = {
    cluster_name    = var.cluster_name
    endpoint        = var.cluster_endpoint
    cluster_ca      = var.cluster_ca_cert
    cluster_token   = data.google_client_config.default.access_token
  }
}

resource "local_file" "kubeconfig" {
  depends_on = [var.cluster_id]
  content  = data.template_file.kubeconfig.rendered
  filename = "${path.root}/kubeconfig"
}

