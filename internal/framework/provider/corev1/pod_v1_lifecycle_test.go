// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesPodV1_generatedNameLifecycle(t *testing.T) {
	var before, after api.Pod
	resourceName := "kubernetes_pod_v1.test"
	prefix := fmt.Sprintf("tf-acc-test-%s-", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPodV1PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigGeneratedName(prefix, busyboxImage, "initial"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &before),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
				),
			},
			{
				Config: testAccKubernetesPodV1ConfigGeneratedName(prefix, busyboxImage, "updated"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &after),
					testAccCheckKubernetesPodForceNew(&before, &after, false),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.managed", "updated"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func testAccKubernetesPodV1ConfigGeneratedName(prefix, imageName, label string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    generate_name = %q
    labels = {
      managed = %q
    }
  }

  spec {
    container {
      image = %q
      name  = "containername"
    }
  }
}
`, prefix, label, imageName)
}

// An explicit "" on an API-defaulted string leaves the value to the API, as it
// did in SDKv2: no inconsistent result on create, a refresh records the live
// values, and configuring those values later neither changes nor replaces the Pod.
func TestAccKubernetesPodV1_emptyAPIDefaultedStrings(t *testing.T) {
	var before, after api.Pod
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	empty := `
    service_account_name = ""
    scheduler_name       = ""
    node_name            = ""
    hostname             = ""`
	live := `
    service_account_name = "default"
    scheduler_name       = "default-scheduler"`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPodV1PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigEmptyAPIDefaultedStrings(name, busyboxImage, "initial", empty, ""),
				Check:  testAccCheckKubernetesPodV1Exists(resourceName, &before),
			},
			{
				RefreshState: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image_pull_policy", "IfNotPresent"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.service_account_name", "default"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.scheduler_name", "default-scheduler"),
					resource.TestMatchResourceAttr(resourceName, "spec.0.node_name", regexp.MustCompile(".")),
				),
			},
			{
				Config: testAccKubernetesPodV1ConfigEmptyAPIDefaultedStrings(name, busyboxImage, "initial", live, "IfNotPresent"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccKubernetesPodV1ConfigEmptyAPIDefaultedStrings(name, busyboxImage, "updated", empty, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &after),
					testAccCheckKubernetesPodForceNew(&before, &after, false),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.step", "updated"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.container.0.image_pull_policy", "IfNotPresent"),
				),
			},
		},
	})
}

func testAccKubernetesPodV1ConfigEmptyAPIDefaultedStrings(name, imageName, label, podFields, pullPolicy string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %q
    labels = {
      step = %q
    }
  }
  spec {%s
    container {
      image             = %q
      name              = "containername"
      image_pull_policy = %q
    }
  }
}
`, name, label, podFields, imageName, pullPolicy)
}

// One sources block may hold several projections; Kubernetes stores them as
// separate entries, and the configured grouping is kept in state.
func TestAccKubernetesPodV1_projectedVolumeGroupedSources(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPodV1PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigGroupedProjection(name, busyboxImage),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.0.config_map.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume.0.projected.0.sources.0.secret.0.name", name),
				),
			},
		},
	})
}

func testAccKubernetesPodV1ConfigGroupedProjection(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    termination_grace_period_seconds = 1
    container {
      image   = %[2]q
      name    = "containername"
      command = ["sleep", "3600"]
      volume_mount {
        name       = "projected"
        mount_path = "/projected"
        read_only  = true
      }
    }
    volume {
      name = "projected"
      projected {
        sources {
          config_map {
            name     = %[1]q
            optional = true
          }
          secret {
            name     = %[1]q
            optional = true
          }
        }
        sources {
          service_account_token {
            path = "token"
          }
        }
      }
    }
  }
}
`, name, imageName)
}
