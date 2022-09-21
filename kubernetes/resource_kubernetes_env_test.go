package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesEnv_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	resourceName := "kubernetes_env.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createEnv(name, namespace)
		},
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return destroyEnv(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEnv_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
				),
			},
			{
				Config: testAccKubernetesEnv_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "NGINX_HOST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "foobar.com"),
					resource.TestCheckResourceAttr(resourceName, "env.1.name", "NGINX_PORT"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value", "90"),
				),
			},
			{
				Config: testAccKubernetesEnv_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "NGINX_HOST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "hashicorp.com"),
					resource.TestCheckResourceAttr(resourceName, "env.1.name", "NGINX_PORT"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value", "90"),
				),
			},
			{
				Config: testAccKubernetesEnv_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
				),
			},
		},
	})
}

func createEnv(name, namespace string) error {
	conn, err := testAccProvider.Meta().(kubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	var deploy appsv1.Deployment = appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name: "nginx",
						},
					},
				},
			},
		},
	}
	deploy.SetName(name)
	_, err = conn.AppsV1().Deployments(namespace).Create(ctx, &deploy, metav1.CreateOptions{})

	return err
}

func destroyEnv(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = conn.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return err
}

func testAccKubernetesEnv_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_env" "test" {
	container = "nginx"
    api_version = "v1"
    kind        = "Deployment"
    metadata {
      name = %q
    }
    env{
		name = ""
		value = ""
	}
  }
`, name)
}

func testAccKubernetesEnv_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_env" "test" {
		container = "nginx"
		api_version = "v1"
		kind        = "Deployment"
		metadata {
		  name = %q
		}
		env{
			name = "NGINX_HOST"
			value = "foobar.com"
		}

		env{
			name = "NGINX_PORT"
			value = "90
		}
	  }
	`, name)
}

func testAccKubernetesEnv_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_env" "test" {
		container = "nginx"
		api_version = "v1"
		kind        = "Deployment"
		metadata {
		  name = %q
		}
		env{
			name = "NGINX_HOST"
			value = "hashicorp.com"
		}

		env{
			name = "NGINX_PORT"
			value = "90
		}
	  }
	`, name)
}
