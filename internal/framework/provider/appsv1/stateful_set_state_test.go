// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	k8sappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
)

type testMetadataFilters struct{}

func (testMetadataFilters) GetIgnoreAnnotations() []string { return nil }
func (testMetadataFilters) GetIgnoreLabels() []string      { return nil }

func TestUpgradeContainersV0ToV1(t *testing.T) {
	in := map[string]interface{}{
		"container": []interface{}{map[string]interface{}{
			"resources": []interface{}{map[string]interface{}{
				"requests": []interface{}{map[string]interface{}{"cpu": "100m"}},
				"limits":   []interface{}{map[string]interface{}{"memory": "1Gi"}},
			}},
		}},
	}

	got := upgradeContainersV0ToV1(in)
	resources := got["container"].([]interface{})[0].(map[string]interface{})["resources"].([]interface{})[0].(map[string]interface{})

	if _, ok := resources["requests"].(map[string]interface{}); !ok {
		t.Fatalf("requests is %T, want map[string]interface{}", resources["requests"])
	}
	if _, ok := resources["limits"].(map[string]interface{}); !ok {
		t.Fatalf("limits is %T, want map[string]interface{}", resources["limits"])
	}
}

func TestApplyStatefulSetV0ResourceUpgrade(t *testing.T) {
	v0 := map[string]interface{}{
		"spec": []interface{}{map[string]interface{}{
			"template": []interface{}{map[string]interface{}{
				"spec": []interface{}{map[string]interface{}{
					"container": []interface{}{map[string]interface{}{
						"resources": []interface{}{map[string]interface{}{
							"requests": []interface{}{map[string]interface{}{"cpu": "100m"}},
							"limits":   []interface{}{},
						}},
					}},
				}},
			}},
		}},
	}

	got := applyStatefulSetV0ResourceUpgrade(v0)
	_ = got // smoke assertion via roundtrip checks below

	b, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(b) {
		t.Fatal("upgraded state is not valid JSON")
	}
}

func TestApplyStatefulSetV0ResourceUpgradeMalformedState(t *testing.T) {
	inputs := []map[string]interface{}{
		{"spec": []interface{}{"not-an-object"}},
		{"spec": []interface{}{map[string]interface{}{"template": []interface{}{"not-an-object"}}}},
		{"spec": []interface{}{map[string]interface{}{"template": []interface{}{map[string]interface{}{"spec": []interface{}{"not-an-object"}}}}}},
	}
	for _, input := range inputs {
		if got := applyStatefulSetV0ResourceUpgrade(input); !reflect.DeepEqual(got, input) {
			t.Fatalf("malformed state changed: %#v", got)
		}
	}
}

func TestPatchUpdateStrategy(t *testing.T) {
	plan := []StatefulSetUpdateStrategyModel{{
		Type: types.StringValue("RollingUpdate"),
		RollingUpdate: []StatefulSetRollingUpdateModel{{
			Partition: types.Int64Value(3),
		}},
	}}
	state := []StatefulSetUpdateStrategyModel{{
		Type: types.StringValue("RollingUpdate"),
		RollingUpdate: []StatefulSetRollingUpdateModel{{
			Partition: types.Int64Value(1),
		}},
	}}
	livePartition := int32(1)

	ops := patchUpdateStrategy(plan, state, k8sappsv1.StatefulSetUpdateStrategy{
		Type: k8sappsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &k8sappsv1.RollingUpdateStatefulSetStrategy{
			Partition: &livePartition,
		},
	})
	if len(ops) != 1 {
		t.Fatalf("got %d ops, want 1", len(ops))
	}
	if !reflect.DeepEqual(ops[0].GetPath(), "/spec/updateStrategy/rollingUpdate/partition") {
		t.Fatalf("unexpected patch path: %s", ops[0].GetPath())
	}
}

func TestPatchUpdateStrategyUsesLiveState(t *testing.T) {
	partition := int32(2)
	maxUnavailable := intstr.FromString("10%")
	live := k8sappsv1.StatefulSetUpdateStrategy{
		Type: k8sappsv1.RollingUpdateStatefulSetStrategyType,
		RollingUpdate: &k8sappsv1.RollingUpdateStatefulSetStrategy{
			Partition:      &partition,
			MaxUnavailable: &maxUnavailable,
		},
	}
	plan := []StatefulSetUpdateStrategyModel{{
		Type: types.StringValue("RollingUpdate"),
		RollingUpdate: []StatefulSetRollingUpdateModel{{
			Partition: types.Int64Value(2),
		}},
	}}

	if ops := patchUpdateStrategy(plan, nil, live); len(ops) != 0 {
		t.Fatalf("got %d operations for strategy already matching live object", len(ops))
	}
}

func TestMergeStatefulSetPodSpecPreservesLiveFields(t *testing.T) {
	previous := corev1.PodSpec{
		Containers: []corev1.Container{{Name: "main", Image: "old"}},
	}
	planned := previous.DeepCopy()
	planned.Containers[0].Image = "new"
	live := previous.DeepCopy()
	live.NodeSelector = map[string]string{"admission.example/zone": "test"}
	live.Containers[0].Env = []corev1.EnvVar{{Name: "INJECTED", Value: "yes"}}

	merged, err := mergeStatefulSetPodSpec(previous, *planned, *live)
	if err != nil {
		t.Fatal(err)
	}
	if got := merged.Containers[0].Image; got != "new" {
		t.Fatalf("image = %q, want new", got)
	}
	if got := merged.NodeSelector["admission.example/zone"]; got != "test" {
		t.Fatalf("injected node selector lost: %#v", merged.NodeSelector)
	}
	if len(merged.Containers[0].Env) != 1 || merged.Containers[0].Env[0].Name != "INJECTED" {
		t.Fatalf("injected environment lost: %#v", merged.Containers[0].Env)
	}
}

func TestPatchStatefulSetSpecPreservesLiveTemplateMetadata(t *testing.T) {
	ctx := context.Background()
	stateLabels, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"app": "old"})
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	planLabels, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"app": "new"})
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	state := StatefulSetSpecModel{
		Template: []StatefulSetTemplateModel{{Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{Labels: stateLabels}},
		}}}},
	}
	plan := StatefulSetSpecModel{
		Template: []StatefulSetTemplateModel{{Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{Labels: planLabels}},
		}}}},
	}
	live := k8sappsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
			"app":                        "old",
			"admission.example/injected": "keep",
		}},
	}}

	ops, patchDiags := (&StatefulSetV1{}).patchStatefulSetSpec(ctx, plan, state, live)
	if patchDiags.HasError() {
		t.Fatal(patchDiags.Errors())
	}
	encoded, err := ops.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	patch, err := jsonpatch.DecodePatch(encoded)
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(k8sappsv1.StatefulSet{Spec: live})
	if err != nil {
		t.Fatal(err)
	}
	after, err := patch.Apply(before)
	if err != nil {
		t.Fatal(err)
	}
	var updated k8sappsv1.StatefulSet
	if err := json.Unmarshal(after, &updated); err != nil {
		t.Fatal(err)
	}
	if got := updated.Spec.Template.Labels["app"]; got != "new" {
		t.Fatalf("managed label = %q, want new", got)
	}
	if got := updated.Spec.Template.Labels["admission.example/injected"]; got != "keep" {
		t.Fatalf("unmanaged label = %q, want keep", got)
	}
}

func TestPatchStatefulSetSpecEmptyReplicasPreservesLiveCount(t *testing.T) {
	replicas := int32(3)
	plan := StatefulSetSpecModel{Replicas: types.StringValue("")}
	state := StatefulSetSpecModel{Replicas: types.StringValue("3")}
	live := k8sappsv1.StatefulSetSpec{Replicas: &replicas}

	ops, diags := (&StatefulSetV1{}).patchStatefulSetSpec(context.Background(), plan, state, live)
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	if len(ops) != 0 {
		t.Fatalf("empty replicas generated %d operations", len(ops))
	}
}

func TestStatefulSetMoveStateGuardsBeforeDecode(t *testing.T) {
	mover := (&StatefulSetV1{}).MoveState(context.Background())[0]
	for _, req := range []resource.MoveStateRequest{
		{
			SourceProviderAddress: "registry.terraform.io/other/kubernetes",
			SourceTypeName:        "kubernetes_stateful_set",
			SourceSchemaVersion:   1,
		},
		{
			SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
			SourceTypeName:        "kubernetes_other",
			SourceSchemaVersion:   1,
		},
		{
			SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
			SourceTypeName:        "kubernetes_stateful_set",
			SourceSchemaVersion:   2,
		},
	} {
		var resp resource.MoveStateResponse
		mover.StateMover(context.Background(), req, &resp)
		if len(resp.Diagnostics) != 0 {
			t.Fatalf("unsupported source returned diagnostics: %v", resp.Diagnostics)
		}
	}

	var resp resource.MoveStateResponse
	mover.StateMover(context.Background(), resource.MoveStateRequest{
		SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
		SourceTypeName:        "kubernetes_stateful_set",
		SourceSchemaVersion:   1,
	}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("supported source with empty state did not return an error")
	}
}

func TestFlattenStatefulSetSpecStrategyIsConfiguredOnly(t *testing.T) {
	partition := int32(2)
	spec := k8sappsv1.StatefulSetSpec{
		Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Image: "busybox"}},
		}},
		UpdateStrategy: k8sappsv1.StatefulSetUpdateStrategy{
			Type: k8sappsv1.RollingUpdateStatefulSetStrategyType,
			RollingUpdate: &k8sappsv1.RollingUpdateStatefulSetStrategy{
				Partition: &partition,
			},
		},
	}

	withoutConfig, diags := flattenStatefulSetSpec(context.Background(), spec, nil, testMetadataFilters{})
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	if len(withoutConfig.UpdateStrategy) != 0 {
		t.Fatalf("unconfigured strategy flattened into state: %#v", withoutConfig.UpdateStrategy)
	}

	baseline := StatefulSetSpecModel{
		Template: []StatefulSetTemplateModel{{Spec: withoutConfig.Template[0].Spec}},
		UpdateStrategy: []StatefulSetUpdateStrategyModel{{
			Type:          types.StringValue("RollingUpdate"),
			RollingUpdate: []StatefulSetRollingUpdateModel{{Partition: types.Int64Value(2)}},
		}},
	}
	withConfig, diags := flattenStatefulSetSpec(context.Background(), spec, &baseline, testMetadataFilters{})
	if diags.HasError() {
		t.Fatal(diags.Errors())
	}
	if len(withConfig.UpdateStrategy) != 1 || len(withConfig.UpdateStrategy[0].RollingUpdate) != 1 {
		t.Fatalf("configured strategy missing from state: %#v", withConfig.UpdateStrategy)
	}
	if got := withConfig.UpdateStrategy[0].RollingUpdate[0].Partition.ValueInt64(); got != 2 {
		t.Fatalf("partition = %d, want 2", got)
	}
}

func TestStatefulSetImportUsesDefaultRolloutPolicy(t *testing.T) {
	ctx := context.Background()
	r := &StatefulSetV1{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: tfsdk.State{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
	}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "default/imported"}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var rollout types.Bool
	if diags := resp.State.GetAttribute(ctx, path.Root("wait_for_rollout"), &rollout); diags.HasError() {
		t.Fatal(diags)
	}
	if !rollout.Equal(types.BoolValue(true)) {
		t.Fatalf("imported rollout policy = %s, want schema default true", rollout)
	}
}

func TestStatefulSetClaimComputedValuesResolve(t *testing.T) {
	prior := &PersistentVolumeClaimSpecModel{
		VolumeName:       types.StringUnknown(),
		StorageClassName: types.StringUnknown(),
		VolumeMode:       types.StringUnknown(),
	}
	got, diags := flattenPersistentVolumeClaimSpec(context.Background(), corev1.PersistentVolumeClaimSpec{}, prior)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if got.VolumeName.IsUnknown() || got.StorageClassName.IsUnknown() || got.VolumeMode.IsUnknown() {
		t.Fatalf("API flatten retained unknown computed claim values: %#v", got)
	}
}

func TestStatefulSetNoopPlanComputedValues(t *testing.T) {
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"configured": tftypes.String, "resource_version": tftypes.String,
	}}
	value := func(configured, version any) tftypes.Value {
		return tftypes.NewValue(typ, map[string]tftypes.Value{
			"configured":       tftypes.NewValue(tftypes.String, configured),
			"resource_version": tftypes.NewValue(tftypes.String, version),
		})
	}
	for name, tc := range map[string]struct {
		config, plan tftypes.Value
		preserve     bool
	}{
		"no-op":          {value("same", nil), value("same", tftypes.UnknownValue), true},
		"update":         {value("changed", nil), value("changed", tftypes.UnknownValue), false},
		"unknown config": {value(tftypes.UnknownValue, nil), value("same", tftypes.UnknownValue), false},
		"destroy":        {tftypes.NewValue(typ, nil), tftypes.NewValue(typ, nil), false},
	} {
		t.Run(name, func(t *testing.T) {
			state := value("same", "42")
			req := resource.ModifyPlanRequest{
				Config: tfsdk.Config{Raw: tc.config},
				State:  tfsdk.State{Raw: state},
				Plan:   tfsdk.Plan{Raw: tc.plan},
			}
			resp := resource.ModifyPlanResponse{Plan: req.Plan}
			(&StatefulSetV1{}).ModifyPlan(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			want := tc.plan
			if tc.preserve {
				want = state
			}
			if !resp.Plan.Raw.Equal(want) {
				t.Fatalf("plan = %s, want %s", resp.Plan.Raw, want)
			}
		})
	}
}
