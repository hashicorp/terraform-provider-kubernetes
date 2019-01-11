package kubernetes

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	appsv1 "k8s.io/api/apps/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
)

func TestAccKubernetesDaemonSet_minimal(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_daemonset.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetConfig_minimal(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetExists("kubernetes_daemonset.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.container.0.image", "nginx:1.7.8"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSet_basic(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_daemonset.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetExists("kubernetes_daemonset.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_daemonset.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_daemonset.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_daemonset.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_daemonset.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.container.0.image", "nginx:1.7.8"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetExists("kubernetes_daemonset.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_daemonset.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_daemonset.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_daemonset.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_daemonset.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.container.0.image", "nginx:1.7.9"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.container.0.name", "tf-acc-test"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSet_importBasic(t *testing.T) {
	resourceName := "kubernetes_daemonset.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetConfig_basic(name),
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

func TestAccKubernetesDaemonSet_with_template_metadata(t *testing.T) {
	var conf appsv1.DaemonSet

	depName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	imageName := "nginx:1.7.9"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetConfigWithTemplateMetadata(depName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetExists("kubernetes_daemonset.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.0.annotations.prometheus.io/scrape", "true"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.0.annotations.prometheus.io/scheme", "https"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.0.annotations.prometheus.io/port", "4000"),
				),
			},
			{
				Config: testAccKubernetesDaemonSetConfigWithTemplateMetadataModified(depName, imageName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetExists("kubernetes_daemonset.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.container.0.image", imageName),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.0.labels.foo", "bar"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.0.annotations.prometheus.io/scrape", "true"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.0.annotations.prometheus.io/scheme", "http"),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.metadata.0.annotations.prometheus.io/port", "8080"),
				),
			},
		},
	})
}

func TestAccKubernetesDaemonSet_initContainer(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_daemonset.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetWithInitContainer(name, "nginx:1.7.8"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetExists("kubernetes_daemonset.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "spec.0.template.0.spec.0.init_container.0.image", "alpine"),
				),
			},
		},
	})
}
func TestAccKubernetesDaemonSet_noTopLevelLabels(t *testing.T) {
	var conf appsv1.DaemonSet
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_daemonset.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesDaemonSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDaemonSetWithNoTopLevelLabels(name, "nginx:1.7.8"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesDaemonSetExists("kubernetes_daemonset.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_daemonset.test", "metadata.0.labels.%", "0"),
				),
			},
		},
	})
}

func testAccCheckKubernetesDaemonSetDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_daemonset" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.AppsV1().DaemonSets(namespace).Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("DaemonSet still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesDaemonSetExists(n string, obj *appsv1.DaemonSet) resource.TestCheckFunc {
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
		out, err := conn.AppsV1().DaemonSets(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesDaemonSetConfig_minimal(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_daemonset" "test" {
  metadata {
    name = "%s"
  }
  spec {
    selector {
			match_labels {
				foo = "bar"
			}
    }
    template {
			metadata {
				labels {
					foo = "bar"
				}
			}
			spec {
				container {
					image = "nginx:1.7.8"
					name  = "tf-acc-test"
				}
			}
    }
  }
}
`, name)
}

func testAccKubernetesDaemonSetConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_daemonset" "test" {
  metadata {
    annotations {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    labels {
      TestLabelOne = "one"
      TestLabelTwo = "two"
      TestLabelThree = "three"
    }
    name = "%s"
  }
  spec {
    selector {
			match_labels {
        TestLabelOne = "one"
        TestLabelTwo = "two"
				TestLabelThree = "three"
			}
    }
    template {
			metadata {
				labels {
					TestLabelOne = "one"
					TestLabelTwo = "two"
					TestLabelThree = "three"
				}
			}
			spec {
				container {
					image = "nginx:1.7.8"
					name  = "tf-acc-test"
				}
			}
    }
  }
}
`, name)
}

func testAccKubernetesDaemonSetConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_daemonset" "test" {
  metadata {
    annotations {
      TestAnnotationOne = "one"
      Different = "1234"
    }
    labels {
      TestLabelOne = "one"
      TestLabelThree = "three"
    }
    name = "%s"
  }
  spec {
    selector {
			match_labels {
				TestLabelOne = "one"
				TestLabelTwo = "two"
				TestLabelThree = "three"
			}
    }
    template {
			metadata {
				labels {
					TestLabelOne = "one"
					TestLabelTwo = "two"
					TestLabelThree = "three"
				}
			}
			spec {
				container {
					image = "nginx:1.7.9"
					name  = "tf-acc-test"
				}
			}
    }
  }
}`, name)
}

func testAccKubernetesDaemonSetConfigWithTemplateMetadata(depName, imageName string) string {
	return fmt.Sprintf(`
resource "kubernetes_daemonset" "test" {
  metadata {
    name = "%s"
    labels {
      Test = "TfAcceptanceTest"
    }
  }
  spec {
    selector {
			match_labels {
				Test = "TfAcceptanceTest"
			}
		}
    template {
			metadata {
				labels {
					foo = "bar"
					Test = "TfAcceptanceTest"
				}
				annotations {
					"prometheus.io/scrape" = "true"
					"prometheus.io/scheme" = "https"
					"prometheus.io/port"   = "4000"
				}
			}
			spec {
				container {
					image = "%s"
					name  = "containername"
				}
			}
    }
  }
}
`, depName, imageName)
}

func testAccKubernetesDaemonSetConfigWithTemplateMetadataModified(depName, imageName string) string {
	return fmt.Sprintf(`
resource "kubernetes_daemonset" "test" {
  metadata {
    name = "%s"
    labels {
      Test = "TfAcceptanceTest"
    }
  }
  spec {
    selector {
			match_labels {
				Test = "TfAcceptanceTest"
			}
		}
    template {
			metadata {
				labels {
					foo = "bar"
					Test = "TfAcceptanceTest"
				}
				annotations {
					"prometheus.io/scrape" = "true"
					"prometheus.io/scheme" = "http"
					"prometheus.io/port"   = "8080"
				}
			}
			spec {
				container {
					image = "%s"
					name  = "containername"
				}
			}
    }
  }
}
`, depName, imageName)
}

func testAccKubernetesDaemonSetWithInitContainer(depName, imageName string) string {
	return fmt.Sprintf(`
resource "kubernetes_daemonset" "test" {
  metadata {
    name = "%s"
    labels {
			foo = "bar"
    }
  }
  spec {
    selector {
			match_labels {
				foo = "bar"
			}
		}
    template {
			metadata {
				labels {
					foo = "bar"
				}
			}
			spec {
				init_container {
					name = "hello"
					image = "alpine"
					command = ["echo", "'hello'"]
				}
				container {
					image = "%s"
					name  = "containername"
				}
			}
    }
  }
}
`, depName, imageName)
}

func testAccKubernetesDaemonSetWithNoTopLevelLabels(depName, imageName string) string {
	return fmt.Sprintf(`
resource "kubernetes_daemonset" "test" {
  metadata {
    name = "%s"
  }
  spec {
    selector {
			match_labels {
				foo = "bar"
			}
		}
    template {
			metadata {
				labels {
					foo = "bar"
				}
			}
			spec {
				container {
					image = "%s"
					name  = "containername"
				}
			}
    }
  }
}
`, depName, imageName)
}
