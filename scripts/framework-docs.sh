#!/usr/bin/env bash
# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0

# Render the registry pages of Plugin Framework resources with tfplugindocs.
# Plain tfplugindocs lists Framework blocks without item counts, so required
# blocks show as optional. This renders from a schema annotated by
# scripts/frameworkdocs and copies back only the pages named as arguments.
#
#   scripts/framework-docs.sh deployment_v1 pod_v1
set -o errexit
set -o nounset
set -o pipefail

if [ $# -eq 0 ]; then
  set -- deployment_v1 daemon_set_v1 stateful_set_v1 pod_v1 job_v1 cron_job_v1
fi

TFPLUGINDOCS=${TFPLUGINDOCS:-go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0}
root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cd "$root"
go build -o "$tmp/bin/terraform-provider-kubernetes" .

cat > "$tmp/terraformrc" <<EOF
provider_installation {
  dev_overrides { "hashicorp/kubernetes" = "$tmp/bin" }
  direct {}
}
EOF
mkdir "$tmp/config"
cat > "$tmp/config/main.tf" <<EOF
terraform {
  required_providers {
    kubernetes = { source = "hashicorp/kubernetes" }
  }
}
EOF
(cd "$tmp/config" && TF_CLI_CONFIG_FILE="$tmp/terraformrc" terraform providers schema -json) > "$tmp/schema.json"
go run ./scripts/frameworkdocs < "$tmp/schema.json" > "$tmp/docs-schema.json"

mkdir "$tmp/site"
cp -R templates examples "$tmp/site/"
$TFPLUGINDOCS generate --provider-dir "$tmp/site" --provider-name kubernetes --providers-schema "$tmp/docs-schema.json" > "$tmp/tfplugindocs.log" 2>&1 ||
  { cat "$tmp/tfplugindocs.log"; exit 1; }

for page in "$@"; do
  cp "$tmp/site/docs/resources/$page.md" "docs/resources/$page.md"
done
