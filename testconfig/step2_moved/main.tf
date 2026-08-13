# Step 2 — Migrate to the new resource type using a moved block.
#
# What happens when `terraform plan` runs here:
#
#   1. Terraform sees the moved block and looks up the source address
#      kubernetes_priority_class.example in the state file.
#   2. It calls MoveState on kubernetes_priority_class_v1, passing the raw
#      SDKv2 JSON state as req.SourceRawState.JSON.
#   3. moveStateFromKubernetesPriorityClassHandler parses the JSON, normalises
#      generate_name / empty maps, and writes the result into resp.TargetState
#      as a PriorityClassModel (schema version 1, ListNestedBlock metadata).
#   4. Plan shows the resource being moved — NO destroy or create actions.
#   5. After apply, state is owned by kubernetes_priority_class_v1.example.
#
# The moved block can be removed once all team members have run `terraform apply`.

moved {
  from = kubernetes_priority_class.example
  to   = kubernetes_priority_class_v1.example
}

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
