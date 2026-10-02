// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	sdkschema "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestWorkloadSelectorReplacementProtocol(t *testing.T) {
	const omitted = `{"match_expressions":[{"key":"app","operator":"Exists"}]}`
	const empty = `{"match_labels":{},"match_expressions":[{"key":"app","operator":"Exists","values":[]}]}`
	for _, workload := range []struct {
		name, resource string
		claim          bool
	}{
		{"deployment", "kubernetes_deployment_v1", false},
		{"daemon-set", "kubernetes_daemon_set_v1", false},
		{"stateful-set", "kubernetes_stateful_set_v1", false},
		{"stateful-set-claim", "kubernetes_stateful_set_v1", true},
	} {
		t.Run(workload.name, func(t *testing.T) {
			ctx := context.Background()
			server, err := testAccProviderFactories["kubernetes"]()
			if err != nil {
				t.Fatal(err)
			}
			schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			deploymentProtocolNoErrors(t, schemas.Diagnostics)
			typ := schemas.ResourceSchemas[workload.resource].ValueType()
			for _, tc := range []struct {
				name, before, after string
				replace             bool
			}{
				{"unchanged-omitted", omitted, omitted, false},
				{"unchanged-empty", empty, empty, false},
				{"legacy-empty-to-omitted", empty, omitted, false},
				{"read-null-to-explicit-empty", omitted, empty, false},
				{"empty-labels-to-null", `{"match_labels":{},"match_expressions":[{"key":"app","operator":"Exists"}]}`, omitted, false},
				{"null-labels-to-empty", omitted, `{"match_labels":{},"match_expressions":[{"key":"app","operator":"Exists"}]}`, false},
				{"empty-values-to-null", `{"match_expressions":[{"key":"app","operator":"Exists","values":[]}]}`, omitted, false},
				{"null-values-to-empty", omitted, `{"match_expressions":[{"key":"app","operator":"Exists","values":[]}]}`, false},
				{"empty-expressions-to-null", `{"match_labels":{"app":"database"},"match_expressions":[]}`, `{"match_labels":{"app":"database"}}`, false},
				{"null-expressions-to-empty", `{"match_labels":{"app":"database"}}`, `{"match_labels":{"app":"database"},"match_expressions":[]}`, false},
				{"add-label", omitted, `{"match_labels":{"app":"database"},"match_expressions":[{"key":"app","operator":"Exists"}]}`, true},
				{"remove-label", `{"match_labels":{"app":"database"},"match_expressions":[{"key":"app","operator":"Exists"}]}`, omitted, true},
				{"change-label", `{"match_labels":{"app":"old"}}`, `{"match_labels":{"app":"database"}}`, true},
				{"add-empty-string-label", omitted, `{"match_labels":{"app":""},"match_expressions":[{"key":"app","operator":"Exists"}]}`, true},
				{"add-expression", `{"match_labels":{"app":"database"}}`, `{"match_labels":{"app":"database"},"match_expressions":[{"key":"app","operator":"Exists"}]}`, true},
				{"remove-expression", `{"match_labels":{"app":"database"},"match_expressions":[{"key":"app","operator":"Exists"}]}`, `{"match_labels":{"app":"database"}}`, true},
				{"change-key", omitted, `{"match_expressions":[{"key":"other","operator":"Exists"}]}`, true},
				{"change-operator", omitted, `{"match_expressions":[{"key":"app","operator":"DoesNotExist"}]}`, true},
				{"change-values", `{"match_expressions":[{"key":"app","operator":"In","values":["old"]}]}`, `{"match_expressions":[{"key":"app","operator":"In","values":["database"]}]}`, true},
				{"empty-string-set-member", `{"match_expressions":[{"key":"app","operator":"In","values":["database"]}]}`, `{"match_expressions":[{"key":"app","operator":"In","values":[""]}]}`, true},
				{"set-order", `{"match_expressions":[{"key":"app","operator":"In","values":["a","b"]}]}`, `{"match_expressions":[{"key":"app","operator":"In","values":["b","a"]}]}`, false},
				{"expression-order", `{"match_expressions":[{"key":"app","operator":"Exists"},{"key":"other","operator":"DoesNotExist"}]}`, `{"match_expressions":[{"key":"other","operator":"DoesNotExist"},{"key":"app","operator":"Exists"}]}`, true},
				{"mixed-normalization-and-change", empty, `{"match_expressions":[{"key":"other","operator":"Exists"}]}`, true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					prior := workloadSelectorProtocolValue(t, typ, workloadSelectorFixture(t, workload.resource, tc.before, true, workload.claim))
					configMap := workloadSelectorFixture(t, workload.resource, tc.after, false, workload.claim)
					// HCL represents omitted nested blocks as empty lists, not null attributes.
					deploymentProtocolPersistedBlocks(schemas.ResourceSchemas[workload.resource].Block, configMap)
					config := workloadSelectorProtocolValue(t, typ, configMap)
					validated, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
						TypeName: workload.resource, Config: &config,
					})
					if err != nil {
						t.Fatal(err)
					}
					deploymentProtocolNoErrors(t, validated.Diagnostics)
					// Exercise the replacement decision without any preceding ReadResource.
					response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
						TypeName: workload.resource, PriorState: &prior, Config: &config, ProposedNewState: &config,
					})
					if err != nil {
						t.Fatal(err)
					}
					deploymentProtocolNoErrors(t, response.Diagnostics)
					if got := len(response.RequiresReplace) != 0; got != tc.replace {
						t.Errorf("replacement = %t, want %t: %v", got, tc.replace, response.RequiresReplace)
					}
					at := workloadSelectorPath(workload.claim)
					planned, err := response.PlannedState.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					configured, err := config.Unmarshal(typ)
					if err != nil {
						t.Fatal(err)
					}
					planSelector, _, err := tftypes.WalkAttributePath(planned, at)
					if err != nil {
						t.Fatal(err)
					}
					configSelector, _, err := tftypes.WalkAttributePath(configured, at)
					if err != nil {
						t.Fatal(err)
					}
					if !planSelector.(tftypes.Value).Equal(configSelector.(tftypes.Value)) {
						t.Error("replacement comparison rewrote the configured selector")
					}
				})
			}
		})
	}
}

func TestWorkloadSelectorSDKStateProtocol(t *testing.T) {
	for target, alias := range map[string]string{
		"kubernetes_deployment_v1":   "kubernetes_deployment",
		"kubernetes_daemon_set_v1":   "kubernetes_daemonset",
		"kubernetes_stateful_set_v1": "kubernetes_stateful_set",
	} {
		t.Run(target, func(t *testing.T) {
			const selector = `{"match_expressions":[{"key":"app","operator":"Exists"}]}`
			ctx := context.Background()
			legacy := kubernetes.Provider().ResourcesMap[alias]
			input := workloadSelectorFixture(t, target, selector, false, false)
			data := sdkschema.TestResourceDataRaw(t, legacy.Schema, input)
			sdkSelector := data.Get("spec.0.selector").([]any)[0].(map[string]any)
			labels := sdkSelector["match_labels"].(map[string]any)
			values := sdkSelector["match_expressions"].([]any)[0].(map[string]any)["values"].(*sdkschema.Set).List()
			if len(labels) != 0 || len(values) != 0 {
				t.Fatalf("unexpected SDK representation: labels=%v values=%v", labels, values)
			}
			priorMap := workloadSelectorFixture(t, target, selector, true, false)
			storedSelector := deploymentProtocolSpec(priorMap)["selector"].([]any)[0].(map[string]any)
			storedSelector["match_labels"] = labels
			storedSelector["match_expressions"].([]any)[0].(map[string]any)["values"] = values
			raw, err := json.Marshal(priorMap)
			if err != nil {
				t.Fatal(err)
			}
			server, err := testAccProviderFactories["kubernetes"]()
			if err != nil {
				t.Fatal(err)
			}
			schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
			if err != nil {
				t.Fatal(err)
			}
			typ := schemas.ResourceSchemas[target].ValueType()
			config := workloadSelectorProtocolValue(t, typ, input)
			for _, moved := range []bool{false, true} {
				t.Run(fmt.Sprintf("moved-%t", moved), func(t *testing.T) {
					prior := statefulSetProtocolValue(t, typ, string(raw))
					if moved {
						response, err := server.MoveResourceState(ctx, &tfprotov6.MoveResourceStateRequest{
							SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
							SourceTypeName:        alias, SourceSchemaVersion: int64(legacy.SchemaVersion),
							SourceState: &tfprotov6.RawState{JSON: raw}, TargetTypeName: target,
						})
						if err != nil {
							t.Fatal(err)
						}
						deploymentProtocolNoErrors(t, response.Diagnostics)
						if response.TargetState == nil {
							t.Fatal("move returned no state")
						}
						prior = *response.TargetState
					}
					response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
						TypeName: target, PriorState: &prior, Config: &config, ProposedNewState: &config,
					})
					if err != nil {
						t.Fatal(err)
					}
					deploymentProtocolNoErrors(t, response.Diagnostics)
					if len(response.RequiresReplace) != 0 {
						t.Fatalf("SDK empty selector fields require replacement: %v", response.RequiresReplace)
					}
				})
			}
		})
	}
}

func workloadSelectorProtocolValue(t *testing.T, typ tftypes.Type, value any) tfprotov6.DynamicValue {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return statefulSetProtocolValue(t, typ, string(raw))
}

func workloadSelectorFixture(t *testing.T, resource, selector string, state, claim bool) map[string]any {
	t.Helper()
	var selected map[string]any
	if err := json.Unmarshal([]byte(selector), &selected); err != nil {
		t.Fatal(err)
	}
	podLabels := map[string]any{"app": "database"}
	if !claim {
		podLabels = workloadSelectorMatchingLabels(t, selector)
	}
	spec := map[string]any{
		"selector": []any{selected},
		"template": []any{map[string]any{
			"metadata": []any{map[string]any{"labels": podLabels}},
			"spec":     []any{map[string]any{"container": []any{map[string]any{"name": "main", "image": "busybox:1.36"}}}},
		}},
	}
	if resource == "kubernetes_stateful_set_v1" {
		spec["service_name"] = "database"
	}
	if claim {
		spec["selector"] = []any{map[string]any{"match_labels": map[string]any{"app": "database"}}}
		spec["volume_claim_template"] = []any{map[string]any{
			"metadata": []any{map[string]any{"name": "data", "namespace": "default"}},
			"spec": []any{map[string]any{
				"access_modes": []any{"ReadWriteOnce"}, "storage_class_name": "standard",
				"resources": []any{map[string]any{"requests": map[string]any{"storage": "1Gi"}}},
				"selector":  []any{selected},
			}},
		}}
	}
	result := map[string]any{
		"metadata": []any{map[string]any{"name": "selector-regression", "namespace": "default"}},
		"spec":     []any{spec},
	}
	if state {
		result["id"] = "default/selector-regression"
	}
	return result
}

func workloadSelectorMatchingLabels(t *testing.T, raw string) map[string]any {
	t.Helper()
	var selected struct {
		MatchLabels      map[string]string                 `json:"match_labels"`
		MatchExpressions []metav1.LabelSelectorRequirement `json:"match_expressions"`
	}
	if err := json.Unmarshal([]byte(raw), &selected); err != nil {
		t.Fatal(err)
	}
	podLabels := map[string]string{"app": "database"}
	for key, value := range selected.MatchLabels {
		podLabels[key] = value
	}
	for _, expression := range selected.MatchExpressions {
		switch expression.Operator {
		case metav1.LabelSelectorOpExists:
			if _, present := podLabels[expression.Key]; !present {
				podLabels[expression.Key] = "database"
			}
		case metav1.LabelSelectorOpDoesNotExist:
			delete(podLabels, expression.Key)
		case metav1.LabelSelectorOpIn:
			if len(expression.Values) == 0 {
				t.Fatal("In selector requires a value")
			}
			value := expression.Values[0]
			for _, candidate := range expression.Values[1:] {
				if candidate < value {
					value = candidate
				}
			}
			podLabels[expression.Key] = value
		default:
			t.Fatalf("unsupported fixture operator: %s", expression.Operator)
		}
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: selected.MatchLabels, MatchExpressions: selected.MatchExpressions,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !selector.Matches(labels.Set(podLabels)) {
		t.Fatalf("fixture labels %v do not match selector %s", podLabels, selector)
	}
	result := make(map[string]any, len(podLabels))
	for key, value := range podLabels {
		result[key] = value
	}
	return result
}

func workloadSelectorPath(claim bool) *tftypes.AttributePath {
	at := tftypes.NewAttributePath().WithAttributeName("spec").WithElementKeyInt(0)
	if claim {
		at = at.WithAttributeName("volume_claim_template").WithElementKeyInt(0).WithAttributeName("spec").WithElementKeyInt(0)
	}
	return at.WithAttributeName("selector")
}
