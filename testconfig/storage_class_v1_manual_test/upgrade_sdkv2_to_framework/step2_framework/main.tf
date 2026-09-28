# Step 2 — Switch to the local Framework binary (dev override active).
#
# Config is IDENTICAL to step1 — same resource type, same attributes.
# The only thing that changes is the provider binary.
#
# Expected result: terraform plan shows "No changes."
# This proves the Framework provider reads the SDKv2 3.2.1 state
# without any drift, destroy, or recreate.
#
# Run:
#   cp ../step1_sdkv2/terraform.tfstate ./terraform.tfstate
#   TF_CLI_CONFIG_FILE=/Users/midhunmohanan/Desktop/terraform-provider-kubernetes/testconfig/.terraformrc terraform init
#   TF_CLI_CONFIG_FILE=/Users/midhunmohanan/Desktop/terraform-provider-kubernetes/testconfig/.terraformrc terraform plan   ← must show: No changes.
#
# Verify:
#   - uid output matches step1 exactly
#   - no destroy/create in plan output
#   - resource_version is unchanged

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
  description = "Must match the UID from step1 — same object, no recreate"
  value       = kubernetes_storage_class_v1.test.metadata[0].uid
}

output "resource_version" {
  description = "Must match step1 — no update was made"
  value       = kubernetes_storage_class_v1.test.metadata[0].resource_version
}
