// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/manifest/openapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func init() {
	install.Install(scheme.Scheme)
}

// RawProviderServer implements the ProviderServer interface as exported from ProtoBuf.
type RawProviderServer struct {
	// Since the provider is essentially a gRPC server, the execution flow is dictated by the order of the client (Terraform) request calls.
	// Thus it needs a way to persist state between the gRPC calls. These attributes store values that need to be persisted between gRPC calls,
	// such as instances of the Kubernetes clients, configuration options needed at runtime.
	logger          hclog.Logger
	clientConfig    *rest.Config
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
	restMapper      meta.RESTMapper
	restClient      rest.Interface
	OAPIFoundry     openapi.Foundry

	providerEnabled bool
	hostTFVersion   string
}

func dump(v interface{}) hclog.Format {
	return hclog.Fmt("%v", v)
}

// PrepareProviderConfig function
func (s *RawProviderServer) ValidateProviderConfig(ctx context.Context, req *tfprotov6.ValidateProviderConfigRequest) (*tfprotov6.ValidateProviderConfigResponse, error) {
	s.logger.Trace("[PrepareProviderConfig][Request]\n%s\n", dump(*req))
	resp := &tfprotov6.ValidateProviderConfigResponse{}
	return resp, nil
}

// GetMetadata function
func (s *RawProviderServer) GetMetadata(ctx context.Context, req *tfprotov6.GetMetadataRequest) (*tfprotov6.GetMetadataResponse, error) {
	s.logger.Trace("[GetMetadata][Request]\n%s\n", dump(*req))

	sch := GetProviderResourceSchema()
	rs := make([]tfprotov6.ResourceMetadata, 0, len(sch))
	for k := range sch {
		rs = append(rs, tfprotov6.ResourceMetadata{TypeName: k})
	}

	sch = GetProviderDataSourceSchema()
	ds := make([]tfprotov6.DataSourceMetadata, 0, len(sch))
	for k := range sch {
		ds = append(ds, tfprotov6.DataSourceMetadata{TypeName: k})
	}

	resp := &tfprotov6.GetMetadataResponse{
		Resources:   rs,
		DataSources: ds,
	}
	return resp, nil
}

// ValidateDataSourceConfig function
func (s *RawProviderServer) ValidateDataResourceConfig(ctx context.Context, req *tfprotov6.ValidateDataResourceConfigRequest) (*tfprotov6.ValidateDataResourceConfigResponse, error) {
	s.logger.Trace("[ValidateDataSourceConfig][Request]\n%s\n", dump(*req))
	resp := &tfprotov6.ValidateDataResourceConfigResponse{}
	return resp, nil
}

// StopProvider function
func (s *RawProviderServer) StopProvider(ctx context.Context, req *tfprotov6.StopProviderRequest) (*tfprotov6.StopProviderResponse, error) {
	s.logger.Trace("[StopProvider][Request]\n%s\n", dump(*req))

	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
