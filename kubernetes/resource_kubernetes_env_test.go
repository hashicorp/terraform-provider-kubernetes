// Copyright (c) HashiCorp, Inc.
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
	"k8s.io/utils/ptr"
)

func TestAccKubernetesEnv_DeploymentBasic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_env.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if err := createEnv(t, name, namespace); err != nil {
				t.Fatal(err)
			}
		},
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
				Config: testAccKubernetesEnv_DeploymentBasic(secretName, configMapName, name, namespace),
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
				Config: testAccKubernetesEnv_DeploymentBasic_modified(secretName, configMapName, name, namespace),
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

func TestAccKubernetesEnv_CronJobBasic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_env.demo"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createCronJobEnv(t, name, namespace)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			err := confirmExistingCronJobEnvs(name, namespace)
			if err != nil {
				return err
			}
			return destroyCronJobEnv(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEnv_CronJobBasic(secretName, configMapName, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "batch/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "CronJob"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "TEST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "123"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "env.1.value_from.0.secret_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.config_map_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "3"),
				),
			},
			{
				Config: testAccKubernetesEnv_CronJobModified(secretName, configMapName, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "batch/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "CronJob"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "TEST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "123"),
					resource.TestCheckResourceAttr(resourceName, "env.1.name", "website"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value", "hashicorp.com"),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.secret_key_ref.0.key", "two"),
					resource.TestCheckResourceAttr(resourceName, "env.3.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr(resourceName, "env.3.value_from.0.config_map_key_ref.0.key", "three"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "4"),
				),
			},
		},
	})
}

func TestAccKubernetesEnv_Deployment_initContainer(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_env.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createInitContainerEnv(t, name, namespace)
		},
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
				Config: testAccKubernetesEnv_Deployment_initContainer(secretName, configMapName, name, namespace),
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
				Config: testAccKubernetesEnv_modified_initContainer(secretName, configMapName, name, namespace),
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

func TestAccKubernetesEnv_CronJob_initContainer(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := "default"
	secretName := acctest.RandomWithPrefix("tf-acc-test")
	configMapName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_env.demo"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createCronJobInitContainerEnv(t, name, namespace)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			err := confirmExistingCronJobEnvs(name, namespace)
			if err != nil {
				return err
			}
			return destroyCronJobEnv(name, namespace)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesEnv_CronJob_initContainer(secretName, configMapName, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "batch/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "CronJob"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "TEST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "123"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "env.1.value_from.0.secret_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.config_map_key_ref.0.key", "one"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "3"),
				),
			},
			{
				Config: testAccKubernetesEnv_CronJobModified_initContainer(secretName, configMapName, name, namespace),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "api_version", "batch/v1"),
					resource.TestCheckResourceAttr(resourceName, "kind", "CronJob"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "env.0.name", "TEST"),
					resource.TestCheckResourceAttr(resourceName, "env.0.value", "123"),
					resource.TestCheckResourceAttr(resourceName, "env.1.name", "website"),
					resource.TestCheckResourceAttr(resourceName, "env.1.value", "hashicorp.com"),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.secret_key_ref.0.name", secretName),
					resource.TestCheckResourceAttr(resourceName, "env.2.value_from.0.secret_key_ref.0.key", "two"),
					resource.TestCheckResourceAttr(resourceName, "env.3.value_from.0.config_map_key_ref.0.name", configMapName),
					resource.TestCheckResourceAttr(resourceName, "env.3.value_from.0.config_map_key_ref.0.key", "three"),
					resource.TestCheckResourceAttr(resourceName, "env.#", "4"),
				),
			},
		},
	})
}

func createInitContainerEnv(t *testing.T, name, namespace string) error {
	conn, err := testAccProvider.Meta().(providerMetadata).MainClientset()
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
					InitContainers: []v1.Container{
						{
							Name:  "hello",
							Image: "busybox",
							Env: []v1.EnvVar{
								{
									Name:  "one",
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

func createEnv(t *testing.T, name, namespace string) error {
	conn, err := testAccProvider.Meta().(providerMetadata).MainClientset()
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

func createCronJobEnv(t *testing.T, name, namespace string) error {
	conn, err := testAccProvider.Meta().(providerMetadata).MainClientset()
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

	_, err = conn.BatchV1().CronJobs(namespace).Create(ctx, &cronjob, metav1.CreateOptions{})
	if err != nil {
		t.Error("could not create test cronjob")
		t.Fatal(err)
	}

	return err
}

func createCronJobInitContainerEnv(t *testing.T, name, namespace string) error {
	conn, err := testAccProvider.Meta().(providerMetadata).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()

	failJobLimit := ptr.To(int32(2))
	startingDeadlineSeconds := ptr.To(int64(2))
	successfulJobsLimit := ptr.To(int32(2))
	boLimit := ptr.To(int32(2))
	ttl := ptr.To(int32(2))
	cronjob := batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.CronJobSpec{
			StartingDeadlineSeconds:    startingDeadlineSeconds,
			FailedJobsHistoryLimit:     failJobLimit,
			SuccessfulJobsHistoryLimit: successfulJobsLimit,
			ConcurrencyPolicy:          "Replace",
			Schedule:                   "1 0 * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					BackoffLimit:            boLimit,
					TTLSecondsAfterFinished: ttl,
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
							InitContainers: []v1.Container{
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
			},
		},
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
	err = conn.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	return err
}

func destroyCronJobEnv(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	err = conn.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
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
	deployEnv := deploy.Spec.Template.Spec.Containers[0].Env
	if len(deployEnv) == 0 {
		return fmt.Errorf("environment variables not managed by terraform were removed")
	}
	return err
}

func confirmExistingCronJobEnvs(name, namespace string) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.Background()
	cronjob, err := conn.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	cronjobEnv := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env
	if len(cronjobEnv) == 0 {
		return fmt.Errorf("environment variables not managed by terraform were removed")
	}
	return err
}

func testAccKubernetesEnv_DeploymentBasic(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map_v1" "test" {
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
        name     = "${kubernetes_secret_v1.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }

}
	`, secretName, configMapName, name, namespace)
}

func testAccKubernetesEnv_DeploymentBasic_modified(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map_v1" "test" {
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
        name     = "${kubernetes_secret_v1.test.metadata.0.name}"
        key      = "two"
        optional = true
      }
    }
  }


  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
        key      = "three"
        optional = true
      }
    }
  }
}
	`, secretName, configMapName, name, namespace)
}

func testAccKubernetesEnv_modified_initContainer(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_env" "test" {
  init_container = "hello"
  api_version    = "apps/v1"
  kind           = "Deployment"
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
        name     = "${kubernetes_secret_v1.test.metadata.0.name}"
        key      = "two"
        optional = true
      }
    }
  }


  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
        key      = "three"
        optional = true
      }
    }
  }
}
	`, secretName, configMapName, name, namespace)
}

func testAccKubernetesEnv_CronJobBasic(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_env" "demo" {
  container   = "hello"
  api_version = "batch/v1"
  kind        = "CronJob"
  metadata {
    name      = "%s"
    namespace = "%s"
  }
  env {
    name  = "TEST"
    value = "123"
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_SECRET"

    value_from {
      secret_key_ref {
        name     = "${kubernetes_secret_v1.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }


  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }
}
	`, secretName, configMapName, name, namespace)
}

func testAccKubernetesEnv_CronJobModified(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_env" "demo" {
  container   = "hello"
  api_version = "batch/v1"
  kind        = "CronJob"
  metadata {
    name      = "%s"
    namespace = "%s"
  }
  env {
    name  = "TEST"
    value = "123"
  }

  env {
    name  = "website"
    value = "hashicorp.com"
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_SECRET"

    value_from {
      secret_key_ref {
        name     = "${kubernetes_secret_v1.test.metadata.0.name}"
        key      = "two"
        optional = true
      }
    }
  }


  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
        key      = "three"
        optional = true
      }
    }
  }
}
	`, secretName, configMapName, name, namespace)
}

func testAccKubernetesEnv_CronJobModified_initContainer(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_env" "demo" {
  init_container = "nginx"
  api_version    = "batch/v1"
  kind           = "CronJob"
  metadata {
    name      = "%s"
    namespace = "%s"
  }
  env {
    name  = "TEST"
    value = "123"
  }

  env {
    name  = "website"
    value = "hashicorp.com"
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_SECRET"

    value_from {
      secret_key_ref {
        name     = "${kubernetes_secret_v1.test.metadata.0.name}"
        key      = "two"
        optional = true
      }
    }
  }


  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
        key      = "three"
        optional = true
      }
    }
  }
}
	`, secretName, configMapName, name, namespace)
}

func testAccKubernetesEnv_Deployment_initContainer(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_env" "test" {
  init_container = "hello"
  api_version    = "apps/v1"
  kind           = "Deployment"
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
        name     = "${kubernetes_secret_v1.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }

}
	`, secretName, configMapName, name, namespace)
}

func testAccKubernetesEnv_CronJob_initContainer(secretName, configMapName, name, namespace string) string {
	return fmt.Sprintf(`resource "kubernetes_secret_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "first"
  }
}

resource "kubernetes_config_map_v1" "test" {
  metadata {
    name = "%s"
  }

  data = {
    one = "ONE"
  }
}

resource "kubernetes_env" "demo" {
  init_container = "nginx"
  api_version    = "batch/v1"
  kind           = "CronJob"
  metadata {
    name      = "%s"
    namespace = "%s"
  }
  env {
    name  = "TEST"
    value = "123"
  }

  env {
    name = "EXPORTED_VARIABLE_FROM_SECRET"

    value_from {
      secret_key_ref {
        name     = "${kubernetes_secret_v1.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }


  env {
    name = "EXPORTED_VARIABLE_FROM_CONFIG_MAP"
    value_from {
      config_map_key_ref {
        name     = "${kubernetes_config_map_v1.test.metadata.0.name}"
        key      = "one"
        optional = true
      }
    }
  }
}
	`, secretName, configMapName, name, namespace)
}
