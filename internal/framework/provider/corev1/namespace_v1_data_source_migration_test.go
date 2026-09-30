// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// sdkv2ExternalProvider is step 1 of every migration test here: the last released
// SDKv2 provider, pinned by the sdkv2ProviderVersion constant this package already
// defines for the resource migration tests.
func sdkv2ExternalProvider() map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {
			VersionConstraint: sdkv2ProviderVersion,
			Source:            "hashicorp/kubernetes",
		},
	}
}

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2 establishes state with the
// published SDKv2 provider, then runs the identical config against the local framework
// implementation. The empty plan proves the attribute values, captured in a
// terraform_data anchor, are unchanged.
func TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccNamespaceDataSourceAnchoredConfig("kube-system"),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccNamespaceDataSourceAnchoredConfig("kube-system"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFound records the one
// place this migration deliberately does not preserve SDKv2's state, so it expects a
// changed plan rather than an empty one. Both sides are pinned: step 1 asserts what
// SDKv2 writes, step 2 what the framework writes. Anyone restoring parity will fail
// this test and have to change it on purpose.
//
// SDKv2 returns nil before either d.Set call, so its state is a flatmap artifact:
// metadata was in the diff and gets zero-filled ({} maps, "" strings, 0 generation),
// while spec was never set and so comes back null. The framework records the requested
// name and leaves the rest null.
//
// Practitioner impact, also in the changelog: the SDKv2 idiom for detecting absence is
// metadata[0].uid != "", and HCL treats null as equal only to null, so that expression
// flips from false to true.
func TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFound(t *testing.T) {
	// Randomised so a concurrent test cannot create it, and held in a variable so both
	// steps read the same name.
	name := fmt.Sprintf("tf-acc-ns-absent-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	meta := func(field string) tfjsonpath.Path {
		return tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(field)
	}

	// id and metadata.name agree: the framework sets the synthetic ID before the API
	// call, as SDKv2's d.SetId does. spec is null under both, for different reasons.
	sdkv2Shape := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(namespaceDataSourceName, tfjsonpath.New("id"), knownvalue.StringExact(name)),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("name"), knownvalue.StringExact(name)),
		statecheck.ExpectKnownValue(namespaceDataSourceName, tfjsonpath.New("spec"), knownvalue.Null()),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("uid"), knownvalue.StringExact("")),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("resource_version"), knownvalue.StringExact("")),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("generation"), knownvalue.Int64Exact(0)),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("annotations"), knownvalue.MapSizeExact(0)),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("labels"), knownvalue.MapSizeExact(0)),
	}

	frameworkShape := append([]statecheck.StateCheck{
		statecheck.ExpectKnownValue(namespaceDataSourceName, tfjsonpath.New("id"), knownvalue.StringExact(name)),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("name"), knownvalue.StringExact(name)),
	}, namespaceNotFoundStateChecks()...)

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				// That this step applies at all is part of the check: the SDKv2 read
				// must not error on a 404.
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccNamespaceDataSourceAnchoredConfig(name),
				ConfigStateChecks: sdkv2Shape,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccNamespaceDataSourceAnchoredConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("terraform_data.anchor", plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: frameworkShape,
			},
		},
	})
}

// testAccNamespaceDataSourceAnchoredConfig is testAccNamespaceDataSourceConfig plus a
// terraform_data resource capturing every attribute the data source exposes.
//
// Only migration tests need this. A data source is re-read on every plan, so its values
// differing between provider versions produces no planned change unless something
// depends on them; the anchor is that dependant. jsonencode keeps null distinguishable
// from "" and {}, which is the distinction these tests turn on, and keeps the anchor's
// object type stable when values go null.
func testAccNamespaceDataSourceAnchoredConfig(name string) string {
	return testAccNamespaceDataSourceConfig(name) + `

resource "terraform_data" "anchor" {
  input = {
    id               = jsonencode(data.kubernetes_namespace_v1.test.id)
    name             = jsonencode(data.kubernetes_namespace_v1.test.metadata[0].name)
    uid              = jsonencode(data.kubernetes_namespace_v1.test.metadata[0].uid)
    resource_version = jsonencode(data.kubernetes_namespace_v1.test.metadata[0].resource_version)
    generation       = jsonencode(data.kubernetes_namespace_v1.test.metadata[0].generation)
    annotations      = jsonencode(data.kubernetes_namespace_v1.test.metadata[0].annotations)
    labels           = jsonencode(data.kubernetes_namespace_v1.test.metadata[0].labels)
    spec             = jsonencode(data.kubernetes_namespace_v1.test.spec)
  }
}`
}
