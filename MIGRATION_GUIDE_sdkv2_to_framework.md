# Migrating a resource from SDKv2 to the Plugin Framework

Written from the `kubernetes_namespace_v1` migration (PR #2979), generalised to any resource in
this provider. Everything here was verified against a live KinD cluster or read out of the
pinned dependency source — where a claim was only reasoned about, it says so.

**This is the converged reference.** Nine separate documents were written during the migration,
one per working session, to keep each session's context small. They all still exist and are
listed in Appendix A; nothing here replaces them as the record. Read this file to *do* a
migration or to revise how one works. Go back to an original only when you need the full
experiment log behind a claim.

---

## How to use this document

| Section | Read it for |
| --- | --- |
| §1 Pre-flight | Deciding whether a resource can be migrated at all, before any work |
| §2 Concepts | The five ideas that make the rest make sense. Do not skip. |
| §3 What breaks | The failure modes, each with the symptom you will actually see |
| §4 Steps | The order to work in, with the trap at each step |
| §5 `common/` | What is already shared, and how to use it correctly |
| §6 Filtering spec | Metadata parity — the behaviour you must reproduce exactly |
| §7 Testing | What to test, and what the harness cannot reach |
| §8 Bug catalogue | Seven real bugs, each reduced to a rule |
| §9 Review | The checks a finished migration must pass |
| §10 Lessons | The generalisable ones, for a tutorial |

Terminology: **SDKv2** is `terraform-plugin-sdk/v2`; **framework** is
`terraform-plugin-framework`. "The last SDKv2 release" means the released provider version that
still served the resource from SDKv2 — for namespace, 3.2.1.

---

## 1. Pre-flight: can this resource be migrated at all?

**Check this before anything else.** Some resources cannot be ported without a breaking change,
and finding out at step 7 is expensive.

The blocker is an SDKv2 block declared `Optional: true, Computed: true` — specifically
`TypeList`/`TypeSet` + `Optional` + `Computed` + `Elem: &schema.Resource{}`. (`Elem:
&schema.Schema{}` is a scalar list and is fine. Plain Optional+Computed *attributes* are fine.)

Port one as a framework block and you get:

```
Error: Provider produced inconsistent result after apply
.<block>: block count changed from 0 to 1
```

The API returns defaults for a block the user omitted, `Read` writes them to state, config has
0 blocks and state has 1. SDKv2 survives this two ways and the framework has neither:

1. `Computed: true` on a block is a **purely SDK-internal flag**. `schemaMap.CoreConfigSchema()`
   routes Optional+Computed with an `Elem: *Resource` down the block path
   (`helper/schema/core_schema.go:96-118`), and `coreConfigSchemaBlock()` (`:183`) builds a
   `configschema.NestedBlock` that has **no Computed field to assign** — so Terraform core never
   learns the block is computed. Internally it means "config empty → no diff, keep prior state".
2. SDKv2 sets `UnsafeToUseLegacyTypeSystem = true` on every plan and apply
   (`grpc_provider.go:1015-1024`, `:1602-1611`), demoting the resulting consistency error to a
   log line. The framework never sets it — verified absent through v1.19.0. There is no opt-in.

These resources have been violating the apply-consistency contract all along; migration only
stops forgiving it. The root cause is protocol v5, not the SDK: nested attributes arrived in v6.

**Detection:**

```bash
grep -rn "Type: *schema.TypeList," -A8 --include="*.go" kubernetes/ \
  | grep -B6 "Computed: *true" | grep -A1 "Optional: *true"
```

`scripts/find-optional-computed-blocks.py` is the precise scanner.
`MIGRATION_TRACKER.md` carries the per-resource verdict: **13 resources are blocked**, mostly
workloads, because `schema_container.go` (`resources`) and `schema_pod_spec.go`
(`readiness_gate`, `image_pull_secrets`) fan out to eight implementations each via
`podSpecFields` ← `podTemplateFields` ← `jobSpecFields`. Flat resources — namespace, config map,
secret, RBAC, storage class — are unaffected. Data sources are never blocked: they have no
apply, so the error cannot arise.

**Cheaper confirmation, still on SDKv2:** set `EnableLegacyTypeSystemPlanErrors` and
`EnableLegacyTypeSystemApplyErrors` on the `schema.Resource`
(`helper/schema/resource.go:605-648`) and run that resource's acceptance suite. Whatever fails
is what will block the port. Schema-shape failures need the migration; data failures are logic
bugs fixable in either SDK.

If a resource is blocked, the options are: convert the block to
`ListNestedAttribute{Optional, Computed}` (works, but changes user HCL from `block {}` to
`block = [{}]` — a major-version change), filter the API defaults out in `Read`, split into two
blocks, or leave it on SDKv2.

---

## 2. The concepts you must hold before writing code

### 2.1 null, empty, and unknown are three different things

SDKv2's type system has two of these; the framework has three. Almost every migration bug lives
in the gap.

- **null** — the attribute has no value. In Go models, a nil map; in `types.Map`, `IsNull()`.
- **empty** — the attribute has a value which is a zero-length collection: `{}`.
- **unknown** — the value will be decided during apply. Only framework types can carry it.

SDKv2 collapses the first two and has no third. Two consequences you will meet:

- **SDKv2 stores `""` where the framework stores null** for an unset optional string. That is
  not cosmetic: see §3.2.
- **SDKv2 wrote `null` for a config that said `{}`**, because its type system cannot represent
  the difference. After migration the framework plans the config's `{}` — one in-place update.

**Use framework collection types in models (`types.Map`, `types.List`), never Go built-ins like
`map[string]types.String`.** Only framework types can carry an unknown, and it is *required* for
`Optional + Computed` collections — a Go map fails to decode an unknown plan value
(`internal/reflect/into.go:116-124`). One rule keeps every model in step with its schema.

A Go nil map and a Terraform null map do convert to each other cleanly
(`internal/reflect/map.go:104`), which is why the Go-map shortcut looks like it
works right up until an `Optional+Computed` collection appears.

### 2.2 Schema shape is a state contract

The framework schema must produce the **same state wire shape** as the SDKv2 one, or every
existing state file fails to decode.

SDKv2 `TypeList` + `MaxItems: 1` serialises as `"metadata": [ {...} ]`. The framework
equivalent is **`schema.ListNestedBlock`**, not `SingleNestedBlock` and not
`SingleNestedAttribute`. `SingleNestedAttribute` also changes user HCL from `metadata { … }` to
`metadata = { … }`, which is a breaking config change.

**Confirm the shape against a real SDKv2 `terraform.tfstate`, never from the HCL syntax.**

Framework blocks have no `Required` field. Required-ness comes from validators:

```go
Validators: []validator.List{
    listvalidator.IsRequired(),   // this is what translates SDKv2 Required: true
    listvalidator.SizeAtMost(1),  // MaxItems: 1
}
```

`SizeAtLeast(1)` alone is **not** enough: size validators skip null, so an omitted block passes.
`IsRequired()` is the one that fires. And because block required-ness is invisible to
`tfplugindocs`, the generated page will show `metadata` as Optional unless the description says
otherwise — hence "Exactly one metadata block is required." in the shared description.

### 2.3 Apply-time consistency decides which method filters

Terraform requires that state after apply equals the planned value. That splits the CRUD
methods by source of truth:

| Method | Source of truth for user-controlled fields | Filter the API response? |
| --- | --- | --- |
| `Create` | the **plan** | no — you are echoing config |
| `Read` | the **API response** | yes — both rules in §6 |
| `Update` | the **plan** (same constraint as Create) | no |

This is the single biggest structural difference from SDKv2, where `Create` ends by calling
`Read` and everything filters in one place. In the framework, `Create` that writes a filtered
API response instead of the plan produces `Provider produced inconsistent result after apply`.

### 2.4 Resource identity

Identity is a stable, provider-defined key for an object, separate from `id`. For this provider
it is `{api_version, kind, name}` — plus `namespace` for namespaced resources.

- SDKv2 declares it via `resourceIdentitySchemaNonNamespaced()`
  (`kubernetes/resourceidentity.go:39`), **at `Version: 1`**.
- The framework declares `identityschema.Schema{Version: 1, …}`. **Match the SDKv2 version.** A
  lower version is a downgrade and Terraform rejects it.
- `client-go` typed clients clear `TypeMeta`, so `apiVersion` and `kind` must be hardcoded
  constants, not read off the response object.

### 2.5 Two upgrade paths, and both must be handled

Terraform compares stored versions against declared versions and calls an upgrader whenever they
differ — **even when nothing is stored**.

- **`UpgradeResourceState`** — fires when `SchemaVersion` differs. If the wire shape is
  preserved (§2.2), the version stays 0 and there is nothing to write.
- **`UpgradeResourceIdentity`** — fires when the stored `identity_schema_version` differs from
  the declared one. State written by provider ≤ 2.37.x has `identity_schema_version: 0` and *no*
  identity, because identity shipped in 2.38.0. Terraform still asks for the upgrade.

SDKv2 answers "nothing to upgrade" generically (`helper/schema/grpc_provider.go`, guarded on
`len(req.RawIdentity.JSON) > 0`). **The framework requires an explicit implementation** and
rejects an empty response. Two traps, both cost a session each:

- **Do not set `PriorSchema`** on the identity upgrader. The framework decodes `RawIdentity`
  against it *before* calling you and fails with `RawState had no JSON or flatmap data set` when
  there is nothing to decode — which is the normal case. Read `req.RawIdentity` directly.
- **Do not return an empty response.** The framework answers `Missing Upgraded Resource
  Identity`. Return an object with **every attribute null** when no identity is stored; framework
  **v1.16.1+** then lets `Read` populate it (earlier versions reject any change to a non-null
  identity, giving `Unexpected Identity Change`).

### 2.6 `moved` blocks need `MoveState`, and it is not automatic

Cross-type `moved` (`kubernetes_namespace` → `kubernetes_namespace_v1`) requires the *target*
resource to implement `ResourceWithMoveState`. SDKv2 cannot, which is why the whole migration
programme exists. Nothing is copied automatically: the mover must set both `TargetState` and
`TargetIdentity`.

---

## 3. The five things that break a migration

| # | Breaks | Symptom you will see |
| --- | --- | --- |
| 1 | Wrong schema shape | state fails to decode, or users must rewrite HCL |
| 2 | SDKv2's `""` vs framework null on a `ForceNew` string | **destroy and recreate** on `plan -refresh=false` |
| 3 | `Computed` + `RequiresReplace` without `UseStateForUnknown` | **destroy and recreate** on an unrelated edit |
| 4 | Filtering in the wrong method | `Provider produced inconsistent result after apply` |
| 5 | The registration flip not being atomic | provider fails to start |

§2.2 covers 1, §2.3 covers 4. The other three:

### 3.1 The registration flip is atomic

The mux refuses duplicate type names, so the framework registration and the SDKv2
`ResourcesMap` deletion **must be in one commit**. Delete the SDKv2 line; do not comment it out.

Leave the deprecated alias (`kubernetes_namespace`) on SDKv2 — it is a different type name, so
there is no conflict, and removing it is a breaking change that belongs in a major release. Two
practical consequences:

- The alias now has no test coverage unless you give it some. Re-point the SDKv2 test file at it.
- **Other SDKv2 acceptance tests that use the migrated type as a fixture will break**, because
  they build their provider from the SDKv2-only factory. For namespace that was 8 files and ~90
  resource blocks. Grep `kubernetes/*_test.go` for the type name. Many such tests are skipped on
  KinD (`skipIfNotRunningInMinikube`, `skipIfNotRunningInGke`), so CI stays green while they are
  broken.

### 3.2 `""` vs null on a `ForceNew` string destroys resources

SDKv2 stores an unset `generate_name` as `""`. The framework plans it as null.
`stringplanmodifier.RequiresReplace()` compares `PlanValue.Equal(StateValue)`; null ≠ `""`, so
it forces replacement.

A normal `terraform plan` is safe, because refresh runs `Read` first and `Read` normalises `""`
to null before the diff. **`terraform plan -refresh=false` is not** — and it is common in CI:

```
Plan: 1 to add, 0 to change, 1 to destroy.
```

The rendered plan does not show why. `generate_name` appears as an unchanged hidden attribute;
only `terraform show -json` exposes `replace_paths = [["metadata",0,"generate_name"]]`,
`before: ""`, `after: null`.

This hits the *plain provider upgrade* path — no `moved` block, config unchanged — which is
everyone already on the `_v1` name. Fixing it in `MoveState` is not enough.

**Fix:** an equivalence exemption on the plan modifier.

```go
stringplanmodifier.RequiresReplaceIf(func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
    // SDKv2 stores an unset generate_name as "", the framework as null.
    sdkv2Unset := req.StateValue.ValueString() == "" && req.PlanValue.IsNull()
    resp.RequiresReplace = !sdkv2Unset
}, "…", "…")
```

**Generalise it: any SDKv2 `Optional` + `ForceNew` string whose unset value was stored as `""`
carries this hazard once migrated.** It cannot be caught by an acceptance test (§7).

### 3.3 `Computed` + `RequiresReplace` needs `UseStateForUnknown`

On any plan that is not already a no-op, the framework marks every `Computed` attribute with a
null config value as **unknown** (`fwserver/server_planresourcechange.go:252`, defined at `:414`,
`MarkComputedNilsAsUnknown`). `RequiresReplace` treats unknown as changed, so an edit to an
unrelated attribute plans a replacement.

For `metadata.name` — `Optional + Computed` because `generate_name` fills it in — that means
adding one label to a `generate_name` namespace planned `[delete create]`. Data loss.

SDKv2 does the opposite: its diff skips `Computed` attributes without a config value
(`helper/schema/schema.go`, `diffString`: `if removed && schema.Computed { return nil }`).

Note the second, cosmetic consequence: migrated resources show more `(known after apply)` than
SDKv2 did. Decide per attribute rather than reflexively suppressing:

- `uid` — `UseStateForUnknown()`. Immutable for an object's lifetime, safe for every resource.
- `resource_version` — **leave unknown.** It changes on every write; planning the prior value
  causes `Provider produced inconsistent result` on metadata updates.
- `generation` — **leave unknown.** Stable for namespaces, but it increases on spec changes for
  resources that have a spec, and the schema is shared.

---

## 4. The migration, step by step

| Step | Do | The trap |
| --- | --- | --- |
| 1 | Prove the SDKv2 acceptance tests pass **today**, on your cluster | Skipped tests look like passes |
| 2 | Audit SDKv2 behaviour (§6) and write the rules down before porting | Deriving rules from the new code instead of the old |
| 3 | Skeleton: package, struct, interface assertions, `Configure` | See below — `Configure` is a real trap |
| 4 | Model, derived from the schema | A field with no matching schema attribute is a **decode error at plan/apply time**, not a zero value — and so is the reverse |
| 5 | Schema (§2.2) | `SingleNestedBlock`; missing `IsRequired()` |
| 6 | Helpers + validators — **reuse `common/` (§5)** | Re-deriving filtering instead of reusing `kubernetes.RemoveInternalKeys` / `RemoveKeys` |
| 7 | CRUD (§2.3) | `Create` early-returning without `State.Set` orphans the resource |
| 8 | Identity, import, `MoveState`, `UpgradeIdentity`, and tests written to **fail first** | §2.5's two traps |
| 9 | The flip — one atomic commit (§3.1) | Other resources' test fixtures |
| 10 | Regressions, docs, changelog | Docs are generated from `templates/` |

**Step 3, `Configure`.** The mux configures the SDKv2 provider independently, and its meta is
not populated until that happens — so resolve it per-request, not in `Configure`. And store the
function, not its result:

```go
// ProviderData holds a func() any. Go function types are invariant, so asserting to
// func() kubernetes.KubeClientsets compiles and panics at runtime. Assert the *result*.
sdkv2Meta, ok := req.ProviderData.(func() any)
```

**Step 7, CRUD specifics.** `Create` must `State.Set` on every path. Timeouts come from
`plan.Timeouts`, not a bare `time.ParseDuration` on a number. Read from `req.Plan`, not
`req.Config`. `Update` builds JSON Patch ops via `common.MetadataPatchOps` (§5). `Delete`
mirrors SDKv2's termination wait — and note `retry.StateChangeConf` signals "gone" with an
**empty** `Target` slice, `[]string{}`, not `[]string{""}` (`helper/retry/state.go:126` consults
only the length).

**Step 10, docs.** `templates/resources/<name>.md.tmpl` is authoritative; `docs/` is generated
by tfplugindocs. Regenerating regenerates *everything*, so revert unrelated churn. Behaviour
changes belong in the resource docs; keep the changelog entry short. `make docs-lint` needs
Docker, and its link-check step needs a TTY.

---

## 5. Shared code: `internal/framework/provider/common`

Reuse this rather than re-declaring metadata handling per resource. It is the framework
counterpart of SDKv2's `schema_metadata.go` + `structures.go`.

| File | Provides |
| --- | --- |
| `metadata_schema.go` | `MetadataSchema`, `NamespacedMetadataSchema` |
| `metadata_model.go` | `MetadataBase`, `MetadataModel`, `NamespacedMetadataModel` |
| `metadata_ops.go` | Base, generated-name, and namespaced expand/flatten helpers; `BaseMetadataPatchOps`, `MetadataPatchOps`, `ExpandMapForPatch` |
| `identity.go` | Cluster-scoped and namespaced identity models, schemas, and version-0 upgraders |
| `id.go` | `BuildID`, `ParseID` for namespaced IDs |
| `validators.go` | `LabelsValidator`, `AnnotationsValidator`, `DNSSubdomainNameValidator`, `DNSLabelPrefixValidator` |

### `generate_name` is a schema variant — pick the right one

```go
common.MetadataSchema("namespace", true)   // resource: generate_name present
common.MetadataSchema("config_map", false) // data source: generate_name absent
```

The flag mirrors SDKv2's `metadataSchema(objectName, generatableName)`, so a migrated call site
is a literal transcription. It is **not** a style choice: it decides what appears in state, and
SDKv2 adds the `name` ↔ `generate_name` `ConflictsWith` pair only in the `generatableName`
branch. `MetadataBase` holds the six shared fields. `MetadataModel` embeds it and adds
`GenerateName`; `NamespacedMetadataModel` embeds `MetadataModel` and adds `Namespace`.
Framework promotes these fields, so all attributes remain flat inside the metadata object.

`ExpandBaseMetadata` and `FlattenBaseMetadata` operate on a single `MetadataBase`.
`ExpandMetadata` and `FlattenMetadata` handle the metadata list and `GenerateName`, delegating
shared fields to those base helpers. The namespaced wrappers add `Namespace` in the same way.
`BaseMetadataPatchOps` handles the shared mutable maps; `MetadataPatchOps` delegates to it.

Identity upgrader helpers take **kind first, API version second**:
`UpgradeIdentity("Namespace", "v1")` and
`UpgradeNamespacedIdentity("Role", "rbac.authorization.k8s.io/v1")`.

**Embedded fields must have no `tfsdk` tag, and promoted tags must not collide.** These are
reflection-time errors, covered by schema-derived model round-trip tests. Go keyed literals
must name the embedded type, although ordinary field reads and assignments stay unchanged:

```go
metadata := common.MetadataModel{
  MetadataBase: common.MetadataBase{Name: types.StringValue("example")},
}
metadata.Name = types.StringValue("updated")
```

Use `NamespacedMetadataSchema(objectName, true)` with `NamespacedMetadataModel` and the
namespaced expand/flatten helpers. The false schema variant needs a model containing
`MetadataBase` and `Namespace`, without `GenerateName`; no exported model exists for it yet.

**RBAC needs additional schema composition.** SDKv2's `metadataSchemaRBAC` replaces both name
validators with Kubernetes path-segment validation, allowing names such as `system:reader`.
The generic common schemas use DNS validation and are not drop-in RBAC replacements. Also
compare map defaults and null-value handling before adopting them: common rejects null map
values, whereas another migration may deliberately preserve them.

### `MetadataPatchOps` — and why the naive version is destructive

`Update` must not diff a metadata map that neither side manages:

```go
oldAnnotations, newAnnotations := ExpandMapForPatch(state.Annotations), ExpandMapForPatch(plan.Annotations)
if len(oldAnnotations) > 0 || len(newAnnotations) > 0 {
    ops = append(ops, kubernetes.DiffStringMap(pathPrefix+"annotations", oldAnnotations, newAnnotations)...)
}
```

Emptiness is measured **after expansion**, so null and `{}` count alike: they differ in
Terraform, but neither names a key to change in Kubernetes. See bug 7 in §8 for what happens
without this. When no operations result, skip PATCH and use GET to resolve computed fields
while preserving the plan's maps. Provider-only settings can change without requiring
Kubernetes patch permission.

---

## 6. Behaviour parity: the metadata filtering spec

`Read` must reproduce SDKv2's filtering **exactly**, or users see phantom drift. Reuse
`kubernetes.RemoveInternalKeys` and `kubernetes.RemoveKeys` rather than reimplementing — both
are exported and both encode quirks you would not reinvent.

### Rule 1 — control-plane keys (`RemoveInternalKeys`)

Drop a key when it is internal **and not declared in the user's config**
(`structures.go:152-158`). Internal means (`isInternalKey`, `:175`): parse the key as
`"//" + key` and take the hostname; drop if that hostname ends in `kubernetes.io`, or if the key
contains `deprecated.daemonset.template.generation`.

Two hostnames are explicitly exempted, and the reason matters:

- `app.kubernetes.io/*` — the recommended label set (`name`, `instance`, `version`, `component`,
  `part-of`, `managed-by`). Users and Helm set these deliberately.
- `service.beta.kubernetes.io/*` — cloud load-balancer configuration (NLB type, SSL cert ARNs).
  Functional configuration the user authored.

### Rule 2 — user-configured ignore lists (`RemoveKeys`)

Drop a key when it matches a provider-level `ignore_labels` / `ignore_annotations` pattern
**and is not declared in config** (`:162-168`).

**The match is `regexp.MatchString` and therefore unanchored.** `ignore_labels = ["env"]` also
catches `environment`, `my-env-label`, `dev-environment-2`. This is arguably a bug, but
anchoring it would silently change which keys existing users' patterns filter. **Replicate it
exactly; do not anchor during migration.**

### What is *not* filtered

Non-internal keys absent from config are **kept** — that is what makes drift detection work.
Only the two rules above remove anything.

### The config-membership escape, and the null/empty rule

Both rules share `&& !isKeyInMap(k, d)`: a key the user declared in config survives filtering.
That is why `filterMetadataMap` needs the prior/declared state, not just the API response.

And when filtering removes every key: **return null if prior state was null, `{}` if prior state
was an explicit empty map.** Prior state only decides which server keys are exempt — it can
never *add* keys the server lacks.

---

## 7. Testing

Full detail in `MIGRATION_TESTS_namespace_v1.md`. The essentials:

**A migration test** is two steps with byte-identical config: step 1 `ExternalProviders` pinned
to a released SDKv2 version, step 2 `ProtoV6ProviderFactories` with a plan check. Set provider
fields per-step, never on the `TestCase`.

**Pin two SDKv2 versions, not one** — post-identity (3.2.1) and pre-identity (2.37.1). Until the
pre-identity tests existed, `UpgradeIdentity` had zero coverage.

**Use `PlanOnly: true` for the framework step.** plugin-testing skips apply and runs a
*non-refresh* plan against the raw SDKv2 state. A normal apply step refreshes first, hiding
state-shape mismatches — including §3.2 entirely.

**Derive scenarios from the config space, not the code:** naming (`name` | `generate_name`) ×
maps (absent | `{}` | populated) × timeouts, plus a declared-internal-key case. Assert the
*specific* action — `ExpectResourceAction(..., ResourceActionUpdate)` — never just "some plan".

**`ExpectEmptyPlan` cannot prove the absence of an unknown-value bug.** The assertion passing is
the same condition as the bug not firing. Every `Optional+Computed` / `generate_name` path needs
an **update step**.

**A filtering test needs a key the control plane will not restore.** `kubernetes.io/metadata.name`
is re-added by the API server on every write, healing the damage before any assertion runs — that
is how bug 7 hid for weeks. Use `example.kubernetes.io/owner` or an `ignore_*` entry, and assert
against the **live object**, since state and plan say nothing about a filtered key.

**The failing-test beat:** break the code on purpose and confirm the *right* test fails. A test
never seen failing has not shown that it tests anything.

Three behaviours the harness cannot reach, and what to use instead:

| Behaviour | Why | Use |
| --- | --- | --- |
| `""` → null replacement on `-refresh=false` | no way to skip refresh before a plan | unit test the plan modifier, plus a guard test that plain `RequiresReplace` still replaces |
| identity after `terraform import <id>` | the identity-aware import kind *supplies* identity rather than reading it back | manual `dev_overrides` run |
| a filtered key surviving an update | invisible to state and plan | live-object assertion via the clientset |

`mux.MuxServer` works as a provider factory, which makes HCL provider settings such as
`ignore_annotations` acceptance-testable.

---

## 8. The bug catalogue

Seven real bugs from one resource. Each reduces to a rule worth carrying forward.

| # | Bug | Rule it yields |
| --- | --- | --- |
| 1 | Import left `wait_for_default_service_account` null, so the next plan showed a phantom update | Apply schema defaults in **`Read`**, not `ImportState` — `Read` runs after import, refresh *and* state upgrade |
| 2 | Adding the **first** label wipes keys managed outside Terraform | Still open, provider-wide (39 SDKv2 resources). See below. |
| 3 | `annotations = { a = null }` failed at apply with a misleading "always an error in the provider" | Validators must reject null map **values**, not just keys |
| 4 | `defaultDeleteTimeout` shipped as 10 seconds under a comment claiming 5 minutes | **A default every test overrides is a default no test covers** |
| 5 | State from ≤ 2.37.x failed its first plan | §2.5 — `UpgradeIdentity` + framework v1.16.1 |
| 6 | `-refresh=false` destroyed every upgraded resource | §3.2 — `""` vs null on a `ForceNew` string |
| 7 | A label-only update deleted an annotation owned by another controller, while the plan said `No changes` | §5 — **port the caller's guard, not just the callee** |

**Bug 7 in one paragraph,** because it is the subtlest. `Update` diffed both metadata maps on
every call. With annotations unmanaged, both sides expanded to `{}` and `DiffStringMap`'s
empty-prior branch emitted `{"op":"add","path":"/metadata/annotations","value":{}}` — and JSON
Patch `add` **replaces** an existing member. State holds no keys for a map both when the user
manages none and when they were filtered out, and the provider cannot tell those apart. SDKv2
never hit it because `patchMetadata` diffs a map only when `d.HasChange` says so; the migration
dropped that calling condition. The defect was in the caller, not the shared helper.

**Bug 2 is open and deliberately out of scope for any single migration.** `DiffStringMap`
(`kubernetes/patch_operations.go:19-25`) emits a whole-object `add` when the prior map is empty,
which replaces whatever the server holds. It affects the 39 SDKv2 resources calling
`patchMetadata` (`structures.go:71`) and the framework equally, so it is **not** a migration
regression — fix it provider-wide. Candidate fixes: a JSON **merge patch** for these two maps
(touches only listed keys, creates the parent if absent, deletes with `"key": null`), or
pre-reading the object to choose between whole-map and per-key adds. Per-key adds alone are not
enough: `add /metadata/labels/env` returns 422 when the server has no `labels` object. Full
write-up and a reproduce recipe: `FIX_namespace_v1_metadata_patch_ops.md`.

---

## 9. Review: what a finished migration must satisfy

`MIGRATION_PR_REVIEW_CHECKLIST.md` is the full, reusable checklist — use it when reviewing
someone else's migration PR. It stays separate because it is written for a reviewer, not a
migrator. The highest-value checks, condensed:

**Schema** — wire shape matches a real SDKv2 state file; `IsRequired()` present on required
blocks; model collections are framework types; `UseStateForUnknown` on every
`Computed + RequiresReplace` attribute.

**Identity and import** — identity schema version matches SDKv2's; `UpgradeIdentity`
implemented; `ImportStateVerify: true` with no ignore entries inherited without a reason. Note
`ImportStateVerify` **cannot see null vs empty** — it drops every `.#`/`.%` key whose value is
`"0"` before comparing (`testing_new_import_state.go:333-339`) — so add an import-block step with
`ExpectResourceAction(..., ResourceActionNoop)`.

**MoveState** — guards on `SourceTypeName`, `SourceProviderAddress` (ignoring hostname) and
`SourceSchemaVersion`; a non-matching mover returns neither state nor diagnostics. Sets
`TargetState` **and** `TargetIdentity`. **Never guards on `SourceIdentitySchemaVersion`** — it
always arrives as `0`, because terraform-plugin-go drops it
(`tfprotov6/internal/fromproto/resource.go:127-135`) while the framework faithfully forwards
that zero; a `!= 1` guard rejects every legitimate move. `req.SourceIdentity` is a raw
`*tfprotov6.RawState`, so rebuild identity rather than decoding it.

**Tests** — every deleted SDKv2 test ported or deliberately dropped; SDKv2 tests that were *not*
deleted no longer reference the migrated type; the deprecated alias keeps local coverage;
`CheckDestroy` everywhere; a downgrade probe (framework step, then an `ExternalProviders` step on
the last SDKv2 release with `ExpectEmptyPlan`).

**Wiring** — the flip is one commit with the SDKv2 entry *deleted*; shared-helper renames run
`make test` on `kubernetes/`; docs regenerated from `templates/`.

---

## 10. Lessons that generalise

- **Verify against the cluster before claiming.** Several confident predictions here were wrong
  until probed: bug 3's create path, the `IsRequired` gap, the `timeouts` state shape, and the
  direction of the `{}`↔null update.
- **Schema shape is a state contract** — confirm it against a real state file, never from HCL.
- **Size validators skip null.** A required block needs `listvalidator.IsRequired()`.
- **`Computed` + `RequiresReplace` needs `UseStateForUnknown`.** Only update-step tests reveal it.
- **SDKv2 stores `""` where the framework stores null**, and `RequiresReplace` compares values.
- **Port the caller's guard, not just the callee.**
- **A filtering test needs a key the control plane will not restore.**
- **Assert against the live object when the value is invisible to Terraform.**
- **A default that every test overrides is a default no test covers.**
- **Parity bugs get documented and fixed provider-wide**, not silently changed inside one
  migration.
- **Reuse `common/`** rather than re-declaring metadata code per resource.

---

## Appendix A — the original documents

Nothing here is deleted; each holds the full experiment log behind its part of this guide.

| Document | What only it has |
| --- | --- |
| `MIGRATION_PLAN_namespace_v1.md` | Full step-by-step plan with the per-step rationale and file layout |
| `MIGRATION_TUTORIAL_namespace_v1.md` | The learning contract, the concepts checklist, and the misses log A–F — the mistakes a learner actually makes, grouped by concept |
| `MIGRATION_PR_REVIEW_CHECKLIST.md` | The reviewer's checklist in full, with the findings from PRs #2960–#2973 that produced each row |
| `MIGRATION_FINDINGS_namespace_v1.md` | The SDKv2 behaviour audit as originally filled in, and the 13-row test-case table the unit tests were transcribed from |
| `MIGRATION_TESTS_namespace_v1.md` | The test suite in detail: helpers, the full coverage matrix, and what the harness cannot reach |
| `MIGRATION_CONCEPTS_identity_and_state.md` | Identity in both SDKs, the import + `-generate-config-out` flow, `UpgradeResourceState`, and the downgrade analysis |
| `MIGRATION_LEARNING_identity_upgrade_pre_2_38.md` | Why pre-2.38 state breaks, fact by fact, with the Terraform and SDK source that decides it |
| `FIX_namespace_v1_metadata_patch_ops.md` | Bug 7 end to end, the alternatives rejected, and the Bug 2 reproduce recipe |
| `MIGRATION_BLOCKER_optional_computed_blocks.md` | The Optional+Computed analysis in full, with the affected-declaration inventory |
| `MIGRATION_TRACKER.md` | Programme-level: every resource, its blocker status, and what `moved` support each needs |

`MIGRATION_USER_LOG.md` is Jitendra's own notes and is not maintained by Claude.
`migration-blocker-major-cloud-providers.txt` is a HashiCorp-internal memo — keep it out of any
public commit and out of anything shared with teammates.

## Appendix B — what is still open

- **Bug 2** — provider-wide, 39 resources. §8.
- **A namespaced model without `GenerateName`**, when a consumer needs it. §5.
- **Findings §5** — whether anything captured from the Create response goes stale during the
  `wait_for_default_service_account` poll. Low risk, never probed.
