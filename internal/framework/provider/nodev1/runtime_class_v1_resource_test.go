// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	nodev1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const runtimeClassAddr = "kubernetes_runtime_class_v1.test"

// testAccRuntimeClassV1ImportStep follows every apply that changes the remote object, so
// each write is shown to have reached Kubernetes and to be readable back unchanged.
func testAccRuntimeClassV1ImportStep() resource.TestStep {
	return resource.TestStep{
		ResourceName:      runtimeClassAddr,
		ImportState:       true,
		ImportStateVerify: true,
	}
}

func testAccRuntimeClassV1Name() string {
	return fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
}

// Ported from the SDKv2 TestAccKubernetesRuntimeClassV1_basic with its assertions unchanged.
// handler is ForceNew, so steps 2 and 3 replace the object.
func TestAccKubernetesRuntimeClassV1_basic(t *testing.T) {
	var conf nodev1.RuntimeClass
	name := testAccRuntimeClassV1Name()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuntimeClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRuntimeClassV1Exists(runtimeClassAddr, &conf),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.name", name),
					resource.TestCheckResourceAttr(runtimeClassAddr, "handler", "myclass"),
				),
			},
			testAccRuntimeClassV1ImportStep(),
			{
				Config: testAccRuntimeClassV1Config_annotations(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(runtimeClassAddr, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRuntimeClassV1Exists(runtimeClassAddr, &conf),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.name", name),
					resource.TestCheckResourceAttr(runtimeClassAddr, "handler", "newclass"),
					resource.TestCheckResourceAttrSet(runtimeClassAddr, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(runtimeClassAddr, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(runtimeClassAddr, "metadata.0.uid"),
				),
			},
			testAccRuntimeClassV1ImportStep(),
			{
				Config: testAccRuntimeClassV1Config_labels(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(runtimeClassAddr, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckRuntimeClassV1Exists(runtimeClassAddr, &conf),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.%", "2"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.name", name),
					resource.TestCheckResourceAttr(runtimeClassAddr, "handler", "my-class"),
					resource.TestCheckResourceAttrSet(runtimeClassAddr, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(runtimeClassAddr, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(runtimeClassAddr, "metadata.0.uid"),
				),
			},
			testAccRuntimeClassV1ImportStep(),
		},
	})
}

// Label and annotation changes are the only in-place updates this resource supports.
// Each must patch the existing object, never replace it.
func TestAccKubernetesRuntimeClassV1_labelsUpdate(t *testing.T) {
	name := testAccRuntimeClassV1Name()
	sameUID := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuntimeClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_withLabel(name, "env", "staging"),
				Check:  resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.env", "staging"),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(runtimeClassAddr, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")),
				},
			},
			testAccRuntimeClassV1ImportStep(),
			{
				Config: testAccRuntimeClassV1Config_withLabel(name, "env", "production"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(runtimeClassAddr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.env", "production"),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(runtimeClassAddr, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")),
				},
			},
			testAccRuntimeClassV1ImportStep(),
		},
	})
}

func TestAccKubernetesRuntimeClassV1_annotationsUpdate(t *testing.T) {
	name := testAccRuntimeClassV1Name()
	sameUID := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuntimeClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_withAnnotation(name, "owner", "team-a"),
				Check:  resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.owner", "team-a"),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(runtimeClassAddr, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")),
				},
			},
			testAccRuntimeClassV1ImportStep(),
			{
				Config: testAccRuntimeClassV1Config_withAnnotation(name, "owner", "team-b"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(runtimeClassAddr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.owner", "team-b"),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(runtimeClassAddr, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")),
				},
			},
			testAccRuntimeClassV1ImportStep(),
		},
	})
}

// name is server-assigned here, so metadata.name is Computed with no config value. An
// unrelated edit marks such values unknown; without UseStateForUnknown ahead of
// RequiresReplace, adding a label would plan a replacement. The update step guards that.
func TestAccKubernetesRuntimeClassV1_generateName(t *testing.T) {
	prefix := "tf-acc-gen-"
	sameName := statecheck.CompareValue(compare.ValuesSame())
	namePath := tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuntimeClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_generateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(runtimeClassAddr, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(runtimeClassAddr, "metadata.0.uid"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "handler", "runc"),
				),
				ConfigStateChecks: []statecheck.StateCheck{sameName.AddStateValue(runtimeClassAddr, namePath)},
			},
			testAccRuntimeClassV1ImportStep(),
			{
				Config: testAccRuntimeClassV1Config_generateNameWithLabel(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(runtimeClassAddr, plancheck.ResourceActionUpdate),
					},
				},
				Check:             resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.env", "staging"),
				ConfigStateChecks: []statecheck.StateCheck{sameName.AddStateValue(runtimeClassAddr, namePath)},
			},
			testAccRuntimeClassV1ImportStep(),
		},
	})
}

func TestAccKubernetesRuntimeClassV1_labelsAndAnnotations(t *testing.T) {
	name := testAccRuntimeClassV1Name()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuntimeClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_labelsAndAnnotations(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.name", name),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.env", "staging"),
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.annotations.owner", "team-a"),
				),
			},
			testAccRuntimeClassV1ImportStep(),
		},
	})
}

// A label-only update must leave annotations it does not manage alone. SDKv2 JSON-patched
// only the maps that changed; a full-object update would silently delete the annotation
// below. The key is under kubernetes.io, so Read filters it and neither state nor plan can
// see it — only the live object can.
func TestAccKubernetesRuntimeClassV1_updatePreservesUnmanagedAnnotation(t *testing.T) {
	name := testAccRuntimeClassV1Name()
	const externalKey = "example.kubernetes.io/owner"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuntimeClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_withLabel(name, "env", "staging"),
			},
			{
				PreConfig: func() {
					conn, err := testAccClientset()
					if err != nil {
						t.Fatal(err)
					}
					patch := fmt.Sprintf(`{"metadata":{"annotations":{%q:"external-controller"}}}`, externalKey)
					if _, err := conn.NodeV1().RuntimeClasses().Patch(context.Background(), name,
						k8stypes.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
						t.Fatal(err)
					}
				},
				Config: testAccRuntimeClassV1Config_withLabel(name, "env", "production"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(runtimeClassAddr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(runtimeClassAddr, "metadata.0.labels.env", "production"),
					resource.TestCheckNoResourceAttr(runtimeClassAddr, "metadata.0.annotations.%"),
					testAccCheckRuntimeClassV1LiveAnnotation(name, externalKey, "external-controller"),
				),
			},
			testAccRuntimeClassV1ImportStep(),
		},
	})
}

// SDKv2's handler pattern is anchored only at the start. Reject what it rejected, accept
// what it accepted — including values the API server may itself refuse later.
func TestAccKubernetesRuntimeClassV1_handlerValidation(t *testing.T) {
	name := testAccRuntimeClassV1Name()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuntimeClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccRuntimeClassV1Config_handler(name, "Runc"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`must match regular expression`),
			},
			// ExpectError must not be last: the harness plans its destroy from the final config.
			{
				Config:             testAccRuntimeClassV1Config_handler(name, "runc_v2"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Structured import through an identity block: {api_version, kind, name}.
func TestAccKubernetesRuntimeClassV1_importWithIdentity(t *testing.T) {
	name := testAccRuntimeClassV1Name()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRuntimeClassV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccRuntimeClassV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(runtimeClassAddr, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(name),
						"kind":        knownvalue.StringExact("RuntimeClass"),
						"api_version": knownvalue.StringExact("node.k8s.io/v1"),
					}),
				},
			},
			{
				ResourceName:    runtimeClassAddr,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
				// ImportStateVerify is unsupported for plannable import blocks; the import
				// planning with no changes is the assertion.
			},
		},
	})
}

func testAccCheckRuntimeClassV1LiveAnnotation(name, key, want string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccClientset()
		if err != nil {
			return err
		}
		out, err := conn.NodeV1().RuntimeClasses().Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if got, ok := out.Annotations[key]; !ok || got != want {
			return fmt.Errorf("live annotation %q = %q (present %t), want %q", key, got, ok, want)
		}
		return nil
	}
}

func testAccRuntimeClassV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
  }
  handler = "myclass"
}
`, name)
}

func testAccRuntimeClassV1Config_annotations(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }
    name = %q
  }
  handler = "newclass"
}
`, name)
}

func testAccRuntimeClassV1Config_labels(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
      TestAnnotationTwo = "two"
    }

    labels = {
      TestLabelOne = "one"
      TestLabelTwo = "two"
    }
    name = %q
  }
  handler = "my-class"
}
`, name)
}

func testAccRuntimeClassV1Config_handler(name, handler string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
  }
  handler = %q
}
`, name, handler)
}

func testAccRuntimeClassV1Config_withLabel(name, key, value string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    labels = {
      %q = %q
    }
  }
  handler = "runc"
}
`, name, key, value)
}

func testAccRuntimeClassV1Config_withAnnotation(name, key, value string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    annotations = {
      %q = %q
    }
  }
  handler = "runc"
}
`, name, key, value)
}

func testAccRuntimeClassV1Config_generateName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    generate_name = %q
  }
  handler = "runc"
}
`, prefix)
}

func testAccRuntimeClassV1Config_generateNameWithLabel(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    generate_name = %q
    labels = {
      env = "staging"
    }
  }
  handler = "runc"
}
`, prefix)
}

func testAccRuntimeClassV1Config_labelsAndAnnotations(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    labels = {
      env = "staging"
    }
    annotations = {
      owner = "team-a"
    }
  }
  handler = "runc"
}
`, name)
}
