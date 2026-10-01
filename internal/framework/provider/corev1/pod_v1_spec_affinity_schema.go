// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func podAffinityObject() schema.NestedBlockObject {
	nodeTerm := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"match_expressions": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"key":      podString(false, false, false, ""),
			"operator": podString(false, false, false, "", stringvalidator.OneOf("In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt")),
			"values":   podSet(false, types.StringType),
		}}, 0, 0, false),
		"match_fields": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"key":      podString(true, false, true, ""),
			"operator": podString(true, false, true, ""),
			"values":   podSet(true, types.StringType),
		}}, 0, 0, true),
	}}
	node := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"required_during_scheduling_ignored_during_execution": podBlock(schema.NestedBlockObject{Blocks: map[string]schema.Block{
			"node_selector_term": podBlock(nodeTerm, 0, 0, false),
		}}, 0, 1, false),
		"preferred_during_scheduling_ignored_during_execution": podBlock(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"weight": podInt(true, false, false, 0)},
			Blocks:     map[string]schema.Block{"preference": podBlock(nodeTerm, 1, 1, false)},
		}, 0, 0, false),
	}}
	pod := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"required_during_scheduling_ignored_during_execution": podBlock(podAffinityTermObject(), 0, 0, false),
		"preferred_during_scheduling_ignored_during_execution": podBlock(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{"weight": podInt(true, false, false, 0)},
			Blocks:     map[string]schema.Block{"pod_affinity_term": podBlock(podAffinityTermObject(), 1, 1, false)},
		}, 0, 0, false),
	}}
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"node_affinity":     podBlock(node, 0, 1, false),
		"pod_affinity":      podBlock(pod, 0, 1, false),
		"pod_anti_affinity": podBlock(pod, 0, 1, false),
	}}
}

func podAffinityTermObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"namespaces":   podSet(false, types.StringType),
			"topology_key": podString(true, false, false, ""),
		},
		Blocks: map[string]schema.Block{
			"label_selector":     podBlock(podLabelSelectorObject(), 0, 0, false),
			"namespace_selector": podBlock(podLabelSelectorObject(), 0, 0, false),
		},
	}
}

func podLabelSelectorObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{"match_labels": podMap(false, false)},
		Blocks: map[string]schema.Block{
			"match_expressions": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"key":      podString(false, false, false, ""),
				"operator": podString(false, false, false, ""),
				"values":   podSet(false, types.StringType),
			}}, 0, 0, false),
		},
	}
}
