// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
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
			// Namespace is cluster-scoped. common reproduces SDKv2's
			// metadataSchema("namespace", false) attribute for attribute, including the
			// Optional/Computed flags and descriptions, so state paths and generated docs
			// are unchanged.
			"metadata": common.DataSourceMetadataSchema("namespace"),
		},
	}
}
