// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	batchapi "k8s.io/api/batch/v1"
)

func expandJobSpec(ctx context.Context, spec types.List) (batchapi.JobSpec, diag.Diagnostics) {
	raw, diags := legacyValue(ctx, spec)
	if diags.HasError() {
		return batchapi.JobSpec{}, diags
	}
	list, ok := raw.([]interface{})
	if !ok {
		diags.AddError("Invalid Job specification", fmt.Sprintf("Expected a list, got %T.", raw))
		return batchapi.JobSpec{}, diags
	}
	result, err := kubernetes.ExpandJobV1Spec(list)
	if err != nil {
		diags.AddError("Invalid Job specification", err.Error())
		return result, diags
	}
	if len(spec.Elements()) == 1 {
		object, ok := spec.Elements()[0].(types.Object)
		if !ok {
			diags.AddError("Invalid Job specification", "Expected spec to contain an object.")
			return result, diags
		}
		// SDKv2 represents omitted integers as zero. For Indexed Jobs these
		// optional pointer fields must distinguish omission from explicit zero.
		attributes := object.Attributes()
		if value, exists := attributes["backoff_limit_per_index"]; exists && value.IsNull() {
			result.BackoffLimitPerIndex = nil
		}
		if value, exists := attributes["max_failed_indexes"]; exists && value.IsNull() {
			result.MaxFailedIndexes = nil
		}
	}
	return result, diags
}

func flattenJobSpec(ctx context.Context, spec batchapi.JobSpec, prior types.List) (types.List, diag.Diagnostics) {
	raw, err := kubernetes.FlattenJobV1Spec(spec)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Unable to read Job specification", err.Error())
		return types.ListNull(jobSpecType()), diags
	}
	return valueFromAPIField(ctx, jobSpecValueField(), raw, prior)
}
