// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
)

// Keep imports on the same provider release as the preceding apply. In particular,
// importing the baseline with the local provider would invalidate the migration test.
func roleImportSteps(resourceName string, steps []resource.TestStep) []resource.TestStep {
	result := make([]resource.TestStep, 0, len(steps)*2)
	for i, step := range steps {
		result = append(result, step)
		if step.Config == "" || step.ExpectError != nil || step.PlanOnly || step.ImportState {
			continue
		}
		if i+1 < len(steps) && steps[i+1].ImportState {
			continue
		}
		name := resourceName
		if strings.Contains(step.Config, `resource "kubernetes_role" "test"`) {
			name = "kubernetes_role.test"
		}
		result = append(result, resource.TestStep{
			ResourceName:             name,
			ImportState:              true,
			ImportStateVerify:        true,
			ExternalProviders:        step.ExternalProviders,
			ProtoV6ProviderFactories: step.ProtoV6ProviderFactories,
		})
	}
	return result
}

func testAccRoleWithImports(t *testing.T, tc resource.TestCase) {
	t.Helper()
	tc.Steps = roleImportSteps("kubernetes_role_v1.test", tc.Steps)
	testAccRoleValidateAndRun(t, tc)
}

func testAccRoleValidateAndRun(t *testing.T, tc resource.TestCase) {
	t.Helper()
	for i, step := range tc.Steps {
		if step.Config == "" {
			continue
		}
		if _, diags := hclsyntax.ParseConfig([]byte(step.Config), "role.tf", hcl.InitialPos); diags.HasErrors() {
			t.Fatalf("step %d has invalid HCL: %s", i, diags)
		}
	}
	resource.ParallelTest(t, tc)
}

func testAccRoleClient() (*client.Clientset, error) {
	meta, err := sdkv2providerMeta()
	if err != nil {
		return nil, err
	}
	return meta().(kubernetes.KubeClientsets).MainClientset()
}

func TestRoleImportSteps(t *testing.T) {
	steps := roleImportSteps("kubernetes_role_v1.test", []resource.TestStep{
		{Config: `resource "kubernetes_role" "test" {}`, ExternalProviders: roleExternalProvider(rolePreIdentityProviderVersion)},
		{Config: `resource "kubernetes_role_v1" "test" {}`, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories},
		{Config: `resource "kubernetes_role" "test" {}`, ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories},
		{Config: "plan", PlanOnly: true},
		{Config: "error", ExpectError: regexp.MustCompile("expected")},
	})
	if len(steps) != 8 || !steps[1].ImportState || !steps[1].ImportStateVerify ||
		steps[1].ResourceName != "kubernetes_role.test" ||
		steps[1].ExternalProviders["kubernetes"].VersionConstraint != rolePreIdentityProviderVersion {
		t.Fatalf("incorrect import steps: %#v", steps)
	}
	for _, tc := range []struct {
		index int
		name  string
		alias bool
	}{
		{3, "kubernetes_role_v1.test", false},
		{5, "kubernetes_role.test", true},
	} {
		step := steps[tc.index]
		if !step.ImportState || !step.ImportStateVerify || step.ResourceName != tc.name {
			t.Fatalf("incorrect import step: %#v", step)
		}
		testRoleProviderSchema(t, step.ProtoV6ProviderFactories, tc.alias)
	}
}

func TestRoleProviderFactories(t *testing.T) {
	t.Run("framework", func(t *testing.T) {
		testRoleProviderSchema(t, testAccProtoV6ProviderFactories, false)
	})
	t.Run("mux", func(t *testing.T) {
		testRoleProviderSchema(t, testAccMuxProtoV6ProviderFactories, true)
	})
}

func testRoleProviderSchema(t *testing.T, factories map[string]func() (tfprotov6.ProviderServer, error), alias bool) {
	t.Helper()
	for _, key := range []string{"KUBE_CONFIG_PATH", "KUBE_CONFIG_PATHS", "KUBE_CTX", "KUBE_CTX_AUTH_INFO", "KUBE_CTX_CLUSTER", "KUBE_HOST", "KUBE_CLIENT_CERT_DATA", "KUBE_CLIENT_KEY_DATA", "KUBE_CLUSTER_CA_CERT_DATA"} {
		t.Setenv(key, "")
	}
	server, err := factories["kubernetes"]()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diag := range resp.Diagnostics {
		if diag.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("provider schema: %s: %s", diag.Summary, diag.Detail)
		}
	}
	if resp.ResourceSchemas["kubernetes_role_v1"] == nil {
		t.Fatal("provider does not register kubernetes_role_v1")
	}
	if got := resp.ResourceSchemas["kubernetes_role"] != nil; got != alias {
		t.Fatalf("kubernetes_role registration = %t, want %t", got, alias)
	}
}

func TestRoleProviderFactoryConfigureError(t *testing.T) {
	t.Setenv("KUBE_CONFIG_PATH", "")
	t.Setenv("KUBE_CONFIG_PATHS", "")
	t.Setenv("KUBE_HOST", "https://[invalid")
	server, err := testAccProtoV6ProviderFactories["kubernetes"]()
	if err == nil || !strings.Contains(err.Error(), "Failed to parse value for host") || server != nil {
		t.Fatalf("expected configuration error and no server, got server %T, error %v", server, err)
	}
}

func TestRoleImportState(t *testing.T) {
	ctx := context.Background()
	r := &rbacv1.RoleV1{}
	var schema frameworkresource.SchemaResponse
	r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
	for _, tc := range []struct {
		name, id, namespace, want string
		identity, wantError       bool
	}{
		{name: "id", id: "team/reader", want: "team/reader"},
		{name: "emptyNamespaceID", id: "/reader", want: "/reader"},
		{name: "invalidID", id: "reader", wantError: true},
		{name: "missingImport", wantError: true},
		{name: "identity", identity: true, namespace: "team", want: "team/reader"},
		{name: "defaultNamespace", identity: true, want: "default/reader"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := frameworkresource.ImportStateRequest{ID: tc.id}
			if tc.identity {
				s := common.NamespacedIdentitySchema()
				req.Identity = &tfsdk.ResourceIdentity{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
				namespace := types.StringNull()
				if tc.namespace != "" {
					namespace = types.StringValue(tc.namespace)
				}
				diags := req.Identity.Set(ctx, common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{Name: types.StringValue("reader"), Kind: types.StringValue("Role"), APIVersion: types.StringValue("rbac.authorization.k8s.io/v1")},
					Namespace:        namespace,
				})
				if diags.HasError() {
					t.Fatal(diags)
				}
			}
			resp := frameworkresource.ImportStateResponse{State: tfsdk.State{
				Schema: schema.Schema, Raw: tftypes.NewValue(schema.Schema.Type().TerraformType(ctx), nil),
			}}
			r.ImportState(ctx, req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantError {
				return
			}
			var id types.String
			if diags := resp.State.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
				t.Fatal(diags)
			}
			if id.ValueString() != tc.want {
				t.Fatalf("ID = %q, want %q", id.ValueString(), tc.want)
			}
		})
	}
}

func TestAccRole_basic(t *testing.T) {
	// Colons are deliberate: RBAC names are path segments, not DNS subdomains, and SDKv2
	// validates them with validateRBACNameFunc (schema_rbac.go). "system:controller:foo" is
	// a real ClusterRole name, so this fixture guards that the RBAC override is in place.
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	testAccRoleWithImports(t, resource.TestCase{
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
		},
	})
}

func TestAccRole_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	resourceName := "kubernetes_role_v1.test"
	var uid string

	testAccRoleWithImports(t, resource.TestCase{
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
				Config: strings.Replace(testAccRoleConfig_generatedName(prefix), "generate_name =", "labels = { added = \"yes\" }\n    generate_name =", 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionUpdate)},
				},
				Check: testAccRoleCheckExists(resourceName, &uid),
			},
		},
	})
}

func TestAccRole_metadataUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	testAccRoleWithImports(t, resource.TestCase{
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
				Config: testAccRoleConfig_labels(name, "updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.0.labels.test", "updated"),
				),
			},
		},
	})
}

func TestAccRole_resourceNames(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	testAccRoleWithImports(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "rule.0.resource_names.#", "0"),
				),
			},
		},
	})
}

func TestAccRole_ruleTransitions(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	testAccRoleWithImports(t, resource.TestCase{
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
		},
	})
}

func TestAccRole_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	resourceName := "kubernetes_role_v1.test"

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy:             testAccRoleCheckDestroy,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccRoleConfig_noResourceNames(name),
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
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateKind:   resource.ImportBlockWithResourceIdentity,
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
				),
			},
			// Import by identity WITH explicit namespace — must succeed.
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateKind:   resource.ImportBlockWithResourceIdentity,
			},
		},
	})
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
