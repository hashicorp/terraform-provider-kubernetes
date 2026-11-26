// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package gatewaynetworkingv1

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type gatewayClassModel struct {
	ID       types.String           `tfsdk:"id"`
	Metadata []common.MetadataModel `tfsdk:"metadata"`
	Spec     []gatewayClassSpec     `tfsdk:"spec"`
	Status   types.Object           `tfsdk:"status"`
	Timeouts timeouts.Value         `tfsdk:"timeouts"`
}

type gatewayClassSpec struct {
	ControllerName types.String    `tfsdk:"controller_name"`
	Description    types.String    `tfsdk:"description"`
	ParametersRef  []parametersRef `tfsdk:"parameters_ref"`
}

type parametersRef struct {
	Group     types.String `tfsdk:"group"`
	Kind      types.String `tfsdk:"kind"`
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
}

var conditionTypes = map[string]attr.Type{
	"type": types.StringType, "status": types.StringType,
	"reason": types.StringType, "message": types.StringType,
	"observed_generation": types.Int64Type, "last_transition_time": types.StringType,
}

var statusTypes = map[string]attr.Type{
	"conditions":         types.ListType{ElemType: types.ObjectType{AttrTypes: conditionTypes}},
	"supported_features": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}},
}

// expand converts the planned spec to the API type. Unset optional fields stay
// nil pointers, so Create omits them; Update sends them as explicit nulls
// instead (see Update).
func (s gatewayClassSpec) expand() gatewayv1.GatewayClassSpec {
	out := gatewayv1.GatewayClassSpec{ControllerName: gatewayv1.GatewayController(s.ControllerName.ValueString())}
	if !s.Description.IsNull() {
		value := s.Description.ValueString()
		out.Description = &value
	}
	if len(s.ParametersRef) == 1 {
		ref := s.ParametersRef[0]
		out.ParametersRef = &gatewayv1.ParametersReference{
			Group: gatewayv1.Group(ref.Group.ValueString()), Kind: gatewayv1.Kind(ref.Kind.ValueString()), Name: ref.Name.ValueString(),
		}
		if !ref.Namespace.IsNull() {
			namespace := gatewayv1.Namespace(ref.Namespace.ValueString())
			out.ParametersRef.Namespace = &namespace
		}
	}
	return out
}

// flattenSpec converts the API spec for both the resource and the data source.
// Absent optional values become null, and an absent parametersRef becomes an
// empty list rather than null, which is how Terraform represents a list block
// with no blocks configured.
func flattenSpec(spec gatewayv1.GatewayClassSpec) gatewayClassSpec {
	out := gatewayClassSpec{ControllerName: types.StringValue(string(spec.ControllerName)), Description: types.StringNull(), ParametersRef: []parametersRef{}}
	if spec.Description != nil {
		out.Description = types.StringValue(*spec.Description)
	}
	if spec.ParametersRef != nil {
		ref := spec.ParametersRef
		value := parametersRef{
			Group: types.StringValue(string(ref.Group)), Kind: types.StringValue(string(ref.Kind)),
			Name: types.StringValue(ref.Name), Namespace: types.StringNull(),
		}
		if ref.Namespace != nil {
			value.Namespace = types.StringValue(string(*ref.Namespace))
		}
		out.ParametersRef = []parametersRef{value}
	}
	return out
}

// flattenStatus converts the status reported by the controller, or defaulted
// by the CRD before one reports, into the computed status object. Both lists
// are always known, possibly empty. metav1.Time decodes into the local time
// zone, so timestamps are formatted in UTC to keep state the same wherever
// Terraform runs.
func flattenStatus(ctx context.Context, status gatewayv1.GatewayClassStatus) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	conditions := make([]attr.Value, 0, len(status.Conditions))
	for _, condition := range status.Conditions {
		value, d := types.ObjectValue(conditionTypes, map[string]attr.Value{
			"type": types.StringValue(condition.Type), "status": types.StringValue(string(condition.Status)),
			"reason": types.StringValue(condition.Reason), "message": types.StringValue(condition.Message),
			"observed_generation":  types.Int64Value(condition.ObservedGeneration),
			"last_transition_time": types.StringValue(condition.LastTransitionTime.UTC().Format(time.RFC3339)),
		})
		diags.Append(d...)
		conditions = append(conditions, value)
	}
	list, d := types.ListValue(types.ObjectType{AttrTypes: conditionTypes}, conditions)
	diags.Append(d...)
	features := make([]map[string]string, 0, len(status.SupportedFeatures))
	for _, feature := range status.SupportedFeatures {
		features = append(features, map[string]string{"name": string(feature.Name)})
	}
	featureList, d := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: map[string]attr.Type{"name": types.StringType}}, features)
	diags.Append(d...)
	value, d := types.ObjectValue(statusTypes, map[string]attr.Value{"conditions": list, "supported_features": featureList})
	diags.Append(d...)
	return value, diags
}
