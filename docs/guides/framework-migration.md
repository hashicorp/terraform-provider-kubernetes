---
subcategory: ""
page_title: "Kubernetes: Plugin Framework Migration Guide"
description: |-
  Upgrade checks and behavior changes for resources and data sources migrating to the Terraform Plugin Framework in Kubernetes provider v3.3.0 and later.
---

# Plugin Framework migration

Starting with Kubernetes provider v3.3.0, existing resources and data sources are being migrated from the Terraform Plugin SDKv2 to the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework). The migration is incremental: each release moves a few of them, and the provider serves migrated and unmigrated ones side by side.

Every migrated resource and data source has its own release notes and, where relevant, a **Behavior notes** section on its documentation page. This guide collates the migration-specific information in one place, and will grow as more migrations are released.

## Migration approach

The migration preserves what your configuration and state depend on:

- Resource and data source names are unchanged.
- Configuration syntax is unchanged. Nested blocks stay blocks, so `metadata { ... }` keeps working and references such as `kubernetes_namespace_v1.example.metadata[0].name` remain valid.
- Existing state is read as-is. No re-import, no manual state editing, and no `moved` block is needed for a provider-only upgrade.
- Resource identity, where a resource already had it, is carried across.
- The Kubernetes API version a resource talks to does not change. This migration changes the provider's implementation, not the API group or version.

Two things can still change, and they are why this guide exists.

**The Framework rejects configurations that SDKv2 accepted silently.** Some of what SDKv2 allowed was never correct: it either did nothing, or it did something other than what the configuration said. Where we found such cases we fixed them rather than reproducing them, because carrying them forward would mean keeping them indefinitely. A configuration relying on one of those behaviors now fails during `plan`. This appears in the release notes as a breaking change even though the configuration never did what it looked like it was doing.

**Some values are stored more precisely.** Framework types represent null and unknown values explicitly, where SDKv2 could collapse those distinctions. This can produce a one-time in-place update on the first plan after upgrading. See [Background: null, empty, and unknown values](#background-null-empty-and-unknown-values).

Neither of those should ever require **replacing** a resource. If a plan proposes replacing or deleting a resource after this upgrade, do not apply it — see [Reporting a problem](#reporting-a-problem).

## Upgrade checklist

### Before upgrading

1. Run `terraform plan` with your current provider version and confirm it is empty.

    This is the step that matters most. If your configuration already has drift or pending changes, you will not be able to tell afterwards which differences came from the upgrade. Resolve them first, then repeat.

2. Back up your state, using your backend's snapshot mechanism or `terraform state pull`.

    This is your rollback path. Once you apply, state holds values in the Framework's representation, and returning to an earlier provider version is simpler from a pre-upgrade copy than by reversing the change — see [Rolling back](#rolling-back). Store the backup securely, since state can contain sensitive values, and keep the current configuration and dependency lock file alongside it.

3. Review the [migration coverage](#migration-coverage) and the behavior notes for every release between your current and target versions. When upgrading from v2, also follow the [v3 upgrade guide](v3-upgrade-guide.md).

4. Check expressions and module inputs that consume metadata for assumptions about empty maps or empty strings. The [background](#background-null-empty-and-unknown-values) section covers what to look for.

5. Test the upgrade in a non-production workspace first.

### Upgrading and verifying

1. Update the provider version constraint and run `terraform init -upgrade`:

    ```terraform
    terraform {
      required_providers {
        kubernetes = {
          source  = "hashicorp/kubernetes"
          version = "3.3.0"
        }
      }
    }
    ```

    `init -upgrade` can also upgrade other providers and modules within their existing constraints, so review the lock-file changes rather than just the Kubernetes provider's.

2. Run `terraform validate`, then `terraform plan`, before applying anything.

3. Read the plan against the notes below:

    - **Empty plan** — nothing further is needed.
    - **In-place updates** — compare each difference with the behavior notes in this guide and on the resource's own page, including output changes and changes to downstream resources.
    - **A validation error** — the configuration relies on something the Framework rejects. The error names the attribute and the line; the notes below list the cases we know about.
    - **A deletion or replacement** — do not apply it. See [Reporting a problem](#reporting-a-problem).

4. Apply only reviewed changes, then run `terraform plan` again and confirm it is empty. A difference that comes back is not a one-time migration adjustment.

5. Commit the reviewed configuration and lock file, and repeat these checks for each workspace using the provider.

Do not remove resources from state or re-import them to make an unexplained difference go away. That hides the problem rather than fixing it, and we would rather hear about it.

### Rolling back

Pin the previous version in `required_providers` and run `terraform init -upgrade`.

Rolling back before you have applied anything is just re-pinning. Rolling back *after* applying needs more care: the applied state may hold values in a representation the older provider writes differently, so expect the first plan on the older version to show the reconciliation in reverse. This is the main reason to test in a non-production workspace first, and to keep the state backup from step 2.

## Background: null, empty, and unknown values

Most of the surprises in this migration come from one distinction, so it is worth understanding once.

Terraform distinguishes an absent value (`null`) from a known empty value such as `""`, `{}` or `[]`. An unknown value, shown as `(known after apply)`, is different again: Terraform does not know it yet and will after the apply.

SDKv2's schema and state handling could collapse those distinctions. An explicitly configured empty metadata map could be recorded as `null`, and computed fields that were never set could be recorded as zero values instead:

| Meaning | Framework | What SDKv2 could store |
| --- | --- | --- |
| Not set | `null` | `""`, `0`, `{}`, or `[]` |
| Set, but empty | `{}` or `[]` | `{}` or `[]`, indistinguishable from not set |
| Set to a value | the value | the value |

This does not mean every empty value becomes `null`, and it does not change your state backend. The result depends on the attribute's schema, your configuration, and the API response. Existing access paths such as `metadata[0].name` remain valid.

Three consequences surface in configurations rather than in state:

- **Omitted, null, and empty maps are not interchangeable.** Omitting `annotations`, setting it to `null`, and configuring `{}` can produce different state. Where this causes a one-time update, the resource notes below say so.

- **`null` is equal only to `null`.** An expression such as `metadata[0].generate_name != ""` evaluates to `true` when `generate_name` is `null`, not `false`, so a check written against `""`, `0` or `{}` can silently invert where a value moves from empty to `null`.

- **Collection functions reject `null`.** `keys`, `length`, `lookup`, `merge` and `for` expressions need a collection, not `null`. Normalize it where an absent map should mean an empty map to your module:

    ```terraform
    locals {
      namespace_annotations = coalesce(data.kubernetes_namespace_v1.example.metadata[0].annotations, tomap({}))
    }
    ```

    Note that `try(value, {})` does **not** substitute for a `null` value — `try` only catches evaluation errors, and `null` is not an error. Use `coalesce` or an explicit null check. Normalizing is a consumer-side choice; it is not a reason to add empty maps to every resource.

## Migration coverage

The version below is the first provider release containing each migration. The common considerations above do not imply that every resource in the provider has migrated or now behaves identically.

| Kind | Terraform type | Migrated in | Behavior notes |
| --- | --- | --- | --- |
| Resource | `kubernetes_namespace_v1` | v3.3.0 | [Namespace resource](#resource-kubernetes_namespace_v1) |
| Data source | `kubernetes_namespace_v1` | v3.3.0 | [Namespace data source](#data-source-kubernetes_namespace_v1) |
| Data source | `kubernetes_all_namespaces` | v3.3.0 | [All namespaces data source](#data-source-kubernetes_all_namespaces) |

## Resource: kubernetes_namespace_v1

- **Empty metadata maps.** SDKv2 could store an explicitly configured `annotations = {}` or `labels = {}` as `null`. The first plan after upgrading can show a one-time in-place update reconciling the two. Conversely, maps stored as `{}` by an earlier import can change to `null` when omitted from configuration. Neither requires replacing the namespace, and both settle after one apply. Do not edit unrelated metadata just to make the plan empty.

- **Computed metadata during updates.** `metadata[0].generation` and `metadata[0].resource_version` can appear as `(known after apply)`, because the provider reads them from the API response. This does not indicate a replacement, but review downstream changes in expressions that reference them.

- **Null annotation and label entries.** Existing configurations such as `annotations = { owner = null }` remain supported. Null entries are omitted from Kubernetes writes but retained in Terraform state so repeated plans are stable. SDKv2 could omit these entries from state, so upgrading can produce a one-time state-only, in-place update. No HCL change is required.

- **Unset `metadata.generate_name`.** Omission is represented as `null`, where SDKv2 could store `""`. The first refreshed plan can show this state-only normalization in `resource_drift` even when the resource action is `no-op`. Saving the refreshed state records it without changing or replacing the namespace. Leave `generate_name` omitted rather than setting an empty string.

- **Namespace and metadata preservation.** A provider-only upgrade or alias move must preserve the namespace UID and existing metadata. Updates preserve keys excluded by internal filtering, provider `ignore_annotations` / `ignore_labels`, or Terraform `lifecycle.ignore_changes`, including when adding the first managed key. Explicit configuration edits and unignored external drift retain the existing SDKv2 reconciliation behavior; resolve pre-existing drift before upgrading.

For the full schema, and for moving from the deprecated `kubernetes_namespace` resource, see the [namespace resource documentation](../resources/namespace_v1.md).

## Data source: kubernetes_namespace_v1

- **`metadata.name` is now required.** Earlier versions accepted a configuration without it and then failed during the read, since there is nothing to look up without a name. The error now appears during `plan` instead. This data source has never had a `generate_name` attribute — a generated-name prefix is not a lookup mechanism.

Everything else behaves as it did under SDKv2, including the two cases most likely to be relied on:

- `metadata.annotations` and `metadata.labels` can still be set in configuration. They do not filter the lookup; when the namespace exists, they are replaced by its actual values.
- Reading a namespace that does not exist still succeeds without an error and records the same state as before: the requested name in `id` and `metadata.name`, `{}` for the two maps, `""` for `uid` and `resource_version`, `0` for `generation`, and a null `spec`. Existence checks such as `metadata[0].uid != ""` keep working.

Do not treat the data source's `id` as proof that the namespace exists; it is set from the requested name either way.

See the [namespace data source documentation](../data-sources/namespace_v1.md) for its lookup configuration and returned attributes.

## Data source: kubernetes_all_namespaces

No configuration changes are required. `namespaces` remains a list of strings, empty (`[]`) if there are no results, and `id` remains a hash of the returned names. With an unchanged API response this migration should produce an empty plan. Namespace additions or removals still change the results and any expressions using them, but those are cluster changes rather than migration steps.

See the [all namespaces data source documentation](../data-sources/all_namespaces.md) for usage examples.

## Moving from deprecated unversioned names

This is a separate operation from the Framework migration, and only applies if you are also changing a deprecated unversioned resource type such as `kubernetes_namespace` to its versioned name. It needs Terraform 1.8 or later. If you already use the versioned name, you do not need it.

Migrated resources support `moved` blocks from their unversioned equivalents. See the [versioned resource names](versioned-resources.md) guide for the naming scheme, and the resource's own documentation page for a worked example and the checks to run afterwards.

## Reporting a problem

If you hit a plan you cannot explain after upgrading, please open an issue in the [issue tracker](https://github.com/hashicorp/terraform-provider-kubernetes/issues) including:

- the provider versions you upgraded from and to, and your Terraform version,
- the resource or data source involved,
- a minimal configuration that reproduces it,
- a redacted plan, and `TF_LOG=DEBUG` output if you can share it.

Do not attach unredacted state or credentials. A plan that proposes deleting or replacing a resource, or one that stays non-empty after a successful apply, is a bug we want to hear about.
