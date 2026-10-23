---
subcategory: ""
page_title: "Kubernetes: Upgrade Guide for Kubernetes Provider v3.3.0+ Plugin Framework"
description: |-
  Upgrade checks and behavior changes for resources and data sources migrating to the Terraform Plugin Framework in Kubernetes provider v3.3.0 and later.
---

# Plugin Framework migration

Starting with Kubernetes provider v3.3.0, existing resources and data sources are being migrated incrementally from the Terraform Plugin SDKv2 to the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework). This guide collects the upgrade steps and behavior changes and will grow as more resources are migrated.

Goal is to avoid breaking changes and preserve existing configurations, state, and Kubernetes objects. However, fixing incorrect behavior and handling empty values more precisely can change validation or plans for some configurations. The Kubernetes API versions used by these resources do not change.

Existing resource state is supported without re-importing objects or manually editing state.

## Upgrade checklist

1. **Before upgrading:** Run `terraform plan` with your current provider version and confirm it is empty. Resolve existing drift or pending changes first.
2. **Review behavior changes:** Read the applicable notes below and release changelogs, including effects on expressions and downstream resources. When upgrading from v2, also follow the [v3 upgrade guide](v3-upgrade-guide.md).
3. **Upgrade:** Update the provider version constraint and run `terraform init -upgrade`. Preferably, test in a non-production workspace first.
4. **Check the plan:** Run `terraform validate` and `terraform plan`. The plan should normally be empty. Compare differences or validation errors with the notes below; previously accepted inputs may now be rejected. Do not apply unexplained deletions, replacements, or other changes.
5. **After upgrading:** Apply only reviewed changes, then confirm the next plan is empty. Repeat these checks for each workspace.

Report unexplained or repeated differences in the [issue tracker](https://github.com/hashicorp/terraform-provider-kubernetes/issues), with provider and Terraform versions, a minimal configuration, and a redacted plan. Do not share unredacted state or remove and re-import resources to suppress a difference.

## Background: null, empty, and unknown values

Terraform distinguishes absent values (`null`), known empty values (`""`, `{}`, or `[]`), and unknown values (`(known after apply)`). SDKv2 could collapse some of these distinctions when reading configuration or storing state. Framework types represent them explicitly; the exact result depends on the attribute and its configuration, not a change to your state backend.

Existing paths such as `metadata[0].name` remain valid.

## Migration coverage

The version below is the first release containing each migration.

| Kind | Terraform type | Migrated in | Behavior notes |
| --- | --- | --- | --- |
| Resource | `kubernetes_namespace_v1` | v3.3.0 | [Namespace resource](#resource-kubernetes_namespace_v1) |
| Resource | `kubernetes_role_v1` | v3.4.0 | [Role resource](#resource-kubernetes_role_v1) |
| Resource | `kubernetes_role_binding_v1` | Unreleased | [RoleBinding resource](#resource-kubernetes_role_binding_v1) |
| Data source | `kubernetes_namespace_v1` | v3.3.0 | [Namespace data source](#data-source-kubernetes_namespace_v1) |
| Data source | `kubernetes_all_namespaces` | v3.3.0 | [All namespaces data source](#data-source-kubernetes_all_namespaces) |

## Resource: kubernetes_role_binding_v1

- **Existing state:** Upgrading `kubernetes_role_binding_v1` does not require re-importing objects or manually editing state. Review the normalizations below before applying.
- **Empty subject API groups:** SDKv2 could store `rbac.authorization.k8s.io` for a User or Group subject explicitly configured with `api_group = ""`. Upgrading may plan a one-time in-place update to preserve the configured empty string. Kubernetes continues to use its canonical API group; the next plan should be empty.
- **Plans without refresh:** Immediately after upgrading from SDKv2, `terraform plan -refresh=false` may show a one-time in-place update normalizing an omitted `metadata.generate_name` from `""` to `null`. This does not replace the RoleBinding. Review a refresh-enabled plan before applying and confirm convergence afterward.
- **Empty metadata maps:** Explicit empty maps and maps omitted after an earlier import may normalize between `{}` and `null` in a one-time in-place update.
- **Null map entries:** Annotation and label entries must be non-null strings. Remove entries such as `owner = null` or supply string values. Omitting the whole map or setting it to `null` remains supported.
- **Computed metadata:** `generation` and `resource_version` may be unknown until apply during updates. This does not itself indicate replacement.
- **Import:** IDs must contain a non-empty namespace and name (`namespace/name`). Identity imports require Terraform 1.12 or later and default an omitted namespace to `default`. For generated bindings, configure the original `generate_name` prefix and omit `name`.

### Configurations with both name and generate_name

Setting both arguments is now rejected during planning, including `generate_name = ""`.

- If `generate_name` is empty, remove it and keep `name`.
- Otherwise, remove `name` and retain the original `generate_name`. The existing binding keeps its name.

Removing or changing a stored non-empty prefix replaces the binding. Review the plan before applying.

### Moving from kubernetes_role_binding

Terraform 1.8 or later supports moving from the deprecated type. Rename the resource block to `kubernetes_role_binding_v1`, retain its configuration, update references, and add:

```terraform
moved {
  from = kubernetes_role_binding.example
  to   = kubernetes_role_binding_v1.example
}
```

Review the move and any in-place normalization changes, then apply and confirm that the next plan is empty. Investigate unexpected replacement rather than removing state or re-importing.

## Resource: kubernetes_namespace_v1

- **Empty metadata maps:** SDKv2 could store explicitly configured `annotations = {}` or `labels = {}` as `null`. Upgrading can produce a one-time in-place update to `{}`. Conversely, maps stored as `{}` by an earlier import can change to `null` when omitted from configuration. These changes do not require replacement and should settle after applying.
- **Computed metadata:** During updates, `metadata[0].generation` and `metadata[0].resource_version` can be `(known after apply)` until read from the API. This does not itself indicate replacement; review downstream changes that depend on these values.
- **Null map entries:** Annotation and label values must be non-null strings. Previously accepted entries such as `annotations = { owner = null }` are now rejected during validation. Remove these entries or provide string values before planning, applying, or destroying. Omitting the whole map or setting it to `null` remains supported.
- **Unset generated-name prefix:** Omitted `metadata.generate_name` is now stored as `null` instead of SDKv2's `""`. Refresh may report this state-only difference even when no resource changes are planned. Leave the attribute omitted; no namespace update or replacement is needed.

### Moving from kubernetes_namespace

This step is only for users upgrading from the deprecated `kubernetes_namespace` resource type to `kubernetes_namespace_v1`. It requires Terraform 1.8 or later; users already managing `kubernetes_namespace_v1` do not need it.

Replace the old resource block with the versioned type, keep its configuration unchanged, and add a `moved` block. For an existing namespace named `terraform-example-namespace`:

```terraform
moved {
  from = kubernetes_namespace.example
  to   = kubernetes_namespace_v1.example
}

resource "kubernetes_namespace_v1" "example" {
  metadata {
    name = "terraform-example-namespace"
  }
}
```

Update references to the new address, for example, `kubernetes_namespace.example.metadata[0].name` to `kubernetes_namespace_v1.example.metadata[0].name`.

Run `terraform plan` to verify the move and review any in-place changes described above. Investigate any proposed replacement before applying. After applying, run `terraform plan` again and confirm it is empty. No state removal or re-import is needed.

## Resource: kubernetes_role_v1

- **Empty metadata maps:** Explicit `labels = {}` or `annotations = {}` can require a one-time in-place update from SDKv2's `null`. Conversely, an earlier import may have stored empty maps that now change to `null` when omitted from configuration. These normalization changes do not themselves replace the Role or change its Kubernetes metadata.
- **Computed metadata:** During updates, `metadata[0].generation` and `metadata[0].resource_version` can be `(known after apply)` until read from the API. This does not itself indicate replacement.
- **Plans without refresh:** Immediately after upgrading, `terraform plan -refresh=false` may show an in-place update adding `resource_names = []` when the argument is omitted. Review a normal, refresh-enabled plan before applying; refresh records the empty set from Kubernetes. This normalization does not itself replace the Role.
- **Null map entries:** `metadata.labels` and `metadata.annotations` entries must contain non-null strings. Remove entries such as `owner = null` or provide string values. Omitting the whole map or setting it to `null` remains supported.
- **Unset generated-name prefix:** Omitted `metadata.generate_name` changes from SDKv2's `""` to `null`. This state-only change needs no Role update or replacement, even with `-refresh=false`. Leave it omitted.
- **Import:** String import IDs must contain a non-empty namespace and name (`namespace/name`). Identity imports require Terraform 1.12 or later and use `default` when namespace is omitted. For a Role created with `generateName`, configure `generate_name` with the original prefix and omit `name` to avoid a replacement plan after import.

### Configurations with both name and generate_name

Setting both `metadata.name` and `metadata.generate_name` is now rejected during planning, including `generate_name = ""`. For an existing Role previously managed with both arguments:

- If `generate_name = ""`, remove `generate_name` and keep `name`.
- If `generate_name` is non-empty, remove `name` and keep the original `generate_name`. The current Role keeps its name; a future replacement can receive a new name generated from that prefix.

Removing or changing a stored non-empty `generate_name` replaces the Role. Review the plan before applying either correction.

### Moving from kubernetes_role

Terraform 1.8 or later supports moving from the deprecated `kubernetes_role` resource to `kubernetes_role_v1`. Change the resource type, update references, and add a `moved` block:

```terraform
moved {
  from = kubernetes_role.example
  to   = kubernetes_role_v1.example
}
```

Resolve the configuration changes above and verify that the plan does not replace the Role. See the [Role resource reference](../resources/role_v1.md#migrating-from-kubernetes_role) for further migration details.

## Data source: kubernetes_namespace_v1

- **Required name:** Omitting `metadata.name` now fails validation instead of failing during the read.

## Data source: kubernetes_all_namespaces

Behavior is unchanged; no configuration updates are required.