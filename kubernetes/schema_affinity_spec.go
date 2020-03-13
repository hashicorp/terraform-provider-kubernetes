package kubernetes

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func affinityFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"node_affinity": {
			Type:        schema.TypeList,
			Description: "Node affinity scheduling rules for the pod.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: nodeAffinityFields(),
			},
		},
		"pod_affinity": {
			Type:        schema.TypeList,
			Description: "Inter-pod topological affinity. rules that specify that certain pods should be placed in the same topological domain (e.g. same node, same rack, same zone, same power domain, etc.)",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podAffinityFields(),
			},
		},
		"pod_anti_affinity": {
			Type:        schema.TypeList,
			Description: "Inter-pod topological affinity. rules that specify that certain pods should be placed in the same topological domain (e.g. same node, same rack, same zone, same power domain, etc.)",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podAffinityFields(),
			},
		},
	}
}

func nodeAffinityFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"required_during_scheduling_ignored_during_execution": {
			Type:        schema.TypeList,
			Description: "If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a node label update), the system may or may not try to eventually evict the pod from its node.",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: nodeSelectorFields(),
			},
		},
		"preferred_during_scheduling_ignored_during_execution": {
			Type:        schema.TypeList,
			Description: "The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, RequiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding 'weight' to the sum if the node matches the corresponding MatchExpressions; the node(s) with the highest sum are the most preferred.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: preferredSchedulingTermFields(),
			},
		},
	}
}

func nodeSelectorFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"node_selector_term": {
			Type:        schema.TypeList,
			Description: "List of node selector terms. The terms are ORed.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: nodeSelectorRequirementsFields(),
			},
		},
	}
}

func preferredSchedulingTermFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"weight": {
			Type:        schema.TypeInt,
			Description: "weight is in the range 1-100",
			Required:    true,
		},
		"preference": {
			Type:        schema.TypeList,
			Description: "A node selector term, associated with the corresponding weight.",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: nodeSelectorRequirementsFields(),
			},
		},
	}
}

func nodeSelectorRequirementsFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"match_expressions": {
			Type:        schema.TypeList,
			Description: "List of node selector requirements. The requirements are ANDed.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"key": {
						Type:        schema.TypeString,
						Description: "The label key that the selector applies to.",
						Optional:    true,
					},
					"operator": {
						Type:         schema.TypeString,
						Description:  "Operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.",
						Optional:     true,
						ValidateFunc: validation.StringInSlice([]string{"In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt"}, false),
					},
					"values": {
						Type:        schema.TypeSet,
						Description: "Values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.",
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Set:         schema.HashString,
					},
				},
			},
		},
	}
}

func podAffinityFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"required_during_scheduling_ignored_during_execution": {
			Type:        schema.TypeList,
			Description: "If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each PodAffinityTerm are intersected, i.e. all terms must be satisfied.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: podAffinityTermFields(),
			},
		},
		"preferred_during_scheduling_ignored_during_execution": {
			Type:        schema.TypeList,
			Description: "The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, RequiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding 'weight' to the sum if the node matches the corresponding MatchExpressions; the node(s) with the highest sum are the most preferred.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: weightedPodAffinityTermFields(),
			},
		},
	}
}

func podAffinityTermFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"label_selector": {
			Type:        schema.TypeList,
			Description: "A label query over a set of resources, in this case pods.",
			Optional:    true,
			Elem: &schema.Resource{
				Schema: labelSelectorFields(false),
			},
		},
		"namespaces": {
			Type:        schema.TypeSet,
			Description: "namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means 'this pod's namespace'",
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Set:         schema.HashString,
		},
		"topology_key": {
			Type:         schema.TypeString,
			Description:  "empty topology key is interpreted by the scheduler as 'all topologies'",
			Optional:     true,
			ValidateFunc: validation.StringMatch(regexp.MustCompile(`^.+$`), "value cannot be empty"),
		},
	}
}

func weightedPodAffinityTermFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"weight": {
			Type:        schema.TypeInt,
			Description: "weight associated with matching the corresponding podAffinityTerm, in the range 1-100",
			Required:    true,
		},
		"pod_affinity_term": {
			Type:        schema.TypeList,
			Description: "A pod affinity term, associated with the corresponding weight",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: podAffinityTermFields(),
			},
		},
	}
}
