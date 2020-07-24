package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	v1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/policy/v1beta1"
)

func flattenPodSecurityPolicySpec(in v1beta1.PodSecurityPolicySpec) []interface{} {
	spec := make(map[string]interface{})

	if in.AllowPrivilegeEscalation != nil {
		spec["allow_privilege_escalation"] = in.AllowPrivilegeEscalation
	}

	if len(in.AllowedCapabilities) > 0 {
		spec["allowed_capabilities"] = flattenCapability(in.AllowedCapabilities)
	}

	if len(in.AllowedFlexVolumes) > 0 {
		spec["allowed_flex_volumes"] = flattenAllowedFlexVolumes(in.AllowedFlexVolumes)
	}

	if len(in.AllowedHostPaths) > 0 {
		spec["allowed_host_paths"] = flattenAllowedHostPaths(in.AllowedHostPaths)
	}

	if len(in.AllowedProcMountTypes) > 0 {
		spec["allowed_proc_mount_types"] = flattenAllowedProcMountTypes(in.AllowedProcMountTypes)
	}

	if len(in.AllowedUnsafeSysctls) > 0 {
		spec["allowed_unsafe_sysctls"] = flattenListOfStrings(in.AllowedUnsafeSysctls)
	}

	if len(in.DefaultAddCapabilities) > 0 {
		spec["default_add_capabilities"] = flattenCapability(in.DefaultAddCapabilities)
	}

	if in.DefaultAllowPrivilegeEscalation != nil {
		spec["default_allow_privilege_escalation"] = in.DefaultAllowPrivilegeEscalation
	}

	if len(in.ForbiddenSysctls) > 0 {
		spec["forbidden_sysctls"] = flattenListOfStrings(in.ForbiddenSysctls)
	}

	spec["fs_group"] = flattenFSGroup(in.FSGroup)
	spec["host_ipc"] = in.HostIPC
	spec["host_network"] = in.HostNetwork
	spec["host_pid"] = in.HostPID

	if len(in.HostPorts) > 0 {
		spec["host_ports"] = flattenHostPortRangeSlice(in.HostPorts)
	}

	spec["privileged"] = in.Privileged
	spec["read_only_root_filesystem"] = in.ReadOnlyRootFilesystem

	if len(in.RequiredDropCapabilities) > 0 {
		spec["required_drop_capabilities"] = flattenCapability(in.RequiredDropCapabilities)
	}

	spec["run_as_user"] = flattenRunAsUser(in.RunAsUser)

	if in.RunAsGroup != nil {
		spec["run_as_group"] = flattenRunAsGroup(*in.RunAsGroup)
	}

	spec["se_linux"] = flattenSELinuxStrategy(in.SELinux)
	spec["supplemental_groups"] = flattenSupplementalGroups(in.SupplementalGroups)
	spec["volumes"] = flattenFSTypes(in.Volumes)

	return []interface{}{spec}
}

func flattenAllowedFlexVolumes(in []v1beta1.AllowedFlexVolume) []interface{} {
	result := make([]interface{}, len(in), len(in))

	for k, v := range in {
		result[k] = map[string]interface{}{
			"driver": v.Driver,
		}
	}

	return result
}

func flattenAllowedHostPaths(in []v1beta1.AllowedHostPath) []interface{} {
	result := make([]interface{}, len(in), len(in))

	for k, v := range in {
		result[k] = map[string]interface{}{
			"path_prefix": v.PathPrefix,
			"read_only":   v.ReadOnly,
		}
	}

	return result
}

func flattenListOfStrings(in []string) []interface{} {
	result := make([]interface{}, len(in), len(in))

	for k, v := range in {
		result[k] = v
	}

	return result
}

func flattenAllowedProcMountTypes(in []v1.ProcMountType) []interface{} {
	result := make([]interface{}, len(in), len(in))

	for k, v := range in {
		result[k] = fmt.Sprintf("%v", v)
	}

	return result
}

func flattenFSGroup(in v1beta1.FSGroupStrategyOptions) []interface{} {
	result := map[string]interface{}{
		"rule":  in.Rule,
		"range": flattenIDRangeSlice(in.Ranges),
	}

	return []interface{}{result}
}

func flattenIDRangeSlice(in []v1beta1.IDRange) []interface{} {
	result := make([]interface{}, len(in), len(in))

	for k, v := range in {
		result[k] = map[string]interface{}{
			"min": int(v.Min),
			"max": int(v.Max),
		}
	}

	return result
}

func flattenHostPortRangeSlice(in []v1beta1.HostPortRange) []interface{} {
	result := make([]interface{}, len(in), len(in))

	for k, v := range in {
		result[k] = map[string]interface{}{
			"min": int(v.Min),
			"max": int(v.Max),
		}
	}

	return result
}

func flattenRunAsUser(in v1beta1.RunAsUserStrategyOptions) []interface{} {
	result := map[string]interface{}{
		"rule":  fmt.Sprintf("%v", in.Rule),
		"range": flattenIDRangeSlice(in.Ranges),
	}

	return []interface{}{result}
}

func flattenRunAsGroup(in v1beta1.RunAsGroupStrategyOptions) []interface{} {
	result := map[string]interface{}{
		"rule":  fmt.Sprintf("%v", in.Rule),
		"range": flattenIDRangeSlice(in.Ranges),
	}

	return []interface{}{result}
}

func flattenSELinuxStrategy(in v1beta1.SELinuxStrategyOptions) []interface{} {
	result := map[string]interface{}{
		"rule": fmt.Sprintf("%v", in.Rule),
	}

	if in.SELinuxOptions != nil {
		result["se_linux_options"] = flattenSeLinuxOptions(in.SELinuxOptions)
	}

	return []interface{}{result}
}

func flattenSupplementalGroups(in v1beta1.SupplementalGroupsStrategyOptions) []interface{} {
	result := map[string]interface{}{
		"rule":  fmt.Sprintf("%v", in.Rule),
		"range": flattenIDRangeSlice(in.Ranges),
	}

	return []interface{}{result}
}

func flattenFSTypes(in []v1beta1.FSType) []interface{} {
	result := make([]interface{}, len(in), len(in))

	for k, v := range in {
		result[k] = fmt.Sprintf("%v", v)
	}

	return result
}

func expandPodSecurityPolicySpec(in []interface{}) (v1beta1.PodSecurityPolicySpec, error) {
	spec := v1beta1.PodSecurityPolicySpec{}
	if len(in) == 0 || in[0] == nil {
		return spec, fmt.Errorf("failed to expand PodSecurityPolicy.Spec: null or empty input")
	}

	m, ok := in[0].(map[string]interface{})
	if !ok {
		return spec, fmt.Errorf("failed to expand PodSecurityPolicy.Spec: malformed input")
	}

	if v, ok := m["allow_privilege_escalation"].(bool); ok {
		spec.AllowPrivilegeEscalation = ptrToBool(v)
	}

	if v, ok := m["allowed_capabilities"].([]interface{}); ok && len(v) > 0 {
		spec.AllowedCapabilities = expandCapabilitySlice(v)
	}

	if v, ok := m["allowed_flex_volumes"].([]interface{}); ok && len(v) > 0 {
		spec.AllowedFlexVolumes = expandAllowedFlexVolumeSlice(v)
	}

	if v, ok := m["allowed_host_paths"].([]interface{}); ok && len(v) > 0 {
		spec.AllowedHostPaths = expandAllowedHostPathSlice(v)
	}

	if v, ok := m["allowed_proc_mount_types"].([]interface{}); ok && len(v) > 0 {
		spec.AllowedProcMountTypes = expandAllowedProcMountTypes(v)
	}

	if v, ok := m["allowed_unsafe_sysctls"].([]interface{}); ok && len(v) > 0 {
		spec.AllowedUnsafeSysctls = expandStringSlice(v)
	}

	if v, ok := m["default_add_capabilities"].([]interface{}); ok && len(v) > 0 {
		spec.DefaultAddCapabilities = expandCapabilitySlice(v)
	}

	if v, ok := m["default_allow_privilege_escalation"].(bool); ok {
		spec.DefaultAllowPrivilegeEscalation = ptrToBool(v)
	}

	if v, ok := m["forbidden_sysctls"].([]interface{}); ok && len(v) > 0 {
		spec.ForbiddenSysctls = expandStringSlice(v)
	}

	if v, ok := m["fs_group"].([]interface{}); ok && len(v) > 0 {
		spec.FSGroup = expandFSGroup(v)
	}

	if v, ok := m["host_ipc"].(bool); ok {
		spec.HostIPC = v
	}

	if v, ok := m["host_network"].(bool); ok {
		spec.HostNetwork = v
	}

	if v, ok := m["host_pid"].(bool); ok {
		spec.HostPID = v
	}

	if v, ok := m["host_ports"].([]interface{}); ok && len(v) > 0 {
		spec.HostPorts = expandHostPortRangeSlice(v)
	}

	if v, ok := m["privileged"].(bool); ok {
		spec.Privileged = v
	}

	if v, ok := m["read_only_root_filesystem"].(bool); ok {
		spec.ReadOnlyRootFilesystem = v
	}

	if v, ok := m["required_drop_capabilities"].([]interface{}); ok && len(v) > 0 {
		spec.RequiredDropCapabilities = expandCapabilitySlice(v)
	}

	if v, ok := m["run_as_user"].([]interface{}); ok && len(v) > 0 {
		spec.RunAsUser = expandRunAsUser(v)
	}

	if v, ok := m["run_as_group"].([]interface{}); ok && len(v) > 0 {
		spec.RunAsGroup = expandRunAsGroup(v)
	}

	if v, ok := m["se_linux"].([]interface{}); ok && len(v) > 0 {
		spec.SELinux = expandSELinux(v)
	}

	if v, ok := m["supplemental_groups"].([]interface{}); ok && len(v) > 0 {
		spec.SupplementalGroups = expandSupplementalGroup(v)
	}

	if v, ok := m["volumes"].([]interface{}); ok && len(v) > 0 {
		spec.Volumes = expandVolumeFSTypeSlice(v)
	}

	return spec, nil
}

func expandAllowedFlexVolumeSlice(in []interface{}) []v1beta1.AllowedFlexVolume {
	result := make([]v1beta1.AllowedFlexVolume, len(in), len(in))
	for k, v := range in {
		result[k] = v1beta1.AllowedFlexVolume{
			Driver: v.(string),
		}
	}
	return result
}

func expandAllowedHostPathSlice(in []interface{}) []v1beta1.AllowedHostPath {
	result := make([]v1beta1.AllowedHostPath, len(in), len(in))
	for k, v := range in {
		if m, ok := v.(map[string]interface{}); ok {
			hp := v1beta1.AllowedHostPath{
				PathPrefix: m["path_prefix"].(string),
			}

			if ro, ok := m["read_only"].(bool); ok {
				hp.ReadOnly = ro
			}

			result[k] = hp
		}
	}
	return result
}

func expandAllowedProcMountTypes(in []interface{}) []v1.ProcMountType {
	result := make([]v1.ProcMountType, len(in), len(in))

	for k, v := range in {
		result[k] = v1.ProcMountType(v.(string))
	}

	return result
}

func expandFSGroup(in []interface{}) v1beta1.FSGroupStrategyOptions {
	result := v1beta1.FSGroupStrategyOptions{}

	m := in[0].(map[string]interface{})

	if v, ok := m["rule"].(string); ok {
		result.Rule = v1beta1.FSGroupStrategyType(v)
	}

	if v, ok := m["range"].([]interface{}); ok && len(v) > 0 {
		result.Ranges = expandIDRangeSlice(v)
	}

	return result
}

func expandIDRangeSlice(in []interface{}) []v1beta1.IDRange {
	result := make([]v1beta1.IDRange, len(in), len(in))

	for k, v := range in {
		if m, ok := v.(map[string]interface{}); ok {
			result[k] = v1beta1.IDRange{
				Min: int64(m["min"].(int)),
				Max: int64(m["max"].(int)),
			}
		}
	}

	return result
}

func expandHostPortRangeSlice(in []interface{}) []v1beta1.HostPortRange {
	result := make([]v1beta1.HostPortRange, len(in), len(in))

	for k, v := range in {
		if m, ok := v.(map[string]interface{}); ok {
			result[k] = v1beta1.HostPortRange{
				Min: int32(m["min"].(int)),
				Max: int32(m["max"].(int)),
			}
		}
	}

	return result
}

func expandRunAsUser(in []interface{}) v1beta1.RunAsUserStrategyOptions {
	result := v1beta1.RunAsUserStrategyOptions{}

	m := in[0].(map[string]interface{})

	if v, ok := m["rule"].(string); ok {
		result.Rule = v1beta1.RunAsUserStrategy(v)
	}

	if v, ok := m["range"].([]interface{}); ok && len(v) > 0 {
		result.Ranges = expandIDRangeSlice(v)
	}

	return result
}

func expandRunAsGroup(in []interface{}) *v1beta1.RunAsGroupStrategyOptions {
	result := v1beta1.RunAsGroupStrategyOptions{}

	m := in[0].(map[string]interface{})

	if v, ok := m["rule"].(string); ok {
		result.Rule = v1beta1.RunAsGroupStrategy(v)
	}

	if v, ok := m["range"].([]interface{}); ok && len(v) > 0 {
		result.Ranges = expandIDRangeSlice(v)
	}

	return &result
}

func expandSELinux(in []interface{}) v1beta1.SELinuxStrategyOptions {
	result := v1beta1.SELinuxStrategyOptions{}

	m := in[0].(map[string]interface{})

	if v, ok := m["rule"].(string); ok {
		result.Rule = v1beta1.SELinuxStrategy(v)
	}

	if v, ok := m["se_linux_options"].([]interface{}); ok && len(v) > 0 {
		result.SELinuxOptions = expandSELinuxOptions(v)
	}

	return result
}

func expandSELinuxOptions(in []interface{}) *v1.SELinuxOptions {
	result := v1.SELinuxOptions{}

	m := in[0].(map[string]interface{})

	if v, ok := m["level"].(string); ok {
		result.Level = v
	}

	if v, ok := m["user"].(string); ok {
		result.User = v
	}

	if v, ok := m["role"].(string); ok {
		result.Role = v
	}

	if v, ok := m["type"].(string); ok {
		result.Type = v
	}

	return &result
}

func expandSupplementalGroup(in []interface{}) v1beta1.SupplementalGroupsStrategyOptions {
	result := v1beta1.SupplementalGroupsStrategyOptions{}

	m := in[0].(map[string]interface{})

	if v, ok := m["rule"].(string); ok {
		result.Rule = v1beta1.SupplementalGroupsStrategyType(v)
	}

	if v, ok := m["range"].([]interface{}); ok && len(v) > 0 {
		result.Ranges = expandIDRangeSlice(v)
	}

	return result
}

func expandVolumeFSTypeSlice(in []interface{}) []v1beta1.FSType {
	result := make([]v1beta1.FSType, len(in), len(in))
	for k, v := range in {
		if s, ok := v.(string); ok {
			result[k] = v1beta1.FSType(s)
		}
	}

	return result
}

// Patchers

func patchPodSecurityPolicySpec(keyPrefix string, pathPrefix string, d *schema.ResourceData) (*PatchOperations, error) {
	ops := make(PatchOperations, 0, 0)

	if d.HasChange(keyPrefix + "allow_privilege_escalation") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/allowPrivilegeEscalation",
			Value: d.Get(keyPrefix + "allow_privilege_escalation").(bool),
		})
	}

	if d.HasChange(keyPrefix + "allowed_capabilities") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/allowedCapabilities",
			Value: d.Get(keyPrefix + "allowed_capabilities").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "allowed_flex_volumes") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/allowedFlexVolumes",
			Value: d.Get(keyPrefix + "allowed_flex_volumes").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "allowed_host_paths") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/allowedHostPaths",
			Value: d.Get(keyPrefix + "allowed_host_paths").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "allowed_proc_mount_types") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/allowedProcMountTypes",
			Value: d.Get(keyPrefix + "allowed_proc_mount_types").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "allowed_unsafe_sysctls") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/allowedUnsafeSysctls",
			Value: d.Get(keyPrefix + "allowed_unsafe_sysctls").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "default_add_capabilities") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/defaultAddCapabilities",
			Value: d.Get(keyPrefix + "default_add_capabilities").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "default_allow_privilege_escalation") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/defaultAllowPrivilegeEscalation",
			Value: d.Get(keyPrefix + "default_allow_privilege_escalation").(bool),
		})
	}

	if d.HasChange(keyPrefix + "forbidden_sysctls") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/forbiddenSysctls",
			Value: d.Get(keyPrefix + "forbidden_sysctls").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "fs_group") {
		fsGroup := expandFSGroup(d.Get(keyPrefix + "fs_group").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/fsGroup",
			Value: fsGroup,
		})
	}

	if d.HasChange(keyPrefix + "host_ipc") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/hostIPC",
			Value: d.Get(keyPrefix + "host_ipc").(bool),
		})
	}

	if d.HasChange(keyPrefix + "host_network") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/hostNetwork",
			Value: d.Get(keyPrefix + "host_network").(bool),
		})
	}

	if d.HasChange(keyPrefix + "host_pid") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/hostPID",
			Value: d.Get(keyPrefix + "host_pid").(bool),
		})
	}

	if d.HasChange(keyPrefix + "host_ports") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/hostPorts",
			Value: d.Get(keyPrefix + "host_ports").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "privileged") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/privileged",
			Value: d.Get(keyPrefix + "privileged").(bool),
		})
	}

	if d.HasChange(keyPrefix + "readonly_root_filesystem") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/readOnlyRootFilesystem",
			Value: d.Get(keyPrefix + "readonly_root_filesystem").(bool),
		})
	}

	if d.HasChange(keyPrefix + "required_drop_capabilities") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/requiredDropCapabilities",
			Value: d.Get(keyPrefix + "required_drop_capabilities").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "run_as_group") {
		runAsGroup := expandRunAsGroup(d.Get(keyPrefix + "run_as_group").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/runAsGroup",
			Value: runAsGroup,
		})
	}

	if d.HasChange(keyPrefix + "run_as_user") {
		runAsUser := expandRunAsUser(d.Get(keyPrefix + "run_as_user").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/runAsUser",
			Value: runAsUser,
		})
	}

	if d.HasChange(keyPrefix + "se_linux") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/seLinux",
			Value: d.Get(keyPrefix + "se_linux").([]interface{}),
		})
	}

	if d.HasChange(keyPrefix + "supplemental_groups") {
		supplementalGroups := expandSupplementalGroup(d.Get(keyPrefix + "supplemental_groups").([]interface{}))
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/supplementalGroups",
			Value: supplementalGroups,
		})
	}

	if d.HasChange(keyPrefix + "volumes") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/volumes",
			Value: d.Get(keyPrefix + "volumes").([]interface{}),
		})
	}

	return &ops, nil
}
