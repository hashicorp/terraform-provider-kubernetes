package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesStorageClass_minikube(t *testing.T) {
	var conf api.StorageClass
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_storage_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInMinikube(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStorageClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStorageClassConfig_basic(name, "k8s.io/minikube-hostpath"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "storage_provisioner", "k8s.io/minikube-hostpath"),
					resource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.0", "foo"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.1", "bar"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesStorageClassConfig_modified(name, "k8s.io/minikube-hostpath"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "storage_provisioner", "k8s.io/minikube-hostpath"),
					resource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Retain"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "WaitForFirstConsumer"),
					resource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "false"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.0", "foo"),
				),
			},
		},
	})
}

func TestAccKubernetesStorageClass_basic(t *testing.T) {
	var conf api.StorageClass
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_storage_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStorageClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStorageClassConfig_basic(name, "kubernetes.io/gce-pd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.1", "foo"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.0", "bar"),
					resource.TestCheckResourceAttr(resourceName, "parameters.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "parameters.type", "pd-ssd"),
					testAccCheckStorageClassParameters(&conf, map[string]string{"type": "pd-ssd"}),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesStorageClassConfig_modified(name, "kubernetes.io/gce-pd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Retain"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "WaitForFirstConsumer"),
					resource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "false"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.0", "foo"),
					resource.TestCheckResourceAttr(resourceName, "parameters.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "parameters.type", "pd-standard"),
					resource.TestCheckResourceAttr(resourceName, "parameters.zones", "us-west1-a,us-west1-b"),
					testAccCheckStorageClassParameters(&conf, map[string]string{"type": "pd-standard", "zones": "us-west1-a,us-west1-b"}),
				),
			},
			{
				Config: testAccKubernetesStorageClassConfig_noParameters(name, "kubernetes.io/gce-pd"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr(resourceName, "mount_options.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "parameters.%", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesStorageClass_allowedTopologies_minikube(t *testing.T) {
	var conf api.StorageClass
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_storage_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInMinikube(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStorageClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStorageClassConfig_allowedTopologies(name, "k8s.io/minikube-hostpath"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "allowed_topologies.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "allowed_topologies.0.match_label_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "allowed_topologies.0.match_label_expressions.0.key", "failure-domain.beta.kubernetes.io/zone"),
					resource.TestCheckResourceAttr(resourceName, "allowed_topologies.0.match_label_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "allowed_topologies.0.match_label_expressions.0.values.0", "us-west1-a"),
					resource.TestCheckResourceAttr(resourceName, "allowed_topologies.0.match_label_expressions.0.values.1", "us-west1-b"),
					testAccCheckStorageClassParameters(&conf, map[string]string{}),
				),
			},
		},
	})
}

func TestAccKubernetesStorageClass_generatedName(t *testing.T) {
	var conf api.StorageClass
	prefix := "tf-acc-test-gen-"
	resourceName := "kubernetes_storage_class.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInMinikube(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesStorageClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStorageClassConfig_generatedName(prefix, "k8s.io/minikube"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists(resourceName, &conf),
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
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func testAccCheckStorageClassParameters(m *api.StorageClass, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(expected) == 0 && len(m.Parameters) == 0 {
			return nil
		}
		if !reflect.DeepEqual(m.Parameters, expected) {
			return fmt.Errorf("%s parameters don't match.\nExpected: %q\nGiven: %q",
				m.Name, expected, m.Parameters)
		}
		return nil
	}
}

func testAccCheckKubernetesStorageClassDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_storage_class" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Storage class still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesStorageClassExists(n string, obj *api.StorageClass) resource.TestCheckFunc {
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

		name := rs.Primary.ID
		out, err := conn.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesStorageClassConfig_basic(name, provisioner string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class" "test" {
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

  storage_provisioner    = "%s"
  reclaim_policy         = "Delete"
  volume_binding_mode    = "Immediate"
  allow_volume_expansion = true

  mount_options = ["foo", "bar"]

  parameters = {
    type = "pd-ssd"
  }
}
`, name, provisioner)
}

func testAccKubernetesStorageClassConfig_modified(name, provisioner string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class" "test" {
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

  storage_provisioner    = "%s"
  reclaim_policy         = "Retain"
  volume_binding_mode    = "WaitForFirstConsumer"
  allow_volume_expansion = false

  mount_options = ["foo"]

  parameters = {
    type  = "pd-standard"
    zones = "us-west1-a,us-west1-b"
  }


}
`, name, provisioner)
}

func testAccKubernetesStorageClassConfig_noParameters(name, provisioner string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "%s"
}
`, name, provisioner)
}

func testAccKubernetesStorageClassConfig_generatedName(prefix, provisioner string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class" "test" {
  metadata {
    generate_name = "%s"
  }

  storage_provisioner = "%s"
}
`, prefix, provisioner)
}

func testAccKubernetesStorageClassConfig_allowedTopologies(name, provisioner string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "%s"
  allowed_topologies {
    match_label_expressions {
    key = "failure-domain.beta.kubernetes.io/zone"
    values = [
      "us-west1-a",
      "us-west1-b"
    ]
  }
 }
}
`, name, provisioner)
}
