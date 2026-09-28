# ─────────────────────────────────────────────────────────────────────────────
# Step 1 — kubernetes_all_namespaces with SDKv2 provider 3.2.1
#
# Purpose
# -------
# Establish terraform.tfstate that looks exactly like what a real upgrading user
# has on disk today.  The terraform_data anchor resource copies every output
# attribute from the data source into managed-resource state, which is what
# ExpectEmptyPlan actually diffs in the migration test.
#
# Namespaces present in the kind cluster when this config was authored:
#   default, kube-node-lease, kube-public, kube-system,
#   local-path-storage, my-test-ns, role-binding-demo, test-namespace
#
# Run
# ---
#   unset TF_CLI_CONFIG_FILE        # ensure no dev_overrides active
#   terraform init
#   terraform apply -auto-approve
#
# Then copy state to step2:
#   cp terraform.tfstate ../step2_framework/terraform.tfstate
# ─────────────────────────────────────────────────────────────────────────────

# ── Read every namespace from the cluster ────────────────────────────────────
data "kubernetes_all_namespaces" "test" {}

# ── Anchor: pins every data-source attribute into managed-resource state ─────
# This is what makes the ExpectEmptyPlan in step2 a real diff, not a no-op.
# Without this resource, Terraform has nothing concrete to compare between the
# SDKv2 state and the Framework re-read, so the plan is trivially empty.
resource "terraform_data" "anchor" {
  input = {
    id         = data.kubernetes_all_namespaces.test.id
    namespaces = data.kubernetes_all_namespaces.test.namespaces
  }
}

# ── Outputs: make the SDKv2 values visible for manual inspection ─────────────
output "namespace_count" {
  description = "Total number of namespaces returned by the SDKv2 provider."
  value       = length(data.kubernetes_all_namespaces.test.namespaces)
}

output "namespaces" {
  description = "Full list of namespace names — must match step2 exactly."
  value       = data.kubernetes_all_namespaces.test.namespaces
}

output "id_fingerprint" {
  description = "SHA-256 fingerprint written by the SDKv2 provider — must match step2."
  value       = data.kubernetes_all_namespaces.test.id
}
