package kubernetes

import (
	"fmt"
	// "regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKubernetesDataSourceServiceAccount_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceAccountConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_service_account.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_service_account.test", "metadata.0.annotations.TestAnnotation", "annotation"),
					resource.TestCheckResourceAttr("kubernetes_service_account.test", "metadata.0.labels.TestLabel", "label"),
					resource.TestCheckResourceAttr("kubernetes_service_account.test", "secret.0.name", name+"-secret"),
					resource.TestCheckResourceAttr("kubernetes_service_account.test", "image_pull_secret.0.name", name+"-image-pull-secret"),
					resource.TestCheckResourceAttr("kubernetes_service_account.test", "automount_service_account_token", "true"),
					resource.TestCheckResourceAttrSet("kubernetes_service_account.test", "default_secret_name"),
				),
			},
			{
				Config: testAccKubernetesDataSourceServiceAccountConfig_basic(name) +
					testAccKubernetesDataSourceServiceAccountConfig_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.annotations.TestAnnotation", "annotation"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.labels.TestLabel", "label"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "secret.0.name", name+"-secret"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "image_pull_secret.0.name", name+"-image-pull-secret"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "automount_service_account_token", "true"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account.test", "default_secret_name"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceServiceAccount_default_secret(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceAccountConfig_default_secret(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("kubernetes_service_account.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_service_account.test", "secret.#", "1"),
				),
			},
			{
				Config: testAccKubernetesServiceAccountConfig_default_secret(name) +
					testAccKubernetesDataSourceServiceAccountConfig_default_secret_read(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "secret.#", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "default_secret_name", ""),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceServiceAccountConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account" "test" {
  metadata {
    annotations = {
      TestAnnotation = "annotation"
    }

    labels = {
      TestLabel = "label"
    }
    name = "%s"
  }

  secret {
    name = "${kubernetes_secret.secret.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret.image_pull_secret.metadata.0.name}"
  }
}

resource "kubernetes_secret" "secret" {
  metadata {
    name = "%s-secret"
  }
}

resource "kubernetes_secret" "image_pull_secret" {
  metadata {
    name = "%s-image-pull-secret"
  }
}
`, name, name, name)
}

func testAccKubernetesDataSourceServiceAccountConfig_read() string {
	return fmt.Sprintf(`data "kubernetes_service_account" "test" {
  metadata {
    name = "${kubernetes_service_account.test.metadata.0.name}"
  }
}
`)
}

func testAccKubernetesServiceAccountConfig_default_secret(name string) string {
	return fmt.Sprintf(`
variable "token_name" {
  default = "%s-token-test0"
}

resource "kubernetes_service_account" "test" {
  metadata {
    name = "%s"
  }
  secret {
    name = var.token_name
  }
}

resource "kubernetes_secret" "test" {
  metadata {
    name = var.token_name
    annotations = {
      "kubernetes.io/service-account.name" = "%s"
    }
  }
  type = "kubernetes.io/service-account-token"
  depends_on = [
    kubernetes_service_account.test
  ]
}
`, name, name, name)
}

func testAccKubernetesDataSourceServiceAccountConfig_default_secret_read(name string) string {
	return fmt.Sprintf(`
data "kubernetes_service_account" "test" {
  metadata {
    name = "%s"
  }
  depends_on = [
    kubernetes_secret.test
  ]
}
`, name)
}
