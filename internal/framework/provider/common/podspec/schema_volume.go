// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func podVolumeObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{"name": podString(false, false, false, "")},
		Blocks: map[string]schema.Block{
			"aws_elastic_block_store": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":   podString(false, false, false, ""),
				"partition": podInt(false, false, 0),
				"read_only": podBool(false, false),
				"volume_id": podString(true, false, false, ""),
			}}, 0, 1, false),
			"azure_disk": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"caching_mode":  podString(true, false, false, ""),
				"data_disk_uri": podString(true, false, false, ""),
				"disk_name":     podString(true, false, false, ""),
				"fs_type":       podString(false, false, false, ""),
				"kind":          podString(false, true, false, ""),
				"read_only":     podBool(false, false),
			}}, 0, 1, false),
			"azure_file": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"read_only":        podBool(false, false),
				"secret_name":      podString(true, false, false, ""),
				"secret_namespace": podString(false, false, true, ""), // SDKv2 ForceNew even in templates.
				"share_name":       podString(true, false, false, ""),
			}}, 0, 1, false),
			"ceph_fs": podBlock(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"monitors":    podRequiredSet(false),
					"path":        podString(false, false, false, ""),
					"read_only":   podBool(false, false),
					"secret_file": podString(false, false, false, ""),
					"user":        podString(false, false, false, ""),
				},
				Blocks: map[string]schema.Block{"secret_ref": podBlock(podVolumeSecretReferenceObject(), 0, 1, false)},
			}, 0, 1, false),
			"cinder": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":   podString(false, false, false, ""),
				"read_only": podBool(false, false),
				"volume_id": podString(true, false, false, ""),
			}}, 0, 1, false),
			"config_map": podBlock(podKeyVolumeObject("name", true), 0, 1, false),
			"csi": podBlock(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"driver":            podString(true, false, false, ""),
					"fs_type":           podString(false, false, false, ""),
					"read_only":         podBool(false, false),
					"volume_attributes": podMap(false, false),
				},
				Blocks: map[string]schema.Block{"node_publish_secret_ref": podBlock(schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{"name": podString(false, false, false, "")},
				}, 0, 1, false)},
			}, 0, 1, false),
			"downward_api": podBlock(podDownwardAPIVolumeObject(false), 0, 1, false),
			"empty_dir": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"medium":     podString(false, false, false, "", stringvalidator.OneOf("", "Memory", "HugePages", "HugePages-2Mi", "HugePages-1Gi")),
				"size_limit": podQuantityString(""),
			}}, 0, 1, false),
			"ephemeral": podBlock(schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"volume_claim_template": podBlock(podVolumeClaimTemplateObject(), 1, 1, false),
			}}, 0, 1, false),
			"fc": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":      podString(false, false, false, ""),
				"lun":          podInt(true, false, 0),
				"read_only":    podBool(false, false),
				"target_ww_ns": podRequiredSet(false),
			}}, 0, 1, false),
			"flex_volume": podBlock(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"driver":    podString(true, false, false, ""),
					"fs_type":   podString(false, false, false, ""),
					"options":   podMap(false, false),
					"read_only": podBool(false, false),
				},
				Blocks: map[string]schema.Block{"secret_ref": podBlock(podVolumeSecretReferenceObject(), 0, 1, false)},
			}, 0, 1, false),
			"flocker": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"dataset_name": podString(false, false, false, ""),
				"dataset_uuid": podString(false, false, false, ""),
			}}, 0, 1, false),
			"gce_persistent_disk": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":   podString(false, false, false, ""),
				"partition": podInt(false, false, 0),
				"pd_name":   podString(true, false, false, ""),
				"read_only": podBool(false, false),
			}}, 0, 1, false),
			"git_repo": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"directory":  podString(false, false, false, "", podStringRule("path")),
				"repository": podString(false, false, false, ""),
				"revision":   podString(false, false, false, ""),
			}}, 0, 1, false),
			"glusterfs": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"endpoints_name": podString(true, false, false, ""),
				"path":           podString(true, false, false, ""),
				"read_only":      podBool(false, false),
			}}, 0, 1, false),
			"host_path": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"path": podString(false, false, false, ""),
				"type": podString(false, false, false, "", stringvalidator.OneOf("", "DirectoryOrCreate", "Directory", "FileOrCreate", "File", "Socket", "CharDevice", "BlockDevice")),
			}}, 0, 1, false),
			"iscsi": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":         podString(false, false, false, ""),
				"iqn":             podString(true, false, false, ""),
				"iscsi_interface": podString(false, false, false, "default"),
				"lun":             podInt(false, false, 0),
				"read_only":       podBool(false, false),
				"target_portal":   podString(true, false, false, ""),
			}}, 0, 1, false),
			"local": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"path": podString(false, false, false, ""),
			}}, 0, 1, false),
			"nfs": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"path":      podString(true, false, false, ""),
				"read_only": podBool(false, false),
				"server":    podString(true, false, false, ""),
			}}, 0, 1, false),
			"persistent_volume_claim": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"claim_name": podString(false, false, false, ""),
				"read_only":  podBool(false, false),
			}}, 0, 1, false),
			"photon_persistent_disk": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type": podString(false, false, false, ""),
				"pd_id":   podString(true, false, false, ""),
			}}, 0, 1, false),
			"projected": podBlock(podProjectedVolumeObject(), 0, 0, false),
			"quobyte": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"group":     podString(false, false, false, ""),
				"read_only": podBool(false, false),
				"registry":  podString(true, false, false, ""),
				"user":      podString(false, false, false, ""),
				"volume":    podString(true, false, false, ""),
			}}, 0, 1, false),
			"rbd": podBlock(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"ceph_monitors": podRequiredSet(false),
					"fs_type":       podString(false, false, false, ""),
					"keyring":       podString(false, true, false, ""),
					"rados_user":    podString(false, false, false, "admin"),
					"rbd_image":     podString(true, false, false, ""),
					"rbd_pool":      podString(false, false, false, "rbd"),
					"read_only":     podBool(false, false),
				},
				Blocks: map[string]schema.Block{"secret_ref": podBlock(podVolumeSecretReferenceObject(), 0, 1, false)},
			}, 0, 1, false),
			"secret": podBlock(podKeyVolumeObject("secret_name", true), 0, 1, false),
			"vsphere_volume": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":     podString(false, false, false, ""),
				"volume_path": podString(true, false, false, ""),
			}}, 0, 1, false),
		},
	}
}

func podRequiredSet(replace bool, validators ...validator.Set) schema.SetAttribute {
	a := podSet(replace, types.StringType)
	a.Required, a.Optional, a.Validators = true, false, validators
	return a
}

func podVolumeSecretReferenceObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"name":      podString(false, false, false, ""),
		"namespace": podString(false, true, false, ""),
	}}
}

func podKeyVolumeObject(name string, defaultMode bool) schema.NestedBlockObject {
	a := map[string]schema.Attribute{
		name:       podString(false, false, false, ""),
		"optional": podBool(false, false),
	}
	if defaultMode {
		a["default_mode"] = podString(false, false, false, "0644", podStringRule("mode"))
	}
	return schema.NestedBlockObject{
		Attributes: a,
		Blocks: map[string]schema.Block{"items": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"key":  podString(false, false, false, ""),
			"mode": podString(false, false, false, "", podStringRule("mode")),
			"path": podString(false, false, false, "", podStringRule("path")),
		}}, 0, 0, false)},
	}
}

func podDownwardAPIVolumeObject(projected bool) schema.NestedBlockObject {
	o := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"items": podBlock(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"mode": podString(false, false, false, "", podStringRule("mode")),
				"path": podString(true, false, false, "", podStringRule("path")),
			},
			Blocks: map[string]schema.Block{
				"field_ref":          podBlock(podFieldReferenceObject(false), podRequiredUnlessProjected(projected), 1, false),
				"resource_field_ref": podBlock(podResourceFieldReferenceObject(true, false), 0, 1, false),
			},
		}, 0, 0, false),
	}}
	if !projected {
		o.Attributes = map[string]schema.Attribute{"default_mode": podString(false, false, false, "0644", podStringRule("mode"))}
	}
	return o
}

func podRequiredUnlessProjected(projected bool) int {
	if projected {
		return 0
	}
	return 1
}

func podProjectedVolumeObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{"default_mode": podString(false, false, false, "0644", podStringRule("mode"))},
		Blocks: map[string]schema.Block{
			"sources": podBlock(schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"secret":       podBlock(podKeyVolumeObject("name", false), 0, 0, false),
				"config_map":   podBlock(podKeyVolumeObject("name", false), 0, 0, false),
				"downward_api": podBlock(podDownwardAPIVolumeObject(true), 0, 1, false),
				"service_account_token": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"audience":           podString(false, false, false, ""),
					"expiration_seconds": podInt(false, false, 3600, int64validator.AtLeast(600)),
					"path":               podString(true, false, false, ""),
				}}, 0, 1, false),
			}}, 1, 0, false),
		},
	}
}

func podVolumeClaimTemplateObject() schema.NestedBlockObject {
	annotations := podMap(false, false)
	annotations.Validators = []validator.Map{common.AnnotationsValidator()}
	labels := podMap(false, false)
	labels.Validators = []validator.Map{common.LabelsValidator()}
	selector := podLabelSelectorObject()
	selector.Attributes["match_labels"] = podMap(false, true)
	selector.Blocks["match_expressions"] = podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"key":      podString(false, false, true, ""),
		"operator": podString(false, false, true, ""),
		"values":   podSet(true, types.StringType),
	}}, 0, 0, true)
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"metadata": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"annotations": annotations, "labels": labels,
		}}, 0, 1, false),
		"spec": podBlock(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"access_modes":       podRequiredSet(true, setvalidator.ValueStringsAre(stringvalidator.OneOf("ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany", "ReadWriteOncePod"))),
				"volume_name":        podString(false, true, true, ""),
				"storage_class_name": podString(false, true, true, ""),
				"volume_mode":        podString(false, true, true, "", stringvalidator.OneOf("Block", "Filesystem")),
			},
			Blocks: map[string]schema.Block{
				"resources": podBlock(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"limits":   podQuantityMap(false, true),
					"requests": podQuantityMap(false, false),
				}}, 1, 1, false),
				"selector": podBlock(selector, 0, 1, true),
			},
		}, 1, 1, false),
	}}
}
