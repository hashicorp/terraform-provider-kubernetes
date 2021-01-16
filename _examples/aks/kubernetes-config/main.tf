provider "azurerm" {
  features {}
}

# The client certificate used for authenticating into the AKS cluster will eventually expire,
# (especially true if your clusters are created and destroyed periodically).
# This data source fetches new authentication certificates.
# Alternatively, use `terraform refresh` to fetch them manually.
data "azurerm_kubernetes_cluster" "main" {
depends_on = [var.cluster_id]
  name                = var.cluster_name
  resource_group_name = var.cluster_name
}

provider "kubernetes" {
  host                   = data.azurerm_kubernetes_cluster.main.kube_config.0.host
  client_key             = base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.client_key)
  client_certificate     = base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.client_certificate)
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.cluster_ca_certificate)
}

resource "kubernetes_namespace" "test" {
depends_on = [var.cluster_id]
  metadata {
    name = "test"
  }
}

resource "kubernetes_persistent_volume" "test" {
depends_on = [var.cluster_id]
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
        data_disk_uri = var.data_disk_uri
        disk_name = "managed"
        kind = "Managed"
      }
    }
  }
}

resource "kubernetes_deployment" "test" {
depends_on = [var.cluster_id]
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
    host                   = data.azurerm_kubernetes_cluster.main.kube_config.0.host
    client_key             = base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.client_key)
    client_certificate     = base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.client_certificate)
    cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.main.kube_config.0.cluster_ca_certificate)
  }
}

resource helm_release nginx_ingress {
depends_on = [var.cluster_id]
  name       = "nginx-ingress-controller"

  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx-ingress-controller"

  set {
    name  = "service.type"
    value = "ClusterIP"
  }
}
