#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


# Check gofmt
echo "==> Checking that manifest code complies with type-assertion requirements..."
manifest_fmt_files=$(gofmt -l `find ./manifest -name '*.go' | grep -v vendor`)
if [[ -n ${manifest_fmt_files} ]]; then
    echo 'manifest_fmt_files needs running on the following files:'
    echo "${gofmt_files}"
    echo "You can use the command: \`make fmt\` to reformat code."
    exit 1
fi

exit 0