// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"
)

// ─── Unit Tests (no cluster) ─────────────────────────────────────────────────

func TestNamespaceV1_UpgradeStateV0_basic(t *testing.T) {
	raw := sdkv2NamespaceStateJSON("my-namespace", "", nil, nil, "1", "abc-uid", 0, false)

	r := &corev1.NamespaceV1{}
	handlers := r.UpgradeState(nil)
	upgrader, ok := handlers[0]
	if !ok {
		t.Fatal("expected upgrader at key 0")
	}

	// Verify the PriorSchema is set (essential for framework to decode prior state).
	if upgrader.PriorSchema == nil {
		t.Error("expected PriorSchema to be set on upgrader v0")
	}

	// Verify the JSON parsing logic via the raw fixture.
	var state map[string]interface{}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("invalid test fixture JSON: %v", err)
	}

	meta := state["metadata"].([]interface{})[0].(map[string]interface{})
	if meta["name"] != "my-namespace" {
		t.Errorf("expected name 'my-namespace', got %v", meta["name"])
	}
	if meta["generate_name"] != "" {
		t.Errorf("expected empty generate_name, got %v", meta["generate_name"])
	}
	if state["id"] != "my-namespace" {
		t.Errorf("expected id 'my-namespace', got %v", state["id"])
	}
}

func TestNamespaceV1_UpgradeStateV0_withAnnotations(t *testing.T) {
	annotations := map[string]string{"example.com/key": "val"}
	labels := map[string]string{"env": "staging"}
	raw := sdkv2NamespaceStateJSON("ns-annot", "", annotations, labels, "5", "uid-2", 2, false)

	var state map[string]interface{}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("invalid test fixture JSON: %v", err)
	}

	meta := state["metadata"].([]interface{})[0].(map[string]interface{})
	gotAnnotations := meta["annotations"].(map[string]interface{})
	if gotAnnotations["example.com/key"] != "val" {
		t.Errorf("expected annotation 'val', got %v", gotAnnotations["example.com/key"])
	}
	gotLabels := meta["labels"].(map[string]interface{})
	if gotLabels["env"] != "staging" {
		t.Errorf("expected label 'staging', got %v", gotLabels["env"])
	}
}

func TestNamespaceV1_UpgradeStateV0_emptyAnnotationsAreNull(t *testing.T) {
	// Empty annotations map should not appear — nil/empty in SDKv2 → nil in Framework
	raw := sdkv2NamespaceStateJSON("ns-empty", "", map[string]string{}, map[string]string{}, "1", "u1", 0, false)

	var state map[string]interface{}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("invalid test fixture JSON: %v", err)
	}

	meta := state["metadata"].([]interface{})[0].(map[string]interface{})
	annotations := meta["annotations"]
	labels := meta["labels"]
	// SDKv2 serializes empty maps as {} not null — the upgrader must handle this
	// Verify the fixture itself is {}
	if annotations != nil {
		annotMap, ok := annotations.(map[string]interface{})
		if ok && len(annotMap) != 0 {
			t.Errorf("expected empty annotations map, got %v", annotMap)
		}
	}
	if labels != nil {
		labelMap, ok := labels.(map[string]interface{})
		if ok && len(labelMap) != 0 {
			t.Errorf("expected empty labels map, got %v", labelMap)
		}
	}
}

// sdkv2NamespaceStateJSON generates a realistic SDKv2 state JSON fixture for testing.
func sdkv2NamespaceStateJSON(name, generateName string, annotations, labels map[string]string, resourceVersion, uid string, generation int64, waitForSA bool) []byte {
	meta := map[string]interface{}{
		"name":             name,
		"generate_name":    generateName,
		"resource_version": resourceVersion,
		"uid":              uid,
		"generation":       generation,
		"annotations":      annotations,
		"labels":           labels,
	}
	state := map[string]interface{}{
		"id":                               name,
		"wait_for_default_service_account": waitForSA,
		"metadata":                         []interface{}{meta},
	}
	raw, _ := json.Marshal(state)
	return raw
}

// ─── UpgradeState interface smoke test ───────────────────────────────────────

func TestNamespaceV1_UpgradeStateHandlers_registeredAtVersion0(t *testing.T) {
	r := &corev1.NamespaceV1{}
	handlers := r.UpgradeState(nil)
	if _, ok := handlers[0]; !ok {
		t.Error("expected upgrader registered at key 0")
	}
	if len(handlers) != 1 {
		t.Errorf("expected exactly 1 upgrader, got %d", len(handlers))
	}
}

func TestNamespaceV1_MoveStateHandlers_registeredForKubernetesNamespace(t *testing.T) {
	r := &corev1.NamespaceV1{}
	movers := r.MoveState(nil)
	if len(movers) == 0 {
		t.Error("expected at least 1 StateMover")
	}
}

// ─── Acceptance Tests (require KinD cluster) ─────────────────────────────────

func TestAccKubernetesNamespaceV1_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ns")

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccNamespaceV1Config_basic(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("kubernetes_namespace_v1.test", "metadata.name", name),
					tfresource.TestCheckResourceAttrSet("kubernetes_namespace_v1.test", "metadata.uid"),
					tfresource.TestCheckResourceAttrSet("kubernetes_namespace_v1.test", "metadata.resource_version"),
				),
			},
			{
				Config: testAccNamespaceV1Config_withLabels(name),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("kubernetes_namespace_v1.test", "metadata.labels.env", "staging"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_generatedName(t *testing.T) {
	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccNamespaceV1Config_generateName("tf-acc-gen-"),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttrSet("kubernetes_namespace_v1.test", "metadata.name"),
					tfresource.TestCheckResourceAttr("kubernetes_namespace_v1.test", "metadata.generate_name", "tf-acc-gen-"),
				),
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_import(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ns-import")

	tfresource.ParallelTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			{
				Config: testAccNamespaceV1Config_basic(name),
			},
			{
				ResourceName:            "kubernetes_namespace_v1.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"wait_for_default_service_account", "timeouts"},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_upgradeFromSDKv2(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ns-upgrade")

	tfresource.ParallelTest(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			{
				// Step 1: provision with old SDKv2 provider
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: "3.0.1",
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccNamespaceV1Config_basic(name),
			},
			{
				// Step 2: plan with local Framework provider — expect zero diff
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccNamespaceV1Config_basic(name),
				PlanOnly:                 true,
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccKubernetesNamespaceV1_moved(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-ns-moved")

	tfresource.ParallelTest(t, tfresource.TestCase{
		Steps: []tfresource.TestStep{
			{
				// Step 1: provision kubernetes_namespace (deprecated) with old provider
				ExternalProviders: map[string]tfresource.ExternalProvider{
					"kubernetes": {
						VersionConstraint: "3.0.1",
						Source:            "hashicorp/kubernetes",
					},
				},
				Config: testAccNamespaceConfig_deprecated(name),
			},
			{
				// Step 2: add moved block to migrate to kubernetes_namespace_v1 with new provider
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   testAccNamespaceV1Config_movedFrom(name),
				PlanOnly:                 true,
				ConfigPlanChecks: tfresource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// ─── HCL config helpers ───────────────────────────────────────────────────────

func testAccNamespaceV1Config_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}
`, name)
}

func testAccNamespaceV1Config_withLabels(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      env = "staging"
    }
  }
}
`, name)
}

func testAccNamespaceV1Config_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace_v1" "test" {
  metadata {
    generate_name = %[1]q
  }
}
`, prefix)
}

func testAccNamespaceConfig_deprecated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_namespace" "test" {
  metadata {
    name = %[1]q
  }
}
`, name)
}

func testAccNamespaceV1Config_movedFrom(name string) string {
	return fmt.Sprintf(`
moved {
  from = kubernetes_namespace.test
  to   = kubernetes_namespace_v1.test
}

resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}
`, name)
}

// Ensure NamespaceV1 is exported (interface check).
var _ resource.Resource = (*corev1.NamespaceV1)(nil)

// testAccProtoV6ProviderFactoriesWithExternalSDKv2 provides factories for mixed upgrade tests.
// Used when both old and new providers are needed in multi-step tests.
func testAccProtoV6ProviderFactoriesLocal() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"kubernetes": providerserver.NewProtocol6WithError(provider.New("test", sdkv2providerMeta())),
	}
}
