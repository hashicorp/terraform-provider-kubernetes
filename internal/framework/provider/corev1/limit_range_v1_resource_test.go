// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	corev1api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const limitRangeResourceName = "kubernetes_limit_range_v1.test"

// splitID splits a "namespace/name" resource ID — the format BuildId produces.
func splitID(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format %q, expected namespace/name", id)
	}
	return parts[0], parts[1], nil
}

// ── lifecycle tests ───────────────────────────────────────────────────────────

func TestAccKubernetesLimitRangeV1_basic(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelFour", "four"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.%", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "512M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default_request.%", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default_request.cpu", "100m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default_request.memory", "256M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			// Import round-trip: the ID is "namespace/name".
			{
				ResourceName:            limitRangeResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			// Verify import produces a no-op plan (ImportStateVerify cannot see null vs empty).
			{
				Config: testAccKubernetesLimitRangeV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_metaModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "512M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_specModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "1024M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.%", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.cpu", "500m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.%", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.cpu", "10m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.memory", "10M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Container"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_empty(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_empty(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.#", "0"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_generatedName(t *testing.T) {
	var conf corev1api.LimitRange
	prefix := "tf-acc-test-"
	ns := fmt.Sprintf("%s-%s", prefix, acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_generatedName(prefix, ns),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(limitRangeResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Pod"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_typeChange(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_typeChange(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "1024M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Container"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_typeChangeModified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.memory", "1024M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Pod"),
				),
			},
		},
	})
}

func TestAccKubernetesLimitRangeV1_multipleLimits(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_multipleLimits(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "3"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.cpu", "200m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.memory", "1024M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.type", "Pod"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.1.min.storage", "24M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.1.type", "PersistentVolumeClaim"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.2.default.cpu", "50m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.2.default.memory", "24M"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.2.type", "Container"),
				),
			},
		},
	})
}

// Identity import: an `import { identity = {...} }` block carries no ID string, so
// ImportState must build the ID from the identity.
func TestAccKubernetesLimitRangeV1_identityImport(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_basic(name),
				Check:  testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(limitRangeResourceName, map[string]knownvalue.Check{
						"namespace":   knownvalue.StringExact(name),
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("v1"),
						"kind":        knownvalue.StringExact("LimitRange"),
					}),
				},
			},
			{
				// Import blocks already assert a no-op plan and unchanged identity;
				// plugin-testing only supports ImportStateVerify for CLI imports.
				ResourceName:    limitRangeResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// ── write-back regressions (live-cluster findings) ───────────────────────────

// The API server canonicalises quantities ("1000m" -> "1", "1024Mi" -> "1Gi"). The
// planned value must survive Create (otherwise Terraform rejects the apply as an
// inconsistent result). Refresh then reports the canonical form, which is the same
// perpetual diff SDKv2 produced; making Read quantity-aware would be a separate,
// approved behaviour change, so this test pins parity with ExpectNonEmptyPlan.
func TestAccKubernetesLimitRangeV1_nonCanonicalQuantities(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_nonCanonical(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "1000m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.memory", "1024Mi"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.cpu", "1000m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.memory", "1024Mi"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				// The follow-up plan must be an in-place update, never a replacement.
				Config: testAccKubernetesLimitRangeV1Config_nonCanonical(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(limitRangeResourceName, plancheck.ResourceActionUpdate),
					},
				},
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// A Container limit with max/min but no default: the API server back-fills default
// from max. default is Optional (not Computed), so the apply must still succeed with
// the planned null. SDKv2 reported the back-filled value as drift on the next plan;
// that behaviour is preserved, hence ExpectNonEmptyPlan.
func TestAccKubernetesLimitRangeV1_serverBackfilledDefault(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_maxMinOnly(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.max.cpu", "2"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.min.cpu", "100m"),
					resource.TestCheckNoResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.%"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// `spec {}` (a block with no limit) is a configured block; state must keep it.
func TestAccKubernetesLimitRangeV1_emptySpecBlock(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_emptySpecBlock(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.#", "0"),
				),
			},
			{
				Config: testAccKubernetesLimitRangeV1Config_emptySpecBlock(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// default_request is Optional+Computed. Changing only default (or only a label) must
// not re-plan default_request as unknown, and must not change its value on the wire:
// SDKv2 re-sent the prior default_request, so the server never back-filled it again.
func TestAccKubernetesLimitRangeV1_defaultRequestStableOnUpdate(t *testing.T) {
	var conf corev1api.LimitRange
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	defaultRequestPath := tfjsonpath.New("spec").AtSliceIndex(0).AtMapKey("limit").
		AtSliceIndex(0).AtMapKey("default_request")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckKubernetesLimitRangeV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesLimitRangeV1Config_defaultOnly(name, "one", "500m"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default_request.cpu", "500m"),
				),
			},
			{
				// Label-only change: default_request must stay known in the plan.
				Config: testAccKubernetesLimitRangeV1Config_defaultOnly(name, "two", "500m"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectKnownValue(limitRangeResourceName, defaultRequestPath,
							knownvalue.MapExact(map[string]knownvalue.Check{
								"cpu": knownvalue.StringExact("500m"),
							})),
					},
				},
			},
			{
				// default-only change: default_request keeps its prior value (SDKv2 parity).
				Config: testAccKubernetesLimitRangeV1Config_defaultOnly(name, "two", "750m"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesLimitRangeV1Exists(limitRangeResourceName, &conf),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default.cpu", "750m"),
					resource.TestCheckResourceAttr(limitRangeResourceName, "spec.0.limit.0.default_request.cpu", "500m"),
					func(*terraform.State) error {
						got := conf.Spec.Limits[0].DefaultRequest.Cpu().String()
						if got != "500m" {
							return fmt.Errorf("wire defaultRequest.cpu = %q, want %q", got, "500m")
						}
						return nil
					},
				),
			},
		},
	})
}

// ── CheckDestroy + Exists helpers ────────────────────────────────────────────

func testAccCheckKubernetesLimitRangeV1Destroy(s *terraform.State) error {
	conn, err := mainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "kubernetes_limit_range_v1" {
			continue
		}
		ns, name, err := splitID(rs.Primary.ID)
		if err != nil {
			return err
		}
		resp, err := conn.CoreV1().LimitRanges(ns).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			if resp.Namespace == ns && resp.Name == name {
				return fmt.Errorf("LimitRange still exists: %s", rs.Primary.ID)
			}
		}
	}
	return nil
}

func testAccCheckKubernetesLimitRangeV1Exists(n string, obj *corev1api.LimitRange) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		conn, err := mainClientset()
		if err != nil {
			return err
		}
		ns, name, err := splitID(rs.Primary.ID)
		if err != nil {
			return err
		}
		out, err := conn.CoreV1().LimitRanges(ns).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		*obj = *out
		return nil
	}
}

// ── HCL configs ──────────────────────────────────────────────────────────────

func testAccKubernetesLimitRangeV1Config_empty(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "512M"
      }

      default_request = {
        cpu    = "100m"
        memory = "256M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_metaModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
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

    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "512M"
      }

      default_request = {
        cpu    = "100m"
        memory = "256M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_specModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "1024M"
      }

      max = {
        cpu = "500m"
      }

      min = {
        cpu    = "10m"
        memory = "10M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_generatedName(prefix, ns string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    generate_name = %[2]q
    namespace     = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Pod"
    }
  }
}
`, ns, prefix)
}

func testAccKubernetesLimitRangeV1Config_typeChange(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"

      default = {
        cpu    = "200m"
        memory = "1024M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_typeChangeModified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Pod"

      min = {
        cpu    = "200m"
        memory = "1024M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_multipleLimits(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Pod"

      max = {
        cpu    = "200m"
        memory = "1024M"
      }
    }

    limit {
      type = "PersistentVolumeClaim"

      min = {
        storage = "24M"
      }
    }

    limit {
      type = "Container"

      default = {
        cpu    = "50m"
        memory = "24M"
      }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_nonCanonical(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type    = "Container"
      default = { cpu = "1000m", memory = "1024Mi" }
      max     = { cpu = "1000m", memory = "1024Mi" }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_maxMinOnly(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {
    limit {
      type = "Container"
      max  = { cpu = "2" }
      min  = { cpu = "100m" }
    }
  }
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_emptySpecBlock(name string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
  }

  spec {}
}
`, name)
}

func testAccKubernetesLimitRangeV1Config_defaultOnly(name, label, cpu string) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %[1]q
  }
}

resource "kubernetes_limit_range_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = kubernetes_namespace_v1.test.metadata.0.name
    labels    = { app = %[2]q }
  }

  spec {
    limit {
      type    = "Container"
      default = { cpu = %[3]q }
    }
  }
}
`, name, label, cpu)
}
