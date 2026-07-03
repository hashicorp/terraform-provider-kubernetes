# Patch an EKS-managed object you do not own: add proxy env to the aws-node DaemonSet,
# and remove an annotation by setting it to null. Destroying this resource relinquishes
# the fields (default) — it never deletes the DaemonSet.
resource "kubernetes_manifest_patch" "aws_node_proxy" {
  api_version = "apps/v1"
  kind        = "DaemonSet"
  name        = "aws-node"
  namespace   = "kube-system"

  field_manager   = "terraform-proxy"
  force_conflicts = true

  patch = {
    spec = {
      template = {
        metadata = {
          annotations = {
            # null removes the field (this manager gives up owning it)
            "eks.amazonaws.com/compute-type" = null
          }
        }
        spec = {
          containers = [
            {
              name = "aws-node"
              env = [
                { name = "HTTP_PROXY", value = "http://proxy.internal:3128" },
                { name = "HTTPS_PROXY", value = "http://proxy.internal:3128" },
              ]
            },
          ]
        }
      }
    }
  }

  destroy_behavior = "relinquish"
}

# Escape hatch: a strategic-merge patch expressed as JSON for cases the object form
# can't express.
resource "kubernetes_manifest_patch" "annotate_service" {
  api_version = "v1"
  kind        = "Service"
  name        = "ingress-nginx-controller"
  namespace   = "ingress-nginx"

  patch_type = "strategic"
  patch_json = jsonencode({
    metadata = {
      annotations = {
        "service.beta.kubernetes.io/do-loadbalancer-hostname" = "example.com"
      }
    }
  })
}
