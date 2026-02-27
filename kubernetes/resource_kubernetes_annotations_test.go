// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesAnnotations_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_annotations.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createConfigMap(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyConfigMap(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesAnnotations_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test2", "two"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test3", "three"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "ConfigMap"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func TestAccKubernetesAnnotations_template_cronjob(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_annotations.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createCronJob(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyCronJob(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesAnnotations_template_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "batch/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "CronJob"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_template_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "batch/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "CronJob"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.test2", "two"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_template_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "batch/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "CronJob"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test2", "two"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.test3", "three"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.test4", "four"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_template_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "batch/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "CronJob"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func TestAccKubernetesAnnotations_template_deployment(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_annotations.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createDeployment(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyDeployment(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesAnnotations_template_deployment_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_template_deployment_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.test2", "two"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_template_deployment_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test1", "one"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test2", "two"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.test3", "three"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.test4", "four"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
			{
				Config: testAccKubernetesAnnotations_template_deployment_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func TestAccKubernetesAnnotations_template_only(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_annotations.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createDeployment(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyDeployment(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesAnnotations_template_only(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "template_annotations.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func TestAccKubernetesAnnotations_resource_only(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_annotations.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createDeployment(name, namespace)
		},

		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyDeployment(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesAnnotations_resource_only(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "annotations.test", "test"),
					resource.TestCheckResourceAttr(resourceName, "field_manager", "tftest"),
				),
			},
		},
	})
}

func testAccKubernetesAnnotations_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "v1"
  kind        = "ConfigMap"
  metadata {
    name = %q
  }
  annotations   = {}
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "v1"
  kind        = "ConfigMap"
  metadata {
    name = %q
  }
  annotations = {
    "test1" = "one"
    "test2" = "two"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "v1"
  kind        = "ConfigMap"
  metadata {
    name = %q
  }
  annotations = {
    "test1" = "one"
    "test3" = "three"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_template_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "batch/v1"
  kind        = "CronJob"
  metadata {
    name = %q
  }
  annotations          = {}
  template_annotations = {}
  field_manager        = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_template_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "batch/v1"
  kind        = "CronJob"
  metadata {
    name = %q
  }
  annotations = {
    "test1" = "one"
  }
  template_annotations = {
    "test2" = "two"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_template_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "batch/v1"
  kind        = "CronJob"
  metadata {
    name = %q
  }
  annotations = {
    "test1" = "one"
    "test2" = "two"
  }
  template_annotations = {
    "test3" = "three"
    "test4" = "four"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_template_only(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name = %q
  }
  template_annotations = {
    "test" = "test"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_resource_only(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name = %q
  }
  annotations = {
    "test" = "test"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_template_deployment_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name = %q
  }
  annotations          = {}
  template_annotations = {}
  field_manager        = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_template_deployment_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name = %q
  }
  annotations = {
    "test1" = "one"
  }
  template_annotations = {
    "test2" = "two"
  }
  field_manager = "tftest"
}
`, name)
}

func testAccKubernetesAnnotations_template_deployment_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_annotations" "test" {
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name = %q
  }
  annotations = {
    "test1" = "one"
    "test2" = "two"
  }
  template_annotations = {
    "test3" = "three"
    "test4" = "four"
  }
  field_manager = "tftest"
}
`, name)
}

func createCronJob(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	cj := batchv1.CronJob{
		Spec: batchv1.CronJobSpec{
			Schedule: "0 * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							RestartPolicy: v1.RestartPolicyNever,
							Containers: []v1.Container{{
								Name:  "test",
								Image: "busybox",
								Command: []string{
									"echo", "hello world",
								},
							}},
						},
					},
				},
			},
		},
	}
	cj.SetName(name)
	cj.SetNamespace(namespace)
	_, err = conn.BatchV1().CronJobs(namespace).Create(ctx, &cj, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	return err
}

func destroyCronJob(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = conn.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return err
}

func createDeployment(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	d := appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  "test",
						Image: "busybox",
						Command: []string{
							"echo", "hello world",
						},
					}},
				},
			},
		},
	}
	d.SetName(name)
	d.SetNamespace(namespace)
	_, err = conn.AppsV1().Deployments(namespace).Create(ctx, &d, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	return err
}

func destroyDeployment(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = conn.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return err
}
