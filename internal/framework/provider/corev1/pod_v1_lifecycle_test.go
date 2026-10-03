// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
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

// Kubernetes accepts only a few Pod spec updates, such as setting or lowering
// active_deadline_seconds. Those happen in place; any other spec change,
// including fields SDKv2 left updatable, replaces the Pod.
func TestAccKubernetesPodV1_specUpdateOrReplace(t *testing.T) {
	var previous, current api.Pod
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	step := func(deadline int, mountPath string, action plancheck.ResourceActionType) resource.TestStep {
		return resource.TestStep{
			Config: testAccKubernetesPodV1ConfigSpecUpdate(name, busyboxImage, deadline, mountPath),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, action)},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				func(*terraform.State) error { previous = current; return nil },
				testAccCheckKubernetesPodV1ActiveDeadline(resourceName, &current, deadline),
				testAccCheckKubernetesPodForceNew(&previous, &current, action == plancheck.ResourceActionReplace),
			),
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPodV1PreCheck(t) },
		ProtoV6ProviderFactories: testAccProviderFactories,
		CheckDestroy:             testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodV1ConfigSpecUpdate(name, busyboxImage, 0, "/data"),
				Check:  testAccCheckKubernetesPodV1Exists(resourceName, &current),
			},
			step(3600, "/data", plancheck.ResourceActionUpdate),
			step(1800, "/data", plancheck.ResourceActionUpdate),
			step(3600, "/data", plancheck.ResourceActionReplace),
			step(0, "/data", plancheck.ResourceActionReplace),
			step(0, "/other", plancheck.ResourceActionReplace),
		},
	})
}

// testAccCheckKubernetesPodV1ActiveDeadline waits until the API serves the Pod
// with the given deadline, so the next step plans against it.
func testAccCheckKubernetesPodV1ActiveDeadline(resourceName string, pod *api.Pod, seconds int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var got int64
		err := wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, true, func(context.Context) (bool, error) {
			if err := testAccCheckKubernetesPodV1Exists(resourceName, pod)(s); err != nil {
				return false, err
			}
			got = ptr.Deref(pod.Spec.ActiveDeadlineSeconds, 0)
			return got == int64(seconds), nil
		})
		if err != nil {
			return fmt.Errorf("activeDeadlineSeconds = %d, want %d: %w", got, seconds, err)
		}
		return nil
	}
}

func testAccKubernetesPodV1ConfigSpecUpdate(name, imageName string, deadline int, mountPath string) string {
	activeDeadline := ""
	if deadline != 0 {
		activeDeadline = fmt.Sprintf("active_deadline_seconds = %d", deadline)
	}
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    %s
    termination_grace_period_seconds = 1
    security_context {
      supplemental_groups = []
    }
    container {
      image   = %q
      name    = "containername"
      command = ["sleep", "3600"]
      volume_mount {
        name       = "data"
        mount_path = %q
      }
    }
    volume {
      name = "data"
      empty_dir {}
    }
  }
}
`, name, activeDeadline, imageName, mountPath)
}

// 3.3.0 sent an explicit pod-level runAsNonRoot false. Removing the block can
// only reach the Pod by replacing it.
func TestAccKubernetesPodV1_upgradeRemoveZeroSecurityContext(t *testing.T) {
	var created, upgraded, replaced api.Pod
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	securityContext := `security_context {
      run_as_non_root = false
    }`

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config: testAccKubernetesPodV1ConfigSecurityContext(name, busyboxImage, securityContext),
				Check:  testAccCheckKubernetesPodV1Exists(resourceName, &created),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccKubernetesPodV1ConfigSecurityContext(name, busyboxImage, securityContext),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &upgraded),
					testAccCheckKubernetesPodForceNew(&created, &upgraded, false),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   testAccKubernetesPodV1ConfigSecurityContext(name, busyboxImage, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &replaced),
					testAccCheckKubernetesPodForceNew(&upgraded, &replaced, true),
				),
			},
		},
	})
}

// Kubernetes stores run_as_user as a number. A configured "01000" is kept as
// spelled, and state that 3.3.0 recorded as "1000" for it plans no change.
func TestAccKubernetesPodV1_numberSpelling(t *testing.T) {
	var created, upgraded, replaced api.Pod
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_pod_v1.test"
	runAsUser := func(user string) string {
		return testAccKubernetesPodV1ConfigSecurityContext(name, busyboxImage, fmt.Sprintf(`security_context {
      run_as_user = %q
    }`, user))
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config: runAsUser("01000"),
				Check:  testAccCheckKubernetesPodV1Exists(resourceName, &created),
				// 3.3.0 replaces the Pod on every plan for this spelling.
				ExpectNonEmptyPlan: true,
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   runAsUser("01000"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionNoop)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &upgraded),
					testAccCheckKubernetesPodForceNew(&created, &upgraded, false),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   runAsUser("02000"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionReplace)},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodV1Exists(resourceName, &replaced),
					testAccCheckKubernetesPodForceNew(&upgraded, &replaced, true),
					resource.TestCheckResourceAttr(resourceName, "spec.0.security_context.0.run_as_user", "02000"),
				),
			},
		},
	})
}

// State written by 3.3.0 plans no change before its first refresh.
func TestAccKubernetesPodV1_upgradeWithoutRefresh(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-test")
	config := testAccKubernetesPodV1ConfigSecurityContext(name, busyboxImage, "")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:             func() { testAccPodV1PreCheck(t) },
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{Plan: resource.PlanOptions{NoRefresh: true}},
		CheckDestroy:         testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config: config,
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}

// 3.3.0 fails every generate_name Pod create and leaves a tainted instance
// without a name; the upgrade takes the name from the id and replaces it.
func TestAccKubernetesPodV1_upgradeTaintedGeneratedName(t *testing.T) {
	prefix := fmt.Sprintf("tf-acc-test-%s-", acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	config := testAccKubernetesPodV1ConfigGeneratedName(prefix, busyboxImage, "initial")
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { testAccPodV1PreCheck(t) },
		CheckDestroy: testAccCheckKubernetesPodV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= 3.3.0"},
				},
				Config:      config,
				ExpectError: regexp.MustCompile("resource name may not be empty"),
			},
			{
				ProtoV6ProviderFactories: testAccProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("kubernetes_pod_v1.test", plancheck.ResourceActionReplace),
				}},
			},
		},
	})
}

func testAccKubernetesPodV1ConfigSecurityContext(name, imageName, securityContext string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    termination_grace_period_seconds = 1
    %s
    container {
      image   = %q
      name    = "containername"
      command = ["sleep", "3600"]
    }
  }
}
`, name, securityContext, imageName)
}
