// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// rollingUpdateSpelling returns max_surge or max_unavailable read from
// Kubernetes, keeping the spelling in the prior strategy list, such as "01",
// when it denotes the same value.
func rollingUpdateSpelling(prior types.List, name, current string) types.String {
	value := types.StringValue(current)
	if prior.IsNull() || prior.IsUnknown() || len(prior.Elements()) != 1 {
		return value
	}
	rolling, ok := priorAttribute(prior.Elements()[0], "rolling_update").(types.List)
	if !ok || rolling.IsNull() || rolling.IsUnknown() || len(rolling.Elements()) != 1 {
		return value
	}
	if spelled, ok := priorAttribute(rolling.Elements()[0], name).(types.String); ok {
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
