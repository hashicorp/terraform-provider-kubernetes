# Step A — Provision kubernetes_priority_class_v1 using the RELEASED SDKv2 provider (v3.2.1).
# In v3.2.1 this resource is still served by the SDKv2 code and writes schema version 0 state.

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
