// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestAccDeploymentV1_SelectorNullEmptyMigration(t *testing.T) {
	for _, tc := range []struct {
		name          string
		explicitEmpty bool
	}{
		{name: "omitted"},
		{name: "explicit-empty", explicitEmpty: true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			namespace := acctest.RandomWithPrefix("tf-selector-deploy-ns")
			name := acctest.RandomWithPrefix("tf-selector-deploy")
			config := workloadSelectorDeploymentConfig(namespace, name, tc.explicitEmpty)
			convergedConfig := config
			if tc.explicitEmpty {
				convergedConfig = workloadSelectorDeploymentConfig(namespace, name, false)
			}

			const address = "kubernetes_deployment_v1.test"
			migrationPlanChecks := workloadSelectorNormalizationPlanChecks(address, tc.explicitEmpty)
			convergencePlanChecks := workloadSelectorNoOpPlanChecks()
			if tc.explicitEmpty {
				convergencePlanChecks = workloadSelectorNormalizationPlanChecks(address, false)
			}
			var snapshot deploymentMigrationSnapshot

			resource.ParallelTest(t, resource.TestCase{
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_5_0)},
				PreCheck: func() {
					testAccPreCheck(t)
					skipIfNotRunningInKind(t)
				},
				CheckDestroy: workloadSelectorCheckDestroy(testAccCheckKubernetesDeploymentV1Destroy),
				Steps: []resource.TestStep{
					{
						ExternalProviders: workloadSelectorReleasedProvider(deploymentSDKv2ProviderVersion),
						Config:            config,
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccDeploymentMigrationCapture(address, &snapshot),
							workloadSelectorStateCheck(address, namespace, name),
							workloadSelectorDeploymentAPICheck(&snapshot),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   config,
						ConfigPlanChecks:         migrationPlanChecks,
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccDeploymentMigrationUnchanged(address, &snapshot),
							workloadSelectorStateCheck(address, namespace, name),
							workloadSelectorDeploymentAPICheck(&snapshot),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ConfigPlanChecks:         convergencePlanChecks,
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccDeploymentMigrationUnchanged(address, &snapshot),
							workloadSelectorStateCheck(address, namespace, name),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ResourceName:             address,
						ImportState:              true,
						ImportStateId:            namespace + "/" + name,
						ImportStateCheck: workloadSelectorImportCheck(namespace, name, true, func() types.UID {
							return snapshot.deployment.UID
						}),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ResourceName:             address,
						ImportState:              true,
						ImportStateKind:          resource.ImportBlockWithID,
						ImportStateId:            namespace + "/" + name,
						ImportPlanChecks: resource.ImportPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionNoop)},
						},
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ConfigPlanChecks:         workloadSelectorNoOpPlanChecks(),
						Check: resource.ComposeAggregateTestCheckFunc(
							testAccDeploymentMigrationUnchanged(address, &snapshot),
							workloadSelectorStateCheck(address, namespace, name),
						),
					},
				},
			})
		})
	}
}

func TestAccDaemonSetV1_SelectorNullEmptyMigration(t *testing.T) {
	for _, tc := range []struct {
		name          string
		explicitEmpty bool
	}{
		{name: "omitted"},
		{name: "explicit-empty", explicitEmpty: true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			namespace := acctest.RandomWithPrefix("tf-selector-daemon-ns")
			name := acctest.RandomWithPrefix("tf-selector-daemon")
			config := workloadSelectorDaemonSetConfig(namespace, name, tc.explicitEmpty, true)
			convergedConfig := config
			if tc.explicitEmpty {
				convergedConfig = workloadSelectorDaemonSetConfig(namespace, name, false, true)
			}

			migrationPlanChecks := workloadSelectorNormalizationPlanChecks(daemonSetResourceName, tc.explicitEmpty)
			convergencePlanChecks := workloadSelectorNoOpPlanChecks()
			if tc.explicitEmpty {
				convergencePlanChecks = workloadSelectorNormalizationPlanChecks(daemonSetResourceName, false)
			}
			var snapshot workloadSelectorDaemonSetSnapshot

			resource.ParallelTest(t, resource.TestCase{
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_5_0)},
				PreCheck: func() {
					testAccPreCheck(t)
					skipIfNotRunningInKind(t)
				},
				CheckDestroy: workloadSelectorCheckDestroy(testAccCheckDaemonSetDestroy),
				Steps: []resource.TestStep{
					{
						ExternalProviders: workloadSelectorReleasedProvider(daemonSetSDKv2ProviderV321),
						Config:            config,
						Check: resource.ComposeAggregateTestCheckFunc(
							workloadSelectorCaptureDaemonSet(daemonSetResourceName, &snapshot),
							workloadSelectorStateCheck(daemonSetResourceName, namespace, name),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   config,
						ConfigPlanChecks:         migrationPlanChecks,
						Check: resource.ComposeAggregateTestCheckFunc(
							workloadSelectorCheckDaemonSetUnchanged(daemonSetResourceName, &snapshot),
							workloadSelectorStateCheck(daemonSetResourceName, namespace, name),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ConfigPlanChecks:         convergencePlanChecks,
						Check: resource.ComposeAggregateTestCheckFunc(
							workloadSelectorCheckDaemonSetUnchanged(daemonSetResourceName, &snapshot),
							workloadSelectorStateCheck(daemonSetResourceName, namespace, name),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ResourceName:             daemonSetResourceName,
						ImportState:              true,
						ImportStateId:            namespace + "/" + name,
						ImportStateCheck: workloadSelectorImportCheck(namespace, name, false, func() types.UID {
							return snapshot.object.UID
						}),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						// Import cannot recover the Terraform-only wait policy and retains the SDKv2 false default.
						Config:          workloadSelectorDaemonSetConfig(namespace, name, false, false),
						ResourceName:    daemonSetResourceName,
						ImportState:     true,
						ImportStateKind: resource.ImportBlockWithID,
						ImportStateId:   namespace + "/" + name,
						ImportPlanChecks: resource.ImportPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(daemonSetResourceName, plancheck.ResourceActionNoop)},
						},
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ConfigPlanChecks:         workloadSelectorNoOpPlanChecks(),
						Check: resource.ComposeAggregateTestCheckFunc(
							workloadSelectorCheckDaemonSetUnchanged(daemonSetResourceName, &snapshot),
							workloadSelectorStateCheck(daemonSetResourceName, namespace, name),
						),
					},
				},
			})
		})
	}
}

func TestAccStatefulSetV1_SelectorNullEmptyMigration(t *testing.T) {
	for _, tc := range []struct {
		name          string
		explicitEmpty bool
	}{
		{name: "omitted"},
		{name: "explicit-empty", explicitEmpty: true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			namespace := acctest.RandomWithPrefix("tf-selector-stateful-ns")
			name := acctest.RandomWithPrefix("tf-selector-stateful")
			config := workloadSelectorStatefulSetConfig(namespace, name, tc.explicitEmpty)
			convergedConfig := config
			if tc.explicitEmpty {
				convergedConfig = workloadSelectorStatefulSetConfig(namespace, name, false)
			}

			const address = "kubernetes_stateful_set_v1.test"
			migrationPlanChecks := workloadSelectorNormalizationPlanChecks(address, tc.explicitEmpty)
			convergencePlanChecks := workloadSelectorNoOpPlanChecks()
			if tc.explicitEmpty {
				convergencePlanChecks = workloadSelectorNormalizationPlanChecks(address, false)
			}
			var snapshot statefulSetMigrationSnapshot

			resource.ParallelTest(t, resource.TestCase{
				TerraformVersionChecks: []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_5_0)},
				PreCheck: func() {
					testAccPreCheck(t)
					skipIfNotRunningInKind(t)
				},
				CheckDestroy: workloadSelectorCheckDestroy(testAccCheckKubernetesStatefulSetV1Destroy),
				Steps: []resource.TestStep{
					{
						ExternalProviders: workloadSelectorReleasedProvider(statefulSetSDKv2ProviderVersion),
						Config:            config,
						Check: resource.ComposeAggregateTestCheckFunc(
							statefulSetCaptureMigrationSnapshot(address, &snapshot),
							workloadSelectorStateCheck(address, namespace, name),
							workloadSelectorStatefulSetAPICheck(&snapshot),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   config,
						ConfigPlanChecks:         migrationPlanChecks,
						Check: resource.ComposeAggregateTestCheckFunc(
							statefulSetCheckMigrationSnapshot(address, &snapshot),
							workloadSelectorStateCheck(address, namespace, name),
							workloadSelectorStatefulSetAPICheck(&snapshot),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ConfigPlanChecks:         convergencePlanChecks,
						Check: resource.ComposeAggregateTestCheckFunc(
							statefulSetCheckMigrationSnapshot(address, &snapshot),
							workloadSelectorStateCheck(address, namespace, name),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ResourceName:             address,
						ImportState:              true,
						ImportStateId:            namespace + "/" + name,
						ImportStateCheck: workloadSelectorImportCheck(namespace, name, true, func() types.UID {
							if snapshot.object == nil {
								return ""
							}
							return snapshot.object.UID
						}),
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ResourceName:             address,
						ImportState:              true,
						ImportStateKind:          resource.ImportBlockWithID,
						ImportStateId:            namespace + "/" + name,
						ImportPlanChecks: resource.ImportPlanChecks{
							PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionNoop)},
						},
					},
					{
						ProtoV6ProviderFactories: testAccProviderFactories,
						Config:                   convergedConfig,
						ConfigPlanChecks:         workloadSelectorNoOpPlanChecks(),
						Check: resource.ComposeAggregateTestCheckFunc(
							statefulSetCheckMigrationSnapshot(address, &snapshot),
							workloadSelectorStateCheck(address, namespace, name),
						),
					},
				},
			})
		})
	}
}

func workloadSelectorReleasedProvider(version string) map[string]resource.ExternalProvider {
	return map[string]resource.ExternalProvider{
		"kubernetes": {
			Source:            "hashicorp/kubernetes",
			VersionConstraint: version,
		},
	}
}

func workloadSelectorNoOpPlanChecks() resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{
		PreApply:             []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
}

type workloadSelectorNormalizationPlanCheck struct {
	address string
	toEmpty bool
}

func (check workloadSelectorNormalizationPlanCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	if req.Plan == nil {
		resp.Error = fmt.Errorf("plan is nil")
		return
	}
	found := false
	for _, change := range req.Plan.ResourceChanges {
		if change == nil || change.Change == nil {
			resp.Error = fmt.Errorf("plan contains a nil resource change")
			return
		}
		if len(change.Change.ReplacePaths) != 0 {
			resp.Error = fmt.Errorf("unexpected replacement paths for %s: %v", change.Address, change.Change.ReplacePaths)
			return
		}
		if change.Address != check.address {
			if !change.Change.Actions.NoOp() {
				resp.Error = fmt.Errorf("unexpected action for %s: %v", change.Address, change.Change.Actions)
				return
			}
			continue
		}
		found = true
		if change.Change.Actions.NoOp() {
			continue
		}
		if !reflect.DeepEqual(change.Change.Actions, tfjson.Actions{tfjson.ActionUpdate}) {
			resp.Error = fmt.Errorf("unexpected action for %s: %v", change.Address, change.Change.Actions)
			return
		}
		const matchLabelsPath = "spec.0.selector.0.match_labels"
		const valuesPath = "spec.0.selector.0.match_expressions.0.values"
		expected := make(map[string]bool, 2)
		expected[matchLabelsPath] = false
		expected[valuesPath] = false
		matches := func(path string, before, after any) bool {
			if check.toEmpty {
				before, after = after, before
			}
			if after != nil {
				return false
			}
			switch path {
			case matchLabelsPath:
				value, ok := before.(map[string]any)
				return ok && len(value) == 0
			case valuesPath:
				value, ok := before.([]any)
				return ok && len(value) == 0
			default:
				return false
			}
		}
		if err := deploymentCheckNormalization("", change.Change.Before, change.Change.After, change.Change.AfterUnknown, expected, matches, true); err != nil {
			resp.Error = err
			return
		}
		if !expected[matchLabelsPath] && !expected[valuesPath] {
			resp.Error = fmt.Errorf("%s update did not normalize selector empty leaves", check.address)
			return
		}
	}
	if !found {
		resp.Error = fmt.Errorf("resource %s not found in plan", check.address)
	}
}

func workloadSelectorNormalizationPlanChecks(address string, toEmpty bool) resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{
		PreApply: []plancheck.PlanCheck{
			workloadSelectorNormalizationPlanCheck{address: address, toEmpty: toEmpty},
		},
		PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
	}
}

func TestWorkloadSelectorNormalizationPlanGuard(t *testing.T) {
	const address = "kubernetes_deployment_v1.test"
	for _, tc := range []struct {
		name    string
		mutate  func(*tfjson.ResourceChange)
		wantErr bool
		toEmpty bool
	}{
		{name: "selector-normalization"},
		{name: "explicit-empty-normalization", toEmpty: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.Before, change.Change.After = change.Change.After, change.Change.Before
		}},
		{name: "wrong-direction", toEmpty: true, wantErr: true},
		{name: "nonempty-addition", toEmpty: true, wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.Before, change.Change.After = change.Change.After, change.Change.Before
			after := change.Change.After.(map[string]any)["spec"].([]any)[0].(map[string]any)["selector"].([]any)[0].(map[string]any)
			after["match_labels"] = map[string]any{"app": ""}
		}},
		{name: "no-op", mutate: func(change *tfjson.ResourceChange) {
			change.Change.Actions = tfjson.Actions{tfjson.ActionNoop}
			change.Change.After = change.Change.Before
		}},
		{name: "replacement", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.Actions = tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
		}},
		{name: "replacement-path", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.ReplacePaths = []any{[]any{"spec", 0, "selector"}}
		}},
		{name: "create", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.Actions = tfjson.Actions{tfjson.ActionCreate}
		}},
		{name: "delete", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.Actions = tfjson.Actions{tfjson.ActionDelete}
		}},
		{name: "other-resource", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Address = "kubernetes_namespace_v1.test"
		}},
		{name: "nonempty-label-removal", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			before := change.Change.Before.(map[string]any)["spec"].([]any)[0].(map[string]any)["selector"].([]any)[0].(map[string]any)
			before["match_labels"] = map[string]any{"app": "database"}
		}},
		{name: "real-expression-change", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			after := change.Change.After.(map[string]any)["spec"].([]any)[0].(map[string]any)["selector"].([]any)[0].(map[string]any)
			after["match_expressions"].([]any)[0].(map[string]any)["operator"] = "DoesNotExist"
		}},
		{name: "unknown-selector", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.AfterUnknown = map[string]any{"spec": []any{map[string]any{"selector": true}}}
		}},
		{name: "unknown-identity", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.AfterUnknown = map[string]any{"id": true}
		}},
		{name: "image-change", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.After.(map[string]any)["spec"].([]any)[0].(map[string]any)["template"] = []any{map[string]any{"image": "changed"}}
		}},
		{name: "missing-normalization", wantErr: true, mutate: func(change *tfjson.ResourceChange) {
			change.Change.After = change.Change.Before
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := func(empty bool) map[string]any {
				var matchLabels, values any
				if empty {
					matchLabels, values = map[string]any{}, []any{}
				}
				return map[string]any{"id": "selector-guard/test", "spec": []any{map[string]any{
					"selector": []any{map[string]any{
						"match_labels": matchLabels,
						"match_expressions": []any{map[string]any{
							"key": "app", "operator": "Exists", "values": values,
						}},
					}},
				}}}
			}
			change := &tfjson.ResourceChange{Address: address, Change: &tfjson.Change{
				Actions: tfjson.Actions{tfjson.ActionUpdate}, Before: value(true), After: value(false),
			}}
			if tc.mutate != nil {
				tc.mutate(change)
			}
			response := &plancheck.CheckPlanResponse{}
			workloadSelectorNormalizationPlanCheck{address: address, toEmpty: tc.toEmpty}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{
				Plan: &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{change}},
			}, response)
			if (response.Error != nil) != tc.wantErr {
				t.Fatalf("guard error = %v, want error: %t", response.Error, tc.wantErr)
			}
		})
	}
}

func workloadSelectorCheckDestroy(workloadCheck resource.TestCheckFunc) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		if err := workloadCheck(state); err != nil {
			return err
		}
		client, err := testAccWorkloadClient()
		if err != nil {
			return err
		}
		for _, resourceState := range state.RootModule().Resources {
			if resourceState.Type != "kubernetes_namespace_v1" && resourceState.Type != "kubernetes_namespace" {
				continue
			}
			name := resourceState.Primary.ID
			_, err := client.CoreV1().Namespaces().Get(context.Background(), name, metav1.GetOptions{})
			if err == nil {
				return fmt.Errorf("namespace still exists: %s", name)
			}
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("checking namespace %s deletion: %w", name, err)
			}
		}
		return nil
	}
}

func workloadSelectorStateCheck(address, namespace, name string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr("kubernetes_namespace_v1.test", "metadata.0.name", namespace),
		resource.TestCheckResourceAttr(address, "metadata.0.name", name),
		resource.TestCheckResourceAttr(address, "metadata.0.namespace", namespace),
		resource.TestCheckResourceAttrSet(address, "metadata.0.uid"),
		resource.TestCheckResourceAttr(address, "wait_for_rollout", "true"),
		resource.TestCheckResourceAttr(address, "spec.0.selector.#", "1"),
		resource.TestCheckResourceAttr(address, "spec.0.selector.0.match_labels.%", "0"),
		resource.TestCheckResourceAttr(address, "spec.0.selector.0.match_expressions.#", "1"),
		resource.TestCheckResourceAttr(address, "spec.0.selector.0.match_expressions.0.key", "app"),
		resource.TestCheckResourceAttr(address, "spec.0.selector.0.match_expressions.0.operator", "Exists"),
		resource.TestCheckResourceAttr(address, "spec.0.selector.0.match_expressions.0.values.#", "0"),
		resource.TestCheckResourceAttr(address, "spec.0.template.0.metadata.0.labels.app", "database"),
	)
}

func workloadSelectorImportCheck(namespace, name string, waitForRollout bool, expectedUID func() types.UID) func([]*terraform.InstanceState) error {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("import returned %d states, want 1", len(states))
		}
		id := namespace + "/" + name
		if states[0].ID != id {
			return fmt.Errorf("imported ID %q, want %q", states[0].ID, id)
		}
		for _, expected := range []struct {
			path  string
			value string
		}{
			{path: "metadata.0.name", value: name},
			{path: "metadata.0.namespace", value: namespace},
			{path: "metadata.0.uid", value: string(expectedUID())},
			{path: "wait_for_rollout", value: fmt.Sprint(waitForRollout)},
			{path: "spec.0.selector.#", value: "1"},
			{path: "spec.0.selector.0.match_expressions.#", value: "1"},
			{path: "spec.0.selector.0.match_expressions.0.key", value: "app"},
			{path: "spec.0.selector.0.match_expressions.0.operator", value: "Exists"},
			{path: "spec.0.template.0.metadata.0.labels.app", value: "database"},
		} {
			if got := states[0].Attributes[expected.path]; got != expected.value {
				return fmt.Errorf("imported %s = %q, want %q", expected.path, got, expected.value)
			}
		}
		for _, path := range []string{
			"spec.0.selector.0.match_labels.%",
			"spec.0.selector.0.match_expressions.0.values.#",
		} {
			if value, exists := states[0].Attributes[path]; exists {
				return fmt.Errorf("import invented empty collection ownership at %s: %q", path, value)
			}
		}
		return nil
	}
}

func workloadSelectorDeploymentAPICheck(snapshot *deploymentMigrationSnapshot) resource.TestCheckFunc {
	return func(*terraform.State) error {
		return workloadSelectorExistsOnly(snapshot.deployment.Spec.Selector, snapshot.deployment.Spec.Template.Labels)
	}
}

func workloadSelectorStatefulSetAPICheck(snapshot *statefulSetMigrationSnapshot) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if snapshot.object == nil {
			return fmt.Errorf("StatefulSet snapshot is missing")
		}
		return workloadSelectorExistsOnly(snapshot.object.Spec.Selector, snapshot.object.Spec.Template.Labels)
	}
}

func workloadSelectorExistsOnly(selector *metav1.LabelSelector, templateLabels map[string]string) error {
	if selector == nil {
		return fmt.Errorf("workload selector is nil")
	}
	if len(selector.MatchLabels) != 0 {
		return fmt.Errorf("workload selector matchLabels = %v, want empty", selector.MatchLabels)
	}
	if len(selector.MatchExpressions) != 1 {
		return fmt.Errorf("workload selector has %d expressions, want 1", len(selector.MatchExpressions))
	}
	expression := selector.MatchExpressions[0]
	if expression.Key != "app" || expression.Operator != metav1.LabelSelectorOpExists || len(expression.Values) != 0 {
		return fmt.Errorf("workload selector expression = %#v, want app Exists with no values", expression)
	}
	if templateLabels["app"] != "database" {
		return fmt.Errorf("pod template app label = %q, want database", templateLabels["app"])
	}
	return nil
}

type workloadSelectorDaemonSetSnapshot struct {
	id     string
	object appsv1.DaemonSet
	pods   map[string]types.UID
}

func workloadSelectorCaptureDaemonSet(address string, snapshot *workloadSelectorDaemonSetSnapshot) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		var object appsv1.DaemonSet
		if err := testAccCheckDaemonSetExists(address, &object)(state); err != nil {
			return err
		}
		if err := workloadSelectorExistsOnly(object.Spec.Selector, object.Spec.Template.Labels); err != nil {
			return err
		}
		resourceState, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %s not found in state", address)
		}
		client, err := testAccWorkloadClient()
		if err != nil {
			return err
		}
		pods, err := client.CoreV1().Pods(object.Namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		ownedPods := make(map[string]types.UID)
		for _, pod := range pods.Items {
			if metav1.IsControlledBy(&pod, &object) {
				ownedPods[pod.Name] = pod.UID
			}
		}
		if len(ownedPods) == 0 {
			return fmt.Errorf("DaemonSet %s/%s has no owned pods", object.Namespace, object.Name)
		}
		*snapshot = workloadSelectorDaemonSetSnapshot{
			id:     resourceState.Primary.ID,
			object: *object.DeepCopy(),
			pods:   ownedPods,
		}
		return nil
	}
}

func workloadSelectorCheckDaemonSetUnchanged(address string, before *workloadSelectorDaemonSetSnapshot) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		var after workloadSelectorDaemonSetSnapshot
		if err := workloadSelectorCaptureDaemonSet(address, &after)(state); err != nil {
			return err
		}
		if before.id != after.id || before.object.UID != after.object.UID ||
			before.object.Generation != after.object.Generation ||
			!before.object.CreationTimestamp.Equal(&after.object.CreationTimestamp) {
			return fmt.Errorf("DaemonSet identity or generation changed during a no-op migration")
		}
		if !reflect.DeepEqual(before.object.Spec, after.object.Spec) {
			return fmt.Errorf("DaemonSet API spec changed during a no-op migration")
		}
		if !reflect.DeepEqual(before.pods, after.pods) {
			return fmt.Errorf("DaemonSet pod identities changed: %v -> %v", before.pods, after.pods)
		}
		return nil
	}
}

func workloadSelectorBlock(explicitEmpty bool) string {
	if explicitEmpty {
		return `    selector {
      match_labels = {}

      match_expressions {
        key      = "app"
        operator = "Exists"
        values   = []
      }
    }`
	}
	return `    selector {
      match_expressions {
        key      = "app"
        operator = "Exists"
      }
    }`
}

func workloadSelectorDeploymentConfig(namespace, name string, explicitEmpty bool) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %q
  }
}

resource "kubernetes_deployment_v1" "test" {
  metadata {
    name      = %q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  wait_for_rollout = true

  spec {
    replicas = 1

%s

    template {
      metadata {
        labels = {
          app = "database"
        }
      }

      spec {
        container {
          name              = "database"
          image             = %q
          image_pull_policy = "IfNotPresent"
          command           = ["sh", "-c", "sleep 3600"]
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, namespace, name, workloadSelectorBlock(explicitEmpty), busyboxImage)
}

func workloadSelectorDaemonSetConfig(namespace, name string, explicitEmpty, waitForRollout bool) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %q
  }
}

resource "kubernetes_daemon_set_v1" "test" {
  metadata {
    name      = %q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  wait_for_rollout = %t

  spec {
%s

    template {
      metadata {
        labels = {
          app = "database"
        }
      }

      spec {
        container {
          name              = "database"
          image             = %q
          image_pull_policy = "IfNotPresent"
          command           = ["sh", "-c", "sleep 3600"]
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, namespace, name, waitForRollout, workloadSelectorBlock(explicitEmpty), busyboxImage)
}

func workloadSelectorStatefulSetConfig(namespace, name string, explicitEmpty bool) string {
	return fmt.Sprintf(`resource "kubernetes_namespace_v1" "test" {
  metadata {
    name = %q
  }
}

resource "kubernetes_stateful_set_v1" "test" {
  metadata {
    name      = %q
    namespace = kubernetes_namespace_v1.test.metadata[0].name
  }

  wait_for_rollout = true

  spec {
    replicas     = 1
    service_name = %q

%s

    template {
      metadata {
        labels = {
          app = "database"
        }
      }

      spec {
        container {
          name              = "database"
          image             = %q
          image_pull_policy = "IfNotPresent"
          command           = ["sh", "-c", "sleep 3600"]
        }

        termination_grace_period_seconds = 1
      }
    }
  }
}
`, namespace, name, name+"-headless", workloadSelectorBlock(explicitEmpty), busyboxImage)
}
