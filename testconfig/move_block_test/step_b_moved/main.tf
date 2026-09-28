terraform {
  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
  }
}

provider "kubernetes" {
  config_path = "~/.kube/config"
}

moved {
  from = kubernetes_priority_class.example
  to   = kubernetes_priority_class_v1.example
}

resource "kubernetes_priority_class_v1" "example" {
  metadata {
    name = "manual-move-test-priority-class"
    labels = {
      environment = "test"
    }
    annotations = {
      "example.com/owner" = "platform-team"
    }
  }

  value             = 600
  description       = "Manual test priority class for moved block verification"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}
