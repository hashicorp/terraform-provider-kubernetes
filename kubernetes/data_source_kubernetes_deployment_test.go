package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

func TestAccKubernetesDataSourceDeployment_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDataSourceDeploymentConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("data.kubernetes_deployment.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("data.kubernetes_deployment.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("data.kubernetes_deployment.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("data.kubernetes_deployment.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.min_ready_seconds", "0"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.paused", "false"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.progress_deadline_seconds", "600"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.replicas", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.revision_history_limit", "10"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.strategy.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.strategy.0.type", "RollingUpdate"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_surge", "25%"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.strategy.0.rolling_update.0.max_unavailable", "25%"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.template.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.template.0.metadata.0.name", ""),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.template.0.spec.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.template.0.spec.0.container.#", "1"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
					resource.TestCheckResourceAttr("data.kubernetes_deployment.test", "spec.0.template.0.spec.0.container.0.image", "nginx:1.7.8"),
				),
			},
		},
	})
}

func testAccKubernetesDataSourceDeploymentConfig_basic(name string) string {
	return testAccKubernetesDeploymentConfig_basic(name) + `
data "kubernetes_deployment" "test" {
	metadata {
		name = "${kubernetes_deployment.test.metadata.0.name}"
		namespace = "${kubernetes_deployment.test.metadata.0.namespace}"
	}
}
`
}
