// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package useragent builds the User-Agent header the provider sends to the
// Kubernetes API server.
package useragent

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ProviderName is the reporting name of this provider, as it appears in the
// User-Agent header.
const ProviderName = "terraform-provider-kubernetes"

// ProviderVersion is the provider version reported in the User-Agent header.
// It defaults to "dev" and is set by main to the version stamped into the
// binary at build time.
var ProviderVersion = "dev"

// Build returns the standard Terraform User-Agent string for the given
// Terraform version, in the form:
//
//	Terraform/<tf-version> (+https://www.terraform.io) Terraform-Plugin-SDK/<sdk-version> terraform-provider-kubernetes/<provider-version>
//
// If TF_APPEND_USER_AGENT is set, its value is appended to the result.
func Build(terraformVersion string) string {
	// The generated string is owned by the plugin SDK so that this provider
	// reports the same format, and honours the same environment variables, as
	// every other Terraform provider. UserAgent only reads TerraformVersion
	// off the Provider, so a bare Provider is enough to call it -- this lets
	// the manifest provider, which is not built on the SDK, report the same
	// User-Agent as the SDKv2 provider.
	p := &schema.Provider{TerraformVersion: terraformVersion}
	return p.UserAgent(ProviderName, ProviderVersion)
}
