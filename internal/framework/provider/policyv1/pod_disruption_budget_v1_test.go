// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sdkterraform "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	policy "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestAccKubernetesPodDisruptionBudgetV1_basic(t *testing.T) {
	var conf policy.PodDisruptionBudget
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_pod_disruption_budget_v1.test"
	p := kubernetes.Provider()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { pdbPreCheck(t, p) },
		ProtoV6ProviderFactories: pdbFactories(nil),
		CheckDestroy:             testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p),
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPodDisruptionBudgetV1Config_maxUnavailable(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetV1Exists(p, resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_unavailable", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_available", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_labels.foo", "bar"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_expressions.#", "0"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesPodDisruptionBudgetV1Config_minAvailable(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesPodDisruptionBudgetV1Exists(p, resourceName, &conf),
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
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.max_unavailable", ""),
					resource.TestCheckResourceAttr(resourceName, "spec.0.min_available", "75%"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_expressions.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_expressions.0.key", "name"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_expressions.0.values.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_expressions.0.values.1", "foo"),
					resource.TestCheckResourceAttr(resourceName, "spec.0.selector.match_expressions.0.values.0", "apps")),
			},
		},
	})
}

func testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p *schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()

		if err != nil {
			return err
		}
		ctx := context.TODO()

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "kubernetes_pod_disruption_budget_v1" {
				continue
			}

			namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
			if err != nil {
				return err
			}

			resp, err := conn.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				if resp.Namespace == namespace && resp.Name == name {
					return fmt.Errorf("Pod Disruption Budget still exists: %s", rs.Primary.ID)
				}
			}
			if err != nil && !apierrors.IsNotFound(err) {
				return err
			}
		}

		return nil
	}
}

func testAccCheckKubernetesPodDisruptionBudgetV1Exists(p *schema.Provider, n string, obj *policy.PodDisruptionBudget) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		ctx := context.TODO()

		namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
		if err != nil {
			return err
		}

		out, err := conn.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		*obj = *out
		return nil
	}
}

func testAccKubernetesPodDisruptionBudgetV1Config_maxUnavailable(name string) string {
	return fmt.Sprintf(`resource "kubernetes_pod_disruption_budget_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name      = "%s"
    namespace = %q
  }

  spec {
    max_unavailable = 1
    selector = {
      match_labels = {
        foo = "bar"
      }
    }
  }
}
`, name, pdbNamespace())
}

func testAccKubernetesPodDisruptionBudgetV1Config_minAvailable(name string) string {
	// Note the percent sign in min_available is golang-escaped to be double percent signs
	return fmt.Sprintf(`resource "kubernetes_pod_disruption_budget_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name      = "%s"
    namespace = %q
  }

  spec {
    min_available = "75%%"
    selector = {
      match_expressions = [{
        key      = "name"
        operator = "In"
        values   = ["foo", "apps"]
      }]
    }
  }
}
`, name, pdbNamespace())
}

const pdbAddress = "kubernetes_pod_disruption_budget_v1.test"

func TestAccPodDisruptionBudgetV1_Upgrade(t *testing.T) {
	for _, version := range []string{"3.3.0", "2.37.1"} {
		for _, variant := range []string{
			"minimal", "full", "zero", "zero_percent", "min_integer", "max_percent",
			"empty_threshold", "empty_selector", "empty_match_labels", "exists",
			"empty_values", "empty_metadata", "generated", "unset_thresholds", "empty_thresholds", "both_names",
		} {
			t.Run(version+"/"+variant, func(t *testing.T) {
				p := kubernetes.Provider()
				name := "tf-pdb-" + acctest.RandString(10)
				config := pdbConfig(name, variant)
				var before policy.PodDisruptionBudget
				var snapshot pdbStateSnapshot
				var writes pdbWriteTracker
				resource.ParallelTest(t, resource.TestCase{
					PreCheck:     func() { pdbPreCheck(t, p) },
					CheckDestroy: testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p),
					Steps: []resource.TestStep{
						{
							ExternalProviders: map[string]resource.ExternalProvider{
								"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: version},
							},
							Config: pdbLegacyConfig(name, variant),
							Check:  pdbCheckRemote(p, &before, true),
							ConfigStateChecks: []statecheck.StateCheck{
								pdbStateCheck{snapshot: &snapshot},
							},
						},
						{
							ProtoV6ProviderFactories: pdbFactories(&writes),
							Config:                   config,
							ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
								pdbUpgradePlan{variant: variant},
							}},
							Check: pdbCheckRemote(p, &before, false),
							ConfigStateChecks: []statecheck.StateCheck{
								pdbStateCheck{snapshot: &snapshot, compare: true, variant: variant}, &writes,
							},
						},
						{
							ProtoV6ProviderFactories: pdbFactories(&writes),
							Config:                   config,
							ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(),
							}},
							Check: pdbCheckRemote(p, &before, false),
							ConfigStateChecks: []statecheck.StateCheck{
								pdbStateCheck{snapshot: &snapshot, compare: true, variant: variant}, &writes,
							},
						},
					},
				})
			})
		}
	}
}

func TestAccPodDisruptionBudgetV1_Metadata(t *testing.T) {
	for _, variant := range []string{"full", "generated"} {
		t.Run(variant, func(t *testing.T) {
			p := kubernetes.Provider()
			config := pdbConfig("tf-pdb-"+acctest.RandString(10), variant)
			updated := strings.Replace(config, `"owner" = "terraform"`, `"owner" = "edited"`, 1)
			removed := strings.Replace(updated, `annotations = { "owner" = "edited" }`, `annotations = {}`, 1)
			var before policy.PodDisruptionBudget
			check := func(annotation string) resource.TestCheckFunc {
				return func(state *terraform.State) error {
					expected := *before.DeepCopy()
					expected.Annotations = map[string]string{}
					if annotation != "" {
						expected.Annotations["owner"] = annotation
					}
					return pdbCheckRemote(p, &expected, false)(state)
				}
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { pdbPreCheck(t, p) },
				ProtoV6ProviderFactories: pdbFactories(nil),
				CheckDestroy:             testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p),
				Steps: []resource.TestStep{
					{Config: config, Check: pdbCheckRemote(p, &before, true)},
					{
						Config: updated,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(pdbAddress, plancheck.ResourceActionUpdate),
						}},
						Check: check("edited"),
					},
					{
						Config: removed,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(pdbAddress, plancheck.ResourceActionUpdate),
						}},
						Check: check(""),
					},
					{
						Config: removed,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						}},
						Check: check(""),
					},
				},
			})
		})
	}
}

func TestAccPodDisruptionBudgetV1_Import(t *testing.T) {
	for _, variant := range []string{"minimal", "full", "empty_selector", "exists", "generated"} {
		t.Run(variant, func(t *testing.T) {
			p := kubernetes.Provider()
			name := "tf-pdb-" + acctest.RandString(10)
			config := pdbConfig(name, variant)
			var before policy.PodDisruptionBudget
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { pdbPreCheck(t, p) },
				ProtoV6ProviderFactories: pdbFactories(nil),
				CheckDestroy:             testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p),
				Steps: []resource.TestStep{
					{
						PreConfig: func() {
							client, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
							if err != nil {
								t.Fatal(err)
							}
							budget := pdbImportFixture(name, variant)
							out, err := client.PolicyV1().PodDisruptionBudgets(pdbNamespace()).Create(context.Background(), budget, metav1.CreateOptions{})
							if err != nil {
								t.Fatal(err)
							}
							before = *out.DeepCopy()
							t.Cleanup(func() {
								err := client.PolicyV1().PodDisruptionBudgets(before.Namespace).Delete(context.Background(), before.Name,
									metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &before.UID}})
								if err != nil && !apierrors.IsNotFound(err) {
									t.Errorf("import fixture cleanup: %s", err)
								}
							})
						},
						Config:       config,
						ResourceName: pdbAddress, ImportState: true, ImportStatePersist: true,
						ImportStateIdFunc: func(*terraform.State) (string, error) { return before.Namespace + "/" + before.Name, nil },
						ImportStateCheck: func(states []*terraform.InstanceState) error {
							if len(states) != 1 || states[0].ID != before.Namespace+"/"+before.Name ||
								states[0].Attributes["metadata.0.uid"] != string(before.UID) {
								return fmt.Errorf("import lost the PDB identity")
							}
							return nil
						},
					},
					{
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						}},
						Check: pdbCheckRemote(p, &before, false),
					},
				},
			})
		})
	}
}

func pdbImportFixture(name, variant string) *policy.PodDisruptionBudget {
	max := intstr.FromInt32(1)
	out := &policy.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: pdbNamespace()},
		Spec: policy.PodDisruptionBudgetSpec{
			MaxUnavailable: &max,
			Selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": "pdb-regression"}},
		},
	}
	switch variant {
	case "full", "generated":
		min := intstr.FromString("75%")
		out.Spec.MinAvailable, out.Spec.MaxUnavailable = &min, nil
		out.Annotations, out.Labels = map[string]string{"owner": "terraform"}, map[string]string{"suite": "pdb"}
		out.Spec.Selector.MatchExpressions = []metav1.LabelSelectorRequirement{
			{Key: "tier", Operator: metav1.LabelSelectorOpIn, Values: []string{"backend", "api"}},
		}
		if variant == "generated" {
			out.GenerateName, out.Name = name, ""
		}
	case "empty_selector":
		out.Spec.Selector = &metav1.LabelSelector{}
	case "exists":
		out.Spec.Selector = &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "tier", Operator: metav1.LabelSelectorOpExists},
		}}
	}
	return out
}

func TestAccPodDisruptionBudgetV1_Replacement(t *testing.T) {
	for _, edit := range []struct{ name, old, new string }{
		{"threshold", `min_available = "75%"`, `min_available = "50%"`},
		{"threshold_mode", `min_available = "75%"`, `max_unavailable = "1"`},
		{"match_labels", `app = "pdb-regression"`, `app = "changed"`},
		{"remove_match_labels", `match_labels = { app = "pdb-regression" }`, `match_labels = {}`},
		{"expression_key", `key      = "tier"`, `key      = "role"`},
		{"expression_operator", `operator = "In"`, `operator = "NotIn"`},
		{"expression_values", `values   = ["backend", "api"]`, `values   = ["worker"]`},
		{"remove_expressions", pdbExpression(), ""},
	} {
		t.Run(edit.name, func(t *testing.T) {
			p := kubernetes.Provider()
			config := pdbConfig("tf-pdb-"+acctest.RandString(10), "full")
			if strings.Count(config, edit.old) != 1 {
				t.Fatal("replacement test fixture must contain exactly one edit target")
			}
			updated := strings.Replace(config, edit.old, edit.new, 1)
			var before, after policy.PodDisruptionBudget
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { pdbPreCheck(t, p) },
				ProtoV6ProviderFactories: pdbFactories(nil),
				CheckDestroy:             testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p),
				Steps: []resource.TestStep{
					{Config: config, Check: pdbCheckRemote(p, &before, true)},
					{
						Config: updated,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(pdbAddress, plancheck.ResourceActionDestroyBeforeCreate),
						}},
						Check: resource.ComposeAggregateTestCheckFunc(
							pdbCheckRemote(p, &after, true),
							func(*terraform.State) error {
								if after.UID == before.UID {
									return fmt.Errorf("intentional spec replacement retained UID %s", before.UID)
								}
								return nil
							},
						),
					},
					{
						Config: updated,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						}},
						Check: pdbCheckRemote(p, &after, false),
					},
				},
			})
		})
	}
}

func TestAccPodDisruptionBudgetV1_DriftAndDisappears(t *testing.T) {
	for _, action := range []string{"metadata", "spec", "delete"} {
		t.Run(action, func(t *testing.T) {
			p := kubernetes.Provider()
			config := pdbConfig("tf-pdb-"+acctest.RandString(10), "full")
			var before policy.PodDisruptionBudget
			expectedAction := plancheck.ResourceActionUpdate
			switch action {
			case "spec":
				expectedAction = plancheck.ResourceActionDestroyBeforeCreate
			case "delete":
				expectedAction = plancheck.ResourceActionCreate
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { pdbPreCheck(t, p) },
				ProtoV6ProviderFactories: pdbFactories(nil),
				CheckDestroy:             testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p),
				Steps: []resource.TestStep{
					{Config: config, Check: pdbCheckRemote(p, &before, true)},
					{
						PreConfig: func() {
							client, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
							if err != nil {
								t.Fatal(err)
							}
							if action == "delete" {
								err = client.PolicyV1().PodDisruptionBudgets(before.Namespace).Delete(context.Background(), before.Name,
									metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &before.UID}})
							} else {
								path, value := "/metadata/annotations/owner", "external"
								if action == "spec" {
									path, value = "/spec/minAvailable", "50%"
								}
								err = pdbPatch(p, &before, path, value)
							}
							if err != nil {
								t.Fatal(err)
							}
						},
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(pdbAddress, expectedAction),
						}},
						Check: func(state *terraform.State) error {
							var got policy.PodDisruptionBudget
							if err := pdbCheckRemote(p, &got, true)(state); err != nil {
								return err
							}
							if action == "metadata" && got.UID != before.UID {
								return fmt.Errorf("metadata drift recreated the PDB")
							}
							if action != "metadata" && got.UID == before.UID {
								return fmt.Errorf("replacement did not create a new PDB")
							}
							expected := *before.DeepCopy()
							expected.UID = got.UID
							if err := pdbCompareRemote(expected, got); err != nil {
								return err
							}
							before = got
							return nil
						},
					},
					{
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						}},
						Check: pdbCheckRemote(p, &before, false),
					},
				},
			})
		})
	}
}

func TestAccPodDisruptionBudgetV1_AddressMove(t *testing.T) {
	p := kubernetes.Provider()
	name := "tf-pdb-" + acctest.RandString(10)
	config := pdbConfig(name, "full")
	moved := strings.ReplaceAll(config, `"kubernetes_pod_disruption_budget_v1" "test"`, `"kubernetes_pod_disruption_budget_v1" "renamed"`)
	moved = strings.ReplaceAll(moved, "kubernetes_pod_disruption_budget_v1.test", "kubernetes_pod_disruption_budget_v1.renamed")
	moved += `
moved {
  from = kubernetes_pod_disruption_budget_v1.test
  to   = kubernetes_pod_disruption_budget_v1.renamed
}
`
	var before policy.PodDisruptionBudget
	var writes pdbWriteTracker
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { pdbPreCheck(t, p) },
		CheckDestroy: testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p),
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "3.3.0"},
				},
				Config: pdbLegacyConfig(name, "full"), Check: pdbCheckRemote(p, &before, true),
			},
			{
				ProtoV6ProviderFactories: pdbFactories(&writes),
				Config:                   moved,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction("kubernetes_pod_disruption_budget_v1.renamed", plancheck.ResourceActionNoop),
				}},
				Check: pdbCheckRemote(p, &before, false), ConfigStateChecks: []statecheck.StateCheck{&writes},
			},
		},
	})
}

func TestAccPodDisruptionBudgetV1_IgnoredMetadata(t *testing.T) {
	p := kubernetes.Provider()
	config := `
provider "kubernetes" {
  ignore_annotations = ["^controller.example.com/"]
  ignore_labels      = ["^controller.example.com/"]
}
` + pdbConfig("tf-pdb-"+acctest.RandString(10), "full")
	updated := strings.Replace(config, `"owner" = "terraform"`, `"owner" = "edited"`, 1)
	removed := strings.Replace(updated, `annotations = { "owner" = "edited" }`, `annotations = {}`, 1)
	var before policy.PodDisruptionBudget
	var writes pdbWriteTracker
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     func() { pdbPreCheck(t, p) },
		CheckDestroy: testAccCheckKubernetesPodDisruptionBudgetV1Destroy(p),
		Steps: []resource.TestStep{
			{ProtoV6ProviderFactories: pdbFactories(nil), Config: config, Check: pdbCheckRemote(p, &before, true)},
			{
				PreConfig: func() {
					before.Annotations["controller.example.com/owner"] = "controller"
					before.Labels["controller.example.com/owner"] = "controller"
					evictionPolicy := policy.AlwaysAllow
					before.Spec.UnhealthyPodEvictionPolicy = &evictionPolicy
					if err := pdbPatch(p, &before, "/spec/unhealthyPodEvictionPolicy", evictionPolicy); err != nil {
						t.Fatal(err)
					}
					for _, item := range []struct {
						path  string
						value map[string]string
					}{
						{"/metadata/annotations", before.Annotations}, {"/metadata/labels", before.Labels},
					} {
						if err := pdbPatch(p, &before, item.path, item.value); err != nil {
							t.Fatal(err)
						}
					}
				},
				ProtoV6ProviderFactories: pdbFactories(&writes), Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:            pdbCheckRemote(p, &before, false), ConfigStateChecks: []statecheck.StateCheck{&writes},
			},
			{
				ProtoV6ProviderFactories: pdbFactories(nil), Config: updated,
				PreConfig: func() { before.Annotations["owner"] = "edited" },
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(pdbAddress, plancheck.ResourceActionUpdate),
				}},
				Check: pdbCheckRemote(p, &before, false),
			},
			{
				ProtoV6ProviderFactories: pdbFactories(nil), Config: removed,
				PreConfig: func() { delete(before.Annotations, "owner") },
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(pdbAddress, plancheck.ResourceActionUpdate),
				}},
				Check: pdbCheckRemote(p, &before, false),
			},
			{
				ProtoV6ProviderFactories: pdbFactories(nil), Config: removed,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check:            pdbCheckRemote(p, &before, false),
			},
		},
	})
}

func TestAccPodDisruptionBudgetV1_Forbidden(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("TF_ACC=1 is required for live authorization tests")
	}
	for _, operation := range []string{"create", "read"} {
		t.Run(operation, func(t *testing.T) {
			p := kubernetes.Provider()
			pdbPreCheck(t, p)
			name := "tf-pdb-" + acctest.RandString(10)
			config := pdbConfig(name, "full")
			loading := clientcmd.NewDefaultClientConfigLoadingRules()
			loading.ExplicitPath = os.Getenv("KUBE_CONFIG_PATH")
			raw, err := loading.Load()
			if err != nil {
				t.Fatal(err)
			}
			if selected := os.Getenv("KUBE_CTX"); selected != "" {
				raw.CurrentContext = selected
			}
			if err := clientcmdapi.FlattenConfig(raw); err != nil {
				t.Fatal(err)
			}
			contextConfig := raw.Contexts[raw.CurrentContext]
			if contextConfig == nil || raw.AuthInfos[contextConfig.AuthInfo] == nil {
				t.Fatal("live forbidden tests require explicit kubeconfig credentials")
			}
			raw.AuthInfos[contextConfig.AuthInfo].Impersonate = "pdb-regression-" + acctest.RandString(10)
			restricted := filepath.Join(t.TempDir(), "restricted.kubeconfig")
			if err := clientcmd.WriteToFile(*raw, restricted); err != nil {
				t.Fatal(err)
			}
			restrictedConfig := fmt.Sprintf(`
provider "kubernetes" {
  config_path = %q
}
`, restricted) + config
			var before policy.PodDisruptionBudget
			var steps []resource.TestStep
			if operation == "read" {
				steps = append(steps, resource.TestStep{Config: config, Check: pdbCheckRemote(p, &before, true)})
			}
			verb := operation
			if operation == "read" {
				verb = "get"
			}
			steps = append(steps, resource.TestStep{
				Config:      restrictedConfig,
				ExpectError: regexp.MustCompile(`(?s)is forbidden:.*cannot ` + verb + ` resource "poddisruptionbudgets"`),
			})
			if operation == "read" {
				steps = append(steps, resource.TestStep{
					Config:           config,
					ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					Check:            pdbCheckRemote(p, &before, false),
				})
			}
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: pdbFactories(nil),
				CheckDestroy: func(*terraform.State) error {
					client, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
					if err != nil {
						return err
					}
					_, err = client.PolicyV1().PodDisruptionBudgets(pdbNamespace()).Get(context.Background(), name, metav1.GetOptions{})
					if apierrors.IsNotFound(err) {
						return nil
					}
					if err != nil {
						return err
					}
					return fmt.Errorf("PDB %s remains after forbidden test", name)
				},
				Steps: steps,
			})
		})
	}
}

func TestPodDisruptionBudgetV1MuxRegistration(t *testing.T) {
	ctx := context.Background()
	sdk := kubernetes.Provider()
	if _, ok := sdk.ResourcesMap["kubernetes_pod_disruption_budget_v1"]; ok {
		t.Fatal("PDB v1 must not remain registered on SDKv2")
	}
	if _, ok := sdk.ResourcesMap["kubernetes_pod_disruption_budget"]; !ok {
		t.Fatal("deprecated policy/v1beta1 resource was removed")
	}
	server, err := mux.MuxServerWithProvider(ctx, "test", sdk)
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range response.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatal(d.Summary, d.Detail)
		}
	}
	s, ok := response.ResourceSchemas["kubernetes_pod_disruption_budget_v1"]
	if !ok || s.Version != 1 {
		t.Fatal("mux is missing PDB schema version one")
	}
	for _, name := range []string{"kubernetes_namespace", "kubernetes_namespace_v1", "kubernetes_manifest", "kubernetes_pod_disruption_budget"} {
		if response.ResourceSchemas[name] == nil {
			t.Fatalf("mux lost sibling resource %s", name)
		}
	}
}

func TestPodDisruptionBudgetV1UpgradePlanGuard(t *testing.T) {
	for _, mutation := range []string{"allowed", "replace", "lost_id", "threshold", "selector", "metadata", "output", "unknown_uid", "extra_resource"} {
		t.Run(mutation, func(t *testing.T) {
			before := map[string]any{
				"id": "test/pdb",
				"metadata": []any{map[string]any{
					"name": "pdb", "namespace": "test", "uid": "uid-1",
					"generation": float64(1), "resource_version": "10",
					"labels": nil, "annotations": nil, "generate_name": nil,
				}},
				"spec": []any{map[string]any{
					"min_available": "", "max_unavailable": "1",
					"selector": map[string]any{"match_labels": nil, "match_expressions": []any{}},
				}},
			}
			raw, err := json.Marshal(before)
			if err != nil {
				t.Fatal(err)
			}
			var after map[string]any
			if err := json.Unmarshal(raw, &after); err != nil {
				t.Fatal(err)
			}
			for _, path := range pdbNormalizationPaths("empty_match_labels") {
				if err := pdbNormalizeEmpty(after, path); err != nil {
					t.Fatal(err)
				}
			}
			metadata := after["metadata"].([]any)[0].(map[string]any)
			delete(metadata, "generation")
			delete(metadata, "resource_version")
			change := &tfjson.ResourceChange{Address: pdbAddress, Change: &tfjson.Change{
				Actions: tfjson.Actions{tfjson.ActionUpdate}, Before: before, After: after,
				AfterUnknown: map[string]any{"metadata": []any{map[string]any{"generation": true, "resource_version": true}}},
			}}
			plan := &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{change}}
			spec := after["spec"].([]any)[0].(map[string]any)
			switch mutation {
			case "replace":
				change.Change.Actions = tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
			case "lost_id":
				delete(after, "id")
			case "threshold":
				spec["max_unavailable"] = "2"
			case "selector":
				spec["selector"].(map[string]any)["match_labels"] = map[string]any{"new": "value"}
			case "metadata":
				metadata["labels"] = map[string]any{"new": "value"}
			case "output":
				plan.OutputChanges = map[string]*tfjson.Change{"value": {Actions: tfjson.Actions{tfjson.ActionUpdate}}}
			case "unknown_uid":
				delete(metadata, "uid")
			case "extra_resource":
				plan.ResourceChanges = append(plan.ResourceChanges, &tfjson.ResourceChange{Address: "unexpected"})
			}
			var response plancheck.CheckPlanResponse
			pdbUpgradePlan{variant: "empty_match_labels"}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
			if (response.Error != nil) != (mutation != "allowed") {
				t.Fatalf("plan guard result for %s: %v", mutation, response.Error)
			}
		})
	}
}

// Keep the released-provider step on its original block syntax.
func pdbLegacyConfig(name, variant string) string {
	config := pdbConfig(name, variant)
	config = strings.ReplaceAll(config, "selector = {", "selector {")
	config = strings.ReplaceAll(config, "match_expressions = [{", "match_expressions {")
	return strings.ReplaceAll(config, "      }]", "      }")
}

func pdbConfig(name, variant string) string {
	metadata := fmt.Sprintf("name = %q", name)
	threshold := `max_unavailable = "1"`
	selector := `match_labels = { app = "pdb-regression" }`
	switch variant {
	case "full", "generated":
		metadata += `
    annotations = { "owner" = "terraform" }
    labels      = { "suite" = "pdb" }
`
		threshold = `min_available = "75%"`
		selector += pdbExpression()
		if variant == "generated" {
			metadata = strings.Replace(metadata, "name =", "generate_name =", 1)
		}
	case "both_names":
		metadata += "\n    generate_name = \"budget-\""
	case "zero":
		threshold = `min_available = "0"`
	case "zero_percent":
		threshold = `max_unavailable = "0%"`
	case "min_integer":
		threshold = `min_available = "1"`
	case "max_percent":
		threshold = `max_unavailable = "100%"`
	case "empty_threshold":
		threshold += "\n    min_available = \"\""
	case "unset_thresholds":
		threshold = ""
	case "empty_thresholds":
		threshold = "min_available = \"\"\n    max_unavailable = \"\""
	case "empty_selector":
		selector = ""
	case "empty_match_labels":
		selector = "match_labels = {}"
	case "exists", "empty_values":
		selector = `
      match_expressions = [{
        key      = "tier"
        operator = "Exists"
      }]
`
		if variant == "empty_values" {
			selector = strings.Replace(selector, `operator = "Exists"`, "operator = \"Exists\"\n        values = []", 1)
		}
	case "empty_metadata":
		metadata += "\n    annotations = {}\n    labels = {}"
	}
	return fmt.Sprintf(`resource "kubernetes_pod_disruption_budget_v1" "test" {
  metadata {
    %s
    namespace = %q
  }
  spec {
    %s
    selector = {
      %s
    }
  }
}
output "thresholds" {
  value = %s
}
`, metadata, pdbNamespace(), threshold, selector, pdbThresholdOutput(threshold))
}

func pdbThresholdOutput(threshold string) string {
	if threshold == "" {
		return "kubernetes_pod_disruption_budget_v1.test.id"
	}
	if strings.Contains(threshold, "max_unavailable") {
		return "kubernetes_pod_disruption_budget_v1.test.spec[0].max_unavailable"
	}
	return "kubernetes_pod_disruption_budget_v1.test.spec[0].min_available"
}

func pdbExpression() string {
	return `
      match_expressions = [{
        key      = "tier"
        operator = "In"
        values   = ["backend", "api"]
      }]
`
}

func pdbNamespace() string {
	if namespace := os.Getenv("KUBE_PDB_TEST_NAMESPACE"); namespace != "" {
		return namespace
	}
	return "default"
}

func pdbPreCheck(t *testing.T, p *schema.Provider) {
	t.Helper()
	if os.Getenv("KUBE_CONFIG_PATH") == "" && os.Getenv("KUBE_HOST") == "" && os.Getenv("KUBE_CTX") == "" {
		t.Fatal("explicit Kubernetes acceptance-test configuration is required")
	}
	if diags := p.Configure(context.Background(), sdkterraform.NewResourceConfigRaw(nil)); diags.HasError() {
		t.Fatal(diags)
	}
}

func pdbCheckRemote(p *schema.Provider, snapshot *policy.PodDisruptionBudget, capture bool) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		var got policy.PodDisruptionBudget
		found := false
		for address, r := range state.RootModule().Resources {
			if r.Type != "kubernetes_pod_disruption_budget_v1" {
				continue
			}
			if found {
				return fmt.Errorf("expected one PDB in state")
			}
			found = true
			if err := testAccCheckKubernetesPodDisruptionBudgetV1Exists(p, address, &got)(state); err != nil {
				return err
			}
			if got.UID == "" || r.Primary.Attributes["metadata.0.uid"] != string(got.UID) {
				return fmt.Errorf("state and API UID differ")
			}
		}
		if !found {
			return fmt.Errorf("PDB missing from state")
		}
		if capture {
			*snapshot = *got.DeepCopy()
			return nil
		}
		return pdbCompareRemote(*snapshot, got)
	}
}

func pdbCompareRemote(before, after policy.PodDisruptionBudget) error {
	type comparable struct {
		UID                           k8stypes.UID
		Name, Namespace, GenerateName string
		Labels, Annotations           map[string]string
		Spec                          policy.PodDisruptionBudgetSpec
	}
	snapshot := func(p policy.PodDisruptionBudget) comparable {
		labels, annotations := maps.Clone(p.Labels), maps.Clone(p.Annotations)
		if len(labels) == 0 {
			labels = nil
		}
		if len(annotations) == 0 {
			annotations = nil
		}
		return comparable{p.UID, p.Name, p.Namespace, p.GenerateName, labels, annotations, p.Spec}
	}
	if diff := cmp.Diff(snapshot(before), snapshot(after)); diff != "" {
		return fmt.Errorf("remote UID or desired settings changed:\n%s", diff)
	}
	return nil
}

func pdbPatch(p *schema.Provider, before *policy.PodDisruptionBudget, path string, value any) error {
	client, err := p.Meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	patch, err := json.Marshal([]map[string]any{
		{"op": "test", "path": "/metadata/uid", "value": before.UID},
		{"op": "add", "path": path, "value": value},
	})
	if err != nil {
		return err
	}
	_, err = client.PolicyV1().PodDisruptionBudgets(before.Namespace).Patch(context.Background(), before.Name, k8stypes.JSONPatchType, patch, metav1.PatchOptions{})
	return err
}

type pdbStateSnapshot struct {
	values   map[string]any
	outputs  map[string]*tfjson.StateOutput
	provider string
}

type pdbStateCheck struct {
	snapshot *pdbStateSnapshot
	compare  bool
	variant  string
}

func (c pdbStateCheck) CheckState(_ context.Context, req statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	if req.State == nil || req.State.Values == nil || req.State.Values.RootModule == nil {
		resp.Error = fmt.Errorf("missing state")
		return
	}
	var found *tfjson.StateResource
	for _, r := range req.State.Values.RootModule.Resources {
		if r.Address == pdbAddress {
			found = r
		}
	}
	wantVersion := uint64(0)
	if c.compare {
		wantVersion = 1
	}
	if found == nil || found.SchemaVersion != wantVersion || len(found.IdentityValues) != 0 || found.IdentitySchemaVersion != nil {
		resp.Error = fmt.Errorf("PDB absent or schema/identity contract changed: %#v", found)
		return
	}
	rawValues, err := json.Marshal(found.AttributeValues)
	if err != nil {
		resp.Error = err
		return
	}
	var values map[string]any
	if err := json.Unmarshal(rawValues, &values); err != nil {
		resp.Error = err
		return
	}
	metadata, ok := values["metadata"].([]any)
	if !ok || len(metadata) != 1 {
		resp.Error = fmt.Errorf("invalid metadata in state")
		return
	}
	fields, ok := metadata[0].(map[string]any)
	if !ok {
		resp.Error = fmt.Errorf("invalid metadata fields")
		return
	}
	// resource_version also changes on asynchronous PDB controller status writes.
	delete(fields, "resource_version")
	if fields["generate_name"] == "" {
		fields["generate_name"] = nil
	}
	if !c.compare {
		// Compare the approved selector shape change separately from its values.
		spec := values["spec"].([]any)[0].(map[string]any)
		selectors, ok := spec["selector"].([]any)
		if !ok || len(selectors) != 1 {
			resp.Error = fmt.Errorf("invalid released selector state")
			return
		}
		spec["selector"] = selectors[0]
	}
	got := pdbStateSnapshot{values, req.State.Values.Outputs, found.ProviderName}
	if !c.compare {
		*c.snapshot = got
		return
	}
	expected := *c.snapshot
	if paths := pdbNormalizationPaths(c.variant); len(paths) != 0 {
		raw, err := json.Marshal(expected.values)
		if err != nil {
			resp.Error = err
			return
		}
		var values map[string]any
		if err := json.Unmarshal(raw, &values); err != nil {
			resp.Error = err
			return
		}
		expected.values = values
		for _, path := range paths {
			if err := pdbNormalizeEmpty(expected.values, path); err != nil {
				resp.Error = err
				return
			}
		}
	}
	if !reflect.DeepEqual(expected, got) {
		resp.Error = fmt.Errorf("migration changed complete state/outputs/provider:\nvalues:%s\noutputs:%s\nprovider:%s -> %s",
			cmp.Diff(expected.values, got.values), cmp.Diff(expected.outputs, got.outputs), expected.provider, got.provider)
	}
}

func pdbNormalizationPaths(variant string) [][]any {
	var paths [][]any
	switch variant {
	case "empty_metadata":
		paths = [][]any{{"metadata", 0, "annotations"}, {"metadata", 0, "labels"}}
	case "empty_match_labels":
		paths = [][]any{{"spec", 0, "selector", "match_labels"}}
	case "empty_values":
		paths = [][]any{{"spec", 0, "selector", "match_expressions", 0, "values"}}
	}
	switch variant {
	case "full", "generated", "exists", "empty_values":
	default:
		paths = append(paths, []any{"spec", 0, "selector", "match_expressions"})
	}
	return paths
}

func pdbNormalizeEmpty(root map[string]any, path []any) error {
	var current any = root
	for i, step := range path {
		if index, ok := step.(int); ok {
			list, ok := current.([]any)
			if !ok || index >= len(list) {
				return fmt.Errorf("invalid normalization list at %v", path[:i+1])
			}
			current = list[index]
			continue
		}
		key, ok := step.(string)
		if !ok {
			return fmt.Errorf("invalid normalization path %v", path)
		}
		object, ok := current.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid normalization object at %v", path[:i+1])
		}
		value, exists := object[key]
		if !exists {
			return fmt.Errorf("missing normalization field %v", path)
		}
		if i == len(path)-1 {
			if key == "match_expressions" {
				list, ok := value.([]any)
				if !ok || len(list) != 0 {
					return fmt.Errorf("expected legacy empty expressions at %v, got %#v", path, value)
				}
				object[key] = nil
				return nil
			}
			if value != nil {
				return fmt.Errorf("expected legacy null at %v, got %#v", path, value)
			}
			if key == "values" {
				object[key] = []any{}
			} else {
				object[key] = map[string]any{}
			}
			return nil
		}
		current = value
	}
	return fmt.Errorf("empty normalization path")
}

type pdbUpgradePlan struct{ variant string }

func (c pdbUpgradePlan) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	paths := pdbNormalizationPaths(c.variant)
	if len(paths) == 0 {
		plancheck.ExpectEmptyPlan().CheckPlan(ctx, req, resp)
		return
	}
	if req.Plan == nil || len(req.Plan.ResourceChanges) != 1 {
		resp.Error = fmt.Errorf("expected exactly one state-only PDB change")
		return
	}
	r := req.Plan.ResourceChanges[0]
	if r.Address != pdbAddress || r.Change == nil || !r.Change.Actions.Update() || len(r.Change.ReplacePaths) != 0 {
		resp.Error = fmt.Errorf("unexpected migration action: %#v", r)
		return
	}
	raw, err := json.Marshal(r.Change.Before)
	if err != nil {
		resp.Error = err
		return
	}
	var expected map[string]any
	if err := json.Unmarshal(raw, &expected); err != nil {
		resp.Error = err
		return
	}
	for _, path := range paths {
		if err := pdbNormalizeEmpty(expected, path); err != nil {
			resp.Error = err
			return
		}
	}
	// Only the two server-recomputed metadata fields may become unknown on update.
	unknown, ok := r.Change.AfterUnknown.(map[string]any)
	if !ok {
		resp.Error = fmt.Errorf("invalid unknown-values shape")
		return
	}
	metadataUnknown, ok := unknown["metadata"].([]any)
	if !ok || len(metadataUnknown) != 1 {
		resp.Error = fmt.Errorf("invalid metadata unknown-values shape")
		return
	}
	fields, ok := metadataUnknown[0].(map[string]any)
	if !ok {
		resp.Error = fmt.Errorf("invalid metadata unknown fields")
		return
	}
	metadata := expected["metadata"].([]any)[0].(map[string]any)
	for _, key := range []string{"generation", "resource_version"} {
		if fields[key] != true {
			resp.Error = fmt.Errorf("expected computed %s unknown", key)
			return
		}
		delete(metadata, key)
	}
	if !reflect.DeepEqual(expected, r.Change.After) {
		resp.Error = fmt.Errorf("unexpected changes outside exact null-to-empty paths:\n%s", cmp.Diff(expected, r.Change.After))
		return
	}
	for name, output := range req.Plan.OutputChanges {
		if !output.Actions.NoOp() {
			resp.Error = fmt.Errorf("migration changed output %s", name)
			return
		}
	}
}

type pdbWriteTracker struct {
	mu       sync.Mutex
	writes   []string
	requests int
}

func (c *pdbWriteTracker) CheckState(_ context.Context, _ statecheck.CheckStateRequest, resp *statecheck.CheckStateResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.writes) != 0 {
		resp.Error = fmt.Errorf("unchanged PDB migration sent API mutations: %v", c.writes)
	} else if c.requests == 0 {
		resp.Error = fmt.Errorf("PDB API observation saw no requests")
	}
}

type pdbObservedTransport struct {
	next   http.RoundTripper
	writes *pdbWriteTracker
}

type pdbObservedMetadata struct {
	kubernetes.KubeClientsets
	kubernetes.MetadataFilters
	client *kubernetesclient.Clientset
}

func (m pdbObservedMetadata) MainClientset() (*kubernetesclient.Clientset, error) {
	return m.client, nil
}

func (t pdbObservedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.writes.mu.Lock()
	t.writes.requests++
	if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions {
		t.writes.writes = append(t.writes.writes, req.Method+" "+req.URL.Path)
	}
	t.writes.mu.Unlock()
	return t.next.RoundTrip(req)
}

func pdbFactories(writes *pdbWriteTracker) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"kubernetes": func() (tfprotov6.ProviderServer, error) {
			p := kubernetes.Provider()
			if writes != nil {
				configure := p.ConfigureProvider
				p.ConfigureProvider = func(ctx context.Context, req schema.ConfigureProviderRequest, resp *schema.ConfigureProviderResponse) {
					configure(ctx, req, resp)
					if resp.Diagnostics.HasError() || resp.Meta == nil {
						return
					}
					client, err := resp.Meta.(kubernetes.KubeClientsets).MainClientset()
					if err != nil {
						resp.Diagnostics = append(resp.Diagnostics, diag.FromErr(err)...)
						return
					}
					restClient := client.PolicyV1().RESTClient().(*rest.RESTClient)
					httpClient := *restClient.Client
					transport := httpClient.Transport
					if transport == nil {
						transport = http.DefaultTransport
					}
					httpClient.Transport = pdbObservedTransport{next: transport, writes: writes}
					restClient.Client = &httpClient
					resp.Meta = pdbObservedMetadata{
						KubeClientsets:  resp.Meta.(kubernetes.KubeClientsets),
						MetadataFilters: resp.Meta.(kubernetes.MetadataFilters),
						client:          client,
					}
				}
			}
			return mux.MuxServerWithProvider(context.Background(), "test", p)
		},
	}
}
