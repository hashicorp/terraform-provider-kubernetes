// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dataSourceKubernetesPersistentVolumeClaimV1() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceKubernetesPersistentVolumeClaimV1Read,

		Schema: map[string]*schema.Schema{
			"metadata": namespacedMetadataSchema("persistent volume claim", true),
			"spec": {
				Type:        schema.TypeList,
				Description: "Spec defines the desired characteristics of a volume requested by a pod author. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims",
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"access_modes": {
							Type:        schema.TypeSet,
							Description: "A set of the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes",
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Set: schema.HashString,
						},
						"resources": {
							Type:        schema.TypeList,
							Description: "A list of the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"limits": {
										Type:        schema.TypeMap,
										Description: "Map describing the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
										Optional:    true,
										Computed:    true,
									},
									"requests": {
										Type:        schema.TypeMap,
										Description: "Map describing the minimum amount of compute resources required. If this is omitted for a container, it defaults to `limits` if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
										Optional:    true,
										Computed:    true,
									},
								},
							},
						},
						"selector": {
							Type:        schema.TypeList,
							Description: "A label query over volumes to consider for binding.",
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: labelSelectorFields(false),
							},
						},
						"volume_name": {
							Type:        schema.TypeString,
							Description: "The binding reference to the PersistentVolume backing this claim.",
							Optional:    true,
							Computed:    true,
						},
						"storage_class_name": {
							Type:        schema.TypeString,
							Description: "Name of the storage class requested by the claim",
							Optional:    true,
							Computed:    true,
						},
						"volume_mode": {
							Type:        schema.TypeString,
							Description: "Defines what type of volume is required by the claim.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceKubernetesPersistentVolumeClaimV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	metadata := expandMetadata(d.Get("metadata").([]interface{}))

	om := meta_v1.ObjectMeta{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
	}
	d.SetId(buildId(om))

	return resourceKubernetesPersistentVolumeClaimV1Read(ctx, d, meta)
}
