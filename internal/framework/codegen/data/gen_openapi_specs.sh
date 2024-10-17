#!/bin/bash

# This script does a sparse checkout of the main Kubernetes repo for the specified tag
# and downloads all of the OpenAPIv3 specification files.

if [[ $# -eq 0 ]]; then
    exit "error: you must specify the tag of the Kubernetes version, e.g. $0 v1.28.3"
fi

git clone -b $1 -n --depth=1 --filter=tree:0 https://github.com/kubernetes/kubernetes kubernetes-$1
cd kubernetes-$1
git sparse-checkout set --no-cone api/openapi-spec/v3
git checkout
rm -rf .git