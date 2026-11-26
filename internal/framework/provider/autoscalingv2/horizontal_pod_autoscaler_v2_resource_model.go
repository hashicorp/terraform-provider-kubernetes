// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type HorizontalPodAutoscalerV2Model struct {
	ID       types.String                     `tfsdk:"id"`
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     []specModel                      `tfsdk:"spec"`
}

type specModel struct {
	MaxReplicas                    types.Int64      `tfsdk:"max_replicas"`
	MinReplicas                    types.Int64      `tfsdk:"min_replicas"`
	TargetCPUUtilizationPercentage types.Int64      `tfsdk:"target_cpu_utilization_percentage"`
	Metric                         types.List       `tfsdk:"metric"`
	Behavior                       types.Object     `tfsdk:"behavior"`
	ScaleTargetRef                 []scaleTargetRef `tfsdk:"scale_target_ref"`
}

type scaleTargetRef struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}

// Metric elements have no Computed attributes, so a pointer represents a null object.

type metricModel struct {
	Type              types.String                `tfsdk:"type"`
	ContainerResource *containerResourceMetricSrc `tfsdk:"container_resource"`
	External          *externalMetricSrc          `tfsdk:"external"`
	Object            *objectMetricSrc            `tfsdk:"object"`
	Pods              *podsMetricSrc              `tfsdk:"pods"`
	Resource          *resourceMetricSrc          `tfsdk:"resource"`
}

type containerResourceMetricSrc struct {
	Container types.String       `tfsdk:"container"`
	Name      types.String       `tfsdk:"name"`
	Target    *metricTargetModel `tfsdk:"target"`
}

type externalMetricSrc struct {
	Metric *metricIdentifierModel `tfsdk:"metric"`
	Target *metricTargetModel     `tfsdk:"target"`
}

type objectMetricSrc struct {
	DescribedObject *crossVersionObjectReference `tfsdk:"described_object"`
	Metric          *metricIdentifierModel       `tfsdk:"metric"`
	Target          *metricTargetModel           `tfsdk:"target"`
}

type podsMetricSrc struct {
	Metric *metricIdentifierModel `tfsdk:"metric"`
	Target *metricTargetModel     `tfsdk:"target"`
}

type resourceMetricSrc struct {
	Name   types.String       `tfsdk:"name"`
	Target *metricTargetModel `tfsdk:"target"`
}

type crossVersionObjectReference struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}

type metricTargetModel struct {
	AverageUtilization types.Int64  `tfsdk:"average_utilization"`
	AverageValue       types.String `tfsdk:"average_value"`
	Type               types.String `tfsdk:"type"`
	Value              types.String `tfsdk:"value"`
}

type metricIdentifierModel struct {
	Name     types.String        `tfsdk:"name"`
	Selector *labelSelectorModel `tfsdk:"selector"`
}

// labelSelectorModel keeps a nil slice (null) distinct from an empty one.
type labelSelectorModel struct {
	MatchExpressions []labelSelectorRequirement `tfsdk:"match_expressions"`
	MatchLabels      types.Map                  `tfsdk:"match_labels"`
}

type labelSelectorRequirement struct {
	Key      types.String `tfsdk:"key"`
	Operator types.String `tfsdk:"operator"`
	Values   types.Set    `tfsdk:"values"`
}

// behaviorModel and scalingRulesModel hold Optional+Computed values, which can be unknown
// in a plan.
type behaviorModel struct {
	ScaleDown types.Object `tfsdk:"scale_down"`
	ScaleUp   types.Object `tfsdk:"scale_up"`
}

type scalingRulesModel struct {
	Policy                     []scalingPolicyModel `tfsdk:"policy"`
	SelectPolicy               types.String         `tfsdk:"select_policy"`
	StabilizationWindowSeconds types.Int64          `tfsdk:"stabilization_window_seconds"`
}

type scalingPolicyModel struct {
	PeriodSeconds types.Int64  `tfsdk:"period_seconds"`
	Type          types.String `tfsdk:"type"`
	Value         types.Int64  `tfsdk:"value"`
}
