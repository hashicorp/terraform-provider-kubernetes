# ─────────────────────────────────────────────────────────────────────────────
# Step 2 — kubernetes_all_namespaces with local Framework binary
#
# Purpose
# -------
# Prove that a user who already has SDKv2 3.2.1 state on disk can switch to
# the Framework provider WITHOUT any state diff, replacement, or error.
#
# The config is IDENTICAL to step1.  The only things that change are:
#   1. The provider binary — local Framework build via dev_overrides
#   2. TF_CLI_CONFIG_FILE points to testconfig/.terraformrc
#
# What "success" looks like
# ─────────────────────────
#   terraform plan  →  "No changes. Your infrastructure matches the configuration."
#   terraform show  →  id fingerprint and namespaces list are identical to step1
#
# Run
# ---
#   cp ../step1_sdkv2/terraform.tfstate ./terraform.tfstate
#   export TF_CLI_CONFIG_FILE=/Users/midhunmohanan/Desktop/terraform-provider-kubernetes/testconfig/.terraformrc
#   terraform init -upgrade
#   terraform plan          ← MUST print "No changes."
#   terraform show -json | python3 -m json.tool   ← inspect state if needed
# ─────────────────────────────────────────────────────────────────────────────

# ── Read every namespace from the cluster ─────────────────────────────────────
# Config is identical to step1 — same block, same attributes, same data source type.
data "kubernetes_all_namespaces" "test" {}

# ── Anchor — IDENTICAL to step1 ───────────────────────────────────────────────
# If the Framework provider returns different id or namespaces values than SDKv2
# did, terraform_data.anchor will plan an in-place update — which fails our
# "No changes" requirement.
resource "terraform_data" "anchor" {
  input = {
    id         = data.kubernetes_all_namespaces.test.id
    namespaces = data.kubernetes_all_namespaces.test.namespaces
  }
}

# ── Outputs — IDENTICAL to step1 ──────────────────────────────────────────────
output "namespace_count" {
  description = "Must equal the count from step1."
  value       = length(data.kubernetes_all_namespaces.test.namespaces)
}

output "namespaces" {
  description = "Must be byte-for-byte identical to the step1 output."
  value       = data.kubernetes_all_namespaces.test.namespaces
}

output "id_fingerprint" {
  description = "Must equal the SHA-256 fingerprint from step1."
  value       = data.kubernetes_all_namespaces.test.id
}
