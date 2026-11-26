// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	autoscalingv2api "k8s.io/api/autoscaling/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
	typedautoscalingv2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
)

func (r *HorizontalPodAutoscalerV2) client(diags *diag.Diagnostics, namespace string) (typedautoscalingv2.HorizontalPodAutoscalerInterface, kubernetes.MetadataFilters) {
	clients, filters, d := common.ResolveSDKv2Meta(r.SDKv2Meta)
	diags.Append(d...)
	if diags.HasError() {
		return nil, nil
	}
	conn, err := clients.MainClientset()
	if err != nil {
		diags.AddError("kubernetes client error", err.Error())
		return nil, nil
	}
	return conn.AutoscalingV2().HorizontalPodAutoscalers(namespace), filters
}

func (r *HorizontalPodAutoscalerV2) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HorizontalPodAutoscalerV2Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	spec, diags := expandSpec(ctx, plan.Spec[0])
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, filters := r.client(&resp.Diagnostics, metadata.Namespace)
	if resp.Diagnostics.HasError() {
		return
	}
	hpa := &autoscalingv2api.HorizontalPodAutoscaler{ObjectMeta: metadata, Spec: spec}
	out, err := client.Create(ctx, hpa, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"error creating HorizontalPodAutoscaler",
			fmt.Sprintf("Failed to create horizontal pod autoscaler %q in namespace %q: %s", metadata.Name, metadata.Namespace, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(writeResult(ctx, &resp.State, req.Plan, plan, out, filters)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, hpaIdentity(out.Namespace, out.Name))...)
}

func (r *HorizontalPodAutoscalerV2) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HorizontalPodAutoscalerV2Model
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

	client, filters := r.client(&resp.Diagnostics, namespace)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := client.Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// Terraform before 1.12 does not send identity on refresh, but the framework
		// still requires it when Read removes a missing resource.
		if resp.Identity.Raw.IsNull() {
			resp.Diagnostics.Append(resp.Identity.Set(ctx, hpaIdentity(namespace, name))...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error reading HorizontalPodAutoscaler",
			fmt.Sprintf("Failed to read horizontal pod autoscaler %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	model, diags := flattenModel(ctx, out, state, filters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, hpaIdentity(out.Namespace, out.Name))...)
}

func (r *HorizontalPodAutoscalerV2) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state HorizontalPodAutoscalerV2Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel)
	specOps, diags := specPatchOps(ctx, state.Spec[0], plan.Spec[0])
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops = append(ops, specOps...)

	client, filters := r.client(&resp.Diagnostics, namespace)
	if resp.Diagnostics.HasError() {
		return
	}
	var out *autoscalingv2api.HorizontalPodAutoscaler
	if len(ops) == 0 {
		out, err = client.Get(ctx, name, metav1.GetOptions{})
	} else {
		var data []byte
		data, err = ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("failed to marshal patch", err.Error())
			return
		}
		out, err = client.Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"error updating HorizontalPodAutoscaler",
			fmt.Sprintf("Failed to update horizontal pod autoscaler %q in namespace %q: %s", name, namespace, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(writeResult(ctx, &resp.State, req.Plan, plan, out, filters)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, hpaIdentity(out.Namespace, out.Name))...)
}

func (r *HorizontalPodAutoscalerV2) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state HorizontalPodAutoscalerV2Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("invalid resource ID", err.Error())
		return
	}

	client, _ := r.client(&resp.Diagnostics, namespace)
	if resp.Diagnostics.HasError() {
		return
	}
	err = client.Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"error deleting HorizontalPodAutoscaler",
			fmt.Sprintf("Failed to delete horizontal pod autoscaler %q in namespace %q: %s", name, namespace, err.Error()),
		)
	}
}

func (r *HorizontalPodAutoscalerV2) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := req.ID
	if id == "" {
		var identity common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.Kind.ValueString() != hpaKind || identity.APIVersion.ValueString() != hpaAPIVersion {
			resp.Diagnostics.AddError("invalid identity import",
				fmt.Sprintf("Expected kind %q and api_version %q, got kind %q and api_version %q.",
					hpaKind, hpaAPIVersion, identity.Kind.ValueString(), identity.APIVersion.ValueString()))
			return
		}
		namespace := identity.Namespace.ValueString()
		if namespace == "" {
			namespace = metav1.NamespaceDefault
		}
		id = kubernetes.BuildId(metav1.ObjectMeta{Namespace: namespace, Name: identity.Name.ValueString()})
	}

	// Read validates the namespaced ID and populates state and identity.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// flattenModel converts the API object. prior supplies the values flattenSpec cannot read
// back and the metadata keys to keep.
func flattenModel(ctx context.Context, out *autoscalingv2api.HorizontalPodAutoscaler, prior HorizontalPodAutoscalerV2Model, filters kubernetes.MetadataFilters) (HorizontalPodAutoscalerV2Model, diag.Diagnostics) {
	var diags diag.Diagnostics
	metadata, d := common.FlattenNamespacedMetadata(ctx, out.ObjectMeta, prior.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	diags.Append(d...)

	var priorSpec *specModel
	if len(prior.Spec) > 0 {
		priorSpec = &prior.Spec[0]
	}
	spec, d := flattenSpec(ctx, out.Spec, priorSpec)
	diags.Append(d...)

	return HorizontalPodAutoscalerV2Model{
		ID:       types.StringValue(kubernetes.BuildId(out.ObjectMeta)),
		Metadata: metadata,
		Spec:     []specModel{spec},
	}, diags
}

// writeResult keeps known planned values and resolves the unknown ones, such as defaulted
// metrics and behavior, from the object Kubernetes returned.
func writeResult(ctx context.Context, state *tfsdk.State, planned tfsdk.Plan, plan HorizontalPodAutoscalerV2Model, out *autoscalingv2api.HorizontalPodAutoscaler, filters kubernetes.MetadataFilters) diag.Diagnostics {
	return common.SetWriteResult(ctx, state, planned, func(actual *tfsdk.State) diag.Diagnostics {
		model, diags := flattenModel(ctx, out, plan, filters)
		if diags.HasError() {
			return diags
		}
		diags.Append(actual.Set(ctx, &model)...)
		return diags
	})
}

// specPatchOps produces one operation per changed field. metrics and behavior use add,
// which replaces an existing member and also works when Kubernetes has no behavior.
func specPatchOps(ctx context.Context, state, plan specModel) (kubernetes.PatchOperations, diag.Diagnostics) {
	var diags diag.Diagnostics
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
	if !plan.Metric.IsNull() && !plan.Metric.IsUnknown() && !plan.Metric.Equal(state.Metric) {
		metrics, d := expandMetrics(ctx, plan.Metric)
		diags.Append(d...)
		ops = append(ops, &kubernetes.AddOperation{
			Path:  "/spec/metrics",
			Value: metrics,
		})
	}
	if !plan.Behavior.IsNull() && !plan.Behavior.IsUnknown() && !plan.Behavior.Equal(state.Behavior) {
		behavior, d := expandBehavior(ctx, plan.Behavior)
		diags.Append(d...)
		ops = append(ops, &kubernetes.AddOperation{
			Path:  "/spec/behavior",
			Value: behavior,
		})
	}
	return ops, diags
}

// scaleTargetRefEqual treats a null and an empty api_version alike, as SDKv2 did.
func scaleTargetRefEqual(a, b scaleTargetRef) bool {
	return a.APIVersion.ValueString() == b.APIVersion.ValueString() &&
		a.Kind.Equal(b.Kind) && a.Name.Equal(b.Name)
}
