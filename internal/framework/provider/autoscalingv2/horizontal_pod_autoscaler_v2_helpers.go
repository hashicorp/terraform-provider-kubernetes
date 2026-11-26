// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	autoscalingv2api "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// expandSpec sends only what is known and set. Unlike SDKv2, an unset behavior, select
// policy or stabilization window is omitted so that Kubernetes applies its defaults.
func expandSpec(ctx context.Context, in specModel) (autoscalingv2api.HorizontalPodAutoscalerSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	spec := autoscalingv2api.HorizontalPodAutoscalerSpec{
		MaxReplicas: int32(in.MaxReplicas.ValueInt64()),
	}
	if v := in.MinReplicas.ValueInt64(); v > 0 {
		spec.MinReplicas = ptr.To(int32(v))
	}
	if len(in.ScaleTargetRef) > 0 {
		spec.ScaleTargetRef = expandScaleTargetRef(in.ScaleTargetRef[0])
	}

	metrics, d := expandMetrics(ctx, in.Metric)
	diags.Append(d...)
	spec.Metrics = metrics

	behavior, d := expandBehavior(ctx, in.Behavior)
	diags.Append(d...)
	spec.Behavior = behavior

	return spec, diags
}

func expandScaleTargetRef(in scaleTargetRef) autoscalingv2api.CrossVersionObjectReference {
	return autoscalingv2api.CrossVersionObjectReference{
		APIVersion: in.APIVersion.ValueString(),
		Kind:       in.Kind.ValueString(),
		Name:       in.Name.ValueString(),
	}
}

func expandMetrics(ctx context.Context, in types.List) ([]autoscalingv2api.MetricSpec, diag.Diagnostics) {
	if in.IsNull() || in.IsUnknown() {
		return nil, nil
	}
	var metrics []metricModel
	diags := in.ElementsAs(ctx, &metrics, false)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]autoscalingv2api.MetricSpec, 0, len(metrics))
	for i, m := range metrics {
		spec, d := expandMetric(ctx, m)
		for _, e := range d {
			diags.AddError(e.Summary(), fmt.Sprintf("spec.metric[%d]: %s", i, e.Detail()))
		}
		out = append(out, spec)
	}
	return out, diags
}

func expandMetric(ctx context.Context, in metricModel) (autoscalingv2api.MetricSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	spec := autoscalingv2api.MetricSpec{
		Type: autoscalingv2api.MetricSourceType(in.Type.ValueString()),
	}
	if s := in.ContainerResource; s != nil {
		target, d := expandMetricTarget(s.Target)
		diags.Append(d...)
		spec.ContainerResource = &autoscalingv2api.ContainerResourceMetricSource{
			Container: s.Container.ValueString(),
			Name:      corev1.ResourceName(s.Name.ValueString()),
			Target:    target,
		}
	}
	if s := in.External; s != nil {
		target, d := expandMetricTarget(s.Target)
		diags.Append(d...)
		metric, d := expandMetricIdentifier(ctx, s.Metric)
		diags.Append(d...)
		spec.External = &autoscalingv2api.ExternalMetricSource{Metric: metric, Target: target}
	}
	if s := in.Object; s != nil {
		target, d := expandMetricTarget(s.Target)
		diags.Append(d...)
		metric, d := expandMetricIdentifier(ctx, s.Metric)
		diags.Append(d...)
		spec.Object = &autoscalingv2api.ObjectMetricSource{
			DescribedObject: expandCrossVersionObjectReference(s.DescribedObject),
			Metric:          metric,
			Target:          target,
		}
	}
	if s := in.Pods; s != nil {
		target, d := expandMetricTarget(s.Target)
		diags.Append(d...)
		metric, d := expandMetricIdentifier(ctx, s.Metric)
		diags.Append(d...)
		spec.Pods = &autoscalingv2api.PodsMetricSource{Metric: metric, Target: target}
	}
	if s := in.Resource; s != nil {
		target, d := expandMetricTarget(s.Target)
		diags.Append(d...)
		spec.Resource = &autoscalingv2api.ResourceMetricSource{
			Name:   corev1.ResourceName(s.Name.ValueString()),
			Target: target,
		}
	}
	return spec, diags
}

func expandCrossVersionObjectReference(in *crossVersionObjectReference) autoscalingv2api.CrossVersionObjectReference {
	if in == nil {
		return autoscalingv2api.CrossVersionObjectReference{}
	}
	return autoscalingv2api.CrossVersionObjectReference{
		APIVersion: in.APIVersion.ValueString(),
		Kind:       in.Kind.ValueString(),
		Name:       in.Name.ValueString(),
	}
}

func expandMetricTarget(in *metricTargetModel) (autoscalingv2api.MetricTarget, diag.Diagnostics) {
	var diags diag.Diagnostics
	if in == nil {
		return autoscalingv2api.MetricTarget{}, nil
	}
	target := autoscalingv2api.MetricTarget{
		Type: autoscalingv2api.MetricTargetType(in.Type.ValueString()),
	}
	if !in.AverageUtilization.IsNull() && !in.AverageUtilization.IsUnknown() {
		target.AverageUtilization = ptr.To(int32(in.AverageUtilization.ValueInt64()))
	}
	target.AverageValue = expandQuantity(in.AverageValue, "average_value", &diags)
	target.Value = expandQuantity(in.Value, "value", &diags)
	return target, diags
}

func expandQuantity(in types.String, name string, diags *diag.Diagnostics) *resource.Quantity {
	if in.IsNull() || in.IsUnknown() {
		return nil
	}
	q, err := resource.ParseQuantity(in.ValueString())
	if err != nil {
		diags.AddError("Invalid quantity", fmt.Sprintf("target.%s %q: %s", name, in.ValueString(), err))
		return nil
	}
	return &q
}

func expandMetricIdentifier(ctx context.Context, in *metricIdentifierModel) (autoscalingv2api.MetricIdentifier, diag.Diagnostics) {
	if in == nil {
		return autoscalingv2api.MetricIdentifier{}, nil
	}
	selector, diags := expandLabelSelector(ctx, in.Selector)
	return autoscalingv2api.MetricIdentifier{
		Name:     in.Name.ValueString(),
		Selector: selector,
	}, diags
}

func expandLabelSelector(ctx context.Context, in *labelSelectorModel) (*metav1.LabelSelector, diag.Diagnostics) {
	if in == nil {
		return nil, nil
	}
	var diags diag.Diagnostics
	out := &metav1.LabelSelector{}
	if !in.MatchLabels.IsNull() && !in.MatchLabels.IsUnknown() && len(in.MatchLabels.Elements()) > 0 {
		diags.Append(in.MatchLabels.ElementsAs(ctx, &out.MatchLabels, false)...)
	}
	for _, e := range in.MatchExpressions {
		req := metav1.LabelSelectorRequirement{
			Key:      e.Key.ValueString(),
			Operator: metav1.LabelSelectorOperator(e.Operator.ValueString()),
		}
		if !e.Values.IsNull() && !e.Values.IsUnknown() {
			diags.Append(e.Values.ElementsAs(ctx, &req.Values, false)...)
		}
		out.MatchExpressions = append(out.MatchExpressions, req)
	}
	return out, diags
}

func expandBehavior(ctx context.Context, in types.Object) (*autoscalingv2api.HorizontalPodAutoscalerBehavior, diag.Diagnostics) {
	if in.IsNull() || in.IsUnknown() {
		return nil, nil
	}
	var b behaviorModel
	diags := in.As(ctx, &b, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}
	scaleUp, d := expandScalingRules(ctx, b.ScaleUp)
	diags.Append(d...)
	scaleDown, d := expandScalingRules(ctx, b.ScaleDown)
	diags.Append(d...)
	return &autoscalingv2api.HorizontalPodAutoscalerBehavior{
		ScaleUp:   scaleUp,
		ScaleDown: scaleDown,
	}, diags
}

func expandScalingRules(ctx context.Context, in types.Object) (*autoscalingv2api.HPAScalingRules, diag.Diagnostics) {
	if in.IsNull() || in.IsUnknown() {
		return nil, nil
	}
	var r scalingRulesModel
	diags := in.As(ctx, &r, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, diags
	}
	rules := &autoscalingv2api.HPAScalingRules{}
	for _, p := range r.Policy {
		rules.Policies = append(rules.Policies, autoscalingv2api.HPAScalingPolicy{
			Type:          autoscalingv2api.HPAScalingPolicyType(p.Type.ValueString()),
			Value:         int32(p.Value.ValueInt64()),
			PeriodSeconds: int32(p.PeriodSeconds.ValueInt64()),
		})
	}
	if !r.SelectPolicy.IsNull() && !r.SelectPolicy.IsUnknown() {
		rules.SelectPolicy = ptr.To(autoscalingv2api.ScalingPolicySelect(r.SelectPolicy.ValueString()))
	}
	if !r.StabilizationWindowSeconds.IsNull() && !r.StabilizationWindowSeconds.IsUnknown() {
		rules.StabilizationWindowSeconds = ptr.To(int32(r.StabilizationWindowSeconds.ValueInt64()))
	}
	return rules, diags
}

// flattenSpec converts the API spec. prior supplies values Kubernetes does not return or
// spells differently: target_cpu_utilization_percentage, equivalent quantities, and empty
// selector collections.
func flattenSpec(ctx context.Context, in autoscalingv2api.HorizontalPodAutoscalerSpec, prior *specModel) (specModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := specModel{
		MaxReplicas:                    types.Int64Value(int64(in.MaxReplicas)),
		MinReplicas:                    int32PtrValue(in.MinReplicas),
		TargetCPUUtilizationPercentage: types.Int64Null(),
		ScaleTargetRef: []scaleTargetRef{{
			APIVersion: stringOrNull(in.ScaleTargetRef.APIVersion),
			Kind:       types.StringValue(in.ScaleTargetRef.Kind),
			Name:       types.StringValue(in.ScaleTargetRef.Name),
		}},
	}
	priorMetric := types.ListNull(metricElemType)
	if prior != nil {
		out.TargetCPUUtilizationPercentage = prior.TargetCPUUtilizationPercentage
		priorMetric = prior.Metric
	}

	metric, d := flattenMetrics(ctx, in.Metrics, priorMetric)
	diags.Append(d...)
	out.Metric = metric

	behavior, d := flattenBehavior(ctx, in.Behavior)
	diags.Append(d...)
	out.Behavior = behavior

	return out, diags
}

func flattenMetrics(ctx context.Context, in []autoscalingv2api.MetricSpec, prior types.List) (types.List, diag.Diagnostics) {
	if len(in) == 0 {
		return types.ListNull(metricElemType), nil
	}
	var diags diag.Diagnostics
	var priorMetrics []metricModel
	if !prior.IsNull() && !prior.IsUnknown() {
		diags.Append(prior.ElementsAs(ctx, &priorMetrics, false)...)
		if diags.HasError() {
			return types.ListNull(metricElemType), diags
		}
	}
	metrics := make([]metricModel, len(in))
	for i, m := range in {
		metric, d := flattenMetric(ctx, m)
		diags.Append(d...)
		metrics[i] = metric
		if i < len(priorMetrics) {
			preserveMetric(&metrics[i], priorMetrics[i])
		}
	}
	list, d := types.ListValueFrom(ctx, metricElemType, metrics)
	diags.Append(d...)
	return list, diags
}

func flattenMetric(ctx context.Context, in autoscalingv2api.MetricSpec) (metricModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := metricModel{Type: types.StringValue(string(in.Type))}
	if s := in.ContainerResource; s != nil {
		out.ContainerResource = &containerResourceMetricSrc{
			Container: types.StringValue(s.Container),
			Name:      types.StringValue(string(s.Name)),
			Target:    flattenMetricTarget(s.Target),
		}
	}
	if s := in.External; s != nil {
		metric, d := flattenMetricIdentifier(ctx, s.Metric)
		diags.Append(d...)
		out.External = &externalMetricSrc{Metric: metric, Target: flattenMetricTarget(s.Target)}
	}
	if s := in.Object; s != nil {
		metric, d := flattenMetricIdentifier(ctx, s.Metric)
		diags.Append(d...)
		out.Object = &objectMetricSrc{
			DescribedObject: &crossVersionObjectReference{
				APIVersion: types.StringValue(s.DescribedObject.APIVersion),
				Kind:       types.StringValue(s.DescribedObject.Kind),
				Name:       types.StringValue(s.DescribedObject.Name),
			},
			Metric: metric,
			Target: flattenMetricTarget(s.Target),
		}
	}
	if s := in.Pods; s != nil {
		metric, d := flattenMetricIdentifier(ctx, s.Metric)
		diags.Append(d...)
		out.Pods = &podsMetricSrc{Metric: metric, Target: flattenMetricTarget(s.Target)}
	}
	if s := in.Resource; s != nil {
		out.Resource = &resourceMetricSrc{
			Name:   types.StringValue(string(s.Name)),
			Target: flattenMetricTarget(s.Target),
		}
	}
	return out, diags
}

func flattenMetricTarget(in autoscalingv2api.MetricTarget) *metricTargetModel {
	if in.Type == "" && in.AverageUtilization == nil && in.AverageValue == nil && in.Value == nil {
		return nil
	}
	return &metricTargetModel{
		AverageUtilization: int32PtrValue(in.AverageUtilization),
		AverageValue:       quantityValue(in.AverageValue),
		Type:               types.StringValue(string(in.Type)),
		Value:              quantityValue(in.Value),
	}
}

func flattenMetricIdentifier(ctx context.Context, in autoscalingv2api.MetricIdentifier) (*metricIdentifierModel, diag.Diagnostics) {
	out := &metricIdentifierModel{Name: types.StringValue(in.Name)}
	if in.Selector == nil {
		return out, nil
	}
	var diags diag.Diagnostics
	selector := &labelSelectorModel{MatchLabels: types.MapNull(types.StringType)}
	if len(in.Selector.MatchLabels) > 0 {
		selector.MatchLabels, diags = types.MapValueFrom(ctx, types.StringType, in.Selector.MatchLabels)
	}
	for _, e := range in.Selector.MatchExpressions {
		values := types.SetNull(types.StringType)
		if len(e.Values) > 0 {
			var d diag.Diagnostics
			values, d = types.SetValueFrom(ctx, types.StringType, e.Values)
			diags.Append(d...)
		}
		selector.MatchExpressions = append(selector.MatchExpressions, labelSelectorRequirement{
			Key:      types.StringValue(e.Key),
			Operator: types.StringValue(string(e.Operator)),
			Values:   values,
		})
	}
	out.Selector = selector
	return out, diags
}

func flattenBehavior(ctx context.Context, in *autoscalingv2api.HorizontalPodAutoscalerBehavior) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(behaviorType.AttrTypes), nil
	}
	var diags diag.Diagnostics
	scaleUp, d := flattenScalingRules(ctx, in.ScaleUp)
	diags.Append(d...)
	scaleDown, d := flattenScalingRules(ctx, in.ScaleDown)
	diags.Append(d...)
	obj, d := types.ObjectValueFrom(ctx, behaviorType.AttrTypes, behaviorModel{ScaleDown: scaleDown, ScaleUp: scaleUp})
	diags.Append(d...)
	return obj, diags
}

func flattenScalingRules(ctx context.Context, in *autoscalingv2api.HPAScalingRules) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(scalingRulesType.AttrTypes), nil
	}
	rules := scalingRulesModel{
		SelectPolicy:               types.StringNull(),
		StabilizationWindowSeconds: int32PtrValue(in.StabilizationWindowSeconds),
	}
	if in.SelectPolicy != nil {
		rules.SelectPolicy = types.StringValue(string(*in.SelectPolicy))
	}
	for _, p := range in.Policies {
		rules.Policy = append(rules.Policy, scalingPolicyModel{
			PeriodSeconds: types.Int64Value(int64(p.PeriodSeconds)),
			Type:          types.StringValue(string(p.Type)),
			Value:         types.Int64Value(int64(p.Value)),
		})
	}
	return types.ObjectValueFrom(ctx, scalingRulesType.AttrTypes, rules)
}

// preserveMetric keeps prior values that Kubernetes returns in a different but equivalent
// form, matching metrics by position because Kubernetes preserves their order.
func preserveMetric(actual *metricModel, prior metricModel) {
	if a, p := actual.ContainerResource, prior.ContainerResource; a != nil && p != nil {
		preserveTarget(a.Target, p.Target)
	}
	if a, p := actual.External, prior.External; a != nil && p != nil {
		preserveTarget(a.Target, p.Target)
		preserveIdentifier(a.Metric, p.Metric)
	}
	if a, p := actual.Object, prior.Object; a != nil && p != nil {
		preserveTarget(a.Target, p.Target)
		preserveIdentifier(a.Metric, p.Metric)
	}
	if a, p := actual.Pods, prior.Pods; a != nil && p != nil {
		preserveTarget(a.Target, p.Target)
		preserveIdentifier(a.Metric, p.Metric)
	}
	if a, p := actual.Resource, prior.Resource; a != nil && p != nil {
		preserveTarget(a.Target, p.Target)
	}
}

func preserveTarget(actual, prior *metricTargetModel) {
	if actual == nil || prior == nil {
		return
	}
	if quantitiesEqual(actual.Value, prior.Value) {
		actual.Value = prior.Value
	}
	if quantitiesEqual(actual.AverageValue, prior.AverageValue) {
		actual.AverageValue = prior.AverageValue
	}
}

// preserveIdentifier keeps an empty selector, match_labels, match_expressions or values
// from prior state where Kubernetes returns none, since configuration can set them empty.
func preserveIdentifier(actual, prior *metricIdentifierModel) {
	if actual == nil || prior == nil || prior.Selector == nil {
		return
	}
	p := prior.Selector
	if actual.Selector == nil {
		if len(p.MatchLabels.Elements()) == 0 && len(p.MatchExpressions) == 0 {
			actual.Selector = p
		}
		return
	}
	a := actual.Selector
	if a.MatchLabels.IsNull() && !p.MatchLabels.IsNull() && len(p.MatchLabels.Elements()) == 0 {
		a.MatchLabels = p.MatchLabels
	}
	if a.MatchExpressions == nil && p.MatchExpressions != nil && len(p.MatchExpressions) == 0 {
		a.MatchExpressions = p.MatchExpressions
	}
	for i := range a.MatchExpressions {
		if i >= len(p.MatchExpressions) {
			break
		}
		pv := p.MatchExpressions[i].Values
		if a.MatchExpressions[i].Values.IsNull() && !pv.IsNull() && len(pv.Elements()) == 0 {
			a.MatchExpressions[i].Values = pv
		}
	}
}

func int32PtrValue(v *int32) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

func quantityValue(q *resource.Quantity) types.String {
	if q == nil {
		return types.StringNull()
	}
	return types.StringValue(q.String())
}

func stringOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
