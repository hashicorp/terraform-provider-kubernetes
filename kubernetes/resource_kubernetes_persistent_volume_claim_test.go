package kubernetes

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	api "k8s.io/api/core/v1"
	storageapi "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAccKubernetesPersistentVolumeClaim_basic(t *testing.T) {
	var conf api.PersistentVolumeClaim
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t) },
		IDRefreshName: "kubernetes_persistent_volume_claim.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeClaimDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.TestLabelFour", "four"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelThree": "three", "TestLabelFour": "four"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
				),
			},
			{ // GKE specific check
				Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
				SkipFunc: func() (bool, error) {
					isInGke, err := isRunningInGke()
					return !isInGke, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{
						"TestAnnotationOne":                             "one",
						"volume.beta.kubernetes.io/storage-provisioner": "kubernetes.io/gce-pd",
					}),
				),
			},
			{ // minikube specific check
				Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
				SkipFunc: func() (bool, error) {
					isInMinikube, err := isRunningInMinikube()
					return !isInMinikube, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{
						"TestAnnotationOne":                             "one",
						"volume.beta.kubernetes.io/storage-provisioner": "k8s.io/minikube-hostpath",
					}),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &conf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.TestLabelThree", "three"),
					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{"TestLabelOne": "one", "TestLabelTwo": "two", "TestLabelThree": "three"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", name),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
				),
			},
			{ // GKE specific check
				Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
				SkipFunc: func() (bool, error) {
					isInGke, err := isRunningInGke()
					return !isInGke, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{
						"TestAnnotationOne":                             "one",
						"TestAnnotationTwo":                             "two",
						"volume.beta.kubernetes.io/storage-provisioner": "kubernetes.io/gce-pd",
					}),
				),
			},
			{ // minikube specific check
				Config: testAccKubernetesPersistentVolumeClaimConfig_basic(name),
				SkipFunc: func() (bool, error) {
					isInMinikube, err := isRunningInMinikube()
					return !isInMinikube, err
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{
						"TestAnnotationOne":                             "one",
						"TestAnnotationTwo":                             "two",
						"volume.beta.kubernetes.io/storage-provisioner": "k8s.io/minikube-hostpath",
					}),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeClaim_googleCloud_importBasic(t *testing.T) {
	resourceName := "kubernetes_persistent_volume_claim.test"
	volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", acctest.RandString(10))
	zone := os.Getenv("GOOGLE_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKubernetesPersistentVolumeClaimDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_import(volumeName, claimName, diskName, zone),
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

func TestAccKubernetesPersistentVolumeClaim_googleCloud_volumeMatch(t *testing.T) {
	var pvcConf api.PersistentVolumeClaim
	var pvConf api.PersistentVolume

	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	volumeNameModified := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", acctest.RandString(10))
	zone := os.Getenv("GOOGLE_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		IDRefreshName: "kubernetes_persistent_volume_claim.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeClaimDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_volumeMatch(volumeName, claimName, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &pvcConf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&pvcConf.ObjectMeta, map[string]string{"pv.kubernetes.io/bind-completed": "yes"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&pvcConf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.volume_name", volumeName),
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &pvConf),
					testAccCheckMetaAnnotations(&pvConf.ObjectMeta, map[string]string{"pv.kubernetes.io/bound-by-controller": "yes"}),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_volumeMatch_modified(volumeNameModified, claimName, diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &pvcConf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&pvcConf.ObjectMeta, map[string]string{"pv.kubernetes.io/bind-completed": "yes"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&pvcConf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.volume_name", volumeNameModified),
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test2", &pvConf),
					testAccCheckMetaAnnotations(&pvConf.ObjectMeta, map[string]string{"pv.kubernetes.io/bound-by-controller": "yes"}),
				),
			},
		},
	})
}

// Label matching isn't supported on GCE
// TODO: Re-enable when we build test env for K8S that supports it

// func TestAccKubernetesPersistentVolumeClaim_labelsMatch(t *testing.T) {
// 	var conf api.PersistentVolumeClaim
// 	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
// 	volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:      func() { testAccPreCheck(t) },
// 		IDRefreshName: "kubernetes_persistent_volume_claim.test",
// 		Providers:     testAccProviders,
// 		CheckDestroy:  testAccCheckKubernetesPersistentVolumeClaimDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccKubernetesPersistentVolumeClaimConfig_labelsMatch(volumeName, claimName),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &conf),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
// 					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"pv.kubernetes.io/bind-completed": "yes", "pv.kubernetes.io/bound-by-controller": "yes"}),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
// 					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
// 					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
// 					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
// 					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
// 					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_labels.%", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_labels.TfAccTestEnvironment", "blablah"),
// 				),
// 			},
// 		},
// 	})
// }

// func TestAccKubernetesPersistentVolumeClaim_labelsMatchExpression(t *testing.T) {
// 	var conf api.PersistentVolumeClaim
// 	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
// 	volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

// 	resource.Test(t, resource.TestCase{
// 		PreCheck:      func() { testAccPreCheck(t) },
// 		IDRefreshName: "kubernetes_persistent_volume_claim.test",
// 		Providers:     testAccProviders,
// 		CheckDestroy:  testAccCheckKubernetesPersistentVolumeClaimDestroy,
// 		Steps: []resource.TestStep{
// 			{
// 				Config: testAccKubernetesPersistentVolumeClaimConfig_labelsMatchExpression(volumeName, claimName),
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &conf),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
// 					testAccCheckMetaAnnotations(&conf.ObjectMeta, map[string]string{"pv.kubernetes.io/bind-completed": "yes", "pv.kubernetes.io/bound-by-controller": "yes"}),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
// 					testAccCheckMetaLabels(&conf.ObjectMeta, map[string]string{}),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
// 					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
// 					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
// 					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
// 					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.#", "1"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.key", "TfAccTestEnvironment"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.operator", "In"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.values.#", "3"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.values.1187371253", "three"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.values.2053932785", "one"),
// 					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.selector.0.match_expressions.0.values.298486374", "two"),
// 				),
// 			},
// 		},
// 	})
// }

func TestAccKubernetesPersistentVolumeClaim_googleCloud_volumeUpdate(t *testing.T) {
	var pvcConf api.PersistentVolumeClaim
	var pvConf api.PersistentVolume

	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	volumeName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	diskName := fmt.Sprintf("tf-acc-test-disk-%s", acctest.RandString(10))
	zone := os.Getenv("GOOGLE_ZONE")

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		IDRefreshName: "kubernetes_persistent_volume_claim.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeClaimDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_volumeUpdate(volumeName, claimName, "5Gi", diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &pvcConf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&pvcConf.ObjectMeta, map[string]string{"pv.kubernetes.io/bind-completed": "yes"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&pvcConf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.volume_name", volumeName),
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &pvConf),
					testAccCheckMetaAnnotations(&pvConf.ObjectMeta, map[string]string{"pv.kubernetes.io/bound-by-controller": "yes"}),
					testAccCheckClaimRef(&pvConf, &ObjectRefStatic{Namespace: "default", Name: claimName}),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_volumeUpdate(volumeName, claimName, "10Gi", diskName, zone),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &pvcConf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&pvcConf.ObjectMeta, map[string]string{"pv.kubernetes.io/bind-completed": "yes"}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&pvcConf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.volume_name", volumeName),
					testAccCheckKubernetesPersistentVolumeExists("kubernetes_persistent_volume.test", &pvConf),
					testAccCheckMetaAnnotations(&pvConf.ObjectMeta, map[string]string{"pv.kubernetes.io/bound-by-controller": "yes"}),
					testAccCheckClaimRef(&pvConf, &ObjectRefStatic{Namespace: "default", Name: claimName}),
				),
			},
		},
	})
}

func TestAccKubernetesPersistentVolumeClaim_googleCloud_storageClass(t *testing.T) {
	var pvcConf api.PersistentVolumeClaim
	var storageClass storageapi.StorageClass
	var secondStorageClass storageapi.StorageClass

	className := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	claimName := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:      func() { testAccPreCheck(t); skipIfNoGoogleCloudSettingsFound(t) },
		IDRefreshName: "kubernetes_persistent_volume_claim.test",
		Providers:     testAccProviders,
		CheckDestroy:  testAccCheckKubernetesPersistentVolumeClaimDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_storageClass(className, claimName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &pvcConf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&pvcConf.ObjectMeta, map[string]string{
						"pv.kubernetes.io/bind-completed":               "yes",
						"pv.kubernetes.io/bound-by-controller":          "yes",
						"volume.beta.kubernetes.io/storage-provisioner": "kubernetes.io/gce-pd",
					}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&pvcConf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.storage_class_name", className),
					testAccCheckKubernetesStorageClassExists("kubernetes_storage_class.test", &storageClass),
					testAccCheckMetaAnnotations(&storageClass.ObjectMeta, map[string]string{}),
				),
			},
			{
				Config: testAccKubernetesPersistentVolumeClaimConfig_storageClassUpdated(className, claimName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPersistentVolumeClaimExists("kubernetes_persistent_volume_claim.test", &pvcConf),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.annotations.%", "0"),
					testAccCheckMetaAnnotations(&pvcConf.ObjectMeta, map[string]string{
						"pv.kubernetes.io/bind-completed":               "yes",
						"pv.kubernetes.io/bound-by-controller":          "yes",
						"volume.beta.kubernetes.io/storage-provisioner": "kubernetes.io/gce-pd",
					}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.labels.%", "0"),
					testAccCheckMetaLabels(&pvcConf.ObjectMeta, map[string]string{}),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "metadata.0.name", claimName),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.generation"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.self_link"),
					resource.TestCheckResourceAttrSet("kubernetes_persistent_volume_claim.test", "metadata.0.uid"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.access_modes.1245328686", "ReadWriteOnce"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.#", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.%", "1"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.resources.0.requests.storage", "5Gi"),
					resource.TestCheckResourceAttr("kubernetes_persistent_volume_claim.test", "spec.0.storage_class_name", className+"-second"),
					testAccCheckKubernetesStorageClassExists("kubernetes_storage_class.test", &storageClass),
					testAccCheckMetaAnnotations(&storageClass.ObjectMeta, map[string]string{}),
					testAccCheckKubernetesStorageClassExists("kubernetes_storage_class.second", &secondStorageClass),
					testAccCheckMetaAnnotations(&secondStorageClass.ObjectMeta, map[string]string{}),
				),
			},
		},
	})
}

func testAccCheckKubernetesPersistentVolumeClaimDestroy(s *terraform.State) error {
	conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_persistent_volume_claim" {
			continue
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		var resp *api.PersistentVolumeClaim
		err = resource.Retry(3*time.Minute, func() *resource.RetryError {
			resp, err = conn.CoreV1().PersistentVolumeClaims(namespace).Get(name, meta_v1.GetOptions{})
			if errors.IsNotFound(err) {
				return nil
			}
			if err == nil && resp != nil {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		})

		if err != nil {
			if resp.Namespace == namespace && resp.Name == name {
				return fmt.Errorf("Persistent Volume still exists: %s", rs.Primary.ID)
			}
		}
	}

	return nil
}

func testAccCheckKubernetesPersistentVolumeClaimExists(n string, obj *api.PersistentVolumeClaim) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := testAccProvider.Meta().(KubeClientsets).MainClientset()
		if err != nil {
			return err
		}

		namespace, name, err := idParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.CoreV1().PersistentVolumeClaims(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccCheckClaimRef(pv *api.PersistentVolume, expected *ObjectRefStatic) resource.TestCheckFunc {
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

func testAccKubernetesPersistentVolumeClaimConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume_claim" "test" {
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

func testAccKubernetesPersistentVolumeClaimConfig_metaModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume_claim" "test" {
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

func testAccKubernetesPersistentVolumeClaimConfig_import(volumeName, claimName, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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
}

resource "kubernetes_persistent_volume_claim" "test" {
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

    volume_name = "${kubernetes_persistent_volume.test.metadata.0.name}"
  }
}
`, volumeName, diskName, zone, claimName)
}

func testAccKubernetesPersistentVolumeClaimConfig_volumeMatch(volumeName, claimName, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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
}

resource "kubernetes_persistent_volume_claim" "test" {
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

    volume_name = "${kubernetes_persistent_volume.test.metadata.0.name}"
  }
}
`, volumeName, diskName, zone, claimName)
}

func testAccKubernetesPersistentVolumeClaimConfig_volumeMatch_modified(volumeName, claimName, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test2" {
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
}

resource "kubernetes_persistent_volume_claim" "test" {
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

    volume_name = "${kubernetes_persistent_volume.test2.metadata.0.name}"
  }
}
`, volumeName, diskName, zone, claimName)
}

// func testAccKubernetesPersistentVolumeClaimConfig_labelsMatch(volumeName, claimName string) string {
// 	return fmt.Sprintf(`
// resource "kubernetes_persistent_volume" "test" {
// 	metadata {
// 		labels {
// 			TfAccTestEnvironment = "blablah"
// 		}
// 		name = "%s"
// 	}
// 	spec {
// 		capacity {
// 			storage = "10Gi"
// 		}
// 		access_modes = ["ReadWriteOnce"]
// 		persistent_volume_source {
// 			gce_persistent_disk {
// 				pd_name = "test123"
// 			}
// 		}
// 	}
// }

// resource "kubernetes_persistent_volume_claim" "test" {
// 	metadata {
// 		name = "%s"
// 	}
// 	spec {
// 		access_modes = ["ReadWriteOnce"]
// 		resources {
// 			requests {
// 				storage = "5Gi"
// 			}
// 		}
// 		selector {
// 			match_labels {
// 				TfAccTestEnvironment = "blablah"
// 			}
// 		}
// 	}
// }
// `, volumeName, claimName)
// }

// func testAccKubernetesPersistentVolumeClaimConfig_labelsMatchExpression(volumeName, claimName string) string {
// 	return fmt.Sprintf(`
// resource "kubernetes_persistent_volume" "test" {
// 	metadata {
// 		labels {
// 			TfAccTestEnvironment = "two"
// 		}
// 		name = "%s"
// 	}
// 	spec {
// 		capacity {
// 			storage = "10Gi"
// 		}
// 		access_modes = ["ReadWriteOnce"]
// 		persistent_volume_source {
// 			gce_persistent_disk {
// 				pd_name = "test123"
// 			}
// 		}
// 	}
// }

// resource "kubernetes_persistent_volume_claim" "test" {
// 	metadata {
// 		name = "%s"
// 	}
// 	spec {
// 		access_modes = ["ReadWriteOnce"]
// 		resources {
// 			requests {
// 				storage = "5Gi"
// 			}
// 		}
// 		selector {
// 			match_expressions {
// 				key = "TfAccTestEnvironment"
// 				operator = "In"
// 				values = ["one", "three", "two"]
// 			}
// 		}
// 	}
// }
// `, volumeName, claimName)
// }

func testAccKubernetesPersistentVolumeClaimConfig_volumeUpdate(volumeName, claimName, storage, diskName, zone string) string {
	return fmt.Sprintf(`
resource "kubernetes_persistent_volume" "test" {
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
}

resource "kubernetes_persistent_volume_claim" "test" {
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

    volume_name = "${kubernetes_persistent_volume.test.metadata.0.name}"
  }
}
`, volumeName, storage, diskName, zone, claimName)
}

func testAccKubernetesPersistentVolumeClaimConfig_storageClass(className, claimName string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "kubernetes.io/gce-pd"

  parameters = {
    type = "pd-standard"
  }
}

resource "kubernetes_persistent_volume_claim" "test" {
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

    storage_class_name = "${kubernetes_storage_class.test.metadata.0.name}"
  }
}
`, className, claimName)
}

func testAccKubernetesPersistentVolumeClaimConfig_storageClassUpdated(className, claimName string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class" "test" {
  metadata {
    name = "%s"
  }

  storage_provisioner = "kubernetes.io/gce-pd"

  parameters = {
    type = "pd-standard"
  }
}

resource "kubernetes_storage_class" "second" {
  metadata {
    name = "%s-second"
  }

  storage_provisioner = "kubernetes.io/gce-pd"

  parameters = {
    type = "pd-ssd"
  }
}

resource "kubernetes_persistent_volume_claim" "test" {
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

    storage_class_name = "${kubernetes_storage_class.second.metadata.0.name}"
  }
}
`, className, className, claimName)
}
