# Step 1 — Provision kubernetes_role_binding_v1 using the published SDKv2
# provider (hashicorp/kubernetes@3.2.1).
#
# This simulates a real user who has existing terraform state written by
# the last published SDKv2 release before the Framework migration.
#
# Run:
#   unset TF_CLI_CONFIG_FILE      # disable dev override — use registry
#   terraform init
#   terraform apply
#
# After apply, note the binding_uid output.
# Then proceed to step2_framework.

resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = "tf-upgrade-test-binding"
    namespace = "default"
    labels = {
      managed-by = "terraform"
    }
    annotations = {
      "example.com/note" = "created-by-sdkv2-3.2.1"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "alice"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    namespace = "default"
    api_group = ""
  }

  subject {
    kind      = "Group"
    name      = "dev-team"
    api_group = "rbac.authorization.k8s.io"
  }
}

output "binding_uid" {
  description = "Kubernetes UID — must be identical after switching to Framework provider"
  value       = kubernetes_role_binding_v1.test.metadata[0].uid
}

output "identity_schema_version" {
  description = "Check terraform.tfstate — should be 1"
  value       = "check terraform.tfstate for identity_schema_version"
}
