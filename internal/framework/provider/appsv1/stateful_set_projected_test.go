// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

func TestAccKubernetesStatefulSetV1_projectedSourceGrouping(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-sts-projected")
	resourceName := "kubernetes_stateful_set_v1.test"
	secret, configMap := name+"-credentials", name+"-settings"
	config := strings.Replace(testAccKubernetesStatefulSetV1ConfigMinimal(name, busyboxImage), "        container {", fmt.Sprintf(`
        volume {
          name = "projected"
          projected {
            sources {
              secret {
                name     = %q
                optional = true
              }
              config_map {
                name     = %q
                optional = true
              }
            }
          }
        }
        container {
          volume_mount {
            name       = "projected"
            mount_path = "/projected"
            read_only  = true
          }`, secret, configMap), 1)
	var before, repaired statefulSetMigrationSnapshot
	groupCheck := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.projected.0.sources.#", "1"),
		resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.projected.0.sources.0.secret.0.name", secret),
		resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.volume.0.projected.0.sources.0.config_map.0.name", configMap),
		func(state *terraform.State) error {
			obj, err := getStatefulSetFromResourceName(state, resourceName)
			if err != nil {
				return err
			}
			volumes := obj.Spec.Template.Spec.Volumes
			if len(volumes) != 1 || volumes[0].Projected == nil {
				return fmt.Errorf("expected the configured projected volume, got %#v", volumes)
			}
			sources := volumes[0].Projected.Sources
			if len(sources) != 2 || sources[0].Secret == nil || sources[0].Secret.Name != secret ||
				sources[1].ConfigMap == nil || sources[1].ConfigMap.Name != configMap {
				return fmt.Errorf("projected API sources differ from configuration: %#v", sources)
			}
			return nil
		},
	)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesStatefulSetV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					groupCheck, statefulSetCaptureMigrationSnapshot(resourceName, &before),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					groupCheck, statefulSetCheckMigrationSnapshot(resourceName, &before),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
			{
				PreConfig: func() {
					client, err := testAccWorkloadClient()
					if err != nil {
						t.Fatal(err)
					}
					patch := fmt.Appendf(nil, `[{"op":"replace","path":"/spec/template/spec/volumes/0/projected/sources/1/configMap/name","value":%q}]`, name+"-drifted")
					if _, err := client.AppsV1().StatefulSets("default").Patch(context.Background(), name, k8stypes.JSONPatchType, patch, metav1.PatchOptions{}); err != nil {
						t.Fatal(err)
					}
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					groupCheck, statefulSetCaptureMigrationSnapshot(resourceName, &repaired),
					func(*terraform.State) error {
						if before.object == nil || repaired.object == nil {
							return fmt.Errorf("missing StatefulSet snapshot")
						}
						if before.object.UID != repaired.object.UID {
							return fmt.Errorf("repairing projected source drift replaced the StatefulSet")
						}
						return nil
					},
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					groupCheck, statefulSetCheckMigrationSnapshot(resourceName, &repaired),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}
