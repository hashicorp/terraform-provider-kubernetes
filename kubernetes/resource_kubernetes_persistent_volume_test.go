package kubernetes

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPersistentVolume_googleCloud_basic(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)

	region := os.Getenv("GOOGLE_REGION")
	zone := os.Getenv("GOOGLE_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_googleCloud_basic(name, diskName, zone, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{
						"TestLabelOne":   "one",
						"TestLabelTwo":   "two",
						"TestLabelThree": "three",
						"failure-domain.beta.kubernetes.io/region": region,
						"failure-domain.beta.kubernetes.io/zone":   zone,
					}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name", diskName),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_googleCloud_modified(name, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{
						"TestLabelOne":   "one",
						"TestLabelTwo":   "two",
						"TestLabelThree": "three",
						"failure-domain.beta.kubernetes.io/region": region,
						"failure-domain.beta.kubernetes.io/zone":   zone,
					}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "42Mi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.3887104832", "ReadOnlyMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.fs_type", "ntfs"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name", diskName),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.read_only", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_aws_basic(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)

	region := os.Getenv("AWS_DEFAULT_REGION")
	zone := os.Getenv("AWS_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNoAwsSettingsFound(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_aws_basic(name, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{
						"TestLabelOne":   "one",
						"TestLabelTwo":   "two",
						"TestLabelThree": "three",
						"failure-domain.beta.kubernetes.io/region": region,
						"failure-domain.beta.kubernetes.io/zone":   zone,
					}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.volume_id"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_aws_modified(name, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{
						"TestLabelOne":   "one",
						"TestLabelTwo":   "two",
						"TestLabelThree": "three",
						"failure-domain.beta.kubernetes.io/region": region,
						"failure-domain.beta.kubernetes.io/zone":   zone,
					}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "42Mi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.volume_id"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.fs_type", "io1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.partition", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.read_only", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_googleCloud_importBasic(t *testing.T) {
	resourceName := "kubernetes_persistent_volume.test"
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-import-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)

	zone := os.Getenv("GOOGLE_ZONE")
	region := os.Getenv("GOOGLE_REGION")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_googleCloud_basic(name, diskName, zone, region),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_googleCloud_volumeSource(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)

	zone := os.Getenv("GOOGLE_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_googleCloud_volumeSource(name, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, "/custom/testing/path", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.0.path", "/custom/testing/path"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.0.type", ""),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_hostPath_volumeSource(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, "/first/path", "DirectoryOrCreate"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.0.path", "/first/path"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.0.type", "DirectoryOrCreate"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, "/second/path", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.0.path", "/second/path"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_local_volumeSource(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_local_volumeSource(name, "/first/path", "test-node"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.local.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.local.0.path", "/first/path"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_local_volumeSource(name, "/second/path", "test-node"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.local.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.local.0.path", "/second/path"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_cephFsSecretRef(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_cephFsSecretRef(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "2Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1254135962", "ReadWriteMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.#", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.2848821021", "10.16.154.78:6789"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.4263435410", "10.16.154.82:6789"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.secret_ref.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.ceph_fs.0.secret_ref.0.name", "ceph-secret"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_storageClass(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	name := fmt.Sprintf("tf-acc-test-%s", randString)
	storageClassName := fmt.Sprintf("tf-acc-test-sc-%s", randString)
	secondStorageClassName := fmt.Sprintf("tf-acc-test-sc2-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)
	zone := os.Getenv("GOOGLE_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_storageClass(name, diskName, storageClassName, secondStorageClassName, zone, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1254135962", "ReadWriteMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.storage_class_name", storageClassName),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_storageClass(name, diskName, storageClassName, secondStorageClassName, zone, "test2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "123Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1254135962", "ReadWriteMany"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.storage_class_name", secondStorageClassName),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_hostPath_nodeAffinity(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	selectorLabel := fmt.Sprintf("tf-acc-test-na-label-%s", randString)
	selectorValue := fmt.Sprintf("tf-acc-test-na-value-%s", randString)
	selectorValueHash := schema.HashString(selectorValue)

	replacedRandString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	replacedSelectorLabel := fmt.Sprintf("tf-acc-test-na-label-%s", replacedRandString)
	replacedSelectorValue := fmt.Sprintf("tf-acc-test-na-value-%s", replacedRandString)
	replacedSelectorValueHash := schema.HashString(replacedSelectorValue)

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_nodeAffinity_match(name, selectorLabel, selectorValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.key", selectorLabel),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.values.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", fmt.Sprintf("spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.values.%d", selectorValueHash), selectorValue),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_nodeAffinity_match(name, replacedSelectorLabel, replacedSelectorValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.key", replacedSelectorLabel),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.values.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", fmt.Sprintf("spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.values.%d", replacedSelectorValueHash), replacedSelectorValue),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_withoutNodeAffinity(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "2Gi"),
					resource.TestCheckNoResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_nodeAffinity_match(name, selectorLabel, selectorValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.key", selectorLabel),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.values.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", fmt.Sprintf("spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions.0.values.%d", selectorValueHash), selectorValue),
				),
			},
		},
	})
}

func waitForPersistenceVolumeDeleted(pvName string, poll, timeout time.Duration) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		_, err := conn.CoreV1().PersistentVolumes().Get(pvName, meta_v1.GetOptions{})
		if err != nil && apierrs.IsNotFound(err) {
			return nil
		}
	}
	return fmt.Errorf("Persistent Volume still exists: %s", pvName)
}

func testAccCheckKubernetesPersistentVolumeDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*KubeClientsets).MainClientset
	timeout := 5 * time.Second
	poll := 1 * time.Second

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return waitForPersistenceVolumeDeleted(name, poll, timeout)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPersistentVolumeExists(n string, obj *api.PersistentVolume) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := testAccProvider.Meta().(*KubeClientsets).MainClientset
		name := rs.Primary.ID
		out, err := conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPersistentVolumeConfig_googleCloud_basic(name, diskName, zone string, region string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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

  spec {
    capacity = {
      storage = "123Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      gce_persistent_disk {
        pd_name = "${google_compute_disk.test.name}"
      }
    }

    node_affinity {
      required {
        node_selector_term {
          match_expressions {
            key      = "test"
            operator = "Exists"
		  }
		  match_expressions {
			  key = "failure-domain.beta.kubernetes.io/zone"
			  operator = "In"
			  values = ["%s"]
		  }
		  match_expressions {
				key = "failure-domain.beta.kubernetes.io/region"
				operator = "In"
				values = ["%s"]
		  }
        }
      }
    }
  }
}

resource "google_compute_disk" "test" {
  name  = "%s"
  type  = "pd-ssd"
  zone  = "%s"
  image = "debian-8-jessie-v20170523"
  size  = 10
}
`, name, zone, region, diskName, zone)
}

func testAccKubernetesPersistentVolumeConfig_googleCloud_modified(name, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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

  spec {
    capacity = {
      storage = "42Mi"
    }

    access_modes = ["ReadWriteOnce", "ReadOnlyMany"]

    persistent_volume_source {
      gce_persistent_disk {
        fs_type   = "ntfs"
        pd_name   = "${google_compute_disk.test.name}"
        read_only = true
      }
    }
  }
}

resource "google_compute_disk" "test" {
  name  = "%s"
  type  = "pd-ssd"
  zone  = "%s"
  image = "debian-8-jessie-v20170523"
  size  = 10
}
`, name, diskName, zone)
}

func testAccKubernetesPersistentVolumeConfig_googleCloud_volumeSource(name, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "123Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      gce_persistent_disk {
        pd_name = "${google_compute_disk.test.name}"
      }
    }
  }
}

resource "google_compute_disk" "test" {
  name  = "%s"
  type  = "pd-ssd"
  zone  = "%s"
  image = "debian-8-jessie-v20170523"
  size  = 12
}
`, name, diskName, zone)
}

func testAccKubernetesPersistentVolumeConfig_aws_basic(name, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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

  spec {
    capacity = {
      storage = "123Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      aws_elastic_block_store {
        volume_id = "${aws_ebs_volume.test.id}"
      }
    }
  }
}

resource "aws_ebs_volume" "test" {
  availability_zone = "%s"
  size              = 10

  tags = {
    Name = "%s"
  }
}
`, name, zone, diskName)
}

func testAccKubernetesPersistentVolumeConfig_aws_modified(name, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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

  spec {
    capacity = {
      storage = "42Mi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      aws_elastic_block_store {
        volume_id = "${aws_ebs_volume.test.id}"
        fs_type   = "io1"
        partition = 1
        read_only = true
      }
    }
  }
}

resource "aws_ebs_volume" "test" {
  availability_zone = "%s"
  size              = 10

  tags = {
    Name = "%s"
  }
}
`, name, zone, diskName)
}

func testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, path, typ string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "123Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      host_path {
        path = "%s"
        type = "%s"
      }
    }
  }
}
`, name, path, typ)
}

func testAccKubernetesPersistentVolumeConfig_local_volumeSource(name, path, hostname string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "123Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      local {
        path = "%s"
      }
    }
    node_affinity {
      required {
        node_selector_term {
          match_expressions {
            key      = "kubernetes.io/hostname"
            operator = "In"
            values   = ["%s"]
          }
        }
      }
    }
  }
}
`, name, path, hostname)
}

func testAccKubernetesPersistentVolumeConfig_cephFsSecretRef(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "2Gi"
    }

    access_modes = ["ReadWriteMany"]

    persistent_volume_source {
      ceph_fs {
        monitors = ["10.16.154.78:6789", "10.16.154.82:6789"]

        secret_ref {
          name = "ceph-secret"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesPersistentVolumeConfig_storageClass(name, diskName, storageClassName, storageClassName2, zone, refName string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "123Gi"
    }

    access_modes = ["ReadWriteMany"]

    persistent_volume_source {
      gce_persistent_disk {
        pd_name = "${google_compute_disk.test.name}"
      }
    }

    storage_class_name = "${kubernetes_storage_class.%s.metadata.0.name}"
  }
}

resource "google_compute_disk" "test" {
  name  = "%s"
  type  = "pd-ssd"
  zone  = "%s"
  image = "debian-8-jessie-v20170523"
  size  = 12
}

resource "kubernetes_storage_class" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "kubernetes.io/gce-pd"

  parameters = {
    type = "pd-ssd"
  }
}

resource "kubernetes_storage_class" "test2" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "kubernetes.io/gce-pd"

  parameters = {
    type = "pd-standard"
  }
}
`, name, refName, diskName, zone, storageClassName, storageClassName2)
}

func testAccKubernetesPersistentVolumeConfig_hostPath_nodeAffinity(name, nodeAffinity string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }
  spec {
    capacity = {
      storage = "2Gi"
    }
    access_modes = ["ReadWriteMany"]
    persistent_volume_source {
      host_path {
        path = "/mnt/local-volume"
      }
    }
    %s
  }
}`, name, nodeAffinity)
}

func testAccKubernetesPersistentVolumeConfig_hostPath_nodeAffinity_match(name, selectorLabel, selectorValue string) string {
	nodeAffinity := fmt.Sprintf(`
    node_affinity {
      required {
        node_selector_term {
          match_expressions {
            key = "%s"
            operator = "In"
            values = ["%s"]
          }
        }
      }
    }
  `, selectorLabel, selectorValue)
	return testAccKubernetesPersistentVolumeConfig_hostPath_nodeAffinity(name, nodeAffinity)
}

func testAccKubernetesPersistentVolumeConfig_hostPath_withoutNodeAffinity(name string) string {
	return testAccKubernetesPersistentVolumeConfig_hostPath_nodeAffinity(name, "")
}
