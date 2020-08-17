package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesDefaultServiceAccount_basic(t *testing.T) {
	var conf api.ServiceAccount
	namespace := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_default_service_account.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDefaultServiceAccountConfig_basic(namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountExists("kubernetes_default_service_account.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.name", "default"),
					resource.TestCheckResourceAttrSet("kubernetes_default_service_account.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_default_service_account.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_default_service_account.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_default_service_account.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "secret.#", "0"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "image_pull_secret.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesDefaultServiceAccount_secrets(t *testing.T) {
	var conf api.ServiceAccount
	namespace := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_default_service_account.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDefaultServiceAccountConfig_secrets(namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountExists("kubernetes_default_service_account.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "metadata.0.name", "default"),
					resource.TestCheckResourceAttrSet("kubernetes_default_service_account.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_default_service_account.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_default_service_account.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_default_service_account.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "secret.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_default_service_account.test", "image_pull_secret.#", "2"),
					testAccCheckServiceAccountImagePullSecrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^three$"),
						regexp.MustCompile("^four$"),
					}),
					testAccCheckServiceAccountSecrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^one$"),
						regexp.MustCompile("^two$"),
						regexp.MustCompile("^default-token-[a-z0-9]+$"),
					}),
				),
			},
		},
	})
}

func TestAccKubernetesDefaultServiceAccount_importBasic(t *testing.T) {
	resourceName := "kubernetes_default_service_account.test"
	namespace := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDefaultServiceAccountConfig_basic(namespace),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "automount_service_account_token"},
			},
		},
	})
}

func testAccKubernetesDefaultServiceAccountConfig_basic(namespace string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace" "test" {
	metadata {
		name = "%s"
	}
}

resource "kubernetes_default_service_account" "test" {
  metadata {
		namespace = "${kubernetes_namespace.test.id}"

    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
	}
}
`, namespace)
}

func testAccKubernetesDefaultServiceAccountConfig_secrets(namespace string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace" "test" {
	metadata {
		name = "%s"
	}
}

resource "kubernetes_default_service_account" "test" {
  metadata {
		namespace = "${kubernetes_namespace.test.id}"
  }

  secret {
    name = "${kubernetes_secret.one.metadata.0.name}"
  }

  secret {
    name = "${kubernetes_secret.two.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret.three.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret.four.metadata.0.name}"
  }
}

resource "kubernetes_secret" "one" {
  metadata {
		name      = "one"
		namespace = "${kubernetes_namespace.test.id}"
  }
}

resource "kubernetes_secret" "two" {
  metadata {
		name      = "two"
		namespace = "${kubernetes_namespace.test.id}"
  }
}

resource "kubernetes_secret" "three" {
  metadata {
		name      = "three"
		namespace = "${kubernetes_namespace.test.id}"
  }
}

resource "kubernetes_secret" "four" {
  metadata {
		name      = "four"
		namespace = "${kubernetes_namespace.test.id}"
  }
}
`, namespace)
}
