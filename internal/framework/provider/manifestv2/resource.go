// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

// Package manifestv2 implements kubernetes_manifest_v2, which manages one object of
// any kind with server-side apply and without the kind's schema.
package manifestv2

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sjson "k8s.io/apimachinery/pkg/util/json"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

var (
	_ resource.Resource                    = (*ManifestV2)(nil)
	_ resource.ResourceWithConfigure       = (*ManifestV2)(nil)
	_ resource.ResourceWithValidateConfig  = (*ManifestV2)(nil)
	_ resource.ResourceWithModifyPlan      = (*ManifestV2)(nil)
	_ resource.ResourceWithImportState     = (*ManifestV2)(nil)
	_ resource.ResourceWithIdentity        = (*ManifestV2)(nil)
	_ resource.ResourceWithUpgradeIdentity = (*ManifestV2)(nil)
	_ resource.ResourceWithMoveState       = (*ManifestV2)(nil)
)

const defaultFieldManager = "Terraform"

type ManifestV2 struct {
	SDKv2Meta func() any
}

func NewManifestV2() resource.Resource {
	return &ManifestV2{}
}

type model struct {
	Manifest            types.Dynamic  `tfsdk:"manifest"`
	YAMLBody            types.String   `tfsdk:"yaml_body"`
	ShowDriftValues     types.Bool     `tfsdk:"show_drift_values"`
	DeletionPropagation types.String   `tfsdk:"deletion_propagation"`
	Drift               types.List     `tfsdk:"drift"`
	Object              types.Dynamic  `tfsdk:"object"`
	ComputedFields      types.List     `tfsdk:"computed_fields"`
	FieldManager        types.Object   `tfsdk:"field_manager"`
	Wait                types.Object   `tfsdk:"wait"`
	APIVersion          types.String   `tfsdk:"api_version"`
	Kind                types.String   `tfsdk:"kind"`
	Namespace           types.String   `tfsdk:"namespace"`
	Name                types.String   `tfsdk:"name"`
	UID                 types.String   `tfsdk:"uid"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

type fieldManagerModel struct {
	Name           types.String `tfsdk:"name"`
	ForceConflicts types.Bool   `tfsdk:"force_conflicts"`
}

// manager returns the field manager name and whether it takes over fields
// other managers own.
func (m model) manager() (string, bool) {
	name, _ := m.FieldManager.Attributes()["name"].(types.String)
	force, _ := m.FieldManager.Attributes()["force_conflicts"].(types.Bool)
	if name.ValueString() == "" {
		return defaultFieldManager, force.ValueBool()
	}
	return name.ValueString(), force.ValueBool()
}

// computedFields returns the manifest paths that are not compared with the
// live object.
func (m model) computedFields() []string {
	var paths []string
	for _, element := range m.ComputedFields.Elements() {
		if text, ok := element.(types.String); ok && text.ValueString() != "" {
			paths = append(paths, text.ValueString())
		}
	}
	return paths
}

func (m model) identity() identity {
	return identity{
		APIVersion: m.APIVersion.ValueString(), Kind: m.Kind.ValueString(),
		Namespace: m.Namespace.ValueString(), Name: m.Name.ValueString(),
	}
}

func (m *model) setIdentity(id identity) {
	m.APIVersion, m.Kind = types.StringValue(id.APIVersion), types.StringValue(id.Kind)
	m.Namespace, m.Name = types.StringValue(id.Namespace), types.StringValue(id.Name)
}

func (m model) resourceIdentity() common.NamespacedResourceIdentity {
	identity := common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{APIVersion: m.APIVersion, Kind: m.Kind, Name: m.Name},
		Namespace:        m.Namespace,
	}
	if m.Namespace.ValueString() == "" {
		identity.Namespace = types.StringNull()
	}
	return identity
}

func (r *ManifestV2) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_manifest_v2"
	// The identity holds the API version, which an update may change.
	resp.ResourceBehavior.MutableIdentity = true
}

func (r *ManifestV2) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.SDKv2Meta = req.ProviderData.(func() any)
	}
}

func (r *ManifestV2) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = common.NamespacedIdentitySchema()
}

// UpgradeIdentity accepts state without an identity, which Terraform before
// 1.12 writes. Read fills it in.
func (r *ManifestV2) UpgradeIdentity(context.Context) map[int64]resource.IdentityUpgrader {
	return common.UpgradeNamespacedIdentity("", "")
}

func (r *ManifestV2) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computed := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: description,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one Kubernetes object of any kind, built-in or custom, from its manifest. " +
			"It is the successor of `kubernetes_manifest`, and a `moved` block moves existing resources to it. " +
			"The manifest is sent with [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/), as `kubectl apply --server-side` does. " +
			"The provider does not need the kind's schema, so planning a new object makes no request to the cluster: " +
			"a CustomResourceDefinition and its custom resources, or a cluster and the objects in it, can be created in one apply.",
		Attributes: map[string]schema.Attribute{
			"yaml_body": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "The object to apply, as YAML or JSON text holding one object. Set either `yaml_body` or `manifest`. " +
					"With `yaml_body`, a plan lists the fields another client changed in `drift` and does not show their values.",
			},
			"show_drift_values": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "With `yaml_body`: also show the values another client set, in the comment that names the changed fields. " +
					"The values appear in plan output, so leave this off for objects that hold secrets.",
			},
			"deletion_propagation": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "How deleting the object treats the objects it owns, such as the pods of a Job: " +
					"`Background` (default) deletes them after the object, `Foreground` before it, and the destroy waits for them, " +
					"`Orphan` leaves them. See [cascading deletion](https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/).",
				Validators: []validator.String{stringvalidator.OneOf(string(metav1.DeletePropagationBackground), string(metav1.DeletePropagationForeground), string(metav1.DeletePropagationOrphan))},
			},
			"drift": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				MarkdownDescription: "With `yaml_body`: the fields of the manifest that another client changed or removed since the last apply. " +
					"A plan that restores them shows this list becoming empty. Always empty with `manifest`, where the plan shows the changed values instead.",
			},
			"manifest": schema.DynamicAttribute{
				Optional: true,
				MarkdownDescription: "The object to apply: an object value with `apiVersion`, `kind` and `metadata.name`. Set either `manifest` or `yaml_body`. " +
					"Write it in HCL or decode YAML with `provider::kubernetes::manifest_decode`. " +
					"Terraform manages the fields set here and leaves all other fields of the object to the cluster. " +
					"`status` and the metadata fields the API server sets are ignored. " +
					"Changing the API group, `kind`, `metadata.name` or `metadata.namespace` replaces the object.",
			},
			"object": schema.DynamicAttribute{
				Computed: true,
				MarkdownDescription: "The object as the cluster returns it, including the fields the cluster sets and `status`. " +
					"It is read when the object is created and on every refresh. An update keeps the value of the last refresh, so references to it stay known in a plan; " +
					"what the update changed in the cluster shows after the next refresh. For a value the manifest sets, refer to `manifest`.",
			},
			"computed_fields": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				MarkdownDescription: "Fields of `manifest` that another controller may change after Terraform sets them, for example `spec.replicas` under an autoscaler. " +
					"Such a change is not reported, and Terraform stops sending a field once another controller owns it. " +
					"Values the API server defaults or rewrites need no entry here.",
			},
			"api_version": computed("The API version of the object."),
			"kind":        computed("The kind of the object."),
			"namespace":   computed("The namespace of the object. Empty for a cluster-scoped object, and `default` when the manifest of a namespaced object names none."),
			"name":        computed("The name of the object."),
			"uid":         computed("The UID the cluster assigned to the object."),
		},
		Blocks: map[string]schema.Block{
			"field_manager": schema.SingleNestedBlock{
				MarkdownDescription: "How Kubernetes records the fields this resource owns.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Optional: true,
						MarkdownDescription: "The name of the field manager. Defaults to `" + defaultFieldManager + "`. " +
							"A resource that applies single fields to the same object, such as `kubernetes_labels`, must use another name.",
						Validators: []validator.String{stringvalidator.LengthBetween(1, 128)},
					},
					"force_conflicts": schema.BoolAttribute{
						Optional: true,
						MarkdownDescription: "Take over fields that another field manager owns with a different value. " +
							"By default such a conflict fails the apply.",
					},
				},
			},
			"wait": schema.SingleNestedBlock{
				MarkdownDescription: "Conditions to wait for after the object is applied. All of them must hold before the apply succeeds.",
				Attributes: map[string]schema.Attribute{
					"rollout": schema.BoolAttribute{
						Optional:            true,
						MarkdownDescription: "Wait until the rollout of a Deployment, StatefulSet or DaemonSet completes, as `kubectl rollout status` does.",
					},
					"ready": schema.BoolAttribute{
						Optional: true,
						MarkdownDescription: "Wait until the object is ready by the rules of its kind: a rollout that completed, a Job that completed, a bound claim, " +
							"or, for any other kind, a status for the current generation in which a `Ready` condition, if there is one, is `True`. " +
							"A status that does not record the generation, or no status at all, counts after 10 seconds. A failed object fails the apply.",
					},
					"fields": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						MarkdownDescription: "Fields to wait for, as a map of field path to a regular expression the value must match, for example `{ \"status.phase\" = \"Running\" }`. " +
							"Use `*` to wait for any value.",
					},
				},
				Blocks: map[string]schema.Block{
					"condition": schema.ListNestedBlock{
						MarkdownDescription: "An entry of `status.conditions` to wait for.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"type":   schema.StringAttribute{Required: true, MarkdownDescription: "The type of the condition."},
								"status": schema.StringAttribute{Optional: true, MarkdownDescription: "The status of the condition. Defaults to `True`."},
							},
						},
					},
				},
			},
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Update: true, Delete: true}),
		},
	}
}

// declared returns the manifest the configuration gives, from whichever
// argument holds it, and the identity it declares. known is false while a
// part of the identity is not known yet; the manifest is nil while any part
// of it is not.
func (m model) declared() (map[string]any, identity, bool, error) {
	if m.YAMLBody.IsNull() {
		id, known, err := manifestIdentity(m.Manifest)
		if err != nil || !known {
			return nil, id, known, err
		}
		value, err := toJSON(m.Manifest)
		if err != nil {
			// Something outside the identity is not known yet.
			return nil, id, true, nil
		}
		return value.(map[string]any), id, true, nil
	}
	if m.YAMLBody.IsUnknown() {
		return nil, identity{}, false, nil
	}
	manifest, err := decodeYAML(m.YAMLBody.ValueString())
	if err != nil {
		return nil, identity{}, false, err
	}
	if metadata, _ := manifest["metadata"].(map[string]any); metadata["generateName"] != nil {
		return nil, identity{}, false, fmt.Errorf("metadata.generateName is not supported: server-side apply needs metadata.name")
	}
	var id identity
	for _, part := range []struct {
		target   *string
		names    []string
		required bool
	}{
		{&id.APIVersion, []string{"apiVersion"}, true}, {&id.Kind, []string{"kind"}, true},
		{&id.Name, []string{"metadata", "name"}, true}, {&id.Namespace, []string{"metadata", "namespace"}, false},
	} {
		value, found, err := unstructured.NestedFieldNoCopy(manifest, part.names...)
		text, isString := value.(string)
		switch {
		case err != nil || found && !isString:
			return nil, id, false, fmt.Errorf("%s in yaml_body must be a string", strings.Join(part.names, "."))
		case text == "" && part.required:
			return nil, id, false, fmt.Errorf("yaml_body has no %s", strings.Join(part.names, "."))
		}
		*part.target = text
	}
	return manifest, id, true, nil
}

// decodeYAML reads the one object a YAML or JSON text holds. A null value
// means "not set", as in a manifest written in HCL.
func decodeYAML(text string) (map[string]any, error) {
	var manifest map[string]any
	reader := k8syaml.NewYAMLReader(bufio.NewReader(strings.NewReader(text)))
	for {
		document, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// Strict: a duplicate key is an error, not the last one winning.
		converted, err := yaml.YAMLToJSONStrict(document)
		if err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
		var data any
		if err := k8sjson.Unmarshal(converted, &data); err != nil {
			return nil, err
		}
		object, isObject := data.(map[string]any)
		switch {
		case data == nil:
			continue
		case !isObject:
			return nil, fmt.Errorf("yaml_body must hold a Kubernetes object")
		case manifest != nil:
			return nil, fmt.Errorf("yaml_body holds more than one object. Use provider::kubernetes::manifest_decode_map with for_each for several")
		}
		manifest = object
	}
	if manifest == nil {
		return nil, fmt.Errorf("yaml_body holds no object")
	}
	return withoutNulls(manifest).(map[string]any), nil
}

func withoutNulls(value any) any {
	switch value := value.(type) {
	case map[string]any:
		for name, member := range value {
			if member == nil {
				delete(value, name)
			} else {
				value[name] = withoutNulls(member)
			}
		}
	case []any:
		for i := range value {
			value[i] = withoutNulls(value[i])
		}
	}
	return value
}

// manifestIdentity reads the identity a manifest declares. known is false
// while a part of it is not known yet.
func manifestIdentity(manifest types.Dynamic) (id identity, known bool, err error) {
	if manifest.IsNull() {
		return id, false, nil
	}
	if manifest.IsUnknown() || manifest.IsUnderlyingValueUnknown() {
		return id, false, nil
	}
	if _, isString := manifest.UnderlyingValue().(types.String); isString {
		return id, false, fmt.Errorf("the manifest is a string. Decode YAML with provider::kubernetes::manifest_decode(...)")
	}
	if member(manifest, "metadata") == nil && member(manifest, "kind") == nil {
		return id, false, fmt.Errorf("the manifest must be an object with apiVersion, kind and metadata")
	}
	if generated := member(manifest, "metadata", "generateName"); generated != nil && !generated.IsNull() {
		return id, false, fmt.Errorf("metadata.generateName is not supported: server-side apply needs metadata.name")
	}
	known = true
	read := func(target *string, required bool, names ...string) {
		// A part of the path that is not known yet, such as all of metadata.
		for i := 1; i < len(names); i++ {
			if parent := member(manifest, names[:i]...); parent != nil && parent.IsUnknown() {
				known = false
				return
			}
		}
		value := member(manifest, names...)
		text, isString := value.(types.String)
		switch {
		case value == nil || value.IsNull():
			if required && err == nil {
				err = fmt.Errorf("the manifest has no %s", strings.Join(names, "."))
			}
		case value.IsUnknown():
			known = false
		case !isString:
			if err == nil {
				err = fmt.Errorf("%s in the manifest must be a string", names[len(names)-1])
			}
		default:
			*target = text.ValueString()
		}
	}
	read(&id.APIVersion, true, "apiVersion")
	read(&id.Kind, true, "kind")
	read(&id.Name, true, "metadata", "name")
	read(&id.Namespace, false, "metadata", "namespace")
	return id, known && err == nil, err
}

func (r *ManifestV2) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config model
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if config.Manifest.IsNull() == config.YAMLBody.IsNull() {
		resp.Diagnostics.AddError("Invalid manifest", "Set exactly one of manifest and yaml_body.")
		return
	}
	argument := path.Root("manifest")
	if config.Manifest.IsNull() {
		argument = path.Root("yaml_body")
	}
	manifest, declared, _, err := config.declared()
	if err != nil {
		resp.Diagnostics.AddAttributeError(argument, "Invalid manifest", err.Error())
	}
	for _, field := range config.computedFields() {
		names, err := fieldNames(field)
		switch {
		case err != nil:
			resp.Diagnostics.AddAttributeError(path.Root("computed_fields"), "Invalid field path", err.Error()+".")
		case manifest != nil && !hasField(manifest, names):
			resp.Diagnostics.AddAttributeWarning(path.Root("computed_fields"), "Field not in the manifest",
				fmt.Sprintf("%s is not set in the manifest, so listing it in computed_fields has no effect.", field))
		}
	}
	if _, err := expectations(ctx, config.Wait, declared); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("wait"), "Invalid wait condition", err.Error())
	}
}

// ImportState accepts the import ID of kubernetes_manifest,
// "apiVersion=apps/v1,kind=Deployment,namespace=default,name=example", or a
// resource identity. Read fills in the manifest from the live object.
func (r *ManifestV2) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var id identity
	if req.ID == "" && (req.Identity == nil || req.Identity.Raw.IsNull()) {
		resp.Diagnostics.AddError("Invalid import ID", `The import ID is empty. Expected "apiVersion=<version>,kind=<kind>,[namespace=<namespace>,]name=<name>".`)
		return
	}
	if req.ID != "" {
		gvk, name, namespace, err := util.ParseResourceID(req.ID)
		if err != nil {
			resp.Diagnostics.AddError("Invalid import ID", err.Error()+`. Expected "apiVersion=<version>,kind=<kind>,[namespace=<namespace>,]name=<name>".`)
			return
		}
		id = identity{APIVersion: gvk.GroupVersion().String(), Kind: gvk.Kind, Namespace: namespace, Name: name}
	} else {
		var imported common.NamespacedResourceIdentity
		resp.Diagnostics.Append(req.Identity.Get(ctx, &imported)...)
		if resp.Diagnostics.HasError() {
			return
		}
		id = identity{
			APIVersion: imported.APIVersion.ValueString(), Kind: imported.Kind.ValueString(),
			Namespace: imported.Namespace.ValueString(), Name: imported.Name.ValueString(),
		}
	}
	for attribute, value := range map[string]attr.Value{
		"api_version": types.StringValue(id.APIVersion), "kind": types.StringValue(id.Kind),
		"namespace": types.StringValue(id.Namespace), "name": types.StringValue(id.Name),
	} {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(attribute), value)...)
	}
}
