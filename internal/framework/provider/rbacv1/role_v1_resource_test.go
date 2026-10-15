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
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type roleUpdateClientsets struct {
	kubernetes.KubeClientsets
	client *client.Clientset
}

func (c roleUpdateClientsets) MainClientset() (*client.Clientset, error) {
	return c.client, nil
}

func TestRoleReadMissingAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		for _, storedIdentity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/identity=%t", status, storedIdentity), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodGet || req.URL.Path != "/apis/rbac.authorization.k8s.io/v1/namespaces/team/roles/reader" {
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
				r := &rbacv1.RoleV1{SDKv2Meta: func() any { return roleUpdateClientsets{client: conn} }}
				var schema frameworkresource.SchemaResponse
				r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
				raw := tfprotov6.RawState{JSON: []byte(`{
					"id":"team/reader",
					"metadata":[{"name":"reader","namespace":"team","uid":"uid-1","resource_version":"10",
					"generation":1,"labels":null,"annotations":null,"generate_name":null}],
					"rule":[{"api_groups":[""],"resources":["pods"],"verbs":["get"],"resource_names":[]}]
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
						Name: types.StringValue("reader"), Kind: types.StringValue("Role"),
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
						t.Fatalf("missing Role must be removed: %v", response.Diagnostics)
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

func testAccRoleCheckImportOwnership(internalMetadata string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if internalMetadata == "" {
			return nil
		}
		if len(states) != 1 {
			return fmt.Errorf("expected one imported Role, got %d", len(states))
		}
		prefix := "metadata.0." + internalMetadata + "."
		if _, ok := states[0].Attributes[prefix+"example.kubernetes.io/owner"]; ok {
			return fmt.Errorf("import unexpectedly claimed ownership of internal %s", internalMetadata)
		}
		if count := states[0].Attributes[prefix+"%"]; count != "" && count != "0" {
			return fmt.Errorf("imported %s count = %q, want empty", internalMetadata, count)
		}
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		role, err := conn.RbacV1().Roles(states[0].Attributes["metadata.0.namespace"]).Get(
			context.Background(), states[0].Attributes["metadata.0.name"], metav1.GetOptions{})
		if err != nil {
			return err
		}
		values := role.Labels
		if internalMetadata == "annotations" {
			values = role.Annotations
		}
		if values["example.kubernetes.io/owner"] != "terraform" || string(role.UID) != states[0].Attributes["metadata.0.uid"] {
			return fmt.Errorf("import changed live Role identity or internal %s", internalMetadata)
		}
		return nil
	}
}

func testAccRoleClient() (*client.Clientset, error) {
	meta, err := sdkv2providerMeta()
	if err != nil {
		return nil, err
	}
	return meta().(kubernetes.KubeClientsets).MainClientset()
}

func TestRoleImportOwnership(t *testing.T) {
	for _, field := range []string{"labels", "annotations"} {
		t.Run(field, func(t *testing.T) {
			prefix := "metadata.0." + field + "."
			for _, attrs := range []map[string]string{
				{prefix + "example.kubernetes.io/owner": "terraform"},
				{prefix + "%": "1"},
			} {
				if err := testAccRoleCheckImportOwnership(field)([]*terraform.InstanceState{{Attributes: attrs}}); err == nil {
					t.Fatalf("import checker accepted unexpected internal metadata: %v", attrs)
				}
			}
		})
	}
}

func TestAccRole_ruleSetValidation(t *testing.T) {
	for _, field := range []string{"api_groups", "resources", "resource_names", "verbs"} {
		for _, tc := range []struct {
			name, value string
			wantError   bool
		}{
			{"mixedNull", `["value", null]`, true},
			{"onlyNull", `[null]`, true},
			{"known", `["value"]`, false},
			{"emptyString", `[""]`, false},
			{"emptySet", `[]`, false},
			{"unknownElement", `["value", kubernetes_role_v1.source.metadata[0].uid]`, false},
			{"unknownSet", `split(",", kubernetes_role_v1.source.metadata[0].uid)`, false},
			{"nullSet", `null`, false},
		} {
			t.Run(field+"/"+tc.name, func(t *testing.T) {
				name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
				config := testAccRoleConfig_noResourceNames(name)
				replacement := regexp.MustCompile(`(?m)^\s*` + field + `\s*=.*$`)
				if field == "resource_names" {
					config = strings.Replace(config, "rule {", "rule {\n    resource_names = "+tc.value, 1)
				} else {
					config = replacement.ReplaceAllString(config, "\n    "+field+" = "+tc.value)
				}
				if strings.Contains(tc.value, "kubernetes_role_v1.source") {
					config += strings.Replace(testAccRoleConfig_noResourceNames(name+"-source"), `"test"`, `"source"`, 1)
				}
				step := resource.TestStep{
					Config:             config,
					PlanOnly:           true,
					ExpectNonEmptyPlan: !tc.wantError,
				}
				if tc.wantError {
					step.ExpectError = regexp.MustCompile("Null Set Value")
				}
				if tc.name == "nullSet" && field != "resource_names" {
					// The element validator permits a null collection; the
					// enclosing required attribute correctly rejects it.
					step.ExpectNonEmptyPlan = false
					step.ExpectError = regexp.MustCompile("Missing Configuration for Required Attribute")
				}
				// Empty strings and sets are schema-valid even when Kubernetes
				// would reject a particular rule; validate these at plan time.
				resource.ParallelTest(t, resource.TestCase{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Steps:                    []resource.TestStep{step},
				})
			})
		}
	}
}

func TestAccRole_noOpUpdate(t *testing.T) {
	name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "kubernetes_role_v1.test"
	resourceVersionUnchanged := statecheck.CompareValue(compare.ValuesSame())
	resourceVersion := tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("resource_version")
	config := strings.Replace(testAccRoleConfig_noResourceNames(name), "metadata {", "metadata {\n    labels = {}", 1)
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
				ConfigStateChecks: []statecheck.StateCheck{
					resourceVersionUnchanged.AddStateValue(resourceName, resourceVersion),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// null -> {} manages no keys: Update must GET without patching.
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					resourceVersionUnchanged.AddStateValue(resourceName, resourceVersion),
				},
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					resourceVersionUnchanged.AddStateValue(resourceName, resourceVersion),
				},
			},
		},
	})
}

func TestAccRole_importInvalidID(t *testing.T) {
	for _, tc := range []struct{ name, id string }{
		{"missingSlash", "reader"},
		{"extraSlash", "team/reader/extra"},
		{"emptyNamespace", "/reader"},
		{"emptyName", "team/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccRoleCheckDestroy,
				Steps: []resource.TestStep{
					{Config: testAccRoleConfig_noResourceNames(name)},
					{
						ResourceName:  "kubernetes_role_v1.test",
						ImportState:   true,
						ImportStateId: tc.id,
						ExpectError:   regexp.MustCompile("invalid resource ID"),
					},
				},
			})
		})
	}
}

func TestAccRole_importInvalidIdentity(t *testing.T) {
	for _, tc := range []struct{ name, old, replacement string }{
		{"wrongKind", `"Role"`, `"ClusterRole"`},
		{"wrongAPIVersion", `"rbac.authorization.k8s.io/v1"`, `"rbac.authorization.k8s.io/v1beta1"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccRoleCheckDestroy,
				TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
				Steps: []resource.TestStep{
					{Config: testAccRoleConfig_noResourceNames(name)},
					{
						ResourceName:           "kubernetes_role_v1.test",
						ImportState:            true,
						ImportStateKind:        resource.ImportBlockWithResourceIdentity,
						ImportStateConfigExact: true,
						Config:                 strings.Replace(testAccRoleConfig_identityImportDefaultNamespace(name), tc.old, tc.replacement, 1),
						ExpectError:            regexp.MustCompile("invalid identity import"),
					},
				},
			})
		})
	}
}

func TestAccRole_basic(t *testing.T) {
	// Colons are deliberate: RBAC names are path segments, not DNS subdomains, and SDKv2
	// validates them with validateRBACNameFunc (schema_rbac.go). "system:controller:foo" is
	// a real ClusterRole name, so this fixture guards that the RBAC override is in place.
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "3"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.verbs.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "rule.1.resource_names.#", "0"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.api_groups.*", "core"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "watch"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resource_names.*", "foo"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.api_groups.*", "apps"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "list"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRoleConfig_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, nil),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.Different", "1234"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.annotations.TestAnnotationTwo"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "2"),
					resource.TestCheckNoResourceAttr(resourceName, "metadata.0.labels.TestLabelTwo"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "rule.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.api_groups.*", "batch"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "jobs"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.api_groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resources.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.verbs.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "0"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "watch"),
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

func TestAccRole_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "kubernetes_role_v1.test"
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_generatedName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccRoleCheckExists(resourceName, &uid),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.annotations.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.%", "0"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.generate_name", prefix),
					resource.TestMatchResourceAttr(resourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttrSet(resourceName, "metadata.0.uid"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: strings.Replace(testAccRoleConfig_generatedName(prefix), "generate_name =", "labels = { added = \"yes\" }\n    generate_name =", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: testAccRoleCheckExists(resourceName, &uid),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRole_metadataUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_labels(name, "acceptance"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "acceptance"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRoleConfig_labels(name, "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "updated"),
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

func TestAccRole_resourceNames(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "0"),
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

func TestAccRole_ruleTransitions(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_ruleTransitionsStep0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.#", "3"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.2.resources.*", "cronjobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.2.verbs.*", "list"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRoleConfig_ruleTransitionsStep1(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "get"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.resources.*", "jobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "get"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccRoleConfig_ruleTransitionsStep2(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.#", "4"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.resources.*", "pods"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.0.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.resources.*", "deployments"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.1.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.2.resources.*", "cronjobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.2.verbs.*", "list"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.3.resources.*", "jobs"),
					resource.TestCheckTypeSetElemAttr(resourceName, "rule.3.verbs.*", "get"),
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

func TestAccRole_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check:  testAccRoleCheckExists(resourceName, &uid),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						resourceName,
						map[string]knownvalue.Check{
							"namespace":   knownvalue.StringExact("default"),
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
							"kind":        knownvalue.StringExact("Role"),
						},
					),
				},
			},
			{
				// Import blocks already assert a no-op plan and unchanged identity;
				// plugin-testing only supports ImportStateVerify for CLI imports.
				ResourceName:    resourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				Config: testAccRoleConfig_noResourceNames(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: testAccRoleCheckExists(resourceName, &uid),
			},
		},
	})
}

func TestAccRole_nameAndGenerateName(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	// Setting both is rejected. SDKv2's namespaced schema declared the conflict against an
	// unresolvable path so it never fired, but the combination was never meaningful — the
	// API ignores generate_name whenever name is present. See the note on
	// TestAccRole_movedFromAlias_nameAndGenerateName for the upgrade impact.
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_nameAndGenerateName(name),
				ExpectError: regexp.MustCompile("(?s)Invalid Attribute Combination"),
			},
		},
	})
}
func TestAccRole_nullMetadata(t *testing.T) {
	name := "tf-acc-test-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	// A null map value is rejected at plan, matching SDKv2's validateLabels and
	// validateAnnotations. Role previously accepted it and silently dropped the key, so
	// every later plan showed the key being added back.
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_nullMetadata(name, "null"),
				ExpectError: regexp.MustCompile("(?s)Invalid Attribute Value.*value must be a string"),
			},
		},
	})
}

func testAccRoleConfig_nullMetadata(name, value string) string {
	config := fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name        = %q
    labels      = { keep = "retained", optional = null }
    annotations = { keep = "retained", optional = null }
  }
  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name)
	return strings.ReplaceAll(config, "optional = null", "optional = "+value)
}

func TestAccRole_missingRule(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_noRule(name),
				ExpectError: regexp.MustCompile(`(?s)rule.*(required|at least 1)`),
			},
		},
	})
}

func TestAccRole_missingMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_noMetadata(),
				ExpectError: regexp.MustCompile(`(?s)metadata.*(required|at least 1)`),
			},
		},
	})
}

func TestAccRole_duplicateMetadata(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_duplicateMetadata(name),
				ExpectError: regexp.MustCompile(`(?i)at most 1`),
			},
		},
	})
}

func TestAccRole_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					testAccRoleDeleteOutOfBand(name),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check:  testAccRoleCheckExists(resourceName, &uid),
			},
			{
				Config: testAccRoleConfig_noResourceNames(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: testAccRoleCheckExists(resourceName, &uid),
			},
		},
	})
}

func testAccRoleDeleteOutOfBand(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		return conn.RbacV1().Roles("default").Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

func testAccRoleConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
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

  rule {
    api_groups     = ["core"]
    resources      = ["pods"]
    resource_names = ["foo"]
    verbs          = ["get", "list", "watch"]
  }

  rule {
    api_groups = ["apps"]
    resources  = ["deployments"]
    verbs      = ["get", "list"]
  }
}
`, name)
}

func testAccRoleConfig_modified(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
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

  rule {
    api_groups = ["batch"]
    resources  = ["jobs"]
    verbs      = ["watch"]
  }
}
`, name)
}

func testAccRoleConfig_generatedName(prefix string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
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

func testAccRoleConfig_labels(name, value string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
    labels = {
      test = %[2]q
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name, value)
}

func testAccRoleConfig_noResourceNames(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
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

func testAccRoleConfig_ruleTransitionsStep0(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
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
    api_groups = [""]
    resources  = ["cronjobs"]
    verbs      = ["list"]
  }
}
`, name)
}

func testAccRoleConfig_ruleTransitionsStep1(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
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

func testAccRoleConfig_ruleTransitionsStep2(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
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
    api_groups = [""]
    resources  = ["cronjobs"]
    verbs      = ["list"]
  }

  rule {
    api_groups = [""]
    resources  = ["jobs"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccRoleConfig_nameAndGenerateName(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name          = %[1]q
    generate_name = %[1]q
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`, name)
}

func testAccRoleConfig_noRule(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }
}
`, name)
}

func testAccRoleConfig_noMetadata() string {
	return `
resource "kubernetes_role_v1" "test" {
  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`
}

func testAccRoleConfig_duplicateMetadata(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
  metadata {
    name = %[1]q
  }
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

func TestAccRole_identityImportDefaultNamespace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"
	var uid string

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			// Create the role so it exists in the cluster.
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
					testAccRoleCheckExists(resourceName, &uid),
				),
			},
			// Use exact configuration so the harness cannot reuse the stored namespace.
			{
				ResourceName:           resourceName,
				ImportState:            true,
				ImportStateKind:        resource.ImportBlockWithResourceIdentity,
				ImportStateConfigExact: true,
				Config:                 testAccRoleConfig_identityImportDefaultNamespace(name),
			},
			{
				Config: testAccRoleConfig_noResourceNames(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", "default/"+name),
					resource.TestCheckResourceAttr(resourceName, "metadata.0.namespace", "default"),
					testAccRoleCheckExists(resourceName, &uid),
				),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(resourceName, map[string]knownvalue.Check{
						"namespace":   knownvalue.StringExact("default"),
						"name":        knownvalue.StringExact(name),
						"kind":        knownvalue.StringExact("Role"),
						"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
					}),
				},
			},
		},
	})
}

func testAccRoleConfig_identityImportDefaultNamespace(name string) string {
	return testAccRoleConfig_noResourceNames(name) + fmt.Sprintf(`
import {
  to = kubernetes_role_v1.test
  identity = {
    name        = %q
    kind        = "Role"
    api_version = "rbac.authorization.k8s.io/v1"
  }
}
`, name)
}

func TestAccRole_invalidAnnotationKey(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_badAnnotationKey(),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value.*annotations\[`),
			},
		},
	})
}

func TestAccRole_invalidLabelValue(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccRoleConfig_badLabelValue(),
				ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value.*labels\[`),
			},
		},
	})
}

func testAccRoleConfig_badAnnotationKey() string {
	return `
resource "kubernetes_role_v1" "test" {
  metadata {
    name = "bad-annotation-role"
    annotations = {
      "Not A Valid Key!" = "value"
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`
}

func testAccRoleConfig_badLabelValue() string {
	return `
resource "kubernetes_role_v1" "test" {
  metadata {
    name = "bad-label-role"
    labels = {
      "env" = "value with spaces and !"
    }
  }

  rule {
    api_groups = [""]
    resources  = ["pods"]
    verbs      = ["get"]
  }
}
`
}

func testAccRoleCheckDestroy(state *terraform.State) error {
	conn, err := testAccRoleClient()
	if err != nil {
		return err
	}
	for _, module := range state.Modules {
		for _, rs := range module.Resources {
			if rs.Type != "kubernetes_role_v1" && rs.Type != "kubernetes_role" {
				continue
			}
			namespace, name, ok := strings.Cut(rs.Primary.ID, "/")
			if !ok {
				return fmt.Errorf("invalid Role ID %q", rs.Primary.ID)
			}
			_, err := conn.RbacV1().Roles(namespace).Get(context.Background(), name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				return fmt.Errorf("checking destruction of Role %q: %w", rs.Primary.ID, err)
			}
			return fmt.Errorf("Role %q still exists", rs.Primary.ID)
		}
	}
	return nil
}

func testAccRoleCheckExists(address string, priorUID *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %q not found in state", address)
		}
		namespace, name, ok := strings.Cut(rs.Primary.ID, "/")
		if !ok {
			return fmt.Errorf("invalid Role ID %q", rs.Primary.ID)
		}
		conn, err := testAccRoleClient()
		if err != nil {
			return err
		}
		role, err := conn.RbacV1().Roles(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		uid := string(role.UID)
		if uid == "" || rs.Primary.Attributes["metadata.0.uid"] != uid {
			return fmt.Errorf("Role %q has mismatched API and state UID", rs.Primary.ID)
		}
		for field, values := range map[string]map[string]string{"labels": role.Labels, "annotations": role.Annotations} {
			key := "example.kubernetes.io/owner"
			if expected, declared := rs.Primary.Attributes["metadata.0."+field+"."+key]; declared && values[key] != expected {
				return fmt.Errorf("Role %q internal %s differs between API and state", rs.Primary.ID, field)
			}
		}
		if priorUID != nil {
			if *priorUID != "" && *priorUID != uid {
				return fmt.Errorf("Role %q was replaced: UID changed from %q to %q", rs.Primary.ID, *priorUID, uid)
			}
			*priorUID = uid
		}
		return nil
	}
}

func testAccRoleConfig_names(name string) string {
	return fmt.Sprintf(`
resource "kubernetes_role_v1" "test" {
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
