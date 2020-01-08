package kubernetes

import (
	"fmt"
	// "regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceServiceAccount_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceAccountConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "secret.#", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "image_pull_secret.#", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_service_account.test", "automount_service_account_token", "false"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceServiceAccountConfig_basic(name string) string {
	return testAccKubernetesServiceAccountConfig_basic(name) + `
data "kubernetes_service_account" "test" {
	metadata {
		name = "${kubernetes_service_account.test.metadata.0.name}"
	}
}
`
}
