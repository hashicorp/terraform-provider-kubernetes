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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesIgnoreKubernetesMetadata_basic(t *testing.T) {
	namespaceName := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	ignoreKubernetesMetadata := "terraform.io/provider"
	dataSourceName := "data.kubernetes_namespace_v1.this"
	oneOrMore := regexp.MustCompile(`^[1-9][0-9]*$`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createNamespaceIgnoreKubernetesMetadata(namespaceName, ignoreKubernetesMetadata)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return deleteNamespaceIgnoreKubernetesMetadata(namespaceName)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesIgnoreKubernetesMetadataProviderConfig(namespaceName, ignoreKubernetesMetadata),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "metadata.0.annotations.%", oneOrMore),
					resource.TestMatchResourceAttr(dataSourceName, "metadata.0.labels.%", oneOrMore),
				),
			},
		},
	})
}

func testAccKubernetesIgnoreKubernetesMetadataProviderConfig(namespaceName string, ignoreKubernetesMetadata string) string {
	return fmt.Sprintf(`provider "kubernetes" {
  ignore_annotations = [
    "%s",
  ]
  ignore_labels = [
    "%s",
  ]
}

data "kubernetes_namespace_v1" "this" {
  metadata {
    name = "%s"
  }
}
`, ignoreKubernetesMetadata, ignoreKubernetesMetadata, namespaceName)
}

func createNamespaceIgnoreKubernetesMetadata(namespaceName string, ignoreKubernetesMetadata string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ns := corev1.Namespace{}
	ns.SetName(namespaceName)
	m := map[string]string{ignoreKubernetesMetadata: "kubernetes"}
	ns.SetAnnotations(m)
	ns.SetLabels(m)
	_, err = conn.CoreV1().Namespaces().Create(context.Background(), &ns, metav1.CreateOptions{})

	return err
}

func deleteNamespaceIgnoreKubernetesMetadata(namespaceName string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	err = conn.CoreV1().Namespaces().Delete(context.Background(), namespaceName, metav1.DeleteOptions{})
	return err
}
