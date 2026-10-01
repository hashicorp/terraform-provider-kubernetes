// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

type deploymentRefreshNormalizationPlanCheck struct{}

type deploymentWaitOnlyPlanCheck struct {
	before bool
	after  bool
}

func (check deploymentWaitOnlyPlanCheck) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	const address = "kubernetes_deployment_v1.test"
	plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate).CheckPlan(ctx, req, resp)
	if resp.Error != nil {
		return
	}
	for _, change := range req.Plan.ResourceChanges {
		if change.Address != address {
			if !change.Change.Actions.NoOp() {
				resp.Error = fmt.Errorf("unexpected action for %s: %v", change.Address, change.Change.Actions)
				return
			}
			continue
		}
		expected := map[string]bool{"wait_for_rollout": false}
		matches := func(_ string, before, after any) bool {
			return before == check.before && after == check.after
		}
		if err := deploymentCheckNormalization("", change.Change.Before, change.Change.After, change.Change.AfterUnknown, expected, matches, true); err != nil {
			resp.Error = err
			return
		}
		if !expected["wait_for_rollout"] {
			resp.Error = fmt.Errorf("expected only wait_for_rollout to change from %t to %t", check.before, check.after)
			return
		}
	}
}

func (deploymentRefreshNormalizationPlanCheck) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	for _, change := range req.Plan.ResourceDrift {
		if change.Change.Actions.NoOp() {
			continue
		}
		if change.Address != "kubernetes_deployment_v1.test" ||
			!reflect.DeepEqual(change.Change.Actions, tfjson.Actions{tfjson.ActionUpdate}) ||
			change.Change.Before == nil || change.Change.After == nil {
			resp.Error = fmt.Errorf("%s: unexpected refresh drift: %v", change.Address, change.Change.Actions)
			return
		}
		allowed := map[string]bool{
			"metadata.0.generate_name":                   false,
			"metadata.0.resource_version":                false,
			"spec.0.template.0.metadata.0.generate_name": false,
			"spec.0.template.0.metadata.0.namespace":     false,
		}
		matches := func(path string, before, after any) bool {
			previous, ok := before.(string)
			if !ok {
				return false
			}
			if path == "metadata.0.resource_version" {
				current, ok := after.(string)
				return ok && previous != "" && current != ""
			}
			return previous == "" && after == nil
		}
		if err := deploymentCheckNormalization("", change.Change.Before, change.Change.After, change.Change.AfterUnknown, allowed, matches, false); err != nil {
			resp.Error = err
			return
		}
	}
}

func TestDeploymentV1RefreshNormalizationGuard(t *testing.T) {
	metadata := func(name string, value any) any {
		return map[string]any{"metadata": []any{map[string]any{
			"uid": "unchanged-uid", "generation": 1, name: value,
		}}}
	}
	templateMetadata := func(name string, value any) any {
		return map[string]any{"spec": []any{map[string]any{
			"template": []any{metadata(name, value)},
		}}}
	}
	for _, tc := range []struct {
		name    string
		actions tfjson.Actions
		before  any
		after   any
		unknown any
		wantErr bool
	}{
		{name: "no-op", actions: tfjson.Actions{tfjson.ActionNoop}},
		{name: "refresh-update", actions: tfjson.Actions{tfjson.ActionUpdate}, wantErr: true},
		{name: "remote-deletion", actions: tfjson.Actions{tfjson.ActionDelete}, wantErr: true},
		{name: "empty-generate-name", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: metadata("generate_name", ""), after: metadata("generate_name", nil)},
		{name: "empty-template-generate-name", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: templateMetadata("generate_name", ""), after: templateMetadata("generate_name", nil)},
		{name: "empty-template-namespace", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: templateMetadata("namespace", ""), after: templateMetadata("namespace", nil)},
		{name: "resource-version", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: metadata("resource_version", "1"), after: metadata("resource_version", "2")},
		{name: "nonempty-generate-name", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: metadata("generate_name", "prefix-"), after: metadata("generate_name", nil), wantErr: true},
		{name: "generation", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: metadata("generation", 1), after: metadata("generation", 2), wantErr: true},
		{name: "uid", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: metadata("uid", "old"), after: metadata("uid", "new"), wantErr: true},
		{name: "actual-template-namespace", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: templateMetadata("namespace", "default"), after: templateMetadata("namespace", nil), wantErr: true},
		{name: "unknown-generation", actions: tfjson.Actions{tfjson.ActionUpdate},
			before: metadata("generate_name", ""), after: metadata("generate_name", nil),
			unknown: map[string]any{"metadata": []any{map[string]any{"generation": true}}}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var response plancheck.CheckPlanResponse
			deploymentRefreshNormalizationPlanCheck{}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{
				Plan: &tfjson.Plan{ResourceDrift: []*tfjson.ResourceChange{{
					Address: "kubernetes_deployment_v1.test",
					Change:  &tfjson.Change{Actions: tc.actions, Before: tc.before, After: tc.after, AfterUnknown: tc.unknown},
				}}},
			}, &response)
			if (response.Error != nil) != tc.wantErr {
				t.Fatalf("unexpected drift check result: %v", response.Error)
			}
		})
	}
}

// SDKv2 stores explicitly configured empty metadata/node-selector maps as null.
// Core requires Framework to plan the configured {}, not the null prior value.
// Permit precisely that state representation change, never an API/spec change.
type deploymentEmptyMapsNormalizationPlanCheck struct{}

func (deploymentEmptyMapsNormalizationPlanCheck) CheckPlan(ctx context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	const address = "kubernetes_deployment_v1.test"
	plancheck.ExpectResourceAction(address, plancheck.ResourceActionUpdate).CheckPlan(ctx, req, resp)
	if resp.Error != nil {
		return
	}
	for _, change := range req.Plan.ResourceChanges {
		if change.Address != address {
			if !change.Change.Actions.NoOp() {
				resp.Error = fmt.Errorf("unexpected action for %s: %v", change.Address, change.Change.Actions)
				return
			}
			continue
		}
		expected := map[string]bool{
			"metadata.0.annotations":                   false,
			"spec.0.template.0.metadata.0.annotations": false,
			"spec.0.template.0.spec.0.node_selector":   false,
		}
		matches := func(_ string, before, after any) bool {
			value, ok := after.(map[string]any)
			return before == nil && ok && len(value) == 0
		}
		if err := deploymentCheckNormalization("", change.Change.Before, change.Change.After, change.Change.AfterUnknown, expected, matches, true); err != nil {
			resp.Error = err
			return
		}
		for path, found := range expected {
			if !found {
				resp.Error = fmt.Errorf("expected explicit empty-map normalization at %s", path)
				return
			}
		}
	}
}

func deploymentCheckNormalization(path string, before, after, unknown any, expected map[string]bool, matches func(string, any, any) bool, allowComputedUnknowns bool) error {
	if isUnknown, _ := unknown.(bool); isUnknown {
		if allowComputedUnknowns {
			switch path {
			case "metadata.0.generation", "metadata.0.resource_version",
				"spec.0.template.0.metadata.0.generation", "spec.0.template.0.metadata.0.name",
				"spec.0.template.0.metadata.0.resource_version", "spec.0.template.0.metadata.0.uid":
				return nil
			}
		}
		return fmt.Errorf("unexpected unknown value at %s", path)
	}
	if reflect.DeepEqual(before, after) && unknown == nil {
		return nil
	}
	if _, allowed := expected[path]; allowed && matches(path, before, after) {
		expected[path] = true
		return nil
	}
	switch before := before.(type) {
	case map[string]any:
		after, ok := after.(map[string]any)
		if !ok {
			return fmt.Errorf("unexpected object change at %s", path)
		}
		unknown, _ := unknown.(map[string]any)
		keys := map[string]bool{}
		for key := range before {
			keys[key] = true
		}
		for key := range after {
			keys[key] = true
		}
		for key := range unknown {
			keys[key] = true
		}
		for key := range keys {
			child := key
			if path != "" {
				child = path + "." + key
			}
			if err := deploymentCheckNormalization(child, before[key], after[key], unknown[key], expected, matches, allowComputedUnknowns); err != nil {
				return err
			}
		}
		return nil
	case []any:
		after, ok := after.([]any)
		if !ok || len(before) != len(after) {
			return fmt.Errorf("unexpected collection change at %s", path)
		}
		unknown, _ := unknown.([]any)
		for i, value := range before {
			var childUnknown any
			if i < len(unknown) {
				childUnknown = unknown[i]
			}
			if err := deploymentCheckNormalization(fmt.Sprintf("%s.%d", path, i), value, after[i], childUnknown, expected, matches, allowComputedUnknowns); err != nil {
				return err
			}
		}
		return nil
	}
	if reflect.DeepEqual(before, after) {
		return nil
	}
	return fmt.Errorf("unexpected value change at %s: %#v -> %#v", path, before, after)
}

func TestDeploymentV1EmptyMapNormalizationGuard(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mutate  func(*tfjson.ResourceChange)
		wantErr bool
	}{
		{name: "precise-normalization"},
		{name: "computed-metadata", mutate: func(change *tfjson.ResourceChange) {
			before := change.Change.Before.(map[string]any)["metadata"].([]any)[0].(map[string]any)
			before["generation"] = 1
			change.Change.AfterUnknown = map[string]any{"metadata": []any{map[string]any{"generation": true}}}
		}},
		{name: "replacement", mutate: func(change *tfjson.ResourceChange) {
			change.Change.Actions = tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
		}, wantErr: true},
		{name: "identity-change", mutate: func(change *tfjson.ResourceChange) {
			change.Change.After.(map[string]any)["id"] = "default/recreated"
		}, wantErr: true},
		{name: "unknown-identity", mutate: func(change *tfjson.ResourceChange) {
			change.Change.AfterUnknown = map[string]any{"id": true}
		}, wantErr: true},
		{name: "missing-normalization", mutate: func(change *tfjson.ResourceChange) {
			change.Change.After.(map[string]any)["metadata"].([]any)[0].(map[string]any)["annotations"] = nil
		}, wantErr: true},
		{name: "nonempty-annotation", mutate: func(change *tfjson.ResourceChange) {
			change.Change.After.(map[string]any)["metadata"].([]any)[0].(map[string]any)["annotations"] = map[string]any{"unexpected": "change"}
		}, wantErr: true},
		{name: "spec-change", mutate: func(change *tfjson.ResourceChange) {
			change.Change.After.(map[string]any)["spec"].([]any)[0].(map[string]any)["replicas"] = "2"
		}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			beforeJSON := []byte(`{
				"id":"default/existing",
				"metadata":[{"annotations":null,"uid":"existing-uid"}],
				"spec":[{"replicas":"1","template":[{
					"metadata":[{"annotations":null}],
					"spec":[{"node_selector":null,"container":[{"name":"main","image":"pause","args":[],"command":[]}]}]
				}]}]
			}`)
			var before, after map[string]any
			if err := json.Unmarshal(beforeJSON, &before); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(beforeJSON, &after); err != nil {
				t.Fatal(err)
			}
			after["metadata"].([]any)[0].(map[string]any)["annotations"] = map[string]any{}
			template := after["spec"].([]any)[0].(map[string]any)["template"].([]any)[0].(map[string]any)
			template["metadata"].([]any)[0].(map[string]any)["annotations"] = map[string]any{}
			template["spec"].([]any)[0].(map[string]any)["node_selector"] = map[string]any{}
			change := &tfjson.ResourceChange{
				Address: "kubernetes_deployment_v1.test",
				Change:  &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}, Before: before, After: after},
			}
			if tc.mutate != nil {
				tc.mutate(change)
			}
			response := plancheck.CheckPlanResponse{}
			deploymentEmptyMapsNormalizationPlanCheck{}.CheckPlan(context.Background(), plancheck.CheckPlanRequest{
				Plan: &tfjson.Plan{ResourceChanges: []*tfjson.ResourceChange{change}},
			}, &response)
			if (response.Error != nil) != tc.wantErr {
				t.Fatalf("error = %v, want error = %t", response.Error, tc.wantErr)
			}
		})
	}
}
