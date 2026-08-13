# testconfig — moved block demo for `kubernetes_priority_class` → `kubernetes_priority_class_v1`

This directory demonstrates how a user migrates from the deprecated
`kubernetes_priority_class` (SDKv2) resource to the new
`kubernetes_priority_class_v1` (Plugin Framework) resource using a Terraform
`moved` block — **without** destroying and recreating the Kubernetes object.

## Prerequisites

- A running Kubernetes cluster (kubeconfig at `~/.kube/config` or set via `KUBECONFIG`)
- The local provider binary already built at the repo root:
  ```bash
  go build -o terraform-provider-kubernetes_v99.0.0 .
  ```

## Step-by-step walkthrough

### Step 1 — Provision with the old (deprecated) resource type

Use `step1_old_resource/` to create the PriorityClass using `kubernetes_priority_class`.

```bash
cd testconfig/step1_old_resource
terraform apply
```

This writes SDKv2 state (schema version 0) for `kubernetes_priority_class.example`.

---

### Step 2 — Migrate using the `moved` block

Use `step2_moved/` which contains:
- The `kubernetes_priority_class_v1` resource declaration (new type)
- A `moved` block that tells Terraform to transfer state from the old address to the new one

```bash
cd testconfig/step2_moved
terraform plan   # must show: "kubernetes_priority_class_v1.example will be moved"
                 # and NO destroy/create actions
terraform apply
```

Terraform calls `MoveState` on the Framework resource, which translates the
SDKv2 JSON state into the v1 schema. After apply the state is owned by
`kubernetes_priority_class_v1.example` at schema version 1.

---

### Step 3 — Steady state (moved block removed)

Use `step3_final/` which is the clean config after the `moved` block is removed.

```bash
cd testconfig/step3_final
terraform plan   # must show: No changes.
```

---

## Shared state

All three steps share a single state file via `terraform.tfstate` in each
step's directory. Copy the state forward between steps:

```bash
# after step 1
cp testconfig/step1_old_resource/terraform.tfstate \
   testconfig/step2_moved/terraform.tfstate

# after step 2
cp testconfig/step2_moved/terraform.tfstate \
   testconfig/step3_final/terraform.tfstate
```

Or use a shared backend (S3, local path, etc.) configured in each `main.tf`.
