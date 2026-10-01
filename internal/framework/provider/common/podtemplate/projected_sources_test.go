// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podtemplate

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func TestFlattenSpecProjectedSourceGrouping(t *testing.T) {
	ctx := context.Background()
	apiSpec := corev1.PodSpec{
		Containers: []corev1.Container{{Name: "app", Image: "busybox"}},
		Volumes: []corev1.Volume{{
			Name: "projected",
			VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{
				DefaultMode: ptr.To(int32(0644)),
				Sources: []corev1.VolumeProjection{
					{Secret: &corev1.SecretProjection{
						LocalObjectReference: corev1.LocalObjectReference{Name: "credentials"},
						Items:                []corev1.KeyToPath{{Key: "password", Path: "password"}},
					}},
					{ConfigMap: &corev1.ConfigMapProjection{
						LocalObjectReference: corev1.LocalObjectReference{Name: "settings"},
					}},
				},
			}},
		}},
	}
	canonical := mustFlatten(t, apiSpec, types.ListNull(specType()))
	sourcesPath := []any{0, "volume", 0, "projected", 0, "sources"}
	sources := get(canonical, sourcesPath...).(types.List)
	group := sources.Elements()[0].(types.Object)
	groupValues := group.Attributes()
	groupValues["config_map"] = sources.Elements()[1].(types.Object).Attributes()["config_map"]
	group = types.ObjectValueMust(group.AttributeTypes(ctx), groupValues)
	grouped := types.ListValueMust(sources.ElementType(ctx), []attr.Value{group})
	baseline := with(t, canonical, grouped, sourcesPath...).(types.List)
	actual := mustExpand(t, baseline)

	t.Run("unchanged", func(t *testing.T) {
		refreshed := mustFlatten(t, actual, baseline)
		if got := get(refreshed, sourcesPath...); !got.Equal(grouped) {
			t.Fatalf("one configured sources block became %s", got)
		}
		if !reflect.DeepEqual(mustExpand(t, refreshed).Volumes, actual.Volumes) {
			t.Fatal("preserving grouping changed the expanded API volumes")
		}
	})

	t.Run("API omits empty slices", func(t *testing.T) {
		serialized := actual.DeepCopy()
		serialized.Volumes[0].Projected.Sources[1].ConfigMap.Items = nil
		refreshed := mustFlatten(t, *serialized, baseline)
		if got := get(refreshed, sourcesPath...); !got.Equal(grouped) {
			t.Fatalf("API serialization discarded the configured grouping: %s", got)
		}
	})

	for name, mutate := range map[string]func(*corev1.ProjectedVolumeSource){
		"secret name": func(v *corev1.ProjectedVolumeSource) { v.Sources[0].Secret.Name = "changed" },
		"secret item": func(v *corev1.ProjectedVolumeSource) { v.Sources[0].Secret.Items[0].Path = "changed" },
		"config map":  func(v *corev1.ProjectedVolumeSource) { v.Sources[1].ConfigMap.Name = "changed" },
		"config map optional": func(v *corev1.ProjectedVolumeSource) {
			v.Sources[1].ConfigMap.Optional = ptr.To(true)
		},
		"removed source": func(v *corev1.ProjectedVolumeSource) { v.Sources = v.Sources[:1] },
		"reordered sources": func(v *corev1.ProjectedVolumeSource) {
			v.Sources[0], v.Sources[1] = v.Sources[1], v.Sources[0]
		},
	} {
		t.Run(name, func(t *testing.T) {
			drifted := actual.DeepCopy()
			mutate(drifted.Volumes[0].Projected)
			refreshed := mustFlatten(t, *drifted, baseline)
			if get(refreshed, sourcesPath...).Equal(grouped) {
				t.Fatal("API projection drift was hidden by the configured grouping")
			}
			if got := mustExpand(t, refreshed).Volumes[0].Projected.Sources; !reflect.DeepEqual(got, drifted.Volumes[0].Projected.Sources) {
				t.Fatalf("refreshed projections do not match API drift: got %#v, want %#v", got, drifted.Volumes[0].Projected.Sources)
			}
		})
	}

	t.Run("import", func(t *testing.T) {
		imported := mustFlatten(t, actual, types.ListNull(specType()))
		if got := get(imported, sourcesPath...).(types.List); len(got.Elements()) != 2 {
			t.Fatalf("import invented a grouping absent from prior state: %s", got)
		}
	})
}
