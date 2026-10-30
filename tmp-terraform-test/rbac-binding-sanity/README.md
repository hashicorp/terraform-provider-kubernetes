# Release 3.5 Sanity Tests: RoleBinding & ClusterRoleBinding Framework Migration

Manual upgrade sanity for the two RBAC binding migrations in the 3.5 release. Start here, then
open a suite, then a case. **Every case folder has its own README with copy-paste steps and the
exact output to expect.**

## Folder Layout

```text
rbac-binding-sanity/
├── README.md                  ← you are here: overview, build, findings
├── registry.tfrc              ← Terraform CLI config: released providers only (step 1 of every case)
├── role_binding/              ← PR #2978 suite         → README.md, released.tfrc, 3 cases
│   ├── 01-unversioned-pre-identity/README.md
│   ├── 02-unversioned-post-identity/README.md
│   └── 03-versioned-post-identity/README.md
└── cluster_role_binding/      ← PR #2976 suite         → same structure
```

| Suite | PR | Resources under test |
| --- | --- | --- |
| [role_binding](role_binding/README.md) | [#2978](https://github.com/hashicorp/terraform-provider-kubernetes/pull/2978) | `kubernetes_role_binding_v1` (+ move from `kubernetes_role_binding`) |
| [cluster_role_binding](cluster_role_binding/README.md) | [#2976](https://github.com/hashicorp/terraform-provider-kubernetes/pull/2976) | `kubernetes_cluster_role_binding_v1` (+ move from `kubernetes_cluster_role_binding`) |

## The Idea in One Paragraph

Each case first creates real objects with a **released** provider (v2.37.1 or v3.4.0), saves the
live Kubernetes JSON and the Terraform state, then switches Terraform to the **locally built** provider.
After every upgrade, move, or update, it checks three things:

1. **Plans.** No create, destroy or replace is ever planned, and follow-up plans say `No changes`.
2. **Live objects.** `diff` of the full `kubectl get -o json` must be identical. Same `uid` and
   `resourceVersion` means nothing was recreated or rewritten.
3. **State.** `diff` of the state file before and after must contain only explained normalisation.

## Prerequisites

- `terraform` ≥ 1.12, `go`, `kubectl`, `jq`, and git access to `origin`.
- A kind cluster whose context is `kind-kind` (`kubectl config current-context`).
- **Do not** edit `~/.terraformrc`. Every step exports `TF_CLI_CONFIG_FILE` explicitly.

## Step 0: Build the Provider

The new version is not released yet, so it is built from source.

Each suite is built from its own PR into `/tmp/k8s-sanity/<suite>`, the path its `released.tfrc` points at
(see F5 for why there is no combined build). Run from inside a checkout of this repository:

```sh
REPO=$(git rev-parse --show-toplevel)
for pr_suite in 2978:role_binding 2976:cluster_role_binding; do
  pr=${pr_suite%%:*}; suite=${pr_suite#*:}
  git -C "$REPO" fetch -q origin "pull/$pr/head"
  src=$(mktemp -d)
  git -C "$REPO" worktree add -q --detach "$src" FETCH_HEAD
  mkdir -p "/tmp/k8s-sanity/$suite"
  (cd "$src" && go build -o "/tmp/k8s-sanity/$suite/terraform-provider-kubernetes" .)
  echo "PR #$pr $(git -C "$src" rev-parse HEAD)" > "/tmp/k8s-sanity/$suite/BUILD_INFO"
  git -C "$REPO" worktree remove --force "$src"
done
cat /tmp/k8s-sanity/*/BUILD_INFO
```

Once **both PRs are merged**, rebuild both suites from `origin/main` (replace `pull/$pr/head` with `main`)
and repeat all cases.

### The Two Terraform CLI Configs

| File | Used in | Effect |
| --- | --- | --- |
| [registry.tfrc](registry.tfrc) | step 1 | Downloads the pinned released provider from the registry. |
| `<suite>/released.tfrc` | steps 2–7 | `dev_overrides`: `hashicorp/kubernetes` → `/tmp/k8s-sanity/<suite>`. Terraform prints `Provider development overrides are in effect`, and **no `terraform init` is needed**. |

## Known Findings (F1–F6)

The case READMEs refer to these codes in their "Expected" lines.

| Code | Resource | What you will see | Severity |
| --- | --- | --- | --- |
| **F1** | both | After `terraform import`, the next plan shows `1 to change` instead of `No changes`. It converges after one apply. | Fails the import check |
| **F2** | both | Subject with `api_group = ""`: after migration (upgrade/move) and import, the plan shows `api_group = "rbac.authorization.k8s.io" -> null`. One apply fixes it, with no write to Kubernetes (live JSON identical). | Plan noise on upgrade |
| **F3** | cluster_role_binding | Subjects with `api_group` omitted show `(known after apply)` in every plan that touches the binding. role_binding has a plan modifier for this; cluster_role_binding does not. | Plan noise |
| **F4** | both | Moving the unversioned type: a binding without labels/annotations shows `labels = {} -> null`, `annotations = {} -> null` once. | Plan noise on move |
| **F5** | release | PR #2978 and PR #2976 do not compile together (duplicate `RoleRefModel`, `SubjectModel`, `expandRoleRef`, `flattenRoleRef`, `expandSubjects` in `rbacv1`). Each suite is therefore built from its own PR. | Blocks the combined build |
| **F6** | role_v1 (not these PRs) | Case `role_binding/01` only: `plan -refresh=false` against v2.37.1 state shows `kubernetes_role_v1.test` `+ resource_names = []`. Gone with refresh. From the already-merged role_v1 migration (#2964). | Plan noise |

## Not Covered by These Cases

Fresh create on the new build alone (covered by PR acceptance tests), import via `import` block or
resource identity, failure paths (invalid input, RBAC denied), and proof that zero no-op API requests
were sent.
