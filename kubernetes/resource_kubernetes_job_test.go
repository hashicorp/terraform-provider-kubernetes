package kubernetes

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesJob_wait_for_completion(t *testing.T) {
	var conf api.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_job.test",
		IDRefreshIgnore: []string{
			"metadata.0.resource_version",
			"spec.0.selector.0.match_expressions.#",
			"spec.0.template.0.spec.0.container.0.resources.0.limits.#",
			"spec.0.template.0.spec.0.container.0.resources.0.requests.#",
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesJobConfig_wait_for_completion(name, busyboxImageVersion),
				Check: resource.ComposeAggregateTestCheckFunc(
					// NOTE this is to check that Terraform actually waited for the Job to complete
					// before considering the Job resource as created
					testAccCheckJobWaited(time.Duration(10)*time.Second),
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_job.test", "wait_for_completion", "true"),
				),
			},
		},
	})
}

func testAccCheckJobWaited(minDuration time.Duration) func(*terraform.State) error {
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

func TestAccKubernetesJob_basic(t *testing.T) {
	var conf api.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := alpineImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_job.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesJobConfig_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_job.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.active_deadline_seconds", "120"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.backoff_limit", "10"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.completions", "10"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.parallelism", "2"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_job.test", "wait_for_completion", "false"),
				),
			},
			{
				Config: testAccKubernetesJobConfig_modified(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_job.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "metadata.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.active_deadline_seconds", "0"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.backoff_limit", "0"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.completions", "1"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.parallelism", "1"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.manual_selector", "true"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_job.test", "wait_for_completion", "false"),
				),
			},
		},
	})
}

func TestAccKubernetesJob_update(t *testing.T) {
	var conf1, conf2, conf3 api.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := alpineImageVersion
	imageName1 := busyboxImageVersion

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_job.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesJobConfig_basic(name, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_job.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.active_deadline_seconds", "120"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.backoff_limit", "10"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.completions", "10"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.parallelism", "2"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_job.test", "wait_for_completion", "false"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.manual_selector", "false"),
				),
			},
			{
				Config: testAccKubernetesJobConfig_updateMutableFields(name, imageName, "121", "10", "false", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.active_deadline_seconds", "121"),
					testAccCheckKubernetesJobForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesJobConfig_updateMutableFields(name, imageName, "121", "11", "false", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.backoff_limit", "11"),
					testAccCheckKubernetesJobForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesJobConfig_updateMutableFields(name, imageName, "121", "11", "true", "2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.manual_selector", "true"),
					testAccCheckKubernetesJobForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesJobConfig_updateMutableFields(name, imageName, "121", "11", "true", "3"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.parallelism", "3"),
					testAccCheckKubernetesJobForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesJobConfig_updateImmutableFields(name, imageName, "12"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.completions", "12"),
					testAccCheckKubernetesJobForceNew(&conf1, &conf2, true),
				),
			},
			{
				Config: testAccKubernetesJobConfig_updateImmutableFields(name, imageName1, "12"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf3),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.image", imageName1),
					testAccCheckKubernetesJobForceNew(&conf2, &conf3, true),
				),
			},
		},
	})
}

// FIXME uncomment this check when the TTLSecondsAfterFinished feature gate defaults to true
//func TestAccKubernetesJob_ttl_seconds_after_finished(t *testing.T) {
//	var conf api.Job
//	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
//
//	resource.Test(t, resource.TestCase{
//		PreCheck:      func() { testAccPreCheck(t) },
//		IDRefreshName: "kubernetes_job.test",
//		IDRefreshIgnore: []string{
//			"spec.0.selector.0.match_expressions.#",
//		},
//		ProviderFactories: testAccProviderFactories,
//		CheckDestroy:      testAccCheckKubernetesJobDestroy,
//		Steps: []resource.TestStep{
//			{
//				Config: testAccKubernetesJobConfig_ttl_seconds_after_finished(name, busyboxImageVersion),
//				Check: resource.ComposeAggregateTestCheckFunc(
//					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf),
//					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.ttl_seconds_after_finished", "60"),
//				),
//			},
//		},
//	})
//}

func testAccCheckKubernetesJobForceNew(old, new *api.Job, wantNew bool) resource.TestCheckFunc {
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

func testAccCheckKubernetesJobDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_job" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Job still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesJobExists(n string, obj *api.Job) resource.TestCheckFunc {
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

		out, err := conn.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesJobConfig_basic(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_job" "test" {
  metadata {
    name = "%s"
  }
  spec {
    active_deadline_seconds = 120
    backoff_limit = 10
    completions = 10
    parallelism = 2
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

  wait_for_completion = false
}`, name, imageName)
}

func testAccKubernetesJobConfig_updateMutableFields(name, imageName, activeDeadlineSeconds, backoffLimit, manualSelector, parallelism string) string {
	return fmt.Sprintf(`resource "kubernetes_job" "test" {
  metadata {
    name = "%s"
  }
  spec {
    active_deadline_seconds = %s
    backoff_limit = %s
    completions = 10
    manual_selector = %s
    parallelism = %s
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

  wait_for_completion = false
}`, name, activeDeadlineSeconds, backoffLimit, manualSelector, parallelism, imageName)
}

func testAccKubernetesJobConfig_updateImmutableFields(name, imageName, completions string) string {
	return fmt.Sprintf(`resource "kubernetes_job" "test" {
  metadata {
    name = "%s"
  }
  spec {
    completions = %s
    template {
      metadata {}
      spec {
        container {
          name = "newname"
          image = "%s"
          command = ["echo", "'hello'"]
        }
      }
    }
  }

  wait_for_completion = false
}`, name, completions, imageName)
}

func testAccKubernetesJobConfig_ttl_seconds_after_finished(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_job" "test" {
  metadata {
    name = "%s"
  }
  spec {
    backoff_limit = 10
    completions = 10
    parallelism = 2
    ttl_seconds_after_finished = "60"
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
}`, name, imageName)
}

func testAccKubernetesJobConfig_wait_for_completion(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_job" "test" {
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
          name = "wait-test"
          image = "%s"
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

func testAccKubernetesJobConfig_modified(name, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_job" "test" {
  metadata {
    name = "%s"
    labels = {
      "foo" = "bar"
    }
  }
  spec {
    manual_selector = true
    selector {
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
          name = "hello"
          image = "%s"
          command = ["echo", "'hello'"]
        }
      }
    }
  }
  wait_for_completion = false
}`, name, imageName)
}
