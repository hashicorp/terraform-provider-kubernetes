// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv1

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

var (
	_ resource.Resource                = (*HorizontalPodAutoscalerV1)(nil)
	_ resource.ResourceWithConfigure   = (*HorizontalPodAutoscalerV1)(nil)
	_ resource.ResourceWithImportState = (*HorizontalPodAutoscalerV1)(nil)
	_ resource.ResourceWithMoveState   = (*HorizontalPodAutoscalerV1)(nil)
)

const (
	unversionedHPATypeName      = "kubernetes_horizontal_pod_autoscaler"
	unversionedHPASchemaVersion = 0
	providerAddressSuffix       = "/hashicorp/kubernetes"
	moveErrorSummary            = "Unable to move kubernetes_horizontal_pod_autoscaler state"
)

type HorizontalPodAutoscalerV1 struct {
	SDKv2Meta func() any
}

func NewHorizontalPodAutoscalerV1() resource.Resource {
	return &HorizontalPodAutoscalerV1{}
}

func (r *HorizontalPodAutoscalerV1) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_horizontal_pod_autoscaler_v1"
}

func (r *HorizontalPodAutoscalerV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.SDKv2Meta = req.ProviderData.(func() any)
}

func (r *HorizontalPodAutoscalerV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Read validates the namespace/name ID and populates the rest of the state.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *HorizontalPodAutoscalerV1) MoveState(_ context.Context) []resource.StateMover {
	return []resource.StateMover{{
		StateMover: r.moveFromUnversioned,
	}}
}

// moveFromUnversioned accepts state from the deprecated kubernetes_horizontal_pod_autoscaler
// resource. That resource used autoscaling/v1 unless metric or behavior was set, so only
// state without either maps onto this resource without losing configuration.
func (r *HorizontalPodAutoscalerV1) moveFromUnversioned(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
	if req.SourceTypeName != unversionedHPATypeName ||
		req.SourceSchemaVersion != unversionedHPASchemaVersion ||
		!strings.HasSuffix(req.SourceProviderAddress, providerAddressSuffix) {
		tflog.Debug(ctx, "MoveState: not a kubernetes_horizontal_pod_autoscaler v0 source, skipping", map[string]any{
			"source_type_name":        req.SourceTypeName,
			"source_schema_version":   req.SourceSchemaVersion,
			"source_provider_address": req.SourceProviderAddress,
		})
		return
	}
	if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
		resp.Diagnostics.AddError(moveErrorSummary, "The source state has no JSON data. Flatmap state is not supported.")
		return
	}

	var src sdkv2HorizontalPodAutoscalerStateV0
	if err := json.Unmarshal(req.SourceRawState.JSON, &src); err != nil {
		resp.Diagnostics.AddError(moveErrorSummary, "The prior state of the source resource could not be decoded: "+err.Error())
		return
	}
	if src.ID == "" {
		resp.Diagnostics.AddError(moveErrorSummary, "The source state must have a non-empty id.")
		return
	}
	if len(src.Metadata) != 1 {
		resp.Diagnostics.AddError(moveErrorSummary,
			fmt.Sprintf("Expected exactly 1 metadata element in the source state, got %d.", len(src.Metadata)))
		return
	}
	if len(src.Spec) != 1 {
		resp.Diagnostics.AddError(moveErrorSummary,
			fmt.Sprintf("Expected exactly 1 spec element in the source state, got %d.", len(src.Spec)))
		return
	}
	spec := src.Spec[0]
	if len(spec.Metric) > 0 || len(spec.Behavior) > 0 {
		resp.Diagnostics.AddError(moveErrorSummary,
			"The source state sets spec.metric or spec.behavior, which kubernetes_horizontal_pod_autoscaler_v1 does not support. "+
				"Manage this autoscaler with kubernetes_horizontal_pod_autoscaler_v2 instead.")
		return
	}
	if len(spec.ScaleTargetRef) != 1 {
		resp.Diagnostics.AddError(moveErrorSummary,
			fmt.Sprintf("Expected exactly 1 spec.scale_target_ref element in the source state, got %d.", len(spec.ScaleTargetRef)))
		return
	}

	m := src.Metadata[0]
	annotations, diags := common.SDKv2MapToFramework(ctx, m.Annotations)
	resp.Diagnostics.Append(diags...)
	labels, diags := common.SDKv2MapToFramework(ctx, m.Labels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ref := spec.ScaleTargetRef[0]
	target := HorizontalPodAutoscalerV1Model{
		ID: types.StringValue(src.ID),
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Annotations:     annotations,
					Generation:      types.Int64Value(m.Generation),
					Labels:          labels,
					Name:            types.StringValue(m.Name),
					ResourceVersion: types.StringValue(m.ResourceVersion),
					UID:             types.StringValue(m.UID),
				},
				// SDKv2 stores unset strings as "". The framework uses null, and a moved
				// state is not upgraded again, so normalise here.
				GenerateName: sdkv2StringToFramework(m.GenerateName),
			},
			Namespace: types.StringValue(m.Namespace),
		}},
		Spec: []HorizontalPodAutoscalerV1Spec{{
			MaxReplicas:                    types.Int64Value(spec.MaxReplicas),
			MinReplicas:                    types.Int64Value(spec.MinReplicas),
			TargetCPUUtilizationPercentage: sdkv2Int64ToFramework(spec.TargetCPUUtilizationPercentage),
			ScaleTargetRef: []ScaleTargetRef{{
				APIVersion: sdkv2StringToFramework(ref.APIVersion),
				Kind:       types.StringValue(ref.Kind),
				Name:       types.StringValue(ref.Name),
			}},
		}},
	}
	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &target)...)
}

func sdkv2StringToFramework(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

func sdkv2Int64ToFramework(v int64) types.Int64 {
	if v == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(v)
}
