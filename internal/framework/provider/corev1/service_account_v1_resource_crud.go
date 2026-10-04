// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const serviceAccountCreateTimeout = 30 * time.Second

func serviceAccountCreate(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, callback func() any, defaultAccount bool) {
	plan, diags := serviceAccountReadModel(ctx, req.Plan, defaultAccount)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid service account metadata", "Exactly one metadata block is required.")
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, serviceAccountCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, _, diags := serviceAccountClient(callback)
	resp.Diagnostics.Append(diags...)
	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	secretNames, diags := serviceAccountReferenceNames(ctx, plan.Secret)
	resp.Diagnostics.Append(diags...)
	imageNames, diags := serviceAccountReferenceNames(ctx, plan.ImagePullSecret)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	automount := plan.AutomountServiceAccountToken.ValueBool()
	config := corev1.ServiceAccount{
		ObjectMeta: metadata, AutomountServiceAccountToken: &automount,
		Secrets:          serviceAccountSecrets(secretNames, "", nil),
		ImagePullSecrets: serviceAccountImageSecrets(imageNames),
	}
	var out *corev1.ServiceAccount
	var err error
	if defaultAccount {
		if metadata.Name != "default" {
			resp.Diagnostics.AddError("Invalid default service account", "The default service account must be named \"default\".")
			return
		}
		err = retry.RetryContext(ctx, timeout, func() *retry.RetryError {
			out, err = conn.CoreV1().ServiceAccounts(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				return retry.RetryableError(err)
			}
			if err != nil {
				return retry.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for default service account", err.Error())
			return
		}
		// SDKv2 looks for the controller token relative to an empty secrets list
		// during adoption, not relative to the user's desired list.
		secret, err := serviceAccountWaitForDefaultSecret(ctx, conn, metadata.Name, corev1.ServiceAccount{ObjectMeta: metadata}, timeout)
		if err != nil {
			resp.Diagnostics.AddError("Error discovering default service account token", err.Error())
			return
		}
		plan.DefaultSecretName = types.StringValue(secret)
		// Token discovery can wait for a controller write. Read its latest
		// version before constructing the guarded adoption patch.
		out, err = conn.CoreV1().ServiceAccounts(metadata.Namespace).Get(ctx, metadata.Name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error reading default service account before adoption", err.Error())
			return
		}
		ops, d := serviceAccountAdoptionPatch(ctx, plan, out)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		payload, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Error encoding service account patch", err.Error())
			return
		}
		out, err = conn.CoreV1().ServiceAccounts(metadata.Namespace).Patch(ctx, metadata.Name, k8stypes.JSONPatchType, payload, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error adopting default service account", err.Error())
			return
		}
	} else {
		out, err = conn.CoreV1().ServiceAccounts(metadata.Namespace).Create(ctx, &config, metav1.CreateOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error creating service account", err.Error())
			return
		}
		// A failed discovery remains distinguishable from a successful >=1.24
		// empty result, allowing Read to retry discovery after partial creation.
		plan.DefaultSecretName = types.StringNull()
	}

	serviceAccountSetComputed(&plan, out)
	// Persist the remote identity before discovery can fail or time out.
	resp.Diagnostics.Append(serviceAccountWriteModel(ctx, &resp.State, plan, defaultAccount)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceAccountIdentity(out.ObjectMeta))...)
	}
	if resp.Diagnostics.HasError() || defaultAccount {
		return
	}
	secret, err := serviceAccountWaitForDefaultSecret(ctx, conn, out.Name, config, timeout)
	if err != nil {
		resp.Diagnostics.AddError("Error discovering default service account token", fmt.Sprintf("Service account %q was created, but its default token could not be discovered: %s", plan.ID.ValueString(), err))
		return
	}
	plan.DefaultSecretName = types.StringValue(secret)
	resp.Diagnostics.Append(serviceAccountWriteModel(ctx, &resp.State, plan, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Token generation may have changed resource_version after the POST.
	out, err = conn.CoreV1().ServiceAccounts(out.Namespace).Get(ctx, out.Name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading service account after creation", err.Error())
		return
	}
	serviceAccountSetComputed(&plan, out)
	resp.Diagnostics.Append(serviceAccountWriteModel(ctx, &resp.State, plan, false)...)
}

func serviceAccountRead(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse, callback func() any, defaultAccount bool) {
	state, diags := serviceAccountReadModel(ctx, req.State, defaultAccount)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := serviceAccountIDParts(state.ID.ValueString(), defaultAccount)
	if err != nil {
		resp.Diagnostics.AddError("Invalid service account identifier", err.Error())
		return
	}
	// Pre-identity SDK state needs identity even when this first refresh finds
	// the object missing: Framework validates identity after RemoveResource.
	if resp.Identity != nil && resp.Identity.Raw.IsFullyNull() {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceAccountIdentity(metav1.ObjectMeta{Namespace: namespace, Name: name}))...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	conn, filters, diags := serviceAccountClient(callback)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading service account", err.Error())
		return
	}
	// A recorded legacy token remains the excluded/default reference even after
	// the cluster upgrades past 1.24. Only undiscovered state needs discovery.
	if state.DefaultSecretName.IsNull() || state.DefaultSecretName.IsUnknown() {
		secret, d := serviceAccountDiscoverDefaultSecret(ctx, conn, out)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.DefaultSecretName = types.StringValue(secret)
	}
	resp.Diagnostics.Append(serviceAccountFlatten(ctx, &state, out, filters)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(serviceAccountWriteModel(ctx, &resp.State, state, defaultAccount)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceAccountIdentity(out.ObjectMeta))...)
	}
}

func serviceAccountUpdate(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, callback func() any, defaultAccount bool) {
	plan, diags := serviceAccountReadModel(ctx, req.Plan, defaultAccount)
	resp.Diagnostics.Append(diags...)
	state, diags := serviceAccountReadModel(ctx, req.State, defaultAccount)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Metadata) != 1 || len(state.Metadata) != 1 {
		resp.Diagnostics.AddError("Invalid service account metadata", "Exactly one metadata block is required in plan and state.")
		return
	}
	namespace, name, err := serviceAccountIDParts(state.ID.ValueString(), defaultAccount)
	if err != nil {
		resp.Diagnostics.AddError("Invalid service account identifier", err.Error())
		return
	}
	conn, _, diags := serviceAccountClient(callback)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := conn.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error reading service account during update", err.Error())
		return
	}
	plan.DefaultSecretName = state.DefaultSecretName
	ops, diags := serviceAccountPatch(ctx, state, plan, out)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(ops) != 0 {
		payload, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Error encoding service account patch", err.Error())
			return
		}
		out, err = conn.CoreV1().ServiceAccounts(namespace).Patch(ctx, name, k8stypes.JSONPatchType, payload, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Error updating service account", err.Error())
			return
		}
	}
	serviceAccountSetComputed(&plan, out)
	resp.Diagnostics.Append(serviceAccountWriteModel(ctx, &resp.State, plan, defaultAccount)...)
	if resp.Identity != nil {
		resp.Diagnostics.Append(resp.Identity.Set(ctx, serviceAccountIdentity(out.ObjectMeta))...)
	}
}

func serviceAccountDelete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse, callback func() any, defaultAccount bool) {
	state, diags := serviceAccountReadModel(ctx, req.State, defaultAccount)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	namespace, name, err := serviceAccountIDParts(state.ID.ValueString(), defaultAccount)
	if err != nil {
		resp.Diagnostics.AddError("Invalid service account identifier", err.Error())
		return
	}
	conn, _, diags := serviceAccountClient(callback)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	err = conn.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting service account", err.Error())
	}
}

func serviceAccountAdoptionPatch(ctx context.Context, plan ServiceAccountV1Model, current *corev1.ServiceAccount) (kubernetes.PatchOperations, diag.Diagnostics) {
	// Adoption has no managed prior references. Using the API object as the
	// baseline would interpret omitted blocks as instructions to delete them.
	empty := ServiceAccountV1Model{
		Metadata:                     []common.NamespacedMetadataModel{{}},
		Secret:                       types.SetNull(serviceAccountReferenceType),
		ImagePullSecret:              types.SetNull(serviceAccountReferenceType),
		AutomountServiceAccountToken: types.BoolNull(),
	}
	return serviceAccountPatch(ctx, empty, plan, current)
}

func serviceAccountPatch(ctx context.Context, state, plan ServiceAccountV1Model, current *corev1.ServiceAccount) (kubernetes.PatchOperations, diag.Diagnostics) {
	var diags diag.Diagnostics
	ops := make(kubernetes.PatchOperations, 0)
	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 {
		diags.AddError("Invalid service account metadata", "Exactly one metadata block is required in plan and state.")
		return ops, diags
	}
	ops = append(ops, serviceAccountMetadataPatch("annotations", state.Metadata[0].Annotations, plan.Metadata[0].Annotations, current.Annotations)...)
	ops = append(ops, serviceAccountMetadataPatch("labels", state.Metadata[0].Labels, plan.Metadata[0].Labels, current.Labels)...)
	oldSecrets, d := serviceAccountReferenceNames(ctx, state.Secret)
	diags.Append(d...)
	newSecrets, d := serviceAccountReferenceNames(ctx, plan.Secret)
	diags.Append(d...)
	oldImages, d := serviceAccountReferenceNames(ctx, state.ImagePullSecret)
	diags.Append(d...)
	newImages, d := serviceAccountReferenceNames(ctx, plan.ImagePullSecret)
	diags.Append(d...)
	slices.Sort(oldSecrets)
	slices.Sort(newSecrets)
	slices.Sort(oldImages)
	slices.Sort(newImages)
	if !slices.Equal(oldSecrets, newSecrets) {
		ops = append(ops, &kubernetes.AddOperation{Path: "/secrets", Value: serviceAccountSecrets(newSecrets, plan.DefaultSecretName.ValueString(), current.Secrets)})
	}
	if !slices.Equal(oldImages, newImages) {
		ops = append(ops, &kubernetes.AddOperation{Path: "/imagePullSecrets", Value: serviceAccountImageSecrets(newImages)})
	}
	if !state.AutomountServiceAccountToken.Equal(plan.AutomountServiceAccountToken) {
		ops = append(ops, &kubernetes.AddOperation{Path: "/automountServiceAccountToken", Value: plan.AutomountServiceAccountToken.ValueBool()})
	}
	if len(ops) > 0 && current.ResourceVersion != "" {
		ops = append(kubernetes.PatchOperations{serviceAccountResourceVersionTest(current.ResourceVersion)}, ops...)
	}
	return ops, diags
}

// Kubernetes applies this precondition atomically with the following mutations,
// rejecting a patch built before a concurrent metadata or reference edit.
type serviceAccountResourceVersionTest string

func (serviceAccountResourceVersionTest) GetPath() string {
	return "/metadata/resourceVersion"
}

func (version serviceAccountResourceVersionTest) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value string `json:"value"`
	}{Op: "test", Path: version.GetPath(), Value: string(version)})
}

func serviceAccountMetadataPatch(field string, old, desired types.Map, current map[string]string) kubernetes.PatchOperations {
	before, after := common.ExpandMapForPatch(old), common.ExpandMapForPatch(desired)
	if maps.Equal(before, after) {
		return nil
	}
	remote := make(map[string]interface{}, len(current))
	for key, value := range current {
		remote[key] = value
	}
	merged := maps.Clone(remote)
	for key := range before {
		if _, retained := after[key]; !retained {
			delete(merged, key)
		}
	}
	maps.Copy(merged, after)
	if maps.Equal(remote, merged) {
		return nil
	}
	// Diff against the actual map so adding the first managed key cannot replace
	// controller-owned keys. Existing SDKv2 helpers escape JSON Pointer keys.
	return kubernetes.DiffStringMap("/metadata/"+field, remote, merged)
}

func serviceAccountSecrets(names []string, defaultName string, current []corev1.ObjectReference) []corev1.ObjectReference {
	references := make(map[string]corev1.ObjectReference, len(current))
	for _, reference := range current {
		references[reference.Name] = reference
	}
	allNames := slices.Clone(names)
	if defaultName != "" && !slices.Contains(allNames, defaultName) {
		allNames = append(allNames, defaultName)
	}
	result := make([]corev1.ObjectReference, 0, len(allNames))
	for _, name := range allNames {
		reference, exists := references[name]
		if !exists {
			reference = corev1.ObjectReference{Name: name}
		}
		result = append(result, reference)
	}
	return result
}

func serviceAccountImageSecrets(names []string) []corev1.LocalObjectReference {
	result := make([]corev1.LocalObjectReference, 0, len(names))
	for _, name := range names {
		result = append(result, corev1.LocalObjectReference{Name: name})
	}
	return result
}
