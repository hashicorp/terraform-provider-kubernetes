package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesLabels_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_labels.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createConfigMap(name, namespace)
		},
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyConfigMap(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLabels_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "0"),
				),
			},
			{
				Config: testAccKubernetesLabels_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "labels.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "labels.test2", "two"),
				),
			},
			{
				Config: testAccKubernetesLabels_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "labels.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "labels.test3", "three"),
				),
			},
			{
				Config: testAccKubernetesLabels_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "0"),
				),
			},
		},
	})
}

func createConfigMap(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	cm := v1.ConfigMap{}
	cm.SetName(name)
	cm.SetNamespace(namespace)
	_, err = conn.CoreV1().ConfigMaps(namespace).Create(ctx, &cm, metav1.CreateOptions{})
	return err
}

func destroyConfigMap(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = conn.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return err
}

func testAccKubernetesLabels_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_labels" "test" {
    api_version = "v1"
    kind        = "ConfigMap"
    metadata {
      name = %q
    }
    labels = {}
  }
`, name)
}

func testAccKubernetesLabels_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_labels" "test" {
    api_version = "v1"
    kind        = "ConfigMap"
    metadata {
      name = %q
    }
    labels = {
      "test1" = "one"
      "test2" = "two"
    }
  }
`, name)
}

func testAccKubernetesLabels_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_labels" "test" {
    api_version = "v1"
    kind        = "ConfigMap"
    metadata {
      name = %q
    }
    labels = {
      "test1" = "one"
      "test3" = "three"
    }
  }
`, name)
}
