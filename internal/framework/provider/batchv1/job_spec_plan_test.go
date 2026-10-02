// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestJobSpecPayloadEquivalence(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	(&JobV1{}).Schema(ctx, resource.SchemaRequest{}, &response)
	base := `{"spec":[{
		"completion_mode":"NonIndexed","backoff_limit_per_index":0,
		"template":[{"metadata":[{"generate_name":""}],
			"spec":[{"container":[{"name":"test","image":"busybox","working_dir":"","image_pull_policy":"Always"}]}]}]
	}]}`
	parse := func(raw string) tfsdk.State {
		t.Helper()
		value, err := (&tfprotov6.RawState{JSON: []byte(raw)}).Unmarshal(response.Schema.Type().TerraformType(ctx))
		if err != nil {
			t.Fatal(err)
		}
		return tfsdk.State{Schema: response.Schema, Raw: value}
	}
	for _, tc := range []struct {
		name, before, after string
		want                bool
	}{
		{"same payload", base, base, true},
		{"legacy optional strings", base, strings.ReplaceAll(base, `:""`, `:null`), true},
		{"legacy unused backoff", base, strings.Replace(base, `"backoff_limit_per_index":0`, `"backoff_limit_per_index":null`, 1), true},
		{"indexed zero backoff removal", strings.Replace(base, `"NonIndexed"`, `"Indexed"`, 1),
			strings.Replace(strings.Replace(base, `"NonIndexed"`, `"Indexed"`, 1), `"backoff_limit_per_index":0`, `"backoff_limit_per_index":null`, 1), false},
		{"mutable parallelism", base, strings.Replace(base, `"completion_mode"`, `"parallelism":2,"completion_mode"`, 1), true},
		{"image changes", base, strings.Replace(base, `"busybox"`, `"busybox:next"`, 1), false},
		{"configured workdir removed", strings.Replace(base, `"working_dir":""`, `"working_dir":"/app"`, 1), strings.Replace(base, `"working_dir":""`, `"working_dir":null`, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, after := parse(tc.before), parse(tc.after)
			if got := jobSpecPayloadEquivalent(ctx,
				tfsdk.Plan{Schema: response.Schema, Raw: after.Raw}, before,
				tfsdk.Config{Schema: response.Schema, Raw: after.Raw}); got != tc.want {
				t.Fatalf("payload equivalent = %t, want %t", got, tc.want)
			}
		})
	}
	t.Run("configured unknown cannot prove equivalence", func(t *testing.T) {
		state := parse(base)
		plan := tfsdk.Plan{Schema: response.Schema, Raw: state.Raw}
		imagePath := path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).
			AtName("spec").AtListIndex(0).AtName("container").AtListIndex(0).AtName("image")
		if diags := plan.SetAttribute(ctx, imagePath, types.StringUnknown()); diags.HasError() {
			t.Fatal(diags)
		}
		if jobSpecPayloadEquivalent(ctx, plan, state, tfsdk.Config{Schema: response.Schema, Raw: plan.Raw}) {
			t.Fatal("configured unknown was treated as equivalent")
		}
	})
	t.Run("computed unknown uses prior only for comparison", func(t *testing.T) {
		state := parse(base)
		plan := tfsdk.Plan{Schema: response.Schema, Raw: state.Raw}
		config := tfsdk.Plan{Schema: response.Schema, Raw: state.Raw}
		policyPath := path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).
			AtName("spec").AtListIndex(0).AtName("container").AtListIndex(0).AtName("image_pull_policy")
		if diags := plan.SetAttribute(ctx, policyPath, types.StringUnknown()); diags.HasError() {
			t.Fatal(diags)
		}
		if diags := config.SetAttribute(ctx, policyPath, types.StringNull()); diags.HasError() {
			t.Fatal(diags)
		}
		if !jobSpecPayloadEquivalent(ctx, plan, state, tfsdk.Config{Schema: response.Schema, Raw: config.Raw}) {
			t.Fatal("unconfigured computed unknown prevented safe comparison")
		}
		var unchanged types.String
		if diags := plan.GetAttribute(ctx, policyPath, &unchanged); diags.HasError() || !unchanged.IsUnknown() {
			t.Fatal("payload comparator rewrote the Terraform plan")
		}
	})
}
