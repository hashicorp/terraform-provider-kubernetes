// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.ResourceWithModifyPlan = (*JobV1)(nil)
	_ resource.ResourceWithModifyPlan = (*CronJobV1)(nil)
)

func (r *JobV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	preserveWorkloadNoopPlan(ctx, req, resp)
}

func (r *CronJobV1) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	preserveWorkloadNoopPlan(ctx, req, resp)
}
