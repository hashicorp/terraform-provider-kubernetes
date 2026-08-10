// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8Types "k8s.io/apimachinery/pkg/types"
)

var (
	_ resource.Resource                = (*NamespaceV1)(nil)
	_ resource.ResourceWithConfigure   = (*NamespaceV1)(nil)
	_ resource.ResourceWithIdentity    = (*NamespaceV1)(nil)
	_ resource.ResourceWithImportState = (*NamespaceV1)(nil)
)

type NamespaceV1 struct {
	// SDKv2Meta must stay func() any: that is the concrete type stored in
	// ProviderData by internal/framework/provider/provider_configure.go. Go
	// function types are invariant, so asserting to func() kubernetes.KubeClientsets
	// compiles but panics at runtime. Assert the *result* instead — see meta().
	SDKv2Meta func() any
}

// ImportState implements [resource.ResourceWithImportState].
func (n *NamespaceV1) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("name"), req, resp)
}

// namespaceAPIVersion and namespaceKind are hardcoded because client-go's typed
// clients do not populate TypeMeta on responses — the decoder clears apiVersion and
// kind for typed objects, so out.APIVersion and out.Kind are always "". SDKv2 does the
// same at resource_kubernetes_namespace_v1.go:115.
const (
	namespaceAPIVersion = "v1"
	namespaceKind       = "Namespace"
)

// IdentitySchema implements [resource.ResourceWithIdentity].
func (n *NamespaceV1) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		// Must match resourceIdentitySchemaNonNamespaced() in
		// kubernetes/resourceidentity.go. State written by the SDKv2 resource records
		// identity schema version 1; declaring 0 here asks Terraform to downgrade.
		Version: 1,
		Attributes: map[string]identityschema.Attribute{
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"kind": identityschema.StringAttribute{
				RequiredForImport: true,
			},
			"api_version": identityschema.StringAttribute{
				RequiredForImport: true,
			},
		},
	}
}

// Configure implements [resource.ResourceWithConfigure].
func (n *NamespaceV1) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	sdkv2Meta, ok := req.ProviderData.(func() any)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("Expected func() any, got %T. This is a bug in the provider.", req.ProviderData),
		)
		return
	}
	n.SDKv2Meta = sdkv2Meta
}

// meta resolves the SDKv2 provider metadata as API clients. The call is deferred
// until now rather than made in Configure because the SDKv2 provider is configured
// independently by the mux server, and its meta is not populated until that happens.
func (n *NamespaceV1) meta() kubernetes.KubeClientsets {
	return n.SDKv2Meta().(kubernetes.KubeClientsets)
}

// filters resolves the same metadata as the provider-level ignore lists.
func (n *NamespaceV1) filters() kubernetes.MetadataFilters {
	return n.SDKv2Meta().(kubernetes.MetadataFilters)
}

func NewNamespaceV1() resource.Resource {
	return &NamespaceV1{}
}

// Metadata implements [resource.Resource].
func (n *NamespaceV1) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_namespace_v1"
}

// Schema implements [resource.Resource].
func (n *NamespaceV1) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Kubernetes supports multiple virtual clusters backed by the same physical cluster. These virtual clusters are called namespaces. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"wait_for_default_service_account": schema.BoolAttribute{
				Description: "Terraform will wait for the default service account to be created.",
				Optional:    true,
				// Optional+Computed+Default is how the framework spells SDKv2's
				// `Default: false`. All three are required, or a config that omits
				// the field lands as null in state instead of false.
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			// SDKv2 declares metadata as TypeList{Required: true, MaxItems: 1}, which
			// serializes as a one-element array. ListNestedBlock reproduces that shape;
			// SingleNestedBlock would produce a bare object and break existing state.
			"metadata": schema.ListNestedBlock{
				Description: "Standard namespace's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1), // SDKv2 Required: true
					listvalidator.SizeAtMost(1),  // SDKv2 MaxItems: 1
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"annotations": schema.MapAttribute{
							Description: "An unstructured key value map stored with the namespace that may be used to store arbitrary metadata. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/",
							ElementType: types.StringType,
							Optional:    true,
							Validators: []validator.Map{
								annotationKeys(),
							},
						},
						"generate_name": schema.StringAttribute{
							Description: "Prefix, used by the server, to generate a unique name ONLY IF the `name` field has not been provided. This value will also be combined with a unique suffix. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#idempotency",
							Optional:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.ConflictsWith(
									path.MatchRelative().AtParent().AtName("name"),
								),
								dnsLabelPrefix(),
							},
						},
						"generation": schema.Int64Attribute{
							Description: "A sequence number representing a specific generation of the desired state.",
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: "Map of string keys and values that can be used to organize and categorize (scope and select) the namespace. May match selectors of replication controllers and services. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/",
							ElementType: types.StringType,
							Optional:    true,
							Validators: []validator.Map{
								labelKeyValues(),
							},
						},
						"name": schema.StringAttribute{
							Description: "Name of the namespace, must be unique. Cannot be updated. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
							Optional:    true,
							Computed:    true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
							Validators: []validator.String{
								stringvalidator.ConflictsWith(
									path.MatchRelative().AtParent().AtName("generate_name"),
								),
								dnsSubdomainName(),
							},
						},
						"resource_version": schema.StringAttribute{
							Description: "An opaque value that represents the internal version of this namespace that can be used by clients to determine when namespace has changed. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency",
							Computed:    true,
						},
						"uid": schema.StringAttribute{
							Description: "The unique in time and space value for this namespace. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids",
							Computed:    true,
						},
					},
				},
			},
			// SDKv2 declares only a Delete timeout, so only Delete is user-settable.
			// timeouts.BlockAll would add create/update/read and change the schema.
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Delete: true}),
		},
	}
}

// defaultDeleteTimeout matches SDKv2's `Delete: schema.DefaultTimeout(5 * time.Minute)`.
// It is only the fallback — this one IS user-settable via the timeouts block.
// const defaultDeleteTimeout = 5 * time.Minute
const defaultDeleteTimeout = 10 * time.Second

// defaultTimeout bounds Create. The schema declares only a Delete timeout, so
// this is not user-settable — matching SDKv2, where Create was not a declared timeout
// either and d.Timeout(schema.TimeoutCreate) fell through to the SDK's 20m default.
const defaultTimeout = 20 * time.Minute

// Create implements [resource.Resource].
func (n *NamespaceV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NamespaceV1Model

	// Plan, not Config: Config has not had computed values resolved, so
	// wait_for_default_service_account would be null instead of false and
	// metadata.name would be null instead of unknown under generate_name.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	conn, err := n.meta().MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	metadata, diags := expandMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace := v1.Namespace{ObjectMeta: metadata}
	tflog.Info(ctx, "Creating namespace", map[string]any{"name": metadata.Name})

	out, err := conn.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating namespace", err.Error())
		return
	}

	// State is built from the plan, with only server-assigned fields overwritten from
	// the response. Labels and annotations are deliberately left as the plan wrote them:
	// the API server adds keys of its own, and echoing those back would not match the
	// plan. Filtering them is Read's job. See MIGRATION_FINDINGS_namespace_v1.md §3.
	// Indexing [0] without a guard is safe: the metadata block is validated with
	// SizeAtLeast(1)/SizeAtMost(1), and the plan is derived from validated config.
	// Name is Computed, and unknown in the plan whenever generate_name is used, so
	// it always comes from the response.
	plan.ID = types.StringValue(out.Name)
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	// Persist before waiting. The namespace exists now; if the wait below fails we
	// still need Terraform to know about it, or it is orphaned and the next apply
	// fails on a name conflict.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, NamespaceResourceIdentity{
		APIVersion: types.StringValue(namespaceAPIVersion),
		Kind:       types.StringValue(namespaceKind),
		Name:       types.StringValue(out.Name),
	})...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.WaitForDefaultServiceAccount.ValueBool() {
		return
	}

	tflog.Debug(ctx, "Waiting for default service account", map[string]any{"namespace": out.Name})

	// NOTE: helper/retry is an SDKv2 package, which CLAUDE.md otherwise keeps out of
	// internal/framework. Deliberate exception: it is standalone (no schema coupling),
	// already exercised throughout this provider, and the framework ships no
	// equivalent. Revisit when the framework gains a wait helper.
	err = retry.RetryContext(ctx, defaultTimeout, func() *retry.RetryError {
		_, err := conn.CoreV1().ServiceAccounts(out.Name).Get(ctx, "default", metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				tflog.Info(ctx, "Default service account does not exist, will retry",
					map[string]any{"namespace": out.Name})
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}
		tflog.Info(ctx, "Default service account exists", map[string]any{"namespace": out.Name})
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error waiting for default service account",
			fmt.Sprintf("Namespace %q was created, but its default service account did not appear: %s", out.Name, err),
		)
		return
	}
}

// Delete implements [resource.Resource].
func (n *NamespaceV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := n.meta().MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	deleteTimeout, d := state.Timeouts.Delete(ctx, defaultDeleteTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.ID.ValueString()
	err = conn.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		// Already gone is success. Erroring here would leave the practitioner
		// unable to destroy a namespace that was removed out of band.
		if apierrors.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Kubernetes delete error", err.Error())
		return
	}
	stateChangePoller := retry.StateChangeConf{
		Pending: []string{"Terminating"},
		Target:  []string{},
		Timeout: deleteTimeout,
		Refresh: func() (result interface{}, state string, err error) {
			ns, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					return nil, "", nil
				}
				return nil, "Error", err
			}
			return ns, string(ns.Status.Phase), nil
		},
	}
	_, err = stateChangePoller.WaitForStateContext(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes delete error", err.Error())
		return
	}
}

// Read implements [resource.Resource].
func (n *NamespaceV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := n.meta().MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	name := state.ID.ValueString()
	namespace, err := conn.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		// Not-found is not an error: it means the namespace was removed outside
		// Terraform. SDKv2 signalled this via its Exists hook and d.SetId("");
		// the framework equivalent is RemoveResource, which lets the next plan
		// offer to recreate it instead of failing permanently.
		if apierrors.IsNotFound(err) {
			tflog.Info(ctx, "Namespace no longer exists, removing from state",
				map[string]any{"name": name})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to read namespace %q", name), err.Error())
		return
	}

	// Prior state is the filtering reference. On import it is empty, which is the
	// correct baseline: nothing was declared, so nothing is exempt from filtering.
	metadata, diags := flattenMetadata(ctx, namespace.ObjectMeta, state.Metadata,
		n.filters().GetIgnoreAnnotations(), n.filters().GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(namespace.Name)
	state.Metadata = metadata

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, NamespaceResourceIdentity{
		APIVersion: types.StringValue(namespaceAPIVersion),
		Kind:       types.StringValue(namespaceKind),
		Name:       types.StringValue(namespace.Name),
	})...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update implements [resource.Resource].
func (n *NamespaceV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// if name or generate_name then delete existing one and create new one,
	//  but we don't have delete yet
	var plan NamespaceV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := n.meta().MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}
	// Only metadata.name and generate_name are RequiresReplace, so Update can only
	// ever see labels and annotations change. Anything else is handled by Terraform
	// destroying and recreating the resource before Update is reached.
	//
	// [0] is safe on both: the block is validated with SizeAtLeast(1)/SizeAtMost(1),
	// and prior state came from an apply that passed the same validation.
	planMeta, stateMeta := plan.Metadata[0], state.Metadata[0]

	// keys -> added: in plan, not in state; replaced: in both, value changed;
	// removed: in state, not in plan. DiffStringMap emits one operation per key so
	// that keys managed outside Terraform are left untouched.
	ops := kubernetes.PatchOperations{}
	ops = append(ops, kubernetes.DiffStringMap("/metadata/annotations",
		expandMapForPatch(stateMeta.Annotations), expandMapForPatch(planMeta.Annotations))...)
	ops = append(ops, kubernetes.DiffStringMap("/metadata/labels",
		expandMapForPatch(stateMeta.Labels), expandMapForPatch(planMeta.Labels))...)

	data, err := ops.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal update json patch", err.Error())
		return
	}
	out, err := conn.CoreV1().Namespaces().Patch(ctx, plan.ID.ValueString(), k8Types.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Failed to execute jsonPatch update", err.Error())
		return
	}
	// As in Create, state is the plan with only server-assigned fields overwritten.
	// The response is NOT filtered and folded back in: labels and annotations are
	// Optional but not Computed, so their planned values must be returned
	// byte-for-byte or Terraform rejects the apply. Filtering happens only in Read,
	// which has no plan to be consistent with.
	plan.ID = types.StringValue(out.Name)
	plan.Metadata[0].Name = types.StringValue(out.Name)
	plan.Metadata[0].UID = types.StringValue(string(out.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(out.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(out.Generation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, NamespaceResourceIdentity{
		APIVersion: types.StringValue(namespaceAPIVersion),
		Kind:       types.StringValue(namespaceKind),
		Name:       types.StringValue(out.Name),
	})...)
	if resp.Diagnostics.HasError() {
		return
	}
}
