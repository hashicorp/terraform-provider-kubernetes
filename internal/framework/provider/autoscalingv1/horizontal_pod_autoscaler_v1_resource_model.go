// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv1

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type HorizontalPodAutoscalerV1Model struct {
	ID       types.String                     `tfsdk:"id"`
	Metadata []common.NamespacedMetadataModel `tfsdk:"metadata"`
	Spec     []HorizontalPodAutoscalerV1Spec  `tfsdk:"spec"`
}

type HorizontalPodAutoscalerV1Spec struct {
	MaxReplicas                    types.Int64      `tfsdk:"max_replicas"`
	MinReplicas                    types.Int64      `tfsdk:"min_replicas"`
	ScaleTargetRef                 []ScaleTargetRef `tfsdk:"scale_target_ref"`
	TargetCPUUtilizationPercentage types.Int64      `tfsdk:"target_cpu_utilization_percentage"`
}

type ScaleTargetRef struct {
	APIVersion types.String `tfsdk:"api_version"`
	Kind       types.String `tfsdk:"kind"`
	Name       types.String `tfsdk:"name"`
}

// sdkv2HorizontalPodAutoscalerStateV0 is the state written by the SDKv2
// kubernetes_horizontal_pod_autoscaler resource. Its spec uses the shared V2 schema, so
// metric and behavior are decoded only to check that they are empty.
type sdkv2HorizontalPodAutoscalerStateV0 struct {
	ID       string                             `json:"id"`
	Metadata []common.SDKv2NamespacedMetadataV0 `json:"metadata"`
	Spec     []sdkv2HorizontalPodAutoscalerSpec `json:"spec"`
}

type sdkv2HorizontalPodAutoscalerSpec struct {
	MaxReplicas                    int64                 `json:"max_replicas"`
	MinReplicas                    int64                 `json:"min_replicas"`
	ScaleTargetRef                 []sdkv2ScaleTargetRef `json:"scale_target_ref"`
	TargetCPUUtilizationPercentage int64                 `json:"target_cpu_utilization_percentage"`
	Metric                         []json.RawMessage     `json:"metric"`
	Behavior                       []json.RawMessage     `json:"behavior"`
}

type sdkv2ScaleTargetRef struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}
