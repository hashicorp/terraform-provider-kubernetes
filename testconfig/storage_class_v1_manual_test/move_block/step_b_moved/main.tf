# Step B — Migrate using a moved block.
#
# The moved block tells Terraform:
#   "the object previously tracked as kubernetes_storage_class.demo
#    is now tracked as kubernetes_storage_class_v1.demo"
#
# Terraform calls MoveState on the Framework resource, which translates
# the SDKv2 JSON state into the Framework model. The plan must show
# "No changes." — proving zero drift between the two implementations.
#
# Run:
#   cp ../step_a_sdkv2/terraform.tfstate ./terraform.tfstate
#   TF_CLI_CONFIG_FILE=/Users/midhunmohanan/Desktop/terraform-provider-kubernetes/testconfig/.terraformrc terraform init
#   TF_CLI_CONFIG_FILE=/Users/midhunmohanan/Desktop/terraform-provider-kubernetes/testconfig/.terraformrc terraform plan   ← must show: No changes.
#
# Verify:
#   - uid output matches step_a exactly (same Kubernetes object, no recreate)
#   - plan shows "No changes. Your infrastructure matches the configuration."
#   - no warnings about unknown attributes

moved {
  from = kubernetes_storage_class.demo
  to   = kubernetes_storage_class_v1.demo
}

# Config is identical to Step A — only the resource TYPE name changed.
resource "kubernetes_storage_class_v1" "demo" {
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
  description = "Must match the UID from step_a — same object, no recreate"
  value       = kubernetes_storage_class_v1.demo.metadata[0].uid
}
