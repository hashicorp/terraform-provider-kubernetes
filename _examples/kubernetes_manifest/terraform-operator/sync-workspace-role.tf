# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "kubernetes_role" "tfc-role" {
  metadata {
    name      = "${kubernetes_manifest.namespace.object.metadata.name}-sync-workspace"
    namespace = kubernetes_manifest.namespace.object.metadata.name
    labels = {
      app = kubernetes_manifest.namespace.object.metadata.name
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "services", "services/finalizers", "endpoints", "persistentvolumeclaims", "events", "configmaps", "secrets"]
    verbs      = ["*"]
  }
  rule {
    api_groups = ["apps"]
    resources  = ["deployments", "daemonsets", "replicasets", "statefulsets"]
    verbs      = ["*"]
  }
  rule {
    api_groups = ["monitoring.coreos.com"]
    resources  = ["servicemonitors"]
    verbs      = ["get", "create"]
  }

  rule {
    api_groups     = ["apps"]
    resource_names = ["terraform-k8s"]
    resources      = ["deployments/finalizers"]
    verbs          = ["update"]
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }

  rule {
    api_groups = ["apps"]
    resources  = ["replicasets"]
    verbs      = ["get"]
  }

  rule {
    api_groups = ["app.terraform.io"]
    resources  = ["*", "workspaces"]
    verbs      = ["*"]
  }
}
