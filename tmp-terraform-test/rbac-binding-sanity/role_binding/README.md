# RoleBinding Suite: PR [#2978](https://github.com/hashicorp/terraform-provider-kubernetes/pull/2978)

PR #2978 moves `kubernetes_role_binding_v1` from SDKv2 to the Plugin Framework and adds `MoveState` from the deprecated
`kubernetes_role_binding`. This suite checks that existing users can upgrade without recreating or rewriting
their RoleBindings, and that the resource still behaves the same afterwards.

The new build for this suite is `/tmp/k8s-sanity/role_binding`, built from the PR head (see the
[top-level README](../README.md#step-0-build-the-provider)).
[released.tfrc](released.tfrc) points Terraform at it.

## Cases

Each case is independent: its own state and its own object names. Do them in any order. Each folder's
README has the full copy-paste steps.

| Case | Starts on | Type at start | Where the migration happens |
| --- | --- | --- | --- |
| [01-unversioned-pre-identity](01-unversioned-pre-identity/README.md) | v2.37.1 | `kubernetes_role_binding` | upgrade (no-op), then `moved` → `_v1` = migration |
| [02-unversioned-post-identity](02-unversioned-post-identity/README.md) | v3.4.0 | `kubernetes_role_binding` | upgrade (no-op), then `moved` → `_v1` = migration |
| [03-versioned-post-identity](03-versioned-post-identity/README.md) | v3.4.0 | `kubernetes_role_binding_v1` | upgrade = migration |

## What `main.tf` Covers (Same in All 3 Cases)

**Exhaustive binding (`role`):** explicit `metadata.name`, `labels`, `annotations`, and
`role_ref` kind `Role`, plus these subjects:

| # | Subject | `api_group` | `namespace` | What it probes |
| --- | --- | --- | --- | --- |
| 1 | User `sanity-user-explicit-group` | `rbac.authorization.k8s.io` | omitted (schema default `default`) | explicit group |
| 2 | User `sanity-user-omitted-group` | omitted | omitted | Optional+Computed api_group |
| 3 | User `sanity-user-empty-group` | `""` | `""` | API defaults `""` → `rbac…` (F2) |
| 4 | User `alice` | omitted | omitted | same name as #5, different kind |
| 5 | Group `alice` | omitted | omitted | positional api_group must not leak between #4/#5 |
| 6 | Group `sanity-group` | `rbac.authorization.k8s.io` | `""` | explicit empty namespace |
| 7 | ServiceAccount `default` | omitted | omitted | SA with defaulted namespace |
| 8 | ServiceAccount `sanity-sa` | `""` | `var.namespace` | SA fully explicit |
| 9 | Group `sanity-group-added-after` | `rbac.authorization.k8s.io` | omitted | only with `phase=after`: update path |

**Alternate binding (`cluster_role`):** `metadata.generate_name`, no labels/annotations, one
ServiceAccount subject, and `role_ref` kind `ClusterRole` (the other kind).

**Provider block:** `ignore_annotations = ["^outside[.]example/keep$"]` and
`ignore_labels = ["^outside[.]example/owner$"]`, used by the ignored-metadata checks.

**`phase=after`:** `managed` label/annotation before → after, label `remove` deleted, label
`added` added, and subject #9 appended. `role_ref` is never changed, because that forces replacement.

## Results from the Last Automated Run

All 3 cases: no recreation and no live-object change in any upgrade/move; update, drift and destroy
behave correctly; 0 leftovers. Known findings that also apply to this suite: **F1, F2, F4**
(see the [top-level README](../README.md#known-findings-f1f6)).
