package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes/structures"

	policy "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func expandPodDisruptionBudgetV1Spec(in []interface{}) (*policy.PodDisruptionBudgetSpec, error) {
	spec := &policy.PodDisruptionBudgetSpec{}
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
		spec.Selector = structures.ExpandLabelSelector(v)
	}

	return spec, nil
}

func flattenPodDisruptionBudgetV1Spec(spec policy.PodDisruptionBudgetSpec) []interface{} {
	m := make(map[string]interface{}, 0)
	if spec.MaxUnavailable != nil {
		m["max_unavailable"] = spec.MaxUnavailable.String()
	}
	if spec.MinAvailable != nil {
		m["min_available"] = spec.MinAvailable.String()
	}
	if spec.Selector != nil {
		m["selector"] = structures.FlattenLabelSelector(spec.Selector)
	}

	return []interface{}{m}
}

// Currently unused, but will be useful for Kubernetes 1.15 when patching is allowed.
func patchPodDisruptionBudgetV1Spec(prefix string, pathPrefix string, d *schema.ResourceData) (*[]structures.PatchOperation, error) {
	ops := make([]structures.PatchOperation, 0)

	if d.HasChange(prefix + "max_unavailable") {
		old, new := d.GetChange(prefix + "max_unavailable")
		oldV, oldOk := old.(string)
		if newV, newOk := new.(string); newOk && len(newV) > 0 {
			if oldOk && len(oldV) > 0 {
				ops = append(ops, &structures.ReplaceOperation{
					Path:  pathPrefix + "/maxUnavailable",
					Value: newV,
				})
			} else {
				ops = append(ops, &structures.AddOperation{
					Path:  pathPrefix + "/maxUnavailable",
					Value: newV,
				})
			}
		} else if oldOk && len(oldV) > 0 {
			ops = append(ops, &structures.RemoveOperation{
				Path: pathPrefix + "/maxUnavailable",
			})
		}
	}
	if d.HasChange(prefix + "min_available") {
		old, new := d.GetChange(prefix + "min_available")
		oldV, oldOk := old.(string)
		if newV, newOk := new.(string); newOk && len(newV) > 0 {
			if oldOk && len(oldV) > 0 {
				ops = append(ops, &structures.ReplaceOperation{
					Path:  pathPrefix + "/minAvailable",
					Value: newV,
				})
			} else {
				ops = append(ops, &structures.AddOperation{
					Path:  pathPrefix + "/minAvailable",
					Value: newV,
				})
			}
		} else if oldOk && len(oldV) > 0 {
			ops = append(ops, &structures.RemoveOperation{
				Path: pathPrefix + "/minAvailable",
			})
		}
	}
	if d.HasChange(prefix + "selector") {
		ops = append(ops, &structures.ReplaceOperation{
			Path:  pathPrefix + "/selector",
			Value: structures.ExpandLabelSelector(d.Get(prefix + "selector").([]interface{})),
		})
	}

	return &ops, nil
}
