// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func TestDeploymentStrategyNestedSchema(t *testing.T) {
	strategy, ok := deploymentSpecBlock().NestedObject.Attributes["strategy"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatal("strategy must carry child optionality rather than require every object field in HCL")
	}
	rolling, ok := strategy.NestedObject.Attributes["rolling_update"].(schema.ListNestedAttribute)
	if !ok || !rolling.Optional || !rolling.Computed {
		t.Fatal("rolling_update must be optional and computed")
	}
	for _, field := range []string{"max_surge", "max_unavailable"} {
		attribute := rolling.NestedObject.Attributes[field].(schema.StringAttribute)
		if !attribute.Optional || !attribute.Computed || attribute.Default == nil || len(attribute.Validators) == 0 {
			t.Fatalf("%s lost its default or validation", field)
		}
	}
}

func TestExpandDeploymentStrategy_Defaults(t *testing.T) {
	t.Parallel()

	value := types.ListNull(deploymentStrategyObjectType())
	strategy, diags := expandDeploymentStrategy(context.Background(), value, path.Empty())
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if strategy.Type != appsv1.RollingUpdateDeploymentStrategyType {
		t.Fatalf("expected RollingUpdate default, got %s", strategy.Type)
	}
	if strategy.RollingUpdate != nil {
		t.Fatalf("expected nil rolling_update for omitted strategy, got %#v", strategy.RollingUpdate)
	}
}

func TestExpandDeploymentSpec_UnknownReturnsDiagnostic(t *testing.T) {
	t.Parallel()

	_, diags := expandDeploymentSpec(
		context.Background(),
		types.ListUnknown(deploymentSpecListType().ElemType),
		path.Root("spec"),
	)
	if !diags.HasError() {
		t.Fatal("expected unknown deployment spec to return a diagnostic")
	}
}

func TestDeploymentMoveStateSourceGuards(t *testing.T) {
	t.Parallel()

	for _, version := range []int64{0, 1} {
		if !isSDKv2SourceType(resource.MoveStateRequest{
			SourceTypeName:        "kubernetes_deployment",
			SourceSchemaVersion:   version,
			SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes",
		}, "kubernetes_deployment") {
			t.Fatalf("valid SDKv2 schema version %d was rejected", version)
		}
	}
	for _, request := range []resource.MoveStateRequest{
		{SourceTypeName: "kubernetes_deployment_v1", SourceSchemaVersion: 1, SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{SourceTypeName: "kubernetes_deployment", SourceSchemaVersion: 2, SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes"},
		{SourceTypeName: "kubernetes_deployment", SourceSchemaVersion: 1, SourceProviderAddress: "registry.terraform.io/example/kubernetes"},
	} {
		if isSDKv2SourceType(request, "kubernetes_deployment") {
			t.Fatalf("unsupported move source accepted: %#v", request)
		}
	}
}

func TestFlattenDeploymentStrategy_RoundTrip(t *testing.T) {
	t.Parallel()

	maxSurge := intstr.Parse("50%")
	maxUnavailable := intstr.Parse("1")
	value, diags := flattenDeploymentStrategy(context.Background(), appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxSurge:       &maxSurge,
			MaxUnavailable: &maxUnavailable,
		},
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var entries []deploymentStrategyModel
	diags = value.ElementsAs(context.Background(), &entries, false)
	if diags.HasError() {
		t.Fatalf("unable to decode flattened strategy: %v", diags)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one strategy, got %d", len(entries))
	}
	if entries[0].Type.ValueString() != "RollingUpdate" {
		t.Fatalf("unexpected strategy type: %s", entries[0].Type.ValueString())
	}

	var updates []deploymentRollingUpdateModel
	diags = entries[0].RollingUpdate.ElementsAs(context.Background(), &updates, false)
	if diags.HasError() {
		t.Fatalf("unable to decode rolling_update: %v", diags)
	}
	if len(updates) != 1 {
		t.Fatalf("expected one rolling_update entry, got %d", len(updates))
	}
	if updates[0].MaxSurge.ValueString() != "50%" {
		t.Fatalf("unexpected max_surge: %s", updates[0].MaxSurge.ValueString())
	}
	if updates[0].MaxUnavailable.ValueString() != "1" {
		t.Fatalf("unexpected max_unavailable: %s", updates[0].MaxUnavailable.ValueString())
	}
}

func TestDeploymentMetadataPatchOps_NullToEmptyNoWrite(t *testing.T) {
	t.Parallel()

	state := DeploymentV1Model{
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
				Annotations: tfMap(nil),
				Labels:      tfMap(nil),
			}},
		}},
	}
	plan := DeploymentV1Model{
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
				Annotations: tfMap(map[string]string{}),
				Labels:      tfMap(nil),
			}},
		}},
	}

	ops := deploymentMetadataPatchOps(state, plan, metav1.ObjectMeta{
		Annotations: map[string]string{"external": "keep"},
	})
	if len(ops) != 0 {
		t.Fatalf("expected no patch operations, got %v", ops)
	}
}

func TestDeploymentMetadataPatchOps_FirstManagedKeyPreservesExternal(t *testing.T) {
	t.Parallel()

	state := DeploymentV1Model{
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
				Annotations: tfMap(nil),
				Labels:      tfMap(nil),
			}},
		}},
	}
	plan := DeploymentV1Model{
		Metadata: []common.NamespacedMetadataModel{{
			MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
				Annotations: tfMap(map[string]string{"managed": "new"}),
				Labels:      tfMap(nil),
			}},
		}},
	}

	live := metav1.ObjectMeta{Annotations: map[string]string{"external": "keep"}}
	before, err := json.Marshal(map[string]metav1.ObjectMeta{"metadata": live})
	if err != nil {
		t.Fatal(err)
	}

	ops := deploymentMetadataPatchOps(state, plan, live)
	if len(ops) == 0 {
		t.Fatal("expected metadata patch operations")
	}
	encoded, err := ops.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	patch, err := jsonpatch.DecodePatch(encoded)
	if err != nil {
		t.Fatal(err)
	}
	after, err := patch.Apply(before)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]metav1.ObjectMeta
	if err := json.Unmarshal(after, &got); err != nil {
		t.Fatal(err)
	}
	want := metav1.ObjectMeta{
		Annotations: map[string]string{"external": "keep", "managed": "new"},
	}
	if !reflect.DeepEqual(got["metadata"], want) {
		t.Fatalf("metadata=%#v want=%#v", got["metadata"], want)
	}
}

func TestDeploymentSpecPatch_PreservesExternalTemplateFields(t *testing.T) {
	t.Parallel()

	original := appsv1.DeploymentSpec{
		Replicas: int32Ptr(1),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"managed": "old"}},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "main", Image: "busybox:1.35"}},
			},
		},
	}
	modified := original.DeepCopy()
	modified.Replicas = int32Ptr(2)
	modified.Template.Labels["managed"] = "new"
	modified.Template.Spec.Containers[0].Image = "busybox:1.36"

	current := *original.DeepCopy()
	current.Template.Labels["external"] = "keep"
	current.Template.Spec.NodeSelector = map[string]string{"external": "keep"}
	current.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "INJECTED", Value: "keep"}}

	patch, err := deploymentSpecPatch(original, *modified, current)
	if err != nil {
		t.Fatal(err)
	}
	currentJSON, err := json.Marshal(appsv1.Deployment{Spec: current})
	if err != nil {
		t.Fatal(err)
	}
	afterJSON, err := strategicpatch.StrategicMergePatch(currentJSON, patch, appsv1.Deployment{})
	if err != nil {
		t.Fatal(err)
	}
	var after appsv1.Deployment
	if err := json.Unmarshal(afterJSON, &after); err != nil {
		t.Fatal(err)
	}
	if got := *after.Spec.Replicas; got != 2 {
		t.Fatalf("replicas=%d want=2", got)
	}
	if got := after.Spec.Template.Labels["managed"]; got != "new" {
		t.Fatalf("managed label=%q want=new", got)
	}
	if got := after.Spec.Template.Labels["external"]; got != "keep" {
		t.Fatalf("external label=%q want=keep", got)
	}
	if got := after.Spec.Template.Spec.NodeSelector["external"]; got != "keep" {
		t.Fatalf("external node selector=%q want=keep", got)
	}
	if got := after.Spec.Template.Spec.Containers[0].Env; len(got) != 1 || got[0].Name != "INJECTED" {
		t.Fatalf("injected environment lost: %#v", got)
	}
	if got := after.Spec.Template.Spec.Containers[0].Image; got != "busybox:1.36" {
		t.Fatalf("image=%q want=busybox:1.36", got)
	}
}

func TestFlattenTemplateMetadata_FiltersExternalKeysAfterCreate(t *testing.T) {
	t.Parallel()

	prior := []common.NamespacedMetadataModel{{
		MetadataModel: common.MetadataModel{MetadataBase: common.MetadataBase{
			Annotations: tfMap(map[string]string{"managed": "old"}),
			Labels:      tfMap(map[string]string{"managed": "old"}),
		}},
	}}
	metadata, diags := flattenTemplateMetadata(context.Background(), metav1.ObjectMeta{
		Annotations: map[string]string{"managed": "new", "external": "keep"},
		Labels:      map[string]string{"managed": "new", "external": "keep"},
	}, prior)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if len(metadata) != 1 {
		t.Fatalf("metadata entries=%d want=1", len(metadata))
	}
	for name, value := range map[string]types.Map{
		"annotations": metadata[0].Annotations,
		"labels":      metadata[0].Labels,
	} {
		got := map[string]string{}
		if diags := value.ElementsAs(context.Background(), &got, false); diags.HasError() {
			t.Fatal(diags)
		}
		if !reflect.DeepEqual(got, map[string]string{"managed": "new"}) {
			t.Fatalf("%s=%v want only managed key", name, got)
		}
	}
}

func int32Ptr(value int32) *int32 {
	return &value
}

func tfMap(kv map[string]string) types.Map {
	if kv == nil {
		return types.MapNull(types.StringType)
	}
	elems := make(map[string]attr.Value, len(kv))
	for k, v := range kv {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}
