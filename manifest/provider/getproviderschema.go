// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// GetProviderSchema function
func (s *RawProviderServer) GetProviderSchema(ctx context.Context, req *tfprotov6.GetProviderSchemaRequest) (*tfprotov6.GetProviderSchemaResponse, error) {
	cfgSchema := GetProviderConfigSchema()
	resSchema := GetProviderResourceSchema()
	dsSchema := GetProviderDataSourceSchema()

	return &tfprotov6.GetProviderSchemaResponse{
		Provider:          cfgSchema,
		ResourceSchemas:   resSchema,
		DataSourceSchemas: dsSchema,
	}, nil
}
