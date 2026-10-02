// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podtemplate"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	defaultCreateTimeout = 10 * time.Minute
	defaultUpdateTimeout = 10 * time.Minute
	defaultDeleteTimeout = 10 * time.Minute
)

type deploymentSpecModel struct {
	MinReadySeconds         types.Int64               `tfsdk:"min_ready_seconds"`
	Paused                  types.Bool                `tfsdk:"paused"`
	ProgressDeadlineSeconds types.Int64               `tfsdk:"progress_deadline_seconds"`
	Replicas                types.String              `tfsdk:"replicas"`
	RevisionHistoryLimit    types.Int64               `tfsdk:"revision_history_limit"`
	Selector                []deploymentSelectorModel `tfsdk:"selector"`
	Strategy                types.List                `tfsdk:"strategy"`
	Template                []deploymentTemplateModel `tfsdk:"template"`
}

type deploymentTemplateModel struct {
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     types.List                       `tfsdk:"spec"`
}

type deploymentSelectorModel = LabelSelectorModel

type deploymentSelectorRequirementModel = LabelSelectorRequirementModel

type deploymentStrategyModel struct {
	Type          types.String `tfsdk:"type"`
	RollingUpdate types.List   `tfsdk:"rolling_update"`
}

type deploymentRollingUpdateModel struct {
	MaxSurge       types.String `tfsdk:"max_surge"`
	MaxUnavailable types.String `tfsdk:"max_unavailable"`
}

func (d *DeploymentV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DeploymentV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, timeoutDiags := plan.Timeouts.Create(ctx, defaultCreateTimeout)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	clients, filters, metaDiags := d.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	spec, diags := expandDeploymentSpec(ctx, plan.Spec, path.Root("spec"))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating deployment", map[string]any{
		"name":      metadata.Name,
		"namespace": metadata.Namespace,
	})
	out, err := conn.AppsV1().Deployments(metadata.Namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metadata,
		Spec:       *spec,
	}, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsInvalid(err) {
			// Preserve SDKv2's unwrapped Kubernetes validation diagnostic.
			resp.Diagnostics.AddError(err.Error(), "")
		} else {
			resp.Diagnostics.AddError("Error creating deployment", err.Error())
		}
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), kubernetes.BuildId(out.ObjectMeta))...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(deploymentAPIVersion),
			Kind:       types.StringValue(deploymentKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
	createdState, stateDiags := deploymentModelFromObject(ctx, out, plan, filters)
	resp.Diagnostics.Append(stateDiags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &createdState)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.WaitForRollout.ValueBool() {
		return
	}
	err = retry.RetryContext(ctx, timeout, kubernetes.WaitForDeploymentReplicasForFramework(ctx, conn, out.Namespace, out.Name))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for deployment rollout",
			fmt.Sprintf("Deployment %q was created but rollout did not complete: %s", createdState.ID.ValueString(), err),
		)
	}
}

func (d *DeploymentV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DeploymentV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, filters, metaDiags := d.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}
	out, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading deployment", err.Error())
		return
	}

	refreshed, diags := deploymentModelFromObject(ctx, out, state, filters)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(deploymentAPIVersion),
			Kind:       types.StringValue(deploymentKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
}

func (d *DeploymentV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DeploymentV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state DeploymentV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, timeoutDiags := plan.Timeouts.Update(ctx, defaultUpdateTimeout)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	clients, filters, metaDiags := d.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}

	live, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading deployment during update", err.Error())
		return
	}

	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid deployment metadata", "Expected exactly one metadata block in state and plan.")
		return
	}
	metadataOps := deploymentMetadataPatchOps(state, plan, live.ObjectMeta)
	out := live
	if len(metadataOps) > 0 {
		data, err := metadataOps.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Error marshalling deployment metadata patch", err.Error())
			return
		}
		out, err = conn.AppsV1().Deployments(namespace).Patch(ctx, name, k8types.JSONPatchType, data, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error patching deployment metadata", err.Error())
			return
		}
	}

	if !plan.Spec.Equal(state.Spec) {
		original, diags := expandDeploymentSpec(ctx, state.Spec, path.Root("spec"))
		resp.Diagnostics.Append(diags...)
		desired, diags := expandDeploymentSpec(ctx, plan.Spec, path.Root("spec"))
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		patch, err := deploymentSpecPatch(*original, *desired, out.Spec)
		if err != nil {
			resp.Diagnostics.AddError("Error creating deployment spec patch", err.Error())
			return
		}
		if string(patch) != "{}" {
			out, err = conn.AppsV1().Deployments(namespace).Patch(ctx, name, k8types.StrategicMergePatchType, patch, metav1.PatchOptions{})
			if err != nil {
				resp.Diagnostics.AddError("Error patching deployment spec", err.Error())
				return
			}
		}
	}

	resp.Diagnostics.Append(resp.Identity.Set(ctx, common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(deploymentAPIVersion),
			Kind:       types.StringValue(deploymentKind),
			Name:       types.StringValue(out.Name),
		},
		Namespace: types.StringValue(out.Namespace),
	})...)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedState, stateDiags := deploymentModelFromObject(ctx, out, plan, filters)
	resp.Diagnostics.Append(stateDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.WaitForRollout.ValueBool() {
		err = retry.RetryContext(ctx, timeout, kubernetes.WaitForDeploymentReplicasForFramework(ctx, conn, out.Namespace, out.Name))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error waiting for deployment rollout",
				fmt.Sprintf("Deployment %q was updated but rollout did not complete: %s", plan.ID.ValueString(), err),
			)
			return
		}
	}

	out, err = conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading deployment after update", err.Error())
		return
	}
	updatedState, stateDiags = deploymentModelFromObject(ctx, out, plan, filters)
	resp.Diagnostics.Append(stateDiags...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
	}
}

func (d *DeploymentV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DeploymentV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	clients, _, metaDiags := d.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid deployment id", err.Error())
		return
	}
	deleteTimeout, dTimeout := state.Timeouts.Delete(ctx, defaultDeleteTimeout)
	resp.Diagnostics.Append(dTimeout...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	err = conn.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting deployment", err.Error())
		return
	}
	err = retry.RetryContext(ctx, deleteTimeout, func() *retry.RetryError {
		_, err := conn.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("deployment (%s) still exists", state.ID.ValueString()))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for deployment delete", err.Error())
	}
}

func expandDeploymentSpec(ctx context.Context, value types.List, at path.Path) (*appsv1.DeploymentSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() || len(value.Elements()) != 1 {
		diags.AddAttributeError(at, "Invalid deployment specification", "Exactly one spec block is required.")
		return nil, diags
	}

	var models []deploymentSpecModel
	diags.Append(value.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	if len(models) != 1 {
		diags.AddAttributeError(at, "Invalid deployment specification", "Exactly one spec block is required.")
		return nil, diags
	}

	input := models[0]
	out := &appsv1.DeploymentSpec{
		MinReadySeconds: int32(input.MinReadySeconds.ValueInt64()),
		Paused:          input.Paused.ValueBool(),
	}
	out.ProgressDeadlineSeconds = ptr.To(int32(input.ProgressDeadlineSeconds.ValueInt64()))
	out.RevisionHistoryLimit = ptr.To(int32(input.RevisionHistoryLimit.ValueInt64()))

	if !input.Replicas.IsNull() && !input.Replicas.IsUnknown() && input.Replicas.ValueString() != "" {
		replicas, err := strconvParseInt32(input.Replicas.ValueString())
		if err != nil {
			diags.AddAttributeError(at.AtName("replicas"), "Invalid replicas value", err.Error())
			return nil, diags
		}
		out.Replicas = ptr.To(replicas)
	}

	if len(input.Selector) > 0 {
		selector, d := expandSelector(ctx, input.Selector[0], at.AtName("selector").AtListIndex(0))
		diags.Append(d...)
		out.Selector = selector
	}

	strategy, d := expandDeploymentStrategy(ctx, input.Strategy, at.AtName("strategy"))
	diags.Append(d...)
	out.Strategy = strategy

	if len(input.Template) != 1 {
		diags.AddAttributeError(at.AtName("template"), "Invalid deployment template", "Exactly one template block is required.")
		return nil, diags
	}
	template := input.Template[0]
	metadata, d := common.ExpandNamespacedMetadata(ctx, template.Metadata)
	diags.Append(d...)
	spec, d := podtemplate.ExpandSpec(ctx, template.Spec, at.AtName("template").AtListIndex(0).AtName("spec"))
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	out.Template = corev1.PodTemplateSpec{
		ObjectMeta: metadata,
		Spec:       spec,
	}
	return out, diags
}

func flattenDeploymentSpec(ctx context.Context, spec appsv1.DeploymentSpec, baseline types.List, at path.Path) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	model := deploymentSpecModel{
		MinReadySeconds:         types.Int64Value(int64(spec.MinReadySeconds)),
		Paused:                  types.BoolValue(spec.Paused),
		ProgressDeadlineSeconds: types.Int64Value(600),
		RevisionHistoryLimit:    types.Int64Value(10),
		Replicas:                types.StringNull(),
	}
	if spec.ProgressDeadlineSeconds != nil {
		model.ProgressDeadlineSeconds = types.Int64Value(int64(*spec.ProgressDeadlineSeconds))
	}
	if spec.RevisionHistoryLimit != nil {
		model.RevisionHistoryLimit = types.Int64Value(int64(*spec.RevisionHistoryLimit))
	}
	if spec.Replicas != nil {
		model.Replicas = types.StringValue(fmt.Sprintf("%d", *spec.Replicas))
	}
	strategyValue, d := flattenDeploymentStrategy(ctx, spec.Strategy)
	diags.Append(d...)
	model.Strategy = strategyValue
	if diags.HasError() {
		return types.ListNull(deploymentSpecListType().ElemType), diags
	}

	var priorTemplateSpec types.List
	var priorTemplateMetadata []common.NamespacedMetadataModel
	var priorSelector []LabelSelectorModel
	if !baseline.IsNull() && !baseline.IsUnknown() {
		var priorSpecs []deploymentSpecModel
		diags.Append(baseline.ElementsAs(ctx, &priorSpecs, false)...)
		if diags.HasError() {
			return types.ListNull(deploymentSpecListType().ElemType), diags
		}
		if len(priorSpecs) == 1 {
			priorSelector = priorSpecs[0].Selector
		}
		if len(priorSpecs) == 1 && len(priorSpecs[0].Template) == 1 {
			priorTemplateSpec = priorSpecs[0].Template[0].Spec
			priorTemplateMetadata = priorSpecs[0].Template[0].Metadata
			replicas := priorSpecs[0].Replicas
			if !replicas.IsNull() && !replicas.IsUnknown() {
				if replicas.ValueString() == "" {
					model.Replicas = replicas
				} else if parsed, err := strconvParseInt32(replicas.ValueString()); err == nil && spec.Replicas != nil && parsed == *spec.Replicas {
					model.Replicas = replicas
				}
			}
		}
	}
	model.Selector, d = flattenWorkloadSelector(ctx, spec.Selector, priorSelector)
	diags.Append(d...)
	if priorTemplateSpec.IsNull() || priorTemplateSpec.IsUnknown() {
		priorTemplateSpec = templateSpecNull()
	}
	templateSpec, d := podtemplate.FlattenSpec(ctx, spec.Template.Spec, priorTemplateSpec, at.AtName("template").AtListIndex(0).AtName("spec"))
	diags.Append(d...)
	templateMetadata, d := flattenTemplateMetadata(ctx, spec.Template.ObjectMeta, priorTemplateMetadata)
	diags.Append(d...)
	model.Template = []deploymentTemplateModel{{
		Metadata: templateMetadata,
		Spec:     templateSpec,
	}}

	lt := deploymentSpecListType()
	value, dValue := types.ListValueFrom(ctx, lt.ElemType, []deploymentSpecModel{model})
	diags.Append(dValue...)
	return value, diags
}

func expandSelector(ctx context.Context, in deploymentSelectorModel, at path.Path) (*metav1.LabelSelector, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &metav1.LabelSelector{}
	if !in.MatchLabels.IsNull() && !in.MatchLabels.IsUnknown() {
		out.MatchLabels = map[string]string{}
		diags.Append(in.MatchLabels.ElementsAs(ctx, &out.MatchLabels, false)...)
	}
	if len(in.MatchExpressions) > 0 {
		out.MatchExpressions = make([]metav1.LabelSelectorRequirement, len(in.MatchExpressions))
		for i, req := range in.MatchExpressions {
			out.MatchExpressions[i].Key = req.Key.ValueString()
			out.MatchExpressions[i].Operator = metav1.LabelSelectorOperator(req.Operator.ValueString())
			if !req.Values.IsNull() && !req.Values.IsUnknown() {
				diags.Append(req.Values.ElementsAs(ctx, &out.MatchExpressions[i].Values, false)...)
			}
		}
	}
	if diags.HasError() {
		diags.AddAttributeError(at, "Invalid selector", "Unable to decode selector values.")
	}
	return out, diags
}

func expandDeploymentStrategy(ctx context.Context, value types.List, at path.Path) (appsv1.DeploymentStrategy, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
	}
	if value.IsNull() || value.IsUnknown() || len(value.Elements()) == 0 {
		return result, diags
	}
	var in []deploymentStrategyModel
	diags.Append(value.ElementsAs(ctx, &in, false)...)
	if diags.HasError() || len(in) == 0 {
		return result, diags
	}
	if !in[0].Type.IsNull() && !in[0].Type.IsUnknown() && in[0].Type.ValueString() != "" {
		result.Type = appsv1.DeploymentStrategyType(in[0].Type.ValueString())
	}
	if !in[0].RollingUpdate.IsNull() && !in[0].RollingUpdate.IsUnknown() && len(in[0].RollingUpdate.Elements()) > 0 {
		var updates []deploymentRollingUpdateModel
		diags.Append(in[0].RollingUpdate.ElementsAs(ctx, &updates, false)...)
		if len(updates) > 0 {
			update := &appsv1.RollingUpdateDeployment{}
			if !updates[0].MaxSurge.IsNull() && !updates[0].MaxSurge.IsUnknown() {
				value := intstr.Parse(updates[0].MaxSurge.ValueString())
				update.MaxSurge = &value
			}
			if !updates[0].MaxUnavailable.IsNull() && !updates[0].MaxUnavailable.IsUnknown() {
				value := intstr.Parse(updates[0].MaxUnavailable.ValueString())
				update.MaxUnavailable = &value
			}
			result.RollingUpdate = update
		}
	}
	if result.Type == appsv1.RecreateDeploymentStrategyType {
		result.RollingUpdate = nil
	} else if result.RollingUpdate == nil {
		defaults := &appsv1.RollingUpdateDeployment{}
		maxSurge := intstr.Parse("25%")
		maxUnavailable := intstr.Parse("25%")
		defaults.MaxSurge = &maxSurge
		defaults.MaxUnavailable = &maxUnavailable
		result.RollingUpdate = defaults
	}
	if diags.HasError() {
		diags.AddAttributeError(at, "Invalid strategy", "Unable to decode strategy values.")
	}
	return result, diags
}

func flattenDeploymentStrategy(ctx context.Context, strategy appsv1.DeploymentStrategy) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if strategy.Type == "" {
		strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
	}
	var rolling types.List
	rollingType := strategyObjectRollingUpdateListType()
	if strategy.RollingUpdate == nil {
		rolling = types.ListNull(rollingType.ElemType)
	} else {
		maxSurge := ""
		maxUnavailable := ""
		if strategy.RollingUpdate.MaxSurge != nil {
			maxSurge = strategy.RollingUpdate.MaxSurge.String()
		}
		if strategy.RollingUpdate.MaxUnavailable != nil {
			maxUnavailable = strategy.RollingUpdate.MaxUnavailable.String()
		}
		object, d := types.ObjectValue(strategyObjectRollingUpdateElementType().AttrTypes, map[string]attr.Value{
			"max_surge":       types.StringValue(maxSurge),
			"max_unavailable": types.StringValue(maxUnavailable),
		})
		diags.Append(d...)
		rolling, d = types.ListValue(rollingType.ElemType, []attr.Value{object})
		diags.Append(d...)
	}

	strategyObject, d := types.ObjectValue(deploymentStrategyObjectType().AttrTypes, map[string]attr.Value{
		"type":           types.StringValue(string(strategy.Type)),
		"rolling_update": rolling,
	})
	diags.Append(d...)
	value, d := types.ListValue(deploymentStrategyObjectType(), []attr.Value{strategyObject})
	diags.Append(d...)
	return value, diags
}

func flattenTemplateMetadata(ctx context.Context, meta metav1.ObjectMeta, prior []common.NamespacedMetadataModel) ([]common.NamespacedMetadataModel, diag.Diagnostics) {
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
				Annotations:     types.MapNull(types.StringType),
				Labels:          types.MapNull(types.StringType),
			},
			GenerateName: types.StringNull(),
		},
		Namespace: types.StringNull(),
	}

	annotations, d := flattenTemplateMetadataMap(ctx, meta.Annotations, previous.Annotations, len(prior) == 0)
	labels, d2 := flattenTemplateMetadataMap(ctx, meta.Labels, previous.Labels, len(prior) == 0)
	result.Annotations = annotations
	result.Labels = labels
	if meta.GenerateName != "" {
		result.GenerateName = types.StringValue(meta.GenerateName)
	}
	if meta.Namespace != "" || previous.Namespace.Equal(types.StringValue("")) {
		result.Namespace = types.StringValue(meta.Namespace)
	}
	var diags diag.Diagnostics
	diags.Append(d...)
	diags.Append(d2...)
	return []common.NamespacedMetadataModel{result}, diags
}

func flattenTemplateMetadataMap(ctx context.Context, value map[string]string, prior types.Map, importing bool) (types.Map, diag.Diagnostics) {
	if !importing && !prior.IsNull() && !prior.IsUnknown() {
		managed := make(map[string]string, len(prior.Elements()))
		for key := range prior.Elements() {
			if current, ok := value[key]; ok {
				managed[key] = current
			}
		}
		value = managed
	}
	if len(value) == 0 && prior.IsNull() {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, value)
}

func templateSpecNull() types.List {
	blockType := podtemplate.SpecBlock(podtemplate.Options{RestartPolicyAlways: true}).Type().(types.ListType)
	return types.ListNull(blockType.ElemType)
}

func deploymentSpecListType() types.ListType {
	return deploymentSpecBlock().Type().(types.ListType)
}

func strategyObjectRollingUpdateListType() types.ListType {
	return deploymentStrategyObjectType().AttrTypes["rolling_update"].(types.ListType)
}

func strategyObjectRollingUpdateElementType() types.ObjectType {
	return strategyObjectRollingUpdateListType().ElemType.(types.ObjectType)
}

func strategyTypeFromSpec(ctx context.Context, value types.List) string {
	if value.IsNull() || value.IsUnknown() {
		return "RollingUpdate"
	}
	var specs []deploymentSpecModel
	if diags := value.ElementsAs(ctx, &specs, false); diags.HasError() || len(specs) == 0 {
		return "RollingUpdate"
	}
	return strategyTypeFromList(ctx, specs[0].Strategy)
}

func strategyTypeFromList(ctx context.Context, value types.List) string {
	if value.IsNull() || value.IsUnknown() || len(value.Elements()) == 0 {
		return "RollingUpdate"
	}
	var strategy []deploymentStrategyModel
	if diags := value.ElementsAs(ctx, &strategy, false); diags.HasError() || len(strategy) == 0 {
		return "RollingUpdate"
	}
	if strategy[0].Type.IsNull() || strategy[0].Type.IsUnknown() || strategy[0].Type.ValueString() == "" {
		return "RollingUpdate"
	}
	return strategy[0].Type.ValueString()
}

func deploymentMetadataPatchOps(state, plan DeploymentV1Model, live metav1.ObjectMeta) kubernetes.PatchOperations {
	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 {
		return nil
	}
	return common.MetadataPatchOpsAgainstLive("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel, live)
}

func deploymentModelFromObject(ctx context.Context, object *appsv1.Deployment, baseline DeploymentV1Model, filters kubernetes.MetadataFilters) (DeploymentV1Model, diag.Diagnostics) {
	var diags diag.Diagnostics
	metadata, metadataDiags := common.FlattenNamespacedMetadata(
		ctx,
		object.ObjectMeta,
		baseline.Metadata,
		filters.GetIgnoreAnnotations(),
		filters.GetIgnoreLabels(),
	)
	diags.Append(metadataDiags...)
	spec, specDiags := flattenDeploymentSpec(ctx, object.Spec, baseline.Spec, path.Root("spec"))
	diags.Append(specDiags...)

	baseline.ID = types.StringValue(kubernetes.BuildId(object.ObjectMeta))
	baseline.Metadata = metadata
	baseline.Spec = spec
	if baseline.WaitForRollout.IsNull() || baseline.WaitForRollout.IsUnknown() {
		baseline.WaitForRollout = types.BoolValue(true)
	}
	return baseline, diags
}

func deploymentSpecPatch(original, modified, current appsv1.DeploymentSpec) ([]byte, error) {
	if modified.Replicas == nil {
		original.Replicas = nil
	}
	originalJSON, err := json.Marshal(appsv1.Deployment{Spec: original})
	if err != nil {
		return nil, err
	}
	modifiedJSON, err := json.Marshal(appsv1.Deployment{Spec: modified})
	if err != nil {
		return nil, err
	}
	currentJSON, err := json.Marshal(appsv1.Deployment{Spec: current})
	if err != nil {
		return nil, err
	}
	return common.TwoWayStrategicMergePatch(originalJSON, modifiedJSON, currentJSON, appsv1.Deployment{})
}

func strconvParseInt32(value string) (int32, error) {
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(parsed), nil
}
