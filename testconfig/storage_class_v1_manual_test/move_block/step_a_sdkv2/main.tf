# Step A — Create using the DEPRECATED kubernetes_storage_class resource type
# via the last published SDKv2 provider (3.0.1).
# This simulates a real user who has existing state written by the SDKv2 provider
# and wants to migrate to kubernetes_storage_class_v1 using a moved block.
#
# Run with registry config (no dev override):
#   TF_CLI_CONFIG_FILE=../../.terraformrc-registry terraform init
#   TF_CLI_CONFIG_FILE=../../.terraformrc-registry terraform apply
#
# After apply, note the uid output.
# Then proceed to step_b_moved.

resource "kubernetes_storage_class" "demo" {
  metadata {
    name = "tf-move-block-test-sc"
    labels = {
      managed-by = "terraform"
    }
    annotations = {
      "example.com/note" = "created-by-deprecated-type"
    }
  }

  storage_provisioner    = "rancher.io/local-path"
  reclaim_policy         = "Retain"
  allow_volume_expansion = true
}

output "uid" {
  description = "Kubernetes UID — must be identical after the moved block"
  value       = kubernetes_storage_class.demo.metadata[0].uid
}
