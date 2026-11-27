// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package schedulingv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/schedulingv1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// compile-time check: PriorityClassV1 satisfies resource.Resource.
var _ frameworkresource.Resource = (*schedulingv1.PriorityClassV1)(nil)

// Unit tests

// priorityClassFakeClientsets serves a client pointed at an httptest server, with no
// ignore_annotations or ignore_labels configured.
type priorityClassFakeClientsets struct {
	kubernetes.KubeClientsets
	client *k8sclient.Clientset
}

func (c priorityClassFakeClientsets) MainClientset() (*k8sclient.Clientset, error) {
	return c.client, nil
}

func (priorityClassFakeClientsets) GetIgnoreAnnotations() []string { return nil }
func (priorityClassFakeClientsets) GetIgnoreLabels() []string      { return nil }

func newPriorityClassForServer(t *testing.T, server *httptest.Server) (*schedulingv1.PriorityClassV1, frameworkresource.SchemaResponse) {
	t.Helper()
	conn, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	r := &schedulingv1.PriorityClassV1{SDKv2Meta: func() any { return priorityClassFakeClientsets{client: conn} }}
	var schemaResp frameworkresource.SchemaResponse
	r.Schema(context.Background(), frameworkresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatal(schemaResp.Diagnostics)
	}
	return r, schemaResp
}

// priorityClassStateJSON is state as the upgrade from SDKv2 leaves it: generate_name is ""
// and the maps are null.
const priorityClassStateJSON = `{
	"id":"test", "value":100, "description":"", "global_default":false, "preemption_policy":"PreemptLowerPriority",
	"metadata":[{"annotations":null,"labels":null,"generate_name":"",
	"name":"test","generation":1,"resource_version":"10","uid":"uid-1"}]
}`

func priorityClassState(t *testing.T, schemaResp frameworkresource.SchemaResponse) tfsdk.State {
	t.Helper()
	raw := tfprotov6.RawState{JSON: []byte(priorityClassStateJSON)}
	value, err := raw.Unmarshal(schemaResp.Schema.Type().TerraformType(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: schemaResp.Schema, Raw: value}
}

func priorityClassIdentityValue(name string) common.ResourceIdentity {
	return common.ResourceIdentity{
		Name:       types.StringValue(name),
		Kind:       types.StringValue("PriorityClass"),
		APIVersion: types.StringValue("scheduling.k8s.io/v1"),
	}
}

// TestPriorityClassReadMissingAndErrors covers Read when the object is gone (F1). The
// framework rejects RemoveResource without an identity, so Read must build one when the
// prior identity is null (Terraform before 1.12) or all-null (state upgraded from SDKv2,
// see common.UpgradeIdentity), and must keep an identity that is already set. Any other
// API error must leave state and identity untouched.
func TestPriorityClassReadMissingAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		for _, stored := range []string{"null", "allNull", "set"} {
			t.Run(fmt.Sprintf("%d/identity=%s", status, stored), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodGet || req.URL.Path != "/apis/scheduling.k8s.io/v1/priorityclasses/test" {
						t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					if err := json.NewEncoder(w).Encode(metav1.Status{
						TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
						Status:   "Failure", Code: int32(status), Message: http.StatusText(status),
					}); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				r, schemaResp := newPriorityClassForServer(t, server)
				state := priorityClassState(t, schemaResp)

				identitySchema := common.IdentitySchema()
				identity := tfsdk.ResourceIdentity{
					Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil),
				}
				wantIdentity := priorityClassIdentityValue("test")
				switch stored {
				case "allNull":
					if diags := identity.Set(ctx, common.ResourceIdentity{
						Name: types.StringNull(), Kind: types.StringNull(), APIVersion: types.StringNull(),
					}); diags.HasError() {
						t.Fatal(diags)
					}
					if identity.Raw.IsNull() || !identity.Raw.IsFullyNull() {
						t.Fatal("test setup: identity must be a non-null object with null attributes")
					}
				case "set":
					// A distinct name proves an existing identity is not rebuilt.
					wantIdentity = priorityClassIdentityValue("existing")
					if diags := identity.Set(ctx, wantIdentity); diags.HasError() {
						t.Fatal(diags)
					}
				}

				responseIdentity := identity
				resp := frameworkresource.ReadResponse{State: state, Identity: &responseIdentity}
				r.Read(ctx, frameworkresource.ReadRequest{State: state, Identity: &identity}, &resp)

				if status != http.StatusNotFound {
					if !resp.Diagnostics.HasError() || !resp.State.Raw.Equal(state.Raw) || !resp.Identity.Raw.Equal(identity.Raw) {
						t.Fatalf("an API failure must preserve state and identity: %v", resp.Diagnostics)
					}
					return
				}
				if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
					t.Fatalf("a missing PriorityClass must be removed from state: %v", resp.Diagnostics)
				}
				var got common.ResourceIdentity
				if diags := resp.Identity.Get(ctx, &got); diags.HasError() {
					t.Fatal(diags)
				}
				if got != wantIdentity {
					t.Fatalf("identity = %#v, want %#v", got, wantIdentity)
				}
			})
		}
	}
}

// TestAccPriorityClassV1_basic starts from the required fields only, moves to every field
// except generate_name, and then changes the mutable fields in later steps. Every update is
// in place (same uid), and an import after each step shows the API holds what state says.
// It carries the assertions of the SDKv2 test it replaces.
func TestAccPriorityClassV1_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_priority_class_v1.test"
	sameUID := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			// Required fields only.
			{
				Config: testAccKubernetesPriorityClassV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Every field except generate_name.
			{
				Config: testAccPriorityClassV1Config_all(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
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
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "description", "Foobar"),
					resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Adds and removes a key in each map at once.
			{
				Config: testAccPriorityClassV1Config_metaModified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelOne", "one"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo", "two"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.TestLabelThree", "three"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.labels.TestLabelFour"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "description", "Foobar"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Removes both maps and changes the description.
			{
				Config: testAccPriorityClassV1Config_modified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated description"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Back to the required fields: description returns to its default, "".
			{
				Config: testAccKubernetesPriorityClassV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPriorityClassV1_completeName sets every attribute with name, then updates every
// mutable one.
func TestAccPriorityClassV1_completeName(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_priority_class_v1.test"
	sameUID := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPriorityClassV1Config_completeName(name),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.environment", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.com/note", "migration"),
					resource.TestCheckResourceAttr(resourceName, "value", "500"),
					resource.TestCheckResourceAttr(resourceName, "description", "Migration test priority class"),
					resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPriorityClassV1Config_completeNameUpdated(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.environment", "updated"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.com/note", "updated"),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated priority class"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPriorityClassV1_completeGenerateName sets every attribute with generate_name, then
// updates every mutable one. The server-assigned name must stay known in the update plan.
func TestAccPriorityClassV1_completeGenerateName(t *testing.T) {
	prefix := fmt.Sprintf("tf-acc-test-%s-", acctest.RandString(10))
	resourceName := "kubernetes_priority_class_v1.test"
	namePattern := regexp.MustCompile("^" + prefix)
	nameMatchesPrefix := knownvalue.StringRegexp(namePattern)
	namePath := tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("name")
	sameUID := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPriorityClassV1Config_completeGenerateName(prefix),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "id", namePattern),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", namePattern),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.environment", "test"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.com/note", "migration"),
					resource.TestCheckResourceAttr(resourceName, "value", "500"),
					resource.TestCheckResourceAttr(resourceName, "description", "Migration test generated priority class"),
					resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPriorityClassV1Config_completeGenerateNameUpdated(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(resourceName, tfjsonpath.New("id"), nameMatchesPrefix),
						plancheck.ExpectKnownValue(resourceName, namePath, nameMatchesPrefix),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.environment", "updated"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.com/note", "updated"),
					resource.TestCheckResourceAttr(resourceName, "description", "Updated priority class"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPriorityClassV1_identity verifies that resource identity is populated correctly and
// import using an import block with resource identity succeeds (TF >= 1.12).
func TestAccPriorityClassV1_identity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesPriorityClassV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("scheduling.k8s.io/v1"),
						"kind":        knownvalue.StringExact("PriorityClass"),
					}),
				},
			},
			{
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// TestAccPriorityClassV1_importInvalidIdentity checks that identity import rejects an
// identity for another kind or API version, and an empty name.
func TestAccPriorityClassV1_importInvalidIdentity(t *testing.T) {
	for _, tc := range []struct {
		name       string
		kind       string
		apiVersion string
		emptyName  bool
		wantError  string
	}{
		{name: "wrongKind", kind: "Namespace", apiVersion: "scheduling.k8s.io/v1", wantError: "invalid identity import"},
		{name: "wrongAPIVersion", kind: "PriorityClass", apiVersion: "v1", wantError: "invalid identity import"},
		{name: "emptyName", kind: "PriorityClass", apiVersion: "scheduling.k8s.io/v1", emptyName: true, wantError: `Identity attribute "name" must not be empty`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-acc-pc")
			importName := name
			if tc.emptyName {
				importName = ""
			}
			config := testAccPriorityClassV1Config_identityImport(name, importName, tc.kind, tc.apiVersion)

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckPriorityClassV1Destroy,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_12_0),
				},
				Steps: []resource.TestStep{
					{
						Config: testAccKubernetesPriorityClassV1Config_basic(name),
					},
					{
						ResourceName:           "kubernetes_priority_class_v1.test",
						ImportState:            true,
						ImportStateKind:        resource.ImportBlockWithResourceIdentity,
						ImportStateConfigExact: true,
						Config:                 config,
						ExpectError:            regexp.MustCompile(tc.wantError),
					},
				},
			})
		})
	}
}

// TestAccPriorityClassV1_generateName ports the SDKv2 test (value 999, preemption_policy
// omitted so the default applies), then updates the object.
func TestAccPriorityClassV1_generateName(t *testing.T) {
	prefix := "tf-acc-test-"
	resourceName := "kubernetes_priority_class_v1.test"
	namePattern := regexp.MustCompile("^" + prefix)
	sameUID := statecheck.CompareValue(compare.ValuesSame())

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityClassV1Config_generatedName(prefix),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", namePattern),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "999"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPriorityClassV1Config_generatedNameUpdated(prefix),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.TestCheckResourceAttr(resourceName, "description", "Updated"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPriorityClassV1_valueRequiresReplace verifies that changing the immutable 'value'
// field destroys the old resource and creates a new one (RequiresReplace plan modifier).
func TestAccPriorityClassV1_valueRequiresReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityClassV1Config_withValue(name, 100),
				Check:  resource.TestCheckResourceAttr(resourceName, "value", "100"),
			},
			// Changing value must trigger destroy + create, not an in-place update.
			{
				Config: testAccPriorityClassV1Config_withValue(name, 200),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "value", "200"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPriorityClassV1_preemptionPolicyRequiresReplace starts from the SDKv2 test's basic
// configuration and assertions, then changes only preemption_policy, which replaces the
// object. Metadata is identical in both steps, so the policy is the only possible cause.
func TestAccPriorityClassV1_preemptionPolicyRequiresReplace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_priority_class_v1.test"
	configNever := testAccPriorityClassV1Config_sdkv2Basic(name, "Never")
	configPreemptLowerPriority := testAccPriorityClassV1Config_sdkv2Basic(name, "PreemptLowerPriority")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: configNever,
				Check: resource.ComposeAggregateTestCheckFunc(
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
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: configPreemptLowerPriority,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "preemption_policy", "PreemptLowerPriority"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPriorityClassV1_globalDefault ports the SDKv2 test (created with
// global_default = true and description "Foobar"), then toggles global_default in place.
//
// NOTE: this test is intentionally non-parallel. Setting global_default = true changes the
// default priority for ALL pods on the cluster for the duration of the test, which can
// corrupt scheduling behaviour in concurrently running tests. The original SDKv2 test was
// also serial for the same reason.
func TestAccPriorityClassV1_globalDefault(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandString(10))
	resourceName := "kubernetes_priority_class_v1.test"
	sameUID := statecheck.CompareValue(compare.ValuesSame())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityClassV1Config_globalDefault(name, true),
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
					resource.TestCheckResourceAttr(resourceName, "description", "Foobar"),
					resource.TestCheckResourceAttr(resourceName, "global_default", "true"),
					resource.TestCheckResourceAttr(resourceName, "preemption_policy", "Never"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Disable it — must be an in-place update, not a replace.
			{
				Config: testAccPriorityClassV1Config_globalDefault(name, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.TestCheckResourceAttr(resourceName, "global_default", "false"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Enable it again.
			{
				Config: testAccPriorityClassV1Config_globalDefault(name, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					sameUID.AddStateValue(resourceName, uidPath),
				},
				Check: resource.TestCheckResourceAttr(resourceName, "global_default", "true"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccPriorityClassV1_disappears verifies that if the PriorityClass is deleted outside
// Terraform, the next refresh removes it from state and the plan creates it again.
func TestAccPriorityClassV1_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:             testAccKubernetesPriorityClassV1Config_basic(name),
				Check:              testAccPriorityClassV1DeleteOutOfBand(name),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccKubernetesPriorityClassV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func testAccPriorityClassV1DeleteOutOfBand(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		return conn.SchedulingV1().PriorityClasses().Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

// TestAccPriorityClassV1_ignoreMetadata verifies the provider-level ignore_annotations and
// ignore_labels:
//   - an annotation and a label added outside Terraform, whose keys match the patterns, do
//     not show as drift; and
//   - an annotation and a label configured in Terraform stay managed even though their keys
//     match the same patterns.
func TestAccPriorityClassV1_ignoreMetadata(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	config := testAccPriorityClassV1Config_ignoreMetadata(name)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				PreConfig: func() { testAccPriorityClassV1InjectExternalMetadata(t, name) },
				Config:    config,
				PlanOnly:  true,
			},
		},
	})
}

// testAccPriorityClassV1InjectExternalMetadata adds an annotation and a label outside
// Terraform, with keys that match the ignore patterns in the config.
func testAccPriorityClassV1InjectExternalMetadata(t *testing.T, name string) {
	t.Helper()
	conn, err := sdkv2providerMeta()().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	obj, err := conn.SchedulingV1().PriorityClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get PriorityClass %q: %v", name, err)
	}
	if obj.Annotations == nil {
		obj.Annotations = map[string]string{}
	}
	if obj.Labels == nil {
		obj.Labels = map[string]string{}
	}
	obj.Annotations["external.io/injected"] = "outside-terraform"
	obj.Labels["external.io/injected"] = "outside-terraform"
	if _, err := conn.SchedulingV1().PriorityClasses().Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update PriorityClass %q: %v", name, err)
	}
}

// TestAccPriorityClassV1_maxUserValue creates a user-defined class at 1,000,000,000, the
// largest value the API accepts and the plan-time validator must allow.
func TestAccPriorityClassV1_maxUserValue(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityClassV1Config_withValue(name, 1_000_000_000),
				Check:  resource.TestCheckResourceAttr(resourceName, "value", "1000000000"),
			},
		},
	})
}

// TestAccPriorityClassV1_invalidValue verifies that value > 1,000,000,000 is rejected at plan
// time for a name or generate_name prefix without the "system-" prefix, and that a value
// below the int32 minimum is rejected for any name.
func TestAccPriorityClassV1_invalidValue(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	invalidValue := regexp.MustCompile(`Invalid Priority Class Value`)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccPriorityClassV1Config_withValue(name, 1_000_000_001),
				PlanOnly:    true,
				ExpectError: invalidValue,
			},
			{
				Config:      testAccPriorityClassV1Config_generatedNameWithValue("tf-acc-pc-", 1_000_000_001),
				PlanOnly:    true,
				ExpectError: invalidValue,
			},
			{
				Config:      testAccPriorityClassV1Config_withValue(name, -3_000_000_000),
				PlanOnly:    true,
				ExpectError: invalidValue,
			},
		},
	})
}

// TestAccPriorityClassV1_systemValueAboveUserLimit verifies that the validator lets a
// "system-" name or generate_name prefix through above 1,000,000,000.
func TestAccPriorityClassV1_systemValueAboveUserLimit(t *testing.T) {
	name := acctest.RandomWithPrefix("system-acc-pc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:             testAccPriorityClassV1Config_withValue(name, 2_000_000_000),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config:             testAccPriorityClassV1Config_generatedNameWithValue("system-acc-pc-", 2_000_000_000),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccPriorityClassV1_nameAndGenerateNameConflict verifies that setting both name and
// generate_name is rejected at plan time by the ConflictsWith validator.
func TestAccPriorityClassV1_nameAndGenerateNameConflict(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccPriorityClassV1Config_nameAndGenerateName(name, "tf-acc-pc-"),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
		},
	})
}

// TestAccPriorityClassV1_missingMetadata verifies that omitting the metadata block is
// rejected at plan time.
func TestAccPriorityClassV1_missingMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccPriorityClassV1Config_missingMetadata(),
				ExpectError: regexp.MustCompile(`(?s)(metadata.*required|Missing required argument|Missing Configuration for Required Attribute|Invalid Block|at least 1 elements)`),
			},
		},
	})
}

// TestAccPriorityClassV1_missingValue verifies that omitting the required value attribute is
// rejected at plan time.
func TestAccPriorityClassV1_missingValue(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccPriorityClassV1Config_missingValue(name),
				ExpectError: regexp.MustCompile(`(?s)(Missing required argument|argument "value" is required|Missing Configuration for Required Attribute)`),
			},
		},
	})
}

// TestAccPriorityClassV1_invalidPreemptionPolicy verifies that an invalid preemption_policy
// is rejected at plan time by the stringvalidator.OneOf validator.
func TestAccPriorityClassV1_invalidPreemptionPolicy(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccPriorityClassV1Config_withPreemptionPolicy(name, "InvalidPolicy"),
				ExpectError: regexp.MustCompile(`(?s)(Invalid Attribute Value Match|value must be one of)`),
			},
		},
	})
}

// TestAccPriorityClassV1_systemNameValueExceedsMaxInt32 verifies that system-prefixed
// priority classes with values exceeding math.MaxInt32 are rejected at plan time.
func TestAccPriorityClassV1_systemNameValueExceedsMaxInt32(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccPriorityClassV1Config_withValue("system-overflow", 3_000_000_000),
				ExpectError: regexp.MustCompile(`Invalid Priority Class Value`),
			},
		},
	})
}

// TestAccPriorityClassV1_invalidMetadataName verifies that an invalid metadata.name (not
// complying with DNS subdomain RFC 1123) is rejected at plan time.
func TestAccPriorityClassV1_invalidMetadataName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesPriorityClassV1Config_basic("INVALID_NAME_UPPERCASE"),
				ExpectError: regexp.MustCompile(`(?s)(Invalid Attribute Value Match|DNS-1123|subdomain)`),
			},
		},
	})
}

// TestAccPriorityClassV1_unknownNameWithValidSystemValue verifies that when metadata.name is
// unknown at plan time, values above 1,000,000,000 (valid for system classes) are not
// prematurely rejected at plan time.
func TestAccPriorityClassV1_unknownNameWithValidSystemValue(t *testing.T) {
	name := acctest.RandomWithPrefix("system-acc-pc")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config:             testAccPriorityClassV1Config_computedName(name, 2_000_000_000),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccPriorityClassV1_unknownMetadataCollection verifies that when the entire metadata
// collection is unknown at plan time (e.g. via dynamic blocks), validation does not fail
// with a Value Conversion Error.
func TestAccPriorityClassV1_unknownMetadataCollection(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pc")
	resourceName := "kubernetes_priority_class_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPriorityClassV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPriorityClassV1Config_dynamicMetadata(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "value", "100"),
				),
			},
		},
	})
}

// HCL config helpers

// testAccPriorityClassV1Config_all sets every attribute except generate_name, which
// conflicts with name. value and preemption_policy keep the values the required-only config
// gets, since changing either replaces the object.
func testAccPriorityClassV1Config_all(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name = %[1]q
  }

  value             = 100
  description       = "Foobar"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}
`, name)
}

func testAccPriorityClassV1Config_metaModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
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

  value             = 100
  description       = "Foobar"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}
`, name)
}

func testAccPriorityClassV1Config_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = 100
  description       = "Updated description"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}
`, name)
}

func testAccPriorityClassV1Config_completeNameUpdated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      environment = "updated"
    }
    annotations = {
      "example.com/note" = "updated"
    }
  }

  value             = 500
  description       = "Updated priority class"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}
`, name)
}

func testAccPriorityClassV1Config_completeGenerateNameUpdated(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = %[1]q
    labels = {
      environment = "updated"
    }
    annotations = {
      "example.com/note" = "updated"
    }
  }

  value             = 500
  description       = "Updated priority class"
  global_default    = false
  preemption_policy = "PreemptLowerPriority"
}
`, prefix)
}

// testAccPriorityClassV1Config_sdkv2Basic is the SDKv2 test's basic configuration
// (preemption_policy = "Never"), with the policy as a parameter so a test can change it alone.
func testAccPriorityClassV1Config_sdkv2Basic(name, preemptionPolicy string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    annotations = {
      TestAnnotationOne = "one"
    }

    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
      TestLabelFour  = "four"
    }

    name = %[1]q
  }

  value             = 100
  preemption_policy = %[2]q
}
`, name, preemptionPolicy)
}

// testAccPriorityClassV1Config_generatedName is the SDKv2 test's generated-name configuration.
func testAccPriorityClassV1Config_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  value = 999
}
`, prefix)
}

func testAccPriorityClassV1Config_generatedNameUpdated(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  value       = 999
  description = "Updated"
}
`, prefix)
}

func testAccPriorityClassV1Config_generatedNameWithValue(prefix string, value int) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  value = %[2]d
}
`, prefix, value)
}

func testAccPriorityClassV1Config_withValue(name string, value int) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = %[2]d
  preemption_policy = "Never"
}
`, name, value)
}

func testAccPriorityClassV1Config_withPreemptionPolicy(name, policy string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = 100
  preemption_policy = %[2]q
}
`, name, policy)
}

func testAccPriorityClassV1Config_globalDefault(name string, globalDefault bool) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value             = 100
  description       = "Foobar"
  preemption_policy = "Never"
  global_default    = %[2]t
}
`, name, globalDefault)
}

func testAccPriorityClassV1Config_identityImport(name, importName, kind, apiVersion string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }

  value = 100
}

import {
  to = kubernetes_priority_class_v1.test
  identity = {
    name        = %[2]q
    kind        = %[3]q
    api_version = %[4]q
  }
}
`, name, importName, kind, apiVersion)
}

// testAccPriorityClassV1Config_ignoreMetadata configures an annotation and a label whose keys
// match the provider's ignore patterns, which testAccPriorityClassV1InjectExternalMetadata
// also matches.
func testAccPriorityClassV1Config_ignoreMetadata(name string) string {
	return fmt.Sprintf(`
provider "kubernetes" {
  ignore_annotations = ["^external\\.io/"]
  ignore_labels      = ["^external\\.io/"]
}

resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
    annotations = {
      "external.io/managed" = "true"
    }
    labels = {
      "external.io/managed" = "true"
    }
  }

  value = 100
}
`, name)
}

func testAccPriorityClassV1Config_nameAndGenerateName(name, prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name          = %[1]q
    generate_name = %[2]q
  }

  value = 100
}
`, name, prefix)
}

func testAccPriorityClassV1Config_missingMetadata() string {
	return `
resource "kubernetes_priority_class_v1" "test" {
  value = 100
}
`
}

func testAccPriorityClassV1Config_missingValue(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = %[1]q
  }
}
`, name)
}

func testAccPriorityClassV1Config_computedName(name string, value int) string {
	return fmt.Sprintf(`
resource "terraform_data" "name" {
  input = %[1]q
}

resource "kubernetes_priority_class_v1" "test" {
  metadata {
    name = terraform_data.name.output
  }

  value = %[2]d
}
`, name, value)
}

func testAccPriorityClassV1Config_dynamicMetadata(name string) string {
	return fmt.Sprintf(`
resource "terraform_data" "meta" {
  input = {
    name = %[1]q
  }
}

resource "kubernetes_priority_class_v1" "test" {
  dynamic "metadata" {
    for_each = terraform_data.meta.output != null ? [terraform_data.meta.output] : []
    content {
      name = metadata.value.name
    }
  }

  value = 100
}
`, name)
}
