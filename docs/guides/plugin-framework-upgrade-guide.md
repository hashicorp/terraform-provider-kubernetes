---
subcategory: ""
page_title: "Kubernetes: Upgrade Guide for Kubernetes Provider v3.3.0+ Plugin Framework"
description: |-
  Upgrade checks and behavior changes for resources and data sources migrating to the Terraform Plugin Framework in Kubernetes provider v3.3.0 and later.
---

# Plugin Framework migration

Starting with Kubernetes provider v3.3.0, existing resources and data sources are being migrated incrementally from the Terraform Plugin SDKv2 to the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework). This guide collects the upgrade steps and behavior changes and will grow as more resources are migrated.

Goal is to avoid breaking changes and preserve existing configurations, state, and Kubernetes objects. However, fixing incorrect behavior and handling empty values more precisely can change validation or plans for some configurations. The Kubernetes API versions used by these resources do not change.

The workload resources `kubernetes_deployment_v1`, `kubernetes_daemon_set_v1`, `kubernetes_stateful_set_v1`, `kubernetes_pod_v1`, `kubernetes_job_v1` and `kubernetes_cron_job_v1` were migrated in v4.0.0 and change the syntax of some arguments. Their upgrade steps and behavior changes are described in the [v4 upgrade guide](v4-upgrade-guide.md).

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
| Resource | `kubernetes_role_binding_v1` | v3.5.0 | [RoleBinding resource](#resource-kubernetes_role_binding_v1) |
| Resource | `kubernetes_cluster_role_binding_v1` | v3.5.0 | [ClusterRoleBinding resource](#resource-kubernetes_cluster_role_binding_v1) |
| Data source | `kubernetes_namespace_v1` | v3.3.0 | [Namespace data source](#data-source-kubernetes_namespace_v1) |
| Data source | `kubernetes_all_namespaces` | v3.3.0 | [All namespaces data source](#data-source-kubernetes_all_namespaces) |
| Resource | `kubernetes_deployment_v1` | v4.0.0 | [v4 upgrade guide](v4-upgrade-guide.md) |
| Resource | `kubernetes_daemon_set_v1` | v4.0.0 | [v4 upgrade guide](v4-upgrade-guide.md) |
| Resource | `kubernetes_stateful_set_v1` | v4.0.0 | [v4 upgrade guide](v4-upgrade-guide.md) |
| Resource | `kubernetes_pod_v1` | v4.0.0 | [v4 upgrade guide](v4-upgrade-guide.md) |
| Resource | `kubernetes_job_v1` | v4.0.0 | [v4 upgrade guide](v4-upgrade-guide.md) |
| Resource | `kubernetes_cron_job_v1` | v4.0.0 | [v4 upgrade guide](v4-upgrade-guide.md) |

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

## Resource: kubernetes_role_binding_v1

- **Existing state:** Upgrading `kubernetes_role_binding_v1` does not require re-importing objects or manually editing state. Review the normalizations below before applying.
- **Empty subject API group:** A `User` or `Group` subject with `api_group = ""` gets a one-time in-place update. SDKv2 ignored the configured `""` and stored the API server's default, `rbac.authorization.k8s.io`; the configured `""` is now stored. Terraform displays the planned empty string as `null`, so the update looks like a removal:

  ```text
  ~ subject {
      - api_group = "rbac.authorization.k8s.io" -> null
    }
  ```

  Only Terraform state changes. Kubernetes stores `""` for these subjects as `rbac.authorization.k8s.io`, so the binding is neither modified nor replaced, and the next plan is empty. To avoid the update, omit `api_group` or set it to `rbac.authorization.k8s.io` before upgrading.
- **Planned subject API group:** SDKv2 showed `api_group = (known after apply)` for both an omitted `api_group` and `api_group = ""`. Now only an omitted `api_group` is unknown, when the binding is created or a subject's `kind` or `name` changes; `api_group = ""` is planned as configured.
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

## Resource: kubernetes_cluster_role_binding_v1

- **Empty metadata maps:** Explicit `labels = {}` or `annotations = {}` can require a one-time in-place update from SDKv2's `null`. Conversely, a binding imported with an earlier provider version may have stored empty maps that now change to `null` when omitted from configuration. Neither replaces the binding.
- **Empty subject API group:** A `User` or `Group` subject with `api_group = ""` gets a one-time in-place update. SDKv2 ignored the configured `""` and stored the API server's default, `rbac.authorization.k8s.io`; the configured `""` is now stored. Terraform displays the planned empty string as `null`, so the update looks like a removal:

  ```text
  ~ subject {
      - api_group = "rbac.authorization.k8s.io" -> null
    }
  ```

  Only Terraform state changes. Kubernetes stores `""` for these subjects as `rbac.authorization.k8s.io`, so the binding is neither modified nor replaced, and the next plan is empty. To avoid the update, omit `api_group` or set it to `rbac.authorization.k8s.io` before upgrading.
- **Planned subject API group:** SDKv2 showed `api_group = (known after apply)` for both an omitted `api_group` and `api_group = ""`. Now only an omitted `api_group` is unknown, when the binding is created or a subject's `kind` or `name` changes; `api_group = ""` is planned as configured.
- **Computed metadata:** During updates, `metadata[0].generation` and `metadata[0].resource_version` can be `(known after apply)` until read from the API. This does not itself indicate replacement.
- **Plans without refresh:** Immediately after upgrading, `terraform plan -refresh=false` shows a one-time in-place update for a binding configured with `name`, because SDKv2 stored the unset `generate_name` as `""` and it is now `null`. A normal, refresh-enabled plan shows no change for it. Neither replaces the binding.
- **Null map entries:** `metadata.labels` and `metadata.annotations` entries must contain non-null strings. Remove entries such as `owner = null` or provide string values. Omitting the whole map or setting it to `null` remains supported.
- **Import:** Import IDs are the binding name. Identity imports require Terraform 1.12 or later. For a binding created with `generateName`, configure `generate_name` with the original prefix and omit `name` to avoid a replacement plan after import.

### Moving from kubernetes_cluster_role_binding

Terraform 1.8 or later supports moving from the deprecated `kubernetes_cluster_role_binding` resource to `kubernetes_cluster_role_binding_v1`. Change the resource type, update references, and add a `moved` block:

```terraform
moved {
  from = kubernetes_cluster_role_binding.example
  to   = kubernetes_cluster_role_binding_v1.example
}
```

Moved state shows the same one-time updates for empty metadata maps and `api_group = ""`. Verify that the plan does not replace the binding. See the [ClusterRoleBinding resource reference](../resources/cluster_role_binding_v1.md#metadata-and-upgrades) for further details.

## Data source: kubernetes_namespace_v1

- **Required name:** Omitting `metadata.name` now fails validation instead of failing during the read.

## Data source: kubernetes_all_namespaces

Behavior is unchanged; no configuration updates are required.

## Version 4: PodDisruptionBudget

`kubernetes_pod_disruption_budget_v1.spec.selector` changes from a block to a
required object, and `match_expressions` becomes a list of objects. This is an
intentional version 4 configuration change. Plugin Framework also supports blocks;
workload selectors such as Deployment selectors retain their existing syntax.

Follow the [PDB migration instructions](../resources/pod_disruption_budget_v1.md#upgrading-the-provider)
for object syntax, references, `ignore_changes`, module variables, conditional
values, and Terraform JSON. `selector = {}` selects every pod in the namespace;
do not substitute it for an absent selector.

Existing versioned resources upgrade at the same address without a moved block
or re-import. Moving from the deprecated `kubernetes_pod_disruption_budget` is
unsupported because it uses `policy/v1beta1`, whose empty-selector semantics differ.

An omitted `match_expressions` can produce a one-time `[]` to `null` update.
Explicit empty label maps or expression values can normalize legacy `null` to
empty collections. These updates only change state, make no Kubernetes writes,
and do not replace the PDB. Changes to the actual specification still replace it.

State schema version increases from 0 to 1. Provider 3.x cannot read the upgraded
state. Back up state and recreate saved plans when upgrading.

## Version 4: Service

`kubernetes_service_v1` changes `spec.session_affinity_config` and its `client_ip`
child from blocks to object arguments. Both support values computed by Kubernetes.
The `metadata`, `spec`, and `port` blocks retain their syntax.

Follow the [Service migration instructions](../resources/service_v1.md#framework-migration-session-affinity-syntax)
for configuration, references, `ignore_changes`, module types, and automatic port allocation.
Existing versioned resources upgrade at the same address. Moving from
`kubernetes_service` requires Terraform 1.8 or later and the documented `moved` block.
State schema version increases to 2; provider 3.x cannot read the upgraded state.
Back up state and recreate saved plans when upgrading.

## Version 4: Networking

These resources intentionally adopt object arguments for singular nested objects.
This is a version 4 configuration change; Plugin Framework does not require every
singleton block to become an object. Apply the syntax changes only to the paths
listed in the resource guides. Workload selectors, such as Deployment selectors,
retain their existing block syntax.

| Resource | Blocks that become objects | Repeated blocks that become object lists |
| --- | --- | --- |
| [Ingress](../resources/ingress_v1.md) | `spec.default_backend`, `spec.rule[*].http`, path `backend`, and each backend's `resource`, `service`, and `service.port` | `spec.rule[*].http.path` |
| [IngressClass](../resources/ingress_class_v1.md) | `spec.parameters` | None |
| [NetworkPolicy](../resources/network_policy_v1.md) | `spec.pod_selector`, each ingress/egress peer's `pod_selector`, `namespace_selector`, and `ip_block` | Each selector's `match_expressions` |

Use `field = { ... }` for objects and `field = [{ ... }]` for repeated objects.
Replace affected `dynamic` blocks with object expressions or list comprehensions.
Remove `[0]` only for converted objects in references and `ignore_changes` paths;
keep indexes for `metadata`, `spec`, rules, paths, and other retained lists.
Update module variables from `list(object(...))` to `object(...)` for the singular
values. The linked resource guides include conditional and Terraform JSON examples.
Preserve the difference between an omitted selector and an empty selector: an empty
selector can select every pod or namespace.

Existing versioned resources upgrade at the same address. IngressClass and
NetworkPolicy support moves from their same-API deprecated aliases with Terraform
1.8 or later. The beta `kubernetes_ingress` resource is not a supported move source.

Explicit empty label maps or expression values can normalize legacy `null` to an
empty collection in a one-time update after upgrading. That update changes state
without writing to Kubernetes or replacing the resource; the next plan should be empty.

State schema versions increase from 0 to 1. Provider 3.x cannot read the upgraded
state. Back up state and recreate saved plans when upgrading.
