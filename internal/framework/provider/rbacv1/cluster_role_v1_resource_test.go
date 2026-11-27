// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
)

const clusterRoleResourceName = "kubernetes_cluster_role_v1.test"

// The acceptance tests below port the scenarios of the SDKv2 test file
// (kubernetes/resource_kubernetes_cluster_role_v1_test.go). Rule attributes are sets in
// the Framework resource, so element assertions use TestCheckTypeSetElemAttr, and an
// omitted rule attribute is an empty set (".# = 0").

func TestClusterRoleReadMissingAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		for _, storedIdentity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/identity=%t", status, storedIdentity), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodGet || req.URL.Path != "/apis/rbac.authorization.k8s.io/v1/clusterroles/reader" {
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
				conn, err := client.NewForConfig(&rest.Config{Host: server.URL})
				if err != nil {
					t.Fatal(err)
				}
				r := &rbacv1.ClusterRoleV1{SDKv2Meta: func() any { return roleUpdateClientsets{client: conn} }}
				var schema frameworkresource.SchemaResponse
				r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
				raw := tfprotov6.RawState{JSON: []byte(`{
					"id":"reader",
					"metadata":[{"name":"reader","uid":"uid-1","resource_version":"10",
					"generation":1,"labels":null,"annotations":null,"generate_name":null}],
					"rule":[{"api_groups":[""],"non_resource_urls":[],"resources":["pods"],"verbs":["get"],"resource_names":[]}],
					"aggregation_rule":[]
				}`)}
				value, err := raw.Unmarshal(schema.Schema.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				state := tfsdk.State{Schema: schema.Schema, Raw: value}
				identitySchema := common.IdentitySchema()
				identity := tfsdk.ResourceIdentity{
					Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil),
				}
				wantIdentity := common.ResourceIdentity{
					Name:       types.StringValue("reader"),
					Kind:       types.StringValue("ClusterRole"),
					APIVersion: types.StringValue("rbac.authorization.k8s.io/v1"),
				}
				if storedIdentity {
					if status == http.StatusNotFound {
						// A distinct value proves an existing identity is not rebuilt.
						wantIdentity.Name = types.StringValue("existing-reader")
					}
					if diags := identity.Set(ctx, wantIdentity); diags.HasError() {
						t.Fatal(diags)
					}
				}
				responseIdentity := identity
				response := frameworkresource.ReadResponse{State: state, Identity: &responseIdentity}
				r.Read(ctx, frameworkresource.ReadRequest{State: state, Identity: &identity}, &response)
				if status == http.StatusNotFound {
					if response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
						t.Fatalf("missing ClusterRole must be removed: %v", response.Diagnostics)
					}
					var got common.ResourceIdentity
					if diags := response.Identity.Get(ctx, &got); diags.HasError() {
						t.Fatal(diags)
					}
					if got != wantIdentity {
						t.Fatalf("identity = %#v, want %#v", got, wantIdentity)
					}
					return
				}
				if !response.Diagnostics.HasError() || !response.State.Raw.Equal(state.Raw) || !response.Identity.Raw.Equal(identity.Raw) {
					t.Fatalf("API failure must preserve state and identity: %v", response.Diagnostics)
				}
			})
		}
	}
}

// TestAccClusterRole_basic follows the SDKv2 basic test: create, import, then add rules
// including a non_resource_urls-only rule that sets no api_groups.
func TestAccClusterRole_basic(t *testing.T) {
	// Colons are deliberate: ClusterRole names are path segments and the RBAC name
	// validation (validateRBACNameFunc) accepts them, e.g. "system:controller:foo".
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "id", name),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttrSet(clusterRoleResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(clusterRoleResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.resources.#", "2"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.resources.*", "pods/log"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.verbs.#", "2"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.non_resource_urls.#", "0"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.resource_names.#", "0"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "0"),
				),
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccClusterRoleConfig_modified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckNoResourceAttr(clusterRoleResourceName, "metadata.0.labels.TestLabelTwo"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.verbs.#", "3"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.verbs.*", "watch"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.1.resources.#", "1"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.1.verbs.#", "2"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.verbs.*", "list"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.2.api_groups.#", "0"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.2.non_resource_urls.#", "1"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.2.non_resource_urls.*", "/metrics"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.2.verbs.#", "1"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.2.verbs.*", "get"),
				),
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccClusterRoleConfig_modified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

func TestAccClusterRole_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_basic(name),
				Check:  testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(clusterRoleResourceName, map[string]knownvalue.Check{
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
						"kind":        knownvalue.StringExact("ClusterRole"),
					}),
				},
			},
			{
				// Import blocks already assert a no-op plan and unchanged identity;
				// plugin-testing only supports ImportStateVerify for CLI imports.
				ResourceName:    clusterRoleResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				Config: testAccClusterRoleConfig_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
			},
		},
	})
}

func TestAccClusterRole_importInvalidIdentity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	importStep := func(identityName, kind, apiVersion, expectError string) resource.TestStep {
		return resource.TestStep{
			ResourceName:           clusterRoleResourceName,
			ImportState:            true,
			ImportStateKind:        resource.ImportBlockWithResourceIdentity,
			ImportStateConfigExact: true,
			Config:                 testAccClusterRoleConfig_identityImport(name, identityName, kind, apiVersion),
			ExpectError:            regexp.MustCompile(expectError),
		}
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{Config: testAccClusterRoleConfig_basic(name)},
			importStep(name, "Role", "rbac.authorization.k8s.io/v1", "invalid identity import"),
			importStep(name, "ClusterRole", "rbac.authorization.k8s.io/v1beta1", "invalid identity import"),
			importStep("", "ClusterRole", "rbac.authorization.k8s.io/v1", `"name" must not be empty`),
		},
	})
}

// TestAccClusterRole_importNonExistent verifies that importing a name that does not exist
// fails instead of writing an empty resource to state.
func TestAccClusterRole_importNonExistent(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{Config: testAccClusterRoleConfig_basic(name)},
			{
				ResourceName:  clusterRoleResourceName,
				ImportState:   true,
				ImportStateId: name + "-does-not-exist",
				ExpectError:   regexp.MustCompile(`(?i)non-existent remote object`),
			},
		},
	})
}

func TestAccClusterRole_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_generateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(clusterRoleResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(clusterRoleResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.resources.#", "1"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.verbs.#", "1"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.verbs.*", "get"),
				),
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: strings.Replace(testAccClusterRoleConfig_generateName(prefix), "generate_name =", "labels = { added = \"yes\" }\n    generate_name =", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate)},
				},
				Check: testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
			},
		},
	})
}

// TestAccClusterRole_UpdatePatchOperationsOrderWithRemovals ports the SDKv2 regression
// test for patch ordering when rules are removed, replaced and added in one update.
func TestAccClusterRole_UpdatePatchOperationsOrderWithRemovals(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_patchStep0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "3"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.verbs.*", "list"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.2.non_resource_urls.#", "1"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.2.non_resource_urls.*", "/metrics"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.2.verbs.*", "get"),
				),
			},
			{
				Config: testAccClusterRoleConfig_patchStep1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "2"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.resources.*", "deployments"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.0.verbs.#", "2"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.resources.*", "jobs"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.verbs.*", "get"),
				),
			},
			{
				Config: testAccClusterRoleConfig_patchStep2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "4"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.1.verbs.*", "list"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.2.non_resource_urls.#", "1"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.2.non_resource_urls.*", "/metrics"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.2.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.3.resources.*", "jobs"),
					resource.TestCheckTypeSetElemAttr(clusterRoleResourceName, "rule.3.verbs.*", "get"),
				),
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccClusterRole_aggregationRuleBasic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_aggRule(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.key", "environment"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.operator", "In"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.values.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_labels.foo", "bar"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "0"),
				),
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccClusterRoleConfig_aggRuleModified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.key", "env"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.operator", "NotIn"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_expressions.0.values.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_labels.bar", "foo"),
				),
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccClusterRole_aggregationRuleMultiple(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_aggRuleMultiple(name, "foo", "bar", "bar", "foo"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.#", "2"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_labels.foo", "bar"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.1.match_labels.bar", "foo"),
				),
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Selector order is significant: swapping the selectors is an update.
				Config: testAccClusterRoleConfig_aggRuleMultiple(name, "bar", "foo", "foo", "bar"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.#", "2"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.0.match_labels.bar", "foo"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.0.cluster_role_selectors.1.match_labels.foo", "bar"),
				),
			},
		},
	})
}

// TestAccClusterRole_aggregationRuleRuleAggregation ports the SDKv2 test in which one
// ClusterRole aggregates the rules of another. The Kubernetes controller manager owns
// the aggregated role's rules, so they are not tracked in state (rule.# = 0); the source
// role keeps its configured rule.
func TestAccClusterRole_aggregationRuleRuleAggregation(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string
	check := resource.ComposeAggregateTestCheckFunc(
		testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
		resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "1"),
		resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "0"),
		resource.TestCheckResourceAttr("kubernetes_cluster_role_v1.source", "rule.#", "1"),
		resource.TestCheckTypeSetElemAttr("kubernetes_cluster_role_v1.source", "rule.0.resources.*", "pods"),
		resource.TestCheckTypeSetElemAttr("kubernetes_cluster_role_v1.source", "rule.0.resources.*", "pods/log"),
		resource.TestCheckTypeSetElemAttr("kubernetes_cluster_role_v1.source", "rule.0.verbs.*", "get"),
		resource.TestCheckTypeSetElemAttr("kubernetes_cluster_role_v1.source", "rule.0.verbs.*", "list"),
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_aggRuleSource(name),
				Check:  check,
			},
			{
				// The controller fills the aggregated rules after the first apply. A
				// refresh must neither show them in state nor plan their removal.
				Config: testAccClusterRoleConfig_aggRuleSource(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: check,
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccClusterRole_ruleAggregationTransitions moves a ClusterRole between explicit
// rules and an aggregation rule, in both directions, without replacing it.
func TestAccClusterRole_ruleAggregationTransitions(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string
	update := resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate)},
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_noAggregation(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "0"),
				),
			},
			{
				Config:           testAccClusterRoleConfig_aggregationOnly(name),
				ConfigPlanChecks: update,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "0"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "1"),
				),
			},
			{
				Config: testAccClusterRoleConfig_aggregationOnly(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config:           testAccClusterRoleConfig_noAggregation(name),
				ConfigPlanChecks: update,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "rule.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleResourceName, "aggregation_rule.#", "0"),
				),
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccClusterRole_noOpUpdate verifies an update that changes nothing the API stores
// (null -> {} labels) issues no patch: the resource version is unchanged.
func TestAccClusterRole_noOpUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceVersionUnchanged := statecheck.CompareValue(compare.ValuesSame())
	resourceVersion := tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("resource_version")
	config := strings.Replace(testAccClusterRoleConfig_generateNameFree(name), "metadata {", "metadata {\n    labels = {}", 1)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterRoleConfig_generateNameFree(name),
				ConfigStateChecks: []statecheck.StateCheck{
					resourceVersionUnchanged.AddStateValue(clusterRoleResourceName, resourceVersion),
				},
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					resourceVersionUnchanged.AddStateValue(clusterRoleResourceName, resourceVersion),
				},
			},
			{
				ResourceName:      clusterRoleResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					resourceVersionUnchanged.AddStateValue(clusterRoleResourceName, resourceVersion),
				},
			},
		},
	})
}

// TestAccClusterRole_disappears verifies that a ClusterRole deleted outside Terraform is
// dropped from state on refresh, planned for re-creation and recreated on apply.
func TestAccClusterRole_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config:             testAccClusterRoleConfig_basic(name),
				Check:              testAccClusterRoleDeleteOutOfBand(name),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccClusterRoleConfig_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: testAccClusterRoleCheckExists(clusterRoleResourceName, &uid),
			},
			{
				Config: testAccClusterRoleConfig_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccClusterRole_ruleValidation checks plan-time validation of the rule attributes.
// SDKv2 required at least one api_groups item when it was set, rejected null list
// elements, and let every other attribute be empty.
func TestAccClusterRole_ruleValidation(t *testing.T) {
	for _, tc := range []struct {
		name, rule, expectError string
	}{
		{"nullElement", `api_groups = [""]
    resources  = ["pods", null]
    verbs      = ["get"]`, "Null Set Value"},
		{"onlyNullVerb", `api_groups = [""]
    resources  = ["pods"]
    verbs      = [null]`, "Null Set Value"},
		{"emptyAPIGroups", `api_groups = []
    resources  = ["pods"]
    verbs      = ["get"]`, "(?s)api_groups.*at least 1"},
		{"missingVerbs", `api_groups = [""]
    resources  = ["pods"]`, "Missing required argument"},
		{"nonResourceURLOnly", `non_resource_urls = ["/metrics"]
    verbs             = ["get"]`, ""},
		{"emptyResourceNames", `api_groups     = [""]
    resources      = ["pods"]
    resource_names = []
    verbs          = ["get"]`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			step := resource.TestStep{
				Config:             testAccClusterRoleConfig_rule(name, tc.rule),
				PlanOnly:           true,
				ExpectNonEmptyPlan: tc.expectError == "",
			}
			if tc.expectError != "" {
				step.ExpectError = regexp.MustCompile(tc.expectError)
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps:                    []resource.TestStep{step},
			})
		})
	}
}

func TestAccClusterRole_missingMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "kubernetes_cluster_role_v1" "test" {
  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`,
				ExpectError: regexp.MustCompile(`(?s)metadata.*(required|at least 1)`),
			},
		},
	})
}

func TestAccClusterRole_invalidMetadataName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccClusterRoleConfig_generateNameFree("not/valid"),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value`),
			},
		},
	})
}

func testAccClusterRoleClient() (*client.Clientset, error) {
	return testAccRoleClient()
}

func testAccClusterRoleCheckDestroy(state *terraform.State) error {
	conn, err := testAccClusterRoleClient()
	if err != nil {
		return err
	}
	for _, module := range state.Modules {
		for _, rs := range module.Resources {
			// The deprecated type is included because a failed move test leaves the
			// role in state under its old address.
			if rs.Type != "kubernetes_cluster_role_v1" && rs.Type != "kubernetes_cluster_role" {
				continue
			}
			_, err := conn.RbacV1().ClusterRoles().Get(context.Background(), rs.Primary.ID, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("checking destruction of ClusterRole %q: %w", rs.Primary.ID, err)
			}
			return fmt.Errorf("ClusterRole %q still exists", rs.Primary.ID)
		}
	}
	return nil
}

// testAccClusterRoleCheckExists confirms the ClusterRole exists and that state and API
// agree on its UID. When priorUID is given it also fails if a later step sees another
// UID, i.e. the role was replaced instead of updated in place.
func testAccClusterRoleCheckExists(address string, priorUID *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %q not found in state", address)
		}
		conn, err := testAccClusterRoleClient()
		if err != nil {
			return err
		}
		role, err := conn.RbacV1().ClusterRoles().Get(context.Background(), rs.Primary.ID, metav1.GetOptions{})
		if err != nil {
			return err
		}
		uid := string(role.UID)
		if uid == "" || rs.Primary.Attributes["metadata.0.uid"] != uid {
			return fmt.Errorf("ClusterRole %q has mismatched API and state UID", rs.Primary.ID)
		}
		if priorUID != nil {
			if *priorUID != "" && *priorUID != uid {
				return fmt.Errorf("ClusterRole %q was replaced: UID changed from %q to %q", rs.Primary.ID, *priorUID, uid)
			}
			*priorUID = uid
		}
		return nil
	}
}

func testAccClusterRoleDeleteOutOfBand(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccClusterRoleClient()
		if err != nil {
			return err
		}
		return conn.RbacV1().ClusterRoles().Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

func testAccClusterRoleConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list"]
  }
}
`, name)
}

func testAccClusterRoleConfig_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    generate_name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, prefix)
}

// testAccClusterRoleConfig_generateNameFree is a minimal named ClusterRole.
func testAccClusterRoleConfig_generateNameFree(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["list"]
  }
}
`, name)
}

func testAccClusterRoleConfig_rule(name, rule string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    %[2]s
  }
}
`, name, rule)
}

func testAccClusterRoleConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelThree = "three"
    }

    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list", "watch"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }

  rule {
    non_resource_urls = ["/metrics"]
    verbs             = ["get"]
  }
}
`, name)
}

func testAccClusterRoleConfig_patchStep0(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["list"]
  }

  rule {
    non_resource_urls = ["/metrics"]
    verbs             = ["get"]
  }
}
`, name)
}

func testAccClusterRoleConfig_patchStep1(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }

  rule {
    api_groups = [""]
    resources  = ["jobs"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccClusterRoleConfig_patchStep2(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["list"]
  }

  rule {
    api_groups = [""]
    resources  = ["deployments"]
    verbs      = ["list"]
  }

  rule {
    non_resource_urls = ["/metrics"]
    verbs             = ["get"]
  }

  rule {
    api_groups = [""]
    resources  = ["jobs"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccClusterRoleConfig_aggRule(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = %[1]q
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        foo = "bar"
      }

      match_expressions {
        key      = "environment"
        operator = "In"
        values   = ["non-exists-12345"]
      }
    }
  }
}
`, name)
}

func testAccClusterRoleConfig_aggRuleModified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    labels = {
      TestLabelOne   = "one"
      TestLabelTwo   = "two"
      TestLabelThree = "three"
    }

    name = %[1]q
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        bar = "foo"
      }

      match_expressions {
        key      = "env"
        operator = "NotIn"
        values   = ["non"]
      }
    }
  }
}
`, name)
}

func testAccClusterRoleConfig_aggRuleMultiple(name, firstKey, firstValue, secondKey, secondValue string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        %[2]s = %[3]q
      }
    }
    cluster_role_selectors {
      match_labels = {
        %[4]s = %[5]q
      }
    }
  }
}
`, name, firstKey, firstValue, secondKey, secondValue)
}

// testAccClusterRoleConfig_aggRuleSource declares an aggregated ClusterRole and the
// ClusterRole whose rules it aggregates.
func testAccClusterRoleConfig_aggRuleSource(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        "rbac.example.com/aggregate-to-monitoring-%[1]s" = "true"
      }
    }
  }
}

resource "kubernetes_cluster_role_v1" "source" {
  metadata {
    labels = {
      "rbac.example.com/aggregate-to-monitoring-%[1]s" = "true"
    }
    name = "%[1]s-source"
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list"]
  }
}
`, name)
}

func testAccClusterRoleConfig_noAggregation(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccClusterRoleConfig_aggregationOnly(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  aggregation_rule {
    cluster_role_selectors {
      match_labels = {
        "rbac.example.com/aggregate-to-%[1]s" = "true"
      }
    }
  }
}
`, name)
}

func testAccClusterRoleConfig_identityImport(name, identityName, kind, apiVersion string) string {
	return fmt.Sprintf(`
import {
  to = kubernetes_cluster_role_v1.test
  identity = {
    name        = %[2]q
    kind        = %[3]q
    api_version = %[4]q
  }
}

resource "kubernetes_cluster_role_v1" "test" {
  metadata {
    name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods", "pods/log"]
    verbs      = ["get", "list"]
  }
}
`, name, identityName, kind, apiVersion)
}
