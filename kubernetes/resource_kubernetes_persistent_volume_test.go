package kubernetes

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPersistentVolume_azure_basic(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tfacctest%s", randString)
	location := os.Getenv("TF_VAR_location")
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	diskURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + name + "/providers/Microsoft.Compute/disks/" + name

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNotRunningInAks(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_azure_managedDisk(name, location) +
					testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKind(name, diskURI, "Managed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("azurerm_resource_group.test", "name", name),
					resource.TestCheckResourceAttr("azurerm_managed_disk.test", "name", name),
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.azure_disk.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.azure_disk.0.kind", "Managed"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_azure_ManagedDiskExpectErrors(t *testing.T) {
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tfacctest%s", randString)
	location := os.Getenv("TF_VAR_location")
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	diskURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + name + "/providers/Microsoft.Compute/disks/" + name
	wantError := persistentVolumeAzureManagedError

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); skipIfNotRunningInAks(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{ // Expect an error when using a Managed Disk with `kind` omitted.
				Config: testAccKubernetesPersistentVolumeConfig_azure_managedDisk(name, location) +
					testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKindOmitted(name, diskURI),
				ExpectError: regexp.MustCompile(wantError),
			},
			{ // Expect an error when using `kind = "Shared"` with a Managed Disk.
				Config: testAccKubernetesPersistentVolumeConfig_azure_managedDisk(name, location) +
					testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKind(name, diskURI, "Shared"),
				ExpectError: regexp.MustCompile(wantError),
			},
			{ // Expect an error when using `kind = "Dedicated"` with a Managed Disk.
				Config: testAccKubernetesPersistentVolumeConfig_azure_managedDisk(name, location) +
					testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKind(name, diskURI, "Dedicated"),
				ExpectError: regexp.MustCompile(wantError),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_azure_blobStorageDisk(t *testing.T) {
	var conf1, conf2 api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	// name must not contain dashes, due to the Azure API requirements for storage accounts.
	name := fmt.Sprintf("tfacctest%s", randString)
	location := os.Getenv("TF_VAR_location")
	diskURI := "https://" + name + ".blob.core.windows.net/" + name
	wantError := persistentVolumeAzureBlobError

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNotRunningInAks(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{ // Create a PV using the existing blob storage disk. Kind is omitted to test backwards compatibility.
				Config: testAccKubernetesPersistentVolumeConfig_azure_blobStorage(name, location) +
					testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKindOmitted(name, diskURI),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf1),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.azure_disk.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.azure_disk.0.kind", "Shared"),
				),
			},
			{ // Test that the resource has not been re-created. The object should have the same UID as the initial create.
				Config: testAccKubernetesPersistentVolumeConfig_azure_blobStorage(name, location) +
					testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKind(name, diskURI, "Shared"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.azure_disk.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.azure_disk.0.kind", "Shared"),
					testAccCheckKubernetesPersistentVolumeForceNew(&conf1, &conf2, false),
				),
			},
			{ // Create a new Persistent Volume, using the same Azure storage blob, but using "Dedicated" mode in PV.
				Config: testAccKubernetesPersistentVolumeConfig_azure_blobStorage(name, location) +
					testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKind(name, diskURI, "Dedicated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf2),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.azure_disk.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.azure_disk.0.kind", "Dedicated"),
					testAccCheckKubernetesPersistentVolumeForceNew(&conf1, &conf2, true),
				),
			},
			{ // Expect an error when attempting to use 'kind = Managed' with blob storage.
				Config: testAccKubernetesPersistentVolumeConfig_azure_blobStorage(name, location) +
					testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKind(name, diskURI, "Managed"),
				ExpectError: regexp.MustCompile(wantError),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_googleCloud_basic(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)

	region := os.Getenv("GOOGLE_REGION")
	zone := os.Getenv("GOOGLE_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
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

	zone := os.Getenv("AWS_ZONE")
	region := os.Getenv("AWS_DEFAULT_REGION")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNotRunningInEks(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_aws_basic(name, region, zone),
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
					}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
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

func TestAccKubernetesPersistentVolume_googleCloud_volumeSource(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)
	zone := os.Getenv("GOOGLE_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_googleCloud_volumeSource(name, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume", &conf),
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name"),
				),
			},
			{
				ResourceName:      "kubernetes_persistent_volume",
				ImportState:       true,
				ImportStateVerify: true,
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.persistent_volume_source.0.host_path.0.path", "/second/path"),
				),
			},
			{
				Config:   testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource_volumeMode(name, "/second/path", ""),
				PlanOnly: true,
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
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
		PreCheck:      func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
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
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "1Gi"),
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

func TestAccKubernetesPersistentVolume_hostPath_mountOptions(t *testing.T) {
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
				Config: testAccKubernetesPersistentVolumeConfig_hostPath_mountOptions(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.mount_options.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.mount_options.2356372769", "foo"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_csi_basic(t *testing.T) {
	var conf api.PersistentVolume
	name := acctest.RandomWithPrefix("tf-acc-test")
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_csi_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.capacity.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.driver", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.volume_handle", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.read_only", "false"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.fs_type", ""),
				),
			},
			{
				ResourceName:      "kubernetes_persistent_volume",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesPersistentVolumeConfig_csi_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.capacity.storage", "10Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.driver", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.volume_handle", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.read_only", "true"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.fs_type", "ext4"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_csi_secrets(t *testing.T) {
	var conf api.PersistentVolume
	name := acctest.RandomWithPrefix("tf-acc-test")
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeConfig_csi_secrets(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.capacity.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.driver", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.volume_handle", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.read_only", "false"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.fs_type", ""),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.controller_publish_secret_ref.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.controller_publish_secret_ref.0.namespace", "default"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.node_stage_secret_ref.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.node_stage_secret_ref.0.namespace", "default"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.node_publish_secret_ref.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.node_publish_secret_ref.0.namespace", "default"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.controller_expand_secret_ref.0.name", name),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.persistent_volume_source.0.csi.0.controller_expand_secret_ref.0.namespace", "default"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolume_volumeMode(t *testing.T) {
	var conf api.PersistentVolume
	name := acctest.RandomWithPrefix("tf-acc-test")
	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeVolumeModeConfig(name, "Block"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.annotations.TestAnnotationTwo", "two"),
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"TestAnnotationOne": "one", "TestAnnotationTwo": "two"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.capacity.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume", "spec.0.volume_mode", "Block"),
				),
			},
		},
	})
}

func testAccCheckKubernetesPersistentVolumeForceNew(old, new *api.PersistentVolume, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for persistent volume %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting persistent volume UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}

func waitForPersistentVolumeDeleted(pvName string, poll, timeout time.Duration) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for start := time.Now(); time.Since(start) < timeout; time.Sleep(poll) {
		_, err := conn.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
		if err != nil && apierrs.IsNotFound(err) {
			return nil
		}
	}
	return fmt.Errorf("Persistent Volume still exists: %s", pvName)
}

func testAccCheckKubernetesPersistentVolumeDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	timeout := 5 * time.Second
	poll := 1 * time.Second

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume" {
			continue
		}
		name := rs.Primary.ID
		resp, err := conn.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Name == rs.Primary.ID {
				return waitForPersistentVolumeDeleted(name, poll, timeout)
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

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()
		name := rs.Primary.ID
		out, err := conn.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPersistentVolumeConfig_googleCloud_basic(name, diskName, zone string, region string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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
      storage = "1Gi"
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
            key      = "failure-domain.beta.kubernetes.io/zone"
            operator = "In"
            values   = ["%s"]
          }
          match_expressions {
            key      = "failure-domain.beta.kubernetes.io/region"
            operator = "In"
            values   = ["%s"]
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
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "1Gi"
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

func testAccKubernetesPersistentVolumeConfig_aws_basic(name, region, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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
      storage = "1Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      aws_elastic_block_store {
        volume_id = "${aws_ebs_volume.test.id}"
      }
    }
  }
}

provider "aws" {
  region = "%s"
}

resource "aws_ebs_volume" "test" {
  availability_zone = "%s"
  size              = 1

  tags = {
    Name = %q[1]
  }
}
`, name, region, zone)
}

func testAccKubernetesPersistentVolumeConfig_aws_modified(name, diskName, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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

func testAccKubernetesPersistentVolumeConfig_azure_managedDisk(name, location string) string {
	return fmt.Sprintf(`provider "azurerm" {
  features {}
}
resource "azurerm_resource_group" "test" {
  name     = %[1]q
  location = %[2]q
  tags = {
    environment = "terraform-provider-kubernetes-test"
  }
}
resource "azurerm_managed_disk" "test" {
  name                 = %[1]q
  location             = azurerm_resource_group.test.location
  resource_group_name  = azurerm_resource_group.test.name
  storage_account_type = "Standard_LRS"
  create_option        = "Empty"
  disk_size_gb         = "1"
}
`, name, location)
}

func testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKindOmitted(name, dataDiskURI string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes = ["ReadWriteOnce"]
    persistent_volume_source {
      azure_disk {
        caching_mode = "None"
        data_disk_uri = %[2]q
        disk_name = %[1]q
      }
    }
  }
}`, name, dataDiskURI)
}

func testAccKubernetesPersistentVolumeConfig_azure_PersistentVolumeKind(name, dataDiskURI, kind string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes = ["ReadWriteOnce"]
    persistent_volume_source {
      azure_disk {
        caching_mode = "None"
        data_disk_uri = %[2]q
        disk_name = %[1]q
		kind      = %[3]q
      }
    }
  }
}`, name, dataDiskURI, kind)
}

func testAccKubernetesPersistentVolumeConfig_azure_blobStorage(name, location string) string {
	return fmt.Sprintf(`provider "azurerm" {
  features {}
}
resource "azurerm_resource_group" "test" {
  name     = "%s"
  location = "%s"
  tags = {
    environment = "terraform-provider-kubernetes-test"
  }
}
resource "azurerm_storage_account" "test" {
  name                     = %[1]q
  resource_group_name      = azurerm_resource_group.test.name
  location                 = azurerm_resource_group.test.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  account_kind             = "BlobStorage"
  tags = {
    environment = "terraform-provider-kubernetes-test"
  }
}
resource "azurerm_storage_container" "test" {
  name                  = %[1]q
  storage_account_name  = azurerm_storage_account.test.name
  container_access_type = "private"
}
`, name, location)
}

func testAccKubernetesPersistentVolumeConfig_csi_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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

    name = %[1]q
  }

  spec {
    capacity = {
      storage = "5Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      csi {
        driver        = %[1]q
        volume_handle = %[1]q

        volume_attributes = {
          "foo" = "bar"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesPersistentVolumeConfig_csi_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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

    name = %[1]q
  }

  spec {
    capacity = {
      storage = "10Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      csi {
        driver        = %[1]q
        volume_handle = %[1]q
        read_only     = true
        fs_type       = "ext4"

        volume_attributes = {
          "bar" = "foo"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesPersistentVolumeConfig_csi_secrets(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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

    name = %[1]q
  }

  spec {
    capacity = {
      storage = "5Gi"
    }

    access_modes = ["ReadWriteOnce"]

    persistent_volume_source {
      csi {
        driver        = %[1]q
        volume_handle = %[1]q

        volume_attributes = {
          "foo" = "bar"
        }
        controller_publish_secret_ref {
          name      = %[1]q
          namespace = "default"
        }
        node_stage_secret_ref {
          name      = %[1]q
          namespace = "default"
        }
        node_publish_secret_ref {
          name      = %[1]q
          namespace = "default"
        }
        controller_expand_secret_ref {
          name      = %[1]q
          namespace = "default"
        }
      }
    }
  }
}
`, name)
}

func testAccKubernetesPersistentVolumeVolumeModeConfig(name, mode string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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
      storage = "5Gi"
    }

    access_modes = ["ReadWriteOnce"]
    volume_mode  = "%s"

    persistent_volume_source {
      host_path {
        path = "/first/path"
        type = "DirectoryOrCreate"
      }
    }
  }
}
`, name, mode)
}

func testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource(name, path, typ string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "1Gi"
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

func testAccKubernetesPersistentVolumeConfig_hostPath_volumeSource_volumeMode(name, path, typ string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "1Gi"
    }

    access_modes = ["ReadWriteOnce"]
    volume_mode  = "Filesystem"

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
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "1Gi"
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
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "1Gi"
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
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
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
	nodeAffinity := fmt.Sprintf(`    node_affinity {
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

func testAccKubernetesPersistentVolumeConfig_hostPath_mountOptions(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume" "test" {
  metadata {
    name = "%s"
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes = ["ReadWriteMany"]
    mount_options = ["foo"]
    persistent_volume_source {
      host_path {
        path = "/mnt/local-volume"
      }
    }
  }
}`, name)
}
