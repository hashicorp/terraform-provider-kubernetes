# kubernetes_all_namespaces — Migration Verification

Manually verifies that a user upgrading from the last published SDKv2 provider
(`hashicorp/kubernetes@3.2.1`) to the local Framework build gets **no state
diff, no replacement, and no error** — the primary release-readiness gate for
this migration.

## Cluster requirements

A kind cluster named `kind` must be running and reachable via `~/.kube/config`.

```bash
kubectl config current-context   # should print: kind-kind
kubectl get namespaces           # should list at least: default, kube-system
```

## Files

```
step1_sdkv2/
  versions.tf   # pins hashicorp/kubernetes 3.2.1 from the registry
  main.tf       # data source + terraform_data anchor + outputs

step2_framework/
  versions.tf   # loose constraint; dev_overrides supplies the local binary
  main.tf       # IDENTICAL config to step1
```

The `terraform_data` anchor in both configs copies every data-source attribute
(`id` and `namespaces`) into managed-resource state. This makes `terraform plan`
in step 2 produce a **real diff** if the Framework provider returns different
values — the plan is not trivially empty.

## Step 1 — Apply with SDKv2 3.2.1

Establish state with the last published SDKv2 provider.

```bash
cd step1_sdkv2

# Make sure no dev_overrides are active — use the registry binary.
unset TF_CLI_CONFIG_FILE

terraform init
terraform apply -auto-approve
```

**Expected outputs (your cluster will differ in count/names but must be stable):**

```
namespace_count   = 8
namespaces        = ["default", "kube-node-lease", "kube-public", "kube-system",
                     "local-path-storage", "my-test-ns", "role-binding-demo", "test-namespace"]
id_fingerprint    = "9d7de5678d269c3f8ea5d8e15f9e2b3ae9e935e5ece2450bd3b2c7ec857fe111"
```

Note the `id_fingerprint` value — it must appear **unchanged** in step 2.

> **Recorded run (kind-kind, 2026-09-25):**
> 8 namespaces · fingerprint `9d7de5678d269c3f8ea5d8e15f9e2b3ae9e935e5ece2450bd3b2c7ec857fe111`

## Step 2 — Plan with local Framework binary

Switch the provider binary and prove zero drift.

```bash
# Copy the SDKv2 state produced by step 1.
cp step1_sdkv2/terraform.tfstate step2_framework/terraform.tfstate

cd step2_framework

# Point Terraform at the local binary via dev_overrides.
export TF_CLI_CONFIG_FILE=/Users/midhunmohanan/Desktop/terraform-provider-kubernetes/testconfig/.terraformrc

terraform init -upgrade
terraform plan
```

### What to look for

| Check | Expected |
|---|---|
| Terraform warning | `Warning: Provider development overrides are in effect` — **required**; proves the local binary ran, not a cached registry binary |
| Plan result | `No changes. Your infrastructure matches the configuration.` |
| `id_fingerprint` output | Identical hex string to step 1 |
| `namespaces` output | Same list, same order |
| `terraform_data.anchor` | No `update` action in the plan |

### If the plan is not empty

A non-empty plan means the Framework provider returned different attribute
values than SDKv2 did. Inspect the diff:

```bash
terraform plan -out=plan.bin
terraform show -json plan.bin | python3 -m json.tool | grep -A20 '"changes"'
```

Fields that changed indicate a parity regression in the Framework implementation.

## What this proves

`No changes` with a `terraform_data` anchor proves:

1. **`id`** — the Framework computes the same SHA-256 hex fingerprint as SDKv2.
2. **`namespaces`** — the Framework returns the same ordered list of strings.
3. **State compatibility** — the Framework provider can read SDKv2 3.2.1 state
   without any UpgradeState handler (the schema shapes are identical).
4. **No replacement** — no managed resource in the user's configuration is
   destroyed or recreated as a side-effect of upgrading the provider.
