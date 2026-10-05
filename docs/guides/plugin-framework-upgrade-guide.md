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
| Data source | `kubernetes_namespace_v1` | v3.3.0 | [Namespace data source](#data-source-kubernetes_namespace_v1) |
| Data source | `kubernetes_all_namespaces` | v3.3.0 | [All namespaces data source](#data-source-kubernetes_all_namespaces) |

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

## Data source: kubernetes_namespace_v1

- **Required name:** Omitting `metadata.name` now fails validation instead of failing during the read.

## Data source: kubernetes_all_namespaces

Behavior is unchanged; no configuration updates are required.