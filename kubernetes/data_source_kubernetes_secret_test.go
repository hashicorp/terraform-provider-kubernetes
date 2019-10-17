package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceSecret_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceSecretConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_secret.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_secret.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_secret.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("data.kubernetes_secret.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "data.%", "2"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "data.one", "first"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "data.two", "second"),
					resource.TestCheckResourceAttr("data.kubernetes_secret.test", "type", "Opaque"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceSecretConfig_basic(name string) string {
	return testAccKubernetesSecretConfig_basic(name) + `
data "kubernetes_secret" "test" {
	metadata {
		name = "${kubernetes_secret.test.metadata.0.name}"
	}
}
`
}
