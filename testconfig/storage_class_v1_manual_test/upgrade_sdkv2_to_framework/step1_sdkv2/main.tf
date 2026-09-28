# Step 1 — Provision kubernetes_storage_class_v1 using the published SDKv2
# provider (hashicorp/kubernetes@3.2.1).
#
# This simulates a real user who has existing terraform state written by
# the last published SDKv2 release before the Framework migration.
#
# Run with registry config (no dev override):
#   TF_CLI_CONFIG_FILE=../../.terraformrc-registry terraform init
#   TF_CLI_CONFIG_FILE=../../.terraformrc-registry terraform apply
#
# After apply, note the uid output.
# Then proceed to step2_framework.

resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "tf-upgrade-test-sc"
    labels = {
      managed-by = "terraform"
    }
    annotations = {
      "example.com/note" = "created-by-sdkv2-3.2.1"
    }
  }

  storage_provisioner    = "rancher.io/local-path"
  reclaim_policy         = "Delete"
  volume_binding_mode    = "WaitForFirstConsumer"
  allow_volume_expansion = true
  mount_options          = ["noatime"]
}

output "uid" {
  description = "Kubernetes UID — must be identical after switching to Framework provider"
  value       = kubernetes_storage_class_v1.test.metadata[0].uid
}

output "resource_version" {
  description = "Check this is unchanged after switching to Framework provider"
  value       = kubernetes_storage_class_v1.test.metadata[0].resource_version
}
