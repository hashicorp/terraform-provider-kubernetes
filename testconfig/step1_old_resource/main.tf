# Step 1 — Create the PriorityClass with the OLD (deprecated) resource type.
#
# kubernetes_priority_class is the SDKv2 resource registered in kubernetes/provider.go.
# Running `terraform apply` here creates the Kubernetes object and writes schema version 0
# state at the address: kubernetes_priority_class.example

resource "kubernetes_priority_class" "example" {
  metadata {
    name = "demo-priority-class"

    labels = {
      managed-by = "terraform"
    }

    annotations = {
      "example.com/note" = "created by the deprecated resource type"
    }
  }

  value             = 500
  description       = "Demo priority class for moved-block migration"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}

output "priority_class_name" {
  value = kubernetes_priority_class.example.metadata[0].name
}

output "priority_class_uid" {
  value = kubernetes_priority_class.example.metadata[0].uid
}
