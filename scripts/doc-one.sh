#!/usr/bin/env bash
# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

set -euo pipefail

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

mode=${1:-}
kind=${2:-}
name=${3:-}
overwrite=${4:-}
case "$mode" in
  generate|check) ;;
  *) fail "Expected generate or check." ;;
esac
case "$kind" in
  resources) schema_key=resource_schemas ;;
  data-sources) schema_key=data_source_schemas ;;
  *) fail "Set KIND=resources or KIND=data-sources." ;;
esac
[[ "$name" =~ ^[a-z][a-z0-9_]*$ ]] || fail "Set NAME to the short resource/data-source name, without paths or extensions."
[[ -z "$overwrite" || "$overwrite" == 1 ]] || fail "OVERWRITE must be empty or 1."

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"
template="templates/$kind/$name.md.tmpl"
destination="docs/$kind/$name.md"
[[ -f "$template" ]] || fail "Missing explicit template: $template. Reconcile this page's content before generating it."
[[ ! -L docs && ! -L "docs/$kind" && ! -L "$destination" ]] || fail "Documentation destinations must not be symbolic links."
for tool in go terraform jq git; do
  command -v "$tool" >/dev/null || fail "Required tool not found: $tool"
done

work=$(mktemp -d "${TMPDIR:-/tmp}/kubernetes-doc.XXXXXXXX")
replacement=
cleanup() {
  if [[ -n "$replacement" ]]; then
    rm -f "$replacement"
  fi
  rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'echo "ERROR: documentation command failed at line $LINENO." >&2' ERR

# Do not inherit provider overrides, CLI arguments, or Terraform state directories.
unset TF_CLI_ARGS TF_CLI_ARGS_init TF_CLI_ARGS_providers TF_CLI_ARGS_version TF_REATTACH_PROVIDERS TF_PLUGIN_CACHE_DIR
export TF_CLI_CONFIG_FILE="$work/terraform.rc"
export TF_DATA_DIR="$work/terraform-data"
export TF_WORKSPACE=default TF_IN_AUTOMATION=1 CHECKPOINT_DISABLE=1
: > "$TF_CLI_CONFIG_FILE"
terraform_version=$(terraform version -json | jq -r .terraform_version)
[[ "$terraform_version" == 1.16.4 ]] || fail "Use Terraform 1.16.4 for reproducible docs; found $terraform_version."

existed=0
if [[ -f "$destination" ]]; then
  existed=1
  cp "$destination" "$work/original.md"
fi

platform=$(go env GOOS)_$(go env GOARCH)
plugin_dir="$work/terraform/plugins"
binary_dir="$plugin_dir/registry.terraform.io/hashicorp/kubernetes/0.0.1/$platform"
mkdir -p "$binary_dir"
echo "==> Building the current provider and exporting its schema..."
go build -o "$binary_dir/terraform-provider-kubernetes" .
cat > "$work/terraform/main.tf" <<'EOF'
terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "0.0.1"
    }
  }
}
EOF
terraform -chdir="$work/terraform" init -backend=false -input=false -no-color -plugin-dir="$plugin_dir"
terraform -chdir="$work/terraform" providers schema -json > "$work/schema.json"
jq -e --arg key "$schema_key" --arg name "kubernetes_$name" \
  '.provider_schemas["registry.terraform.io/hashicorp/kubernetes"][$key][$name] != null' \
  "$work/schema.json" >/dev/null || fail "kubernetes_$name is not registered in $schema_key."

mkdir -p "$work/staging/templates/$kind"
cp "$template" "$work/staging/templates/$kind/$name.md.tmpl"
cp -R examples "$work/staging/examples"
echo "==> Rendering $template in isolated output with tfplugindocs v0.25.0..."
if ! go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate \
  --provider-dir "$work/staging" \
  --providers-schema "$work/schema.json" \
  --provider-name kubernetes \
  --rendered-provider-name kubernetes \
  --website-source-dir templates \
  --examples-dir examples \
  --rendered-website-dir generated > "$work/generate.log" 2>&1; then
  cat "$work/generate.log" >&2
  fail "tfplugindocs generation failed; published documentation was not changed."
fi

generated="$work/staging/generated/$kind/$name.md"
[[ -s "$generated" ]] || fail "Generator did not produce $kind/$name.md."
if grep -Fq '<no value>' "$generated"; then
  fail "Generated page contains unresolved template values."
fi
[[ "$(head -n 1 "$generated")" == --- ]] || fail "Generated page has no frontmatter."
grep -q '^## Schema' "$generated" || fail "Generated page has no schema section."

if [[ "$existed" == 1 ]]; then
  cmp -s "$destination" "$work/original.md" || fail "$destination changed during generation; retry after reviewing your edits."
else
  [[ ! -e "$destination" ]] || fail "$destination appeared during generation; retry after reviewing it."
fi
if [[ -f "$destination" ]] && cmp -s "$destination" "$generated"; then
  echo "==> $destination is up to date."
  exit 0
fi

original=/dev/null
if [[ "$existed" == 1 ]]; then
  original="$work/original.md"
fi
diff_status=0
diff -u --label "$destination (current)" --label "$destination (generated)" "$original" "$generated" || diff_status=$?
[[ "$diff_status" -le 1 ]] || fail "Unable to compare generated documentation."
if [[ "$mode" == check ]]; then
  fail "$destination is out of date. Run make doc-generate KIND=$kind NAME=$name."
fi

if [[ "$overwrite" != 1 && -n "$(git status --porcelain --untracked-files=all -- "$destination")" ]]; then
  fail "$destination has uncommitted changes. Review them, then use OVERWRITE=1 only if replacing this page is intended."
fi
mkdir -p "docs/$kind"
replacement=$(mktemp "docs/$kind/.$name.XXXXXXXX")
cp "$generated" "$replacement"
chmod 644 "$replacement"
mv "$replacement" "$destination"
replacement=
echo "==> Updated only $destination. Review its diff and run doc-check."
