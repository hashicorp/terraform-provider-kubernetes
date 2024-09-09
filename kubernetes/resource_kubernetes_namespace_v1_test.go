// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	v6resource "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesNamespaceV1_basic(t *testing.T) {
	var conf corev1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_namespace_v1.test"

	v6resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_basic(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			// {
			// 	ResourceName:            resourceName,
			// 	ImportState:             true,
			// 	ImportStateVerify:       true,
			// 	ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_default_service_account"},
			// },
			{
				Config: testAccKubernetesNamespaceV1Config_addAnnotations(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesNamespaceV1Config_addLabels(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesNamespaceV1Config_smallerLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				Config: testAccKubernetesNamespaceV1Config_noLists(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					v6resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					v6resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

// func TestAccKubernetesNamespaceV1_default_service_account(t *testing.T) {
// 	var nsConf corev1.Namespace
// 	var saConf corev1.ServiceAccount
// 	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
// 	resourceName := "kubernetes_namespace_v1.test"

// 	resource.ParallelTest(t, resource.TestCase{
// 		PreCheck:          func() { testAccPreCheck(t) },
// 		IDRefreshName:     resourceName,
// 		ProviderFactories: testAccProviderFactories,
// 		CheckDestroy:      testAccCheckKubernetesNamespaceV1Destroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccKubernetesNamespaceV1Config_wait_for_default_service_acccount(nsName),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					testAccCheckKubernetesNamespaceV1Exists(resourceName, &nsConf),
// 					testAccCheckKubernetesDefaultServiceAccountExists(resourceName, &saConf),
// 				),
// 			},
// 			{
// 				ResourceName:            resourceName,
// 				ImportState:             true,
// 				ImportStateVerify:       true,
// 				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_default_service_account"},
// 			},
// 		},
// 	})
// }

func TestAccKubernetesNamespaceV1_generatedName(t *testing.T) {
	var conf corev1.Namespace
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_namespace_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version", "wait_for_default_service_account"},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_withSpecialCharacters(t *testing.T) {
	var conf corev1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_namespace_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_specialCharacters(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.myhost.co.uk/any-path", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_deleteTimeout(t *testing.T) {
	var conf corev1.Namespace
	nsName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_namespace_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesNamespaceV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesNamespaceV1Config_deleteTimeout(nsName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesNamespaceV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", nsName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
		},
	})
}

func testAccCheckKubernetesNamespaceV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_namespace_v1" {
			continue
		}

		resp, err := conn.CoreV1().Namespaces().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Namespace still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesNamespaceV1Exists(n string, obj *corev1.Namespace) resource.TestCheckFunc {
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

		out, err := conn.CoreV1().Namespaces().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesNamespaceV1Config_basic(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_addAnnotations(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    name = "%s"
  }
}
`, nsName)
}
func testAccKubernetesNamespaceV1Config_addLabels(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
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
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_smallerLists(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
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
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_noLists(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    generate_name = "%s"
  }
}
`, prefix)
}

func testAccKubernetesNamespaceV1Config_specialCharacters(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    annotations = {
      "myhost.co.uk/any-path" = "one"
      "Different"             = "1234"
    }

    labels = {
      "myhost.co.uk/any-path" = "one"
      "TestLabelThree"        = "three"
    }

    name = "%s"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_deleteTimeout(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
  timeouts {
    delete = "30m"
  }
}
`, nsName)
}

func testAccKubernetesNamespaceV1Config_wait_for_default_service_acccount(nsName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = "%s"
  }
  wait_for_default_service_account = "true"
}
`, nsName)
}

func testAccCheckKubernetesDefaultServiceAccountExists(n string,
	obj *corev1.ServiceAccount) resource.TestCheckFunc {
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

		namespace, _, err := idParts(rs.Primary.ID + "/")
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}
