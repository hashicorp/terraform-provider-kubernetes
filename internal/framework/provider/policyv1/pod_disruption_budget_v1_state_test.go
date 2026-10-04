// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package policyv1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	policy "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func pdbLegacyJSON(selector string) []byte {
	return []byte(fmt.Sprintf(`{
  "id":"test/budget",
  "metadata":[{"name":"budget","namespace":"test","generate_name":"","annotations":null,"labels":null,"generation":1,"resource_version":"1","uid":"uid"}],
  "spec":[{"min_available":"1","max_unavailable":"","selector":%s}]
}`, selector))
}

func pdbUpgradeForTest(t *testing.T, raw []byte) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	server := providerserver.NewProtocol6(pdbProtocolProvider{})()
	resp, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "kubernetes_pod_disruption_budget_v1", Version: 0, RawState: &tfprotov6.RawState{JSON: raw},
	})
	if err != nil {
		t.Fatal(err)
	}
	pdbProtocolNoErrors(t, resp.Diagnostics)
	s := pdbTestSchema(t)
	value, err := resp.UpgradedState.Unmarshal(s.Type().TerraformType(ctx))
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: s, Raw: value}
}

func TestPDBSelectorStateConversion(t *testing.T) {
	for _, selector := range []string{
		`[{}]`, `{}`, `[{"match_labels":null,"match_expressions":[]}]`,
		`{"match_labels":{},"match_expressions":null}`,
		`[{"match_labels":{"app":"test"},"match_expressions":[{"key":"tier","operator":"In","values":["b","a"]}]}]`,
	} {
		t.Run(selector, func(t *testing.T) {
			raw := pdbLegacyJSON(selector)
			converted, err := upgradePDBSelectorState(raw)
			if err != nil {
				t.Fatal(err)
			}
			again, err := upgradePDBSelectorState(converted)
			if err != nil || string(again) != string(converted) {
				t.Fatalf("conversion is not idempotent: %s, %v", again, err)
			}
			var before, after map[string]any
			if err := json.Unmarshal(raw, &before); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(converted, &after); err != nil {
				t.Fatal(err)
			}
			spec := before["spec"].([]any)[0].(map[string]any)
			if list, ok := spec["selector"].([]any); ok {
				spec["selector"] = list[0]
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("converter changed values outside selector shape: %#v => %#v", before, after)
			}
			upgraded := pdbUpgradeForTest(t, raw)
			current := pdbUpgradeForTest(t, converted)
			if !upgraded.Raw.Equal(current.Raw) {
				t.Fatal("legacy and already-converted state decoded differently")
			}
		})
	}
}

func TestPDBSelectorStateRejectsMalformed(t *testing.T) {
	fixtures := []string{`null`, `[]`, `{}`, `{"spec":null}`, `{"spec":[]}`, `{"spec":[{},{}]}`, `{"spec":[null]}`, `{"spec":[{}]}`, `{"spec":{}}`}
	for _, selector := range []string{`null`, `[]`, `[null]`, `[{},{}]`, `[1]`, `1`, `"x"`, `{"match_expressions":{}}`, `{"match_labels":[]}`} {
		fixtures = append(fixtures, string(pdbLegacyJSON(selector)))
	}
	for i, fixture := range fixtures {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			server := providerserver.NewProtocol6(pdbProtocolProvider{})()
			resp, err := server.UpgradeResourceState(context.Background(), &tfprotov6.UpgradeResourceStateRequest{
				TypeName: "kubernetes_pod_disruption_budget_v1", Version: 0, RawState: &tfprotov6.RawState{JSON: []byte(fixture)},
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range resp.Diagnostics {
				if d.Severity == tfprotov6.DiagnosticSeverityError {
					return
				}
			}
			t.Fatalf("malformed source accepted: %s", fixture)
		})
	}
}

// Upgrade and plan are separate RPCs. Deliberately omit ReadResource so refresh
// cannot hide a legacy state replacement, including metadata child modifiers.
func TestPDBUpgradePlanWithoutRead(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		selector string
		mutate   func(*podDisruptionBudgetV1Model)
		replace  bool
	}{
		{"empty-selector", `[{}]`, func(m *podDisruptionBudgetV1Model) {}, false},
		{"omitted-expressions", `[{"match_labels":null,"match_expressions":[]}]`, func(m *podDisruptionBudgetV1Model) {}, false},
		{"sdk-noop-empty-maps", `[{"match_labels":{},"match_expressions":[]}]`, func(m *podDisruptionBudgetV1Model) {}, false},
		{"explicit-empty-labels", `[{"match_labels":null,"match_expressions":[]}]`, func(m *podDisruptionBudgetV1Model) {
			m.Spec[0].Selector.MatchLabels = types.MapValueMust(types.StringType, map[string]attr.Value{})
		}, false},
		{"explicit-empty-expressions", `[{"match_labels":null,"match_expressions":[]}]`, func(m *podDisruptionBudgetV1Model) {
			m.Spec[0].Selector.MatchExpressions = []pdbLabelRequirementModel{}
		}, false},
		{"explicit-empty-values", `[{"match_labels":null,"match_expressions":[{"key":"tier","operator":"Exists","values":null}]}]`, func(m *podDisruptionBudgetV1Model) {
			m.Spec[0].Selector.MatchExpressions = []pdbLabelRequirementModel{{Key: types.StringValue("tier"), Operator: types.StringValue("Exists"), Values: types.SetValueMust(types.StringType, []attr.Value{})}}
		}, false},
		{"omitted-values", `[{"match_labels":{},"match_expressions":[{"key":"tier","operator":"Exists","values":[]}]}]`, func(m *podDisruptionBudgetV1Model) {
			m.Spec[0].Selector.MatchExpressions = []pdbLabelRequirementModel{{Key: types.StringValue("tier"), Operator: types.StringValue("Exists"), Values: types.SetNull(types.StringType)}}
		}, false},
		{"reordered-values", `[{"match_labels":null,"match_expressions":[{"key":"tier","operator":"In","values":["b","a"]}]}]`, func(m *podDisruptionBudgetV1Model) {
			m.Spec[0].Selector.MatchExpressions = []pdbLabelRequirementModel{{Key: types.StringValue("tier"), Operator: types.StringValue("In"), Values: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("a"), types.StringValue("b")})}}
		}, false},
		{"real-threshold-edit", `[{}]`, func(m *podDisruptionBudgetV1Model) { m.Spec[0].MinAvailable = types.StringValue("2") }, true},
		{"real-selector-edit", `[{}]`, func(m *podDisruptionBudgetV1Model) {
			m.Spec[0].Selector.MatchLabels = types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("changed")})
		}, true},
		{"real-metadata-name-edit", `[{}]`, func(m *podDisruptionBudgetV1Model) { m.Metadata[0].Name = types.StringValue("changed") }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := pdbLegacyJSON(tc.selector)
			if tc.name == "sdk-noop-empty-maps" {
				// A released SDK no-op apply can materialize omitted maps as {}.
				raw = []byte(strings.ReplaceAll(strings.ReplaceAll(string(raw), `"annotations":null`, `"annotations":{}`), `"labels":null`, `"labels":{}`))
			}
			prior := pdbUpgradeForTest(t, raw)
			model := pdbTestModel()
			tc.mutate(&model)
			proposed := pdbTestState(t, model)
			config := pdbTestState(t, pdbTestConfig(model))
			server := providerserver.NewProtocol6(pdbProtocolProvider{})()
			resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_pod_disruption_budget_v1", PriorState: pdbTestDynamic(t, prior),
				ProposedNewState: pdbTestDynamic(t, proposed), Config: pdbTestDynamic(t, config),
			})
			if err != nil {
				t.Fatal(err)
			}
			pdbProtocolNoErrors(t, resp.Diagnostics)
			if got := len(resp.RequiresReplace) != 0; got != tc.replace {
				t.Fatalf("replacement=%v, want %t", resp.RequiresReplace, tc.replace)
			}
		})
	}
}

func TestPDBUnknownSelectorPlan(t *testing.T) {
	ctx := context.Background()
	prior := pdbUpgradeForTest(t, pdbLegacyJSON(`[{"match_labels":null,"match_expressions":[]}]`))
	for _, p := range []path.Path{
		path.Root("spec").AtListIndex(0).AtName("selector"),
		path.Root("spec").AtListIndex(0).AtName("selector").AtName("match_expressions"),
	} {
		t.Run(p.String(), func(t *testing.T) {
			plan := pdbTestState(t, pdbTestModel())
			config := pdbTestState(t, pdbTestConfig(pdbTestModel()))
			typ, d := plan.Schema.TypeAtPath(ctx, p)
			pdbNoErrors(t, d)
			var unknown attr.Value
			switch typ := typ.(type) {
			case types.ObjectType:
				unknown = types.ObjectUnknown(typ.AttrTypes)
			case types.ListType:
				unknown = types.ListUnknown(typ.ElemType)
			default:
				t.Fatalf("unexpected type %T", typ)
			}
			pdbNoErrors(t, plan.SetAttribute(ctx, p, unknown))
			pdbNoErrors(t, config.SetAttribute(ctx, p, unknown))
			server := providerserver.NewProtocol6(pdbProtocolProvider{})()
			response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_pod_disruption_budget_v1", PriorState: pdbTestDynamic(t, prior),
				Config: pdbTestDynamic(t, config), ProposedNewState: pdbTestDynamic(t, plan),
			})
			if err != nil {
				t.Fatal(err)
			}
			pdbProtocolNoErrors(t, response.Diagnostics)
			if len(response.RequiresReplace) == 0 {
				t.Fatal("an unknown spec edit must conservatively require replacement")
			}
		})
	}
}

func TestPDBUpgradeRefreshAndConvergence(t *testing.T) {
	ctx := context.Background()
	for _, expressions := range []string{`null`, `[]`} {
		t.Run(expressions, func(t *testing.T) {
			// Keep generate_name null to compare selector conversion with Read;
			// its independent SDK empty-string normalization is tested without Read.
			raw := strings.ReplaceAll(string(pdbLegacyJSON(fmt.Sprintf(`[{"match_labels":null,"match_expressions":%s}]`, expressions))), `"generate_name":""`, `"generate_name":null`)
			prior := pdbUpgradeForTest(t, []byte(raw))
			one := intstr.FromInt32(1)
			remote := policy.PodDisruptionBudget{
				ObjectMeta: metav1.ObjectMeta{Name: "budget", Namespace: "test", UID: "uid", Generation: 1, ResourceVersion: "1"},
				Spec:       policy.PodDisruptionBudgetSpec{MinAvailable: &one, Selector: &metav1.LabelSelector{}},
			}
			requests := 0
			r := pdbTestResource(t, func(w http.ResponseWriter, req *http.Request) {
				requests++
				if req.Method != http.MethodGet {
					t.Errorf("state normalization sent %s", req.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(remote); err != nil {
					t.Error(err)
				}
			})
			read := resource.ReadResponse{State: prior}
			r.Read(ctx, resource.ReadRequest{State: prior}, &read)
			pdbNoErrors(t, read.Diagnostics)
			if !read.State.Raw.Equal(prior.Raw) {
				t.Fatalf("Read changed selector state:\nbefore: %s\nafter: %s", prior.Raw, read.State.Raw)
			}
			// Switching [] to omitted expressions is state-only and converges.
			model := pdbTestModel()
			plan := pdbTestState(t, model)
			update := resource.UpdateResponse{State: prior}
			r.Update(ctx, resource.UpdateRequest{State: prior, Plan: tfsdk.Plan{Schema: plan.Schema, Raw: plan.Raw}}, &update)
			pdbNoErrors(t, update.Diagnostics)
			if !update.State.Raw.Equal(plan.Raw) {
				t.Fatal("state-only update did not preserve planned null expressions and identity")
			}
			next := pdbTestProtocolPlan(t, model, model)
			nextRaw, err := next.PlannedState.Unmarshal(plan.Schema.Type().TerraformType(ctx))
			if err != nil || !nextRaw.Equal(update.State.Raw) || len(next.RequiresReplace) != 0 {
				t.Fatalf("normalization did not converge: %v, %v", err, next.RequiresReplace)
			}
			if requests != 2 {
				t.Fatalf("expected Read and state-only Update GETs, got %d", requests)
			}
		})
	}
}
