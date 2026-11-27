// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package rbacv1_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// MoveState tests: moving state from the deprecated kubernetes_cluster_role alias, which
// stays on SDKv2, to this framework resource with a moved block.
//
// Distinct from the migration tests, which upgrade state in place under the same type
// name. Here MoveState translates the SDKv2 state field by field. Configs come from the
// resource tests so both families describe the same shapes.

// testAccClusterRoleMoveSource rewrites a kubernetes_cluster_role_v1 config to the
// deprecated alias, for the step that runs under the SDKv2 provider.
func testAccClusterRoleMoveSource(config string) string {
	return strings.ReplaceAll(config, `"kubernetes_cluster_role_v1"`, `"kubernetes_cluster_role"`)
}

// testAccClusterRoleMoveTarget appends the moved block. Apart from that and the type name
// the two steps' configs are identical, so a non-empty plan can only come from the move.
func testAccClusterRoleMoveTarget(config string) string {
	return config + `
moved {
  from = kubernetes_cluster_role.test
  to   = kubernetes_cluster_role_v1.test
}
`
}

// testAccClusterRoleMoveState applies config as kubernetes_cluster_role under the given
// SDKv2 release, then re-plans it as kubernetes_cluster_role_v1 with a moved block and
// asserts the plan is empty. An empty plan proves the moved state matches the
// configuration; the framework also re-plans after the apply. Cross-type moves need
// Terraform 1.8+.
func testAccClusterRoleMoveState(t *testing.T, version, config string) {
	t.Helper()
	testAccClusterRoleMoveStateWithPlanCheck(t, version, config, plancheck.ExpectEmptyPlan())
}

// testAccClusterRoleMoveStateExpectingUpdate is for shapes SDKv2 cannot store exactly as
// configured, so the first plan after the move reconciles them. An in-place update is
// acceptable; a replacement would delete the role, which is why the action is asserted.
func testAccClusterRoleMoveStateExpectingUpdate(t *testing.T, version, config string) {
	t.Helper()
	testAccClusterRoleMoveStateWithPlanCheck(t, version, config,
		plancheck.ExpectResourceAction(clusterRoleResourceName, plancheck.ResourceActionUpdate))
}

func testAccClusterRoleMoveStateWithPlanCheck(t *testing.T, version, config string, check plancheck.PlanCheck) {
	t.Helper()
	checkUID := testAccClusterRoleCheckSameUID()

	resource.ParallelTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: roleExternalProvider(version),
				Config:            testAccClusterRoleMoveSource(config),
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   testAccClusterRoleMoveTarget(config),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{check},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkUID,
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				ResourceName:             clusterRoleResourceName,
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

// A single rule and labels: the baseline shape.
func TestAccMoveClusterRole_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleMoveState(t, roleSDKv2ProviderVersion, testAccClusterRoleConfig_basic(name))
}

// generate_name: metadata.name is server-assigned, and the only shape where SDKv2 stores
// a real generate_name instead of "".
func TestAccMoveClusterRole_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	testAccClusterRoleMoveState(t, roleSDKv2ProviderVersion, testAccClusterRoleConfig_generateName(prefix))
}

// Three rules, one of which has only non_resource_urls and verbs.
func TestAccMoveClusterRole_multipleRules(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleMoveState(t, roleSDKv2ProviderVersion, testAccClusterRoleConfig_modified(name))
}

// An aggregation rule with a label selector and a match expression.
func TestAccMoveClusterRole_aggregationRule(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleMoveState(t, roleSDKv2ProviderVersion, testAccClusterRoleConfig_aggRule(name))
}

// An aggregated role next to its source. SDKv2 state holds the rules the controller
// copied; MoveState drops them, as Read does, so the move plans nothing.
func TestAccMoveClusterRole_aggregatedRulesFilledByController(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleMoveStateBoth(t, roleSDKv2ProviderVersion, testAccClusterRoleConfig_aggRuleSource(name))
}

// labels = {} and annotations = {}: SDKv2 stores null for both, so the first plan after
// the move updates them to {}, the same one-off update the in-place upgrade makes.
func TestAccMoveClusterRole_emptyMetadata(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleMoveStateExpectingUpdate(t, roleSDKv2ProviderVersion, testAccClusterRoleConfig_emptyMetadata(name))
}

// testAccClusterRoleMoveStateBoth moves both resources of a two-resource config.
func testAccClusterRoleMoveStateBoth(t *testing.T, version, config string) {
	t.Helper()
	checkUID := testAccClusterRoleCheckSameUID()
	target := config + `
moved {
  from = kubernetes_cluster_role.test
  to   = kubernetes_cluster_role_v1.test
}

moved {
  from = kubernetes_cluster_role.source
  to   = kubernetes_cluster_role_v1.source
}
`

	resource.ParallelTest(t, resource.TestCase{
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_8_0),
		},
		CheckDestroy: testAccClusterRoleCheckDestroy,
		Steps: []resource.TestStep{
			{
				ExternalProviders: roleExternalProvider(version),
				Config:            testAccClusterRoleMoveSource(config),
			},
			{
				ProtoV6ProviderFactories: testAccMuxProtoV6ProviderFactories,
				Config:                   target,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkUID,
			},
		},
	})
}

// The pre-identity tests below are the only move coverage of state written before
// resource identity existed. There the identity MoveState writes is built rather than
// carried across.
func TestAccMoveClusterRole_preIdentity_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleMoveState(t, rolePreIdentityProviderVersion, testAccClusterRoleConfig_basic(name))
}

func TestAccMoveClusterRole_preIdentity_generatedName(t *testing.T) {
	prefix := "tf-acc-test-gen:" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	testAccClusterRoleMoveState(t, rolePreIdentityProviderVersion, testAccClusterRoleConfig_generateName(prefix))
}

func TestAccMoveClusterRole_preIdentity_multipleRules(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test:%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleMoveState(t, rolePreIdentityProviderVersion, testAccClusterRoleConfig_modified(name))
}

func TestAccMoveClusterRole_preIdentity_aggregationRule(t *testing.T) {
	name := fmt.Sprintf("tf-acc-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
	testAccClusterRoleMoveState(t, rolePreIdentityProviderVersion, testAccClusterRoleConfig_aggRule(name))
}
