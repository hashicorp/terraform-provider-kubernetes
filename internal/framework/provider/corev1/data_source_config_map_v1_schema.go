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

func (d *ConfigMapV1DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Config Maps are key-value pairs containing configuration data. The Config Map data source provides a mechanism for extracting these key-value pairs.",
		Blocks: map[string]schema.Block{
			"metadata": common.NamespacedDataSourceMetadataSchema("config_map"),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique ID for this terraform resource",
				Computed:    true,
			},
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
