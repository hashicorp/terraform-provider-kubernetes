// Copyright IBM Corp. 2017, 2026
// SPDX-License-Identifier: MPL-2.0

package podtemplate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
)

// One Terraform sources block can expand into several API projections. Restore
// that grouping only when its complete, ordered expansion still matches the API.
func preserveProjectedSourceGroups(ctx context.Context, spec corev1.PodSpec, baseline types.List, flattened []interface{}) {
	priorSpec, ok := singleKnownObject(baseline)
	if !ok || len(flattened) != 1 {
		return
	}
	priorVolumes, ok := priorSpec.Attributes()["volume"].(types.List)
	if !ok || priorVolumes.IsNull() || priorVolumes.IsUnknown() {
		return
	}
	flatSpec, ok := flattened[0].(map[string]interface{})
	if !ok {
		return
	}
	flatVolumes, ok := flatSpec["volume"].([]interface{})
	if !ok {
		return
	}
	actual := make(map[string]*corev1.ProjectedVolumeSource, len(spec.Volumes))
	for _, volume := range spec.Volumes {
		actual[volume.Name] = volume.Projected
	}
	for _, value := range priorVolumes.Elements() {
		volume, ok := value.(types.Object)
		if !ok || volume.IsNull() || volume.IsUnknown() {
			continue
		}
		name, ok := volume.Attributes()["name"].(types.String)
		if !ok || name.IsNull() || name.IsUnknown() || actual[name.ValueString()] == nil {
			continue
		}
		projected, ok := singleKnownObject(volume.Attributes()["projected"])
		if !ok {
			continue
		}
		sources, ok := projected.Attributes()["sources"].(types.List)
		if !ok || sources.IsNull() || sources.IsUnknown() {
			continue
		}
		var diagnostics diag.Diagnostics
		rawSources := podSpecAPIValue(ctx, sources, path.Empty(), "sources", nil, &diagnostics)
		if diagnostics.HasError() {
			continue
		}
		expanded, err := kubernetes.ExpandTemplatePodSpecForFramework([]interface{}{
			map[string]interface{}{"volume": []interface{}{
				map[string]interface{}{"projected": []interface{}{
					map[string]interface{}{"sources": rawSources},
				}},
			}},
		})
		if err != nil || len(expanded.Volumes) != 1 || expanded.Volumes[0].Projected == nil ||
			!apiequality.Semantic.DeepEqual(expanded.Volumes[0].Projected.Sources, actual[name.ValueString()].Sources) {
			continue
		}
		for _, value := range flatVolumes {
			flatVolume, ok := value.(map[string]interface{})
			if !ok || flatVolume["name"] != name.ValueString() {
				continue
			}
			flatProjected, ok := flatVolume["projected"].([]interface{})
			if !ok || len(flatProjected) != 1 {
				continue
			}
			if object, ok := flatProjected[0].(map[string]interface{}); ok {
				object["sources"] = rawSources
			}
		}
	}
}

func singleKnownObject(value attr.Value) (types.Object, bool) {
	list, ok := value.(types.List)
	if !ok || list.IsNull() || list.IsUnknown() || len(list.Elements()) != 1 {
		return types.Object{}, false
	}
	object, ok := list.Elements()[0].(types.Object)
	return object, ok && !object.IsNull() && !object.IsUnknown()
}
