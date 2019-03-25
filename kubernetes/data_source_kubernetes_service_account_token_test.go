package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccKubernetesDataSourceServiceAccountToken_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceServiceAccountTokenConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(

					resource.TestMatchResourceAttr("data.kubernetes_service_account_token.test", "metadata.0.name", regexp.MustCompile(fmt.Sprintf("%s-token.*", name))),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_token.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_token.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_token.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_token.test", "metadata.0.uid"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_token.test", "data.0.ca_crt"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_token.test", "data.0.namespace"),
					resource.TestCheckResourceAttrSet("data.kubernetes_service_account_token.test", "data.0.token"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceServiceAccountTokenConfig_basic(name string) string {
	return testAccKubernetesServiceAccountConfig_basic(name) + `
data "kubernetes_service_account_token" "test" {
	metadata {
		name = "${kubernetes_service_account.test.default_secret_name}"
	}
}
`
}
