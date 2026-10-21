// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	coreapi "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

// Migration tests: state written by released SDKv2 providers, upgraded in place under the
// same resource type by the local Framework build. No MoveState runs here: the Framework
// decodes SDKv2 schema-0 state itself.
//
// Moves from the deprecated kubernetes_service_account and kubernetes_default_service_account
// aliases are in service_account_v1_move_test.go. Both families use the shapes defined here,
// so they describe the same stored state and cannot drift apart.
//
// serviceAccountLiveMigrationEnvironment isolates the Terraform CLI configuration, so a
// developer's dev_overrides cannot substitute the local build for the released provider.
// It sets process environment, so these tests run serially.

// serviceAccountSDKv2Version is the last release that served both ServiceAccount resources
// from SDKv2. The per-shape tests start from it.
const serviceAccountSDKv2Version = "3.4.0"

// serviceAccountSDKv2PreIdentityVersion predates resource identity, which shipped in 2.38.0.
// Its state has no identity, so the upgrade builds one rather than carrying it across.
const serviceAccountSDKv2PreIdentityVersion = "2.37.1"

// serviceAccountReleasedVersions are the releases the fully populated matrix starts from.
var serviceAccountReleasedVersions = []string{serviceAccountSDKv2Version, "3.3.0", "3.2.1", serviceAccountSDKv2PreIdentityVersion}

// serviceAccountDeclaredInternalLabel matches the internal-key filter, so Read keeps it only
// because the configuration declares it.
const serviceAccountDeclaredInternalLabel = "example.kubernetes.io/declared"

type serviceAccountKind = struct {
	name, alias string
	adopt       bool
}

// serviceAccountShape returns one stored-state shape. Each isolates a branch of the state
// translation, so a failure names the branch.
func serviceAccountShape(t *testing.T, shape string) serviceAccountScenario {
	t.Helper()
	complete := serviceAccountScenarios()[1]
	switch shape {
	case "minimum":
		return serviceAccountScenario{name: shape}
	case "annotations":
		return serviceAccountScenario{name: shape, annotations: complete.annotations}
	case "labels":
		return serviceAccountScenario{name: shape, labels: complete.labels}
	case "declared-internal-label":
		return serviceAccountScenario{name: shape, labels: map[string]string{serviceAccountDeclaredInternalLabel: "declared"}}
	case "explicit-empty":
		return serviceAccountScenarios()[2]
	case "generate-name":
		return serviceAccountScenario{name: shape, generate: true}
	case "complete-generate-name":
		complete.name, complete.generate = shape, true
		return complete
	}
	t.Fatalf("unknown ServiceAccount shape %q", shape)
	return serviceAccountScenario{}
}

func serviceAccountUpgradeName(kind serviceAccountKind, scenario serviceAccountScenario) string {
	switch {
	case kind.adopt:
		return "default"
	case scenario.generate:
		return "tf-acc-sa-"
	}
	return "account"
}

// serviceAccountUpgradeConfigs returns the released-provider and local-build configurations.
// With move, the first uses the deprecated alias and the second adds a moved block. Otherwise
// the two are identical, so a non-empty plan can only come from the upgrade.
func serviceAccountUpgradeConfigs(kind serviceAccountKind, move bool, namespace, name string, scenario serviceAccountScenario) (before, after, source, target string) {
	sourceType := kind.name
	if move {
		sourceType = kind.alias
	}
	source, target = sourceType+".test", kind.name+".test"
	fixture := serviceAccountLiveFixture(namespace)
	const namespaceExpr = "kubernetes_namespace.test.metadata[0].name"
	before = fixture + serviceAccountConfig(sourceType, "test", namespaceExpr, name, scenario, true)
	after = fixture + serviceAccountConfig(kind.name, "test", namespaceExpr, name, scenario, true)
	if move {
		after += fmt.Sprintf("\nmoved {\n  from = %s\n  to   = %s\n}\n", source, target)
	}
	return before, after, source, target
}

func serviceAccountReleased(version string) map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{"kubernetes": {Source: "hashicorp/kubernetes", VersionConstraint: "= " + version}}
}

// Cross-type moved blocks need Terraform 1.8.
func serviceAccountUpgradeVersionChecks(move bool) []tfversion.TerraformVersionCheck {
	if move {
		return []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_8_0)}
	}
	return nil
}

// serviceAccountLiveDestroyNamespace also requires the fixture namespace to be gone, so a
// failed run cannot leak it silently.
func serviceAccountLiveDestroyNamespace(namespace string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if err := serviceAccountLiveDestroy(state); err != nil {
			return err
		}
		client, err := mainClientset()
		if err != nil {
			return err
		}
		_, err = client.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err == nil {
			return fmt.Errorf("test-owned namespace still exists: %s", namespace)
		}
		return err
	}
}

func serviceAccountStateObject(state *terraform.State, address string) (*coreapi.ServiceAccount, error) {
	rs, ok := state.RootModule().Resources[address]
	if !ok {
		return nil, fmt.Errorf("%s is not in state", address)
	}
	namespace, name, err := kubernetes.IdParts(rs.Primary.ID)
	if err != nil {
		return nil, err
	}
	return serviceAccountLiveGet(namespace, name)
}

func serviceAccountCaptureUID(address string, uid *k8stypes.UID) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		object, err := serviceAccountStateObject(state, address)
		if err == nil {
			*uid = object.UID
		}
		return err
	}
}

// serviceAccountCheckUID fails if the upgrade replaced the ServiceAccount.
func serviceAccountCheckUID(address string, uid *k8stypes.UID) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		object, err := serviceAccountStateObject(state, address)
		if err != nil {
			return err
		}
		if *uid == "" || object.UID != *uid {
			return fmt.Errorf("ServiceAccount was replaced: UID %q -> %q", *uid, object.UID)
		}
		return nil
	}
}

// testAccServiceAccountUpgrade applies one shape under an exact released SDKv2 provider, then
// plans it under the local build and asserts expect. The post-apply plan proves an expected
// one-off update settles, and the UID check rules out replacement.
func testAccServiceAccountUpgrade(t *testing.T, kind serviceAccountKind, version string, move bool, scenario serviceAccountScenario, expect plancheck.PlanCheck) {
	t.Helper()
	dir := serviceAccountLiveMigrationEnvironment(t)
	namespace := "tf-acc-sa-upgrade-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	before, after, source, target := serviceAccountUpgradeConfigs(kind, move, namespace, serviceAccountUpgradeName(kind, scenario), scenario)
	var uid k8stypes.UID
	resource.Test(t, resource.TestCase{
		PreCheck:               func() { testAccPreCheck(t) },
		TerraformVersionChecks: serviceAccountUpgradeVersionChecks(move),
		CheckDestroy:           serviceAccountLiveDestroyNamespace(namespace),
		Steps: []resource.TestStep{
			{
				ExternalProviders: serviceAccountReleased(version),
				Config:            before,
				Check:             resource.ComposeAggregateTestCheckFunc(serviceAccountReleasedVersion(dir, version), serviceAccountCaptureUID(source, &uid)),
			},
			{
				ProtoV6ProviderFactories: serviceAccountMuxFactories,
				Config:                   after,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply:             []plancheck.PlanCheck{expect},
					PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: serviceAccountCheckUID(target, &uid),
			},
		},
	})
}

// testAccServiceAccountUpgradeShape runs one shape for each resource type that supports it.
// The default ServiceAccount has a fixed name, so generate_name shapes apply only to the
// regular resource.
func testAccServiceAccountUpgradeShape(t *testing.T, version string, move bool, shape string, expect func(address string) plancheck.PlanCheck) {
	t.Helper()
	for _, kind := range serviceAccountTypes {
		scenario := serviceAccountShape(t, shape)
		if kind.adopt && scenario.generate {
			continue
		}
		t.Run(kind.name, func(t *testing.T) {
			testAccServiceAccountUpgrade(t, kind, version, move, scenario, expect(kind.name+".test"))
		})
	}
}

func serviceAccountExpectEmpty(string) plancheck.PlanCheck {
	return plancheck.ExpectEmptyPlan()
}

// serviceAccountExpectUpdate asserts the action rather than tolerating any non-empty plan,
// since a replacement would recreate the ServiceAccount.
func serviceAccountExpectUpdate(address string) plancheck.PlanCheck {
	return plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)
}

func serviceAccountInjectMetadata(t *testing.T, namespace, name, field, key string) {
	t.Helper()
	client, err := mainClientset()
	if err != nil {
		t.Fatal(err)
	}
	patch, err := json.Marshal(map[string]any{"metadata": map[string]any{field: map[string]string{key: "controller-owned"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CoreV1().ServiceAccounts(namespace).Patch(context.Background(), name,
		k8stypes.MergePatchType, patch, metav1.PatchOptions{}); err != nil {
		t.Fatal(err)
	}
}

func serviceAccountCheckMetadataKey(namespace, name, field, key string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		object, err := serviceAccountLiveGet(namespace, name)
		if err != nil {
			return err
		}
		values := object.Labels
		if field == "annotations" {
			values = object.Annotations
		}
		if values[key] != "controller-owned" {
			return fmt.Errorf("%s key %q was removed: %v", field, key, values)
		}
		return nil
	}
}

// testAccServiceAccountIgnoredMetadataSurvivesUpdate covers a key Terraform never manages:
// another controller writes it before the upgrade, and the first update afterwards must patch
// around it rather than replace the live map. ignoredMetadataKey is filtered like a key
// matching ignore_labels, without provider configuration. Both maps are covered, plus the
// case where the managed and ignored keys share a map.
func testAccServiceAccountIgnoredMetadataSurvivesUpdate(t *testing.T, move bool) {
	t.Helper()
	kind := serviceAccountTypes[0]
	const name = "account"
	for _, tc := range []struct{ ignored, managed string }{
		{ignored: "annotations", managed: "labels"},
		{ignored: "labels", managed: "annotations"},
		{ignored: "labels", managed: "labels"},
	} {
		t.Run(tc.ignored+"-ignored-"+tc.managed+"-managed", func(t *testing.T) {
			dir := serviceAccountLiveMigrationEnvironment(t)
			namespace := "tf-acc-sa-ignored-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			managed := func(value string) serviceAccountScenario {
				scenario := serviceAccountScenario{name: "managed-" + tc.managed}
				if tc.managed == "labels" {
					scenario.labels = map[string]string{"managed": value}
				} else {
					scenario.annotations = map[string]string{"managed": value}
				}
				return scenario
			}
			before, upgraded, source, target := serviceAccountUpgradeConfigs(kind, move, namespace, name, managed("before"))
			_, updated, _, _ := serviceAccountUpgradeConfigs(kind, move, namespace, name, managed("after"))
			ignoredKept := serviceAccountCheckMetadataKey(namespace, name, tc.ignored, ignoredMetadataKey)
			var uid k8stypes.UID
			resource.Test(t, resource.TestCase{
				PreCheck:               func() { testAccPreCheck(t) },
				TerraformVersionChecks: serviceAccountUpgradeVersionChecks(move),
				CheckDestroy:           serviceAccountLiveDestroyNamespace(namespace),
				Steps: []resource.TestStep{
					{
						ExternalProviders: serviceAccountReleased(serviceAccountSDKv2Version),
						Config:            before,
						Check: resource.ComposeAggregateTestCheckFunc(
							serviceAccountReleasedVersion(dir, serviceAccountSDKv2Version), serviceAccountCaptureUID(source, &uid)),
					},
					{
						PreConfig:                func() { serviceAccountInjectMetadata(t, namespace, name, tc.ignored, ignoredMetadataKey) },
						ProtoV6ProviderFactories: serviceAccountMuxFactories,
						Config:                   upgraded,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(serviceAccountCheckUID(target, &uid), ignoredKept),
					},
					{
						ProtoV6ProviderFactories: serviceAccountMuxFactories,
						Config:                   updated,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{serviceAccountExpectUpdate(target)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							serviceAccountCheckUID(target, &uid),
							resource.TestCheckResourceAttr(target, "metadata.0."+tc.managed+".managed", "after"),
							ignoredKept,
						),
					},
				},
			})
		})
	}
}

// The baseline shape: no optional fields, so both metadata maps are stored null.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, false, "minimum", serviceAccountExpectEmpty)
}

// name is server-assigned, Optional+Computed and RequiresReplace. If the upgrade left it
// unknown, the plan would replace the ServiceAccount. SDKv2 also stores a real generate_name
// here instead of the "" it writes when the field was never set.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_generateName(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, false, "generate-name", serviceAccountExpectEmpty)
}

// annotations populated, labels absent: one map stored null and one not.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_annotations(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, false, "annotations", serviceAccountExpectEmpty)
}

// The mirror: labels populated, annotations absent.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_labels(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, false, "labels", serviceAccountExpectEmpty)
}

// A key the filtering rules would strip, declared in config so it must survive the upgrade
// and the Read that follows it.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_declaredInternalLabel(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, false, "declared-internal-label", serviceAccountExpectEmpty)
}

// Every optional field populated, named by generate_name. The named equivalent runs from
// every release in TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_releases.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_completeGenerateName(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2Version, false, "complete-generate-name", serviceAccountExpectEmpty)
}

// Every attribute populated, from every supported release, as a same-type upgrade and as a
// same-type address move. Stronger than the per-shape tests: it also requires state, outputs,
// identity and the live object (including resourceVersion) to be unchanged.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_releases(t *testing.T) {
	testAccServiceAccountReleaseMatrix(t, "same-type", "address-move")
}

// The pre-identity tests are the only coverage of state written before resource identity
// existed, where UpgradeIdentity starts from no identity. The fully populated pre-identity
// shape runs in the release matrix.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2PreIdentity_basic(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2PreIdentityVersion, false, "minimum", serviceAccountExpectEmpty)
}

func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2PreIdentity_basicGenerateName(t *testing.T) {
	testAccServiceAccountUpgradeShape(t, serviceAccountSDKv2PreIdentityVersion, false, "generate-name", serviceAccountExpectEmpty)
}

func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_ignoredMetadataSurvivesUpdate(t *testing.T) {
	testAccServiceAccountIgnoredMetadataSurvivesUpdate(t, false)
}

func serviceAccountLiveMigrationEnvironment(t *testing.T) string {
	t.Helper()
	// Keep HOME and all KUBE settings: live tests must use the explicitly chosen cluster.
	if os.Getenv("TF_ACC") == "" {
		t.Skip("live migration requires TF_ACC")
	}
	cli, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal(err)
	}
	// Keep provider reattachment sockets below macOS's 104-byte path limit,
	// including when this repository is checked out in a deeply nested worktree.
	t.Setenv("TMPDIR", "/tmp")
	dir := t.TempDir()
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "TF_CLI_ARGS") || strings.HasPrefix(name, "TF_VAR_") ||
			name == "TF_REATTACH_PROVIDERS" || name == "TF_PLUGIN_CACHE_DIR" || name == "TF_ACC_TERRAFORM_VERSION" ||
			name == "TF_DATA_DIR" || name == "TF_WORKSPACE" {
			t.Setenv(name, "")
		}
	}
	rc := filepath.Join(dir, "terraformrc")
	if err := os.WriteFile(rc, []byte("disable_checkpoint = true\nprovider_installation {\n  direct {}\n}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TF_CLI_CONFIG_FILE", rc)
	t.Setenv("TF_ACC_TERRAFORM_PATH", cli)
	t.Setenv("TF_ACC_TEMP_DIR", dir)
	t.Setenv("CHECKPOINT_DISABLE", "1")
	return dir
}

const serviceAccountControllerAnnotation = "controller.kubernetes.io/service-account-migration"

type serviceAccountLiveSnapshot struct {
	read     func() (*coreapi.ServiceAccount, error)
	object   *coreapi.ServiceAccount
	resource *tfjson.StateResource
	outputs  map[string]*tfjson.StateOutput
}

func serviceAccountStateResource(values *tfjson.StateValues, address string) *tfjson.StateResource {
	if values == nil || values.RootModule == nil {
		return nil
	}
	for _, r := range values.RootModule.Resources {
		if r.Address == address {
			return r
		}
	}
	return nil
}

func (snapshot *serviceAccountLiveSnapshot) check(address string, capture bool) statecheck.StateCheck {
	return serviceAccountStateCheck(func(req statecheck.CheckStateRequest) error {
		r := serviceAccountStateResource(req.State.Values, address)
		if r == nil || r.SchemaVersion != 0 || r.Tainted || r.DeposedKey != "" {
			return fmt.Errorf("missing or invalid ServiceAccount schema0 state at %s", address)
		}
		object, err := snapshot.read()
		if err != nil {
			return err
		}
		if object.Annotations[serviceAccountControllerAnnotation] != "preserve" {
			return fmt.Errorf("controller annotation disappeared")
		}
		if capture {
			snapshot.object, snapshot.resource, snapshot.outputs = object.DeepCopy(), r, req.State.Values.Outputs
			return nil
		}
		normalized := *r
		normalized.Address, normalized.Type, normalized.Name = snapshot.resource.Address, snapshot.resource.Type, snapshot.resource.Name
		// Pre-identity releases legitimately gain identity1, checked independently below.
		if snapshot.resource.IdentitySchemaVersion == nil || *snapshot.resource.IdentitySchemaVersion == 0 {
			normalized.IdentitySchemaVersion, normalized.IdentityValues = snapshot.resource.IdentitySchemaVersion, snapshot.resource.IdentityValues
		}
		if !reflect.DeepEqual(snapshot.resource, &normalized) || !reflect.DeepEqual(snapshot.outputs, req.State.Values.Outputs) ||
			!reflect.DeepEqual(snapshot.object, object) {
			return fmt.Errorf("migration changed complete SA state, outputs, or API object (including UID/resourceVersion)")
		}
		return nil
	})
}

type serviceAccountLivePlanGuard struct {
	snapshot *serviceAccountLiveSnapshot
	address  string
}

func (guard serviceAccountLivePlanGuard) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if guard.snapshot == nil || guard.snapshot.resource == nil || guard.snapshot.object == nil {
		resp.Error = fmt.Errorf("live migration has no protected baseline")
		return
	}
	if req.Plan == nil || (req.Plan.Complete != nil && !*req.Plan.Complete) || len(req.Plan.DeferredChanges) != 0 {
		resp.Error = fmt.Errorf("live migration plan is incomplete")
		return
	}
	r := serviceAccountStateResource(req.Plan.PlannedValues, guard.address)
	if r == nil || r.SchemaVersion != 0 || !reflect.DeepEqual(r.AttributeValues, guard.snapshot.resource.AttributeValues) ||
		!reflect.DeepEqual(req.Plan.PlannedValues.Outputs, guard.snapshot.outputs) {
		resp.Error = fmt.Errorf("live migration changed protected planned SA state or outputs before apply")
		return
	}
	for _, drift := range req.Plan.ResourceDrift {
		if drift.Address == guard.address || drift.Address == guard.snapshot.resource.Address {
			if drift.Change == nil || !drift.Change.Actions.NoOp() || !reflect.DeepEqual(drift.Change.Before, drift.Change.After) {
				resp.Error = fmt.Errorf("live migration normalized protected SA state during refresh")
				return
			}
		}
	}
	object, err := guard.snapshot.read()
	if err != nil {
		resp.Error = err
	} else if !reflect.DeepEqual(object, guard.snapshot.object) {
		resp.Error = fmt.Errorf("live migration wrote or replaced the SA before apply")
	}
}

func TestServiceAccountLiveMigrationGuards(t *testing.T) {
	const address = "kubernetes_service_account_v1.test"
	for _, tc := range []string{"unchanged", "missing-account", "changed-state", "changed-output", "fixture-update", "changed-uid", "changed-resource-version"} {
		t.Run(tc, func(t *testing.T) {
			object := &coreapi.ServiceAccount{ObjectMeta: metav1.ObjectMeta{UID: "original", ResourceVersion: "1"}}
			current := object.DeepCopy()
			baseline := &tfjson.StateResource{Address: address, AttributeValues: map[string]any{"id": "owned/account"}}
			planned := &tfjson.StateResource{Address: address, AttributeValues: map[string]any{"id": "owned/account"}}
			snapshot := &serviceAccountLiveSnapshot{
				object: object, resource: baseline,
				read: func() (*coreapi.ServiceAccount, error) { return current, nil },
			}
			plan := &tfjson.Plan{
				PlannedValues: &tfjson.StateValues{RootModule: &tfjson.StateModule{Resources: []*tfjson.StateResource{planned}}},
				ResourceChanges: []*tfjson.ResourceChange{
					{Address: address, Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}}},
					{Address: "kubernetes_namespace.test", Change: &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}}},
				},
			}
			switch tc {
			case "missing-account":
				plan.PlannedValues.RootModule.Resources = nil
			case "changed-state":
				planned.AttributeValues["id"] = "owned/different"
			case "changed-output":
				plan.PlannedValues.Outputs = map[string]*tfjson.StateOutput{"account": {Value: "different"}}
			case "fixture-update":
				plan.ResourceChanges[1].Change.Actions = tfjson.Actions{tfjson.ActionUpdate}
			case "changed-uid":
				current.UID = "replacement"
			case "changed-resource-version":
				current.ResourceVersion = "2"
			}
			failed := false
			for _, check := range []plancheck.PlanCheck{plancheck.ExpectEmptyPlan(), serviceAccountLivePlanGuard{snapshot: snapshot, address: address}} {
				var response plancheck.CheckPlanResponse
				check.CheckPlan(context.Background(), plancheck.CheckPlanRequest{Plan: plan}, &response)
				failed = failed || response.Error != nil
			}
			if failed != (tc != "unchanged") {
				t.Fatalf("guard rejected plan=%t, want %t", failed, tc != "unchanged")
			}
		})
	}
}

// testAccServiceAccountReleaseMatrix migrates the fully populated shape from every supported
// release through each requested route. Fixtures stay on SDKv2 throughout; only the
// ServiceAccount migrates. Identity assertions need Terraform 1.12.
func testAccServiceAccountReleaseMatrix(t *testing.T, moves ...string) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("live exact-release migrations require TF_ACC")
	}
	// Serial execution is required by the per-case CLI configuration environment.
	for _, kind := range serviceAccountTypes {
		for _, version := range serviceAccountReleasedVersions {
			for _, move := range moves {
				t.Run(kind.name+"/"+version+"/"+move, func(t *testing.T) {
					testAccPreCheck(t)
					dir := serviceAccountLiveMigrationEnvironment(t)
					namespace := "tf-acc-sa-migrate-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
					name := "account"
					if kind.adopt {
						name = "default"
					}
					sourceType, sourceLabel := kind.name, "test"
					switch move {
					case "alias-move":
						sourceType = kind.alias
					case "address-move":
						sourceLabel = "old"
					}
					source, target := sourceType+"."+sourceLabel, kind.name+".test"
					fixture := serviceAccountLiveFixture(namespace)
					scenario := serviceAccountScenarios()[1]
					build := func(kind, label string) string {
						return fixture + serviceAccountConfig(kind, label, "kubernetes_namespace.test.metadata[0].name", name, scenario, true) +
							serviceAccountOutputs(kind+"."+label)
					}
					before, after := build(sourceType, sourceLabel), build(kind.name, "test")
					if move != "same-type" {
						after += fmt.Sprintf("\nmoved {\n  from = %s\n  to = %s\n}\n", source, target)
					} else if before != after {
						t.Fatal("same-type upgrade HCL must be byte-identical")
					}
					snapshot := &serviceAccountLiveSnapshot{read: func() (*coreapi.ServiceAccount, error) {
						return serviceAccountLiveGet(namespace, name)
					}}
					released := serviceAccountReleased(version)
					check := func(address string) resource.TestCheckFunc {
						return func(state *terraform.State) error {
							object, err := snapshot.read()
							if err != nil {
								return err
							}
							return serviceAccountCheckValues(state, address, object, scenario)
						}
					}
					guards := resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(), serviceAccountLivePlanGuard{snapshot: snapshot, address: target},
					}}
					resource.Test(t, resource.TestCase{
						PreCheck:               func() { testAccPreCheck(t) },
						TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
						CheckDestroy:           serviceAccountLiveDestroyNamespace(namespace),
						Steps: []resource.TestStep{
							{ExternalProviders: released, Config: before, Check: resource.ComposeAggregateTestCheckFunc(serviceAccountReleasedVersion(dir, version), check(source))},
							{
								PreConfig: func() {
									client, err := mainClientset()
									if err != nil {
										t.Fatal(err)
									}
									patch, _ := json.Marshal(map[string]any{"metadata": map[string]any{"annotations": map[string]string{serviceAccountControllerAnnotation: "preserve"}}})
									if _, err := client.CoreV1().ServiceAccounts(namespace).Patch(context.Background(), name, k8stypes.MergePatchType, patch,
										metav1.PatchOptions{FieldManager: "service-account-migration-controller"}); err != nil {
										t.Fatal(err)
									}
								},
								ExternalProviders: released, Config: before, Check: check(source),
								ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
								ConfigStateChecks: []statecheck.StateCheck{snapshot.check(source, true)},
							},
							{
								ProtoV6ProviderFactories: serviceAccountMuxFactories, Config: after, Check: check(target),
								ConfigPlanChecks:  guards,
								ConfigStateChecks: []statecheck.StateCheck{snapshot.check(target, false), serviceAccountIdentity(target, namespace, name)},
							},
							{
								ProtoV6ProviderFactories: serviceAccountMuxFactories, Config: after, Check: check(target),
								ConfigPlanChecks:  guards,
								ConfigStateChecks: []statecheck.StateCheck{snapshot.check(target, false), serviceAccountIdentity(target, namespace, name)},
							},
						},
					})
				})
			}
		}
	}
}

// SDKv2 stores `labels = {}` and `annotations = {}` as null until a later refresh
// persists {}. The Framework keeps the distinction, so upgrading fresh SDKv2
// state plans one in-place update. It must write nothing to the API and settle.
// The namespace equivalent is testAccNamespaceMigrationExpectingUpdate.
func TestAccKubernetesServiceAccountV1_UpgradeFromSDKV2_emptyValues(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("live exact-release migrations require TF_ACC")
	}
	const version = serviceAccountSDKv2Version
	scenario := serviceAccountScenarios()[2]
	if scenario.name != "explicit-empty" {
		t.Fatalf("scenario = %q, want explicit-empty", scenario.name)
	}
	for _, kind := range serviceAccountTypes {
		t.Run(kind.name, func(t *testing.T) {
			testAccPreCheck(t)
			dir := serviceAccountLiveMigrationEnvironment(t)
			namespace := "tf-acc-sa-empty-" + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
			name := "account"
			if kind.adopt {
				name = "default"
			}
			address := kind.name + ".test"
			config := serviceAccountLiveFixture(namespace) +
				serviceAccountConfig(kind.name, "test", "kubernetes_namespace.test.metadata[0].name", name, scenario, true)
			var before *coreapi.ServiceAccount
			capture := func(*terraform.State) error {
				object, err := serviceAccountLiveGet(namespace, name)
				before = object
				return err
			}
			unchanged := func(*terraform.State) error {
				object, err := serviceAccountLiveGet(namespace, name)
				if err != nil {
					return err
				}
				if before == nil || object.UID != before.UID || object.ResourceVersion != before.ResourceVersion {
					return fmt.Errorf("upgrade wrote or replaced the ServiceAccount")
				}
				return nil
			}
			emptyMap := func(field string) statecheck.StateCheck {
				return statecheck.ExpectKnownValue(address, tfjsonpath.New("metadata").AtSliceIndex(0).AtMapKey(field),
					knownvalue.MapExact(map[string]knownvalue.Check{}))
			}
			resource.Test(t, resource.TestCase{
				PreCheck:               func() { testAccPreCheck(t) },
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_12_0)},
				CheckDestroy:           serviceAccountLiveDestroy,
				Steps: []resource.TestStep{
					{
						ExternalProviders: serviceAccountReleased(version),
						Config:            config,
						Check:             resource.ComposeAggregateTestCheckFunc(serviceAccountReleasedVersion(dir, version), capture),
					},
					{
						ProtoV6ProviderFactories: serviceAccountMuxFactories, Config: config, Check: unchanged,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply:             []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate)},
							PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
						},
						ConfigStateChecks: []statecheck.StateCheck{emptyMap("labels"), emptyMap("annotations")},
					},
					{
						ProtoV6ProviderFactories: serviceAccountMuxFactories, Config: config, Check: unchanged,
						ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
					},
				},
			})
		})
	}
}
