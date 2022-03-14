package tfmux

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

var _ tfprotov5.ProviderServer = SchemaServer{}

var cmpOptions = []cmp.Option{
	cmpopts.SortSlices(func(i, j *tfprotov5.SchemaAttribute) bool {
		return i.Name < j.Name
	}),
	cmpopts.SortSlices(func(i, j *tfprotov5.SchemaNestedBlock) bool {
		return i.TypeName < j.TypeName
	}),
}

// SchemaServerFactory is a generator for SchemaServers, which are Terraform
// gRPC servers that route requests to different gRPC provider implementations
// based on which gRPC provider implementation supports the resource the
// request is for.
//
// SchemaServerFactory should always be instantiated by NewSchemaServerFactory.
type SchemaServerFactory struct {
	// determine which servers will respond to which requests
	resources   map[string]int
	dataSources map[string]int
	servers     []func() tfprotov5.ProviderServer

	// we respond to GetSchema requests using these schemas
	resourceSchemas    map[string]*tfprotov5.Schema
	dataSourceSchemas  map[string]*tfprotov5.Schema
	providerSchema     *tfprotov5.Schema
	providerMetaSchema *tfprotov5.Schema

	// any non-error diagnostics should get bubbled up, so we store them here
	diagnostics []*tfprotov5.Diagnostic

	// we just store these to surface better errors
	// track which server we got the provider schema and provider meta
	// schema from
	providerSchemaFrom     int
	providerMetaSchemaFrom int
}

// NewSchemaServerFactory returns a SchemaServerFactory that will route gRPC
// requests between the tfprotov5.ProviderServers specified. Each function
// specified is called, and the tfprotov5.ProviderServer has its
// GetProviderSchema method called. The schemas are used to determine which
// server handles each request, with requests for resources and data sources
// directed to the server that specified that data source or resource in its
// schema. Data sources and resources can only be specified in the schema of
// one ProviderServer.
func NewSchemaServerFactory(ctx context.Context, servers ...func() tfprotov5.ProviderServer) (SchemaServerFactory, error) {
	var factory SchemaServerFactory

	// know when these are unset vs set to the element in pos 0
	factory.providerSchemaFrom = -1
	factory.providerMetaSchemaFrom = -1

	factory.servers = make([]func() tfprotov5.ProviderServer, len(servers))
	factory.resources = make(map[string]int)
	factory.resourceSchemas = make(map[string]*tfprotov5.Schema)
	factory.dataSources = make(map[string]int)
	factory.dataSourceSchemas = make(map[string]*tfprotov5.Schema)

	for pos, server := range servers {
		s := server()
		resp, err := s.GetProviderSchema(ctx, &tfprotov5.GetProviderSchemaRequest{})
		if err != nil {
			return factory, fmt.Errorf("error retrieving schema for %T: %w", s, err)
		}

		factory.servers[pos] = server

		for _, diag := range resp.Diagnostics {
			if diag == nil {
				continue
			}
			if diag.Severity != tfprotov5.DiagnosticSeverityError {
				factory.diagnostics = append(factory.diagnostics, diag)
				continue
			}
			return factory, fmt.Errorf("error retrieving schema for %T:\n\n\tAttribute: %s\n\tSummary: %s\n\tDetail: %s", s, diag.Attribute, diag.Summary, diag.Detail)
		}
		if resp.Provider != nil && factory.providerSchema != nil && !cmp.Equal(resp.Provider, factory.providerSchema, cmpOptions...) {
			return factory, fmt.Errorf("got a different provider schema from two servers (%T, %T). Provider schemas must be identical across providers. Diff: %s", factory.servers[factory.providerSchemaFrom](), s, cmp.Diff(resp.Provider, factory.providerSchema, cmpOptions...))
		} else if resp.Provider != nil {
			factory.providerSchemaFrom = pos
			factory.providerSchema = resp.Provider
		}
		if resp.ProviderMeta != nil && factory.providerMetaSchema != nil && !cmp.Equal(resp.ProviderMeta, factory.providerMetaSchema, cmpOptions...) {
			return factory, fmt.Errorf("got a different provider_meta schema from two servers (%T, %T). Provider metadata schemas must be identical across providers. Diff: %s", factory.servers[factory.providerMetaSchemaFrom](), s, cmp.Diff(resp.ProviderMeta, factory.providerMetaSchema, cmpOptions...))
		} else if resp.ProviderMeta != nil {
			factory.providerMetaSchemaFrom = pos
			factory.providerMetaSchema = resp.ProviderMeta
		}
		for resource, schema := range resp.ResourceSchemas {
			if v, ok := factory.resources[resource]; ok {
				return factory, fmt.Errorf("resource %q supported by multiple server implementations (%T, %T); remove support from one", resource, factory.servers[v], s)
			}
			factory.resources[resource] = pos
			factory.resourceSchemas[resource] = schema
		}
		for data, schema := range resp.DataSourceSchemas {
			if v, ok := factory.dataSources[data]; ok {
				return factory, fmt.Errorf("data source %q supported by multiple server implementations (%T, %T); remove support from one", data, factory.servers[v], s)
			}
			factory.dataSources[data] = pos
			factory.dataSourceSchemas[data] = schema
		}
	}
	return factory, nil
}

func (s SchemaServerFactory) getSchemaHandler(_ context.Context, _ *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error) {
	return &tfprotov5.GetProviderSchemaResponse{
		Provider:          s.providerSchema,
		ResourceSchemas:   s.resourceSchemas,
		DataSourceSchemas: s.dataSourceSchemas,
		ProviderMeta:      s.providerMetaSchema,
	}, nil
}

// Server returns the SchemaServer that muxes between the
// tfprotov5.ProviderServers associated with the SchemaServerFactory.
func (s SchemaServerFactory) Server() SchemaServer {
	res := SchemaServer{
		getSchemaHandler:            s.getSchemaHandler,
		prepareProviderConfigServer: s.providerSchemaFrom,
		servers:                     make([]tfprotov5.ProviderServer, len(s.servers)),
	}
	for pos, server := range s.servers {
		res.servers[pos] = server()
	}
	res.resources = make(map[string]tfprotov5.ProviderServer)
	for r, pos := range s.resources {
		res.resources[r] = res.servers[pos]
	}
	res.dataSources = make(map[string]tfprotov5.ProviderServer)
	for ds, pos := range s.dataSources {
		res.dataSources[ds] = res.servers[pos]
	}
	return res
}

// SchemaServer is a gRPC server implementation that stands in front of other
// gRPC servers, routing requests to them as if they were a single server. It
// should always be instantiated by calling SchemaServerFactory.Server().
type SchemaServer struct {
	resources   map[string]tfprotov5.ProviderServer
	dataSources map[string]tfprotov5.ProviderServer
	servers     []tfprotov5.ProviderServer

	getSchemaHandler            func(context.Context, *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error)
	prepareProviderConfigServer int
}

// GetProviderSchema merges the schemas returned by the
// tfprotov5.ProviderServers associated with SchemaServer into a single schema.
// Resources and data sources must be returned from only one server. Provider
// and ProviderMeta schemas must be identical between all servers.
func (s SchemaServer) GetProviderSchema(ctx context.Context, req *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error) {
	return s.getSchemaHandler(ctx, req)
}

// PrepareProviderConfig calls the PrepareProviderConfig method on each server
// in order, passing `req`. Only one may respond with a non-nil PreparedConfig
// or a non-empty Diagnostics.
func (s SchemaServer) PrepareProviderConfig(ctx context.Context, req *tfprotov5.PrepareProviderConfigRequest) (*tfprotov5.PrepareProviderConfigResponse, error) {
	respondedServer := -1
	var resp *tfprotov5.PrepareProviderConfigResponse
	for pos, server := range s.servers {
		res, err := server.PrepareProviderConfig(ctx, req)
		if err != nil {
			return resp, fmt.Errorf("error from %T preparing provider config: %w", server, err)
		}
		if res == nil {
			continue
		}
		if res.PreparedConfig != nil || len(res.Diagnostics) > 0 {
			if respondedServer >= 0 {
				return nil, fmt.Errorf("got a PrepareProviderConfig response from multiple servers, %d and %d, not sure which to use", respondedServer, pos)
			}
			resp = res
			respondedServer = pos
			continue
		}
	}
	return resp, nil
}

// ValidateResourceTypeConfig calls the ValidateResourceTypeConfig method,
// passing `req`, on the provider that returned the resource specified by
// req.TypeName in its schema.
func (s SchemaServer) ValidateResourceTypeConfig(ctx context.Context, req *tfprotov5.ValidateResourceTypeConfigRequest) (*tfprotov5.ValidateResourceTypeConfigResponse, error) {
	h, ok := s.resources[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}
	return h.ValidateResourceTypeConfig(ctx, req)
}

// ValidateDataSourceConfig calls the ValidateDataSourceConfig method, passing
// `req`, on the provider that returned the data source specified by
// req.TypeName in its schema.
func (s SchemaServer) ValidateDataSourceConfig(ctx context.Context, req *tfprotov5.ValidateDataSourceConfigRequest) (*tfprotov5.ValidateDataSourceConfigResponse, error) {
	h, ok := s.dataSources[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}
	return h.ValidateDataSourceConfig(ctx, req)
}

// UpgradeResourceState calls the UpgradeResourceState method, passing `req`,
// on the provider that returned the resource specified by req.TypeName in its
// schema.
func (s SchemaServer) UpgradeResourceState(ctx context.Context, req *tfprotov5.UpgradeResourceStateRequest) (*tfprotov5.UpgradeResourceStateResponse, error) {
	h, ok := s.resources[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}
	return h.UpgradeResourceState(ctx, req)
}

// ConfigureProvider calls each provider's ConfigureProvider method, one at a
// time, passing `req`. Any Diagnostic with severity error will abort the
// process and return immediately; non-Error severity Diagnostics will be
// combined and returned.
func (s SchemaServer) ConfigureProvider(ctx context.Context, req *tfprotov5.ConfigureProviderRequest) (*tfprotov5.ConfigureProviderResponse, error) {
	var diags []*tfprotov5.Diagnostic
	for _, server := range s.servers {
		resp, err := server.ConfigureProvider(ctx, req)
		if err != nil {
			return resp, fmt.Errorf("error configuring %T: %w", server, err)
		}
		for _, diag := range resp.Diagnostics {
			if diag == nil {
				continue
			}
			diags = append(diags, diag)
			if diag.Severity != tfprotov5.DiagnosticSeverityError {
				continue
			}
			resp.Diagnostics = diags
			return resp, err
		}
	}
	return &tfprotov5.ConfigureProviderResponse{Diagnostics: diags}, nil
}

// ReadResource calls the ReadResource method, passing `req`, on the provider
// that returned the resource specified by req.TypeName in its schema.
func (s SchemaServer) ReadResource(ctx context.Context, req *tfprotov5.ReadResourceRequest) (*tfprotov5.ReadResourceResponse, error) {
	h, ok := s.resources[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}
	return h.ReadResource(ctx, req)
}

// PlanResourceChange calls the PlanResourceChange method, passing `req`, on
// the provider that returned the resource specified by req.TypeName in its
// schema.
func (s SchemaServer) PlanResourceChange(ctx context.Context, req *tfprotov5.PlanResourceChangeRequest) (*tfprotov5.PlanResourceChangeResponse, error) {
	h, ok := s.resources[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}
	return h.PlanResourceChange(ctx, req)
}

// ApplyResourceChange calls the ApplyResourceChange method, passing `req`, on
// the provider that returned the resource specified by req.TypeName in its
// schema.
func (s SchemaServer) ApplyResourceChange(ctx context.Context, req *tfprotov5.ApplyResourceChangeRequest) (*tfprotov5.ApplyResourceChangeResponse, error) {
	h, ok := s.resources[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}
	return h.ApplyResourceChange(ctx, req)
}

// ImportResourceState calls the ImportResourceState method, passing `req`, on
// the provider that returned the resource specified by req.TypeName in its
// schema.
func (s SchemaServer) ImportResourceState(ctx context.Context, req *tfprotov5.ImportResourceStateRequest) (*tfprotov5.ImportResourceStateResponse, error) {
	h, ok := s.resources[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}
	return h.ImportResourceState(ctx, req)
}

// ReadDataSource calls the ReadDataSource method, passing `req`, on the
// provider that returned the data source specified by req.TypeName in its
// schema.
func (s SchemaServer) ReadDataSource(ctx context.Context, req *tfprotov5.ReadDataSourceRequest) (*tfprotov5.ReadDataSourceResponse, error) {
	h, ok := s.dataSources[req.TypeName]
	if !ok {
		return nil, fmt.Errorf("%q isn't supported by any servers", req.TypeName)
	}
	return h.ReadDataSource(ctx, req)
}

// StopProvider calls the StopProvider function for each provider associated
// with the SchemaServer, one at a time. All Error fields will be joined
// together and returned, but will not prevent the rest of the providers'
// StopProvider methods from being called.
func (s SchemaServer) StopProvider(ctx context.Context, req *tfprotov5.StopProviderRequest) (*tfprotov5.StopProviderResponse, error) {
	var errs []string
	for _, server := range s.servers {
		resp, err := server.StopProvider(ctx, req)
		if err != nil {
			return resp, fmt.Errorf("error stopping %T: %w", server, err)
		}
		if resp.Error != "" {
			errs = append(errs, resp.Error)
		}
	}
	return &tfprotov5.StopProviderResponse{
		Error: strings.Join(errs, "\n"),
	}, nil
}
