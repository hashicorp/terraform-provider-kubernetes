package kubernetes

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesPodTemplateV1_basic(t *testing.T) {
	var conf1 api.PodTemplate

	podName := acctest.RandomWithPrefix("tf-acc-test")
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")

	imageName1 := nginxImageVersion
	resourceName := "kubernetes_pod_template_v1.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPodDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodTemplateV1ConfigBasic(secretName, configMapName, podName, imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodTemplateV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.app", "pod_label"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", podName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env.0.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env.0.value_from.0.secret_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env.0.value_from.0.secret_key_ref.0.optional", "true"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env.1.value_from.0.config_map_key_ref.0.optional", "true"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.0.config_map_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.0.config_map_ref.0.name", fmt.Sprintf("%s-from", configMapName)),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.0.config_map_ref.0.optional", "true"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.0.prefix", "FROM_CM_"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.1.secret_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.1.secret_ref.0.name", fmt.Sprintf("%s-from", secretName)),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.1.secret_ref.0.optional", "false"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.env_from.1.prefix", "FROM_S_"),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.container.0.image", imageName1),
					resource.TestCheckResourceAttr(resourceName, "template.0.spec.0.topology_spread_constraint.#", "0"),
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

func testAccKubernetesPodTemplateV1ConfigBasic(secretName, configMapName, podName, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_secret" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one    = "first_from"
    second = "second_from"
  }
}

resource "kubernetes_config_map" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_config_map" "test_from" {
  metadata {
    name = "%s-from"
  }

  data = {
    one = "ONE_FROM"
    two = "TWO_FROM"
  }
}

resource "kubernetes_pod_template_v1" "test" {
  metadata {
    labels = {
      app = "pod_label"
    }

    name = "%s"
  }

  template {
      metadata {
		labels = {
		  app = "pod_label"
		}
	  }

	  spec {
		automount_service_account_token = false
	
		container {
		  image = "%s"
		  name  = "containername"
	
		  env {
			name = "EXPORTED_VARIABLE_FROM_SECRET"
	
			value_from {
			  secret_key_ref {
				name     = "${kubernetes_secret.test.metadata.0.name}"
				key      = "one"
				optional = true
			  }
			}
		  }
		  env {
			name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
			value_from {
			  config_map_key_ref {
				name     = "${kubernetes_config_map.test.metadata.0.name}"
				key      = "one"
				optional = true
			  }
			}
		  }
	
		  env_from {
			config_map_ref {
			  name     = "${kubernetes_config_map.test_from.metadata.0.name}"
			  optional = true
			}
			prefix = "FROM_CM_"
		  }
		  env_from {
			secret_ref {
			  name     = "${kubernetes_secret.test_from.metadata.0.name}"
			  optional = false
			}
			prefix = "FROM_S_"
		  }
		}
	
		volume {
		  name = "db"
	
		  secret {
			secret_name = "${kubernetes_secret.test.metadata.0.name}"
		  }
		}
	  }
  }
}
`, secretName, secretName, configMapName, configMapName, podName, imageName)
}

func testAccCheckKubernetesPodTemplateV1Exists(n string, obj *api.PodTemplate) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().PodTemplates(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}
