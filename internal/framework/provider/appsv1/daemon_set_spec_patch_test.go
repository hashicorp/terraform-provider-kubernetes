// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func TestDaemonSetStrategicSpecPatchPreservesLiveOnlyFields(t *testing.T) {
	t.Parallel()

	oldAPI := daemonSetPatchTestSpec()
	state, diags := flattenDaemonSetSpecModel(context.Background(), oldAPI, nil)
	if diags.HasError() {
		t.Fatalf("flattening prior spec: %v", diags)
	}
	plan := append([]DaemonSetV1SpecModel(nil), state...)
	plan[0].MinReadySeconds = types.Int64Value(5)

	patch, diags := daemonSetStrategicSpecPatch(context.Background(), state, plan, nil)
	if diags.HasError() {
		t.Fatalf("creating patch: %v", diags)
	}

	live := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default"},
		Spec:       oldAPI,
	}
	live.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "INJECTED", Value: "keep"}}
	liveJSON, err := json.Marshal(live)
	if err != nil {
		t.Fatal(err)
	}
	updatedJSON, err := strategicpatch.StrategicMergePatch(liveJSON, patch, appsv1.DaemonSet{})
	if err != nil {
		t.Fatalf("applying patch: %v", err)
	}
	var updated appsv1.DaemonSet
	if err := json.Unmarshal(updatedJSON, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Spec.MinReadySeconds != 5 {
		t.Fatalf("minReadySeconds = %d, want 5", updated.Spec.MinReadySeconds)
	}
	if got := updated.Spec.Template.Spec.Containers[0].Env; len(got) != 1 || got[0].Name != "INJECTED" {
		t.Fatalf("live-only env was not preserved: %#v", got)
	}
}

func TestDaemonSetStrategicSpecPatchOnDeleteClearsRollingUpdate(t *testing.T) {
	t.Parallel()

	oldAPI := daemonSetPatchTestSpec()
	state, diags := flattenDaemonSetSpecModel(context.Background(), oldAPI, nil)
	if diags.HasError() {
		t.Fatalf("flattening prior spec: %v", diags)
	}
	plan := append([]DaemonSetV1SpecModel(nil), state...)
	plan[0].Strategy = daemonSetStrategyValue(t, "OnDelete", "0", "1")

	patch, diags := daemonSetStrategicSpecPatch(context.Background(), state, plan, nil)
	if diags.HasError() {
		t.Fatalf("creating patch: %v", diags)
	}
	liveJSON, _ := json.Marshal(appsv1.DaemonSet{Spec: oldAPI})
	updatedJSON, err := strategicpatch.StrategicMergePatch(liveJSON, patch, appsv1.DaemonSet{})
	if err != nil {
		t.Fatalf("applying patch: %v", err)
	}
	var updated appsv1.DaemonSet
	if err := json.Unmarshal(updatedJSON, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Spec.UpdateStrategy.Type != appsv1.OnDeleteDaemonSetStrategyType {
		t.Fatalf("strategy type = %q, want OnDelete", updated.Spec.UpdateStrategy.Type)
	}
	if updated.Spec.UpdateStrategy.RollingUpdate != nil {
		t.Fatalf("rollingUpdate was not cleared: %#v", updated.Spec.UpdateStrategy.RollingUpdate)
	}
}

func daemonSetPatchTestSpec() appsv1.DaemonSetSpec {
	maxSurge := intstr.FromInt32(0)
	maxUnavailable := intstr.FromInt32(1)
	revisionHistory := int32(10)
	return appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "test"}},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
			Spec: corev1.PodSpec{
				Containers:    []corev1.Container{{Name: "test", Image: "busybox:1.36"}},
				RestartPolicy: corev1.RestartPolicyAlways,
			},
		},
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxSurge:       &maxSurge,
				MaxUnavailable: &maxUnavailable,
			},
		},
		RevisionHistoryLimit: &revisionHistory,
	}
}
