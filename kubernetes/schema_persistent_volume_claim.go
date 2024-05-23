// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	corev1 "k8s.io/api/core/v1"
)

func persistentVolumeClaimFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("persistent volume claim", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired characteristics of a volume requested by a pod author. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: persistentVolumeClaimSpecFields(),
			},
		},
	}
}

func persistentVolumeClaimSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"access_modes": {
			Type:        schema.TypeSet,
			Description: "A set of the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes",
			Required:    true,
			ForceNew:    true,
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
		"resources": {
			Type:        schema.TypeList,
			Description: "A list of the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources",
			Required:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"limits": {
						Type:             schema.TypeMap,
						Description:      "Map describing the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
						Optional:         true,
						ForceNew:         true,
						DiffSuppressFunc: suppressEquivalentResourceQuantity,
					},
					// This is the only field the API will allow modifying in-place, so ForceNew is not used.
					"requests": {
						Type:             schema.TypeMap,
						Description:      "Map describing the minimum amount of compute resources required. If this is omitted for a container, it defaults to `limits` if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
						Optional:         true,
						DiffSuppressFunc: suppressEquivalentResourceQuantity,
					},
				},
			},
		},
		"selector": {
			Type:        schema.TypeList,
			Description: "A label query over volumes to consider for binding.",
			Optional:    true,
			ForceNew:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: labelSelectorFields(false),
			},
		},
		"volume_name": {
			Type:        schema.TypeString,
			Description: "The binding reference to the PersistentVolume backing this claim.",
			Optional:    true,
			ForceNew:    true,
			Computed:    true,
		},
		"storage_class_name": {
			Type:        schema.TypeString,
			Description: "Name of the storage class requested by the claim",
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
		},
		"volume_mode": {
			Type:        schema.TypeString,
			Description: "Defines what type of volume is required by the claim.",
			Optional:    true,
			Computed:    true,
			ForceNew:    true,
			ValidateFunc: validation.StringInSlice([]string{
				string(corev1.PersistentVolumeBlock),
				string(corev1.PersistentVolumeFilesystem),
			}, false),
		},
	}
}
