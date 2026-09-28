# =============================================================================
# kubernetes_role_binding_v1 — demonstration config
#
# What this does:
#   1. Creates a Role  (pod-reader) that can only GET/LIST/WATCH pods
#      inside the "demo" namespace.
#   2. Binds that Role to three different subject types so you can see
#      every flavour of the subject block in one place.
#
# Apply order: namespace → role → role_binding  (Terraform resolves deps)
# =============================================================================

resource "kubernetes_namespace_v1" "demo" {
  metadata {
    name = "role-binding-demo"
  }
}

# ── The Role that grants read-only access to pods ─────────────────────────────
resource "kubernetes_role_v1" "pod_reader" {
  metadata {
    name      = "pod-reader"
    namespace = kubernetes_namespace_v1.demo.metadata[0].name
  }

  rule {
    api_groups = [""]          # "" means the core API group
    resources  = ["pods"]
    verbs      = ["get", "list", "watch"]
  }
}

# ── The RoleBinding that wires up subjects → Role ────────────────────────────
resource "kubernetes_role_binding_v1" "demo" {
  metadata {
    name      = "read-pods-binding"
    namespace = kubernetes_namespace_v1.demo.metadata[0].name

    labels = {
      managed-by = "terraform"
    }
  }

  # ── role_ref: which Role (or ClusterRole) to grant  ───────────────────────
  # ForceNew = true → changing role_ref destroys & re-creates the binding.
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"                          # "Role" or "ClusterRole"
    name      = kubernetes_role_v1.pod_reader.metadata[0].name
  }

  # ── subject 1 : a plain Kubernetes User  ──────────────────────────────────
  # Users are not K8s objects; they come from the cluster's auth backend
  # (cert CN, OIDC claim, etc.).  No namespace field is meaningful here.
  subject {
    kind      = "User"
    name      = "alice"
    api_group = "rbac.authorization.k8s.io"
  }

  # ── subject 2 : a ServiceAccount  ─────────────────────────────────────────
  # ServiceAccounts ARE namespaced K8s objects.  api_group must be "".
  subject {
    kind      = "ServiceAccount"
    name      = "default"
    namespace = kubernetes_namespace_v1.demo.metadata[0].name
    api_group = ""
  }

  # ── subject 3 : a Group  ──────────────────────────────────────────────────
  # Groups are virtual; well-known ones: system:masters, system:authenticated.
  subject {
    kind      = "Group"
    name      = "dev-team"
    api_group = "rbac.authorization.k8s.io"
  }
}

# ── Outputs ───────────────────────────────────────────────────────────────────
output "role_binding_name" {
  value = kubernetes_role_binding_v1.demo.metadata[0].name
}

output "role_binding_uid" {
  value = kubernetes_role_binding_v1.demo.metadata[0].uid
}

output "role_binding_namespace" {
  value = kubernetes_role_binding_v1.demo.metadata[0].namespace
}
