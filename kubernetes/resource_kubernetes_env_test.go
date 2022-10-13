package kubernetes

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesEnv_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")
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
				Config: testAccKubernetesEnv_basic(secretName, configMapName, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "NGINX_HOST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "foobar.com"),
					resource.TestCheckResourceAttr(resourceName, "env.1.name", "NGINX_PORT"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value", "90"),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.secret_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "env.3.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr(resourceName, "env.3.value_from.0.config_map_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "4"),
				),
			},
			{
				Config: testAccKubernetesEnv_modified(secretName, configMapName, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "apps/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "Deployment"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "NGINX_HOST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "hashicorp.com"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "env.1.value_from.0.secret_key_ref.0.key", "two"),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.config_map_key_ref.0.key", "three"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "3"),
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

	var failJobLimit int32 = 5
	var startingDeadlineSeconds int64 = 2
	var successfulJobsLimit int32 = 2
	var boLimit int32 = 2
	var ttl int32 = 2
	var cronjob batchv1.CronJob = batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.CronJobSpec{
			StartingDeadlineSeconds:    &startingDeadlineSeconds,
			FailedJobsHistoryLimit:     &failJobLimit,
			SuccessfulJobsHistoryLimit: &successfulJobsLimit,
			ConcurrencyPolicy:          "Replace",
			Schedule:                   "1 0 * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit:            &boLimit,
					TTLSecondsAfterFinished: &ttl,
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							RestartPolicy: "Never",
							Containers: []v1.Container{
								{
									Name:    "hello",
									Image:   "busybox",
									Command: []string{"/bin/sh", "-c", "date; echo Goodbye from the Kubernetes cluster"},
									Env: []v1.EnvVar{
										{
											Name:  "kubernetes",
											Value: "80",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

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

	_, err = conn.BatchV1().CronJobs(namespace).Create(ctx, &cronjob, metav1.CreateOptions{})
	if err != nil {
		t.Error("could not create test cronjob")
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

	err = conn.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	err = conn.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return err
}

func confirmExistingEnvs(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	cronjob, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	deploy, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	cronjobEnv := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	deployEnv := deploy.Spec.Template.Spec.Containers[0].Env
	if len(deployEnv) == 0 || len(cronjobEnv) == 0 {
		return fmt.Errorf("environment variables not managed by terraform were removed")
	}
	return err
}

func testAccKubernetesEnv_basic(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_env" "test" {
  container   = "nginx"
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name      = %q
    namespace = %q
  }
  env {
    name  = "NGINX_HOST"
    value = "foobar.com"
  }

  env {
    name  = "NGINX_PORT"
    value = "90"
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_SECRET"
    value_from {
      secret_key_ref {
        name     = "${kubernetes_secret.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }

}
	`, secretName, configMapName, name, namespace)
}

func testAccKubernetesEnv_modified(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_env" "test" {
  container   = "nginx"
  api_version = "apps/v1"
  kind        = "Deployment"
  metadata {
    name      = %q
    namespace = %q
  }
  env {
    name  = "NGINX_HOST"
    value = "hashicorp.com"
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_SECRET"

    value_from {
      secret_key_ref {
        name     = "${kubernetes_secret.test.metadata.0.name}"
        key      = "two"
        optional = true
      }
    }
  }


  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map.test.metadata.0.name}"
        key      = "three"
        optional = true
      }
    }
  }
}
	`, secretName, configMapName, name, namespace)
}
