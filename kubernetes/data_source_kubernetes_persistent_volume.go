package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceKubernetesPersistentVolume() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceKubernetesPersistentVolumeRead,

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("persistent volume", false),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired characteristics of a volume requested by a pod author. More info: http://kubernetes.io/docs/user-guide/persistent-volumes",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_modes": {
							Type:        schema.TypeSet,
							Description: "Contains all ways the volume can be mounted. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#access-modes",
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
						},
						"capacity": {
							Type:         schema.TypeMap,
							Description:  "A description of the persistent volume's resources and capacity. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#capacity",
							Required:     true,
							Elem:         schema.TypeString,
							ValidateFunc: validateResourceList,
						},
						"persistent_volume_reclaim_policy": {
							Type:        schema.TypeString,
							Description: "What happens to a persistent volume when released from its claim. Valid options are Retain (default) and Recycle. Recycling must be supported by the volume plugin underlying this persistent volume. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#recycling-policy",
							Optional:    true,
							Default:     "Retain",
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
					},
				},
			},
		},
	}
}

func dataSourceKubernetesPersistentVolumeRead(d *schema.ResourceData, meta interface{}) error {
	name := d.Get("metadata.0.name").(string)
	d.SetId(name)
	return resourceKubernetesPersistentVolumeRead(d, meta)
}
