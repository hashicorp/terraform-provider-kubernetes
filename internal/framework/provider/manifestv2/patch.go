// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package manifestv2

import (
	"context"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/util"

	"sigs.k8s.io/yaml"
)

var (
	_ resource.Resource                    = (*ManifestPatch)(nil)
	_ resource.ResourceWithConfigure       = (*ManifestPatch)(nil)
	_ resource.ResourceWithValidateConfig  = (*ManifestPatch)(nil)
	_ resource.ResourceWithModifyPlan      = (*ManifestPatch)(nil)
	_ resource.ResourceWithImportState     = (*ManifestPatch)(nil)
	_ resource.ResourceWithIdentity        = (*ManifestPatch)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ManifestPatch)(nil)
)

const (
	defaultPatchManager = "terraform-patch"
	relinquishFields    = "relinquish"
	removeFields        = "remove_fields"
)

// ManifestPatch implements kubernetes_manifest_patch: fields of an object
// that something else creates and manages, applied with the server-side
// apply of kubernetes_manifest_v2. It never creates or deletes the object.
type ManifestPatch struct {
	engine ManifestV2
}

func NewManifestPatch() resource.Resource {
	return &ManifestPatch{}
}

type patchModel struct {
	Manifest        types.Dynamic  `tfsdk:"manifest"`
	YAMLBody        types.String   `tfsdk:"yaml_body"`
	ShowDriftValues types.Bool     `tfsdk:"show_drift_values"`
	DestroyBehavior types.String   `tfsdk:"destroy_behavior"`
	Drift           types.List     `tfsdk:"drift"`
	ComputedFields  types.List     `tfsdk:"computed_fields"`
	FieldManager    types.Object   `tfsdk:"field_manager"`
	Wait            types.Object   `tfsdk:"wait"`
	APIVersion      types.String   `tfsdk:"api_version"`
	Kind            types.String   `tfsdk:"kind"`
	Namespace       types.String   `tfsdk:"namespace"`
	Name            types.String   `tfsdk:"name"`
	UID             types.String   `tfsdk:"uid"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}

// manifest returns the arguments a patch shares with kubernetes_manifest_v2,
// in the form its functions take.
func (m patchModel) manifest() model {
	return model{
		Manifest: m.Manifest, YAMLBody: m.YAMLBody, ShowDriftValues: m.ShowDriftValues,
		DeletionPropagation: types.StringNull(), Drift: m.Drift, Object: types.DynamicNull(),
		ComputedFields: m.ComputedFields, FieldManager: m.FieldManager, Wait: m.Wait,
		APIVersion: m.APIVersion, Kind: m.Kind, Namespace: m.Namespace, Name: m.Name, UID: m.UID,
		Timeouts: m.Timeouts,
	}
}

// update takes over the values those functions set.
func (m *patchModel) update(from model) {
	m.Manifest, m.YAMLBody, m.Drift = from.Manifest, from.YAMLBody, from.Drift
	m.APIVersion, m.Kind, m.Namespace, m.Name, m.UID = from.APIVersion, from.Kind, from.Namespace, from.Name, from.UID
}

// manager returns the field manager name and whether it takes over fields
// other managers own.
func (m patchModel) manager() (string, bool) {
	name, _ := m.FieldManager.Attributes()["name"].(types.String)
	force, _ := m.FieldManager.Attributes()["force_conflicts"].(types.Bool)
	if name.ValueString() == "" {
		return defaultPatchManager, force.ValueBool()
	}
	return name.ValueString(), force.ValueBool()
}

// patchIdentity names the object and the field manager: each patch of one
// object applies its fields under its own name.
type patchIdentity struct {
	common.NamespacedResourceIdentity
	FieldManager types.String `tfsdk:"field_manager"`
}

func (m patchModel) resourceIdentity() patchIdentity {
	manager, _ := m.manager()
	return patchIdentity{NamespacedResourceIdentity: m.manifest().resourceIdentity(), FieldManager: types.StringValue(manager)}
}

func (r *ManifestPatch) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_manifest_patch"
	// The identity holds the API version and the field manager, which an
	// update may change.
	resp.ResourceBehavior.MutableIdentity = true
}

func (r *ManifestPatch) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.engine.Configure(ctx, req, resp)
}

func (r *ManifestPatch) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
	resp.IdentitySchema.Attributes["field_manager"] = identityschema.StringAttribute{OptionalForImport: true}
}

// UpgradeIdentity accepts state without an identity, which Terraform before
// 1.12 writes. Read fills it in.
func (r *ManifestPatch) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return map[int64]resource.IdentityUpgrader{0: {
		IdentityUpgrader: func(ctx context.Context, _ resource.UpgradeIdentityRequest, resp *resource.UpgradeIdentityResponse) {
			null := types.StringNull()
			resp.Diagnostics.Append(resp.Identity.Set(ctx, patchIdentity{
				NamespacedResourceIdentity: common.NamespacedResourceIdentity{
					ResourceIdentity: common.ResourceIdentity{Name: null, Kind: null, APIVersion: null}, Namespace: null,
				},
				FieldManager: null,
			})...)
		},
	}}
}

func (r *ManifestPatch) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// The arguments both resources take mean the same and are described once.
	var shared resource.SchemaResponse
	r.engine.Schema(ctx, req, &shared)
	attributes := map[string]schema.Attribute{
		"manifest": schema.DynamicAttribute{
			Optional: true,
			MarkdownDescription: "The object to patch and the fields to apply to it: an object value with `apiVersion`, `kind` and `metadata.name`, " +
				"and `metadata.namespace` for an object outside `default`, followed by only the fields this resource sets. Set either `manifest` or `yaml_body`. " +
				"Write it in HCL or decode YAML with `provider::kubernetes::manifest_decode`. " +
				"Changing the API group, `kind`, `metadata.name` or `metadata.namespace` moves the patch to another object.",
		},
		"yaml_body": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "The object to patch and the fields to apply to it, as YAML or JSON text holding one object. Set either `yaml_body` or `manifest`. " +
				"With `yaml_body`, a plan lists the fields another client changed in `drift` and does not show their values.",
		},
		"destroy_behavior": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "What destroying this resource does with the fields it applied. The object itself is never deleted. " +
				"`" + relinquishFields + "` (default) leaves the fields at their values and gives up their ownership, so that any client can change them without a conflict. " +
				"`" + removeFields + "` removes the fields that no other field manager applies, as removing them from the manifest does, " +
				"also those taken over with `force_conflicts`: Kubernetes keeps no earlier value to restore.",
			Validators: []validator.String{stringvalidator.OneOf(relinquishFields, removeFields)},
		},
	}
	for _, name := range []string{"show_drift_values", "drift", "computed_fields", "api_version", "kind", "namespace", "name", "uid"} {
		attributes[name] = shared.Schema.Attributes[name]
	}
	fieldManager := shared.Schema.Blocks["field_manager"].(schema.SingleNestedBlock)
	fieldManager.Attributes = maps.Clone(fieldManager.Attributes)
	name := fieldManager.Attributes["name"].(schema.StringAttribute)
	name.MarkdownDescription = "The name of the field manager. Defaults to `" + defaultPatchManager + "`. " +
		"Every patch of one object needs a name of its own, and so does a `kubernetes_manifest_v2` or `kubernetes_labels` resource of the same object."
	fieldManager.Attributes["name"] = name
	resp.Schema = schema.Schema{
		MarkdownDescription: "Applies fields of a Kubernetes object that something else creates and manages, such as a cluster add-on or an operator, " +
			"and keeps them set. The fields are sent with [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/) " +
			"under their own field manager, as `kubectl apply --server-side` sends a partial manifest, so the object keeps its other fields and their owners. " +
			"The object must exist: this resource never creates or deletes it. It takes the arguments of `kubernetes_manifest_v2`, which manages whole objects.",
		Attributes: attributes,
		Blocks: map[string]schema.Block{
			"field_manager": fieldManager,
			"wait":          shared.Schema.Blocks["wait"],
			"timeouts":      shared.Schema.Blocks["timeouts"],
		},
	}
}

func (r *ManifestPatch) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config patchModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(config.manifest().validate(ctx)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// null means "not set" in a manifest, but in a patch it is easily taken
	// to remove a field that another client set.
	argument, nulls := path.Root("manifest"), nullValues(config.Manifest, location{})
	if !config.YAMLBody.IsNull() {
		argument, nulls = path.Root("yaml_body"), yamlNulls(config.YAMLBody)
	}
	if len(nulls) > 0 {
		sort.Strings(nulls)
		resp.Diagnostics.AddAttributeWarning(argument, "Null value not applied", fmt.Sprintf("%s: a null value is left out of the apply, "+
			"as in kubernetes_manifest_v2. A field that only this resource applied before is removed. A field that another field manager set "+
			"stays as it is: server-side apply cannot remove it.", strings.Join(nulls, ", ")))
	}
}

// nullValues returns the paths of the null members of a manifest value.
func nullValues(value attr.Value, at location) []string {
	if value == nil || value.IsNull() || value.IsUnknown() {
		return nil
	}
	var paths []string
	var members map[string]attr.Value
	switch value := value.(type) {
	case basetypes.DynamicValue:
		return nullValues(value.UnderlyingValue(), at)
	case basetypes.ObjectValue:
		members = value.Attributes()
	case basetypes.MapValue:
		members = value.Elements()
	case basetypes.TupleValue:
		for i, element := range value.Elements() {
			paths = append(paths, nullValues(element, at.element(strconv.Itoa(i)))...)
		}
	case basetypes.ListValue:
		for i, element := range value.Elements() {
			paths = append(paths, nullValues(element, at.element(strconv.Itoa(i)))...)
		}
	}
	for name, member := range members {
		if member.IsNull() {
			paths = append(paths, at.field(name).path)
		} else {
			paths = append(paths, nullValues(member, at.field(name))...)
		}
	}
	return paths
}

// yamlNulls returns the paths of the null values in a manifest given as text.
func yamlNulls(body types.String) []string {
	var data any
	if body.IsUnknown() || yaml.Unmarshal([]byte(body.ValueString()), &data) != nil {
		return nil
	}
	var walk func(value any, at location) []string
	walk = func(value any, at location) []string {
		var paths []string
		switch value := value.(type) {
		case map[string]any:
			for name, member := range value {
				if member == nil {
					paths = append(paths, at.field(name).path)
				} else {
					paths = append(paths, walk(member, at.field(name))...)
				}
			}
		case []any:
			for i, element := range value {
				paths = append(paths, walk(element, at.element(strconv.Itoa(i)))...)
			}
		}
		return paths
	}
	return walk(data, location{})
}

// ModifyPlan works from the configuration and state alone, as for
// kubernetes_manifest_v2: a plan makes no request to the cluster.
func (r *ManifestPatch) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan patchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state *model
	drift := types.ListValueMust(types.StringType, nil)
	if !req.State.Raw.IsNull() {
		var prior patchModel
		resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
		if resp.Diagnostics.HasError() {
			return
		}
		state = new(model)
		*state = prior.manifest()
		// After an update whose wait failed, the next apply applies the
		// fields again and waits again.
		pending, _ := req.Private.GetKey(ctx, pendingKey)
		switch {
		case pending != nil || !req.Config.Raw.IsFullyKnown():
			drift = types.ListUnknown(types.StringType)
		case sameArguments(plan.manifest(), *state):
			drift = state.Drift
		}
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("drift"), drift)...)
	planIdentity(ctx, plan.manifest(), state, resp)
}

// ImportState accepts the import ID of kubernetes_manifest_v2, with the field
// manager as an optional part, as in
// "apiVersion=apps/v1,kind=DaemonSet,namespace=kube-system,name=node-agent,fieldManager=terraform-patch",
// or a resource identity. Read fills in the fields the field manager applied.
func (r *ManifestPatch) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = `Expected "apiVersion=<version>,kind=<kind>,[namespace=<namespace>,]name=<name>[,fieldManager=<field manager>]".`
	var id identity
	var manager string
	switch {
	case req.ID != "":
		var parts []string
		for _, part := range strings.Split(req.ID, ",") {
			if name, found := strings.CutPrefix(part, "fieldManager="); found {
				manager = name
			} else {
				parts = append(parts, part)
			}
		}
		gvk, name, namespace, err := util.ParseResourceID(strings.Join(parts, ","))
		if err != nil {
			resp.Diagnostics.AddError("Invalid import ID", err.Error()+". "+expected)
			return
		}
		id = identity{APIVersion: gvk.GroupVersion().String(), Kind: gvk.Kind, Namespace: namespace, Name: name}
	case req.Identity != nil && !req.Identity.Raw.IsNull():
		var imported patchIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &imported)...)
		if resp.Diagnostics.HasError() {
			return
		}
		id = identity{
			APIVersion: imported.APIVersion.ValueString(), Kind: imported.Kind.ValueString(),
			Namespace: imported.Namespace.ValueString(), Name: imported.Name.ValueString(),
		}
		manager = imported.FieldManager.ValueString()
	default:
		resp.Diagnostics.AddError("Invalid import ID", "The import ID is empty. "+expected)
		return
	}
	for attribute, value := range map[string]attr.Value{
		"api_version": types.StringValue(id.APIVersion), "kind": types.StringValue(id.Kind),
		"namespace": types.StringValue(id.Namespace), "name": types.StringValue(id.Name),
	} {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attribute), value)...)
	}
	if manager != "" && manager != defaultPatchManager {
		block := types.ObjectValueMust(map[string]attr.Type{"name": types.StringType, "force_conflicts": types.BoolType},
			map[string]attr.Value{"name": types.StringValue(manager), "force_conflicts": types.BoolNull()})
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("field_manager"), block)...)
	}
}
