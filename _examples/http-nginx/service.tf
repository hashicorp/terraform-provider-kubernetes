resource "kubernetes_service" "example" {
  metadata {
    name = "terraform-nginx-example"
  }
  spec {
    selector {
      App = "${kubernetes_replication_controller.example.metadata.0.labels.App}"
    }
    session_affinity = "ClientIP"
    port {
      port = 80
      target_port = 80
    }

    type = "LoadBalancer"
  }
}

output "lb_ip" {
  value = "${kubernetes_service.example.load_balancer_ingress.0.ip}"
}
