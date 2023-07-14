#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

echo "==> Checking for unchecked errors..."

if ! which errcheck > /dev/null; then
    echo "==> Installing errcheck..."
    go install github.com/kisielk/errcheck@latest
fi

err_files=$($(go env GOPATH)/bin/errcheck -exclude scripts/errcheck_excludes.txt \
                     -verbose \
                     -ignoretests \
                     -ignore 'github.com/hashicorp/terraform/helper/schema:Set' \
                     -ignore 'bytes:.*' \
                     -ignore 'io:Close|Write' \
                     -asserts ./manifest/.../ \
                     )

if [[ -n ${err_files} ]]; then
    echo 'Unchecked errors found in the following places:'
    echo "${err_files}"
    echo "Please handle returned errors. You can check directly with \`make errcheck\`"
    exit 1
fi

exit 0
