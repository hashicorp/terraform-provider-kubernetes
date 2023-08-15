// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	autoscalingv2beta2 "k8s.io/api/autoscaling/v2beta2"
)

func horizontalPodAutoscalerSchemaV2() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("horizontal pod autoscaler", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Behaviour of the autoscaler. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"max_replicas": {
						Type:        schema.TypeInt,
						Description: "Upper limit for the number of pods that can be set by the autoscaler.",
						Required:    true,
					},
					"metric": {
						Type:        schema.TypeList,
						Computed:    true,
						Optional:    true,
						Description: "The specifications for which to use to calculate the desired replica count (the maximum replica count across all metrics will be used). The desired replica count is calculated multiplying the ratio between the target value and the current value by the current number of pods. Ergo, metrics used must decrease as the pod count is increased, and vice-versa. See the individual metric source types for more information about how each type of metric must respond. If not set, the default metric will be set to 80% average CPU utilization.",
						Elem:        metricSpecFields(),
					},
					"min_replicas": {
						Type:        schema.TypeInt,
						Description: "Lower limit for the number of pods that can be set by the autoscaler, defaults to `1`.",
						Optional:    true,
						Default:     1,
					},
					"behavior": {
						Type:        schema.TypeList,
						Description: "Behavior configures the scaling behavior of the target in both Up and Down directions (`scale_up` and `scale_down` fields respectively).",
						Optional:    true,
						Computed:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"scale_up": {
									Type:        schema.TypeList,
									Description: "Scaling policy for scaling Up",
									Optional:    true,
									Computed:    true,
									Elem:        scalingRulesSpecFields(),
								},
								"scale_down": {
									Type:        schema.TypeList,
									Description: "Scaling policy for scaling Down",
									Optional:    true,
									Computed:    true,
									Elem:        scalingRulesSpecFields(),
								},
							},
						},
					},
					"scale_target_ref": {
						Type:        schema.TypeList,
						Description: "Reference to scaled resource. e.g. Replication Controller",
						Required:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"api_version": {
									Type:        schema.TypeString,
									Description: "API version of the referent",
									Optional:    true,
								},
								"kind": {
									Type:        schema.TypeString,
									Description: "Kind of the referent. e.g. `ReplicationController`. More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds",
									Required:    true,
								},
								"name": {
									Type:        schema.TypeString,
									Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
									Required:    true,
								},
							},
						},
					},
					"target_cpu_utilization_percentage": {
						Type:        schema.TypeInt,
						Description: "Target average CPU utilization (represented as a percentage of requested CPU) over all the pods. If not specified the default autoscaling policy will be used.",
						Optional:    true,
						Computed:    true,
					},
				},
			},
		},
	}
}

func metricTargetFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"average_utilization": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "averageUtilization is the target value of the average of the resource metric across all relevant pods, represented as a percentage of the requested value of the resource for the pods. Currently only valid for Resource metric source type",
			},
			"average_value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "averageValue is the target value of the average of the metric across all relevant pods (as a quantity)",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "type represents whether the metric type is Utilization, Value, or AverageValue",
			},
			"value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "value is the target value of the metric (as a quantity).",
			},
		},
	}
}

func resourceMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "name is the name of the resource in question.",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "Target specifies the target value for the given metric",
			},
		},
	}
}

func metricIdentifierFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "name is the name of the given metric",
			},
			"selector": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "selector is the string-encoded form of a standard kubernetes label selector for the given metric When set, it is passed as an additional parameter to the metrics server for more specific metrics scoping. When unset, just the metricName will be used to gather metrics.",
				Elem: &schema.Resource{
					Schema: labelSelectorFields(true),
				},
			},
		},
	}
}

func podsMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"metric": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem:        metricIdentifierFields(),
				Description: "metric identifies the target metric by name and selector",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "target specifies the target value for the given metric",
			},
		},
	}
}

func externalMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"metric": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem:        metricIdentifierFields(),
				Description: "metric identifies the target metric by name and selector",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "target specifies the target value for the given metric",
			},
		},
	}
}

func containerResourceMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"container": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "name of the container in the pods of the scaling target",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "name of the resource in question",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "target specifies the target value for the given metric",
			},
		},
	}
}

func crossVersionObjectReferenceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"api_version": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "API version of the referent",
			},
			"kind": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the referent; More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
			},
		},
	}
}

func objectMetricSourceFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"described_object": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem:     crossVersionObjectReferenceFields(),
				// NOTE Description is undocumented in K8s API docs
			},
			"metric": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Elem:        metricIdentifierFields(),
				Description: "metric identifies the target metric by name and selector",
			},
			"target": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        metricTargetFields(),
				Description: "target specifies the target value for the given metric",
			},
		},
	}
}

func metricSpecFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"container_resource": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        containerResourceMetricSourceFields(),
				Description: "",
			},
			"external": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        externalMetricSourceFields(),
				Description: "",
			},
			"object": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        objectMetricSourceFields(),
				Description: "",
			},
			"pods": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        podsMetricSourceFields(),
				Description: "",
			},
			"resource": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Elem:        resourceMetricSourceFields(),
				Description: "",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: `type is the type of metric source. It should be one of "ContainerResource", "External", "Object", "Pods" or "Resource", each mapping to a matching field in the object. Note: "ContainerResource" type is available on when the feature-gate HPAContainerMetrics is enabled`,
				ValidateFunc: validation.StringInSlice([]string{
					string(autoscalingv2beta2.ContainerResourceMetricSourceType),
					string(autoscalingv2beta2.ExternalMetricSourceType),
					string(autoscalingv2beta2.ObjectMetricSourceType),
					string(autoscalingv2beta2.PodsMetricSourceType),
					string(autoscalingv2beta2.ResourceMetricSourceType),
				}, false),
			},
		},
	}
}

func scalingRulesSpecFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"policy": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Elem:        scalingPolicySpecFields(),
				Description: "List of potential scaling polices which can be used during scaling. At least one policy must be specified, otherwise the scaling rule will be discarded as invalid.",
			},
			"select_policy": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Used to specify which policy should be used. If not set, the default value Max is used.",
			},
			"stabilization_window_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of seconds for which past recommendations should be considered while scaling up or scaling down. This value must be greater than or equal to zero and less than or equal to 3600 (one hour). If not set, use the default values: - For scale up: 0 (i.e. no stabilization is done). - For scale down: 300 (i.e. the stabilization window is 300 seconds long).",
			},
		},
	}
}

func scalingPolicySpecFields() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"period_seconds": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Period specifies the window of time for which the policy should hold true. PeriodSeconds must be greater than zero and less than or equal to 1800 (30 min).",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Type is used to specify the scaling policy: Percent or Pods",
			},
			"value": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Value contains the amount of change which is permitted by the policy. It must be greater than zero.",
			},
		},
	}
}
