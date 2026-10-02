// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

func daemonSetRegressionConfig(name, label, containerExtra, podExtra, strategy string) string {
	config := daemonSetFrameworkConfig(label, busyboxImage, containerExtra, podExtra, strategy)
	config = strings.ReplaceAll(config, "${NAME}", name)
	config = strings.ReplaceAll(config, "daemonset-migration", name)
	config = strings.Replace(config, "      metadata {\n", "      metadata {\n        annotations = {}\n", 1)
	return daemonSetNoRolloutConfig(config)
}

func daemonSetNoRolloutConfig(config string) string {
	return strings.Replace(config, "  metadata {", "  wait_for_rollout = false\n  metadata {", 1)
}

func daemonSetRegressionPlan(action plancheck.ResourceActionType) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{
		PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(daemonSetResourceName, action)},
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
}

func testAccCheckDaemonSetSpecUnchanged(before, after *appsv1.DaemonSet) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if !reflect.DeepEqual(before.Spec, after.Spec) {
			return fmt.Errorf("daemonset desired specification changed during a no-op migration")
		}
		if before.Generation != after.Generation {
			return fmt.Errorf("daemonset generation changed during a no-op migration: %d -> %d", before.Generation, after.Generation)
		}
		return nil
	}
}

func TestAccDaemonSetV1_GeneratedNameAndRemoval(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-ds-generated")
	config := func(label, extra, strategy string) string {
		return strings.Replace(daemonSetRegressionConfig(name, label, extra, "", strategy),
			fmt.Sprintf("name = %q", name), fmt.Sprintf("generate_name = %q", name+"-"), 1)
	}
	strategy := `strategy = [{ type = "OnDelete" }]`
	env := `env {
  name  = "MANAGED"
  value = "present"
}`
	initial := config("before", env, strategy)
	metadataUpdate := config("after", env, strategy)
	removed := config("after", "", strategy)
	unmanagedStrategy := config("unmanaged-strategy", "", "")
	var before, after appsv1.DaemonSet
	check := resource.ComposeAggregateTestCheckFunc(
		testAccCheckDaemonSetExists(daemonSetResourceName, &after),
		testAccCheckDaemonSetNotRecreated(&before, &after),
		testAccCheckDaemonSetObject(&after, checkOnDeleteClearsRollingUpdate()),
	)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDaemonSetDestroy,
		Steps: []resource.TestStep{
			{Config: initial, Check: testAccCheckDaemonSetExists(daemonSetResourceName, &before)},
			{Config: metadataUpdate, ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate), Check: check},
			{
				Config: removed, ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(check, resource.TestCheckResourceAttr(daemonSetResourceName, "spec.0.template.0.spec.0.container.0.env.#", "0")),
			},
			{
				Config: unmanagedStrategy, ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate),
				Check: check,
			},
			{
				ResourceName: daemonSetResourceName, ImportState: true, ImportStateVerify: true,
			},
			{Config: unmanagedStrategy, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}},
		},
	})
}

func TestAccDaemonSetV1_DriftAndDisappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-ds-drift")
	config := daemonSetRegressionConfig(name, "drift", `args = ["3600"]
env {
  name  = "MANAGED"
  value = "present"
}`, `node_selector = { "kubernetes.io/os" = "linux" }`, "")
	var before, after appsv1.DaemonSet
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDaemonSetDestroy,
		Steps: []resource.TestStep{
			{Config: config, Check: testAccCheckDaemonSetExists(daemonSetResourceName, &before)},
			{
				PreConfig: func() {
					client, err := testAccWorkloadClient()
					if err != nil {
						t.Fatal(err)
					}
					patch := []byte(`[{"op":"remove","path":"/spec/template/spec/containers/0/args"},{"op":"remove","path":"/spec/template/spec/containers/0/env"},{"op":"remove","path":"/spec/template/spec/nodeSelector"}]`)
					if _, err := client.AppsV1().DaemonSets("default").Patch(context.Background(), name, k8stypes.JSONPatchType, patch, metav1.PatchOptions{}); err != nil {
						t.Fatal(err)
					}
				},
				Config: config, ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDaemonSetExists(daemonSetResourceName, &after),
					testAccCheckDaemonSetNotRecreated(&before, &after),
					testAccCheckDaemonSetObject(&after, func(obj *appsv1.DaemonSet) error {
						if !reflect.DeepEqual(obj.Spec.Template.Spec.Containers, before.Spec.Template.Spec.Containers) ||
							!reflect.DeepEqual(obj.Spec.Template.Spec.NodeSelector, before.Spec.Template.Spec.NodeSelector) {
							return fmt.Errorf("daemonset API drift was not repaired")
						}
						return nil
					}),
				),
			},
			{
				PreConfig: func() {
					client, err := testAccWorkloadClient()
					if err != nil {
						t.Fatal(err)
					}
					if err := client.AppsV1().DaemonSets("default").Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
						t.Fatal(err)
					}
				},
				Config: config, ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDaemonSetExists(daemonSetResourceName, &after),
					testAccCheckDaemonSetObject(&after, func(obj *appsv1.DaemonSet) error {
						if obj.UID == before.UID {
							return fmt.Errorf("out-of-band deletion did not create a new daemonset")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccDaemonSetV1_ProjectedSourceGrouping(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-ds-projected")
	config := daemonSetRegressionConfig(name, "projected", "", `
volume {
  name = "projected"
  projected {
    sources {
      secret {
        name     = "credentials"
        optional = true
      }
      config_map {
        name     = "settings"
        optional = true
      }
    }
  }
}`, "")
	var before, after appsv1.DaemonSet
	check := resource.ComposeAggregateTestCheckFunc(
		testAccCheckDaemonSetExists(daemonSetResourceName, &after),
		testAccCheckDaemonSetNotRecreated(&before, &after),
		resource.TestCheckResourceAttr(daemonSetResourceName, "spec.0.template.0.spec.0.volume.0.projected.0.sources.#", "1"),
	)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  testAccCheckDaemonSetExists(daemonSetResourceName, &before),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: config, Check: check,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				PreConfig: func() {
					client, err := testAccWorkloadClient()
					if err != nil {
						t.Fatal(err)
					}
					patch := []byte(`[{"op":"replace","path":"/spec/template/spec/volumes/0/projected/sources/1/configMap/name","value":"changed"}]`)
					if _, err := client.AppsV1().DaemonSets("default").Patch(context.Background(), name, k8stypes.JSONPatchType, patch, metav1.PatchOptions{}); err != nil {
						t.Fatal(err)
					}
				},
				Config: config, Check: check,
				ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate),
			},
		},
	})
}

func TestAccDaemonSetV1_EmptyPrimitiveCollectionsLifecycle(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-ds-empty")
	config := func(mode string) string {
		root, template, pod, container := "", "", "", ""
		switch mode {
		case "managed":
			root = `annotations = { managed = "yes" }
labels = { managed = "yes" }`
			template = `annotations = { managed = "yes" }`
			pod = `node_selector = { "kubernetes.io/os" = "linux" }`
			container = `command = ["sleep"]
args = ["300"]`
		case "empty":
			root = "annotations = {}\nlabels = {}"
			template = "annotations = {}"
			pod = "node_selector = {}"
			container = "command = []\nargs = []"
		}
		return fmt.Sprintf(`
resource "kubernetes_daemon_set_v1" "test" {
  wait_for_rollout = false
  metadata {
    name = %q
    %s
  }
  spec {
    selector {
      match_labels = {
        app = %q
      }
    }
    template {
      metadata {
        labels = {
          app = %q
        }
        %s
      }
      spec {
        %s
        container {
          name  = "app"
          image = %q
          %s
        }
      }
    }
  }
}
`, name, root, name, name, template, pod, busyboxImage, container)
	}
	stateChecks := func(empty bool) []statecheck.StateCheck {
		var mapCheck, listCheck knownvalue.Check = knownvalue.Null(), knownvalue.Null()
		if empty {
			mapCheck = knownvalue.MapExact(map[string]knownvalue.Check{})
			listCheck = knownvalue.ListExact([]knownvalue.Check{})
		}
		metadata := tfjsonpath.New("metadata").AtSliceIndex(0)
		template := tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0)
		podPath := func() tfjsonpath.Path {
			return tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("template").AtSliceIndex(0).AtMapKey("spec").AtSliceIndex(0)
		}
		container := podPath().AtMapKey("container").AtSliceIndex(0)
		return []statecheck.StateCheck{
			statecheck.ExpectKnownValue(daemonSetResourceName, metadata.AtMapKey("annotations"), mapCheck),
			statecheck.ExpectKnownValue(daemonSetResourceName, metadata.AtMapKey("labels"), mapCheck),
			statecheck.ExpectKnownValue(daemonSetResourceName, template.AtMapKey("metadata").AtSliceIndex(0).AtMapKey("annotations"), mapCheck),
			statecheck.ExpectKnownValue(daemonSetResourceName, podPath().AtMapKey("node_selector"), mapCheck),
			statecheck.ExpectKnownValue(daemonSetResourceName, container.AtMapKey("args"), listCheck),
			statecheck.ExpectKnownValue(daemonSetResourceName, container.AtMapKey("command"), listCheck),
		}
	}
	var before, after appsv1.DaemonSet
	unchangedIdentity := resource.ComposeAggregateTestCheckFunc(
		testAccCheckDaemonSetExists(daemonSetResourceName, &after),
		testAccCheckDaemonSetNotRecreated(&before, &after),
	)
	removed := resource.ComposeAggregateTestCheckFunc(unchangedIdentity,
		testAccCheckDaemonSetObject(&after, func(obj *appsv1.DaemonSet) error {
			pod := obj.Spec.Template.Spec
			_, annotation := obj.Annotations["managed"]
			_, label := obj.Labels["managed"]
			_, templateAnnotation := obj.Spec.Template.Annotations["managed"]
			if annotation || label || templateAnnotation ||
				len(pod.NodeSelector) != 0 || len(pod.Containers[0].Args) != 0 || len(pod.Containers[0].Command) != 0 {
				return fmt.Errorf("omitted nonempty collections remain in the API")
			}
			return nil
		}),
	)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDaemonSetDestroy,
		Steps: []resource.TestStep{
			{Config: config("omitted"), ConfigStateChecks: stateChecks(false), Check: testAccCheckDaemonSetExists(daemonSetResourceName, &before)},
			{Config: config("managed"), ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate), Check: unchangedIdentity},
			{Config: config("omitted"), ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate), ConfigStateChecks: stateChecks(false), Check: removed},
			{Config: config("empty"), ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate), ConfigStateChecks: stateChecks(true), Check: removed},
			{
				Config: config("omitted"), ConfigStateChecks: stateChecks(true), Check: removed,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccDaemonSetV1_IgnoredMetadataSurvivesUpdates(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-ds-metadata")
	provider := `
provider "kubernetes" {
  ignore_annotations = ["^example[.]com/external$"]
  ignore_labels      = ["^example[.]com/external$"]
}
`
	initial := daemonSetRegressionConfig(name, "before", "", "", "") + provider
	initial = strings.Replace(initial, "  metadata {\n", "  metadata {\n    annotations = { \"example.com/managed\" = \"before\" }\n", 1)
	updated := strings.Replace(initial, "  spec {\n", "  spec {\n    min_ready_seconds = 1\n", 1)
	removed := strings.Replace(updated, "    annotations = { \"example.com/managed\" = \"before\" }\n", "", 1)
	var before, after appsv1.DaemonSet
	check := func(managed bool) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			testAccCheckDaemonSetExists(daemonSetResourceName, &after),
			testAccCheckDaemonSetNotRecreated(&before, &after),
			testAccCheckDaemonSetObject(&after, func(obj *appsv1.DaemonSet) error {
				if obj.Annotations["example.com/external"] != "kept" || obj.Labels["example.com/external"] != "kept" {
					return fmt.Errorf("ignored metadata was lost during an update")
				}
				_, exists := obj.Annotations["example.com/managed"]
				if exists != managed {
					return fmt.Errorf("managed annotation present=%t, want %t", exists, managed)
				}
				return nil
			}),
		)
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckDaemonSetDestroy,
		Steps: []resource.TestStep{
			{Config: initial, Check: testAccCheckDaemonSetExists(daemonSetResourceName, &before)},
			{
				PreConfig: func() {
					client, err := testAccWorkloadClient()
					if err != nil {
						t.Fatal(err)
					}
					patch := []byte(`{"metadata":{"annotations":{"example.com/external":"kept"},"labels":{"example.com/external":"kept"}}}`)
					if _, err := client.AppsV1().DaemonSets("default").Patch(context.Background(), name, k8stypes.MergePatchType, patch, metav1.PatchOptions{}); err != nil {
						t.Fatal(err)
					}
				},
				Config: updated, ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate), Check: check(true),
			},
			{Config: removed, ConfigPlanChecks: daemonSetRegressionPlan(plancheck.ResourceActionUpdate), Check: check(false)},
		},
	})
}

func TestAccDaemonSetV1_UpgradeFromPreIdentityProvider(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-ds-v0")
	sdkConfig := strings.ReplaceAll(daemonSetSDKConfigWithContainerExtra("v0", busyboxImage, sdkContainerResourcesBlock), "${NAME}", name)
	localConfig := strings.ReplaceAll(daemonSetFrameworkConfigWithContainerExtra("v0", busyboxImage, frameworkContainerResourcesAttribute), "${NAME}", name)
	var before, after appsv1.DaemonSet
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		CheckDestroy: testAccCheckDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "2.23.0"},
				},
				Config: sdkConfig, Check: testAccCheckDaemonSetExists(daemonSetResourceName, &before),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   localConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}, PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckDaemonSetExists(daemonSetResourceName, &after),
					testAccCheckDaemonSetNotRecreated(&before, &after),
					testAccCheckDaemonSetSpecUnchanged(&before, &after),
					testAccCheckDaemonSetStateIdentity(daemonSetResourceName, &after),
				),
			},
		},
	})
}
