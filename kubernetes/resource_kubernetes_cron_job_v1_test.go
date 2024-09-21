// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesCronJobV1_basic(t *testing.T) {
	var conf1, conf2 batchv1.CronJob
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_cron_job_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.25.0")
		},
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1Config_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.hashicorp", "terraform"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.schedule"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.timezone"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.concurrency_policy", "Replace"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.failed_jobs_history_limit", "5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.schedule", "1 0 * * *"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.timezone", "Etc/UTC"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.starting_deadline_seconds", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.successful_jobs_history_limit", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.suspend", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.metadata.0.annotations.cluster-autoscaler.kubernetes.io/safe-to-evict", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.parallelism", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.backoff_limit", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.template.0.metadata.0.annotations.controller.kubernetes.io/pod-deletion-cost", "10000"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesCronJobV1Config_modified(name, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.concurrency_policy", "Allow"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.failed_jobs_history_limit", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.schedule", "1 0 * * *"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.timezone", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.starting_deadline_seconds", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.successful_jobs_history_limit", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.suspend", "false"),
					testAccCheckKubernetesCronJobV1ForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesCronJobV1_extra(t *testing.T) {
	var conf batchv1.CronJob
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_cron_job_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.25.0")
		},
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1Config_extra(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.schedule"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.concurrency_policy", "Forbid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.successful_jobs_history_limit", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.failed_jobs_history_limit", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.starting_deadline_seconds", "60"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.backoff_limit", "2"),
				),
			},
			{
				Config: testAccKubernetesCronJobV1Config_extraModified(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.schedule"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.concurrency_policy", "Forbid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.successful_jobs_history_limit", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.failed_jobs_history_limit", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.starting_deadline_seconds", "120"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.backoff_limit", "3"),
				),
			},
		},
	})
}

func TestAccKubernetesCronJobV1_minimalWithTemplateNamespace(t *testing.T) {
	var conf1, conf2 batchv1.CronJob

	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_cron_job_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.metadata.0.namespace", ""),
				),
			},
			{
				Config: testAccKubernetesCronJobV1ConfigMinimalWithJobTemplateNamespace(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.job_template.0.metadata.0.namespace"),
					testAccCheckKubernetesCronJobV1ForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesCronJobV1_minimalWithPodFailurePolicy(t *testing.T) {
	var conf1, conf2 batchv1.CronJob

	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_cron_job_v1.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionLessThan(t, "1.25.0")
		},
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.metadata.0.namespace", ""),
				),
			},
			{
				Config: testAccKubernetesCronJobV1ConfigMinimalWithPodFailurePolicy(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.backoff_limit_per_index", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.max_failed_indexes", "4"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.0.action", "FailJob"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.container_name", "test"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.0", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.1", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.0.on_exit_codes.0.values.2", "42"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.1.action", "Ignore"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.1.on_pod_condition.0.type", "DisruptionTarget"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.pod_failure_policy.0.rule.1.on_pod_condition.0.status", "False"),
					testAccCheckKubernetesCronJobV1ForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func testAccCheckKubernetesCronJobV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cron_job_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("CronJob still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesCronJobV1Exists(n string, obj *batchv1.CronJob) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesCronJobV1Config_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = "%s"
    annotations = {
      "hashicorp" = "terraform"
    }
  }
  spec {
    concurrency_policy            = "Replace"
    failed_jobs_history_limit     = 5
    schedule                      = "1 0 * * *"
    timezone                      = "Etc/UTC"
    starting_deadline_seconds     = 10
    successful_jobs_history_limit = 10
    suspend                       = true
    job_template {
      metadata {
        annotations = {
          "cluster-autoscaler.kubernetes.io/safe-to-evict" = "false"
        }
      }
      spec {
        backoff_limit = 2
        template {
          metadata {
            annotations = {
              "controller.kubernetes.io/pod-deletion-cost" = 10000
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
    }
  }
}`, name, imageName)
}

func testAccKubernetesCronJobV1Config_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    schedule = "1 0 * * *"
    job_template {
      metadata {}
      spec {
        parallelism = 2
        template {
          metadata {
            labels = {
              foo = "bar"
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
    }
  }
}`, name, imageName)
}

func testAccKubernetesCronJobV1Config_extra(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    schedule                      = "1 0 * * *"
    timezone                      = "Etc/UTC"
    concurrency_policy            = "Forbid"
    successful_jobs_history_limit = 10
    failed_jobs_history_limit     = 10
    starting_deadline_seconds     = 60
    job_template {
      metadata {}
      spec {
        backoff_limit = 2
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
    }
  }
}`, name, imageName)
}

func testAccKubernetesCronJobV1Config_extraModified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    schedule                      = "1 0 * * *"
    timezone                      = "Etc/UTC"
    concurrency_policy            = "Forbid"
    successful_jobs_history_limit = 2
    failed_jobs_history_limit     = 2
    starting_deadline_seconds     = 120
    job_template {
      metadata {}
      spec {
        backoff_limit = 3
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
    }
  }
}`, name, imageName)
}

func testAccCheckKubernetesCronJobV1ForceNew(old, new *batchv1.CronJob, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting forced replacement")
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Unexpected forced replacement")
			}
		}
		return nil
	}
}

func testAccKubernetesCronJobV1ConfigMinimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    schedule = "*/1 * * * *"
    job_template {
      metadata {}
      spec {
        template {
          metadata {}
          spec {
            container {
              name    = "test"
              image   = "%s"
              command = ["sleep", "5"]
            }
            termination_grace_period_seconds = 1
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesCronJobV1ConfigMinimalWithPodFailurePolicy(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    schedule = "*/1 * * * *"
    job_template {
      metadata {}
      spec {
        backoff_limit_per_index = 3
        max_failed_indexes      = 4
        completions             = 4
        completion_mode         = "Indexed"
        pod_failure_policy {
          rule {
            action = "FailJob"
            on_exit_codes {
              container_name = "test"
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
              name    = "test"
              image   = "%s"
              command = ["sleep", "5"]
            }
            termination_grace_period_seconds = 1
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesCronJobV1ConfigMinimalWithJobTemplateNamespace(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    schedule = "*/1 * * * *"
    job_template {
      metadata {
        // The namespace field is just a stub and does not influence where the Pod will be created.
        // The Pod will be created within the same Namespace as the Cron Job resource.
        namespace = "fake" // Doesn't have to exist.
      }
      spec {
        template {
          metadata {}
          spec {
            container {
              name    = "test"
              image   = "%s"
              command = ["sleep", "1"]
            }
            termination_grace_period_seconds = 1
          }
        }
      }
    }
  }
}
`, name, imageName)
}
