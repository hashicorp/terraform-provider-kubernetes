// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestyaml

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func (r *ManifestYAML) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Applies a single Kubernetes object from raw YAML using Server-Side Apply " +
			"(SSA). Unlike `kubernetes_manifest`, it does not require the typed OpenAPI schema at plan time. " +
			"See RFC-011.",
		Attributes: map[string]schema.Attribute{
			"yaml_body": schema.StringAttribute{
				MarkdownDescription: "Raw YAML for exactly one Kubernetes object. Marked sensitive because " +
					"the manifest may contain a Secret.",
				Required:  true,
				Sensitive: true,
			},
			"field_manager": schema.StringAttribute{
				MarkdownDescription: "Server-Side Apply field manager name. Use a unique value when sharing " +
					"an object with other managers.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("terraform"),
			},
			"force_conflicts": schema.BoolAttribute{
				MarkdownDescription: "Force apply, taking ownership of fields managed by another field manager.",
				Optional:            true,
			},

			// Computed identity / status.
			"id": schema.StringAttribute{
				MarkdownDescription: "apiVersion=<>,kind=<>,namespace=<>,name=<>",
				Computed:            true,
			},
			"api_version":      schema.StringAttribute{Computed: true},
			"kind":             schema.StringAttribute{Computed: true},
			"name":             schema.StringAttribute{Computed: true},
			"namespace":        schema.StringAttribute{Computed: true},
			"uid":              schema.StringAttribute{Computed: true},
			"resource_version": schema.StringAttribute{Computed: true},
		},
		Blocks: map[string]schema.Block{
			"delete": schema.SingleNestedBlock{
				MarkdownDescription: "Deletion options.",
				Attributes: map[string]schema.Attribute{
					"propagation_policy": schema.StringAttribute{
						MarkdownDescription: "Foreground | Background | Orphan (default: Background).",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("Foreground", "Background", "Orphan"),
						},
					},
					"grace_period_seconds": schema.Int64Attribute{
						Optional: true,
					},
				},
			},
		},
	}
}
