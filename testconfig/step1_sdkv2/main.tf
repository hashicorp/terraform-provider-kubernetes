resource "kubernetes_priority_class_v1" "demo" {
  metadata {
    name = "tf-demo-priority-class"
    labels = {
      managed-by = "terraform"
    }
    annotations = {
      "example.com/note" = "created-by-sdkv2"
    }
  }

  value             = 500
  description       = "Demo priority class"
  preemption_policy = "PreemptLowerPriority"
  global_default    = false
}

output "name" {
  value = kubernetes_priority_class_v1.demo.metadata[0].name
}

output "uid" {
  value = kubernetes_priority_class_v1.demo.metadata[0].uid
}
