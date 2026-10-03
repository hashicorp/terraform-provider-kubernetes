// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common/podspec"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// These synthetic source states exercise every released source-version route.
// The resources empty list models SDKv2's omitted init-container resources.
func TestWorkloadSingletonStateMigration(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, alias, singleton, value string
		resource                      resource.Resource
		podOptions                    podspec.Options
	}{
		{"deployment", "kubernetes_deployment", "strategy", `[{"type":"RollingUpdate","rolling_update":[{"max_surge":"25%","max_unavailable":"25%"}]}]`, &DeploymentV1{}, podspec.Deployment()},
		{"daemonset", "kubernetes_daemonset", "strategy", `[{"type":"OnDelete","rolling_update":[]}]`, &DaemonSetV1{}, podspec.DaemonSet()},
		{"statefulset", "kubernetes_stateful_set", "persistent_volume_claim_retention_policy", `[{"when_deleted":"Delete","when_scaled":"Delete"}]`, &StatefulSetV1{}, podspec.StatefulSet()},
	} {
		for _, version := range []int64{0, 1} {
			t.Run(fmt.Sprintf("%s/v%d", tc.name, version), func(t *testing.T) {
				limits := `{"cpu":"0.5"}`
				requests := `{}`
				if version == 0 {
					limits = `[{"cpu":"0.5"}]`
					requests = `[]`
				}
				raw := &tfprotov6.RawState{JSON: []byte(fmt.Sprintf(`{"id":"ns/name","metadata":[{"name":"name","namespace":"ns","generate_name":""}],"spec":[{%q:%s,"template":[{"spec":[{"container":[{"name":"c","resources":[{"limits":%s,"requests":%s}]}],"init_container":[{"name":"init","resources":[]}]}]}]}]}`, tc.singleton, tc.value, limits, requests))}
				var schemaResponse resource.SchemaResponse
				tc.resource.Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
				if schemaResponse.Schema.Version != 2 {
					t.Fatalf("schema version = %d", schemaResponse.Schema.Version)
				}
				upgraded := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
				tc.resource.(resource.ResourceWithUpgradeState).UpgradeState(ctx)[version].StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: raw}, &upgraded)
				if upgraded.Diagnostics.HasError() {
					t.Fatal(upgraded.Diagnostics)
				}
				moved := resource.MoveStateResponse{TargetState: tfsdk.State{Schema: schemaResponse.Schema}}
				tc.resource.(resource.ResourceWithMoveState).MoveState(ctx)[0].StateMover(ctx, resource.MoveStateRequest{
					SourceTypeName: tc.alias, SourceSchemaVersion: version,
					SourceProviderAddress: "registry.terraform.io/hashicorp/kubernetes", SourceRawState: raw,
				}, &moved)
				if moved.Diagnostics.HasError() {
					t.Fatal(moved.Diagnostics)
				}
				if !moved.TargetState.Raw.Equal(upgraded.State.Raw) {
					t.Fatal("alias move and same-type upgrade differ")
				}
				podPath := path.Root("spec").AtListIndex(0).AtName("template").AtListIndex(0).AtName("spec")
				var prior types.List
				if d := upgraded.State.GetAttribute(ctx, podPath, &prior); d.HasError() {
					t.Fatal(d)
				}
				live := corev1.PodSpec{
					Containers:     []corev1.Container{{Name: "c", Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceCPU: k8sresource.MustParse("500m")}}}},
					InitContainers: []corev1.Container{{Name: "init"}},
				}
				refreshed, d := podspec.For(tc.podOptions).RefreshSpec(ctx, live, prior, podPath)
				if d.HasError() {
					t.Fatal(d)
				}
				for _, name := range []string{"container", "init_container"} {
					get := func(spec types.List) types.Object {
						return spec.Elements()[0].(types.Object).Attributes()[name].(types.List).Elements()[0].(types.Object).Attributes()["resources"].(types.Object)
					}
					if before, after := get(prior), get(refreshed); !before.Equal(after) {
						t.Fatalf("%s resources did not converge: upgrade %s; refresh %s", name, before, after)
					}
				}
				var actual types.Object
				if d := upgraded.State.GetAttribute(ctx, path.Root("spec").AtListIndex(0).AtName(tc.singleton), &actual); d.HasError() {
					t.Fatal(d)
				}
				var want types.Object
				switch tc.name {
				case "deployment":
					surge, unavailable := intstr.FromString("25%"), intstr.FromString("25%")
					want, d = flattenDeploymentStrategy(ctx, appsv1.DeploymentStrategy{Type: appsv1.RollingUpdateDeploymentStrategyType, RollingUpdate: &appsv1.RollingUpdateDeployment{MaxSurge: &surge, MaxUnavailable: &unavailable}}, actual)
				case "daemonset":
					want = flattenDaemonSetStrategyModel(ctx, appsv1.DaemonSetUpdateStrategy{Type: appsv1.OnDeleteDaemonSetStrategyType}, actual, &d)
				case "statefulset":
					model, diagnostics := flattenStatefulSetSpec(ctx, appsv1.StatefulSetSpec{PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType, WhenScaled: appsv1.DeletePersistentVolumeClaimRetentionPolicyType}}, nil, true)
					d.Append(diagnostics...)
					want = model.PersistentVolumeClaimRetentionPolicy
				}
				if d.HasError() {
					t.Fatal(d)
				}
				if !actual.Equal(want) {
					t.Fatalf("%s did not converge: upgrade %s; refresh %s", tc.singleton, actual, want)
				}
			})
		}
	}
}

func TestWorkloadSingletonStateValidation(t *testing.T) {
	for _, tc := range []struct{ name, value, wantError string }{
		{"missing", `{}`, ""},
		{"null", `{"strategy":null}`, ""},
		{"empty list", `{"strategy":[]}`, ""},
		{"object", `{"strategy":{"type":"Recreate","rolling_update":null}}`, ""},
		{"empty rolling list", `{"strategy":[{"rolling_update":[]}]}`, ""},
		{"multiple strategy", `{"strategy":[{},{}]}`, "state.spec[0].strategy"},
		{"null strategy member", `{"strategy":[null]}`, "state.spec[0].strategy[0]"},
		{"multiple rolling", `{"strategy":[{"rolling_update":[{},{}]}]}`, "state.spec[0].strategy.rolling_update"},
		{"scalar rolling", `{"strategy":[{"rolling_update":1}]}`, "state.spec[0].strategy.rolling_update"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var raw map[string]any
			if err := json.Unmarshal([]byte(`{"spec":[`+tc.value+`]}`), &raw); err != nil {
				t.Fatal(err)
			}
			convert := upgradeWorkloadState(1, "strategy")
			err := convert(raw)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v; want path %s", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			once, _ := json.Marshal(raw)
			if err := convert(raw); err != nil {
				t.Fatal(err)
			}
			twice, _ := json.Marshal(raw)
			if !reflect.DeepEqual(once, twice) {
				t.Fatal("converter is not idempotent")
			}
			spec := raw["spec"].([]any)[0].(map[string]any)
			if tc.name == "empty list" && spec["strategy"] != nil {
				t.Fatal("empty historical strategy must become null")
			}
		})
	}
	raw := map[string]any{"spec": []any{map[string]any{"persistent_volume_claim_retention_policy": []any{map[string]any{}, map[string]any{}}}}}
	if err := upgradeWorkloadState(1, "persistent_volume_claim_retention_policy")(raw); err == nil {
		t.Fatal("multiple retention policies must fail instead of truncating")
	}
}
