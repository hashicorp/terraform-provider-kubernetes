# Turn on prefix delegation in the VPC CNI of Amazon EKS.
resource "kubernetes_manifest_patch" "aws_node" {
  manifest = {
    apiVersion = "apps/v1"
    kind       = "DaemonSet"
    metadata = {
      name      = "aws-node"
      namespace = "kube-system"
    }
    spec = {
      template = {
        spec = {
          containers = [{
            name = "aws-node"
            env = [
              { name = "ENABLE_PREFIX_DELEGATION", value = "true" },
              { name = "WARM_PREFIX_TARGET", value = "1" },
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
