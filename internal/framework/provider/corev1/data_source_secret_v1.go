// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

var (
	_ datasource.DataSource              = (*SecretV1DataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*SecretV1DataSource)(nil)
)

// SecretV1DataSource is the Framework data source for kubernetes_secret_v1.
type SecretV1DataSource struct {
	SDKv2Meta func() any
}

// NewSecretV1DataSource returns a new instance of the SecretV1DataSource.
func NewSecretV1DataSource() datasource.DataSource {
	return &SecretV1DataSource{}
}

// Metadata sets the type name for this data source.
func (d *SecretV1DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret_v1"
}

// Configure stores the provider meta function for later use in Read.
func (d *SecretV1DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.SDKv2Meta = req.ProviderData.(func() any)
}
