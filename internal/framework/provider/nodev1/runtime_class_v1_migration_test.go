// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package nodev1_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// sdkv2ProviderVersion is the last release that served kubernetes_runtime_class_v1 from
// the SDKv2 implementation.
const sdkv2ProviderVersion = "3.2.1"

// sdkv2PreIdentityProviderVersion predates provider-wide resource identity (2.38.0).
//
// Unlike most resources, the SDKv2 runtime class never declared an identity in any release,
// so 3.2.1 state is identity-less too: both write identity_schema_version 0 and a null
// identity. Both baselines therefore go through UpgradeIdentity. 2.37.1 is pinned as well so
// the pre-2.38 state shape stays covered if the two ever diverge.
const sdkv2PreIdentityProviderVersion = "2.37.1"

// migrationPlans is what the first plan after upgrading must show, on each of the two
// paths a user can take.
type migrationPlans struct {
	// refresh is a plain `terraform plan`, which calls Read before diffing.
	refresh []plancheck.PlanCheck
	// noRefresh is `terraform plan -refresh=false`, which diffs the upgraded SDKv2 state
	// as stored. It is the path that exposes state-shape differences a refresh would hide.
	noRefresh []plancheck.PlanCheck
	// noRefreshApplyWrites is whether noRefresh plans a change. If it does, apply runs
	// Update and writes the identity. If not, apply calls nothing and the identity stays
	// as UpgradeIdentity left it — all null — until the next refresh runs Read, which the
	// "refresh" run of the same config covers.
	noRefreshApplyWrites bool
}

// namedConfigPlans are the expectations for a config that sets metadata.name.
//
// SDKv2 stored the unset generate_name as "" where the framework plans null. A refreshing
// plan is empty, because Read writes null first. A non-refresh plan cannot be: generate_name
// is Optional without Computed, so the plan must carry the config's null. The shared schema
// exempts that transition from replacement, leaving an in-place update that sends no request
// to Kubernetes. That is asserted exactly, so any other difference fails.
var namedConfigPlans = migrationPlans{
	refresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	noRefresh: []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(runtimeClassAddr, plancheck.ResourceActionUpdate),
		expectOnlySDKv2GenerateNameNormalized(runtimeClassAddr),
	},
	noRefreshApplyWrites: true,
}

// generatedNameConfigPlans are the expectations for a config that sets generate_name: SDKv2
// stored the configured prefix, so nothing differs on either path.
var generatedNameConfigPlans = migrationPlans{
	refresh:   []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	noRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
}

// testAccRuntimeClassV1Migration applies config with a released SDKv2 provider, then applies
// the byte-identical config with the local framework provider, once with a refreshing plan
// and once with -refresh=false. Both paths must plan what `plans` says, keep the same object
// (UID unchanged) and settle: the harness fails if either post-apply plan is non-empty.
//
// Every released SDKv2 runtime class state is identity-less, so every run also goes through
// UpgradeIdentity; a missing upgrader fails the first framework plan with "Unable to Upgrade
// Resource Identity". checkIdentity additionally asserts the identity Read then persisted.
func testAccRuntimeClassV1Migration(t *testing.T, version string, config func() string, plans migrationPlans, checkIdentity bool) {
	t.Helper()

	for _, path := range []struct {
		name       string
		noRefresh  bool
		planChecks []plancheck.PlanCheck
	}{
		{"refresh", false, plans.refresh},
		{"no_refresh", true, plans.noRefresh},
	} {
		t.Run(path.name, func(t *testing.T) {
			cfg := config()
			sameUID := statecheck.CompareValue(compare.ValuesSame())
			uidPath := tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("uid")

			frameworkChecks := []statecheck.StateCheck{sameUID.AddStateValue(runtimeClassAddr, uidPath)}
			var versionChecks []tfversion.TerraformVersionCheck
			identityFilledByApply := !path.noRefresh || plans.noRefreshApplyWrites
			if checkIdentity {
				// Resource identity is stored in state only from Terraform 1.12.
				versionChecks = append(versionChecks, tfversion.SkipBelow(tfversion.Version1_12_0))
				if identityFilledByApply {
					frameworkChecks = append(frameworkChecks, testAccRuntimeClassV1IdentityChecks()...)
				} else {
					frameworkChecks = append(frameworkChecks, testAccRuntimeClassV1NullIdentityChecks()...)
				}
			}

			steps := []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"kubernetes": {
							VersionConstraint: version,
							Source:            "hashicorp/kubernetes",
						},
					},
					Config:            cfg,
					ConfigStateChecks: []statecheck.StateCheck{sameUID.AddStateValue(runtimeClassAddr, uidPath)},
				},
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   cfg,
					ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: path.planChecks},
					ConfigStateChecks:        frameworkChecks,
				},
			}
			resource.ParallelTest(t, resource.TestCase{
				PreCheck:               func() { testAccPreCheck(t) },
				CheckDestroy:           testAccCheckRuntimeClassV1Destroy,
				TerraformVersionChecks: versionChecks,
				AdditionalCLIOptions: &resource.AdditionalCLIOptions{
					Plan: resource.PlanOptions{NoRefresh: path.noRefresh},
				},
				Steps: steps,
			})
		})
	}
}

func testAccRuntimeClassV1IdentityChecks() []statecheck.StateCheck {
	return []statecheck.StateCheck{
		statecheck.ExpectIdentityValue(runtimeClassAddr, tfjsonpath.New("kind"),
			knownvalue.StringExact("RuntimeClass")),
		statecheck.ExpectIdentityValue(runtimeClassAddr, tfjsonpath.New("api_version"),
			knownvalue.StringExact("node.k8s.io/v1")),
		statecheck.ExpectIdentityValueMatchesStateAtPath(runtimeClassAddr, tfjsonpath.New("name"),
			tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey("name")),
	}
}

func testAccRuntimeClassV1NullIdentityChecks() []statecheck.StateCheck {
	var checks []statecheck.StateCheck
	for _, attr := range []string{"name", "kind", "api_version"} {
		checks = append(checks, statecheck.ExpectIdentityValue(runtimeClassAddr, tfjsonpath.New(attr), knownvalue.Null()))
	}
	return checks
}

// expectOnlySDKv2GenerateNameNormalized passes only if the planned change to addr differs
// from prior state at exactly one path, metadata.0.generate_name, going from SDKv2's "" to
// null. Values the plan marks unknown are excluded, since an in-place update makes computed
// attributes such as resource_version unknown.
func expectOnlySDKv2GenerateNameNormalized(addr string) plancheck.PlanCheck {
	return generateNameNormalizedCheck{addr: addr}
}

type generateNameNormalizedCheck struct{ addr string }

func (c generateNameNormalizedCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, rc := range req.Plan.ResourceChanges {
		if rc.Address != c.addr {
			continue
		}
		if len(rc.Change.ReplacePaths) > 0 {
			resp.Error = fmt.Errorf("%s: unexpected replace_paths %v", c.addr, rc.Change.ReplacePaths)
			return
		}

		before, after, unknown := map[string]any{}, map[string]any{}, map[string]any{}
		flattenPlanValue("", rc.Change.Before, before)
		flattenPlanValue("", rc.Change.After, after)
		flattenPlanValue("", rc.Change.AfterUnknown, unknown)

		const generateName = "metadata.0.generate_name"
		if before[generateName] != "" || after[generateName] != nil {
			resp.Error = fmt.Errorf("%s: want %s \"\" -> null, got %#v -> %#v",
				c.addr, generateName, before[generateName], after[generateName])
			return
		}

		var diffs []string
		for _, p := range unionKeys(before, after) {
			if p == generateName || unknown[p] == true {
				continue
			}
			if !reflect.DeepEqual(before[p], after[p]) {
				diffs = append(diffs, fmt.Sprintf("%s: %#v -> %#v", p, before[p], after[p]))
			}
		}
		if len(diffs) > 0 {
			resp.Error = fmt.Errorf("%s: unexpected planned changes besides %s:\n%s",
				c.addr, generateName, strings.Join(diffs, "\n"))
		}
		return
	}
	resp.Error = fmt.Errorf("%s: no planned change found", c.addr)
}

// flattenPlanValue flattens decoded plan JSON into dotted paths. Empty collections are kept
// as leaves so that null and empty remain distinguishable.
func flattenPlanValue(prefix string, v any, out map[string]any) {
	join := func(k string) string {
		if prefix == "" {
			return k
		}
		return prefix + "." + k
	}
	switch v := v.(type) {
	case map[string]any:
		if len(v) == 0 && prefix != "" {
			out[prefix] = map[string]any{}
		}
		for k, e := range v {
			flattenPlanValue(join(k), e, out)
		}
	case []any:
		if len(v) == 0 {
			out[prefix] = []any{}
		}
		for i, e := range v {
			flattenPlanValue(join(strconv.Itoa(i)), e, out)
		}
	default:
		out[prefix] = v
	}
}

func unionKeys(a, b map[string]any) []string {
	seen := map[string]bool{}
	var keys []string
	for _, m := range []map[string]any{a, b} {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

func testAccRuntimeClassV1MigrationName() string {
	return fmt.Sprintf("tf-migration-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
}

func testAccRuntimeClassV1MigrationPrefix() string {
	return fmt.Sprintf("tf-migration-%s-", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
}

func named(build func(string) string) func() string {
	return func() string { return build(testAccRuntimeClassV1MigrationName()) }
}

func generated(build func(string) string) func() string {
	return func() string { return build(testAccRuntimeClassV1MigrationPrefix()) }
}

// Upgrades from the last SDKv2 release.

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2_basic(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2ProviderVersion,
		named(testAccRuntimeClassV1MigConfig_basic), namedConfigPlans, false)
}

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2_labels(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2ProviderVersion,
		named(testAccRuntimeClassV1MigConfig_labels), namedConfigPlans, false)
}

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2_annotations(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2ProviderVersion,
		named(testAccRuntimeClassV1MigConfig_annotations), namedConfigPlans, false)
}

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2_generateName(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2ProviderVersion,
		generated(testAccRuntimeClassV1MigConfig_generateName), generatedNameConfigPlans, false)
}

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2_completeName(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2ProviderVersion,
		named(testAccRuntimeClassV1MigConfig_completeName), namedConfigPlans, false)
}

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2_completeGenerateName(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2ProviderVersion,
		generated(testAccRuntimeClassV1MigConfig_completeGenerateName), generatedNameConfigPlans, false)
}

// SDKv2 stored null for `labels = {}` and `annotations = {}`, so the first plan after the
// upgrade reconciles null to {} on both paths. The specific action is asserted — an in-place
// update that sends no request to Kubernetes — so it can never silently become a
// replacement, and the harness's post-apply plans prove it settles.
func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2_emptyMaps(t *testing.T) {
	update := []plancheck.PlanCheck{
		plancheck.ExpectResourceAction(runtimeClassAddr, plancheck.ResourceActionUpdate),
	}
	testAccRuntimeClassV1Migration(t, sdkv2ProviderVersion,
		named(testAccRuntimeClassV1MigConfig_emptyMaps),
		migrationPlans{refresh: update, noRefresh: update, noRefreshApplyWrites: true}, false)
}

// Upgrades from before provider-wide identity (2.38.0), additionally asserting the identity
// persisted after the upgrade.

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2PreIdentity_basicName(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2PreIdentityProviderVersion,
		named(testAccRuntimeClassV1MigConfig_basic), namedConfigPlans, true)
}

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2PreIdentity_completeName(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2PreIdentityProviderVersion,
		named(testAccRuntimeClassV1MigConfig_completeName), namedConfigPlans, true)
}

func TestAccKubernetesRuntimeClassV1_UpgradeFromSDKV2PreIdentity_completeGenerateName(t *testing.T) {
	testAccRuntimeClassV1Migration(t, sdkv2PreIdentityProviderVersion,
		generated(testAccRuntimeClassV1MigConfig_completeGenerateName), generatedNameConfigPlans, true)
}

func testAccRuntimeClassV1MigConfig_basic(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
  }
  handler = "runc"
}
`, name)
}

func testAccRuntimeClassV1MigConfig_labels(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    labels = {
      env  = "staging"
      team = "platform"
    }
  }
  handler = "runc"
}
`, name)
}

func testAccRuntimeClassV1MigConfig_annotations(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    annotations = {
      owner   = "team-a"
      version = "v1"
    }
  }
  handler = "runc"
}
`, name)
}

func testAccRuntimeClassV1MigConfig_generateName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    generate_name = %q
  }
  handler = "runc"
}
`, prefix)
}

func testAccRuntimeClassV1MigConfig_emptyMaps(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name        = %q
    labels      = {}
    annotations = {}
  }
  handler = "runc"
}
`, name)
}

// Every configurable argument set: both metadata maps and handler, named explicitly.
func testAccRuntimeClassV1MigConfig_completeName(name string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    name = %q
    labels = {
      env  = "staging"
      team = "platform"
    }
    annotations = {
      owner   = "team-a"
      version = "v1"
    }
  }
  handler = "runc"
}
`, name)
}

// Every configurable argument set, with a server-generated name.
func testAccRuntimeClassV1MigConfig_completeGenerateName(prefix string) string {
	return fmt.Sprintf(`resource "kubernetes_runtime_class_v1" "test" {
  metadata {
    generate_name = %q
    labels = {
      env  = "staging"
      team = "platform"
    }
    annotations = {
      owner   = "team-a"
      version = "v1"
    }
  }
  handler = "runc"
}
`, prefix)
}
