// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// workloadTemplateMetadataBlock is the pod template metadata of Deployment,
// DaemonSet and StatefulSet. The API never assigns a template generation,
// resource version or UID, so those keep their prior values in a plan.
func workloadTemplateMetadataBlock() schema.ListNestedBlock {
	block := common.WithEmptyMetadataCompatibility(common.NamespacedMetadataSchema("pod", true))
	attributes := block.NestedObject.Attributes
	generation := attributes["generation"].(schema.Int64Attribute)
	generation.PlanModifiers = []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}
	attributes["generation"] = generation
	resourceVersion := attributes["resource_version"].(schema.StringAttribute)
	resourceVersion.PlanModifiers = []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	attributes["resource_version"] = resourceVersion
	namespace := attributes["namespace"].(schema.StringAttribute)
	namespace.Computed = true
	namespace.Default = workloadTemplateNamespace{}
	namespace.PlanModifiers = []planmodifier.String{
		workloadTemplateNamespace{},
		stringplanmodifier.RequiresReplace(),
	}
	attributes["namespace"] = namespace
	return block
}

// flattenWorkloadTemplateMetadata records every live template annotation and
// label on refresh or import, since ignore_annotations and ignore_labels do not
// apply to templates, and the planned maps after Create and Update.
func flattenWorkloadTemplateMetadata(ctx context.Context, meta metav1.ObjectMeta, prior []common.NamespacedMetadataModel, refresh bool) ([]common.NamespacedMetadataModel, diag.Diagnostics) {
	var previous common.NamespacedMetadataModel
	if len(prior) > 0 {
		previous = prior[0]
	}
	result := common.NamespacedMetadataModel{
		MetadataModel: common.MetadataModel{
			MetadataBase: common.MetadataBase{
				Name:            types.StringValue(meta.Name),
				Generation:      types.Int64Value(meta.Generation),
				ResourceVersion: types.StringValue(meta.ResourceVersion),
				UID:             types.StringValue(string(meta.UID)),
			},
			GenerateName: types.StringNull(),
		},
		Namespace: types.StringNull(),
	}
	if meta.GenerateName != "" {
		result.GenerateName = types.StringValue(meta.GenerateName)
	}
	if meta.Namespace != "" || previous.Namespace.Equal(types.StringValue("")) {
		result.Namespace = types.StringValue(meta.Namespace)
	}

	var diags, d diag.Diagnostics
	result.Annotations, d = workloadTemplateMetadataMap(ctx, meta.Annotations, previous.Annotations, refresh || len(prior) == 0)
	diags.Append(d...)
	result.Labels, d = workloadTemplateMetadataMap(ctx, meta.Labels, previous.Labels, refresh || len(prior) == 0)
	diags.Append(d...)
	return []common.NamespacedMetadataModel{result}, diags
}

// workloadTemplateMetadataMap returns the live map, keeping a prior empty map.
// Unless all is set, a known prior map is the plan and is returned as it is.
func workloadTemplateMetadataMap(ctx context.Context, live map[string]string, prior types.Map, all bool) (types.Map, diag.Diagnostics) {
	if !all && !prior.IsUnknown() {
		return prior, nil
	}
	if len(live) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() {
			return types.MapValueMust(types.StringType, nil), nil
		}
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, live)
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
