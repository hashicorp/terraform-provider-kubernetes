# RoleBinding — case 02: `kubernetes_role_binding` on v3.4.0

Part of the [RoleBinding suite](../README.md) (PR [#2978](https://github.com/hashicorp/terraform-provider-kubernetes/pull/2978)). Finding codes (F1–F6) are explained in the [top-level README](../../README.md#known-findings-f1f6).

## What This Case Proves

Starts on **v3.4.0** (identity already in state) with the deprecated **unversioned** type. Step 2 is a plain provider bump. Step 3 moves it to `_v1` with `moved` blocks, which is where the SDKv2 → Framework migration happens.

Through every step, the Kubernetes objects must **never be recreated or rewritten** by the upgrade or move. Terraform must converge to `No changes`.

## Objects It Creates

| Terraform address (at start) | Kubernetes object | Why it exists |
| --- | --- | --- |
| `kubernetes_role_v1.test` | Role `rb-sanity-post-unv-manual` (namespace `default`) | target of `role_ref` kind `Role` |
| `kubernetes_cluster_role_v1.test` | ClusterRole `rb-sanity-post-unv-manual` | target of `role_ref` kind `ClusterRole` |
| `kubernetes_role_binding.role` | RoleBinding `rb-sanity-post-unv-manual-role` | **exhaustive binding**: name, labels, annotations, every subject variant, binds the Role |
| `kubernetes_role_binding.cluster_role` | RoleBinding `rb-sanity-post-unv-manual-gen-xxxxx` | **alternate binding**: `generate_name`, no labels/annotations, binds the ClusterRole |

## Files in This Folder

| File | Purpose |
| --- | --- |
| [main.tf](main.tf) | Pinned to `hashicorp/kubernetes` **3.4.0**. `phase` variable switches between the before/after update. |
| *created while testing* | `terraform.tfstate*` backups, `live.*.json` snapshots, `moved.tf`, `main.tf.unversioned` |

## Step 0: Prepare the Shell

Run every step in **one terminal** (zsh or bash). The variables set here are reused later.

```sh
SANITY="$(git rev-parse --show-toplevel)/tmp-terraform-test/rbac-binding-sanity"
cd "$SANITY/role_binding/02-unversioned-post-identity"
kind=rolebinding; ns=default
cat "/tmp/k8s-sanity/role_binding/BUILD_INFO"     # which PR commit the new build is
```

**Starting over?** Only if you already ran steps here before. This destroys the objects using the
newest provider (it can read state written by any version) and deletes local files **only if the destroy
succeeded**. If you used a custom `-var run_id=...`, add it to the destroy command.

```sh
export TF_CLI_CONFIG_FILE="$SANITY/role_binding/released.tfrc"
if [ ! -f terraform.tfstate ] || terraform destroy -auto-approve -var=phase=after; then
    [ -f main.tf.unversioned ] && mv main.tf.unversioned main.tf
    rm -rf .terraform .terraform.lock.hcl terraform.tfstate* live.*.json moved.tf
else
    echo "STOP: destroy failed, state kept. Fix the error before resetting."
fi
```

## Step 1: Create Everything with the Released Provider (v3.4.0)

**Why:** Builds the state a real user has today, before upgrading.

```sh
export TF_CLI_CONFIG_FILE="$SANITY/registry.tfrc"
terraform init
terraform apply
```

- `init` prints `Installed hashicorp/kubernetes v3.4.0`.
- `apply`: `Plan: 4 to add, 0 to change, 0 to destroy`.
- Fail: If you see `Provider development overrides are in effect`, stop. The baseline must come from the registry.

Add metadata outside Terraform that the provider is configured to ignore (`ignore_labels`/`ignore_annotations`):

```sh
names=($(terraform output -json binding_names | jq -r '.[]'))
first=$(terraform output -json binding_names | jq -r '.[0]')
printf '%s\n' "${names[@]}"     # 2 names: the exhaustive binding, then the generated one
kubectl annotate "$kind" -n "$ns" "$first" outside.example/keep=external
kubectl label    "$kind" -n "$ns" "$first" outside.example/owner=external
terraform plan
```

- `No changes`, because ignored keys are not managed.

Save the baseline (live objects + state). Everything later is compared against these files:

```sh
kubectl get "$kind" -n "$ns" "${names[@]}" -o json > live.sdk-v2.json
cp terraform.tfstate terraform.tfstate.sdk-v2
jq -r '.items[] | .metadata.name + "  uid=" + .metadata.uid + "  rv=" + .metadata.resourceVersion' live.sdk-v2.json
```

Note the `uid`/`rv` values. They must not change in steps 2 and 3.

## Step 2: Switch to the New Build (Provider Upgrade Only)

**Why:** Users who only bump the provider keep `kubernetes_role_binding`, which is still SDKv2 in the new build. Nothing should change.

```sh
export TF_CLI_CONFIG_FILE="$SANITY/role_binding/released.tfrc"    # no terraform init needed
terraform plan -refresh=false
terraform plan
```

- Every command prints `Warning: Provider development overrides are in effect`. Without it, you are not testing the new build.
- Never `to add`, `to destroy`, or `must be replaced`.
- Both plans: `No changes. Your infrastructure matches the configuration.`

```sh
terraform apply
terraform plan
terraform plan -refresh=false
```

- Both follow-up plans: `No changes`.

Compare with the baseline:

```sh
kubectl get "$kind" -n "$ns" "${names[@]}" -o json > live.upgrade.json
diff <(jq -S . live.sdk-v2.json) <(jq -S . live.upgrade.json) && echo "LIVE: identical"
diff terraform.tfstate.sdk-v2 terraform.tfstate
```

- **`LIVE: identical`** is the most important check. Same `uid` and `resourceVersion` means the provider did not recreate or write the objects.
- Known: State diff, observed in the last run (acceptable):
  - `serial` and `resource_version` (the refresh picks up your `kubectl annotate/label` from step 1)
  - `labels`/`annotations` `null` → `{}`
  - the Role/ClusterRole rule `resource_names`/`non_resource_urls` `null` → `[]`
- Fail: Anything else under `attributes` (for example `id`, `name`, `uid`, `role_ref`, or a subject's `kind`/`name`) is a failure.

## Step 3: Move the Unversioned Type to `_v1`

**Why:** This is the migration for users of the deprecated type. `moved` blocks hand state from `kubernetes_role_binding` (SDKv2) to `kubernetes_role_binding_v1` (Framework) via the provider's `MoveState`.

```sh
cp terraform.tfstate terraform.tfstate.pre-move
cp main.tf main.tf.unversioned
sed -i.bak -E 's/kubernetes_role_binding([" .])/kubernetes_role_binding_v1\1/g' main.tf && rm main.tf.bak
cat > moved.tf <<'EOF'
moved {
  from = kubernetes_role_binding.role
  to   = kubernetes_role_binding_v1.role
}

moved {
  from = kubernetes_role_binding.cluster_role
  to   = kubernetes_role_binding_v1.cluster_role
}
EOF
grep -n role_binding main.tf moved.tf
terraform plan -refresh=false
terraform plan
```

- Each binding shows `# (moved from kubernetes_role_binding.<name>)`, and `Plan: 0 to add, 2 to change, 0 to destroy`.
- Known: Expected one-time changes: **F2** (`api_group -> null` on `sanity-user-empty-group`), **F4** (`annotations = {} -> null`, `labels = {} -> null` on the generated binding).
- Fail: Any `to add` / `to destroy`. Never change the type without the `moved` blocks; that would recreate the bindings.

```sh
terraform apply
terraform plan
terraform plan -refresh=false
kubectl get "$kind" -n "$ns" "${names[@]}" -o json > live.moved.json
diff <(jq -S . live.sdk-v2.json) <(jq -S . live.moved.json) && echo "LIVE: identical"
diff terraform.tfstate.pre-move terraform.tfstate
```

- Plans: `No changes`. `LIVE: identical`.
- Known: State diff (observed): `type` `kubernetes_role_binding` → `kubernetes_role_binding_v1`; `api_group` `"rbac…"` → `""` (F2); `generate_name` `""` → `null`; `labels`/`annotations` `{}` → `null` on the generated binding (F4); `private` removed; `serial`.

## Step 4: Update in Place (`phase=after`)

**Why:** Proves updates on the new build send the right fields. It changes labels (update, remove, add) and annotations, and appends one subject.

```sh
terraform plan  -var=phase=after
terraform apply -var=phase=after
terraform plan  -var=phase=after
terraform plan  -var=phase=after -refresh=false
```

- First plan: `0 to add, 1 to change, 0 to destroy`, only on the exhaustive binding (`kubernetes_role_binding_v1.role`): `managed` before→after, `remove` removed, `added` added, and new subject `sanity-group-added-after`.
- Both follow-up plans: `No changes`.

Check that the update really reached Kubernetes:

```sh
kubectl get "$kind" -n "$ns" "$first" -o json | jq '{uid: .metadata.uid, labels: .metadata.labels, annotations: .metadata.annotations, lastSubject: .subjects[-1]}'
jq -r '.items[0].metadata.uid' live.sdk-v2.json        # must equal the uid above
grep -c outside.example terraform.tfstate                # must print 0
```

- `uid` unchanged; labels contain `managed=after`, `added=yes`, **no** `remove`, and still `outside.example/owner`.
- Annotations contain `managed=after` and still `outside.example/keep`. `lastSubject` is Group `sanity-group-added-after`.
- The ignored keys never enter state (`0`).

From here on, keep passing `-var=phase=after`.

## Step 5: Drift (Change Made Outside Terraform)

```sh
kubectl patch "$kind" -n "$ns" "$first" --type=json -p '[{"op":"remove","path":"/subjects/0"}]'
terraform plan  -var=phase=after      # expect: 1 to change: re-adds subject sanity-user-explicit-group
terraform apply -var=phase=after
terraform plan  -var=phase=after      # expect: No changes
```

## Step 6: Import

**Why:** An imported object must converge with the config. The import ID format is `default/<name>`.

```sh
id=$(terraform output -json binding_ids | jq -r '.role')
echo "$id"
terraform state rm kubernetes_role_binding_v1.role
terraform import -var=phase=after kubernetes_role_binding_v1.role "$id"
terraform plan -var=phase=after
```

- `Import successful!`
- Fail: **Known failure (F1):** the plan is **not** empty. It shows `1 to change` with the F2 `api_group -> null` on `sanity-user-empty-group`. Record it, then `terraform apply -var=phase=after` and `terraform plan -var=phase=after`, which should show `No changes`.

## Step 7: Clean Up

```sh
terraform destroy -var=phase=after                                    # expect: Destroy complete! Resources: 4 destroyed.
kubectl get "$kind" -n "$ns" "${names[@]}" --ignore-not-found -o name   # expect: prints nothing
```

A connection or permission error is **not** proof of deletion. Re-run until the lookup succeeds with no output.

## Record Your Result

| Step | Expected | Your result |
| --- | --- | --- |
| 1 Baseline | registry version installed, no override warning | |
| 2 Upgrade | No changes; `LIVE: identical` | |
| 3 Move | moved-from, 2 changes (F2/F4), then No changes; `LIVE: identical` | |
| 4 Update | 1 in-place change; uid kept; ignored keys kept | |
| 5 Drift | detected, repaired, No changes | |
| 6 Import | F1: 1 change, converges after apply | |
| 7 Destroy | 4 destroyed, nothing left | |
