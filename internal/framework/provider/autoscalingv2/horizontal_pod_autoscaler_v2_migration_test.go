// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

const (
	// hpaSDKv2ProviderVersion is the last release in which kubernetes_horizontal_pod_autoscaler_v2
	// was implemented with SDKv2.
	hpaSDKv2ProviderVersion = "3.5.0"
	// hpaPreIdentityProviderVersion is the last release before resource identity was added.
	hpaPreIdentityProviderVersion = "2.37.1"
)

type hpaMigrationScenario struct {
	name      string
	sdkv2     func(string) string
	framework func(string) string
	checks    []resource.TestCheckFunc
}

func hpaMigrationScenarios() []hpaMigrationScenario {
	return []hpaMigrationScenario{
		{
			name:      "minimal",
			sdkv2:     testAccHorizontalPodAutoscalerV2SDKv2Config_minimal,
			framework: testAccHorizontalPodAutoscalerV2Config_minimal,
			checks: []resource.TestCheckFunc{
				// SDKv2 sent an empty behavior, so Kubernetes filled in the default rules.
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.select_policy", "Max"),
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.#", "1"),
				// SDKv2 stored 0 for the scale-down window that Kubernetes leaves unset.
				resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.stabilization_window_seconds"),
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.resource.target.average_utilization", "80"),
				resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.metric.0.resource.target.value"),
				resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.target_cpu_utilization_percentage"),
			},
		},
		{
			name:      "basic",
			sdkv2:     testAccHorizontalPodAutoscalerV2SDKv2Config_basic,
			framework: testAccHorizontalPodAutoscalerV2Config_basic,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.#", "4"),
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.1.external.metric.selector.match_labels.queue_name", "test"),
				resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.metric.1.external.metric.selector.match_expressions"),
				resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.metric.1.resource"),
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.3.object.described_object.kind", "Ingress"),
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.stabilization_window_seconds", "300"),
			},
		},
		{
			name:      "modified",
			sdkv2:     testAccHorizontalPodAutoscalerV2SDKv2Config_modified,
			framework: testAccHorizontalPodAutoscalerV2Config_modified,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.select_policy", "Disabled"),
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.stabilization_window_seconds", "0"),
			},
		},
		{
			name:      "containerResource",
			sdkv2:     testAccHorizontalPodAutoscalerV2SDKv2Config_containerResource,
			framework: testAccHorizontalPodAutoscalerV2Config_containerResource,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.container_resource.container", "test"),
			},
		},
		{
			name:      "selector",
			sdkv2:     testAccHorizontalPodAutoscalerV2SDKv2Config_selector,
			framework: testAccHorizontalPodAutoscalerV2Config_selector,
			checks: []resource.TestCheckFunc{
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.metric.selector.match_expressions.#", "2"),
				resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.metric.selector.match_expressions.0.values.#", "2"),
				resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.metric.selector.match_expressions.1.values"),
			},
		},
	}
}

func TestAccKubernetesHorizontalPodAutoscalerV2_UpgradeFromSDKv2(t *testing.T) {
	for _, sc := range hpaMigrationScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			var uid string
			name := randomName()
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccHorizontalPodAutoscalerCheckDestroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: externalProvider(hpaSDKv2ProviderVersion),
						Config:            sc.sdkv2(name),
						Check:             testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   sc.framework(name),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							append(sc.checks, testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid))...,
						),
					},
					{
						ResourceName:             hpaV2ResourceName,
						ImportState:              true,
						ImportStateVerify:        true,
						ImportStateVerifyIgnore:  []string{"metadata.0.resource_version"},
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					},
				},
			})
		})
	}
}

// State written before resource identity existed gets its identity on the first refresh.
func TestAccKubernetesHorizontalPodAutoscalerV2_UpgradeFromPreIdentity(t *testing.T) {
	var uid string
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccHorizontalPodAutoscalerCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				ExternalProviders: externalProvider(hpaPreIdentityProviderVersion),
				Config:            testAccHorizontalPodAutoscalerV2SDKv2Config_basic(name),
				Check:             testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccHorizontalPodAutoscalerV2Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(hpaV2ResourceName, map[string]knownvalue.Check{
						"namespace":   knownvalue.StringExact("default"),
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("autoscaling/v2"),
						"kind":        knownvalue.StringExact("HorizontalPodAutoscaler"),
					}),
				},
				Check: testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid),
			},
		},
	})
}

func testAccHorizontalPodAutoscalerV2SDKv2Config_minimal(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q

    annotations = {
      test = "test"
    }

    labels = {
      test = "test"
    }
  }

  spec {
    max_replicas = 10

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = "TerraformAccTest"
    }
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV2SDKv2Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q

    annotations = {
      test = "test"
    }

    labels = {
      test = "test"
    }
  }

  spec {
    max_replicas = 10

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = "TerraformAccTest"
    }

    behavior {
      scale_down {
        stabilization_window_seconds = 300
        select_policy                = "Min"

        policy {
          period_seconds = 120
          type           = "Pods"
          value          = 1
        }

        policy {
          period_seconds = 310
          type           = "Percent"
          value          = 100
        }
      }

      scale_up {
        stabilization_window_seconds = 600
        select_policy                = "Max"

        policy {
          period_seconds = 180
          type           = "Percent"
          value          = 100
        }

        policy {
          period_seconds = 600
          type           = "Pods"
          value          = 5
        }
      }
    }

    metric {
      type = "Resource"
      resource {
        name = "test"
        target {
          type                = "Utilization"
          average_utilization = 1
        }
      }
    }

    metric {
      type = "External"
      external {
        metric {
          name = "queue_size"
          selector {
            match_labels = {
              queue_name = "test"
            }
          }
        }
        target {
          type  = "Value"
          value = "10"
        }
      }
    }

    metric {
      type = "Pods"
      pods {
        metric {
          name = "packets-per-second"
        }
        target {
          type          = "AverageValue"
          average_value = "1k"
        }
      }
    }

    metric {
      type = "Object"
      object {
        metric {
          name = "requests-per-second"
        }
        described_object {
          kind        = "Ingress"
          name        = "main-route"
          api_version = "networking.k8s.io/v1beta1"
        }
        target {
          type          = "AverageValue"
          average_value = "2k"
        }
      }
    }
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV2SDKv2Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q

    annotations = {
      test = "test"
    }

    labels = {
      test = "test"
    }
  }

  spec {
    min_replicas = 50
    max_replicas = 100

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = "TerraformAccTest"
    }

    behavior {
      scale_down {
        stabilization_window_seconds = 100
        select_policy                = "Max"

        policy {
          period_seconds = 120
          type           = "Pods"
          value          = 10
        }
      }

      scale_up {
        select_policy = "Disabled"

        policy {
          period_seconds = 60
          type           = "Pods"
          value          = 1
        }
      }
    }

    metric {
      type = "External"
      external {
        metric {
          name = "latency"
          selector {
            match_labels = {
              lb_name = "test"
            }
          }
        }
        target {
          type  = "Value"
          value = "100"
        }
      }
    }
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV2SDKv2Config_containerResource(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q

    annotations = {
      test = "test"
    }

    labels = {
      test = "test"
    }
  }

  spec {
    min_replicas = 50
    max_replicas = 100

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = "TerraformAccTest"
    }

    metric {
      type = "ContainerResource"
      container_resource {
        name      = "cpu"
        container = "test"
        target {
          type                = "Utilization"
          average_utilization = "75"
        }
      }
    }
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV2SDKv2Config_selector(name string) string {
	return fmt.Sprintf(`resource "kubernetes_horizontal_pod_autoscaler_v2" "test" {
  metadata {
    name = %q
  }

  spec {
    max_replicas = 10

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = "TerraformAccTest"
    }

    metric {
      type = "External"
      external {
        metric {
          name = "queue_size"
          selector {
            match_labels = {
              queue = "jobs"
            }
            match_expressions {
              key      = "tier"
              operator = "In"
              values   = ["a", "b"]
            }
            match_expressions {
              key      = "canary"
              operator = "DoesNotExist"
            }
          }
        }
        target {
          type  = "Value"
          value = "10"
        }
      }
    }
  }
}
`, name)
}
