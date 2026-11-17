// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	autoscalingv1api "k8s.io/api/autoscaling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func (r *HorizontalPodAutoscalerV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HorizontalPodAutoscalerV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hpa := &autoscalingv1api.HorizontalPodAutoscaler{
		ObjectMeta: metadata,
		Spec:       expandSpec(plan.Spec[0]),
	}
	out, err := conn.AutoscalingV1().HorizontalPodAutoscalers(metadata.Namespace).Create(ctx, hpa, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating HorizontalPodAutoscaler",
			fmt.Sprintf("Failed to create horizontal pod autoscaler %q in namespace %q: %s", metadata.Name, metadata.Namespace, err.Error()),
		)
		return
	}

	populateFromResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HorizontalPodAutoscalerV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HorizontalPodAutoscalerV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// IdParts checks separators, but permits an empty namespace or name.
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil || namespace == "" || name == "" {
		resp.Diagnostics.AddError("invalid resource ID",
			fmt.Sprintf("Expected format: namespace/name with a non-empty namespace and name, got: %s", state.ID.ValueString()))
		return
	}

	clients, filters, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	hpa, err := conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"error reading HorizontalPodAutoscaler",
			fmt.Sprintf("Failed to read horizontal pod autoscaler %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	state.ID = types.StringValue(kubernetes.BuildId(hpa.ObjectMeta))
	metadata, diags := common.FlattenNamespacedMetadata(ctx, hpa.ObjectMeta, state.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Metadata = metadata
	state.Spec = []HorizontalPodAutoscalerV1Spec{flattenSpec(hpa.Spec)}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HorizontalPodAutoscalerV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state HorizontalPodAutoscalerV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel)
	ops = append(ops, specPatchOps(state.Spec[0], plan.Spec[0])...)

	var out *autoscalingv1api.HorizontalPodAutoscaler
	if len(ops) == 0 {
		out, err = conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		var data []byte
		data, err = ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("failed to marshal patch", err.Error())
			return
		}
		out, err = conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating HorizontalPodAutoscaler",
			fmt.Sprintf("Failed to update horizontal pod autoscaler %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	populateFromResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HorizontalPodAutoscalerV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HorizontalPodAutoscalerV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, diags := common.ResolveSDKv2Meta(r.SDKv2Meta)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	err = conn.AutoscalingV1().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting HorizontalPodAutoscaler",
			fmt.Sprintf("Failed to delete horizontal pod autoscaler %q in namespace %q: %s", name, namespace, err.Error()),
		)
	}
}

// populateFromResponse keeps planned values and fills only server-assigned ones. Labels and
// annotations stay as planned because the API server may add keys; filtering them is Read's job.
func populateFromResponse(plan *HorizontalPodAutoscalerV1Model, out *autoscalingv1api.HorizontalPodAutoscaler) {
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	common.PopulateNamespacedMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)
	if plan.Spec[0].TargetCPUUtilizationPercentage.IsUnknown() {
		plan.Spec[0].TargetCPUUtilizationPercentage = int32PtrToFramework(out.Spec.TargetCPUUtilizationPercentage)
	}
}

// expandSpec mirrors SDKv2's expandHorizontalPodAutoscalerSpec, which sent min_replicas and
// target_cpu_utilization_percentage only when positive.
func expandSpec(in HorizontalPodAutoscalerV1Spec) autoscalingv1api.HorizontalPodAutoscalerSpec {
	spec := autoscalingv1api.HorizontalPodAutoscalerSpec{
		MaxReplicas: int32(in.MaxReplicas.ValueInt64()),
	}
	if v := in.MinReplicas.ValueInt64(); v > 0 {
		spec.MinReplicas = ptr.To(int32(v))
	}
	if len(in.ScaleTargetRef) > 0 {
		spec.ScaleTargetRef = expandScaleTargetRef(in.ScaleTargetRef[0])
	}
	if v := in.TargetCPUUtilizationPercentage.ValueInt64(); v > 0 {
		spec.TargetCPUUtilizationPercentage = ptr.To(int32(v))
	}
	return spec
}

func expandScaleTargetRef(in ScaleTargetRef) autoscalingv1api.CrossVersionObjectReference {
	return autoscalingv1api.CrossVersionObjectReference{
		APIVersion: in.APIVersion.ValueString(),
		Kind:       in.Kind.ValueString(),
		Name:       in.Name.ValueString(),
	}
}

func flattenSpec(in autoscalingv1api.HorizontalPodAutoscalerSpec) HorizontalPodAutoscalerV1Spec {
	return HorizontalPodAutoscalerV1Spec{
		MaxReplicas:                    types.Int64Value(int64(in.MaxReplicas)),
		MinReplicas:                    int32PtrToFramework(in.MinReplicas),
		TargetCPUUtilizationPercentage: int32PtrToFramework(in.TargetCPUUtilizationPercentage),
		ScaleTargetRef: []ScaleTargetRef{{
			APIVersion: sdkv2StringToFramework(in.ScaleTargetRef.APIVersion),
			Kind:       types.StringValue(in.ScaleTargetRef.Kind),
			Name:       types.StringValue(in.ScaleTargetRef.Name),
		}},
	}
}

// specPatchOps mirrors SDKv2's patchHorizontalPodAutoscalerSpec: one replace per changed field.
func specPatchOps(state, plan HorizontalPodAutoscalerV1Spec) kubernetes.PatchOperations {
	ops := kubernetes.PatchOperations{}
	if !plan.MaxReplicas.Equal(state.MaxReplicas) {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/spec/maxReplicas",
			Value: plan.MaxReplicas.ValueInt64(),
		})
	}
	if !plan.MinReplicas.IsUnknown() && !plan.MinReplicas.Equal(state.MinReplicas) {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/spec/minReplicas",
			Value: plan.MinReplicas.ValueInt64(),
		})
	}
	if len(plan.ScaleTargetRef) > 0 && (len(state.ScaleTargetRef) == 0 || !scaleTargetRefEqual(plan.ScaleTargetRef[0], state.ScaleTargetRef[0])) {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/spec/scaleTargetRef",
			Value: expandScaleTargetRef(plan.ScaleTargetRef[0]),
		})
	}
	if !plan.TargetCPUUtilizationPercentage.IsUnknown() && !plan.TargetCPUUtilizationPercentage.IsNull() &&
		!plan.TargetCPUUtilizationPercentage.Equal(state.TargetCPUUtilizationPercentage) {
		ops = append(ops, &kubernetes.ReplaceOperation{
			Path:  "/spec/targetCPUUtilizationPercentage",
			Value: plan.TargetCPUUtilizationPercentage.ValueInt64(),
		})
	}
	return ops
}

// scaleTargetRefEqual treats a null and an empty api_version alike, as SDKv2 did.
func scaleTargetRefEqual(a, b ScaleTargetRef) bool {
	return a.APIVersion.ValueString() == b.APIVersion.ValueString() &&
		a.Kind.Equal(b.Kind) && a.Name.Equal(b.Name)
}

func int32PtrToFramework(v *int32) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}
