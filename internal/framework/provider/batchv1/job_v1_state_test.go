// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/batchv1"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/mux"
	batchapi "k8s.io/api/batch/v1"
	"k8s.io/utils/ptr"
)

func TestJobHistoricalState(t *testing.T) {
	ctx := context.Background()
	r := &batchv1.JobV1{}
	state := jobUnitState(t, r, jobUnitStateJSON)
	v0 := strings.Replace(jobUnitStateJSON, `"command":["true"]`,
		`"command":["true"],"resources":[{"limits":[{"cpu":"250m","memory":"64Mi"}],"requests":[{"cpu":"100m"}]}]`, 1)
	v0 = strings.Replace(v0, `"container":[`,
		`"init_container":[{"name":"init","image":"busybox","resources":[{"limits":[{"cpu":"50m"}],"requests":[]}]}],"container":[`, 1)
	for _, tc := range []struct {
		name    string
		version int64
		raw     string
		move    bool
	}{
		{"upgrade v0", 0, v0, false},
		{"move v0", 0, v0, true},
		{"move v1", 1, jobUnitStateJSON, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got tfsdk.State
			if tc.move {
				response := fwresource.MoveStateResponse{
					TargetState:    tfsdk.State{Schema: state.Schema},
					TargetIdentity: &tfsdk.ResourceIdentity{Schema: common.NamespacedIdentitySchema()},
				}
				r.MoveState(ctx)[0].StateMover(ctx, fwresource.MoveStateRequest{
					SourceTypeName: "kubernetes_job", SourceSchemaVersion: tc.version,
					SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
					SourceRawState:        &tfprotov6.RawState{JSON: []byte(tc.raw)},
				}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				var identity common.NamespacedResourceIdentity
				if diags := response.TargetIdentity.Get(ctx, &identity); diags.HasError() {
					t.Fatal(diags)
				}
				if identity.Name.ValueString() != "test" || identity.Namespace.ValueString() != "default" ||
					identity.Kind.ValueString() != "Job" || identity.APIVersion.ValueString() != "batch/v1" {
					t.Fatalf("unexpected moved identity: %+v", identity)
				}
				got = response.TargetState
			} else {
				response := fwresource.UpgradeStateResponse{State: tfsdk.State{Schema: state.Schema}}
				r.UpgradeState(ctx)[tc.version].StateUpgrader(ctx, fwresource.UpgradeStateRequest{
					RawState: &tfprotov6.RawState{JSON: []byte(tc.raw)},
				}, &response)
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				got = response.State
			}
			model := jobUnitModel(t, got)
			if model.ID.ValueString() != "default/test" || model.Metadata[0].UID.ValueString() != "uid-1" {
				t.Fatalf("lost object identity: %+v", model)
			}
			if tc.version == 0 {
				var limits map[string]string
				diags := got.GetAttribute(ctx, path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).
					AtName("spec").AtListIndex(0).AtName("container").AtListIndex(0).
					AtName("resources").AtListIndex(0).AtName("limits"), &limits)
				if diags.HasError() || limits["cpu"] != "250m" || limits["memory"] != "64Mi" {
					t.Fatalf("v0 resource limits not upgraded: %#v, %v", limits, diags)
				}
				var initLimits map[string]string
				diags = got.GetAttribute(ctx, path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).
					AtName("spec").AtListIndex(0).AtName("init_container").AtListIndex(0).
					AtName("resources").AtListIndex(0).AtName("limits"), &initLimits)
				if diags.HasError() || initLimits["cpu"] != "50m" {
					t.Fatalf("v0 init-container limits not upgraded: %#v, %v", initLimits, diags)
				}
			}
		})
	}
}

func TestJobMoveStateGuards(t *testing.T) {
	ctx := context.Background()
	r := &batchv1.JobV1{}
	state := jobUnitState(t, r, jobUnitStateJSON)
	for _, tc := range []struct {
		name, source, address, raw string
		version                    int64
		wantError                  bool
	}{
		{"wrong type", "kubernetes_cron_job", "registry.terraform.io/hashicorp/kubernetes", jobUnitStateJSON, 1, false},
		{"wrong namespace", "kubernetes_job", "registry.terraform.io/nothashicorp/kubernetes", jobUnitStateJSON, 1, false},
		{"future schema", "kubernetes_job", "registry.terraform.io/hashicorp/kubernetes", jobUnitStateJSON, 2, false},
		{"missing JSON", "kubernetes_job", "registry.terraform.io/hashicorp/kubernetes", "", 1, true},
		{"invalid JSON", "kubernetes_job", "registry.terraform.io/hashicorp/kubernetes", "{", 1, true},
		{"empty ID", "kubernetes_job", "registry.terraform.io/hashicorp/kubernetes", strings.Replace(jobUnitStateJSON, `"default/test"`, `""`, 1), 1, true},
		{"mismatched identity", "kubernetes_job", "registry.terraform.io/hashicorp/kubernetes", strings.Replace(jobUnitStateJSON, `"default/test"`, `"default/other"`, 1), 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := fwresource.MoveStateResponse{
				TargetState: tfsdk.State{Schema: state.Schema, Raw: tftypes.NewValue(state.Raw.Type(), nil)},
			}
			r.MoveState(ctx)[0].StateMover(ctx, fwresource.MoveStateRequest{
				SourceTypeName: tc.source, SourceProviderAddress: tc.address,
				SourceSchemaVersion: tc.version, SourceRawState: &tfprotov6.RawState{JSON: []byte(tc.raw)},
			}, &response)
			if response.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("diagnostics = %v, want error %t", response.Diagnostics, tc.wantError)
			}
			if !response.TargetState.Raw.IsNull() {
				t.Fatal("rejected source wrote target state")
			}
		})
	}
}

func TestJobSameVersionSDKStateProtocol(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "kubernetes_job_v1", Version: 1,
		RawState: &tfprotov6.RawState{JSON: []byte(jobUnitStateJSON)},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatal(diagnostic)
		}
	}
	if response.UpgradedState == nil {
		t.Fatal("same-version SDK state was not returned")
	}
	r := &batchv1.JobV1{}
	state := jobUnitState(t, r, jobUnitStateJSON)
	raw, err := response.UpgradedState.Unmarshal(state.Raw.Type())
	if err != nil {
		t.Fatal(err)
	}
	model := jobUnitModel(t, tfsdk.State{Schema: state.Schema, Raw: raw})
	if model.ID.ValueString() != "default/test" || model.Metadata[0].UID.ValueString() != "uid-1" ||
		!model.Metadata[0].GenerateName.IsNull() || !model.Metadata[0].Annotations.IsNull() {
		t.Fatal("same-version conversion changed the identity or optional metadata nulls")
	}
	var deadline types.Int64
	if diags := (tfsdk.State{Schema: state.Schema, Raw: raw}).GetAttribute(ctx,
		path.Root("spec").AtListIndex(0).AtName("active_deadline_seconds"), &deadline); diags.HasError() || !deadline.IsNull() {
		t.Fatalf("optional deadline null changed: %s, diagnostics %v", deadline, diags)
	}
}

func TestJobMutablePlanDoesNotReplace(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	r := &batchv1.JobV1{}
	prior := jobUnitState(t, r, jobUnitStateJSON)
	model := jobUnitModel(t, prior)
	model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{"changed": types.StringValue("yes")})
	proposed := jobUnitModelState(t, prior, model)
	model.ID = types.StringNull()
	model.Metadata[0].Generation = types.Int64Null()
	model.Metadata[0].UID = types.StringNull()
	model.Metadata[0].ResourceVersion = types.StringNull()
	config := jobUnitModelState(t, prior, model)
	dynamic := func(state tfsdk.State) *tfprotov6.DynamicValue {
		value, err := tfprotov6.NewDynamicValue(state.Raw.Type(), state.Raw)
		if err != nil {
			t.Fatal(err)
		}
		return &value
	}
	response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "kubernetes_job_v1", PriorState: dynamic(prior),
		ProposedNewState: dynamic(proposed), Config: dynamic(config),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatal(diagnostic)
		}
	}
	if len(response.RequiresReplace) != 0 {
		t.Fatalf("metadata-only edit unexpectedly replaces Job at %v", response.RequiresReplace)
	}
	for _, tc := range []struct {
		field   string
		value   attr.Value
		replace bool
	}{
		{"parallelism", types.Int64Value(0), false},
		{"max_failed_indexes", types.Int64Value(0), false},
		{"ttl_seconds_after_finished", types.StringValue("0"), false},
		{"backoff_limit", types.Int64Value(0), false},
		{"manual_selector", types.BoolValue(true), false},
		{"completions", types.Int64Value(2), true},
		{"backoff_limit_per_index", types.Int64Value(2), true},
		{"completion_mode", types.StringValue("Indexed"), true},
	} {
		t.Run(tc.field, func(t *testing.T) {
			model := jobUnitModel(t, prior)
			jobUnitSpec(&model, tc.field, tc.value)
			proposed := jobUnitModelState(t, prior, model)
			model.ID = types.StringNull()
			model.Metadata[0].Generation = types.Int64Null()
			model.Metadata[0].UID = types.StringNull()
			model.Metadata[0].ResourceVersion = types.StringNull()
			config := jobUnitModelState(t, prior, model)
			response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_job_v1", PriorState: dynamic(prior),
				ProposedNewState: dynamic(proposed), Config: dynamic(config),
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatal(diagnostic)
				}
			}
			if (len(response.RequiresReplace) > 0) != tc.replace {
				t.Fatalf("replacement paths = %v, want replacement %t", response.RequiresReplace, tc.replace)
			}
		})
	}
}

func TestJobSelectorReplacementRules(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, before, after string
		replace             bool
	}{
		{"match label", `[{"match_labels":{"team":"old"}}]`, `[{"match_labels":{"team":"new"}}]`, true},
		{"expression key", `[{"match_expressions":[{"key":"team","operator":"In","values":["one"]}]}]`,
			`[{"match_expressions":[{"key":"app","operator":"In","values":["one"]}]}]`, true},
		{"expression operator", `[{"match_expressions":[{"key":"team","operator":"In","values":["one"]}]}]`,
			`[{"match_expressions":[{"key":"team","operator":"NotIn","values":["one"]}]}]`, true},
		{"expression value", `[{"match_expressions":[{"key":"team","operator":"In","values":["one"]}]}]`,
			`[{"match_expressions":[{"key":"team","operator":"In","values":["two"]}]}]`, true},
		{"expression removed", `[{"match_expressions":[{"key":"team","operator":"Exists"}]}]`,
			`[{"match_expressions":[]}]`, true},
		{"empty expression normalization", `[{"match_labels":{"team":"same"},"match_expressions":[]}]`,
			`[{"match_labels":{"team":"same"}}]`, false},
		{"empty label normalization", `[{"match_labels":{},"match_expressions":[{"key":"team","operator":"Exists"}]}]`,
			`[{"match_expressions":[{"key":"team","operator":"Exists"}]}]`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := func(selector string) tfsdk.State {
				raw := strings.Replace(jobUnitStateJSON, `"manual_selector":false`,
					`"manual_selector":true,"selector":`+selector, 1)
				return jobUnitState(t, &batchv1.JobV1{}, raw)
			}
			prior, proposed := state(test.before), state(test.after)
			model := jobUnitModel(t, proposed)
			model.ID = types.StringNull()
			model.Metadata[0].Generation = types.Int64Null()
			model.Metadata[0].UID = types.StringNull()
			model.Metadata[0].ResourceVersion = types.StringNull()
			config := jobUnitModelState(t, proposed, model)
			dynamic := func(value tfsdk.State) *tfprotov6.DynamicValue {
				result, err := tfprotov6.NewDynamicValue(value.Raw.Type(), value.Raw)
				if err != nil {
					t.Fatal(err)
				}
				return &result
			}
			response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_job_v1", PriorState: dynamic(prior),
				ProposedNewState: dynamic(proposed), Config: dynamic(config),
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatal(diagnostic)
				}
			}
			if (len(response.RequiresReplace) > 0) != test.replace {
				t.Fatalf("replacement paths=%v, expected replacement=%t", response.RequiresReplace, test.replace)
			}
		})
	}
}

func TestJobIndexedBackoffRemovalDefaults(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	dynamic := func(value tfsdk.State) *tfprotov6.DynamicValue {
		result, err := tfprotov6.NewDynamicValue(value.Raw.Type(), value.Raw)
		if err != nil {
			t.Fatal(err)
		}
		return &result
	}
	for _, test := range []struct {
		name    string
		value   int64
		replace bool
	}{
		{"zero-no-op", 0, false},
		{"nonzero-replacement", 2, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := jobUnitState(t, &batchv1.JobV1{}, jobUnitStateJSON)
			model := jobUnitModel(t, state)
			jobUnitSpec(&model, "completion_mode", types.StringValue("Indexed"))
			jobUnitSpec(&model, "backoff_limit_per_index", types.Int64Value(test.value))
			state = jobUnitModelState(t, state, model)
			jobUnitSpec(&model, "backoff_limit_per_index", types.Int64Null())
			proposed := jobUnitModelState(t, state, model)
			model.ID = types.StringNull()
			model.Metadata[0].Generation = types.Int64Null()
			model.Metadata[0].UID = types.StringNull()
			model.Metadata[0].ResourceVersion = types.StringNull()
			config := jobUnitModelState(t, state, model)
			response, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
				TypeName: "kubernetes_job_v1", PriorState: dynamic(state),
				ProposedNewState: dynamic(proposed), Config: dynamic(config),
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatal(diagnostic)
				}
			}
			found := false
			for _, replacement := range response.RequiresReplace {
				if strings.Contains(replacement.String(), "backoff_limit_per_index") {
					found = true
				}
			}
			if found != test.replace {
				t.Fatalf("replacement paths = %v, want replacement %t", response.RequiresReplace, test.replace)
			}
			raw, err := response.PlannedState.Unmarshal(state.Raw.Type())
			if err != nil {
				t.Fatal(err)
			}
			planned := tfsdk.State{Schema: state.Schema, Raw: raw}
			var limit types.Int64
			if diagnostics := planned.GetAttribute(ctx, path.Root("spec").AtListIndex(0).AtName("backoff_limit_per_index"), &limit); diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if limit != types.Int64Value(0) {
				t.Fatalf("removed backoff limit = %v, want default zero", limit)
			}
		})
	}
}

func TestJobProtocolUpdateKeepsComputedCompletionMode(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	job := jobUnitAPIJob()
	job.Spec.CompletionMode = ptr.To(batchapi.IndexedCompletion)
	job.Spec.BackoffLimitPerIndex = ptr.To(int32(2))
	job.Spec.MaxFailedIndexes = ptr.To(int32(1))
	requests := 0
	r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
		requests++
		if req.Method != http.MethodPatch {
			t.Errorf("expected spec patch, received %s", req.Method)
		}
		var operations []map[string]any
		if err := json.NewDecoder(req.Body).Decode(&operations); err != nil {
			t.Error(err)
		}
		if len(operations) != 1 || operations[0]["path"] != "/spec/maxFailedIndexes" || operations[0]["value"] != float64(0) {
			t.Errorf("expected maxFailedIndexes=0, received %#v", operations)
		}
		job.Spec.MaxFailedIndexes = nil
		if len(operations) == 1 {
			if value, ok := operations[0]["value"].(float64); ok {
				job.Spec.MaxFailedIndexes = ptr.To(int32(value))
			}
		}
		jobUnitWrite(t, w, job)
	})
	state := jobUnitState(t, r, jobUnitStateJSON)
	model := jobUnitModel(t, state)
	jobUnitSpec(&model, "completion_mode", types.StringValue("Indexed"))
	jobUnitSpec(&model, "backoff_limit_per_index", types.Int64Value(2))
	jobUnitSpec(&model, "max_failed_indexes", types.Int64Value(1))
	state = jobUnitModelState(t, state, model)
	jobUnitSpec(&model, "max_failed_indexes", types.Int64Value(0))
	proposed := jobUnitModelState(t, state, model)
	jobUnitSpec(&model, "completion_mode", types.StringNull())
	model.ID = types.StringNull()
	model.Metadata[0].Generation = types.Int64Null()
	model.Metadata[0].UID = types.StringNull()
	model.Metadata[0].ResourceVersion = types.StringNull()
	config := jobUnitModelState(t, state, model)
	dynamic := func(value tfsdk.State) *tfprotov6.DynamicValue {
		result, err := tfprotov6.NewDynamicValue(value.Raw.Type(), value.Raw)
		if err != nil {
			t.Fatal(err)
		}
		return &result
	}
	planned, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "kubernetes_job_v1", PriorState: dynamic(state),
		ProposedNewState: dynamic(proposed), Config: dynamic(config),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range planned.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatal(diagnostic)
		}
	}
	if len(planned.RequiresReplace) != 0 {
		t.Fatalf("max_failed_indexes update requested replacement: %v", planned.RequiresReplace)
	}
	raw, err := planned.PlannedState.Unmarshal(state.Raw.Type())
	if err != nil {
		t.Fatal(err)
	}
	plan := tfsdk.Plan{Schema: state.Schema, Raw: raw}
	modePath := path.Root("spec").AtListIndex(0).AtName("completion_mode")
	var mode types.String
	if diagnostics := plan.GetAttribute(ctx, modePath, &mode); diagnostics.HasError() || !mode.IsUnknown() {
		t.Fatalf("expected omitted computed completion mode to become unknown, got %s; %s", mode, diagnostics)
	}
	response := fwresource.UpdateResponse{State: state}
	r.Update(ctx, fwresource.UpdateRequest{State: state, Plan: plan}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	if requests != 1 || job.Spec.MaxFailedIndexes == nil || *job.Spec.MaxFailedIndexes != 0 {
		t.Fatalf("requested limit not applied: requests=%d, maxFailedIndexes=%v", requests, job.Spec.MaxFailedIndexes)
	}
	var limit types.Int64
	if diagnostics := response.State.GetAttribute(ctx, path.Root("spec").AtListIndex(0).AtName("max_failed_indexes"), &limit); diagnostics.HasError() || limit != types.Int64Value(0) {
		t.Fatalf("updated limit missing from state: value=%s; %s", limit, diagnostics)
	}
	if diagnostics := plan.GetAttribute(ctx, modePath, &mode); diagnostics.HasError() || !mode.IsUnknown() {
		t.Fatalf("API payload preparation changed the actual plan: mode=%s; %s", mode, diagnostics)
	}
}

func TestJobProtocolUpdateKeepsUnconfiguredLabels(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	job := jobUnitAPIJob()
	job.Labels = map[string]string{"external-controller": "keep"}
	r := jobUnitResource(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPatch {
			t.Errorf("expected spec patch only, received %s", req.Method)
		}
		var operations []map[string]any
		if err := json.NewDecoder(req.Body).Decode(&operations); err != nil {
			t.Error(err)
		}
		if len(operations) != 1 || operations[0]["path"] != "/spec/parallelism" {
			t.Errorf("unconfigured labels were patched: %#v", operations)
		}
		job.Spec.Parallelism = ptr.To(int32(2))
		jobUnitWrite(t, w, job)
	})
	state := jobUnitState(t, r, jobUnitStateJSON)
	model := jobUnitModel(t, state)
	model.Metadata[0].Labels = types.MapValueMust(types.StringType, map[string]attr.Value{
		"external-controller": types.StringValue("keep"),
	})
	state = jobUnitModelState(t, state, model)
	jobUnitSpec(&model, "parallelism", types.Int64Value(2))
	proposed := jobUnitModelState(t, state, model)
	model.Metadata[0].Labels = types.MapNull(types.StringType)
	model.Metadata[0].UID = types.StringNull()
	model.Metadata[0].ResourceVersion = types.StringNull()
	model.Metadata[0].Generation = types.Int64Null()
	model.ID = types.StringNull()
	config := jobUnitModelState(t, state, model)
	dynamic := func(state tfsdk.State) *tfprotov6.DynamicValue {
		value, err := tfprotov6.NewDynamicValue(state.Raw.Type(), state.Raw)
		if err != nil {
			t.Fatal(err)
		}
		return &value
	}
	planned, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName: "kubernetes_job_v1", PriorState: dynamic(state),
		ProposedNewState: dynamic(proposed), Config: dynamic(config),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range planned.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatal(diagnostic)
		}
	}
	if len(planned.RequiresReplace) != 0 {
		t.Fatalf("parallelism-only plan requested replacement: %v", planned.RequiresReplace)
	}
	raw, err := planned.PlannedState.Unmarshal(state.Raw.Type())
	if err != nil {
		t.Fatal(err)
	}
	plannedModel := jobUnitModel(t, tfsdk.State{Schema: state.Schema, Raw: raw})
	if !plannedModel.Metadata[0].Labels.IsUnknown() {
		t.Fatalf("expected unconfigured computed labels to become unknown, received %s", plannedModel.Metadata[0].Labels)
	}
	response := fwresource.UpdateResponse{State: state}
	r.Update(ctx, fwresource.UpdateRequest{State: state, Plan: tfsdk.Plan{Schema: state.Schema, Raw: raw}}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	actual := jobUnitModel(t, response.State)
	if actual.Metadata[0].Labels.Elements()["external-controller"] != types.StringValue("keep") {
		t.Fatalf("external label lost from refreshed state: %s", actual.Metadata[0].Labels)
	}
}

func TestJobProtocolValidators(t *testing.T) {
	ctx := context.Background()
	server, err := mux.MuxServer(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	r := &batchv1.JobV1{}
	state := jobUnitState(t, r, jobUnitStateJSON)
	for _, tc := range []struct {
		field string
		value attr.Value
	}{
		{"active_deadline_seconds", types.Int64Value(0)},
		{"backoff_limit", types.Int64Value(-1)},
		{"backoff_limit_per_index", types.Int64Value(-1)},
		{"max_failed_indexes", types.Int64Value(-1)},
		{"parallelism", types.Int64Value(-1)},
		{"completion_mode", types.StringValue("invalid")},
		{"ttl_seconds_after_finished", types.StringValue("-1")},
		{"ttl_seconds_after_finished", types.StringValue("not-an-integer")},
	} {
		t.Run(tc.field+"/"+tc.value.String(), func(t *testing.T) {
			model := jobUnitModel(t, state)
			model.ID = types.StringNull()
			model.Metadata[0].Generation = types.Int64Null()
			model.Metadata[0].UID = types.StringNull()
			model.Metadata[0].ResourceVersion = types.StringNull()
			jobUnitSpec(&model, tc.field, tc.value)
			config := jobUnitModelState(t, state, model)
			value, err := tfprotov6.NewDynamicValue(config.Raw.Type(), config.Raw)
			if err != nil {
				t.Fatal(err)
			}
			response, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
				TypeName: "kubernetes_job_v1", Config: &value,
			})
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, diagnostic := range response.Diagnostics {
				if diagnostic.Severity == tfprotov6.DiagnosticSeverityError && diagnostic.Attribute != nil &&
					strings.Contains(diagnostic.Attribute.String(), tc.field) {
					found = true
				}
			}
			if !found {
				t.Fatalf("invalid %s accepted or errored at wrong path: %v", tc.field, response.Diagnostics)
			}
		})
	}
}
