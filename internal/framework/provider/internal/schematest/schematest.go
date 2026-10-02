// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package schematest serves Framework resource schemas through the real
// protocol server so that tests can compare what Terraform receives. It is
// imported only by tests.
package schematest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// WithSchema returns r with its Schema method replaced by schemaFunc. Only the
// resource.Resource methods of r are kept, which is all GetProviderSchema uses.
func WithSchema(r resource.Resource, schemaFunc func(context.Context, resource.SchemaRequest, *resource.SchemaResponse)) resource.Resource {
	return schemaOverride{Resource: r, schema: schemaFunc}
}

type schemaOverride struct {
	resource.Resource
	schema func(context.Context, resource.SchemaRequest, *resource.SchemaResponse)
}

func (r schemaOverride) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.schema(ctx, req, resp)
}

// ProtocolSchemaJSON returns the JSON encoding of the GetProviderSchema
// resource schema that a protocol 6 Framework server reports for r, under the
// provider type name "kubernetes". Error diagnostics, including those from
// the framework's schema ValidateImplementation, are returned as an error.
func ProtocolSchemaJSON(ctx context.Context, r resource.Resource) ([]byte, error) {
	var md resource.MetadataResponse
	r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "kubernetes"}, &md)
	server := providerserver.NewProtocol6(&singleResourceProvider{resource: r})()
	resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		return nil, err
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			return nil, fmt.Errorf("%s: %s", d.Summary, d.Detail)
		}
	}
	s, ok := resp.ResourceSchemas[md.TypeName]
	if !ok {
		return nil, fmt.Errorf("no schema returned for %q", md.TypeName)
	}
	return json.Marshal(s)
}

type singleResourceProvider struct {
	resource resource.Resource
}

func (p *singleResourceProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubernetes"
}

func (p *singleResourceProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}

func (p *singleResourceProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}

func (p *singleResourceProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return p.resource }}
}

func (p *singleResourceProvider) DataSources(context.Context) []func() datasource.DataSource {
	return nil
}
