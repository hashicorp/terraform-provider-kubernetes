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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const clusterRoleBindingResourceName = "kubernetes_cluster_role_binding_v1.test"

// TestClusterRoleBindingReadMissingAndErrors runs Read for each API status
// (404, 403, 500), with and without a stored identity:
//
//   - 404 without identity (Terraform 1.11 and earlier): state is removed and
//     Read builds the identity from the state ID.
//   - 404 with identity (Terraform 1.12 and later): state is removed and the
//     stored identity is kept, not rebuilt.
//   - 403 and 500: Read returns an error diagnostic and leaves state and
//     identity unchanged, so an API failure never drops the resource.
//
// It uses a fake API server so it runs without a cluster on every PR. CI
// acceptance tests run Terraform 1.12 or later, which always sends identity,
// so TestAccFrameworkClusterRoleBinding_disappears cannot catch the missing
// identity case there. Every case fails before Read applies metadata filters,
// so Role's roleUpdateClientsets is enough.
func TestClusterRoleBindingReadMissingAndErrors(t *testing.T) {
	ctx := context.Background()
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		for _, storedIdentity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/identity=%t", status, storedIdentity), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					if req.Method != http.MethodGet || req.URL.Path != "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings/reader" {
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
				r := &rbacv1.ClusterRoleBinding{SDKv2Meta: func() any { return roleUpdateClientsets{client: conn} }}
				var schema frameworkresource.SchemaResponse
				r.Schema(ctx, frameworkresource.SchemaRequest{}, &schema)
				raw := tfprotov6.RawState{JSON: []byte(`{
					"id":"reader",
					"metadata":[{"name":"reader","uid":"uid-1","resource_version":"10",
					"generation":1,"labels":null,"annotations":null,"generate_name":null}],
					"role_ref":[{"api_group":"rbac.authorization.k8s.io","kind":"ClusterRole","name":"view"}],
					"subject":[{"api_group":"","kind":"User","name":"alice","namespace":"default"}]
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
					Kind:       types.StringValue("ClusterRoleBinding"),
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
						t.Fatalf("missing ClusterRoleBinding must be removed: %v", response.Diagnostics)
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

func TestAccFrameworkClusterRoleBinding_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "default"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", ""),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "Group"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				// role_ref change forces replacement (ForceNew).
				Config: testAccKubernetesClusterRoleBindingV1Config_modifiedRoleRef(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccFrameworkClusterRoleBinding_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_generateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

// TestAccFrameworkClusterRoleBinding_metadataUpdate moves one binding through
// no metadata maps, empty maps, managed labels and annotations, and back to
// none, checking that every change is applied in place.
func TestAccFrameworkClusterRoleBinding_metadataUpdate(t *testing.T) {
	name := "tf-acc-test:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	checkUID := testAccClusterRoleBindingCheckSameUID()
	steps := []resource.TestStep{}
	for _, metadata := range []string{"", "labels = {}\nannotations = {}",
		`labels = { "app.kubernetes.io/name" = "managed" }
annotations = { "example.com/owner" = "terraform" }`, ""} {
		steps = append(steps, resource.TestStep{
			Config: testAccKubernetesClusterRoleBindingV1Config_metadata(name, metadata),
			Check:  checkUID,
		})
	}
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps:                    steps,
	})
}

// TestAccFrameworkClusterRoleBinding_generatedNameMetadataUpdate verifies that
// adding a label to a binding created with generate_name updates it in place.
func TestAccFrameworkClusterRoleBinding_generatedNameMetadataUpdate(t *testing.T) {
	prefix := "tf-acc-test:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum) + "-"
	config := testAccKubernetesClusterRoleBindingV1Config_generateName(prefix)
	updated := strings.Replace(config, "metadata {", `metadata {
    labels = { "managed" = "true" }`, 1)
	checkUID := testAccClusterRoleBindingCheckSameUID()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{Config: config, Check: checkUID},
			{
				Config: updated,
				Check:  checkUID,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate),
					},
				},
			},
		},
	})
}

func TestAccFrameworkClusterRoleBinding_serviceAccountSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_serviceAccountSubject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", ""),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "someservice"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "ServiceAccount"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccFrameworkClusterRoleBinding_groupSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_groupSubject(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "somegroup"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "Group"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

// TestAccFrameworkClusterRoleBinding_emptySubjectNamespace verifies that an
// explicitly empty subject namespace is kept as-is and does not produce a
// diff, while an omitted namespace still takes the schema default.
func TestAccFrameworkClusterRoleBinding_emptySubjectNamespace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_emptySubjectNamespace(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "2"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "explicit-empty"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.namespace", ""),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "omitted"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.namespace", "default"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func TestAccFrameworkClusterRoleBinding_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_12_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_basic(name),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(
						clusterRoleBindingResourceName, map[string]knownvalue.Check{
							"name":        knownvalue.StringExact(name),
							"api_version": knownvalue.StringExact("rbac.authorization.k8s.io/v1"),
							"kind":        knownvalue.StringExact("ClusterRoleBinding"),
						},
					),
				},
			},
			{
				ResourceName:    clusterRoleBindingResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
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

func testAccKubernetesClusterRoleBindingV1Config_generateName(namePrefix string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    generate_name = "%s"
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
`, namePrefix)
}

// testAccKubernetesClusterRoleBindingV1Config_metadata is the basic
// configuration with extra lines inserted into the metadata block.
func testAccKubernetesClusterRoleBindingV1Config_metadata(name, metadata string) string {
	return strings.Replace(testAccKubernetesClusterRoleBindingV1Config_basic(name),
		fmt.Sprintf("name = %q", name), fmt.Sprintf("name = %q\n%s", name, metadata), 1)
}

func testAccKubernetesClusterRoleBindingV1Config_modified(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
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

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_modifiedRoleRef(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
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
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_serviceAccountSubject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind = "ServiceAccount"
    name = "someservice"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_groupSubject(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Group"
    name      = "somegroup"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_emptySubjectNamespace(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "explicit-empty"
    namespace = ""
  }

  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "omitted"
  }
}
`, name)
}

// TestAccFrameworkClusterRoleBinding_ignoreAnnotations verifies that:
//   - provider-level ignore_annotations prevents drift from untracked
//     server-side annotations matching the pattern
//   - annotations the user has explicitly configured are retained even
//     when their key matches an ignore pattern
func TestAccFrameworkClusterRoleBinding_ignoreAnnotations(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				// Step 1 — create a binding with an annotation that is
				// tracked in config. The provider is configured to ignore
				// the same pattern so that any server-added keys with that
				// prefix are transparently dropped.
				Config: testAccKubernetesClusterRoleBindingV1Config_ignoreAnnotations(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					// The user-configured annotation must appear in state.
					resource.TestCheckResourceAttr(
						clusterRoleBindingResourceName,
						"metadata.0.annotations.acme.io/managed",
						"true",
					),
				),
			},
			{
				// Step 2 — re-plan; must be empty (no phantom diff from
				// internal Kubernetes annotations or the ignored pattern).
				Config:             testAccKubernetesClusterRoleBindingV1Config_ignoreAnnotations(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_ignoreAnnotations(name string) string {
	return fmt.Sprintf(`
provider "kubernetes" {
  ignore_annotations = ["^acme\\.io/"]
}

resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = %q
    annotations = {
      "acme.io/managed" = "true"
    }
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

// TestAccFrameworkClusterRoleBinding_subjectOrderWithRemovals is the
// Framework port of the SDKv2
// TestAccKubernetesClusterRoleBindingV1_UpdatePatchOperationsOrderWithRemovals
// test. It verifies that removing a subject, then re-adding subjects in a
// different order, produces the exact expected list with no stale entries.
// This validates the single replace /subjects operation instead of the
// error-prone per-index patch approach.
func TestAccFrameworkClusterRoleBinding_subjectOrderWithRemovals(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				// Step 1 — three subjects.
				Config: testAccKubernetesClusterRoleBindingV1Config_modified(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "Group"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				// Step 2 — remove the ServiceAccount; two subjects remain.
				Config: testAccKubernetesClusterRoleBindingV1Config_subjectRemoval(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "2"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "Group"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
			{
				// Step 3 — add subjects back in a different order.
				Config: testAccKubernetesClusterRoleBindingV1Config_subjectReorder(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "Group"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "User"),
				),
			},
			{
				ResourceName:            clusterRoleBindingResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"metadata.0.resource_version"},
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_subjectRemoval(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
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

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

func testAccKubernetesClusterRoleBindingV1Config_subjectReorder(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "%s"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "Group"
    name      = "system:masters"
    api_group = "rbac.authorization.k8s.io"
  }

  subject {
    kind      = "ServiceAccount"
    name      = "default"
    api_group = ""
    namespace = "kube-system"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`, name)
}

// TestAccFrameworkClusterRoleBinding_invalidMetadataName verifies that a
// metadata.name containing a path separator is rejected during Terraform
// validation, matching the SDKv2 validateRBACNameFunc behavior.
func TestAccFrameworkClusterRoleBinding_invalidMetadataName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_invalidName(),
				ExpectError: regexp.MustCompile(`may not contain`),
			},
		},
	})
}

// TestAccFrameworkClusterRoleBinding_invalidMetadataGenerateName verifies
// that an invalid generate_name prefix is also rejected during validation.
func TestAccFrameworkClusterRoleBinding_invalidMetadataGenerateName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_invalidGenerateName(),
				ExpectError: regexp.MustCompile(`may not contain`),
			},
		},
	})
}

// TestAccFrameworkClusterRoleBinding_metadataValidation verifies that invalid
// metadata is rejected at plan time and that a valid configuration still
// applies afterwards.
func TestAccFrameworkClusterRoleBinding_metadataValidation(t *testing.T) {
	for _, tc := range []struct{ name, metadata, errorPattern string }{
		{"nullLabel", `labels = { x = null }`, "value must be a string"},
		{"nullAnnotation", `annotations = { x = null }`, "value must be a string"},
		{"invalidLabelKey", `labels = { "invalid/key/extra" = "value" }`, "qualified name"},
		{"invalidLabelValue", `labels = { x = "invalid value" }`, "valid label must be an empty string"},
		{"invalidAnnotationKey", `annotations = { "invalid/key/extra" = "value" }`, "qualified name"},
		{"nameConflict", `generate_name = "tf-acc:"`, "cannot be specified when"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "tf-acc-test:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
				Steps: []resource.TestStep{
					{Config: testAccKubernetesClusterRoleBindingV1Config_metadata(name, tc.metadata), ExpectError: regexp.MustCompile(tc.errorPattern)},
					{Config: testAccKubernetesClusterRoleBindingV1Config_metadata(name, "")},
				},
			})
		})
	}
}

// TestAccFrameworkClusterRoleBinding_missingMetadata verifies that omitting
// the required metadata block is rejected during Terraform validation by
// listvalidator.IsRequired(), rather than deferring the failure to apply.
func TestAccFrameworkClusterRoleBinding_missingMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_missingMetadata(),
				ExpectError: regexp.MustCompile(`must have a configuration value`),
			},
		},
	})
}

// TestAccFrameworkClusterRoleBinding_missingRoleRef verifies that omitting
// the required role_ref block is rejected during Terraform validation by
// listvalidator.IsRequired().
func TestAccFrameworkClusterRoleBinding_missingRoleRef(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_missingRoleRef(),
				ExpectError: regexp.MustCompile(`must have a configuration value`),
			},
		},
	})
}

// TestAccFrameworkClusterRoleBinding_missingSubject verifies that omitting
// the required subject block is rejected during Terraform validation by
// listvalidator.IsRequired().
func TestAccFrameworkClusterRoleBinding_missingSubject(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_missingSubject(),
				ExpectError: regexp.MustCompile(`must have a configuration value`),
			},
		},
	})
}

func testAccKubernetesClusterRoleBindingV1Config_missingMetadata() string {
	return `
resource "kubernetes_cluster_role_binding_v1" "test" {
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
`
}

func testAccKubernetesClusterRoleBindingV1Config_missingRoleRef() string {
	return `
resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "tf-acc-test-missing-role-ref"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = "rbac.authorization.k8s.io"
  }
}
`
}

func testAccKubernetesClusterRoleBindingV1Config_missingSubject() string {
	return `
resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "tf-acc-test-missing-subject"
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }
}
`
}

func testAccKubernetesClusterRoleBindingV1Config_invalidName() string {
	return `
resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = "invalid/name"
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
`
}

func testAccKubernetesClusterRoleBindingV1Config_invalidGenerateName() string {
	return `
resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    generate_name = "invalid/prefix"
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
`
}

// TestAccFrameworkClusterRoleBinding_emptyUserAPIGroup verifies that
// api_group = "" on a User subject, which the API server defaults to
// rbac.authorization.k8s.io, does not drift: apply -> refresh -> empty plan.
func TestAccFrameworkClusterRoleBinding_emptyUserAPIGroup(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_emptyUserAPIGroup(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPreRefresh:  []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", ""),
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_emptyUserAPIGroup(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccFrameworkClusterRoleBinding_disappears verifies that if the
// ClusterRoleBinding is deleted outside Terraform, the next refresh removes it
// from state (Read 404 -> RemoveResource), the plan proposes to recreate it,
// and re-applying converges.
func TestAccFrameworkClusterRoleBinding_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_basic(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					testAccKubernetesClusterRoleBindingV1DeleteOutOfBand(name),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionCreate),
					},
				},
				Check: resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_basic(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// TestAccFrameworkClusterRoleBinding_importInvalidIdentity verifies that an
// identity import with the wrong kind or api_version is rejected.
func TestAccFrameworkClusterRoleBinding_importInvalidIdentity(t *testing.T) {
	for _, tc := range []struct{ name, kind, apiVersion string }{
		{"otherResource", "ConfigMap", "v9"},
		{"wrongKind", "RoleBinding", "rbac.authorization.k8s.io/v1"},
		{"wrongAPIVersion", "ClusterRoleBinding", "rbac.authorization.k8s.io/v1beta1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			resource.ParallelTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_12_0),
				},
				Steps: []resource.TestStep{
					{Config: testAccKubernetesClusterRoleBindingV1Config_basic(name)},
					{
						ResourceName:           clusterRoleBindingResourceName,
						ImportState:            true,
						ImportStateKind:        resource.ImportBlockWithResourceIdentity,
						ImportStateConfigExact: true,
						Config:                 testAccKubernetesClusterRoleBindingV1Config_identityImport(name, tc.kind, tc.apiVersion),
						ExpectError:            regexp.MustCompile("invalid identity import"),
					},
				},
			})
		})
	}
}

// testAccKubernetesClusterRoleBindingV1DeleteOutOfBand deletes the
// ClusterRoleBinding through the API, simulating a change made outside
// Terraform.
func testAccKubernetesClusterRoleBindingV1DeleteOutOfBand(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		meta, err := sdkv2providerMeta()
		if err != nil {
			return err
		}
		conn, err := meta().(kubernetes.KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		return conn.RbacV1().ClusterRoleBindings().Delete(context.Background(), name, metav1.DeleteOptions{})
	}
}

func testAccKubernetesClusterRoleBindingV1Config_identityImport(name, kind, apiVersion string) string {
	return testAccKubernetesClusterRoleBindingV1Config_basic(name) + fmt.Sprintf(`
import {
  to = kubernetes_cluster_role_binding_v1.test
  identity = {
    name        = %q
    kind        = %q
    api_version = %q
  }
}
`, name, kind, apiVersion)
}

func testAccKubernetesClusterRoleBindingV1Config_emptyUserAPIGroup(name string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = %q
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "ClusterRole"
    name      = "cluster-admin"
  }

  subject {
    kind      = "User"
    name      = "notauser"
    api_group = ""
  }
}
`, name)
}

// testAccClusterRoleBindingCheckSameUID records the binding UID on its first
// call and fails if a later step sees a different UID, i.e. the binding was
// destroyed and recreated instead of updated in place.
func testAccClusterRoleBindingCheckSameUID() resource.TestCheckFunc {
	var prior string
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[clusterRoleBindingResourceName]
		if !ok {
			return fmt.Errorf("missing %s", clusterRoleBindingResourceName)
		}
		uid := rs.Primary.Attributes["metadata.0.uid"]
		if uid == "" {
			return fmt.Errorf("missing ClusterRoleBinding UID")
		}
		if prior != "" && uid != prior {
			return fmt.Errorf("ClusterRoleBinding replaced: UID %q became %q", prior, uid)
		}
		prior = uid
		return nil
	}
}
