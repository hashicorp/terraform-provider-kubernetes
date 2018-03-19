package kubernetes

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	api "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
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
				Config: testAccKubernetesPersistentVolumeConfig_googleCloud_basic(name, diskName, zone),
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
						"TestLabelOne":                             "one",
						"TestLabelTwo":                             "two",
						"TestLabelThree":                           "three",
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
						"TestLabelOne":                             "one",
						"TestLabelTwo":                             "two",
						"TestLabelThree":                           "three",
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
						"TestLabelOne":                             "one",
						"TestLabelTwo":                             "two",
						"TestLabelThree":                           "three",
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
						"TestLabelOne":                             "one",
						"TestLabelTwo":                             "two",
						"TestLabelThree":                           "three",
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

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_googleCloud_basic(name, diskName, zone),
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

	region := os.Getenv("GOOGLE_REGION")
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
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{
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
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, "/custom/testing/path"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.0.path", "/custom/testing/path"),
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
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, "/first/path"),
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
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, "/second/path"),
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

func testAccCheckKubernetesPersistentVolumeDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*kubernetes.Clientset)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return fmt.Errorf("Persistent Volume still exists: %s", rs.Primary.ID)
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

		conn := testAccProvider.Meta().(*kubernetes.Clientset)
		name := rs.Primary.ID
		out, err := conn.CoreV1().PersistentVolumes().Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPersistentVolumeConfig_googleCloud_basic(name, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}
	spec {
		capacity {
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
  size = 10
}
`, name, diskName, zone)
}

func testAccKubernetesPersistentVolumeConfig_googleCloud_modified(name, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}
	spec {
		capacity {
			storage = "42Mi"
		}
		access_modes = ["ReadWriteOnce", "ReadOnlyMany"]
		persistent_volume_source {
			gce_persistent_disk {
				fs_type = "ntfs"
				pd_name = "${google_compute_disk.test.name}"
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
  size = 10
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
		capacity {
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
  size = 12
}
`, name, diskName, zone)
}

func testAccKubernetesPersistentVolumeConfig_aws_basic(name, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}
	spec {
		capacity {
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
	size = 10
	tags {
		Name = "%s"
	}
}
`, name, zone, diskName)
}

func testAccKubernetesPersistentVolumeConfig_aws_modified(name, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		annotations {
			TestAnnotationOne = "one"
			TestAnnotationTwo = "two"
		}
		labels {
			TestLabelOne = "one"
			TestLabelTwo = "two"
			TestLabelThree = "three"
		}
		name = "%s"
	}
	spec {
		capacity {
			storage = "42Mi"
		}
		access_modes = ["ReadWriteOnce"]
		persistent_volume_source {
			aws_elastic_block_store {
				volume_id = "${aws_ebs_volume.test.id}"
				fs_type = "io1"
				partition = 1
				read_only = true
			}
		}
	}
}

resource "aws_ebs_volume" "test" {
	availability_zone = "%s"
	size = 10
	tags {
		Name = "%s"
	}
}
`, name, zone, diskName)
}

func testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, path string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		name = "%s"
	}
	spec {
		capacity {
			storage = "123Gi"
		}
		access_modes = ["ReadWriteOnce"]
		persistent_volume_source {
			host_path {
				path = "%s"
			}
		}
	}
}`, name, path)
}

func testAccKubernetesPersistentVolumeConfig_cephFsSecretRef(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		name = "%s"
	}
	spec {
		capacity {
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
}`, name)
}

func testAccKubernetesPersistentVolumeConfig_storageClass(name, diskName, storageClassName, storageClassName2, zone, refName string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
	metadata {
		name = "%s"
	}
	spec {
		capacity {
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
  size = 12
}

resource "kubernetes_storage_class" "test" {
	metadata {
		name = "%s"
	}
	storage_provisioner = "kubernetes.io/gce-pd"
	parameters {
		type = "pd-ssd"
	}
}

resource "kubernetes_storage_class" "test2" {
	metadata {
		name = "%s"
	}
	storage_provisioner = "kubernetes.io/gce-pd"
	parameters {
		type = "pd-standard"
	}
}
`, name, refName, diskName, zone, storageClassName, storageClassName2)
}
