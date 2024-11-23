// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	corev1 "k8s.io/api/core/v1"
	storageapi "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPersistentVolumeClaimV1_basic(t *testing.T) {
	var conf corev1.PersistentVolumeClaim
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_persistent_volume_claim_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeClaimV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "5Gi"),
				),
			},
			//      { // GKE specific check
			//        Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
			//        SkipFunc: func() (bool, error) {
			//          isInGke, err := isRunningInGke()
			//          return !isInGke, err
			//        },
			//        Check: resource.ComposeAggregateTestCheckFunc(
			//          //
			//         (&conf.ObjectMeta, map[string]string{
			//          //  "TestAnnotationOne":                             "one",
			//          //  "volume.beta.kubernetes.io/storage-provisioner": "kubernetes.io/gce-pd",
			//          //}),
			//        ),
			//      },
			//      { // minikube specific check
			//        Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
			//        SkipFunc: func() (bool, error) {
			//          isInMinikube, err := isRunningInMinikube()
			//          return !isInMinikube, err
			//        },
			//        Check: resource.ComposeAggregateTestCheckFunc(
			//            "TestAnnotationOne":                             "one",
			//            "volume.beta.kubernetes.io/storage-provisioner": "k8s.io/minikube-hostpath",
			//          }),
			//        ),
			//      },
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "5Gi"),
				),
			},
			//      { // GKE specific check
			//        Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
			//        SkipFunc: func() (bool, error) {
			//          isInGke, err := isRunningInGke()
			//          return !isInGke, err
			//        },
			//        Check: resource.ComposeAggregateTestCheckFunc(
			//            "TestAnnotationOne":                             "one",
			//            "TestAnnotationTwo":                             "two",
			//            "volume.beta.kubernetes.io/storage-provisioner": "kubernetes.io/gce-pd",
			//          }),
			//        ),
			//      },
			//      { // minikube specific check
			//        Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
			//        SkipFunc: func() (bool, error) {
			//          isInMinikube, err := isRunningInMinikube()
			//          return !isInMinikube, err
			//        },
			//        Check: resource.ComposeAggregateTestCheckFunc(
			//            "TestAnnotationOne":                             "one",
			//            "TestAnnotationTwo":                             "two",
			//            "volume.beta.kubernetes.io/storage-provisioner": "k8s.io/minikube-hostpath",
			//          }),
			//        ),
			//      },
		},
	})
}

func TestAccKubernetesPersistentVolumeClaimV1_googleCloud_volumeMatch(t *testing.T) {
	var pvcConf corev1.PersistentVolumeClaim
	var pvConf corev1.PersistentVolume
	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	volumeNameModified := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", acctest.RandString(10))
	zone := os.Getenv("GOOGLE_ZONE")
	resourceName := "kubernetes_persistent_volume_claim_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeClaimV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_volumeMatch(volumeName, claimName, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &pvcConf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_name", volumeName),
					testAccCheckKubernetesPersistentVolumeV1Exists("kubernetes_persistent_volume_v1.test", &pvConf),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_volumeMatch_modified(volumeNameModified, claimName, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &pvcConf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_name", volumeNameModified),
					testAccCheckKubernetesPersistentVolumeV1Exists("kubernetes_persistent_volume_v1.test2", &pvConf),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeClaimV1_withDataSource(t *testing.T) {
	var sourcePVC, clonedPVC corev1.PersistentVolumeClaim
	sourcePVCName := fmt.Sprintf("tf-acc-test-source-%s", acctest.RandString(10))
	clonedPVCName := fmt.Sprintf("tf-acc-test-clone-%s", acctest.RandString(10))
	resourceName := "kubernetes_persistent_volume_claim_v1.cloned"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeClaimV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_withDataSource(sourcePVCName, clonedPVCName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Checking if the PVC exists
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(
						"kubernetes_persistent_volume_claim_v1.source", &sourcePVC),
					// Check if the cloned PVC exists and references the source PVC
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &clonedPVC),
					// Validate that the data source field is set correctly
					resource.TestCheckResourceAttr(resourceName, "spec.0.data_source.0.persistent_volume_claim.claim_name", sourcePVCName),
				),
			},
		},
	})
}

func testAccKubernetesPersistentVolumeClaimV1Config_withDataSource(sourcePVCName, clonedPVCName string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume_claim_v1" "source" {
  metadata {
    name = "%s"
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "1Gi"
      }
    }
  }
}

resource "kubernetes_persistent_volume_claim_v1" "cloned" {
  metadata {
    name = "%s"
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "1Gi"
      }
    }

    data_source {
      persistent_volume_claim {
        claim_name = kubernetes_persistent_volume_claim_v1.source.metadata.0.name
      }
    }
  }
}
`, sourcePVCName, clonedPVCName)
}

// Label matching isn't supported on GCE
// TODO: Re-enable when we build test env for K8S that supports it

// func TestAccKubernetesPersistentVolumeClaim_labelsMatch(t *testing.T) {
//   var conf corev1.PersistentVolumeClaim
//   claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
//   volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

//   resource.Test(t, resource.TestCase{
//     PreCheck:      func() { testAccPreCheck(t) },
//     IDRefreshName: "kubernetes_persistent_volume_claim.test",
//     ProviderFactories: testAccProviderFactories,
//     CheckDestroy:  testAccCheckKubernetesPersistentVolumeClaimDestroy,
//     Steps: []resource.TestStep{
//       {
//         Config: testAccKubernetesPersistentVolumeClaimConfig_labelsMatch(volumeName, claimName),
//         Check: resource.ComposeAggregateTestCheckFunc(
//           testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &conf),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
//           resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
//           resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
//           resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.0", "ReadWriteOnce"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.#", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_labels.%", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_labels.TfAccTestEnvironment", "blablah"),
//         ),
//       },
//     },
//   })
// }

// func TestAccKubernetesPersistentVolumeClaim_labelsMatchExpression(t *testing.T) {
//   var conf corev1.PersistentVolumeClaim
//   claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
//   volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

//   resource.Test(t, resource.TestCase{
//     PreCheck:      func() { testAccPreCheck(t) },
//     IDRefreshName: "kubernetes_persistent_volume_claim.test",
//     ProviderFactories: testAccProviderFactories,
//     CheckDestroy:  testAccCheckKubernetesPersistentVolumeClaimDestroy,
//     Steps: []resource.TestStep{
//       {
//         Config: testAccKubernetesPersistentVolumeClaimConfig_labelsMatchExpression(volumeName, claimName),
//         Check: resource.ComposeAggregateTestCheckFunc(
//           testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &conf),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
//           resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
//           resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
//           resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.0", "ReadWriteOnce"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.#", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.#", "1"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.key", "TfAccTestEnvironment"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.operator", "In"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.values.#", "3"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.values.0", "three"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.values.0", "one"),
//           resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.values.0", "two"),
//         ),
//       },
//     },
//   })
// }

func TestAccKubernetesPersistentVolumeClaimV1_googleCloud_volumeUpdate(t *testing.T) {
	var pvcConf corev1.PersistentVolumeClaim
	var pvConf corev1.PersistentVolume

	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", acctest.RandString(10))
	zone := os.Getenv("GOOGLE_ZONE")
	resourceName := "kubernetes_persistent_volume_claim_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeClaimV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_volumeUpdate(volumeName, claimName, "5Gi", diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &pvcConf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_name", volumeName),
					testAccCheckKubernetesPersistentVolumeV1Exists("kubernetes_persistent_volume_v1.test", &pvConf),
					testAccCheckClaimRef(&pvConf, &ObjectRefStatic{Namespace: "default", Name: claimName}),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_volumeUpdate(volumeName, claimName, "10Gi", diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &pvcConf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_name", volumeName),
					testAccCheckKubernetesPersistentVolumeV1Exists("kubernetes_persistent_volume_v1.test", &pvConf),
					testAccCheckClaimRef(&pvConf, &ObjectRefStatic{Namespace: "default", Name: claimName}),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeClaimV1_googleCloud_storageClass(t *testing.T) {
	var pvcConf corev1.PersistentVolumeClaim
	var storageClass storageapi.StorageClass
	var secondStorageClass storageapi.StorageClass

	className := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_persistent_volume_claim_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeClaimV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_storageClass(className, claimName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &pvcConf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					//  "pv.kubernetes.io/bind-completed":               "yes",
					//  "pv.kubernetes.io/bound-by-controller":          "yes",
					//  "volume.beta.kubernetes.io/storage-provisioner": "kubernetes.io/gce-pd",
					//}),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.storage_class_name", className),
					testAccCheckKubernetesStorageClassV1Exists("kubernetes_storage_class_v1.test", &storageClass),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_storageClassUpdated(className, claimName),
				Check: resource.ComposeAggregateTestCheckFunc(
					//          testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &pvcConf),
					//          resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
					//            "pv.kubernetes.io/bind-completed":               "yes",
					//            "pv.kubernetes.io/bound-by-controller":          "yes",
					//            "volume.beta.kubernetes.io/storage-provisioner": "kubernetes.io/gce-pd",
					//          }),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.access_modes.0", "ReadWriteOnce"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.storage_class_name", className+"-second"),
					testAccCheckKubernetesStorageClassV1Exists("kubernetes_storage_class_v1.test", &storageClass),
					testAccCheckKubernetesStorageClassV1Exists("kubernetes_storage_class_v1.second", &secondStorageClass),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeClaimV1_expansionGoogleCloud(t *testing.T) {
	var conf1, conf2 corev1.PersistentVolumeClaim
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	imageName := busyboxImage
	resourceName := "kubernetes_persistent_volume_claim_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInGke(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		ExternalProviders: testAccExternalProviders,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeClaimV1Destroy,
		Steps: []resource.TestStep{
			{ // GKE specific check -- initial create.
				Config: testAccKubernetesPersistentVolumeClaimV1Config_updateStorageGKE(name, "1Gi", imageName),
				SkipFunc: func() (bool, error) {
					isInGke, err := isRunningInGke()
					return !isInGke, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "1Gi"),
				),
			},
			{ // GKE specific check -- Update -- storage is increased in place.
				Config: testAccKubernetesPersistentVolumeClaimV1Config_updateStorageGKE(name, "2Gi", imageName),
				SkipFunc: func() (bool, error) {
					isInGke, err := isRunningInGke()
					return !isInGke, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "2Gi"),
					testAccCheckKubernetesPersistentVolumeClaimV1ForceNew(&conf1, &conf2, false),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeClaimV1_expansionMinikube(t *testing.T) {
	var conf1, conf2 corev1.PersistentVolumeClaim
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_persistent_volume_claim_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); skipIfNotRunningInMinikube(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeClaimV1Destroy,
		Steps: []resource.TestStep{
			{ // Minikube specific check -- initial create.
				Config: testAccKubernetesPersistentVolumeClaimV1Config_updateStorageMinikube(name, "1Gi", "5Gi"),
				SkipFunc: func() (bool, error) {
					isInMinikube, err := isRunningInMinikube()
					return !isInMinikube, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf1),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "1Gi"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.limits.storage", "5Gi"),
				),
			},
			{ // Minikube specific check -- Update -- PVC is updated in-place when `resources.requests` is increased.
				Config: testAccKubernetesPersistentVolumeClaimV1Config_updateStorageMinikube(name, "2Gi", "5Gi"),
				SkipFunc: func() (bool, error) {
					isInMinikube, err := isRunningInMinikube()
					return !isInMinikube, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "2Gi"),
					testAccCheckKubernetesPersistentVolumeClaimV1ForceNew(&conf1, &conf2, false),
				),
			},
			{ // Minikube specific check -- PVC is recreated when when `resources.limits` is increased.
				Config: testAccKubernetesPersistentVolumeClaimV1Config_updateStorageMinikube(name, "2Gi", "6Gi"),
				SkipFunc: func() (bool, error) {
					isInMinikube, err := isRunningInMinikube()
					return !isInMinikube, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.limits.storage", "6Gi"),
					testAccCheckKubernetesPersistentVolumeClaimV1ForceNew(&conf1, &conf2, true),
				),
			},
			{ // Minikube specific check -- PVC is recreated when `resources.requests` is decreased.
				Config: testAccKubernetesPersistentVolumeClaimV1Config_updateStorageMinikube(name, "1Gi", "6Gi"),
				SkipFunc: func() (bool, error) {
					isInMinikube, err := isRunningInMinikube()
					return !isInMinikube, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf2),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "spec.0.resources.0.requests.storage", "1Gi"),
					testAccCheckKubernetesPersistentVolumeClaimV1ForceNew(&conf1, &conf2, true),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeClaimV1_volumeMode(t *testing.T) {
	var conf corev1.PersistentVolumeClaim
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_persistent_volume_claim_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		IDRefreshIgnore:   []string{"metadata.0.resource_version"},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKubernetesPersistentVolumeClaimV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_volumeModeDefault(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_mode", string(corev1.PersistentVolumeFilesystem)),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_volumeMode(name, string(corev1.PersistentVolumeFilesystem)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_mode", string(corev1.PersistentVolumeFilesystem)),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimV1Config_volumeMode(name, string(corev1.PersistentVolumeBlock)),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimV1Exists(resourceName, &conf),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.volume_mode", string(corev1.PersistentVolumeBlock)),
				),
			},
		},
	})
}

func testAccCheckKubernetesPersistentVolumeClaimV1Destroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume_claim_v1" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		var resp *corev1.PersistentVolumeClaim
		err = retry.RetryContext(ctx, 3*time.Minute, func() *retry.RetryError {
			resp, err = conn.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
			if errors.IsNotFound(err) {
				return nil
			}
			if err == nil && resp != nil {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		})

		if err != nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Persistent Volume still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPersistentVolumeClaimV1Exists(n string, obj *corev1.PersistentVolumeClaim) resource.TestCheckFunc {
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
		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckKubernetesPersistentVolumeClaimV1IsDestroyed(obj *corev1.PersistentVolumeClaim) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		meta := obj.GetObjectMeta()
		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()
		out, err := conn.CoreV1().PersistentVolumeClaims(meta.GetNamespace()).Get(ctx, meta.GetName(), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}

		return fmt.Errorf("Expected no PVC but still found %q", out.GetObjectMeta().GetName())
	}
}

func testAccCheckClaimRef(pv *corev1.PersistentVolume, expected *ObjectRefStatic) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		or := pv.Spec.ClaimRef
		if or == nil {
			return fmt.Errorf("Expected ClaimRef to be not-nil, specifically %#v", *expected)
		}
		if or.Namespace != expected.Namespace {
			return fmt.Errorf("Expected object reference %q, given: %q", expected.Namespace, or.Namespace)
		}
		if or.Name != expected.Name {
			return fmt.Errorf("Expected object reference %q, given: %q", expected.Name, or.Name)
		}
		return nil
	}
}

type ObjectRefStatic struct {
	Namespace string
	Name      string
}

func testAccKubernetesPersistentVolumeClaimV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name = "%s"
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    selector {
      match_expressions {
        key      = "environment"
        operator = "In"
        values   = ["non-exists-12345"]
      }
    }
  }

  wait_until_bound = false
}
`, name)
}

func testAccKubernetesPersistentVolumeClaimV1Config_updateStorageMinikube(name, requests, limits string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "allow-expansion"
  }
  reclaim_policy      = "Delete"
  storage_provisioner = "k8s.io/minikube-hostpath"
}
resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "test"
  }
  spec {
    capacity = {
      storage = "5Gi"
    }
    access_modes                     = ["ReadWriteOnce"]
    storage_class_name               = kubernetes_storage_class_v1.test.metadata.0.name
    persistent_volume_reclaim_policy = "Recycle"
    persistent_volume_source {
      host_path {
        path = "/tmp/minikubetest"
        type = "DirectoryOrCreate"
      }
    }
  }
}
resource "kubernetes_persistent_volume_claim_v1" "test" {
  wait_until_bound = true
  metadata {
    name = "%s"
  }
  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = kubernetes_storage_class_v1.test.metadata.0.name
    volume_name        = kubernetes_persistent_volume_v1.test.metadata.0.name
    resources {
      requests = {
        storage = "%s"
      }
      limits = {
        storage = "%s"
      }
    }
  }
}
`, name, requests, limits)
}

func testAccKubernetesPersistentVolumeClaimV1Config_updateStorageGKE(name, requests, imageName string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "test"
  }
  allow_volume_expansion = true
  storage_provisioner    = "pd.csi.storage.gke.io"
}
resource "kubernetes_persistent_volume_claim_v1" "test" {
  wait_until_bound = true
  metadata {
    name = "%s"
  }
  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = kubernetes_storage_class_v1.test.metadata.0.name
    resources {
      requests = {
        storage = "%s"
      }
    }
  }
}
resource "kubernetes_pod_v1" "main" {
  metadata {
    name = "test"
  }
  spec {
    container {
      name    = "default"
      image   = "%s"
      command = ["sleep", "3600s"]
      volume_mount {
        mount_path = "/etc/test"
        name       = "pvc"
      }
    }
    volume {
      name = "pvc"
      persistent_volume_claim {
        claim_name = kubernetes_persistent_volume_claim_v1.test.metadata.0.name
      }
    }
  }
}
`, name, requests, imageName)
}

func testAccKubernetesPersistentVolumeClaimV1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_claim_v1" "test" {
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
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    selector {
      match_expressions {
        key      = "environment"
        operator = "In"
        values   = ["non-exists-12345"]
      }
    }
  }

  wait_until_bound = false
}
`, name)
}

func testAccKubernetesPersistentVolumeClaimV1Config_volumeMatch(volumeName, claimName, diskName, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "10Gi"
    }

    access_modes       = ["ReadWriteOnce"]
    storage_class_name = "standard"

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
  size  = 10
  labels = {
    "test" = "tf-k8s-acc-test"
  }

  lifecycle {
    ignore_changes = [labels]
  }
}

resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = "standard"

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    volume_name = "${kubernetes_persistent_volume_v1.test.metadata.0.name}"
  }
}
`, volumeName, diskName, zone, claimName)
}

func testAccKubernetesPersistentVolumeClaimV1Config_volumeMatch_modified(volumeName, claimName, diskName, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test2" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "10Gi"
    }

    access_modes       = ["ReadWriteOnce"]
    storage_class_name = "standard"

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
  size  = 10
  labels = {
    "test" = "tf-k8s-acc-test"
  }

  lifecycle {
    ignore_changes = [labels]
  }
}

resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = "standard"

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    volume_name = "${kubernetes_persistent_volume_v1.test2.metadata.0.name}"
  }
}
`, volumeName, diskName, zone, claimName)
}

// func testAccKubernetesPersistentVolumeClaimConfig_labelsMatch(volumeName, claimName string) string {
//   return fmt.Sprintf(`// resource "kubernetes_persistent_volume" "test" {
//   metadata {
//     labels {
//       TfAccTestEnvironment = "blablah"
//     }
//     name = "%s"
//   }
//   spec {
//     capacity {
//       storage = "10Gi"
//     }
//     access_modes = ["ReadWriteOnce"]
//     persistent_volume_source {
//       gce_persistent_disk {
//         pd_name = "test123"
//       }
//     }
//   }
// }

// resource "kubernetes_persistent_volume_claim" "test" {
//   metadata {
//     name = "%s"
//   }
//   spec {
//     access_modes = ["ReadWriteOnce"]
//     resources {
//       requests {
//         storage = "5Gi"
//       }
//     }
//     selector {
//       match_labels {
//         TfAccTestEnvironment = "blablah"
//       }
//     }
//   }
// }
// `, volumeName, claimName)
// }

// func testAccKubernetesPersistentVolumeClaimConfig_labelsMatchExpression(volumeName, claimName string) string {
//   return fmt.Sprintf(`// resource "kubernetes_persistent_volume" "test" {
//   metadata {
//     labels {
//       TfAccTestEnvironment = "two"
//     }
//     name = "%s"
//   }
//   spec {
//     capacity {
//       storage = "10Gi"
//     }
//     access_modes = ["ReadWriteOnce"]
//     persistent_volume_source {
//       gce_persistent_disk {
//         pd_name = "test123"
//       }
//     }
//   }
// }

// resource "kubernetes_persistent_volume_claim" "test" {
//   metadata {
//     name = "%s"
//   }
//   spec {
//     access_modes = ["ReadWriteOnce"]
//     resources {
//       requests {
//         storage = "5Gi"
//       }
//     }
//     selector {
//       match_expressions {
//         key = "TfAccTestEnvironment"
//         operator = "In"
//         values = ["one", "three", "two"]
//       }
//     }
//   }
// }
// `, volumeName, claimName)
// }

func testAccKubernetesPersistentVolumeClaimV1Config_volumeUpdate(volumeName, claimName, storage, diskName, zone string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    capacity = {
      storage = "%s"
    }

    access_modes       = ["ReadWriteOnce"]
    storage_class_name = "standard"

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
  size  = 10
  labels = {
    "test" = "tf-k8s-acc-test"
  }

  lifecycle {
    ignore_changes = [labels]
  }
}

resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    access_modes       = ["ReadWriteOnce"]
    storage_class_name = "standard"

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    volume_name = "${kubernetes_persistent_volume_v1.test.metadata.0.name}"
  }
}
`, volumeName, storage, diskName, zone, claimName)
}

func testAccKubernetesPersistentVolumeClaimV1Config_storageClass(className, claimName string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "pd.csi.storage.gke.io"

  parameters = {
    type = "pd-standard"
  }
}

resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    storage_class_name = "${kubernetes_storage_class_v1.test.metadata.0.name}"
  }
}
`, className, claimName)
}

func testAccKubernetesPersistentVolumeClaimV1Config_storageClassUpdated(className, claimName string) string {
	return fmt.Sprintf(`resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "pd.csi.storage.gke.io"

  parameters = {
    type = "pd-standard"
  }
}

resource "kubernetes_storage_class_v1" "second" {
  metadata {
    name = "%s-second"
  }

  storage_provisioner = "pd.csi.storage.gke.io"

  parameters = {
    type = "pd-ssd"
  }
}

resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = "%s"
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    storage_class_name = "${kubernetes_storage_class_v1.second.metadata.0.name}"
  }
}
`, className, className, claimName)
}

func testAccKubernetesPersistentVolumeClaimV1Config_volumeModeDefault(name string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = %q
  }

  spec {
    access_modes = ["ReadWriteOnce"]
    resources {
      requests = {
        storage = "1Gi"
      }
    }
  }

  wait_until_bound = false
}
`, name)
}

func testAccKubernetesPersistentVolumeClaimV1Config_volumeMode(name, volumeMode string) string {
	return fmt.Sprintf(`resource "kubernetes_persistent_volume_claim_v1" "test" {
  metadata {
    name = %q
  }

  spec {
    access_modes = ["ReadWriteOnce"]
    volume_mode  = %q
    resources {
      requests = {
        storage = "1Gi"
      }
    }
  }

  wait_until_bound = false
}
`, name, volumeMode)
}

func testAccCheckKubernetesPersistentVolumeClaimV1ForceNew(old, new *corev1.PersistentVolumeClaim, wantNew bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if wantNew {
			if old.ObjectMeta.UID == new.ObjectMeta.UID {
				return fmt.Errorf("Expecting new resource for persistent volume claim %s", old.ObjectMeta.UID)
			}
		} else {
			if old.ObjectMeta.UID != new.ObjectMeta.UID {
				return fmt.Errorf("Expecting persistent volume claim UIDs to be the same: expected %s got %s", old.ObjectMeta.UID, new.ObjectMeta.UID)
			}
		}
		return nil
	}
}
