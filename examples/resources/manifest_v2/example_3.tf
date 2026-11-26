resource "kubernetes_manifest_v2" "web" {
  manifest = provider::kubernetes::manifest_decode(file("${path.module}/deployment.yaml"))

  # Succeed only when the new pods are running.
  wait {
    rollout = true
    condition {
      type   = "Available"
      status = "True"
    }
  }

  # A HorizontalPodAutoscaler sets the replica count after creation.
  computed_fields = ["spec.replicas"]

  timeouts {
    create = "15m"
    update = "15m"
  }
}

# Values the cluster sets are in object.
output "ready_replicas" {
  value = kubernetes_manifest_v2.web.object.status.readyReplicas
}
