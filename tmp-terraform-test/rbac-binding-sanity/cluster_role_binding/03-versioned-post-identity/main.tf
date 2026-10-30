terraform {
  required_version = ">= 1.12.0"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "3.4.0"
    }
  }
}

variable "config_path" {
  type    = string
  default = "~/.kube/config"
}

variable "config_context" {
  type    = string
  default = "kind-kind"
}

variable "namespace" {
  type    = string
  default = "default"
}

variable "run_id" {
  type    = string
  default = "manual"
}

variable "phase" {
  type    = string
  default = "before"

  validation {
    condition     = contains(["before", "after"], var.phase)
    error_message = "phase must be before or after."
  }
}

provider "kubernetes" {
  config_path        = pathexpand(var.config_path)
  config_context     = var.config_context
  ignore_annotations = ["^outside[.]example/keep$"]
  ignore_labels      = ["^outside[.]example/owner$"]
}

locals {
  prefix = "crb-sanity-post-v1-${var.run_id}"
  after  = var.phase == "after"

  labels = local.after ? {
    managed                  = "after"
    added                    = "yes"
    "app.kubernetes.io/name" = "crb-sanity"
    } : {
    managed                  = "before"
    remove                   = "yes"
    "app.kubernetes.io/name" = "crb-sanity"
  }

  annotations = {
    managed             = var.phase
    "example.com/owner" = "sanity"
  }
}

resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = local.prefix
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get", "list"]
  }
}

# Exhaustive: explicit name, labels, annotations, every subject variant.
resource "kubernetes_cluster_role_binding_v1" "named" {
  metadata {
    name        = "${local.prefix}-named"
    labels      = local.labels
    annotations = local.annotations
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role_v1.test.metadata[0].name
  }

  # User: api_group explicit, namespace omitted (schema default "default").
  subject {
    kind      = "User"
    name      = "sanity-user-explicit-group"
    api_group = "rbac.authorization.k8s.io"
  }

  # User: api_group and namespace omitted.
  subject {
    kind = "User"
    name = "sanity-user-omitted-group"
  }

  # User: api_group "" (API defaults it to rbac.authorization.k8s.io) and namespace "".
  subject {
    kind      = "User"
    name      = "sanity-user-empty-group"
    api_group = ""
    namespace = ""
  }

  # User and Group sharing a name: positional api_group must not leak between them.
  subject {
    kind = "User"
    name = "alice"
  }

  subject {
    kind = "Group"
    name = "alice"
  }

  # Group: api_group explicit, namespace "".
  subject {
    kind      = "Group"
    name      = "sanity-group"
    api_group = "rbac.authorization.k8s.io"
    namespace = ""
  }

  # ServiceAccount: namespace omitted (schema default "default").
  subject {
    kind = "ServiceAccount"
    name = "default"
  }

  # ServiceAccount: namespace explicit, api_group "".
  subject {
    kind      = "ServiceAccount"
    name      = "sanity-sa"
    namespace = var.namespace
    api_group = ""
  }

  # Appended in phase=after: exercises an in-place subject update.
  dynamic "subject" {
    for_each = local.after ? ["added"] : []
    content {
      kind      = "Group"
      name      = "sanity-group-added-after"
      api_group = "rbac.authorization.k8s.io"
    }
  }
}

# Alternate shape: generate_name, no labels/annotations, single subject.
resource "kubernetes_cluster_role_binding_v1" "generated" {
  metadata {
    generate_name = "${local.prefix}-gen-"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = kubernetes_cluster_role_v1.test.metadata[0].name
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    namespace = var.namespace
  }
}

output "binding_names" {
  value = [kubernetes_cluster_role_binding_v1.named.metadata[0].name, kubernetes_cluster_role_binding_v1.generated.metadata[0].name]
}

output "binding_ids" {
  value = {
    named     = kubernetes_cluster_role_binding_v1.named.id
    generated = kubernetes_cluster_role_binding_v1.generated.id
  }
}

output "binding_uids" {
  value = {
    named     = kubernetes_cluster_role_binding_v1.named.metadata[0].uid
    generated = kubernetes_cluster_role_binding_v1.generated.metadata[0].uid
  }
}
