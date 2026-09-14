// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package storagev1_test

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	sdkv2terraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	storagev1api "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ── destroy / exists helpers ──────────────────────────────────────────────────

func testAccCheckStorageClassV1Destroy(s *terraform.State) error {
	p := kubernetes.Provider()
	p.Configure(context.Background(), sdkv2terraform.NewResourceConfigRaw(nil))
	conn, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_storage_class_v1" {
			continue
		}
		resp, err := conn.StorageV1().StorageClasses().Get(
			context.Background(), rs.Primary.ID, metav1.GetOptions{},
		)
		if err == nil && resp.Name == rs.Primary.ID {
			return fmt.Errorf("StorageClass still exists: %s", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckStorageClassV1Exists(n string, obj *storagev1api.StorageClass) tfresource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		p := kubernetes.Provider()
		p.Configure(context.Background(), sdkv2terraform.NewResourceConfigRaw(nil))
		conn, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		out, err := conn.StorageV1().StorageClasses().Get(
			context.Background(), rs.Primary.ID, metav1.GetOptions{},
		)
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

// ── SDKv2 parity: basic (kind cluster) ───────────────────────────────────────

// TestAccStorageClassV1_basic is the primary parity test for the kind cluster.
// It covers: full metadata, mount_options, explicit scalar fields, import,
// in-place update of metadata + reclaim_policy + allow_volume_expansion, and
// a final step that removes parameters and mount_options to verify clean state.
func TestAccStorageClassV1_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-sc")
	resourceName := "kubernetes_storage_class_v1.test"
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckStorageClassV1Destroy,
		Steps: []tfresource.TestStep{
			// Step 1: create with full metadata, mount_options, explicit scalars.
			{
				Config: testAccStorageClassV1Config_basic(name, provisioner),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					testAccCheckStorageClassV1Exists(resourceName, &storagev1api.StorageClass{}),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					tfresource.TestCheckResourceAttr(resourceName, "storage_provisioner", provisioner),
					tfresource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					tfresource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					tfresource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
					tfresource.TestCheckResourceAttr(resourceName, "mount_options.#", "2"),
				),
			},
			// Import step — verify no drift after import.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"metadata.0.generation",
				},
			},
			// Step 2: update mutable fields (metadata, reclaim_policy, allow_volume_expansion).
			// mount_options and volume_binding_mode are ForceNew so they trigger replace.
			{
				Config: testAccStorageClassV1Config_modified(name, provisioner),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttr(resourceName, "storage_provisioner", provisioner),
					tfresource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Retain"),
					tfresource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "WaitForFirstConsumer"),
					tfresource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "false"),
					tfresource.TestCheckResourceAttr(resourceName, "mount_options.#", "1"),
				),
			},
			// Step 3: remove mount_options entirely — verify clean empty state.
			{
				Config: testAccStorageClassV1Config_noParameters(name, provisioner),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					tfresource.TestCheckResourceAttr(resourceName, "storage_provisioner", provisioner),
					tfresource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					tfresource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					tfresource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
					tfresource.TestCheckResourceAttr(resourceName, "mount_options.#", "0"),
				),
			},
		},
	})
}

// ── SDKv2 parity: volume expansion toggle ─────────────────────────────────────

// TestAccStorageClassV1_volumeExpansion verifies allow_volume_expansion can be
// toggled true→false in-place without a destroy/recreate.
func TestAccStorageClassV1_volumeExpansion(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-sc")
	resourceName := "kubernetes_storage_class_v1.test"
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckStorageClassV1Destroy,
		Steps: []tfresource.TestStep{
			{
				Config: testAccStorageClassV1Config_volumeExpansion(name, provisioner),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					testAccCheckStorageClassV1Exists(resourceName, &storagev1api.StorageClass{}),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					tfresource.TestCheckResourceAttr(resourceName, "storage_provisioner", provisioner),
					tfresource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					tfresource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					tfresource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
				),
			},
			{
				Config: testAccStorageClassV1Config_volumeExpansionModified(name, provisioner),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					tfresource.TestCheckResourceAttr(resourceName, "storage_provisioner", provisioner),
					tfresource.TestCheckResourceAttr(resourceName, "reclaim_policy", "Delete"),
					tfresource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
					tfresource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "false"),
				),
			},
		},
	})
}

// ── SDKv2 parity: volume_expansion is in-place update, not replace ────────────

// TestAccStorageClassV1_volumeExpansionIsUpdate asserts the plan action is
// Update (not replace) when allow_volume_expansion changes.
func TestAccStorageClassV1_volumeExpansionIsUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-sc")
	resourceName := "kubernetes_storage_class_v1.test"
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccStorageClassV1Config_volumeExpansion(name, provisioner),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "true"),
			},
			{
				Config: testAccStorageClassV1Config_volumeExpansionModified(name, provisioner),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: tfresource.TestCheckResourceAttr(resourceName, "allow_volume_expansion", "false"),
			},
		},
	})
}

// ── SDKv2 parity: allowed_topologies ─────────────────────────────────────────

// TestAccStorageClassV1_allowedTopologies verifies the allowed_topologies block
// is created with correct key and zone values, and that parameters are empty.
func TestAccStorageClassV1_allowedTopologies(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-sc")
	resourceName := "kubernetes_storage_class_v1.test"
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckStorageClassV1Destroy,
		Steps: []tfresource.TestStep{
			{
				Config: testAccStorageClassV1Config_allowedTopologies(name, provisioner),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					testAccCheckStorageClassV1Exists(resourceName, &storagev1api.StorageClass{}),
					tfresource.TestCheckResourceAttr(resourceName, "allowed_topologies.#", "1"),
					tfresource.TestCheckResourceAttr(resourceName, "allowed_topologies.0.match_label_expressions.#", "1"),
					tfresource.TestCheckResourceAttr(resourceName, "allowed_topologies.0.match_label_expressions.0.key", "topology.kubernetes.io/zone"),
					tfresource.TestCheckResourceAttr(resourceName, "allowed_topologies.0.match_label_expressions.0.values.#", "2"),
					// TestCheckTypeSetElemAttr used for set values — order not guaranteed.
					tfresource.TestCheckTypeSetElemAttr(resourceName, "allowed_topologies.0.match_label_expressions.0.values.*", "us-west1-a"),
					tfresource.TestCheckTypeSetElemAttr(resourceName, "allowed_topologies.0.match_label_expressions.0.values.*", "us-west1-b"),
					tfresource.TestCheckResourceAttr(resourceName, "parameters.%", "0"),
				),
			},
		},
	})
}

// ── SDKv2 parity: generate_name ───────────────────────────────────────────────

// TestAccStorageClassV1_generateName creates a StorageClass using generate_name
// and verifies the server-assigned name matches the prefix. Also verifies import.
func TestAccStorageClassV1_generateName(t *testing.T) {
	prefix := "tf-acc-sc-gen-"
	resourceName := "kubernetes_storage_class_v1.test"
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckStorageClassV1Destroy,
		Steps: []tfresource.TestStep{
			{
				Config: testAccStorageClassV1Config_generateName(prefix, provisioner),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					testAccCheckStorageClassV1Exists(resourceName, &storagev1api.StorageClass{}),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					tfresource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					tfresource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
					"metadata.0.generation",
				},
			},
		},
	})
}

// ── Framework-specific: ForceNew assertions ───────────────────────────────────

// TestAccStorageClassV1_provisionerRequiresReplace verifies that changing
// storage_provisioner triggers a destroy + create (RequiresReplace).
func TestAccStorageClassV1_provisionerRequiresReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-sc")
	resourceName := "kubernetes_storage_class_v1.test"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccStorageClassV1Config_noParameters(name, "rancher.io/local-path"),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "storage_provisioner", "rancher.io/local-path"),
			},
			{
				Config: testAccStorageClassV1Config_noParameters(name, "docker.io/hostpath"),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
			},
		},
	})
}

// TestAccStorageClassV1_volumeBindingModeRequiresReplace verifies that changing
// volume_binding_mode triggers a destroy + create (ForceNew/RequiresReplace).
func TestAccStorageClassV1_volumeBindingModeRequiresReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-sc")
	resourceName := "kubernetes_storage_class_v1.test"
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccStorageClassV1Config_volumeExpansion(name, provisioner), // Immediate
				Check:  tfresource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "Immediate"),
			},
			{
				Config: testAccStorageClassV1Config_withVolumeBindingMode(name, provisioner, "WaitForFirstConsumer"),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: tfresource.TestCheckResourceAttr(resourceName, "volume_binding_mode", "WaitForFirstConsumer"),
			},
		},
	})
}

// ── Framework-specific: disappears ───────────────────────────────────────────

// TestAccStorageClassV1_disappears verifies that if the StorageClass is deleted
// outside Terraform, the next plan detects it and proposes to recreate it.
func TestAccStorageClassV1_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-sc")
	resourceName := "kubernetes_storage_class_v1.test"
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccStorageClassV1Config_noParameters(name, provisioner),
				Check:  tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
			},
			{
				// Delete out-of-band, then refresh — provider detects 404 and
				// clears state; subsequent plan proposes a create.
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
				PreConfig: func() {
					_ = exec.Command(
						"kubectl", "delete", "storageclass", name, "--ignore-not-found",
					).Run()
				},
			},
		},
	})
}

// ── Framework-specific: upgrade from SDKv2 ───────────────────────────────────

// TestAccStorageClassV1_upgradeFromSDKv2 provisions the resource with the last
// SDKv2 release, then switches to the local Framework provider and verifies
// zero plan diff — proving the Framework reads SDKv2 state without any upgrade.
//
// Skipped in -short mode because it downloads from the Terraform registry.
func TestAccStorageClassV1_upgradeFromSDKv2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping registry-dependent upgrade test in -short mode")
	}

	name := acctest.RandomWithPrefix("tf-acc-sc")
	resourceName := "kubernetes_storage_class_v1.test"
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			// Step 1: provision with the last SDKv2 release.
			// Writes state at schema version 0 with TypeList metadata.
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.2.1",
					},
				},
				Config: testAccStorageClassV1Config_noParameters(name, provisioner),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					tfresource.TestCheckResourceAttr(resourceName, "storage_provisioner", provisioner),
				),
			},
			// Step 2: switch to the local Framework provider.
			// ListNestedBlock produces identical state JSON to TypeList{MaxItems:1}
			// so no UpgradeState is needed — plan must be empty.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccStorageClassV1Config_noParameters(name, provisioner),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// TestAccStorageClassV1_moved provisions the deprecated kubernetes_storage_class
// with the last SDKv2 release then uses a moved block to migrate state to
// kubernetes_storage_class_v1 with the Framework provider. The plan must be
// empty — proving MoveState translates the state without drift.
//
// Skipped in -short mode because it downloads from the Terraform registry.
func TestAccStorageClassV1_moved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping registry-dependent moved-block test in -short mode")
	}

	name := acctest.RandomWithPrefix("tf-acc-sc")
	provisioner := "rancher.io/local-path"

	tfresource.ParallelTest(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			// Step 1: provision kubernetes_storage_class (deprecated type) with
			// the last SDKv2 release.
			{
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						Source:            "hashicorp/kubernetes",
						VersionConstraint: "3.0.1",
					},
				},
				Config: testAccStorageClassConfig_deprecated(name, provisioner),
			},
			// Step 2: add a moved block and switch to the Framework provider.
			// MoveState translates kubernetes_storage_class → kubernetes_storage_class_v1.
			// Plan must be empty — no destroy, no create.
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccStorageClassV1Config_movedFrom(name, provisioner),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ── HCL config helpers ────────────────────────────────────────────────────────

func testAccStorageClassV1Config_basic(name, provisioner string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }
  }

  storage_provisioner    = %[2]q
  reclaim_policy         = "Delete"
  volume_binding_mode    = "Immediate"
  allow_volume_expansion = true
  mount_options          = ["foo", "bar"]
}
`, name, provisioner)
}

func testAccStorageClassV1Config_modified(name, provisioner string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
    annotations = {
      TestAnnotationOne = "one"
      Different         = "1234"
    }
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }
  }

  storage_provisioner    = %[2]q
  reclaim_policy         = "Retain"
  volume_binding_mode    = "WaitForFirstConsumer"
  allow_volume_expansion = false
  mount_options          = ["foo"]
}
`, name, provisioner)
}

func testAccStorageClassV1Config_volumeExpansion(name, provisioner string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  storage_provisioner    = %[2]q
  allow_volume_expansion = true
}
`, name, provisioner)
}

func testAccStorageClassV1Config_volumeExpansionModified(name, provisioner string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  storage_provisioner    = %[2]q
  allow_volume_expansion = false
}
`, name, provisioner)
}

func testAccStorageClassV1Config_noParameters(name, provisioner string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  storage_provisioner = %[2]q
}
`, name, provisioner)
}

func testAccStorageClassV1Config_generateName(prefix, provisioner string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  storage_provisioner = %[2]q
}
`, prefix, provisioner)
}

func testAccStorageClassV1Config_allowedTopologies(name, provisioner string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  storage_provisioner = %[2]q

  allowed_topologies {
    match_label_expressions {
      key    = "topology.kubernetes.io/zone"
      values = ["us-west1-a", "us-west1-b"]
    }
  }
}
`, name, provisioner)
}

func testAccStorageClassV1Config_withVolumeBindingMode(name, provisioner, mode string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  storage_provisioner = %[2]q
  volume_binding_mode = %[3]q
}
`, name, provisioner, mode)
}

// testAccStorageClassConfig_deprecated uses the old resource type name —
// the one still registered in the SDKv2 provider for backwards compatibility.
func testAccStorageClassConfig_deprecated(name, provisioner string) string {
	return fmt.Sprintf(`
resource "kubernetes_storage_class" "test" {
  metadata {
    name = %[1]q
  }

  storage_provisioner = %[2]q
}
`, name, provisioner)
}

// testAccStorageClassV1Config_movedFrom contains the moved block that migrates
// kubernetes_storage_class.test → kubernetes_storage_class_v1.test.
func testAccStorageClassV1Config_movedFrom(name, provisioner string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_storage_class.test
  to   = kubernetes_storage_class_v1.test
}

resource "kubernetes_storage_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  storage_provisioner = %[2]q
}
`, name, provisioner)
}
