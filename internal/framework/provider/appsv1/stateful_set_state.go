// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type StatefulSetV1Model struct {
	ID             types.String                     `tfsdk:"id"`
	Metadata       []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec           []StatefulSetSpecModel           `tfsdk:"spec"`
	WaitForRollout types.Bool                       `tfsdk:"wait_for_rollout"`
	Timeouts       timeouts.Value                   `tfsdk:"timeouts"`
}

type StatefulSetSpecModel struct {
	PodManagementPolicy                  types.String                     `tfsdk:"pod_management_policy"`
	Replicas                             types.String                     `tfsdk:"replicas"`
	RevisionHistoryLimit                 types.Int64                      `tfsdk:"revision_history_limit"`
	Selector                             []LabelSelectorModel             `tfsdk:"selector"`
	ServiceName                          types.String                     `tfsdk:"service_name"`
	Template                             []StatefulSetTemplateModel       `tfsdk:"template"`
	UpdateStrategy                       []StatefulSetUpdateStrategyModel `tfsdk:"update_strategy"`
	VolumeClaimTemplate                  []PersistentVolumeClaimModel     `tfsdk:"volume_claim_template"`
	PersistentVolumeClaimRetentionPolicy types.List                       `tfsdk:"persistent_volume_claim_retention_policy"`
	MinReadySeconds                      types.Int64                      `tfsdk:"min_ready_seconds"`
}

type StatefulSetTemplateModel struct {
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     types.List                       `tfsdk:"spec"`
}

type StatefulSetUpdateStrategyModel struct {
	Type          types.String                    `tfsdk:"type"`
	RollingUpdate []StatefulSetRollingUpdateModel `tfsdk:"rolling_update"`
}

type StatefulSetRollingUpdateModel struct {
	Partition types.Int64 `tfsdk:"partition"`
}

type StatefulSetPVCRetentionPolicyModel struct {
	WhenDeleted types.String `tfsdk:"when_deleted"`
	WhenScaled  types.String `tfsdk:"when_scaled"`
}

type PersistentVolumeClaimModel struct {
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     []PersistentVolumeClaimSpecModel `tfsdk:"spec"`
}

type PersistentVolumeClaimSpecModel struct {
	AccessModes      types.Set              `tfsdk:"access_modes"`
	Resources        []VolumeResourcesModel `tfsdk:"resources"`
	Selector         []LabelSelectorModel   `tfsdk:"selector"`
	VolumeName       types.String           `tfsdk:"volume_name"`
	StorageClassName types.String           `tfsdk:"storage_class_name"`
	VolumeMode       types.String           `tfsdk:"volume_mode"`
}

type VolumeResourcesModel struct {
	Limits   types.Map `tfsdk:"limits"`
	Requests types.Map `tfsdk:"requests"`
}

type statefulSetIdentityModel struct {
	common.ResourceIdentity
	Namespace types.String `tfsdk:"namespace"`
}

func expandStatefulSetSpec(ctx context.Context, spec StatefulSetSpecModel) (*appsv1.StatefulSetSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &appsv1.StatefulSetSpec{}

	if !spec.PodManagementPolicy.IsNull() && !spec.PodManagementPolicy.IsUnknown() {
		out.PodManagementPolicy = appsv1.PodManagementPolicyType(spec.PodManagementPolicy.ValueString())
	}

	if !spec.Replicas.IsNull() && !spec.Replicas.IsUnknown() {
		v := spec.Replicas.ValueString()
		if v != "" {
			i, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				diags.AddAttributeError(path.Root("spec").AtListIndex(0).AtName("replicas"), "Invalid replicas", err.Error())
				return out, diags
			}
			out.Replicas = ptr.To(int32(i))
		}
	}

	if !spec.RevisionHistoryLimit.IsNull() && !spec.RevisionHistoryLimit.IsUnknown() {
		v := int32(spec.RevisionHistoryLimit.ValueInt64())
		out.RevisionHistoryLimit = &v
	}

	if len(spec.Selector) > 0 {
		sel, d := expandLabelSelector(ctx, spec.Selector[0])
		diags.Append(d...)
		out.Selector = sel
	}

	if !spec.ServiceName.IsNull() && !spec.ServiceName.IsUnknown() {
		out.ServiceName = spec.ServiceName.ValueString()
	}

	if len(spec.UpdateStrategy) > 0 {
		u, d := expandStatefulSetUpdateStrategy(spec.UpdateStrategy[0])
		diags.Append(d...)
		out.UpdateStrategy = u
	}

	retention, d := statefulSetPVCRetentionPolicyModels(ctx, spec.PersistentVolumeClaimRetentionPolicy, path.Root("spec").AtListIndex(0).AtName("persistent_volume_claim_retention_policy"))
	diags.Append(d...)
	if len(retention) > 0 {
		out.PersistentVolumeClaimRetentionPolicy = &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
			WhenDeleted: appsv1.PersistentVolumeClaimRetentionPolicyType(retention[0].WhenDeleted.ValueString()),
			WhenScaled:  appsv1.PersistentVolumeClaimRetentionPolicyType(retention[0].WhenScaled.ValueString()),
		}
	}

	if len(spec.Template) > 0 {
		tm := spec.Template[0]
		meta, d := common.ExpandNamespacedMetadata(ctx, tm.Metadata)
		diags.Append(d...)
		podSpec, d := expandPodTemplateSpec(ctx, tm.Spec, path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("spec"))
		diags.Append(d...)
		out.Template = corev1.PodTemplateSpec{ObjectMeta: meta, Spec: podSpec}
	}

	if len(spec.VolumeClaimTemplate) > 0 {
		out.VolumeClaimTemplates = make([]corev1.PersistentVolumeClaim, 0, len(spec.VolumeClaimTemplate))
		for idx, pvc := range spec.VolumeClaimTemplate {
			expanded, d := expandPersistentVolumeClaim(ctx, pvc, idx)
			diags.Append(d...)
			if d.HasError() {
				continue
			}
			out.VolumeClaimTemplates = append(out.VolumeClaimTemplates, expanded)
		}
	}

	if !spec.MinReadySeconds.IsNull() && !spec.MinReadySeconds.IsUnknown() {
		out.MinReadySeconds = int32(spec.MinReadySeconds.ValueInt64())
	}

	return out, diags
}

func expandPodTemplateSpec(ctx context.Context, value types.List, at path.Path) (corev1.PodSpec, diag.Diagnostics) {
	return podspec.For(podspec.StatefulSet()).ExpandSpec(ctx, value, at)
}

func flattenStatefulSetSpec(ctx context.Context, spec appsv1.StatefulSetSpec, baseline *StatefulSetSpecModel, filters kubernetes.MetadataFilters, refresh bool) (StatefulSetSpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := StatefulSetSpecModel{
		PodManagementPolicy:                  types.StringNull(),
		Replicas:                             types.StringNull(),
		RevisionHistoryLimit:                 types.Int64Null(),
		ServiceName:                          types.StringNull(),
		PersistentVolumeClaimRetentionPolicy: types.ListNull(statefulSetPVCRetentionPolicyObjectType()),
		MinReadySeconds:                      types.Int64Value(int64(spec.MinReadySeconds)),
	}

	if spec.PodManagementPolicy != "" {
		out.PodManagementPolicy = types.StringValue(string(spec.PodManagementPolicy))
	}
	if spec.Replicas != nil {
		out.Replicas = types.StringValue(strconv.Itoa(int(*spec.Replicas)))
	}
	if spec.RevisionHistoryLimit != nil {
		out.RevisionHistoryLimit = types.Int64Value(int64(*spec.RevisionHistoryLimit))
	}
	var selectorBaseline []LabelSelectorModel
	if baseline != nil {
		selectorBaseline = baseline.Selector
	}
	selectors, selectorDiags := flattenWorkloadSelector(ctx, spec.Selector, selectorBaseline)
	diags.Append(selectorDiags...)
	out.Selector = selectors
	if spec.ServiceName != "" {
		out.ServiceName = types.StringValue(spec.ServiceName)
	}

	template, d := flattenTemplate(ctx, spec.Template, baseline, filters, refresh)
	diags.Append(d...)
	out.Template = []StatefulSetTemplateModel{template}

	out.VolumeClaimTemplate = make([]PersistentVolumeClaimModel, len(spec.VolumeClaimTemplates))
	for i, pvc := range spec.VolumeClaimTemplates {
		var prior *PersistentVolumeClaimModel
		if baseline != nil && i < len(baseline.VolumeClaimTemplate) {
			prior = &baseline.VolumeClaimTemplate[i]
		}
		model, fd := flattenPersistentVolumeClaim(ctx, pvc, prior, filters)
		diags.Append(fd...)
		out.VolumeClaimTemplate[i] = model
	}

	if baseline != nil && len(baseline.UpdateStrategy) > 0 {
		out.UpdateStrategy = []StatefulSetUpdateStrategyModel{flattenStatefulSetUpdateStrategy(spec.UpdateStrategy)}
		if len(baseline.UpdateStrategy[0].RollingUpdate) == 0 {
			out.UpdateStrategy[0].RollingUpdate = nil
		}
	}

	if spec.PersistentVolumeClaimRetentionPolicy != nil {
		value, d := types.ListValueFrom(ctx, statefulSetPVCRetentionPolicyObjectType(), []StatefulSetPVCRetentionPolicyModel{{
			WhenDeleted: types.StringValue(string(spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted)),
			WhenScaled:  types.StringValue(string(spec.PersistentVolumeClaimRetentionPolicy.WhenScaled)),
		}})
		diags.Append(d...)
		out.PersistentVolumeClaimRetentionPolicy = value
	}

	return out, diags
}

func expandStatefulSetUpdateStrategy(in StatefulSetUpdateStrategyModel) (appsv1.StatefulSetUpdateStrategy, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := appsv1.StatefulSetUpdateStrategy{}

	if !in.Type.IsNull() && !in.Type.IsUnknown() {
		out.Type = appsv1.StatefulSetUpdateStrategyType(in.Type.ValueString())
	}
	if len(in.RollingUpdate) > 0 {
		ru := appsv1.RollingUpdateStatefulSetStrategy{}
		if !in.RollingUpdate[0].Partition.IsNull() && !in.RollingUpdate[0].Partition.IsUnknown() {
			v := int32(in.RollingUpdate[0].Partition.ValueInt64())
			ru.Partition = &v
		}
		out.RollingUpdate = &ru
	}
	return out, diags
}

func flattenStatefulSetUpdateStrategy(in appsv1.StatefulSetUpdateStrategy) StatefulSetUpdateStrategyModel {
	out := StatefulSetUpdateStrategyModel{Type: types.StringValue(string(in.Type))}
	if in.RollingUpdate != nil {
		partition := int64(0)
		if in.RollingUpdate.Partition != nil {
			partition = int64(*in.RollingUpdate.Partition)
		}
		out.RollingUpdate = []StatefulSetRollingUpdateModel{{Partition: types.Int64Value(partition)}}
	}
	return out
}

func expandLabelSelector(ctx context.Context, in LabelSelectorModel) (*metav1.LabelSelector, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := &metav1.LabelSelector{}

	if !in.MatchLabels.IsNull() && !in.MatchLabels.IsUnknown() {
		labels := make(map[string]string)
		diags.Append(in.MatchLabels.ElementsAs(ctx, &labels, false)...)
		out.MatchLabels = labels
	}

	if len(in.MatchExpressions) > 0 {
		out.MatchExpressions = make([]metav1.LabelSelectorRequirement, len(in.MatchExpressions))
		for i, expr := range in.MatchExpressions {
			req := metav1.LabelSelectorRequirement{}
			if !expr.Key.IsNull() && !expr.Key.IsUnknown() {
				req.Key = expr.Key.ValueString()
			}
			if !expr.Operator.IsNull() && !expr.Operator.IsUnknown() {
				req.Operator = metav1.LabelSelectorOperator(expr.Operator.ValueString())
			}
			if !expr.Values.IsNull() && !expr.Values.IsUnknown() {
				vals := make([]string, 0, len(expr.Values.Elements()))
				for _, v := range expr.Values.Elements() {
					s, ok := v.(types.String)
					if !ok || s.IsNull() || s.IsUnknown() {
						continue
					}
					vals = append(vals, s.ValueString())
				}
				req.Values = vals
			}
			out.MatchExpressions[i] = req
		}
	}

	return out, diags
}

func expandPersistentVolumeClaim(ctx context.Context, in PersistentVolumeClaimModel, index int) (corev1.PersistentVolumeClaim, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := corev1.PersistentVolumeClaim{}

	meta, d := common.ExpandNamespacedMetadata(ctx, in.Metadata)
	diags.Append(d...)
	out.ObjectMeta = meta

	if len(in.Spec) > 0 {
		spec, sd := expandPersistentVolumeClaimSpec(ctx, in.Spec[0], index)
		diags.Append(sd...)
		out.Spec = spec
	}
	return out, diags
}

func expandPersistentVolumeClaimSpec(ctx context.Context, in PersistentVolumeClaimSpecModel, index int) (corev1.PersistentVolumeClaimSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := corev1.PersistentVolumeClaimSpec{}

	if !in.AccessModes.IsNull() && !in.AccessModes.IsUnknown() {
		for _, elem := range in.AccessModes.Elements() {
			s, ok := elem.(types.String)
			if !ok || s.IsNull() || s.IsUnknown() {
				continue
			}
			out.AccessModes = append(out.AccessModes, corev1.PersistentVolumeAccessMode(s.ValueString()))
		}
	}

	if len(in.Resources) > 0 {
		resources, d := expandVolumeResources(ctx, in.Resources[0], index)
		diags.Append(d...)
		out.Resources = resources
	}

	if len(in.Selector) > 0 {
		sel, d := expandLabelSelector(ctx, in.Selector[0])
		diags.Append(d...)
		out.Selector = sel
	}

	if !in.VolumeName.IsNull() && !in.VolumeName.IsUnknown() {
		out.VolumeName = in.VolumeName.ValueString()
	}
	if !in.StorageClassName.IsNull() && !in.StorageClassName.IsUnknown() && in.StorageClassName.ValueString() != "" {
		out.StorageClassName = ptr.To(in.StorageClassName.ValueString())
	}
	if !in.VolumeMode.IsNull() && !in.VolumeMode.IsUnknown() && in.VolumeMode.ValueString() != "" {
		mode := corev1.PersistentVolumeMode(in.VolumeMode.ValueString())
		out.VolumeMode = &mode
	}

	return out, diags
}

func expandVolumeResources(ctx context.Context, in VolumeResourcesModel, _ int) (corev1.VolumeResourceRequirements, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := corev1.VolumeResourceRequirements{}

	if !in.Limits.IsNull() && !in.Limits.IsUnknown() {
		limits, d := expandMapToResourceListFromMap(ctx, in.Limits)
		diags.Append(d...)
		out.Limits = limits
	}
	if !in.Requests.IsNull() && !in.Requests.IsUnknown() {
		requests, d := expandMapToResourceListFromMap(ctx, in.Requests)
		diags.Append(d...)
		out.Requests = requests
	}

	return out, diags
}

func expandMapToResourceListFromMap(ctx context.Context, m types.Map) (corev1.ResourceList, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make(corev1.ResourceList)
	if m.IsNull() || m.IsUnknown() {
		return out, diags
	}

	raw := make(map[string]string)
	diags.Append(m.ElementsAs(ctx, &raw, false)...)
	for k, v := range raw {
		q, err := k8sresource.ParseQuantity(v)
		if err != nil {
			diags.AddError("Invalid resource quantity", fmt.Sprintf("%s: %s", k, err))
			continue
		}
		out[corev1.ResourceName(k)] = q
	}
	return out, diags
}

func flattenTemplate(ctx context.Context, in corev1.PodTemplateSpec, baseline *StatefulSetSpecModel, filters kubernetes.MetadataFilters, refresh bool) (StatefulSetTemplateModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := StatefulSetTemplateModel{}

	var priorMetadata []common.NamespacedMetadataModel
	if baseline != nil && len(baseline.Template) > 0 {
		priorMetadata = baseline.Template[0].Metadata
	}
	meta, d := common.FlattenNamespacedMetadata(ctx, in.ObjectMeta, priorMetadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	diags.Append(d...)
	preserveEmbeddedMetadataNamespace(meta, priorMetadata, in.Namespace)
	out.Metadata = meta

	var baselineSpec types.List
	if baseline != nil && len(baseline.Template) > 0 {
		baselineSpec = baseline.Template[0].Spec
	}
	flatten := podspec.For(podspec.StatefulSet()).FlattenSpec
	if refresh {
		flatten = podspec.For(podspec.StatefulSet()).RefreshSpec
	}
	podSpec, d2 := flatten(ctx, in.Spec, baselineSpec, path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("spec"))
	diags.Append(d2...)
	out.Spec = podSpec

	return out, diags
}

func flattenPersistentVolumeClaim(ctx context.Context, in corev1.PersistentVolumeClaim, baseline *PersistentVolumeClaimModel, filters kubernetes.MetadataFilters) (PersistentVolumeClaimModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := PersistentVolumeClaimModel{}

	var priorMetadata []common.NamespacedMetadataModel
	if baseline != nil {
		priorMetadata = baseline.Metadata
	}
	meta, d := common.FlattenNamespacedMetadata(ctx, in.ObjectMeta, priorMetadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	diags.Append(d...)
	preserveEmbeddedMetadataNamespace(meta, priorMetadata, in.Namespace)
	out.Metadata = meta

	var priorSpec *PersistentVolumeClaimSpecModel
	if baseline != nil && len(baseline.Spec) > 0 {
		priorSpec = &baseline.Spec[0]
	}
	spec, d := flattenPersistentVolumeClaimSpec(ctx, in.Spec, priorSpec)
	diags.Append(d...)
	out.Spec = []PersistentVolumeClaimSpecModel{spec}

	return out, diags
}

func flattenPersistentVolumeClaimSpec(ctx context.Context, in corev1.PersistentVolumeClaimSpec, prior *PersistentVolumeClaimSpecModel) (PersistentVolumeClaimSpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := PersistentVolumeClaimSpecModel{
		AccessModes:      types.SetNull(types.StringType),
		VolumeName:       types.StringNull(),
		StorageClassName: types.StringNull(),
		VolumeMode:       types.StringNull(),
	}
	if prior != nil {
		if !prior.VolumeName.IsUnknown() {
			out.VolumeName = prior.VolumeName
		}
		if !prior.StorageClassName.IsUnknown() {
			out.StorageClassName = prior.StorageClassName
		}
		if !prior.VolumeMode.IsUnknown() {
			out.VolumeMode = prior.VolumeMode
		}
		out.Selector = prior.Selector
	}

	modes := make([]string, 0, len(in.AccessModes))
	for _, mode := range in.AccessModes {
		modes = append(modes, string(mode))
	}
	setVal, d := types.SetValueFrom(ctx, types.StringType, modes)
	diags.Append(d...)
	out.AccessModes = setVal

	resources, d := flattenVolumeResources(ctx, in.Resources)
	diags.Append(d...)
	if prior != nil && len(prior.Resources) > 0 {
		resources.Limits, d = statefulSetPreserveQuantityMap(resources.Limits, prior.Resources[0].Limits)
		diags.Append(d...)
		resources.Requests, d = statefulSetPreserveQuantityMap(resources.Requests, prior.Resources[0].Requests)
		diags.Append(d...)
	}
	out.Resources = []VolumeResourcesModel{resources}

	if in.Selector != nil {
		selectors, sd := flattenWorkloadSelector(ctx, in.Selector, out.Selector)
		diags.Append(sd...)
		out.Selector = selectors
	}
	if in.VolumeName != "" {
		out.VolumeName = types.StringValue(in.VolumeName)
	}
	if in.StorageClassName != nil {
		out.StorageClassName = types.StringValue(*in.StorageClassName)
	}
	if in.VolumeMode != nil {
		out.VolumeMode = types.StringValue(string(*in.VolumeMode))
	}

	return out, diags
}

func flattenVolumeResources(ctx context.Context, in corev1.VolumeResourceRequirements) (VolumeResourcesModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := VolumeResourcesModel{Limits: types.MapNull(types.StringType), Requests: types.MapNull(types.StringType)}

	if len(in.Limits) > 0 {
		limits := make(map[string]string, len(in.Limits))
		for k, v := range in.Limits {
			limits[string(k)] = v.String()
		}
		mv, d := types.MapValueFrom(ctx, types.StringType, limits)
		diags.Append(d...)
		out.Limits = mv
	}
	if len(in.Requests) > 0 {
		requests := make(map[string]string, len(in.Requests))
		for k, v := range in.Requests {
			requests[string(k)] = v.String()
		}
		mv, d := types.MapValueFrom(ctx, types.StringType, requests)
		diags.Append(d...)
		out.Requests = mv
	}
	return out, diags
}

func preserveEmbeddedMetadataNamespace(flattened, prior []common.NamespacedMetadataModel, apiNamespace string) {
	if apiNamespace != "" || len(flattened) == 0 {
		return
	}
	if len(prior) > 0 && !prior[0].Namespace.IsNull() && !prior[0].Namespace.IsUnknown() {
		flattened[0].Namespace = prior[0].Namespace
		return
	}
	flattened[0].Namespace = types.StringValue("")
}

func statefulSetPVCRetentionPolicyObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"when_deleted": types.StringType,
		"when_scaled":  types.StringType,
	}}
}

func statefulSetPVCRetentionPolicyModels(ctx context.Context, value types.List, at path.Path) ([]StatefulSetPVCRetentionPolicyModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() || len(value.Elements()) == 0 {
		return nil, diags
	}
	var models []StatefulSetPVCRetentionPolicyModel
	diags.Append(value.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		diags.AddAttributeError(at, "Invalid persistent volume claim retention policy", "Unable to decode the configured retention policy.")
	}
	return models, diags
}

func applyStatefulSetV0ResourceUpgrade(rawState map[string]interface{}) map[string]interface{} {
	s, ok := rawState["spec"].([]interface{})
	if !ok || len(s) == 0 {
		return rawState
	}

	spec, ok := s[0].(map[string]interface{})
	if !ok {
		return rawState
	}
	t, ok := spec["template"].([]interface{})
	if !ok || len(t) == 0 {
		return rawState
	}
	template, ok := t[0].(map[string]interface{})
	if !ok {
		return rawState
	}
	ps, ok := template["spec"].([]interface{})
	if !ok || len(ps) == 0 {
		return rawState
	}

	podSpec, ok := ps[0].(map[string]interface{})
	if !ok {
		return rawState
	}
	template["spec"] = []interface{}{upgradeContainersV0ToV1(podSpec)}
	return rawState
}

func upgradeContainersV0ToV1(rawState map[string]interface{}) map[string]interface{} {
	upgrade := func(listKey string) {
		containers, ok := rawState[listKey].([]interface{})
		if !ok || len(containers) == 0 {
			return
		}
		for _, c := range containers {
			container, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			r, ok := container["resources"].([]interface{})
			if !ok || len(r) == 0 {
				continue
			}
			resources, ok := r[0].(map[string]interface{})
			if !ok {
				continue
			}

			if req, ok := resources["requests"].([]interface{}); ok {
				if len(req) > 0 {
					if m, ok := req[0].(map[string]interface{}); ok {
						resources["requests"] = m
					} else {
						resources["requests"] = map[string]interface{}{}
					}
				} else {
					resources["requests"] = map[string]interface{}{}
				}
			}
			if lim, ok := resources["limits"].([]interface{}); ok {
				if len(lim) > 0 {
					if m, ok := lim[0].(map[string]interface{}); ok {
						resources["limits"] = m
					} else {
						resources["limits"] = map[string]interface{}{}
					}
				} else {
					resources["limits"] = map[string]interface{}{}
				}
			}
		}
	}

	upgrade("init_container")
	upgrade("container")
	return rawState
}

func (r *StatefulSetV1) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	targetSchema := schemaResp.Schema
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				resp.Diagnostics.Append(schemaResp.Diagnostics...)
				if resp.Diagnostics.HasError() {
					return
				}
				if req.RawState == nil || len(req.RawState.JSON) == 0 {
					resp.Diagnostics.AddError("Unable to upgrade StatefulSet state", "Source state is empty")
					return
				}

				var raw map[string]interface{}
				if err := json.Unmarshal(req.RawState.JSON, &raw); err != nil {
					resp.Diagnostics.AddError("Unable to upgrade StatefulSet state", err.Error())
					return
				}
				upgraded := applyStatefulSetV0ResourceUpgrade(raw)
				value, err := decodeStatefulSetStateValue(ctx, upgraded, targetSchema)
				if err != nil {
					resp.Diagnostics.AddError("Unable to upgrade StatefulSet state", fmt.Sprintf("Could not decode upgraded state: %s", err))
					return
				}
				resp.State = tfsdk.State{Schema: targetSchema, Raw: value}
			},
		},
	}
}

func decodeStatefulSetStateValue(ctx context.Context, raw map[string]interface{}, targetSchema schema.Schema) (tftypes.Value, error) {
	stateJSON, err := json.Marshal(raw)
	if err != nil {
		return tftypes.Value{}, err
	}
	return (&tfprotov6.RawState{JSON: stateJSON}).Unmarshal(targetSchema.Type().TerraformType(ctx))
}

func (r *StatefulSetV1) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	return []resource.StateMover{
		{
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != "kubernetes_stateful_set" ||
					(req.SourceSchemaVersion != 0 && req.SourceSchemaVersion != schemaResp.Schema.Version) {
					return
				}
				if req.SourceProviderAddress == "" || !hasProviderSuffix(req.SourceProviderAddress) {
					return
				}
				if req.SourceRawState == nil || len(req.SourceRawState.JSON) == 0 {
					resp.Diagnostics.AddError("Unable to move StatefulSet state", "The source state has no JSON data")
					return
				}
				var value tftypes.Value
				var err error
				if req.SourceSchemaVersion == 0 {
					var raw map[string]interface{}
					if err = json.Unmarshal(req.SourceRawState.JSON, &raw); err == nil {
						value, err = decodeStatefulSetStateValue(ctx, applyStatefulSetV0ResourceUpgrade(raw), schemaResp.Schema)
					}
				} else {
					value, err = req.SourceRawState.Unmarshal(schemaResp.Schema.Type().TerraformType(ctx))
				}
				if err != nil {
					resp.Diagnostics.AddError("Unable to move StatefulSet state", fmt.Sprintf("The source state could not be decoded: %s", err))
					return
				}

				sourceState := tfsdk.State{Schema: schemaResp.Schema, Raw: value}
				var moved StatefulSetV1Model
				resp.Diagnostics.Append(sourceState.Get(ctx, &moved)...)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.Diagnostics.Append(resp.TargetState.Set(ctx, &moved)...)
				if resp.Diagnostics.HasError() || resp.TargetIdentity == nil {
					return
				}

				if len(moved.Metadata) == 0 {
					resp.Diagnostics.AddError("Unable to move StatefulSet state", "metadata is empty")
					return
				}
				resp.Diagnostics.Append(resp.TargetIdentity.Set(ctx, statefulSetIdentityModel{
					ResourceIdentity: common.ResourceIdentity{
						APIVersion: types.StringValue(statefulSetAPIVersion),
						Kind:       types.StringValue(statefulSetKind),
						Name:       moved.Metadata[0].Name,
					},
					Namespace: moved.Metadata[0].Namespace,
				})...)
			},
		},
	}
}
