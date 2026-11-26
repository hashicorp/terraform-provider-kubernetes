// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package gatewaynetworkingv1

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	kvalidation "k8s.io/apimachinery/pkg/util/validation"
)

// GatewayClassResource implements kubernetes_gateway_class_v1.
type GatewayClassResource struct{ clientProvider }

var (
	_ resource.ResourceWithConfigure      = (*GatewayClassResource)(nil)
	_ resource.ResourceWithImportState    = (*GatewayClassResource)(nil)
	_ resource.ResourceWithIdentity       = (*GatewayClassResource)(nil)
	_ resource.ResourceWithValidateConfig = (*GatewayClassResource)(nil)
)

// NewGatewayClass is registered in the Resources list
func NewGatewayClass() resource.Resource { return &GatewayClassResource{} }

// Metadata implements [resource.Resource].
func (r *GatewayClassResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway_class_v1"
}

// Configure implements [resource.ResourceWithConfigure].
func (r *GatewayClassResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(r.configure(req.ProviderData)...)
}

// Schema implements [resource.Resource]. GatewayClass is cluster-scoped, so the
// shared metadata block has no namespace. Field validators copy the CRD's
// OpenAPI rules, so invalid values fail at plan time rather than during apply;
// controller_name forces replacement because the CRD makes it immutable, and
// status is computed only because the controller writes it.
func (r *GatewayClassResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a cluster-scoped Gateway API v1 GatewayClass. Requires preinstalled Gateway API CRDs. Creation does not wait for controller acceptance.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true, Description: "Cluster-scoped GatewayClass name.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"status": schema.ObjectAttribute{Computed: true, AttributeTypes: statusTypes, Description: "Controller-reported conditions and supported_features. Conditions include type, status, reason, message, observed_generation and last_transition_time."},
		},
		Blocks: map[string]schema.Block{
			"metadata": common.MetadataSchema("GatewayClass", true),
			"timeouts": timeouts.BlockAll(ctx),
			"spec": schema.ListNestedBlock{
				Description: "Desired GatewayClass configuration. Exactly one block is required.",
				Validators:  []validator.List{listvalidator.IsRequired(), listvalidator.SizeBetween(1, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"controller_name": schema.StringAttribute{
							Required: true, Description: "Domain-prefixed controller identifier. Immutable.",
							PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 253),
								stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\/[A-Za-z0-9/\-._~%!$&'()*+,;=:]+$`), "must be a domain-prefixed controller identifier"),
							},
						},
						"description": schema.StringAttribute{Optional: true, Validators: []validator.String{stringvalidator.LengthAtMost(64)}, Description: "Human-readable description, at most 64 characters."},
					},
					Blocks: map[string]schema.Block{
						"parameters_ref": schema.ListNestedBlock{
							Description: "Optional controller-specific parameters reference.",
							Validators:  []validator.List{listvalidator.SizeAtMost(1)},
							NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
								"group": schema.StringAttribute{Required: true, Description: "API group. Use an empty string for the core group.", Validators: []validator.String{
									stringvalidator.LengthAtMost(253),
									stringvalidator.RegexMatches(regexp.MustCompile(`^$|^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`), "must be an API group or empty for the core group"),
								}},
								"kind":      schema.StringAttribute{Required: true, Description: "Referenced Kubernetes kind.", Validators: []validator.String{stringvalidator.LengthBetween(1, 63), stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z]([-a-zA-Z0-9]*[a-zA-Z0-9])?$`), "must be a Kubernetes kind")}},
								"name":      schema.StringAttribute{Required: true, Description: "Referenced object name.", Validators: []validator.String{stringvalidator.LengthBetween(1, 253)}},
								"namespace": schema.StringAttribute{Optional: true, Description: "Required for a namespaced referent; omit for a cluster-scoped referent.", Validators: []validator.String{stringvalidator.LengthBetween(1, 63), stringvalidator.RegexMatches(regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`), "must be a namespace name")}},
							}},
						},
					},
				},
			},
		},
	}
}

// ValidateConfig implements [resource.ResourceWithValidateConfig]. The shared
// metadata block makes name and generate_name conflict but requires neither,
// and the API needs one of them. Only null is rejected: an unknown value, or
// an unknown metadata block, may still resolve during apply.
func (r *GatewayClassResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var value types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata"), &value)...)
	if resp.Diagnostics.HasError() || value.IsUnknown() || value.IsNull() {
		return
	}
	var metadata []common.MetadataModel
	resp.Diagnostics.Append(value.ElementsAs(ctx, &metadata, false)...)
	if resp.Diagnostics.HasError() || len(metadata) != 1 {
		return
	}
	meta := metadata[0]
	if meta.Name.IsNull() && meta.GenerateName.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("metadata"), "Missing GatewayClass name", "Set metadata.name or metadata.generate_name.")
	}
}

// IdentitySchema implements [resource.ResourceWithIdentity].
// common.IdentitySchema declares version 1 to match identity stored by the
// SDKv2 resources. GatewayClass has no SDKv2 predecessor, so its identity
// starts at version 0 and needs no UpgradeIdentity until the identity changes.
func (r *GatewayClassResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.IdentitySchema()
	resp.IdentitySchema.Version = 0
}

// ImportState implements [resource.ResourceWithImportState]. The import ID is
// the bare object name because GatewayClass is cluster-scoped, and an identity
// must name this API version and kind, so one written for another resource
// type is rejected. Both are validated before any API call; the Read that
// follows fills in metadata, spec and status.
func (r *GatewayClassResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "" {
		if errors := kvalidation.IsDNS1123Subdomain(req.ID); len(errors) != 0 {
			resp.Diagnostics.AddError("Invalid GatewayClass import ID", "Import using the cluster-scoped object name, without a namespace.")
			return
		}
	} else if req.Identity != nil {
		var imported common.ResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &imported)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if imported.APIVersion.ValueString() != apiVersion || imported.Kind.ValueString() != "GatewayClass" || len(kvalidation.IsDNS1123Subdomain(imported.Name.ValueString())) != 0 {
			resp.Diagnostics.AddError("Invalid GatewayClass identity", "Expected api_version gateway.networking.k8s.io/v1, kind GatewayClass, and a cluster-scoped object name.")
			return
		}
	}
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("name"), req, resp)
}
