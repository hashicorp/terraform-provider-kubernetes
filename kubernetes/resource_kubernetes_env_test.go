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
			createEnv(t, name, namespace)
		},
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			err := confirmExistingEnvs(name, namespace)
			if err != nil {
				return err
			}
			return destroyEnv(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEnv_basic(name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "NGINX_HOST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "foobar.com"),
					resource.TestCheckResourceAttr(resourceName, "env.1.name", "NGINX_PORT"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value", "90"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "2"),
				),
			},
			{
				Config: testAccKubernetesEnv_modified(name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "NGINX_HOST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "hashicorp.com"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "1"),
				),
			},
		},
	})
}

func createEnv(t *testing.T, name, namespace string) error {
	conn, err := testAccProvider.Meta().(kubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	var deploy appsv1.Deployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "terraform",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "terraform",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
							Env: []v1.EnvVar{
								{
									Name:  "TEST",
									Value: "123",
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = conn.AppsV1().Deployments(namespace).Create(ctx, &deploy, metav1.CreateOptions{})
	if err != nil {
		t.Error("could not create test deployment")
		t.Fatal(err)
	}

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

func confirmExistingEnvs(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	deploy, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	env := deploy.Spec.Template.Spec.Containers[0].Env
	if len(env) == 0 {
		return fmt.Errorf("environment variables not managed by terraform were removed")
	}
	return err
}

func testAccKubernetesEnv_basic(name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_env" "test" {
		container = "nginx"
		api_version = "apps/v1"
		kind        = "Deployment"
		metadata {
		  name      = %q
		  namespace = %q
		}
		env {
			name = "NGINX_HOST"
			value = "foobar.com"
		}

		env {
			name = "NGINX_PORT"
			value = "90"
		}
	  }
	`, name, namespace)
}

func testAccKubernetesEnv_modified(name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_env" "test" {
		container = "nginx"
		api_version = "apps/v1"
		kind        = "Deployment"
		metadata {
		  name      = %q
		  namespace = %q
		}
		env {
			name = "NGINX_HOST"
			value = "hashicorp.com"
		}
	  }
	`, name, namespace)
}
