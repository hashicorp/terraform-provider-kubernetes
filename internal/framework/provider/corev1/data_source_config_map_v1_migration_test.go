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

// TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2 — basic ConfigMap, no annotations/labels.
// The terraform_data anchor makes any data source attribute change visible as a planned update.
func TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccConfigMapDataSourceAnchoredConfig(name),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccConfigMapDataSourceAnchoredConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2_withAnnotations — ConfigMap with annotations and labels.
func TestAccKubernetesDataSourceConfigMapV1_MigrateFromSDKv2_withAnnotations(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	check := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(configMapDataSourceName, "metadata.0.name", name),
		resource.TestCheckResourceAttr(configMapDataSourceName, "metadata.0.annotations.TestAnnotation", "value"),
		resource.TestCheckResourceAttr(configMapDataSourceName, "metadata.0.labels.TestLabel", "value"),
		resource.TestCheckResourceAttr(configMapDataSourceName, "data.one", "first"),
	)

	resource.ParallelTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: sdkv2ExternalProvider(),
				Config:            testAccConfigMapDataSourceAnchoredWithAnnotationsConfig(name),
				Check:             check,
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccConfigMapDataSourceAnchoredWithAnnotationsConfig(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check:             check,
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
		statecheck.ExpectKnownValue(configMapDataSourceName, tfjsonpath.New("data"), knownvalue.MapSizeExact(0)),
		statecheck.ExpectKnownValue(configMapDataSourceName, tfjsonpath.New("binary_data"), knownvalue.MapSizeExact(0)),
		statecheck.ExpectKnownValue(configMapDataSourceName, tfjsonpath.New("immutable"), knownvalue.Bool(false)),
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
resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = %q
  }

  data = {
    one = "first"
  }
}

data "kubernetes_config_map_v1" "test" {
  metadata {
    name = kubernetes_config_map_v1.test.metadata[0].name
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

func testAccConfigMapDataSourceAnchoredWithAnnotationsConfig(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = %q
    annotations = {
      TestAnnotation = "value"
    }
    labels = {
      TestLabel = "value"
    }
  }

  data = {
    one = "first"
  }
}

data "kubernetes_config_map_v1" "test" {
  metadata {
    name = kubernetes_config_map_v1.test.metadata[0].name
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
