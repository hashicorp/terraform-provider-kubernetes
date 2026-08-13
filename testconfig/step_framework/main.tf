# Step B — Switch to the local Framework provider binary.
# The config is IDENTICAL to step_sdkv2/main.tf — same resource type, same attributes.
# Running terraform plan here must show: No changes.
# That proves the Framework resource reads the SDKv2 state correctly after UpgradeState runs.

resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = "sdkv2-to-framework-test"

    labels = {
      managed-by = "terraform"
    }

    annotations = {
      "example.com/note" = "provisioned by sdkv2 provider"
    }
  }

  value             = 200
  description       = "Testing SDKv2 to Framework migration"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}

output "name" {
  value = kubernetes_priority_class_v1.test.metadata[0].name
}

output "uid" {
  value = kubernetes_priority_class_v1.test.metadata[0].uid
}
