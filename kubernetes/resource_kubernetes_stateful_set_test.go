package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
)

func TestAccKubernetesStatefulSet_basic(t *testing.T) {
	var sset v1beta1.StatefulSet

	statefulSetName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	imageName1 := "nginx:1.7.9"
	imageName2 := "nginx:1.11"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfig_basic(statefulSetName, imageName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &sset),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", statefulSetName),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.app", "one"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.service_name", statefulSetName),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.container.0.image", imageName1),
				),
			},
			{
				Config: testAccKubernetesStatefulSetConfig_basic(statefulSetName, imageName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &sset),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "spec.0.template.0.container.0.image", imageName2),
				),
			},
		},
	})
}

func testAccCheckKubernetesStatefulSetExists(n string, obj *v1beta1.StatefulSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)

		namespace, name, _ := idParts(rs.Primary.ID)
		out, err := conn.AppsV1beta1().StatefulSets(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesStatefulSetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_stateful_set" {
			continue
		}
		namespace, name, _ := idParts(rs.Primary.ID)
		resp, err := conn.AppsV1beta1().StatefulSets(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Stateful Set still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccKubernetesStatefulSetConfig_basic(name, image string) string {
	return fmt.Sprintf(`
resource "kubernetes_stateful_set" "test" {
  metadata {
		name = "%s"
		labels {
			app = "one"
		}
  }
  spec {
    replicas = 2
    selector {
      app = "one"
    }
    service_name = "%s"
    template {
      container {
        image = "%s"
        name  = "tf-acc-test"
      }
    }
  }
}
`, name, name, image)
}
