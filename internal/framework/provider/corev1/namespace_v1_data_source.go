// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

// Compile-time checks: NamespaceV1DataSource must satisfy both interfaces.
var (
	_ datasource.DataSource              = (*NamespaceV1DataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*NamespaceV1DataSource)(nil)
)

// NamespaceV1DataSource is the framework implementation of the
// kubernetes_namespace_v1 data source.
type NamespaceV1DataSource struct {
	// SDKv2Meta is a function that returns the provider meta struct initialised
	// by the SDKv2 codebase. All migrated resources and data sources in this
	// provider use this pattern while the mux migration is in progress.
	SDKv2Meta func() any
}

func NewNamespaceV1DataSource() datasource.DataSource {
	return &NamespaceV1DataSource{}
}

func (d *NamespaceV1DataSource) Metadata(
	_ context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_namespace_v1"
}

// Configure receives the provider-level Kubernetes client.
// req.ProviderData is the SDKv2Meta func set in provider_configure.go.
func (d *NamespaceV1DataSource) Configure(
	_ context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	d.SDKv2Meta = req.ProviderData.(func() any)
}
