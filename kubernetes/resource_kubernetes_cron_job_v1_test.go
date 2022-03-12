package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesCronJobV1_basic(t *testing.T) {
	var conf1, conf2 batch.CronJob
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := alpineImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_cron_job_v1.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1Config_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists("kubernetes_cron_job_v1.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "spec.0.schedule"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.concurrency_policy", "Replace"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.failed_jobs_history_limit", "5"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.schedule", "1 0 * * *"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.starting_deadline_seconds", "10"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.successful_jobs_history_limit", "10"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.suspend", "true"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.parallelism", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.backoff_limit", "2"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.image", imageName),
				),
			},
			{
				Config: testAccKubernetesCronJobV1Config_modified(name, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists("kubernetes_cron_job_v1.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.concurrency_policy", "Allow"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.failed_jobs_history_limit", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.schedule", "1 0 * * *"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.starting_deadline_seconds", "0"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.successful_jobs_history_limit", "3"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.suspend", "false"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.parallelism", "2"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.backoff_limit", "0"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.template.0.metadata.0.labels.%", "1"),
					testAccCheckKubernetesCronJobV1ForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesCronJobV1_extra(t *testing.T) {
	var conf batch.CronJob
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := alpineImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_cron_job_v1.test",
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesCronJobV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesCronJobV1Config_extra(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists("kubernetes_cron_job_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "spec.0.schedule"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.concurrency_policy", "Forbid"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.successful_jobs_history_limit", "10"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.failed_jobs_history_limit", "10"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.starting_deadline_seconds", "60"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.backoff_limit", "2"),
				),
			},
			{
				Config: testAccKubernetesCronJobV1Config_extraModified(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesCronJobV1Exists("kubernetes_cron_job_v1.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_cron_job_v1.test", "spec.0.schedule"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.concurrency_policy", "Forbid"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.successful_jobs_history_limit", "2"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.failed_jobs_history_limit", "2"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.starting_deadline_seconds", "120"),
					resource.TestCheckResourceAttr("kubernetes_cron_job_v1.test", "spec.0.job_template.0.spec.0.backoff_limit", "3"),
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
		if rs.Type != "kubernetes_cron_job" {
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

func testAccCheckKubernetesCronJobV1Exists(n string, obj *batch.CronJob) resource.TestCheckFunc {
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
  }
  spec {
    concurrency_policy = "Replace"
    failed_jobs_history_limit = 5
    schedule = "1 0 * * *"
    starting_deadline_seconds = 10
    successful_jobs_history_limit = 10
    suspend = true
    job_template {
      metadata {}
      spec {
        backoff_limit = 2
        template {
          metadata {}
          spec {
            container {
              name = "hello"
              image = "%s"
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
              name = "hello"
              image = "%s"
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
    schedule = "1 0 * * *"
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
              name = "hello"
              image = "%s"
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
    schedule = "1 0 * * *"
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
              name = "hello"
              image = "%s"
              command = ["echo", "'hello'"]
            }
          }
        }
      }
    }
  }
}`, name, imageName)
}

func testAccCheckKubernetesCronJobV1ForceNew(old, new *batch.CronJob, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
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
