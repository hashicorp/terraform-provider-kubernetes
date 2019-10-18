package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/batch/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesJob_basic(t *testing.T) {
	var conf api.Job
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_job.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesJobDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesJobConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_job.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.active_deadline_seconds", "120"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.backoff_limit", "10"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.completions", "10"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.parallelism", "2"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.name", "hello"),
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.image", "alpine"),
				),
			},
			{
				Config: testAccKubernetesJobConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesJobExists("kubernetes_job.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_job.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_job.test", "metadata.0.self_link"),
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
					resource.TestCheckResourceAttr("kubernetes_job.test", "spec.0.template.0.spec.0.container.0.image", "alpine"),
				),
			},
		},
	})
}

func testAccCheckKubernetesJobDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_job" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.BatchV1().Jobs(namespace).Get(name, meta_v1.GetOptions{})
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

		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.BatchV1().Jobs(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesJobConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_job" "test" {
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
					image = "alpine"
					command = ["echo", "'hello'"]
				}
			}
		}
	}
}`, name)
}

func testAccKubernetesJobConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_job" "test" {
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
					image = "alpine"
					command = ["echo", "'hello'"]
				}
			}
		}
	}
}`, name)
}
