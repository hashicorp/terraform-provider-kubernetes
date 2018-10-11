package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	api "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
)

func TestAccKubernetesStatefulSet_basic(t *testing.T) {
	var conf api.StatefulSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_stateful_set.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStatefulSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStatefulSetConfigBasic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStatefulSetExists("kubernetes_stateful_set.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_stateful_set.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.uid"),
					resource.TestCheckResourceAttrSet("kubernetes_stateful_set.test", "metadata.0.namespace"),
				),
			},
		},
	})
}
func testAccCheckKubernetesStatefulSetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_stateful_set" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("StatefulSet still exists: %s: (Generation %#v)", rs.Primary.ID, resp.Status.ObservedGeneration)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesStatefulSetExists(n string, obj *api.StatefulSet) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*kubernetes.Clientset)

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

func testAccKubernetesStatefulSetConfigBasic(name string) string {
	return fmt.Sprintf(`
	resource "kubernetes_stateful_set" "test" {
		metadata {
		  annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		  }
	  
		  labels {
			TestLabelOne   = "one"
			TestLabelTwo   = "two"
			TestLabelThree = "three"
		  }
	  
		  name = "%s"
		}
	  
		spec {
		  service_name = "ss-test-service"
		  replicas     = 2
	  
		  selector {
				match_labels {
					app = "ss-test"
				}
		  }
	  
		  template {
			metadata {
			  labels {
				app = "ss-test"
			  }
			}
	  
			spec {
			  container {
				name  = "ss-test"
				image = "k8s.gcr.io/pause:latest"
	  
				port {
				  container_port = "80"
				  name           = "web"
				}
	  
				volume_mount {
				  name       = "workdir"
				  mount_path = "/work-dir"
				}
			  }
			}
		  }
	  
		  volume_claim_template {
			metadata {
			  name = "ss-test"
			}
	  
			spec {
			  access_modes = ["ReadWriteOnce"]
	  
			  resources {
				requests {
				  storage = "1Gi"
				}
			  }
			}
		  }
		}
	  }`, name)
}
