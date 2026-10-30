# Basic Role Upgrade Sanity

These are independent Terraform configurations, not an automated runner. Each
directory has its own state and a distinct
Role name. Defaults use `~/.kube/config`, context `kind-kind`, and the existing
`default` namespace. No namespace or RoleBinding is managed.

| Configuration | Starting provider | Resource |
| --- | --- | --- |
| [01-unversioned-pre-identity/main.tf](01-unversioned-pre-identity/main.tf) | 2.37.1, pre-identity | `kubernetes_role` |
| [02-unversioned-post-identity/main.tf](02-unversioned-post-identity/main.tf) | 3.2.1, post-identity | `kubernetes_role` |
| [03-versioned-post-identity/main.tf](03-versioned-post-identity/main.tf) | 3.2.1, post-identity | `kubernetes_role_v1` |

## 1. Create With SDKv2

The table describes the original starting configurations. If you have already
changed a resource type, added a `moved` block, or selected a development override,
preserve those edits and continue at the appropriate step, rather than restarting
creation. In particular, check `released.tfrc`: despite its name, it may now contain
your local-provider override. Only use it for a fresh SDKv2 baseline when it has
no `dev_overrides` entry.

From the repository root, choose one directory:

```sh
cd tmp-terraform-test/role-basic-sanity/01-unversioned-pre-identity
export TF_CLI_CONFIG_FILE="$(cd .. && pwd)/released.tfrc"
terraform init
terraform apply
terraform output role_uid
terraform state pull > sdkv2-before-upgrade.state.json
```

For SDKv2 creation, the selected CLI configuration must use direct installation
without development overrides. A version constraint does not override a
`dev_overrides` entry. Confirm the version shown by `init` and the absence of a
development-override warning during `plan`/`apply`.

These commands use the defaults. To change `config_path`, `config_context`,
`namespace`, or `run_id`, pass the same `-var` values on every applicable command.
Keep `phase=before` throughout the upgrade. For another independent run, use a
fresh working directory and a different `run_id`; retain previous state and evidence.

### Capture the Live Baseline

These checks require `kubectl`, `jq`, and Bash or Zsh.

Run these commands in the chosen case directory, in the same terminal used for
the later checks. Select the Role name explicitly from the configuration, not
from Terraform state. Adjust the kubeconfig, context, namespace, and `run_id`
suffix if you changed their Terraform values.

| Case | Default live Role name |
| --- | --- |
| 01 | `role-sanity-pre-unversioned-manual` |
| 02 | `role-sanity-post-unversioned-manual` |
| 03 | `role-sanity-post-versioned-manual` |

```sh
kubeconfig="$HOME/.kube/config"
context="kind-kind"
namespace="default"
role_name="role-sanity-post-unversioned-manual"
checks_dir=$(mktemp -d "$PWD/role-live-checks.XXXXXX")

kc() {
  kubectl --kubeconfig="$kubeconfig" --context="$context" --namespace="$namespace" --request-timeout=30s "$@"
}

role_view() {
  jq -S '{
    apiVersion, kind,
    name: .metadata.name, namespace: .metadata.namespace,
    uid: .metadata.uid, creationTimestamp: .metadata.creationTimestamp,
    labels: (.metadata.labels // {}), annotations: (.metadata.annotations // {}),
    rules: [(.rules // [])[] | {
      apiGroups: (.apiGroups // [] | sort), resources: (.resources // [] | sort),
      resourceNames: (.resourceNames // [] | sort), verbs: (.verbs // [] | sort)
    }] | sort_by(.apiGroups, .resources, .resourceNames, .verbs)
  }' "$1"
}
```

If testing ignored metadata, run the optional metadata commands below now, using
these variables and `kc`. Then capture the baseline, before changing providers:

```sh
kc get role "$role_name" -o yaml
kc get role "$role_name" -o json > "$checks_dir/before-upgrade.live.json"
role_view "$checks_dir/before-upgrade.live.json"
```

Confirm `managed=before` in annotations and labels, `remove=yes` in labels,
`get/list` permissions for core `pods/configmaps`, and `get` for the named
`apps/deployments` resource `example`.

The unique evidence directory preserves older runs. Do not overwrite its
baseline. If the upgrade already happened, capture the current object under a
different filename: a new snapshot cannot reconstruct the pre-upgrade object.

## 2. Upgrade Only the Provider

Change `required_providers.kubernetes.version` in the chosen configuration to
the new published provider version. Leave the resource type and other settings
unchanged initially, then run:

```sh
terraform init -upgrade
terraform plan -refresh=false
terraform plan
terraform apply
terraform plan
terraform plan -refresh=false
terraform output role_uid
```

Stop if either upgrade plan proposes creation, destruction, or replacement.
The UID must remain unchanged. Most upgrade plans should be empty; an explained
one-time state normalization may be acceptable, but subsequent plans must be
empty. An unchanged UID alone does not prove zero Kubernetes writes; these basic
configurations do not record API requests.

### Compare Kubernetes Before and After the Upgrade

After the upgrade apply and convergence checks, query the API again:

```sh
kc get role "$role_name" -o yaml
kc get role "$role_name" -o json > "$checks_dir/after-upgrade.live.json"

diff -u \
  <(role_view "$checks_dir/before-upgrade.live.json") \
  <(role_view "$checks_dir/after-upgrade.live.json")

diff -u \
  <(jq -S . "$checks_dir/before-upgrade.live.json") \
  <(jq -S . "$checks_dir/after-upgrade.live.json")
```

The first diff compares object identity, creation time, ALL labels/annotations,
and permissions, normalizing only rule ordering and omitted empty collections.
It should be empty for an unchanged configuration. A changed UID means recreation;
changed permissions or missing metadata must be investigated even if Terraform
reports no changes.

The second diff compares the full API response, including `resourceVersion` and
any `managedFields`. Preserve and investigate differences. A controller may also
change these fields, so a difference alone does not identify the writer. Conversely,
an unchanged live object does not prove that no API write request was sent.

## 3. Exercise an Update

```sh
terraform plan -var='phase=after'
terraform apply -var='phase=after'
terraform plan -var='phase=after'
terraform plan -refresh=false -var='phase=after'
terraform output role_uid
```

This adds the `watch` verb, changes the managed annotation and label, removes one
label, and adds another. Expect an in-place update, the same UID, and empty
follow-up plans. Continue passing `phase=after` once this update is applied.

### Verify the Update Reached Kubernetes

```sh
kc get role "$role_name" -o json > "$checks_dir/after-update.live.json"
role_view "$checks_dir/after-update.live.json"

jq -e --slurpfile before "$checks_dir/before-upgrade.live.json" '
  def rules:
    map({apiGroups: (.apiGroups // [] | sort), resources: (.resources // [] | sort),
         resourceNames: (.resourceNames // [] | sort), verbs: (.verbs // [] | sort)})
    | sort_by(.apiGroups, .resources, .resourceNames, .verbs);
  ($before[0].metadata) as $old
  | .kind == "Role" and .apiVersion == "rbac.authorization.k8s.io/v1"
    and .metadata.name == $old.name and .metadata.namespace == $old.namespace
    and .metadata.uid == $old.uid
    and .metadata.creationTimestamp == $old.creationTimestamp
    and (.metadata.annotations // {}) == (($old.annotations // {}) + {managed:"after"})
    and (.metadata.labels // {}) == (($old.labels // {} | del(.remove)) + {managed:"after",added:"yes"})
    and ((.rules // [] | rules) == ([
      {apiGroups:[""],resources:["pods","configmaps"],verbs:["get","list","watch"]},
      {apiGroups:["apps"],resources:["deployments"],resourceNames:["example"],verbs:["get"]}
    ] | rules))
' "$checks_dir/after-update.live.json"
```

Expect `true` and exit code 0. This checks the intended configuration directly
against Kubernetes, including exact permissions and preservation of all baseline
unmanaged keys, not just Terraform's claimed result. Unexpected controller
metadata changes also fail the exact-map comparison; inspect the saved objects
before attributing the difference to the provider.

To compare the two views side by side without refreshing or changing state:

```sh
terraform state pull > "$checks_dir/after-update.state.json"
jq '[.resources[]? | select(.type == "kubernetes_role" or .type == "kubernetes_role_v1")
  | {type, instances: [.instances[] | .attributes | {id,metadata,rule}]}]' "$checks_dir/after-update.state.json"
kc get role "$role_name" -o yaml
```

UID, namespace/name, managed metadata, and permissions must agree. Ignored
metadata should remain visible in Kubernetes but absent from managed state;
that difference is intentional.

## Optional: Ignored Metadata

After SDKv2 creation and before upgrading, add external and internal metadata.
Use the explicitly selected `kc` function and Role name from the live checks:

```sh
kc annotate role "$role_name" outside.example/keep=external example.kubernetes.io/owner=internal
kc label role "$role_name" outside.example/owner=external example.kubernetes.io/team=internal
terraform apply
```

Capture the pre-upgrade live baseline after these commands. Inspect the Role after
the upgrade and update; all four keys must survive and
remain outside managed metadata in Terraform state. These examples already have
managed keys in both maps, so they do not exercise the known first-key same-map
metadata-loss limitation.

## Optional: Move the Unversioned Resource to Framework

In the tested implementation, a provider-only upgrade leaves `kubernetes_role`
on SDKv2. The third configuration already targets `kubernetes_role_v1`, so its
provider upgrade exercises the Framework migration directly.

For either unversioned configuration, after selecting a provider with Role move
support, change the resource header to `resource "kubernetes_role_v1" "test"`,
update the three output references from `kubernetes_role.test` to
`kubernetes_role_v1.test`, and add:

```hcl
moved {
  from = kubernetes_role.test
  to   = kubernetes_role_v1.test
}
```

Repeat the upgrade plan/apply checks without changing the name or namespace.
Repeat the live-object comparison too: changing the Terraform address must not
change the Kubernetes Role. Preserve the current `phase` value. Do not change the resource type without the
`moved` block; that can propose destruction and creation instead of a state move.

## Cleanup

```sh
terraform destroy -var='phase=after'
```

Use `phase=before` if no update was applied. Verify the Role is absent and retain
the directory, state, and snapshots:

```sh
if remaining=$(kc get role "$role_name" --ignore-not-found -o name); then
  if [[ -z "$remaining" ]]; then
    printf 'PASS: Role is absent\n'
  else
    printf 'FAIL: Role still exists: %s\n' "$remaining"
  fi
else
  printf 'NOT VERIFIED: Kubernetes lookup failed\n'
fi
```

A connection or permission error is not proof of deletion. This directory is
gitignored, not backed up by a clean Git worktree.