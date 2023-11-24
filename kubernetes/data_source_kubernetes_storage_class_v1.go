// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesStorageClassV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesStorageClassV1Read,
		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("storage class", false),
			"parameters": {
				Type:        schema.TypeMap,
				Description: "The parameters for the provisioner that should create volumes of this storage class",
				Optional:    true,
				Computed:    true,
			},
			"storage_provisioner": {
				Type:        schema.TypeString,
				Description: "Indicates the type of the provisioner",
				Computed:    true,
			},
			"reclaim_policy": {
				Type:        schema.TypeString,
				Description: "Indicates the type of the reclaim policy",
				Optional:    true,
				Computed:    true,
			},
			"volume_binding_mode": {
				Type:        schema.TypeString,
				Description: "Indicates when volume binding and dynamic provisioning should occur",
				Optional:    true,
				Computed:    true,
			},
			"allow_volume_expansion": {
				Type:        schema.TypeBool,
				Description: "Indicates whether the storage class allow volume expand",
				Optional:    true,
				Computed:    true,
			},
			"mount_options": {
				Type:        schema.TypeSet,
				Description: "Persistent Volumes that are dynamically created by a storage class will have the mount options specified",
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
			},
			"allowed_topologies": {
				Type:        schema.TypeList,
				Description: "Restrict the node topologies where volumes can be dynamically provisioned.",
				Optional:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"match_label_expressions": {
							Type:        schema.TypeList,
							Description: "A list of topology selector requirements by labels.",
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Description: "The label key that the selector applies to.",
										Optional:    true,
									},
									"values": {
										Type:        schema.TypeSet,
										Description: "An array of string values. One value must match the label to be selected.",
										Optional:    true,
										Elem:        &schema.Schema{Type: schema.TypeString},
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

func dataSourceKubernetesStorageClassV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return diag.FromErr(err)
	}

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	d.SetId(metadata.Name)

	log.Printf("[INFO] Reading storage class %s", metadata.Name)
	storageClass, err := conn.StorageV1().StorageClasses().Get(ctx, metadata.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return diag.FromErr(err)
	}
	log.Printf("[INFO] Received storage class: %#v", storageClass)

	diags := diag.Diagnostics{}

	err = d.Set("metadata", flattenMetadataFields(storageClass.ObjectMeta))
	if err != nil {
		diags = append(diags, diag.FromErr(err)[0])
	}

	sc := flattenStorageClass(*storageClass)
	for k, v := range sc {
		err = d.Set(k, v)
		if err != nil {
			diags = append(diags, diag.FromErr(err)[0])
		}
	}

	return diags
}
