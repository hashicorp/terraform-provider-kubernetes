// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package gatewaynetworkingv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GatewayClassDataSource implements the kubernetes_gateway_class_v1 data
// source. It shares clientProvider, gatewayClassSpec and the flatteners with
// the resource, so both report an object the same way.
type GatewayClassDataSource struct{ clientProvider }

// gatewayClassDataModel uses common.MetadataBase, which has no generate_name
// because a lookup is by exact name, and has no timeouts: Read always uses
// defaultTimeout.
type gatewayClassDataModel struct {
	ID       types.String          `tfsdk:"id"`
	Metadata []common.MetadataBase `tfsdk:"metadata"`
	Spec     []gatewayClassSpec    `tfsdk:"spec"`
	Status   types.Object          `tfsdk:"status"`
}

var _ datasource.DataSourceWithConfigure = (*GatewayClassDataSource)(nil)

// NewGatewayClassDataSource is registered in the DataSources list
func NewGatewayClassDataSource() datasource.DataSource { return &GatewayClassDataSource{} }

// Metadata implements [datasource.DataSource].
func (d *GatewayClassDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway_class_v1"
}

// Configure implements [datasource.DataSourceWithConfigure].
func (d *GatewayClassDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(d.configure(req.ProviderData)...)
}

// Schema implements [datasource.DataSource].
func (d *GatewayClassDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	metadata := common.DataSourceMetadataSchema("GatewayClass")
	// New lookups accept only a name; all other metadata is observed, not configured.
	metadata.NestedObject.Attributes["labels"] = schema.MapAttribute{Computed: true, ElementType: types.StringType, Description: "All observed labels, including controller-owned labels."}
	metadata.NestedObject.Attributes["annotations"] = schema.MapAttribute{Computed: true, ElementType: types.StringType, Description: "All observed annotations."}
	resp.Schema = schema.Schema{
		Description: "Reads an existing cluster-scoped Gateway API v1 GatewayClass without managing it.",
		Blocks:      map[string]schema.Block{"metadata": metadata},
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true, Description: "Cluster-scoped GatewayClass name."},
			"status": schema.ObjectAttribute{Computed: true, AttributeTypes: statusTypes, Description: "Controller conditions and supported_features."},
			"spec": schema.ListNestedAttribute{
				Computed: true, Description: "Observed GatewayClass specification.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"controller_name": schema.StringAttribute{Computed: true, Description: "Controller identifier."},
					"description":     schema.StringAttribute{Computed: true, Description: "Human-readable class description."},
					"parameters_ref": schema.ListNestedAttribute{
						Computed: true, Description: "Controller-specific parameters reference.",
						NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
							"group":     schema.StringAttribute{Computed: true, Description: "Referenced API group; empty for the core group."},
							"kind":      schema.StringAttribute{Computed: true, Description: "Referenced Kubernetes kind."},
							"name":      schema.StringAttribute{Computed: true, Description: "Referenced object name."},
							"namespace": schema.StringAttribute{Computed: true, Description: "Namespace of a namespaced referent."},
						}},
					},
				}},
			},
		},
	}
}

// Read implements [datasource.DataSource]. A missing GatewayClass is an error
// rather than an empty result, and discovery tells it apart from a cluster
// without the Gateway API CRDs. Metadata is returned unfiltered, including
// controller-owned keys, because a data source manages nothing.
func (d *GatewayClassDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model gatewayClassDataModel
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata"), &model.Metadata)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, discover, _, diags := d.clients()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	object, err := client.Get(ctx, model.Metadata[0].Name.ValueString(), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if endpointErr := gatewayClassEndpoint(ctx, discover); endpointErr != nil {
				resp.Diagnostics.AddError("GatewayClass API unavailable", endpointErr.Error())
				return
			}
		}
		resp.Diagnostics.AddError("Error reading GatewayClass", err.Error())
		return
	}
	meta, diags := common.FlattenDataSourceMetadataFields(ctx, object.ObjectMeta)
	resp.Diagnostics.Append(diags...)
	model.Metadata = []common.MetadataBase{meta}
	model.ID = types.StringValue(object.Name)
	model.Spec = []gatewayClassSpec{flattenSpec(object.Spec)}
	model.Status, diags = flattenStatus(ctx, object.Status)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
