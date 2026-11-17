// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv1_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// hpaSDKv2ProviderVersion is the last release in which kubernetes_horizontal_pod_autoscaler_v1
// was implemented with SDKv2.
const hpaSDKv2ProviderVersion = "3.5.0"

type hpaMigrationScenario struct {
	name   string
	config func(string) string
	// importable is false when import cannot recover the configured value, as with generate_name.
	importable bool
}

func hpaMigrationScenarios() []hpaMigrationScenario {
	return []hpaMigrationScenario{
		{name: "basic", config: testAccHorizontalPodAutoscalerV1Config_basic, importable: true},
		{name: "specModified", config: testAccHorizontalPodAutoscalerV1Config_specModified, importable: true},
		{name: "targetCPU", config: func(name string) string {
			return testAccHorizontalPodAutoscalerV1Config_targetCPU(name, 2, 50)
		}, importable: true},
		{name: "generatedName", config: func(string) string {
			return testAccHorizontalPodAutoscalerV1Config_generatedName("tf-acc-test-")
		}},
	}
}

func TestAccKubernetesHorizontalPodAutoscalerV1_UpgradeFromSDKv2(t *testing.T) {
	for _, sc := range hpaMigrationScenarios() {
		t.Run(sc.name, func(t *testing.T) {
			var uid string
			config := sc.config("tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			steps := []resource.TestStep{
				{
					ExternalProviders: externalProvider(hpaSDKv2ProviderVersion),
					Config:            config,
					Check:             testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
				},
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
					Check: testAccHorizontalPodAutoscalerCheckExists(hpaV1ResourceName, &uid),
				},
			}
			if sc.importable {
				steps = append(steps, resource.TestStep{
					ResourceName:             hpaV1ResourceName,
					ImportState:              true,
					ImportStateVerify:        true,
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				})
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccHorizontalPodAutoscalerCheckDestroy,
				Steps:        steps,
			})
		})
	}
}
