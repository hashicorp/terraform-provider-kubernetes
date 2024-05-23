#!/bin/bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0


GOOS=darwin GOARCH=amd64 go build -v -o build/terraform-provider-kubernetes_darwin-amd64
GOOS=linux GOARCH=amd64 go build -v -o build/terraform-provider-kubernetes_linux-amd64
GOOS=windows GOARCH=amd64 go build -v -o build/terraform-provider-kubernetes_windows-amd64

gzip build/*
