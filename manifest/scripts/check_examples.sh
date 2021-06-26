#!/usr/bin/env bash

set -e

TF_IN_AUTOMATION=true
TF_PLUGIN_VERSION="99.0.0"
TF_PLUGIN_BINARY_NAME="terraform-provider-kubernetes-alpha"
TF_PLUGIN_BINARY_PATH="${HOME}/.terraform.d/plugins/registry.terraform.io/hashicorp/kubernetes-alpha/$TF_PLUGIN_VERSION/$(go env GOOS)_$(go env GOARCH)/"

if [ ! -f $TF_PLUGIN_BINARY_PATH ]; then
    mkdir -p $TF_PLUGIN_BINARY_PATH
fi

cp ./terraform-provider-kubernetes-alpha $TF_PLUGIN_BINARY_PATH

SKIP_CHECKS=.skip_checks
for example in $PWD/examples/*; do
    cd $example
    echo 🔍 $(tput bold)$(tput setaf 3)Checking $(basename $example)...
    if [ -f "$SKIP_CHECKS" ]; then
        echo "$SKIP_CHECKS specified. Skipping this example."
        continue
    fi
    terraform init
    terraform validate
    terraform plan -out tfplan > /dev/null
    terraform apply tfplan
    terraform refresh
    terraform destroy -auto-approve
    echo
done