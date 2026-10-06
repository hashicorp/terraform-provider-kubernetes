// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var (
	_ datasource.DataSource              = (*ConfigMapV1DataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*ConfigMapV1DataSource)(nil)
)

type ConfigMapV1DataSource struct {
	SDKv2Meta func() any
}

func NewConfigMapV1DataSource() datasource.DataSource {
	return &ConfigMapV1DataSource{}
}

func (d *ConfigMapV1DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config_map_v1"
}

func (d *ConfigMapV1DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.SDKv2Meta = req.ProviderData.(func() any)
}
