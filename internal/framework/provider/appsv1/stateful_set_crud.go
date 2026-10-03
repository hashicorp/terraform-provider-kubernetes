// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
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
	spec, d := expandStatefulSetSpec(ctx, plan.Spec[0])
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	obj := appsv1.StatefulSet{ObjectMeta: meta, Spec: *spec}
	created, err := conn.AppsV1().StatefulSets(meta.Namespace).Create(ctx, &obj, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError("Error creating StatefulSet", err.Error())
		return
	}

	plan.ID = types.StringValue(kubernetes.BuildId(created.ObjectMeta))
	plan.Metadata[0].Name = types.StringValue(created.Name)
	plan.Metadata[0].Namespace = types.StringValue(created.Namespace)
	plan.Metadata[0].UID = types.StringValue(string(created.UID))
	plan.Metadata[0].ResourceVersion = types.StringValue(created.ResourceVersion)
	plan.Metadata[0].Generation = types.Int64Value(created.Generation)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, statefulSetIdentityModel{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(statefulSetAPIVersion),
			Kind:       types.StringValue(statefulSetKind),
			Name:       types.StringValue(created.Name),
		},
		Namespace: types.StringValue(created.Namespace),
	})...)
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

	state, ident, d := r.readStateFromAPI(ctx, conn, filters, plan)
	resp.Diagnostics.Append(d...)
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		resp.Diagnostics.Append(resp.Identity.Set(ctx, ident)...)
	}
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

	if len(state.Metadata) != 1 || len(plan.Metadata) != 1 {
		resp.Diagnostics.AddAttributeError(path.Root("metadata"), "Invalid metadata", "Expected exactly one metadata block in state and plan")
		return
	}
	var original, desired *appsv1.StatefulSetSpec
	if len(plan.Spec) == 1 && len(state.Spec) == 1 {
		original, d = expandStatefulSetSpec(ctx, state.Spec[0])
		resp.Diagnostics.Append(d...)
		desired, d = expandStatefulSetSpec(ctx, plan.Spec[0])
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

	err = k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		raw, err := dynamicClient.Resource(appsv1.SchemeGroupVersion.WithResource("statefulsets")).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		live := &appsv1.StatefulSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw.Object, live); err != nil {
			return err
		}
		ops := common.MetadataPatchOpsAgainstLive("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel, live.ObjectMeta)
		if desired != nil {
			from, to := statefulSetPatchSpecs(*original, *desired, live.Spec)
			if !reflect.DeepEqual(from, to) {
				specOps, err := common.StrategicMergeSpecOps(raw, from, to, appsv1.StatefulSet{})
				if err != nil {
					return err
				}
				ops = append(ops, specOps...)
			}
		}
		if len(ops) == 0 {
			return nil
		}
		ops = append(kubernetes.PatchOperations{common.ResourceVersionGuard(raw.GetResourceVersion())}, ops...)
		payload, err := ops.MarshalJSON()
		if err != nil {
			return err
		}
		_, err = conn.AppsV1().StatefulSets(namespace).Patch(ctx, name, k8types.JSONPatchType, payload, metav1.PatchOptions{})
		return err
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update StatefulSet", err.Error())
		return
	}

	if plan.WaitForRollout.ValueBool() {
		err = retry.RetryContext(ctx, updateTimeout, retryUntilStatefulSetRolloutComplete(ctx, conn, namespace, name))
		if err != nil {
			resp.Diagnostics.AddError("Error waiting for StatefulSet rollout", err.Error())
			return
		}
	}

	stateOut, identOut, d := r.readStateFromAPI(ctx, conn, filters, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	keepPlannedClaimTemplateChanges(stateOut.Spec, plan.Spec)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateOut)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identOut)...)
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

func (r *StatefulSetV1) readStateFromAPI(ctx context.Context, conn *k8sclient.Clientset, filters kubernetes.MetadataFilters, baseline StatefulSetV1Model) (StatefulSetV1Model, statefulSetIdentityModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	namespace, name, err := kubernetes.IdParts(baseline.ID.ValueString())
	if err != nil {
		diags.AddError("Error parsing resource ID", err.Error())
		return StatefulSetV1Model{}, statefulSetIdentityModel{}, diags
	}
	obj, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		diags.AddError("Error reading StatefulSet", err.Error())
		return StatefulSetV1Model{}, statefulSetIdentityModel{}, diags
	}
	return r.flattenStateFromObject(ctx, filters, baseline, obj, false)
}

func (r *StatefulSetV1) refreshStateFromObject(ctx context.Context, filters kubernetes.MetadataFilters, baseline StatefulSetV1Model, obj *appsv1.StatefulSet, resp *resource.ReadResponse) {
	state, ident, diags := r.flattenStateFromObject(ctx, filters, baseline, obj, true)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, ident)...)
}

func (r *StatefulSetV1) flattenStateFromObject(ctx context.Context, filters kubernetes.MetadataFilters, baseline StatefulSetV1Model, obj *appsv1.StatefulSet, refresh bool) (StatefulSetV1Model, statefulSetIdentityModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	metadata, d := common.FlattenNamespacedMetadata(ctx, obj.ObjectMeta, baseline.Metadata, filters.GetIgnoreAnnotations(), filters.GetIgnoreLabels())
	diags.Append(d...)
	if diags.HasError() {
		return StatefulSetV1Model{}, statefulSetIdentityModel{}, diags
	}
	if !refresh {
		common.KeepPlannedMetadataMaps(metadata, baseline.Metadata)
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
	if !refresh && baselineSpec != nil &&
		!baselineSpec.Replicas.IsNull() && !baselineSpec.Replicas.IsUnknown() &&
		baselineSpec.Replicas.ValueString() == "" {
		spec.Replicas = baselineSpec.Replicas
	}

	state := baseline
	state.ID = types.StringValue(kubernetes.BuildId(obj.ObjectMeta))
	state.Metadata = metadata
	state.Spec = []StatefulSetSpecModel{spec}

	ident := statefulSetIdentityModel{
		ResourceIdentity: common.ResourceIdentity{
			APIVersion: types.StringValue(statefulSetAPIVersion),
			Kind:       types.StringValue(statefulSetKind),
			Name:       types.StringValue(obj.Name),
		},
		Namespace: types.StringValue(obj.Namespace),
	}
	return state, ident, diags
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

// keepPlannedClaimTemplateChanges records the planned claim-template fields an
// update cannot send (see statefulSetVolumeClaimRequiresReplace).
func keepPlannedClaimTemplateChanges(out, plan []StatefulSetSpecModel) {
	if len(out) != 1 || len(plan) != 1 {
		return
	}
	for i := range out[0].VolumeClaimTemplate {
		if i >= len(plan[0].VolumeClaimTemplate) {
			return
		}
		got, want := &out[0].VolumeClaimTemplate[i], plan[0].VolumeClaimTemplate[i]
		if len(got.Metadata) == 1 && len(want.Metadata) == 1 {
			got.Metadata[0].Labels = want.Metadata[0].Labels
			got.Metadata[0].Annotations = want.Metadata[0].Annotations
		}
		if len(got.Spec) == 1 && len(want.Spec) == 1 && len(got.Spec[0].Resources) == 1 && len(want.Spec[0].Resources) == 1 {
			got.Spec[0].Resources[0].Requests = want.Spec[0].Resources[0].Requests
		}
	}
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
