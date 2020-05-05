package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func expandPodDisruptionBudgetSpec(in []interface{}) (*api.PodDisruptionBudgetSpec, error) {
	spec := &api.PodDisruptionBudgetSpec{}
	if len(in) == 0 || in[0] == nil {
		return nil, fmt.Errorf("failed to expand PodDisruptionBudget.Spec: null or empty input")
	}
	m := in[0].(map[string]interface{})
	if v, ok := m["max_unavailable"].(string); ok && len(v) > 0 {
		val := intstr.Parse(v)
		spec.MaxUnavailable = &val
	}
	if v, ok := m["min_available"].(string); ok && len(v) > 0 {
		val := intstr.Parse(v)
		spec.MinAvailable = &val
	}
	if v, ok := m["selector"].([]interface{}); ok && len(v) > 0 {
		spec.Selector = expandLabelSelector(v)
	}

	return spec, nil
}

func flattenPodDisruptionBudgetSpec(spec api.PodDisruptionBudgetSpec) []interface{} {
	m := make(map[string]interface{}, 0)
	if spec.MaxUnavailable != nil {
		m["max_unavailable"] = spec.MaxUnavailable.String()
	}
	if spec.MinAvailable != nil {
		m["min_available"] = spec.MinAvailable.String()
	}
	if spec.Selector != nil {
		m["selector"] = flattenLabelSelector(spec.Selector)
	}

	return []interface{}{m}
}

// Currently unused, but will be useful for Kubernetes 1.15 when patching is allowed.
func patchPodDisruptionBudgetSpec(prefix string, pathPrefix string, d *schema.ResourceData) (*[]PatchOperation, error) {
	ops := make([]PatchOperation, 0)

	if d.HasChange(prefix + "max_unavailable") {
		old, new := d.GetChange(prefix + "max_unavailable")
		oldV, oldOk := old.(string)
		if newV, newOk := new.(string); newOk && len(newV) > 0 {
			if oldOk && len(oldV) > 0 {
				ops = append(ops, &ReplaceOperation{
					Path:  pathPrefix + "/maxUnavailable",
					Value: newV,
				})
			} else {
				ops = append(ops, &AddOperation{
					Path:  pathPrefix + "/maxUnavailable",
					Value: newV,
				})
			}
		} else if oldOk && len(oldV) > 0 {
			ops = append(ops, &RemoveOperation{
				Path: pathPrefix + "/maxUnavailable",
			})
		}
	}
	if d.HasChange(prefix + "min_available") {
		old, new := d.GetChange(prefix + "min_available")
		oldV, oldOk := old.(string)
		if newV, newOk := new.(string); newOk && len(newV) > 0 {
			if oldOk && len(oldV) > 0 {
				ops = append(ops, &ReplaceOperation{
					Path:  pathPrefix + "/minAvailable",
					Value: newV,
				})
			} else {
				ops = append(ops, &AddOperation{
					Path:  pathPrefix + "/minAvailable",
					Value: newV,
				})
			}
		} else if oldOk && len(oldV) > 0 {
			ops = append(ops, &RemoveOperation{
				Path: pathPrefix + "/minAvailable",
			})
		}
	}
	if d.HasChange(prefix + "selector") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/selector",
			Value: expandLabelSelector(d.Get(prefix + "selector").([]interface{})),
		})
	}

	return &ops, nil
}
