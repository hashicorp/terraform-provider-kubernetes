package provider

import (
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// GetProviderConfigSchema contains the definitions of all configuration attributes
func GetProviderConfigSchema() *tfprotov5.Schema {
	b := tfprotov5.SchemaBlock{
		Attributes: []*tfprotov5.SchemaAttribute{
			{
				Name:            "host",
				Type:            tftypes.String,
				Description:     "Host must be a host string, a host:port pair, or a URL to the base of the apiserver.",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "username",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "password",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       true,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "client_certificate",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "client_key",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       true,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "cluster_ca_certificate",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "config_path",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "config_context",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "config_context_user",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "config_context_cluster",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "token",
				Type:            tftypes.String,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       true,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name:            "insecure",
				Type:            tftypes.Bool,
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
			{
				Name: "exec",
				Type: tftypes.Object{
					AttributeTypes: map[string]tftypes.Type{
						"api_version": tftypes.String,
						"command":     tftypes.String,
						"env":         tftypes.Map{AttributeType: tftypes.String},
						"args":        tftypes.List{ElementType: tftypes.String},
					},
				},
				Description:     "",
				Required:        false,
				Optional:        true,
				Computed:        false,
				Sensitive:       false,
				DescriptionKind: 0,
				Deprecated:      false,
			},
		},
	}

	return &tfprotov5.Schema{
		Version: 1,
		Block:   &b,
	}
}

// GetTypeFromSchema returns the equivalent tftypes.Type representation of a given tfprotov5.Schema
func GetTypeFromSchema(s *tfprotov5.Schema) tftypes.Type {
	schemaTypeAttributes := map[string]tftypes.Type{}
	for _, att := range s.Block.Attributes {
		schemaTypeAttributes[att.Name] = att.Type
	}
	return tftypes.Object{
		AttributeTypes: schemaTypeAttributes,
	}
}
