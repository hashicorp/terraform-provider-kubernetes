// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func workloadMatchExpressionsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Description: "The label key that the selector applies to.",
				Optional:    true,
			},
			"operator": schema.StringAttribute{
				Description: "Operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.OneOf("In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt")},
			},
			"values": schema.SetAttribute{
				Description: "Values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func workloadMatchFieldsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Description:   "The label key that the selector applies to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"operator": schema.StringAttribute{
				Description:   "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists`, `DoesNotExist`, `Gt`, and `Lt`.",
				Required:      true,
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"values": schema.SetAttribute{
				Description:   "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{workloadSetRequiresReplace()},
			},
		},
	}
}

func workloadPreferenceObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"match_expressions": schema.ListNestedBlock{
				Description:  "List of node selector requirements. The requirements are ANDed.",
				NestedObject: workloadMatchExpressionsObject(updatable),
			},
			"match_fields": schema.ListNestedBlock{
				Description:   "A list of node selector requirements by node's fields. The requirements are ANDed.",
				NestedObject:  workloadMatchFieldsObject(updatable),
				PlanModifiers: []planmodifier.List{workloadObjectListRequiresReplace()},
			},
		},
	}
}

func workloadPreferredDuringSchedulingIgnoredDuringExecutionObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"weight": schema.Int64Attribute{
				Description: "weight is in the range 1-100",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"preference": schema.ListNestedBlock{
				Description:  "A node selector term, associated with the corresponding weight.",
				NestedObject: workloadPreferenceObject(updatable),
				Validators:   []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
			},
		},
	}
}

func workloadRequiredDuringSchedulingIgnoredDuringExecutionObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"node_selector_term": schema.ListNestedBlock{
				Description:  "List of node selector terms. The terms are ORed.",
				NestedObject: workloadPreferenceObject(updatable),
			},
		},
	}
}

func workloadNodeAffinityObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"preferred_during_scheduling_ignored_during_execution": schema.ListNestedBlock{
				Description:  "The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, RequiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding 'weight' to the sum if the node matches the corresponding MatchExpressions; the node(s) with the highest sum are the most preferred.",
				NestedObject: workloadPreferredDuringSchedulingIgnoredDuringExecutionObject(updatable),
			},
			"required_during_scheduling_ignored_during_execution": schema.ListNestedBlock{
				Description:  "If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a node label update), the system may or may not try to eventually evict the pod from its node.",
				NestedObject: workloadRequiredDuringSchedulingIgnoredDuringExecutionObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadMatchExpressionsObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
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
	}
}

func workloadLabelSelectorObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"match_labels": schema.MapAttribute{
				Description: "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"match_expressions": schema.ListNestedBlock{
				Description:  "A list of label selector requirements. The requirements are ANDed.",
				NestedObject: workloadMatchExpressionsObject2(updatable),
			},
		},
	}
}

func workloadPodAffinityTermObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"namespaces": schema.SetAttribute{
				Description: "namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means 'this pod's namespace'",
				Optional:    true,
				ElementType: types.StringType,
			},
			"topology_key": schema.StringAttribute{
				Description: "empty topology key is interpreted by the scheduler as 'all topologies'",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"label_selector": schema.ListNestedBlock{
				Description:  "A label query over a set of resources, in this case pods.",
				NestedObject: workloadLabelSelectorObject(updatable),
			},
			"namespace_selector": schema.ListNestedBlock{
				Description:  "A label query over a set of namespaces that matches the namespaceSelector in Kubernetes.",
				NestedObject: workloadLabelSelectorObject(updatable),
			},
		},
	}
}

func workloadPreferredDuringSchedulingIgnoredDuringExecutionObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"weight": schema.Int64Attribute{
				Description: "weight associated with matching the corresponding podAffinityTerm, in the range 1-100",
				Required:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"pod_affinity_term": schema.ListNestedBlock{
				Description:  "A pod affinity term, associated with the corresponding weight",
				NestedObject: workloadPodAffinityTermObject(updatable),
				Validators:   []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
			},
		},
	}
}

func workloadPodAffinityObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"preferred_during_scheduling_ignored_during_execution": schema.ListNestedBlock{
				Description:  "The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, RequiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding 'weight' to the sum if the node matches the corresponding MatchExpressions; the node(s) with the highest sum are the most preferred.",
				NestedObject: workloadPreferredDuringSchedulingIgnoredDuringExecutionObject2(updatable),
			},
			"required_during_scheduling_ignored_during_execution": schema.ListNestedBlock{
				Description:  "If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each PodAffinityTerm are intersected, i.e. all terms must be satisfied.",
				NestedObject: workloadPodAffinityTermObject(updatable),
			},
		},
	}
}

func workloadAffinityObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"node_affinity": schema.ListNestedBlock{
				Description:  "Node affinity scheduling rules for the pod.",
				NestedObject: workloadNodeAffinityObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"pod_affinity": schema.ListNestedBlock{
				Description:  "Inter-pod topological affinity. rules that specify that certain pods should be placed in the same topological domain (e.g. same node, same rack, same zone, same power domain, etc.)",
				NestedObject: workloadPodAffinityObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"pod_anti_affinity": schema.ListNestedBlock{
				Description:  "Inter-pod topological affinity. rules that specify that certain pods should be placed in the same topological domain (e.g. same node, same rack, same zone, same power domain, etc.)",
				NestedObject: workloadPodAffinityObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}
