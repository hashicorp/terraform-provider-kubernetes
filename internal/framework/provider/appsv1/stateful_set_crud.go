// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	var specDiags diag.Diagnostics
	err = k8sretry.RetryOnConflict(k8sretry.DefaultRetry, func() error {
		live, err := conn.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		ops := common.MetadataPatchOpsAgainstLive("/metadata/", state.Metadata[0].MetadataModel, plan.Metadata[0].MetadataModel, live.ObjectMeta)
		if len(plan.Spec) == 1 && len(state.Spec) == 1 {
			var specOps kubernetes.PatchOperations
			specOps, specDiags = r.patchStatefulSetSpec(ctx, plan.Spec[0], state.Spec[0], live.Spec)
			if specDiags.HasError() {
				return nil
			}
			ops = append(ops, specOps...)
		}
		if len(ops) == 0 {
			return nil
		}
		ops = append(kubernetes.PatchOperations{common.ResourceVersionGuard(live.ResourceVersion)}, ops...)
		payload, err := ops.MarshalJSON()
		if err != nil {
			return err
		}
		_, err = conn.AppsV1().StatefulSets(namespace).Patch(ctx, name, k8types.JSONPatchType, payload, metav1.PatchOptions{})
		return err
	})
	if apierrors.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to update StatefulSet", err.Error())
		return
	}
	resp.Diagnostics.Append(specDiags...)
	if resp.Diagnostics.HasError() {
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
	if !resp.Diagnostics.HasError() {
		resp.Diagnostics.Append(resp.State.Set(ctx, &stateOut)...)
		resp.Diagnostics.Append(resp.Identity.Set(ctx, identOut)...)
	}
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
	var baselineSpec *StatefulSetSpecModel
	if len(baseline.Spec) > 0 {
		baselineSpec = &baseline.Spec[0]
	}
	spec, d := flattenStatefulSetSpec(ctx, obj.Spec, baselineSpec, filters, refresh)
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

func (r *StatefulSetV1) patchStatefulSetSpec(ctx context.Context, plan, state StatefulSetSpecModel, live appsv1.StatefulSetSpec) (kubernetes.PatchOperations, diag.Diagnostics) {
	ops := make(kubernetes.PatchOperations, 0)
	var diags diag.Diagnostics

	if !plan.Replicas.Equal(state.Replicas) {
		if !plan.Replicas.IsNull() && !plan.Replicas.IsUnknown() && plan.Replicas.ValueString() != "" {
			vv, err := strconv.Atoi(plan.Replicas.ValueString())
			if err != nil {
				diags.AddAttributeError(path.Root("spec").AtListIndex(0).AtName("replicas"), "Invalid replicas", err.Error())
				return ops, diags
			}
			ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec/replicas", Value: vv})
		}
	}

	if len(plan.Template) == 1 && len(state.Template) == 1 {
		if len(plan.Template[0].Metadata) == 1 && len(state.Template[0].Metadata) == 1 {
			ops = append(ops, common.MetadataPatchOpsAgainstLive(
				"/spec/template/metadata/",
				state.Template[0].Metadata[0].MetadataModel,
				plan.Template[0].Metadata[0].MetadataModel,
				live.Template.ObjectMeta,
			)...)
		}
		if !plan.Template[0].Spec.Equal(state.Template[0].Spec) {
			at := path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("spec")
			planned, d := expandPodTemplateSpec(ctx, plan.Template[0].Spec, at)
			diags.Append(d...)
			previous, d := expandPodTemplateSpec(ctx, state.Template[0].Spec, at)
			diags.Append(d...)
			if !diags.HasError() {
				if !reflect.DeepEqual(planned, previous) {
					merged, err := mergeStatefulSetPodSpec(previous, planned, live.Template.Spec)
					if err != nil {
						diags.AddAttributeError(at, "Unable to preserve live Pod specification", err.Error())
					} else {
						ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec/template/spec", Value: merged})
					}
				}
			}
		}
	}

	if !plan.MinReadySeconds.Equal(state.MinReadySeconds) && !plan.MinReadySeconds.IsNull() && !plan.MinReadySeconds.IsUnknown() {
		ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec/minReadySeconds", Value: int32(plan.MinReadySeconds.ValueInt64())})
	}

	if !updateStrategyEqual(plan.UpdateStrategy, state.UpdateStrategy) {
		ops = append(ops, patchUpdateStrategy(plan.UpdateStrategy, state.UpdateStrategy, live.UpdateStrategy)...)
	}

	if !plan.PersistentVolumeClaimRetentionPolicy.Equal(state.PersistentVolumeClaimRetentionPolicy) {
		retention, d := statefulSetPVCRetentionPolicyModels(ctx, plan.PersistentVolumeClaimRetentionPolicy, path.Root("spec").AtListIndex(0).AtName("persistent_volume_claim_retention_policy"))
		diags.Append(d...)
		if len(retention) == 0 {
			return ops, diags
		}
		desired := appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
			WhenDeleted: appsv1.PersistentVolumeClaimRetentionPolicyType(retention[0].WhenDeleted.ValueString()),
			WhenScaled:  appsv1.PersistentVolumeClaimRetentionPolicyType(retention[0].WhenScaled.ValueString()),
		}
		if live.PersistentVolumeClaimRetentionPolicy == nil {
			ops = append(ops, &kubernetes.AddOperation{Path: "/spec/persistentVolumeClaimRetentionPolicy", Value: desired})
		} else {
			if live.PersistentVolumeClaimRetentionPolicy.WhenDeleted != desired.WhenDeleted {
				ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec/persistentVolumeClaimRetentionPolicy/whenDeleted", Value: desired.WhenDeleted})
			}
			if live.PersistentVolumeClaimRetentionPolicy.WhenScaled != desired.WhenScaled {
				ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec/persistentVolumeClaimRetentionPolicy/whenScaled", Value: desired.WhenScaled})
			}
		}
	}

	return ops, diags
}

func patchUpdateStrategy(plan, state []StatefulSetUpdateStrategyModel, live appsv1.StatefulSetUpdateStrategy) kubernetes.PatchOperations {
	ops := make(kubernetes.PatchOperations, 0)
	if len(plan) == 0 {
		return ops
	}
	planType := plan[0].Type.ValueString()
	if planType != string(live.Type) {
		ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec/updateStrategy/type", Value: planType})
	}

	planRU := plan[0].RollingUpdate
	var stateRU []StatefulSetRollingUpdateModel
	if len(state) > 0 {
		stateRU = state[0].RollingUpdate
	}
	if len(planRU) == 0 && live.RollingUpdate != nil && (len(stateRU) > 0 || planType == string(appsv1.OnDeleteStatefulSetStrategyType)) {
		ops = append(ops, &kubernetes.RemoveOperation{Path: "/spec/updateStrategy/rollingUpdate"})
	}
	if len(planRU) > 0 {
		partition := int32(planRU[0].Partition.ValueInt64())
		if live.RollingUpdate == nil {
			ops = append(ops, &kubernetes.AddOperation{
				Path:  "/spec/updateStrategy/rollingUpdate",
				Value: appsv1.RollingUpdateStatefulSetStrategy{Partition: &partition},
			})
		} else if live.RollingUpdate.Partition == nil {
			ops = append(ops, &kubernetes.AddOperation{Path: "/spec/updateStrategy/rollingUpdate/partition", Value: partition})
		} else if *live.RollingUpdate.Partition != partition {
			ops = append(ops, &kubernetes.ReplaceOperation{Path: "/spec/updateStrategy/rollingUpdate/partition", Value: partition})
		}
	}
	return ops
}

func mergeStatefulSetPodSpec(previous, planned, live corev1.PodSpec) (corev1.PodSpec, error) {
	previousJSON, err := json.Marshal(previous)
	if err != nil {
		return corev1.PodSpec{}, err
	}
	plannedJSON, err := json.Marshal(planned)
	if err != nil {
		return corev1.PodSpec{}, err
	}
	liveJSON, err := json.Marshal(live)
	if err != nil {
		return corev1.PodSpec{}, err
	}
	mergedJSON, err := common.ThreeWayStrategicMerge(previousJSON, plannedJSON, liveJSON, corev1.PodSpec{})
	if err != nil {
		return corev1.PodSpec{}, err
	}
	var merged corev1.PodSpec
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		return corev1.PodSpec{}, err
	}
	return merged, nil
}

func updateStrategyEqual(a, b []StatefulSetUpdateStrategyModel) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	if !a[0].Type.Equal(b[0].Type) || len(a[0].RollingUpdate) != len(b[0].RollingUpdate) {
		return false
	}
	if len(a[0].RollingUpdate) == 0 {
		return true
	}
	return a[0].RollingUpdate[0].Partition.Equal(b[0].RollingUpdate[0].Partition)
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
