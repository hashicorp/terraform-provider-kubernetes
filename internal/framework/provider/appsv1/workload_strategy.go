// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// deploymentStrategyValidator rejects rolling_update with the Recreate
// strategy, a combination Kubernetes forbids.
type deploymentStrategyValidator struct{}

func (deploymentStrategyValidator) Description(context.Context) string {
	return "rolling_update cannot be set when type is Recreate"
}

func (v deploymentStrategyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (deploymentStrategyValidator) ValidateObject(_ context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	attributes := req.ConfigValue.Attributes()
	strategyType, _ := attributes["type"].(types.String)
	rolling := attributes["rolling_update"]
	if strategyType.ValueString() != "Recreate" || rolling == nil || rolling.IsNull() || rolling.IsUnknown() {
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path.AtName("rolling_update"), "Invalid strategy",
		"rolling_update cannot be set when type is \"Recreate\"; Kubernetes rejects the combination. Remove rolling_update or set it to null.")
}

// rollingUpdateSpelling returns max_surge or max_unavailable read from
// Kubernetes, keeping the spelling in the prior strategy object, such as "01",
// when it denotes the same value.
func rollingUpdateSpelling(prior types.Object, name, current string) types.String {
	value := types.StringValue(current)
	if prior.IsNull() || prior.IsUnknown() {
		return value
	}
	rolling, ok := priorAttribute(prior, "rolling_update").(types.Object)
	if !ok || rolling.IsNull() || rolling.IsUnknown() {
		return value
	}
	if spelled, ok := priorAttribute(rolling, name).(types.String); ok {
		return common.KeepIntOrStringSpelling(spelled, value)
	}
	return value
}

func priorAttribute(element attr.Value, name string) attr.Value {
	object, ok := element.(types.Object)
	if !ok || object.IsNull() || object.IsUnknown() {
		return nil
	}
	return object.Attributes()[name]
}
