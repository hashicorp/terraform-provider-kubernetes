// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesCronJobV1Beta1_basic(t *testing.T) {
	var conf1, conf2 batchv1beta1.CronJob
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_cron_job.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.25.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1Beta1Config_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Beta1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.schedule"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.concurrency_policy", "Replace"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.failed_jobs_history_limit", "5"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.schedule", "1 0 * * *"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.starting_deadline_seconds", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.successful_jobs_history_limit", "10"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.suspend", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.parallelism", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.backoff_limit", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesCronJobV1Beta1Config_modified(name, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Beta1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.concurrency_policy", "Allow"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.failed_jobs_history_limit", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.schedule", "1 0 * * *"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.starting_deadline_seconds", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.successful_jobs_history_limit", "3"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.suspend", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.parallelism", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.backoff_limit", "6"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.spec.0.template.0.metadata.0.labels.%", "1"),
					testAccCheckKubernetesCronJobV1Beta1ForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesCronJobV1Beta1_extra(t *testing.T) {
	var conf batchv1beta1.CronJob
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := busyboxImage
	resourceName := "kubernetes_cron_job.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.25.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1Beta1Config_extra(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Beta1Exists(resourceName, &conf),
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
				Config: testAccKubernetesCronJobV1Beta1Config_extraModified(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Beta1Exists(resourceName, &conf),
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

func TestAccKubernetesCronJobV1Beta1_minimalWithTemplateNamespace(t *testing.T) {
	var conf1, conf2 batchv1beta1.CronJob

	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_cron_job.test"
	imageName := busyboxImage

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.25.0")
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Beta1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1Beta1ConfigMinimal(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Beta1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.job_template.0.metadata.0.namespace", ""),
				),
			},
			{
				Config: testAccKubernetesCronJobV1Beta1ConfigMinimalWithJobTemplateNamespace(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Beta1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.namespace"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.job_template.0.metadata.0.namespace"),
					testAccCheckKubernetesCronJobV1Beta1ForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func testAccCheckKubernetesCronJobV1Beta1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_cron_job" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.BatchV1beta1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("CronJob still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesCronJobV1Beta1Exists(n string, obj *batchv1beta1.CronJob) resource.TestCheckFunc {
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

		out, err := conn.BatchV1beta1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesCronJobV1Beta1Config_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job" "test" {
  metadata {
    name = "%s"
  }
  spec {
    concurrency_policy            = "Replace"
    failed_jobs_history_limit     = 5
    schedule                      = "1 0 * * *"
    starting_deadline_seconds     = 10
    successful_jobs_history_limit = 10
    suspend                       = true
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

func testAccKubernetesCronJobV1Beta1Config_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job" "test" {
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

func testAccKubernetesCronJobV1Beta1Config_extra(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job" "test" {
  metadata {
    name = "%s"
  }
  spec {
    schedule                      = "1 0 * * *"
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

func testAccKubernetesCronJobV1Beta1Config_extraModified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job" "test" {
  metadata {
    name = "%s"
  }
  spec {
    schedule                      = "1 0 * * *"
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

func testAccCheckKubernetesCronJobV1Beta1ForceNew(old, new *batchv1beta1.CronJob, wantNew bool) resource.TestCheckFunc {
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

func testAccKubernetesCronJobV1Beta1ConfigMinimal(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job" "test" {
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
          }
        }
      }
    }
  }
}
`, name, imageName)
}

func testAccKubernetesCronJobV1Beta1ConfigMinimalWithJobTemplateNamespace(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_cron_job" "test" {
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
              command = ["sleep", "5"]
            }
          }
        }
      }
    }
  }
}
`, name, imageName)
}
