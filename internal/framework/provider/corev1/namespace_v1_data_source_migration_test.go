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

// namespaceMigrationSteps runs config under the published SDKv2 provider, then under the
// local framework implementation, asserting an empty plan and the same state checks after
// both steps — so a pass means the framework matches SDKv2, not merely itself.
func namespaceMigrationSteps(config string, checks []statecheck.StateCheck) []resource.TestStep {
	return []resource.TestStep{
		{
			ExternalProviders: sdkv2ExternalProvider(),
			Config:            config,
			ConfigStateChecks: checks,
		},
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   config,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
			ConfigStateChecks: checks,
		},
	}
}

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2 covers a namespace that exists.
func TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		Steps: namespaceMigrationSteps(withNamespaceAnchor(testAccNamespaceDataSourceConfig("kube-system")), nil),
	})
}

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFound covers a namespace
// that does not exist. The read succeeds under both, and unset metadata must be recorded
// as SDKv2's zero values rather than null: existence checks written as
// metadata[0].uid != "" depend on it, and null would silently invert them.
func TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFound(t *testing.T) {
	// Randomised so a concurrent test cannot create it.
	name := fmt.Sprintf("tf-acc-ns-absent-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	checks := append([]statecheck.StateCheck{
		statecheck.ExpectKnownValue(namespaceDataSourceName, tfjsonpath.New("id"), knownvalue.StringExact(name)),
		statecheck.ExpectKnownValue(namespaceDataSourceName, namespaceMetadataPath("name"), knownvalue.StringExact(name)),
	}, namespaceNotFoundStateChecks()...)

	resource.ParallelTest(t, resource.TestCase{
		Steps: namespaceMigrationSteps(withNamespaceAnchor(testAccNamespaceDataSourceConfig(name)), checks),
	})
}

// TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFoundWithInputs covers a
// missing namespace whose configuration sets annotations and labels. SDKv2 keeps the
// configured values on a 404 but drops null entries, so "null-key" must be absent.
func TestAccKubernetesDataSourceNamespaceV1_MigrateFromSDKv2_notFoundWithInputs(t *testing.T) {
	name := fmt.Sprintf("tf-acc-ns-absent-in-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(namespaceDataSourceName, namespaceMetadataPath("annotations"),
			knownvalue.MapExact(map[string]knownvalue.Check{"anno-key": knownvalue.StringExact("anno-value")})),
		statecheck.ExpectKnownValue(namespaceDataSourceName, namespaceMetadataPath("labels"),
			knownvalue.MapExact(map[string]knownvalue.Check{"label-key": knownvalue.StringExact("label-value")})),
		statecheck.ExpectKnownValue(namespaceDataSourceName, tfjsonpath.New("spec"), knownvalue.Null()),
	}

	resource.ParallelTest(t, resource.TestCase{
		Steps: namespaceMigrationSteps(withNamespaceAnchor(testAccNamespaceDataSourceWithInputsConfig(name)), checks),
	})
}

func namespaceMetadataPath(field string) tfjsonpath.Path {
	return tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(field)
}

// withNamespaceAnchor appends a terraform_data resource capturing every attribute the
// data source exposes.
//
// Only migration tests need this. A data source is re-read on every plan, so its values
// differing between provider versions produces no planned change unless something
// depends on them; the anchor is that dependant. jsonencode keeps null distinguishable
// from "" and {}, which is the distinction these tests turn on.
func withNamespaceAnchor(dataSourceConfig string) string {
	return dataSourceConfig + `

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
