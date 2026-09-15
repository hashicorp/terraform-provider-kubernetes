# kubernetes_namespace_v1 migration decision

Baseline provider release/commit: **v3.2.1** (`5b3b7cef`, `main`)
Current HCL and complete stored type:

```hcl
resource "kubernetes_namespace_v1" "example" {
  metadata {                       # TypeList, Required, MaxItems 1  -> list(object)
    name          = "..."          # Optional + Computed + ForceNew, ConflictsWith generate_name
    generate_name = "..."          # Optional + ForceNew, ConflictsWith name
    annotations   = { ... }        # Optional TypeMap(string)
    labels        = { ... }        # Optional TypeMap(string)
    # generation, resource_version, uid : Computed
  }
  wait_for_default_service_account = false   # Optional TypeBool, Default false
  timeouts { delete = "5m" }                 # Delete only
}
```

Resource schema version: **0** (no `SchemaVersion`, no `StateUpgraders`).
Identity version: **1** (`resourceIdentitySchemaNonNamespaced`: `name`, `api_version`, `kind`, all
`RequiredForImport`).

Applicable domain rules: `K8S-MIGRATE-001`, `-002`, `-003`, `-004`, `-007`, `-008`, `-013`, `-015`,
`-017`, `-019`, `-020`, `-021`, `-022`, `-024`; `K8S-SCHEMA-001/002/003/005`;
`K8S-CRUD-001/002/003/004/005/006`.

---

## The decision

**SDKv2 persisted a known zero where the Framework will plan `null`. How is that reconciled without
a permanent diff for existing users?**

**Measured 2026-09-15**, by applying with `hashicorp/kubernetes v3.2.1` from the registry against
kind v1.34.0 and reading `terraform.tfstate` — not predicted from the pack:

| Field | SDKv2 flags | Released 3.2.1 state holds | Framework plans for omitted config | Needs conversion |
| --- | --- | --- | --- | --- |
| `metadata.0.generate_name` | Optional | **`""`** — scalar leaves of a block are zero-filled | `null` | **yes** |
| `metadata.0.annotations` | Optional | **`null`** | `null` | no |
| `metadata.0.labels` | Optional | **`null`** | `null` | no |
| `metadata.0.name` | Optional + Computed | API value | API value (`UseStateForUnknown`) | no |

> **This corrects the premise this document was originally written on.** The first draft asserted,
> on the strength of `knowledge/inbox/2026-09-13-measured-null-vs-empty-contract.md`, that SDKv2
> stored `{}` for an omitted `annotations`/`labels`. That note's rule — "TypeMap written by the
> flattener → `{}`" — was measured on `kubernetes_pod_v1` and **does not generalise**: for
> `kubernetes_namespace_v1`, SDKv2 persists `null` for an empty TypeMap whether the flattener handed
> it a nil map or a non-nil empty one. Notably the labels map *is* non-nil-but-empty here —
> Kubernetes ≥1.21 stamps `kubernetes.io/metadata.name` and `removeInternalKeys` deletes it — and it
> still lands in state as `null`.
>
> The design does not change: `generate_name` alone requires the version bump and the upgrader, and
> every conclusion below still holds. But the maps were the expected problem and were not one, and
> the only reason that is known is that the state file was actually read. The measurement takes one
> apply; the argument took considerably longer and was wrong.

| Option | Before/after HCL | Stored type and converters | Ownership/default/output changes | Expected API effects | Feasibility/evidence | Recommendation |
| --- | --- | --- | --- | --- | --- | --- |
| **A — `ListNestedBlock`, zero⇒null in state writers, `SchemaVersion: 1` + `UpgradeState(0)`** | unchanged | `list(object)` unchanged; upgrader rewrites `""`→null (and `{}`→null defensively) | omitted maps become `null` instead of `{}` in *state*; config unchanged | none — state-only conversion, no API call | pod_v1 proved the identical rule against 3.2.1 on kind; upgrader runs because stored version 0 < 1 | **Selected** |
| B — make `annotations`/`labels`/`generate_name` `Optional + Computed` | unchanged | unchanged; no upgrader | removal semantics change: dropping `annotations` from config would no longer delete them | would stop deleting annotations the user removed | violates `K8S-MIGRATE-007`/`-022`; breaks `_noLists` test | Rejected |
| C — zero⇒null in state writers only, keep `SchemaVersion: 0` | unchanged | unchanged; no upgrader | none intended | none | **permadiff**: prior state `generate_name: ""` vs planned `null` on the very first plan after upgrade | Rejected |
| D — defer the resource | n/a | n/a | n/a | n/a | the task is to migrate it | Rejected |

### Why B lost

`Computed` is how SDKv2 says "the provider owns this when config is silent". SDKv2 deliberately did
**not** mark `annotations`/`labels` Computed, and `TestAccKubernetesNamespaceV1_basic` step 5
(`_noLists`) asserts that removing them from config drives them back to empty. Adding `Computed`
would make a removal a no-op — the annotations would stay on the live object forever. That is a
silent, unannounced behaviour change smuggled into a migration (`K8S-MIGRATE-022`), and it is the
direction of breakage `K8S-MIGRATE-007` calls out as easy to miss.

### Why C lost

It is the option that looks correct and fails on first contact. `UpgradeState` is only consulted
when the **stored** version is lower than the current one
(`knowledge/inbox/2026-09-13-singlenestedblock-breaks-state-decode.md`). Keeping version 0 means no
conversion ever runs, so every existing namespace plans `annotations: {} -> null` immediately after
upgrade and `ExpectEmptyPlan` fails. C is A minus the only mechanism that makes A work.

### Residual ambiguity, recorded not hidden (`K8S-MIGRATE-013`)

A user who wrote `annotations = {}` **explicitly** stored exactly the same `null` as a user who
omitted it — SDKv2 cannot tell them apart, and neither can the upgrader. After upgrade the explicit
writer sees one in-place update planning `null -> {}`; it converges in one apply and issues no API
call, because `expandMetadata` only sends a non-empty map. The omitted case is the overwhelmingly
common one and is the one kept diff-free. This is stated in the changelog entry and asserted by
`TestAccKubernetesNamespaceV1_migration_explicitEmptyMaps`, which requires exactly one in-place
update, an unchanged UID, and an empty plan on the step after.

Other viable alternatives, or reasons an option does not apply: `SingleNestedBlock` for `metadata`
is excluded outright — it renders as a protocol Object where SDKv2 renders NestingList, and at an
unchanged version the decode fails before any upgrader can run.

Architecture/helper changes: three functions are exported from package `kubernetes` into a new
`kubernetes/framework_bridge.go` — `IgnoredMetadataKeys`, `FilterMetadataKeys` and `DiffStringMap` —
because `providerMetadata`, `removeInternalKeys`, `removeKeys` and `diffStringMap` are unexported and
a Framework package cannot reach the provider's metadata-ownership policy without them
(`K8S-CRUD-006`). Names and signatures deliberately match the seam already used by the in-flight
`kubernetes_pod_v1` migration so the two reconcile as one file rather than as two divergent helpers
(`K8S-MIGRATE-009`).

Two further changes fall out of being the FIRST resource to leave the SDKv2 server, and are scoped
deliberately rather than incidentally:

- `internal/framework/provider/provider_configure.go` gains the deferred-action gate the SDKv2
  server already had. Without it this migration would silently drop `kubernetes_namespace_v1` off
  the deferral path (`K8S-MIGRATE-022`).
- Six SDKv2 test files used `kubernetes_namespace_v1` as a fixture (41 blocks). They are in
  `package kubernetes`, which cannot import `internal/mux` — that is an import cycle — so
  `K8S-MIGRATE-019`'s mux factory is unavailable to them. Their fixtures move to the deprecated
  `kubernetes_namespace` alias, which stays on SDKv2. The migrated resource's own SDKv2 tests are
  re-pointed at that alias too rather than deleted, so the shared constructor that still serves it
  keeps its coverage.

Required user/module edits: **none**.

Historical versions covered directly: `0 -> 1` only; no other version has ever been published.

Expected Terraform actions and UID/spec/rollout assertions: `no-op` for unchanged configuration;
namespace UID unchanged; no API write during upgrade.

Unverified cases and rollback/downgrade limits: downgrading to 3.2.1 after upgrading is not
supported — the stored `SchemaVersion: 1` is rejected by the older provider. This is the normal
consequence of any schema version bump and matches how the provider has bumped versions before.

## Scope and any required confirmation

Selected option and exact resource/fields: **A**, for `kubernetes_namespace_v1` only —
`metadata.annotations`, `metadata.labels`, `metadata.generate_name`.

Existing authorization or new confirmation: within the authorized migration scope. No HCL change,
no stored-type change, no API-payload change, so `K8S-MIGRATE-023` needs no separate confirmation.

Permitted behavior/API changes: none beyond the state-only `{}`/`""` → `null` normalisation
described above, which is the mechanism of compatibility rather than a change to it.

---

## Adversarial critique and responses

Reviewed by `tfp-critic` against the pinned checkout and module cache. Eight blocking findings were
raised; all eight were independently re-verified before being accepted, and all eight are addressed.

| # | Finding | Verified how | Response |
| --- | --- | --- | --- |
| 1 | Deregistering `kubernetes_namespace_v1` breaks 6 sibling SDKv2 test files, not just its own — `testAccProviderFactories` (`kubernetes/provider_test.go:36`) is SDKv2-only | Counted 41 `resource "kubernetes_namespace_v1"` fixture blocks across 7 files | **Accepted.** A mux-backed `testAccProtoV6ProviderFactories` is added to `kubernetes/provider_test.go`, and only the `TestCase`s whose configs use the fixture are switched to it. `K8S-MIGRATE-019` names the mux factory as the correct default here. |
| 2 | The Framework server has no deferred-action support, so the migration silently regresses it and breaks `kubernetes/test-dfa/deferred_actions_test.go:46` | `grep Deferred internal/framework/` returns nothing; SDKv2 handles it centrally at `kubernetes/provider.go:365-370` | **Accepted.** `provider_configure.go` now mirrors the SDKv2 gate using `provider.Deferred{Reason: provider.DeferredReasonProviderConfigUnknown}`. Without this the migration would be a silent behaviour change (`K8S-MIGRATE-022`). |
| 3 | `id` was never mentioned; SDKv2 injects it, it is published, and it is referenced by in-repo configs | `docs/resources/namespace_v1.md:26`; `_examples/deferred-actions/workspace.tf:16`; `kubernetes/test-dfa/config-basic/workspace.tf:16` | **Accepted.** `id` is declared `Optional + Computed` with `UseStateForUnknown()` and set to the namespace name in Create, Read, Import and the upgrader. |
| 4 | `RequiresReplace()` before `UseStateForUnknown()` on `metadata.name` recreates the namespace of every `generate_name` user on every plan | Read the pinned v1.15.1 modifier loop: `attribute_plan_modification.go:2391` runs modifiers in declaration order, feeds `PlanValue` forward at :2419, and latches replacement at :2422-2424 where no later modifier can clear it; `MarkComputedNilsAsUnknown` runs first (`server_planresourcechange.go:252` before `:293`) | **Accepted — this was a real defect.** Order is now `UseStateForUnknown(), RequiresReplace()`. A second config step asserting `ExpectEmptyPlan` is added to the `generatedName` test, which previously applied only once and could never have caught it. |
| 5 | "Converges in one apply" was false: `Read` has no plan, so an explicit `annotations = {}` would permadiff | Reasoned through refresh → plan → update → refresh | **Accepted.** The empty-value rule is now *prior-value-aware*: an empty live value keeps the prior **known** empty when there is one (plan on Create/Update, state on Read) and becomes null otherwise. Only the one-time upgrader nulls empties unconditionally. |
| 6 | Update was unspecified; the obvious implementations either cannot express removal or wipe controller-owned metadata (`K8S-CRUD-006`) | `expandMetadata` (`structures.go:50-56`) skips `len == 0`; SDKv2 Update uses `patchMetadata` → `diffStringMap`, which touches only changed keys | **Accepted.** `diffStringMap` is exported as `DiffStringMap` and Update builds the same JSON Patch from state→plan, so unmanaged and ignored keys are untouched by construction. |
| 7 | Import by ID string was dropped; SDKv2 supports both ID and identity | `resourceidentity.go:85-103` (`if rd.Id() != ""` first); documented at `docs/resources/namespace_v1.md:84` | **Accepted.** `ImportState` handles `req.ID` first, then `req.Identity`. |
| 8 | No released-to-local upgrade test was specified (`K8S-MIGRATE-015`, `-017`) | Absent from the record | **Accepted.** A migration test pinned to exactly `3.2.1` is added, covering omitted maps, explicit `annotations = {}`, `generate_name`, a configured `timeouts`, and an output referencing `.id`. |

Suggestions also taken: the create waiter's bound is SDKv2's **20-minute system default**
(`resource_data.go:552-582` — `Timeouts.Create` is not declared, so `d.Timeout` falls through), and
that exact default is used rather than inventing one; `timeouts` is set to a typed
`types.ObjectNull` on import; identity is set in Update as well as Create and Read; `metadata` is
written through one whole-model `State.Set` rather than an indexed path walk.

Not adopted: `wait_for_default_service_account` is **not** force-set on import. SDKv2 zero-fills it
there, but it is already in the pre-existing `ImportStateVerifyIgnore` list, so writing it would be
a behaviour change rather than parity — and `K8S-MIGRATE-020` only forbids *growing* that list,
which this does not.

---

## Live verification (kind v1.34.0, 2026-09-15)

Cluster: disposable kind `tfp-k8s`, Kubernetes v1.34.0, via `~/.kube/cluster-kind/env.sh`.

| Test | Result |
| --- | --- |
| `TestAccKubernetesNamespaceV1_basic` | PASS — 6 steps incl. import, add/shrink/remove annotations and labels |
| `TestAccKubernetesNamespaceV1_explicitEmptyMaps` | PASS |
| `TestAccKubernetesNamespaceV1_identity` | PASS (after the import fix below) |
| `TestAccKubernetesNamespaceV1_default_service_account` | PASS |
| `TestAccKubernetesNamespaceV1_generatedName` | PASS — incl. the second-plan `ExpectEmptyPlan` guard |
| `TestAccKubernetesNamespaceV1_withSpecialCharacters` | PASS |
| `TestAccKubernetesNamespaceV1_deleteTimeout` | PASS |
| `TestAccKubernetesNamespaceV1_validation` | PASS — all 5 ported validators + required block + ConflictsWith |
| `TestAccKubernetesNamespaceV1_migration_*` (5 tests) | PASS against genuine registry `v3.2.1` |

No namespaces leaked; `kubectl get ns` is back to the five cluster defaults.

### Two defects the cluster found that inspection did not

**1. Identity import planned an update.** `TestAccKubernetesNamespaceV1_identity` failed with

    Step 2/2 error running import: importing resource kubernetes_namespace_v1.test:
    expected a no-op import operation, got ["update"] action with plan

`ImportState` left `wait_for_default_service_account` null; the schema `Default` then planned
`null -> false`. SDKv2 zero-filled it, because `ResourceData.State()` wrote `d.Get` for every
top-level field. The reviewer raised exactly this and it was declined on the grounds that the
attribute was already in `ImportStateVerifyIgnore` — **that reasoning was wrong**:
`ImportStateVerifyIgnore` only relaxes the legacy `ImportStateVerify` comparison and has no effect on
the `import`-block path, which requires a genuine no-op plan. Fixed by setting it to `false` on
import, which is parity with SDKv2 rather than a behaviour change.

**2. The released baseline was not the released provider.** The first run of the migration tests
passed while silently testing nothing, because `~/.terraformrc` on this machine carries

    provider_installation {
      dev_overrides {
        "hashicorp/kubernetes" = "/Users/prabuddha/codes/terraform-provider-kubernetes/bin"
      }
      direct {}
    }

`ExternalProviders` performs a real `terraform init`, which honours that override, so the "3.2.1"
step was served by a 2026-07-03 build of another branch. Every migration result above was re-run with
`TF_CLI_CONFIG_FILE` pointed at a clean config; `terraform init` then reports
`Installed hashicorp/kubernetes v3.2.1 (signed by HashiCorp)`. The developer's `~/.terraformrc` was
not modified.

### Manual released-to-local upgrade, with plan JSON

Created with genuine 3.2.1, then planned against a `dev_overrides` build of this worktree
(`Provider development overrides are in effect` confirmed in the transcript):

    resource_changes: NONE (no-op)
    resource_drift  : NONE
    schema_version  : 0 -> 1
    uid             : 6a0ade65-29b5-40cd-b776-fe0bb736f4c2  (unchanged; matches the live namespace)
    generate_name   : ""  -> null
    annotations     : null -> null
    labels          : null -> null
    outputs         : {'ns_id': 'tf-upgrade-probe'}

`resource_drift` is checked separately from `resource_changes` because an empty action plan can
still sit on top of a refresh-time normalisation. There is none.
