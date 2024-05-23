// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package kubernetes

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	storage "k8s.io/api/storage/v1"
	"k8s.io/utils/ptr"
)

func expandCSIDriverV1Spec(l []interface{}) storage.CSIDriverSpec {
	if len(l) == 0 || l[0] == nil {
		return storage.CSIDriverSpec{}
	}
	in := l[0].(map[string]interface{})
	obj := storage.CSIDriverSpec{}

	if v, ok := in["attach_required"].(bool); ok {
		obj.AttachRequired = ptr.To(v)
	}

	if v, ok := in["pod_info_on_mount"].(bool); ok {
		obj.PodInfoOnMount = ptr.To(v)
	}

	if v, ok := in["volume_lifecycle_modes"].([]interface{}); ok && len(v) > 0 {
		obj.VolumeLifecycleModes = expandCSIDriverV1VolumeLifecycleModes(v)
	}

	return obj
}

func expandCSIDriverV1VolumeLifecycleModes(l []interface{}) []storage.VolumeLifecycleMode {
	lifecycleModes := make([]storage.VolumeLifecycleMode, 0)
	for _, lifecycleMode := range l {
		lifecycleModes = append(lifecycleModes, storage.VolumeLifecycleMode(lifecycleMode.(string)))
	}
	return lifecycleModes
}

func flattenCSIDriverV1Spec(in storage.CSIDriverSpec) []interface{} {
	att := make(map[string]interface{})

	att["attach_required"] = in.AttachRequired

	if in.PodInfoOnMount != nil {
		att["pod_info_on_mount"] = in.PodInfoOnMount
	}

	if len(in.VolumeLifecycleModes) > 0 {
		att["volume_lifecycle_modes"] = in.VolumeLifecycleModes
	}

	return []interface{}{att}
}

func patchCSIDriverV1Spec(keyPrefix, pathPrefix string, d *schema.ResourceData) *PatchOperations {
	ops := make(PatchOperations, 0)
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
			Value: expandCSIDriverV1VolumeLifecycleModes(d.Get(keyPrefix + "volume_lifecycle_modes").([]interface{})),
		})
	}

	return &ops
}
