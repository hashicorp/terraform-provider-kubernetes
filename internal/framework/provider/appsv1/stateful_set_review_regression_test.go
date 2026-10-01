// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package appsv1

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	k8sappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
)

func TestStatefulSetClaimQuantityFlatten(t *testing.T) {
	ctx := context.Background()
	requests, diags := types.MapValueFrom(ctx, types.StringType, map[string]string{"storage": "1024Mi"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	prior := &PersistentVolumeClaimSpecModel{Resources: []VolumeResourcesModel{{Requests: requests}}}
	for name, tc := range map[string]struct{ api, want string }{
		"equivalent": {"1Gi", "1024Mi"},
		"changed":    {"2Gi", "2Gi"},
		"removed":    {"", ""},
	} {
		t.Run(name, func(t *testing.T) {
			spec := corev1.PersistentVolumeClaimSpec{}
			if tc.api != "" {
				spec.Resources.Requests = corev1.ResourceList{corev1.ResourceStorage: k8sresource.MustParse(tc.api)}
			}
			got, diags := flattenPersistentVolumeClaimSpec(ctx, spec, prior)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if tc.want == "" {
				if !got.Resources[0].Requests.IsNull() {
					t.Fatal("absent API quantity retained prior configuration")
				}
			} else if value := got.Resources[0].Requests.Elements()["storage"]; !value.Equal(types.StringValue(tc.want)) {
				t.Fatalf("storage = %s, want %s", value, tc.want)
			}
		})
	}
}

func TestStatefulSetTypeOnlyStrategyDoesNotAddRollingUpdate(t *testing.T) {
	partition := int32(0)
	api := k8sappsv1.StatefulSetSpec{
		Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main", Image: "busybox"}},
		}},
		UpdateStrategy: k8sappsv1.StatefulSetUpdateStrategy{
			Type:          k8sappsv1.RollingUpdateStatefulSetStrategyType,
			RollingUpdate: &k8sappsv1.RollingUpdateStatefulSetStrategy{Partition: &partition},
		},
	}
	prior := StatefulSetSpecModel{UpdateStrategy: []StatefulSetUpdateStrategyModel{{Type: types.StringValue("RollingUpdate")}}}
	got, diags := flattenStatefulSetSpec(context.Background(), api, &prior, testMetadataFilters{})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if len(got.UpdateStrategy) != 1 || len(got.UpdateStrategy[0].RollingUpdate) != 0 {
		t.Fatalf("API default added an unconfigured rolling_update block: %#v", got.UpdateStrategy)
	}
}
