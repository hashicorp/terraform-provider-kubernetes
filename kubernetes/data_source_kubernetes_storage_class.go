package kubernetes

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceKubernetesStorageClass() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesStorageClassRead,
		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("storage class", false),
			"parameters": {
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
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

func dataSourceKubernetesStorageClassRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	name := d.Get("metadata.0.name").(string)
	d.SetId(name)
	return resourceKubernetesStorageClassRead(ctx, d, meta)
}
