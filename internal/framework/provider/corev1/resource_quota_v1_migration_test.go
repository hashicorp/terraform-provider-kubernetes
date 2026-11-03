// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/corev1"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// resourceQuotaSDKv2Version is the last release that served kubernetes_resource_quota_v1
// from the SDKv2 implementation. It writes state with identity_schema_version 0 and no
// identity, so upgrading from it also goes through UpgradeIdentity.
//
// Unlike namespace_v1, which migrated earlier, this resource was still SDKv2 in 3.4.0, so
// the constant here is not shared with the namespace tests.
const resourceQuotaSDKv2Version = "3.4.0"

// quotaOutcome is what the first plan after leaving SDKv2 must show.
type quotaOutcome int

const (
	// quotaNoChange: the migration is invisible.
	quotaNoChange quotaOutcome = iota
	// quotaRespelled: SDKv2 state holds the API's spelling of a quantity (2Gi) where the
	// config says 2048Mi. Terraform core compares the two as plain strings, so the first plan
	// is one in-place update. The apply must not touch the API, because the quantities are
	// equivalent: the resource version stays put, and the state then holds the config's
	// spelling so the plan after it is empty. The one-time cost of keeping the user's spelling.
	quotaRespelled
	// quotaUpdate: SDKv2 stored null for a config that declares an empty collection, which the
	// framework can tell apart. The first plan reconciles the two with one non-destructive
	// update; the post-apply plan then proves it settles instead of recurring.
	quotaUpdate
)

// quotaScenario is one shape of configuration, shared by the upgrade and move tests so the two
// families cannot drift apart.
type quotaScenario struct {
	name   string
	config func(name string) string
	// updated, when set, is applied after the upgrade as an in-place update: an upgrade that
	// plans nothing proves Read, the update proves the rest of the resource also works on state
	// it did not write.
	updated func(name string) string
	outcome quotaOutcome
	// importIgnore lists attributes that import cannot reproduce, each with a reason at the
	// scenario. Entries match as prefixes.
	importIgnore []string
	// sourceNotIdempotent is set when 3.4.0 itself plans a change against the state it just
	// wrote, so the first step cannot assert an empty plan. That is a defect of the released
	// provider, not of the migration.
	sourceNotIdempotent bool
	// internalMetadata is "labels" or "annotations" when the config declares a key the
	// provider normally filters out (example.kubernetes.io/owner): import has no prior state to
	// exempt it, so it must not claim it.
	internalMetadata string
}

// Import cannot reproduce these: generate_name is not returned by the API (k8s#44539), and a
// timeouts block is configuration only.
var (
	quotaIgnoreGenerateName = []string{"metadata.0.generate_name"}
	quotaIgnoreTimeouts     = []string{"timeouts"}
	// Import has no configuration to take a spelling from, so it reports the API's canonical
	// spelling for the one the config used. Equivalent values.
	quotaIgnoreSpelling = []string{"spec.0.hard.limits.memory", "spec.0.hard.limits.cpu", "spec.0.hard.requests.cpu"}
)

func quotaConcat(lists ...[]string) []string {
	var out []string
	for _, l := range lists {
		out = append(out, l...)
	}
	return out
}

func quotaInternalIgnore(field string) []string {
	if field == "" {
		return nil
	}
	prefix := "metadata.0." + field + "."
	return []string{prefix + "%", prefix + "example.kubernetes.io/owner"}
}

func quotaScenarios() []quotaScenario {
	generated := func(spec ...string) func(string) string {
		return func(name string) string {
			return testAccKubernetesResourceQuotaV1Config_scenario(name, true, nil, spec, nil)
		}
	}
	named := func(metadata []string, spec []string, extra []string) func(string) string {
		return func(name string) string {
			return testAccKubernetesResourceQuotaV1Config_scenario(name, false, metadata, spec, extra)
		}
	}
	pods := []string{`hard = { pods = 4 }`}
	complete := []string{
		`hard   = { pods = 4, "limits.cpu" = "2", "limits.memory" = "2Gi" }`,
		`scopes = ["NotTerminating"]`,
	}
	completeMetadata := []string{`labels      = { team = "platform" }`, `annotations = { note = "retained" }`}
	completeTimeouts := []string{`timeouts {`, `  create = "2m"`, `  update = "3m"`, `}`}

	return []quotaScenario{
		{
			name:   "basic",
			config: testAccKubernetesResourceQuotaV1Config_basic, updated: testAccKubernetesResourceQuotaV1Config_metaModified,
			outcome: quotaRespelled, importIgnore: quotaIgnoreSpelling,
		},
		{name: "minimal", config: named(nil, pods, nil)},
		{name: "generateName", config: generated(pods...), importIgnore: quotaIgnoreGenerateName},
		{name: "annotations", config: named([]string{`annotations = { note = "retained" }`}, pods, nil)},
		{name: "labels", config: named([]string{`labels = { team = "platform" }`}, pods, nil)},
		{
			name:             "declaredInternalLabel",
			config:           named([]string{`labels = { "example.kubernetes.io/owner" = "terraform" }`}, pods, nil),
			internalMetadata: "labels",
		},
		{
			name:             "declaredInternalAnnotation",
			config:           named([]string{`annotations = { "example.kubernetes.io/owner" = "terraform" }`}, pods, nil),
			internalMetadata: "annotations",
		},
		{name: "withScopes", config: testAccKubernetesResourceQuotaV1Config_withScopes},
		{
			name:   "scopeSelector",
			config: testAccKubernetesResourceQuotaV1ConfigScopeSelector, updated: testAccKubernetesResourceQuotaV1ConfigScopeSelectorModified,
		},
		{name: "multipleMatchExpression", config: testAccKubernetesResourceQuotaV1ConfigMultipleMatchExpression},
		{
			// 1000m becomes 1, 2048Mi becomes 2Gi and 0.5 becomes 500m in the API.
			name: "quantitySpelling",
			config: func(name string) string {
				return testAccKubernetesResourceQuotaV1Config_spelling(name, "1000m", "2048Mi", "0.5")
			},
			updated: func(name string) string {
				return testAccKubernetesResourceQuotaV1Config_spelling(name, "2", "4Gi", "500m")
			},
			outcome: quotaRespelled, importIgnore: quotaIgnoreSpelling,
		},
		{
			// The same quantities already in the API's canonical form: nothing to change.
			name: "canonicalQuantities",
			config: func(name string) string {
				return testAccKubernetesResourceQuotaV1Config_spelling(name, "1", "2Gi", "500m")
			},
			updated: func(name string) string {
				return testAccKubernetesResourceQuotaV1Config_spelling(name, "2", "4Gi", "500m")
			},
		},
		{
			// SDKv2 stored a configured timeouts block as an object of strings.
			name: "timeouts",
			config: func(name string) string {
				return testAccKubernetesResourceQuotaV1Config_timeouts(name, "3m")
			},
			importIgnore: quotaIgnoreTimeouts,
		},
		{
			// SDKv2 stores an omitted spec as one spec with empty collections, where the
			// framework keeps it omitted. 3.4.0 itself plans `hard = {} -> null` and
			// `scopes = [] -> null` against that state, so its own step cannot be idempotent.
			//
			// `spec {}` and `spec { hard = {} }` have no scenario: 3.4.0 panics on both in
			// expandResourceQuotaSpec (structures.go:336), so no such state can exist.
			name:                "omittedSpec",
			config:              named(nil, nil, nil),
			outcome:             quotaUpdate,
			sourceNotIdempotent: true,
		},
		{
			// SDKv2 stores null for `{}`, `[]` and an empty timeouts block.
			name: "emptyValuesName",
			config: named([]string{`labels      = {}`, `annotations = {}`},
				[]string{`hard   = { pods = 4 }`, `scopes = []`}, []string{`timeouts {}`}),
			outcome: quotaUpdate, importIgnore: quotaIgnoreTimeouts,
		},
		{
			name: "emptyValuesGenerateName",
			config: func(name string) string {
				return testAccKubernetesResourceQuotaV1Config_scenario(name, true,
					[]string{`labels      = {}`, `annotations = {}`},
					[]string{`hard   = { pods = 4 }`, `scopes = []`}, []string{`timeouts {}`})
			},
			outcome: quotaUpdate, importIgnore: quotaConcat(quotaIgnoreGenerateName, quotaIgnoreTimeouts),
		},
		{
			name:         "completeName",
			config:       named(completeMetadata, complete, completeTimeouts),
			importIgnore: quotaIgnoreTimeouts,
		},
		{
			name: "completeGenerateName",
			config: func(name string) string {
				return testAccKubernetesResourceQuotaV1Config_scenario(name, true, completeMetadata, complete, completeTimeouts)
			},
			importIgnore: quotaConcat(quotaIgnoreGenerateName, quotaIgnoreTimeouts),
		},
	}
}

// quotaPreIdentityScenarios are the shapes also run from a release that predates identity. The
// stored identity is absent there rather than merely older, so UpgradeIdentity builds one; the
// shapes are the minimal one, a respelled one, a generated name and the fully populated ones.
var quotaPreIdentityScenarios = []string{"basic", "generateName", "scopeSelector", "completeName", "completeGenerateName"}

func quotaScenarioByName(t *testing.T, name string) quotaScenario {
	t.Helper()
	for _, s := range quotaScenarios() {
		if s.name == name {
			return s
		}
	}
	t.Fatalf("unknown scenario %q", name)
	return quotaScenario{}
}

func testAccResourceQuotaMigrationName() string {
	return fmt.Sprintf("tf-migration-test-%s", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))
}

func testAccResourceQuotaExternalProvider(version string) map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {VersionConstraint: version, Source: "hashicorp/kubernetes"},
	}
}

func quotaUpgradePlanChecks(s quotaScenario) resource.ConfigPlanChecks {
	if s.outcome == quotaNoChange {
		return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}}
	}
	return resource.ConfigPlanChecks{
		PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionUpdate)},
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
}

// testAccCheckResourceQuotaUntouched fails when the apply wrote to the API.
func testAccCheckResourceQuotaUntouched(before, after *k8sv1.ResourceQuota) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if before.ResourceVersion != after.ResourceVersion {
			return fmt.Errorf("resource version changed from %s to %s: the apply wrote to the API",
				before.ResourceVersion, after.ResourceVersion)
		}
		return nil
	}
}

// testAccCheckResourceQuotaImportOwnership proves import did not claim a key the provider
// filters out, while the live object still carries it.
func testAccCheckResourceQuotaImportOwnership(internalMetadata string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if internalMetadata == "" {
			return nil
		}
		if len(states) != 1 {
			return fmt.Errorf("expected one imported resource quota, got %d", len(states))
		}
		attrs := states[0].Attributes
		prefix := "metadata.0." + internalMetadata + "."
		if _, ok := attrs[prefix+"example.kubernetes.io/owner"]; ok {
			return fmt.Errorf("import unexpectedly claimed ownership of internal %s", internalMetadata)
		}
		if count := attrs[prefix+"%"]; count != "" && count != "0" {
			return fmt.Errorf("imported %s count = %q, want empty", internalMetadata, count)
		}
		conn, err := mainClientset()
		if err != nil {
			return err
		}
		quota, err := conn.CoreV1().ResourceQuotas(attrs["metadata.0.namespace"]).Get(
			context.Background(), attrs["metadata.0.name"], metav1.GetOptions{})
		if err != nil {
			return err
		}
		live := quota.Labels
		if internalMetadata == "annotations" {
			live = quota.Annotations
		}
		if live["example.kubernetes.io/owner"] != "terraform" {
			return fmt.Errorf("import changed the live internal %s: %v", internalMetadata, live)
		}
		return nil
	}
}

// testAccResourceQuotaMigrationScenario applies the scenario's config under an SDKv2 release,
// then re-plans it under the local framework provider and asserts the outcome the scenario
// declares. Import is verified at each stage, since an attribute that stops round-tripping is
// otherwise invisible.
//
// kubernetes_namespace_v1 in the config is served by whichever provider runs the step, so the
// namespace is upgraded alongside the quota the way a real configuration would be.
func testAccResourceQuotaMigrationScenario(t *testing.T, version string, s quotaScenario) {
	t.Helper()

	name := testAccResourceQuotaMigrationName()
	config := s.config(name)
	var before, after k8sv1.ResourceQuota
	ignore := quotaConcat(s.importIgnore, quotaInternalIgnore(s.internalMetadata))
	importStep := func(sdkv2 bool) resource.TestStep {
		step := resource.TestStep{
			ResourceName:            resourceQuotaResourceName,
			ImportState:             true,
			ImportStateVerify:       true,
			ImportStateVerifyIgnore: ignore,
			ImportStateCheck:        testAccCheckResourceQuotaImportOwnership(s.internalMetadata),
		}
		if sdkv2 {
			step.ExternalProviders = testAccResourceQuotaExternalProvider(version)
		} else {
			step.ProtoV6ProviderFactories = testAccProtoV6ProviderFactories
		}
		return step
	}

	checks := []resource.TestCheckFunc{
		testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
		testAccCheckResourceQuotaNotRecreated(&before, &after),
	}
	if s.outcome == quotaRespelled {
		checks = append(checks, testAccCheckResourceQuotaUntouched(&before, &after))
	}

	steps := []resource.TestStep{
		{
			ExternalProviders:  testAccResourceQuotaExternalProvider(version),
			Config:             config,
			Check:              testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &before),
			ExpectNonEmptyPlan: s.sourceNotIdempotent,
		},
		importStep(true),
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   config,
			ConfigPlanChecks:         quotaUpgradePlanChecks(s),
			Check:                    resource.ComposeAggregateTestCheckFunc(checks...),
		},
		importStep(false),
	}
	if s.updated != nil {
		steps = append(steps,
			resource.TestStep{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   s.updated(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionUpdate)},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
					testAccCheckResourceQuotaNotRecreated(&before, &after),
				),
			},
			importStep(false),
		)
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		// Without this a failed run leaks quotas and namespaces, and the next run collides.
		CheckDestroy: testAccCheckKubernetesResourceQuotaV1Destroy,
		Steps:        steps,
	})
}

func TestAccKubernetesResourceQuotaV1_UpgradeFromSDKV2_scenarios(t *testing.T) {
	for _, s := range quotaScenarios() {
		t.Run(s.name, func(t *testing.T) { testAccResourceQuotaMigrationScenario(t, resourceQuotaSDKv2Version, s) })
	}
}

func TestAccKubernetesResourceQuotaV1_UpgradeFromSDKV2PreIdentity_scenarios(t *testing.T) {
	for _, name := range quotaPreIdentityScenarios {
		s := quotaScenarioByName(t, name)
		t.Run(s.name, func(t *testing.T) { testAccResourceQuotaMigrationScenario(t, sdkv2PreIdentityProviderVersion, s) })
	}
}

// Identity is written by Read, so state upgraded from a release without one, or with an older
// one, must have it after the first refresh and be importable by it.
func testAccResourceQuotaUpgradeIdentity(t *testing.T, version string) {
	t.Helper()
	name := testAccResourceQuotaMigrationName()
	config := testAccKubernetesResourceQuotaV1Config_withScopes(name)
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		CheckDestroy:           testAccCheckKubernetesResourceQuotaV1Destroy,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
		Steps: []resource.TestStep{
			{ExternalProviders: testAccResourceQuotaExternalProvider(version), Config: config},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks:         resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectIdentity(resourceQuotaResourceName, map[string]knownvalue.Check{
						"namespace":   knownvalue.StringExact(name),
						"name":        knownvalue.StringExact(name),
						"api_version": knownvalue.StringExact("v1"),
						"kind":        knownvalue.StringExact("ResourceQuota"),
					}),
				},
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				ResourceName:             resourceQuotaResourceName,
				ImportState:              true,
				ImportStateKind:          resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

func TestAccKubernetesResourceQuotaV1_UpgradeFromSDKV2_identity(t *testing.T) {
	testAccResourceQuotaUpgradeIdentity(t, resourceQuotaSDKv2Version)
}

func TestAccKubernetesResourceQuotaV1_UpgradeFromSDKV2PreIdentity_identity(t *testing.T) {
	testAccResourceQuotaUpgradeIdentity(t, sdkv2PreIdentityProviderVersion)
}

// Another controller's metadata must survive the upgrade AND the first update after it.
//
// The upgrade itself is the easy half: it plans nothing, so nothing can be destroyed. The
// dangerous half is the update that follows. Prior state written by SDKv2 has no keys for the
// map Terraform does not manage, so Update would diff an empty map against an empty map and,
// without the skip in common.MetadataPatchOps, replace the live map with {}.
//
// Both directions are covered, since annotations and labels are separate operations built from
// separate values.
func TestAccKubernetesResourceQuotaV1_UpgradeFromSDKV2_ignoredMetadataSurvivesUpdate(t *testing.T) {
	for _, tc := range []struct{ ignored, managed string }{
		{ignored: "annotations", managed: "labels"},
		{ignored: "labels", managed: "annotations"},
	} {
		t.Run(tc.ignored, func(t *testing.T) {
			var before, after k8sv1.ResourceQuota
			name := testAccResourceQuotaMigrationName()

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckKubernetesResourceQuotaV1Destroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: testAccResourceQuotaExternalProvider(resourceQuotaSDKv2Version),
						Config:            testAccKubernetesResourceQuotaV1Config_managedMetadata(name, tc.managed, "before"),
						Check:             testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &before),
					},
					{
						// Another controller writes to the map Terraform does not manage,
						// while SDKv2 still owns the resource.
						PreConfig:                func() { testAccInjectIgnoredQuotaMetadata(t, name, tc.ignored) },
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   testAccKubernetesResourceQuotaV1Config_managedMetadata(name, tc.managed, "before"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: testAccCheckIgnoredQuotaMetadata(name, tc.ignored),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   testAccKubernetesResourceQuotaV1Config_managedMetadata(name, tc.managed, "after"),
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(resourceQuotaResourceName, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccCheckKubernetesResourceQuotaV1Exists(resourceQuotaResourceName, &after),
							testAccCheckResourceQuotaNotRecreated(&before, &after),
							resource.TestCheckResourceAttr(resourceQuotaResourceName, "metadata.0."+tc.managed+".managed", "after"),
							testAccCheckIgnoredQuotaMetadata(name, tc.ignored),
						),
					},
				},
			})
		})
	}
}

// ignoredQuotaMetadataKey is filtered out of state by common.FlattenNamespacedMetadata for every
// resource, so Terraform never manages it. The API server does not restore it, which is what
// makes it usable as evidence.
const ignoredQuotaMetadataKey = "example.kubernetes.io/owner"

// testAccInjectIgnoredQuotaMetadata writes ignoredQuotaMetadataKey into one of the quota's
// metadata maps, standing in for another controller. The quota and namespace share a name.
func testAccInjectIgnoredQuotaMetadata(t *testing.T, name, field string) {
	t.Helper()

	client, err := mainClientset()
	if err != nil {
		t.Fatal(err)
	}
	patch := fmt.Sprintf(`{"metadata":{%q:{%q:"controller-owned"}}}`, field, ignoredQuotaMetadataKey)
	if _, err := client.CoreV1().ResourceQuotas(name).Patch(context.Background(), name,
		k8stypes.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		t.Fatal(err)
	}
}

// testAccCheckIgnoredQuotaMetadata reads the live object: the ignored key is absent from both
// the plan and state, so only the API server can say whether it survived.
func testAccCheckIgnoredQuotaMetadata(name, field string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := mainClientset()
		if err != nil {
			return err
		}
		quota, err := client.CoreV1().ResourceQuotas(name).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		live := map[string]map[string]string{"annotations": quota.Annotations, "labels": quota.Labels}
		if got := live[field][ignoredQuotaMetadataKey]; got != "controller-owned" {
			return fmt.Errorf("ignored %s[%q] = %q, expected %q (live: %v)",
				field, ignoredQuotaMetadataKey, got, "controller-owned", live[field])
		}
		return nil
	}
}

// SDKv2 accepted a null value in a map and silently dropped the key. The framework rejects it
// at plan, so state created by SDKv2 from a config containing nulls cannot be planned until the
// practitioner removes them.
//
// That is the intended fix, not a regression: under SDKv2 the dropped key reappeared in every
// subsequent plan, because Update wrote the config's null into state while Read rebuilt the map
// from a server response that never had it. For spec.hard a null quantity has no meaning at
// all. Needs a release note.
//
// The last steps prove the documented remediation: a configuration without the nulls plans and
// applies, which also leaves state the harness can destroy.
func testAccResourceQuotaNullInput(t *testing.T, sourceType string, targetConfig func(string) string) {
	t.Helper()
	for _, tc := range []struct{ field, want string }{
		{"labels", `(?s)Invalid Attribute Value.*value must be a string`},
		{"hard", `(?s)Invalid Attribute Value.*quantity`},
	} {
		t.Run(sourceType+"/"+tc.field, func(t *testing.T) {
			name := testAccResourceQuotaMigrationName()
			retype := func(c string) string {
				return strings.Replace(c, `"kubernetes_resource_quota_v1"`, fmt.Sprintf("%q", sourceType), 1)
			}
			sourceConfig := retype(testAccKubernetesResourceQuotaV1Config_nullInput(name, tc.field, true))
			remediated := retype(testAccKubernetesResourceQuotaV1Config_nullInput(name, tc.field, false))
			sdkv2 := testAccResourceQuotaExternalProvider(resourceQuotaSDKv2Version)

			resource.ParallelTest(t, resource.TestCase{
				PreCheck:     func() { testAccPreCheck(t) },
				CheckDestroy: testAccCheckKubernetesResourceQuotaV1Destroy,
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{
					tfversion.SkipBelow(tfversion.Version1_8_0),
				},
				Steps: []resource.TestStep{
					{ExternalProviders: sdkv2, Config: sourceConfig},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   targetConfig(testAccKubernetesResourceQuotaV1Config_nullInput(name, tc.field, true)),
						ExpectError:              regexp.MustCompile(tc.want),
					},
					{ExternalProviders: sdkv2, Config: remediated},
				},
			})
		})
	}
}

func TestAccKubernetesResourceQuotaV1_UpgradeFromSDKV2_nullInput(t *testing.T) {
	testAccResourceQuotaNullInput(t, "kubernetes_resource_quota_v1", func(c string) string { return c })
}

// UpgradeIdentity is the only thing standing between state written before identity existed and
// a resource Terraform cannot refresh. The shared implementation is unit-tested in package
// common; this checks the resource wires it up with its own kind and API version, at the
// version SDKv2 used.
func TestResourceQuotaUpgradeIdentity(t *testing.T) {
	ctx := context.Background()
	r := &corev1.ResourceQuotaV1{}
	var schema fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &schema)
	if schema.IdentitySchema.Version != 1 {
		t.Fatalf("identity version = %d; it must stay compatible with SDKv2's version 1", schema.IdentitySchema.Version)
	}
	upgrader, ok := r.UpgradeIdentity(ctx)[0]
	if !ok {
		t.Fatal("missing version 0 upgrader")
	}
	for _, tc := range []struct {
		name, raw string
		wantError bool
	}{
		{name: "preIdentity"},
		{name: "stored", raw: `{"name":"quota","namespace":"team"}`},
		{name: "invalidJSON", raw: `{`, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := fwresource.UpgradeIdentityRequest{}
			if tc.raw != "" {
				req.RawIdentity = &tfprotov6.RawState{JSON: []byte(tc.raw)}
			}
			resp := fwresource.UpgradeIdentityResponse{Identity: &tfsdk.ResourceIdentity{
				Schema: schema.IdentitySchema,
				Raw:    tftypes.NewValue(schema.IdentitySchema.Type().TerraformType(ctx), nil),
			}}
			upgrader.IdentityUpgrader(ctx, req, &resp)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics: %v", resp.Diagnostics)
			}
			if tc.wantError {
				return
			}
			var got common.NamespacedResourceIdentity
			if diags := resp.Identity.Get(ctx, &got); diags.HasError() {
				t.Fatal(diags)
			}
			if tc.raw == "" {
				if !got.Name.IsNull() || !got.Namespace.IsNull() || !got.Kind.IsNull() || !got.APIVersion.IsNull() {
					t.Fatalf("pre-identity state must stay null until Read: %#v", got)
				}
			} else if got.Name.ValueString() != "quota" || got.Namespace.ValueString() != "team" ||
				got.Kind.ValueString() != "ResourceQuota" || got.APIVersion.ValueString() != "v1" {
				t.Fatalf("incorrect upgraded identity: %#v", got)
			}
		})
	}
}

// testAccKubernetesResourceQuotaV1Config_scenario assembles a config from lines, so the
// scenarios in this file share one shape. A nil spec omits the block; an empty one writes
// `spec {}`. The quota and its namespace share the given name.
func testAccKubernetesResourceQuotaV1Config_scenario(name string, generated bool, metadata, spec, extra []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "resource \"kubernetes_namespace_v1\" \"test\" {\n  metadata {\n    name = %q\n  }\n}\n\n", name)
	b.WriteString("resource \"kubernetes_resource_quota_v1\" \"test\" {\n  metadata {\n")
	if generated {
		fmt.Fprintf(&b, "    generate_name = %q\n", name+"-")
	} else {
		fmt.Fprintf(&b, "    name = %q\n", name)
	}
	b.WriteString("    namespace = kubernetes_namespace_v1.test.metadata[0].name\n")
	for _, line := range metadata {
		fmt.Fprintf(&b, "    %s\n", line)
	}
	b.WriteString("  }\n")
	if spec != nil {
		b.WriteString("\n  spec {\n")
		for _, line := range spec {
			fmt.Fprintf(&b, "    %s\n", line)
		}
		b.WriteString("  }\n")
	}
	if len(extra) > 0 {
		b.WriteString("\n")
		for _, line := range extra {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// managedMetadata is the shape both upgrade paths use for ignored-metadata coverage: one managed
// key in field, the other map never mentioned.
func testAccKubernetesResourceQuotaV1Config_managedMetadata(name, field, value string) string {
	return testAccKubernetesResourceQuotaV1Config_scenario(name, false,
		[]string{fmt.Sprintf("%s = {\n      managed = %q\n    }", field, value)},
		[]string{`hard = { pods = 4 }`}, nil)
}

// nullInput declares a null map value: labels["optional"], or spec.hard["limits.cpu"]. With
// withNull false the key is simply left out, which is the remediation.
func testAccKubernetesResourceQuotaV1Config_nullInput(name, field string, withNull bool) string {
	metadata := []string{`labels = { keep = "retained" }`}
	hard := `hard = { pods = 4 }`
	switch {
	case field == "labels" && withNull:
		metadata = []string{`labels = { keep = "retained", optional = null }`}
	case field == "hard" && withNull:
		hard = `hard = { pods = 4, "limits.cpu" = null }`
	}
	return testAccKubernetesResourceQuotaV1Config_scenario(name, false, metadata, []string{hard}, nil)
}
