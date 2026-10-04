// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Every accepted historical route must return the current shape directly;
// CronJob v1 schema 0 already had quantity maps, unlike the beta alias.
func TestBatchSingletonStateRoutes(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name             string
		resource         resource.Resource
		version          int64
		source           string
		legacyQuantities bool
	}{
		{"job upgrade v0", &JobV1{}, 0, "", true},
		{"job upgrade v1", &JobV1{}, 1, "", false},
		{"job move v0", &JobV1{}, 0, "kubernetes_job", true},
		{"job move v1", &JobV1{}, 1, "kubernetes_job", false},
		{"cronjob upgrade v0", &CronJobV1{}, 0, "", false},
		{"cronjob move v0", &CronJobV1{}, 0, "kubernetes_cron_job", true},
		{"cronjob move v1", &CronJobV1{}, 1, "kubernetes_cron_job", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var schemaResponse resource.SchemaResponse
			tc.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
			quantities := `{"limits":{"cpu":"100m"},"requests":null}`
			if tc.legacyQuantities {
				quantities = `{"limits":[{"cpu":"100m"}],"requests":[]}`
			}
			spec := fmt.Sprintf(`{"selector":[{}],"template":[{"spec":[{"container":[{"name":"c","resources":[%s]}],"init_container":[{"name":"init","resources":[]}]}]}]}`, quantities)
			at := path.Root("spec").AtListIndex(0)
			if _, cron := tc.resource.(*CronJobV1); cron {
				spec = `{"schedule":"0 0 1 1 *","job_template":[{"spec":[` + spec + `]}]}`
				at = at.AtName("job_template").AtListIndex(0).AtName("spec").AtListIndex(0)
			}
			raw := &tfprotov6.RawState{JSON: []byte(`{"id":"ns/test","metadata":[{"name":"test","namespace":"ns"}],"spec":[` + spec + `]}`)}
			state := tfsdk.State{Schema: schemaResponse.Schema}
			if tc.source == "" {
				response := resource.UpgradeStateResponse{State: state}
				upgraders := tc.resource.(resource.ResourceWithUpgradeState).UpgradeState(ctx)
				upgraders[tc.version].StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: raw}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				state = response.State
			} else {
				response := resource.MoveStateResponse{TargetState: state}
				movers := tc.resource.(resource.ResourceWithMoveState).MoveState(ctx)
				movers[0].StateMover(ctx, resource.MoveStateRequest{
					SourceTypeName: tc.source, SourceSchemaVersion: tc.version,
					SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceRawState: raw,
				}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				state = response.TargetState
			}
			var selector types.Object
			if diags := state.GetAttribute(ctx, at.AtName("selector"), &selector); diags.HasError() {
				t.Fatal(diags)
			}
			wantSelector := flattenLabelSelector(&metav1.LabelSelector{}, nil, types.ObjectType{AttrTypes: selector.AttributeTypes(ctx)}, nil)
			if !selector.Equal(wantSelector) {
				t.Fatalf("selector = %s, want %s", selector, wantSelector)
			}
			pod := at.AtName("template").AtListIndex(0).AtName("spec").AtListIndex(0)
			var cpu types.String
			if diags := state.GetAttribute(ctx, pod.AtName("container").AtListIndex(0).AtName("resources").AtName("limits").AtMapKey("cpu"), &cpu); diags.HasError() {
				t.Fatal(diags)
			}
			if cpu.ValueString() != "100m" {
				t.Fatalf("cpu = %s", cpu)
			}
			var requests types.Map
			if diags := state.GetAttribute(ctx, pod.AtName("container").AtListIndex(0).AtName("resources").AtName("requests"), &requests); diags.HasError() {
				t.Fatal(diags)
			}
			if requests.IsNull() == tc.legacyQuantities {
				t.Fatalf("requests = %s, historical quantity repair = %t", requests, tc.legacyQuantities)
			}
			var zero types.Object
			if diags := state.GetAttribute(ctx, pod.AtName("init_container").AtListIndex(0).AtName("resources"), &zero); diags.HasError() {
				t.Fatal(diags)
			}
			for _, name := range []string{"limits", "requests"} {
				if !zero.Attributes()[name].Equal(types.MapValueMust(types.StringType, map[string]attr.Value{})) {
					t.Fatalf("zero resources.%s = %s", name, zero.Attributes()[name])
				}
			}
		})
	}
}

func TestJobSelectorObjectSemantics(t *testing.T) {
	ctx := context.Background()
	typ := jobSpecObjectType(true).AttrTypes["selector"].(types.ObjectType)
	null := types.ObjectNull(typ.AttrTypes)
	empty := flattenLabelSelector(&metav1.LabelSelector{}, nil, typ, nil)
	generated := flattenLabelSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"batch.kubernetes.io/controller-uid": "uid"}}, nil, typ, jobGeneratedLabels)
	if generated.IsNull() || !generated.Equal(empty) {
		t.Fatalf("generated selector = %s, want %s", generated, empty)
	}
	if got := flattenLabelSelector(nil, empty, typ, nil); !got.IsNull() {
		t.Fatalf("nil API selector = %s", got)
	}
	if expandLabelSelector(null) != nil || expandLabelSelector(empty) == nil {
		t.Fatal("nil and empty selector payloads must retain their presence")
	}
	populated := flattenLabelSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": "job"}}, nil, typ, nil)
	raw := tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{}}, map[string]tftypes.Value{})
	for _, tc := range []struct {
		name        string
		state, plan types.Object
		replace     bool
	}{
		{"empty generated selector", generated, empty, false},
		{"empty versus nil", null, empty, false},
		{"configured label", empty, populated, true},
		{"removed label", populated, empty, true},
		{"unknown selector", empty, types.ObjectUnknown(typ.AttrTypes), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := planmodifier.ObjectResponse{PlanValue: tc.plan}
			jobSelectorRequiresReplace{}.PlanModifyObject(ctx, planmodifier.ObjectRequest{
				State: tfsdk.State{Raw: raw}, Plan: tfsdk.Plan{Raw: raw}, StateValue: tc.state, PlanValue: tc.plan,
			}, &response)
			if response.RequiresReplace != tc.replace {
				t.Fatalf("replace = %t, want %t", response.RequiresReplace, tc.replace)
			}
		})
	}
}

// A shape-only rewrite must not need Kubernetes during a no-refresh plan.
// Returning HTTP 500 makes an accidental planning read fail rather than hide it.
func TestJobV1SingletonUpgradeNoPlanningRead(t *testing.T) {
	ctx := context.Background()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected planning read", http.StatusInternalServerError)
	}))
	defer server.Close()
	client, err := k8sclient.NewForConfig(&rest.Config{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	job := &JobV1{SDKv2Meta: func() any { return planClientsets{client: client} }}
	var schemaResponse resource.SchemaResponse
	job.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
	for _, old := range []string{`[]`, `[{}]`, `[{"limits":{},"requests":{}}]`} {
		t.Run(old, func(t *testing.T) {
			legacy := `{"id":"ns/j","metadata":[{"name":"j","namespace":"ns"}],"spec":[{"selector":[],"template":[{"spec":[{"container":[{"name":"c","image":"i","resources":RESOURCES}]}]}]}]}`
			response := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			job.UpgradeState(ctx)[1].StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: []byte(strings.Replace(legacy, "RESOURCES", old, 1))}}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			plannedJSON := strings.Replace(strings.Replace(legacy, "RESOURCES", `{"limits":{},"requests":{}}`, 1), `"selector":[]`, `"selector":null`, 1)
			planned, err := (&tfprotov6.RawState{JSON: []byte(plannedJSON)}).Unmarshal(schemaResponse.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			plan := tfsdk.Plan{Schema: schemaResponse.Schema, Raw: planned}
			result := resource.ModifyPlanResponse{Plan: plan}
			job.ModifyPlan(ctx, resource.ModifyPlanRequest{State: response.State, Plan: plan, Config: tfsdk.Config(plan)}, &result)
			if result.Diagnostics.HasError() || len(result.RequiresReplace) != 0 || !result.Plan.Raw.Equal(response.State.Raw) {
				t.Fatalf("plan = %s, replace = %v, diagnostics = %v", result.Plan.Raw, result.RequiresReplace, result.Diagnostics)
			}
		})
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("planning made %d API requests", got)
	}
}

func TestBatchSelectorStateValidation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		selector  any
		wantError bool
	}{
		{"omitted", nil, false},
		{"empty list", []any{}, false},
		{"empty selector", []any{map[string]any{}}, false},
		{"object", map[string]any{"match_labels": map[string]any{"app": "job"}}, false},
		{"multiple selectors", []any{map[string]any{}, map[string]any{}}, true},
		{"null element", []any{nil}, true},
		{"scalar element", []any{"selector"}, true},
		{"scalar", "selector", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := map[string]any{"selector": tc.selector}
			err := upgradeJobSpecState(spec, "spec[0]", false)
			if (err != nil) != tc.wantError {
				t.Fatalf("conversion error = %v, want error = %t", err, tc.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), "spec[0].selector") {
				t.Fatalf("missing selector path: %v", err)
			}
			if err == nil {
				before := fmt.Sprint(spec)
				if err := upgradeJobSpecState(spec, "spec[0]", false); err != nil || before != fmt.Sprint(spec) {
					t.Fatalf("conversion is not idempotent: %v", err)
				}
				if tc.name == "empty list" && spec["selector"] != nil {
					t.Fatalf("empty selector list = %#v, want nil", spec["selector"])
				}
			}
		})
	}
}
