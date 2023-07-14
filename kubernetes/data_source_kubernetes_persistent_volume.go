// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceKubernetesPersistentVolume() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesPersistentVolumeRead,

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("persistent volume", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec of the persistent volume owned by the cluster",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_modes": {
							Type:        schema.TypeSet,
							Description: "Contains all ways the volume can be mounted. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#access-modes",
							Required:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
								ValidateFunc: validation.StringInSlice([]string{
									"ReadWriteOnce",
									"ReadOnlyMany",
									"ReadWriteMany",
								}, false),
							},
							Set: schema.HashString,
						},
						"capacity": {
							Type:             schema.TypeMap,
							Description:      "A description of the persistent volume's resources and capacity. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#capacity",
							Required:         true,
							Elem:             schema.TypeString,
							ValidateFunc:     validateResourceList,
							DiffSuppressFunc: suppressEquivalentResourceQuantity,
						},
						"persistent_volume_reclaim_policy": {
							Type:        schema.TypeString,
							Description: "What happens to a persistent volume when released from its claim. Valid options are Retain (default) and Recycle. Recycling must be supported by the volume plugin underlying this persistent volume. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#recycling-policy",
							Optional:    true,
							Default:     "Retain",
							ValidateFunc: validation.StringInSlice([]string{
								"Recycle",
								"Delete",
								"Retain",
							}, false),
						},
						"claim_ref": {
							Type:        schema.TypeList,
							Description: "A reference to the persistent volume claim details for statically managed PVs. More Info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#binding",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"namespace": {
										Type:        schema.TypeString,
										Description: "The namespace of the PersistentVolumeClaim. Uses 'default' namespace if none is specified.",
										Elem:        schema.TypeString,
										Optional:    true,
										Default:     "default",
									},
									"name": {
										Type:        schema.TypeString,
										Description: "The name of the PersistentVolumeClaim",
										Elem:        schema.TypeString,
										Required:    true,
									},
								},
							},
						},
						"persistent_volume_source": {
							Type:        schema.TypeList,
							Description: "The specification of a persistent volume.",
							Required:    true,
							MaxItems:    1,
							Elem:        persistentVolumeSourceSchema(),
						},
						"storage_class_name": {
							Type:        schema.TypeString,
							Description: "A description of the persistent volume's class. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#class",
							Optional:    true,
						},
						"node_affinity": {
							Type:        schema.TypeList,
							Description: "A description of the persistent volume's node affinity. More info: https://kubernetes.io/docs/concepts/storage/volumes/#local",
							Optional:    true,
							Computed:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"required": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"node_selector_term": {
													Type:     schema.TypeList,
													Required: true,
													Elem: &schema.Resource{
														Schema: nodeSelectorTermFields(),
													},
												},
											},
										},
									},
								},
							},
						},
						"mount_options": {
							Type:        schema.TypeSet,
							Description: "A list of mount options, e.g. [\"ro\", \"soft\"]. Not validated - mount will simply fail if one is invalid.",
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"volume_mode": {
							Type:        schema.TypeString,
							Description: "Defines if a volume is intended to be used with a formatted filesystem. or to remain in raw block state.",
							Optional:    true,
							ForceNew:    true,
							Default:     "Filesystem",
							ValidateFunc: validation.StringInSlice([]string{
								"Block",
								"Filesystem",
							}, false),
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesPersistentVolumeRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("metadata.0.name").(string)
	d.SetId(name)
	return resourceKubernetesPersistentVolumeRead(ctx, d, meta)
}
