// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var (
	_ datasource.DataSource              = (*SecretV1DataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*SecretV1DataSource)(nil)
)

// SecretV1DataSource is the Framework data source for kubernetes_secret_v1.
type SecretV1DataSource struct {
	// SDKv2Meta must stay func() any: that is the concrete type stored in
	// ProviderData by internal/framework/provider/provider_configure.go. Go
	// function types are invariant, so asserting to func() kubernetes.KubeClientsets
	// would compile and panic at runtime — assert the result instead.
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

	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	d.SDKv2Meta = sdkv2Meta
}

// clientset resolves the SDKv2 provider metadata into the KubeClientsets interface the
// data source needs. The call is deferred until the request rather than made in Configure,
// because the mux server configures the SDKv2 provider independently and its meta is not
// populated until that happens.
//
// The data source does not need kubernetes.MetadataFilters: unlike the resource Read path,
// it flattens metadata unfiltered, matching the SDKv2 data source's flattenMetadataFields.
func (d *SecretV1DataSource) clientset() (kubernetes.KubeClientsets, diag.Diagnostics) {
	var diags diag.Diagnostics

	if d.SDKv2Meta == nil {
		diags.AddError("Provider not configured",
			"The SDKv2 provider metadata is unavailable. This is a bug in the provider.")
		return nil, diags
	}

	meta := d.SDKv2Meta()
	clients, ok := meta.(kubernetes.KubeClientsets)
	if !ok {
		diags.AddError("Unexpected provider data",
			fmt.Sprintf("Expected kubernetes.KubeClientsets, got %T. This is a bug in the provider.", meta))
		return nil, diags
	}

	return clients, diags
}
