// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podtemplate"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8Types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
)

func (d *DaemonSetV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DaemonSetV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, timeoutDiags := plan.Timeouts.Create(ctx, defaultDaemonSetCreateTimeout)
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

	metadata, metadataDiags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(metadataDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	spec, specDiags := expandDaemonSetSpecModel(ctx, plan.Spec, daemonSetSpecPath())
	resp.Diagnostics.Append(specDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := conn.AppsV1().DaemonSets(metadata.Namespace).Create(ctx, &appsv1.DaemonSet{
		ObjectMeta: metadata,
		Spec:       spec,
	}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating daemonset", err.Error())
		return
	}

	createdState := d.daemonSetStateFromObject(ctx, plan, created, filters, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &createdState)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, daemonSetIdentity(created.Namespace, created.Name))...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.WaitForRollout.ValueBool() {
		err = retry.RetryContext(ctx, timeout, kubernetes.WaitForDaemonSetPodsForFramework(ctx, conn, created.Namespace, created.Name))
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for daemonset rollout", err.Error())
			return
		}
	}

	fresh, exists := d.readDaemonSetState(ctx, createdState, filters, conn, &resp.Diagnostics)
	if !exists || resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &fresh)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, daemonSetIdentity(fresh.Metadata[0].Namespace.ValueString(), fresh.Metadata[0].Name.ValueString()))...)
}

func (d *DaemonSetV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DaemonSetV1Model
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

	fresh, exists := d.readDaemonSetState(ctx, state, filters, conn, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &fresh)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, daemonSetIdentity(fresh.Metadata[0].Namespace.ValueString(), fresh.Metadata[0].Name.ValueString()))...)
}

func (d *DaemonSetV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DaemonSetV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state DaemonSetV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, timeoutDiags := plan.Timeouts.Update(ctx, defaultDaemonSetUpdateTimeout)
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
		resp.Diagnostics.AddError("Invalid daemonset ID", err.Error())
		return
	}
	live, err := conn.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading daemonset before update", err.Error())
		return
	}
	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid daemonset metadata", "Expected exactly one metadata block in state and plan.")
		return
	}

	ops := daemonSetMetadataPatchOps(state, plan, live.ObjectMeta)
	updated := live
	if len(ops) > 0 {
		data, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Error marshalling daemonset patch", err.Error())
			return
		}
		updated, err = conn.AppsV1().DaemonSets(namespace).Patch(ctx, name, k8Types.JSONPatchType, data, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error updating daemonset", err.Error())
			return
		}
	}

	if !reflect.DeepEqual(plan.Spec, state.Spec) {
		specPatch, patchDiags := daemonSetStrategicSpecPatch(ctx, state.Spec, plan.Spec, &updated.Spec)
		resp.Diagnostics.Append(patchDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(specPatch) > 0 {
			updated, err = conn.AppsV1().DaemonSets(namespace).Patch(ctx, name, k8Types.StrategicMergePatchType, specPatch, metav1.PatchOptions{})
			if err != nil {
				resp.Diagnostics.AddError("Error updating daemonset spec", err.Error())
				return
			}
		}
	}

	updatedState := d.daemonSetStateFromObject(ctx, plan, updated, filters, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &updatedState)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, daemonSetIdentity(updated.Namespace, updated.Name))...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.WaitForRollout.ValueBool() {
		err = retry.RetryContext(ctx, timeout, kubernetes.WaitForDaemonSetPodsForFramework(ctx, conn, namespace, name))
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for daemonset rollout", err.Error())
			return
		}
	}

	fresh, exists := d.readDaemonSetState(ctx, plan, filters, conn, &resp.Diagnostics)
	if !exists || resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &fresh)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, daemonSetIdentity(fresh.Metadata[0].Namespace.ValueString(), fresh.Metadata[0].Name.ValueString()))...)
}

func (d *DaemonSetV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DaemonSetV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, timeoutDiags := state.Timeouts.Delete(ctx, defaultDaemonSetDeleteTimeout)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
		resp.Diagnostics.AddError("Invalid daemonset ID", err.Error())
		return
	}
	err = conn.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting daemonset", err.Error())
		return
	}
	err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		_, err := conn.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return retry.RetryableError(fmt.Errorf("daemonset %s/%s still exists", namespace, name))
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for daemonset deletion", err.Error())
	}
}

func (d *DaemonSetV1) readDaemonSetState(
	ctx context.Context,
	prior DaemonSetV1Model,
	filters kubernetes.MetadataFilters,
	conn *k8sclient.Clientset,
	diags *diag.Diagnostics,
) (DaemonSetV1Model, bool) {
	namespace, name, err := kubernetes.IdParts(prior.ID.ValueString())
	if err != nil {
		diags.AddError("Invalid daemonset ID", err.Error())
		return DaemonSetV1Model{}, false
	}
	current, err := conn.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return DaemonSetV1Model{}, false
	}
	if err != nil {
		diags.AddError("Error reading daemonset", err.Error())
		return DaemonSetV1Model{}, false
	}

	return d.daemonSetStateFromObject(ctx, prior, current, filters, diags), true
}

func (d *DaemonSetV1) daemonSetStateFromObject(
	ctx context.Context,
	prior DaemonSetV1Model,
	current *appsv1.DaemonSet,
	filters kubernetes.MetadataFilters,
	diags *diag.Diagnostics,
) DaemonSetV1Model {
	metadata, metadataDiags := common.FlattenNamespacedMetadata(
		ctx,
		current.ObjectMeta,
		prior.Metadata,
		filters.GetIgnoreAnnotations(),
		filters.GetIgnoreLabels(),
	)
	diags.Append(metadataDiags...)
	spec, specDiags := flattenDaemonSetSpecModel(ctx, current.Spec, prior.Spec)
	diags.Append(specDiags...)
	if diags.HasError() {
		return DaemonSetV1Model{}
	}

	result := prior
	result.ID = types.StringValue(kubernetes.BuildId(current.ObjectMeta))
	result.Metadata = metadata
	result.Spec = spec
	if result.WaitForRollout.IsNull() || result.WaitForRollout.IsUnknown() {
		// SDKv2 import state has no configuration/default pass and therefore
		// records the bool zero value until configuration is applied.
		result.WaitForRollout = types.BoolValue(false)
	}
	return result
}

func expandDaemonSetSpecModel(ctx context.Context, spec []DaemonSetV1SpecModel, at path.Path) (appsv1.DaemonSetSpec, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if len(spec) != 1 {
		diagnostics.AddAttributeError(at, "Invalid daemonset spec", "Exactly one spec block is required.")
		return appsv1.DaemonSetSpec{}, diagnostics
	}

	in := spec[0]
	out := appsv1.DaemonSetSpec{
		MinReadySeconds: int32(in.MinReadySeconds.ValueInt64()),
	}

	if !in.RevisionHistoryLimit.IsNull() && !in.RevisionHistoryLimit.IsUnknown() {
		out.RevisionHistoryLimit = ptr.To(int32(in.RevisionHistoryLimit.ValueInt64()))
	}
	selector, selectorDiags := expandDaemonSetSelector(ctx, in.Selector, at.AtName("selector"))
	diagnostics.Append(selectorDiags...)
	out.Selector = selector
	strategy, strategyDiags := expandDaemonSetStrategyModel(ctx, in.Strategy, at.AtName("strategy"))
	diagnostics.Append(strategyDiags...)
	out.UpdateStrategy = strategy

	if len(in.Template) != 1 {
		diagnostics.AddAttributeError(at.AtName("template"), "Invalid daemonset template", "Exactly one template block is required.")
		return out, diagnostics
	}
	template := in.Template[0]
	templateMetadata, templateMetadataDiags := common.ExpandNamespacedMetadata(ctx, template.Metadata)
	diagnostics.Append(templateMetadataDiags...)
	templateSpec, templateSpecDiags := podtemplate.ExpandSpec(ctx, template.Spec, at.AtName("template").AtListIndex(0).AtName("spec"))
	diagnostics.Append(templateSpecDiags...)
	if diagnostics.HasError() {
		return out, diagnostics
	}

	out.Template = corev1.PodTemplateSpec{
		ObjectMeta: templateMetadata,
		Spec:       templateSpec,
	}
	return out, diagnostics
}

func flattenDaemonSetSpecModel(ctx context.Context, spec appsv1.DaemonSetSpec, baseline []DaemonSetV1SpecModel) ([]DaemonSetV1SpecModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	specType := podtemplate.SpecBlock(podtemplate.Options{RestartPolicyAlways: false}).NestedObject.Type()
	templateBaseline := types.ListNull(specType)
	templateMetadataBaseline := []common.NamespacedMetadataModel(nil)
	var selectorBaseline []LabelSelectorModel
	if len(baseline) == 1 {
		selectorBaseline = baseline[0].Selector
	}
	if len(baseline) == 1 && len(baseline[0].Template) == 1 {
		templateBaseline = baseline[0].Template[0].Spec
		templateMetadataBaseline = baseline[0].Template[0].Metadata
	}
	templateSpec, templateSpecDiags := podtemplate.FlattenSpec(
		ctx,
		spec.Template.Spec,
		templateBaseline,
		daemonSetSpecPath().AtName("template").AtListIndex(0).AtName("spec"),
	)
	diagnostics.Append(templateSpecDiags...)
	templateMetadata, templateMetadataDiags := flattenDaemonSetTemplateMetadata(ctx, spec.Template.ObjectMeta, templateMetadataBaseline)
	diagnostics.Append(templateMetadataDiags...)
	selector, selectorDiags := flattenWorkloadSelector(ctx, spec.Selector, selectorBaseline)
	diagnostics.Append(selectorDiags...)
	if diagnostics.HasError() {
		return nil, diagnostics
	}

	revisionHistoryLimit := types.Int64Value(10)
	if spec.RevisionHistoryLimit != nil {
		revisionHistoryLimit = types.Int64Value(int64(*spec.RevisionHistoryLimit))
	}
	return []DaemonSetV1SpecModel{
		{
			MinReadySeconds:      types.Int64Value(int64(spec.MinReadySeconds)),
			RevisionHistoryLimit: revisionHistoryLimit,
			Selector:             selector,
			Strategy:             flattenDaemonSetStrategyModel(ctx, spec.UpdateStrategy, &diagnostics),
			Template: []DaemonSetTemplateModel{
				{
					Metadata: templateMetadata,
					Spec:     templateSpec,
				},
			},
		},
	}, diagnostics
}

func expandDaemonSetSelector(ctx context.Context, in []DaemonSetLabelSelectorModel, at path.Path) (*metav1.LabelSelector, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	if len(in) == 0 {
		return nil, diagnostics
	}
	selector := &metav1.LabelSelector{}
	block := in[0]
	if !block.MatchLabels.IsNull() && !block.MatchLabels.IsUnknown() {
		labels := map[string]string{}
		diagnostics.Append(block.MatchLabels.ElementsAs(ctx, &labels, false)...)
		selector.MatchLabels = labels
	}
	if len(block.MatchExpressions) > 0 {
		selector.MatchExpressions = make([]metav1.LabelSelectorRequirement, 0, len(block.MatchExpressions))
		for i, expression := range block.MatchExpressions {
			requirement := metav1.LabelSelectorRequirement{}
			if !expression.Key.IsNull() && !expression.Key.IsUnknown() {
				requirement.Key = expression.Key.ValueString()
			}
			if !expression.Operator.IsNull() && !expression.Operator.IsUnknown() {
				requirement.Operator = metav1.LabelSelectorOperator(expression.Operator.ValueString())
			}
			if !expression.Values.IsNull() && !expression.Values.IsUnknown() {
				values := []string{}
				diagnostics.Append(expression.Values.ElementsAs(ctx, &values, false)...)
				requirement.Values = values
			}
			if diagnostics.HasError() {
				return nil, diagnostics
			}
			if requirement.Key == "" && requirement.Operator == "" && len(requirement.Values) == 0 {
				diagnostics.AddAttributeError(
					at.AtName("match_expressions").AtListIndex(i),
					"Invalid selector expression",
					"Empty selector expression is not valid.",
				)
				continue
			}
			selector.MatchExpressions = append(selector.MatchExpressions, requirement)
		}
	}
	return selector, diagnostics
}

func expandDaemonSetStrategyModel(ctx context.Context, in types.List, at path.Path) (appsv1.DaemonSetUpdateStrategy, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	strategy := appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType}
	if in.IsNull() || in.IsUnknown() || len(in.Elements()) == 0 {
		return strategy, diagnostics
	}
	var models []DaemonSetStrategyModel
	diagnostics.Append(in.ElementsAs(ctx, &models, false)...)
	if diagnostics.HasError() || len(models) == 0 {
		diagnostics.AddAttributeError(at, "Invalid daemonset strategy", "Unable to decode the strategy value.")
		return strategy, diagnostics
	}
	model := models[0]
	if !model.Type.IsNull() && !model.Type.IsUnknown() && model.Type.ValueString() != "" {
		strategy.Type = appsv1.DaemonSetUpdateStrategyType(model.Type.ValueString())
	}
	if strategy.Type == appsv1.OnDeleteDaemonSetStrategyType || model.RollingUpdate.IsNull() ||
		model.RollingUpdate.IsUnknown() || len(model.RollingUpdate.Elements()) == 0 {
		return strategy, diagnostics
	}
	var updates []DaemonSetRollingUpdateModel
	diagnostics.Append(model.RollingUpdate.ElementsAs(ctx, &updates, false)...)
	if diagnostics.HasError() || len(updates) == 0 {
		diagnostics.AddAttributeError(at.AtName("rolling_update"), "Invalid rolling update strategy", "Unable to decode the rolling_update value.")
		return strategy, diagnostics
	}
	rolling := &appsv1.RollingUpdateDaemonSet{}
	if !updates[0].MaxSurge.IsNull() && !updates[0].MaxSurge.IsUnknown() {
		value := intstr.Parse(updates[0].MaxSurge.ValueString())
		rolling.MaxSurge = &value
	}
	if !updates[0].MaxUnavailable.IsNull() && !updates[0].MaxUnavailable.IsUnknown() {
		value := intstr.Parse(updates[0].MaxUnavailable.ValueString())
		rolling.MaxUnavailable = &value
	}
	strategy.RollingUpdate = rolling
	return strategy, diagnostics
}

func flattenDaemonSetStrategyModel(ctx context.Context, in appsv1.DaemonSetUpdateStrategy, diagnostics *diag.Diagnostics) types.List {
	strategyType := in.Type
	if strategyType == "" {
		strategyType = appsv1.RollingUpdateDaemonSetStrategyType
	}
	model := DaemonSetStrategyModel{
		Type:          types.StringValue(string(strategyType)),
		RollingUpdate: types.ListNull(daemonSetRollingUpdateObjectType()),
	}
	if in.RollingUpdate != nil {
		rolling := DaemonSetRollingUpdateModel{
			MaxSurge:       types.StringValue("0"),
			MaxUnavailable: types.StringValue("1"),
		}
		if in.RollingUpdate.MaxSurge != nil {
			rolling.MaxSurge = types.StringValue(in.RollingUpdate.MaxSurge.String())
		}
		if in.RollingUpdate.MaxUnavailable != nil {
			rolling.MaxUnavailable = types.StringValue(in.RollingUpdate.MaxUnavailable.String())
		}
		value, diags := types.ListValueFrom(ctx, daemonSetRollingUpdateObjectType(), []DaemonSetRollingUpdateModel{rolling})
		diagnostics.Append(diags...)
		model.RollingUpdate = value
	}
	value, diags := types.ListValueFrom(ctx, daemonSetStrategyObjectType(), []DaemonSetStrategyModel{model})
	diagnostics.Append(diags...)
	return value
}

func flattenDaemonSetTemplateMetadata(ctx context.Context, in metav1.ObjectMeta, prior []common.NamespacedMetadataModel) ([]common.NamespacedMetadataModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	annotations := types.MapNull(types.StringType)
	labels := types.MapNull(types.StringType)
	if in.Annotations != nil {
		value, diags := types.MapValueFrom(ctx, types.StringType, in.Annotations)
		diagnostics.Append(diags...)
		annotations = value
	} else if len(prior) > 0 && !prior[0].Annotations.IsNull() {
		annotations = types.MapValueMust(types.StringType, nil)
	}
	if in.Labels != nil {
		value, diags := types.MapValueFrom(ctx, types.StringType, in.Labels)
		diagnostics.Append(diags...)
		labels = value
	} else if len(prior) > 0 && !prior[0].Labels.IsNull() {
		labels = types.MapValueMust(types.StringType, nil)
	}
	generateName := types.StringNull()
	if in.GenerateName != "" {
		generateName = types.StringValue(in.GenerateName)
	}
	namespace := types.StringNull()
	if in.Namespace != "" || len(prior) > 0 && prior[0].Namespace.Equal(types.StringValue("")) {
		namespace = types.StringValue(in.Namespace)
	}

	return []common.NamespacedMetadataModel{
		{
			MetadataModel: common.MetadataModel{
				MetadataBase: common.MetadataBase{
					Annotations:     annotations,
					Generation:      types.Int64Value(in.Generation),
					Labels:          labels,
					Name:            types.StringValue(in.Name),
					ResourceVersion: types.StringValue(in.ResourceVersion),
					UID:             types.StringValue(string(in.UID)),
				},
				GenerateName: generateName,
			},
			Namespace: namespace,
		},
	}, diagnostics
}

func daemonSetMetadataPatchOps(state, plan DaemonSetV1Model, live metav1.ObjectMeta) kubernetes.PatchOperations {
	return common.MetadataPatchOpsAgainstLive("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel, live)
}

// daemonSetStrategicSpecPatch returns the strategic merge patch from the prior
// to the planned spec. live is the spec the server holds, or nil when unknown.
func daemonSetStrategicSpecPatch(ctx context.Context, state, plan []DaemonSetV1SpecModel, live *appsv1.DaemonSetSpec) ([]byte, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	oldSpec, oldDiags := expandDaemonSetSpecModel(ctx, state, daemonSetSpecPath())
	diagnostics.Append(oldDiags...)
	newSpec, newDiags := expandDaemonSetSpecModel(ctx, plan, daemonSetSpecPath())
	diagnostics.Append(newDiags...)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	oldJSON, err := json.Marshal(appsv1.DaemonSet{Spec: oldSpec})
	if err != nil {
		diagnostics.AddError("Error encoding prior daemonset spec", err.Error())
		return nil, diagnostics
	}
	newJSON, err := json.Marshal(appsv1.DaemonSet{Spec: newSpec})
	if err != nil {
		diagnostics.AddError("Error encoding planned daemonset spec", err.Error())
		return nil, diagnostics
	}
	var liveJSON []byte
	if live != nil {
		liveJSON, err = json.Marshal(appsv1.DaemonSet{Spec: *live})
		if err != nil {
			diagnostics.AddError("Error encoding live daemonset spec", err.Error())
			return nil, diagnostics
		}
	}
	patch, err := common.TwoWayStrategicMergePatch(oldJSON, newJSON, liveJSON, appsv1.DaemonSet{})
	if err != nil {
		diagnostics.AddError("Error creating daemonset spec patch", err.Error())
		return nil, diagnostics
	}
	if string(patch) == "{}" {
		return nil, diagnostics
	}
	return patch, diagnostics
}
