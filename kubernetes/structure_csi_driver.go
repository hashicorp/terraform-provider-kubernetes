package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	storage "k8s.io/api/storage/v1beta1"
)

func expandCSIDriverSpec(l []interface{}) storage.CSIDriverSpec {
	if len(l) == 0 || l[0] == nil {
		return storage.CSIDriverSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := storage.CSIDriverSpec{}

	if v, ok := in["attach_required"].(bool); ok {
		obj.AttachRequired = ptrToBool(v)
	}

	if v, ok := in["pod_info_on_mount"].(bool); ok {
		obj.PodInfoOnMount = ptrToBool(v)
	}

	if v, ok := in["volume_lifecycle_modes"].([]interface{}); ok && len(v) > 0 {
		obj.VolumeLifecycleModes = expandCSIDriverVolumeLifecycleModes(v)
	}

	return obj
}

func expandCSIDriverVolumeLifecycleModes(l []interface{}) []storage.VolumeLifecycleMode {
	lifecycleModes := make([]storage.VolumeLifecycleMode, 0, 0)
	for _, lifecycleMode := range l {
		lifecycleModes = append(lifecycleModes, storage.VolumeLifecycleMode(lifecycleMode.(string)))
	}
	return lifecycleModes
}

func flattenCSIDriverSpec(in storage.CSIDriverSpec) ([]interface{}, error) {
	att := make(map[string]interface{})

	att["attach_required"] = in.AttachRequired

	if in.PodInfoOnMount != nil {
		att["pod_info_on_mount"] = in.PodInfoOnMount
	}

	if len(in.VolumeLifecycleModes) > 0 {
		att["volume_lifecycle_modes"] = in.VolumeLifecycleModes
	}

	return []interface{}{att}, nil
}

func patchCSIDriverSpec(keyPrefix, pathPrefix string, d *schema.ResourceData) (*PatchOperations, error) {
	ops := make(PatchOperations, 0, 0)
	if d.HasChange(keyPrefix + "attach_required") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/attachRequired",
			Value: d.Get(keyPrefix + "attach_required").(bool),
		})
	}

	if d.HasChange(keyPrefix + "pod_info_on_mount") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/podInfoOnMount",
			Value: d.Get(keyPrefix + "pod_info_on_mount").(bool),
		})
	}

	if d.HasChange(keyPrefix + "volume_lifecycle_modes") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/volumeLifecycleModes",
			Value: expandCSIDriverVolumeLifecycleModes(d.Get(keyPrefix + "volume_lifecycle_modes").([]interface{})),
		})
	}

	return &ops, nil
}
