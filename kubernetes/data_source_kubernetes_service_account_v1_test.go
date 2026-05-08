// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	// "regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKubernetesDataSourceServiceAccountV1_basic(t *testing.T) {
	resourceName := "kubernetes_service_account_v1.test"
	dataSourceName := "data.kubernetes_service_account_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceAccountV1_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotation", "annotation"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabel", "label"),
					resource.TestCheckResourceAttr(resourceName, "secret.0.name", name+"-secret"),
					resource.TestCheckResourceAttr(resourceName, "image_pull_secret.0.name", name+"-image-pull-secret"),
					resource.TestCheckResourceAttr(resourceName, "automount_service_account_token", "true"),
				),
			},
			{
				Config: testAccKubernetesDataSourceServiceAccountV1_basic(name) +
					testAccKubernetesDataSourceServiceAccountV1_read(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.annotations.TestAnnotation", "annotation"),
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.labels.TestLabel", "label"),
					resource.TestCheckResourceAttr(dataSourceName, "secret.0.name", name+"-secret"),
					resource.TestCheckResourceAttr(dataSourceName, "image_pull_secret.0.name", name+"-image-pull-secret"),
					resource.TestCheckResourceAttr(dataSourceName, "automount_service_account_token", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceServiceAccountV1_default_secret(t *testing.T) {
	resourceName := "kubernetes_service_account_v1.test"
	dataSourceName := "data.kubernetes_service_account_v1.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.24.0")
		},
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceAccountV1_default_secret(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "secret.#", "1"),
				),
			},
			{
				Config: testAccKubernetesDataSourceServiceAccountV1_default_secret(name) +
					testAccKubernetesDataSourceServiceAccountV1_default_secret_read(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "secret.#", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "default_secret_name", ""),
				),
			},
		},
	})
}

func TestAccKubernetesDataSourceServiceAccountV1_not_found(t *testing.T) {
	dataSourceName := "data.kubernetes_service_account_v1.test"
	name := fmt.Sprintf("ceci-n.est-pas-une-service-account-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceAccountV1_nonexistent(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(dataSourceName, "secret.#", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "image_pull_secret.#", "0"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceServiceAccountV1_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
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
    name = "${kubernetes_secret_v1.secret.metadata.0.name}"
  }
  image_pull_secret {
    name = "${kubernetes_secret_v1.image_pull_secret.metadata.0.name}"
  }
}

resource "kubernetes_secret_v1" "secret" {
  metadata {
    name = "%s-secret"
  }
}

resource "kubernetes_secret_v1" "image_pull_secret" {
  metadata {
    name = "%s-image-pull-secret"
  }
}
`, name, name, name)
}

func testAccKubernetesDataSourceServiceAccountV1_read() string {
	return `data "kubernetes_service_account_v1" "test" {
  metadata {
    name = "${kubernetes_service_account_v1.test.metadata.0.name}"
  }
}
`
}

func testAccKubernetesDataSourceServiceAccountV1_default_secret(name string) string {
	return fmt.Sprintf(`variable "token_name" {
  default = "%s-token-test0"
}

resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "%s"
  }
  secret {
    name = var.token_name
  }
}

resource "kubernetes_secret_v1" "test" {
  metadata {
    name = var.token_name
    annotations = {
      "kubernetes.io/service-account.name" = "%s"
    }
  }
  type = "kubernetes.io/service-account-token"
  depends_on = [
    kubernetes_service_account_v1.test
  ]
}
`, name, name, name)
}

func testAccKubernetesDataSourceServiceAccountV1_default_secret_read(name string) string {
	return fmt.Sprintf(`data "kubernetes_service_account_v1" "test" {
  metadata {
    name = "%s"
  }
  depends_on = [
    kubernetes_secret_v1.test
  ]
}
`, name)
}

func testAccKubernetesDataSourceServiceAccountV1_nonexistent(name string) string {
	return fmt.Sprintf(`data "kubernetes_service_account_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}
