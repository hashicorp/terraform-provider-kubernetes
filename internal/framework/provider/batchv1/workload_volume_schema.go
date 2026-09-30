// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package batchv1

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func workloadAwsElasticBlockStoreObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore",
				Optional:    true,
			},
			"partition": schema.Int64Attribute{
				Description: "The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as \"1\". Similarly, the volume partition for /dev/sda is \"0\" (or you can leave the property empty).",
				Optional:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to set the read-only property in VolumeMounts to \"true\". If omitted, the default is \"false\". More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore",
				Optional:    true,
			},
			"volume_id": schema.StringAttribute{
				Description: "Unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore",
				Required:    true,
			},
		},
	}
}

func workloadAzureDiskObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"caching_mode": schema.StringAttribute{
				Description: "Host Caching mode: None, Read Only, Read Write.",
				Required:    true,
			},
			"data_disk_uri": schema.StringAttribute{
				Description: "The URI the data disk in the blob storage",
				Required:    true,
			},
			"disk_name": schema.StringAttribute{
				Description: "The Name of the data disk in the blob storage",
				Required:    true,
			},
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
				Optional:    true,
			},
			"kind": schema.StringAttribute{
				Description: "The type for the data disk. Expected values: Shared, Dedicated, Managed. Defaults to Shared",
				Optional:    true,
				Computed:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func workloadAzureFileObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).",
				Optional:    true,
			},
			"secret_name": schema.StringAttribute{
				Description: "The name of secret that contains Azure Storage Account Name and Key",
				Required:    true,
			},
			"secret_namespace": schema.StringAttribute{
				Description:   "The namespace of the secret that contains Azure Storage Account Name and Key. For Kubernetes up to 1.18.x the default is the same as the Pod. For Kubernetes 1.19.x and later the default is \"default\" namespace.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"share_name": schema.StringAttribute{
				Description: "Share Name",
				Required:    true,
			},
		},
	}
}

func workloadSecretRefObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func workloadCephFsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"monitors": schema.SetAttribute{
				Description: "Monitors is a collection of Ceph monitors. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it",
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"path": schema.StringAttribute{
				Description: "Used as the mounted root, rather than the full Ceph tree, default is /",
				Optional:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the read-only setting in VolumeMounts. Defaults to `false` (read/write). More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it",
				Optional:    true,
			},
			"secret_file": schema.StringAttribute{
				Description: "The path to key ring for User, default is `/etc/ceph/user.secret`. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it",
				Optional:    true,
			},
			"user": schema.StringAttribute{
				Description: "User is the rados user name, default is admin. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"secret_ref": schema.ListNestedBlock{
				Description:  "Reference to the authentication secret for User, default is empty. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it",
				NestedObject: workloadSecretRefObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadCinderObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: https://examples.k8s.io/mysql-cinder-pd/README.md",
				Optional:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write). More info: https://examples.k8s.io/mysql-cinder-pd/README.md",
				Optional:    true,
			},
			"volume_id": schema.StringAttribute{
				Description: "Volume ID used to identify the volume in Cinder. More info: https://examples.k8s.io/mysql-cinder-pd/README.md",
				Required:    true,
			},
		},
	}
}

func workloadItemsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Description: "The key to project.",
				Optional:    true,
			},
			"mode": schema.StringAttribute{
				Description: "Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
				Optional:    true,
				Validators:  []validator.String{workloadModeBitsValidator()},
			},
			"path": schema.StringAttribute{
				Description: "The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.",
				Optional:    true,
				Validators:  []validator.String{workloadPathValidator()},
			},
		},
	}
}

func workloadConfigMapObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"default_mode": schema.StringAttribute{
				Description: "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
				Validators:  []validator.String{workloadModeBitsValidator()},
			},
			"name": schema.StringAttribute{
				Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:    true,
			},
			"optional": schema.BoolAttribute{
				Description: "Optional: Specify whether the ConfigMap or its keys must be defined.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"items": schema.ListNestedBlock{
				Description:  "If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.",
				NestedObject: workloadItemsObject(updatable),
			},
		},
	}
}

func workloadNodePublishSecretRefObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:    true,
			},
		},
	}
}

func workloadCsiObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"driver": schema.StringAttribute{
				Description: "the name of the volume driver to use. More info: https://kubernetes.io/docs/concepts/storage/volumes/#csi",
				Required:    true,
			},
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
				Optional:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to set the read-only property in VolumeMounts to \"true\". If omitted, the default is \"false\". More info: https://kubernetes.io/docs/concepts/storage/volumes#csi",
				Optional:    true,
			},
			"volume_attributes": schema.MapAttribute{
				Description: "Attributes of the volume to publish.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"node_publish_secret_ref": schema.ListNestedBlock{
				Description:  "A reference to the secret object containing sensitive information to pass to the CSI driver to complete the CSI NodePublishVolume and NodeUnpublishVolume calls.",
				NestedObject: workloadNodePublishSecretRefObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadFieldRefObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				Description: "Version of the schema the FieldPath is written in terms of, defaults to \"v1\".",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("v1"),
			},
			"field_path": schema.StringAttribute{
				Description: "Path of the field to select in the specified API version",
				Optional:    true,
			},
		},
	}
}

func workloadResourceFieldRefObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"container_name": schema.StringAttribute{
				Required: true,
			},
			"divisor": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("1"),
				CustomType:    workloadQuantityType{},
				Validators:    []validator.String{workloadQuantityValidator()},
				PlanModifiers: []planmodifier.String{workloadQuantityStringPlanModifier{}},
			},
			"resource": schema.StringAttribute{
				Description: "Resource to select",
				Required:    true,
			},
		},
	}
}

func workloadItemsObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				Description: "Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
				Optional:    true,
				Validators:  []validator.String{workloadModeBitsValidator()},
			},
			"path": schema.StringAttribute{
				Description: "Path is the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'",
				Required:    true,
				Validators:  []validator.String{workloadPathValidator()},
			},
		},
		Blocks: map[string]schema.Block{
			"field_ref": schema.ListNestedBlock{
				Description:  "Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.",
				NestedObject: workloadFieldRefObject2(updatable),
				Validators:   []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
			},
			"resource_field_ref": schema.ListNestedBlock{
				Description:  "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.",
				NestedObject: workloadResourceFieldRefObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadDownwardApiObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"default_mode": schema.StringAttribute{
				Description: "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
				Validators:  []validator.String{workloadModeBitsValidator()},
			},
		},
		Blocks: map[string]schema.Block{
			"items": schema.ListNestedBlock{
				Description:  "If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.",
				NestedObject: workloadItemsObject2(updatable),
			},
		},
	}
}

func workloadEmptyDirObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"medium": schema.StringAttribute{
				Description:   "What type of storage medium should back this directory. The default is \"\" which means to use the node's default medium. Must be one of [\"\" \"Memory\" \"HugePages\" \"HugePages-2Mi\" \"HugePages-1Gi\"]. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir",
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString(""),
				Validators:    []validator.String{stringvalidator.OneOf("", "Memory", "HugePages", "HugePages-2Mi", "HugePages-1Gi")},
				PlanModifiers: workloadStringPlanModifiers(updatable),
			},
			"size_limit": schema.StringAttribute{
				Description:   "Total amount of local storage required for this EmptyDir volume.",
				Optional:      true,
				CustomType:    workloadQuantityType{},
				Validators:    []validator.String{workloadQuantityValidator()},
				PlanModifiers: []planmodifier.String{workloadQuantityStringPlanModifier{requiresReplace: !updatable}},
			},
		},
	}
}

func workloadMetadataObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"annotations": schema.MapAttribute{
				Description: "An unstructured key value map stored with the persistent volume claim that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.Map{common.AnnotationsValidator()},
			},
			"labels": schema.MapAttribute{
				Description: "Map of string keys and values that can be used to organize and categorize (scope and select) the persistent volume claim. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.Map{common.LabelsValidator()},
			},
		},
	}
}

func workloadResourcesObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"limits": schema.MapAttribute{
				Description:   "Map describing the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
				Optional:      true,
				ElementType:   workloadQuantityType{},
				PlanModifiers: []planmodifier.Map{workloadQuantityMapPlanModifier{requiresReplace: true}},
			},
			"requests": schema.MapAttribute{
				Description:   "Map describing the minimum amount of compute resources required. If this is omitted for a container, it defaults to `limits` if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
				Optional:      true,
				ElementType:   workloadQuantityType{},
				PlanModifiers: []planmodifier.Map{workloadQuantityMapPlanModifier{}},
			},
		},
	}
}

func workloadMatchExpressionsObject3(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"key": schema.StringAttribute{
				Description:   "The label key that the selector applies to.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"operator": schema.StringAttribute{
				Description:   "A key's relationship to a set of values. Valid operators ard `In`, `NotIn`, `Exists` and `DoesNotExist`.",
				Optional:      true,
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"values": schema.SetAttribute{
				Description:   "An array of string values. If the operator is `In` or `NotIn`, the values array must be non-empty. If the operator is `Exists` or `DoesNotExist`, the values array must be empty. This array is replaced during a strategic merge patch.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{workloadSetRequiresReplace()},
			},
		},
	}
}

func workloadSelectorObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"match_labels": schema.MapAttribute{
				Description:   "A map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of `match_expressions`, whose key field is \"key\", the operator is \"In\", and the values array contains only \"value\". The requirements are ANDed.",
				Optional:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{workloadMapRequiresReplace()},
			},
		},
		Blocks: map[string]schema.Block{
			"match_expressions": schema.ListNestedBlock{
				Description:   "A list of label selector requirements. The requirements are ANDed.",
				NestedObject:  workloadMatchExpressionsObject3(updatable),
				PlanModifiers: []planmodifier.List{workloadObjectListRequiresReplace()},
			},
		},
	}
}

func workloadSpecObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"access_modes": schema.SetAttribute{
				Description:   "A set of the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes",
				Required:      true,
				ElementType:   types.StringType,
				Validators:    []validator.Set{setvalidator.ValueStringsAre(stringvalidator.OneOf("ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany", "ReadWriteOncePod")), setvalidator.SizeAtLeast(1)},
				PlanModifiers: []planmodifier.Set{workloadSetRequiresReplace()},
			},
			"storage_class_name": schema.StringAttribute{
				Description:   "Name of the storage class requested by the claim",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"volume_mode": schema.StringAttribute{
				Description:   "Defines what type of volume is required by the claim.",
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.OneOf("Block", "Filesystem")},
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
			"volume_name": schema.StringAttribute{
				Description:   "The binding reference to the PersistentVolume backing this claim.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{workloadStringRequiresReplace()},
			},
		},
		Blocks: map[string]schema.Block{
			"resources": schema.ListNestedBlock{
				Description:  "A list of the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources",
				NestedObject: workloadResourcesObject2(updatable),
				Validators:   []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
			},
			"selector": schema.ListNestedBlock{
				Description:   "A label query over volumes to consider for binding.",
				NestedObject:  workloadSelectorObject(updatable),
				Validators:    []validator.List{listvalidator.SizeAtMost(1)},
				PlanModifiers: []planmodifier.List{workloadObjectListRequiresReplace()},
			},
		},
	}
}

func workloadVolumeClaimTemplateObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"metadata": schema.ListNestedBlock{
				Description:  "May contain labels and annotations that will be copied into the PVC when creating it. No other fields are allowed and will be rejected during marker",
				NestedObject: workloadMetadataObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"spec": schema.ListNestedBlock{
				Description:  "The specification for the PersistentVolumeClaim. The entire content is copied unchanged into the PVC that gets created from this template. The same fields as in a PersistentVolumeClaim are also valid here.",
				NestedObject: workloadSpecObject(updatable),
				Validators:   []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
			},
		},
	}
}

func workloadEphemeralObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"volume_claim_template": schema.ListNestedBlock{
				Description:  "Will be used to create a stand-alone PVC to provision the volume. The pod in which this EphemeralVolumeSource is embedded will be the owner of the PVC.",
				NestedObject: workloadVolumeClaimTemplateObject(updatable),
				Validators:   []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
			},
		},
	}
}

func workloadFcObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
				Optional:    true,
			},
			"lun": schema.Int64Attribute{
				Description: "FC target lun number",
				Required:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).",
				Optional:    true,
			},
			"target_ww_ns": schema.SetAttribute{
				Description: "FC target worldwide names (WWNs)",
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
			},
		},
	}
}

func workloadFlexVolumeObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"driver": schema.StringAttribute{
				Description: "Driver is the name of the driver to use for this volume.",
				Required:    true,
			},
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". The default filesystem depends on FlexVolume script.",
				Optional:    true,
			},
			"options": schema.MapAttribute{
				Description: "Extra command options if any.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the ReadOnly setting in VolumeMounts. Defaults to false (read/write).",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"secret_ref": schema.ListNestedBlock{
				Description:  "Reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts.",
				NestedObject: workloadSecretRefObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadFlockerObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"dataset_name": schema.StringAttribute{
				Description: "Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated",
				Optional:    true,
			},
			"dataset_uuid": schema.StringAttribute{
				Description: "UUID of the dataset. This is unique identifier of a Flocker dataset",
				Optional:    true,
			},
		},
	}
}

func workloadGcePersistentDiskObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk",
				Optional:    true,
			},
			"partition": schema.Int64Attribute{
				Description: "The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as \"1\". Similarly, the volume partition for /dev/sda is \"0\" (or you can leave the property empty). More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk",
				Optional:    true,
			},
			"pd_name": schema.StringAttribute{
				Description: "Unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk",
				Required:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the ReadOnly setting in VolumeMounts. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk",
				Optional:    true,
			},
		},
	}
}

func workloadGitRepoObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"directory": schema.StringAttribute{
				Description: "Target directory name. Must not contain or start with '..'. If '.' is supplied, the volume directory will be the git repository. Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name.",
				Optional:    true,
				Validators:  []validator.String{workloadPathValidator()},
			},
			"repository": schema.StringAttribute{
				Description: "Repository URL",
				Optional:    true,
			},
			"revision": schema.StringAttribute{
				Description: "Commit hash for the specified revision.",
				Optional:    true,
			},
		},
	}
}

func workloadGlusterfsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"endpoints_name": schema.StringAttribute{
				Description: "The endpoint name that details Glusterfs topology. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod",
				Required:    true,
			},
			"path": schema.StringAttribute{
				Description: "The Glusterfs volume path. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod",
				Required:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod",
				Optional:    true,
			},
		},
	}
}

func workloadHostPathObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "Path of the directory on the host. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "Type for HostPath volume. Allowed values are \"\" (default), DirectoryOrCreate, Directory, FileOrCreate, File, Socket, CharDevice and BlockDevice",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.OneOf("", "DirectoryOrCreate", "Directory", "FileOrCreate", "File", "Socket", "CharDevice", "BlockDevice")},
			},
		},
	}
}

func workloadIscsiObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi",
				Optional:    true,
			},
			"iqn": schema.StringAttribute{
				Description: "Target iSCSI Qualified Name.",
				Required:    true,
			},
			"iscsi_interface": schema.StringAttribute{
				Description: "iSCSI interface name that uses an iSCSI transport. Defaults to 'default' (tcp).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("default"),
			},
			"lun": schema.Int64Attribute{
				Description: "iSCSI target lun number.",
				Optional:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false.",
				Optional:    true,
			},
			"target_portal": schema.StringAttribute{
				Description: "iSCSI target portal. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).",
				Required:    true,
			},
		},
	}
}

func workloadLocalObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "Path of the directory on the host. More info: https://kubernetes.io/docs/concepts/storage/volumes#local",
				Optional:    true,
			},
		},
	}
}

func workloadNfsObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				Description: "Path that is exported by the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs",
				Required:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the NFS export to be mounted with read-only permissions. Defaults to false. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs",
				Optional:    true,
			},
			"server": schema.StringAttribute{
				Description: "Server is the hostname or IP address of the NFS server. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs",
				Required:    true,
			},
		},
	}
}

func workloadPersistentVolumeClaimObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"claim_name": schema.StringAttribute{
				Description: "ClaimName is the name of a PersistentVolumeClaim in the same ",
				Optional:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Will force the ReadOnly setting in VolumeMounts.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func workloadPhotonPersistentDiskObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
				Optional:    true,
			},
			"pd_id": schema.StringAttribute{
				Description: "ID that identifies Photon Controller persistent disk",
				Required:    true,
			},
		},
	}
}

func workloadConfigMapObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:    true,
			},
			"optional": schema.BoolAttribute{
				Description: "Optional: Specify whether the ConfigMap or it's keys must be defined.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"items": schema.ListNestedBlock{
				Description:  "If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.",
				NestedObject: workloadItemsObject(updatable),
			},
		},
	}
}

func workloadFieldRefObject3(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				Description: "Version of the schema the FieldPath is written in terms of, defaults to 'v1'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("v1"),
			},
			"field_path": schema.StringAttribute{
				Description: "Path of the field to select in the specified API version",
				Optional:    true,
			},
		},
	}
}

func workloadItemsObject3(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				Description: "Mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
				Optional:    true,
				Validators:  []validator.String{workloadModeBitsValidator()},
			},
			"path": schema.StringAttribute{
				Description: "Path is the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'",
				Required:    true,
				Validators:  []validator.String{workloadPathValidator()},
			},
		},
		Blocks: map[string]schema.Block{
			"field_ref": schema.ListNestedBlock{
				Description:  "Selects a field of the pod: only annotations, labels, name and namespace are supported.",
				NestedObject: workloadFieldRefObject3(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"resource_field_ref": schema.ListNestedBlock{
				Description:  "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.",
				NestedObject: workloadResourceFieldRefObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadDownwardApiObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"items": schema.ListNestedBlock{
				Description:  "Represents a volume containing downward API info. Downward API volumes support ownership management and SELinux relabeling.",
				NestedObject: workloadItemsObject3(updatable),
			},
		},
	}
}

func workloadSecretObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secrets",
				Optional:    true,
			},
			"optional": schema.BoolAttribute{
				Description: "Optional: Specify whether the Secret or it's keys must be defined.",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"items": schema.ListNestedBlock{
				Description:  "If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.",
				NestedObject: workloadItemsObject(updatable),
			},
		},
	}
}

func workloadServiceAccountTokenObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"audience": schema.StringAttribute{
				Description: "Audience is the intended audience of the token",
				Optional:    true,
			},
			"expiration_seconds": schema.Int64Attribute{
				Description: "ExpirationSeconds is the expected duration of validity of the service account token. It defaults to 1 hour and must be at least 10 minutes (600 seconds).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3600),
				Validators:  []validator.Int64{int64validator.AtLeast(600)},
			},
			"path": schema.StringAttribute{
				Description: "Path specifies a relative path to the mount point of the projected volume.",
				Required:    true,
			},
		},
	}
}

func workloadSourcesObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"config_map": schema.ListNestedBlock{
				Description:  "ConfigMap represents a configMap that should populate this volume",
				NestedObject: workloadConfigMapObject2(updatable),
			},
			"downward_api": schema.ListNestedBlock{
				Description:  "DownwardAPI represents downward API about the pod that should populate this volume",
				NestedObject: workloadDownwardApiObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"secret": schema.ListNestedBlock{
				Description:  "Secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secrets",
				NestedObject: workloadSecretObject(updatable),
			},
			"service_account_token": schema.ListNestedBlock{
				Description:  "A projected service account token volume",
				NestedObject: workloadServiceAccountTokenObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadProjectedObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"default_mode": schema.StringAttribute{
				Description: "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
				Validators:  []validator.String{workloadModeBitsValidator()},
			},
		},
		Blocks: map[string]schema.Block{
			"sources": schema.ListNestedBlock{
				Description:  "Source of the volume to project in the directory.",
				NestedObject: workloadSourcesObject(updatable),
				Validators:   []validator.List{listvalidator.IsRequired(), listvalidator.SizeAtLeast(1)},
			},
		},
	}
}

func workloadQuobyteObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"group": schema.StringAttribute{
				Description: "Group to map volume access to Default is no group",
				Optional:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the Quobyte volume to be mounted with read-only permissions. Defaults to false.",
				Optional:    true,
			},
			"registry": schema.StringAttribute{
				Description: "Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes",
				Required:    true,
			},
			"user": schema.StringAttribute{
				Description: "User to map volume access to Defaults to serivceaccount user",
				Optional:    true,
			},
			"volume": schema.StringAttribute{
				Description: "Volume is a string that references an already created Quobyte volume by name.",
				Required:    true,
			},
		},
	}
}

func workloadRbdObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"ceph_monitors": schema.SetAttribute{
				Description: "A collection of Ceph monitors. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it",
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1)},
			},
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd",
				Optional:    true,
			},
			"keyring": schema.StringAttribute{
				Description: "Keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it",
				Optional:    true,
				Computed:    true,
			},
			"rados_user": schema.StringAttribute{
				Description: "The rados user name. Default is admin. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("admin"),
			},
			"rbd_image": schema.StringAttribute{
				Description: "The rados image name. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it",
				Required:    true,
			},
			"rbd_pool": schema.StringAttribute{
				Description: "The rados pool name. Default is rbd. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("rbd"),
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"secret_ref": schema.ListNestedBlock{
				Description:  "Name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it",
				NestedObject: workloadSecretRefObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}

func workloadSecretObject2(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"default_mode": schema.StringAttribute{
				Description: "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
				Validators:  []validator.String{workloadModeBitsValidator()},
			},
			"optional": schema.BoolAttribute{
				Description: "Optional: Specify whether the Secret or its keys must be defined.",
				Optional:    true,
			},
			"secret_name": schema.StringAttribute{
				Description: "Name of the secret in the pod's namespace to use. More info: https://kubernetes.io/docs/concepts/storage/volumes#secrets",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"items": schema.ListNestedBlock{
				Description:  "If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.",
				NestedObject: workloadItemsObject(updatable),
			},
		},
	}
}

func workloadVsphereVolumeObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"fs_type": schema.StringAttribute{
				Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
				Optional:    true,
			},
			"volume_path": schema.StringAttribute{
				Description: "Path that identifies vSphere volume vmdk",
				Required:    true,
			},
		},
	}
}

func workloadVolumeObject(updatable bool) schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Volume's name. Must be a DNS_LABEL and unique within the pod. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				Optional:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"aws_elastic_block_store": schema.ListNestedBlock{
				Description:  "Represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore",
				NestedObject: workloadAwsElasticBlockStoreObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"azure_disk": schema.ListNestedBlock{
				Description:  "Represents an Azure Data Disk mount on the host and bind mount to the pod.",
				NestedObject: workloadAzureDiskObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"azure_file": schema.ListNestedBlock{
				Description:  "Represents an Azure File Service mount on the host and bind mount to the pod.",
				NestedObject: workloadAzureFileObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"ceph_fs": schema.ListNestedBlock{
				Description:  "Represents a Ceph FS mount on the host that shares a pod's lifetime",
				NestedObject: workloadCephFsObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"cinder": schema.ListNestedBlock{
				Description:  "Represents a cinder volume attached and mounted on kubelets host machine. More info: https://examples.k8s.io/mysql-cinder-pd/README.md",
				NestedObject: workloadCinderObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"config_map": schema.ListNestedBlock{
				Description:  "ConfigMap represents a configMap that should populate this volume",
				NestedObject: workloadConfigMapObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"csi": schema.ListNestedBlock{
				Description:  "Represents a CSI Volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#csi",
				NestedObject: workloadCsiObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"downward_api": schema.ListNestedBlock{
				Description:  "DownwardAPI represents downward API about the pod that should populate this volume",
				NestedObject: workloadDownwardApiObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"empty_dir": schema.ListNestedBlock{
				Description:  "EmptyDir represents a temporary directory that shares a pod's lifetime. More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir",
				NestedObject: workloadEmptyDirObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"ephemeral": schema.ListNestedBlock{
				Description:  "Represents an ephemeral volume that is handled by a normal storage driver. More info: https://kubernetes.io/docs/concepts/storage/ephemeral-volumes/#generic-ephemeral-volumes",
				NestedObject: workloadEphemeralObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"fc": schema.ListNestedBlock{
				Description:  "Represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.",
				NestedObject: workloadFcObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"flex_volume": schema.ListNestedBlock{
				Description:  "Represents a generic volume resource that is provisioned/attached using an exec based plugin. This is an alpha feature and may change in future.",
				NestedObject: workloadFlexVolumeObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"flocker": schema.ListNestedBlock{
				Description:  "Represents a Flocker volume attached to a kubelet's host machine and exposed to the pod for its usage. This depends on the Flocker control service being running",
				NestedObject: workloadFlockerObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"gce_persistent_disk": schema.ListNestedBlock{
				Description:  "Represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin. More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk",
				NestedObject: workloadGcePersistentDiskObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"git_repo": schema.ListNestedBlock{
				Description:  "GitRepo represents a git repository at a particular revision.",
				NestedObject: workloadGitRepoObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"glusterfs": schema.ListNestedBlock{
				Description:  "Represents a Glusterfs volume that is attached to a host and exposed to the pod. Provisioned by an admin. More info: https://examples.k8s.io/volumes/glusterfs/README.md",
				NestedObject: workloadGlusterfsObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"host_path": schema.ListNestedBlock{
				Description:  "Represents a directory on the host. Provisioned by a developer or tester. This is useful for single-node development and testing only! On-host storage is not supported in any way and WILL NOT WORK in a multi-node cluster. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath",
				NestedObject: workloadHostPathObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"iscsi": schema.ListNestedBlock{
				Description:  "Represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin.",
				NestedObject: workloadIscsiObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"local": schema.ListNestedBlock{
				Description:  "Represents a mounted local storage device such as a disk, partition or directory. Local volumes can only be used as a statically created PersistentVolume. Dynamic provisioning is not supported yet. More info: https://kubernetes.io/docs/concepts/storage/volumes#local",
				NestedObject: workloadLocalObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"nfs": schema.ListNestedBlock{
				Description:  "Represents an NFS mount on the host. Provisioned by an admin. More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs",
				NestedObject: workloadNfsObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"persistent_volume_claim": schema.ListNestedBlock{
				Description:  "The specification of a persistent volume.",
				NestedObject: workloadPersistentVolumeClaimObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"photon_persistent_disk": schema.ListNestedBlock{
				Description:  "Represents a PhotonController persistent disk attached and mounted on kubelets host machine",
				NestedObject: workloadPhotonPersistentDiskObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"projected": schema.ListNestedBlock{
				Description:  "Projected represents a single volume that projects several volume sources into the same directory. More info: https://kubernetes.io/docs/concepts/storage/volumes/#projected",
				NestedObject: workloadProjectedObject(updatable),
			},
			"quobyte": schema.ListNestedBlock{
				Description:  "Quobyte represents a Quobyte mount on the host that shares a pod's lifetime",
				NestedObject: workloadQuobyteObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"rbd": schema.ListNestedBlock{
				Description:  "Represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: https://examples.k8s.io/volumes/rbd/README.md",
				NestedObject: workloadRbdObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"secret": schema.ListNestedBlock{
				Description:  "Secret represents a secret that should populate this volume. More info: https://kubernetes.io/docs/concepts/storage/volumes#secrets",
				NestedObject: workloadSecretObject2(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
			"vsphere_volume": schema.ListNestedBlock{
				Description:  "Represents a vSphere volume attached and mounted on kubelets host machine",
				NestedObject: workloadVsphereVolumeObject(updatable),
				Validators:   []validator.List{listvalidator.SizeAtMost(1)},
			},
		},
	}
}
