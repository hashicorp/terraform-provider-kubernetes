// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Schema translates the SDKv2 schema defined in
// dataSourceKubernetesNamespaceV1 (data_source_kubernetes_namespace_v1.go)
// into the plugin framework equivalent.
//
// Key translation decisions:
//   - metadata uses ListNestedBlock (not SingleNestedAttribute) to preserve the
//     SDKv2 state path metadata[0].* — required for state compatibility.
//   - metadata.name is Required; all other metadata fields are Computed so the
//     API fills them in, but annotations and labels are also Optional to preserve
//     the SDKv2 contract (users were allowed to set them in config).
//   - spec uses ListNestedAttribute (not ListNestedBlock) with Computed:true so
//     that Terraform can plan the collection as unknown on a deferred read and
//     avoid a "Provider produced inconsistent final plan" error when name is
//     unknown at plan time.
func (d *NamespaceV1DataSource) Schema(
	_ context.Context,
	_ datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description: "This data source provides a mechanism to query attributes of any specific namespace within a Kubernetes cluster. In Kubernetes, namespaces provide a scope for names and are intended as a way to divide cluster resources between multiple users.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			// spec is a ListNestedAttribute (not a Block) so that Terraform can plan it
			// as unknown when the data source has unknown inputs and the read is deferred
			// to apply time. A ListNestedBlock would be planned as a known empty list,
			// causing an "inconsistent final plan" error when Read populates finalizers.
			// This mirrors the SDKv2 schema where spec is a Computed TypeList.
			"spec": schema.ListNestedAttribute{
				Description: "Spec defines the behavior of the Namespace.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"finalizers": schema.ListAttribute{
							Description: "Finalizers is an opaque list of values that must be empty to permanently remove object from storage.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			// metadata mirrors metadataSchema("namespace", false) in schema_metadata.go.
			// ListNestedBlock with SizeBetween(1,1) is the framework equivalent of
			// TypeList + MaxItems:1 — produces identical state paths (metadata[0].*).
			"metadata": schema.ListNestedBlock{
				Description: "Standard namespace metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata",
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						// name is the lookup key — the user provides it.
						"name": schema.StringAttribute{
							Description: "Name of the namespace, must be unique. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Required:    true,
						},
						// annotations and labels are Optional+Computed: Optional preserves
						// the SDKv2 contract (users could set them in config); Computed lets
						// the API overwrite them on every read, which is the correct
						// data-source behaviour.
						"annotations": schema.MapAttribute{
							Description: "An unstructured key value map stored with the namespace that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
							ElementType: types.StringType,
							Optional:    true,
							Computed:    true,
						},
						"generation": schema.Int64Attribute{
							Description: "A sequence number representing a specific generation of the desired state.",
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: "Map of string keys and values that can be used to organize and categorize the namespace. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
							ElementType: types.StringType,
							Optional:    true,
							Computed:    true,
						},
						"resource_version": schema.StringAttribute{
							Description: "An opaque value that represents the internal version of this namespace that can be used by clients to determine when the namespace has changed. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
							Computed:    true,
						},
						"uid": schema.StringAttribute{
							Description: "The unique in time and space value for this namespace. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
