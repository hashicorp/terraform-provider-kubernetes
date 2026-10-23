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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/rbacv1"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

const clusterRoleBindingResourceName = "kubernetes_cluster_role_binding_v1.test"

// TestAccClusterRoleBinding_basic follows SDKv2's TestAccKubernetesClusterRoleBindingV1_basic
// step for step with the same assertions, plus an import after each step as in the
// SDKv2 kubernetes_cluster_role_binding test. Adding subjects must update in place
// (same UID); changing role_ref must replace the binding.
func TestAccClusterRoleBinding_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	checkUID := testAccClusterRoleBindingCheckSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					checkUID,
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_modified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "default"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", ""),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "Group"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
					checkUID,
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// role_ref is immutable, so changing it replaces the binding.
				Config: testAccKubernetesClusterRoleBindingV1Config_modifiedRoleRef(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "default"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", ""),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "Group"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_modified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.namespace", "kube-system"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "default"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", ""),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "system:masters"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "Group"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccClusterRoleBinding_identity(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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

// TestAccClusterRoleBinding_generatedName follows SDKv2's TestAccKubernetesClusterRoleBindingV1_generatedName.
func TestAccClusterRoleBinding_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_generateName(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", regexp.MustCompile("^"+prefix)),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.generate_name", prefix),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccClusterRoleBinding_serviceAccountSubject follows SDKv2's TestAccKubernetesClusterRoleBindingV1_serviceaccount_subject.
func TestAccClusterRoleBinding_serviceAccountSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_serviceAccountSubject(name),
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
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "someservice"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "ServiceAccount"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", ""),
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccClusterRoleBinding_groupSubject follows SDKv2's TestAccKubernetesClusterRoleBindingV1_group_subject.
func TestAccClusterRoleBinding_groupSubject(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_groupSubject(name),
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
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "somegroup"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "Group"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccClusterRoleBinding_updatePatchOperationsOrderWithRemovals ports SDKv2's
// TestAccKubernetesClusterRoleBindingV1_UpdatePatchOperationsOrderWithRemovals with the
// same sequence, 1,2,3 -> 2,4 -> 0,1,2,3: it removes from the front, adds a new name
// and inserts at the front. Every step must update in place.
func TestAccClusterRoleBinding_updatePatchOperationsOrderWithRemovals(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	checkUID := testAccClusterRoleBindingCheckSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1ConfigBug_step_0(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "notauser2"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "notauser3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
					checkUID,
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1ConfigBug_step_1(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "2"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser2"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "notauser4"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", "rbac.authorization.k8s.io"),
					checkUID,
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1ConfigBug_step_2(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "metadata.0.name", name),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.generation"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.resource_version"),
					resource.TestCheckResourceAttrSet(clusterRoleBindingResourceName, "metadata.0.uid"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.#", "1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.kind", "ClusterRole"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "role_ref.0.name", "cluster-admin"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "4"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.name", "notauser0"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.0.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.name", "notauser1"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.1.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.name", "notauser2"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.2.api_group", "rbac.authorization.k8s.io"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.3.name", "notauser3"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.3.kind", "User"),
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.3.api_group", "rbac.authorization.k8s.io"),
					checkUID,
				),
			},
			{
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccClusterRoleBinding_generatedNameMetadataUpdate verifies that
// adding a label to a binding created with generate_name updates it in place.
func TestAccClusterRoleBinding_generatedNameMetadataUpdate(t *testing.T) {
	prefix := "tf-acc-test:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum) + "-"
	config := testAccKubernetesClusterRoleBindingV1Config_generateName(prefix)
	updated := testAccKubernetesClusterRoleBindingV1Config_generateNameWithLabels(prefix)
	checkUID := testAccClusterRoleBindingCheckSameUID()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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

// TestAccClusterRoleBinding_metadataUpdate moves one binding through
// no metadata maps, empty maps, managed labels and annotations, and back to
// none, checking that every change is applied in place.
func TestAccClusterRoleBinding_metadataUpdate(t *testing.T) {
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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps:                    steps,
	})
}

// TestAccClusterRoleBinding_emptySubjectNamespace verifies that an
// explicitly empty subject namespace is kept as-is and does not produce a
// diff, while an omitted namespace still takes the schema default.
func TestAccClusterRoleBinding_emptySubjectNamespace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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
				ResourceName:      clusterRoleBindingResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccClusterRoleBinding_emptyUserAPIGroup verifies that
// api_group = "" on a User subject, which the API server defaults to
// rbac.authorization.k8s.io, does not drift: apply -> refresh -> empty plan.
func TestAccClusterRoleBinding_emptyUserAPIGroup(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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

// TestAccClusterRoleBinding_ignoreAnnotations verifies that:
//   - provider-level ignore_annotations prevents drift from untracked
//     server-side annotations matching the pattern
//   - annotations the user has explicitly configured are retained even
//     when their key matches an ignore pattern
func TestAccClusterRoleBinding_ignoreAnnotations(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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

// TestAccClusterRoleBinding_externalMetadataSurvivesSubjectUpdate
// verifies that an annotation added outside Terraform, whose key matches the
// provider-level ignore_annotations pattern, never shows as drift and is not
// removed when the binding's subjects are updated.
func TestAccClusterRoleBinding_externalMetadataSurvivesSubjectUpdate(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	checkUID := testAccClusterRoleBindingCheckSameUID()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		CheckDestroy:             testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIO(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "1"),
					checkUID,
				),
			},
			{
				// The injected key is filtered by ignore_annotations, so the plan stays empty.
				PreConfig: func() {
					testAccClusterRoleBindingInjectAnnotation(t, name, externalAnnotationKey, externalAnnotationValue)
				},
				Config:   testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIO(name),
				PlanOnly: true,
			},
			{
				Config: testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIOModified(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(clusterRoleBindingResourceName, "subject.#", "3"),
					testAccCheckClusterRoleBindingAnnotation(name, externalAnnotationKey, externalAnnotationValue),
					checkUID,
				),
			},
		},
	})
}

// TestAccClusterRoleBinding_disappears verifies that if the
// ClusterRoleBinding is deleted outside Terraform, the next refresh removes it
// from state (Read 404 -> RemoveResource), the plan proposes to recreate it,
// and re-applying converges.
func TestAccClusterRoleBinding_disappears(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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

// TestAccClusterRoleBinding_importInvalidIdentity verifies that an
// identity import with the wrong kind or api_version is rejected.
func TestAccClusterRoleBinding_importInvalidIdentity(t *testing.T) {
	for _, tc := range []struct{ name, kind, apiVersion string }{
		{"otherResource", "ConfigMap", "v9"},
		{"wrongKind", "RoleBinding", "rbac.authorization.k8s.io/v1"},
		{"wrongAPIVersion", "ClusterRoleBinding", "rbac.authorization.k8s.io/v1beta1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
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

// TestAccClusterRoleBinding_invalidMetadataName verifies that a
// metadata.name containing a path separator is rejected during Terraform
// validation, matching the SDKv2 validateRBACNameFunc behavior.
func TestAccClusterRoleBinding_invalidMetadataName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_invalidName(),
				ExpectError: regexp.MustCompile(`may not contain`),
			},
		},
	})
}

// TestAccClusterRoleBinding_invalidMetadataGenerateName verifies
// that an invalid generate_name prefix is also rejected during validation.
func TestAccClusterRoleBinding_invalidMetadataGenerateName(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_invalidGenerateName(),
				ExpectError: regexp.MustCompile(`may not contain`),
			},
		},
	})
}

// TestAccClusterRoleBinding_metadataValidation verifies that invalid
// metadata is rejected at plan time and that a valid configuration still
// applies afterwards.
func TestAccClusterRoleBinding_metadataValidation(t *testing.T) {
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
				PreCheck:                 func() { testAccPreCheck(t) },
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

// TestAccClusterRoleBinding_missingMetadata verifies that omitting
// the required metadata block is rejected during Terraform validation by
// listvalidator.IsRequired(), rather than deferring the failure to apply.
func TestAccClusterRoleBinding_missingMetadata(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_missingMetadata(),
				ExpectError: regexp.MustCompile(`must have a configuration value`),
			},
		},
	})
}

// TestAccClusterRoleBinding_missingRoleRef verifies that omitting
// the required role_ref block is rejected during Terraform validation by
// listvalidator.IsRequired().
func TestAccClusterRoleBinding_missingRoleRef(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_missingRoleRef(),
				ExpectError: regexp.MustCompile(`must have a configuration value`),
			},
		},
	})
}

// TestAccClusterRoleBinding_missingSubject verifies that omitting
// the required subject block is rejected during Terraform validation by
// listvalidator.IsRequired().
func TestAccClusterRoleBinding_missingSubject(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKubernetesClusterRoleBindingV1Config_missingSubject(),
				ExpectError: regexp.MustCompile(`must have a configuration value`),
			},
		},
	})
}

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
// so TestAccClusterRoleBinding_disappears cannot catch the missing
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

// testAccKubernetesClusterRoleBindingV1Destroy checks every binding left in state is gone.
func testAccKubernetesClusterRoleBindingV1Destroy(s *terraform.State) error {
	meta, err := sdkv2providerMeta()
	if err != nil {
		return err
	}
	conn, err := meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	for _, rs := range s.RootModule().Resources {
		// The deprecated type is included because a failed move test leaves the
		// binding in state under its old address.
		if rs.Type != "kubernetes_cluster_role_binding_v1" && rs.Type != "kubernetes_cluster_role_binding" {
			continue
		}
		_, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, rs.Primary.ID, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		return fmt.Errorf("ClusterRoleBinding still exists: %s", rs.Primary.ID)
	}
	return nil
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

// externalAnnotationKey matches the ignore_annotations pattern in
// testAccKubernetesClusterRoleBindingV1Config_ignoreExternalIO.
const (
	externalAnnotationKey   = "external.io/injected"
	externalAnnotationValue = "true"
)

// testAccClusterRoleBindingInjectAnnotation adds an annotation to the live
// object, simulating another controller writing to it.
func testAccClusterRoleBindingInjectAnnotation(t *testing.T, name, key, value string) {
	t.Helper()
	meta, err := sdkv2providerMeta()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := meta().(kubernetes.KubeClientsets).MainClientset()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	obj, err := conn.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get ClusterRoleBinding %q: %v", name, err)
	}
	if obj.Annotations == nil {
		obj.Annotations = map[string]string{}
	}
	obj.Annotations[key] = value
	if _, err := conn.RbacV1().ClusterRoleBindings().Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
		t.Fatalf("update ClusterRoleBinding %q: %v", name, err)
	}
}

// testAccCheckClusterRoleBindingAnnotation checks the annotation on the live object.
func testAccCheckClusterRoleBindingAnnotation(name, key, value string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		meta, err := sdkv2providerMeta()
		if err != nil {
			return err
		}
		conn, err := meta().(kubernetes.KubeClientsets).MainClientset()
		if err != nil {
			return err
		}
		obj, err := conn.RbacV1().ClusterRoleBindings().Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get ClusterRoleBinding %q: %w", name, err)
		}
		if got, ok := obj.Annotations[key]; !ok || got != value {
			return fmt.Errorf("annotation %q = %q (present=%t), want %q", key, got, ok, value)
		}
		return nil
	}
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

// testAccKubernetesClusterRoleBindingV1Config_generateNameWithLabels is the
// generate_name configuration with a managed label added.
func testAccKubernetesClusterRoleBindingV1Config_generateNameWithLabels(namePrefix string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    generate_name = %q
    labels = {
      "managed" = "true"
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
`, namePrefix)
}

// testAccKubernetesClusterRoleBindingV1Config_metadata is the basic configuration
// with the given lines (labels, annotations or generate_name) added to the
// metadata block.
func testAccKubernetesClusterRoleBindingV1Config_metadata(name, metadata string) string {
	return fmt.Sprintf(`resource "kubernetes_cluster_role_binding_v1" "test" {
  metadata {
    name = %q
    %s
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
`, name, metadata)
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

// testAccKubernetesClusterRoleBindingV1ConfigBug_step_0 is SDKv2's ConfigBug_step_0: notauser1, notauser2, notauser3.
func testAccKubernetesClusterRoleBindingV1ConfigBug_step_0(name string) string {
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
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser1"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser3"
  }
}
`, name)
}

// testAccKubernetesClusterRoleBindingV1ConfigBug_step_1 removes notauser1 and notauser3 and adds notauser4.
func testAccKubernetesClusterRoleBindingV1ConfigBug_step_1(name string) string {
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
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser4"
  }
}
`, name)
}

// testAccKubernetesClusterRoleBindingV1ConfigBug_step_2 inserts notauser0 and notauser1 at the front and adds notauser3.
func testAccKubernetesClusterRoleBindingV1ConfigBug_step_2(name string) string {
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
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser0"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser1"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser2"
  }
  subject {
    api_group = "rbac.authorization.k8s.io"
    kind      = "User"
    name      = "notauser3"
  }
}
`, name)
}
