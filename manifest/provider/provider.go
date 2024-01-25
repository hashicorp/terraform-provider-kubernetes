// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// GetObjectTypeFromSchema returns a tftypes.Type that can wholy represent the schema input
func GetObjectTypeFromSchema(schema *tfprotov5.Schema) tftypes.Type {
	bm := map[string]tftypes.Type{}

	for _, att := range schema.Block.Attributes {
		bm[att.Name] = att.Type
	}

	for _, b := range schema.Block.BlockTypes {
		a := map[string]tftypes.Type{}
		for _, att := range b.Block.Attributes {
			a[att.Name] = att.Type
		}
		bm[b.TypeName] = tftypes.List{
			ElementType: tftypes.Object{AttributeTypes: a},
		}

		// FIXME we can make this function recursive to handle
		// n levels of nested blocks
		for _, bb := range b.Block.BlockTypes {
			aa := map[string]tftypes.Type{}
			for _, att := range bb.Block.Attributes {
				aa[att.Name] = att.Type
			}
			a[bb.TypeName] = tftypes.List{
				ElementType: tftypes.Object{AttributeTypes: aa},
			}
		}
	}

	return tftypes.Object{AttributeTypes: bm}
}

// GetResourceType returns the tftypes.Type of a resource of type 'name'
func GetResourceType(name string) (tftypes.Type, error) {
	sch := GetProviderResourceSchema()
	rsch, ok := sch[name]
	if !ok {
		return tftypes.DynamicPseudoType, fmt.Errorf("unknown resource %s - cannot find schema", name)
	}
	return GetObjectTypeFromSchema(rsch), nil
}

// GetDataSourceType returns the tftypes.Type of a datasource of type 'name'
func GetDataSourceType(name string) (tftypes.Type, error) {
	sch := GetProviderDataSourceSchema()
	rsch, ok := sch[name]
	if !ok {
		return tftypes.DynamicPseudoType, fmt.Errorf("unknown data source %q: cannot find schema", name)
	}
	return GetObjectTypeFromSchema(rsch), nil
}

// GetProviderResourceSchema contains the definitions of all supported resources
func GetProviderResourceSchema() map[string]*tfprotov5.Schema {
	return map[string]*tfprotov5.Schema{
		"kubernetes_manifest": {
			Version: 1,
			Block: &tfprotov5.SchemaBlock{
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "timeouts",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
						MinItems: 0,
						MaxItems: 1,
						Block: &tfprotov5.SchemaBlock{
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:        "create",
									Type:        tftypes.String,
									Description: "Timeout for the create operation.",
									Optional:    true,
								},
								{
									Name:        "update",
									Type:        tftypes.String,
									Description: "Timeout for the update operation.",
									Optional:    true,
								},
								{
									Name:        "delete",
									Type:        tftypes.String,
									Description: "Timeout for the delete operation.",
									Optional:    true,
								},
							},
						},
					},
					{
						TypeName: "field_manager",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
						MinItems: 0,
						MaxItems: 1,
						Block: &tfprotov5.SchemaBlock{
							Description: "Configure field manager options.",
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:            "name",
									Type:            tftypes.String,
									Required:        false,
									Optional:        true,
									Computed:        false,
									Sensitive:       false,
									Description:     "The name to use for the field manager when creating and updating the resource.",
									DescriptionKind: 0,
									Deprecated:      false,
								},
								{
									Name:            "force_conflicts",
									Type:            tftypes.Bool,
									Required:        false,
									Optional:        true,
									Computed:        false,
									Sensitive:       false,
									Description:     "Force changes against conflicts.",
									DescriptionKind: 0,
									Deprecated:      false,
								},
							},
						},
					},
					{
						TypeName: "wait",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
						MinItems: 0,
						MaxItems: 1,
						Block: &tfprotov5.SchemaBlock{
							Description: "Configure waiter options.",
							BlockTypes: []*tfprotov5.SchemaNestedBlock{
								{
									TypeName: "condition",
									Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
									MinItems: 0,
									Block: &tfprotov5.SchemaBlock{
										Attributes: []*tfprotov5.SchemaAttribute{
											{
												Name:        "status",
												Type:        tftypes.String,
												Optional:    true,
												Description: "The condition status.",
											}, {
												Name:        "type",
												Type:        tftypes.String,
												Optional:    true,
												Description: "The type of condition.",
											},
										},
									},
								},
							},
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:        "rollout",
									Type:        tftypes.Bool,
									Optional:    true,
									Description: "Wait for rollout to complete on resources that support `kubectl rollout status`.",
								},
								{
									Name:        "fields",
									Type:        tftypes.Map{ElementType: tftypes.String},
									Optional:    true,
									Description: "A map of paths to fields to wait for a specific field value.",
								},
							},
						},
					},
				},
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:        "manifest",
						Type:        tftypes.DynamicPseudoType,
						Required:    true,
						Description: "A Kubernetes manifest describing the desired state of the resource in HCL format.",
					},
					{
						Name:        "object",
						Type:        tftypes.DynamicPseudoType,
						Optional:    true,
						Computed:    true,
						Description: "The resulting resource state, as returned by the API server after applying the desired state from `manifest`.",
					},
					{
						Name: "wait_for",
						Type: tftypes.Object{
							AttributeTypes: map[string]tftypes.Type{
								"fields": tftypes.Map{
									ElementType: tftypes.String,
								},
							},
						},
						Optional:    true,
						Deprecated:  true,
						Description: "A map of attribute paths and desired patterns to be matched. After each apply the provider will wait for all attributes listed here to reach a value that matches the desired pattern.",
					},
					{
						Name:        "computed_fields",
						Type:        tftypes.List{ElementType: tftypes.String},
						Description: "List of manifest fields whose values can be altered by the API server during 'apply'. Defaults to: [\"metadata.annotations\", \"metadata.labels\"]",
						Optional:    true,
					},
				},
			},
		},
	}
}

// GetProviderDataSourceSchema contains the definitions of all supported data sources
func GetProviderDataSourceSchema() map[string]*tfprotov5.Schema {
	return map[string]*tfprotov5.Schema{
		"kubernetes_resource": {
			Version: 1,
			Block: &tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:        "api_version",
						Type:        tftypes.String,
						Required:    true,
						Description: "The resource apiVersion.",
					},
					{
						Name:        "kind",
						Type:        tftypes.String,
						Required:    true,
						Description: "The resource kind.",
					},
					{
						Name:        "object",
						Type:        tftypes.DynamicPseudoType,
						Optional:    true,
						Computed:    true,
						Description: "The response from the API server.",
					},
				},
				BlockTypes: []*tfprotov5.SchemaNestedBlock{
					{
						TypeName: "metadata",
						Nesting:  tfprotov5.SchemaNestedBlockNestingModeList,
						MinItems: 1,
						MaxItems: 1,
						Block: &tfprotov5.SchemaBlock{
							Description: "Metadata for the resource",
							Attributes: []*tfprotov5.SchemaAttribute{
								{
									Name:        "name",
									Type:        tftypes.String,
									Required:    true,
									Description: "The resource name.",
								},
								{
									Name:        "namespace",
									Type:        tftypes.String,
									Optional:    true,
									Description: "The resource namespace.",
								},
							},
						},
					},
				},
			},
		},
		"kubernetes_resources": {
			Version: 1,
			Block: &tfprotov5.SchemaBlock{
				Attributes: []*tfprotov5.SchemaAttribute{
					{
						Name:        "api_version",
						Type:        tftypes.String,
						Required:    true,
						Description: "The resource apiVersion.",
					},
					{
						Name:        "kind",
						Type:        tftypes.String,
						Required:    true,
						Description: "The resource kind.",
					},
					{
						Name:        "objects",
						Type:        tftypes.DynamicPseudoType,
						Optional:    true,
						Computed:    true,
						Description: "The response from the API server.",
					},
					{
						Name:        "namespace",
						Type:        tftypes.String,
						Optional:    true,
						Description: "The resource namespace.",
					},
					{
						Name:        "field_selector",
						Type:        tftypes.String,
						Optional:    true,
						Description: "A selector to restrict the list of returned objects by their fields.",
					},
					{
						Name:        "label_selector",
						Type:        tftypes.String,
						Optional:    true,
						Description: "A selector to restrict the list of returned objects by their labels.",
					},
					{
						Name:        "limit",
						Type:        tftypes.Number,
						Optional:    true,
						Description: "Limit is a maximum number of responses to return for a list call.",
					},
				},
			},
		},
	}
}
