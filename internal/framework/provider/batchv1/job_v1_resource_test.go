// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/kubetest"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccKubernetesJobV1_wait_for_completion(t *testing.T) {
	var conf batchv1.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := kubetest.BusyboxImage
	resourceName := "kubernetes_job_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesJobV1Config_wait_for_completion(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// NOTE this is to check that Terraform actually waited for the Job to complete
					// before considering the Job resource as created
					testAccCheckJobV1Waited(time.Duration(10)*time.Second),
					testAccCheckKubernetesJobV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "wait_for_completion", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesJobV1_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := kubetest.BusyboxImage
	resourceName := "kubernetes_job_v1.test"
	// Import cannot discover this provider-only setting from Kubernetes.
	config := strings.Replace(testAccKubernetesJobV1Config_basic(name, imageName),
		"wait_for_completion = false", "wait_for_completion = true", 1)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},

		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName, map[string]knownvalue.Check{
							"namespace":   knownvalue.StringExact("default"),
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("batch/v1"),
							"kind":        knownvalue.StringExact("Job"),
						},
					),
				},
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesJobV1_basic(t *testing.T) {
	var conf batchv1.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := kubetest.BusyboxImage
	resourceName := "kubernetes_job_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			kubetest.PreCheck(t)
			kubetest.SkipIfClusterVersionLessThan(t, "1.26.0")
		},

		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesJobV1Config_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.active_deadline_seconds", "120"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backoff_limit", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.completions", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parallelism", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "wait_for_completion", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.action", "FailJob"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.container_name", "hello"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.0", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.1", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.2", "42"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.1.action", "Ignore"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.1.on_pod_condition.0.type", "DisruptionTarget"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.1.on_pod_condition.0.status", "False"),
				),
			},
			{
				Config: testAccKubernetesJobV1Config_modified(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.active_deadline_seconds", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backoff_limit", "6"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.completions", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parallelism", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.manual_selector", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "wait_for_completion", "false"),
				),
			},
		},
	})
}

func TestAccKubernetesJobV1_update(t *testing.T) {
	var conf1, conf2, conf3 batchv1.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := kubetest.BusyboxImage
	imageName1 := kubetest.AgnhostImage
	resourceName := "kubernetes_job_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			kubetest.PreCheck(t)
			kubetest.SkipIfClusterVersionLessThan(t, "1.26.0")
		},

		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesJobV1Config_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.active_deadline_seconds", "120"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backoff_limit", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.completions", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parallelism", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr(resourceName, "wait_for_completion", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.manual_selector", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.action", "FailJob"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.container_name", "hello"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.0", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.1", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.2", "42"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.1.action", "Ignore"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.1.on_pod_condition.0.type", "DisruptionTarget"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.pod_failure_policy.0.rule.1.on_pod_condition.0.status", "False"),
				),
			},
			{
				Config: testAccKubernetesJobV1Config_updateMutableFields(name, imageName, "121", "4", "false", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.active_deadline_seconds", "121"),
					testAccCheckKubernetesJobV1ForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesJobV1Config_updateMutableFields(name, imageName, "121", "5", "false", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.backoff_limit", "5"),
					testAccCheckKubernetesJobV1ForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesJobV1Config_updateMutableFields(name, imageName, "121", "5", "true", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.manual_selector", "true"),
					testAccCheckKubernetesJobV1ForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesJobV1Config_updateMutableFields(name, imageName, "121", "5", "true", "3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.parallelism", "3"),
					testAccCheckKubernetesJobV1ForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesJobV1Config_updateImmutableFields(name, imageName, "6"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.completions", "6"),
					testAccCheckKubernetesJobV1ForceNew(&conf1, &conf2, true),
				),
			},
			{
				Config: testAccKubernetesJobV1Config_updateImmutableFields(name, imageName1, "6"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf3),
					resource.TestCheckResourceAttr(resourceName, "spec.0.template.0.spec.0.container.0.image", imageName1),
					testAccCheckKubernetesJobV1ForceNew(&conf2, &conf3, true),
				),
			},
		},
	})
}

func TestAccKubernetesJobV1_pod_failure_policy_update(t *testing.T) {
	name := fmt.Sprintf("tf-acc-policy-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	initial := strings.Replace(testAccKubernetesJobV1Config_basic(name, kubetest.BusyboxImage),
		`container_name = "hello"`, "", 1)
	updated := strings.Replace(initial, `action = "FailJob"`, `action = "Ignore"`, 1)
	var before, after batchv1.Job
	const address = "kubernetes_job_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			kubetest.PreCheck(t)
			kubetest.SkipIfClusterVersionLessThan(t, "1.26.0")
		},
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(address, &before),
					resource.TestCheckResourceAttr(address, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.container_name", ""),
				),
			},
			{
				Config: updated,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(address, &after),
					testAccCheckKubernetesJobV1ForceNew(&before, &after, true),
					resource.TestCheckResourceAttr(address, "spec.0.pod_failure_policy.0.rule.0.action", "Ignore"),
					resource.TestCheckResourceAttr(address, "spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.container_name", ""),
				),
			},
			{
				Config: updated,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccKubernetesJobV1_removeTemplateValue(t *testing.T) {
	var before, after batchv1.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	const address = "kubernetes_job_v1.test"
	withSubPath := testAccKubernetesJobV1Config_volumeMount(name, kubetest.BusyboxImage, `sub_path = "data"`)
	withoutSubPath := testAccKubernetesJobV1Config_volumeMount(name, kubetest.BusyboxImage, "")
	emptySubPath := testAccKubernetesJobV1Config_volumeMount(name, kubetest.BusyboxImage, `sub_path = ""`)
	noEscalation := strings.Replace(emptySubPath, `command = ["sh", "-c", "true"]`, `command = ["sh", "-c", "true"]
          security_context {
            allow_privilege_escalation = false
          }`, 1)
	podNonRoot := strings.Replace(noEscalation, `restart_policy = "Never"`, `restart_policy = "Never"
        security_context {
          run_as_non_root = false
        }`, 1)
	// A Job's pod template cannot change, so each of these replaces the Job.
	step := func(config string, check func(corev1.PodSpec) error) resource.TestStep {
		return resource.TestStep{
			Config: config,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)},
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
			Check: resource.ComposeAggregateTestCheckFunc(
				testAccCheckKubernetesJobV1Exists(address, &after),
				testAccCheckKubernetesJobV1ForceNew(&before, &after, true),
				func(*terraform.State) error {
					before = after
					return check(after.Spec.Template.Spec)
				},
			),
		}
	}
	subPath := func(want string) func(corev1.PodSpec) error {
		return func(spec corev1.PodSpec) error {
			if got := spec.Containers[0].VolumeMounts[0].SubPath; got != want {
				return fmt.Errorf("live subPath = %q, want %q", got, want)
			}
			return nil
		}
	}
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: withSubPath,
				Check:  testAccCheckKubernetesJobV1Exists(address, &before),
			},
			step(withoutSubPath, subPath("")),
			step(withSubPath, subPath("data")),
			step(emptySubPath, subPath("")),
			step(podNonRoot, func(spec corev1.PodSpec) error {
				if sc := spec.Containers[0].SecurityContext; sc == nil || sc.AllowPrivilegeEscalation == nil || *sc.AllowPrivilegeEscalation {
					return fmt.Errorf("live securityContext = %+v, want allowPrivilegeEscalation false", sc)
				}
				if sc := spec.SecurityContext; sc == nil || sc.RunAsNonRoot == nil || *sc.RunAsNonRoot {
					return fmt.Errorf("live pod securityContext = %+v, want runAsNonRoot false", sc)
				}
				return nil
			}),
			step(noEscalation, func(spec corev1.PodSpec) error {
				if sc := spec.SecurityContext; sc != nil && sc.RunAsNonRoot != nil {
					return fmt.Errorf("live pod securityContext = %+v, want no runAsNonRoot", sc)
				}
				return nil
			}),
		},
	})
}

func TestAccKubernetesJobV1_ttl_seconds_after_finished(t *testing.T) {
	var conf batchv1.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := kubetest.BusyboxImage
	resourceName := "kubernetes_job_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t); kubetest.SkipIfClusterVersionLessThan(t, "1.21.0") },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesJobV1Config_ttl_seconds_after_finished(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.ttl_seconds_after_finished", "60"),
				),
			},
		},
	})
}

func testAccCheckJobV1Waited(minDuration time.Duration) func(*terraform.State) error {
	// NOTE this works because this function is called when setting up the test
	// and the function it returns is called after the resource has been created
	startTime := time.Now()

	return func(_ *terraform.State) error {
		testDuration := time.Since(startTime)
		if testDuration < minDuration {
			return fmt.Errorf("the job should have waited at least %s before being created", minDuration)
		}
		return nil
	}
}

func testAccCheckKubernetesJobV1ForceNew(old, new *batchv1.Job, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for Job %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting Job UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func testAccCheckKubernetesJobV1Destroy(s *terraform.State) error {
	conn, err := kubetest.Clientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_job_v1" {
			continue
		}

		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, err = conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			return fmt.Errorf("Job still exists: %s", rs.Primary.ID)
		}
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func testAccCheckKubernetesJobV1Exists(n string, obj *batchv1.Job) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := kubetest.Clientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesJobV1Config_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    active_deadline_seconds = 120
    backoff_limit           = 10
    completions             = 4
    parallelism             = 2
    pod_failure_policy {
      rule {
        action = "FailJob"
        on_exit_codes {
          container_name = "hello"
          operator       = "In"
          values         = [1, 2, 42]
        }
      }
      rule {
        action = "Ignore"
        on_pod_condition {
          status = "False"
          type   = "DisruptionTarget"
        }
      }
    }
    template {
      metadata {}
      spec {
        container {
          name    = "hello"
          image   = "%s"
          command = ["echo", "'hello'"]
        }
      }
    }
  }

  wait_for_completion = false
}`, name, imageName)
}

func testAccKubernetesJobV1Config_updateMutableFields(name, imageName, activeDeadlineSeconds, backoffLimit, manualSelector, parallelism string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    active_deadline_seconds = %s
    backoff_limit           = %s
    completions             = 4
    manual_selector         = %s
    parallelism             = %s
    pod_failure_policy {
      rule {
        action = "FailJob"
        on_exit_codes {
          container_name = "hello"
          operator       = "In"
          values         = [1, 2, 42]
        }
      }
      rule {
        action = "Ignore"
        on_pod_condition {
          status = "False"
          type   = "DisruptionTarget"
        }
      }
    }
    template {
      metadata {}
      spec {
        container {
          name    = "hello"
          image   = "%s"
          command = ["echo", "'hello'"]
        }
      }
    }
  }

  wait_for_completion = false
}`, name, activeDeadlineSeconds, backoffLimit, manualSelector, parallelism, imageName)
}

func testAccKubernetesJobV1Config_updateImmutableFields(name, imageName, completions string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    completions = %s
    template {
      metadata {}
      spec {
        container {
          name    = "newname"
          image   = "%s"
          command = ["echo", "'hello'"]
        }
      }
    }
  }

  wait_for_completion = false
}`, name, completions, imageName)
}

func testAccKubernetesJobV1Config_volumeMount(name, imageName, subPath string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    template {
      metadata {}
      spec {
        container {
          name    = "hello"
          image   = "%s"
          command = ["sh", "-c", "true"]
          volume_mount {
            name       = "scratch"
            mount_path = "/scratch"
            %s
          }
        }
        volume {
          name = "scratch"
          empty_dir {}
        }
        restart_policy = "Never"
      }
    }
  }

  wait_for_completion = false
}`, name, imageName, subPath)
}

func testAccKubernetesJobV1Config_ttl_seconds_after_finished(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    backoff_limit              = 10
    completions                = 4
    parallelism                = 2
    ttl_seconds_after_finished = "60"
    template {
      metadata {}
      spec {
        container {
          name    = "hello"
          image   = "%s"
          command = ["echo", "'hello'"]
        }
      }
    }
  }
}`, name, imageName)
}

func testAccKubernetesJobV1Config_wait_for_completion(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    template {
      metadata {
        name = "wait-test"
      }
      spec {
        container {
          name    = "wait-test"
          image   = "%s"
          command = ["sleep", "10"]
        }
      }
    }
  }
  wait_for_completion = true
  timeouts {
    create = "1m"
  }
}`, name, imageName)
}

// Kubernetes drops empty selector values; a configured empty set still
// converges instead of planning it again after every refresh.
func TestAccKubernetesJobV1_emptySelectorValues(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := strings.Replace(testAccKubernetesJobV1Config_modified(name, kubetest.BusyboxImage), `selector = {`,
		`selector = {
      match_expressions = [{ key = "foo", operator = "Exists", values = [] }]`, 1)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{{
			Config: config,
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
		}},
	})
}

func testAccKubernetesJobV1Config_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = "%s"
    labels = {
      "foo" = "bar"
    }
  }
  spec {
    manual_selector = true
    selector = {
      match_labels = {
        "foo" = "bar"
      }
    }
    template {
      metadata {
        labels = {
          "foo" = "bar"
        }
      }
      spec {
        container {
          name    = "hello"
          image   = "%s"
          command = ["echo", "'hello'"]
        }
      }
    }
  }
  wait_for_completion = false
}`, name, imageName)
}

func TestAccKubernetesJobV1_upgrade(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := testAccKubernetesJobV1Config_basic(name, kubetest.BusyboxImage)
	var before, after batchv1.Job
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			kubetest.PreCheck(t)
			kubetest.SkipIfClusterVersionLessThan(t, "1.26.0")
		},
		CheckDestroy: testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:            config,
				ExternalProviders: kubetest.ReleasedProvider("3.2.1"),
				Check:             testAccCheckKubernetesJobV1Exists("kubernetes_job_v1.test", &before),
			},
			{
				Config:                   config,
				ProtoV6ProviderFactories: kubetest.ProviderFactories,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists("kubernetes_job_v1.test", &after),
					testAccCheckKubernetesJobV1ForceNew(&before, &after, false),
					func(_ *terraform.State) error {
						if !reflect.DeepEqual(before.Spec, after.Spec) {
							return fmt.Errorf("migration changed the Job API spec:\nbefore: %#v\nafter: %#v", before.Spec, after.Spec)
						}
						return nil
					},
					resource.TestCheckResourceAttr("kubernetes_job_v1.test", "spec.0.backoff_limit", "10"),
					resource.TestCheckResourceAttr("kubernetes_job_v1.test", "spec.0.pod_failure_policy.0.rule.#", "2"),
				),
			},
			{
				Config:                   config,
				ProtoV6ProviderFactories: kubetest.ProviderFactories,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// State written by 2.3.0, the last release that stored metadata.self_link,
// moves without replacing the Job.
func TestAccKubernetesJobV1_moveFromLegacyState(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	config := fmt.Sprintf(`resource "kubernetes_job_v1" "test" {
  metadata {
    name = %q
  }
  spec {
    backoff_limit = 0
    template {
      metadata {}
      spec {
        restart_policy = "Never"
        container {
          name    = "c"
          image   = %q
          command = ["true"]
        }
      }
    }
  }
  wait_for_completion = false
}
`, name, kubetest.BusyboxImage)
	var before, after batchv1.Job
	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { kubetest.PreCheck(t) },
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: kubetest.ReleasedProvider("2.3.0"),
				Config:            strings.Replace(config, `"kubernetes_job_v1"`, `"kubernetes_job"`, 1),
				Check:             testAccCheckKubernetesJobV1Exists("kubernetes_job.test", &before),
				// 2.3.0 does not settle on empty values it reads back.
				ExpectNonEmptyPlan: true,
			},
			{
				ProtoV6ProviderFactories: kubetest.ProviderFactories,
				Config: config + `
moved {
  from = kubernetes_job.test
  to   = kubernetes_job_v1.test
}
`,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{expectNoReplacement("kubernetes_job_v1.test")},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobV1Exists("kubernetes_job_v1.test", &after),
					testAccCheckKubernetesJobV1ForceNew(&before, &after, false),
				),
			},
		},
	})
}

func TestAccKubernetesJobV1_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var job batchv1.Job
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { kubetest.PreCheck(t) },
		ProtoV6ProviderFactories: kubetest.ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesJobV1Destroy,
		Steps: []resource.TestStep{{
			Config: testAccKubernetesJobV1Config_updateImmutableFields(name, kubetest.BusyboxImage, "1"),
			Check: resource.ComposeAggregateTestCheckFunc(
				testAccCheckKubernetesJobV1Exists("kubernetes_job_v1.test", &job),
				func(_ *terraform.State) error {
					client, err := kubetest.Clientset()
					if err != nil {
						return err
					}
					propagation := metav1.DeletePropagationBackground
					return client.BatchV1().Jobs(job.Namespace).Delete(context.Background(), job.Name, metav1.DeleteOptions{PropagationPolicy: &propagation})
				},
			),
			ExpectNonEmptyPlan: true,
		}},
	})
}
