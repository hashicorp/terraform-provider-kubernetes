// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-provider-kubernetes/internal/framework/provider/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Only SDKv2's ForceNew claim-template fields replace; requests, labels and
// annotations, which the API also refuses to change, are planned in place.
func TestStatefulSetVolumeClaimRequiresReplace(t *testing.T) {
	strings := func(kv ...string) types.Map {
		values := map[string]attr.Value{}
		for i := 0; i < len(kv); i += 2 {
			values[kv[i]] = types.StringValue(kv[i+1])
		}
		return types.MapValueMust(types.StringType, values)
	}
	claim := func(change func(*PersistentVolumeClaimModel)) PersistentVolumeClaimModel {
		c := PersistentVolumeClaimModel{
			Metadata: []common.NamespacedMetadataModel{{
				MetadataModel: common.MetadataModel{
					MetadataBase: common.MetadataBase{
						Annotations: types.MapNull(types.StringType), Labels: types.MapNull(types.StringType),
						Name: types.StringValue("data"), Generation: types.Int64Value(0),
						ResourceVersion: types.StringValue(""), UID: types.StringValue(""),
					},
					GenerateName: types.StringNull(),
				},
				Namespace: types.StringValue("default"),
			}},
			Spec: []PersistentVolumeClaimSpecModel{{
				AccessModes:      types.SetValueMust(types.StringType, []attr.Value{types.StringValue("ReadWriteOnce")}),
				Resources:        []VolumeResourcesModel{{Limits: types.MapNull(types.StringType), Requests: strings("storage", "1Gi")}},
				VolumeName:       types.StringNull(),
				StorageClassName: types.StringValue("standard"),
				VolumeMode:       types.StringValue("Filesystem"),
			}},
		}
		if change != nil {
			change(&c)
		}
		return c
	}
	for _, tc := range []struct {
		name             string
		plan             []PersistentVolumeClaimModel
		unknown          bool
		replace, warning bool
	}{
		{name: "unchanged", plan: []PersistentVolumeClaimModel{claim(nil)}},
		{name: "equivalent requests", plan: []PersistentVolumeClaimModel{claim(func(c *PersistentVolumeClaimModel) {
			c.Spec[0].Resources[0].Requests = strings("storage", "1024Mi")
		})}},
		{name: "empty limits", plan: []PersistentVolumeClaimModel{claim(func(c *PersistentVolumeClaimModel) {
			c.Spec[0].Resources[0].Limits = strings()
			c.Metadata[0].Labels = strings()
		})}},
		{name: "requests", warning: true, plan: []PersistentVolumeClaimModel{claim(func(c *PersistentVolumeClaimModel) {
			c.Spec[0].Resources[0].Requests = strings("storage", "2Gi")
		})}},
		{name: "labels", warning: true, plan: []PersistentVolumeClaimModel{claim(func(c *PersistentVolumeClaimModel) {
			c.Metadata[0].Labels = strings("team", "db")
		})}},
		{name: "unknown annotations", warning: true, plan: []PersistentVolumeClaimModel{claim(func(c *PersistentVolumeClaimModel) {
			c.Metadata[0].Annotations = types.MapUnknown(types.StringType)
		})}},
		{name: "limits", replace: true, plan: []PersistentVolumeClaimModel{claim(func(c *PersistentVolumeClaimModel) {
			c.Spec[0].Resources[0].Limits = strings("storage", "2Gi")
		})}},
		{name: "access modes", replace: true, plan: []PersistentVolumeClaimModel{claim(func(c *PersistentVolumeClaimModel) {
			c.Spec[0].AccessModes = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("ReadWriteMany")})
		})}},
		{name: "unknown access modes", replace: true, plan: []PersistentVolumeClaimModel{claim(func(c *PersistentVolumeClaimModel) {
			c.Spec[0].AccessModes = types.SetUnknown(types.StringType)
		})}},
		{name: "added", replace: true, plan: []PersistentVolumeClaimModel{claim(nil), claim(nil)}},
		{name: "removed", replace: true},
		{name: "unknown list", replace: true, unknown: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			elementType := persistentVolumeClaimBlock().NestedObject.Type()
			state, diags := types.ListValueFrom(ctx, elementType, []PersistentVolumeClaimModel{claim(nil)})
			if diags.HasError() {
				t.Fatal(diags)
			}
			plan, diags := types.ListValueFrom(ctx, elementType, tc.plan)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if tc.unknown {
				plan = types.ListUnknown(elementType)
			}
			object := tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})
			req := planmodifier.ListRequest{
				Path:       path.Root("spec").AtListIndex(0).AtName("volume_claim_template"),
				State:      tfsdk.State{Raw: object},
				Plan:       tfsdk.Plan{Raw: object},
				StateValue: state,
				PlanValue:  plan,
			}
			resp := &planmodifier.ListResponse{PlanValue: plan}
			statefulSetVolumeClaimRequiresReplace{}.PlanModifyList(ctx, req, resp)
			if resp.RequiresReplace != tc.replace {
				t.Errorf("RequiresReplace = %t, want %t", resp.RequiresReplace, tc.replace)
			}
			if got := resp.Diagnostics.WarningsCount() > 0; got != tc.warning {
				t.Errorf("warning = %t, want %t: %v", got, tc.warning, resp.Diagnostics)
			}
		})
	}
}

func TestFlattenClaimTemplateMetadataMaps(t *testing.T) {
	ctx := context.Background()
	live := corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
		Name: "data", Labels: map[string]string{"app": "a", "admission.example.com/injected": "true"},
	}}
	planned := types.MapValueMust(types.StringType, map[string]attr.Value{"app": types.StringValue("a")})
	baseline := &PersistentVolumeClaimModel{Metadata: []common.NamespacedMetadataModel{{MetadataModel: common.MetadataModel{
		MetadataBase: common.MetadataBase{Labels: planned, Annotations: types.MapNull(types.StringType)},
	}}}}
	for _, tc := range []struct {
		name    string
		refresh bool
		labels  int
	}{
		{"write records the plan", false, 1},
		{"read records live keys", true, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := flattenPersistentVolumeClaim(ctx, live, baseline, tc.refresh)
			if diags.HasError() || len(got.Metadata[0].Labels.Elements()) != tc.labels || !got.Metadata[0].Annotations.IsNull() {
				t.Errorf("got %s, %s (%v)", got.Metadata[0].Labels, got.Metadata[0].Annotations, diags)
			}
		})
	}
}
