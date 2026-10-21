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
	"strconv"
	"strings"
	"testing"

	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	rbacv1 "github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

// compile-time check: RoleBindingV1 satisfies frameworkresource.Resource.
var _ frameworkresource.Resource = (*rbacv1.RoleBindingV1)(nil)

// roleBindingTestClientsets points Read at a fake API server; the ignore lists
// are empty.
type roleBindingTestClientsets struct {
	kubernetes.KubeClientsets
	client *client.Clientset
}

func (c roleBindingTestClientsets) MainClientset() (*client.Clientset, error) {
	return c.client, nil
}

func (roleBindingTestClientsets) GetIgnoreAnnotations() []string { return nil }

func (roleBindingTestClientsets) GetIgnoreLabels() []string { return nil }

// TestRoleBindingReadMissingAndErrors covers Read paths CI acceptance tests do
// not reach: Terraform before 1.12 sends no identity, so removing a missing
// RoleBinding must set identity first; other API errors must keep state and
// identity unchanged.
func TestRoleBindingReadMissingAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		for _, storedIdentity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/identity=%t", status, storedIdentity), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodGet || req.URL.Path != "/apis/rbac.authorization.k8s.io/v1/namespaces/team/rolebindings/reader" {
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
				r := &rbacv1.RoleBindingV1{SDKv2Meta: func() any { return roleBindingTestClientsets{client: conn} }}
				var schema frameworkresource.SchemaResponse
				r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
				raw := tfprotov6.RawState{JSON: []byte(`{
					"id":"team/reader",
					"metadata":[{"name":"reader","namespace":"team","uid":"uid-1","resource_version":"10",
					"generation":1,"labels":null,"annotations":null,"generate_name":null}],
					"role_ref":[{"api_group":"rbac.authorization.k8s.io","kind":"Role","name":"reader"}],
					"subject":[{"api_group":"rbac.authorization.k8s.io","kind":"User","name":"alice","namespace":""}]
				}`)}
				value, err := raw.Unmarshal(schema.Schema.Type().TerraformType(ctx))
				if err != nil {
					t.Fatal(err)
				}
				state := tfsdk.State{Schema: schema.Schema, Raw: value}
				identitySchema := common.NamespacedIdentitySchema()
				identity := tfsdk.ResourceIdentity{
					Schema: identitySchema, Raw: tftypes.NewValue(identitySchema.Type().TerraformType(ctx), nil),
				}
				wantIdentity := common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{
						Name: types.StringValue("reader"), Kind: types.StringValue("RoleBinding"),
						APIVersion: types.StringValue("rbac.authorization.k8s.io/v1"),
					},
					Namespace: types.StringValue("team"),
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
						t.Fatalf("missing RoleBinding must be removed: %v", response.Diagnostics)
					}
					var got common.NamespacedResourceIdentity
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

// TestAccRoleBinding_basic creates a RoleBinding with a single User subject,
// verifies all computed fields are populated, and confirms import round-trips
// without drift.
func TestAccRoleBinding_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
					resource.TestCheckResourceAttr(resourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
				),
			},
			// Import by "namespace/name" — verify no drift after import.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     "default/" + name,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
		},
	})
}

// TestAccRoleBinding_update verifies that mutable fields — subjects, labels,
// and annotations — can be changed in-place without destroy/recreate.
func TestAccRoleBinding_update(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: create with single subject.
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
			// Step 2: add two more subjects and metadata — must be in-place update.
			{
				Config: testAccRoleBindingV1Config_updated(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.managed-by", "terraform"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.example.com/note", "updated"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
			// Step 3: revert to basic — subjects removed in-place, not replaced.
			{
				Config: testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.labels.managed-by"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
		},
	})
}

// TestAccRoleBinding_generateName creates a RoleBinding using generate_name,
// verifies server assigned name and that updating a secondary attribute (labels)
// triggers an in-place update rather than a destroy/re-create (K8S-MIGRATE-025).
func TestAccRoleBinding_generateName(t *testing.T) {
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: create with generate_name
			{
				Config: testAccRoleBindingV1Config_generateName("tf-acc-rb-"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.name"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", "tf-acc-rb-"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
			// Step 2: update secondary field (labels) — must update in-place without replacement
			{
				Config: testAccRoleBindingV1Config_generateName_updated("tf-acc-rb-"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", "tf-acc-rb-"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.team", "sec"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
		},
	})
}

// TestAccRoleBinding_saSubject verifies that a ServiceAccount subject works
// correctly, including the empty api_group and explicit namespace.
func TestAccRoleBinding_saSubject(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleBindingV1Config_saSubject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "default"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", ""),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
		},
	})
}

// TestAccRoleBinding_groupSubject verifies that a Group subject works correctly.
func TestAccRoleBinding_groupSubject(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleBindingV1Config_groupSubject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "Group"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.name", "dev-team"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
		},
	})
}

// TestAccRoleBinding_emptySubjectNamespace verifies that an explicitly empty
// subject namespace is preserved: apply -> refresh -> empty plan, and import
// round-trips without drift.
func TestAccRoleBinding_emptySubjectNamespace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleBindingV1Config_emptySubjectNamespace(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.namespace", ""),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccRoleBindingV1Config_emptySubjectNamespace(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
		},
	})
}

// TestAccRoleBinding_emptyUserAPIGroup verifies that api_group = "" on a User subject
// (which the API server defaults to rbac.authorization.k8s.io) does not drift:
// apply -> refresh -> empty plan.
func TestAccRoleBinding_emptyUserAPIGroup(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleBindingV1Config_emptyUserAPIGroup(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: testAccRoleBindingV1Config_emptyUserAPIGroup(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccRoleBinding_roleRefRequiresReplace verifies that changing role_ref
// destroys the existing RoleBinding and creates a new one (RequiresReplace).
func TestAccRoleBinding_roleRefRequiresReplace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check:  resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "Role"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
			// Changing role_ref.kind must trigger destroy + create, not in-place update.
			{
				Config: testAccRoleBindingV1Config_clusterRoleRef(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "role_ref.0.kind", "ClusterRole"),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.0.resource_version",
				},
			},
		},
	})
}

// TestAccRoleBinding_disappears verifies that if the RoleBinding is deleted
// outside Terraform, the next refresh removes it from state (Read 404 →
// RemoveResource), the plan proposes to recreate it, and re-applying converges.
func TestAccRoleBinding_disappears(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create, then delete out-of-band; the post-apply refresh must
			// remove the binding from state and plan a create.
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					testAccRoleBindingDeleteOutOfBand(name),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccRoleBindingV1Config_basic(name),
				Check:  resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
			},
			{
				Config: testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccRoleBinding_subjectReorder replays the SDKv2 _Bug sequence on the
// Framework resource: removing subjects from the front and middle and
// inserting at the front must update in place and converge with no drift.
func TestAccRoleBinding_subjectReorder(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	step := func(users ...string) resource.TestStep {
		checks := []resource.TestCheckFunc{
			resource.TestCheckResourceAttr(resourceName, "subject.#", strconv.Itoa(len(users))),
		}
		for i, user := range users {
			checks = append(checks, resource.TestCheckResourceAttr(resourceName, fmt.Sprintf("subject.%d.name", i), user))
		}
		return resource.TestStep{
			Config: testAccRoleBindingV1Config_users(name, users...),
			ConfigPlanChecks: resource.ConfigPlanChecks{
				PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
			},
			Check: resource.ComposeAggregateTestCheckFunc(checks...),
		}
	}
	create := step("notauser1", "notauser2", "notauser3")
	create.ConfigPlanChecks.PreApply = nil

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			create,
			step("notauser2", "notauser4"),
			step("notauser0", "notauser1", "notauser2", "notauser3"),
		},
	})
}

// TestAccRoleBinding_computedAPIGroup verifies the Optional+Computed
// subject.api_group path: subjects that omit api_group resolve it from the API
// on create and on update, and a label-only update leaves subjects unchanged.
func TestAccRoleBinding_computedAPIGroup(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"
	inPlaceUpdate := resource.ConfigPlanChecks{
		PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleBindingV1Config_serviceAccounts(name, "", "sa-a"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", ""),
			},
			{
				Config:           testAccRoleBindingV1Config_serviceAccounts(name, "", "sa-a", "sa-b"),
				ConfigPlanChecks: inPlaceUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subject.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", ""),
				),
			},
			{
				Config:           testAccRoleBindingV1Config_serviceAccounts(name, "team", "sa-a", "sa-b"),
				ConfigPlanChecks: inPlaceUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.team", "team"),
					resource.TestCheckResourceAttr(resourceName, "subject.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "subject.0.api_group", ""),
					resource.TestCheckResourceAttr(resourceName, "subject.1.api_group", ""),
				),
			},
		},
	})
}

// TestAccRoleBinding_noRefreshLabelUpdate reproduces the review scenario for
// skipping unchanged subjects: a subject is removed outside Terraform, then a
// label-only change is applied with -refresh=false. Update must patch only
// metadata, as SDKv2 did when !d.HasChange("subject"), so the apply succeeds
// and the out-of-band subject removal is left untouched.
func TestAccRoleBinding_noRefreshLabelUpdate(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"
	config := testAccRoleBindingV1Config_users(name, "notauser1", "notauser2")
	labeled := strings.Replace(config, `namespace = "default"`, "namespace = \"default\"\n    labels = {\n      team = \"x\"\n    }", 1)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		AdditionalCLIOptions: &resource.AdditionalCLIOptions{
			Plan: resource.PlanOptions{NoRefresh: true},
		},
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "subject.#", "2"),
					testAccRoleBindingKeepOnlyFirstSubject(name),
				),
			},
			{
				Config: labeled,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.team", "x"),
					testAccRoleBindingCheckLive(name, "x", "notauser1"),
				),
			},
		},
	})
}

// TestAccRoleBinding_identity verifies identity after create and an import
// block keyed by resource identity.
func TestAccRoleBinding_identity(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config:            testAccRoleBindingV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
			},
			{
				// Import blocks already assert a no-op plan and unchanged identity;
				// plugin-testing only supports ImportStateVerify for CLI imports.
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// TestAccRoleBinding_identityImportDefaultNamespace verifies that an identity
// import without namespace targets the default namespace.
func TestAccRoleBinding_identityImportDefaultNamespace(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")
	resourceName := "kubernetes_role_binding_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleBindingCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{Config: testAccRoleBindingV1Config_basic(name)},
			// Use exact configuration so the harness cannot reuse the stored namespace.
			{
				ResourceName:           resourceName,
				ImportState:            true,
				ImportStateKind:        resource.ImportBlockWithResourceIdentity,
				ImportStateConfigExact: true,
				Config:                 testAccRoleBindingV1Config_identityImport(name, "RoleBinding", "rbac.authorization.k8s.io/v1"),
			},
			{
				Config: testAccRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{testAccRoleBindingExpectIdentity(resourceName, name)},
			},
		},
	})
}

// TestAccRoleBinding_importInvalidIdentity verifies that an identity import
// with the wrong kind or api_version is rejected.
func TestAccRoleBinding_importInvalidIdentity(t *testing.T) {
	for _, tc := range []struct{ name, kind, apiVersion string }{
		{"wrongKind", "ClusterRoleBinding", "rbac.authorization.k8s.io/v1"},
		{"wrongAPIVersion", "RoleBinding", "rbac.authorization.k8s.io/v1beta1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-acc-rb")
			resource.ParallelTest(t, resource.TestCase{
				CheckDestroy:             testAccRoleBindingCheckDestroy,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_12_0),
				},
				Steps: []resource.TestStep{
					{Config: testAccRoleBindingV1Config_basic(name)},
					{
						ResourceName:           "kubernetes_role_binding_v1.test",
						ImportState:            true,
						ImportStateKind:        resource.ImportBlockWithResourceIdentity,
						ImportStateConfigExact: true,
						Config:                 testAccRoleBindingV1Config_identityImport(name, tc.kind, tc.apiVersion),
						ExpectError:            regexp.MustCompile("invalid identity import"),
					},
				},
			})
		})
	}
}

// TestAccRoleBinding_importInvalidID verifies that import IDs must be
// "namespace/name" with both parts non-empty.
func TestAccRoleBinding_importInvalidID(t *testing.T) {
	for _, tc := range []struct{ name, id string }{
		{"missingSlash", "reader"},
		{"extraSlash", "team/reader/extra"},
		{"emptyNamespace", "/reader"},
		{"emptyName", "team/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := acctest.RandomWithPrefix("tf-acc-rb")
			resource.ParallelTest(t, resource.TestCase{
				CheckDestroy:             testAccRoleBindingCheckDestroy,
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{Config: testAccRoleBindingV1Config_basic(name)},
					{
						ResourceName:  "kubernetes_role_binding_v1.test",
						ImportState:   true,
						ImportStateId: tc.id,
						ExpectError:   regexp.MustCompile("invalid resource ID"),
					},
				},
			})
		})
	}
}

// ── Validation Acceptance Tests (Plan-time checks) ────────────────────────────

// TestAccRoleBinding_validation_invalidAnnotationKey verifies that an invalid
// annotation key is rejected at plan time.
func TestAccRoleBinding_validation_invalidAnnotationKey(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// common.AnnotationsValidator uses validatordiag.InvalidAttributeValueDiagnostic:
				// summary "Invalid Attribute Value", detail "Attribute ... annotations[...] key ..., got: ..."
				Config:      testAccRoleBindingV1Config_invalidAnnotationKey(name),
				ExpectError: regexp.MustCompile(`(?i)Invalid Attribute Value`),
			},
		},
	})
}

// TestAccRoleBinding_validation_invalidLabelKey verifies that an invalid
// label key is rejected at plan time.
func TestAccRoleBinding_validation_invalidLabelKey(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// common.LabelsValidator uses validatordiag.InvalidAttributeValueDiagnostic:
				// summary "Invalid Attribute Value", detail "Attribute ... labels[...] key ..., got: ..."
				Config:      testAccRoleBindingV1Config_invalidLabelKey(name),
				ExpectError: regexp.MustCompile(`(?i)Invalid Attribute Value`),
			},
		},
	})
}

// TestAccRoleBinding_validation_invalidLabelValue verifies that an invalid
// label value is rejected at plan time.
func TestAccRoleBinding_validation_invalidLabelValue(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// common.LabelsValidator uses validatordiag.InvalidAttributeValueDiagnostic:
				// summary "Invalid Attribute Value", detail "Attribute ... labels[...] value ..., got: ..."
				Config:      testAccRoleBindingV1Config_invalidLabelValue(name),
				ExpectError: regexp.MustCompile(`(?i)Invalid Attribute Value`),
			},
		},
	})
}

// TestAccRoleBinding_validation_invalidName verifies that an invalid
// name (not a valid path segment) is rejected at plan time.
func TestAccRoleBinding_validation_invalidName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// common.RBACNameValidator uses validatordiag.InvalidAttributeValueDiagnostic:
				// summary "Invalid Attribute Value", detail "Attribute metadata[0].name ..., got: ..."
				Config:      testAccRoleBindingV1Config_basic("invalid/name"),
				ExpectError: regexp.MustCompile(`(?i)Invalid Attribute Value`),
			},
		},
	})
}

// TestAccRoleBinding_validation_missingMetadata verifies that omitting the
// metadata block is rejected at plan time.
func TestAccRoleBinding_validation_missingMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleBindingV1Config_missingMetadata(),
				ExpectError: regexp.MustCompile(`(?s)Block metadata must have a configuration value.*as required`),
			},
		},
	})
}

// TestAccRoleBinding_validation_missingRoleRef verifies that omitting the
// role_ref block is rejected at plan time.
func TestAccRoleBinding_validation_missingRoleRef(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleBindingV1Config_missingRoleRef(name),
				ExpectError: regexp.MustCompile(`(?s)Block role_ref must have a configuration value.*as required`),
			},
		},
	})
}

// TestAccRoleBinding_validation_missingSubject verifies that omitting the
// subject block is rejected at plan time.
func TestAccRoleBinding_validation_missingSubject(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rb")

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleBindingV1Config_missingSubject(name),
				ExpectError: regexp.MustCompile(`(?s)Block subject must have a configuration value.*as required`),
			},
		},
	})
}

// TestAccRoleBinding_validation_nameAndGenerateName verifies that setting both
// metadata.name and metadata.generate_name is rejected, including an empty
// generate_name. The Kubernetes API ignores generate_name whenever name is set.
func TestAccRoleBinding_validation_nameAndGenerateName(t *testing.T) {
	for _, tc := range []struct{ name, generateName string }{
		{"empty", ""},
		{"prefix", "tf-acc-rb-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      testAccRoleBindingV1Config_nameAndGenerateName(acctest.RandomWithPrefix("tf-acc-rb"), tc.generateName),
						ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination`),
					},
				},
			})
		})
	}
}

// TestAccRoleBinding_validation_nullMetadata verifies that a null label or
// annotation value is rejected at plan time. SDKv2 accepted it and silently
// dropped the key.
func TestAccRoleBinding_validation_nullMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleBindingV1Config_nullMetadata(acctest.RandomWithPrefix("tf-acc-rb")),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value.*value must be a string`),
			},
		},
	})
}

// ── Check helpers ─────────────────────────────────────────────────────────────

// testAccRoleBindingCheckDestroy verifies that every RoleBinding in state was
// deleted from the cluster.
func testAccRoleBindingCheckDestroy(state *terraform.State) error {
	conn, err := testAccRoleClient()
	if err != nil {
		return err
	}
	for _, module := range state.Modules {
		for _, rs := range module.Resources {
			if rs.Type != "kubernetes_role_binding_v1" && rs.Type != "kubernetes_role_binding" {
				continue
			}
			namespace, name, ok := strings.Cut(rs.Primary.ID, "/")
			if !ok {
				return fmt.Errorf("invalid RoleBinding ID %q", rs.Primary.ID)
			}
			_, err := conn.RbacV1().RoleBindings(namespace).Get(context.Background(), name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("checking destruction of RoleBinding %q: %w", rs.Primary.ID, err)
			}
			return fmt.Errorf("RoleBinding %q still exists", rs.Primary.ID)
		}
	}
	return nil
}

// testAccRoleBindingDeleteOutOfBand deletes the RoleBinding through the API,
// simulating a deletion made outside Terraform.
func testAccRoleBindingDeleteOutOfBand(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		return conn.RbacV1().RoleBindings("default").Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

// testAccRoleBindingKeepOnlyFirstSubject removes all but the first subject
// through the API, simulating a change made outside Terraform.
func testAccRoleBindingKeepOnlyFirstSubject(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		rb, err := conn.RbacV1().RoleBindings("default").Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		rb.Subjects = rb.Subjects[:1]
		_, err = conn.RbacV1().RoleBindings("default").Update(context.Background(), rb, metav1.UpdateOptions{})
		return err
	}
}

// testAccRoleBindingCheckLive verifies the live RoleBinding's team label and
// subject names, independent of Terraform state.
func testAccRoleBindingCheckLive(name, team string, subjects ...string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		rb, err := conn.RbacV1().RoleBindings("default").Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if rb.Labels["team"] != team {
			return fmt.Errorf("live label team = %q, want %q", rb.Labels["team"], team)
		}
		got := make([]string, 0, len(rb.Subjects))
		for _, s := range rb.Subjects {
			got = append(got, s.Name)
		}
		if strings.Join(got, ",") != strings.Join(subjects, ",") {
			return fmt.Errorf("live subjects = %v, want %v", got, subjects)
		}
		return nil
	}
}

func testAccRoleBindingExpectIdentity(resourceName, name string) statecheck.StateCheck {
	return statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
		"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
		"kind":        knownvalue.StringExact("RoleBinding"),
		"name":        knownvalue.StringExact(name),
		"namespace":   knownvalue.StringExact("default"),
	})
}

// ── HCL config helpers ────────────────────────────────────────────────────────

// testAccRoleBindingV1Config_users binds the admin Role to one User subject
// per name, in the given order.
func testAccRoleBindingV1Config_users(name string, users ...string) string {
	var subjects strings.Builder
	for _, user := range users {
		fmt.Fprintf(&subjects, `
  subject {
    kind      = "User"
    name      = %q
    api_group = "rbac.authorization.k8s.io"
  }
`, user)
	}
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }
%s}
`, name, subjects.String())
}

// testAccRoleBindingV1Config_serviceAccounts uses ServiceAccount subjects that
// omit api_group, so the computed value comes from the API. A non-empty team
// adds a "team" label.
func testAccRoleBindingV1Config_serviceAccounts(name, team string, serviceAccounts ...string) string {
	labels := ""
	if team != "" {
		labels = fmt.Sprintf("\n    labels = {\n      team = %q\n    }", team)
	}
	var subjects strings.Builder
	for _, sa := range serviceAccounts {
		fmt.Fprintf(&subjects, `
  subject {
    kind      = "ServiceAccount"
    name      = %q
    namespace = "default"
  }
`, sa)
	}
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %q
    namespace = "default"%s
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }
%s}
`, name, labels, subjects.String())
}

// testAccRoleBindingV1Config_identityImport adds an identity import block
// without namespace, so ImportState must default it.
func testAccRoleBindingV1Config_identityImport(name, kind, apiVersion string) string {
	return testAccRoleBindingV1Config_basic(name) + fmt.Sprintf(`
import {
  to = kubernetes_role_binding_v1.test
  identity = {
    name        = %q
    kind        = %q
    api_version = %q
  }
}
`, name, kind, apiVersion)
}

func testAccRoleBindingV1Config_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_updated(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
    labels = {
      managed-by = "terraform"
    }
    annotations = {
      "example.com/note" = "updated"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    namespace = "kube-system"
    api_group = ""
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_generateName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    generate_name = %[1]q
    namespace     = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, prefix)
}

func testAccRoleBindingV1Config_generateName_updated(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    generate_name = %[1]q
    namespace     = "default"
    labels = {
      team = "sec"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, prefix)
}

func testAccRoleBindingV1Config_saSubject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    namespace = "default"
    api_group = ""
  }
}
`, name)
}

func testAccRoleBindingV1Config_groupSubject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "Group"
    name      = "dev-team"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_emptySubjectNamespace(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
    namespace = ""
  }
}
`, name)
}

func testAccRoleBindingV1Config_emptyUserAPIGroup(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = ""
  }
}
`, name)
}

func testAccRoleBindingV1Config_clusterRoleRef(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_invalidAnnotationKey(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
    annotations = {
      "Not A Valid Key!" = "value"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_invalidLabelKey(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
    labels = {
      "Not A Valid Label Key!" = "value"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_invalidLabelValue(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
    labels = {
      "valid-key" = "Not A Valid Label Value!"
    }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_missingMetadata() string {
	return `
resource "kubernetes_role_binding_v1" "test" {
  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`
}

func testAccRoleBindingV1Config_missingRoleRef(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccRoleBindingV1Config_missingSubject(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name      = %[1]q
    namespace = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }
}
`, name)
}

func testAccRoleBindingV1Config_nameAndGenerateName(name, generateName string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name          = %[1]q
    generate_name = %[2]q
    namespace     = "default"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name, generateName)
}

func testAccRoleBindingV1Config_nullMetadata(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_binding_v1" "test" {
  metadata {
    name        = %[1]q
    namespace   = "default"
    labels      = { keep = "retained", optional = null }
    annotations = { keep = "retained", optional = null }
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = "admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}
