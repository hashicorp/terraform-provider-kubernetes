// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	storagev1api "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	csiDriverResourceName = "kubernetes_csi_driver_v1.test"
	csiDriverNamePrefix   = "tf-acc-test-"
)

// The minimal configuration: only what is required, plus volume_lifecycle_modes, without
// which SDKv2 never settles (see TestAccCSIDriverV1_UpgradeFromSDKV2_apiDefaults).
func TestAccCSIDriverV1_basic(t *testing.T) {
	var driver storagev1api.CSIDriver
	name := acctest.RandomWithPrefix(csiDriverNamePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSIDriverV1Config_basic(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &driver),
					resource.TestCheckResourceAttr(csiDriverResourceName, "id", name),
					resource.TestCheckResourceAttr(csiDriverResourceName, "metadata.0.name", name),
					resource.TestCheckNoResourceAttr(csiDriverResourceName, "metadata.0.generate_name"),
					resource.TestCheckResourceAttrSet(csiDriverResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(csiDriverResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.attach_required", "true"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.pod_info_on_mount", "false"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.volume_lifecycle_modes.#", "1"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.volume_lifecycle_modes.0", "Persistent"),
				),
			},
			testAccCSIDriverV1ImportStep(),
		},
	})
}

// Every argument set, then labels, annotations and pod_info_on_mount changed in place.
// Covers what the SDKv2 test asserted (pod_info_on_mount and an Ephemeral mode).
func TestAccCSIDriverV1_update(t *testing.T) {
	var created, updated storagev1api.CSIDriver
	name := acctest.RandomWithPrefix(csiDriverNamePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if !testAccClusterVersionAtLeast(t, "1.29.0") {
				t.Skip("spec.podInfoOnMount is immutable before Kubernetes 1.29")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSIDriverV1Config_completeName(name, "one", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &created),
					resource.TestCheckResourceAttr(csiDriverResourceName, "metadata.0.labels.app", "one"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "metadata.0.annotations.example.com/owner", "one"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.pod_info_on_mount", "false"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.volume_lifecycle_modes.#", "2"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.volume_lifecycle_modes.0", "Persistent"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.volume_lifecycle_modes.1", "Ephemeral"),
				),
			},
			testAccCSIDriverV1ImportStep(),
			{
				Config: testAccCSIDriverV1Config_completeName(name, "two", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(csiDriverResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &updated),
					testAccCheckCSIDriverV1Recreated(&created, &updated, false),
					resource.TestCheckResourceAttr(csiDriverResourceName, "metadata.0.labels.app", "two"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "metadata.0.annotations.example.com/owner", "two"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.pod_info_on_mount", "true"),
				),
			},
			testAccCSIDriverV1ImportStep(),
		},
	})
}

// The name is server-assigned, so metadata.name is unknown until apply. A later label
// change must update in place: if name were left unknown, RequiresReplace would fire.
func TestAccCSIDriverV1_generatedName(t *testing.T) {
	var created, updated storagev1api.CSIDriver

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSIDriverV1Config_completeGenerateName(csiDriverNamePrefix, "one", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &created),
					resource.TestCheckResourceAttr(csiDriverResourceName, "metadata.0.generate_name", csiDriverNamePrefix),
					resource.TestMatchResourceAttr(csiDriverResourceName, "metadata.0.name", regexp.MustCompile("^"+csiDriverNamePrefix)),
				),
			},
			testAccCSIDriverV1ImportStep(),
			{
				Config: testAccCSIDriverV1Config_completeGenerateName(csiDriverNamePrefix, "two", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(csiDriverResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &updated),
					testAccCheckCSIDriverV1Recreated(&created, &updated, false),
					resource.TestCheckResourceAttr(csiDriverResourceName, "metadata.0.labels.app", "two"),
				),
			},
			testAccCSIDriverV1ImportStep(),
		},
	})
}

// attach_required is ForceNew in SDKv2 and immutable in the API, so a change replaces.
func TestAccCSIDriverV1_attachRequiredForcesReplacement(t *testing.T) {
	var created, replaced storagev1api.CSIDriver
	name := acctest.RandomWithPrefix(csiDriverNamePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSIDriverV1Config_basic(name, true),
				Check:  testAccCheckCSIDriverV1Exists(csiDriverResourceName, &created),
			},
			testAccCSIDriverV1ImportStep(),
			{
				Config: testAccCSIDriverV1Config_basic(name, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(csiDriverResourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &replaced),
					testAccCheckCSIDriverV1Recreated(&created, &replaced, true),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.attach_required", "false"),
				),
			},
			testAccCSIDriverV1ImportStep(),
		},
	})
}

// pod_info_on_mount and volume_lifecycle_modes omitted: state holds the API's defaults and
// the plan stays empty. SDKv2 planned to remove the defaulted modes on every run.
func TestAccCSIDriverV1_apiDefaults(t *testing.T) {
	name := acctest.RandomWithPrefix(csiDriverNamePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSIDriverV1Config_apiDefaults(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.pod_info_on_mount", "false"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.volume_lifecycle_modes.#", "1"),
					resource.TestCheckResourceAttr(csiDriverResourceName, "spec.0.volume_lifecycle_modes.0", "Persistent"),
				),
			},
			testAccCSIDriverV1ImportStep(),
		},
	})
}

// labels = {} and annotations = {} are stored as {}, not collapsed to null as SDKv2 did.
func TestAccCSIDriverV1_emptyMaps(t *testing.T) {
	name := acctest.RandomWithPrefix(csiDriverNamePrefix)
	emptyMap := knownvalue.MapExact(map[string]knownvalue.Check{})

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSIDriverV1Config_emptyMaps(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(csiDriverResourceName,
						tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("labels"), emptyMap),
					statecheck.ExpectKnownValue(csiDriverResourceName,
						tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("annotations"), emptyMap),
				},
			},
			testAccCSIDriverV1ImportStep(),
		},
	})
}

// Configurations SDKv2 never settled on. Read reports the API's defaults, so each plan
// updates towards the configuration; what matters is that it never plans a replacement,
// which SDKv2 did for an omitted spec block.
func TestAccCSIDriverV1_unsettledConfigsUpdateInPlace(t *testing.T) {
	for name, config := range map[string]func(string) string{
		"withoutSpec":               testAccCSIDriverV1Config_withoutSpec,
		"emptyVolumeLifecycleModes": testAccCSIDriverV1Config_emptyVolumeLifecycleModes,
	} {
		t.Run(name, func(t *testing.T) {
			var driver storagev1api.CSIDriver
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckCSIDriverV1Destroy,
				Steps: []resource.TestStep{
					{
						Config:             config(acctest.RandomWithPrefix(csiDriverNamePrefix)),
						Check:              testAccCheckCSIDriverV1Exists(csiDriverResourceName, &driver),
						ExpectNonEmptyPlan: true,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PostApplyPostRefresh: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction(csiDriverResourceName, plancheck.ResourceActionUpdate),
							},
						},
					},
				},
			})
		})
	}
}

func TestAccCSIDriverV1_identity(t *testing.T) {
	name := acctest.RandomWithPrefix(csiDriverNamePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		CheckDestroy: testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSIDriverV1Config_basic(name, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(csiDriverResourceName, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(name),
						"kind":        knownvalue.StringExact("CSIDriver"),
						"api_version": knownvalue.StringExact("storage.k8s.io/v1"),
					}),
				},
			},
			{
				ResourceName:    csiDriverResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// Deleted outside Terraform: Read must drop it from state so the next plan recreates it.
func TestAccCSIDriverV1_disappears(t *testing.T) {
	var driver storagev1api.CSIDriver
	name := acctest.RandomWithPrefix(csiDriverNamePrefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckCSIDriverV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCSIDriverV1Config_basic(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckCSIDriverV1Exists(csiDriverResourceName, &driver),
					testAccDeleteCSIDriverV1(name),
				),
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(csiDriverResourceName, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

// Each step is rejected while planning, so no cluster object is created.
func TestAccCSIDriverV1_validation(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name          = "tf-acc-test"
    generate_name = "tf-acc-test-"
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: `resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = "Invalid_Name"
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`lowercase\s+RFC\s+1123\s+subdomain`),
			},
			{
				Config: `resource "kubernetes_csi_driver_v1" "test" {
  spec {
    attach_required = true
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Invalid Block|at\s+least\s+1\s+elements`),
			},
			{
				Config: `resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = "tf-acc-test-one"
  }
  metadata {
    name = "tf-acc-test-two"
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`at\s+most\s+1\s+elements`),
			},
			{
				Config: `resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = "tf-acc-test"
  }
  spec {
    attach_required = true
  }
  spec {
    attach_required = false
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`at\s+most\s+1\s+elements`),
			},
			{
				Config: `resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = "tf-acc-test"
  }
  spec {
    pod_info_on_mount = true
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Missing required argument`),
			},
			{
				Config: `resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = "tf-acc-test"
  }
  spec {
    attach_required        = true
    volume_lifecycle_modes = ["persistent"]
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`value\s+must\s+be\s+one\s+of`),
			},
		},
	})
}

// The import step for tests whose providers are set on the TestCase. Tests that set
// providers per step add ProtoV6ProviderFactories to it.
func testAccCSIDriverV1ImportStep() resource.TestStep {
	return resource.TestStep{
		ResourceName:      csiDriverResourceName,
		ImportState:       true,
		ImportStateVerify: true,
	}
}

// testAccCheckCSIDriverV1Destroy also matches the deprecated alias, because a move test
// that fails before the move leaves its driver under the old address.
func testAccCheckCSIDriverV1Destroy(s *terraform.State) error {
	conn, err := testAccClientset()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_csi_driver_v1" && rs.Type != "kubernetes_csi_driver" {
			continue
		}
		_, err := conn.StorageV1().CSIDrivers().Get(context.Background(), rs.Primary.ID, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("CSIDriver %q still exists", rs.Primary.ID)
	}
	return nil
}

func testAccCheckCSIDriverV1Exists(address string, out *storagev1api.CSIDriver) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("%s not found in state", address)
		}
		conn, err := testAccClientset()
		if err != nil {
			return err
		}
		driver, err := conn.StorageV1().CSIDrivers().Get(context.Background(), rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*out = *driver
		return nil
	}
}

// testAccCheckCSIDriverV1Recreated compares UIDs, which Kubernetes never reuses, so it
// tells an in-place update from a delete and create of the same name.
func testAccCheckCSIDriverV1Recreated(before, after *storagev1api.CSIDriver, wantRecreated bool) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		recreated := before.UID != after.UID
		if recreated != wantRecreated {
			return fmt.Errorf("CSIDriver %q: recreated = %t, want %t (UID %s -> %s)",
				after.Name, recreated, wantRecreated, before.UID, after.UID)
		}
		return nil
	}
}

func testAccDeleteCSIDriverV1(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccClientset()
		if err != nil {
			return err
		}
		return conn.StorageV1().CSIDrivers().Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

func testAccCSIDriverV1Config_basic(name string, attachRequired bool) string {
	return fmt.Sprintf(`resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = %q
  }

  spec {
    attach_required        = %t
    volume_lifecycle_modes = ["Persistent"]
  }
}
`, name, attachRequired)
}

func testAccCSIDriverV1Config_completeName(name, metadataValue string, podInfoOnMount bool) string {
	return fmt.Sprintf(`resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = %q
    annotations = {
      "example.com/owner" = %[2]q
    }
    labels = {
      "app" = %[2]q
    }
  }

  spec {
    attach_required        = true
    pod_info_on_mount      = %[3]t
    volume_lifecycle_modes = ["Persistent", "Ephemeral"]
  }
}
`, name, metadataValue, podInfoOnMount)
}

func testAccCSIDriverV1Config_completeGenerateName(prefix, metadataValue string, podInfoOnMount bool) string {
	return fmt.Sprintf(`resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    generate_name = %q
    annotations = {
      "example.com/owner" = %[2]q
    }
    labels = {
      "app" = %[2]q
    }
  }

  spec {
    attach_required        = true
    pod_info_on_mount      = %[3]t
    volume_lifecycle_modes = ["Persistent", "Ephemeral"]
  }
}
`, prefix, metadataValue, podInfoOnMount)
}

func testAccCSIDriverV1Config_apiDefaults(name string) string {
	return fmt.Sprintf(`resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = %q
  }

  spec {
    attach_required = true
  }
}
`, name)
}

func testAccCSIDriverV1Config_emptyMaps(name string) string {
	return fmt.Sprintf(`resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name        = %q
    annotations = {}
    labels      = {}
  }

  spec {
    attach_required        = true
    volume_lifecycle_modes = ["Persistent"]
  }
}
`, name)
}

func testAccCSIDriverV1Config_withoutSpec(name string) string {
	return fmt.Sprintf(`resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = %q
  }
}
`, name)
}

func testAccCSIDriverV1Config_emptyVolumeLifecycleModes(name string) string {
	return fmt.Sprintf(`resource "kubernetes_csi_driver_v1" "test" {
  metadata {
    name = %q
  }

  spec {
    attach_required        = true
    volume_lifecycle_modes = []
  }
}
`, name)
}
