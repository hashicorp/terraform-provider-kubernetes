package kubernetes

import (
	"fmt"
	"reflect"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/storage/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesStorageClass_basic(t *testing.T) {
	var conf api.StorageClass
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_storage_class.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStorageClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStorageClassConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists("kubernetes_storage_class.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.type", "pd-ssd"),
					testAccCheckStorageClassParameters(&conf, map[string]string{"type": "pd-ssd"}),
				),
			},
			{
				Config: testAccKubernetesStorageClassConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists("kubernetes_storage_class.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.Different", "1234"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "Different": "1234"}),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "reclaim_policy", "Retain"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "volume_binding_mode", "WaitForFirstConsumer"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "allow_volume_expansion", "false"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.type", "pd-standard"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.zones", "us-west1-a,us-west1-b"),
					testAccCheckStorageClassParameters(&conf, map[string]string{"type": "pd-standard", "zones": "us-west1-a,us-west1-b"}),
				),
			},
			{
				Config: testAccKubernetesStorageClassConfig_noParameters(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists("kubernetes_storage_class.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "storage_provisioner", "kubernetes.io/gce-pd"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "reclaim_policy", "Delete"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "volume_binding_mode", "Immediate"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "allow_volume_expansion", "true"),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "parameters.%", "0"),
					testAccCheckStorageClassParameters(&conf, map[string]string{}),
				),
			},
		},
	})
}

func TestAccKubernetesStorageClass_importBasic(t *testing.T) {
	resourceName := "kubernetes_storage_class.test"
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesStorageClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStorageClassConfig_basic(name),
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

func TestAccKubernetesStorageClass_generatedName(t *testing.T) {
	var conf api.StorageClass
	prefix := "tf-acc-test-gen-"

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_storage_class.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesStorageClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStorageClassConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesStorageClassExists("kubernetes_storage_class.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_storage_class.test", "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr("kubernetes_storage_class.test", "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_storage_class.test", "metadata.0.uid"),
				),
			},
		},
	})
}

func TestAccKubernetesStorageClass_importGeneratedName(t *testing.T) {
	resourceName := "kubernetes_storage_class.test"
	prefix := "tf-acc-test-gen-import-"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesStorageClassDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesStorageClassConfig_generatedName(prefix),
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
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_storage_class" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.StorageV1().StorageClasses().Get(name, meta_v1.GetOptions{})
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

		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset
		name := rs.Primary.ID
		out, err := conn.StorageV1().StorageClasses().Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesStorageClassConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class" "test" {
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

  storage_provisioner    = "kubernetes.io/gce-pd"
  reclaim_policy         = "Delete"
  volume_binding_mode    = "Immediate"
  allow_volume_expansion = true

  parameters = {
    type = "pd-ssd"
  }
}
`, name)
}

func testAccKubernetesStorageClassConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class" "test" {
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

  storage_provisioner    = "kubernetes.io/gce-pd"
  reclaim_policy         = "Retain"
  volume_binding_mode    = "WaitForFirstConsumer"
  allow_volume_expansion = false

  parameters = {
    type  = "pd-standard"
    zones = "us-west1-a,us-west1-b"
  }
}
`, name)
}

func testAccKubernetesStorageClassConfig_noParameters(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "kubernetes.io/gce-pd"
}
`, name)
}

func testAccKubernetesStorageClassConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class" "test" {
  metadata {
    generate_name = "%s"
  }

  storage_provisioner = "kubernetes.io/gce-pd"
}
`, prefix)
}
