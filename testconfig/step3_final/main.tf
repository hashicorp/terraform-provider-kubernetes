# Step 3 — Clean config after the moved block has been removed.
#
# The moved block is gone. This is the steady-state config that every team
# member uses going forward. Running `terraform plan` here must show:
#
#   No changes. Your infrastructure matches the configuration.
#
# This confirms the MoveState handler wrote correct state in step 2 —
# no phantom diffs, no destroy/create, no attribute churn.

resource "kubernetes_priority_class_v1" "example" {
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
  value = kubernetes_priority_class_v1.example.metadata[0].name
}

output "priority_class_uid" {
  value = kubernetes_priority_class_v1.example.metadata[0].uid
}
