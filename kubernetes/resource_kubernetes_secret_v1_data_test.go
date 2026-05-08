// Copyright IBM Corp. 2017, 2026
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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Handling the case for a empty secret
func TestAccKubernetesSecretV1Data_empty(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_secret_v1_data.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createEmptySecret(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroySecret(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Data_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func createEmptySecret(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	secret := v1.Secret{}
	secret.SetName(name)
	secret.SetNamespace(namespace)
	secret.Data = map[string][]byte{}
	_, err = conn.CoreV1().Secrets(namespace).Create(ctx, &secret, metav1.CreateOptions{
		FieldManager: "tftest",
	})
	return err
}

// Handling the case of secret creation with basic data
func TestAccKubernetesSecretV1Data_basic_data(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_secret_v1_data.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createSecretWithData(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroySecret(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Data_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.key1", "value1"),
					resource.TestCheckResourceAttr(resourceName, "data.key2", "value2"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func createSecretWithData(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	data := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}

	secret := v1.Secret{}
	secret.SetName(name)
	secret.SetNamespace(namespace)
	secret.Data = data
	_, err = conn.CoreV1().Secrets(namespace).Create(ctx, &secret, metav1.CreateOptions{
		FieldManager: "tftest",
	})
	return err
}

// Handling the case for a modified secret
func TestAccKubernetesSecretV1Data_modified(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_secret_v1_data.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createModifiedSecret(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroySecret(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesSecretV1Data_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "data.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "data.key1", "one"),
					resource.TestCheckResourceAttr(resourceName, "data.key3", "three"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func createModifiedSecret(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	data := map[string][]byte{
		"key1": []byte("one"),
		"key3": []byte("three"),
	}

	secret := v1.Secret{}
	secret.SetName(name)
	secret.SetNamespace(namespace)
	secret.Data = data
	_, err = conn.CoreV1().Secrets(namespace).Create(ctx, &secret, metav1.CreateOptions{
		FieldManager: "tftest",
	})
	return err
}

// deletes a kubernetes secret
func destroySecret(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = conn.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return err
}

// Handling the case where it attempts to read a secret that doesnt exist in the cluster
func TestAcctKubernetesSecretV1Data_validation(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				// Testing a non-existing secret
				Config:      testAccKubernetesSecretV1Data_empty(name),
				ExpectError: regexp.MustCompile("The Secret .* does not exist"),
			},
		},
	})
}

// Generate config for creating a secret with empty data
func testAccKubernetesSecretV1Data_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1_data" "test" {
  metadata {
    name = %q
  }
  data          = {}
  field_manager = "tftest"

}
`, name)
}

// Generate some basic config, with a secret with basic data
func testAccKubernetesSecretV1Data_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret_v1_data" "test" {
  metadata {
    name = %q
  }
  data = {
    "key1" = "value1"
    "key2" = "value2"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesSecretV1Data_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_secret_v1_data" "test" {
  metadata {
    name = %q
  }
  data = {
    "key1" = "one"
    "key3" = "three"
  }
  field_manager = "tftest"
}
`, name)
}
