# Step 2 — Switch to the local Framework binary (dev override active).
#
# Config is IDENTICAL to step1 — same resource type, same attributes.
# The only thing that changes is the provider binary.
#
# Expected result: terraform plan shows "No changes."
# This proves the Framework provider reads the SDKv2 3.2.1 state
# without any drift, destroy, or recreate.
#
# Run:
#   cp ../step1_sdkv2/terraform.tfstate ./terraform.tfstate
#   export TF_CLI_CONFIG_FILE=/Users/midhunmohanan/Desktop/terraform-provider-kubernetes/testconfig/.terraformrc
#   terraform init
#   terraform plan    ← must show: No changes.
#
# Also verify:
#   - binding_uid output matches the value from step1
#   - no destroy/create in the plan
#   - terraform.tfstate identity_schema_version stays at 1

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
  description = "Must match the UID from step1 — same object, no recreate"
  value       = kubernetes_role_binding_v1.test.metadata[0].uid
}
