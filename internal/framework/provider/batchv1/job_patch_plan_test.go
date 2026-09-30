// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestJobPatchWithUnknownCompletionMode(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	(&JobV1{}).Schema(ctx, resource.SchemaRequest{}, &response)
	for _, test := range []struct {
		name  string
		prior string
		value types.Int64
		want  string
	}{
		{"increase", "1", types.Int64Value(2), `[{"op":"add","path":"/spec/maxFailedIndexes","value":2}]`},
		{"add", "null", types.Int64Value(2), `[{"op":"add","path":"/spec/maxFailedIndexes","value":2}]`},
		{"zero", "1", types.Int64Value(0), `[{"op":"add","path":"/spec/maxFailedIndexes","value":0}]`},
		{"remove", "1", types.Int64Null(), `[{"op":"add","path":"/spec/maxFailedIndexes","value":null}]`},
		{"unchanged", "1", types.Int64Value(1), `[]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := fmt.Sprintf(`{"spec":[{
				"completion_mode":"Indexed","backoff_limit_per_index":3,"max_failed_indexes":%s,
				"template":[{"metadata":[{}],"spec":[{"container":[{"name":"test","image":"busybox"}]}]}]
			}]}`, test.prior)
			value, err := (&tfprotov6.RawState{JSON: []byte(raw)}).Unmarshal(response.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}
			state := tfsdk.State{Schema: response.Schema, Raw: value}
			var previous types.List
			if diagnostics := state.GetAttribute(ctx, path.Root("spec"), &previous); diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			object := previous.Elements()[0].(types.Object)
			fields := object.Attributes()
			fields["completion_mode"] = types.StringUnknown()
			fields["max_failed_indexes"] = test.value
			plan := types.ListValueMust(previous.ElementType(ctx), []attr.Value{
				types.ObjectValueMust(object.AttributeTypes(ctx), fields),
			})
			before, err := plan.ToTerraformValue(ctx)
			if err != nil {
				t.Fatal(err)
			}
			operations, diagnostics := patchJobSpec(ctx, previous, plan)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			data, err := operations.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(test.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("patch = %s, want %s", data, test.want)
			}
			after, err := plan.ToTerraformValue(ctx)
			if err != nil || !after.Equal(before) {
				t.Fatal("patch construction changed the Terraform plan")
			}
		})
	}
}
