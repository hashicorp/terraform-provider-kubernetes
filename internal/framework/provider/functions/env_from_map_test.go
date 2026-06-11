// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package functions_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestEnvFromMap(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testEnvFromMapConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The list is sorted by key, so GREETING comes before NAME
					// regardless of the order the map literal was written in.
					resource.TestCheckOutput("length", "2"),
					resource.TestCheckOutput("name_0", "GREETING"),
					resource.TestCheckOutput("value_0", "Hello from the environment"),
					resource.TestCheckOutput("name_1", "NAME"),
					resource.TestCheckOutput("value_1", "Kubernetes"),
				),
			},
		},
	})
}

func TestEnvFromMapEmpty(t *testing.T) {
	t.Parallel()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testEnvFromMapEmptyConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("length", "0"),
				),
			},
		},
	})
}

func testEnvFromMapConfig() string {
	return `
locals {
  env = provider::kubernetes::env_from_map({
    NAME     = "Kubernetes"
    GREETING = "Hello from the environment"
  })
}

output "length" {
  value = tostring(length(local.env))
}

output "name_0" {
  value = local.env[0].name
}

output "value_0" {
  value = local.env[0].value
}

output "name_1" {
  value = local.env[1].name
}

output "value_1" {
  value = local.env[1].value
}`
}

func testEnvFromMapEmptyConfig() string {
	return `
locals {
  env = provider::kubernetes::env_from_map({})
}

output "length" {
  value = tostring(length(local.env))
}`
}
