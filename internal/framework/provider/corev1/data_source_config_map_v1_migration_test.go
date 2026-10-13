// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const configMapDataSourceName = "data.kubernetes_config_map_v1.test"

// kubeRootCAConfigMapName is published into every namespace by the root-ca-cert-publisher
// controller. The found-case tests read it, as the namespace tests read kube-system,
// instead of creating a ConfigMap in the same config: that makes Terraform read the data
// source at plan time, before the ConfigMap exists, and the apply then fails with
// "Provider produced inconsistent final plan".
const kubeRootCAConfigMapName = "kube-root-ca.crt"

// TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2 — an existing ConfigMap.
// The terraform_data anchor makes any data source attribute change visible as a planned update.
func TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccConfigMapDataSourceAnchoredConfig(kubeRootCAConfigMapName),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccConfigMapDataSourceAnchoredConfig(kubeRootCAConfigMapName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2_withInputs — an existing ConfigMap
// with annotations and labels set in configuration, one label null, which SDKv2 accepted.
// The read reports what the ConfigMap carries, not what was configured.
func TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2_withInputs(t *testing.T) {
	check := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(configMapDataSourceName, "metadata.0.name", kubeRootCAConfigMapName),
		resource.TestCheckResourceAttrSet(configMapDataSourceName, "metadata.0.annotations.kubernetes.io/description"),
		resource.TestCheckResourceAttrSet(configMapDataSourceName, "data.ca.crt"),
		resource.TestCheckNoResourceAttr(configMapDataSourceName, "metadata.0.annotations.anno-key"),
		resource.TestCheckNoResourceAttr(configMapDataSourceName, "metadata.0.labels.label-key"),
		resource.TestCheckNoResourceAttr(configMapDataSourceName, "metadata.0.labels.null-key"),
	)

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccConfigMapDataSourceAnchoredWithInputsConfig(kubeRootCAConfigMapName),
				Check:             check,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccConfigMapDataSourceAnchoredWithInputsConfig(kubeRootCAConfigMapName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: check,
			},
		},
	})
}

// TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2_notFound — missing ConfigMap.
// Pins the SDKv2 state shape then asserts the Framework produces the same shape.
func TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2_notFound(t *testing.T) {
	name := fmt.Sprintf("tf-acc-cm-absent-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	meta := func(field string) tfjsonpath.Path {
		return tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(field)
	}

	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(configMapDataSourceName, tfjsonpath.New("id"), knownvalue.StringExact(fmt.Sprintf("default/%s", name))),
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("name"), knownvalue.StringExact(name)),
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("namespace"), knownvalue.StringExact("default")),
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("uid"), knownvalue.StringExact("")),
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("resource_version"), knownvalue.StringExact("")),
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("generation"), knownvalue.Int64Exact(0)),
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("annotations"), knownvalue.MapSizeExact(0)),
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("labels"), knownvalue.MapSizeExact(0)),
		statecheck.ExpectKnownValue(configMapDataSourceName, tfjsonpath.New("data"), knownvalue.Null()),
		statecheck.ExpectKnownValue(configMapDataSourceName, tfjsonpath.New("binary_data"), knownvalue.Null()),
		statecheck.ExpectKnownValue(configMapDataSourceName, tfjsonpath.New("immutable"), knownvalue.Null()),
	}

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccConfigMapDataSourceNotFoundConfig(name),
				ConfigStateChecks: checks,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccConfigMapDataSourceNotFoundConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: checks,
			},
		},
	})
}

// TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2_notFoundWithAnnotations — missing
// ConfigMap whose config sets annotations/labels; asserts they survive 404.
func TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2_notFoundWithAnnotations(t *testing.T) {
	name := fmt.Sprintf("tf-acc-cm-absent-in-%s", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	meta := func(field string) tfjsonpath.Path {
		return tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(field)
	}

	checks := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("annotations"),
			knownvalue.MapExact(map[string]knownvalue.Check{"anno-key": knownvalue.StringExact("anno-value")})),
		statecheck.ExpectKnownValue(configMapDataSourceName, meta("labels"),
			knownvalue.MapExact(map[string]knownvalue.Check{"label-key": knownvalue.StringExact("label-value")})),
	}

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccConfigMapDataSourceNotFoundWithAnnotationsConfig(name),
				ConfigStateChecks: checks,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccConfigMapDataSourceNotFoundWithAnnotationsConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				ConfigStateChecks: checks,
			},
		},
	})
}

func testAccConfigMapDataSourceAnchoredConfig(name string) string {
	return fmt.Sprintf(`
data "kubernetes_config_map_v1" "test" {
  metadata {
    name = %q
  }
}

resource "terraform_data" "anchor" {
  input = {
    id               = jsonencode(data.kubernetes_config_map_v1.test.id)
    name             = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].name)
    namespace        = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].namespace)
    uid              = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].uid)
    resource_version = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].resource_version)
    generation       = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].generation)
    annotations      = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].annotations)
    labels           = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].labels)
    data             = jsonencode(data.kubernetes_config_map_v1.test.data)
    binary_data      = jsonencode(data.kubernetes_config_map_v1.test.binary_data)
    immutable        = jsonencode(data.kubernetes_config_map_v1.test.immutable)
  }
}
`, name)
}

// testAccConfigMapDataSourceAnchoredWithInputsConfig is testAccConfigMapDataSourceAnchoredConfig
// with both settable metadata maps set, one label null, which SDKv2 accepted on data sources.
func testAccConfigMapDataSourceAnchoredWithInputsConfig(name string) string {
	return fmt.Sprintf(`
data "kubernetes_config_map_v1" "test" {
  metadata {
    name        = %q
    annotations = { "anno-key" = "anno-value" }
    labels      = { "label-key" = "label-value", "null-key" = null }
  }
}

resource "terraform_data" "anchor" {
  input = {
    id               = jsonencode(data.kubernetes_config_map_v1.test.id)
    name             = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].name)
    namespace        = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].namespace)
    uid              = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].uid)
    resource_version = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].resource_version)
    generation       = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].generation)
    annotations      = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].annotations)
    labels           = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].labels)
    data             = jsonencode(data.kubernetes_config_map_v1.test.data)
    binary_data      = jsonencode(data.kubernetes_config_map_v1.test.binary_data)
    immutable        = jsonencode(data.kubernetes_config_map_v1.test.immutable)
  }
}
`, name)
}

func testAccConfigMapDataSourceNotFoundConfig(name string) string {
	return fmt.Sprintf(`
data "kubernetes_config_map_v1" "test" {
  metadata {
    name = %q
  }
}

resource "terraform_data" "anchor" {
  input = {
    id               = jsonencode(data.kubernetes_config_map_v1.test.id)
    name             = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].name)
    namespace        = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].namespace)
    uid              = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].uid)
    resource_version = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].resource_version)
    generation       = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].generation)
    annotations      = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].annotations)
    labels           = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].labels)
    data             = jsonencode(data.kubernetes_config_map_v1.test.data)
    binary_data      = jsonencode(data.kubernetes_config_map_v1.test.binary_data)
    immutable        = jsonencode(data.kubernetes_config_map_v1.test.immutable)
  }
}
`, name)
}

func testAccConfigMapDataSourceNotFoundWithAnnotationsConfig(name string) string {
	return fmt.Sprintf(`
data "kubernetes_config_map_v1" "test" {
  metadata {
    name = %q
    annotations = {
      anno-key = "anno-value"
    }
    labels = {
      label-key = "label-value"
    }
  }
}

resource "terraform_data" "anchor" {
  input = {
    id               = jsonencode(data.kubernetes_config_map_v1.test.id)
    name             = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].name)
    namespace        = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].namespace)
    uid              = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].uid)
    resource_version = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].resource_version)
    generation       = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].generation)
    annotations      = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].annotations)
    labels           = jsonencode(data.kubernetes_config_map_v1.test.metadata[0].labels)
    data             = jsonencode(data.kubernetes_config_map_v1.test.data)
    binary_data      = jsonencode(data.kubernetes_config_map_v1.test.binary_data)
    immutable        = jsonencode(data.kubernetes_config_map_v1.test.immutable)
  }
}
`, name)
}
