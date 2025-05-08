// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesServiceAccountV1_basic(t *testing.T) {
	var conf corev1.ServiceAccount
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_service_account_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceAccountV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "secret.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "image_pull_secret.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "automount_service_account_token", "true"),
					testAccCheckServiceAccountV1ImagePullSecrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-three$"),
						regexp.MustCompile("^" + name + "-four$"),
					}),
					testAccCheckServiceAccountV1Secrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-one$"),
						regexp.MustCompile("^" + name + "-two$"),
						regexp.MustCompile("^" + name + "-token-[a-z0-9]+$"),
					}),
				),
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

func TestAccKubernetesServiceAccountV1_default_secret(t *testing.T) {
	var conf corev1.ServiceAccount
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_service_account_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			skipIfClusterVersionGreaterThanOrEqual(t, "1.24.0")
		},
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceAccountV1Config_default_secret(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "default_secret_name"),
				),
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

func TestAccKubernetesServiceAccountV1_automount(t *testing.T) {
	var conf corev1.ServiceAccount
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_service_account_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceAccountV1Config_automount(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "secret.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "image_pull_secret.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "automount_service_account_token", "false"),
					testAccCheckServiceAccountV1ImagePullSecrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-three$"),
						regexp.MustCompile("^" + name + "-four$"),
					}),
					testAccCheckServiceAccountV1Secrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-one$"),
						regexp.MustCompile("^" + name + "-two$"),
						regexp.MustCompile("^" + name + "-token-[a-z0-9]+$"),
					}),
				),
			},
		},
	})
}

func TestAccKubernetesServiceAccountV1_update(t *testing.T) {
	var conf corev1.ServiceAccount
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_service_account_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceAccountV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "secret.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "image_pull_secret.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "automount_service_account_token", "true"),
					testAccCheckServiceAccountV1ImagePullSecrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-three$"),
						regexp.MustCompile("^" + name + "-four$"),
					}),
					testAccCheckServiceAccountV1Secrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-one$"),
						regexp.MustCompile("^" + name + "-two$"),
						regexp.MustCompile("^" + name + "-token-[a-z0-9]+$"),
					}),
				),
			},
			{
				Config: testAccKubernetesServiceAccountV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "secret.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "image_pull_secret.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "automount_service_account_token", "false"),
					testAccCheckServiceAccountV1ImagePullSecrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-three$"),
						regexp.MustCompile("^" + name + "-four$"),
					}),
					testAccCheckServiceAccountV1Secrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-one$"),
						regexp.MustCompile("^" + name + "-two$"),
						regexp.MustCompile("^" + name + "-token-[a-z0-9]+$"),
					}),
				),
			},
			{
				Config: testAccKubernetesServiceAccountV1Config_noAttributes(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "secret.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "image_pull_secret.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "automount_service_account_token", "true"),
					testAccCheckServiceAccountV1ImagePullSecrets(&conf, []*regexp.Regexp{}),
					testAccCheckServiceAccountV1Secrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-token-[a-z0-9]+$"),
					}),
				),
			},
		},
	})
}

func TestAccKubernetesServiceAccount_generatedName(t *testing.T) {
	var conf corev1.ServiceAccount
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_service_account_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_service_account.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesServiceAccountV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "automount_service_account_token", "true"),
					testAccCheckServiceAccountV1ImagePullSecrets(&conf, []*regexp.Regexp{}),
					testAccCheckServiceAccountV1Secrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + prefix + "[a-z0-9]+-token-[a-z0-9]+$"),
					}),
				),
			},
		},
	})
}

func testAccCheckServiceAccountV1ImagePullSecrets(m *corev1.ServiceAccount, expected []*regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(m.ImagePullSecrets) == 0 {
			return nil
		}

		if !matchLocalObjectReferenceName(m.ImagePullSecrets, expected) {
			return fmt.Errorf("%s image pull secrets don't match.\nExpected: %q\nGiven: %q",
				m.Name, expected, m.ImagePullSecrets)
		}

		return nil
	}
}

func matchLocalObjectReferenceName(lor []corev1.LocalObjectReference, expected []*regexp.Regexp) bool {
	for _, r := range expected {
		for _, ps := range lor {
			matched := r.MatchString(ps.Name)
			if matched {
				return true
			}
		}
	}
	return false
}

func testAccCheckServiceAccountV1Secrets(m *corev1.ServiceAccount, expected []*regexp.Regexp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if clusterVersionGreaterThanOrEqual("1.24.0") {
			return nil
		}
		if len(expected) == 0 && len(m.Secrets) == 0 {
			return nil
		}
		if !matchObjectReferenceName(m.Secrets, expected) {
			return fmt.Errorf("%s secrets don't match.\nExpected: %q\nGiven: %q",
				m.Name, expected, m.Secrets)
		}
		return nil
	}
}

func matchObjectReferenceName(lor []corev1.ObjectReference, expected []*regexp.Regexp) bool {
	for _, r := range expected {
		for _, ps := range lor {
			matched := r.MatchString(ps.Name)
			if matched {
				return true
			}
		}
	}
	return false
}

func testAccCheckKubernetesServiceAccountV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_service_account" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		resp, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Service Account still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesServiceAccountV1Exists(n string, obj *corev1.ServiceAccount) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesServiceAccountV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  secret {
    name = "${kubernetes_secret_v1.one.metadata.0.name}"
  }

  secret {
    name = "${kubernetes_secret_v1.two.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret_v1.three.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret_v1.four.metadata.0.name}"
  }
}

resource "kubernetes_secret_v1" "one" {
  metadata {
    name = "%s-one"
  }
}

resource "kubernetes_secret_v1" "two" {
  metadata {
    name = "%s-two"
  }
}

resource "kubernetes_secret_v1" "three" {
  metadata {
    name = "%s-three"
  }
}

resource "kubernetes_secret_v1" "four" {
  metadata {
    name = "%s-four"
  }
}
`, name, name, name, name, name)
}

func testAccKubernetesServiceAccountV1Config_default_secret(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "%s"
  }
}`, name)
}

func testAccKubernetesServiceAccountV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  secret {
    name = "${kubernetes_secret_v1.one.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret_v1.two.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret_v1.three.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret_v1.four.metadata.0.name}"
  }

  automount_service_account_token = false
}

resource "kubernetes_secret_v1" "one" {
  metadata {
    name = "%s-one"
  }
}

resource "kubernetes_secret_v1" "two" {
  metadata {
    name = "%s-two"
  }
}

resource "kubernetes_secret_v1" "three" {
  metadata {
    name = "%s-three"
  }
}

resource "kubernetes_secret_v1" "four" {
  metadata {
    name = "%s-four"
  }
}
`, name, name, name, name, name)
}

func testAccKubernetesServiceAccountV1Config_noAttributes(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, name)
}

func testAccKubernetesServiceAccountV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    generate_name = "%s"
  }
}
`, prefix)
}

func testAccKubernetesServiceAccountV1Config_automount(name string) string {
	return fmt.Sprintf(`resource "kubernetes_service_account_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = "%s"
  }

  secret {
    name = "${kubernetes_secret_v1.one.metadata.0.name}"
  }

  secret {
    name = "${kubernetes_secret_v1.two.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret_v1.three.metadata.0.name}"
  }

  image_pull_secret {
    name = "${kubernetes_secret_v1.four.metadata.0.name}"
  }

  automount_service_account_token = false
}

resource "kubernetes_secret_v1" "one" {
  metadata {
    name = "%s-one"
  }
}

resource "kubernetes_secret_v1" "two" {
  metadata {
    name = "%s-two"
  }
}

resource "kubernetes_secret_v1" "three" {
  metadata {
    name = "%s-three"
  }
}

resource "kubernetes_secret_v1" "four" {
  metadata {
    name = "%s-four"
  }
}
`, name, name, name, name, name)
}
