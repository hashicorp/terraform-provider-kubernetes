// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sclient "k8s.io/client-go/kubernetes"
	k8sretry "k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/polymorphichelpers"
)

const (
	defaultStatefulSetTimeout = 10 * time.Minute
)

func (r *StatefulSetV1) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StatefulSetV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, d := plan.Timeouts.Create(ctx, defaultStatefulSetTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	clients, filters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	meta, d := common.ExpandNamespacedMetadata(ctx, plan.Metadata)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Metadata) != 1 {
		resp.Diagnostics.AddAttributeError(path.Root("metadata"), "Invalid metadata", "Expected exactly one metadata block")
		return
	}
	if len(plan.Spec) != 1 {
		resp.Diagnostics.AddAttributeError(path.Root("spec"), "Invalid spec", "Expected exactly one spec block")
		return
	}
	spec, d := expandStatefulSetSpec(ctx, plan.Spec[0], &req.Config)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj := appsv1.StatefulSet{ObjectMeta: meta, Spec: *spec}
	if podspec.HasGatedFeatures(&spec.Template.Spec) {
		preview, err := conn.AppsV1().StatefulSets(meta.Namespace).Create(ctx, &obj, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		if err == nil {
			err = podspec.CheckPodFeaturePreservation(&spec.Template.Spec, &preview.Spec.Template.Spec)
		}
		if err != nil {
			resp.Diagnostics.AddError("Kubernetes Pod feature preflight failed", err.Error())
			return
		}
	}
	created, err := conn.AppsV1().StatefulSets(meta.Namespace).Create(ctx, &obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating StatefulSet", err.Error())
		return
	}

	plan.ID = types.StringValue(kubernetes.BuildId(created.ObjectMeta))
	resp.Diagnostics.Append(r.statefulSetWriteResult(ctx, &resp.State, req.Plan, plan, created, filters)...)
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, statefulSetIdentity(created.Namespace, created.Name))...)
	if err := podspec.CheckPodFeaturePreservation(&spec.Template.Spec, &created.Spec.Template.Spec); err != nil {
		resp.Diagnostics.AddError("Kubernetes did not preserve Pod features", err.Error())
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.WaitForRollout.ValueBool() {
		err = retry.RetryContext(ctx, createTimeout, retryUntilStatefulSetRolloutComplete(ctx, conn, created.Namespace, created.Name))
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for StatefulSet rollout", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(r.statefulSetReadWriteResult(ctx, &resp.State, req.Plan, plan, conn, filters)...)
}

func (r *StatefulSetV1) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StatefulSetV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, d := state.Timeouts.Read(ctx, defaultStatefulSetTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	clients, filters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing resource ID", err.Error())
		return
	}
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, statefulSetIdentity(namespace, name))...)
	obj, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading StatefulSet", err.Error())
		return
	}

	r.refreshStateFromObject(ctx, filters, state, obj, resp)
}

func (r *StatefulSetV1) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StatefulSetV1Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state StatefulSetV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, d := plan.Timeouts.Update(ctx, defaultStatefulSetTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	clients, filters, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing resource ID", err.Error())
		return
	}
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, statefulSetIdentity(namespace, name))...)

	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 {
		resp.Diagnostics.AddAttributeError(path.Root("metadata"), "Invalid metadata", "Expected exactly one metadata block in state and plan")
		return
	}
	var original, desired *appsv1.StatefulSetSpec
	if len(plan.Spec) == 1 && len(state.Spec) == 1 {
		// The configuration is read only for a changed spec, so an unchanged one
		// expands alike from plan and state and is not patched.
		var config *tfsdk.Config
		if !reflect.DeepEqual(plan.Spec, state.Spec) {
			config = &req.Config
		}
		original, d = expandStatefulSetSpec(ctx, state.Spec[0], nil)
		resp.Diagnostics.Append(d...)
		desired, d = expandStatefulSetSpec(ctx, plan.Spec[0], config)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	dynamicClient, err := clients.DynamicClient()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	var updated *appsv1.StatefulSet
	err = k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		raw, err := dynamicClient.Resource(appsv1.SchemeGroupVersion.WithResource("statefulsets")).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		live := &appsv1.StatefulSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw.Object, live); err != nil {
			return err
		}
		updated = live
		ops := common.MetadataPatchOpsAgainstLive("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel, live.ObjectMeta)
		if desired != nil {
			from, to := statefulSetPatchSpecs(*original.DeepCopy(), *desired, live.Spec)
			if diags := podspec.TCPHostPatchBaseline(ctx, req.State, req.Plan, path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("spec"), &from.Template.Spec, &live.Spec.Template.Spec); diags.HasError() {
				return fmt.Errorf("unable to prepare TCP probe host update: %v", diags)
			}
			specOps, err := common.StrategicMergeSpecOps(raw, from, to, appsv1.StatefulSet{})
			if err != nil {
				return err
			}
			ops = append(ops, specOps...)
		}
		if len(ops) == 0 {
			return nil
		}
		ops = append(kubernetes.PatchOperations{common.ResourceVersionGuard(raw.GetResourceVersion())}, ops...)
		payload, err := ops.MarshalJSON()
		if err != nil {
			return err
		}
		if desired != nil && podspec.HasGatedFeatures(&desired.Template.Spec) {
			preview, err := conn.AppsV1().StatefulSets(namespace).Patch(ctx, name, k8types.JSONPatchType, payload, metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}})
			if err != nil {
				return fmt.Errorf("Pod feature preflight failed: %w", err)
			}
			if err := podspec.CheckPodFeaturePreservation(&desired.Template.Spec, &preview.Spec.Template.Spec); err != nil {
				return err
			}
		}
		updated, err = conn.AppsV1().StatefulSets(namespace).Patch(ctx, name, k8types.JSONPatchType, payload, metav1.PatchOptions{})
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update StatefulSet", err.Error())
		return
	}

	if desired != nil {
		if err := podspec.CheckPodFeaturePreservation(&desired.Template.Spec, &updated.Spec.Template.Spec); err != nil {
			resp.Diagnostics.Append(r.statefulSetWriteResult(ctx, &resp.State, req.Plan, plan, updated, filters)...)
			resp.Diagnostics.AddError("Kubernetes did not preserve Pod features", err.Error())
			return
		}
	}

	if plan.WaitForRollout.ValueBool() {
		err = retry.RetryContext(ctx, updateTimeout, retryUntilStatefulSetRolloutComplete(ctx, conn, namespace, name))
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for StatefulSet rollout", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(r.statefulSetReadWriteResult(ctx, &resp.State, req.Plan, plan, conn, filters)...)
}

func (r *StatefulSetV1) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StatefulSetV1Model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, d := state.Timeouts.Delete(ctx, defaultStatefulSetTimeout)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	clients, _, diags := r.sdkv2Meta()
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := clients.MainClientset()
	if err != nil {
		resp.Diagnostics.AddError("Kubernetes client error", err.Error())
		return
	}

	namespace, name, err := kubernetes.IdParts(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing resource ID", err.Error())
		return
	}
	resp.Diagnostics.Append(common.SetIdentity(ctx, resp.Identity, statefulSetIdentity(namespace, name))...)

	err = conn.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		resp.Diagnostics.AddError("Kubernetes delete error", err.Error())
		return
	}

	err = wait.PollUntilContextTimeout(ctx, 500*time.Millisecond, deleteTimeout, true, func(ctx context.Context) (bool, error) {
		_, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Error waiting for StatefulSet deletion", err.Error())
	}
}

var statefulSetSpecListType = workloadSpecListType(statefulSetFrozenSchema)

// setStatefulSetState sets state to model; see workloadStateModel.
func setStatefulSetState(ctx context.Context, state *tfsdk.State, model StatefulSetV1Model) diag.Diagnostics {
	spec, diags := workloadListValue(statefulSetSpecListType(), model.Spec, func(in StatefulSetSpecModel, typ types.ObjectType) (attr.Value, diag.Diagnostics) {
		elemType := func(name string) attr.Type {
			return typ.AttrTypes[name].(types.ListType).ElemType
		}
		selector, diags := types.ListValueFrom(ctx, elemType("selector"), in.Selector)
		updateStrategy, d := types.ListValueFrom(ctx, elemType("update_strategy"), in.UpdateStrategy)
		diags.Append(d...)
		claims, d := types.ListValueFrom(ctx, elemType("volume_claim_template"), in.VolumeClaimTemplate)
		diags.Append(d...)
		template, d := workloadTemplateListValue(ctx, typ.AttrTypes["template"].(types.ListType), in.Template)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		object, d := types.ObjectValue(typ.AttrTypes, map[string]attr.Value{
			"pod_management_policy":  in.PodManagementPolicy,
			"replicas":               in.Replicas,
			"revision_history_limit": in.RevisionHistoryLimit,
			"ordinals":               in.Ordinals,
			"selector":               selector,
			"service_name":           in.ServiceName,
			"template":               template,
			"update_strategy":        updateStrategy,
			"volume_claim_template":  claims,
			"persistent_volume_claim_retention_policy": in.PersistentVolumeClaimRetentionPolicy,
			"min_ready_seconds":                        in.MinReadySeconds,
		})
		diags.Append(d...)
		return object, diags
	})
	if diags.HasError() {
		return diags
	}
	diags.Append(state.Set(ctx, &workloadStateModel{
		ID:             model.ID,
		Metadata:       model.Metadata,
		Spec:           spec,
		WaitForRollout: model.WaitForRollout,
		Timeouts:       model.Timeouts,
	})...)
	return diags
}

// statefulSetWriteResult records the plan after a write, with the values
// Kubernetes chose for those it left unknown.
func (r *StatefulSetV1) statefulSetWriteResult(ctx context.Context, state *tfsdk.State, plan tfsdk.Plan, model StatefulSetV1Model, obj *appsv1.StatefulSet, filters kubernetes.MetadataFilters) diag.Diagnostics {
	return common.SetWriteResult(ctx, state, plan, func(actual *tfsdk.State) diag.Diagnostics {
		written, _, diags := r.flattenStateFromObject(ctx, filters, model, obj, false)
		if diags.HasError() {
			return diags
		}
		return append(diags, setStatefulSetState(ctx, actual, written)...)
	})
}

// statefulSetReadWriteResult is statefulSetWriteResult for the StatefulSet as
// it is after a rollout.
func (r *StatefulSetV1) statefulSetReadWriteResult(ctx context.Context, state *tfsdk.State, plan tfsdk.Plan, model StatefulSetV1Model, conn *k8sclient.Clientset, filters kubernetes.MetadataFilters) diag.Diagnostics {
	var diags diag.Diagnostics
	namespace, name, err := kubernetes.IdParts(model.ID.ValueString())
	if err != nil {
		diags.AddError("Error parsing resource ID", err.Error())
		return diags
	}
	obj, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		diags.AddError("Error reading StatefulSet", err.Error())
		return diags
	}
	return r.statefulSetWriteResult(ctx, state, plan, model, obj, filters)
}

func (r *StatefulSetV1) refreshStateFromObject(ctx context.Context, filters kubernetes.MetadataFilters, baseline StatefulSetV1Model, obj *appsv1.StatefulSet, resp *resource.ReadResponse) {
	state, ident, diags := r.flattenStateFromObject(ctx, filters, baseline, obj, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(setStatefulSetState(ctx, &resp.State, state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, ident)...)
}

func (r *StatefulSetV1) flattenStateFromObject(ctx context.Context, filters kubernetes.MetadataFilters, baseline StatefulSetV1Model, obj *appsv1.StatefulSet, refresh bool) (StatefulSetV1Model, statefulSetIdentityModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	metadata, d := common.FlattenNamespacedMetadata(ctx, obj.ObjectMeta, baseline.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	diags.Append(d...)
	if diags.HasError() {
		return StatefulSetV1Model{}, statefulSetIdentityModel{}, diags
	}
	var baselineSpec *StatefulSetSpecModel
	if len(baseline.Spec) > 0 {
		baselineSpec = &baseline.Spec[0]
	}
	spec, d := flattenStatefulSetSpec(ctx, obj.Spec, baselineSpec, refresh)
	diags.Append(d...)
	if diags.HasError() {
		return StatefulSetV1Model{}, statefulSetIdentityModel{}, diags
	}

	state := baseline
	state.ID = types.StringValue(kubernetes.BuildId(obj.ObjectMeta))
	state.Metadata = metadata
	state.Spec = []StatefulSetSpecModel{spec}

	return state, statefulSetIdentity(obj.Namespace, obj.Name), diags
}

func statefulSetIdentity(namespace, name string) statefulSetIdentityModel {
	return statefulSetIdentityModel{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(statefulSetAPIVersion),
			Kind:       types.StringValue(statefulSetKind),
			Name:       types.StringValue(name),
		},
		Namespace: types.StringValue(namespace),
	}
}

// statefulSetPatchSpecs returns the specs to strategic-merge from and to.
// Fields an update may not change, and those the configuration leaves to the
// cluster, are left out or take their live values so the merge keeps them.
func statefulSetPatchSpecs(original, desired, live appsv1.StatefulSetSpec) (appsv1.StatefulSetSpec, appsv1.StatefulSetSpec) {
	from, to := original, desired
	from.Selector, to.Selector = live.Selector, live.Selector
	from.ServiceName, to.ServiceName = live.ServiceName, live.ServiceName
	from.PodManagementPolicy, to.PodManagementPolicy = "", ""
	from.VolumeClaimTemplates, to.VolumeClaimTemplates = nil, nil
	// Unset replicas are left to an autoscaler.
	if desired.Replicas == nil {
		from.Replicas, to.Replicas = nil, nil
	}
	from.UpdateStrategy, to.UpdateStrategy = live.UpdateStrategy, *live.UpdateStrategy.DeepCopy()
	if desired.UpdateStrategy.Type != "" {
		to.UpdateStrategy.Type = desired.UpdateStrategy.Type
		switch {
		case desired.UpdateStrategy.RollingUpdate != nil:
			if to.UpdateStrategy.RollingUpdate == nil {
				to.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{}
			}
			to.UpdateStrategy.RollingUpdate.Partition = desired.UpdateStrategy.RollingUpdate.Partition
		case original.UpdateStrategy.RollingUpdate != nil || desired.UpdateStrategy.Type == appsv1.OnDeleteStatefulSetStrategyType:
			to.UpdateStrategy.RollingUpdate = nil
		}
	}
	if desired.PersistentVolumeClaimRetentionPolicy == nil {
		from.PersistentVolumeClaimRetentionPolicy, to.PersistentVolumeClaimRetentionPolicy = live.PersistentVolumeClaimRetentionPolicy, live.PersistentVolumeClaimRetentionPolicy
	} else {
		from.PersistentVolumeClaimRetentionPolicy = live.PersistentVolumeClaimRetentionPolicy
	}
	return from, to
}

func retryUntilStatefulSetRolloutComplete(ctx context.Context, conn *k8sclient.Clientset, ns, name string) retry.RetryFunc {
	return func() *retry.RetryError {
		res, err := conn.AppsV1().StatefulSets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return retry.NonRetryableError(err)
		}

		if res.Spec.Replicas != nil && res.Status.ReadyReplicas != *res.Spec.Replicas {
			return retry.RetryableError(fmt.Errorf("StatefulSet %s/%s is not finished rolling out", ns, name))
		}

		gvk := appsv1.SchemeGroupVersion.WithKind("StatefulSet")
		statusViewer, err := polymorphichelpers.StatusViewerFor(gvk.GroupKind())
		if err != nil {
			return retry.NonRetryableError(err)
		}
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(res)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		obj["apiVersion"] = gvk.GroupVersion().String()
		obj["kind"] = gvk.Kind
		u := unstructured.Unstructured{Object: obj}
		_, done, err := statusViewer.Status(&u, 0)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		if done {
			return nil
		}
		return retry.RetryableError(fmt.Errorf("StatefulSet %s/%s is not finished rolling out", ns, name))
	}
}
