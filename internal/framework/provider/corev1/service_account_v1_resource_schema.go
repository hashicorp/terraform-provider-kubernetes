// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

func (r *ServiceAccountV1) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = serviceAccountSchema(ctx, false)
}

func serviceAccountSchema(ctx context.Context, defaultAccount bool) schema.Schema {
	metadata := common.NamespacedMetadataSchema("service account", !defaultAccount)
	if defaultAccount {
		name := metadata.NestedObject.Attributes["name"].(schema.StringAttribute)
		name.Default = stringdefault.StaticString("default")
		name.Validators = []validator.String{stringvalidator.OneOf("default")}
		metadata.NestedObject.Attributes["name"] = name
	} else {
		generateName := metadata.NestedObject.Attributes["generate_name"].(schema.StringAttribute)
		// SDKv2 exposes an omitted prefix as "", including in downstream outputs.
		generateName.Computed = true
		generateName.PlanModifiers = append([]planmodifier.String{serviceAccountGenerateNamePlanModifier{}}, generateName.PlanModifiers...)
		metadata.NestedObject.Attributes["generate_name"] = generateName
	}
	for _, key := range []string{"annotations", "labels"} {
		field := metadata.NestedObject.Attributes[key].(schema.MapAttribute)
		// SDKv2 imports may store {} while omitted configuration is null. Computed
		// permits retaining that empty representation; removing managed keys still
		// plans null rather than inheriting their prior values.
		field.Computed = true
		field.PlanModifiers = []planmodifier.Map{serviceAccountMetadataMapPlanModifier{}}
		metadata.NestedObject.Attributes[key] = field
	}
	return schema.Schema{
		Description: "A service account provides an identity for processes that run in a Pod. More info: https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"automount_service_account_token": schema.BoolAttribute{
				Description: "Enable automatic mounting of the service account token",
				Optional:    true, Computed: true, Default: booldefault.StaticBool(true),
			},
			"default_secret_name": schema.StringAttribute{
				Computed:           true,
				DeprecationMessage: "Starting from version 1.24.0 Kubernetes does not automatically generate a token for service accounts, in this case, `default_secret_name` will be empty",
				PlanModifiers:      []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"metadata":          metadata,
			"secret":            serviceAccountReferenceBlock("A list of secrets allowed to be used by pods running using this Service Account. More info: https://kubernetes.io/docs/concepts/configuration/secret"),
			"image_pull_secret": serviceAccountReferenceBlock("A list of references to secrets in the same namespace to use for pulling any images in pods that reference this Service Account. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod"),
			"timeouts":          timeouts.Block(ctx, timeouts.Opts{Create: true}),
		},
	}
}

func serviceAccountReferenceBlock(description string) schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: description,
		NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
				// SDKv2 records the optional name as "" even for an empty block.
				Optional: true, Computed: true, Default: stringdefault.StaticString(""),
			},
		}},
	}
}

type serviceAccountGenerateNamePlanModifier struct{}

func (serviceAccountGenerateNamePlanModifier) Description(context.Context) string {
	return "Preserves an unset name prefix while requiring replacement when a configured prefix is removed."
}

func (m serviceAccountGenerateNamePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (serviceAccountGenerateNamePlanModifier) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	resp.PlanValue = types.StringValue("")
	if !req.State.Raw.IsNull() && req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
	}
}

type serviceAccountMetadataMapPlanModifier struct{}

func (serviceAccountMetadataMapPlanModifier) Description(context.Context) string {
	return "Preserves empty SDKv2 metadata maps without retaining removed managed keys."
}

func (m serviceAccountMetadataMapPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (serviceAccountMetadataMapPlanModifier) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.ConfigValue.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	resp.PlanValue = types.MapNull(types.StringType)
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
	}
}
