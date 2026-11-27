# Set environment variables of a DaemonSet that a cluster add-on manages.
resource "kubernetes_manifest_patch" "node_agent" {
  manifest = {
    apiVersion = "apps/v1"
    kind       = "DaemonSet"
    metadata = {
      name      = "node-agent"
      namespace = "kube-system"
    }
    spec = {
      template = {
        spec = {
          containers = [{
            name = "agent"
            env = [
              { name = "LOG_LEVEL", value = "debug" },
              { name = "HTTPS_PROXY", value = "http://proxy.example.com:3128" },
            ]
          }]
        }
      }
    }
  }

  # The add-on sets some of these variables itself.
  field_manager {
    force_conflicts = true
  }

  # The nodes run with the new setting when the apply ends.
  wait {
    rollout = true
  }
}
