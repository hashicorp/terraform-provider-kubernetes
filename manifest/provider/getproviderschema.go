// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
)

// GetProviderSchema function
func (s *RawProviderServer) GetProviderSchema(ctx context.Context, req *tfprotov5.GetProviderSchemaRequest) (*tfprotov5.GetProviderSchemaResponse, error) {
	cfgSchema := GetProviderConfigSchema()
	resSchema := GetProviderResourceSchema()
	dsSchema := GetProviderDataSourceSchema()

	return &tfprotov5.GetProviderSchemaResponse{
		Provider:          cfgSchema,
		ResourceSchemas:   resSchema,
		DataSourceSchemas: dsSchema,
	}, nil
}
