package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/client-go/pkg/api/v1"
)

// Flatteners

func flattenPersistentVolumeClaimSpec(in v1.PersistentVolumeClaimSpec) []interface{} {
	att := make(map[string]interface{})
	att["access_modes"] = flattenPersistentVolumeAccessModes(in.AccessModes)
	att["resources"] = flattenResourceRequirements(in.Resources)
	if in.Selector != nil {
		att["selector"] = flattenLabelSelector(in.Selector)
	}
	if in.VolumeName != "" {
		att["volume_name"] = in.VolumeName
	}
	if in.StorageClassName != nil {
		att["storage_class_name"] = *in.StorageClassName
	}
	return []interface{}{att}
}

func flattenResourceRequirements(in v1.ResourceRequirements) []interface{} {
	att := make(map[string]interface{})
	if len(in.Limits) > 0 {
		att["limits"] = flattenResourceList(in.Limits)
	}
	if len(in.Requests) > 0 {
		att["requests"] = flattenResourceList(in.Requests)
	}
	return []interface{}{att}
}

// Expanders

func expandPersistentVolumeClaimSpec(l []interface{}) (v1.PersistentVolumeClaimSpec, error) {
	if len(l) == 0 || l[0] == nil {
		return v1.PersistentVolumeClaimSpec{}, nil
	}
	in := l[0].(map[string]interface{})
	resourceRequirements, err := expandResourceRequirements(in["resources"].([]interface{}))
	if err != nil {
		return v1.PersistentVolumeClaimSpec{}, err
	}
	obj := v1.PersistentVolumeClaimSpec{
		AccessModes: expandPersistentVolumeAccessModes(in["access_modes"].(*schema.Set).List()),
		Resources:   resourceRequirements,
	}
	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelector(v)
	}
	if v, ok := in["volume_name"].(string); ok {
		obj.VolumeName = v
	}
	if v, ok := in["storage_class_name"].(string); ok && v != "" {
		obj.StorageClassName = ptrToString(v)
	}
	return obj, nil
}

func expandResourceRequirements(l []interface{}) (v1.ResourceRequirements, error) {
	if len(l) == 0 || l[0] == nil {
		return v1.ResourceRequirements{}, nil
	}
	in := l[0].(map[string]interface{})
	obj := v1.ResourceRequirements{}
	if v, ok := in["limits"].(map[string]interface{}); ok && len(v) > 0 {
		var err error
		obj.Limits, err = expandMapToResourceList(v)
		if err != nil {
			return obj, err
		}
	}
	if v, ok := in["requests"].(map[string]interface{}); ok && len(v) > 0 {
		var err error
		obj.Requests, err = expandMapToResourceList(v)
		if err != nil {
			return obj, err
		}
	}
	return obj, nil
}
