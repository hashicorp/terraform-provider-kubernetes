terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "3.2.1"
    }
  }
}

provider "kubernetes" {
  config_path = "~/.kube/config"
}

resource "kubernetes_priority_class" "example" {
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
