// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	autoscalingv2api "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const hpaV2ResourceName = "kubernetes_horizontal_pod_autoscaler_v2.test"

func randomName() string {
	return "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
}

func TestAccKubernetesHorizontalPodAutoscalerV2_minimal(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_minimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, nil),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.target_cpu_utilization_percentage"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					// Kubernetes defaults an omitted metric list to 80% average CPU utilization.
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.type", "Resource"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.resource.name", "cpu"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.resource.target.type", "Utilization"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.resource.target.average_utilization", "80"),
					resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.behavior"),
				),
			},
			{
				ResourceName:            hpaV2ResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				// The stored behavior is null, so the omitted select_policy and windows are
				// unknown until Kubernetes fills them in.
				Config: testAccHorizontalPodAutoscalerV2Config_behavior(name, `{
      scale_down = {
        policy = [{ period_seconds = 60, type = "Pods", value = 2 }]
      }
    }`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(hpaV2ResourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(hpaV2ResourceName,
							tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("behavior").AtMapKey("scale_down").AtMapKey("select_policy")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, nil),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.select_policy", "Max"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.#", "2"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV2_identity(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(hpaV2ResourceName, map[string]knownvalue.Check{
						"namespace":   knownvalue.StringExact("default"),
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("autoscaling/v2"),
						"kind":        knownvalue.StringExact("HorizontalPodAutoscaler"),
					}),
				},
			},
			{
				ResourceName:    hpaV2ResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				ResourceName:    hpaV2ResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV2_basic(t *testing.T) {
	var uid string
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.#", "2"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.0.period_seconds", "120"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.0.type", "Pods"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.0.value", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.1.period_seconds", "310"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.1.type", "Percent"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.1.value", "100"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.select_policy", "Min"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.stabilization_window_seconds", "300"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.#", "2"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.0.period_seconds", "180"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.0.type", "Percent"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.0.value", "100"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.1.period_seconds", "600"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.1.type", "Pods"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.1.value", "5"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.select_policy", "Max"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.stabilization_window_seconds", "600"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.max_replicas", "10"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.min_replicas", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.#", "4"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.type", "Resource"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.resource.name", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.resource.target.type", "Utilization"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.resource.target.average_utilization", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.1.type", "External"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.1.external.metric.name", "queue_size"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.1.external.metric.selector.match_labels.queue_name", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.1.external.target.type", "Value"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.1.external.target.value", "10"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.2.type", "Pods"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.2.pods.metric.name", "packets-per-second"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.2.pods.target.type", "AverageValue"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.2.pods.target.average_value", "1k"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.3.type", "Object"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.3.object.metric.name", "requests-per-second"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.3.object.target.type", "AverageValue"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.3.object.target.average_value", "2k"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.3.object.described_object.kind", "Ingress"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.3.object.described_object.name", "main-route"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.3.object.described_object.api_version", "networking.k8s.io/v1beta1"),
				),
			},
			{
				ResourceName:            hpaV2ResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccHorizontalPodAutoscalerV2Config_modified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(hpaV2ResourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.0.period_seconds", "120"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.0.type", "Pods"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.0.value", "10"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.select_policy", "Max"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.stabilization_window_seconds", "100"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.select_policy", "Disabled"),
					// The omitted window is Optional+Computed and keeps its stored value.
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.stabilization_window_seconds", "600"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.0.period_seconds", "60"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.0.type", "Pods"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.0.value", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.max_replicas", "100"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.min_replicas", "50"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.type", "External"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.metric.name", "latency"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.metric.selector.match_labels.lb_name", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.target.type", "Value"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.target.value", "100"),
				),
			},
			{
				// metric and behavior are Optional+Computed, so removing them keeps the stored values.
				Config: testAccHorizontalPodAutoscalerV2Config_scaleOnly(name, 50, 100),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.metric.name", "latency"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.select_policy", "Disabled"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV2_containerResource(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		ErrorCheck: func(err error) error {
			t.Skipf("HPAContainerMetrics feature might not be enabled on the cluster and therefore this step will be skipped if an error occurs. Refer to the error for more details:\n%s", err)
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_containerResource(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, nil),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.annotations.test", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.labels.test", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(hpaV2ResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.min_replicas", "50"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.max_replicas", "100"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.api_version", "apps/v1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.kind", "Deployment"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.scale_target_ref.0.name", "TerraformAccTest"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.type", "ContainerResource"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.container_resource.name", "cpu"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.container_resource.container", "test"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.container_resource.target.type", "Utilization"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.container_resource.target.average_utilization", "75"),
				),
			},
		},
	})
}

// SDKv2 sent select_policy = "" when it was omitted, which Kubernetes rejects.
func TestAccKubernetesHorizontalPodAutoscalerV2_scaleDownWithoutSelectPolicy(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_behavior(name, `{
      scale_down = {
        policy = [{ period_seconds = 60, type = "Pods", value = 2 }]
      }
    }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, nil),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.select_policy", "Max"),
					// Kubernetes leaves the scale-down window unset and applies 300 seconds at runtime.
					resource.TestCheckNoResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.stabilization_window_seconds"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.select_policy", "Max"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.stabilization_window_seconds", "0"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.#", "2"),
				),
			},
			{
				ResourceName:            hpaV2ResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

// An empty behavior object asks Kubernetes for its default scaling rules.
func TestAccKubernetesHorizontalPodAutoscalerV2_emptyBehavior(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_behavior(name, "{}"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, nil),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.#", "2"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.0.type", "Pods"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.0.value", "4"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.1.type", "Percent"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_up.policy.1.value", "100"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.#", "1"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.policy.0.type", "Percent"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.behavior.scale_down.select_policy", "Max"),
				),
			},
		},
	})
}

// Kubernetes canonicalises quantities, so "1000m" is read back as "1".
func TestAccKubernetesHorizontalPodAutoscalerV2_quantity(t *testing.T) {
	var uid string
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_quantity(name, "1000m", "0.5"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid),
					testAccHorizontalPodAutoscalerCheckSpec(hpaV2ResourceName, func(spec autoscalingv2api.HorizontalPodAutoscalerSpec) error {
						if got := spec.Metrics[0].External.Target.Value.String(); got != "1" {
							return fmt.Errorf("expected Kubernetes to store external target value 1, got %q", got)
						}
						return nil
					}),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.target.value", "1000m"),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.1.pods.target.average_value", "0.5"),
				),
			},
			{
				Config: testAccHorizontalPodAutoscalerV2Config_quantity(name, "1", "500m"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccHorizontalPodAutoscalerV2Config_quantity(name, "2", "500m"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(hpaV2ResourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, &uid),
					resource.TestCheckResourceAttr(hpaV2ResourceName, "spec.0.metric.0.external.target.value", "2"),
				),
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV2_selector(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_selector(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(hpaV2ResourceName,
						tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("metric").AtSliceIndex(0).AtMapKey("external").AtMapKey("metric").AtMapKey("selector"),
						knownvalue.ObjectExact(map[string]knownvalue.Check{
							"match_labels": knownvalue.MapExact(map[string]knownvalue.Check{"queue": knownvalue.StringExact("jobs")}),
							"match_expressions": knownvalue.ListExact([]knownvalue.Check{
								knownvalue.ObjectExact(map[string]knownvalue.Check{
									"key":      knownvalue.StringExact("tier"),
									"operator": knownvalue.StringExact("In"),
									"values":   knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact("a"), knownvalue.StringExact("b")}),
								}),
								knownvalue.ObjectExact(map[string]knownvalue.Check{
									"key":      knownvalue.StringExact("canary"),
									"operator": knownvalue.StringExact("DoesNotExist"),
									"values":   knownvalue.Null(),
								}),
							}),
						})),
				},
			},
			{
				ResourceName:            hpaV2ResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV2_disappears(t *testing.T) {
	name := randomName()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccHorizontalPodAutoscalerCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccHorizontalPodAutoscalerV2Config_minimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccHorizontalPodAutoscalerCheckExists(hpaV2ResourceName, nil),
					testAccHorizontalPodAutoscalerDelete(hpaV2ResourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccKubernetesHorizontalPodAutoscalerV2_invalidConfig(t *testing.T) {
	name := randomName()
	cases := map[string]struct {
		behavior string
		metric   string
		err      string
	}{
		"metricType": {
			metric: `[{ type = "Unknown" }]`,
			err:    `value must be one of`,
		},
		"emptyMetric": {
			metric: `[]`,
			err:    `must be omitted or null`,
		},
		"emptyPolicy": {
			behavior: `{ scale_up = { policy = [] } }`,
			err:      `must contain at least 1`,
		},
		"selectPolicy": {
			behavior: `{ scale_up = { select_policy = "Fastest", policy = [{ period_seconds = 60, type = "Pods", value = 1 }] } }`,
			err:      `value must be one of`,
		},
	}
	for caseName, tc := range cases {
		t.Run(caseName, func(t *testing.T) {
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      testAccHorizontalPodAutoscalerV2Config_raw(name, tc.metric, tc.behavior),
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(tc.err),
					},
				},
			})
		})
	}
}

func testAccHorizontalPodAutoscalerDelete(address string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %q not found in state", address)
		}
		namespace, name, _ := strings.Cut(rs.Primary.ID, "/")
		conn, err := testAccClient()
		if err != nil {
			return err
		}
		return conn.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

func testAccHorizontalPodAutoscalerCheckSpec(address string, check func(autoscalingv2api.HorizontalPodAutoscalerSpec) error) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %q not found in state", address)
		}
		namespace, name, _ := strings.Cut(rs.Primary.ID, "/")
		conn, err := testAccClient()
		if err != nil {
			return err
		}
		hpa, err := conn.AutoscalingV2().HorizontalPodAutoscalers(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		return check(hpa.Spec)
	}
}

func testAccHorizontalPodAutoscalerV2Config_minimal(name string) string {
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

func testAccHorizontalPodAutoscalerV2Config_basic(name string) string {
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

    behavior = {
      scale_down = {
        stabilization_window_seconds = 300
        select_policy                = "Min"

        policy = [
          {
            period_seconds = 120
            type           = "Pods"
            value          = 1
          },
          {
            period_seconds = 310
            type           = "Percent"
            value          = 100
          },
        ]
      }

      scale_up = {
        stabilization_window_seconds = 600
        select_policy                = "Max"

        policy = [
          {
            period_seconds = 180
            type           = "Percent"
            value          = 100
          },
          {
            period_seconds = 600
            type           = "Pods"
            value          = 5
          },
        ]
      }
    }

    metric = [
      {
        type = "Resource"
        resource = {
          name = "test"
          target = {
            type                = "Utilization"
            average_utilization = 1
          }
        }
      },
      {
        type = "External"
        external = {
          metric = {
            name = "queue_size"
            selector = {
              match_labels = {
                queue_name = "test"
              }
            }
          }
          target = {
            type  = "Value"
            value = "10"
          }
        }
      },
      {
        type = "Pods"
        pods = {
          metric = {
            name = "packets-per-second"
          }
          target = {
            type          = "AverageValue"
            average_value = "1k"
          }
        }
      },
      {
        type = "Object"
        object = {
          metric = {
            name = "requests-per-second"
          }
          described_object = {
            kind        = "Ingress"
            name        = "main-route"
            api_version = "networking.k8s.io/v1beta1"
          }
          target = {
            type          = "AverageValue"
            average_value = "2k"
          }
        }
      },
    ]
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV2Config_modified(name string) string {
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

    behavior = {
      scale_down = {
        stabilization_window_seconds = 100
        select_policy                = "Max"

        policy = [
          {
            period_seconds = 120
            type           = "Pods"
            value          = 10
          },
        ]
      }

      scale_up = {
        select_policy = "Disabled"

        policy = [
          {
            period_seconds = 60
            type           = "Pods"
            value          = 1
          },
        ]
      }
    }

    metric = [
      {
        type = "External"
        external = {
          metric = {
            name = "latency"
            selector = {
              match_labels = {
                lb_name = "test"
              }
            }
          }
          target = {
            type  = "Value"
            value = "100"
          }
        }
      },
    ]
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV2Config_scaleOnly(name string, minReplicas, maxReplicas int) string {
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
    min_replicas = %d
    max_replicas = %d

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = "TerraformAccTest"
    }
  }
}
`, name, minReplicas, maxReplicas)
}

func testAccHorizontalPodAutoscalerV2Config_containerResource(name string) string {
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

    metric = [
      {
        type = "ContainerResource"
        container_resource = {
          name      = "cpu"
          container = "test"
          target = {
            type                = "Utilization"
            average_utilization = 75
          }
        }
      },
    ]
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV2Config_behavior(name, behavior string) string {
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

    behavior = %s
  }
}
`, name, behavior)
}

func testAccHorizontalPodAutoscalerV2Config_quantity(name, value, averageValue string) string {
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

    metric = [
      {
        type = "External"
        external = {
          metric = {
            name = "queue_size"
          }
          target = {
            type  = "Value"
            value = %q
          }
        }
      },
      {
        type = "Pods"
        pods = {
          metric = {
            name = "packets-per-second"
          }
          target = {
            type          = "AverageValue"
            average_value = %q
          }
        }
      },
    ]
  }
}
`, name, value, averageValue)
}

func testAccHorizontalPodAutoscalerV2Config_selector(name string) string {
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

    metric = [
      {
        type = "External"
        external = {
          metric = {
            name = "queue_size"
            selector = {
              match_labels = {
                queue = "jobs"
              }
              match_expressions = [
                {
                  key      = "tier"
                  operator = "In"
                  values   = ["a", "b"]
                },
                {
                  key      = "canary"
                  operator = "DoesNotExist"
                },
              ]
            }
          }
          target = {
            type  = "Value"
            value = "10"
          }
        }
      },
    ]
  }
}
`, name)
}

func testAccHorizontalPodAutoscalerV2Config_raw(name, metric, behavior string) string {
	var extra string
	if metric != "" {
		extra += "\n    metric = " + metric
	}
	if behavior != "" {
		extra += "\n    behavior = " + behavior
	}
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
%s
  }
}
`, name, extra)
}
