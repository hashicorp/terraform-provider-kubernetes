// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *ConfigMapV1DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Config Maps are key-value pairs containing configuration data. The Config Map data source provides a mechanism for extracting these key-value pairs.",
		Blocks: map[string]schema.Block{
			"metadata": schema.ListNestedBlock{
				Description: "Standard config_map's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Name of the config_map, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Required:    true,
						},
						"namespace": schema.StringAttribute{
							Description: "Namespace defines the space within which name of the config_map must be unique.",
							Optional:    true,
						},
						"annotations": schema.MapAttribute{
							Description: "An unstructured key value map stored with the config_map that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
							ElementType: types.StringType,
							Optional:    true,
						},
						"labels": schema.MapAttribute{
							Description: "Map of string keys and values that can be used to organize and categorize (scope and select) the config_map. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
							ElementType: types.StringType,
							Optional:    true,
						},
						"resource_version": schema.StringAttribute{
							Description: "An opaque value that represents the internal version of this config_map that can be used by clients to determine when config_map has changed.",
							Computed:    true,
						},
						"uid": schema.StringAttribute{
							Description: "The unique in time and space value for this config_map. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
							Computed:    true,
						},
						"generation": schema.Int64Attribute{
							Description: "A sequence number representing a specific generation of the desired state.",
							Computed:    true,
						},
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"data": schema.MapAttribute{
				Description: "A map of the config map data.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"binary_data": schema.MapAttribute{
				Description: "A map of the config map binary data.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"immutable": schema.BoolAttribute{
				Description: "Immutable, if set to true, ensures that data stored in the ConfigMap cannot be updated (only object metadata can be modified). If not set to true, the field can be modified at any time. Defaulted to nil.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}
