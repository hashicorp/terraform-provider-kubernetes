// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	batchapi "k8s.io/api/batch/v1"
)

func TestJobSpecIndexedOptionalIntegerPresence(t *testing.T) {
	ctx := context.Background()
	var response resource.SchemaResponse
	(&JobV1{}).Schema(ctx, resource.SchemaRequest{}, &response)
	for _, test := range []struct {
		name, value string
		present     bool
	}{
		{"omitted", "null", false},
		{"explicit zero", "0", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := fmt.Sprintf(`{"spec":[{
				"completion_mode":"Indexed",
				"backoff_limit_per_index":%s,"max_failed_indexes":%s,
				"template":[{"metadata":[{}],"spec":[{"container":[{"name":"test","image":"busybox"}]}]}]
			}]}`, test.value, test.value)
			value, err := (&tfprotov6.RawState{JSON: []byte(raw)}).Unmarshal(response.Schema.Type().TerraformType(ctx))
			if err != nil {
				t.Fatal(err)
			}

			var model JobV1Model
			if diagnostics := (tfsdk.State{Schema: response.Schema, Raw: value}).Get(ctx, &model); diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			spec, diagnostics := expandJobSpec(ctx, model.Spec)
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			if (spec.BackoffLimitPerIndex != nil) != test.present || (spec.MaxFailedIndexes != nil) != test.present {
				t.Fatalf("optional integer presence lost: backoff=%v, maxFailed=%v; want present=%t",
					spec.BackoffLimitPerIndex, spec.MaxFailedIndexes, test.present)
			}
			if test.present && (*spec.BackoffLimitPerIndex != 0 || *spec.MaxFailedIndexes != 0) {
				t.Fatal("explicit zero was not preserved")
			}
		})
	}
}

func TestJobSpecPolicyWithoutContainerName(t *testing.T) {
	spec := batchapi.JobSpec{PodFailurePolicy: &batchapi.PodFailurePolicy{
		Rules: []batchapi.PodFailurePolicyRule{{
			Action: batchapi.PodFailurePolicyActionIgnore,
			OnExitCodes: &batchapi.PodFailurePolicyOnExitCodesRequirement{
				Operator: batchapi.PodFailurePolicyOnExitCodesOpIn,
				Values:   []int32{1},
			},
		}},
	}}
	value, diagnostics := flattenJobSpec(context.Background(), spec, types.ListNull(jobSpecType()))
	if diagnostics.HasError() {
		t.Fatal(diagnostics)
	}
	attributes := value.Elements()[0].(types.Object).Attributes()
	policy := attributes["pod_failure_policy"].(types.List).Elements()[0].(types.Object).Attributes()
	rule := policy["rule"].(types.List).Elements()[0].(types.Object).Attributes()
	requirement := rule["on_exit_codes"].(types.List).Elements()[0].(types.Object).Attributes()
	if !requirement["container_name"].IsNull() {
		t.Fatalf("omitted container name became %s", requirement["container_name"])
	}
}
