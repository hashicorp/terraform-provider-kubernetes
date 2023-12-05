// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func nodeSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"pod_cidr": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    false,
			Description: "The pod IP range assigned to the node.",
		},
		"pod_cidrs": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    false,
			Description: "The IP ranges assigned to the node for usage by pods on that node.",
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"provider_id": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "ID of the node assigned by the cloud provider.",
		},
		"unschedulable": {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Controls the schedulability of new pods on the node.  By default, node is schedulable.",
		},
		"taints": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Taints applied to the node",
			Elem: &schema.Resource{
				Schema: nodeTaintFields(),
			},
		},
	}
}

func nodeStatusFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"addresses": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Set of IP addresses and/or Hostname assigned to the node. More info: https://kubernetes.io/docs/concepts/architecture/nodes/#addresses/node/#info",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"type": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"address": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
		"allocatable": {
			Type:        schema.TypeMap,
			Description: "Represents the total resources of a node.",
			Computed:    true,
			Elem:        schema.TypeString,
		},
		"capacity": {
			Type:        schema.TypeMap,
			Description: "Represents the resources of a node that are available for scheduling.",
			Computed:    true,
			Elem:        schema.TypeString,
		},
		"node_info": {
			Type:        schema.TypeList,
			Computed:    true,
			Description: "Set of ids/uuids to uniquely identify the node. More info: https://kubernetes.io/docs/concepts/nodes/node/#info",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"machine_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"system_uuid": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"boot_id": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"kernel_version": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"os_image": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"container_runtime_version": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"kubelet_version": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"kube_proxy_version": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"operating_system": {
						Type:     schema.TypeString,
						Computed: true,
					},
					"architecture": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
	}
}

func nodeTaintFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"key": {
			Type:        schema.TypeString,
			Description: "The taint key",
			Required:    true,
		},
		"value": {
			Type:        schema.TypeString,
			Description: "The taint value",
			Required:    true,
		},
		"effect": {
			Type:        schema.TypeString,
			Description: "The taint effect",
			Required:    true,
		},
	}
}
