// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8Types "k8s.io/apimachinery/pkg/types"
)

const (
	namespaceAPIVersion = "v1"
	namespaceKind       = "Namespace"

	// defaultDeleteTimeout matches SDKv2's `Delete: schema.DefaultTimeout(5 * time.Minute)`.
	// It is only the fallback — this one IS user-settable via the timeouts block.
	defaultDeleteTimeout = 5 * time.Minute

	// defaultCreateTimeout is not user-settable — matching SDKv2, where Create was not a declared timeout
	// either and d.Timeout(schema.TimeoutCreate) fell through to the SDK's 20m default.
	defaultCreateTimeout = 20 * time.Minute
)

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

	ctx, cancel := context.WithTimeout(ctx, defaultCreateTimeout)
	defer cancel()

	clients, _, metaDiags := n.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	metadata, diags := common.ExpandMetadata(ctx, plan.Metadata)
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
	// plan. Filtering them is Read's job.
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

	err = retry.RetryContext(ctx, defaultCreateTimeout, func() *retry.RetryError {
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

func (n *NamespaceV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	clients, metadataFilters, metaDiags := n.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
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
	metadata, diags := common.FlattenMetadata(ctx, namespace.ObjectMeta, state.Metadata,
		metadataFilters.GetIgnoreAnnotations(), metadataFilters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(namespace.Name)
	state.Metadata = metadata

	// server does not store and return this field, hence this field was left as null during import,
	// etting its default schema value in such scenario.
	if state.WaitForDefaultServiceAccount.IsNull() {
		state.WaitForDefaultServiceAccount = types.BoolValue(false)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	resp.Diagnostics.Append(resp.Identity.Set(ctx, NamespaceResourceIdentity{
		APIVersion: types.StringValue(namespaceAPIVersion),
		Kind:       types.StringValue(namespaceKind),
		Name:       types.StringValue(namespace.Name),
	})...)
}

func (n *NamespaceV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
	// metadata.name and generate_name are RequiresReplace, so the only server-side change
	// Update can see is to labels or annotations; anything else is a replacement Terraform
	// performs before reaching here. It can still be called with nothing to send, when the
	// change was to an attribute Kubernetes does not store — see below.
	//
	// [0] is safe on both: the block is validated with SizeAtLeast(1)/SizeAtMost(1),
	// and prior state came from an apply that passed the same validation.
	planMeta, stateMeta := plan.Metadata[0], state.Metadata[0]

	ops := common.MetadataPatchOps("/metadata/", stateMeta, planMeta)

	clients, _, metaDiags := n.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

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
}

// Delete implements [resource.Resource].
func (n *NamespaceV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NamespaceV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	clients, _, metaDiags := n.sdkv2Meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	conn, err := clients.MainClientset()
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
