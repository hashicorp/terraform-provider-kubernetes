// Copyright IBM Corp. 2017, 2026
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
				Schema: persistentVolumeClaimSpecFields(false),
			},
		},
	}
}

func persistentVolumeClaimSpecFields(isUpdatable bool) map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"access_modes": {
			Type:        schema.TypeSet,
			Description: "A set of the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes",
			Required:    true,
			ForceNew:    !isUpdatable,
			Elem: &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validation.StringInSlice([]string{
					"ReadWriteOnce",
					"ReadOnlyMany",
					"ReadWriteMany",
					"ReadWriteOncePod",
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
						ForceNew:         !isUpdatable,
						DiffSuppressFunc: suppressEquivalentResourceQuantity,
					},
					// The API permits in-place storage expansion for standalone PVCs, so requests
					// remains updateable in both schema contexts.
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
			ForceNew:    !isUpdatable,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: labelSelectorFields(isUpdatable),
			},
		},
		"volume_name": {
			Type:        schema.TypeString,
			Description: "The binding reference to the PersistentVolume backing this claim.",
			Optional:    true,
			ForceNew:    !isUpdatable,
			Computed:    true,
		},
		"storage_class_name": {
			Type:        schema.TypeString,
			Description: "Name of the storage class requested by the claim",
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
		},
		"volume_mode": {
			Type:        schema.TypeString,
			Description: "Defines what type of volume is required by the claim.",
			Optional:    true,
			Computed:    true,
			ForceNew:    !isUpdatable,
			ValidateFunc: validation.StringInSlice([]string{
				string(corev1.PersistentVolumeBlock),
				string(corev1.PersistentVolumeFilesystem),
			}, false),
		},
	}
}
