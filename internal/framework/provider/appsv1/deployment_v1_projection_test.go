// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8types "k8s.io/apimachinery/pkg/types"
)

func TestAccDeploymentV1_GroupedProjectedSources(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-apps-deploy-projection")
	t.Logf("Deployment: default/%s", name)
	config := testAccDeploymentGroupedProjectionConfig(name)
	address := "kubernetes_deployment_v1.test"
	sourcePath := "spec.0.template.0.spec.0.volume.0.projected.0.sources"
	var before, restored appsv1.Deployment
	var snapshot deploymentMigrationSnapshot
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesDeploymentV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(address, &before),
					testAccDeploymentMigrationCapture(address, &snapshot),
					resource.TestCheckResourceAttr(address, sourcePath+".#", "1"),
					resource.TestCheckResourceAttr(address, sourcePath+".0.secret.0.name", name+"-secret"),
					resource.TestCheckResourceAttr(address, sourcePath+".0.config_map.0.name", name+"-config"),
				),
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: testAccDeploymentMigrationUnchanged(address, &snapshot),
			},
			{
				PreConfig: func() {
					client, err := testAccWorkloadClient()
					if err != nil {
						t.Fatal(err)
					}
					for i, volume := range before.Spec.Template.Spec.Volumes {
						if volume.Name != "projected" || volume.Projected == nil {
							continue
						}
						for j, source := range volume.Projected.Sources {
							if source.ConfigMap == nil {
								continue
							}
							patch := fmt.Sprintf(`[{"op":"remove","path":"/spec/template/spec/volumes/%d/projected/sources/%d"}]`, i, j)
							_, err := client.AppsV1().Deployments(before.Namespace).Patch(context.Background(), before.Name, k8types.JSONPatchType, []byte(patch), metav1.PatchOptions{})
							if err != nil {
								t.Fatal(err)
							}
							return
						}
					}
					t.Fatal("expected an API config-map projection to remove")
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDeploymentV1Exists(address, &restored),
					resource.TestCheckResourceAttr(address, sourcePath+".#", "1"),
					resource.TestCheckResourceAttr(address, sourcePath+".0.config_map.0.name", name+"-config"),
					func(*terraform.State) error {
						if before.UID != restored.UID || !reflect.DeepEqual(before.Spec, restored.Spec) {
							return fmt.Errorf("projection drift repair replaced the Deployment or failed to restore its full spec")
						}
						return nil
					},
				),
				ConfigStateChecks: deploymentIdentityChecks(address, &snapshot),
			},
		},
	})
}

func testAccDeploymentGroupedProjectionConfig(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_deployment_v1" "test" {
  metadata { name = "%[1]s" }
  spec {
    selector { match_labels = { app = "%[1]s" } }
    template {
      metadata { labels = { app = "%[1]s" } }
      spec {
        container {
          name  = "pause"
          image = "registry.k8s.io/pause:3.10"
          volume_mount {
            name       = "projected"
            mount_path = "/projected"
          }
        }
        volume {
          name = "projected"
          projected {
            sources {
              secret {
                name     = "%[1]s-secret"
                optional = true
              }
              config_map {
                name     = "%[1]s-config"
                optional = true
              }
            }
          }
        }
      }
    }
  }
}
`, name)
}
