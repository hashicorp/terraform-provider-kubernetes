package kubernetes

import "github.com/hashicorp/terraform/helper/schema"

func persistentVolumeClaimSpecFields(pvcTemplate bool) map[string]*schema.Schema {
	s := map[string]*schema.Schema{
		"metadata": namespacedMetadataSchema("persistent volume claim", true),
		"spec": {
			Type:        schema.TypeList,
			Description: "Spec defines the desired characteristics of a volume requested by a pod author. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#persistentvolumeclaims",
			Required:    true,
			ForceNew:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"access_modes": {
						Type:        schema.TypeSet,
						Description: "A set of the desired access modes the volume should have. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#access-modes-1",
						Required:    true,
						ForceNew:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Set:         schema.HashString,
					},
					"resources": {
						Type:        schema.TypeList,
						Description: "A list of the minimum resources the volume should have. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#resources",
						Required:    true,
						ForceNew:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"limits": {
									Type:        schema.TypeMap,
									Description: "Map describing the maximum amount of compute resources allowed. More info: http://kubernetes.io/docs/user-guide/compute-resources/",
									Optional:    true,
									ForceNew:    true,
								},
								"requests": {
									Type:        schema.TypeMap,
									Description: "Map describing the minimum amount of compute resources required. If this is omitted for a container, it defaults to `limits` if that is explicitly specified, otherwise to an implementation-defined value. More info: http://kubernetes.io/docs/user-guide/compute-resources/",
									Optional:    true,
									ForceNew:    true,
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
							Schema: map[string]*schema.Schema{
								"match_expressions": {
									Type:        schema.TypeList,
									Description: "A list of label selector requirements. The requirements are ANDed.",
									Optional:    true,
									ForceNew:    true,
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"key": {
												Type:        schema.TypeString,
												Description: "The label key that the selector applies to.",
												Optional:    true,
												ForceNew:    true,
											},
											"operator": {
												Type:        schema.TypeString,
												Description: "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.",
												Optional:    true,
												ForceNew:    true,
											},
											"values": {
												Type:        schema.TypeSet,
												Description: "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
												Optional:    true,
												ForceNew:    true,
												Elem:        &schema.Schema{Type: schema.TypeString},
												Set:         schema.HashString,
											},
										},
									},
								},
								"match_labels": {
									Type:        schema.TypeMap,
									Description: "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
									Optional:    true,
									ForceNew:    true,
								},
							},
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
				},
			},
		},
		"wait_until_bound": {
			Type:        schema.TypeBool,
			Description: "Whether to wait for the claim to reach `Bound` state (to find volume in which to claim the space)",
			Optional:    true,
			Default:     !pvcTemplate,
		},
	}

	return s
}
