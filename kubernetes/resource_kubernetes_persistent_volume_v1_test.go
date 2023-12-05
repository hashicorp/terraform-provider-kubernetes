// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	api "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPersistentVolumeV1_minimal(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)

	resourceName := "kubernetes_persistent_volume_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.claim_ref.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_azure_basic(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tfacctest%s", randString)
	location := os.Getenv("TF_VAR_location")
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	diskURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + name + "/providers/Microsoft.Compute/disks/" + name
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInAks(t) },
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_azure_managedDisk(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKind(name, diskURI, "Managed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("azurerm_resource_group.test", "name", name),
					resource.TestCheckResourceAttr("azurerm_managed_disk.test", "name", name),
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_disk.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_disk.0.kind", "Managed"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_azure_ManagedDiskExpectErrors(t *testing.T) {
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tfacctest%s", randString)
	location := os.Getenv("TF_VAR_location")
	subscriptionID := os.Getenv("ARM_SUBSCRIPTION_ID")
	diskURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + name + "/providers/Microsoft.Compute/disks/" + name
	wantError := persistentVolumeAzureManagedError

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInAks(t) },
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{ // Expect an error when using a Managed Disk with `kind` omitted.
				Config: testAccKubernetesPersistentVolumeV1Config_azure_managedDisk(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKindOmitted(name, diskURI),
				ExpectError: regexp.MustCompile(wantError),
			},
			{ // Expect an error when using `kind = "Shared"` with a Managed Disk.
				Config: testAccKubernetesPersistentVolumeV1Config_azure_managedDisk(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKind(name, diskURI, "Shared"),
				ExpectError: regexp.MustCompile(wantError),
			},
			{ // Expect an error when using `kind = "Dedicated"` with a Managed Disk.
				Config: testAccKubernetesPersistentVolumeV1Config_azure_managedDisk(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKind(name, diskURI, "Dedicated"),
				ExpectError: regexp.MustCompile(wantError),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_azure_blobStorageDisk(t *testing.T) {
	var conf1, conf2 api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	// name must not contain dashes, due to the Azure API requirements for storage accounts.
	name := fmt.Sprintf("tfacctest%s", randString)
	location := os.Getenv("TF_VAR_location")
	diskURI := "https://" + name + ".blob.core.windows.net/" + name
	wantError := persistentVolumeAzureBlobError
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInAks(t) },
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{ // Create a PV using the existing blob storage disk. Kind is omitted to test backwards compatibility.
				Config: testAccKubernetesPersistentVolumeV1Config_azure_blobStorage(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKindOmitted(name, diskURI),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_disk.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_disk.0.kind", "Shared"),
				),
			},
			{ // Test that the resource has not been re-created. The object should have the same UID as the initial create.
				Config: testAccKubernetesPersistentVolumeV1Config_azure_blobStorage(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKind(name, diskURI, "Shared"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_disk.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_disk.0.kind", "Shared"),
					testAccCheckKubernetesPersistentVolumeV1ForceNew(&conf1, &conf2, false),
				),
			},
			{ // Create a new Persistent Volume, using the same Azure storage blob, but using "Dedicated" mode in PV.
				Config: testAccKubernetesPersistentVolumeV1Config_azure_blobStorage(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKind(name, diskURI, "Dedicated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_disk.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_disk.0.kind", "Dedicated"),
					testAccCheckKubernetesPersistentVolumeV1ForceNew(&conf1, &conf2, true),
				),
			},
			{ // Expect an error when attempting to use 'kind = Managed' with blob storage.
				Config: testAccKubernetesPersistentVolumeV1Config_azure_blobStorage(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKind(name, diskURI, "Managed"),
				ExpectError: regexp.MustCompile(wantError),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_azure_file(t *testing.T) {
	var conf1, conf2 api.PersistentVolume
	// name must not contain dashes, due to the Azure API requirements for storage accounts.
	name := fmt.Sprintf("tfacctest%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	namespace := fmt.Sprintf("tfacctest%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	secretName := fmt.Sprintf("tfacctest%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	location := os.Getenv("TF_VAR_location")
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInAks(t) },
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{ // Create a PV using the existing Azure storage share (without secret_namespace).
				Config: testAccKubernetesPersistentVolumeV1Config_azure_file(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeAzureFile(name, secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_file.0.share_name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_file.0.secret_name", secretName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_file.0.secret_namespace", ""),
				),
			},
			{ // Create a PV using the existing Azure storage share (with secret_namespace).
				Config: testAccKubernetesPersistentVolumeV1Config_azure_file(name, location) +
					testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeAzureFileNamespace(name, namespace, secretName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_file.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_file.0.share_name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_file.0.secret_name", secretName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.azure_file.0.secret_namespace", namespace),
					testAccCheckKubernetesPersistentVolumeV1ForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_googleCloud_basic(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)
	resourceName := "kubernetes_persistent_volume_v1.test"

	region := os.Getenv("GOOGLE_REGION")
	zone := os.Getenv("GOOGLE_ZONE")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_googleCloud_basic(name, diskName, zone, region),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					//  "TestLabelOne":   "one",
					//  "TestLabelTwo":   "two",
					//  "TestLabelThree": "three",
					//  "failure-domain.beta.kubernetes.io/region": region,
					//  "failure-domain.beta.kubernetes.io/zone":   zone,
					//}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name", diskName),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_googleCloud_modified(name, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					//  "TestLabelOne":   "one",
					//  "TestLabelTwo":   "two",
					//  "TestLabelThree": "three",
					//  "failure-domain.beta.kubernetes.io/region": region,
					//  "failure-domain.beta.kubernetes.io/zone":   zone,
					//}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "42Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.1", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadOnlyMany"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.gce_persistent_disk.0.fs_type", "ntfs"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name", diskName),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.gce_persistent_disk.0.read_only", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_aws_basic(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	resourceName := "kubernetes_persistent_volume_v1.test"

	region := os.Getenv("AWS_DEFAULT_REGION")
	zone := region + "a"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInEks(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_aws_basic(name, region, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					//"TestLabelOne":   "one",
					//"TestLabelTwo":   "two",
					//"TestLabelThree": "three",
					//}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.aws_elastic_block_store.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.volume_id"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_aws_modified(name, region, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					//"TestLabelOne":   "one",
					//"TestLabelTwo":   "two",
					//"TestLabelThree": "three",
					//}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "42Mi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.aws_elastic_block_store.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.volume_id"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.fs_type", "io1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.partition", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.aws_elastic_block_store.0.read_only", "true"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_googleCloud_volumeSource(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)
	zone := os.Getenv("GOOGLE_ZONE")
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_googleCloud_volumeSource(name, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.gce_persistent_disk.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "spec.0.persistent_volume_source.0.gce_persistent_disk.0.pd_name"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_volumeSource(name, "/custom/testing/path", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.host_path.0.path", "/custom/testing/path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.host_path.0.type", ""),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_hostPath_volumeSource(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_volumeSource(name, "/first/path", "DirectoryOrCreate"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.host_path.0.path", "/first/path"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.host_path.0.type", "DirectoryOrCreate"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_volumeSource(name, "/second/path", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.host_path.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.host_path.0.path", "/second/path"),
				),
			},
			{
				Config:   testAccKubernetesPersistentVolumeV1Config_hostPath_volumeSource_volumeMode(name, "/second/path", ""),
				PlanOnly: true,
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_local_volumeSource(t *testing.T) {
	var conf1, conf2 api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_local_volumeSource(name, "1Gi", "/first/path", "test-node"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.local.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.local.0.path", "/first/path"),
				),
			},
			{ // Test updating storage capacity in-place.
				Config: testAccKubernetesPersistentVolumeV1Config_local_volumeSource(name, "2Gi", "/first/path", "test-node"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "2Gi"),
					testAccCheckKubernetesPersistentVolumeV1ForceNew(&conf1, &conf2, false),
				),
			},
			{ // Test updating persistentVolumeSource. This should create a new resource.
				Config: testAccKubernetesPersistentVolumeV1Config_local_volumeSource(name, "2Gi", "/second/path", "test-node"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.local.0.path", "/second/path"),
					testAccCheckKubernetesPersistentVolumeV1ForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_cephFsSecretRef(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_cephFsSecretRef(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "2Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteMany"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.ceph_fs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.0", "10.16.154.78:6789"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.ceph_fs.0.monitors.1", "10.16.154.82:6789"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.ceph_fs.0.secret_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.ceph_fs.0.secret_ref.0.name", "ceph-secret"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_storageClass(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	name := fmt.Sprintf("tf-acc-test-%s", randString)
	storageClassName := fmt.Sprintf("tf-acc-test-sc-%s", randString)
	secondStorageClassName := fmt.Sprintf("tf-acc-test-sc2-%s", randString)
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", randString)
	zone := os.Getenv("GOOGLE_ZONE")
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_storageClass(name, diskName, storageClassName, secondStorageClassName, zone, "test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteMany"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.storage_class_name", storageClassName),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_storageClass(name, diskName, storageClassName, secondStorageClassName, zone, "test2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteMany"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.storage_class_name", secondStorageClassName),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_hostPath_nodeAffinity(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	keyName := "spec.0.node_affinity.0.required.0.node_selector_term.0.match_expressions"
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_nodeAffinity_match(name, "selectorLabelTest", "selectorValueTest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.0.required.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.0.required.0.node_selector_term.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.key", keyName), "selectorLabelTest"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.operator", keyName), "In"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.values.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.values.0", keyName), "selectorValueTest"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_nodeAffinity_match(name, "selectorLabel2", "selectorValue2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.0.required.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.0.required.0.node_selector_term.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.key", keyName), "selectorLabel2"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.operator", keyName), "In"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.values.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.values.0", keyName), "selectorValue2"),
				),
			},
			// Disabled due to SDK bug around Optional+Computed attributes
			// {
			// 	Config: testAccKubernetesPersistentVolumeConfig_hostPath_withoutNodeAffinity(name),
			// 	Check: resource.ComposeAggregateTestCheckFunc(
			// 		testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &conf),
			// 		resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.capacity.storage", "2Gi"),
			// 		resource.TestCheckResourceAttr("kubernetes_persistent_volume.test", "spec.0.node_affinity.#", "0"),
			// 	),
			// },
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_nodeAffinity_match(name, "selectorLabelTest", "selectorValueTest"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.0.required.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.node_affinity.0.required.0.node_selector_term.#", "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.key", keyName), "selectorLabelTest"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.operator", keyName), "In"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.values.#", keyName), "1"),
					resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("%s.0.values.0", keyName), "selectorValueTest"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_hostPath_mountOptions(t *testing.T) {
	var conf api.PersistentVolume
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_mountOptions(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "spec.0.mount_options.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.mount_options.0", "foo"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_csi_basic(t *testing.T) {
	var conf api.PersistentVolume
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_persistent_volume_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_csi_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.driver", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.volume_handle", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.fs_type", ""),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_csi_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "10Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.driver", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.volume_handle", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.read_only", "true"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.fs_type", "ext4"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_csi_secrets(t *testing.T) {
	var conf api.PersistentVolume
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_persistent_volume_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_csi_secrets(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.driver", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.volume_handle", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.read_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.fs_type", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.controller_publish_secret_ref.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.controller_publish_secret_ref.0.namespace", "default"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.node_stage_secret_ref.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.node_stage_secret_ref.0.namespace", "default"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.node_publish_secret_ref.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.node_publish_secret_ref.0.namespace", "default"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.controller_expand_secret_ref.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.persistent_volume_source.0.csi.0.controller_expand_secret_ref.0.namespace", "default"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_volumeMode(t *testing.T) {
	var conf api.PersistentVolume
	name := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "kubernetes_persistent_volume_v1.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1VolumeModeConfig(name, "Block"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.capacity.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_mode", "Block"),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeV1_hostpath_claimRef(t *testing.T) {
	var conf1, conf2 api.PersistentVolume
	var conf3 api.PersistentVolumeClaim
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	name := fmt.Sprintf("tf-acc-test-%s", randString)
	resourceName := "kubernetes_persistent_volume_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_claimRef_noNamespace(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.claim_ref.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.claim_ref.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.claim_ref.0.namespace", "default"),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_claimRef_withNamespace(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.claim_ref.0.namespace", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.claim_ref.0.name", name),
					testAccCheckKubernetesPersistentVolumeV1ForceNew(&conf1, &conf2, false),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeV1Config_hostPath_claimRef_withPVC(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeV1Exists(resourceName, &conf2),
					testAccCheckKubernetesPersistentVolumeClaimV1Exists("kubernetes_persistent_volume_claim_v1.test", &conf3),
					resource.TestCheckResourceAttr(resourceName, "spec.0.claim_ref.0.namespace", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.claim_ref.0.name", name),
					testAccCheckKubernetesPersistentVolumeV1ForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func testAccCheckKubernetesPersistentVolumeV1ForceNew(old, new *api.PersistentVolume, wantNew bool) resource.TestCheckFunc {
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

func testAccCheckKubernetesPersistentVolumeV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()

	if err != nil {
		return err
	}
	ctx := context.TODO()

	timeout := 5 * time.Second
	poll := 1 * time.Second

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume_v1.test" {
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

func testAccCheckKubernetesPersistentVolumeV1Exists(n string, obj *api.PersistentVolume) resource.TestCheckFunc {
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

func testAccKubernetesPersistentVolumeV1Config_googleCloud_basic(name, diskName, zone string, region string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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
            key      = "topology.kubernetes.io/zone"
            operator = "In"
            values   = ["%s"]
          }
          match_expressions {
            key      = "topology.kubernetes.io/region"
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
  labels = {
    "test" = "tf-k8s-acc-test"
  }

  lifecycle {
    ignore_changes = [labels]
  }
}
`, name, zone, region, diskName, zone)
}

func testAccKubernetesPersistentVolumeV1Config_googleCloud_modified(name, diskName, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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
  labels = {
    "test" = "tf-k8s-acc-test"
  }

  lifecycle {
    ignore_changes = [labels]
  }
}
`, name, diskName, zone)
}

func testAccKubernetesPersistentVolumeV1Config_googleCloud_volumeSource(name, diskName, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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
  labels = {
    "test" = "tf-k8s-acc-test"
  }

  lifecycle {
    ignore_changes = [labels]
  }
}
`, name, diskName, zone)
}

func testAccKubernetesPersistentVolumeV1Config_aws_basic(name, region, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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
    Name = %[1]q
  }
}
`, name, region, zone)
}

func testAccKubernetesPersistentVolumeV1Config_aws_modified(name, region, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

provider "aws" {
  region = "%s"
}

resource "aws_ebs_volume" "test" {
  availability_zone = "%s"
  size              = 10

  tags = {
    Name = %[1]q
  }
}
`, name, region, zone)
}

func testAccKubernetesPersistentVolumeV1Config_azure_managedDisk(name, location string) string {
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

func testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKindOmitted(name, dataDiskURI string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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
        caching_mode  = "None"
        data_disk_uri = %[2]q
        disk_name     = %[1]q
      }
    }
  }
}`, name, dataDiskURI)
}

func testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeKind(name, dataDiskURI, kind string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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
        caching_mode  = "None"
        data_disk_uri = %[2]q
        disk_name     = %[1]q
        kind          = %[3]q
      }
    }
  }
}`, name, dataDiskURI, kind)
}

func testAccKubernetesPersistentVolumeV1Config_azure_blobStorage(name, location string) string {
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

func testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeAzureFile(name, secretName string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes = ["ReadWriteOnce"]
    persistent_volume_source {
      azure_file {
        secret_name = %[2]q
        share_name  = %[1]q
        read_only   = false
      }
    }
  }
}`, name, secretName)
}

func testAccKubernetesPersistentVolumeV1Config_azure_PersistentVolumeAzureFileNamespace(name, namespace, secretName string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[2]q
  }
}

resource "kubernetes_secret_v1" "test" {
  metadata {
    name      = %[3]q
    namespace = %[2]q
  }

  data = {
    azurestorageaccountname = azurerm_storage_account.test.name
    azurestorageaccountkey  = azurerm_storage_account.test.primary_access_key
  }
}

resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes = ["ReadWriteOnce"]
    persistent_volume_source {
      azure_file {
        secret_name      = %[3]q
        secret_namespace = %[2]q
        share_name       = %[1]q
        read_only        = false
      }
    }
  }
}`, name, namespace, secretName)
}

func testAccKubernetesPersistentVolumeV1Config_azure_file(name, location string) string {
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
  account_kind             = "StorageV2"
  # needed for Azure File kubernetes cifs mount
  enable_https_traffic_only = false
  tags = {
    environment = "terraform-provider-kubernetes-test"
  }
}

resource "azurerm_storage_share" "test" {
  name                 = %[1]q
  storage_account_name = azurerm_storage_account.test.name
  quota                = 1
}
`, name, location)
}

func testAccKubernetesPersistentVolumeV1Config_csi_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

func testAccKubernetesPersistentVolumeV1Config_csi_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

func testAccKubernetesPersistentVolumeV1Config_csi_secrets(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

func testAccKubernetesPersistentVolumeV1VolumeModeConfig(name, mode string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

func testAccKubernetesPersistentVolumeV1Config_hostPath_volumeSource(name, path, typ string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

func testAccKubernetesPersistentVolumeV1Config_hostPath_volumeSource_volumeMode(name, path, typ string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

func testAccKubernetesPersistentVolumeV1Config_local_volumeSource(name, storage, path, hostname string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "%s"
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
`, name, storage, path, hostname)
}

func testAccKubernetesPersistentVolumeV1Config_cephFsSecretRef(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

func testAccKubernetesPersistentVolumeV1Config_storageClass(name, diskName, storageClassName, storageClassName2, zone, refName string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

    storage_class_name = "${kubernetes_storage_class_v1.%s.metadata.0.name}"
  }
}

resource "google_compute_disk" "test" {
  name  = "%s"
  type  = "pd-ssd"
  zone  = "%s"
  image = "debian-8-jessie-v20170523"
  size  = 12
  labels = {
    "test" = "tf-k8s-acc-test"
  }

  lifecycle {
    ignore_changes = [labels]
  }
}

resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "kubernetes.io/gce-pd"

  parameters = {
    type = "pd-ssd"
  }
}

resource "kubernetes_storage_class_v1" "test2" {
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

func testAccKubernetesPersistentVolumeV1Config_hostPath_nodeAffinity(name, nodeAffinity string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
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

func testAccKubernetesPersistentVolumeV1Config_hostPath_nodeAffinity_match(name, selectorLabel, selectorValue string) string {
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
	return testAccKubernetesPersistentVolumeV1Config_hostPath_nodeAffinity(name, nodeAffinity)
}

//func testAccKubernetesPersistentVolumeV1Config_hostPath_withoutNodeAffinity(name string) string {
//	return testAccKubernetesPersistentVolumeV1Config_hostPath_nodeAffinity(name, ``)
//}

func testAccKubernetesPersistentVolumeV1Config_hostPath_mountOptions(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes  = ["ReadWriteMany"]
    mount_options = ["foo"]
    persistent_volume_source {
      host_path {
        path = "/mnt/local-volume"
      }
    }
  }
}`, name)
}

func testAccKubernetesPersistentVolumeV1Config_hostPath_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes  = ["ReadWriteMany"]
    mount_options = ["foo"]

    persistent_volume_source {
      host_path {
        path = "/mnt/local-volume"
      }
    }
  }
}`, name)
}

func testAccKubernetesPersistentVolumeV1Config_hostPath_claimRef_noNamespace(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "%s"
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes  = ["ReadWriteMany"]
    mount_options = ["foo"]
    claim_ref {
      name = "%s"
    }

    persistent_volume_source {
      host_path {
        path = "/mnt/local-volume"
      }
    }
  }
}`, name, name)
}

func testAccKubernetesPersistentVolumeV1Config_hostPath_claimRef_withNamespace(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}
resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes  = ["ReadWriteMany"]
    mount_options = ["foo"]
    claim_ref {
      name      = %[1]q
      namespace = kubernetes_namespace_v1.test.metadata.0.name
    }

    persistent_volume_source {
      host_path {
        path = "/mnt/local-volume"
      }
    }
  }
}`, name)
}

func testAccKubernetesPersistentVolumeV1Config_hostPath_claimRef_withPVC(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = %[1]q
  }
  spec {
    capacity = {
      storage = "1Gi"
    }
    access_modes  = ["ReadWriteMany"]
    mount_options = ["foo"]
    claim_ref {
      name      = %[1]q
      namespace = kubernetes_namespace_v1.test.metadata.0.name
    }

    persistent_volume_source {
      host_path {
        path = "/mnt/local-volume"
      }
    }
  }
}

resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "1Gi"
      }
    }

    volume_name = kubernetes_persistent_volume_v1.test.metadata.0.name
  }
}
`, name)
}
