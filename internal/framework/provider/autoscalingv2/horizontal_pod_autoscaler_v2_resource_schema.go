// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package autoscalingv2

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	autoscalingv2api "k8s.io/api/autoscaling/v2"
)

// schemaVersion 1 replaces the SDKv2 list blocks under spec.metric and spec.behavior with
// nested attributes.
const schemaVersion = 1

// singleRequiredBlock reproduces SDKv2's Required + MaxItems: 1 for a list block.
func singleRequiredBlock() []validator.List {
	return []validator.List{
		listvalidator.IsRequired(),
		listvalidator.SizeAtLeast(1),
		listvalidator.SizeAtMost(1),
	}
}

func (r *HorizontalPodAutoscalerV2) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = hpaSchema()
}

func hpaSchema() schema.Schema {
	return schema.Schema{
		Version:     schemaVersion,
		Description: "Horizontal Pod Autoscaler automatically scales the number of pods in a replication controller, deployment or replica set based on observed CPU utilization or custom metrics.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedMetadataSchema("horizontal pod autoscaler", true),
			"spec": schema.ListNestedBlock{
				Description: "Behaviour of the autoscaler. Exactly one spec block is required. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
				Validators:  singleRequiredBlock(),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"max_replicas": schema.Int64Attribute{
							Description: "Upper limit for the number of pods that can be set by the autoscaler.",
							Required:    true,
						},
						"min_replicas": schema.Int64Attribute{
							Description: "Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(1),
						},
						"target_cpu_utilization_percentage": schema.Int64Attribute{
							Description: "Ignored by `autoscaling/v2` and never sent to Kubernetes. Use a `Resource` metric instead.",
							Optional:    true,
						},
						"metric":   metricAttribute(),
						"behavior": behaviorAttribute(),
					},
					Blocks: map[string]schema.Block{
						"scale_target_ref": schema.ListNestedBlock{
							Description: "Reference to scaled resource. e.g. Replication Controller. Exactly one scale_target_ref block is required.",
							Validators:  singleRequiredBlock(),
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"api_version": schema.StringAttribute{
										Description: "API version of the referent",
										Optional:    true,
									},
									"kind": schema.StringAttribute{
										Description: "Kind of the referent. e.g. `ReplicationController`. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds",
										Required:    true,
									},
									"name": schema.StringAttribute{
										Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
										Required:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func metricAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Description: "The specifications for which to use to calculate the desired replica count (the maximum replica count across all metrics will be used). The desired replica count is calculated multiplying the ratio between the target value and the current value by the current number of pods. Ergo, metrics used must decrease as the pod count is increased, and vice-versa. See the individual metric source types for more information about how each type of metric must respond. If not set, Kubernetes sets an 80% average CPU utilization metric. Removing the attribute keeps the current metrics.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
		Validators: []validator.List{
			common.NotEmptyList(),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Description: `type is the type of metric source. It should be one of "ContainerResource", "External", "Object", "Pods" or "Resource", each mapping to a matching field in the object.`,
					Required:    true,
					Validators: []validator.String{
						stringvalidator.OneOf(
							string(autoscalingv2api.ContainerResourceMetricSourceType),
							string(autoscalingv2api.ExternalMetricSourceType),
							string(autoscalingv2api.ObjectMetricSourceType),
							string(autoscalingv2api.PodsMetricSourceType),
							string(autoscalingv2api.ResourceMetricSourceType),
						),
					},
				},
				"container_resource": schema.SingleNestedAttribute{
					Description: "A resource metric known to Kubernetes, as specified in requests and limits, describing a single container in each pod of the current scale target.",
					Optional:    true,
					Attributes: map[string]schema.Attribute{
						"container": schema.StringAttribute{
							Description: "name of the container in the pods of the scaling target",
							Required:    true,
						},
						"name": schema.StringAttribute{
							Description: "name of the resource in question",
							Required:    true,
						},
						"target": metricTargetAttribute(),
					},
				},
				"external": schema.SingleNestedAttribute{
					Description: "A global metric that is not associated with any Kubernetes object.",
					Optional:    true,
					Attributes: map[string]schema.Attribute{
						"metric": metricIdentifierAttribute(),
						"target": metricTargetAttribute(),
					},
				},
				"object": schema.SingleNestedAttribute{
					Description: "A metric describing a single kubernetes object.",
					Optional:    true,
					Attributes: map[string]schema.Attribute{
						"described_object": schema.SingleNestedAttribute{
							Description: "The object the metric describes.",
							Required:    true,
							Attributes: map[string]schema.Attribute{
								"api_version": schema.StringAttribute{
									Description: "API version of the referent",
									Required:    true,
								},
								"kind": schema.StringAttribute{
									Description: "Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
									Required:    true,
								},
								"name": schema.StringAttribute{
									Description: "Name of the referent; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
									Required:    true,
								},
							},
						},
						"metric": metricIdentifierAttribute(),
						"target": metricTargetAttribute(),
					},
				},
				"pods": schema.SingleNestedAttribute{
					Description: "A metric describing each pod in the current scale target.",
					Optional:    true,
					Attributes: map[string]schema.Attribute{
						"metric": metricIdentifierAttribute(),
						"target": metricTargetAttribute(),
					},
				},
				"resource": schema.SingleNestedAttribute{
					Description: "A resource metric known to Kubernetes, as specified in requests and limits, describing each pod in the current scale target.",
					Optional:    true,
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "name is the name of the resource in question.",
							Required:    true,
						},
						"target": metricTargetAttribute(),
					},
				},
			},
		},
	}
}

func metricTargetAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "target specifies the target value for the given metric",
		Optional:    true,
		Attributes: map[string]schema.Attribute{
			"average_utilization": schema.Int64Attribute{
				Description: "averageUtilization is the target value of the average of the resource metric across all relevant pods, represented as a percentage of the requested value of the resource for the pods. Currently only valid for Resource metric source type",
				Optional:    true,
			},
			"average_value": schema.StringAttribute{
				Description: "averageValue is the target value of the average of the metric across all relevant pods (as a quantity)",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					quantityPlanModifier{},
				},
			},
			"type": schema.StringAttribute{
				Description: "type represents whether the metric type is Utilization, Value, or AverageValue",
				Required:    true,
			},
			"value": schema.StringAttribute{
				Description: "value is the target value of the metric (as a quantity).",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					quantityPlanModifier{},
				},
			},
		},
	}
}

func metricIdentifierAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "metric identifies the target metric by name and selector",
		Required:    true,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "name is the name of the given metric",
				Required:    true,
			},
			"selector": schema.SingleNestedAttribute{
				Description: "selector is the string-encoded form of a standard kubernetes label selector for the given metric When set, it is passed as an additional parameter to the metrics server for more specific metrics scoping. When unset, just the metricName will be used to gather metrics.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"match_expressions": schema.ListNestedAttribute{
						Description: "A list of label selector requirements. The requirements are ANDed.",
						Optional:    true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									Description: "The label key that the selector applies to.",
									Optional:    true,
								},
								"operator": schema.StringAttribute{
									Description: "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.",
									Optional:    true,
								},
								"values": schema.SetAttribute{
									Description: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
									Optional:    true,
									ElementType: types.StringType,
								},
							},
						},
					},
					"match_labels": schema.MapAttribute{
						Description: "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func behaviorAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: "Behavior configures the scaling behavior of the target in both Up and Down directions (`scale_up` and `scale_down` fields respectively). If not set, Kubernetes uses its default scaling behavior. Removing the attribute keeps the current behavior.",
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
		Attributes: map[string]schema.Attribute{
			"scale_down": scalingRulesAttribute("Scaling policy for scaling Down. If not set, Kubernetes scales down by up to 100% every 15 seconds after a 300 second stabilization window."),
			"scale_up":   scalingRulesAttribute("Scaling policy for scaling Up. If not set, Kubernetes scales up by the larger of 4 pods or 100% every 15 seconds, with no stabilization window."),
		},
	}
}

func scalingRulesAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.Object{
			useStateForUnknownInExistingParent{},
		},
		Attributes: map[string]schema.Attribute{
			"policy": schema.ListNestedAttribute{
				Description: "List of potential scaling polices which can be used during scaling. At least one policy must be specified, otherwise the scaling rule will be discarded as invalid.",
				Required:    true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"period_seconds": schema.Int64Attribute{
							Description: "Period specifies the window of time for which the policy should hold true. PeriodSeconds must be greater than zero and less than or equal to 1800 (30 min).",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Type is used to specify the scaling policy: Percent or Pods",
							Required:    true,
						},
						"value": schema.Int64Attribute{
							Description: "Value contains the amount of change which is permitted by the policy. It must be greater than zero.",
							Required:    true,
						},
					},
				},
			},
			"select_policy": schema.StringAttribute{
				Description: "Used to specify which policy should be used. One of `Max`, `Min` or `Disabled`. If not set, Kubernetes uses `Max`.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					useStateForUnknownInExistingParent{},
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(autoscalingv2api.MaxChangePolicySelect),
						string(autoscalingv2api.MinChangePolicySelect),
						string(autoscalingv2api.DisabledPolicySelect),
					),
				},
			},
			"stabilization_window_seconds": schema.Int64Attribute{
				Description: "Number of seconds for which past recommendations should be considered while scaling up or scaling down. This value must be greater than or equal to zero and less than or equal to 3600 (one hour). If not set, Kubernetes uses 0 for scale up and 300 for scale down.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					useStateForUnknownInExistingParent{},
				},
			},
		},
	}
}

// Types of the Computed attributes, derived from the schema so the two cannot drift.
var (
	metricElemType   = metricAttribute().GetType().(types.ListType).ElemType.(types.ObjectType)
	behaviorType     = behaviorAttribute().GetType().(types.ObjectType)
	scalingRulesType = behaviorType.AttrTypes["scale_up"].(types.ObjectType)
)
