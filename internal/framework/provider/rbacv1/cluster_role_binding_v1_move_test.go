// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// MoveState tests: moving state from the deprecated kubernetes_cluster_role_binding
// alias, which stays on SDKv2, to this framework resource with a moved block.
//
// Distinct from the migration tests, which upgrade state in place under the same
// type name. Here MoveState translates the SDKv2 state field by field. Configs come
// from the resource tests so both families describe the same shapes, as in the
// namespace move tests.

// testAccClusterRoleBindingMoveSource rewrites a kubernetes_cluster_role_binding_v1
// config to the deprecated alias, for the step that runs under the SDKv2 provider.
func testAccClusterRoleBindingMoveSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_cluster_role_binding_v1"`, `"kubernetes_cluster_role_binding"`)
}

// testAccClusterRoleBindingMoveTarget appends the moved block. Apart from that and
// the type name the two steps' configs are identical, so a non-empty plan can only
// come from the move itself.
func testAccClusterRoleBindingMoveTarget(config string) string {
	return config + `
moved {
  from = kubernetes_cluster_role_binding.test
  to   = kubernetes_cluster_role_binding_v1.test
}
`
}

// testAccClusterRoleBindingMoveState applies config as kubernetes_cluster_role_binding
// under the given SDKv2 release, then re-plans it as kubernetes_cluster_role_binding_v1
// with a moved block and asserts the move plans nothing. No attribute checks are
// needed: an empty plan already proves the moved state matches the configuration,
// and the test framework re-plans after the apply. Cross-type moves need
// Terraform 1.8+.
func testAccClusterRoleBindingMoveState(t *testing.T, version, config string) {
	t.Helper()
	testAccClusterRoleBindingMoveStateWithPlanCheck(t, version, config, plancheck.ExpectEmptyPlan())
}

// testAccClusterRoleBindingMoveStateExpectingUpdate is for shapes SDKv2 cannot store
// exactly as configured, so the first plan after the move reconciles them. An
// in-place update is acceptable; a replacement would delete the binding, which is
// why the action is asserted rather than just a non-empty plan.
func testAccClusterRoleBindingMoveStateExpectingUpdate(t *testing.T, version, config string) {
	t.Helper()
	testAccClusterRoleBindingMoveStateWithPlanCheck(t, version, config,
		plancheck.ExpectResourceAction(clusterRoleBindingResourceName, plancheck.ResourceActionUpdate))
}

func testAccClusterRoleBindingMoveStateWithPlanCheck(t *testing.T, version, config string, check plancheck.PlanCheck) {
	t.Helper()

	resource.ParallelTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccKubernetesClusterRoleBindingV1Destroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: releasedKubernetesProviderAt(version),
				Config:            testAccClusterRoleBindingMoveSource(config),
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   testAccClusterRoleBindingMoveTarget(config),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{check},
				},
			},
		},
	})
}

// A single User subject and no metadata maps: the baseline shape.
func TestAccMoveClusterRoleBinding_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleBindingMoveState(t, migratedProviderVersion, testAccKubernetesClusterRoleBindingV1Config_basic(name))
}

// generate_name: metadata.name is server-assigned, and the only shape where SDKv2
// stores a real generate_name instead of "".
func TestAccMoveClusterRoleBinding_generatedName(t *testing.T) {
	testAccClusterRoleBindingMoveState(t, migratedProviderVersion, testAccKubernetesClusterRoleBindingV1Config_generateName("tf-acc-test-gen:"))
}

// User, ServiceAccount (empty api_group, kube-system) and Group subjects, so every
// subject field is translated.
func TestAccMoveClusterRoleBinding_multipleSubjects(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleBindingMoveState(t, migratedProviderVersion, testAccKubernetesClusterRoleBindingV1Config_modified(name))
}

// An explicitly empty subject namespace next to one that takes the "default" default.
func TestAccMoveClusterRoleBinding_emptySubjectNamespace(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleBindingMoveState(t, migratedProviderVersion, testAccKubernetesClusterRoleBindingV1Config_emptySubjectNamespace(name))
}

// labels = {} and annotations = {}: SDKv2 stores null for both, so the first plan
// after the move updates them to {}, the same one-off update the in-place upgrade
// makes for this shape.
func TestAccMoveClusterRoleBinding_emptyMetadata(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleBindingMoveStateExpectingUpdate(t, migratedProviderVersion,
		testAccKubernetesClusterRoleBindingV1Config_metadata(name, "labels = {}\nannotations = {}"))
}

// api_group = "" on a User subject: SDKv2 stores the API default
// rbac.authorization.k8s.io, so the first plan after the move updates it to the
// configured "", the same one-off update the in-place upgrade makes.
func TestAccMoveClusterRoleBinding_emptyUserAPIGroup(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleBindingMoveStateExpectingUpdate(t, migratedProviderVersion,
		testAccKubernetesClusterRoleBindingV1Config_emptyUserAPIGroup(name))
}

// The pre-identity tests below are the only move coverage of state written before
// resource identity existed (2.38.0). There the identity MoveState writes is built
// rather than carried across. As in the namespace move tests, a few shapes are
// enough: name, generate_name, and every subject kind.
func TestAccMoveClusterRoleBinding_preIdentity_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleBindingMoveState(t, clusterRoleBindingPreIdentityProviderVersion, testAccKubernetesClusterRoleBindingV1Config_basic(name))
}

func TestAccMoveClusterRoleBinding_preIdentity_generatedName(t *testing.T) {
	testAccClusterRoleBindingMoveState(t, clusterRoleBindingPreIdentityProviderVersion, testAccKubernetesClusterRoleBindingV1Config_generateName("tf-acc-test-gen:"))
}

func TestAccMoveClusterRoleBinding_preIdentity_multipleSubjects(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleBindingMoveState(t, clusterRoleBindingPreIdentityProviderVersion, testAccKubernetesClusterRoleBindingV1Config_modified(name))
}
