// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package useragent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/meta"
)

func TestBuild(t *testing.T) {
	//nolint:staticcheck // matches the deprecated call the SDK itself makes
	sdkVersion := meta.SDKVersionString()

	cases := map[string]struct {
		terraformVersion string
		appendUserAgent  string
		expected         string
	}{
		"standard": {
			terraformVersion: "1.9.5",
			expected: fmt.Sprintf(
				"Terraform/1.9.5 (+https://www.terraform.io) Terraform-Plugin-SDK/%s terraform-provider-kubernetes/dev",
				sdkVersion),
		},
		"appended": {
			terraformVersion: "1.9.5",
			appendUserAgent:  "MyCompany/1.2.3",
			expected: fmt.Sprintf(
				"Terraform/1.9.5 (+https://www.terraform.io) Terraform-Plugin-SDK/%s terraform-provider-kubernetes/dev MyCompany/1.2.3",
				sdkVersion),
		},
		"appended value is trimmed": {
			terraformVersion: "1.9.5",
			appendUserAgent:  "  MyCompany/1.2.3\n",
			expected: fmt.Sprintf(
				"Terraform/1.9.5 (+https://www.terraform.io) Terraform-Plugin-SDK/%s terraform-provider-kubernetes/dev MyCompany/1.2.3",
				sdkVersion),
		},
		"unknown terraform version": {
			terraformVersion: "",
			expected: fmt.Sprintf(
				"Terraform/ (+https://www.terraform.io) Terraform-Plugin-SDK/%s terraform-provider-kubernetes/dev",
				sdkVersion),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("TF_APPEND_USER_AGENT", tc.appendUserAgent)

			got := Build(tc.terraformVersion)
			if got != tc.expected {
				t.Fatalf("unexpected User-Agent\n got: %q\nwant: %q", got, tc.expected)
			}
		})
	}
}

func TestBuildReportsProviderVersion(t *testing.T) {
	original := ProviderVersion
	t.Cleanup(func() { ProviderVersion = original })

	ProviderVersion = "3.2.1"

	got := Build("1.9.5")
	if want := "terraform-provider-kubernetes/3.2.1"; !strings.HasSuffix(got, want) {
		t.Fatalf("User-Agent %q does not end with %q", got, want)
	}
}
