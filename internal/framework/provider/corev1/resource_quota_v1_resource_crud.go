// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package corev1

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"

	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8Types "k8s.io/apimachinery/pkg/types"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// resourceQuotaDefaultTimeout matches SDKv2's `Create` and `Update: schema.DefaultTimeout(1 * time.Minute)`.
const resourceQuotaDefaultTimeout = 1 * time.Minute

// Create implements [resource.Resource].
func (r *ResourceQuotaV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceQuotaV1Model
	// Plan, not Config: metadata.name is unknown in the plan under generate_name.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, d := plan.Timeouts.Create(ctx, resourceQuotaDefaultTimeout)
	resp.Diagnostics.Append(d...)

	clients, _, metaDiags := r.meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	metadata, diags := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(diags...)
	spec, diags := expandResourceQuotaSpec(ctx, plan.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating resource quota", map[string]any{"name": metadata.Name, "namespace": metadata.Namespace})
	quotas := conn.CoreV1().ResourceQuotas(metadata.Namespace)
	out, err := quotas.Create(ctx, &api.ResourceQuota{ObjectMeta: metadata, Spec: spec}, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create resource quota", err.Error())
		return
	}

	// State is built from the plan, with only server-assigned fields overwritten from the
	// response. In particular spec.hard is not replaced by the API's canonical spelling:
	// that would not match the plan, which Terraform rejects as an inconsistent result.
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	common.PopulateNamespacedMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)

	// Persist before waiting. The quota exists now; if the wait fails Terraform must still
	// know about it, or it is orphaned and the next apply fails on a name conflict.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, resourceQuotaIdentity(out.Namespace, out.Name))...)
	if resp.Diagnostics.HasError() {
		return
	}

	settled, err := waitForResourceQuotaHard(ctx, quotas, out.Name, spec.Hard, createTimeout, "creation")
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for resource quota",
			fmt.Sprintf("Resource quota %q was created, but: %s", out.Name, err))
		return
	}

	// The quota controller updates status after creation, which bumps resourceVersion.
	plan.ID = types.StringValue(kubernetes.BuildId(settled.ObjectMeta))
	common.PopulateNamespacedMetadataFromResponse(&plan.Metadata[0], settled.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read implements [resource.Resource].
func (r *ResourceQuotaV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceQuotaV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// IdParts checks separators but permits an empty namespace or name.
	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil || namespace == "" || name == "" {
		resp.Diagnostics.AddError("Invalid resource ID",
			fmt.Sprintf("Expected format: namespace/name with a non-empty namespace and name, got: %q", state.ID.ValueString()))
		return
	}

	clients, filters, metaDiags := r.meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	quota, err := conn.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Terraform before 1.12 does not send identity on refresh, but the framework
			// still requires it when Read removes a missing resource.
			if resp.Identity.Raw.IsNull() {
				resp.Diagnostics.Append(resp.Identity.Set(ctx, resourceQuotaIdentity(namespace, name))...)
				if resp.Diagnostics.HasError() {
					return
				}
			}
			tflog.Info(ctx, "Resource quota no longer exists, removing from state",
				map[string]any{"name": name, "namespace": namespace})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read resource quota",
			fmt.Sprintf("Resource quota %q in namespace %q: %s", name, namespace, err))
		return
	}

	metadata, diags := common.FlattenNamespacedMetadata(ctx, quota.ObjectMeta, state.Metadata,
		filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Work around https://github.com/kubernetes/kubernetes/issues/44539: the API does not
	// always return generateName, so keep the value already in state. An empty string is
	// what SDKv2 stored for an unset value, so it is not kept: it has to become null.
	if quota.GenerateName == "" && len(state.Metadata) > 0 &&
		!state.Metadata[0].GenerateName.IsNull() && state.Metadata[0].GenerateName.ValueString() != "" {
		metadata[0].GenerateName = state.Metadata[0].GenerateName
	}

	spec, diags := flattenResourceQuotaSpec(ctx, quota.Spec, state.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ID = types.StringValue(kubernetes.BuildId(quota.ObjectMeta))
	state.Metadata = metadata
	state.Spec = spec
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, resourceQuotaIdentity(quota.Namespace, quota.Name))...)
}

// Update implements [resource.Resource].
func (r *ResourceQuotaV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceQuotaV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, d := plan.Timeouts.Update(ctx, resourceQuotaDefaultTimeout)
	resp.Diagnostics.Append(d...)

	clients, _, metaDiags := r.meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil || namespace == "" || name == "" {
		resp.Diagnostics.AddError("Invalid resource ID",
			fmt.Sprintf("Expected format: namespace/name with a non-empty namespace and name, got: %q", state.ID.ValueString()))
		return
	}

	planSpec, diags := expandResourceQuotaSpec(ctx, plan.Spec)
	resp.Diagnostics.Append(diags...)
	stateSpec, diags := expandResourceQuotaSpec(ctx, state.Spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ops := common.MetadataPatchOps("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel)
	specChanged := !equality.Semantic.DeepEqual(planSpec, stateSpec)
	if specChanged {
		ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec", Value: planSpec})
	}

	quotas := conn.CoreV1().ResourceQuotas(namespace)
	var out *api.ResourceQuota
	if len(ops) == 0 {
		// Only provider-side attributes changed (timeouts): nothing to send.
		out, err = quotas.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to read resource quota during update", err.Error())
			return
		}
	} else {
		data, err := ops.MarshalJSON()
		if err != nil {
			resp.Diagnostics.AddError("Failed to marshal update operations", err.Error())
			return
		}
		tflog.Info(ctx, "Updating resource quota", map[string]any{"name": name, "namespace": namespace, "patch": string(data)})
		out, err = quotas.Patch(ctx, name, k8Types.JSONPatchType, data, metav1.PatchOptions{})
		if err != nil {
			resp.Diagnostics.AddError("Failed to update resource quota", err.Error())
			return
		}
	}

	// Preserve the planned values; only server-assigned fields come from the response.
	plan.ID = types.StringValue(kubernetes.BuildId(out.ObjectMeta))
	common.PopulateNamespacedMetadataFromResponse(&plan.Metadata[0], out.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, resourceQuotaIdentity(out.Namespace, out.Name))...)
	if resp.Diagnostics.HasError() || !specChanged {
		return
	}

	settled, err := waitForResourceQuotaHard(ctx, quotas, name, planSpec.Hard, updateTimeout, "update")
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for resource quota",
			fmt.Sprintf("Resource quota %q was updated, but: %s", name, err))
		return
	}
	common.PopulateNamespacedMetadataFromResponse(&plan.Metadata[0], settled.ObjectMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete implements [resource.Resource].
func (r *ResourceQuotaV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceQuotaV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clients, _, metaDiags := r.meta()
	resp.Diagnostics.Append(metaDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil || namespace == "" || name == "" {
		resp.Diagnostics.AddError("Invalid resource ID",
			fmt.Sprintf("Expected format: namespace/name with a non-empty namespace and name, got: %q", state.ID.ValueString()))
		return
	}

	tflog.Info(ctx, "Deleting resource quota", map[string]any{"name": name, "namespace": namespace})
	// Already gone is success.
	if err := conn.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete resource quota", err.Error())
	}
}

func resourceQuotaIdentity(namespace, name string) common.NamespacedResourceIdentity {
	return common.NamespacedResourceIdentity{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(resourceQuotaAPIVersion),
			Kind:       types.StringValue(resourceQuotaKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}

// waitForResourceQuotaHard polls until status.hard matches the requested hard limits, as SDKv2
// did, and returns the last object read.
func waitForResourceQuotaHard(ctx context.Context, quotas typedcorev1.ResourceQuotaInterface, name string, want api.ResourceList, timeout time.Duration, operation string) (*api.ResourceQuota, error) {
	var last *api.ResourceQuota
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		quota, err := quotas.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}
		last = quota
		if resourceListsEqual(want, quota.Status.Hard) {
			return nil
		}
		return retry.RetryableError(fmt.Errorf("quotas don't match after %s.\nExpected: %#v\nGiven: %#v",
			operation, want, quota.Status.Hard))
	})
	return last, err
}

// resourceListsEqual compares quantities by value, so "1Gi" equals "1024Mi".
func resourceListsEqual(x, y api.ResourceList) bool {
	if len(x) != len(y) {
		return false
	}
	for k, v := range x {
		yv, ok := y[k]
		if !ok || v.Cmp(yv) != 0 {
			return false
		}
	}
	return true
}

// expandResourceQuotaSpec turns the planned spec into the API type. An absent spec block
// is an empty spec, as in SDKv2.
func expandResourceQuotaSpec(ctx context.Context, in []ResourceQuotaSpecModel) (api.ResourceQuotaSpec, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := api.ResourceQuotaSpec{}
	if len(in) == 0 {
		return out, diags
	}
	s := in[0]

	if !s.Hard.IsNull() && !s.Hard.IsUnknown() {
		out.Hard = make(api.ResourceList, len(s.Hard.Elements()))
		for k, v := range s.Hard.Elements() {
			q, ok := v.(common.QuantityValue)
			if !ok {
				diags.AddError("Invalid hard limit", fmt.Sprintf("Unexpected value type %T for %q. This is a bug in the provider.", v, k))
				continue
			}
			parsed, err := k8sresource.ParseQuantity(q.ValueString())
			if err != nil {
				diags.AddError("Invalid hard limit", fmt.Sprintf("%q: %s", k, err))
				continue
			}
			out.Hard[api.ResourceName(k)] = parsed
		}
	}

	if !s.Scopes.IsNull() && !s.Scopes.IsUnknown() {
		var scopes []string
		diags.Append(s.Scopes.ElementsAs(ctx, &scopes, false)...)
		sort.Strings(scopes)
		out.Scopes = make([]api.ResourceQuotaScope, len(scopes))
		for i, scope := range scopes {
			out.Scopes[i] = api.ResourceQuotaScope(scope)
		}
	}

	if len(s.ScopeSelector) > 0 {
		sel := &api.ScopeSelector{}
		exprs := s.ScopeSelector[0].MatchExpression
		sel.MatchExpressions = make([]api.ScopedResourceSelectorRequirement, len(exprs))
		for i, e := range exprs {
			sel.MatchExpressions[i].ScopeName = api.ResourceQuotaScope(e.ScopeName.ValueString())
			sel.MatchExpressions[i].Operator = api.ScopeSelectorOperator(e.Operator.ValueString())
			if !e.Values.IsNull() && !e.Values.IsUnknown() && len(e.Values.Elements()) > 0 {
				var values []string
				diags.Append(e.Values.ElementsAs(ctx, &values, false)...)
				sort.Strings(values)
				sel.MatchExpressions[i].Values = values
			}
		}
		out.ScopeSelector = sel
	}

	return out, diags
}

// flattenResourceQuotaSpec converts the API spec for Read. prior is the spec in state, and it
// decides the two things the API cannot: the spelling of each quantity, and whether an empty
// collection is null or empty. Both must match what the plan and the earlier apply wrote, or
// Terraform reports a diff or an inconsistent result.
func flattenResourceQuotaSpec(ctx context.Context, in api.ResourceQuotaSpec, prior []ResourceQuotaSpecModel) ([]ResourceQuotaSpecModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(in.Hard) == 0 && len(in.Scopes) == 0 && in.ScopeSelector == nil && len(prior) == 0 {
		return []ResourceQuotaSpecModel{}, diags
	}
	var p *ResourceQuotaSpecModel
	if len(prior) > 0 {
		p = &prior[0]
	}
	out := ResourceQuotaSpecModel{
		Hard:          types.MapNull(common.QuantityType{}),
		Scopes:        types.SetNull(types.StringType),
		ScopeSelector: []ResourceQuotaScopeSelectorModel{},
	}

	if len(in.Hard) > 0 || (p != nil && !p.Hard.IsNull()) {
		elems := make(map[string]attr.Value, len(in.Hard))
		for k, v := range in.Hard {
			value := common.NewQuantityValue(v.String())
			if p != nil {
				if old, ok := p.Hard.Elements()[string(k)].(common.QuantityValue); ok {
					if equal, _ := old.StringSemanticEquals(ctx, value); equal {
						value = old
					}
				}
			}
			elems[string(k)] = value
		}
		hard, d := types.MapValue(common.QuantityType{}, elems)
		diags.Append(d...)
		out.Hard = hard
	}

	if len(in.Scopes) > 0 || (p != nil && !p.Scopes.IsNull()) {
		scopes := make([]string, len(in.Scopes))
		for i, s := range in.Scopes {
			scopes[i] = string(s)
		}
		set, d := types.SetValueFrom(ctx, types.StringType, scopes)
		diags.Append(d...)
		out.Scopes = set
	}

	if in.ScopeSelector != nil {
		exprs := make([]ResourceQuotaMatchExpressionModel, len(in.ScopeSelector.MatchExpressions))
		for i, e := range in.ScopeSelector.MatchExpressions {
			values := types.SetNull(types.StringType)
			if len(e.Values) > 0 || priorValuesEmpty(p, i) {
				v, d := types.SetValueFrom(ctx, types.StringType, e.Values)
				diags.Append(d...)
				values = v
			}
			exprs[i] = ResourceQuotaMatchExpressionModel{
				ScopeName: types.StringValue(string(e.ScopeName)),
				Operator:  types.StringValue(string(e.Operator)),
				Values:    values,
			}
		}
		out.ScopeSelector = []ResourceQuotaScopeSelectorModel{{MatchExpression: exprs}}
	}

	return []ResourceQuotaSpecModel{out}, diags
}

// priorValuesEmpty reports whether the prior state held an empty, non-null values set for
// the i-th match expression, so a configured `values = []` is not turned into null.
func priorValuesEmpty(p *ResourceQuotaSpecModel, i int) bool {
	if p == nil || len(p.ScopeSelector) == 0 || i >= len(p.ScopeSelector[0].MatchExpression) {
		return false
	}
	v := p.ScopeSelector[0].MatchExpression[i].Values
	return !v.IsNull() && !v.IsUnknown() && len(v.Elements()) == 0
}
