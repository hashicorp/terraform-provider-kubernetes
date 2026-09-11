// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package admissionregistrationv1

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ValidatingAdmissionPolicyBindingModel struct {
	Timeouts timeouts.Value                            `tfsdk:"timeouts"`
	ID       types.String                              `tfsdk:"id"`
	Metadata MetadataModel                             `tfsdk:"metadata"`
	Spec     ValidatingAdmissionPolicyBindingSpecModel `tfsdk:"spec"`
}

type ValidatingAdmissionPolicyBindingSpecModel struct {
	MatchResources    *MatchConstraintsModel `tfsdk:"match_resources"`
	ParamRef          *ParamRefModel         `tfsdk:"param_ref"`
	PolicyName        types.String           `tfsdk:"policy_name"`
	ValidationActions []types.String         `tfsdk:"validation_actions"`
}

type ParamRefModel struct {
	Name                    types.String        `tfsdk:"name"`
	Namespace               types.String        `tfsdk:"namespace"`
	ParameterNotFoundAction types.String        `tfsdk:"parameter_not_found_action"`
	Selector                *LabelSelectorModel `tfsdk:"selector"`
}
