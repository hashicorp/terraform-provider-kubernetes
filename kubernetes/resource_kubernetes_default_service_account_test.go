// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	api "k8s.io/api/core/v1"
)

func TestAccKubernetesDefaultServiceAccount_basic(t *testing.T) {
	var conf api.ServiceAccount
	namespace := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_default_service_account_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDefaultServiceAccountConfig_basic(namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", "default"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "secret.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "image_pull_secret.#", "0"),
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

func TestAccKubernetesDefaultServiceAccount_secrets(t *testing.T) {
	var conf api.ServiceAccount
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_default_service_account_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     "kubernetes_default_service_account.test",
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDefaultServiceAccountConfig_secrets(namespace, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", "default"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "secret.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "image_pull_secret.#", "1"),
					testAccCheckServiceAccountImagePullSecrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "-two$"),
					}),
					testAccCheckServiceAccountSecrets(&conf, []*regexp.Regexp{
						regexp.MustCompile("^" + name + "one$"),
						regexp.MustCompile("^default-token-[a-z0-9]+$"),
					}),
				),
			},
		},
	})
}

func TestAccKubernetesDefaultServiceAccount_automountServiceAccountToken(t *testing.T) {
	var conf api.ServiceAccount
	namespace := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_default_service_account_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesServiceAccountDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesDefaultServiceAccountConfig_automountServiceAccountToken(namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesServiceAccountExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", "default"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "automount_service_account_token", "false"),
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

func testAccKubernetesDefaultServiceAccountConfig_basic(namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}

resource "kubernetes_default_service_account_v1" "test" {
  metadata {
    namespace = kubernetes_namespace_v1.test.metadata.0.name

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

func testAccKubernetesDefaultServiceAccountConfig_secrets(namespace string, name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}

resource "kubernetes_default_service_account_v1" "test" {
  metadata {
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  secret {
    name = kubernetes_secret_v1.one.metadata[0].name
  }

  image_pull_secret {
    name = kubernetes_secret_v1.two.metadata[0].name
  }
}

resource "kubernetes_secret_v1" "one" {
  metadata {
    name      = "%s-one"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}

resource "kubernetes_secret_v1" "two" {
  metadata {
    name      = "%s-two"
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, namespace, name, name)
}

func testAccKubernetesDefaultServiceAccountConfig_automountServiceAccountToken(namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}

resource "kubernetes_default_service_account_v1" "test" {
  metadata {
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  automount_service_account_token = false
}
`, namespace)
}
