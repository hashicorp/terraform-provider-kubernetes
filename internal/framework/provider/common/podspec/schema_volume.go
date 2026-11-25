// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podspec

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (b builder) podVolumeObject() schema.NestedBlockObject {
	image := schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Mount an OCI image or artifact as a read-only volume. Requires Kubernetes 1.31 or later, a compatible container runtime and the ImageVolume feature gate, which is enabled by default from Kubernetes 1.35.",
		Attributes: map[string]schema.Attribute{
			"reference":   b.str(!b.o.Template, b.o.Template, updatable, "", stringvalidator.LengthAtLeast(1)),
			"pull_policy": b.str(false, true, updatable, "", stringvalidator.OneOf("Always", "Never", "IfNotPresent")),
		},
	}
	if b.replace(immutable) {
		image.PlanModifiers = []planmodifier.Object{podOptionalObjectRequiresReplace{}}
	}
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{"name": b.str(false, false, updatable, ""), "image": image},
		Blocks: map[string]schema.Block{
			"aws_elastic_block_store": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":   b.str(false, false, updatable, ""),
				"partition": b.integer(false, false, updatable, 0),
				"read_only": b.boolean(false, updatable, false),
				"volume_id": b.str(true, false, updatable, ""),
			}}, 0, 1, updatable),
			"azure_disk": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"caching_mode":  b.str(true, false, updatable, ""),
				"data_disk_uri": b.str(true, false, updatable, ""),
				"disk_name":     b.str(true, false, updatable, ""),
				"fs_type":       b.str(false, false, updatable, ""),
				"kind":          b.str(false, true, updatable, ""),
				"read_only":     b.boolean(false, updatable, false),
			}}, 0, 1, updatable),
			"azure_file": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"read_only":        b.boolean(false, updatable, false),
				"secret_name":      b.str(true, false, updatable, ""),
				"secret_namespace": b.str(false, false, alwaysNew, ""),
				"share_name":       b.str(true, false, updatable, ""),
			}}, 0, 1, updatable),
			"ceph_fs": b.block(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"monitors":    b.requiredSet(updatable),
					"path":        b.str(false, false, updatable, ""),
					"read_only":   b.boolean(false, updatable, false),
					"secret_file": b.str(false, false, updatable, ""),
					"user":        b.str(false, false, updatable, ""),
				},
				Blocks: map[string]schema.Block{"secret_ref": b.block(b.podVolumeSecretReferenceObject(), 0, 1, updatable)},
			}, 0, 1, updatable),
			"cinder": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":   b.str(false, false, updatable, ""),
				"read_only": b.boolean(false, updatable, false),
				"volume_id": b.str(true, false, updatable, ""),
			}}, 0, 1, updatable),
			"config_map": b.block(b.podKeyVolumeObject("name", true), 0, 1, updatable),
			"csi": b.block(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"driver":            b.str(true, false, updatable, ""),
					"fs_type":           b.str(false, false, updatable, ""),
					"read_only":         b.boolean(false, updatable, false),
					"volume_attributes": b.mapping(false, updatable),
				},
				Blocks: map[string]schema.Block{"node_publish_secret_ref": b.block(schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{"name": b.str(false, false, updatable, "")},
				}, 0, 1, updatable)},
			}, 0, 1, updatable),
			"downward_api": b.block(b.podDownwardAPIVolumeObject(false), 0, 1, updatable),
			"empty_dir": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"medium":     b.str(false, false, immutable, "", podStringRule("empty-dir-medium")),
				"size_limit": b.quantityString(immutable, ""),
			}}, 0, 1, updatable),
			"ephemeral": b.block(schema.NestedBlockObject{Blocks: map[string]schema.Block{
				"volume_claim_template": b.block(b.podVolumeClaimTemplateObject(), 1, 1, updatable),
			}}, 0, 1, updatable),
			"fc": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":      b.str(false, false, updatable, ""),
				"lun":          b.integer(true, false, updatable, 0),
				"read_only":    b.boolean(false, updatable, false),
				"target_ww_ns": b.requiredSet(updatable),
			}}, 0, 1, updatable),
			"flex_volume": b.block(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"driver":    b.str(true, false, updatable, ""),
					"fs_type":   b.str(false, false, updatable, ""),
					"options":   b.mapping(false, updatable),
					"read_only": b.boolean(false, updatable, false),
				},
				Blocks: map[string]schema.Block{"secret_ref": b.block(b.podVolumeSecretReferenceObject(), 0, 1, updatable)},
			}, 0, 1, updatable),
			"flocker": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"dataset_name": b.str(false, false, updatable, ""),
				"dataset_uuid": b.str(false, false, updatable, ""),
			}}, 0, 1, updatable),
			"gce_persistent_disk": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":   b.str(false, false, updatable, ""),
				"partition": b.integer(false, false, updatable, 0),
				"pd_name":   b.str(true, false, updatable, ""),
				"read_only": b.boolean(false, updatable, false),
			}}, 0, 1, updatable),
			"git_repo": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"directory":  b.str(false, false, updatable, "", podStringRule("path")),
				"repository": b.str(false, false, updatable, ""),
				"revision":   b.str(false, false, updatable, ""),
			}}, 0, 1, updatable),
			"glusterfs": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"endpoints_name": b.str(true, false, updatable, ""),
				"path":           b.str(true, false, updatable, ""),
				"read_only":      b.boolean(false, updatable, false),
			}}, 0, 1, updatable),
			"host_path": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"path": b.str(false, false, updatable, ""),
				"type": b.str(false, false, updatable, "", stringvalidator.OneOf("", "DirectoryOrCreate", "Directory", "FileOrCreate", "File", "Socket", "CharDevice", "BlockDevice")),
			}}, 0, 1, updatable),
			"iscsi": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":         b.str(false, false, updatable, ""),
				"iqn":             b.str(true, false, updatable, ""),
				"iscsi_interface": b.str(false, false, updatable, "default"),
				"lun":             b.integer(false, false, updatable, 0),
				"read_only":       b.boolean(false, updatable, false),
				"target_portal":   b.str(true, false, updatable, ""),
			}}, 0, 1, updatable),
			"local": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"path": b.str(false, false, updatable, ""),
			}}, 0, 1, updatable),
			"nfs": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"path":      b.str(true, false, updatable, ""),
				"read_only": b.boolean(false, updatable, false),
				"server":    b.str(true, false, updatable, ""),
			}}, 0, 1, updatable),
			"persistent_volume_claim": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"claim_name": b.str(false, false, updatable, ""),
				"read_only":  b.boolean(false, updatable, false),
			}}, 0, 1, updatable),
			"photon_persistent_disk": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type": b.str(false, false, updatable, ""),
				"pd_id":   b.str(true, false, updatable, ""),
			}}, 0, 1, updatable),
			"projected": b.block(b.podProjectedVolumeObject(), 0, 0, updatable),
			"quobyte": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"group":     b.str(false, false, updatable, ""),
				"read_only": b.boolean(false, updatable, false),
				"registry":  b.str(true, false, updatable, ""),
				"user":      b.str(false, false, updatable, ""),
				"volume":    b.str(true, false, updatable, ""),
			}}, 0, 1, updatable),
			"rbd": b.block(schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"ceph_monitors": b.requiredSet(updatable),
					"fs_type":       b.str(false, false, updatable, ""),
					"keyring":       b.str(false, true, updatable, ""),
					"rados_user":    b.str(false, false, updatable, "admin"),
					"rbd_image":     b.str(true, false, updatable, ""),
					"rbd_pool":      b.str(false, false, updatable, "rbd"),
					"read_only":     b.boolean(false, updatable, false),
				},
				Blocks: map[string]schema.Block{"secret_ref": b.block(b.podVolumeSecretReferenceObject(), 0, 1, updatable)},
			}, 0, 1, updatable),
			"secret": b.block(b.podKeyVolumeObject("secret_name", true), 0, 1, updatable),
			"vsphere_volume": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
				"fs_type":     b.str(false, false, updatable, ""),
				"volume_path": b.str(true, false, updatable, ""),
			}}, 0, 1, updatable),
		},
	}
}

func (b builder) requiredSet(f forceNew, validators ...validator.Set) schema.SetAttribute {
	a := b.set(f, types.StringType)
	a.Required, a.Optional, a.Validators = true, false, validators
	return a
}

func (b builder) podVolumeSecretReferenceObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"name":      b.str(false, false, updatable, ""),
		"namespace": b.str(false, true, updatable, ""),
	}}
}

func (b builder) podKeyVolumeObject(name string, defaultMode bool) schema.NestedBlockObject {
	a := map[string]schema.Attribute{
		name:       b.str(false, false, updatable, ""),
		"optional": b.boolean(false, updatable, false),
	}
	if defaultMode {
		a["default_mode"] = b.str(false, false, updatable, "0644", podStringRule("mode"))
	}
	return schema.NestedBlockObject{
		Attributes: a,
		Blocks: map[string]schema.Block{"items": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"key":  b.str(false, false, updatable, ""),
			"mode": b.str(false, false, updatable, "", podStringRule("mode")),
			"path": b.str(false, false, updatable, "", podStringRule("path")),
		}}, 0, 0, updatable)},
	}
}

func (b builder) podDownwardAPIVolumeObject(projected bool) schema.NestedBlockObject {
	o := schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"items": b.block(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"mode": b.str(false, false, updatable, "", podStringRule("mode")),
				"path": b.str(true, false, updatable, "", podStringRule("path")),
			},
			Blocks: map[string]schema.Block{
				"field_ref":          b.block(b.podFieldReferenceObject(updatable), 0, 1, updatable),
				"resource_field_ref": b.block(b.podResourceFieldReferenceObject(true, updatable), 0, 1, updatable),
			},
		}, 0, 0, updatable),
	}}
	if !projected {
		o.Attributes = map[string]schema.Attribute{"default_mode": b.str(false, false, updatable, "0644", podStringRule("mode"))}
	}
	return o
}

// podProjectionAttributes are the projected volume sources added as object
// arguments; each is one API source, after those of the blocks beside it.
func (b builder) podProjectionAttributes() map[string]schema.Attribute {
	trustBundle := schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Projects the trust anchors of ClusterTrustBundle objects into a file that the kubelet keeps up to date. Selects one bundle by name, or every bundle of a signer that matches label_selector. Requires Kubernetes 1.37 or later, or the ClusterTrustBundleProjection feature gate on Kubernetes 1.29 to 1.36.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional: true, Description: "Name of a single ClusterTrustBundle. Set either this or signer_name.",
				Validators: []validator.String{stringvalidator.ExactlyOneOf(path.MatchRelative().AtParent().AtName("signer_name"))},
			},
			"signer_name": schema.StringAttribute{Optional: true, Description: "Signer whose ClusterTrustBundles are selected with label_selector and combined."},
			"label_selector": schema.SingleNestedAttribute{
				Optional: true, Description: "Selects the ClusterTrustBundles of signer_name. Omission matches none and {} matches all. Cannot be set with name.",
				Validators: []validator.Object{objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("name"))},
				Attributes: map[string]schema.Attribute{
					"match_labels": schema.MapAttribute{Optional: true, ElementType: types.StringType, Description: common.LabelSelectorMatchLabelsDescription},
					"match_expressions": schema.ListNestedAttribute{Optional: true, Description: common.LabelSelectorMatchExpressionsDescription, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
						"key":      schema.StringAttribute{Required: true, Description: common.LabelSelectorKeyDescription},
						"operator": schema.StringAttribute{Required: true, Description: common.LabelSelectorOperatorDescription},
						"values":   schema.SetAttribute{Optional: true, ElementType: types.StringType, Description: common.LabelSelectorValuesDescription},
					}}},
				},
			},
			"optional": schema.BoolAttribute{Optional: true, Description: "Whether the Pod may start when the named bundle does not exist or the signer has no matching bundle."},
			"path":     schema.StringAttribute{Required: true, Description: "Path of the bundle file, relative to the volume root.", Validators: []validator.String{podStringRule("path")}},
		},
	}
	if b.replace(immutable) {
		trustBundle.PlanModifiers = []planmodifier.Object{podOptionalObjectRequiresReplace{}}
	}
	filePath := func(description string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Description: description, Validators: append(validators, stringvalidator.LengthAtLeast(1), podStringRule("path"))}
	}
	certificate := schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Projects a private key and a certificate chain that the kubelet requests from signer_name and rotates. Set credential_bundle_path, or key_path and certificate_chain_path. Requires Kubernetes 1.37 or later, or the PodCertificateRequest feature gate on Kubernetes 1.34 to 1.36.",
		Attributes: map[string]schema.Attribute{
			"signer_name": schema.StringAttribute{Required: true, Description: "Signer the kubelet addresses its certificate requests to."},
			"key_type": schema.StringAttribute{
				Required: true, Description: "Type of the generated key pair: RSA3072, RSA4096, ECDSAP256, ECDSAP384, ECDSAP521 or ED25519.",
				Validators: []validator.String{stringvalidator.OneOf("RSA3072", "RSA4096", "ECDSAP256", "ECDSAP384", "ECDSAP521", "ED25519")},
			},
			"max_expiration_seconds": schema.Int64Attribute{
				Optional: true, Description: "Longest certificate lifetime to request, from 3600 to 7862400 seconds; kubernetes.io signers allow at most 86400. Omission lets Kubernetes request 24 hours.",
				Validators: []validator.Int64{int64validator.Between(3600, 7862400)},
			},
			"credential_bundle_path": filePath("Path, relative to the volume root, of a single file holding the private key followed by the certificate chain.",
				stringvalidator.AtLeastOneOf(path.MatchRelative().AtParent().AtName("key_path"), path.MatchRelative().AtParent().AtName("certificate_chain_path"))),
			"key_path":               filePath("Path of the private key file, relative to the volume root."),
			"certificate_chain_path": filePath("Path of the certificate chain file, relative to the volume root."),
			"user_annotations":       schema.MapAttribute{Optional: true, ElementType: types.StringType, Description: "Annotations with domain-prefixed keys that are passed to the signer unverified."},
		},
	}
	if b.replace(immutable) {
		certificate.PlanModifiers = []planmodifier.Object{podOptionalObjectRequiresReplace{}}
	}
	return map[string]schema.Attribute{"cluster_trust_bundle": trustBundle, "pod_certificate": certificate}
}

func (b builder) podProjectedVolumeObject() schema.NestedBlockObject {
	return schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{"default_mode": b.str(false, false, updatable, "0644", podStringRule("mode"))},
		Blocks: map[string]schema.Block{
			"sources": b.block(schema.NestedBlockObject{Attributes: b.podProjectionAttributes(), Blocks: map[string]schema.Block{
				"secret":       b.block(b.podKeyVolumeObject("name", false), 0, 0, updatable),
				"config_map":   b.block(b.podKeyVolumeObject("name", false), 0, 0, updatable),
				"downward_api": b.block(b.podDownwardAPIVolumeObject(true), 0, 1, updatable),
				"service_account_token": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"audience":           b.str(false, false, updatable, ""),
					"expiration_seconds": b.integer(false, false, updatable, 3600, int64validator.AtLeast(600)),
					"path":               b.str(true, false, updatable, ""),
				}}, 0, 1, updatable),
			}}, 1, 0, updatable),
		},
	}
}

func (b builder) podVolumeClaimTemplateObject() schema.NestedBlockObject {
	annotations := b.mapping(false, updatable)
	annotations.Validators = []validator.Map{common.AnnotationsValidator()}
	labels := b.mapping(false, updatable)
	labels.Validators = []validator.Map{common.LabelsValidator()}
	attributesClass := b.str(false, true, alwaysNew, "")
	attributesClass.Description = "Name of the VolumeAttributesClass requested by the claim. Requires Kubernetes 1.34 or later, or the VolumeAttributesClass feature gate on Kubernetes 1.29 to 1.33; without the feature the Pod keeps the name and Kubernetes creates the claim without the class."
	selector := b.podLabelSelectorObject()
	selector.Attributes["match_labels"] = b.mapping(false, alwaysNew)
	selector.Blocks["match_expressions"] = b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
		"key":      b.str(false, false, alwaysNew, ""),
		"operator": b.str(false, false, alwaysNew, ""),
		"values":   b.set(alwaysNew, types.StringType),
	}}, 0, 0, alwaysNew)
	return schema.NestedBlockObject{Blocks: map[string]schema.Block{
		"metadata": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"annotations": annotations, "labels": labels,
		}}, 0, 1, updatable),
		"spec": b.block(schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"access_modes":       b.requiredSet(alwaysNew, setvalidator.ValueStringsAre(stringvalidator.OneOf("ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany", "ReadWriteOncePod"))),
				"volume_name":        b.str(false, true, alwaysNew, ""),
				"storage_class_name": b.str(false, true, alwaysNew, ""),
				"volume_mode":        b.str(false, true, alwaysNew, "", stringvalidator.OneOf("Block", "Filesystem")),

				"volume_attributes_class_name": attributesClass,
			},
			Blocks: map[string]schema.Block{
				"resources": b.block(schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"limits":   b.quantityMap(false, alwaysNew),
					"requests": b.quantityMap(false, updatable),
				}}, 1, 1, updatable),
				"selector": b.block(selector, 0, 1, alwaysNew),
			},
		}, 1, 1, updatable),
	}}
}
