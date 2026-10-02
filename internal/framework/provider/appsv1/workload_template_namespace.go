// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

// workloadTemplateMetadataBlock is the pod template metadata of Deployment and DaemonSet.
func workloadTemplateMetadataBlock() schema.ListNestedBlock {
	block := common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("pod", true))
	namespace := block.NestedObject.Attributes["namespace"].(schema.StringAttribute)
	namespace.Computed = true
	namespace.Default = workloadTemplateNamespace{}
	namespace.PlanModifiers = []planmodifier.String{
		workloadTemplateNamespace{},
		stringplanmodifier.RequiresReplace(),
	}
	block.NestedObject.Attributes["namespace"] = namespace
	return block
}

// Template namespace accepts an explicit empty string. Retain that representation
// from SDKv2 so an unchanged configuration never replaces the workload.
type workloadTemplateNamespace struct{}

func (workloadTemplateNamespace) Description(context.Context) string {
	return "Default to null while preserving an existing empty pod-template namespace."
}

func (modifier workloadTemplateNamespace) MarkdownDescription(ctx context.Context) string {
	return modifier.Description(ctx)
}

func (workloadTemplateNamespace) DefaultString(_ context.Context, _ defaults.StringRequest, resp *defaults.StringResponse) {
	resp.PlanValue = types.StringNull()
}

func (workloadTemplateNamespace) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.Plan.Raw.IsNull() || !req.ConfigValue.IsNull() {
		return
	}
	resp.PlanValue = req.ConfigValue
	if req.StateValue.Equal(types.StringValue("")) {
		resp.PlanValue = req.StateValue
	}
}
