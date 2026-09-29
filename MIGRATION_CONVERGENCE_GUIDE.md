# Converging a migration PR onto `common`

For the open SDKv2 → Plugin Framework migration PRs. Each was written before
`internal/framework/provider/common` existed, so each carries its own copy of the metadata
schema, model, conversion helpers, validators and identity handling. This describes how to
replace those copies with the shared ones.

Written from doing it on `kubernetes_role_v1` (#2964). Everything marked **verified** was hit
on that branch; everything else is expected to follow but has not been run on your resource.

You can hand this to a coding agent. Point it at your PR branch and this file.

---

## What `common` already provides

Anything on this list should not exist in your resource package.

**Metadata schema** — pick the one matching your SDKv2 call site:

| SDKv2 | `common` |
| --- | --- |
| `metadataSchema(name, generatable)` | `MetadataSchema(name, generatable)` |
| `namespacedMetadataSchema(name, generatable)` | `NamespacedMetadataSchema(name, generatable)` |
| `metadataSchemaRBAC(name, generatable, namespaced)` | `MetadataSchemaRBAC(name, generatable, namespaced)` |

**Models** — must match the schema exactly, in both directions. A struct field with no
matching attribute, or an attribute with no field, is a decode error at plan time, not a
compile error.

| Schema | Model |
| --- | --- |
| `MetadataSchema(_, true)` | `common.MetadataModel` |
| `NamespacedMetadataSchema(_, true)` / `MetadataSchemaRBAC(_, true, true)` | `common.NamespacedMetadataModel` |

`NamespacedMetadataModel` embeds `MetadataModel`, which embeds `MetadataBase`. Reads use
promoted fields (`m.Name`); construction must name the embedded type, because Go forbids
setting a promoted field in a composite literal:

```go
common.NamespacedMetadataModel{
    MetadataModel: common.MetadataModel{
        MetadataBase: common.MetadataBase{Name: types.StringValue("x")},
    },
    Namespace: types.StringValue("default"),
}
```

**Conversion and patching** — `ExpandMetadata`, `ExpandNamespacedMetadata`,
`FlattenMetadata`, `FlattenNamespacedMetadata`, `ExpandMapForPatch`, `MetadataPatchOps`.

**Validators** — `LabelsValidator`, `AnnotationsValidator`, `DNSSubdomainNameValidator`,
`DNSLabelPrefixValidator`, `RBACNameValidator`. These are already wired into the schemas
above; you only reference them directly for a non-metadata attribute.

**Identity** — `IdentitySchema`, `NamespacedIdentitySchema`, `ResourceIdentity`,
`NamespacedResourceIdentity`, `UpgradeIdentity`, `UpgradeNamespacedIdentity`,
`IdentitySchemaVersion`.

**IDs** — `kubernetes.BuildId(meta)` and `kubernetes.IdParts(id)`, from the SDKv2 package.
Not `common` — the framework calls SDKv2's directly, like `DiffStringMap`, so the
`namespace/name` format cannot drift between the two providers.

**Provider metadata** — `kubernetes.KubeClientsets` for API clients, `kubernetes.MetadataFilters`
for `ignore_labels` / `ignore_annotations`. Do not add methods to `KubeClientsets`.

---

## Steps

### 1. Branch, then merge main

If the PR branch is not yours, branch off it rather than committing to someone else's PR:

```bash
git switch -c <you>/<resource>-converge origin/<their-branch>
git merge main
```

### 2. Resolve the conflicts

Three are near-certain. The first two are textual; the third is not.

**`internal/framework/provider/provider.go`** — the `Resources()` list. Keep **every** entry
from both sides. Fails at build, so it is hard to get wrong.

**`kubernetes/provider.go`** — `ResourcesMap`. Both sides delete a different `_v1` entry.
**Keep both deletions.** *(verified)*

This one fails **silently**: leaving an entry in still compiles, because the mux only rejects
a duplicate type name when the server starts. You find out during acceptance tests, with a
confusing duplicate-resource error. Check explicitly:

```bash
# expect 0 — no _v1 resource your branch or main has migrated
grep -nE '"kubernetes_(<yours>|namespace|role)_v1":' kubernetes/provider.go
```

**Duplicate `GetIgnoreAnnotations` / `GetIgnoreLabels`** *(verified)* — git merges this
cleanly and the build then fails:

```
kubernetes/provider.go:472:27: method providerMetadata.GetIgnoreAnnotations already declared at :406:27
```

Several PRs added these methods to `KubeClientsets`; main has them on the separate
`MetadataFilters` interface. Delete your branch's pair and its `KubeClientsets` widening, and
resolve `MetadataFilters` where you need the ignore lists.

**Shared files in a package that already has a merged resource** *(verified on #2960)* —
if your resource lives in a package main has already migrated something into, you will
collide on the package's test scaffolding. `corev1_test.go` is the case: both sides declare
`sdkv2providerMeta` and `testAccProtoV6ProviderFactories`.

**The rule for anything shared: on conflict, take main's version and delete yours.** Not
whichever looks better, and not whichever landed first — main is the source of truth, so
every branch converges on the same code and the next person to merge sees no conflict at
all. If main's version is genuinely wrong, fix it in a separate PR against main rather than
carrying a different copy on your branch.

This applies to `rbacv1` in particular: #2964, #2978 and #2976 all target it, and two of them
declare `expandStringMap` / `flattenStringMap` with **incompatible signatures**. Most of those
helpers disappear entirely once you finish §4, so converge first and merge the RBAC branches
into each other afterwards — otherwise you are resolving conflicts between code that is about
to be deleted.

Watch for the quieter version of this, which git does **not** flag: two files with different
names in the same package declaring the same symbol. Nothing conflicts textually, and the
build fails on a duplicate declaration. Grep the package for any type or function you are
adding before assuming it is new.

Confirm the merge is sound before touching anything else:

```bash
go build ./... && go test ./internal/framework/... ./kubernetes/...
```

Your resource still uses its own helpers at this point, so this should pass. If it does not,
it is a bad conflict resolution, not a convergence problem.

### 3. Swap the schema and model

Change the metadata block to the `common` call, delete your local metadata attribute
function, and change your model's metadata field to the matching `common` type. Then delete
your own metadata model.

The compiler now lists every remaining call site. Work through them.

### 4. Swap the helpers, then delete yours

Replace expand / flatten / filter / patch / validators / identity / ID helpers with the
`common` equivalents, then delete the local versions. Keep anything genuinely specific to
your resource — for role that was policy-rule conversion and string-set helpers.

These are the local helpers the open PRs were found to carry. **None of them should survive
convergence.** Names vary between PRs; match on what the function does.

| Local helper | Replace with |
| --- | --- |
| `expandStringMap`, `flattenStringMap` | nothing — the model holds `types.Map`, so use `ElementsAs` / `types.MapValueFrom` at the point of use |
| `toStringInterfaceMap`, `typesMapElements` | `common.ExpandMapForPatch` |
| `filterIgnoredMetadataKeys`, `filterManagedMetadataKeys` | nothing — filtering happens inside `common.Flatten*Metadata` |
| `removeKeys`, `removeInternalKeys`, `isInternalKey`, `isInternalMetadataKey`, `matchesIgnorePattern` | `kubernetes.RemoveKeys`, `kubernetes.RemoveInternalKeys` (already called by `common`) |
| `flattenMetadata`, `flattenNamespacedMetadata`, `flatten<Resource>Metadata` | `common.FlattenMetadata` / `FlattenNamespacedMetadata` / `FlattenBaseMetadata` |
| `expandMetadata` and variants | `common.ExpandMetadata` / `ExpandNamespacedMetadata` / `ExpandBaseMetadata` |
| `diffStringMap`, `escapeJSONPointer` | `kubernetes.DiffStringMap` (it escapes internally) |
| `diffMetadataPatch`, `buildMetadataPatch` | `common.MetadataPatchOps` / `BaseMetadataPatchOps` |
| `buildID`, `parseID`, `splitID` | `kubernetes.BuildId`, `kubernetes.IdParts` |
| a local `kubeIgnoreKeys`-style interface | `kubernetes.MetadataFilters` |
| own identity schema and model | `common.IdentitySchema` / `NamespacedIdentitySchema`, `common.ResourceIdentity` / `NamespacedResourceIdentity` |

Keep what is genuinely yours: the resource's own spec conversion, its `MoveState` and frozen
SDKv2 state structs, and its own `populate-from-response` helper.

Two things to get right while doing this:

**`common` helpers return diagnostics.** Most hand-written ones did not. Append them to
`resp.Diagnostics` and check `HasError()` before writing state, rather than discarding them.

**`Create` and `Update` must echo the plan, not the API response.** Build state from the plan
and overwrite only server-assigned fields (`name`, `namespace`, `uid`, `resource_version`,
`generation`). Filtering belongs in `Read` alone. *(verified — role was flattening in Create
and Update; this only surfaces once labels/annotations stop being `Computed`, see §5)*

If you extract that into a helper, take the model by **pointer** *(verified)*:

```go
func populateMetadataFromResponse(plan *YourModel, out *api.Thing)
```

With a value receiver, `plan.Metadata[0].X = …` still lands because slices share their
backing array, but `plan.ID = …` is silently discarded — so `id` is unknown after apply while
metadata looks correct. It half-works, which is what makes it hard to spot.

### 5. Expect these behaviour changes, and fix the tests

Adopting `common` restores SDKv2 parity in ways your tests may currently contradict. **Rewrite
the tests to assert the new behaviour — do not relax the code to keep them green.** *(all
verified on role)*

| Change | What breaks |
| --- | --- |
| `labels` / `annotations` lose `Computed` + `Default {}` | tests asserting `{}` in state when config omits them |
| `null` map values rejected at plan | tests that set `x = null` and expect it dropped |
| `name` ↔ `generate_name` conflict enforced | tests setting both |
| Plan modifiers ordered `UseStateForUnknown` then `RequiresReplace` | nothing — this fixes a replace-on-unrelated-edit bug |
| `generate_name` gets the `""` → null replace exemption | nothing — this fixes a destroy on `plan -refresh=false` |

Two test shapes need specific handling:

- **Configs declaring `labels = {}` / `annotations = {}`** cannot migrate to an empty plan.
  SDKv2 stored `null` for those. Assert `plancheck.ExpectResourceAction(addr, ResourceActionUpdate)`
  rather than `ExpectEmptyPlan`. Asserting the *specific* action is what stops the divergence
  quietly becoming a replacement later.
- **A step with `ExpectError` must not be last** *(verified)*. The harness plans its destroy
  from the final config, and cannot destroy from one that fails validation — you get dangling
  objects and a confusing failure. End with a config that applies.

### 5b. If you are converging a data source

The guide above was written from a resource. Four things differ, and the first is a blocker
rather than an adjustment.

**`common.MetadataSchema` and friends cannot be used at all.** They return
`resource/schema.ListNestedBlock`; a data source needs `datasource/schema.ListNestedBlock`,
which is a different type. There is no `datasource/schema` variant in `common` yet. Whoever
converges the first data source has to add one — mirroring the existing functions — and that
is a change everyone else inherits, so raise it before writing it.

**The model for `generatableName = false` is `common.MetadataBase`.** Six fields, no
`generate_name`, no `namespace`. Its helpers are `ExpandBaseMetadata`, `FlattenBaseMetadata`
and `BaseMetadataPatchOps`. Check which variant your SDKv2 call site passes: the namespace and
config_map data sources pass `false`, secret passes `true`.

**A namespaced, non-generatable model does not exist.** `MetadataBase` + `Namespace` has no
type yet. A namespaced data source — config_map, secret, service_account — needs one added to
`common`.

**Identity, `MoveState`, `UpgradeIdentity` and `MetadataPatchOps` do not apply.** Data sources
have no apply and no state to move, so ignore every part of this guide that mentions them.
`moved` blocks do not apply to data sources either.

One more thing that is not in the guide because the resource path did not need it: some data
source tests build a muxed provider so the HCL `provider` block is honoured, which may mean
touching `internal/mux/`. If your PR exports something new from there, check whether main
already offers an equivalent before keeping your version.

### 6. Verify

```bash
gofmt -l internal/ kubernetes/     # empty
make go-lint                       # 0 issues
make test                          # runs fmtcheck + vet first

KUBE_CONFIG_PATH=$HOME/.kube/config make frameworkacc \
  PROVIDER_FRAMEWORK_DIR=./internal/framework/provider/<yourpkg> \
  TESTARGS="-run '^TestAcc<Yours>' -timeout 40m"
```

**Unit tests cannot see any of §5.** Schema changes only show up against a real cluster. A
green `make test` means nothing here.

After any red acceptance run, check for leaked objects — a failing step often cannot be
destroyed:

```bash
kubectl get <kind> -A --no-headers | grep tf-acc
```

### 7. Changelog

The §5 changes are user-visible. Two need `release-note:breaking-change` entries, worded as
bug fixes rather than implementation notes:

- rejecting `null` in `labels` / `annotations` — earlier versions accepted it and silently
  dropped the entry, leaving a difference that never settled
- rejecting `name` together with `generate_name` — earlier versions allowed it, which was
  never intended, since the API ignores `generate_name` whenever `name` is set

---

## Checks worth running before you call it done

```bash
# nothing left that common provides
grep -rn 'func \(expand\|flatten\)Metadata\|func buildMetadataPatch\|func \(build\|parse\|split\)ID' \
  internal/framework/provider/<yourpkg>/

# no unchecked provider-metadata assertions
grep -rn 'SDKv2Meta()\.' internal/framework/provider/<yourpkg>/

# identity: present, and delegating
grep -rn 'IdentitySchema\|UpgradeIdentity' internal/framework/provider/<yourpkg>/
```

### `UpgradeIdentity` is usually missing — add it

Most of the open PRs do not implement it. Without it, any object created by provider ≤ 2.37.x
fails its first plan with `Unable to Upgrade Resource Identity`: identity shipped in 2.38.0, so
older state carries version 0 and no identity, and Terraform asks for an upgrade whenever the
stored version differs from the declared one. SDKv2 answers that generically in its gRPC
server; the framework requires each resource to supply it. *(verified — role had none)*

`common` already does the work. Add the interface assertion and one method:

```go
var _ resource.ResourceWithUpgradeIdentity = (*YourResource)(nil)

// UpgradeIdentity implements [resource.ResourceWithUpgradeIdentity].
//
// Without this, any object created by provider 2.37.x or older fails its first plan with
// "Unable to Upgrade Resource Identity": identity shipped in 2.38.0, so older state carries
// identity_schema_version 0 and no identity, and Terraform asks for an upgrade whenever the
// stored version differs from the declared one. SDKv2 answers that generically in its gRPC
// server; the framework requires each resource to supply it.
func (r *YourResource) UpgradeIdentity(ctx context.Context) map[int64]resource.IdentityUpgrader {
    return common.UpgradeNamespacedIdentity(yourAPIVersion, yourKind) // cluster-scoped: common.UpgradeIdentity
}
```

While you are there, check the identity schema declares `Version`. `common.IdentitySchema()`
and `NamespacedIdentitySchema()` set it for you, but a hand-written one that omits it defaults
to 0 — and Terraform refuses an identity schema version going backwards, so existing state
breaks. *(found missing in #2954)*

**No test will catch either problem** if your migration tests pin a recent SDKv2 version.
Covering it needs a test whose first step pins a pre-identity release (2.37.1).

Note this is invisible to any test pinning a recent SDKv2 version. Covering it needs a
migration test whose first step pins a pre-identity release (2.37.1).
