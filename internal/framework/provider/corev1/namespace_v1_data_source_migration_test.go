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

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_withInputs covers a namespace
// that exists, with annotations and labels set in configuration — one label null, which
// SDKv2 accepted. The read reports what the namespace carries, not what was configured.
func TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_withInputs(t *testing.T) {
	check := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(namespaceDataSourceName, "id", "kube-system"),
		resource.TestCheckResourceAttrSet(namespaceDataSourceName, "metadata.0.uid"),
		resource.TestCheckResourceAttr(namespaceDataSourceName, "metadata.0.labels.kubernetes.io/metadata.name", "kube-system"),
		resource.TestCheckNoResourceAttr(namespaceDataSourceName, "metadata.0.labels.label-key"),
		resource.TestCheckNoResourceAttr(namespaceDataSourceName, "metadata.0.labels.null-key"),
		resource.TestCheckNoResourceAttr(namespaceDataSourceName, "metadata.0.annotations.anno-key"),
	)

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccNamespaceDataSourceAnchoredWithInputsConfig("kube-system"),
				Check:             check,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccNamespaceDataSourceAnchoredWithInputsConfig("kube-system"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: check,
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFound covers a namespace
// that does not exist. Step 1 pins what SDKv2 records, step 2 that the framework records
// the same — unset metadata as zero values ({} maps, "" strings, 0 generation) and a null
// spec — so the plan stays empty. Existence checks written as metadata[0].uid != ""
// depend on those zero values.
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
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: frameworkShape,
			},
		},
	})
}

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFoundWithInputs covers a
// missing namespace whose configuration sets annotations and labels. SDKv2 keeps the
// configured values on a 404 but drops null entries, so null-key must be absent.
func TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFoundWithInputs(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ns-absent-in-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	meta := func(field string) tfjsonpath.Path {
		return tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(field)
	}

	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("annotations"),
			knownvalue.MapExact(map[string]knownvalue.Check{"anno-key": knownvalue.StringExact("anno-value")})),
		statecheck.ExpectKnownValue(namespaceDataSourceName, meta("labels"),
			knownvalue.MapExact(map[string]knownvalue.Check{"label-key": knownvalue.StringExact("label-value")})),
		statecheck.ExpectKnownValue(namespaceDataSourceName, tfjsonpath.New("spec"), knownvalue.Null()),
	}

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccNamespaceDataSourceAnchoredWithInputsConfig(name),
				ConfigStateChecks: checks,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccNamespaceDataSourceAnchoredWithInputsConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: checks,
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

// testAccNamespaceDataSourceAnchoredWithInputsConfig is testAccNamespaceDataSourceAnchoredConfig
// with both settable metadata maps set, one label null, which SDKv2 accepted on data sources.
func testAccNamespaceDataSourceAnchoredWithInputsConfig(name string) string {
	return fmt.Sprintf(`
data "kubernetes_namespace_v1" "test" {
  metadata {
    name        = %q
    annotations = { "anno-key" = "anno-value" }
    labels      = { "label-key" = "label-value", "null-key" = null }
  }
}

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
}`, name)
}
