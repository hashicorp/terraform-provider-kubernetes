// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestpatch_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
)

var cmGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

// ---- target-object helpers (the patch targets objects it does not own) --------

func createTargetCM(t *testing.T, name string, data map[string]string) {
	t.Helper()
	dyn, err := testAccClients().DynamicClient()
	if err != nil {
		t.Fatalf("dynamic client: %v", err)
	}
	d := map[string]interface{}{}
	for k, v := range data {
		d[k] = v
	}
	cm := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata":   map[string]interface{}{"name": name, "namespace": "default"},
		"data":       d,
	}}
	if _, err := dyn.Resource(cmGVR).Namespace("default").
		Create(context.Background(), cm, metav1.CreateOptions{FieldManager: "external-owner"}); err != nil {
		t.Fatalf("create target ConfigMap: %v", err)
	}
}

func externalMergeCMData(t *testing.T, name, key, val string) {
	t.Helper()
	dyn, err := testAccClients().DynamicClient()
	if err != nil {
		t.Fatalf("dynamic client: %v", err)
	}
	patch := []byte(fmt.Sprintf(`{"data":{%q:%q}}`, key, val))
	if _, err := dyn.Resource(cmGVR).Namespace("default").
		Patch(context.Background(), name, apitypes.MergePatchType, patch, metav1.PatchOptions{FieldManager: "external-tool"}); err != nil {
		t.Fatalf("external merge patch: %v", err)
	}
}

func getTargetCM(name string) (*unstructured.Unstructured, bool) {
	dyn, err := testAccClients().DynamicClient()
	if err != nil {
		return nil, false
	}
	live, err := dyn.Resource(cmGVR).Namespace("default").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, false
	}
	return live, true
}

func deleteTargetCM(name string) {
	dyn, err := testAccClients().DynamicClient()
	if err != nil {
		return
	}
	_ = dyn.Resource(cmGVR).Namespace("default").Delete(context.Background(), name, metav1.DeleteOptions{})
}

func cmData(u *unstructured.Unstructured, key string) (string, bool) {
	v, found, _ := unstructured.NestedString(u.Object, "data", key)
	return v, found
}

func cmAnnotation(u *unstructured.Unstructured, key string) (string, bool) {
	anns, found, _ := unstructured.NestedStringMap(u.Object, "metadata", "annotations")
	if !found {
		return "", false
	}
	v, ok := anns[key]
	return v, ok
}

// ---- tests --------------------------------------------------------------------

// basic: patch adds a data key to a pre-existing ConfigMap; the original data is
// preserved and the patch owns only its key.
func TestAccManifestPatch_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-patch")
	rn := "kubernetes_manifest_patch.test"
	t.Cleanup(func() { deleteTargetCM(name) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			PreConfig: func() { createTargetCM(t, name, map[string]string{"base": "1"}) },
			Config:    testConfigPatchData(name, "yes"),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(rn, "object_exists", "true"),
				resource.TestCheckResourceAttr(rn, "field_manager", "terraform-patch"),
				resource.TestCheckResourceAttrSet(rn, "owned_manifest"),
				func(*terraform.State) error {
					u, ok := getTargetCM(name)
					if !ok {
						return fmt.Errorf("target ConfigMap missing")
					}
					if v, _ := cmData(u, "patched"); v != "yes" {
						return fmt.Errorf("patched not applied: %q", v)
					}
					if v, _ := cmData(u, "base"); v != "1" {
						return fmt.Errorf("base data not preserved: %q", v)
					}
					return nil
				},
			),
		}},
	})
}

// existence check: patching a non-existent object errors (never creates it).
func TestAccManifestPatch_targetMustExist(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-patch")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      testConfigPatchData(name, "yes"),
			ExpectError: regexp.MustCompile("target object not found"),
		}},
	})
}

// null leaf removes an owned field.
func TestAccManifestPatch_nullRemovesField(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-patch")
	t.Cleanup(func() { deleteTargetCM(name) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { createTargetCM(t, name, map[string]string{"base": "1"}) },
				Config:    testConfigPatchAnnotation(name, `"present"`),
				Check: func(*terraform.State) error {
					u, _ := getTargetCM(name)
					if v, ok := cmAnnotation(u, "example.com/x"); !ok || v != "present" {
						return fmt.Errorf("annotation not set: %q ok=%v", v, ok)
					}
					return nil
				},
			},
			{
				Config: testConfigPatchAnnotation(name, `null`),
				Check: func(*terraform.State) error {
					u, _ := getTargetCM(name)
					if v, ok := cmAnnotation(u, "example.com/x"); ok {
						return fmt.Errorf("annotation should be removed, still %q", v)
					}
					return nil
				},
			},
		},
	})
}

// co-owned: a field written by another manager does not cause drift.
func TestAccManifestPatch_coOwnedNoDrift(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-patch")
	t.Cleanup(func() { deleteTargetCM(name) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { createTargetCM(t, name, map[string]string{"base": "1"}) },
				Config:    testConfigPatchData(name, "yes"),
			},
			{
				PreConfig: func() { externalMergeCMData(t, name, "other", "set-externally") },
				Config:    testConfigPatchData(name, "yes"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// relinquish (default): destroying the patch leaves the object and the field intact.
func TestAccManifestPatch_destroyRelinquish(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-patch")
	t.Cleanup(func() { deleteTargetCM(name) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			u, ok := getTargetCM(name)
			if !ok {
				return fmt.Errorf("relinquish must NOT delete the target object")
			}
			if v, _ := cmData(u, "patched"); v != "yes" {
				return fmt.Errorf("relinquish must leave the patched field, got %q", v)
			}
			return nil
		},
		Steps: []resource.TestStep{{
			PreConfig: func() { createTargetCM(t, name, map[string]string{"base": "1"}) },
			Config:    testConfigPatchData(name, "yes"),
		}},
	})
}

// remove_fields: destroying the patch removes the owned field but keeps the object.
func TestAccManifestPatch_destroyRemoveFields(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-patch")
	t.Cleanup(func() { deleteTargetCM(name) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			u, ok := getTargetCM(name)
			if !ok {
				return fmt.Errorf("remove_fields must not delete the object")
			}
			if v, ok := cmData(u, "patched"); ok {
				return fmt.Errorf("remove_fields must remove the owned field, still %q", v)
			}
			if v, _ := cmData(u, "base"); v != "1" {
				return fmt.Errorf("remove_fields must keep other managers' fields, base=%q", v)
			}
			return nil
		},
		Steps: []resource.TestStep{{
			PreConfig: func() { createTargetCM(t, name, map[string]string{"base": "1"}) },
			Config:    testConfigPatchDataRemove(name, "yes"),
		}},
	})
}

// patch_json escape hatch (merge patch).
func TestAccManifestPatch_patchJSON(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-patch")
	rn := "kubernetes_manifest_patch.test"
	t.Cleanup(func() { deleteTargetCM(name) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			PreConfig: func() { createTargetCM(t, name, map[string]string{"base": "1"}) },
			Config:    testConfigPatchJSON(name),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(rn, "object_exists", "true"),
				func(*terraform.State) error {
					u, _ := getTargetCM(name)
					if v, _ := cmData(u, "viajson"); v != "yes" {
						return fmt.Errorf("patch_json not applied: %q", v)
					}
					return nil
				},
			),
		}},
	})
}

// import by key=value id.
func TestAccManifestPatch_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-patch")
	rn := "kubernetes_manifest_patch.test"
	t.Cleanup(func() { deleteTargetCM(name) })

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { createTargetCM(t, name, map[string]string{"base": "1"}) },
				Config:    testConfigPatchData(name, "yes"),
			},
			{
				ResourceName: rn,
				ImportState:  true,
				ImportStateId: fmt.Sprintf(
					"apiVersion=v1,kind=ConfigMap,namespace=default,name=%s,fieldManager=terraform-patch", name),
				ImportStateVerify: true,
				// patch is user config and cannot be reconstructed from the live object.
				ImportStateVerifyIgnore: []string{"patch", "patch_json", "patch_type", "force_conflicts", "ignore_fields"},
			},
		},
	})
}

// ---- config builders ----------------------------------------------------------

func testConfigPatchData(name, val string) string {
	return fmt.Sprintf(`
resource "kubernetes_manifest_patch" "test" {
  api_version = "v1"
  kind        = "ConfigMap"
  name        = %q
  namespace   = "default"
  patch = {
    data = {
      patched = %q
    }
  }
}
`, name, val)
}

func testConfigPatchDataRemove(name, val string) string {
	return fmt.Sprintf(`
resource "kubernetes_manifest_patch" "test" {
  api_version      = "v1"
  kind             = "ConfigMap"
  name             = %q
  namespace        = "default"
  destroy_behavior = "remove_fields"
  patch = {
    data = {
      patched = %q
    }
  }
}
`, name, val)
}

func testConfigPatchAnnotation(name, valueExpr string) string {
	return fmt.Sprintf(`
resource "kubernetes_manifest_patch" "test" {
  api_version = "v1"
  kind        = "ConfigMap"
  name        = %q
  namespace   = "default"
  patch = {
    metadata = {
      annotations = {
        "example.com/x" = %s
      }
    }
  }
}
`, name, valueExpr)
}

func testConfigPatchJSON(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_manifest_patch" "test" {
  api_version = "v1"
  kind        = "ConfigMap"
  name        = %q
  namespace   = "default"
  patch_type  = "merge"
  patch_json  = jsonencode({ data = { viajson = "yes" } })
}
`, name)
}
